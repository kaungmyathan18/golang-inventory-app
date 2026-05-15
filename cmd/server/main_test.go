package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kaungmyathan18/golang-inventory-app/internal/config"
	"github.com/kaungmyathan18/golang-inventory-app/internal/handler"
	"github.com/kaungmyathan18/golang-inventory-app/internal/observability"
	"go.uber.org/zap"
)

func TestSetupRouter(t *testing.T) {
	metrics, err := observability.NewMetrics()
	if err != nil {
		t.Fatalf("NewMetrics: %v", err)
	}
	cfg := &config.Config{
		Server: config.ServerConfig{
			CORS: config.CORSConfig{
				Enabled:          true,
				AllowedOrigins:   []string{"https://example.com"},
				AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
				AllowedHeaders:   []string{"Content-Type"},
				ExposedHeaders:   []string{"X-Request-ID"},
				AllowCredentials: false,
				MaxAge:           300,
			},
		},
	}
	router := setupRouter(
		cfg,
		zap.NewNop(),
		metrics,
		handler.NewHealthHandler(zap.NewNop(), nil, nil),
		handler.NewAPIHandler(nil, zap.NewNop(), nil),
		handler.NewProductAPIHandler(nil, zap.NewNop(), nil),
		handler.NewCategoryAPIHandler(nil, zap.NewNop(), nil),
		handler.NewInventoryAPIHandler(nil, zap.NewNop(), nil),
	)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health/live", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Fatalf("health live status=%d body=%q", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/users", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("cors preflight status=%d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("allow origin = %q", got)
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("metrics status=%d", rec.Code)
	}
}

func TestServerConfigTimeoutsAreUsable(t *testing.T) {
	cfg := config.DefaultConfig("inventory")
	server := &http.Server{
		Addr:              ":0",
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
	}
	if server.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("read header timeout = %s", server.ReadHeaderTimeout)
	}
}
