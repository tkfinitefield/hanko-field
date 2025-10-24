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

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/services"
)

func TestAssetHandlers_IssueSignedUpload_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	stub := &stubAssetService{
		response: domain.SignedAssetResponse{
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
	if resp.AssetID != stub.response.AssetID {
		t.Fatalf("expected asset id %s, got %s", stub.response.AssetID, resp.AssetID)
	}
	if resp.UploadURL != stub.response.URL {
		t.Fatalf("expected upload url %s, got %s", stub.response.URL, resp.UploadURL)
	}
	if resp.ExpiresAt != now.Format(time.RFC3339) {
		t.Fatalf("expected expires at %s, got %s", now.Format(time.RFC3339), resp.ExpiresAt)
	}
	if stub.calls != 1 {
		t.Fatalf("expected service called once, got %d", stub.calls)
	}
	if stub.lastCommand.ActorID != "user_123" {
		t.Fatalf("expected actor id user_123, got %s", stub.lastCommand.ActorID)
	}
	if stub.lastCommand.ContentType != "image/png" {
		t.Fatalf("expected content type image/png, got %s", stub.lastCommand.ContentType)
	}
	if stub.lastCommand.SizeBytes != 2048 {
		t.Fatalf("expected size 2048, got %d", stub.lastCommand.SizeBytes)
	}
}

func TestAssetHandlers_IssueSignedUpload_InvalidInput(t *testing.T) {
	stub := &stubAssetService{
		err: services.ErrAssetInvalidInput,
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
		err: services.ErrAssetRepositoryUnavailable,
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

type stubAssetService struct {
	response    domain.SignedAssetResponse
	err         error
	calls       int
	lastCommand services.SignedUploadCommand
}

func (s *stubAssetService) IssueSignedUpload(_ context.Context, cmd services.SignedUploadCommand) (domain.SignedAssetResponse, error) {
	s.calls++
	s.lastCommand = cmd
	if s.err != nil {
		return domain.SignedAssetResponse{}, s.err
	}
	return s.response, nil
}

func (s *stubAssetService) IssueSignedDownload(context.Context, services.SignedDownloadCommand) (domain.SignedAssetResponse, error) {
	return domain.SignedAssetResponse{}, errors.New("not implemented")
}
