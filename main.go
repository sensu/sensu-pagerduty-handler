package main

import (
	"fmt"
	"github.com/PagerDuty/go-pagerduty"
	corev2 "github.com/sensu/sensu-go/api/core/v2"
	"github.com/sensu/sensu-plugins-go-library/sensu"
	"github.com/sensu/sensu-plugins-go-library/templates"
)

type HandlerConfig struct {
	sensu.PluginConfig
	authToken        string
	dedupKey         string
	dedupKeyTemplate string
}

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
	severity := "warning"

	if event.Check.Status < 3 {
		severities := []string{"info", "warning", "critical"}
		severity = severities[event.Check.Status]
	}

	summary := fmt.Sprintf("%s/%s : %s", event.Entity.Name, event.Check.Name, event.Check.Output)

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
