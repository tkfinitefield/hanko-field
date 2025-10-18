package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hanko-field/api/internal/repositories"
)

var (
	// ErrCounterInvalidInput indicates the caller supplied invalid counter parameters.
	ErrCounterInvalidInput = errors.New("counter: invalid input")
	// ErrCounterExhausted indicates the requested counter cannot increment further due to max bounds.
	ErrCounterExhausted = errors.New("counter: exhausted")
)

// CounterServiceDeps bundles collaborators required to construct a counter service instance.
type CounterServiceDeps struct {
	Repository repositories.CounterRepository
	Clock      func() time.Time
}

type counterService struct {
	repo       repositories.CounterRepository
	clock      func() time.Time
	configMu   sync.Mutex
	configured map[string]counterConfigSignature
}

type counterConfigSignature struct {
	stepSet      bool
	step         int64
	maxSet       bool
	maxValue     int64
	initialSet   bool
	initialValue int64
}

// CounterGenerationOptions controls how counter values are incremented and formatted.
// NewCounterService constructs a service that manages counter sequences on top of the repository.
func NewCounterService(deps CounterServiceDeps) (CounterService, error) {
	if deps.Repository == nil {
		return nil, errors.New("counter service: repository is required")
	}

	clock := deps.Clock
	if clock == nil {
		clock = time.Now
	}

	return &counterService{
		repo: deps.Repository,
		clock: func() time.Time {
			return clock().UTC()
		},
		configured: make(map[string]counterConfigSignature),
	}, nil
}

func (s *counterService) Next(ctx context.Context, scope, name string, opts CounterGenerationOptions) (CounterValue, error) {
	scope = strings.TrimSpace(scope)
	name = strings.TrimSpace(name)
	if scope == "" {
		return CounterValue{}, fmt.Errorf("%w: scope is required", ErrCounterInvalidInput)
	}
	if name == "" {
		return CounterValue{}, fmt.Errorf("%w: name is required", ErrCounterInvalidInput)
	}

	counterID := scope + ":" + name

	if err := s.ensureConfiguration(ctx, counterID, opts); err != nil {
		return CounterValue{}, err
	}

	value, err := s.repo.Next(ctx, counterID, opts.Step)
	if err != nil {
		var counterErr *repositories.CounterError
		if errors.As(err, &counterErr) {
			switch counterErr.Code {
			case repositories.CounterErrorInvalidInput:
				return CounterValue{}, fmt.Errorf("%w: %s", ErrCounterInvalidInput, counterErr.Message)
			case repositories.CounterErrorExhausted:
				return CounterValue{}, fmt.Errorf("%w: %s", ErrCounterExhausted, counterErr.Message)
			}
		}
		return CounterValue{}, err
	}

	now := s.clock()
	formatted := s.formatValue(now, value, opts)
	return CounterValue{Value: value, Formatted: formatted}, nil
}

func (s *counterService) NextOrderNumber(ctx context.Context) (string, error) {
	now := s.clock()
	opts := CounterGenerationOptions{
		Step:      0,
		Formatter: func(current time.Time, seq int64) string { return fmt.Sprintf("HF-%04d-%06d", current.Year(), seq) },
	}
	result, err := s.Next(ctx, "orders", fmt.Sprintf("%04d", now.Year()), opts)
	if err != nil {
		return "", err
	}
	return result.Formatted, nil
}

func (s *counterService) NextInvoiceNumber(ctx context.Context) (string, error) {
	now := s.clock()
	prefix := fmt.Sprintf("INV-%04d%02d-", now.Year(), int(now.Month()))
	opts := CounterGenerationOptions{
		Step:      1,
		Prefix:    prefix,
		PadLength: 6,
	}
	result, err := s.Next(ctx, "invoices", fmt.Sprintf("%04d%02d", now.Year(), int(now.Month())), opts)
	if err != nil {
		return "", err
	}
	return result.Formatted, nil
}

func (s *counterService) ensureConfiguration(ctx context.Context, counterID string, opts CounterGenerationOptions) error {
	signature := counterConfigSignature{}
	if opts.Step > 0 {
		signature.stepSet = true
		signature.step = opts.Step
	}
	if opts.MaxValue != nil {
		signature.maxSet = true
		signature.maxValue = *opts.MaxValue
	}
	if opts.InitialValue != nil {
		signature.initialSet = true
		signature.initialValue = *opts.InitialValue
	}

	s.configMu.Lock()
	defer s.configMu.Unlock()

	if existing, ok := s.configured[counterID]; ok && existing == signature {
		return nil
	}

	cfg := repositories.CounterConfig{}
	if signature.stepSet {
		cfg.Step = signature.step
	}
	if signature.maxSet {
		cfg.MaxValue = &signature.maxValue
	}
	if signature.initialSet {
		cfg.InitialValue = &signature.initialValue
	}

	if signature.stepSet || signature.maxSet || signature.initialSet {
		if err := s.repo.Configure(ctx, counterID, cfg); err != nil {
			return err
		}
	}
	s.configured[counterID] = signature
	return nil
}

func (s *counterService) formatValue(now time.Time, value int64, opts CounterGenerationOptions) string {
	if opts.Formatter != nil {
		return opts.Formatter(now, value)
	}

	formatted := strconv.FormatInt(value, 10)
	if opts.PadLength > 0 {
		formatted = fmt.Sprintf("%0*d", opts.PadLength, value)
	}
	if opts.Prefix != "" {
		formatted = opts.Prefix + formatted
	}
	if opts.Suffix != "" {
		formatted += opts.Suffix
	}
	return formatted
}
