package pagerduty

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// PagerDuty utilities to access the PagerDuty events API v2. This code was
// copied and adapted from the https://github.com/PagerDuty/go-pagerduty library
// and adapted to our needs by adding support for alternate endpoint URLs and
// for PagerDuty agents which don't  always return the same response object or
// JSON data.

// V2Event includes the incident/alert details
type V2Event struct {
	RoutingKey string        `json:"routing_key"`
	Action     string        `json:"event_action"`
	DedupKey   string        `json:"dedup_key,omitempty"`
	Images     []interface{} `json:"images,omitempty"`
	Links      []interface{} `json:"links,omitempty"`
	Client     string        `json:"client,omitempty"`
	ClientURL  string        `json:"client_url,omitempty"`
	Payload    *V2Payload    `json:"payload,omitempty"`
}

// V2Payload represents the individual event details for an event
type V2Payload struct {
	Summary   string      `json:"summary"`
	Source    string      `json:"source"`
	Severity  string      `json:"severity"`
	Timestamp string      `json:"timestamp,omitempty"`
	Component string      `json:"component,omitempty"`
	Group     string      `json:"group,omitempty"`
	Class     string      `json:"class,omitempty"`
	Details   interface{} `json:"custom_details,omitempty"`
}

// V2EventResponse is the json response body for an event
type V2EventResponse struct {
	Status   string   `json:"status,omitempty"`
	DedupKey string   `json:"dedup_key,omitempty"`
	Message  string   `json:"message,omitempty"`
	Errors   []string `json:"errors,omitempty"`
}

// EventsAPIV2Error represents the error response received when an Events API V2 call fails. The
// HTTP response code is set inside the StatusCode field, with the EventsAPIV2Error
// field being the wrapper around the JSON error object returned from the Events API V2.
type EventsAPIV2Error struct {
	// StatusCode is the HTTP response status code.
	StatusCode int `json:"-"`

	// APIError represents the object returned by the API when an error occurs,
	// which includes messages that should hopefully provide useful context
	// to the end user.
	//
	// If the API response did not contain an error object, the .Valid field of
	// APIError will be false. If .Valid is true, the .ErrorObject field is
	// valid and should be consulted.
	APIError NullEventsAPIV2ErrorObject

	message string
}

// Error satisfies the error interface, and should contain the StatusCode,
// APIError.Message, APIError.ErrorObject.Status, and APIError.Errors.
func (e EventsAPIV2Error) Error() string {
	if len(e.message) > 0 {
		return e.message
	}

	if !e.APIError.Valid {
		return fmt.Sprintf("HTTP response failed with status code %d and no JSON error object was present", e.StatusCode)
	}

	if len(e.APIError.ErrorObject.Errors) == 0 {
		return fmt.Sprintf(
			"HTTP response failed with status code %d, status: %s, message: %s",
			e.StatusCode, e.APIError.ErrorObject.Status, e.APIError.ErrorObject.Message,
		)
	}

	return fmt.Sprintf(
		"HTTP response failed with status code %d, status: %s, message: %s: %s",
		e.StatusCode,
		e.APIError.ErrorObject.Status,
		e.APIError.ErrorObject.Message,
		apiErrorsDetailString(e.APIError.ErrorObject.Errors),
	)
}

// NullEventsAPIV2ErrorObject is a wrapper around the EventsAPIV2ErrorObject type. If the Valid
// field is true, the API response included a structured error JSON object. This
// structured object is then set on the ErrorObject field.
//
// We assume it's possible in exceptional failure modes for error objects to be omitted by PagerDuty.
// As such, this wrapper type provides us a way to check if the object was
// provided while avoiding consumers accidentally missing a nil pointer check,
// thus crashing their whole program.
type NullEventsAPIV2ErrorObject struct {
	Valid       bool
	ErrorObject EventsAPIV2ErrorObject
}

// EventsAPIV2ErrorObject represents the object returned by the Events V2 API when an error
// occurs. This includes messages that should hopefully provide useful context
// to the end user.
type EventsAPIV2ErrorObject struct {
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`

	// Errors is likely to be empty, with the relevant error presented via the
	// Status field instead.
	Errors []string `json:"errors,omitempty"`
}

const v2EventsAPIEndpoint = "https://events.pagerduty.com/v2/enqueue"
const version = "2.5.0"

type Client struct {
	endpoint string
}

func NewClient() *Client {
	return &Client{endpoint: v2EventsAPIEndpoint}
}

func (c *Client) AlternateEndpoint(alternateEndpoint string) {
	c.endpoint = alternateEndpoint
}

// ManageEventWithContext handles the trigger, acknowledge, and resolve methods for an event.
// When connecting directly to the PagerDuty events API the response is returned as is. When
// using a proxy a response is artificially built using the status code and returned data.
func (c *Client) ManageEventWithContext(ctx context.Context, e *V2Event) (*V2EventResponse, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("User-Agent", "sensu-pagerduty-handler/"+version)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }() // explicitly discard error
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		errResp, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, EventsAPIV2Error{
				StatusCode: resp.StatusCode,
				message:    fmt.Sprintf("HTTP response with status code: %d: error: %s", resp.StatusCode, err),
			}
		}
		// now try to decode the response body into the error object.
		var eae EventsAPIV2Error
		err = json.Unmarshal(errResp, &eae)
		if err != nil {
			eae = EventsAPIV2Error{
				StatusCode: resp.StatusCode,
				message:    fmt.Sprintf("HTTP response with status code: %d, JSON unmarshal object body failed: %s, body: %s", resp.StatusCode, err, string(errResp)),
			}
			return nil, eae
		}

		eae.StatusCode = resp.StatusCode
		return nil, eae
	}

	var eventResponse V2EventResponse
	bodyContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(bodyContent, &eventResponse); err != nil {
		if c.endpoint == v2EventsAPIEndpoint {
			return nil, err
		}

		// Some PD agents return non-JSON content. Read the response and set it in the message.
		eventResponse.Status = resp.Status
		if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
			eventResponse.DedupKey = e.DedupKey
		}
		eventResponse.Message = string(bodyContent)
	}

	return &eventResponse, nil
}

func apiErrorsDetailString(errs []string) string {
	switch n := len(errs); n {
	case 0:
		panic("errs slice is empty")

	case 1:
		return errs[0]

	default:
		e := "error"
		if n > 2 {
			e += "s"
		}

		return fmt.Sprintf("%s (and %d more %s...)", errs[0], n-1, e)
	}
}
