package logs

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
)

func AttachZerologToOTel() {
	loggerProvider := global.GetLoggerProvider()
	otelLogger := loggerProvider.Logger("mxl-zerolog-otel-bridge")

	// Add a custom Zerolog hook to forward logs
	// User configuration should come first  on this
	logger := func() *zerolog.Logger {

		// logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		logger := log.Logger

		// Hook to forward logs to OpenTelemetry

		// Unfortunately, zerolog does not allow looking into internal fields. so custom key-values are lost. e.g errors set with log.Error().Err(err)
		logger = logger.Hook(zerolog.HookFunc(func(e *zerolog.Event, level zerolog.Level, msg string) {

			if !e.Enabled() {
				return
			}

			r := otellog.Record{}

			r.SetSeverity(convertLevel(level))
			r.SetBody(otellog.StringValue(msg))
			r.SetSeverityText(level.String())
			r.AddAttributes(getZerologAttrs(e)...)

			otelLogger.Emit(e.GetCtx(), r)
		}))

		return &logger
	}()

	log.Logger = *logger

}
func convertLevel(level zerolog.Level) otellog.Severity {
	switch level {
	case zerolog.DebugLevel:
		return otellog.SeverityDebug
	case zerolog.InfoLevel:
		return otellog.SeverityInfo
	case zerolog.WarnLevel:
		return otellog.SeverityWarn
	case zerolog.ErrorLevel:
		return otellog.SeverityError
	case zerolog.PanicLevel:
		return otellog.SeverityFatal1
	case zerolog.FatalLevel:
		return otellog.SeverityFatal2
	default:
		return otellog.SeverityUndefined
	}
}

func getZerologAttrs(e *zerolog.Event) []otellog.KeyValue {
	customProps := make(map[string]any)

	// create a string that appends } to the end of the buf variable you access via reflection
	ev := fmt.Sprintf("%s}", reflect.ValueOf(e).Elem().FieldByName("buf"))
	_ = json.Unmarshal([]byte(ev), &customProps)

	var attributes []otellog.KeyValue

	for k, v := range customProps {
		attributes = append(attributes, zerologToOtelAttr(k, v)...)
	}

	return attributes
}

func zerologToOtelAttr(key string, value any) []otellog.KeyValue {
	switch value := value.(type) {
	case bool:
		return []otellog.KeyValue{otellog.Bool(key, value)}
		// Number information is lost when we're converting to byte to interface{}, let's recover it
	case float64:
		if _, frac := math.Modf(value); frac == 0.0 {
			return []otellog.KeyValue{otellog.Int64(key, int64(value))}
		} else {
			return []otellog.KeyValue{otellog.Float64(key, value)}
		}
	case string:
		return []otellog.KeyValue{otellog.String(key, value)}
	case []interface{}:
		var result []otellog.KeyValue
		for _, v := range value {
			// recursively call otelAttribute to handle nested arrays
			result = append(result, zerologToOtelAttr(key, v)...)
		}
		return result
	}
	// Default case
	return []otellog.KeyValue{otellog.String(key, fmt.Sprintf("%v", value))}
}
