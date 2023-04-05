package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

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
		t.Run(
			test.name, func(t *testing.T) {
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
			},
		)
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
		t.Run(
			tt.name, func(t *testing.T) {
				config = tt.config

				err := checkArgs(tt.args.event)
				if (err != nil) != tt.wantErr {
					t.Errorf("checkArgs() error = %v, wantErr %v", err, tt.wantErr)
				}
				if err != nil && err.Error() != tt.wantErrMsg {
					t.Errorf("checkArgs() error msg = %v, want %v", err, tt.wantErrMsg)
				}
			},
		)
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
		t.Run(
			tt.name, func(t *testing.T) {
				if err := handleEvent(tt.args.event); (err != nil) != tt.wantErr {
					t.Errorf("handleEvent() error = %v, wantErr %v", err, tt.wantErr)
				}
			},
		)
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
		t.Run(
			tt.name, func(t *testing.T) {
				if err := handleEventContactRouting(tt.args.event); (err != nil) != tt.wantErr {
					t.Errorf("handleEventContactRouting() error = %v, wantErr %v", err, tt.wantErr)
				}
			},
		)
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
		t.Run(
			tt.name, func(t *testing.T) {
				if err := handleEventForContact(tt.args.event, tt.args.contact); (err != nil) != tt.wantErr {
					t.Errorf("handleEventForContact() error = %v, wantErr %v", err, tt.wantErr)
				}
			},
		)
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
		t.Run(
			tt.name, func(t *testing.T) {
				if got := getContacts(tt.args.event); !reflect.DeepEqual(got, tt.want) {
					t.Errorf("getContacts() = %v, want %v", got, tt.want)
				}
			},
		)
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
		t.Run(
			tt.name, func(t *testing.T) {
				got, err := getContactToken(tt.args.contact)
				if (err != nil) != tt.wantErr {
					t.Errorf("getContactToken() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if got != tt.want {
					t.Errorf("getContactToken() = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func Test_getTimestamp(t *testing.T) {
	originalConfig := config

	now := time.Now()

	type args struct {
		event *corev2.Event
	}
	tests := []struct {
		config HandlerConfig
		name   string
		args   args
		want   string
	}{
		{
			config: HandlerConfig{useEventTimestamp: false},
			name:   "dont-use-event-timestamp",
			args:   args{event: &eventWithStatus},
			want:   "",
		},
		{
			config: HandlerConfig{useEventTimestamp: true},
			name:   "use-event-timestamp",
			args:   args{event: &corev2.Event{Timestamp: now.Unix()}},
			want:   now.Format(time.RFC3339),
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				config = tt.config
				assert.Equalf(t, tt.want, getTimestamp(tt.args.event), "getTimestamp(%v)", tt.args.event)
			},
		)
		config = originalConfig
	}
}

func Test_getGroup(t *testing.T) {
	originalConfig := config

	type args struct {
		event *corev2.Event
	}
	tests := []struct {
		name    string
		config  HandlerConfig
		args    args
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:   "invalid template returns error",
			config: HandlerConfig{groupTemplate: "{{.BadVar}}"},
			args:   args{event: &eventWithStatus},
			want:   "",
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Error(t, err)
			},
		},
		{
			name: "empty result returned when no group template is defined",
			args: args{event: &eventWithStatus},
			want: "",
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Nil(t, err)
			},
		},
		{
			name:   "valid template returns correct result and no error",
			config: HandlerConfig{groupTemplate: "{{.Check.Labels.group}}"},
			args: args{
				event: &corev2.Event{
					Check: &corev2.Check{
						ObjectMeta: corev2.
							ObjectMeta{Labels: map[string]string{"group": "foobar"}},
					},
				},
			},
			want: "foobar",
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Nil(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				config = tt.config
				got, err := getGroup(tt.args.event)
				if !tt.wantErr(t, err, fmt.Sprintf("getGroup(%v)", tt.args.event)) {
					return
				}
				assert.Equalf(t, tt.want, got, "getGroup(%v)", tt.args.event)
			},
		)
		config = originalConfig
	}
}

func Test_isLink(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "valid http link is link",
			args: args{s: "http://foobar.net/one/two/three.html"},
			want: true,
		},
		{
			name: "valid http link is link",
			args: args{s: "https://foobar.net"},
			want: true,
		},
		{
			name: "invalid link is not link",
			args: args{s: "this is not a link"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				assert.Equalf(t, tt.want, isLink(tt.args.s), "isLink(%v)", tt.args.s)
			},
		)
	}
}

func Test_getComponent(t *testing.T) {
	originalConfig := config

	type args struct {
		event *corev2.Event
	}
	tests := []struct {
		name    string
		config  HandlerConfig
		args    args
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:   "bad template returns error",
			config: HandlerConfig{componentTemplate: "{{.BadVar}}"},
			args:   args{event: &eventWithStatus},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Error(t, err)
			},
			want: "",
		},
		{
			name:   "no result returned when component template not set",
			config: HandlerConfig{componentTemplate: ""},
			args:   args{event: &corev2.Event{Check: &corev2.Check{ObjectMeta: corev2.ObjectMeta{Name: "test-check"}}}},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Nil(t, err)
			},
			want: "test-check",
		},
		{
			name:   "good template",
			config: HandlerConfig{componentTemplate: "{{.Entity.Labels.component}}"},
			args: args{
				event: &corev2.Event{
					Entity: &corev2.Entity{
						ObjectMeta: corev2.ObjectMeta{Labels: map[string]string{"component": "component"}},
					},
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Nil(t, err)
			},
			want: "component",
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				config = tt.config
				got, err := getComponent(tt.args.event)
				if !tt.wantErr(t, err, fmt.Sprintf("getComponent(%v)", tt.args.event)) {
					return
				}
				assert.Equalf(t, tt.want, got, "getComponent(%v)", tt.args.event)
			},
		)
		config = originalConfig
	}
}

func Test_getClass(t *testing.T) {
	originalConfig := config

	type args struct {
		event *corev2.Event
	}
	tests := []struct {
		name    string
		config  HandlerConfig
		args    args
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:   "bad template",
			config: HandlerConfig{classTemplate: "{{.BadVar}}"},
			args:   args{event: &eventWithStatus},
			want:   "",
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Error(t, err)
			},
		},
		{
			name:   "no template",
			config: HandlerConfig{classTemplate: ""},
			args:   args{event: &eventWithStatus},
			want:   "",
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Nil(t, err)
			},
		},
		{
			name:   "good template",
			config: HandlerConfig{classTemplate: "{{.Check.Name}}"},
			args: args{
				event: &corev2.Event{
					Check: &corev2.Check{
						ObjectMeta: corev2.
							ObjectMeta{Name: "some-check"},
					},
				},
			},
			want: "some-check",
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Nil(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				config = tt.config
				got, err := getClass(tt.args.event)
				if !tt.wantErr(t, err, fmt.Sprintf("getClass(%v)", tt.args.event)) {
					return
				}
				assert.Equalf(t, tt.want, got, "getClass(%v)", tt.args.event)
			},
		)
		config = originalConfig
	}
}

func Test_getClientUrl(t *testing.T) {
	originalConfig := config
	type args struct {
		event *corev2.Event
	}
	tests := []struct {
		name   string
		config HandlerConfig
		args   args
		want   string
	}{
		{
			name: "no base url defined",
			args: args{event: &eventWithStatus},
			want: "",
		},
		{
			name:   "base url defined",
			config: HandlerConfig{sensuBaseUrl: "https://test-sensu.some-company.com/"},
			args: args{
				event: &corev2.Event{
					ObjectMeta: corev2.ObjectMeta{Namespace: "test-namespace"},
					Entity:     &corev2.Entity{ObjectMeta: corev2.ObjectMeta{Name: "test-entity"}},
					Check:      &corev2.Check{ObjectMeta: corev2.ObjectMeta{Name: "test-check"}},
				},
			},
			want: "https://test-sensu.some-company.com/c/~/n/test-namespace/events/test-entity/test-check",
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				config = tt.config
				assert.Equalf(t, tt.want, getClientUrl(tt.args.event), "getClientUrl(%v)", tt.args.event)
			},
		)
		config = originalConfig
	}
}

func Test_getLinks(t *testing.T) {
	originalConfig := config
	type args struct {
		event *corev2.Event
	}
	tests := []struct {
		name   string
		config HandlerConfig
		args   args
		want   []interface{}
	}{
		{
			name: "no links",
			args: args{event: &eventWithStatus},
			want: []interface{}{},
		},
		{
			name:   "check and entity links",
			config: HandlerConfig{linkAnnotations: true},
			args: args{
				event: &corev2.Event{
					Check: &corev2.Check{
						ObjectMeta: corev2.
							ObjectMeta{
							Annotations: map[string]string{
								"link":       "https://123.foobar.com/somepage",
								"not-a-link": "nolink",
							},
						},
					},
					Entity: &corev2.Entity{
						ObjectMeta: corev2.ObjectMeta{
							Annotations: map[string]string{
								"link":       "https://123.foobar.com/somepage",
								"not-a-link": "nolink",
							},
						},
					},
				},
			},
			want: []interface{}{
				Link{
					Text: "check link",
					Href: "https://123.foobar.com/somepage",
				},
				Link{
					Text: "entity link",
					Href: "https://123.foobar.com/somepage",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				config = tt.config
				assert.Equalf(t, tt.want, getLinks(tt.args.event), "getLinks(%v)", tt.args.event)
			},
		)
		config = originalConfig
	}
}
