package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/services"
)

type routerStubSystemService struct {
	report services.SystemHealthReport
	err    error
}

func (s *routerStubSystemService) HealthReport(context.Context) (services.SystemHealthReport, error) {
	return s.report, s.err
}

func (s *routerStubSystemService) ListAuditLogs(context.Context, services.AuditLogFilter) (domain.CursorPage[domain.AuditLogEntry], error) {
	return domain.CursorPage[domain.AuditLogEntry]{}, nil
}

func (s *routerStubSystemService) NextCounterValue(context.Context, services.CounterCommand) (int64, error) {
	return 0, nil
}

func TestNewRouter_DefaultMounts(t *testing.T) {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	healthHandlers := NewHealthHandlers(
		WithHealthSystemService(&routerStubSystemService{
			report: services.SystemHealthReport{
				Status:      domain.HealthStatusOK,
				Uptime:      5 * time.Second,
				GeneratedAt: now,
				Checks: map[string]domain.SystemHealthCheck{
					"firestore": {Status: domain.HealthStatusOK},
				},
			},
		}),
		WithHealthClock(func() time.Time { return now }),
	)

	router := NewRouter(WithHealthHandlers(healthHandlers))

	t.Run("healthz", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}
		if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
			t.Fatalf("expected content-type application/json, got %s", ct)
		}
	})

	t.Run("readyz", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}
	})

	t.Run("default not implemented group", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/public", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotImplemented {
			t.Fatalf("expected status 501, got %d", rr.Code)
		}

		var body map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("expected JSON body: %v", err)
		}
		if body["error"] != "not_implemented" {
			t.Fatalf("expected not_implemented error, got %v", body["error"])
		}
	})
}

func TestNewRouter_WithRegistrars(t *testing.T) {
	registrar := func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
	}

	router := NewRouter(WithPublicRoutes(registrar))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rr.Code)
	}
}

func TestNewRouter_NotFound(t *testing.T) {
	router := NewRouter()

	req := httptest.NewRequest(http.MethodGet, "/does/not/exist", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON body: %v", err)
	}
	if body["error"] != "route_not_found" {
		t.Fatalf("expected route_not_found error, got %v", body["error"])
	}
}

func TestNewRouter_GroupMiddleware(t *testing.T) {
	webhookHeader := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test-Middleware", "webhooks")
			next.ServeHTTP(w, r)
		})
	}

	router := NewRouter(WithWebhookMiddlewares(webhookHeader))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks/sample", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Header().Get("X-Test-Middleware") != "webhooks" {
		t.Fatalf("expected webhook middleware to set header")
	}
}
