package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/sensu/sensu-go/types"
	"github.com/spf13/cobra"
)

var (
	authToken string
	stdin     *os.File
)

func main() {
	rootCmd := configureRootCommand()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func configureRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sensu-pagerduty-handler",
		Short: "The Sensu Go PagerDuty handler for incident management",
		RunE:  run,
	}

	/*
	   Security Sensitive flags
	     - default to using envvar value
	     - do not mark as required
	     - manually test for empty value
	*/
	cmd.Flags().StringVarP(&authToken,
		"token",
		"t",
		os.Getenv("PAGERDUTY_TOKEN"),
		"The PagerDuty V2 API authentication token, use default from PAGERDUTY_TOKEN env var")

	return cmd
}

func run(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		_ = cmd.Help()
		return fmt.Errorf("invalid argument(s) received")
	}

	if authToken == "" {
		_ = cmd.Help()
		return fmt.Errorf("authentication token is empty")

	}

	if stdin == nil {
		stdin = os.Stdin
	}

	eventJSON, err := ioutil.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("failed to read stdin: %s", err)
	}

	event := &types.Event{}
	err = json.Unmarshal(eventJSON, event)
	if err != nil {
		return fmt.Errorf("failed to unmarshal stdin data: %s", err)
	}

	if err = event.Validate(); err != nil {
		return fmt.Errorf("failed to validate event: %s", err)
	}

	if !event.HasCheck() {
		return fmt.Errorf("event does not contain check")
	}

	return manageIncident(event)
}

func manageIncident(event *types.Event) error {
	severity := "warning"

	if event.Check.Status < 3 {
		severities := []string{"info", "warning", "critical"}
		severity = severities[event.Check.Status]
	}

	pdPayload := pagerduty.V2Payload{
		Source:    event.Entity.Name,
		Component: event.Check.Name,
		Severity:  severity,
		Summary:   event.Check.Output,
		Details:   event,
	}

	action := "trigger"

	if event.Check.Status == 0 {
		action = "resolve"
	}

	dedupKey := fmt.Sprintf("%s-%s", event.Entity.Name, event.Check.Name)

	pdEvent := pagerduty.V2Event{
		RoutingKey: authToken,
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
