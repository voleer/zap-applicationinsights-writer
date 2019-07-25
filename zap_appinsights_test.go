package zapappinsights

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Microsoft/ApplicationInsights-Go/appinsights"
	"github.com/Microsoft/ApplicationInsights-Go/appinsights/contracts"
	"github.com/stretchr/testify/require"
)

func TestWriteIntegration(t *testing.T) {
	appInsightsConfig := AppInsightsConfig{
		client:  appinsights.NewTelemetryClient("00000000-0000-0000-0000-000000000000"),
		filters: make(map[string]func(interface{}) interface{}),
	}

	message := "hello world"
	msg := fmt.Sprintf(`{"source": "test", "msg": "%s", "level": "Information"}`, message)

	// Act
	n, err := appInsightsConfig.Write([]byte(msg))

	// Assert
	require.Equal(t, len(message), n, "appInsightsConfig.Write length")
	require.NoError(t, err, "appInsightsConfig.Write")
}

func TestBuildTrace(t *testing.T) {
	message := "hello world"
	level := contracts.Information
	source := "test"
	msg := "{\"source\": \"" + source + "\", \"msg\": \"" + message + "\", \"level\":\"" + level.String() + "\"}"
	msgbyte := []byte(msg)
	var data map[string]interface{}
	json.Unmarshal(msgbyte, &data)

	// Act
	trace := BuildTrace(data)

	// Assert
	require.Equal(t, message, trace.Message, "trace.Message")
	require.Equalf(t, level, trace.SeverityLevel, "trace.SeverityLevel")
	require.Equalf(t, source, trace.BaseTelemetry.Properties["source"], "trace.Properties[\"source\"]")
}

func TestMinLogLevelEnabler_ValidData_ReturnExpectedEnabler(t *testing.T) {
	for i := 0; i < len(defaultLevels); i++ {
		minLevel := defaultLevels[i]
		minLogFilter := minLogLevelFilter(minLevel)

		t.Run(fmt.Sprintf("min level: %s", minLevel.String()), func(t *testing.T) {
			for j := 0; j < len(defaultLevels); j++ {
				// defaultLevels is sorted from highest to lowest, i.e. smaller index
				// means higher level
				level := defaultLevels[j]

				// Assert
				if j <= i {
					// True for higher or equal level
					require.True(t, minLogFilter(level), "minLogFilter(level)")
				} else {
					// False for lower level
					require.False(t, minLogFilter(level), "minLogFilter(level)")
				}
			}
		})
	}
}
