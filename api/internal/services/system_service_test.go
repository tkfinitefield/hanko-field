package services

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

type stubHealthRepository struct {
	report domain.SystemHealthReport
	err    error
	calls  int
}

func (s *stubHealthRepository) Collect(ctx context.Context) (domain.SystemHealthReport, error) {
	s.calls++
	return s.report, s.err
}

type stubAuditService struct {
	filter AuditLogFilter
	result domain.CursorPage[domain.AuditLogEntry]
	err    error
}

func (s *stubAuditService) Record(context.Context, AuditLogRecord) {}

func (s *stubAuditService) List(ctx context.Context, filter AuditLogFilter) (domain.CursorPage[domain.AuditLogEntry], error) {
	s.filter = filter
	return s.result, s.err
}

type stubCounterService struct {
	scope string
	name  string
	opts  CounterGenerationOptions
	value CounterValue
	err   error
}

func (s *stubCounterService) Next(ctx context.Context, scope, name string, opts CounterGenerationOptions) (CounterValue, error) {
	s.scope = scope
	s.name = name
	s.opts = opts
	return s.value, s.err
}

func (s *stubCounterService) NextOrderNumber(context.Context) (string, error) { return "", nil }

func (s *stubCounterService) NextInvoiceNumber(context.Context) (string, error) { return "", nil }

func TestSystemServiceHealthReportEnrichesMetadata(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	now := start.Add(5 * time.Minute)
	repo := &stubHealthRepository{
		report: domain.SystemHealthReport{
			Checks: map[string]domain.SystemHealthCheck{
				"firestore": {Status: domain.HealthStatusOK},
			},
		},
	}

	svc, err := NewSystemService(SystemServiceDeps{
		HealthRepository: repo,
		Clock:            func() time.Time { return now },
		Build: BuildInfo{
			Version:     "1.2.3",
			CommitSHA:   "abc123",
			Environment: "prod",
			StartedAt:   start,
		},
	})
	if err != nil {
		t.Fatalf("NewSystemService: %v", err)
	}

	report, err := svc.HealthReport(context.Background())
	if err != nil {
		t.Fatalf("HealthReport: %v", err)
	}

	if report.Status != domain.HealthStatusOK {
		t.Fatalf("expected status ok, got %s", report.Status)
	}
	if report.Version != "1.2.3" {
		t.Fatalf("expected version 1.2.3, got %s", report.Version)
	}
	if report.CommitSHA != "abc123" {
		t.Fatalf("expected commit abc123, got %s", report.CommitSHA)
	}
	if report.Environment != "prod" {
		t.Fatalf("expected environment prod, got %s", report.Environment)
	}
	if report.Uptime != now.Sub(start) {
		t.Fatalf("expected uptime %s, got %s", now.Sub(start), report.Uptime)
	}
	if report.GeneratedAt != now {
		t.Fatalf("expected generatedAt %s, got %s", now, report.GeneratedAt)
	}
}

func TestSystemServiceHealthReportErrors(t *testing.T) {
	expected := errors.New("collect failed")
	repo := &stubHealthRepository{err: expected}

	svc, err := NewSystemService(SystemServiceDeps{
		HealthRepository: repo,
	})
	if err != nil {
		t.Fatalf("NewSystemService: %v", err)
	}

	_, err = svc.HealthReport(context.Background())
	if !errors.Is(err, expected) {
		t.Fatalf("expected error %v, got %v", expected, err)
	}
}

func TestNewSystemServiceRequiresRepository(t *testing.T) {
	_, err := NewSystemService(SystemServiceDeps{})
	if err == nil {
		t.Fatalf("expected error when repository missing")
	}
}

func TestSystemServiceDerivesStatusWhenMissing(t *testing.T) {
	repo := &stubHealthRepository{
		report: domain.SystemHealthReport{
			Checks: map[string]domain.SystemHealthCheck{
				"pubsub": {Status: domain.HealthStatusDegraded},
				"secret": {Status: domain.HealthStatusOK},
			},
		},
	}

	svc, err := NewSystemService(SystemServiceDeps{
		HealthRepository: repo,
	})
	if err != nil {
		t.Fatalf("NewSystemService: %v", err)
	}

	report, err := svc.HealthReport(context.Background())
	if err != nil {
		t.Fatalf("HealthReport: %v", err)
	}
	if report.Status != domain.HealthStatusDegraded {
		t.Fatalf("expected status degraded, got %s", report.Status)
	}
}

func TestSystemServiceListAuditLogsDelegates(t *testing.T) {
	repo := &stubHealthRepository{}
	audit := &stubAuditService{
		result: domain.CursorPage[domain.AuditLogEntry]{Items: []domain.AuditLogEntry{{ID: "1"}}},
	}

	svc, err := NewSystemService(SystemServiceDeps{HealthRepository: repo, Audit: audit})
	if err != nil {
		t.Fatalf("NewSystemService: %v", err)
	}

	filter := AuditLogFilter{Actor: "user-1"}
	result, err := svc.ListAuditLogs(context.Background(), filter)
	if err != nil {
		t.Fatalf("ListAuditLogs: %v", err)
	}
	if audit.filter.Actor != "user-1" {
		t.Fatalf("expected actor filter propagated, got %s", audit.filter.Actor)
	}
	if len(result.Items) != 1 || result.Items[0].ID != "1" {
		t.Fatalf("unexpected result: %+v", result.Items)
	}
}

func TestSystemServiceListAuditLogsMissing(t *testing.T) {
	repo := &stubHealthRepository{}
	svc, err := NewSystemService(SystemServiceDeps{HealthRepository: repo})
	if err != nil {
		t.Fatalf("NewSystemService: %v", err)
	}

	_, err = svc.ListAuditLogs(context.Background(), AuditLogFilter{})
	if err == nil {
		t.Fatalf("expected error when audit service missing")
	}
}

func TestSystemServiceNextCounterValueDelegates(t *testing.T) {
	repo := &stubHealthRepository{}
	counters := &stubCounterService{value: CounterValue{Value: 42}}

	svc, err := NewSystemService(SystemServiceDeps{HealthRepository: repo, Counters: counters})
	if err != nil {
		t.Fatalf("NewSystemService: %v", err)
	}

	value, err := svc.NextCounterValue(context.Background(), CounterCommand{CounterID: "orders:2024", Step: 5})
	if err != nil {
		t.Fatalf("NextCounterValue: %v", err)
	}
	if value != 42 {
		t.Fatalf("expected 42, got %d", value)
	}
	if counters.scope != "orders" || counters.name != "2024" {
		t.Fatalf("expected scope orders and name 2024, got %s:%s", counters.scope, counters.name)
	}
	if counters.opts.Step != 5 {
		t.Fatalf("expected step 5, got %d", counters.opts.Step)
	}
}

func TestSystemServiceNextCounterValueMissing(t *testing.T) {
	repo := &stubHealthRepository{}
	svc, err := NewSystemService(SystemServiceDeps{HealthRepository: repo})
	if err != nil {
		t.Fatalf("NewSystemService: %v", err)
	}

	if _, err := svc.NextCounterValue(context.Background(), CounterCommand{CounterID: "orders:2024"}); err == nil {
		t.Fatalf("expected error when counters missing")
	}
}

func TestSystemServiceNextCounterValueInvalidID(t *testing.T) {
	repo := &stubHealthRepository{}
	svc, err := NewSystemService(SystemServiceDeps{HealthRepository: repo, Counters: &stubCounterService{}})
	if err != nil {
		t.Fatalf("NewSystemService: %v", err)
	}

	if _, err := svc.NextCounterValue(context.Background(), CounterCommand{CounterID: "invalid"}); err == nil {
		t.Fatalf("expected error for invalid counter id")
	}
}

var _ repositories.HealthRepository = (*stubHealthRepository)(nil)
var _ AuditLogService = (*stubAuditService)(nil)
var _ CounterService = (*stubCounterService)(nil)
