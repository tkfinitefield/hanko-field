package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	pstorage "github.com/hanko-field/api/internal/platform/storage"
	"github.com/hanko-field/api/internal/repositories"
)

func TestAssetService_IssueSignedUpload_Success(t *testing.T) {
	repo := &stubAssetRepository{
		uploadResponse: domain.SignedAssetResponse{
			AssetID:   "asset_123",
			URL:       "https://storage.example/upload",
			Method:    "PUT",
			ExpiresAt: time.Now().Add(10 * time.Minute),
			Headers: map[string]string{
				"Content-Type": "image/png",
			},
		},
	}

	svc, err := NewAssetService(AssetServiceDeps{Repository: repo})
	if err != nil {
		t.Fatalf("NewAssetService error: %v", err)
	}

	resp, err := svc.IssueSignedUpload(context.Background(), SignedUploadCommand{
		ActorID:     "user_123",
		Kind:        "PNG",
		Purpose:     "preview",
		FileName:    "Preview.PNG",
		ContentType: "image/png",
		SizeBytes:   1024,
	})
	if err != nil {
		t.Fatalf("IssueSignedUpload error: %v", err)
	}

	if resp.AssetID != repo.uploadResponse.AssetID {
		t.Fatalf("expected asset id %q, got %q", repo.uploadResponse.AssetID, resp.AssetID)
	}
	if repo.uploadRecord.Kind != "png" {
		t.Fatalf("expected kind normalised to png, got %q", repo.uploadRecord.Kind)
	}
	if repo.uploadRecord.Purpose != "preview" {
		t.Fatalf("expected purpose preview, got %q", repo.uploadRecord.Purpose)
	}
	if repo.uploadRecord.ContentType != "image/png" {
		t.Fatalf("expected content type image/png, got %q", repo.uploadRecord.ContentType)
	}
	if repo.uploadRecord.SizeBytes != 1024 {
		t.Fatalf("expected size 1024, got %d", repo.uploadRecord.SizeBytes)
	}
	if repo.uploadRecord.ActorID != "user_123" {
		t.Fatalf("expected actor id user_123, got %q", repo.uploadRecord.ActorID)
	}
	if repo.uploadCalls != 1 {
		t.Fatalf("expected repository called once, got %d", repo.uploadCalls)
	}
}

func TestAssetService_IssueSignedUpload_InvalidInput(t *testing.T) {
	svc, err := NewAssetService(AssetServiceDeps{Repository: &stubAssetRepository{uploadResponse: domain.SignedAssetResponse{}}})
	if err != nil {
		t.Fatalf("NewAssetService error: %v", err)
	}

	cases := []struct {
		name string
		cmd  SignedUploadCommand
	}{
		{
			name: "missing actor",
			cmd: SignedUploadCommand{
				Kind:        "png",
				Purpose:     "preview",
				ContentType: "image/png",
				SizeBytes:   10,
			},
		},
		{
			name: "invalid kind",
			cmd: SignedUploadCommand{
				ActorID:     "user",
				Kind:        "unknown",
				Purpose:     "preview",
				ContentType: "image/png",
				SizeBytes:   10,
			},
		},
		{
			name: "invalid purpose",
			cmd: SignedUploadCommand{
				ActorID:     "user",
				Kind:        "png",
				Purpose:     "invalid",
				ContentType: "image/png",
				SizeBytes:   10,
			},
		},
		{
			name: "invalid content type",
			cmd: SignedUploadCommand{
				ActorID:     "user",
				Kind:        "png",
				Purpose:     "preview",
				ContentType: "application/pdf",
				SizeBytes:   10,
			},
		},
		{
			name: "size zero",
			cmd: SignedUploadCommand{
				ActorID:     "user",
				Kind:        "png",
				Purpose:     "preview",
				ContentType: "image/png",
				SizeBytes:   0,
			},
		},
		{
			name: "size too large",
			cmd: SignedUploadCommand{
				ActorID:     "user",
				Kind:        "json",
				Purpose:     "other",
				ContentType: "application/json",
				SizeBytes:   maxStructuredAssetSize + 1,
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.IssueSignedUpload(context.Background(), tc.cmd)
			if err == nil {
				t.Fatalf("expected error")
			}
			if !errors.Is(err, ErrAssetInvalidInput) {
				t.Fatalf("expected ErrAssetInvalidInput, got %v", err)
			}
		})
	}
}

func TestAssetService_IssueSignedUpload_RepositoryError(t *testing.T) {
	repoErr := fakeRepositoryError{unavailable: true}
	repo := &stubAssetRepository{
		uploadErr: repoErr,
	}

	svc, err := NewAssetService(AssetServiceDeps{Repository: repo})
	if err != nil {
		t.Fatalf("NewAssetService error: %v", err)
	}

	_, err = svc.IssueSignedUpload(context.Background(), SignedUploadCommand{
		ActorID:     "user",
		Kind:        "png",
		Purpose:     "preview",
		ContentType: "image/png",
		SizeBytes:   1024,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, ErrAssetRepositoryUnavailable) {
		t.Fatalf("expected ErrAssetRepositoryUnavailable, got %v", err)
	}
	if repo.uploadCalls != 1 {
		t.Fatalf("expected repository called once, got %d", repo.uploadCalls)
	}
}

func TestAssetService_IssueSignedDownload_Success(t *testing.T) {
	expiry := time.Now().Add(5 * time.Minute)
	repo := &stubAssetRepository{
		downloadResponse: domain.SignedAssetResponse{
			AssetID:   "asset_123",
			URL:       "https://storage.example/download",
			Method:    "GET",
			ExpiresAt: expiry,
		},
	}

	var logged bool
	var loggedEvent string
	var loggedFields map[string]any

	svc, err := NewAssetService(AssetServiceDeps{
		Repository: repo,
		Logger: func(_ context.Context, event string, fields map[string]any) {
			if event == assetLoggerEventDownload {
				logged = true
				loggedEvent = event
				loggedFields = fields
			}
		},
	})
	if err != nil {
		t.Fatalf("NewAssetService error: %v", err)
	}

	resp, err := svc.IssueSignedDownload(context.Background(), SignedDownloadCommand{ActorID: "user_123", AssetID: "asset_123"})
	if err != nil {
		t.Fatalf("IssueSignedDownload error: %v", err)
	}

	if resp.URL != repo.downloadResponse.URL {
		t.Fatalf("expected url %s, got %s", repo.downloadResponse.URL, resp.URL)
	}
	if repo.downloadRecord.ActorID != "user_123" {
		t.Fatalf("expected actor id user_123, got %s", repo.downloadRecord.ActorID)
	}
	if repo.downloadRecord.AssetID != "asset_123" {
		t.Fatalf("expected asset id asset_123, got %s", repo.downloadRecord.AssetID)
	}
	if repo.downloadCalls != 1 {
		t.Fatalf("expected repository called once, got %d", repo.downloadCalls)
	}
	if !logged || loggedEvent != assetLoggerEventDownload {
		t.Fatalf("expected download event logged")
	}
	if loggedFields == nil {
		t.Fatalf("expected log fields")
	}
	if loggedFields["actorId"] != "user_123" {
		t.Fatalf("expected actorId user_123, got %v", loggedFields["actorId"])
	}
	if loggedFields["assetId"] != "asset_123" {
		t.Fatalf("expected assetId asset_123, got %v", loggedFields["assetId"])
	}
}

func TestAssetService_IssueSignedDownload_InvalidInput(t *testing.T) {
	svc, err := NewAssetService(AssetServiceDeps{Repository: &stubAssetRepository{}})
	if err != nil {
		t.Fatalf("NewAssetService error: %v", err)
	}

	cases := []struct {
		name string
		cmd  SignedDownloadCommand
	}{
		{name: "missing actor", cmd: SignedDownloadCommand{AssetID: "asset_1"}},
		{name: "missing asset", cmd: SignedDownloadCommand{ActorID: "user"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.IssueSignedDownload(context.Background(), tc.cmd)
			if err == nil {
				t.Fatalf("expected error")
			}
			if !errors.Is(err, ErrAssetInvalidInput) {
				t.Fatalf("expected ErrAssetInvalidInput, got %v", err)
			}
		})
	}
}

func TestAssetService_IssueSignedDownload_ErrorMapping(t *testing.T) {
	cases := []struct {
		name        string
		repoErr     error
		expectedErr error
	}{
		{name: "permission", repoErr: pstorage.ErrPermissionDenied, expectedErr: ErrAssetForbidden},
		{name: "not_ready", repoErr: repositories.ErrAssetNotReady, expectedErr: ErrAssetUnavailable},
		{name: "soft_deleted", repoErr: repositories.ErrAssetSoftDeleted, expectedErr: ErrAssetNotFound},
		{name: "not_found", repoErr: fakeRepositoryError{notFound: true}, expectedErr: ErrAssetNotFound},
		{name: "unavailable", repoErr: fakeRepositoryError{unavailable: true}, expectedErr: ErrAssetRepositoryUnavailable},
		{name: "generic", repoErr: errors.New("boom"), expectedErr: ErrAssetRepositoryFailure},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &stubAssetRepository{downloadErr: tc.repoErr}
			svc, err := NewAssetService(AssetServiceDeps{Repository: repo})
			if err != nil {
				t.Fatalf("NewAssetService error: %v", err)
			}

			_, err = svc.IssueSignedDownload(context.Background(), SignedDownloadCommand{ActorID: "user", AssetID: "asset"})
			if err == nil {
				t.Fatalf("expected error")
			}
			if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("expected %v, got %v", tc.expectedErr, err)
			}
			if repo.downloadCalls != 1 {
				t.Fatalf("expected repository called once, got %d", repo.downloadCalls)
			}
		})
	}
}

type stubAssetRepository struct {
	uploadRecord   repositories.SignedUploadRecord
	uploadResponse domain.SignedAssetResponse
	uploadErr      error
	uploadCalls    int

	downloadRecord   repositories.SignedDownloadRecord
	downloadResponse domain.SignedAssetResponse
	downloadErr      error
	downloadCalls    int
}

func (s *stubAssetRepository) CreateSignedUpload(_ context.Context, cmd repositories.SignedUploadRecord) (domain.SignedAssetResponse, error) {
	s.uploadCalls++
	s.uploadRecord = cmd
	if s.uploadErr != nil {
		return domain.SignedAssetResponse{}, s.uploadErr
	}
	return s.uploadResponse, nil
}

func (s *stubAssetRepository) CreateSignedDownload(_ context.Context, cmd repositories.SignedDownloadRecord) (domain.SignedAssetResponse, error) {
	s.downloadCalls++
	s.downloadRecord = cmd
	if s.downloadErr != nil {
		return domain.SignedAssetResponse{}, s.downloadErr
	}
	return s.downloadResponse, nil
}

func (s *stubAssetRepository) MarkUploaded(context.Context, string, string, map[string]any) error {
	return errors.New("not implemented")
}

type fakeRepositoryError struct {
	notFound    bool
	conflict    bool
	unavailable bool
}

func (e fakeRepositoryError) Error() string {
	parts := []string{"repository error"}
	if e.unavailable {
		parts = append(parts, "(unavailable)")
	}
	return strings.Join(parts, " ")
}

func (e fakeRepositoryError) IsNotFound() bool    { return e.notFound }
func (e fakeRepositoryError) IsConflict() bool    { return e.conflict }
func (e fakeRepositoryError) IsUnavailable() bool { return e.unavailable }
