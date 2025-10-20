package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/services"
)

func TestDesignHandlers_CreateDesign_Success(t *testing.T) {
	var captured services.CreateDesignCommand
	stub := &stubDesignService{
		createFn: func(_ context.Context, cmd services.CreateDesignCommand) (services.Design, error) {
			captured = cmd
			return services.Design{
				ID:               "dsg_test",
				OwnerID:          cmd.OwnerID,
				Label:            "My Design",
				Type:             services.DesignType(cmd.Type),
				TextLines:        []string{"Name"},
				FontID:           "font-1",
				MaterialID:       "material-1",
				Template:         "tmpl-1",
				Locale:           "ja-JP",
				Shape:            "round",
				SizeMM:           15,
				Status:           services.DesignStatusDraft,
				ThumbnailURL:     "https://example.com/thumb.png",
				Version:          1,
				CurrentVersionID: "ver_test",
				Assets: services.DesignAssets{
					SourcePath:  "assets/designs/dsg_test/sources/upload-1/source.png",
					VectorPath:  "",
					PreviewPath: "assets/designs/dsg_test/previews/ver_test/preview.png",
					PreviewURL:  "https://example.com/preview.png",
				},
				Source: services.DesignSource{
					Type:      services.DesignTypeTyped,
					RawName:   "Name",
					TextLines: []string{"Name"},
				},
				Snapshot:  map[string]any{"label": "My Design"},
				CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			}, nil
		},
	}

	handler := NewDesignHandlers(nil, stub)

	body := `{
        "label": "My Design",
        "type": "typed",
        "text_lines": ["Name"],
        "font_id": "font-1",
        "material_id": "material-1",
        "template_id": "tmpl-1",
        "locale": "ja-JP",
        "shape": "round",
        "size_mm": 15,
        "metadata": {"key": "value"}
    }`

	req := httptest.NewRequest(http.MethodPost, "/designs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-1")
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	res := httptest.NewRecorder()
	handler.createDesign(res, req)

	if res.Result().StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", res.Result().StatusCode)
	}
	if captured.OwnerID != "user-1" || captured.ActorID != "user-1" {
		t.Fatalf("expected owner/actor user-1, got %s/%s", captured.OwnerID, captured.ActorID)
	}
	if captured.IdempotencyKey != "idem-1" {
		t.Fatalf("expected idempotency key propagated")
	}
	if len(captured.TextLines) != 1 || captured.TextLines[0] != "Name" {
		t.Fatalf("unexpected text lines: %v", captured.TextLines)
	}
	if val, ok := captured.Metadata["key"]; !ok || val != "value" {
		t.Fatalf("metadata not propagated: %v", captured.Metadata)
	}

	var payload createDesignResponse
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response decode error: %v", err)
	}
	if payload.Design.ID != "dsg_test" {
		t.Fatalf("unexpected design id in response: %s", payload.Design.ID)
	}
	if payload.Design.Assets.PreviewURL != "https://example.com/preview.png" {
		t.Fatalf("unexpected preview url in response")
	}
}

func TestDesignHandlers_CreateDesign_Invalid(t *testing.T) {
	stub := &stubDesignService{
		createFn: func(context.Context, services.CreateDesignCommand) (services.Design, error) {
			return services.Design{}, services.ErrDesignInvalidInput
		},
	}

	handler := NewDesignHandlers(nil, stub)
	req := httptest.NewRequest(http.MethodPost, "/designs", strings.NewReader(`{"type":"typed"}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	res := httptest.NewRecorder()
	handler.createDesign(res, req)

	if res.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.Result().StatusCode)
	}
}

func TestDesignHandlers_ListDesigns_Success(t *testing.T) {
	var captured services.DesignListFilter
	now := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
	stub := &stubDesignService{
		listFn: func(_ context.Context, filter services.DesignListFilter) (domain.CursorPage[services.Design], error) {
			captured = filter
			return domain.CursorPage[services.Design]{
				Items: []services.Design{
					{
						ID:               "dsg_123",
						OwnerID:          filter.OwnerID,
						Label:            "Sample",
						Type:             services.DesignTypeTyped,
						Status:           services.DesignStatusReady,
						CurrentVersionID: "ver_1",
						Assets: services.DesignAssets{
							PreviewURL: "https://example.com/preview.png",
						},
						Snapshot:  map[string]any{"label": "Sample"},
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
				NextPageToken: "next-token",
			}, nil
		},
	}

	handler := NewDesignHandlers(nil, stub)
	req := httptest.NewRequest(http.MethodGet, "/designs?status=ready&type=typed&page_size=15&page_token=tok123&updatedAfter=2024-01-01T00:00:00Z", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	res := httptest.NewRecorder()
	handler.listDesigns(res, req)

	if res.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.Result().StatusCode)
	}

	if captured.OwnerID != "user-1" {
		t.Fatalf("expected owner filter user-1, got %s", captured.OwnerID)
	}
	if len(captured.Status) != 1 || captured.Status[0] != "ready" {
		t.Fatalf("unexpected status filter: %v", captured.Status)
	}
	if len(captured.Types) != 1 || captured.Types[0] != "typed" {
		t.Fatalf("unexpected type filter: %v", captured.Types)
	}
	if captured.UpdatedAfter == nil || !captured.UpdatedAfter.Equal(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("updatedAfter filter not applied: %v", captured.UpdatedAfter)
	}
	if captured.Pagination.PageSize != 15 || captured.Pagination.PageToken != "tok123" {
		t.Fatalf("unexpected pagination filter: %+v", captured.Pagination)
	}

	var payload designListResponse
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.NextPageToken != "next-token" {
		t.Fatalf("expected next token next-token, got %s", payload.NextPageToken)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(payload.Items))
	}
	if payload.Items[0].ID != "dsg_123" || payload.Items[0].Status != string(services.DesignStatusReady) {
		t.Fatalf("unexpected payload item: %+v", payload.Items[0])
	}
}

func TestDesignHandlers_ListDesigns_ForbiddenOverride(t *testing.T) {
	handler := NewDesignHandlers(nil, &stubDesignService{})
	req := httptest.NewRequest(http.MethodGet, "/designs?user=someone-else", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	res := httptest.NewRecorder()
	handler.listDesigns(res, req)

	if res.Result().StatusCode != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", res.Result().StatusCode)
	}
}

func TestDesignHandlers_ListDesigns_InvalidUpdatedAfter(t *testing.T) {
	handler := NewDesignHandlers(nil, &stubDesignService{})
	req := httptest.NewRequest(http.MethodGet, "/designs?updatedAfter=not-a-date", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	res := httptest.NewRecorder()
	handler.listDesigns(res, req)

	if res.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.Result().StatusCode)
	}
}

func TestDesignHandlers_GetDesign_SuccessWithHistory(t *testing.T) {
	var capturedOpts services.DesignReadOptions
	now := time.Date(2024, 3, 1, 9, 0, 0, 0, time.UTC)
	stub := &stubDesignService{
		getFn: func(_ context.Context, id string, opts services.DesignReadOptions) (services.Design, error) {
			capturedOpts = opts
			return services.Design{
				ID:               id,
				OwnerID:          "user-1",
				Label:            "Detail",
				Type:             services.DesignTypeTyped,
				Status:           services.DesignStatusReady,
				CurrentVersionID: "ver_1",
				Versions: []services.DesignVersion{
					{
						ID:        "ver_1",
						Version:   1,
						Snapshot:  map[string]any{"label": "Detail"},
						CreatedAt: now,
						CreatedBy: "user-1",
					},
				},
				CreatedAt: now,
				UpdatedAt: now,
			}, nil
		},
	}

	handler := NewDesignHandlers(nil, stub)
	req := httptest.NewRequest(http.MethodGet, "/designs/dsg_123?includeHistory=true", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "dsg_123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	res := httptest.NewRecorder()
	handler.getDesign(res, req)

	if res.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.Result().StatusCode)
	}
	if !capturedOpts.IncludeVersions {
		t.Fatalf("expected IncludeVersions to be true")
	}

	var payload designPayload
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.ID != "dsg_123" {
		t.Fatalf("unexpected payload id: %s", payload.ID)
	}
	if len(payload.Versions) != 1 || payload.Versions[0].ID != "ver_1" {
		t.Fatalf("versions not included: %+v", payload.Versions)
	}
}

func TestDesignHandlers_GetDesign_NotFoundForOtherUser(t *testing.T) {
	stub := &stubDesignService{
		getFn: func(context.Context, string, services.DesignReadOptions) (services.Design, error) {
			return services.Design{
				ID:      "dsg_123",
				OwnerID: "someone-else",
			}, nil
		},
	}

	handler := NewDesignHandlers(nil, stub)
	req := httptest.NewRequest(http.MethodGet, "/designs/dsg_123", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "dsg_123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	res := httptest.NewRecorder()
	handler.getDesign(res, req)

	if res.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", res.Result().StatusCode)
	}
}

type stubDesignService struct {
	createFn func(context.Context, services.CreateDesignCommand) (services.Design, error)
	getFn    func(context.Context, string, services.DesignReadOptions) (services.Design, error)
	listFn   func(context.Context, services.DesignListFilter) (domain.CursorPage[services.Design], error)
}

func (s *stubDesignService) CreateDesign(ctx context.Context, cmd services.CreateDesignCommand) (services.Design, error) {
	if s.createFn != nil {
		return s.createFn(ctx, cmd)
	}
	return services.Design{}, nil
}

func (s *stubDesignService) GetDesign(ctx context.Context, designID string, opts services.DesignReadOptions) (services.Design, error) {
	if s.getFn != nil {
		return s.getFn(ctx, designID, opts)
	}
	return services.Design{}, nil
}

func (s *stubDesignService) ListDesigns(ctx context.Context, filter services.DesignListFilter) (domain.CursorPage[services.Design], error) {
	if s.listFn != nil {
		return s.listFn(ctx, filter)
	}
	return domain.CursorPage[services.Design]{}, nil
}

func (s *stubDesignService) UpdateDesign(context.Context, services.UpdateDesignCommand) (services.Design, error) {
	return services.Design{}, nil
}

func (s *stubDesignService) DeleteDesign(context.Context, services.DeleteDesignCommand) error {
	return nil
}

func (s *stubDesignService) DuplicateDesign(context.Context, services.DuplicateDesignCommand) (services.Design, error) {
	return services.Design{}, nil
}

func (s *stubDesignService) RequestAISuggestion(context.Context, services.AISuggestionRequest) (services.AISuggestion, error) {
	return services.AISuggestion{}, nil
}

func (s *stubDesignService) ListAISuggestions(context.Context, string, services.AISuggestionFilter) (domain.CursorPage[services.AISuggestion], error) {
	return domain.CursorPage[services.AISuggestion]{}, nil
}

func (s *stubDesignService) UpdateAISuggestionStatus(context.Context, services.AISuggestionStatusCommand) (services.AISuggestion, error) {
	return services.AISuggestion{}, nil
}

func (s *stubDesignService) RequestRegistrabilityCheck(context.Context, services.RegistrabilityCheckCommand) (services.RegistrabilityCheckResult, error) {
	return services.RegistrabilityCheckResult{}, nil
}
