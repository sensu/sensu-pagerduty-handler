package main

import (
	"encoding/json"

	"os"
	"testing"

	corev2 "github.com/sensu/sensu-go/api/core/v2"
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
	json := "{\"info\":[130,10],\"error\":[4]}"

	statusMap, err := parseStatusMap(json)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(statusMap))
	assert.Equal(t, "info", statusMap[130])
	assert.Equal(t, "info", statusMap[10])
	assert.Equal(t, "error", statusMap[4])
}

func Test_ParseStatusMap_EmptyStatus(t *testing.T) {
	json := "{\"info\":[130,10],\"error\":[]}"

	statusMap, err := parseStatusMap(json)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(statusMap))
	assert.Equal(t, "info", statusMap[130])
	assert.Equal(t, "info", statusMap[10])
	assert.Equal(t, "", statusMap[4])
}

func Test_ParseStatusMap_InvalidJson(t *testing.T) {
	json := "{\"info\":[130,10],\"error:[]}"

	statusMap, err := parseStatusMap(json)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "unexpected end of JSON input")
	assert.Nil(t, statusMap)
}

func Test_ParseStatusMap_InvalidSeverity(t *testing.T) {
	json := "{\"info\":[130,10],\"invalid\":[4]}"

	statusMap, err := parseStatusMap(json)
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
	os.Setenv("test_team_test_suffix", "token_value")
	teamToken, err := getTeamToken()
	assert.Nil(t, err)
	assert.NotNil(t, teamToken)
	assert.Equal(t, "token_value", teamToken)
}

func Test_PagerIllegalTeamToken(t *testing.T) {
	config.teamName = "test-team"
	config.teamSuffix = "_test-a-suffix"
	os.Setenv("test_team_test_a_suffix", "token_value")
	teamToken, err := getTeamToken()
	assert.Nil(t, err)
	assert.NotNil(t, teamToken)
	assert.Equal(t, "token_value", teamToken)
}

func Test_PagerTeamNoSuffix(t *testing.T) {
	config.teamName = "test-team"
	config.teamSuffix = ""
	os.Setenv("test_team", "token_value")
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
}
