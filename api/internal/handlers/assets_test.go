package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/services"
)

func TestAssetHandlers_IssueSignedUpload_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	stub := &stubAssetService{
		uploadResponse: domain.SignedAssetResponse{
			AssetID:   "asset_123",
			URL:       "https://storage.example/upload",
			Method:    "PUT",
			ExpiresAt: now,
			Headers: map[string]string{
				"Content-Type": "image/png",
			},
		},
	}

	handler := NewAssetHandlers(nil, stub)
	payload := map[string]any{
		"kind":       "png",
		"purpose":    "preview",
		"mime_type":  "image/png",
		"file_name":  "preview.png",
		"size_bytes": 2048,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/assets:signed-upload", bytes.NewReader(body))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user_123"}))

	rr := httptest.NewRecorder()
	handler.issueSignedUpload(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp signedUploadResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.AssetID != stub.uploadResponse.AssetID {
		t.Fatalf("expected asset id %s, got %s", stub.uploadResponse.AssetID, resp.AssetID)
	}
	if resp.UploadURL != stub.uploadResponse.URL {
		t.Fatalf("expected upload url %s, got %s", stub.uploadResponse.URL, resp.UploadURL)
	}
	if resp.ExpiresAt != now.Format(time.RFC3339) {
		t.Fatalf("expected expires at %s, got %s", now.Format(time.RFC3339), resp.ExpiresAt)
	}
	if stub.uploadCalls != 1 {
		t.Fatalf("expected service called once, got %d", stub.uploadCalls)
	}
	if stub.lastUploadCommand.ActorID != "user_123" {
		t.Fatalf("expected actor id user_123, got %s", stub.lastUploadCommand.ActorID)
	}
	if stub.lastUploadCommand.ContentType != "image/png" {
		t.Fatalf("expected content type image/png, got %s", stub.lastUploadCommand.ContentType)
	}
	if stub.lastUploadCommand.SizeBytes != 2048 {
		t.Fatalf("expected size 2048, got %d", stub.lastUploadCommand.SizeBytes)
	}
}

func TestAssetHandlers_IssueSignedUpload_InvalidInput(t *testing.T) {
	stub := &stubAssetService{
		uploadErr: services.ErrAssetInvalidInput,
	}
	handler := NewAssetHandlers(nil, stub)
	body := `{"kind":"png","purpose":"preview","mime_type":"image/png"}`
	req := httptest.NewRequest(http.MethodPost, "/assets:signed-upload", bytes.NewBufferString(body))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user"}))

	rr := httptest.NewRecorder()
	handler.issueSignedUpload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestAssetHandlers_IssueSignedUpload_RepositoryUnavailable(t *testing.T) {
	stub := &stubAssetService{
		uploadErr: services.ErrAssetRepositoryUnavailable,
	}
	handler := NewAssetHandlers(nil, stub)
	body := `{"kind":"png","purpose":"preview","mime_type":"image/png","size_bytes":1}`
	req := httptest.NewRequest(http.MethodPost, "/assets:signed-upload", bytes.NewBufferString(body))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user"}))

	rr := httptest.NewRecorder()
	handler.issueSignedUpload(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rr.Code)
	}
}

func TestAssetHandlers_IssueSignedUpload_Unauthenticated(t *testing.T) {
	handler := NewAssetHandlers(nil, &stubAssetService{})
	req := httptest.NewRequest(http.MethodPost, "/assets:signed-upload", bytes.NewBufferString(`{}`))
	rr := httptest.NewRecorder()
	handler.issueSignedUpload(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestAssetHandlers_IssueSignedUpload_BodyTooLarge(t *testing.T) {
	handler := NewAssetHandlers(nil, &stubAssetService{})
	oversized := bytes.Repeat([]byte("a"), maxAssetRequestBody+1)
	req := httptest.NewRequest(http.MethodPost, "/assets:signed-upload", bytes.NewBuffer(oversized))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user"}))
	rr := httptest.NewRecorder()
	handler.issueSignedUpload(rr, req)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status 413, got %d", rr.Code)
	}
}

func TestAssetHandlers_IssueSignedDownload_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	stub := &stubAssetService{
		downloadResponse: domain.SignedAssetResponse{
			AssetID:   "asset_123",
			URL:       "https://storage.example/download",
			Method:    "GET",
			ExpiresAt: now,
			Headers: map[string]string{
				"Cache-Control": "private",
			},
		},
	}
	handler := NewAssetHandlers(nil, stub)

	req := httptest.NewRequest(http.MethodPost, "/assets/asset_123:signed-download", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("assetId", "asset_123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user_123"}))

	rr := httptest.NewRecorder()
	handler.issueSignedDownload(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp signedDownloadResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.URL != stub.downloadResponse.URL {
		t.Fatalf("expected url %s, got %s", stub.downloadResponse.URL, resp.URL)
	}
	if resp.Method != stub.downloadResponse.Method {
		t.Fatalf("expected method %s, got %s", stub.downloadResponse.Method, resp.Method)
	}
	if resp.ExpiresAt != now.Format(time.RFC3339) {
		t.Fatalf("expected expires at %s, got %s", now.Format(time.RFC3339), resp.ExpiresAt)
	}
	if stub.downloadCalls != 1 {
		t.Fatalf("expected download calls 1, got %d", stub.downloadCalls)
	}
	if stub.lastDownloadCommand.ActorID != "user_123" {
		t.Fatalf("expected actor id user_123, got %s", stub.lastDownloadCommand.ActorID)
	}
	if stub.lastDownloadCommand.AssetID != "asset_123" {
		t.Fatalf("expected asset id asset_123, got %s", stub.lastDownloadCommand.AssetID)
	}
}

func TestAssetHandlers_IssueSignedDownload_Errors(t *testing.T) {
	cases := []struct {
		name           string
		err            error
		expectedStatus int
	}{
		{"invalid_input", services.ErrAssetInvalidInput, http.StatusBadRequest},
		{"forbidden", services.ErrAssetForbidden, http.StatusForbidden},
		{"unavailable", services.ErrAssetUnavailable, http.StatusConflict},
		{"not_found", services.ErrAssetNotFound, http.StatusNotFound},
		{"repo_unavailable", services.ErrAssetRepositoryUnavailable, http.StatusServiceUnavailable},
		{"other", errors.New("boom"), http.StatusBadGateway},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &stubAssetService{downloadErr: tc.err}
			handler := NewAssetHandlers(nil, stub)
			req := httptest.NewRequest(http.MethodPost, "/assets/asset_456:signed-download", nil)
			routeCtx := chi.NewRouteContext()
			routeCtx.URLParams.Add("assetId", "asset_456")
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
			req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user_456"}))

			rr := httptest.NewRecorder()
			handler.issueSignedDownload(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, rr.Code)
			}
		})
	}
}

func TestAssetHandlers_IssueSignedDownload_RequiresAuthentication(t *testing.T) {
	handler := NewAssetHandlers(nil, &stubAssetService{})
	req := httptest.NewRequest(http.MethodPost, "/assets/asset_123:signed-download", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("assetId", "asset_123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rr := httptest.NewRecorder()
	handler.issueSignedDownload(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestAssetHandlers_IssueSignedDownload_RequiresAssetID(t *testing.T) {
	handler := NewAssetHandlers(nil, &stubAssetService{})
	req := httptest.NewRequest(http.MethodPost, "/assets/:signed-download", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user"}))

	rr := httptest.NewRecorder()
	handler.issueSignedDownload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

type stubAssetService struct {
	uploadResponse    domain.SignedAssetResponse
	uploadErr         error
	uploadCalls       int
	lastUploadCommand services.SignedUploadCommand

	downloadResponse    domain.SignedAssetResponse
	downloadErr         error
	downloadCalls       int
	lastDownloadCommand services.SignedDownloadCommand
}

func (s *stubAssetService) IssueSignedUpload(_ context.Context, cmd services.SignedUploadCommand) (domain.SignedAssetResponse, error) {
	s.uploadCalls++
	s.lastUploadCommand = cmd
	if s.uploadErr != nil {
		return domain.SignedAssetResponse{}, s.uploadErr
	}
	return s.uploadResponse, nil
}

func (s *stubAssetService) IssueSignedDownload(_ context.Context, cmd services.SignedDownloadCommand) (domain.SignedAssetResponse, error) {
	s.downloadCalls++
	s.lastDownloadCommand = cmd
	if s.downloadErr != nil {
		return domain.SignedAssetResponse{}, s.downloadErr
	}
	return s.downloadResponse, nil
}
