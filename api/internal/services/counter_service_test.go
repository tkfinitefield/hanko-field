package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/hanko-field/api/internal/repositories"
)

type stubCounterRepository struct {
	mu             sync.Mutex
	nextFn         func(context.Context, string, int64) (int64, error)
	configureFn    func(context.Context, string, repositories.CounterConfig) error
	nextCalls      []counterCall
	configureCalls []configureCall
}

type counterCall struct {
	ID   string
	Step int64
}

type configureCall struct {
	ID  string
	Cfg repositories.CounterConfig
}

func (s *stubCounterRepository) Next(ctx context.Context, counterID string, step int64) (int64, error) {
	s.mu.Lock()
	s.nextCalls = append(s.nextCalls, counterCall{ID: counterID, Step: step})
	s.mu.Unlock()
	if s.nextFn != nil {
		return s.nextFn(ctx, counterID, step)
	}
	return 0, nil
}

func (s *stubCounterRepository) Configure(ctx context.Context, counterID string, cfg repositories.CounterConfig) error {
	s.mu.Lock()
	s.configureCalls = append(s.configureCalls, configureCall{ID: counterID, Cfg: cfg})
	s.mu.Unlock()
	if s.configureFn != nil {
		return s.configureFn(ctx, counterID, cfg)
	}
	return nil
}

func TestCounterServiceNextFormatsAndConfigures(t *testing.T) {
	repo := &stubCounterRepository{}
	repo.nextFn = func(context.Context, string, int64) (int64, error) {
		return 42, nil
	}

	svc, err := NewCounterService(CounterServiceDeps{Repository: repo, Clock: func() time.Time {
		return time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	}})
	if err != nil {
		t.Fatalf("new counter service: %v", err)
	}

	ctx := context.Background()
	value, err := svc.Next(ctx, "design", "global", CounterGenerationOptions{
		Step:      5,
		Prefix:    "DES-",
		PadLength: 4,
	})
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if value.Value != 42 {
		t.Fatalf("expected raw value 42, got %d", value.Value)
	}
	if value.Formatted != "DES-0042" {
		t.Fatalf("expected formatted DES-0042, got %s", value.Formatted)
	}

	repo.mu.Lock()
	if len(repo.configureCalls) != 1 {
		t.Fatalf("expected configure called once, got %d", len(repo.configureCalls))
	}
	if repo.configureCalls[0].Cfg.Step != 5 {
		t.Fatalf("expected configure step 5, got %d", repo.configureCalls[0].Cfg.Step)
	}
	repo.mu.Unlock()
}

func TestCounterServiceMapsRepositoryErrors(t *testing.T) {
	repo := &stubCounterRepository{}
	repo.nextFn = func(context.Context, string, int64) (int64, error) {
		return 0, repositories.NewCounterError(repositories.CounterErrorExhausted, "limit", nil)
	}

	svc, err := NewCounterService(CounterServiceDeps{Repository: repo})
	if err != nil {
		t.Fatalf("new counter service: %v", err)
	}

	_, err = svc.Next(context.Background(), "test", "limit", CounterGenerationOptions{})
	if !errors.Is(err, ErrCounterExhausted) {
		t.Fatalf("expected exhausted error, got %v", err)
	}
}

func TestCounterServiceNextOrderNumber(t *testing.T) {
	repo := &stubCounterRepository{}
	repo.nextFn = func(context.Context, string, int64) (int64, error) {
		return 7, nil
	}
	repo.configureFn = func(context.Context, string, repositories.CounterConfig) error {
		return nil
	}

	svc, err := NewCounterService(CounterServiceDeps{Repository: repo, Clock: func() time.Time {
		return time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	}})
	if err != nil {
		t.Fatalf("new counter service: %v", err)
	}

	result, err := svc.NextOrderNumber(context.Background())
	if err != nil {
		t.Fatalf("next order number: %v", err)
	}
	if result != "HF-2025-000007" {
		t.Fatalf("expected formatted order number, got %s", result)
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.nextCalls) != 1 {
		t.Fatalf("expected one next call, got %d", len(repo.nextCalls))
	}
	if repo.nextCalls[0].ID != "orders:2025" {
		t.Fatalf("expected counter id orders:2025, got %s", repo.nextCalls[0].ID)
	}
}
