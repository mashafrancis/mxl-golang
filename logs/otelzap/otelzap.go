package otelzap

import (
	"github.com/mashafrancis/mxl-golang/tracker"
	"go.opentelemetry.io/contrib/bridges/otelzap"
	otellog "go.opentelemetry.io/otel/sdk/log"
)

const loggerName = "mxlzap"
const MXLTraceID = "traceId"
const MXLSpanID = "spanId"

type config struct {
	provider *otellog.LoggerProvider
	name     string
}

type Option interface {
	apply(config) config
}

type optFunc func(config) config

func (f optFunc) apply(c config) config { return f(c) }

func WithName(name string) Option {
	return optFunc(func(c config) config {
		c.name = name
		return c
	})
}

func newConfig(cfg *tracker.Config, options []Option) config {
	var c config
	for _, opt := range options {
		c = opt.apply(c)
	}

	if c.name == "" {
		c.name = loggerName
	}
	if c.provider == nil {
		c.provider = cfg.Lp
	}

	return c
}

func NewMXLOTelCore(config *tracker.Config, options ...Option) *otelzap.Core {
	cfg := newConfig(config, options)
	return otelzap.NewCore(cfg.name, otelzap.WithLoggerProvider(cfg.provider))
}
