package mxlgolang

import (
	"context"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
)

func newLogProvider(ctx context.Context, cfg *Config, resources *resource.Resource) (*log.LoggerProvider, error) {

	var exporter log.Exporter

	if cfg.useHTTP {
		e, err := httpLogExporter(ctx, cfg)
		if err != nil {
			return nil, err
		}
		exporter = e
	} else {
		e, err := grpcLogExporter(ctx, cfg)
		if err != nil {
			return nil, err
		}
		exporter = e
	}

	logProvider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(exporter)),
		log.WithResource(resources),
	)

	return logProvider, nil

}

func httpLogExporter(ctx context.Context, cfg *Config) (log.Exporter, error) {

	options := []otlploghttp.Option{
		otlploghttp.WithCompression(otlploghttp.GzipCompression),
		otlploghttp.WithEndpointURL(cfg.oTLPEndpoint),
		otlploghttp.WithURLPath("/v1/logs"),
	}

	if len(cfg.headers) > 0 {
		options = append(options, otlploghttp.WithHeaders(cfg.headers))
	}

	if cfg.insecure {
		options = append(options, otlploghttp.WithInsecure())
	}

	exporter, err := otlploghttp.New(ctx, options...)
	if err != nil {
		return nil, joinErrors(err, "failed to setup http log exporter")
	}

	return exporter, nil

}

func grpcLogExporter(ctx context.Context, cfg *Config) (log.Exporter, error) {

	options := []otlploggrpc.Option{
		otlploggrpc.WithCompressor("gzip"),
		otlploggrpc.WithEndpointURL(cfg.oTLPEndpoint),
	}

	if cfg.insecure {
		options = append(options, otlploggrpc.WithInsecure())
	}

	if len(cfg.headers) > 0 {
		options = append(options, otlploggrpc.WithHeaders(cfg.headers))
	}

	exporter, err := otlploggrpc.New(ctx, options...)
	if err != nil {
		return nil, joinErrors(err, "failed to setup grpc log exporter")
	}

	return exporter, nil

}
