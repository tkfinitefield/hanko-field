package repositories

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
)

func TestDependencyHealthRepositoryCollectSuccess(t *testing.T) {
	checks := []DependencyCheck{
		{
			Name: "firestore",
			Check: func(ctx context.Context) error {
				select {
				case <-time.After(10 * time.Millisecond):
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		},
		{
			Name: "storage",
			Check: func(context.Context) error {
				return nil
			},
		},
	}

	now := time.Date(2024, time.March, 1, 12, 0, 0, 0, time.UTC)
	repo, err := NewDependencyHealthRepository(checks,
		WithDependencyClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("NewDependencyHealthRepository: %v", err)
	}

	ctx := context.Background()
	report, err := repo.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if report.Status != domain.HealthStatusOK {
		t.Fatalf("expected status ok, got %s", report.Status)
	}
	if len(report.Checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(report.Checks))
	}
	for name, check := range report.Checks {
		if check.Status != domain.HealthStatusOK {
			t.Fatalf("expected check %s to be ok, got %s", name, check.Status)
		}
		if check.CheckedAt != now {
			t.Fatalf("expected check %s checkedAt %s, got %s", name, now, check.CheckedAt)
		}
	}
	if report.GeneratedAt != now {
		t.Fatalf("expected generatedAt %s, got %s", now, report.GeneratedAt)
	}
}

func TestDependencyHealthRepositoryCollectFailure(t *testing.T) {
	expectedErr := errors.New("boom")
	checks := []DependencyCheck{
		{
			Name: "firestore",
			Check: func(context.Context) error {
				return expectedErr
			},
		},
		{
			Name: "pubsub",
			Check: func(context.Context) error {
				return nil
			},
		},
	}

	repo, err := NewDependencyHealthRepository(checks)
	if err != nil {
		t.Fatalf("NewDependencyHealthRepository: %v", err)
	}

	ctx := context.Background()
	report, err := repo.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if report.Status != domain.HealthStatusDegraded {
		t.Fatalf("expected status degraded, got %s", report.Status)
	}
	check := report.Checks["firestore"]
	if check.Status != domain.HealthStatusDegraded {
		t.Fatalf("expected firestore status degraded, got %s", check.Status)
	}
	if check.Error != expectedErr.Error() {
		t.Fatalf("expected error %q, got %q", expectedErr.Error(), check.Error)
	}
}

func TestDependencyHealthRepositoryCollectTimeout(t *testing.T) {
	checks := []DependencyCheck{
		{
			Name:    "secrets",
			Timeout: 5 * time.Millisecond,
			Check: func(ctx context.Context) error {
				select {
				case <-time.After(20 * time.Millisecond):
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		},
	}

	repo, err := NewDependencyHealthRepository(checks)
	if err != nil {
		t.Fatalf("NewDependencyHealthRepository: %v", err)
	}

	ctx := context.Background()
	report, err := repo.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if report.Status != domain.HealthStatusError {
		t.Fatalf("expected status error, got %s", report.Status)
	}
	check := report.Checks["secrets"]
	if check.Status != domain.HealthStatusError {
		t.Fatalf("expected secrets status error, got %s", check.Status)
	}
	if check.Detail != "timeout" {
		t.Fatalf("expected detail timeout, got %s", check.Detail)
	}
}
