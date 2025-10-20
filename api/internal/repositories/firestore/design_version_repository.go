package firestore

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	domain "github.com/hanko-field/api/internal/domain"
	pfirestore "github.com/hanko-field/api/internal/platform/firestore"
)

const designVersionsCollection = "versions"

// DesignVersionRepository persists immutable snapshots for designs.
type DesignVersionRepository struct {
	provider *pfirestore.Provider
}

// NewDesignVersionRepository constructs a Firestore-backed design version repository.
func NewDesignVersionRepository(provider *pfirestore.Provider) (*DesignVersionRepository, error) {
	if provider == nil {
		return nil, errors.New("design version repository: firestore provider is required")
	}
	return &DesignVersionRepository{provider: provider}, nil
}

// Append stores a new version snapshot under the given design.
func (r *DesignVersionRepository) Append(ctx context.Context, version domain.DesignVersion) error {
	if r == nil || r.provider == nil {
		return errors.New("design version repository not initialised")
	}
	designID := normalizeDesignID(version.DesignID)
	if designID == "" {
		return errors.New("design version repository: design id is required")
	}
	versionID := strings.TrimSpace(version.ID)
	if versionID == "" {
		return errors.New("design version repository: version id is required")
	}

	config, assetsDoc := splitDesignVersionSnapshot(version.Snapshot)

	coll, err := r.collection(ctx, designID)
	if err != nil {
		return err
	}

	doc := designVersionDocument{
		Sequence:   version.Version,
		Config:     config,
		Assets:     assetsDoc,
		Snapshot:   cloneMap(version.Snapshot),
		ChangeNote: "",
		CreatedAt:  version.CreatedAt.UTC(),
		CreatedBy:  strings.TrimSpace(version.CreatedBy),
	}
	if version.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now().UTC()
	}

	if _, err := coll.Doc(versionID).Set(ctx, doc); err != nil {
		return pfirestore.WrapError("design_versions.append", err)
	}
	return nil
}

// ListByDesign returns versions for a design ordered by version (newest first).
func (r *DesignVersionRepository) ListByDesign(ctx context.Context, designID string, pager domain.Pagination) (domain.CursorPage[domain.DesignVersion], error) {
	if r == nil || r.provider == nil {
		return domain.CursorPage[domain.DesignVersion]{}, errors.New("design version repository not initialised")
	}
	designID = normalizeDesignID(designID)
	if designID == "" {
		return domain.CursorPage[domain.DesignVersion]{}, errors.New("design version repository: design id is required")
	}

	limit := pager.PageSize
	if limit < 0 {
		limit = 0
	}
	fetchLimit := limit
	if limit > 0 {
		fetchLimit = limit + 1
	}

	var startAfter []any
	if token := strings.TrimSpace(pager.PageToken); token != "" {
		sequence, docID, err := decodeDesignVersionToken(token)
		if err != nil {
			return domain.CursorPage[domain.DesignVersion]{}, fmt.Errorf("design version repository: invalid page token: %w", err)
		}
		startAfter = []any{sequence, docID}
	}

	coll, err := r.collection(ctx, designID)
	if err != nil {
		return domain.CursorPage[domain.DesignVersion]{}, err
	}

	query := coll.OrderBy("sequence", firestore.Desc).OrderBy(firestore.DocumentID, firestore.Desc)
	if len(startAfter) == 2 {
		query = query.StartAfter(startAfter...)
	}
	if fetchLimit > 0 {
		query = query.Limit(fetchLimit)
	}

	iter := query.Documents(ctx)
	defer iter.Stop()

	type versionRow struct {
		id         string
		data       designVersionDocument
		createTime time.Time
		updateTime time.Time
	}

	rows := make([]versionRow, 0, fetchLimit)
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return domain.CursorPage[domain.DesignVersion]{}, pfirestore.WrapError("design_versions.list", err)
		}
		var doc designVersionDocument
		if err := snap.DataTo(&doc); err != nil {
			return domain.CursorPage[domain.DesignVersion]{}, fmt.Errorf("design version repository: decode %s: %w", snap.Ref.ID, err)
		}
		rows = append(rows, versionRow{
			id:         snap.Ref.ID,
			data:       doc,
			createTime: snap.CreateTime,
			updateTime: snap.UpdateTime,
		})
	}

	nextToken := ""
	if limit > 0 && len(rows) == fetchLimit {
		last := rows[len(rows)-1]
		nextToken = encodeDesignVersionToken(last.data.Sequence, last.id)
		rows = rows[:len(rows)-1]
	}

	items := make([]domain.DesignVersion, 0, len(rows))
	for _, row := range rows {
		items = append(items, decodeDesignVersion(designID, row.id, row.data, row.createTime, row.updateTime))
	}

	return domain.CursorPage[domain.DesignVersion]{
		Items:         items,
		NextPageToken: nextToken,
	}, nil
}

// FindByID returns a single version of the design.
func (r *DesignVersionRepository) FindByID(ctx context.Context, designID string, versionID string) (domain.DesignVersion, error) {
	if r == nil || r.provider == nil {
		return domain.DesignVersion{}, errors.New("design version repository not initialised")
	}
	designID = normalizeDesignID(designID)
	if designID == "" {
		return domain.DesignVersion{}, errors.New("design version repository: design id is required")
	}
	versionID = strings.TrimSpace(versionID)
	if versionID == "" {
		return domain.DesignVersion{}, errors.New("design version repository: version id is required")
	}

	coll, err := r.collection(ctx, designID)
	if err != nil {
		return domain.DesignVersion{}, err
	}

	snap, err := coll.Doc(versionID).Get(ctx)
	if err != nil {
		return domain.DesignVersion{}, pfirestore.WrapError("design_versions.get", err)
	}

	var doc designVersionDocument
	if err := snap.DataTo(&doc); err != nil {
		return domain.DesignVersion{}, fmt.Errorf("design version repository: decode %s: %w", snap.Ref.ID, err)
	}

	return decodeDesignVersion(designID, snap.Ref.ID, doc, snap.CreateTime, snap.UpdateTime), nil
}

type designVersionDocument struct {
	Sequence   int                          `firestore:"sequence"`
	Config     map[string]any               `firestore:"config,omitempty"`
	Assets     *designVersionAssetsDocument `firestore:"assets,omitempty"`
	Snapshot   map[string]any               `firestore:"snapshot,omitempty"`
	ChangeNote string                       `firestore:"changeNote,omitempty"`
	CreatedAt  time.Time                    `firestore:"createdAt"`
	CreatedBy  string                       `firestore:"createdBy"`
}

type designVersionAssetsDocument struct {
	SourcePath  string `firestore:"sourcePath,omitempty"`
	VectorPath  string `firestore:"vectorPath,omitempty"`
	PreviewPath string `firestore:"previewPath,omitempty"`
	PreviewURL  string `firestore:"previewUrl,omitempty"`
}

func (a *designVersionAssetsDocument) isZero() bool {
	if a == nil {
		return true
	}
	return strings.TrimSpace(a.SourcePath) == "" &&
		strings.TrimSpace(a.VectorPath) == "" &&
		strings.TrimSpace(a.PreviewPath) == "" &&
		strings.TrimSpace(a.PreviewURL) == ""
}

func (a *designVersionAssetsDocument) toMap() map[string]any {
	if a == nil || a.isZero() {
		return nil
	}
	result := make(map[string]any)
	if value := strings.TrimSpace(a.SourcePath); value != "" {
		result["sourcePath"] = value
	}
	if value := strings.TrimSpace(a.VectorPath); value != "" {
		result["vectorPath"] = value
	}
	if value := strings.TrimSpace(a.PreviewPath); value != "" {
		result["previewPath"] = value
	}
	if value := strings.TrimSpace(a.PreviewURL); value != "" {
		result["previewUrl"] = value
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func splitDesignVersionSnapshot(snapshot map[string]any) (map[string]any, *designVersionAssetsDocument) {
	if len(snapshot) == 0 {
		return nil, nil
	}
	config := cloneMap(snapshot)
	if len(config) == 0 {
		return nil, nil
	}
	rawAssets, ok := config["assets"]
	if !ok {
		return config, nil
	}

	assetsMap := extractAssetMap(rawAssets)
	delete(config, "assets")

	var assetsDoc *designVersionAssetsDocument
	if len(assetsMap) > 0 {
		doc := &designVersionAssetsDocument{
			SourcePath:  stringValue(assetsMap, "sourcePath"),
			VectorPath:  stringValue(assetsMap, "vectorPath"),
			PreviewPath: stringValue(assetsMap, "previewPath"),
			PreviewURL:  stringValue(assetsMap, "previewUrl"),
		}
		if !doc.isZero() {
			assetsDoc = doc
		}
	}

	if len(config) == 0 {
		config = nil
	}
	return config, assetsDoc
}

func extractAssetMap(value any) map[string]any {
	if value == nil {
		return nil
	}
	raw, ok := value.(map[string]any)
	if !ok || len(raw) == 0 {
		return nil
	}
	return cloneMap(raw)
}

func stringValue(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if value, ok := m[key]; ok {
		if s, ok := value.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func (r *DesignVersionRepository) collection(ctx context.Context, designID string) (*firestore.CollectionRef, error) {
	client, err := r.provider.Client(ctx)
	if err != nil {
		return nil, err
	}
	return client.Collection(designsCollection).Doc(designID).Collection(designVersionsCollection), nil
}

func decodeDesignVersion(designID, versionID string, doc designVersionDocument, createdAt, updatedAt time.Time) domain.DesignVersion {
	snapshot := cloneMap(doc.Snapshot)
	if len(snapshot) == 0 && len(doc.Config) > 0 {
		snapshot = cloneMap(doc.Config)
	}
	if doc.Assets != nil && !doc.Assets.isZero() {
		if snapshot == nil {
			snapshot = make(map[string]any)
		}
		if assets := doc.Assets.toMap(); len(assets) > 0 {
			snapshot["assets"] = assets
		}
	}

	version := domain.DesignVersion{
		ID:        strings.TrimSpace(versionID),
		DesignID:  designID,
		Version:   doc.Sequence,
		Snapshot:  cloneMap(snapshot),
		CreatedAt: chooseTime(doc.CreatedAt, createdAt),
		CreatedBy: strings.TrimSpace(doc.CreatedBy),
	}
	return version
}

func normalizeDesignID(designID string) string {
	trimmed := strings.TrimSpace(designID)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "/designs/") {
		trimmed = trimmed[len("/designs/"):]
	}
	if strings.HasPrefix(trimmed, "designs/") {
		trimmed = trimmed[len("designs/"):]
	}
	return trimmed
}

func encodeDesignVersionToken(versionNumber int, docID string) string {
	payload := fmt.Sprintf("%d|%s", versionNumber, docID)
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func decodeDesignVersionToken(token string) (int, string, error) {
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, "", err
	}
	parts := strings.SplitN(string(data), "|", 2)
	if len(parts) != 2 {
		return 0, "", errors.New("invalid token structure")
	}
	versionNum, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", err
	}
	return versionNum, parts[1], nil
}
