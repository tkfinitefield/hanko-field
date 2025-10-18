package idempotency

import (
	"context"
	"sync"
	"time"
)

// MemoryStore provides an in-memory implementation useful for testing and local development.
type MemoryStore struct {
	mu      sync.Mutex
	records map[string]Record
}

// NewMemoryStore constructs an empty memory-backed idempotency store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{records: make(map[string]Record)}
}

// Reserve implements the Store interface.
func (s *MemoryStore) Reserve(_ context.Context, key, fingerprint string, now time.Time, ttl time.Duration) (Reservation, error) {
	now = now.UTC()
	s.mu.Lock()
	defer s.mu.Unlock()

	if ttl <= 0 {
		ttl = DefaultTTL
	}

	id := compositeKey(key, fingerprint)

	record, ok := s.records[id]
	if !ok || (!record.ExpiresAt.IsZero() && !now.Before(record.ExpiresAt)) {
		expires := now.Add(ttl)
		record = Record{
			Key:         key,
			Fingerprint: fingerprint,
			Status:      StatusPending,
			CreatedAt:   now,
			UpdatedAt:   now,
			ExpiresAt:   expires,
		}
		s.records[id] = record
		return Reservation{State: ReservationStateNew, Record: record}, nil
	}

	if record.Fingerprint != fingerprint {
		return Reservation{}, ErrFingerprintMismatch
	}

	if record.Status == StatusCompleted {
		return Reservation{State: ReservationStateCompleted, Record: record}, nil
	}

	return Reservation{State: ReservationStatePending, Record: record}, nil
}

// SaveResponse implements the Store interface.
func (s *MemoryStore) SaveResponse(_ context.Context, key, fingerprint string, resp Response, now time.Time, ttl time.Duration) error {
	now = now.UTC()
	s.mu.Lock()
	defer s.mu.Unlock()

	if ttl <= 0 {
		ttl = DefaultTTL
	}

	id := compositeKey(key, fingerprint)

	record, ok := s.records[id]
	if ok && record.Fingerprint != fingerprint {
		return ErrFingerprintMismatch
	}
	if !ok {
		record = Record{Key: key, Fingerprint: fingerprint, CreatedAt: now}
	}

	record.Status = StatusCompleted
	record.ResponseStatus = resp.Status
	record.ResponseHeaders = sanitizeHeaders(resp.Headers)
	if len(resp.Body) > 0 {
		record.ResponseBody = append([]byte(nil), resp.Body...)
	} else {
		record.ResponseBody = nil
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.UpdatedAt = now
	record.ExpiresAt = now.Add(ttl)
	s.records[id] = record

	return nil
}

// CleanupExpired implements the Store interface.
func (s *MemoryStore) CleanupExpired(_ context.Context, now time.Time, limit int) (int, error) {
	now = now.UTC()
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 || limit > len(s.records) {
		limit = len(s.records)
	}

	removed := 0
	for id, record := range s.records {
		if record.ExpiresAt.IsZero() || now.Before(record.ExpiresAt) {
			continue
		}
		delete(s.records, id)
		removed++
		if removed >= limit {
			break
		}
	}

	return removed, nil
}

// Release deletes the reservation so that subsequent attempts may retry.
func (s *MemoryStore) Release(_ context.Context, key, fingerprint string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.records, compositeKey(key, fingerprint))
	return nil
}
