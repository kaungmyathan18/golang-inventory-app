package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kaungmyathan18/golang-inventory-app/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap/zapcore"
)

func TestNewLogger(t *testing.T) {
	logger, err := NewLogger(config.LogConfig{Level: "debug", Format: "json"})
	if err != nil {
		t.Fatalf("NewLogger json: %v", err)
	}
	if !logger.Core().Enabled(zapcore.DebugLevel) {
		t.Fatal("logger does not enable debug level")
	}
	_ = logger.Sync()

	if _, err := NewLogger(config.LogConfig{Level: "not-a-level", Format: "console"}); err == nil {
		t.Fatal("NewLogger invalid level returned nil error")
	}
}

func TestMetricsMiddlewareAndHandler(t *testing.T) {
	registry := prometheus.NewRegistry()
	oldRegisterer := prometheus.DefaultRegisterer
	oldGatherer := prometheus.DefaultGatherer
	prometheus.DefaultRegisterer = registry
	prometheus.DefaultGatherer = registry
	t.Cleanup(func() {
		prometheus.DefaultRegisterer = oldRegisterer
		prometheus.DefaultGatherer = oldGatherer
	})

	metrics, err := NewMetrics()
	if err != nil {
		t.Fatalf("NewMetrics: %v", err)
	}

	r := chi.NewRouter()
	r.Use(metrics.Middleware())
	r.Get("/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/items/123", nil))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rec.Code)
	}

	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}
	var sawRequests, sawLatency bool
	for _, mf := range metricFamilies {
		switch mf.GetName() {
		case "http_requests_total":
			sawRequests = true
		case "http_request_duration_seconds":
			sawLatency = true
		}
	}
	if !sawRequests || !sawLatency {
		t.Fatalf("saw requests=%t latency=%t", sawRequests, sawLatency)
	}

	metricsRec := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(metricsRec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if metricsRec.Code != http.StatusOK {
		t.Fatalf("metrics status = %d", metricsRec.Code)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	logger, err := NewLogger(config.LogConfig{Level: "info", Format: "console"})
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	defer logger.Sync()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	rec := httptest.NewRecorder()
	LoggingMiddleware(logger)(next).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/logged", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestTracingMiddlewareAndDisabledProvider(t *testing.T) {
	shutdown, err := NewTracerProvider(config.OtelConfig{Enabled: false})
	if err != nil {
		t.Fatalf("NewTracerProvider disabled: %v", err)
	}
	if err := shutdown.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown tracer: %v", err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	rec := httptest.NewRecorder()
	TracingMiddleware(next).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/trace", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestMaybeStartPyroscopeNoop(t *testing.T) {
	t.Setenv("PYROSCOPE_SERVER_ADDRESS", "")
	stop, err := MaybeStartPyroscope("inventory")
	if err != nil {
		t.Fatalf("MaybeStartPyroscope: %v", err)
	}
	stop()
}
