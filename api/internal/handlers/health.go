package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/services"
)

const (
	defaultReadyTimeout = 2 * time.Second
)

// HealthHandlers exposes the /healthz and /readyz endpoints with dependency-aware readiness checks.
type HealthHandlers struct {
	system       services.SystemService
	build        services.BuildInfo
	readyTimeout time.Duration
	clock        func() time.Time
	startedAt    time.Time
}

// HealthOption customises handler behaviour.
type HealthOption func(*HealthHandlers)

// WithHealthSystemService injects the system service used to produce readiness reports.
func WithHealthSystemService(system services.SystemService) HealthOption {
	return func(h *HealthHandlers) {
		h.system = system
	}
}

// WithHealthBuildInfo supplies build metadata rendered in handler responses.
func WithHealthBuildInfo(info services.BuildInfo) HealthOption {
	return func(h *HealthHandlers) {
		h.build = info
		if !info.StartedAt.IsZero() {
			h.startedAt = info.StartedAt.UTC()
		}
	}
}

// WithHealthReadyTimeout overrides the timeout applied to readiness checks.
func WithHealthReadyTimeout(timeout time.Duration) HealthOption {
	return func(h *HealthHandlers) {
		if timeout > 0 {
			h.readyTimeout = timeout
		}
	}
}

// WithHealthClock injects a custom clock (primarily for tests).
func WithHealthClock(clock func() time.Time) HealthOption {
	return func(h *HealthHandlers) {
		if clock != nil {
			h.clock = clock
		}
	}
}

// NewHealthHandlers constructs handlers with sensible defaults.
func NewHealthHandlers(opts ...HealthOption) *HealthHandlers {
	handler := &HealthHandlers{
		readyTimeout: defaultReadyTimeout,
		clock: func() time.Time {
			return time.Now().UTC()
		},
		startedAt: time.Now().UTC(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(handler)
		}
	}
	if handler.startedAt.IsZero() {
		handler.startedAt = handler.clock()
	}
	return handler
}

// Healthz responds immediately indicating the process is alive.
func (h *HealthHandlers) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	now := h.clock()
	uptime := now.Sub(h.startedAt)

	payload := map[string]any{
		"status":        domain.HealthStatusOK,
		"version":       h.build.Version,
		"commitSha":     h.build.CommitSHA,
		"environment":   h.build.Environment,
		"uptimeSeconds": uptime.Seconds(),
		"timestamp":     now.Format(time.RFC3339),
	}

	_ = json.NewEncoder(w).Encode(payload)
}

// Readyz performs dependency checks before reporting readiness.
func (h *HealthHandlers) Readyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.system == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":    domain.HealthStatusError,
			"error":     "system service not configured",
			"timestamp": h.clock().Format(time.RFC3339),
		})
		return
	}

	ctx := r.Context()
	var cancel context.CancelFunc
	if h.readyTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, h.readyTimeout)
		defer cancel()
	}

	report, err := h.system.HealthReport(ctx)
	if err != nil {
		code := http.StatusServiceUnavailable
		if errors.Is(err, context.Canceled) {
			code = http.StatusRequestTimeout
		}
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":    domain.HealthStatusError,
			"error":     err.Error(),
			"timestamp": h.clock().Format(time.RFC3339),
		})
		return
	}

	response := h.newReadyResponse(report)
	statusCode := http.StatusOK
	if report.Status != domain.HealthStatusOK {
		statusCode = http.StatusServiceUnavailable
	}
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(response)
}

type readyResponse struct {
	Status        string                        `json:"status"`
	Version       string                        `json:"version,omitempty"`
	CommitSHA     string                        `json:"commitSha,omitempty"`
	Environment   string                        `json:"environment,omitempty"`
	UptimeSeconds float64                       `json:"uptimeSeconds"`
	Timestamp     string                        `json:"timestamp"`
	Checks        map[string]readyCheckResponse `json:"checks,omitempty"`
	Details       []string                      `json:"details,omitempty"`
}

type readyCheckResponse struct {
	Status    string  `json:"status"`
	Detail    string  `json:"detail,omitempty"`
	Error     string  `json:"error,omitempty"`
	LatencyMS float64 `json:"latencyMs,omitempty"`
	CheckedAt string  `json:"checkedAt,omitempty"`
}

func (h *HealthHandlers) newReadyResponse(report services.SystemHealthReport) readyResponse {
	now := h.clock()
	uptime := report.Uptime
	if uptime <= 0 {
		uptime = now.Sub(h.startedAt)
	}

	if report.Version == "" {
		report.Version = h.build.Version
	}
	if report.CommitSHA == "" {
		report.CommitSHA = h.build.CommitSHA
	}
	if report.Environment == "" {
		report.Environment = h.build.Environment
	}

	resp := readyResponse{
		Status:        report.Status,
		Version:       report.Version,
		CommitSHA:     report.CommitSHA,
		Environment:   report.Environment,
		UptimeSeconds: uptime.Seconds(),
		Timestamp:     ensureTime(report.GeneratedAt, now).Format(time.RFC3339),
		Checks:        map[string]readyCheckResponse{},
	}

	var details []string
	for name, check := range report.Checks {
		checkResp := readyCheckResponse{
			Status:    chooseNonEmpty(check.Status, domain.HealthStatusOK),
			Detail:    check.Detail,
			Error:     check.Error,
			LatencyMS: float64(check.Latency) / float64(time.Millisecond),
		}
		if !check.CheckedAt.IsZero() {
			checkResp.CheckedAt = check.CheckedAt.UTC().Format(time.RFC3339)
		}
		resp.Checks[name] = checkResp

		if checkResp.Status != domain.HealthStatusOK {
			switch {
			case checkResp.Error != "":
				details = append(details, name+": "+checkResp.Error)
			case checkResp.Detail != "":
				details = append(details, name+": "+checkResp.Detail)
			default:
				details = append(details, name+": unhealthy")
			}
		}
	}

	if len(details) > 0 {
		resp.Details = details
	}
	return resp
}

func ensureTime(value time.Time, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback.UTC()
	}
	return value.UTC()
}

func chooseNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
