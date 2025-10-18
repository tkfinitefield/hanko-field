package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/services"
)

type stubSystemService struct {
	report services.SystemHealthReport
	err    error
}

func (s *stubSystemService) HealthReport(context.Context) (services.SystemHealthReport, error) {
	return s.report, s.err
}

func (s *stubSystemService) ListAuditLogs(context.Context, services.AuditLogFilter) (domain.CursorPage[domain.AuditLogEntry], error) {
	return domain.CursorPage[domain.AuditLogEntry]{}, nil
}

func (s *stubSystemService) NextCounterValue(context.Context, services.CounterCommand) (int64, error) {
	return 0, nil
}

func TestHealthHandlersHealthz(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	now := start.Add(30 * time.Second)
	handlers := NewHealthHandlers(
		WithHealthBuildInfo(services.BuildInfo{
			Version:     "1.0.0",
			CommitSHA:   "abc123",
			Environment: "prod",
			StartedAt:   start,
		}),
		WithHealthClock(func() time.Time { return now }),
	)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	handlers.Healthz(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["status"] != domain.HealthStatusOK {
		t.Fatalf("expected status ok, got %v", body["status"])
	}
	if body["version"] != "1.0.0" {
		t.Fatalf("expected version 1.0.0, got %v", body["version"])
	}
	if body["commitSha"] != "abc123" {
		t.Fatalf("expected commit abc123, got %v", body["commitSha"])
	}
	if body["environment"] != "prod" {
		t.Fatalf("expected environment prod, got %v", body["environment"])
	}
}

func TestHealthHandlersReadyzSuccess(t *testing.T) {
	now := time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC)
	svc := &stubSystemService{
		report: services.SystemHealthReport{
			Status:      domain.HealthStatusOK,
			Version:     "1.0.0",
			CommitSHA:   "abc123",
			Environment: "prod",
			Uptime:      time.Minute,
			GeneratedAt: now,
			Checks: map[string]domain.SystemHealthCheck{
				"firestore": {Status: domain.HealthStatusOK, Latency: 10 * time.Millisecond, CheckedAt: now},
			},
		},
	}

	handlers := NewHealthHandlers(
		WithHealthSystemService(svc),
		WithHealthClock(func() time.Time { return now }),
	)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()

	handlers.Readyz(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body struct {
		Status string `json:"status"`
		Checks map[string]struct {
			Status string `json:"status"`
		} `json:"checks"`
		Details []string `json:"details"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if body.Status != domain.HealthStatusOK {
		t.Fatalf("expected status ok, got %s", body.Status)
	}
	if len(body.Details) != 0 {
		t.Fatalf("expected no details, got %v", body.Details)
	}
	if body.Checks["firestore"].Status != domain.HealthStatusOK {
		t.Fatalf("expected firestore status ok, got %s", body.Checks["firestore"].Status)
	}
}

func TestHealthHandlersReadyzFailure(t *testing.T) {
	now := time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC)
	svc := &stubSystemService{
		report: services.SystemHealthReport{
			Status: domain.HealthStatusDegraded,
			Checks: map[string]domain.SystemHealthCheck{
				"pubsub": {Status: domain.HealthStatusDegraded, Error: "publish failed"},
			},
		},
	}

	handlers := NewHealthHandlers(
		WithHealthSystemService(svc),
		WithHealthClock(func() time.Time { return now }),
	)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()

	handlers.Readyz(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rr.Code)
	}

	var body struct {
		Status  string   `json:"status"`
		Details []string `json:"details"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if body.Status != domain.HealthStatusDegraded {
		t.Fatalf("expected status degraded, got %s", body.Status)
	}
	if len(body.Details) != 1 || body.Details[0] != "pubsub: publish failed" {
		t.Fatalf("expected details with pubsub failure, got %v", body.Details)
	}
}

var _ services.SystemService = (*stubSystemService)(nil)
