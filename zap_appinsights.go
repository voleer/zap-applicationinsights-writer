package zapappinsights

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/Microsoft/ApplicationInsights-Go/appinsights"
	"github.com/Microsoft/ApplicationInsights-Go/appinsights/contracts"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type AppInsightsConfig struct {
	client appinsights.TelemetryClient

	async bool
	//levels  []zapcore.Level
	filters map[string]func(interface{}) interface{}
}

// Config for application insights
type Config struct {
	InstrumentationKey string
	EndpointURL        string
	MinLogLevel        zapcore.Level
	MaxBatchSize       int
	MaxBatchInterval   time.Duration
}

func NewAppInsightsCore(conf Config, fs ...zapcore.Field) (zapcore.Core, zap.Option, error) {
	// Prepare the zap writer
	logLevelFilter := zap.LevelEnablerFunc(minLogLevelFilter(conf.MinLogLevel))

	config := zap.NewProductionEncoderConfig()
	config.EncodeLevel = appInsightsLevelEncoder
	jsonEncode := zapcore.NewJSONEncoder(config)

	option := zap.Fields(append(fs)...)

	// Prepare the AppInsights client
	if conf.InstrumentationKey == "" {
		return nil, nil, fmt.Errorf("InstrumentationKey is required and missing from configuration")
	}
	telemetryConf := appinsights.NewTelemetryConfiguration(conf.InstrumentationKey)
	if conf.MaxBatchSize > 0 {
		telemetryConf.MaxBatchSize = conf.MaxBatchSize
	}
	if conf.MaxBatchInterval > 0 {
		telemetryConf.MaxBatchInterval = conf.MaxBatchInterval
	}
	if conf.EndpointURL != "" {
		telemetryConf.EndpointUrl = conf.EndpointURL
	}
	telemetryClient := appinsights.NewTelemetryClientFromConfig(telemetryConf)

	appInsightsConfig := AppInsightsConfig{
		client: telemetryClient,
		//levels:  defaultLevels,
		filters: make(map[string]func(interface{}) interface{}),
	}
	syncer := New(&appInsightsConfig)

	return zapcore.NewCore(jsonEncode, syncer, logLevelFilter), option, nil
}

var defaultLevels = []zapcore.Level{
	zapcore.FatalLevel,
	zapcore.PanicLevel,
	zapcore.DPanicLevel,
	zapcore.ErrorLevel,
	zapcore.WarnLevel,
	zapcore.InfoLevel,
	zapcore.DebugLevel,
}

var levelMap = map[string]contracts.SeverityLevel{
	"Critical":    appinsights.Critical,
	"Error":       appinsights.Error,
	"Warning":     appinsights.Warning,
	"Information": appinsights.Information,
	"Verbose":     appinsights.Verbose,
}

// appInsightsLevelEncoder maps the zap log levels to Application Insights levels.
func appInsightsLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	switch l {
	case zapcore.DebugLevel:
		enc.AppendString(contracts.Verbose.String())
	case zapcore.InfoLevel:
		enc.AppendString(contracts.Information.String())
	case zapcore.WarnLevel:
		enc.AppendString(contracts.Warning.String())
	case zapcore.ErrorLevel:
		enc.AppendString(contracts.Error.String())
	case zapcore.DPanicLevel:
		enc.AppendString(contracts.Critical.String())
	case zapcore.PanicLevel:
		enc.AppendString(contracts.Critical.String())
	case zapcore.FatalLevel:
		enc.AppendString(contracts.Critical.String())
	}
}

// New returns an implementation of ZapWriteSyncer which should be compatible with zap.WriteSyncer
func New(appInsightsConfig *AppInsightsConfig) zapcore.WriteSyncer {
	return appInsightsConfig
}

// Sync flushes the buffered App Insights data (if any).
func (appInsightsConfig *AppInsightsConfig) Sync() error {
	if channel := appInsightsConfig.client.Channel(); channel != nil {
		channel.Flush()
	}
	return nil
}

func BuildTrace(data map[string]interface{}) *appinsights.TraceTelemetry {
	message := data["msg"].(string)
	level := levelMap[data["level"].(string)]
	trace := appinsights.NewTraceTelemetry(message, level)

	for k, v := range data {
		switch k {
		case "msg", "level":
			break
		default:
			// Currently AppInsights Go SDK only support custom dimension (filter with string values)
			switch val := v.(type) {
			case int:
				trace.BaseTelemetry.Properties[k] = string(val)
			case string:
				trace.BaseTelemetry.Properties[k] = val
			case float64:
				trace.BaseTelemetry.Properties[k] = strconv.FormatFloat(val, 'f', 6, 64)
			}
		}
	}

	return trace
}

func (appInsightsConfig *AppInsightsConfig) Write(p []byte) (int, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(p, &data); err != nil {
		panic(err)
	}

	trace := BuildTrace(data)
	go appInsightsConfig.client.Track(trace)

	return len(trace.Message), nil
}

// minLogLevelFilter creates a log level filter func that returns true if the log level
// is higher than the minimum log level, and false otherwise.
func minLogLevelFilter(minLevel zapcore.Level) func(zapcore.Level) bool {
	return func(lvl zapcore.Level) bool {
		return minLevel.Enabled(lvl)
	}
}
