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

	coll, err := r.collection(ctx, designID)
	if err != nil {
		return err
	}

	doc := designVersionDocument{
		Version:    version.Version,
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
		versionNum, docID, err := decodeDesignVersionToken(token)
		if err != nil {
			return domain.CursorPage[domain.DesignVersion]{}, fmt.Errorf("design version repository: invalid page token: %w", err)
		}
		startAfter = []any{versionNum, docID}
	}

	coll, err := r.collection(ctx, designID)
	if err != nil {
		return domain.CursorPage[domain.DesignVersion]{}, err
	}

	query := coll.OrderBy("version", firestore.Desc).OrderBy(firestore.DocumentID, firestore.Desc)
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
		nextToken = encodeDesignVersionToken(last.data.Version, last.id)
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

type designVersionDocument struct {
	Version    int            `firestore:"version"`
	Snapshot   map[string]any `firestore:"snapshot"`
	ChangeNote string         `firestore:"changeNote,omitempty"`
	CreatedAt  time.Time      `firestore:"createdAt"`
	CreatedBy  string         `firestore:"createdBy"`
}

func (r *DesignVersionRepository) collection(ctx context.Context, designID string) (*firestore.CollectionRef, error) {
	client, err := r.provider.Client(ctx)
	if err != nil {
		return nil, err
	}
	return client.Collection(designsCollection).Doc(designID).Collection(designVersionsCollection), nil
}

func decodeDesignVersion(designID, versionID string, doc designVersionDocument, createdAt, updatedAt time.Time) domain.DesignVersion {
	version := domain.DesignVersion{
		ID:        strings.TrimSpace(versionID),
		DesignID:  designID,
		Version:   doc.Version,
		Snapshot:  cloneMap(doc.Snapshot),
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
