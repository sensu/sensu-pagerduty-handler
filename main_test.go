package main

import (
	corev2 "github.com/sensu/sensu-go/api/core/v2"
	"github.com/stretchr/testify/assert"
	"testing"
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
	statusMapJson := "{\"info\":[130,10],\"error\":[4]}"

	eventWithStatus.Check.Status = 10
	pagerDutySeverity, err := getPagerDutySeverity(&eventWithStatus, statusMapJson)
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
	statusMapJson := "{\"info\":[130,10],\"error\"[4]}"

	eventWithStatus.Check.Status = 2
	pagerDutySeverity, err := getPagerDutySeverity(&eventWithStatus, statusMapJson)
	assert.NotNil(t, err)
	assert.Empty(t, pagerDutySeverity)
}

func Test_GetPagerDutySeverity_StatusMapSeverityNotInMap(t *testing.T) {
	statusMapJson := "{\"info\":[130,10],\"error\":[4]}"

	eventWithStatus.Check.Status = 2
	pagerDutySeverity, err := getPagerDutySeverity(&eventWithStatus, statusMapJson)
	assert.Nil(t, err)
	assert.Equal(t, "critical", pagerDutySeverity)
}
