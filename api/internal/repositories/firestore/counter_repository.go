package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pfirestore "github.com/hanko-field/api/internal/platform/firestore"
	"github.com/hanko-field/api/internal/repositories"
)

const countersCollection = "counters"

type counterDocument struct {
	CurrentValue int64     `firestore:"currentValue"`
	Step         int64     `firestore:"step"`
	MaxValue     *int64    `firestore:"maxValue,omitempty"`
	UpdatedAt    time.Time `firestore:"updatedAt"`
}

// CounterRepository implements repositories.CounterRepository backed by Firestore transactions.
type CounterRepository struct {
	provider *pfirestore.Provider
	counters *pfirestore.BaseRepository[counterDocument]
}

// NewCounterRepository constructs a Firestore-backed counter repository.
func NewCounterRepository(provider *pfirestore.Provider) (*CounterRepository, error) {
	if provider == nil {
		return nil, errors.New("counter repository requires firestore provider")
	}
	base := pfirestore.NewBaseRepository[counterDocument](provider, countersCollection, nil, nil)
	return &CounterRepository{
		provider: provider,
		counters: base,
	}, nil
}

// Next atomically increments the counter identified by counterID and returns the next value.
func (r *CounterRepository) Next(ctx context.Context, counterID string, step int64) (int64, error) {
	if r == nil || r.provider == nil {
		return 0, errors.New("counter repository not initialised")
	}
	id := strings.TrimSpace(counterID)
	if id == "" {
		return 0, repositories.NewCounterError(repositories.CounterErrorInvalidInput, "counter id is required", nil)
	}
	if step < 0 {
		return 0, repositories.NewCounterError(repositories.CounterErrorInvalidInput, fmt.Sprintf("step must be positive, got %d", step), nil)
	}

	now := time.Now().UTC()
	var nextValue int64

	err := r.provider.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		ref, err := r.counters.DocumentRef(ctx, id)
		if err != nil {
			return err
		}

		snapshot, err := tx.Get(ref)
		switch status.Code(err) {
		case codes.NotFound:
			increment := step
			if increment <= 0 {
				increment = 1
			}
			doc := counterDocument{
				CurrentValue: increment,
				Step:         increment,
				UpdatedAt:    now,
			}
			if err := tx.Create(ref, doc); err != nil {
				return err
			}
			nextValue = doc.CurrentValue
			return nil
		case codes.OK:
			// proceed
		default:
			return err
		}

		var doc counterDocument
		if err := snapshot.DataTo(&doc); err != nil {
			return fmt.Errorf("firestore counters decode %s: %w", id, err)
		}

		increment := step
		if increment <= 0 {
			if doc.Step > 0 {
				increment = doc.Step
			} else {
				increment = 1
			}
		}

		newValue := doc.CurrentValue + increment
		if doc.MaxValue != nil && newValue > *doc.MaxValue {
			return repositories.NewCounterError(repositories.CounterErrorExhausted, fmt.Sprintf("counter %s exceeded max value %d", id, *doc.MaxValue), nil)
		}

		doc.CurrentValue = newValue
		doc.Step = increment
		doc.UpdatedAt = now

		if err := tx.Set(ref, doc, firestore.MergeAll); err != nil {
			return err
		}
		nextValue = newValue
		return nil
	})
	if err != nil {
		var counterErr *repositories.CounterError
		if errors.As(err, &counterErr) {
			return 0, counterErr
		}
		return 0, pfirestore.WrapError("counters.next", err)
	}
	return nextValue, nil
}

// Configure updates optional settings for the counter such as step size, max value, or initial value.
func (r *CounterRepository) Configure(ctx context.Context, counterID string, cfg repositories.CounterConfig) error {
	if r == nil || r.provider == nil {
		return errors.New("counter repository not initialised")
	}
	id := strings.TrimSpace(counterID)
	if id == "" {
		return repositories.NewCounterError(repositories.CounterErrorInvalidInput, "counter id is required", nil)
	}

	payload := make(map[string]any)
	now := time.Now().UTC()
	payload["updatedAt"] = now
	if cfg.Step > 0 {
		payload["step"] = cfg.Step
	}
	if cfg.MaxValue != nil {
		payload["maxValue"] = *cfg.MaxValue
	}
	if cfg.InitialValue != nil {
		payload["currentValue"] = *cfg.InitialValue
	}

	ref, err := r.counters.DocumentRef(ctx, id)
	if err != nil {
		return err
	}

	_, err = ref.Set(ctx, payload, firestore.MergeAll)
	if err != nil {
		return pfirestore.WrapError("counters.configure", err)
	}
	return nil
}
