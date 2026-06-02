package httpx

import (
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter       = otel.Meter("github.com/acai-travel/tech-challenge/internal/httpx")
	requests, _ = meter.Int64Counter("http_requests_total", metric.WithDescription("Total number of HTTP requests"))
	latency, _  = meter.Float64Histogram("http_request_duration_seconds", metric.WithDescription("HTTP request latency in seconds"))
)

// Metrics returns a middleware that records HTTP request metrics.
func Metrics() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			saw := &statusAwareResponseWriter{ResponseWriter: w}

			next.ServeHTTP(saw, r)

			duration := time.Since(start).Seconds()
			ctx := r.Context()

			// If the handler never called WriteHeader, the status code is an
			// implicit 200 OK.
			status := saw.status
			if status == 0 {
				status = http.StatusOK
			}

			attrs := []attribute.KeyValue{
				attribute.String("http.method", r.Method),
				attribute.String("http.path", r.URL.Path),
				attribute.Int("http.status", status),
			}

			requests.Add(ctx, 1, metric.WithAttributes(attrs...))
			latency.Record(ctx, duration, metric.WithAttributes(attrs...))
		})
	}
}
