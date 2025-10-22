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

func TestDesignHandlers_DuplicateDesign_Success(t *testing.T) {
	var captured services.DuplicateDesignCommand
	stub := &stubDesignService{
		duplicateFn: func(_ context.Context, cmd services.DuplicateDesignCommand) (services.Design, error) {
			captured = cmd
			return services.Design{
				ID:               "dsg_copy",
				OwnerID:          "user-1",
				Label:            "Copy Label",
				Type:             services.DesignTypeTyped,
				Status:           services.DesignStatusDraft,
				Version:          1,
				CurrentVersionID: "ver_copy",
				Assets: services.DesignAssets{
					PreviewPath: "assets/designs/dsg_copy/previews/ver_copy/preview.png",
					PreviewURL:  "https://example.com/preview.png",
				},
				Snapshot:  map[string]any{"label": "Copy Label"},
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}, nil
		},
	}

	handler := NewDesignHandlers(nil, stub)
	reqBody := `{"label":"  Copy Label  "}`
	req := httptest.NewRequest(http.MethodPost, "/designs/dsg_src/duplicate", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "dsg_src")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rec := httptest.NewRecorder()
	handler.duplicateDesign(rec, req)

	if rec.Result().StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Result().StatusCode)
	}
	if captured.SourceDesignID != "dsg_src" {
		t.Fatalf("expected source design id dsg_src, got %s", captured.SourceDesignID)
	}
	if captured.RequestedBy != "user-1" {
		t.Fatalf("expected requested_by user-1, got %s", captured.RequestedBy)
	}
	if captured.OverrideName == nil || *captured.OverrideName != "Copy Label" {
		t.Fatalf("expected override Copy Label, got %#v", captured.OverrideName)
	}

	var payload createDesignResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Design.ID != "dsg_copy" {
		t.Fatalf("unexpected design id %s", payload.Design.ID)
	}
}

func TestDesignHandlers_DuplicateDesign_NoOverride(t *testing.T) {
	var captured services.DuplicateDesignCommand
	stub := &stubDesignService{
		duplicateFn: func(_ context.Context, cmd services.DuplicateDesignCommand) (services.Design, error) {
			captured = cmd
			return services.Design{ID: "dsg_copy"}, nil
		},
	}
	handler := NewDesignHandlers(nil, stub)
	req := httptest.NewRequest(http.MethodPost, "/designs/dsg_src/duplicate", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "dsg_src")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rec := httptest.NewRecorder()
	handler.duplicateDesign(rec, req)

	if rec.Result().StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Result().StatusCode)
	}
	if captured.OverrideName != nil {
		t.Fatalf("expected no override name, got %#v", captured.OverrideName)
	}
}

func TestDesignHandlers_DuplicateDesign_InvalidJSON(t *testing.T) {
	handler := NewDesignHandlers(nil, &stubDesignService{})
	req := httptest.NewRequest(http.MethodPost, "/designs/dsg_src/duplicate", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "dsg_src")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rec := httptest.NewRecorder()
	handler.duplicateDesign(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Result().StatusCode)
	}
}

func TestDesignHandlers_DuplicateDesign_NotFound(t *testing.T) {
	stub := &stubDesignService{
		duplicateFn: func(context.Context, services.DuplicateDesignCommand) (services.Design, error) {
			return services.Design{}, services.ErrDesignNotFound
		},
	}
	handler := NewDesignHandlers(nil, stub)
	req := httptest.NewRequest(http.MethodPost, "/designs/dsg_missing/duplicate", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "dsg_missing")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rec := httptest.NewRecorder()
	handler.duplicateDesign(rec, req)

	if rec.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Result().StatusCode)
	}
}

func TestDesignHandlers_RequestAISuggestion_Success(t *testing.T) {
	var captured services.AISuggestionRequest
	stub := &stubDesignService{
		aiRequestFn: func(_ context.Context, req services.AISuggestionRequest) (services.AISuggestion, error) {
			captured = req
			return services.AISuggestion{
				ID:     "as_123",
				Status: "queued",
			}, nil
		},
	}
	handler := NewDesignHandlers(nil, stub)

	reqBody := `{
		"method": " balance ",
		"model": " glyph@2025 ",
		"prompt": " refine ",
		"idempotency_key": " idem-123 ",
		"priority": 10,
		"parameters": {"strength": 0.7},
		"metadata": {"channel":"app"}
	}`
	req := httptest.NewRequest(http.MethodPost, "/designs/dsg_src/ai-suggestions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "dsg_src")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rec := httptest.NewRecorder()
	handler.requestAISuggestion(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", res.StatusCode)
	}
	location := res.Header.Get("Location")
	if location != "/designs/dsg_src/ai-suggestions/as_123" {
		t.Fatalf("unexpected Location header: %s", location)
	}

	var payload aiSuggestionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.SuggestionID != "as_123" {
		t.Fatalf("expected suggestion id as_123, got %s", payload.SuggestionID)
	}
	if payload.Status != "queued" {
		t.Fatalf("expected status queued, got %s", payload.Status)
	}
	if payload.PollingURL != location {
		t.Fatalf("expected polling url to match location, got %s", payload.PollingURL)
	}

	if captured.DesignID != "dsg_src" {
		t.Fatalf("expected design id dsg_src, got %s", captured.DesignID)
	}
	if captured.Method != "balance" {
		t.Fatalf("expected method trimmed to balance, got %s", captured.Method)
	}
	if captured.Model != "glyph@2025" {
		t.Fatalf("expected model trimmed, got %s", captured.Model)
	}
	if captured.Prompt != "refine" {
		t.Fatalf("expected prompt trimmed, got %s", captured.Prompt)
	}
	if captured.IdempotencyKey != "idem-123" {
		t.Fatalf("expected idempotency key trimmed, got %s", captured.IdempotencyKey)
	}
	if captured.Priority != 10 {
		t.Fatalf("expected priority 10, got %d", captured.Priority)
	}
	if captured.ActorID != "user-1" {
		t.Fatalf("expected actor id user-1, got %s", captured.ActorID)
	}
	if captured.Parameters == nil || captured.Parameters["strength"] != 0.7 {
		t.Fatalf("expected parameters propagated, got %+v", captured.Parameters)
	}
	if captured.Metadata == nil || captured.Metadata["channel"] != "app" {
		t.Fatalf("expected metadata propagated, got %+v", captured.Metadata)
	}
}

func TestDesignHandlers_RequestAISuggestion_InvalidJSON(t *testing.T) {
	handler := NewDesignHandlers(nil, &stubDesignService{})
	req := httptest.NewRequest(http.MethodPost, "/designs/dsg_src/ai-suggestions", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "dsg_src")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rec := httptest.NewRecorder()
	handler.requestAISuggestion(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Result().StatusCode)
	}
}

func TestDesignHandlers_RequestAISuggestion_ServiceError(t *testing.T) {
	stub := &stubDesignService{
		aiRequestFn: func(context.Context, services.AISuggestionRequest) (services.AISuggestion, error) {
			return services.AISuggestion{}, services.ErrDesignInvalidInput
		},
	}
	handler := NewDesignHandlers(nil, stub)
	reqBody := `{"method":"balance","model":"glyph"}`
	req := httptest.NewRequest(http.MethodPost, "/designs/dsg_src/ai-suggestions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "dsg_src")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rec := httptest.NewRecorder()
	handler.requestAISuggestion(rec, req)
	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Result().StatusCode)
	}
}

func TestDesignHandlers_ListAISuggestions_Success(t *testing.T) {
	now := time.Date(2025, 7, 1, 9, 0, 0, 0, time.UTC)
	next := now.Add(2 * time.Minute)
	capturedFilter := services.AISuggestionFilter{}
	var capturedDesignID string

	stub := &stubDesignService{
		getFn: func(context.Context, string, services.DesignReadOptions) (services.Design, error) {
			return services.Design{ID: "dsg_1", OwnerID: "user-1"}, nil
		},
		listAISuggestionsFn: func(_ context.Context, designID string, filter services.AISuggestionFilter) (domain.CursorPage[services.AISuggestion], error) {
			capturedDesignID = designID
			capturedFilter = filter
			return domain.CursorPage[services.AISuggestion]{
				Items: []services.AISuggestion{
					{
						ID:       "as_ready",
						DesignID: "dsg_1",
						Method:   "balance",
						Status:   "proposed",
						Payload: map[string]any{
							"model":    "glyph-balancer@2025",
							"score":    0.92,
							"scores":   map[string]any{"balance": 0.95},
							"queuedAt": now.Format(time.RFC3339Nano),
							"parameters": map[string]any{
								"strength": 0.8,
							},
							"metadata": map[string]any{
								"channel": "app",
							},
							"tags": []any{"balanced", "featured"},
							"preview": map[string]any{
								"previewUrl":       "https://cdn.example/suggestion.png",
								"signedPreviewUrl": "https://signed.example/suggestion.png?token=abc",
								"diffUrl":          "https://cdn.example/suggestion-diff.png",
								"thumbnailUrl":     "https://cdn.example/suggestion-thumb.png",
								"assetRef":         "/assets/as_ready",
								"bucket":           "design-assets",
								"objectPath":       "ai/designs/dsg_1/as_ready.png",
								"svgUrl":           "https://cdn.example/suggestion.svg",
							},
							"diagnostics": []any{
								map[string]any{"code": "alignment", "severity": "warn", "detail": "Shifted baseline"},
							},
						},
						CreatedAt: now,
						UpdatedAt: next,
					},
				},
				NextPageToken: "token-123",
			}, nil
		},
	}

	handler := NewDesignHandlers(nil, stub)
	req := httptest.NewRequest(http.MethodGet, "/designs/dsg_1/ai-suggestions?status=completed&page_size=5", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "dsg_1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rec := httptest.NewRecorder()
	handler.listAISuggestions(rec, req)

	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Result().StatusCode)
	}
	if capturedDesignID != "dsg_1" {
		t.Fatalf("expected designID dsg_1, got %s", capturedDesignID)
	}
	if capturedFilter.Pagination.PageSize != 5 {
		t.Fatalf("expected page_size 5, got %d", capturedFilter.Pagination.PageSize)
	}
	if len(capturedFilter.Status) != 1 || capturedFilter.Status[0] != "completed" {
		t.Fatalf("expected status filter [completed], got %+v", capturedFilter.Status)
	}

	var payload aiSuggestionListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.NextPageToken != "token-123" {
		t.Fatalf("unexpected next page token: %s", payload.NextPageToken)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(payload.Items))
	}
	item := payload.Items[0]
	if item.SuggestionID != "as_ready" {
		t.Fatalf("unexpected suggestion id %s", item.SuggestionID)
	}
	if item.StatusCategory != "completed" {
		t.Fatalf("expected status category completed, got %s", item.StatusCategory)
	}
	if item.Summary == nil || item.Summary.Score == nil || *item.Summary.Score != 0.92 {
		t.Fatalf("expected summary score 0.92, got %+v", item.Summary)
	}
	if item.Summary == nil || len(item.Summary.Scores) == 0 || item.Summary.Scores["balance"] != 0.95 {
		t.Fatalf("expected balance score 0.95, got %+v", item.Summary)
	}
	if item.Preview == nil || item.Preview.SignedPreviewURL != "https://signed.example/suggestion.png?token=abc" {
		t.Fatalf("expected signed preview url, got %+v", item.Preview)
	}
	if len(item.Diagnostics) != 1 || item.Diagnostics[0].Code != "alignment" {
		t.Fatalf("expected diagnostics, got %+v", item.Diagnostics)
	}
	if item.Parameters == nil || item.Parameters["strength"] != 0.8 {
		t.Fatalf("expected parameters propagated, got %+v", item.Parameters)
	}
	if item.Metadata == nil || item.Metadata["channel"] != "app" {
		t.Fatalf("expected metadata propagated, got %+v", item.Metadata)
	}
	if len(item.Tags) != 2 {
		t.Fatalf("expected tags populated, got %+v", item.Tags)
	}
	if item.CreatedAt != formatTime(now) {
		t.Fatalf("expected createdAt %s, got %s", formatTime(now), item.CreatedAt)
	}
	if item.UpdatedAt != formatTime(next) {
		t.Fatalf("expected updatedAt %s, got %s", formatTime(next), item.UpdatedAt)
	}
	if item.Payload == nil {
		t.Fatalf("expected payload to be preserved")
	}
}

func TestDesignHandlers_GetAISuggestion_Success(t *testing.T) {
	now := time.Date(2025, 7, 2, 10, 0, 0, 0, time.UTC)
	expires := now.Add(24 * time.Hour)

	stub := &stubDesignService{
		getFn: func(context.Context, string, services.DesignReadOptions) (services.Design, error) {
			return services.Design{ID: "dsg_1", OwnerID: "user-1"}, nil
		},
		getAISuggestionFn: func(context.Context, string, string) (services.AISuggestion, error) {
			return services.AISuggestion{
				ID:       "as_detail",
				DesignID: "dsg_1",
				Method:   "balance",
				Status:   "queued",
				Payload: map[string]any{
					"jobId":  "aj_55",
					"prompt": "Balance glyph",
					"result": map[string]any{
						"score": 0.88,
						"scores": map[string]any{
							"legibility": 0.9,
						},
						"preview": map[string]any{
							"bucket":     "design-assets",
							"objectPath": "ai/dsg_1/as_detail.png",
						},
						"diagnostics": []any{
							map[string]any{"code": "kerning", "detail": "Tight spacing"},
						},
					},
				},
				CreatedAt: now,
				UpdatedAt: now,
				ExpiresAt: &expires,
			}, nil
		},
	}

	handler := NewDesignHandlers(nil, stub)
	req := httptest.NewRequest(http.MethodGet, "/designs/dsg_1/ai-suggestions/as_detail", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "dsg_1")
	routeCtx.URLParams.Add("suggestionID", "as_detail")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rec := httptest.NewRecorder()
	handler.getAISuggestion(rec, req)

	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Result().StatusCode)
	}

	var payload aiSuggestionPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.SuggestionID != "as_detail" {
		t.Fatalf("expected suggestion id as_detail, got %s", payload.SuggestionID)
	}
	if payload.JobRef != "/aiJobs/aj_55" {
		t.Fatalf("expected job ref /aiJobs/aj_55, got %s", payload.JobRef)
	}
	if payload.StatusCategory != "queued" {
		t.Fatalf("expected status category queued, got %s", payload.StatusCategory)
	}
	if payload.Summary == nil || payload.Summary.Score == nil || *payload.Summary.Score != 0.88 {
		t.Fatalf("expected summary score 0.88, got %+v", payload.Summary)
	}
	if payload.Preview == nil || payload.Preview.PreviewURL != "https://storage.googleapis.com/design-assets/ai/dsg_1/as_detail.png" {
		t.Fatalf("expected preview url built from storage path, got %+v", payload.Preview)
	}
	if len(payload.Diagnostics) != 1 || payload.Diagnostics[0].Code != "kerning" {
		t.Fatalf("expected diagnostics, got %+v", payload.Diagnostics)
	}
	if payload.ExpiresAt != formatTime(expires) {
		t.Fatalf("expected expiresAt %s, got %s", formatTime(expires), payload.ExpiresAt)
	}
}

func TestDesignHandlers_AcceptAISuggestion_Success(t *testing.T) {
	var captured services.AISuggestionStatusCommand
	stub := &stubDesignService{
		updateAISuggestionStatusFn: func(ctx context.Context, cmd services.AISuggestionStatusCommand) (services.AISuggestion, error) {
			captured = cmd
			return services.AISuggestion{
				ID:       "as_1",
				DesignID: "dsg_1",
				Status:   "accepted",
				Payload: map[string]any{
					"acceptedAt": "2025-01-01T00:00:00Z",
				},
			}, nil
		},
	}

	handler := NewDesignHandlers(nil, stub)
	req := httptest.NewRequest(http.MethodPost, "/designs/dsg_1/ai-suggestions/as_1:accept", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "dsg_1")
	routeCtx.URLParams.Add("suggestionID", "as_1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rec := httptest.NewRecorder()
	handler.acceptAISuggestion(rec, req)

	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Result().StatusCode)
	}
	if captured.Action != "accept" || captured.ActorID != "user-1" || captured.DesignID != "dsg_1" || captured.SuggestionID != "as_1" {
		t.Fatalf("unexpected command captured %#v", captured)
	}
	if captured.Reason != nil {
		t.Fatalf("expected nil reason for accept")
	}

	var payload aiSuggestionPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.SuggestionID != "as_1" || payload.Status != "accepted" {
		t.Fatalf("unexpected payload %#v", payload)
	}
}

func TestDesignHandlers_RejectAISuggestion_WithReason(t *testing.T) {
	var captured services.AISuggestionStatusCommand
	stub := &stubDesignService{
		updateAISuggestionStatusFn: func(ctx context.Context, cmd services.AISuggestionStatusCommand) (services.AISuggestion, error) {
			captured = cmd
			return services.AISuggestion{
				ID:       "as_reject",
				DesignID: "dsg_1",
				Status:   "rejected",
				Payload: map[string]any{
					"rejectionReason": "worse_quality",
				},
			}, nil
		},
	}

	handler := NewDesignHandlers(nil, stub)
	req := httptest.NewRequest(http.MethodPost, "/designs/dsg_1/ai-suggestions/as_reject:reject", strings.NewReader(`{"reason":"WORSE_QUALITY"}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "dsg_1")
	routeCtx.URLParams.Add("suggestionID", "as_reject")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rec := httptest.NewRecorder()
	handler.rejectAISuggestion(rec, req)

	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Result().StatusCode)
	}
	if captured.Action != "reject" || captured.ActorID != "user-1" || captured.DesignID != "dsg_1" || captured.SuggestionID != "as_reject" {
		t.Fatalf("unexpected command %#v", captured)
	}
	if captured.Reason == nil || *captured.Reason != "worse_quality" {
		t.Fatalf("expected normalized reason worse_quality, got %v", captured.Reason)
	}

	var payload aiSuggestionPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Status != "rejected" || payload.RejectionReason != "worse_quality" {
		t.Fatalf("unexpected payload %#v", payload)
	}
}

func TestDesignHandlers_RejectAISuggestion_InvalidJSON(t *testing.T) {
	stub := &stubDesignService{}
	handler := NewDesignHandlers(nil, stub)

	req := httptest.NewRequest(http.MethodPost, "/designs/dsg_1/ai-suggestions/as_1:reject", strings.NewReader(`{"reason":123}`))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "dsg_1")
	routeCtx.URLParams.Add("suggestionID", "as_1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rec := httptest.NewRecorder()
	handler.rejectAISuggestion(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid JSON, got %d", rec.Result().StatusCode)
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

func TestDesignHandlers_ListDesignVersions_Success(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	var capturedFilter services.DesignVersionListFilter
	stub := &stubDesignService{
		getFn: func(context.Context, string, services.DesignReadOptions) (services.Design, error) {
			return services.Design{ID: "dsg_1", OwnerID: "user-1"}, nil
		},
		listVersionsFn: func(_ context.Context, designID string, filter services.DesignVersionListFilter) (domain.CursorPage[services.DesignVersion], error) {
			if designID != "dsg_1" {
				t.Fatalf("expected designID dsg_1, got %s", designID)
			}
			capturedFilter = filter
			return domain.CursorPage[services.DesignVersion]{
				Items: []services.DesignVersion{
					{
						ID:      "ver_1",
						Version: 1,
						Snapshot: map[string]any{
							"label": "Initial",
							"assets": map[string]any{
								"previewUrl": "https://cdn.example/ver_1.png",
							},
						},
						CreatedAt: now,
						CreatedBy: "user-1",
					},
				},
				NextPageToken: "next-token",
			}, nil
		},
	}

	handler := NewDesignHandlers(nil, stub)
	req := httptest.NewRequest(http.MethodGet, "/designs/dsg_1/versions?page_size=10", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "dsg_1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	res := httptest.NewRecorder()
	handler.listDesignVersions(res, req)

	if res.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.Result().StatusCode)
	}
	if capturedFilter.IncludeAssets {
		t.Fatalf("expected IncludeAssets to be false")
	}
	if capturedFilter.Pagination.PageSize != 10 {
		t.Fatalf("expected page_size 10, got %d", capturedFilter.Pagination.PageSize)
	}

	var payload designVersionListResponse
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.NextPageToken != "next-token" {
		t.Fatalf("unexpected next page token: %s", payload.NextPageToken)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(payload.Items))
	}
	if payload.Items[0].Assets != nil {
		t.Fatalf("expected assets omitted when includeAssets=false")
	}
	if payload.Items[0].Snapshot == nil {
		t.Fatalf("expected snapshot to be present")
	}
	if _, ok := payload.Items[0].Snapshot["assets"]; ok {
		t.Fatalf("expected assets removed from snapshot")
	}
}

func TestDesignHandlers_ListDesignVersions_IncludeAssets(t *testing.T) {
	stub := &stubDesignService{
		getFn: func(context.Context, string, services.DesignReadOptions) (services.Design, error) {
			return services.Design{ID: "dsg_1", OwnerID: "user-1"}, nil
		},
		listVersionsFn: func(_ context.Context, designID string, filter services.DesignVersionListFilter) (domain.CursorPage[services.DesignVersion], error) {
			if !filter.IncludeAssets {
				t.Fatalf("expected IncludeAssets to be true")
			}
			return domain.CursorPage[services.DesignVersion]{
				Items: []services.DesignVersion{
					{
						ID:      "ver_2",
						Version: 2,
						Snapshot: map[string]any{
							"label": "Updated",
							"assets": map[string]any{
								"previewUrl": "https://cdn.example/ver_2.png",
							},
						},
					},
				},
			}, nil
		},
	}

	handler := NewDesignHandlers(nil, stub)
	req := httptest.NewRequest(http.MethodGet, "/designs/dsg_1/versions?includeAssets=true", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "dsg_1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	res := httptest.NewRecorder()
	handler.listDesignVersions(res, req)

	if res.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.Result().StatusCode)
	}

	var payload designVersionListResponse
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(payload.Items))
	}
	if payload.Items[0].Assets == nil || payload.Items[0].Assets.PreviewURL != "https://cdn.example/ver_2.png" {
		t.Fatalf("expected assets with previewUrl, got %+v", payload.Items[0].Assets)
	}
	if payload.Items[0].Snapshot == nil {
		t.Fatalf("expected snapshot to be present")
	}
	if _, ok := payload.Items[0].Snapshot["assets"]; !ok {
		t.Fatalf("expected assets retained in snapshot")
	}
}

func TestDesignHandlers_GetDesignVersion_Success(t *testing.T) {
	now := time.Now().UTC()
	var capturedOpts services.DesignVersionReadOptions
	stub := &stubDesignService{
		getFn: func(context.Context, string, services.DesignReadOptions) (services.Design, error) {
			return services.Design{ID: "dsg_1", OwnerID: "user-1"}, nil
		},
		getVersionFn: func(_ context.Context, designID, versionID string, opts services.DesignVersionReadOptions) (services.DesignVersion, error) {
			if designID != "dsg_1" || versionID != "ver_1" {
				t.Fatalf("unexpected ids: %s / %s", designID, versionID)
			}
			capturedOpts = opts
			return services.DesignVersion{
				ID:      versionID,
				Version: 1,
				Snapshot: map[string]any{
					"label": "Initial",
					"assets": map[string]any{
						"previewUrl": "https://cdn.example/ver_1.png",
					},
				},
				CreatedAt: now,
				CreatedBy: "user-1",
			}, nil
		},
	}

	handler := NewDesignHandlers(nil, stub)
	req := httptest.NewRequest(http.MethodGet, "/designs/dsg_1/versions/ver_1?includeAssets=true", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "dsg_1")
	routeCtx.URLParams.Add("versionID", "ver_1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	res := httptest.NewRecorder()
	handler.getDesignVersion(res, req)

	if res.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.Result().StatusCode)
	}
	if !capturedOpts.IncludeAssets {
		t.Fatalf("expected IncludeAssets to be true")
	}

	var payload designVersionPayload
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.ID != "ver_1" {
		t.Fatalf("unexpected payload id: %s", payload.ID)
	}
	if payload.Assets == nil || payload.Assets.PreviewURL != "https://cdn.example/ver_1.png" {
		t.Fatalf("expected assets with previewUrl, got %+v", payload.Assets)
	}
	if payload.Snapshot == nil {
		t.Fatalf("expected snapshot")
	}
}

func TestDesignHandlers_GetDesignVersion_NotOwner(t *testing.T) {
	stub := &stubDesignService{
		getFn: func(context.Context, string, services.DesignReadOptions) (services.Design, error) {
			return services.Design{ID: "dsg_1", OwnerID: "someone-else"}, nil
		},
	}

	handler := NewDesignHandlers(nil, stub)
	req := httptest.NewRequest(http.MethodGet, "/designs/dsg_1/versions/ver_1", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "dsg_1")
	routeCtx.URLParams.Add("versionID", "ver_1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	res := httptest.NewRecorder()
	handler.getDesignVersion(res, req)

	if res.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", res.Result().StatusCode)
	}
}

func TestDesignHandlers_UpdateDesign_Success(t *testing.T) {
	updatedAt := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	var captured services.UpdateDesignCommand
	stub := &stubDesignService{
		getFn: func(context.Context, string, services.DesignReadOptions) (services.Design, error) {
			return services.Design{
				ID:        "dsg_1",
				OwnerID:   "user-1",
				UpdatedAt: updatedAt,
			}, nil
		},
		updateFn: func(_ context.Context, cmd services.UpdateDesignCommand) (services.Design, error) {
			captured = cmd
			return services.Design{
				ID:               "dsg_1",
				OwnerID:          "user-1",
				Label:            "Updated",
				Status:           services.DesignStatusReady,
				Version:          2,
				CurrentVersionID: "ver_2",
				UpdatedAt:        updatedAt.Add(time.Hour),
			}, nil
		},
	}

	handler := NewDesignHandlers(nil, stub)
	body := `{"label":"Updated","status":"ready","snapshot":{"label":"Updated"}}`
	req := httptest.NewRequest(http.MethodPut, "/designs/dsg_1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Unmodified-Since", updatedAt.Format(time.RFC3339))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "dsg_1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	res := httptest.NewRecorder()
	handler.updateDesign(res, req)

	if res.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.Result().StatusCode)
	}
	if captured.DesignID != "dsg_1" || captured.Label == nil || *captured.Label != "Updated" {
		t.Fatalf("captured command mismatch: %+v", captured)
	}
	if captured.Status == nil || *captured.Status != "ready" {
		t.Fatalf("status not propagated: %+v", captured)
	}
	if captured.ExpectedUpdatedAt == nil || !captured.ExpectedUpdatedAt.Equal(updatedAt) {
		t.Fatalf("expected updatedAt to propagate")
	}
	if captured.Snapshot == nil || captured.Snapshot["label"] != "Updated" {
		t.Fatalf("snapshot not forwarded: %+v", captured.Snapshot)
	}

	var payload designPayload
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Label != "Updated" || payload.Status != string(services.DesignStatusReady) {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestDesignHandlers_UpdateDesign_Conflict(t *testing.T) {
	stub := &stubDesignService{
		getFn: func(context.Context, string, services.DesignReadOptions) (services.Design, error) {
			return services.Design{ID: "dsg_1", OwnerID: "user-1", UpdatedAt: time.Unix(0, 0).UTC()}, nil
		},
		updateFn: func(context.Context, services.UpdateDesignCommand) (services.Design, error) {
			return services.Design{}, services.ErrDesignConflict
		},
	}

	handler := NewDesignHandlers(nil, stub)
	req := httptest.NewRequest(http.MethodPut, "/designs/dsg_1", strings.NewReader(`{"status":"ready"}`))
	req.Header.Set("If-Unmodified-Since", time.Unix(0, 0).UTC().Format(time.RFC3339))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "dsg_1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	res := httptest.NewRecorder()
	handler.updateDesign(res, req)

	if res.Result().StatusCode != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", res.Result().StatusCode)
	}
}

func TestDesignHandlers_DeleteDesign_Success(t *testing.T) {
	updatedAt := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	var captured services.DeleteDesignCommand
	stub := &stubDesignService{
		getFn: func(context.Context, string, services.DesignReadOptions) (services.Design, error) {
			return services.Design{ID: "dsg_1", OwnerID: "user-1", UpdatedAt: updatedAt}, nil
		},
		deleteFn: func(_ context.Context, cmd services.DeleteDesignCommand) error {
			captured = cmd
			return nil
		},
	}

	handler := NewDesignHandlers(nil, stub)
	req := httptest.NewRequest(http.MethodDelete, "/designs/dsg_1", nil)
	req.Header.Set("If-Unmodified-Since", updatedAt.Format(time.RFC3339))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "dsg_1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	res := httptest.NewRecorder()
	handler.deleteDesign(res, req)

	if res.Result().StatusCode != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", res.Result().StatusCode)
	}
	if captured.DesignID != "dsg_1" || !captured.SoftDelete {
		t.Fatalf("captured delete command incorrect: %+v", captured)
	}
	if captured.ExpectedUpdatedAt == nil || !captured.ExpectedUpdatedAt.Equal(updatedAt) {
		t.Fatalf("expected updatedAt propagated")
	}
}

func TestDesignHandlers_DeleteDesign_NotOwner(t *testing.T) {
	stub := &stubDesignService{
		getFn: func(context.Context, string, services.DesignReadOptions) (services.Design, error) {
			return services.Design{ID: "dsg_1", OwnerID: "other", UpdatedAt: time.Now().UTC()}, nil
		},
	}

	handler := NewDesignHandlers(nil, stub)
	req := httptest.NewRequest(http.MethodDelete, "/designs/dsg_1", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "dsg_1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	res := httptest.NewRecorder()
	handler.deleteDesign(res, req)

	if res.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", res.Result().StatusCode)
	}
}

func TestDesignHandlers_CheckRegistrability_Success(t *testing.T) {
	now := time.Date(2025, time.January, 3, 9, 0, 0, 0, time.UTC)
	expires := now.Add(2 * time.Hour)
	score := 0.82

	stub := &stubDesignService{
		registrabilityFn: func(ctx context.Context, cmd services.RegistrabilityCheckCommand) (services.RegistrabilityCheckResult, error) {
			if cmd.DesignID != "dsg_123" {
				t.Fatalf("unexpected design id: %s", cmd.DesignID)
			}
			if cmd.UserID != "user-1" {
				t.Fatalf("unexpected user id: %s", cmd.UserID)
			}
			if cmd.Locale != "en" {
				t.Fatalf("unexpected locale: %s", cmd.Locale)
			}
			return services.RegistrabilityCheckResult{
				DesignID:    "dsg_123",
				Status:      "pass",
				Passed:      true,
				Score:       &score,
				Reasons:     []string{"all checks passed"},
				RequestedAt: now,
				ExpiresAt:   &expires,
			}, nil
		},
	}

	handler := NewDesignHandlers(nil, stub)
	handler.registrabilityLimiter = nil

	req := httptest.NewRequest(http.MethodPost, "/designs/dsg_123:registrability-check?locale=en", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "dsg_123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rec := httptest.NewRecorder()
	handler.checkRegistrability(rec, req)

	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Result().StatusCode)
	}

	var payload registrabilityCheckResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !payload.Registrable || payload.Status != "pass" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if payload.Score == nil || *payload.Score != score {
		t.Fatalf("expected score %.2f, got %+v", score, payload.Score)
	}
	if len(payload.Diagnostics) != 1 || payload.Diagnostics[0] != "all checks passed" {
		t.Fatalf("unexpected diagnostics: %+v", payload.Diagnostics)
	}
	if payload.RequestedAt == "" || payload.ExpiresAt == "" {
		t.Fatalf("expected timestamps in payload: %+v", payload)
	}
}

func TestDesignHandlers_CheckRegistrability_RateLimited(t *testing.T) {
	stub := &stubDesignService{}
	handler := NewDesignHandlers(nil, stub)
	handler.registrabilityLimiter = rateLimiterFunc(func(string) bool { return false })

	req := httptest.NewRequest(http.MethodPost, "/designs/dsg_456:registrability-check", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-9"}))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "dsg_456")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rec := httptest.NewRecorder()
	handler.checkRegistrability(rec, req)

	if rec.Result().StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", rec.Result().StatusCode)
	}
}

type stubDesignService struct {
	createFn                   func(context.Context, services.CreateDesignCommand) (services.Design, error)
	getFn                      func(context.Context, string, services.DesignReadOptions) (services.Design, error)
	listFn                     func(context.Context, services.DesignListFilter) (domain.CursorPage[services.Design], error)
	listVersionsFn             func(context.Context, string, services.DesignVersionListFilter) (domain.CursorPage[services.DesignVersion], error)
	getVersionFn               func(context.Context, string, string, services.DesignVersionReadOptions) (services.DesignVersion, error)
	updateFn                   func(context.Context, services.UpdateDesignCommand) (services.Design, error)
	deleteFn                   func(context.Context, services.DeleteDesignCommand) error
	duplicateFn                func(context.Context, services.DuplicateDesignCommand) (services.Design, error)
	aiRequestFn                func(context.Context, services.AISuggestionRequest) (services.AISuggestion, error)
	listAISuggestionsFn        func(context.Context, string, services.AISuggestionFilter) (domain.CursorPage[services.AISuggestion], error)
	getAISuggestionFn          func(context.Context, string, string) (services.AISuggestion, error)
	updateAISuggestionStatusFn func(context.Context, services.AISuggestionStatusCommand) (services.AISuggestion, error)
	registrabilityFn           func(context.Context, services.RegistrabilityCheckCommand) (services.RegistrabilityCheckResult, error)
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

func (s *stubDesignService) ListDesignVersions(ctx context.Context, designID string, filter services.DesignVersionListFilter) (domain.CursorPage[services.DesignVersion], error) {
	if s.listVersionsFn != nil {
		return s.listVersionsFn(ctx, designID, filter)
	}
	return domain.CursorPage[services.DesignVersion]{}, nil
}

func (s *stubDesignService) GetDesignVersion(ctx context.Context, designID, versionID string, opts services.DesignVersionReadOptions) (services.DesignVersion, error) {
	if s.getVersionFn != nil {
		return s.getVersionFn(ctx, designID, versionID, opts)
	}
	return services.DesignVersion{}, nil
}

func (s *stubDesignService) UpdateDesign(ctx context.Context, cmd services.UpdateDesignCommand) (services.Design, error) {
	if s.updateFn != nil {
		return s.updateFn(ctx, cmd)
	}
	return services.Design{}, nil
}

func (s *stubDesignService) DeleteDesign(ctx context.Context, cmd services.DeleteDesignCommand) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, cmd)
	}
	return nil
}

func (s *stubDesignService) DuplicateDesign(ctx context.Context, cmd services.DuplicateDesignCommand) (services.Design, error) {
	if s.duplicateFn != nil {
		return s.duplicateFn(ctx, cmd)
	}
	return services.Design{}, nil
}

func (s *stubDesignService) RequestAISuggestion(ctx context.Context, req services.AISuggestionRequest) (services.AISuggestion, error) {
	if s.aiRequestFn != nil {
		return s.aiRequestFn(ctx, req)
	}
	return services.AISuggestion{}, nil
}

func (s *stubDesignService) ListAISuggestions(ctx context.Context, designID string, filter services.AISuggestionFilter) (domain.CursorPage[services.AISuggestion], error) {
	if s.listAISuggestionsFn != nil {
		return s.listAISuggestionsFn(ctx, designID, filter)
	}
	return domain.CursorPage[services.AISuggestion]{}, nil
}

func (s *stubDesignService) GetAISuggestion(ctx context.Context, designID, suggestionID string) (services.AISuggestion, error) {
	if s.getAISuggestionFn != nil {
		return s.getAISuggestionFn(ctx, designID, suggestionID)
	}
	return services.AISuggestion{}, nil
}

func (s *stubDesignService) UpdateAISuggestionStatus(ctx context.Context, cmd services.AISuggestionStatusCommand) (services.AISuggestion, error) {
	if s.updateAISuggestionStatusFn != nil {
		return s.updateAISuggestionStatusFn(ctx, cmd)
	}
	return services.AISuggestion{}, nil
}

func (s *stubDesignService) RequestRegistrabilityCheck(ctx context.Context, cmd services.RegistrabilityCheckCommand) (services.RegistrabilityCheckResult, error) {
	if s.registrabilityFn != nil {
		return s.registrabilityFn(ctx, cmd)
	}
	return services.RegistrabilityCheckResult{}, nil
}

type rateLimiterFunc func(string) bool

func (f rateLimiterFunc) Allow(key string) bool {
	return f(key)
}
