package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"

	domain "github.com/hanko-field/api/internal/domain"
	pfirestore "github.com/hanko-field/api/internal/platform/firestore"
)

const (
	designRegistrabilityCollection = "registrability"
	designRegistrabilityDocID      = "latest"
)

// DesignRegistrabilityRepository stores registrability assessment results for designs.
type DesignRegistrabilityRepository struct {
	provider *pfirestore.Provider
}

// NewDesignRegistrabilityRepository constructs a Firestore-backed registrability repository.
func NewDesignRegistrabilityRepository(provider *pfirestore.Provider) (*DesignRegistrabilityRepository, error) {
	if provider == nil {
		return nil, errors.New("design registrability repository: firestore provider is required")
	}
	return &DesignRegistrabilityRepository{provider: provider}, nil
}

// Get returns the cached registrability assessment for the design.
func (r *DesignRegistrabilityRepository) Get(ctx context.Context, designID string) (domain.RegistrabilityCheckResult, error) {
	docRef, err := r.document(ctx, designID)
	if err != nil {
		return domain.RegistrabilityCheckResult{}, err
	}
	snap, err := docRef.Get(ctx)
	if err != nil {
		return domain.RegistrabilityCheckResult{}, pfirestore.WrapError("design_registrability.get", err)
	}
	var doc registrabilityDocument
	if err := snap.DataTo(&doc); err != nil {
		return domain.RegistrabilityCheckResult{}, fmt.Errorf("design registrability repository: decode document %s: %w", snap.Ref.ID, err)
	}
	return decodeRegistrabilityDocument(docRef.Parent.Parent.ID, doc), nil
}

// Save upserts the registrability assessment for the design.
func (r *DesignRegistrabilityRepository) Save(ctx context.Context, result domain.RegistrabilityCheckResult) error {
	docRef, err := r.document(ctx, result.DesignID)
	if err != nil {
		return err
	}
	doc := encodeRegistrabilityDocument(result)
	if _, err := docRef.Set(ctx, doc); err != nil {
		return pfirestore.WrapError("design_registrability.save", err)
	}
	return nil
}

func (r *DesignRegistrabilityRepository) document(ctx context.Context, designID string) (*firestore.DocumentRef, error) {
	if r == nil || r.provider == nil {
		return nil, errors.New("design registrability repository not initialised")
	}
	designID = normalizeRegistrabilityDesignID(designID)
	if designID == "" {
		return nil, errors.New("design registrability repository: design id is required")
	}
	client, err := r.provider.Client(ctx)
	if err != nil {
		return nil, err
	}
	return client.Collection(designsCollection).Doc(designID).Collection(designRegistrabilityCollection).Doc(designRegistrabilityDocID), nil
}

type registrabilityDocument struct {
	Status      string         `firestore:"status"`
	Passed      bool           `firestore:"passed"`
	Score       *float64       `firestore:"score,omitempty"`
	Reasons     []string       `firestore:"reasons"`
	RequestedAt time.Time      `firestore:"requestedAt"`
	ExpiresAt   *time.Time     `firestore:"expiresAt,omitempty"`
	Metadata    map[string]any `firestore:"metadata,omitempty"`
	UpdatedAt   time.Time      `firestore:"updatedAt"`
}

func encodeRegistrabilityDocument(result domain.RegistrabilityCheckResult) registrabilityDocument {
	now := result.RequestedAt
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	doc := registrabilityDocument{
		Status:      strings.TrimSpace(result.Status),
		Passed:      result.Passed,
		Reasons:     copyRegistrabilityStrings(result.Reasons),
		RequestedAt: now,
		Metadata:    copyRegistrabilityMap(result.Metadata),
		UpdatedAt:   now,
	}
	if result.Score != nil {
		doc.Score = result.Score
	}
	if result.ExpiresAt != nil && !result.ExpiresAt.IsZero() {
		expiry := result.ExpiresAt.UTC()
		doc.ExpiresAt = &expiry
	}
	return doc
}

func decodeRegistrabilityDocument(designID string, doc registrabilityDocument) domain.RegistrabilityCheckResult {
	var expires *time.Time
	if doc.ExpiresAt != nil && !doc.ExpiresAt.IsZero() {
		expiry := doc.ExpiresAt.UTC()
		expires = &expiry
	}
	return domain.RegistrabilityCheckResult{
		DesignID:    strings.TrimSpace(designID),
		Status:      strings.TrimSpace(doc.Status),
		Passed:      doc.Passed,
		Score:       doc.Score,
		Reasons:     copyRegistrabilityStrings(doc.Reasons),
		RequestedAt: doc.RequestedAt.UTC(),
		ExpiresAt:   expires,
		Metadata:    copyRegistrabilityMap(doc.Metadata),
	}
}

func normalizeRegistrabilityDesignID(designID string) string {
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

func copyRegistrabilityStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func copyRegistrabilityMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]any, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}
