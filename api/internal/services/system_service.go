package services

import (
	"context"
	"errors"
	"strings"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

// BuildInfo captures runtime metadata exposed via health endpoints.
type BuildInfo struct {
	Version     string
	CommitSHA   string
	Environment string
	StartedAt   time.Time
}

// SystemServiceDeps bundles collaborators required to construct a system service.
type SystemServiceDeps struct {
	HealthRepository repositories.HealthRepository
	Clock            func() time.Time
	Build            BuildInfo
	Audit            AuditLogService
	Counters         CounterService
}

type systemService struct {
	healthRepo repositories.HealthRepository
	clock      func() time.Time
	build      BuildInfo
	audit      AuditLogService
	counters   CounterService
}

var _ SystemService = (*systemService)(nil)

// NewSystemService assembles the system utility service providing health reports and metadata.
func NewSystemService(deps SystemServiceDeps) (SystemService, error) {
	if deps.HealthRepository == nil {
		return nil, errors.New("system service: health repository is required")
	}

	clock := deps.Clock
	if clock == nil {
		clock = time.Now
	}

	build := deps.Build
	if build.StartedAt.IsZero() {
		build.StartedAt = clock()
	}

	return &systemService{
		healthRepo: deps.HealthRepository,
		clock: func() time.Time {
			return clock().UTC()
		},
		build:    build,
		audit:    deps.Audit,
		counters: deps.Counters,
	}, nil
}

func (s *systemService) HealthReport(ctx context.Context) (SystemHealthReport, error) {
	if ctx == nil {
		return SystemHealthReport{}, errors.New("system service: context is required")
	}

	report, err := s.healthRepo.Collect(ctx)
	if err != nil {
		return SystemHealthReport{}, err
	}

	now := s.clock()
	report.GeneratedAt = ensureTimestamp(report.GeneratedAt, now)
	report.Version = chooseFirstNonEmpty(report.Version, s.build.Version)
	report.CommitSHA = chooseFirstNonEmpty(report.CommitSHA, s.build.CommitSHA)
	report.Environment = chooseFirstNonEmpty(report.Environment, s.build.Environment)

	if report.Uptime <= 0 && !s.build.StartedAt.IsZero() {
		report.Uptime = now.Sub(s.build.StartedAt)
	}

	if len(report.Checks) == 0 {
		report.Checks = map[string]domain.SystemHealthCheck{}
	}

	if strings.TrimSpace(report.Status) == "" {
		report.Status = deriveStatus(report.Checks)
	}

	return report, nil
}

func (s *systemService) ListAuditLogs(ctx context.Context, filter AuditLogFilter) (domain.CursorPage[domain.AuditLogEntry], error) {
	if ctx == nil {
		return domain.CursorPage[domain.AuditLogEntry]{}, errors.New("system service: context is required")
	}
	if s.audit == nil {
		return domain.CursorPage[domain.AuditLogEntry]{}, errors.New("system service: audit service not configured")
	}
	return s.audit.List(ctx, filter)
}

func (s *systemService) NextCounterValue(ctx context.Context, cmd CounterCommand) (int64, error) {
	if ctx == nil {
		return 0, errors.New("system service: context is required")
	}
	if s.counters == nil {
		return 0, errors.New("system service: counter service not configured")
	}
	scope, name, err := parseCounterID(cmd.CounterID)
	if err != nil {
		return 0, err
	}
	value, err := s.counters.Next(ctx, scope, name, CounterGenerationOptions{Step: cmd.Step})
	if err != nil {
		return 0, err
	}
	return value.Value, nil
}

func ensureTimestamp(ts time.Time, fallback time.Time) time.Time {
	if ts.IsZero() {
		return fallback
	}
	return ts.UTC()
}

func chooseFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func deriveStatus(checks map[string]domain.SystemHealthCheck) string {
	if len(checks) == 0 {
		return domain.HealthStatusOK
	}
	status := domain.HealthStatusOK
	for _, check := range checks {
		switch check.Status {
		case domain.HealthStatusOK, "":
			continue
		case domain.HealthStatusError:
			return domain.HealthStatusError
		default:
			status = domain.HealthStatusDegraded
		}
	}
	return status
}

func parseCounterID(counterID string) (string, string, error) {
	id := strings.TrimSpace(counterID)
	if id == "" {
		return "", "", errors.New("system service: counter id is required")
	}
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", errors.New("system service: counter id must be in scope:name format")
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}
