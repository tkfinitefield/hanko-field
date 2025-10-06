package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// Document represents a strongly typed Firestore document with metadata timestamps.
type Document[T any] struct {
	ID         string
	Data       T
	CreateTime time.Time
	UpdateTime time.Time
	ReadTime   time.Time
}

// MutationResult captures the update timestamp returned by Firestore mutations.
type MutationResult struct {
	UpdateTime time.Time
}

// Encoder serialises the strongly typed entity prior to persistence.
type Encoder[T any] func(ctx context.Context, value T) (any, error)

// Decoder hydrates the strongly typed entity from a snapshot.
type Decoder[T any] func(ctx context.Context, snap *firestore.DocumentSnapshot) (T, error)

// QueryBuilder customises Firestore queries before execution.
type QueryBuilder func(query firestore.Query) firestore.Query

// BaseRepository provides typed helpers wrapping Firestore collection access.
type BaseRepository[T any] struct {
	provider   *Provider
	collection string
	encode     Encoder[T]
	decode     Decoder[T]
}

// NewBaseRepository constructs a BaseRepository bound to a collection.
func NewBaseRepository[T any](provider *Provider, collection string, encode Encoder[T], decode Decoder[T]) *BaseRepository[T] {
	if encode == nil {
		encode = IdentityEncoder[T]()
	}
	if decode == nil {
		decode = StructDecoder[T]()
	}
	return &BaseRepository[T]{
		provider:   provider,
		collection: strings.TrimSpace(collection),
		encode:     encode,
		decode:     decode,
	}
}

// Set upserts the given value under the provided document ID.
func (r *BaseRepository[T]) Set(ctx context.Context, id string, value T, opts ...firestore.SetOption) (MutationResult, error) {
	doc, err := r.documentRef(ctx, id)
	if err != nil {
		return MutationResult{}, err
	}

	payload, err := r.encode(ctx, value)
	if err != nil {
		return MutationResult{}, fmt.Errorf("firestore: encode document %s: %w", id, err)
	}

	result, err := doc.Set(ctx, payload, opts...)
	if err != nil {
		return MutationResult{}, WrapError(r.op("set"), err)
	}
	return MutationResult{UpdateTime: result.UpdateTime}, nil
}

// Update applies partial updates to the document.
func (r *BaseRepository[T]) Update(ctx context.Context, id string, updates []firestore.Update, opts ...firestore.Precondition) (MutationResult, error) {
	doc, err := r.documentRef(ctx, id)
	if err != nil {
		return MutationResult{}, err
	}
	result, err := doc.Update(ctx, updates, opts...)
	if err != nil {
		return MutationResult{}, WrapError(r.op("update"), err)
	}
	return MutationResult{UpdateTime: result.UpdateTime}, nil
}

// Get fetches the document by ID and decodes it into the strongly typed entity.
func (r *BaseRepository[T]) Get(ctx context.Context, id string) (Document[T], error) {
	doc, err := r.documentRef(ctx, id)
	if err != nil {
		return Document[T]{}, err
	}

	snapshot, err := doc.Get(ctx)
	if err != nil {
		return Document[T]{}, WrapError(r.op("get"), err)
	}

	return r.decodeDocument(ctx, snapshot)
}

// Query executes a collection query and returns the decoded documents.
func (r *BaseRepository[T]) Query(ctx context.Context, build QueryBuilder) ([]Document[T], error) {
	coll, err := r.collectionRef(ctx)
	if err != nil {
		return nil, err
	}

	query := coll.Query
	if build != nil {
		query = build(query)
	}

	iter := query.Documents(ctx)
	defer iter.Stop()

	var docs []Document[T]
	for {
		snapshot, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, WrapError(r.op("query"), err)
		}
		decoded, err := r.decodeDocument(ctx, snapshot)
		if err != nil {
			return nil, fmt.Errorf("firestore: decode document %s: %w", snapshot.Ref.ID, err)
		}
		docs = append(docs, decoded)
	}
	return docs, nil
}

// DocumentRef exposes the underlying document reference for advanced scenarios such as transactions.
func (r *BaseRepository[T]) DocumentRef(ctx context.Context, id string) (*firestore.DocumentRef, error) {
	return r.documentRef(ctx, id)
}

func (r *BaseRepository[T]) decodeDocument(ctx context.Context, snapshot *firestore.DocumentSnapshot) (Document[T], error) {
	entity, err := r.decode(ctx, snapshot)
	if err != nil {
		return Document[T]{}, err
	}
	return Document[T]{
		ID:         snapshot.Ref.ID,
		Data:       entity,
		CreateTime: snapshot.CreateTime,
		UpdateTime: snapshot.UpdateTime,
		ReadTime:   snapshot.ReadTime,
	}, nil
}

func (r *BaseRepository[T]) collectionRef(ctx context.Context) (*firestore.CollectionRef, error) {
	if r == nil || r.provider == nil {
		return nil, WrapError(r.op("collection"), errors.New("firestore: provider is nil"))
	}
	if r.collection == "" {
		return nil, WrapError(r.op("collection"), errors.New("firestore: collection name is required"))
	}
	client, err := r.provider.Client(ctx)
	if err != nil {
		return nil, err
	}
	return client.Collection(r.collection), nil
}

func (r *BaseRepository[T]) documentRef(ctx context.Context, id string) (*firestore.DocumentRef, error) {
	if strings.TrimSpace(id) == "" {
		return nil, WrapError(r.op("document"), errors.New("firestore: document id is required"))
	}
	coll, err := r.collectionRef(ctx)
	if err != nil {
		return nil, err
	}
	return coll.Doc(id), nil
}

func (r *BaseRepository[T]) op(action string) string {
	name := "firestore"
	if r != nil {
		trimmed := strings.TrimSpace(r.collection)
		if trimmed != "" {
			name = trimmed
		}
	}
	return fmt.Sprintf("%s.%s", name, strings.ToLower(action))
}

// IdentityEncoder returns an encoder that writes the value unchanged.
func IdentityEncoder[T any]() Encoder[T] {
	return func(_ context.Context, value T) (any, error) {
		return value, nil
	}
}

// MapEncoder adapts map values that are already Firestore compatible.
func MapEncoder[T ~map[string]any]() Encoder[T] {
	return func(_ context.Context, value T) (any, error) {
		return map[string]any(value), nil
	}
}

// StructDecoder populates the target struct using Firestore's native decoding.
func StructDecoder[T any]() Decoder[T] {
	return func(_ context.Context, snap *firestore.DocumentSnapshot) (T, error) {
		var target T
		if err := snap.DataTo(&target); err != nil {
			return target, err
		}
		return target, nil
	}
}

// MapDecoder returns the underlying map representation for documents.
func MapDecoder() Decoder[map[string]any] {
	return func(_ context.Context, snap *firestore.DocumentSnapshot) (map[string]any, error) {
		data := snap.Data()
		if data == nil {
			data = map[string]any{}
		}
		return data, nil
	}
}
