package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	"github.com/sensu-community/sensu-plugin-sdk/templates"
	corev2 "github.com/sensu/sensu-go/api/core/v2"
)

type HandlerConfig struct {
	sensu.PluginConfig
	authToken        string
	dedupKey         string
	dedupKeyTemplate string
	statusMapJson    string
}

type eventStatusMap map[string][]uint32

var (
	config = HandlerConfig{
		PluginConfig: sensu.PluginConfig{
			Name:  "sensu-pagerduty-handler",
			Short: "The Sensu Go PagerDuty handler for incident management",
		},
	}

	pagerDutyConfigOptions = []*sensu.PluginConfigOption{
		{
			Path:      "token",
			Env:       "PAGERDUTY_TOKEN",
			Argument:  "token",
			Shorthand: "t",
			Usage:     "The PagerDuty V2 API authentication token, use default from PAGERDUTY_TOKEN env var",
			Value:     &config.authToken,
			Default:   "",
		},
		{
			Path:      "dedup-key",
			Env:       "PAGERDUTY_DEDUP_KEY",
			Argument:  "dedup-key",
			Shorthand: "d",
			Usage:     "The Sensu event label specifying the PagerDuty V2 API deduplication key, use default from PAGERDUTY_DEDUP_KEY env var",
			Value:     &config.dedupKey,
			Default:   "",
		},
		{
			Path:      "dedup-key-template",
			Env:       "PAGERDUTY_DEDUP_KEY_TEMPLATE",
			Argument:  "dedup-key-template",
			Shorthand: "k",
			Usage:     "The PagerDuty V2 API deduplication key template, use default from PAGERDUTY_DEDUP_KEY_TEMPLATE env var",
			Value:     &config.dedupKeyTemplate,
			Default:   "",
		},
		{
			Path:      "status-map",
			Env:       "PAGERDUTY_STATUS_MAP",
			Argument:  "status-map",
			Shorthand: "s",
			Usage:     "The status map used to translate a Sensu check status to a PagerDuty severity, use default from PAGERDUTY_STATUS_MAP env var",
			Value:     &config.statusMapJson,
			Default:   "",
		},
	}
)

func main() {
	goHandler := sensu.NewGoHandler(&config.PluginConfig, pagerDutyConfigOptions, checkArgs, manageIncident)
	goHandler.Execute()
}

func checkArgs(event *corev2.Event) error {
	if !event.HasCheck() {
		return fmt.Errorf("event does not contain check")
	}
	return nil
}

func manageIncident(event *corev2.Event) error {
	severity, err := getPagerDutySeverity(event, config.statusMapJson)
	if err != nil {
		return err
	}
	log.Printf("Incident severity: %s", severity)

	summary := fmt.Sprintf("%s/%s : %s", event.Entity.Name, event.Check.Name, event.Check.Output)
	// "The maximum permitted length of this property is 1024 characters."
	if len(summary) > 1024 {
		summary = summary[:1024]
	}

	pdPayload := pagerduty.V2Payload{
		Source:    event.Entity.Name,
		Component: event.Check.Name,
		Severity:  severity,
		Summary:   summary,
		Details:   event,
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
		RoutingKey: config.authToken,
		Action:     action,
		Payload:    &pdPayload,
		DedupKey:   dedupKey,
	}

	_, err = pagerduty.ManageEvent(pdEvent)
	if err != nil {
		return err
	}

	return nil
}

// getPagerDutyDedupKey returns the PagerDuty deduplication key. The following priority is used to determine the
// deduplication key.
// 1. --dedup-key  --  specifies the entity label containing the key
// 2. --dedup-key-template  --  a template containing the values
// 3. the default value including the entity and check names
func getPagerDutyDedupKey(event *corev2.Event) (string, error) {
	if len(config.dedupKey) > 0 {
		labelValue := event.Entity.Labels[config.dedupKey]
		if len(labelValue) > 0 {
			return labelValue, nil
		} else {
			return "", fmt.Errorf("no deduplication key value in label %s", config.dedupKey)
		}
	}

	if len(config.dedupKeyTemplate) > 0 {
		return templates.EvalTemplate("dedupKey", config.dedupKeyTemplate, event)
	} else {
		return fmt.Sprintf("%s-%s", event.Entity.Name, event.Check.Name), nil
	}
}

func getPagerDutySeverity(event *corev2.Event, statusMapJson string) (string, error) {
	var statusMap map[uint32]string
	var err error

	if len(statusMapJson) > 0 {
		statusMap, err = parseStatusMap(statusMapJson)
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

func parseStatusMap(statusMapJson string) (map[uint32]string, error) {
	validPagerDutySeverities := map[string]bool{"info": true, "critical": true, "warning": true, "error": true}

	statusMap := eventStatusMap{}
	err := json.Unmarshal([]byte(statusMapJson), &statusMap)
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
			statusToSeverityMap[uint32(statuses[i])] = severity
		}
	}

	return statusToSeverityMap, nil
}
