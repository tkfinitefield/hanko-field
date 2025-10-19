package firestore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

const addressCollectionPattern = "users/%s/addresses"

// AddressRepository persists user addresses in Firestore.
type AddressRepository struct {
	provider *pfirestore.Provider
}

// NewAddressRepository constructs a Firestore-backed address repository.
func NewAddressRepository(provider *pfirestore.Provider) (*AddressRepository, error) {
	if provider == nil {
		return nil, errors.New("address repository requires firestore provider")
	}
	return &AddressRepository{provider: provider}, nil
}

// List returns all addresses for the specified user ordered by most recent update.
func (r *AddressRepository) List(ctx context.Context, userID string) ([]domain.Address, error) {
	coll, err := r.collection(ctx, userID)
	if err != nil {
		return nil, err
	}

	iter := coll.OrderBy("updatedAt", firestore.Desc).Documents(ctx)
	defer iter.Stop()

	var results []domain.Address
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, pfirestore.WrapError("addresses.list", err)
		}
		doc, err := decodeAddressDocument(snap)
		if err != nil {
			return nil, err
		}
		if doc.NormalizedHash == "" {
			doc.NormalizedHash = computeAddressHash(doc)
		}
		results = append(results, doc)
	}
	return results, nil
}

// Upsert creates or updates an address, guaranteeing deduplication and default management.
func (r *AddressRepository) Upsert(ctx context.Context, userID string, addressID *string, addr domain.Address) (domain.Address, error) {
	coll, err := r.collection(ctx, userID)
	if err != nil {
		return domain.Address{}, err
	}

	hash := strings.TrimSpace(addr.NormalizedHash)
	if hash == "" {
		hash = computeAddressHash(addr)
	}

	var saved domain.Address
	err = r.provider.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		var docRef *firestore.DocumentRef
		var existingSnap *firestore.DocumentSnapshot

		if addressID != nil {
			id := strings.TrimSpace(*addressID)
			if id != "" {
				docRef = coll.Doc(id)
			}
		}

		if docRef == nil {
			query := coll.Where("hash", "==", hash).Limit(1)
			docsIter := tx.Documents(query)
			snaps, err := docsIter.GetAll()
			if err != nil {
				if status.Code(err) != codes.NotFound {
					return err
				}
			}
			if len(snaps) > 0 {
				existingSnap = snaps[0]
				docRef = existingSnap.Ref
			}
		}

		if docRef == nil {
			docRef = coll.NewDoc()
		}

		var doc addressDocument
		snapshot, err := tx.Get(docRef)
		switch status.Code(err) {
		case codes.NotFound:
			// new document, leave doc zeroed
		case codes.OK:
			if existingSnap == nil {
				existingSnap = snapshot
			}
		default:
			return err
		}

		if existingSnap != nil {
			if err := existingSnap.DataTo(&doc); err != nil {
				return fmt.Errorf("decode address %s: %w", existingSnap.Ref.ID, err)
			}
		}

		now := time.Now().UTC()
		if doc.CreatedAt.IsZero() {
			if !addr.CreatedAt.IsZero() {
				doc.CreatedAt = addr.CreatedAt.UTC()
			} else {
				doc.CreatedAt = now
			}
		}
		doc.UpdatedAt = now
		doc.Label = strings.TrimSpace(addr.Label)
		doc.Recipient = addr.Recipient
		doc.Company = strings.TrimSpace(addr.Company)
		doc.Line1 = addr.Line1
		doc.Line2 = cloneOptionalString(addr.Line2)
		doc.City = addr.City
		doc.State = cloneOptionalString(addr.State)
		doc.PostalCode = addr.PostalCode
		doc.Country = addr.Country
		doc.Phone = cloneOptionalString(addr.Phone)
		doc.DefaultShipping = addr.DefaultShipping
		doc.DefaultBilling = addr.DefaultBilling
		doc.Hash = hash

		if err := tx.Set(docRef, doc); err != nil {
			return err
		}

		if doc.DefaultShipping {
			if err := r.clearDefault(ctx, tx, coll, docRef.ID, "defaultShipping"); err != nil {
				return err
			}
		}
		if doc.DefaultBilling {
			if err := r.clearDefault(ctx, tx, coll, docRef.ID, "defaultBilling"); err != nil {
				return err
			}
		}

		saved = doc.toDomain(docRef.ID)
		saved.NormalizedHash = hash
		return nil
	})
	if err != nil {
		return domain.Address{}, pfirestore.WrapError("addresses.upsert", err)
	}
	return saved, nil
}

// Delete removes the specified address document.
func (r *AddressRepository) Delete(ctx context.Context, userID string, addressID string) error {
	coll, err := r.collection(ctx, userID)
	if err != nil {
		return err
	}
	id := strings.TrimSpace(addressID)
	if id == "" {
		return errors.New("address repository: address id is required")
	}
	if _, err := coll.Doc(id).Delete(ctx); err != nil {
		return pfirestore.WrapError("addresses.delete", err)
	}
	return nil
}

func (r *AddressRepository) Get(ctx context.Context, userID string, addressID string) (domain.Address, error) {
	coll, err := r.collection(ctx, userID)
	if err != nil {
		return domain.Address{}, err
	}
	id := strings.TrimSpace(addressID)
	if id == "" {
		return domain.Address{}, errors.New("address repository: address id is required")
	}
	snap, err := coll.Doc(id).Get(ctx)
	if err != nil {
		return domain.Address{}, pfirestore.WrapError("addresses.get", err)
	}
	addr, err := decodeAddressDocument(snap)
	if err != nil {
		return domain.Address{}, err
	}
	return addr, nil
}

func (r *AddressRepository) FindByHash(ctx context.Context, userID string, hash string) (domain.Address, bool, error) {
	coll, err := r.collection(ctx, userID)
	if err != nil {
		return domain.Address{}, false, err
	}
	trimmed := strings.TrimSpace(hash)
	if trimmed == "" {
		return domain.Address{}, false, nil
	}
	iter := coll.Where("hash", "==", trimmed).Limit(1).Documents(ctx)
	defer iter.Stop()
	snaps, err := iter.GetAll()
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return domain.Address{}, false, nil
		}
		return domain.Address{}, false, pfirestore.WrapError("addresses.findByHash", err)
	}
	if len(snaps) == 0 {
		return domain.Address{}, false, nil
	}
	addr, err := decodeAddressDocument(snaps[0])
	if err != nil {
		return domain.Address{}, false, err
	}
	return addr, true, nil
}

func (r *AddressRepository) HasAny(ctx context.Context, userID string) (bool, error) {
	coll, err := r.collection(ctx, userID)
	if err != nil {
		return false, err
	}
	iter := coll.Limit(1).Documents(ctx)
	defer iter.Stop()
	if _, err := iter.Next(); err != nil {
		if errors.Is(err, iterator.Done) {
			return false, nil
		}
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, pfirestore.WrapError("addresses.hasAny", err)
	}
	return true, nil
}

func (r *AddressRepository) SetDefaultFlags(ctx context.Context, userID string, addressID string, shipping, billing *bool) (domain.Address, error) {
	coll, err := r.collection(ctx, userID)
	if err != nil {
		return domain.Address{}, err
	}
	id := strings.TrimSpace(addressID)
	if id == "" {
		return domain.Address{}, errors.New("address repository: address id is required")
	}

	var saved domain.Address
	err = r.provider.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		docRef := coll.Doc(id)
		snap, err := tx.Get(docRef)
		if err != nil {
			return err
		}
		var doc addressDocument
		if err := snap.DataTo(&doc); err != nil {
			return fmt.Errorf("decode address %s: %w", snap.Ref.ID, err)
		}

		updates := make([]firestore.Update, 0, 3)
		now := time.Now().UTC()
		if shipping != nil {
			doc.DefaultShipping = *shipping
			updates = append(updates, firestore.Update{Path: "defaultShipping", Value: *shipping})
		}
		if billing != nil {
			doc.DefaultBilling = *billing
			updates = append(updates, firestore.Update{Path: "defaultBilling", Value: *billing})
		}
		if len(updates) > 0 {
			updates = append(updates, firestore.Update{Path: "updatedAt", Value: now})
			if err := tx.Update(docRef, updates); err != nil {
				return err
			}
			doc.UpdatedAt = now
		}

		if shipping != nil && *shipping {
			if err := r.clearDefault(ctx, tx, coll, docRef.ID, "defaultShipping"); err != nil {
				return err
			}
		}
		if billing != nil && *billing {
			if err := r.clearDefault(ctx, tx, coll, docRef.ID, "defaultBilling"); err != nil {
				return err
			}
		}

		saved = doc.toDomain(docRef.ID)
		saved.NormalizedHash = doc.Hash
		return nil
	})
	if err != nil {
		return domain.Address{}, pfirestore.WrapError("addresses.setDefaultFlags", err)
	}
	return saved, nil
}

func (r *AddressRepository) collection(ctx context.Context, userID string) (*firestore.CollectionRef, error) {
	if r == nil || r.provider == nil {
		return nil, errors.New("address repository not initialised")
	}
	uid := strings.TrimSpace(userID)
	if uid == "" {
		return nil, errors.New("address repository: user id is required")
	}
	client, err := r.provider.Client(ctx)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf(addressCollectionPattern, uid)
	return client.Collection(path), nil
}

func (r *AddressRepository) clearDefault(ctx context.Context, tx *firestore.Transaction, coll *firestore.CollectionRef, currentID, field string) error {
	query := coll.Where(field, "==", true).OrderBy("updatedAt", firestore.Desc).Limit(10)
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
		if err := tx.Update(snap.Ref, []firestore.Update{{Path: field, Value: false}}); err != nil {
			return err
		}
	}
	return nil
}

func decodeAddressDocument(snapshot *firestore.DocumentSnapshot) (domain.Address, error) {
	var doc addressDocument
	if err := snapshot.DataTo(&doc); err != nil {
		return domain.Address{}, fmt.Errorf("decode address %s: %w", snapshot.Ref.ID, err)
	}
	addr := doc.toDomain(snapshot.Ref.ID)
	return addr, nil
}

type addressDocument struct {
	Label           string    `firestore:"label,omitempty"`
	Recipient       string    `firestore:"recipient"`
	Company         string    `firestore:"company,omitempty"`
	Line1           string    `firestore:"line1"`
	Line2           *string   `firestore:"line2,omitempty"`
	City            string    `firestore:"city"`
	State           *string   `firestore:"state,omitempty"`
	PostalCode      string    `firestore:"postalCode"`
	Country         string    `firestore:"country"`
	Phone           *string   `firestore:"phone,omitempty"`
	DefaultShipping bool      `firestore:"defaultShipping"`
	DefaultBilling  bool      `firestore:"defaultBilling"`
	Hash            string    `firestore:"hash"`
	CreatedAt       time.Time `firestore:"createdAt"`
	UpdatedAt       time.Time `firestore:"updatedAt"`
}

func (d addressDocument) toDomain(id string) domain.Address {
	return domain.Address{
		ID:              id,
		Label:           d.Label,
		Recipient:       d.Recipient,
		Company:         d.Company,
		Line1:           d.Line1,
		Line2:           cloneOptionalString(d.Line2),
		City:            d.City,
		State:           cloneOptionalString(d.State),
		PostalCode:      d.PostalCode,
		Country:         d.Country,
		Phone:           cloneOptionalString(d.Phone),
		DefaultShipping: d.DefaultShipping,
		DefaultBilling:  d.DefaultBilling,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
		NormalizedHash:  d.Hash,
	}
}

func cloneOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	if strings.TrimSpace(cloned) == "" {
		return nil
	}
	return &cloned
}

func computeAddressHash(addr domain.Address) string {
	parts := []string{
		strings.ToLower(strings.TrimSpace(addr.Recipient)),
		strings.ToLower(strings.TrimSpace(addr.Company)),
		strings.ToLower(strings.TrimSpace(addr.Line1)),
		strings.ToLower(optionalValue(addr.Line2)),
		strings.ToLower(strings.TrimSpace(addr.City)),
		strings.ToLower(optionalValue(addr.State)),
		strings.ToLower(strings.TrimSpace(addr.PostalCode)),
		strings.ToLower(strings.TrimSpace(addr.Country)),
	}
	input := strings.Join(parts, "|")
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])
}

func optionalValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

// Ensure interface compliance.
var _ repositories.AddressRepository = (*AddressRepository)(nil)
