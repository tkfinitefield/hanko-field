package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/services"
)

func TestAdminCatalogHandlers_CreateTemplate(t *testing.T) {
	svc := &stubCatalogService{}
	now := time.Date(2024, time.March, 3, 9, 0, 0, 0, time.UTC)
	svc.adminUpsertResp = services.Template{
		TemplateSummary: services.TemplateSummary{
			ID:               "tpl_new",
			Name:             "New Template",
			Description:      "Desc",
			Category:         "round",
			Style:            "classic",
			Tags:             []string{"a", "b"},
			PreviewImagePath: "previews/new.png",
			IsPublished:      true,
			CreatedAt:        now,
			UpdatedAt:        now,
			PublishedAt:      now,
			Version:          1,
		},
		SVGPath: "vectors/new.svg",
		Draft: services.TemplateDraft{
			Notes:     "internal",
			UpdatedAt: now,
			UpdatedBy: "admin",
			Metadata:  map[string]any{"preview": true},
		},
	}

	handler := NewAdminCatalogHandlers(nil, svc)
	router := chi.NewRouter()
	handler.Routes(router)

	body := map[string]any{
		"id":                 "tpl_new",
		"name":               "New Template",
		"description":        "Desc",
		"category":           "round",
		"style":              "classic",
		"tags":               []string{"a", "b"},
		"preview_image_path": "previews/new.png",
		"svg_path":           "vectors/new.svg",
		"is_published":       true,
		"draft": map[string]any{
			"notes": "internal",
		},
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/catalog/templates", bytes.NewReader(payload))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "admin", Roles: []string{auth.RoleAdmin}}))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}
	if svc.adminUpsertCmd.ActorID != "admin" {
		t.Fatalf("expected actor admin, got %s", svc.adminUpsertCmd.ActorID)
	}
	if svc.adminUpsertCmd.Template.ID != "tpl_new" {
		t.Fatalf("expected template id to remain tpl_new, got %s", svc.adminUpsertCmd.Template.ID)
	}
	var decoded adminTemplateResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if decoded.Version != 1 || decoded.Draft == nil {
		t.Fatalf("expected draft payload with version, got %#v", decoded)
	}
}

func TestAdminCatalogHandlers_UpdateTemplateUsesPathID(t *testing.T) {
	svc := &stubCatalogService{}
	handler := NewAdminCatalogHandlers(nil, svc)
	router := chi.NewRouter()
	handler.Routes(router)

	body := map[string]any{
		"id":                 "tpl_other",
		"name":               "Updated",
		"category":           "round",
		"style":              "modern",
		"tags":               []string{},
		"preview_image_path": "previews/img.png",
		"svg_path":           "vectors/img.svg",
		"is_published":       false,
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/catalog/templates/tpl_123", bytes.NewReader(payload))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "staff", Roles: []string{auth.RoleAdmin}}))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if svc.adminUpsertCmd.Template.ID != "tpl_123" {
		t.Fatalf("expected path id tpl_123, got %s", svc.adminUpsertCmd.Template.ID)
	}
}

func TestAdminCatalogHandlers_DeleteTemplateRequiresIdentity(t *testing.T) {
	svc := &stubCatalogService{}
	handler := NewAdminCatalogHandlers(nil, svc)
	router := chi.NewRouter()
	handler.Routes(router)

	req := httptest.NewRequest(http.MethodDelete, "/catalog/templates/tpl_x", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when identity missing, got %d", resp.Code)
	}
}

func TestAdminCatalogHandlers_DeleteTemplateCallsService(t *testing.T) {
	svc := &stubCatalogService{}
	handler := NewAdminCatalogHandlers(nil, svc)
	router := chi.NewRouter()
	handler.Routes(router)

	req := httptest.NewRequest(http.MethodDelete, "/catalog/templates/tpl_del", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "admin", Roles: []string{auth.RoleAdmin}}))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.Code)
	}
	if svc.adminDeleteCmd.TemplateID != "tpl_del" {
		t.Fatalf("expected delete template id tpl_del, got %s", svc.adminDeleteCmd.TemplateID)
	}
	if svc.adminDeleteCmd.ActorID != "admin" {
		t.Fatalf("expected actor admin, got %s", svc.adminDeleteCmd.ActorID)
	}
}

func TestAdminCatalogHandlers_ServiceUnavailable(t *testing.T) {
	handler := NewAdminCatalogHandlers(nil, nil)
	router := chi.NewRouter()
	handler.Routes(router)

	req := httptest.NewRequest(http.MethodPost, "/catalog/templates", bytes.NewBufferString(`{"name":"Missing"}`))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "admin", Roles: []string{auth.RoleAdmin}}))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when service missing, got %d", resp.Code)
	}
}

func TestAdminCatalogHandlers_InvalidPayload(t *testing.T) {
	svc := &stubCatalogService{}
	handler := NewAdminCatalogHandlers(nil, svc)
	router := chi.NewRouter()
	handler.Routes(router)

	req := httptest.NewRequest(http.MethodPost, "/catalog/templates", bytes.NewBufferString(`{"id":`))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "admin", Roles: []string{auth.RoleAdmin}}))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid json, got %d", resp.Code)
	}
}

func TestAdminCatalogHandlers_CreateFont(t *testing.T) {
	svc := &stubCatalogService{}
	now := time.Date(2024, time.May, 5, 12, 0, 0, 0, time.UTC)
	svc.adminFontUpsertResp = services.FontSummary{
		ID:               "tensho-regular",
		Slug:             "tensho-regular",
		DisplayName:      "Tensho Regular",
		Family:           "Tensho",
		Weight:           "regular",
		Scripts:          []string{"kanji"},
		PreviewImagePath: "fonts/tensho.png",
		LetterSpacing:    0.1,
		IsPremium:        true,
		SupportedWeights: []string{"regular"},
		License: services.FontLicense{
			Name:          "Commercial",
			URL:           "https://example.com/license",
			AllowedUsages: []string{"app"},
		},
		IsPublished: true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	handler := NewAdminCatalogHandlers(nil, svc)
	router := chi.NewRouter()
	handler.Routes(router)
	body := map[string]any{
		"display_name":       "Tensho Regular",
		"family":             "Tensho",
		"weight":             "Regular",
		"scripts":            []string{"kanji"},
		"preview_image_path": "fonts/tensho.png",
		"letter_spacing":     0.1,
		"is_premium":         true,
		"supported_weights":  []string{"Regular"},
		"license": map[string]any{
			"name":           "Commercial",
			"url":            "https://example.com/license",
			"allowed_usages": []string{"app"},
		},
		"is_published": true,
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/catalog/fonts", bytes.NewReader(payload))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "admin", Roles: []string{auth.RoleAdmin}}))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}
	if svc.adminFontUpsertCmd.ActorID != "admin" {
		t.Fatalf("expected actor admin, got %s", svc.adminFontUpsertCmd.ActorID)
	}
	if svc.adminFontUpsertCmd.Font.Family != "Tensho" {
		t.Fatalf("expected family propagated, got %s", svc.adminFontUpsertCmd.Font.Family)
	}
	if svc.adminFontUpsertCmd.Font.Weight != "regular" {
		t.Fatalf("expected weight normalized to lowercase, got %s", svc.adminFontUpsertCmd.Font.Weight)
	}
	var decoded adminFontResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if decoded.ID != "tensho-regular" || decoded.Slug != "tensho-regular" {
		t.Fatalf("expected slug response, got %#v", decoded)
	}
	if len(decoded.License.AllowedUsages) != 1 || decoded.License.AllowedUsages[0] != "app" {
		t.Fatalf("expected allowed usages echoed, got %#v", decoded.License.AllowedUsages)
	}
}

func TestAdminCatalogHandlers_UpdateFontUsesPathID(t *testing.T) {
	svc := &stubCatalogService{}
	handler := NewAdminCatalogHandlers(nil, svc)
	router := chi.NewRouter()
	handler.Routes(router)
	body := map[string]any{
		"id":                 "other",
		"display_name":       "Updated",
		"family":             "Tensho",
		"weight":             "Regular",
		"scripts":            []string{"kanji"},
		"preview_image_path": "fonts/tensho.png",
		"license": map[string]any{
			"name":           "Commercial",
			"url":            "https://example.com/license",
			"allowed_usages": []string{"app"},
		},
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/catalog/fonts/tensho-regular", bytes.NewReader(payload))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "editor", Roles: []string{auth.RoleAdmin}}))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if svc.adminFontUpsertCmd.Font.ID != "tensho-regular" {
		t.Fatalf("expected path id to override, got %s", svc.adminFontUpsertCmd.Font.ID)
	}
}

func TestAdminCatalogHandlers_FontValidationError(t *testing.T) {
	svc := &stubCatalogService{adminFontUpsertErr: services.ErrCatalogInvalidInput}
	handler := NewAdminCatalogHandlers(nil, svc)
	router := chi.NewRouter()
	handler.Routes(router)
	req := httptest.NewRequest(http.MethodPost, "/catalog/fonts", bytes.NewBufferString(`{"display_name":"Missing"}`))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "admin", Roles: []string{auth.RoleAdmin}}))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAdminCatalogHandlers_DeleteFontErrors(t *testing.T) {
	svc := &stubCatalogService{adminFontDeleteErr: services.ErrCatalogFontInUse}
	handler := NewAdminCatalogHandlers(nil, svc)
	router := chi.NewRouter()
	handler.Routes(router)
	req := httptest.NewRequest(http.MethodDelete, "/catalog/fonts/tensho-regular", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "admin", Roles: []string{auth.RoleAdmin}}))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.Code)
	}
	svc.adminFontDeleteErr = services.ErrCatalogInvalidInput
	req = httptest.NewRequest(http.MethodDelete, "/catalog/fonts/tensho-regular", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "admin", Roles: []string{auth.RoleAdmin}}))
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}
