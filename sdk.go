package mxlgolang

import (
	"context"
	"errors"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
)

type Cleanup func(ctx context.Context) error

func Init(ctx context.Context, cfg *Config) (Cleanup, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}

	if cfg.disableTraces && cfg.disableMetrics && cfg.disableLogs {
		slog.Warn("otel setup skipped as traces, metrics and logs are disabled")
		return nil, nil
	}

	shutdownFuncs := []Cleanup{}
	var setupError error

	var shutdown = func(ctx context.Context) error {
		var err error
		for _, cleanupFn := range shutdownFuncs {

			err = errors.Join(err, cleanupFn(ctx))
		}
		shutdownFuncs = nil

		return err
	}
	//Ensure errors are recorded and shutdown called
	setupCleaner := func(err error) {
		setupError = errors.Join(err, shutdown(ctx))
	}

	// Propagators
	// Attach tracing info to the various transports when communicating with other services

	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	resources, err := resource.New(ctx,
		resource.WithAttributes(
			cfg.attributes...,
		),
	)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to create resources with attributes"))
	}

	if !cfg.disableTraces {
		var tracerProvider *trace.TracerProvider
		tracerProvider, err = newTraceProvider(ctx, cfg, resources)
		if err != nil {
			setupCleaner(err)
			return nil, setupError
		}

		shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
		otel.SetTracerProvider(tracerProvider)

	}

	if !cfg.disableLogs {

		loggerprovider, err := newLogProvider(ctx, cfg, resources)
		if err != nil {
			setupCleaner(err)
			return nil, setupError
		}
		shutdownFuncs = append(shutdownFuncs, loggerprovider.Shutdown)
		global.SetLoggerProvider(loggerprovider)
	}

	if !cfg.disableMetrics {
		meterProvider, err := newMeterProvider(ctx, cfg, resources)
		if err != nil {
			setupCleaner(err)
			return nil, setupError
		}

		shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
		otel.SetMeterProvider(meterProvider)
	}

	return shutdown, setupError
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}
