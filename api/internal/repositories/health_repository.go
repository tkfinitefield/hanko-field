package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
)

const (
	defaultDependencyTimeout = 1500 * time.Millisecond
)

// DependencyCheck describes a dependency probe executed during readiness checks.
type DependencyCheck struct {
	Name    string
	Timeout time.Duration
	Check   func(context.Context) error
}

// DependencyHealthOption customises the behaviour of the dependency-backed health repository.
type DependencyHealthOption func(*dependencyHealthRepository)

// WithDependencyTimeout overrides the default timeout applied when a check omits its own timeout.
func WithDependencyTimeout(timeout time.Duration) DependencyHealthOption {
	return func(repo *dependencyHealthRepository) {
		if timeout > 0 {
			repo.defaultTimeout = timeout
		}
	}
}

// WithDependencyClock injects a custom clock primarily for tests.
func WithDependencyClock(clock func() time.Time) DependencyHealthOption {
	return func(repo *dependencyHealthRepository) {
		if clock != nil {
			repo.now = clock
		}
	}
}

type dependencyHealthRepository struct {
	checks         []DependencyCheck
	defaultTimeout time.Duration
	now            func() time.Time
}

var _ HealthRepository = (*dependencyHealthRepository)(nil)

// NewDependencyHealthRepository constructs a HealthRepository that evaluates the provided check set.
func NewDependencyHealthRepository(checks []DependencyCheck, opts ...DependencyHealthOption) (HealthRepository, error) {
	if len(checks) == 0 {
		return nil, errors.New("health repository: at least one dependency check is required")
	}

	repo := &dependencyHealthRepository{
		checks:         make([]DependencyCheck, len(checks)),
		defaultTimeout: defaultDependencyTimeout,
		now:            time.Now,
	}

	copy(repo.checks, checks)

	for _, opt := range opts {
		if opt != nil {
			opt(repo)
		}
	}

	return repo, nil
}

func (r *dependencyHealthRepository) Collect(ctx context.Context) (domain.SystemHealthReport, error) {
	if ctx == nil {
		return domain.SystemHealthReport{}, errors.New("health repository: context is required")
	}

	results := make(map[string]domain.SystemHealthCheck, len(r.checks))
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
	)

	wg.Add(len(r.checks))
	for _, check := range r.checks {
		check := check
		if strings.TrimSpace(check.Name) == "" {
			wg.Done()
			mu.Lock()
			if firstErr == nil {
				firstErr = errors.New("health repository: dependency check missing name")
			}
			mu.Unlock()
			continue
		}
		if check.Check == nil {
			wg.Done()
			mu.Lock()
			if firstErr == nil {
				firstErr = fmt.Errorf("health repository: dependency %s missing check function", check.Name)
			}
			mu.Unlock()
			continue
		}

		go func() {
			defer wg.Done()

			timeout := check.Timeout
			if timeout <= 0 {
				timeout = r.defaultTimeout
			}

			var (
				checkCtx    context.Context
				cancel      context.CancelFunc
				start       = r.now()
				status      = domain.HealthStatusOK
				detail      = "ok"
				errorString string
			)

			if timeout > 0 {
				checkCtx, cancel = context.WithTimeout(ctx, timeout)
			} else {
				checkCtx, cancel = context.WithCancel(ctx)
			}
			defer cancel()

			err := check.Check(checkCtx)
			end := r.now()
			elapsed := end.Sub(start)

			switch {
			case err == nil:
				// already ok
			case errors.Is(err, context.Canceled):
				status = domain.HealthStatusError
				detail = "cancelled"
				errorString = err.Error()
			case errors.Is(err, context.DeadlineExceeded):
				status = domain.HealthStatusError
				detail = "timeout"
				errorString = err.Error()
			default:
				status = domain.HealthStatusDegraded
				detail = err.Error()
				errorString = err.Error()
			}

			if checkCtx.Err() != nil && err == nil {
				// Timed out without returning an error.
				status = domain.HealthStatusError
				detail = checkCtx.Err().Error()
				errorString = checkCtx.Err().Error()
			}

			mu.Lock()
			results[check.Name] = domain.SystemHealthCheck{
				Status:    status,
				Detail:    detail,
				Error:     errorString,
				Latency:   elapsed,
				CheckedAt: end,
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	if firstErr != nil {
		return domain.SystemHealthReport{}, firstErr
	}

	reportStatus := domain.HealthStatusOK
	for _, result := range results {
		if result.Status != domain.HealthStatusOK {
			if result.Status == domain.HealthStatusError {
				reportStatus = domain.HealthStatusError
				break
			}
			reportStatus = domain.HealthStatusDegraded
		}
	}

	return domain.SystemHealthReport{
		Status:      reportStatus,
		Checks:      results,
		GeneratedAt: r.now(),
	}, nil
}
