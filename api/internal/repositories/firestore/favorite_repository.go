package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	domain "github.com/hanko-field/api/internal/domain"
	pfirestore "github.com/hanko-field/api/internal/platform/firestore"
	"github.com/hanko-field/api/internal/repositories"
)

const favoriteCollectionPattern = "users/%s/favorites"

// FavoriteRepository persists design favorites per user.
type FavoriteRepository struct {
	provider *pfirestore.Provider
}

// NewFavoriteRepository constructs a Firestore-backed favorite repository.
func NewFavoriteRepository(provider *pfirestore.Provider) (*FavoriteRepository, error) {
	if provider == nil {
		return nil, errors.New("favorite repository requires firestore provider")
	}
	return &FavoriteRepository{provider: provider}, nil
}

// List returns favorites ordered by most recent addition.
func (r *FavoriteRepository) List(ctx context.Context, userID string, pager domain.Pagination) (domain.CursorPage[domain.FavoriteDesign], error) {
	coll, err := r.collection(ctx, userID)
	if err != nil {
		return domain.CursorPage[domain.FavoriteDesign]{}, err
	}

	limit := pager.PageSize
	if limit < 0 {
		limit = 0
	}

	query := coll.OrderBy("addedAt", firestore.Desc).OrderBy(firestore.DocumentID, firestore.Desc)
	var fetchLimit int
	if limit > 0 {
		fetchLimit = limit + 1
		query = query.Limit(fetchLimit)
	}

	if token := strings.TrimSpace(pager.PageToken); token != "" {
		snap, err := coll.Doc(token).Get(ctx)
		if err != nil {
			return domain.CursorPage[domain.FavoriteDesign]{}, pfirestore.WrapError("favorites.list.pageToken", err)
		}
		var doc favoriteDocument
		if err := snap.DataTo(&doc); err != nil {
			return domain.CursorPage[domain.FavoriteDesign]{}, fmt.Errorf("decode favorite %s: %w", snap.Ref.ID, err)
		}
		query = query.StartAfter(doc.AddedAt, snap.Ref.ID)
	}

	iter := query.Documents(ctx)
	defer iter.Stop()

	var items []domain.FavoriteDesign
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return domain.CursorPage[domain.FavoriteDesign]{}, pfirestore.WrapError("favorites.list", err)
		}
		fav, err := decodeFavoriteDocument(snap)
		if err != nil {
			return domain.CursorPage[domain.FavoriteDesign]{}, err
		}
		items = append(items, fav)
	}

	nextToken := ""
	if limit > 0 && len(items) == fetchLimit {
		nextToken = items[len(items)-1].DesignID
		items = items[:len(items)-1]
	}

	return domain.CursorPage[domain.FavoriteDesign]{
		Items:         items,
		NextPageToken: nextToken,
	}, nil
}

// Put stores or preserves a favorite.
func (r *FavoriteRepository) Put(ctx context.Context, userID string, designID string, addedAt time.Time) error {
	coll, err := r.collection(ctx, userID)
	if err != nil {
		return err
	}

	designID = strings.TrimSpace(designID)
	if designID == "" {
		return errors.New("favorite repository: design id is required")
	}

	return pfirestore.WrapError("favorites.put", r.provider.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		docRef := coll.Doc(designID)
		if _, err := tx.Get(docRef); err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				doc := favoriteDocument{
					DesignRef: designDocPath(designID),
					AddedAt:   addedAt.UTC(),
				}
				return tx.Set(docRef, doc)
			default:
				return err
			}
		}
		return nil
	}))
}

// Delete removes the favorite document.
func (r *FavoriteRepository) Delete(ctx context.Context, userID string, designID string) error {
	coll, err := r.collection(ctx, userID)
	if err != nil {
		return err
	}
	designID = strings.TrimSpace(designID)
	if designID == "" {
		return errors.New("favorite repository: design id is required")
	}
	if _, err := coll.Doc(designID).Delete(ctx); err != nil {
		return pfirestore.WrapError("favorites.delete", err)
	}
	return nil
}

func (r *FavoriteRepository) collection(ctx context.Context, userID string) (*firestore.CollectionRef, error) {
	if r == nil || r.provider == nil {
		return nil, errors.New("favorite repository not initialised")
	}
	uid := strings.TrimSpace(userID)
	if uid == "" {
		return nil, errors.New("favorite repository: user id is required")
	}
	client, err := r.provider.Client(ctx)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf(favoriteCollectionPattern, uid)
	return client.Collection(path), nil
}

func decodeFavoriteDocument(snapshot *firestore.DocumentSnapshot) (domain.FavoriteDesign, error) {
	var doc favoriteDocument
	if err := snapshot.DataTo(&doc); err != nil {
		return domain.FavoriteDesign{}, fmt.Errorf("decode favorite %s: %w", snapshot.Ref.ID, err)
	}
	designID := snapshot.Ref.ID
	if trimmed := extractDesignID(doc.DesignRef); trimmed != "" {
		designID = trimmed
	}
	return domain.FavoriteDesign{
		DesignID: designID,
		AddedAt:  doc.AddedAt,
	}, nil
}

type favoriteDocument struct {
	DesignRef string    `firestore:"designRef"`
	AddedAt   time.Time `firestore:"addedAt"`
}

func designDocPath(designID string) string {
	trimmed := strings.TrimSpace(designID)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "/designs/") {
		return trimmed
	}
	if strings.HasPrefix(trimmed, "designs/") {
		return "/" + trimmed
	}
	return "/designs/" + trimmed
}

func extractDesignID(ref string) string {
	trimmed := strings.TrimSpace(ref)
	if trimmed == "" {
		return ""
	}
	trimmed = strings.TrimPrefix(trimmed, "/")
	const prefix = "designs/"
	if strings.HasPrefix(trimmed, prefix) {
		return trimmed[len(prefix):]
	}
	return trimmed
}

// Ensure interface compliance.
var _ repositories.FavoriteRepository = (*FavoriteRepository)(nil)
