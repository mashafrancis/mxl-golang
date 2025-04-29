package mxlgolang

import (
	"context"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
)

func newTraceProvider(ctx context.Context, config *Config, resources *resource.Resource) (*trace.TracerProvider, error) {
	var exporter trace.SpanExporter

	if config.useHTTP {
		e, err := newHTTPTraceExporter(ctx, config, resources)
		if err != nil {
			return nil, err
		}

		exporter = e

	} else {
		e, err := newGRPCTraceExporter(ctx, config, resources)

		if err != nil {
			return nil, err
		}
		exporter = e
	}
	tracerProvider := trace.NewTracerProvider(

		trace.WithBatcher(exporter),
		trace.WithResource(resources),
	)
	return tracerProvider, nil

}

func newHTTPTraceExporter(ctx context.Context, config *Config, resources *resource.Resource) (trace.SpanExporter, error) {

	options := []otlptracehttp.Option{
		otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
		otlptracehttp.WithEndpointURL(config.oTLPEndpoint),
	}

	if config.insecure {
		options = append(options, otlptracehttp.WithInsecure())
	}
	if len(config.headers) > 0 {

		options = append(options, otlptracehttp.WithHeaders(config.headers))
	}
	exporter, err := otlptracehttp.New(ctx, options...)

	if err != nil {
		return nil, joinErrors(err, "failed to setup http trace exporter")
	}
	return exporter, nil
}

func newGRPCTraceExporter(ctx context.Context, cfg *Config, resources *resource.Resource) (trace.SpanExporter, error) {

	options := []otlptracegrpc.Option{
		otlptracegrpc.WithCompressor("gzip"),
		otlptracegrpc.WithEndpointURL(cfg.oTLPEndpoint),
	}

	if len(cfg.headers) > 0 {
		options = append(options, otlptracegrpc.WithHeaders(cfg.headers))
	}

	if cfg.insecure {
		options = append(options, otlptracegrpc.WithInsecure())
	}

	traceExporter, err := otlptracegrpc.New(context.Background(), options...)

	if err != nil {
		return nil, joinErrors(err, "failed to setup grpc trace exporter")
	}
	return traceExporter, nil
}
