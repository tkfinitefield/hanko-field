package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/services"
)

type stubNameMappingService struct {
	convertFunc func(ctx context.Context, cmd services.NameConversionCommand) (services.NameMapping, error)
	selectFunc  func(ctx context.Context, cmd services.NameMappingSelectCommand) (services.NameMapping, error)
}

func (s *stubNameMappingService) ConvertName(ctx context.Context, cmd services.NameConversionCommand) (services.NameMapping, error) {
	if s.convertFunc != nil {
		return s.convertFunc(ctx, cmd)
	}
	return services.NameMapping{}, nil
}

func (s *stubNameMappingService) SelectCandidate(ctx context.Context, cmd services.NameMappingSelectCommand) (services.NameMapping, error) {
	if s.selectFunc != nil {
		return s.selectFunc(ctx, cmd)
	}
	return services.NameMapping{}, nil
}

func TestNameMappingHandlersConvert_Success(t *testing.T) {
	now := time.Date(2024, 5, 1, 9, 30, 0, 0, time.UTC)
	expires := now.Add(12 * time.Hour)
	mapping := services.NameMapping{
		ID:      "nmap_01",
		UserID:  "user-123",
		UserRef: "/users/user-123",
		Input: services.NameMappingInput{
			Latin:   "Smith",
			Locale:  "en",
			Gender:  "neutral",
			Context: map[string]string{"hint": "soft"},
		},
		Status: services.NameMappingStatusReady,
		Source: "fallback",
		Candidates: []services.NameMappingCandidate{
			{ID: "cand-1", Kanji: "須密", Kana: []string{"スミス"}, Score: 0.82, Notes: "heuristic"},
		},
		ExpiresAt: &expires,
		CreatedAt: now,
		UpdatedAt: now,
	}

	var received services.NameConversionCommand
	svc := &stubNameMappingService{
		convertFunc: func(ctx context.Context, cmd services.NameConversionCommand) (services.NameMapping, error) {
			received = cmd
			return mapping, nil
		},
	}

	handler := NewNameMappingHandlers(nil, svc)
	body := bytes.NewBufferString(`{"latin":"Smith","locale":"en","gender":"neutral","context":{"hint":"soft"}}`)
	req := httptest.NewRequest(http.MethodPost, "/name-mappings:convert", body)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-123"}))
	resp := httptest.NewRecorder()

	handler.convert(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}

	if received.UserID != "user-123" {
		t.Fatalf("expected user id user-123, got %s", received.UserID)
	}
	if received.Latin != "Smith" {
		t.Fatalf("expected latin Smith, got %s", received.Latin)
	}
	if received.Locale != "en" {
		t.Fatalf("expected locale en, got %s", received.Locale)
	}

	var payload nameMappingResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON response: %v", err)
	}
	if payload.Mapping.ID != mapping.ID {
		t.Fatalf("expected mapping id %s, got %s", mapping.ID, payload.Mapping.ID)
	}
	if payload.Mapping.Status != string(services.NameMappingStatusReady) {
		t.Fatalf("expected status ready, got %s", payload.Mapping.Status)
	}
	if payload.Mapping.ExpiresAt != formatTime(expires) {
		t.Fatalf("expected expiresAt %s, got %s", formatTime(expires), payload.Mapping.ExpiresAt)
	}
	if len(payload.Mapping.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(payload.Mapping.Candidates))
	}
}

func TestNameMappingHandlersConvert_Error(t *testing.T) {
	svc := &stubNameMappingService{
		convertFunc: func(ctx context.Context, cmd services.NameConversionCommand) (services.NameMapping, error) {
			return services.NameMapping{}, services.ErrNameMappingInvalidInput
		},
	}

	handler := NewNameMappingHandlers(nil, svc)
	body := bytes.NewBufferString(`{"latin":"","locale":"en"}`)
	req := httptest.NewRequest(http.MethodPost, "/name-mappings:convert", body)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-123"}))
	resp := httptest.NewRecorder()

	handler.convert(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}
}

func TestNameMappingHandlersConvert_Unauthenticated(t *testing.T) {
	handler := NewNameMappingHandlers(nil, &stubNameMappingService{})
	req := httptest.NewRequest(http.MethodPost, "/name-mappings:convert", bytes.NewBufferString(`{"latin":"Smith"}`))
	resp := httptest.NewRecorder()

	handler.convert(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", resp.Code)
	}
}

func TestNameMappingHandlersSelect_Success(t *testing.T) {
	now := time.Date(2024, 6, 10, 8, 0, 0, 0, time.UTC)
	selected := services.NameMappingCandidate{ID: "cand-2", Kanji: "齋藤", Kana: []string{"サイトウ"}, Score: 0.91}
	mapping := services.NameMapping{
		ID:                "nmap_ready",
		UserID:            "user-1",
		Input:             services.NameMappingInput{Latin: "Saito", Locale: "en"},
		Status:            services.NameMappingStatusSelected,
		SelectedCandidate: &selected,
		SelectedAt:        &now,
		Candidates: []services.NameMappingCandidate{
			{ID: "cand-1", Kanji: "佐藤", Kana: []string{"サトウ"}, Score: 0.88},
			selected,
		},
		CreatedAt: now.Add(-time.Hour),
		UpdatedAt: now,
	}

	var received services.NameMappingSelectCommand
	svc := &stubNameMappingService{
		selectFunc: func(ctx context.Context, cmd services.NameMappingSelectCommand) (services.NameMapping, error) {
			received = cmd
			return mapping, nil
		},
	}

	handler := NewNameMappingHandlers(nil, svc)
	body := bytes.NewBufferString(`{"selected":"cand-2"}`)
	req := httptest.NewRequest(http.MethodPost, "/name-mappings/nmap_ready:select", body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("mappingId", "nmap_ready")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	resp := httptest.NewRecorder()

	handler.selectCandidate(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}

	if received.MappingID != "nmap_ready" {
		t.Fatalf("expected mapping id nmap_ready, got %s", received.MappingID)
	}
	if received.CandidateID != "cand-2" {
		t.Fatalf("expected candidate cand-2, got %s", received.CandidateID)
	}

	var payload nameMappingResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON payload: %v", err)
	}
	if payload.Mapping.SelectedCandidate == nil || payload.Mapping.SelectedCandidate.ID != "cand-2" {
		t.Fatalf("expected selected candidate cand-2 in payload, got %#v", payload.Mapping.SelectedCandidate)
	}
	if payload.Mapping.SelectedAt != formatTime(now) {
		t.Fatalf("expected selected_at %s, got %s", formatTime(now), payload.Mapping.SelectedAt)
	}
}

func TestNameMappingHandlersSelect_Error(t *testing.T) {
	svc := &stubNameMappingService{
		selectFunc: func(ctx context.Context, cmd services.NameMappingSelectCommand) (services.NameMapping, error) {
			return services.NameMapping{}, services.ErrNameMappingConflict
		},
	}

	handler := NewNameMappingHandlers(nil, svc)
	req := httptest.NewRequest(http.MethodPost, "/name-mappings/nmap_ready:select", bytes.NewBufferString(`{"selected":"cand-1"}`))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("mappingId", "nmap_ready")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	resp := httptest.NewRecorder()

	handler.selectCandidate(resp, req)

	if resp.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", resp.Code)
	}
}

func TestNameMappingHandlersSelect_Unauthenticated(t *testing.T) {
	handler := NewNameMappingHandlers(nil, &stubNameMappingService{})
	req := httptest.NewRequest(http.MethodPost, "/name-mappings/nmap_ready:select", bytes.NewBufferString(`{"selected":"cand-1"}`))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("mappingId", "nmap_ready")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	resp := httptest.NewRecorder()

	handler.selectCandidate(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", resp.Code)
	}
}
