package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

type stubDesignService struct {
	createFn func(context.Context, services.CreateDesignCommand) (services.Design, error)
}

func (s *stubDesignService) CreateDesign(ctx context.Context, cmd services.CreateDesignCommand) (services.Design, error) {
	if s.createFn != nil {
		return s.createFn(ctx, cmd)
	}
	return services.Design{}, nil
}

func (s *stubDesignService) GetDesign(context.Context, string, services.DesignReadOptions) (services.Design, error) {
	return services.Design{}, nil
}

func (s *stubDesignService) ListDesigns(context.Context, services.DesignListFilter) (domain.CursorPage[services.Design], error) {
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
