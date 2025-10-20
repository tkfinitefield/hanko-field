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

const paymentMethodCollectionPattern = "users/%s/paymentMethods"

// PaymentMethodRepository persists PSP payment references in Firestore.
type PaymentMethodRepository struct {
	provider *pfirestore.Provider
}

// NewPaymentMethodRepository constructs a Firestore-backed payment method repository.
func NewPaymentMethodRepository(provider *pfirestore.Provider) (*PaymentMethodRepository, error) {
	if provider == nil {
		return nil, errors.New("payment method repository requires firestore provider")
	}
	return &PaymentMethodRepository{provider: provider}, nil
}

// List returns all payment methods for the specified user ordered by creation time descending.
func (r *PaymentMethodRepository) List(ctx context.Context, userID string) ([]domain.PaymentMethod, error) {
	coll, err := r.collection(ctx, userID)
	if err != nil {
		return nil, err
	}

	iter := coll.OrderBy("createdAt", firestore.Desc).Documents(ctx)
	defer iter.Stop()

	var methods []domain.PaymentMethod
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, pfirestore.WrapError("payment_methods.list", err)
		}
		method, err := decodePaymentMethodDocument(snap)
		if err != nil {
			return nil, err
		}
		methods = append(methods, method)
	}
	return methods, nil
}

// Insert stores a new payment method, ensuring token uniqueness and default exclusivity.
func (r *PaymentMethodRepository) Insert(ctx context.Context, userID string, method domain.PaymentMethod) (domain.PaymentMethod, error) {
	coll, err := r.collection(ctx, userID)
	if err != nil {
		return domain.PaymentMethod{}, err
	}

	now := time.Now().UTC()
	token := strings.TrimSpace(method.Token)

	var saved domain.PaymentMethod
	err = r.provider.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		query := coll.Where("token", "==", token).Limit(1)
		snaps, err := tx.Documents(query).GetAll()
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}
		if len(snaps) > 0 {
			return status.Error(codes.AlreadyExists, "payment method already exists")
		}

		docRef := coll.NewDoc()
		if id := strings.TrimSpace(method.ID); id != "" {
			docRef = coll.Doc(id)
		}

		doc := paymentMethodDocument{
			Provider:  strings.TrimSpace(method.Provider),
			Token:     token,
			Brand:     strings.TrimSpace(method.Brand),
			Last4:     strings.TrimSpace(method.Last4),
			ExpMonth:  method.ExpMonth,
			ExpYear:   method.ExpYear,
			IsDefault: method.IsDefault,
			CreatedAt: method.CreatedAt,
			UpdatedAt: method.UpdatedAt,
		}
		if doc.CreatedAt.IsZero() {
			doc.CreatedAt = now
		} else {
			doc.CreatedAt = doc.CreatedAt.UTC()
		}
		doc.UpdatedAt = now

		if err := tx.Set(docRef, doc); err != nil {
			return err
		}

		if doc.IsDefault {
			if err := r.clearDefault(ctx, tx, coll, docRef.ID, now); err != nil {
				return err
			}
		}

		saved = doc.toDomain(docRef.ID)
		return nil
	})
	if err != nil {
		return domain.PaymentMethod{}, pfirestore.WrapError("payment_methods.insert", err)
	}
	return saved, nil
}

// Delete removes the specified payment method.
func (r *PaymentMethodRepository) Delete(ctx context.Context, userID string, paymentMethodID string) error {
	coll, err := r.collection(ctx, userID)
	if err != nil {
		return err
	}
	id := strings.TrimSpace(paymentMethodID)
	if id == "" {
		return errors.New("payment method repository: id is required")
	}
	if _, err := coll.Doc(id).Delete(ctx); err != nil {
		return pfirestore.WrapError("payment_methods.delete", err)
	}
	return nil
}

// Get loads a single payment method by ID.
func (r *PaymentMethodRepository) Get(ctx context.Context, userID string, paymentMethodID string) (domain.PaymentMethod, error) {
	coll, err := r.collection(ctx, userID)
	if err != nil {
		return domain.PaymentMethod{}, err
	}
	id := strings.TrimSpace(paymentMethodID)
	if id == "" {
		return domain.PaymentMethod{}, errors.New("payment method repository: id is required")
	}
	snap, err := coll.Doc(id).Get(ctx)
	if err != nil {
		return domain.PaymentMethod{}, pfirestore.WrapError("payment_methods.get", err)
	}
	method, err := decodePaymentMethodDocument(snap)
	if err != nil {
		return domain.PaymentMethod{}, err
	}
	return method, nil
}

// SetDefault marks the specified payment method as default, clearing any previous default.
func (r *PaymentMethodRepository) SetDefault(ctx context.Context, userID string, paymentMethodID string) (domain.PaymentMethod, error) {
	coll, err := r.collection(ctx, userID)
	if err != nil {
		return domain.PaymentMethod{}, err
	}
	id := strings.TrimSpace(paymentMethodID)
	if id == "" {
		return domain.PaymentMethod{}, errors.New("payment method repository: id is required")
	}

	now := time.Now().UTC()
	var saved domain.PaymentMethod
	err = r.provider.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		docRef := coll.Doc(id)
		snap, err := tx.Get(docRef)
		if err != nil {
			return err
		}
		var doc paymentMethodDocument
		if err := snap.DataTo(&doc); err != nil {
			return fmt.Errorf("decode payment method %s: %w", id, err)
		}

		updates := []firestore.Update{
			{Path: "isDefault", Value: true},
			{Path: "updatedAt", Value: now},
		}
		if err := tx.Update(docRef, updates); err != nil {
			return err
		}
		if err := r.clearDefault(ctx, tx, coll, docRef.ID, now); err != nil {
			return err
		}

		doc.IsDefault = true
		doc.UpdatedAt = now
		saved = doc.toDomain(docRef.ID)
		return nil
	})
	if err != nil {
		return domain.PaymentMethod{}, pfirestore.WrapError("payment_methods.set_default", err)
	}
	return saved, nil
}

func (r *PaymentMethodRepository) collection(ctx context.Context, userID string) (*firestore.CollectionRef, error) {
	if r == nil || r.provider == nil {
		return nil, errors.New("payment method repository not initialised")
	}
	uid := strings.TrimSpace(userID)
	if uid == "" {
		return nil, errors.New("payment method repository: user id is required")
	}
	client, err := r.provider.Client(ctx)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf(paymentMethodCollectionPattern, uid)
	return client.Collection(path), nil
}

func (r *PaymentMethodRepository) clearDefault(ctx context.Context, tx *firestore.Transaction, coll *firestore.CollectionRef, currentID string, now time.Time) error {
	query := coll.Where("isDefault", "==", true)
	iter := tx.Documents(query)
	snaps, err := iter.GetAll()
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil
		}
		return err
	}
	for _, snap := range snaps {
		if snap.Ref.ID == currentID {
			continue
		}
		if err := tx.Update(snap.Ref, []firestore.Update{
			{Path: "isDefault", Value: false},
			{Path: "updatedAt", Value: now},
		}); err != nil {
			return err
		}
	}
	return nil
}

func decodePaymentMethodDocument(snapshot *firestore.DocumentSnapshot) (domain.PaymentMethod, error) {
	var doc paymentMethodDocument
	if err := snapshot.DataTo(&doc); err != nil {
		return domain.PaymentMethod{}, fmt.Errorf("decode payment method %s: %w", snapshot.Ref.ID, err)
	}
	return doc.toDomain(snapshot.Ref.ID), nil
}

type paymentMethodDocument struct {
	Provider  string    `firestore:"provider"`
	Token     string    `firestore:"token"`
	Brand     string    `firestore:"brand,omitempty"`
	Last4     string    `firestore:"last4,omitempty"`
	ExpMonth  int       `firestore:"expMonth,omitempty"`
	ExpYear   int       `firestore:"expYear,omitempty"`
	IsDefault bool      `firestore:"isDefault"`
	CreatedAt time.Time `firestore:"createdAt"`
	UpdatedAt time.Time `firestore:"updatedAt"`
}

func (d paymentMethodDocument) toDomain(id string) domain.PaymentMethod {
	method := domain.PaymentMethod{
		ID:        id,
		Provider:  strings.TrimSpace(d.Provider),
		Token:     strings.TrimSpace(d.Token),
		Brand:     strings.TrimSpace(d.Brand),
		Last4:     strings.TrimSpace(d.Last4),
		ExpMonth:  d.ExpMonth,
		ExpYear:   d.ExpYear,
		IsDefault: d.IsDefault,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
	return method
}

// Ensure interface compliance.
var _ repositories.PaymentMethodRepository = (*PaymentMethodRepository)(nil)
