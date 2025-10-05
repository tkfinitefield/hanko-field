package idempotency

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultCollection  = "idempotency_keys"
	defaultMaxAttempts = 5
)

// FirestoreOption customises the FirestoreStore behaviour.
type FirestoreOption func(*FirestoreStore)

// WithCollection overrides the collection name used to store idempotency keys.
func WithCollection(name string) FirestoreOption {
	return func(store *FirestoreStore) {
		if name != "" {
			store.collection = name
		}
	}
}

// WithMaxAttempts configures the transaction retry attempts.
func WithMaxAttempts(attempts int) FirestoreOption {
	return func(store *FirestoreStore) {
		if attempts > 0 {
			store.maxAttempts = attempts
		}
	}
}

// FirestoreStore implements Store backed by Google Cloud Firestore.
type FirestoreStore struct {
	client      *firestore.Client
	collection  string
	maxAttempts int
}

// NewFirestoreStore constructs a Firestore-backed idempotency store.
func NewFirestoreStore(client *firestore.Client, opts ...FirestoreOption) *FirestoreStore {
	store := &FirestoreStore{
		client:      client,
		collection:  defaultCollection,
		maxAttempts: defaultMaxAttempts,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(store)
		}
	}
	return store
}

// Reserve ensures the key is uniquely associated with the fingerprint and returns any stored response.
func (s *FirestoreStore) Reserve(ctx context.Context, key, fingerprint string, now time.Time, ttl time.Duration) (Reservation, error) {
	now = now.UTC()
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	ref := s.client.Collection(s.collection).Doc(compositeKey(key, fingerprint))
	attempts := s.maxAttempts
	if attempts <= 0 {
		attempts = 1
	}

	var result Reservation
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				record := firestoreRecord{
					Key:         key,
					Fingerprint: fingerprint,
					Status:      string(StatusPending),
					CreatedAt:   now,
					UpdatedAt:   now,
					ExpiresAt:   now.Add(ttl),
				}
				if err := tx.Set(ref, record); err != nil {
					return err
				}
				result = Reservation{State: ReservationStateNew, Record: record.toRecord()}
				return nil
			}
			return err
		}

		var record firestoreRecord
		if err := snap.DataTo(&record); err != nil {
			return err
		}
		if record.Fingerprint != fingerprint {
			return ErrFingerprintMismatch
		}

		if !record.ExpiresAt.IsZero() && !now.Before(record.ExpiresAt) {
			// Treat expired records as new reservations.
			record = firestoreRecord{
				Key:         key,
				Fingerprint: fingerprint,
				Status:      string(StatusPending),
				CreatedAt:   now,
				UpdatedAt:   now,
				ExpiresAt:   now.Add(ttl),
			}
			if err := tx.Set(ref, record); err != nil {
				return err
			}
			result = Reservation{State: ReservationStateNew, Record: record.toRecord()}
			return nil
		}

		if record.Status == string(StatusCompleted) {
			result = Reservation{State: ReservationStateCompleted, Record: record.toRecord()}
			return nil
		}

		result = Reservation{State: ReservationStatePending, Record: record.toRecord()}
		return nil
	}, firestore.MaxAttempts(attempts))

	return result, err
}

// SaveResponse persists the completed HTTP response associated with the key.
func (s *FirestoreStore) SaveResponse(ctx context.Context, key, fingerprint string, resp Response, now time.Time, ttl time.Duration) error {
	now = now.UTC()
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	ref := s.client.Collection(s.collection).Doc(compositeKey(key, fingerprint))
	attempts := s.maxAttempts
	if attempts <= 0 {
		attempts = 1
	}

	headers := sanitizeHeaders(resp.Headers)
	var bodyCopy []byte
	if len(resp.Body) > 0 {
		bodyCopy = append([]byte(nil), resp.Body...)
	}

	return s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		var record firestoreRecord
		if err != nil {
			if status.Code(err) != codes.NotFound {
				return err
			}
			record = firestoreRecord{
				Key:         key,
				Fingerprint: fingerprint,
				CreatedAt:   now,
			}
		} else {
			if err := snap.DataTo(&record); err != nil {
				return err
			}
			if record.Fingerprint != fingerprint {
				return ErrFingerprintMismatch
			}
			if record.CreatedAt.IsZero() {
				record.CreatedAt = now
			}
		}

		record.Status = string(StatusCompleted)
		record.ResponseStatus = resp.Status
		record.ResponseHeaders = headers
		record.ResponseBody = bodyCopy
		record.UpdatedAt = now
		record.ExpiresAt = now.Add(ttl)

		return tx.Set(ref, record)
	}, firestore.MaxAttempts(attempts))
}

// CleanupExpired removes expired idempotency records up to the provided limit.
func (s *FirestoreStore) CleanupExpired(ctx context.Context, now time.Time, limit int) (int, error) {
	now = now.UTC()
	if limit <= 0 {
		limit = 100
	}

	query := s.client.Collection(s.collection).Where("expires_at", "<=", now).Limit(limit)
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return 0, err
	}
	if len(docs) == 0 {
		return 0, nil
	}

	batch := s.client.Batch()
	for _, doc := range docs {
		batch.Delete(doc.Ref)
	}
	if _, err := batch.Commit(ctx); err != nil {
		return 0, err
	}

	return len(docs), nil
}

// Release removes the reservation to allow callers to retry.
func (s *FirestoreStore) Release(ctx context.Context, key, fingerprint string) error {
	ref := s.client.Collection(s.collection).Doc(compositeKey(key, fingerprint))
	_, err := ref.Delete(ctx)
	if status.Code(err) == codes.NotFound {
		return nil
	}
	return err
}

type firestoreRecord struct {
	Key             string              `firestore:"key"`
	Fingerprint     string              `firestore:"fingerprint"`
	Status          string              `firestore:"status"`
	ResponseStatus  int                 `firestore:"response_status"`
	ResponseHeaders map[string][]string `firestore:"response_headers"`
	ResponseBody    []byte              `firestore:"response_body"`
	CreatedAt       time.Time           `firestore:"created_at"`
	UpdatedAt       time.Time           `firestore:"updated_at"`
	ExpiresAt       time.Time           `firestore:"expires_at"`
}

func (r firestoreRecord) toRecord() Record {
	return Record{
		Key:             r.Key,
		Fingerprint:     r.Fingerprint,
		Status:          Status(r.Status),
		ResponseStatus:  r.ResponseStatus,
		ResponseHeaders: r.ResponseHeaders,
		ResponseBody:    r.ResponseBody,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
		ExpiresAt:       r.ExpiresAt,
	}
}
