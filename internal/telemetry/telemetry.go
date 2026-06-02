package telemetry

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

// Init wires up OpenTelemetry metrics and tracing with simple exporters:
//   - Metrics: a Prometheus exporter, scraped from the /metrics HTTP endpoint.
//   - Tracing: a stdout exporter, so request spans are printed to the logs.
//
// It returns a cleanup function that flushes and shuts down both providers.
func Init(ctx context.Context) (func(), error) {
	// Metrics.
	metricExporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize prometheus exporter: %w", err)
	}

	meterProvider := metric.NewMeterProvider(metric.WithReader(metricExporter))
	otel.SetMeterProvider(meterProvider)

	// Tracing.
	traceExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize stdout trace exporter: %w", err)
	}

	tracerProvider := trace.NewTracerProvider(trace.WithBatcher(traceExporter))
	otel.SetTracerProvider(tracerProvider)

	cleanup := func() {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			slog.Error("failed to shutdown tracer provider", "error", err)
		}
		if err := meterProvider.Shutdown(ctx); err != nil {
			slog.Error("failed to shutdown meter provider", "error", err)
		}
	}

	return cleanup, nil
}
