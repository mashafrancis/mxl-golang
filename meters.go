package mxlgolang

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

func newMeterProvider(ctx context.Context, cfg *Config, resources *resource.Resource) (*metric.MeterProvider, error) {

	var metricExporter metric.Exporter

	if cfg.useHTTP {
		e, err := httpMetricExporter(ctx, cfg)
		if err != nil {
			return nil, err
		}
		metricExporter = e
	} else {
		e, err := grpcMetricExporter(ctx, cfg)
		if err != nil {
			return nil, err
		}
		metricExporter = e
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter,
			metric.WithInterval(1*time.Minute))),
		metric.WithResource(resources),
	)
	return meterProvider, nil
}

func httpMetricExporter(ctx context.Context, cfg *Config) (metric.Exporter, error) {
	options := []otlpmetrichttp.Option{
		otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression),
		otlpmetrichttp.WithEndpointURL(cfg.oTLPEndpoint),
		otlpmetrichttp.WithURLPath("/v1/metrics"),
	}

	if cfg.insecure {
		options = append(options, otlpmetrichttp.WithInsecure())
	}

	if len(cfg.headers) > 0 {
		options = append(options, otlpmetrichttp.WithHeaders(cfg.headers))
	}

	exporter, err := otlpmetrichttp.New(ctx, options...)
	if err != nil {
		return nil, joinErrors(err, "failed to setup http metric exporter")
	}

	return exporter, nil
}

func grpcMetricExporter(ctx context.Context, cfg *Config) (metric.Exporter, error) {
	options := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithCompressor("gzip"),
		otlpmetricgrpc.WithEndpointURL(cfg.oTLPEndpoint),
	}

	if cfg.insecure {
		options = append(options, otlpmetricgrpc.WithInsecure())
	}
	if len(cfg.headers) > 0 {
		options = append(options, otlpmetricgrpc.WithHeaders(cfg.headers))
	}

	exporter, err := otlpmetricgrpc.New(ctx, options...)
	if err != nil {
		return nil, joinErrors(err, "failed to setup grpc metric exporter")
	}

	return exporter, nil
}
