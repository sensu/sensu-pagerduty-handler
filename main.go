package main

import (
	"fmt"

	"github.com/PagerDuty/go-pagerduty"
	corev2 "github.com/sensu/sensu-go/api/core/v2"
	"github.com/sensu/sensu-plugins-go-library/sensu"
)

type HandlerConfig struct {
	sensu.PluginConfig
	authToken string
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

	dedupKey := fmt.Sprintf("%s-%s", event.Entity.Name, event.Check.Name)

	pdEvent := pagerduty.V2Event{
		RoutingKey: config.authToken,
		Action:     action,
		Payload:    &pdPayload,
		DedupKey:   dedupKey,
	}

	_, err := pagerduty.ManageEvent(pdEvent)
	if err != nil {
		return err
	}

	return nil
}
