package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/sensu/sensu-pagerduty-handler/pagerduty"

	corev2 "github.com/sensu/core/v2"
	"github.com/sensu/sensu-plugin-sdk/sensu"
	"github.com/sensu/sensu-plugin-sdk/templates"
	"golang.org/x/exp/slices"
)

type HandlerConfig struct {
	sensu.PluginConfig
	authToken         string
	dedupKeyTemplate  string
	statusMapJSON     string
	summaryTemplate   string
	teamName          string
	teamSuffix        string
	detailsTemplate   string
	detailsFormat     string
	alternateEndpoint string
	contactRouting    bool
	contacts          []string
	clientName        string
	sensuBaseUrl      string
	linkAnnotations   bool
	useEventTimestamp bool
	classTemplate     string
	groupTemplate     string
	componentTemplate string
}

type eventStatusMap map[string][]uint32

type detailsFormat string

const (
	stringDetailsFormat detailsFormat = "string"
	jsonDetailsFormat   detailsFormat = "json"
)

func (df detailsFormat) IsValid() bool {
	switch df {
	case stringDetailsFormat, jsonDetailsFormat:
		return true
	}
	return false
}

func (df detailsFormat) String() string {
	return string(df)
}

var (
	config = HandlerConfig{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-pagerduty-handler",
			Short:    "The Sensu Go PagerDuty handler for incident management",
			Keyspace: "sensu.io/plugins/sensu-pagerduty-handler/config",
		},
	}

	pagerDutyConfigOptions = []sensu.ConfigOption{
		&sensu.PluginConfigOption[string]{
			Path:      "token",
			Env:       "PAGERDUTY_TOKEN",
			Argument:  "token",
			Shorthand: "t",
			Secret:    true,
			Usage:     "The PagerDuty V2 API authentication token, can be set with PAGERDUTY_TOKEN",
			Value:     &config.authToken,
			Default:   "",
		},
		&sensu.PluginConfigOption[string]{
			Path:     "team",
			Env:      "PAGERDUTY_TEAM",
			Argument: "team",
			Usage:    "Envvar name for pager team(alphanumeric and underscores) holding PagerDuty V2 API authentication token, can be set with PAGERDUTY_TEAM",
			Value:    &config.teamName,
			Default:  "",
		},
		&sensu.PluginConfigOption[string]{
			Path:     "team-suffix",
			Env:      "PAGERDUTY_TEAM_SUFFIX",
			Argument: "team-suffix",
			Usage:    "Pager team suffix string to append if missing from team name, can be set with PAGERDUTY_TEAM_SUFFIX",
			Value:    &config.teamSuffix,
			Default:  "_pagerduty_token",
		},
		&sensu.PluginConfigOption[string]{
			Path:      "dedup-key-template",
			Env:       "PAGERDUTY_DEDUP_KEY_TEMPLATE",
			Argument:  "dedup-key-template",
			Shorthand: "k",
			Usage:     "The PagerDuty V2 API deduplication key template, can be set with PAGERDUTY_DEDUP_KEY_TEMPLATE",
			Value:     &config.dedupKeyTemplate,
			Default:   "{{.Entity.Name}}-{{.Check.Name}}",
		},
		&sensu.PluginConfigOption[string]{
			Path:      "status-map",
			Env:       "PAGERDUTY_STATUS_MAP",
			Argument:  "status-map",
			Shorthand: "s",
			Usage:     "The status map used to translate a Sensu check status to a PagerDuty severity, can be set with PAGERDUTY_STATUS_MAP",
			Value:     &config.statusMapJSON,
			Default:   "",
		},
		&sensu.PluginConfigOption[string]{
			Path:      "summary-template",
			Env:       "PAGERDUTY_SUMMARY_TEMPLATE",
			Argument:  "summary-template",
			Shorthand: "S",
			Usage:     "The template for the alert summary, can be set with PAGERDUTY_SUMMARY_TEMPLATE",
			Value:     &config.summaryTemplate,
			Default:   "{{.Entity.Name}}/{{.Check.Name}} : {{.Check.Output}}",
		},
		&sensu.PluginConfigOption[string]{
			Path:      "details-template",
			Env:       "PAGERDUTY_DETAILS_TEMPLATE",
			Argument:  "details-template",
			Shorthand: "d",
			Usage:     "The template for the alert details, can be set with PAGERDUTY_DETAILS_TEMPLATE (default full event JSON)",
			Value:     &config.detailsTemplate,
			Default:   "",
		},
		&sensu.PluginConfigOption[string]{
			Path:      "details-format",
			Env:       "PAGERDUTY_DETAILS_FORMAT",
			Argument:  "details-format",
			Shorthand: "",
			Usage:     "The format of the details output ('string' or 'json'), can be set with PAGERDUTY_DETAILS_FORMAT",
			Value:     &config.detailsFormat,
			Default:   "string",
		},
		&sensu.PluginConfigOption[string]{
			Path:      "alternate-endpoint",
			Env:       "PAGERDUTY_ALTERNATE_ENDPOINT",
			Argument:  "alternate-endpoint",
			Shorthand: "e",
			Usage:     "The endpoint to use to send the PagerDuty events, can be set with PAGERDUTY_ALTERNATE_ENDPOINT",
			Value:     &config.alternateEndpoint,
			Default:   "",
		},
		&sensu.PluginConfigOption[uint64]{
			Path:      "timeout",
			Env:       "PAGERDUTY_TIMEOUT",
			Argument:  "timeout",
			Shorthand: "",
			Usage:     "The maximum amount of time in seconds to wait for the event to be created, can be set with PAGERDUTY_TIMEOUT",
			Value:     &config.Timeout,
			Default:   uint64(30),
		},
		&sensu.PluginConfigOption[bool]{
			Path:     "",
			Env:      "",
			Argument: "contact-routing",
			Usage:    "Enable contact routing",
			Value:    &config.contactRouting,
			Default:  false,
		},
		&sensu.PluginConfigOption[string]{
			Path:     "client-name",
			Env:      "",
			Argument: "client-name",
			Usage:    "Name for the client, this will appear in Pagerduty when events are logged",
			Value:    &config.clientName,
			Default:  "Sensu",
		},
		&sensu.PluginConfigOption[string]{
			Path:      "sensu-base-url",
			Env:       "PAGERDUTY_SENSU_BASE_URL",
			Argument:  "sensu-base-url",
			Shorthand: "u",
			Usage:     "Base URL for sensu. The handler will add a link to the event using this",
			Value:     &config.sensuBaseUrl,
			Default:   "",
		},
		&sensu.PluginConfigOption[bool]{
			Path:      "link-annotations",
			Env:       "",
			Argument:  "link-annotations",
			Shorthand: "l",
			Usage:     "Add links for any annotations that are a URL",
			Value:     &config.linkAnnotations,
			Default:   false,
		},
		&sensu.PluginConfigOption[bool]{
			Path:      "use-event-timestamp",
			Env:       "",
			Argument:  "use-event-timestamp",
			Shorthand: "T",
			Usage:     "Use the timestamp from the Sensu event for the PD-CEF timestamp field",
			Value:     &config.useEventTimestamp,
			Default:   false,
		},
		&sensu.PluginConfigOption[string]{
			Path:      "class-template",
			Env:       "PAGERDUTY_CLASS_TEMPLATE",
			Argument:  "class-template",
			Shorthand: "",
			Usage:     "Template for PD-CEF class field, can be set with PAGERDUTY_CLASS_TEMPLATE",
			Value:     &config.classTemplate,
			Default:   "",
		},
		&sensu.PluginConfigOption[string]{
			Path:      "group-template",
			Env:       "PAGERDUTY_GROUP_TEMPLATE",
			Argument:  "group-template",
			Shorthand: "",
			Usage:     "Template for PD-CEF group field, can be set with PAGERDUTY_GROUP_TEMPLATE",
			Value:     &config.groupTemplate,
			Default:   "",
		},
		&sensu.PluginConfigOption[string]{
			Path:      "component-template",
			Env:       "PAGERDUTY_COMPONENT_TEMPLATE",
			Argument:  "component-template",
			Shorthand: "",
			Usage:     "Template for PD-CEF component field, can be set with PAGERDUTY_COMPONENT_TEMPLATE",
			Value:     &config.componentTemplate,
			Default:   "",
		},
	}
)

func main() {
	//goHandler := sensu.NewGoHandler(&config.PluginConfig, pagerDutyConfigOptions, checkArgs, handleEvent)
	goHandler := sensu.NewHandler(&config.PluginConfig, pagerDutyConfigOptions, checkArgs, handleEvent)
	goHandler.Execute()
}

func getTeamToken() (string, error) {
	//replace illegal characters
	reg, err := regexp.Compile("[^A-Za-z0-9]+")
	if err != nil {
		return "", err
	}
	//sanitize
	teamEnvVar := reg.ReplaceAllString(config.teamName, "_")
	teamEnvVarSuffix := reg.ReplaceAllString(config.teamSuffix, "_")
	//add suffix if needed
	if len(config.teamSuffix) > 0 {
		matched, err := regexp.MatchString(config.teamSuffix+"$", teamEnvVar)
		if err != nil {
			return "", err
		}
		if !matched {
			teamEnvVar = teamEnvVar + teamEnvVarSuffix
		}
	}
	if len(teamEnvVar) == 0 {
		return "", fmt.Errorf("unknown problem with team evironment variable")
	}
	log.Printf("Looking up token envvar: %s", teamEnvVar)
	teamToken := os.Getenv(teamEnvVar)
	if len(teamToken) == 0 {
		log.Printf("Token envvar %s is empty, using default token instead", teamEnvVar)
	} else {
		log.Printf("Token envvar %s found, replacing default token", teamEnvVar)
	}
	return teamToken, err
}

func checkArgs(event *corev2.Event) error {
	if !event.HasCheck() {
		return errors.New("event does not contain check")
	}

	if len(config.teamName) != 0 {
		teamToken, err := getTeamToken()
		if err != nil {
			return err
		}
		if len(teamToken) != 0 {
			config.authToken = teamToken
		}
	}

	if config.contactRouting {
		contacts := getContacts(event)
		if len(contacts) == 0 {
			return errors.New("contact routing enabled but no contacts were found")
		}
		if err := validateContacts(contacts); err != nil {
			return err
		}
		config.contacts = contacts
	} else {
		if len(config.authToken) == 0 {
			return errors.New("no auth token provided")
		}
	}

	if !detailsFormat(config.detailsFormat).IsValid() {
		return fmt.Errorf("invalid details format: %s", config.detailsFormat)
	}

	if len(config.alternateEndpoint) != 0 {
		if _, err := url.Parse(config.alternateEndpoint); err != nil {
			return fmt.Errorf("invalid alternate endpoint: %s", config.alternateEndpoint)
		}
	}

	return nil
}

func handleEvent(event *corev2.Event) error {
	if config.contactRouting {
		return handleEventContactRouting(event)
	}
	return manageIncident(event, config.authToken)
}

func handleEventContactRouting(event *corev2.Event) error {
	errd := false
	contacts := config.contacts
	log.Printf("Contact routing is enabled (contacts: %s)", strings.Join(contacts, ", "))

	for _, contact := range contacts {
		if err := handleEventForContact(event, contact); err != nil {
			log.Printf("WARNING: skipping contact \"%s\" (%s)", contact, err)
			errd = true
		}
	}

	if errd {
		return errors.New("handler execution error for one or more contacts")
	}
	return nil
}

func handleEventForContact(event *corev2.Event, contact string) error {
	token, err := getContactToken(contact)
	if err != nil {
		return err
	}

	return manageIncident(event, token)
}

func validateContacts(contacts []string) error {
	for _, contact := range contacts {
		if err := validateContact(contact); err != nil {
			return err
		}
	}
	return nil
}

func validateContact(contact string) error {
	validContact := regexp.MustCompile(`^[A-Za-z0-9_]+$`)
	if !validContact.MatchString(contact) {
		return fmt.Errorf("invalid contact syntax: %s", contact)
	}
	return nil
}

func getContacts(event *corev2.Event) []string {
	contacts := []string{}
	loadContactsFromMap(&contacts, event.Annotations)
	loadContactsFromMap(&contacts, event.Check.Annotations)
	loadContactsFromMap(&contacts, event.Entity.Annotations)

	return contacts
}

func loadContactsFromMap(contacts *[]string, m map[string]string) {
	if str, ok := m["contacts"]; ok {
		newContacts := strings.Split(str, ",")
		for _, contact := range newContacts {
			if !slices.Contains(*contacts, contact) {
				*contacts = append(*contacts, contact)
			}
		}
	}
}

func getContactToken(contact string) (string, error) {
	name := fmt.Sprintf("PAGERDUTY_TOKEN_%s", strings.ToUpper(contact))
	token := os.Getenv(name)
	if token == "" {
		return "", fmt.Errorf("no environment variable found for \"%s\"", name)
	}
	return token, nil
}

func manageIncident(event *corev2.Event, token string) error {
	ctx := context.Background()
	if config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(config.Timeout)*time.Second)
		defer cancel()
	}

	severity, err := getPagerDutySeverity(event, config.statusMapJSON)
	if err != nil {
		return err
	}
	log.Printf("Incident severity: %s", severity)

	summary, err := getSummary(event)
	if err != nil {
		return err
	}

	details, err := getDetails(event)
	if err != nil {
		return err
	}

	group, err := getGroup(event)
	if err != nil {
		return err
	}

	component, err := getComponent(event)
	if err != nil {
		return err
	}

	class, err := getClass(event)
	if err != nil {
		return err
	}

	// "The maximum permitted length of PG event is 512 KB. Let's limit check output to 256KB to prevent triggering a failed send"
	if len(event.Check.Output) > 256000 {
		log.Printf("Warning Incident Payload Truncated!")
		event.Check.Output = "WARNING Truncated:i\n" + event.Check.Output[:256000] + "..."
	}

	pdPayload := pagerduty.V2Payload{
		Source:    event.Entity.Name,
		Component: component,
		Severity:  severity,
		Summary:   summary,
		Details:   details,
		Class:     class,
		Group:     group,
		Timestamp: getTimestamp(event),
	}

	action := "trigger"

	if event.Check.Status == 0 {
		action = "resolve"
	}

	dedupKey, err := getPagerDutyDedupKey(event)
	if err != nil {
		return err
	}
	if len(dedupKey) == 0 {
		return fmt.Errorf("pagerduty dedup key is empty")
	}
	pdEvent := pagerduty.V2Event{
		RoutingKey: token,
		Action:     action,
		Payload:    &pdPayload,
		DedupKey:   dedupKey,
		Client:     config.clientName,
		ClientURL:  getClientUrl(event),
		Links:      getLinks(event),
	}

	client := pagerduty.NewClient()
	if len(config.alternateEndpoint) > 0 {
		client.AlternateEndpoint(config.alternateEndpoint)
	}

	eventResponse, err := client.ManageEventWithContext(ctx, &pdEvent)
	if err != nil {
		log.Printf("Warning Event Send failed, sending fallback event\n %s", err.Error())
		failPayload := pagerduty.V2Payload{
			Source:    event.Entity.Name,
			Component: event.Check.Name,
			Severity:  severity,
			Summary:   summary,
			Details:   "Original payload had an error, maybe due to event length. PagerDuty Events must be less than 512KB",
		}
		failEvent := pagerduty.V2Event{
			RoutingKey: token,
			Action:     action,
			Payload:    &failPayload,
			DedupKey:   dedupKey,
		}

		failResponse, err := client.ManageEventWithContext(ctx, &failEvent)
		if err != nil {
			return err
		}
		// FUTURE send to AH
		log.Printf(
			"Failback event (%s) submitted to PagerDuty, Status: %s, Dedup Key: %s, Message: %s", action,
			failResponse.Status, failResponse.DedupKey, failResponse.Message,
		)
	}

	// FUTURE send to AH
	log.Printf(
		"Event (%s) submitted to PagerDuty, Status: %s, Dedup Key: %s, Message: %s", action, eventResponse.Status,
		eventResponse.DedupKey, eventResponse.Message,
	)
	return nil
}

func getPagerDutyDedupKey(event *corev2.Event) (string, error) {
	return templates.EvalTemplate("dedupKey", config.dedupKeyTemplate, event)
}

func getPagerDutySeverity(event *corev2.Event, statusMapJSON string) (string, error) {
	var statusMap map[uint32]string
	var err error

	if len(statusMapJSON) > 0 {
		statusMap, err = parseStatusMap(statusMapJSON)
		if err != nil {
			return "", err
		}
	}

	if len(statusMap) > 0 {
		status := event.Check.Status
		severity := statusMap[status]
		if len(severity) > 0 {
			return severity, nil
		}
	}

	// Default to these values is no status map is found
	severity := "warning"
	if event.Check.Status < 3 {
		severities := []string{"info", "warning", "critical"}
		severity = severities[event.Check.Status]
	}

	return severity, nil
}

func parseStatusMap(statusMapJSON string) (map[uint32]string, error) {
	validPagerDutySeverities := map[string]bool{"info": true, "critical": true, "warning": true, "error": true}

	statusMap := eventStatusMap{}
	err := json.Unmarshal([]byte(statusMapJSON), &statusMap)
	if err != nil {
		return nil, err
	}

	// Reverse the map to key it on the status
	statusToSeverityMap := map[uint32]string{}
	for severity, statuses := range statusMap {
		if !validPagerDutySeverities[severity] {
			return nil, fmt.Errorf("invalid pagerduty severity: %s", severity)
		}
		for i := range statuses {
			statusToSeverityMap[statuses[i]] = severity
		}
	}

	return statusToSeverityMap, nil
}

func getSummary(event *corev2.Event) (string, error) {
	summary, err := templates.EvalTemplate("summary", config.summaryTemplate, event)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate template %s: %v", config.summaryTemplate, err)
	}
	// "The maximum permitted length of this property is 1024 characters."
	if len(summary) > 1024 {
		summary = summary[:1024]
	}
	log.Printf("Incident Summary: %s", summary)
	return summary, nil
}

func getTimestamp(event *corev2.Event) string {
	timestamp := ""
	if config.useEventTimestamp {
		timestamp = time.Unix(event.Timestamp, 0).Format(time.RFC3339)
	}

	return timestamp
}

func getGroup(event *corev2.Event) (string, error) {
	var (
		group string
		err   error
	)

	if len(config.groupTemplate) > 0 {
		group, err = templates.EvalTemplate("group", config.groupTemplate, event)
		if err != nil {
			return "", fmt.Errorf("failued to evaluate template %s: %v", config.groupTemplate, err)
		}
	} else {
		group = ""
	}

	return group, nil
}

func getComponent(event *corev2.Event) (string, error) {
	var (
		component string
		err       error
	)

	if len(config.componentTemplate) > 0 {
		component, err = templates.EvalTemplate("component", config.componentTemplate, event)
		if err != nil {
			return "", fmt.Errorf("failued to evaluate template %s: %v", config.componentTemplate, err)
		}
	} else {
		component = event.Check.Name
	}

	return component, nil
}

func getClass(event *corev2.Event) (string, error) {
	var (
		class string
		err   error
	)

	if len(config.classTemplate) > 0 {
		class, err = templates.EvalTemplate("class", config.classTemplate, event)
		if err != nil {
			return "", fmt.Errorf("failued to evaluate template %s: %v", config.classTemplate, err)
		}
	} else {
		class = ""
	}

	return class, nil
}

func getDetails(event *corev2.Event) (details interface{}, err error) {
	if len(config.detailsTemplate) > 0 {
		detailsStr, err := templates.EvalTemplate("details", config.detailsTemplate, event)
		if err != nil {
			return "", fmt.Errorf("failed to evaluate template %s: %v", config.detailsTemplate, err)
		}

		details = detailsStr
		if config.detailsFormat == jsonDetailsFormat.String() {
			var msgMap interface{}
			err = json.Unmarshal([]byte(detailsStr), &msgMap)
			if err != nil {
				return "", fmt.Errorf("failed to unmarshal json details: %v", err)
			}
			details = msgMap
		}
	} else {
		details = event
	}
	return details, nil
}

func getClientUrl(event *corev2.Event) string {
	if config.sensuBaseUrl == "" {
		return ""
	}

	return fmt.Sprintf(
		"%s/c/~/n/%s/events/%s/%s",
		strings.TrimSuffix(config.sensuBaseUrl, "/"),
		event.Namespace,
		event.Entity.Name,
		event.Check.Name,
	)
}

type Link struct {
	Text string `json:"text"`
	Href string `json:"href"`
}

func isLink(s string) bool {
	_, err := url.ParseRequestURI(s)

	return err == nil
}

func getLinks(event *corev2.Event) []interface{} {
	links := make([]interface{}, 0, len(event.Check.Annotations))

	if !config.linkAnnotations {
		return links
	}

	for key, value := range event.Check.Annotations {
		if isLink(value) {
			links = append(
				links, Link{
					Text: fmt.Sprintf("check %s", key),
					Href: value,
				},
			)
		}
	}

	for key, value := range event.Entity.Annotations {
		if isLink(value) {
			links = append(
				links, Link{
					Text: fmt.Sprintf("entity %s", key),
					Href: value,
				},
			)
		}
	}

	return links
}
