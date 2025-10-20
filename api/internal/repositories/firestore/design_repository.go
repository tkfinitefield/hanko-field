package firestore

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"cloud.google.com/go/firestore"

	domain "github.com/hanko-field/api/internal/domain"
	pfirestore "github.com/hanko-field/api/internal/platform/firestore"
	"github.com/hanko-field/api/internal/repositories"
)

const designsCollection = "designs"

// DesignRepository persists design documents and metadata snapshots.
type DesignRepository struct {
	base *pfirestore.BaseRepository[designDocument]
}

// NewDesignRepository constructs a Firestore-backed design repository.
func NewDesignRepository(provider *pfirestore.Provider) (*DesignRepository, error) {
	if provider == nil {
		return nil, errors.New("design repository: firestore provider is required")
	}
	base := pfirestore.NewBaseRepository[designDocument](provider, designsCollection, nil, nil)
	return &DesignRepository{base: base}, nil
}

// Insert stores a new design document. The ID must be unique.
func (r *DesignRepository) Insert(ctx context.Context, design domain.Design) error {
	if r == nil || r.base == nil {
		return errors.New("design repository not initialised")
	}
	designID := strings.TrimSpace(design.ID)
	if designID == "" {
		return errors.New("design repository: design id is required")
	}
	docRef, err := r.base.DocumentRef(ctx, designID)
	if err != nil {
		return err
	}
	doc := encodeDesignDocument(design)
	if _, err := docRef.Create(ctx, doc); err != nil {
		return pfirestore.WrapError("designs.insert", err)
	}
	return nil
}

// Update replaces the persisted design state with the provided snapshot.
func (r *DesignRepository) Update(ctx context.Context, design domain.Design) error {
	if r == nil || r.base == nil {
		return errors.New("design repository not initialised")
	}
	designID := strings.TrimSpace(design.ID)
	if designID == "" {
		return errors.New("design repository: design id is required")
	}
	docRef, err := r.base.DocumentRef(ctx, designID)
	if err != nil {
		return err
	}
	doc := encodeDesignDocument(design)
	if _, err := docRef.Set(ctx, doc); err != nil {
		return pfirestore.WrapError("designs.update", err)
	}
	return nil
}

// SoftDelete marks the design as deleted while retaining the record for audit/history.
func (r *DesignRepository) SoftDelete(ctx context.Context, designID string, deletedAt time.Time) error {
	if r == nil || r.base == nil {
		return errors.New("design repository not initialised")
	}
	designID = strings.TrimSpace(designID)
	if designID == "" {
		return errors.New("design repository: design id is required")
	}
	docRef, err := r.base.DocumentRef(ctx, designID)
	if err != nil {
		return err
	}
	deletedAt = deletedAt.UTC()
	updates := []firestore.Update{
		{Path: "status", Value: string(domain.DesignStatusDeleted)},
		{Path: "deletedAt", Value: deletedAt},
		{Path: "updatedAt", Value: deletedAt},
	}
	if _, err := docRef.Update(ctx, updates); err != nil {
		return pfirestore.WrapError("designs.soft_delete", err)
	}
	return nil
}

// FindByID fetches a single design.
func (r *DesignRepository) FindByID(ctx context.Context, designID string) (domain.Design, error) {
	if r == nil || r.base == nil {
		return domain.Design{}, errors.New("design repository not initialised")
	}
	designID = strings.TrimSpace(designID)
	if designID == "" {
		return domain.Design{}, errors.New("design repository: design id is required")
	}
	doc, err := r.base.Get(ctx, designID)
	if err != nil {
		return domain.Design{}, err
	}
	return decodeDesignDocument(designID, doc.Data, doc.CreateTime, doc.UpdateTime), nil
}

// ListByOwner returns designs owned by the specified user ordered by most recent update.
func (r *DesignRepository) ListByOwner(ctx context.Context, ownerID string, filter repositories.DesignListFilter) (domain.CursorPage[domain.Design], error) {
	if r == nil || r.base == nil {
		return domain.CursorPage[domain.Design]{}, errors.New("design repository not initialised")
	}
	ownerID = strings.TrimSpace(ownerID)
	if ownerID == "" {
		return domain.CursorPage[domain.Design]{}, errors.New("design repository: owner id is required")
	}

	limit := filter.Pagination.PageSize
	if limit < 0 {
		limit = 0
	}
	fetchLimit := limit
	if limit > 0 {
		fetchLimit = limit + 1
	}

	var startAfter []any
	if token := strings.TrimSpace(filter.Pagination.PageToken); token != "" {
		tokenTime, tokenID, err := decodeDesignListToken(token)
		if err != nil {
			return domain.CursorPage[domain.Design]{}, fmt.Errorf("design repository: invalid page token: %w", err)
		}
		startAfter = []any{tokenTime, tokenID}
	}

	statusFilters := normaliseStatuses(filter.Status)
	typeFilters := normaliseTypes(filter.Types)

	var updatedAfter *time.Time
	if filter.UpdatedAfter != nil {
		value := filter.UpdatedAfter.UTC()
		if !value.IsZero() {
			updatedAfter = &value
		}
	}

	docs, err := r.base.Query(ctx, func(q firestore.Query) firestore.Query {
		q = q.Where("ownerUid", "==", ownerID)

		if len(statusFilters) == 1 {
			q = q.Where("status", "==", statusFilters[0])
		} else if len(statusFilters) > 1 {
			// Firestore in clause supports up to 10 values.
			if len(statusFilters) > 10 {
				statusFilters = statusFilters[:10]
			}
			q = q.Where("status", "in", statusFilters)
		}

		if len(typeFilters) == 1 {
			q = q.Where("type", "==", typeFilters[0])
		} else if len(typeFilters) > 1 {
			if len(typeFilters) > 10 {
				typeFilters = typeFilters[:10]
			}
			q = q.Where("type", "in", typeFilters)
		}

		if updatedAfter != nil {
			q = q.Where("updatedAt", ">", *updatedAfter)
		}

		q = q.OrderBy("updatedAt", firestore.Desc).OrderBy(firestore.DocumentID, firestore.Desc)
		if len(startAfter) == 2 {
			q = q.StartAfter(startAfter...)
		}
		if fetchLimit > 0 {
			q = q.Limit(fetchLimit)
		}
		return q
	})
	if err != nil {
		return domain.CursorPage[domain.Design]{}, err
	}

	valueDocs := docs
	nextToken := ""
	if limit > 0 && len(valueDocs) == fetchLimit {
		last := valueDocs[len(valueDocs)-1]
		tokenTime := last.Data.UpdatedAt
		if tokenTime.IsZero() {
			tokenTime = last.UpdateTime
		}
		nextToken = encodeDesignListToken(tokenTime, last.ID)
		valueDocs = valueDocs[:len(valueDocs)-1]
	}

	items := make([]domain.Design, 0, len(valueDocs))
	for _, doc := range valueDocs {
		items = append(items, decodeDesignDocument(doc.ID, doc.Data, doc.CreateTime, doc.UpdateTime))
	}

	return domain.CursorPage[domain.Design]{
		Items:         items,
		NextPageToken: nextToken,
	}, nil
}

type designDocument struct {
	OwnerRef       string                `firestore:"ownerRef"`
	OwnerUID       string                `firestore:"ownerUid"`
	Label          string                `firestore:"label"`
	Type           string                `firestore:"type"`
	TextLines      []string              `firestore:"textLines"`
	FontID         string                `firestore:"fontId"`
	MaterialID     string                `firestore:"materialId"`
	TemplateRef    string                `firestore:"templateRef"`
	Locale         string                `firestore:"locale"`
	Shape          string                `firestore:"shape"`
	SizeMM         float64               `firestore:"sizeMm"`
	Source         *designSourceDocument `firestore:"source,omitempty"`
	Assets         *designAssetsDocument `firestore:"assets,omitempty"`
	Status         string                `firestore:"status"`
	ThumbnailURL   string                `firestore:"thumbnailUrl"`
	Version        int                   `firestore:"version"`
	CurrentVersion string                `firestore:"currentVersionId"`
	Snapshot       map[string]any        `firestore:"snapshot"`
	Hash           string                `firestore:"hash"`
	Metadata       map[string]any        `firestore:"metadata"`
	CreatedAt      time.Time             `firestore:"createdAt"`
	UpdatedAt      time.Time             `firestore:"updatedAt"`
	LastOrderedAt  *time.Time            `firestore:"lastOrderedAt,omitempty"`
	DeletedAt      *time.Time            `firestore:"deletedAt,omitempty"`
}

type designSourceDocument struct {
	Type        string                  `firestore:"type"`
	RawName     string                  `firestore:"rawName"`
	TextLines   []string                `firestore:"textLines"`
	UploadAsset *designAssetRefDocument `firestore:"uploadAsset,omitempty"`
	LogoAsset   *designAssetRefDocument `firestore:"logoAsset,omitempty"`
}

type designAssetRefDocument struct {
	AssetID     string `firestore:"assetId"`
	Bucket      string `firestore:"bucket"`
	ObjectPath  string `firestore:"objectPath"`
	FileName    string `firestore:"fileName"`
	ContentType string `firestore:"contentType"`
	SizeBytes   int64  `firestore:"sizeBytes"`
	Checksum    string `firestore:"checksum,omitempty"`
}

type designAssetsDocument struct {
	SourcePath  string `firestore:"sourcePath"`
	VectorPath  string `firestore:"vectorPath"`
	PreviewPath string `firestore:"previewPath"`
	PreviewURL  string `firestore:"previewUrl"`
}

func encodeDesignDocument(design domain.Design) designDocument {
	nowCreated := design.CreatedAt.UTC()
	nowUpdated := design.UpdatedAt.UTC()
	doc := designDocument{
		OwnerRef:       userDocPath(design.OwnerID),
		OwnerUID:       strings.TrimSpace(design.OwnerID),
		Label:          strings.TrimSpace(design.Label),
		Type:           strings.TrimSpace(string(design.Type)),
		TextLines:      cloneStrings(design.TextLines),
		FontID:         strings.TrimSpace(design.FontID),
		MaterialID:     strings.TrimSpace(design.MaterialID),
		TemplateRef:    strings.TrimSpace(design.Template),
		Locale:         strings.TrimSpace(design.Locale),
		Shape:          strings.TrimSpace(design.Shape),
		SizeMM:         design.SizeMM,
		Status:         strings.TrimSpace(string(design.Status)),
		ThumbnailURL:   strings.TrimSpace(design.ThumbnailURL),
		Version:        design.Version,
		CurrentVersion: strings.TrimSpace(design.CurrentVersionID),
		Snapshot:       cloneMap(design.Snapshot),
		Hash:           "",
		Metadata:       map[string]any{},
		CreatedAt:      nowCreated,
		UpdatedAt:      nowUpdated,
		LastOrderedAt:  normalizeTimePointer(design.LastOrderedAt),
	}
	if design.Source.Type != "" || len(design.Source.TextLines) > 0 || design.Source.UploadAsset != nil || design.Source.LogoAsset != nil || strings.TrimSpace(design.Source.RawName) != "" {
		doc.Source = encodeDesignSource(design.Source)
	}
	if design.Assets.SourcePath != "" || design.Assets.VectorPath != "" || design.Assets.PreviewPath != "" || design.Assets.PreviewURL != "" {
		doc.Assets = &designAssetsDocument{
			SourcePath:  strings.TrimSpace(design.Assets.SourcePath),
			VectorPath:  strings.TrimSpace(design.Assets.VectorPath),
			PreviewPath: strings.TrimSpace(design.Assets.PreviewPath),
			PreviewURL:  strings.TrimSpace(design.Assets.PreviewURL),
		}
	}
	if len(doc.Snapshot) == 0 && design.Snapshot != nil {
		doc.Snapshot = cloneMap(design.Snapshot)
	}
	return doc
}

func encodeDesignSource(src domain.DesignSource) *designSourceDocument {
	doc := &designSourceDocument{
		Type:      strings.TrimSpace(string(src.Type)),
		RawName:   strings.TrimSpace(src.RawName),
		TextLines: cloneStrings(src.TextLines),
	}
	if ref := encodeAssetReference(src.UploadAsset); ref != nil {
		doc.UploadAsset = ref
	}
	if ref := encodeAssetReference(src.LogoAsset); ref != nil {
		doc.LogoAsset = ref
	}
	return doc
}

func encodeAssetReference(ref *domain.DesignAssetReference) *designAssetRefDocument {
	if ref == nil {
		return nil
	}
	if strings.TrimSpace(ref.AssetID) == "" && strings.TrimSpace(ref.ObjectPath) == "" {
		return nil
	}
	return &designAssetRefDocument{
		AssetID:     strings.TrimSpace(ref.AssetID),
		Bucket:      strings.TrimSpace(ref.Bucket),
		ObjectPath:  strings.TrimSpace(ref.ObjectPath),
		FileName:    strings.TrimSpace(ref.FileName),
		ContentType: strings.TrimSpace(ref.ContentType),
		SizeBytes:   ref.SizeBytes,
		Checksum:    strings.TrimSpace(ref.Checksum),
	}
}

func decodeDesignDocument(id string, doc designDocument, createdAt, updatedAt time.Time) domain.Design {
	source := domain.DesignSource{}
	if doc.Source != nil {
		source = domain.DesignSource{
			Type:      domain.DesignType(strings.TrimSpace(doc.Source.Type)),
			RawName:   strings.TrimSpace(doc.Source.RawName),
			TextLines: cloneStrings(doc.Source.TextLines),
		}
		if doc.Source.UploadAsset != nil {
			source.UploadAsset = decodeAssetReference(doc.Source.UploadAsset)
		}
		if doc.Source.LogoAsset != nil {
			source.LogoAsset = decodeAssetReference(doc.Source.LogoAsset)
		}
	}

	assets := domain.DesignAssets{}
	if doc.Assets != nil {
		assets = domain.DesignAssets{
			SourcePath:  strings.TrimSpace(doc.Assets.SourcePath),
			VectorPath:  strings.TrimSpace(doc.Assets.VectorPath),
			PreviewPath: strings.TrimSpace(doc.Assets.PreviewPath),
			PreviewURL:  strings.TrimSpace(doc.Assets.PreviewURL),
		}
	}

	design := domain.Design{
		ID:               strings.TrimSpace(id),
		OwnerID:          extractOwner(doc.OwnerRef, doc.OwnerUID),
		Label:            strings.TrimSpace(doc.Label),
		Type:             domain.DesignType(strings.TrimSpace(doc.Type)),
		TextLines:        cloneStrings(doc.TextLines),
		FontID:           strings.TrimSpace(doc.FontID),
		MaterialID:       strings.TrimSpace(doc.MaterialID),
		Template:         strings.TrimSpace(doc.TemplateRef),
		Locale:           strings.TrimSpace(doc.Locale),
		Shape:            strings.TrimSpace(doc.Shape),
		SizeMM:           doc.SizeMM,
		Source:           source,
		Assets:           assets,
		Status:           domain.DesignStatus(strings.TrimSpace(doc.Status)),
		ThumbnailURL:     strings.TrimSpace(doc.ThumbnailURL),
		Version:          doc.Version,
		CurrentVersionID: strings.TrimSpace(doc.CurrentVersion),
		Snapshot:         cloneMap(doc.Snapshot),
		CreatedAt:        chooseTime(doc.CreatedAt, createdAt),
		UpdatedAt:        chooseTime(doc.UpdatedAt, updatedAt),
		LastOrderedAt:    normalizeTimePointer(doc.LastOrderedAt),
	}
	if len(design.TextLines) == 0 && len(design.Source.TextLines) > 0 {
		design.TextLines = cloneStrings(design.Source.TextLines)
	}
	return design
}

func decodeAssetReference(doc *designAssetRefDocument) *domain.DesignAssetReference {
	if doc == nil {
		return nil
	}
	return &domain.DesignAssetReference{
		AssetID:     strings.TrimSpace(doc.AssetID),
		Bucket:      strings.TrimSpace(doc.Bucket),
		ObjectPath:  strings.TrimSpace(doc.ObjectPath),
		FileName:    strings.TrimSpace(doc.FileName),
		ContentType: strings.TrimSpace(doc.ContentType),
		SizeBytes:   doc.SizeBytes,
		Checksum:    strings.TrimSpace(doc.Checksum),
	}
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		if t := strings.TrimSpace(value); t != "" {
			trimmed = append(trimmed, t)
		} else {
			trimmed = append(trimmed, "")
		}
	}
	return slices.Clone(trimmed)
}

func cloneMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	return maps.Clone(src)
}

func chooseTime(primary time.Time, fallback time.Time) time.Time {
	if !primary.IsZero() {
		return primary.UTC()
	}
	if !fallback.IsZero() {
		return fallback.UTC()
	}
	return time.Time{}
}

func normalizeTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	if value.IsZero() {
		return nil
	}
	ts := value.UTC()
	return &ts
}

func userDocPath(userID string) string {
	trimmed := strings.TrimSpace(userID)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "/users/") {
		return trimmed
	}
	if strings.HasPrefix(trimmed, "users/") {
		return "/" + trimmed
	}
	return "/users/" + trimmed
}

func extractOwner(ownerRef string, ownerUID string) string {
	if trimmed := strings.TrimSpace(ownerUID); trimmed != "" {
		return trimmed
	}
	ref := strings.TrimSpace(ownerRef)
	ref = strings.TrimPrefix(ref, "/")
	const prefix = "users/"
	if strings.HasPrefix(ref, prefix) {
		return ref[len(prefix):]
	}
	return ref
}

func encodeDesignListToken(updatedAt time.Time, docID string) string {
	payload := fmt.Sprintf("%s|%s", updatedAt.UTC().Format(time.RFC3339Nano), docID)
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func decodeDesignListToken(token string) (time.Time, string, error) {
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return time.Time{}, "", err
	}
	parts := strings.SplitN(string(data), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", errors.New("invalid token structure")
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", err
	}
	return ts, parts[1], nil
}

func normaliseStatuses(statuses []string) []string {
	if len(statuses) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(statuses))
	seen := make(map[string]struct{})
	for _, status := range statuses {
		trimmed := strings.ToLower(strings.TrimSpace(status))
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func normaliseTypes(types []string) []string {
	if len(types) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(types))
	seen := make(map[string]struct{})
	for _, t := range types {
		trimmed := strings.ToLower(strings.TrimSpace(t))
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized
}
