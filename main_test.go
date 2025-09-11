package main

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	corev2 "github.com/sensu/core/v2"
	"github.com/stretchr/testify/assert"
)

var (
	eventWithStatus = corev2.Event{
		Check: &corev2.Check{
			Status: 10,
		},
	}
)

func Test_ParseStatusMap_Success(t *testing.T) {
	statusJSON := "{\"info\":[130,10],\"error\":[4]}"

	statusMap, err := parseStatusMap(statusJSON)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(statusMap))
	assert.Equal(t, "info", statusMap[130])
	assert.Equal(t, "info", statusMap[10])
	assert.Equal(t, "error", statusMap[4])
}

func Test_ParseStatusMap_EmptyStatus(t *testing.T) {
	statusJSON := "{\"info\":[130,10],\"error\":[]}"

	statusMap, err := parseStatusMap(statusJSON)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(statusMap))
	assert.Equal(t, "info", statusMap[130])
	assert.Equal(t, "info", statusMap[10])
	assert.Equal(t, "", statusMap[4])
}

func Test_ParseStatusMap_InvalidJson(t *testing.T) {
	statusJSON := "{\"info\":[130,10],\"error:[]}"

	statusMap, err := parseStatusMap(statusJSON)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "unexpected end of JSON input")
	assert.Nil(t, statusMap)
}

func Test_ParseStatusMap_InvalidSeverity(t *testing.T) {
	statusJSON := "{\"info\":[130,10],\"invalid\":[4]}"

	statusMap, err := parseStatusMap(statusJSON)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "invalid pagerduty severity: invalid")
	assert.Nil(t, statusMap)
}

func Test_GetPagerDutySeverity_Success(t *testing.T) {
	statusMapJSON := "{\"info\":[130,10],\"error\":[4]}"

	eventWithStatus.Check.Status = 10
	pagerDutySeverity, err := getPagerDutySeverity(&eventWithStatus, statusMapJSON)
	assert.Nil(t, err)
	assert.Equal(t, "info", pagerDutySeverity)
}

func Test_GetPagerDutySeverity_NoStatusMapHighStatus(t *testing.T) {
	eventWithStatus.Check.Status = 3
	pagerDutySeverity, err := getPagerDutySeverity(&eventWithStatus, "")
	assert.Nil(t, err)
	assert.Equal(t, "warning", pagerDutySeverity)
}

func Test_GetPagerDutySeverity_NoStatusMapLowStatus(t *testing.T) {
	eventWithStatus.Check.Status = 2
	pagerDutySeverity, err := getPagerDutySeverity(&eventWithStatus, "")
	assert.Nil(t, err)
	assert.Equal(t, "critical", pagerDutySeverity)
}

func Test_GetPagerDutySeverity_InvalidStatusMap(t *testing.T) {
	statusMapJSON := "{\"info\":[130,10],\"error\"[4]}"

	eventWithStatus.Check.Status = 2
	pagerDutySeverity, err := getPagerDutySeverity(&eventWithStatus, statusMapJSON)
	assert.NotNil(t, err)
	assert.Empty(t, pagerDutySeverity)
}

func Test_GetPagerDutySeverity_StatusMapSeverityNotInMap(t *testing.T) {
	statusMapJSON := "{\"info\":[130,10],\"error\":[4]}"

	eventWithStatus.Check.Status = 2
	pagerDutySeverity, err := getPagerDutySeverity(&eventWithStatus, statusMapJSON)
	assert.Nil(t, err)
	assert.Equal(t, "critical", pagerDutySeverity)
}

func Test_GetPagerDutyDedupKey(t *testing.T) {
	event := corev2.FixtureEvent("foo", "bar")
	config.dedupKeyTemplate = "{{.Entity.Name}}-{{.Check.Name}}"

	dedupKey, err := getPagerDutyDedupKey(event)
	assert.Nil(t, err)
	assert.Equal(t, "foo-bar", dedupKey)
}

func Test_PagerTeamToken(t *testing.T) {
	config.teamName = "test_team"
	config.teamSuffix = "_test_suffix"
	_ = os.Setenv("test_team_test_suffix", "token_value")
	teamToken, err := getTeamToken()
	assert.Nil(t, err)
	assert.NotNil(t, teamToken)
	assert.Equal(t, "token_value", teamToken)
}

func Test_PagerIllegalTeamToken(t *testing.T) {
	config.teamName = "test-team"
	config.teamSuffix = "_test-a-suffix"
	_ = os.Setenv("test_team_test_a_suffix", "token_value")
	teamToken, err := getTeamToken()
	assert.Nil(t, err)
	assert.NotNil(t, teamToken)
	assert.Equal(t, "token_value", teamToken)
}

func Test_PagerTeamNoSuffix(t *testing.T) {
	config.teamName = "test-team"
	config.teamSuffix = ""
	_ = os.Setenv("test_team", "token_value")
	teamToken, err := getTeamToken()
	assert.Nil(t, err)
	assert.NotNil(t, teamToken)
	assert.Equal(t, "token_value", teamToken)
}

func Test_GetSummary(t *testing.T) {
	event := corev2.FixtureEvent("foo", "bar")
	config.summaryTemplate = "{{.Entity.Name}}-{{.Check.Name}}"

	summary, err := getSummary(event)
	assert.Nil(t, err)
	assert.Equal(t, "foo-bar", summary)
}

func Test_GetDetailsJSON(t *testing.T) {
	event := corev2.FixtureEvent("foo", "bar")
	config.detailsTemplate = ""

	details, err := getDetails(event)
	assert.Nil(t, err)
	b, err := json.Marshal(details)
	assert.Nil(t, err)
	j := &corev2.Event{}
	err = json.Unmarshal(b, &j)
	assert.Nil(t, err)
	assert.Equal(t, "foo", j.Entity.Name)
	assert.Equal(t, "bar", j.Check.Name)
}

func Test_GetDetailsTemplate(t *testing.T) {
	event := corev2.FixtureEvent("foo", "bar")
	config.detailsTemplate = "{{.Entity.Name}}-{{.Check.Name}}"

	details, err := getDetails(event)
	assert.Nil(t, err)
	assert.Equal(t, "foo-bar", details)

	// Test newline in check output with JSON details template
	config.detailsFormat = "json"
	config.detailsTemplate = `{"Output": {{ toJSON .Check.Output }}}`
	event.Check.Output = "bar\nxaz\n"
	details, err = getDetails(event)
	assert.Nil(t, err)
	assert.Equal(t, map[string]interface{}{"Output": "bar\nxaz\n"}, details)
}

func Test_GetDetailsObj(t *testing.T) {
	tests := []struct {
		name            string
		detailsTemplate string
		expectError     bool
	}{
		{
			name:            "valid-json",
			detailsTemplate: `{"entity": "{{.Entity.Name}}", "check":"{{.Check.Name}}", "namespace":"{{.Namespace}}", "id":"{{.GetUUID.String}}"}`,
			expectError:     false,
		},
		{
			name:            "invalid-json",
			detailsTemplate: `{"entity": "{{.Entity.Name}}"WHAT?}`,
			expectError:     true,
		},
	}
	event := corev2.FixtureEvent("entity-name", "check-name")
	config.detailsFormat = "json"

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config.detailsTemplate = test.detailsTemplate
			details, err := getDetails(event)
			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, details)
				detailMap, ok := details.(map[string]interface{})
				assert.True(t, ok)
				assert.Equal(t, "entity-name", detailMap["entity"])
				assert.Equal(t, "check-name", detailMap["check"])
				assert.Equal(t, "default", detailMap["namespace"])
				assert.Equal(t, event.GetUUID().String(), detailMap["id"])
			}
		})
	}
}

func Test_checkArgs(t *testing.T) {
	originalConfig := config
	type args struct {
		event *corev2.Event
	}
	tests := []struct {
		name       string
		config     HandlerConfig
		args       args
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "error when event has no check",
			args: args{
				event: func() *corev2.Event {
					event := corev2.FixtureEvent("foo", "bar")
					event.Check = nil
					return event
				}(),
			},
			wantErr:    true,
			wantErrMsg: "event does not contain check",
		},
		{
			name: "error when contacts have invalid characters",
			config: HandlerConfig{
				contactRouting: true,
			},
			args: args{
				event: func() *corev2.Event {
					event := corev2.FixtureEvent("foo", "bar")
					event.Annotations["contacts"] = "valid_contact,invalid-contact"
					return event
				}(),
			},
			wantErr:    true,
			wantErrMsg: "invalid contact syntax: invalid-contact",
		},
		{
			name: "no error with json details format",
			config: HandlerConfig{
				detailsFormat: "json",
				authToken:     "aaa",
			},
			args: args{
				event: func() *corev2.Event {
					event := corev2.FixtureEvent("foo", "bar")
					return event
				}(),
			},
			wantErr:    false,
			wantErrMsg: "",
		},
		{
			name: "no error with string details format",
			config: HandlerConfig{
				detailsFormat: "string",
				authToken:     "aaa",
			},
			args: args{
				event: func() *corev2.Event {
					event := corev2.FixtureEvent("foo", "bar")
					return event
				}(),
			},
			wantErr:    false,
			wantErrMsg: "",
		},
		{
			name: "no error with string details format",
			config: HandlerConfig{
				detailsFormat: "invalidformat",
				authToken:     "aaa",
			},
			args: args{
				event: func() *corev2.Event {
					event := corev2.FixtureEvent("foo", "bar")
					return event
				}(),
			},
			wantErr:    true,
			wantErrMsg: "invalid details format: invalidformat",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config = tt.config

			err := checkArgs(tt.args.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && err.Error() != tt.wantErrMsg {
				t.Errorf("checkArgs() error msg = %v, want %v", err, tt.wantErrMsg)
			}
		})
		config = originalConfig
	}
}

func Test_handleEvent(t *testing.T) {
	type args struct {
		event *corev2.Event
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := handleEvent(tt.args.event); (err != nil) != tt.wantErr {
				t.Errorf("handleEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_handleEventContactRouting(t *testing.T) {
	type args struct {
		event *corev2.Event
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := handleEventContactRouting(tt.args.event); (err != nil) != tt.wantErr {
				t.Errorf("handleEventContactRouting() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_handleEventForContact(t *testing.T) {
	type args struct {
		event   *corev2.Event
		contact string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := handleEventForContact(tt.args.event, tt.args.contact); (err != nil) != tt.wantErr {
				t.Errorf("handleEventForContact() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getContacts(t *testing.T) {
	type args struct {
		event *corev2.Event
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getContacts(tt.args.event); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getContacts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getContactToken(t *testing.T) {
	type args struct {
		contact string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getContactToken(tt.args.contact)
			if (err != nil) != tt.wantErr {
				t.Errorf("getContactToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getContactToken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getCustomFields(t *testing.T) {
	originalConfig := config
	defer func() { config = originalConfig }()

	event := corev2.FixtureEvent("test-entity", "test-check")
	event.Check.Output = "test output"
	event.Entity.Namespace = "default"

	tests := []struct {
		name           string
		config         HandlerConfig
		event          *corev2.Event
		wantErr        bool
		expectedFields map[string]interface{}
	}{
		{
			name: "empty custom field templates",
			config: HandlerConfig{
				customFieldTemplates: "",
			},
			event:          event,
			wantErr:        false,
			expectedFields: map[string]interface{}{},
		},
		{
			name: "single custom field template",
			config: HandlerConfig{
				customFieldTemplates: "check_output={{.Check.Output}}",
			},
			event:   event,
			wantErr: false,
			expectedFields: map[string]interface{}{
				"check_output": "test output",
			},
		},
		{
			name: "multiple custom field templates",
			config: HandlerConfig{
				customFieldTemplates: "check_output={{.Check.Output}};client={{.Entity.Name}};namespace={{.Entity.Namespace}}",
			},
			event:   event,
			wantErr: false,
			expectedFields: map[string]interface{}{
				"check_output": "test output",
				"client":       "test-entity",
				"namespace":    "default",
			},
		},
		{
			name: "custom field with complex template",
			config: HandlerConfig{
				customFieldTemplates: "client_url=https://sensugourl.com/c/~/n/{{.Entity.Namespace}}/events/{{.Entity.Name}}",
			},
			event:   event,
			wantErr: false,
			expectedFields: map[string]interface{}{
				"client_url": "https://sensugourl.com/c/~/n/default/events/test-entity",
			},
		},
		{
			name: "invalid template format - missing equals",
			config: HandlerConfig{
				customFieldTemplates: "invalid_template",
			},
			event:   event,
			wantErr: true,
		},
		{
			name: "invalid template format - empty key",
			config: HandlerConfig{
				customFieldTemplates: "={{.Check.Output}}",
			},
			event:   event,
			wantErr: true,
		},
		{
			name: "invalid template syntax",
			config: HandlerConfig{
				customFieldTemplates: "check_output={{.InvalidField}}",
			},
			event:   event,
			wantErr: true,
		},
		{
			name: "multiple templates with empty ones",
			config: HandlerConfig{
				customFieldTemplates: "check_output={{.Check.Output}};;client={{.Entity.Name}};",
			},
			event:   event,
			wantErr: false,
			expectedFields: map[string]interface{}{
				"check_output": "test output",
				"client":       "test-entity",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config = tt.config
			got, err := getCustomFields(tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("getCustomFields() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.expectedFields, got)
			}
		})
	}
}

func Test_mergeDetailsWithCustomFields(t *testing.T) {
	tests := []struct {
		name         string
		details      interface{}
		customFields map[string]interface{}
		expected     interface{}
	}{
		{
			name:         "no custom fields",
			details:      "test details",
			customFields: map[string]interface{}{},
			expected:     "test details",
		},
		{
			name:    "details is string, has custom fields",
			details: "test details",
			customFields: map[string]interface{}{
				"custom1": "value1",
				"custom2": "value2",
			},
			expected: map[string]interface{}{
				"details": "test details",
				"custom1": "value1",
				"custom2": "value2",
			},
		},
		{
			name: "details is map, has custom fields",
			details: map[string]interface{}{
				"existing": "value",
			},
			customFields: map[string]interface{}{
				"custom1": "value1",
				"custom2": "value2",
			},
			expected: map[string]interface{}{
				"existing": "value",
				"custom1":  "value1",
				"custom2":  "value2",
			},
		},
		{
			name: "details is map, custom fields override existing",
			details: map[string]interface{}{
				"existing": "original_value",
				"keep":     "keep_value",
			},
			customFields: map[string]interface{}{
				"existing": "override_value",
				"custom1":  "value1",
			},
			expected: map[string]interface{}{
				"existing": "override_value",
				"keep":     "keep_value",
				"custom1":  "value1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeDetailsWithCustomFields(tt.details, tt.customFields)
			assert.Equal(t, tt.expected, got)
		})
	}
}
