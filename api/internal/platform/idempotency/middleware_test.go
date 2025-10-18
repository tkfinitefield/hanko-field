package idempotency

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var fixedTime = time.Date(2024, time.January, 1, 12, 0, 0, 0, time.UTC)

func TestMiddleware_MissingHeader(t *testing.T) {
	store := NewMemoryStore()
	middleware := Middleware(store, WithClock(func() time.Time { return fixedTime }))

	handlerCalled := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		handlerCalled = true
	})

	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(`{"foo":"bar"}`))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	middleware(next).ServeHTTP(rr, req)

	if handlerCalled {
		t.Fatal("handler should not be invoked when header is missing")
	}
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
	assertErrorResponse(t, rr.Body.Bytes(), "idempotency_key_required")
}

func TestMiddleware_ReplaysStoredResponse(t *testing.T) {
	store := NewMemoryStore()
	var calls int
	middleware := Middleware(store, WithClock(func() time.Time { return fixedTime }))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	handler := middleware(next)

	req1 := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(`{"foo":"bar"}`))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Idempotency-Key", "abc-123")

	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	if calls != 1 {
		t.Fatalf("expected handler to be called once, got %d", calls)
	}
	if rr1.Code != http.StatusCreated {
		t.Fatalf("unexpected first response status: %d", rr1.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(`{"foo":"bar"}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", "abc-123")

	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if calls != 1 {
		t.Fatalf("expected handler not to be called again, got %d calls", calls)
	}
	if rr2.Code != http.StatusCreated {
		t.Fatalf("expected replayed status 201, got %d", rr2.Code)
	}
	if rr2.Header().Get(replayHeaderName) != "true" {
		t.Fatalf("expected replay header to be present")
	}
	if got := rr2.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected content-type json, got %s", got)
	}
	if body := rr2.Body.String(); body != rr1.Body.String() {
		t.Fatalf("expected response body %s, got %s", rr1.Body.String(), body)
	}
}

func TestMiddleware_ConflictingFingerprintReturnsConflict(t *testing.T) {
	store := NewMemoryStore()
	middleware := Middleware(store, WithClock(func() time.Time { return fixedTime }))

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(`{"foo":"bar"}`))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Idempotency-Key", "same-key")

	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("expected first request success, got %d", rr1.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(`{"foo":"baz"}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", "same-key")

	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusConflict {
		t.Fatalf("expected conflict status, got %d", rr2.Code)
	}
	assertErrorResponse(t, rr2.Body.Bytes(), "idempotency_key_conflict")
}

func TestMiddleware_PendingReservationReturnsConflict(t *testing.T) {
	store := NewMemoryStore()
	clock := fixedTime
	middleware := Middleware(store, WithClock(func() time.Time { return clock }))
	handler := middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not be invoked when reservation pending")
	}))

	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(`{"foo":"bar"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "pending-key")

	body, err := readAndReplayBody(req)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	identity := extractRequester(req.Context())
	fingerprint := requestFingerprint(req, body, identity)
	scoped := scopedKey("pending-key", identity)
	if _, err := store.Reserve(req.Context(), scoped, fingerprint, clock, time.Hour); err != nil {
		t.Fatalf("failed to seed reservation: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409 for pending reservation, got %d", rr.Code)
	}
	assertErrorResponse(t, rr.Body.Bytes(), "idempotency_in_progress")
}

func TestMiddleware_SaveFailureRollsBackReservation(t *testing.T) {
	store := &stubStore{failSave: true}
	middleware := Middleware(store, WithClock(func() time.Time { return fixedTime }))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(`{"foo":"bar"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "fail-key")

	rr := httptest.NewRecorder()
	middleware(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 response, got %d", rr.Code)
	}
	assertErrorResponse(t, rr.Body.Bytes(), "idempotency_store_error")
	if !store.released {
		t.Fatalf("expected reservation to be released on failure")
	}
}

type stubStore struct {
	failSave bool
	released bool
}

func (s *stubStore) Reserve(context.Context, string, string, time.Time, time.Duration) (Reservation, error) {
	return Reservation{State: ReservationStateNew, Record: Record{}}, nil
}

func (s *stubStore) SaveResponse(context.Context, string, string, Response, time.Time, time.Duration) error {
	if s.failSave {
		return errors.New("save failed")
	}
	return nil
}

func (s *stubStore) Release(context.Context, string, string) error {
	s.released = true
	return nil
}

func (s *stubStore) CleanupExpired(context.Context, time.Time, int) (int, error) {
	return 0, nil
}

func assertErrorResponse(t *testing.T, payload []byte, expected string) {
	t.Helper()

	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("failed to decode error payload: %v", err)
	}
	if body.Error != expected {
		t.Fatalf("expected error code %s, got %s", expected, body.Error)
	}
}
