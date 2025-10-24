package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/oklog/ulid/v2"

	domain "github.com/hanko-field/api/internal/domain"
	pfirestore "github.com/hanko-field/api/internal/platform/firestore"
	pstorage "github.com/hanko-field/api/internal/platform/storage"
	"github.com/hanko-field/api/internal/repositories"
)

const (
	assetsCollection         = "assets"
	defaultAssetIDPrefix     = "asset_"
	assetStatusPendingUpload = "pending_upload"
	defaultAssetUploadTTL    = 15 * time.Minute
)

// AssetRepository persists asset metadata and coordinates signed URL issuance.
type AssetRepository struct {
	base    *pfirestore.BaseRepository[assetDocument]
	storage *pstorage.Client
	bucket  string
	clock   func() time.Time
	newID   func() string
}

// AssetRepositoryOption customises the repository behaviour.
type AssetRepositoryOption func(*AssetRepository)

// WithAssetRepositoryClock overrides the clock used by the repository.
func WithAssetRepositoryClock(clock func() time.Time) AssetRepositoryOption {
	return func(r *AssetRepository) {
		if clock != nil {
			r.clock = func() time.Time { return clock().UTC() }
		}
	}
}

// WithAssetRepositoryIDGenerator overrides the ID generator used by the repository.
func WithAssetRepositoryIDGenerator(generator func() string) AssetRepositoryOption {
	return func(r *AssetRepository) {
		if generator != nil {
			r.newID = generator
		}
	}
}

// NewAssetRepository constructs a Firestore-backed asset repository.
func NewAssetRepository(provider *pfirestore.Provider, storageClient *pstorage.Client, bucket string, opts ...AssetRepositoryOption) (*AssetRepository, error) {
	if provider == nil {
		return nil, errors.New("asset repository: firestore provider is required")
	}
	if storageClient == nil {
		return nil, errors.New("asset repository: storage client is required")
	}
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return nil, errors.New("asset repository: bucket is required")
	}

	repo := &AssetRepository{
		base:    pfirestore.NewBaseRepository[assetDocument](provider, assetsCollection, nil, nil),
		storage: storageClient,
		bucket:  bucket,
		clock: func() time.Time {
			return time.Now().UTC()
		},
		newID: func() string {
			return ulid.Make().String()
		},
	}

	for _, opt := range opts {
		if opt != nil {
			opt(repo)
		}
	}

	return repo, nil
}

// CreateSignedUpload stores a pending asset record and returns an upload URL.
func (r *AssetRepository) CreateSignedUpload(ctx context.Context, cmd repositories.SignedUploadRecord) (domain.SignedAssetResponse, error) {
	if r == nil || r.base == nil || r.storage == nil {
		return domain.SignedAssetResponse{}, errors.New("asset repository: not initialised")
	}

	actorID := strings.TrimSpace(cmd.ActorID)
	if actorID == "" {
		return domain.SignedAssetResponse{}, errors.New("asset repository: actor id is required")
	}
	kind := strings.ToLower(strings.TrimSpace(cmd.Kind))
	if kind == "" {
		return domain.SignedAssetResponse{}, errors.New("asset repository: kind is required")
	}
	purpose := strings.ToLower(strings.TrimSpace(cmd.Purpose))
	if purpose == "" {
		return domain.SignedAssetResponse{}, errors.New("asset repository: purpose is required")
	}
	contentType := strings.ToLower(strings.TrimSpace(cmd.ContentType))
	if contentType == "" {
		return domain.SignedAssetResponse{}, errors.New("asset repository: content type is required")
	}
	size := cmd.SizeBytes
	if size <= 0 {
		return domain.SignedAssetResponse{}, errors.New("asset repository: size bytes must be positive")
	}

	var designID string
	if cmd.DesignID != nil {
		designID = strings.TrimSpace(*cmd.DesignID)
	}

	fileName := strings.TrimSpace(cmd.FileName)

	rawID := r.newID()
	assetID := ensureAssetID(rawID)
	objectID := strings.TrimPrefix(assetID, defaultAssetIDPrefix)
	objectPath := fmt.Sprintf("assets/%s/%s", kind, objectID)

	signed, err := r.storage.SignedURL(ctx, r.bucket, objectPath, pstorage.SignedURLOptions{
		Upload: &pstorage.UploadOptions{
			Method:              "PUT",
			ContentType:         contentType,
			AllowedMethods:      []string{"PUT"},
			AllowedContentTypes: []string{contentType},
			MaxSize:             size,
			ExpiresIn:           defaultAssetUploadTTL,
			AdditionalHeaders: map[string]string{
				"x-goog-meta-asset-id": assetID,
			},
		},
	})
	if err != nil {
		return domain.SignedAssetResponse{}, fmt.Errorf("asset repository: sign upload url: %w", err)
	}

	now := r.clock()
	doc := assetDocument{
		OwnerUID:        actorID,
		DesignID:        designID,
		Kind:            kind,
		Purpose:         purpose,
		Status:          assetStatusPendingUpload,
		Bucket:          r.bucket,
		ObjectPath:      objectPath,
		FileName:        fileName,
		ContentType:     contentType,
		SizeBytes:       size,
		UploadIssuedBy:  actorID,
		UploadExpiresAt: signed.ExpiresAt,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if _, err := r.base.Set(ctx, assetID, doc); err != nil {
		return domain.SignedAssetResponse{}, err
	}

	return domain.SignedAssetResponse{
		AssetID:   assetID,
		URL:       signed.URL,
		Method:    signed.Method,
		ExpiresAt: signed.ExpiresAt,
		Headers:   signed.Headers,
	}, nil
}

// CreateSignedDownload currently not implemented.
func (r *AssetRepository) CreateSignedDownload(context.Context, repositories.SignedDownloadRecord) (domain.SignedAssetResponse, error) {
	return domain.SignedAssetResponse{}, errors.New("asset repository: signed download not implemented")
}

// MarkUploaded updates the asset status to uploaded and merges metadata.
func (r *AssetRepository) MarkUploaded(ctx context.Context, assetID string, actorID string, metadata map[string]any) error {
	if r == nil || r.base == nil {
		return errors.New("asset repository: not initialised")
	}
	id := strings.TrimSpace(assetID)
	if id == "" {
		return errors.New("asset repository: asset id is required")
	}
	now := r.clock()

	updates := []firestore.Update{
		{Path: "status", Value: "uploaded"},
		{Path: "updatedAt", Value: now},
	}

	issuedBy := strings.TrimSpace(actorID)
	if issuedBy != "" {
		updates = append(updates, firestore.Update{Path: "uploadCompletedBy", Value: issuedBy})
	}
	completedAt := now
	updates = append(updates, firestore.Update{Path: "uploadCompletedAt", Value: completedAt})

	if len(metadata) > 0 {
		updates = append(updates, firestore.Update{Path: "metadata", Value: metadata})
	}

	_, err := r.base.Update(ctx, id, updates)
	return err
}

type assetDocument struct {
	OwnerUID          string         `firestore:"ownerUid"`
	DesignID          string         `firestore:"designId,omitempty"`
	Kind              string         `firestore:"kind"`
	Purpose           string         `firestore:"purpose"`
	Status            string         `firestore:"status"`
	Bucket            string         `firestore:"bucket"`
	ObjectPath        string         `firestore:"objectPath"`
	FileName          string         `firestore:"fileName,omitempty"`
	ContentType       string         `firestore:"contentType"`
	SizeBytes         int64          `firestore:"sizeBytes"`
	Metadata          map[string]any `firestore:"metadata,omitempty"`
	UploadIssuedBy    string         `firestore:"uploadIssuedBy,omitempty"`
	UploadExpiresAt   time.Time      `firestore:"uploadExpiresAt"`
	UploadCompletedBy string         `firestore:"uploadCompletedBy,omitempty"`
	UploadCompletedAt *time.Time     `firestore:"uploadCompletedAt,omitempty"`
	CreatedAt         time.Time      `firestore:"createdAt"`
	UpdatedAt         time.Time      `firestore:"updatedAt"`
}

func ensureAssetID(candidate string) string {
	trimmed := strings.TrimSpace(candidate)
	if trimmed == "" {
		trimmed = ulid.Make().String()
	}
	if strings.HasPrefix(trimmed, defaultAssetIDPrefix) {
		return trimmed
	}
	return defaultAssetIDPrefix + trimmed
}
