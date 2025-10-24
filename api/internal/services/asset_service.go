package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	pstorage "github.com/hanko-field/api/internal/platform/storage"
	"github.com/hanko-field/api/internal/repositories"
)

const (
	defaultMaxBinaryAssetSize  = int64(25 * 1024 * 1024)  // 25 MiB
	maxArchiveAssetSize        = int64(100 * 1024 * 1024) // 100 MiB
	maxMediaAssetSize          = int64(150 * 1024 * 1024) // 150 MiB
	maxStructuredAssetSize     = int64(5 * 1024 * 1024)   // 5 MiB
	maxAudioAssetSize          = int64(30 * 1024 * 1024)  // 30 MiB
	assetLoggerEventValidation = "asset.upload.validate"
	assetLoggerEventIssued     = "asset.upload.issued"
	assetLoggerEventDownload   = "asset.download.issued"
)

var (
	// ErrAssetInvalidInput indicates the caller provided an invalid argument.
	ErrAssetInvalidInput = errors.New("asset: invalid input")
	// ErrAssetRepositoryUnavailable indicates the persistence layer is unavailable.
	ErrAssetRepositoryUnavailable = errors.New("asset: repository unavailable")
	// ErrAssetRepositoryFailure wraps unexpected repository failures.
	ErrAssetRepositoryFailure = errors.New("asset: repository failure")
	// ErrAssetForbidden indicates the caller lacks permission to access the asset.
	ErrAssetForbidden = errors.New("asset: forbidden")
	// ErrAssetNotFound indicates the asset does not exist or is no longer available.
	ErrAssetNotFound = errors.New("asset: not found")
	// ErrAssetUnavailable indicates the asset exists but is not ready for download.
	ErrAssetUnavailable = errors.New("asset: unavailable")
)

// AssetServiceDeps wires dependencies for the asset service implementation.
type AssetServiceDeps struct {
	Repository repositories.AssetRepository
	Clock      func() time.Time
	Logger     func(ctx context.Context, event string, fields map[string]any)
}

type assetService struct {
	repo   repositories.AssetRepository
	clock  func() time.Time
	logger func(context.Context, string, map[string]any)
}

type assetKindPolicy struct {
	contentTypes []string
	maxSize      int64
}

var (
	allowedAssetPurposes = map[string]struct{}{
		"design-master":    {},
		"preview":          {},
		"3d-model":         {},
		"certificate":      {},
		"social-mock":      {},
		"guide-image":      {},
		"page-hero":        {},
		"shipment-label":   {},
		"production-photo": {},
		"receipt":          {},
		"other":            {},
	}

	assetKindPolicies = map[string]assetKindPolicy{
		"svg":  {contentTypes: []string{"image/svg+xml"}, maxSize: defaultMaxBinaryAssetSize},
		"png":  {contentTypes: []string{"image/png"}, maxSize: defaultMaxBinaryAssetSize},
		"jpg":  {contentTypes: []string{"image/jpeg", "image/jpg"}, maxSize: defaultMaxBinaryAssetSize},
		"webp": {contentTypes: []string{"image/webp"}, maxSize: defaultMaxBinaryAssetSize},
		"gltf": {contentTypes: []string{"model/gltf+json", "model/gltf-binary", "model/vnd.gltf+json", "application/octet-stream"}, maxSize: maxArchiveAssetSize},
		"pdf":  {contentTypes: []string{"application/pdf"}, maxSize: defaultMaxBinaryAssetSize},
		"zip":  {contentTypes: []string{"application/zip", "application/x-zip-compressed"}, maxSize: maxArchiveAssetSize},
		"mp4":  {contentTypes: []string{"video/mp4"}, maxSize: maxMediaAssetSize},
		"mp3":  {contentTypes: []string{"audio/mpeg", "audio/mp3"}, maxSize: maxAudioAssetSize},
		"json": {contentTypes: []string{"application/json"}, maxSize: maxStructuredAssetSize},
		"other": {
			contentTypes: []string{"*"},
			maxSize:      defaultMaxBinaryAssetSize,
		},
	}
)

// NewAssetService constructs an AssetService backed by the provided dependencies.
func NewAssetService(deps AssetServiceDeps) (AssetService, error) {
	if deps.Repository == nil {
		return nil, errors.New("asset service: repository is required")
	}

	clock := deps.Clock
	if clock == nil {
		clock = time.Now
	}

	logger := deps.Logger
	if logger == nil {
		logger = func(context.Context, string, map[string]any) {}
	}

	return &assetService{
		repo: deps.Repository,
		clock: func() time.Time {
			return clock().UTC()
		},
		logger: logger,
	}, nil
}

func (s *assetService) IssueSignedUpload(ctx context.Context, cmd SignedUploadCommand) (SignedAssetResponse, error) {
	params, err := s.validateUploadInput(cmd)
	if err != nil {
		return SignedAssetResponse{}, err
	}

	if s.logger != nil {
		s.logger(ctx, assetLoggerEventValidation, map[string]any{
			"actorId": params.actorID,
			"kind":    params.kind,
			"purpose": params.purpose,
			"size":    params.sizeBytes,
		})
	}

	record := repositories.SignedUploadRecord{
		ActorID:     params.actorID,
		DesignID:    params.designID,
		Kind:        params.kind,
		Purpose:     params.purpose,
		FileName:    params.fileName,
		ContentType: params.contentType,
		SizeBytes:   params.sizeBytes,
	}

	response, err := s.repo.CreateSignedUpload(ctx, record)
	if err != nil {
		return SignedAssetResponse{}, s.mapRepositoryError(err)
	}

	if s.logger != nil {
		s.logger(ctx, assetLoggerEventIssued, map[string]any{
			"actorId":    params.actorID,
			"assetId":    response.AssetID,
			"method":     response.Method,
			"expiresAt":  response.ExpiresAt,
			"uploadSize": params.sizeBytes,
		})
	}

	return response, nil
}

func (s *assetService) IssueSignedDownload(ctx context.Context, cmd SignedDownloadCommand) (SignedAssetResponse, error) {
	actorID := strings.TrimSpace(cmd.ActorID)
	if actorID == "" {
		return SignedAssetResponse{}, fmt.Errorf("%w: actor id is required", ErrAssetInvalidInput)
	}

	assetID := strings.TrimSpace(cmd.AssetID)
	if assetID == "" {
		return SignedAssetResponse{}, fmt.Errorf("%w: asset id is required", ErrAssetInvalidInput)
	}

	response, err := s.repo.CreateSignedDownload(ctx, repositories.SignedDownloadRecord{
		ActorID: actorID,
		AssetID: assetID,
	})
	if err != nil {
		return SignedAssetResponse{}, s.mapDownloadError(err)
	}

	if s.logger != nil {
		s.logger(ctx, assetLoggerEventDownload, map[string]any{
			"actorId":   actorID,
			"assetId":   response.AssetID,
			"expiresAt": response.ExpiresAt,
		})
	}

	return response, nil
}

type uploadParams struct {
	actorID     string
	designID    *string
	kind        string
	purpose     string
	fileName    string
	contentType string
	sizeBytes   int64
}

func (s *assetService) validateUploadInput(cmd SignedUploadCommand) (uploadParams, error) {
	actorID := strings.TrimSpace(cmd.ActorID)
	if actorID == "" {
		return uploadParams{}, fmt.Errorf("%w: actor id is required", ErrAssetInvalidInput)
	}

	var designID *string
	if cmd.DesignID != nil {
		if trimmed := strings.TrimSpace(*cmd.DesignID); trimmed != "" {
			designID = &trimmed
		}
	}

	kind := strings.ToLower(strings.TrimSpace(cmd.Kind))
	policy, ok := assetKindPolicies[kind]
	if !ok {
		return uploadParams{}, fmt.Errorf("%w: asset kind %q not allowed", ErrAssetInvalidInput, cmd.Kind)
	}

	purpose := strings.ToLower(strings.TrimSpace(cmd.Purpose))
	if _, ok := allowedAssetPurposes[purpose]; !ok {
		return uploadParams{}, fmt.Errorf("%w: asset purpose %q not allowed", ErrAssetInvalidInput, cmd.Purpose)
	}

	contentType := strings.ToLower(strings.TrimSpace(cmd.ContentType))
	if contentType == "" {
		return uploadParams{}, fmt.Errorf("%w: content_type is required", ErrAssetInvalidInput)
	}
	if !contentTypeAllowed(contentType, policy.contentTypes) {
		return uploadParams{}, fmt.Errorf("%w: content_type %q not allowed for kind %q", ErrAssetInvalidInput, contentType, kind)
	}

	size := cmd.SizeBytes
	if size <= 0 {
		return uploadParams{}, fmt.Errorf("%w: size_bytes must be positive", ErrAssetInvalidInput)
	}
	if policy.maxSize > 0 && size > policy.maxSize {
		return uploadParams{}, fmt.Errorf("%w: size_bytes exceeds maximum (%d)", ErrAssetInvalidInput, policy.maxSize)
	}

	fileName := strings.TrimSpace(cmd.FileName)
	if fileName == "" {
		fileName = fmt.Sprintf("%s_%d", kind, time.Now().UnixNano())
	}

	return uploadParams{
		actorID:     actorID,
		designID:    designID,
		kind:        kind,
		purpose:     purpose,
		fileName:    fileName,
		contentType: contentType,
		sizeBytes:   size,
	}, nil
}

func (s *assetService) mapRepositoryError(err error) error {
	if err == nil {
		return nil
	}
	var repoErr repositories.RepositoryError
	if errors.As(err, &repoErr) {
		switch {
		case repoErr.IsUnavailable():
			return fmt.Errorf("%w: %v", ErrAssetRepositoryUnavailable, err)
		case repoErr.IsConflict(), repoErr.IsNotFound():
			return fmt.Errorf("%w: %v", ErrAssetRepositoryFailure, err)
		default:
			return fmt.Errorf("%w: %v", ErrAssetRepositoryFailure, err)
		}
	}
	return fmt.Errorf("%w: %v", ErrAssetRepositoryFailure, err)
}

func (s *assetService) mapDownloadError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, pstorage.ErrPermissionDenied):
		return ErrAssetForbidden
	case errors.Is(err, repositories.ErrAssetNotReady):
		return ErrAssetUnavailable
	case errors.Is(err, repositories.ErrAssetSoftDeleted):
		return ErrAssetNotFound
	}

	var repoErr repositories.RepositoryError
	if errors.As(err, &repoErr) {
		switch {
		case repoErr.IsNotFound():
			return ErrAssetNotFound
		case repoErr.IsUnavailable():
			return fmt.Errorf("%w: %v", ErrAssetRepositoryUnavailable, err)
		default:
			return fmt.Errorf("%w: %v", ErrAssetRepositoryFailure, err)
		}
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}

	return fmt.Errorf("%w: %v", ErrAssetRepositoryFailure, err)
}

func contentTypeAllowed(contentType string, allowed []string) bool {
	if len(allowed) == 0 {
		return false
	}
	ct := strings.ToLower(strings.TrimSpace(contentType))
	for _, candidate := range allowed {
		candidate = strings.ToLower(strings.TrimSpace(candidate))
		if candidate == "" {
			continue
		}
		if candidate == "*" {
			return true
		}
		if strings.HasSuffix(candidate, "/*") {
			prefix := strings.TrimSuffix(candidate, "*")
			if strings.HasPrefix(ct, strings.TrimSuffix(prefix, "/")) {
				return true
			}
			continue
		}
		if ct == candidate {
			return true
		}
	}
	return false
}
