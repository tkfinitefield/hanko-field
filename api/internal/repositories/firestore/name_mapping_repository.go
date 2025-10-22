package firestore

import (
	"context"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	domain "github.com/hanko-field/api/internal/domain"
	pfirestore "github.com/hanko-field/api/internal/platform/firestore"
	"github.com/hanko-field/api/internal/repositories"
)

const nameMappingsCollection = "nameMappings"

// NameMappingRepository persists transliteration results and lookup metadata.
type NameMappingRepository struct {
	base *pfirestore.BaseRepository[domain.NameMapping]
}

// NewNameMappingRepository constructs a Firestore-backed name mapping repository.
func NewNameMappingRepository(provider *pfirestore.Provider) (*NameMappingRepository, error) {
	if provider == nil {
		return nil, errors.New("name mapping repository: firestore provider is required")
	}

	encoder := func(ctx context.Context, value domain.NameMapping) (any, error) {
		return encodeNameMappingDocument(value), nil
	}
	decoder := func(ctx context.Context, snap *firestore.DocumentSnapshot) (domain.NameMapping, error) {
		var doc nameMappingDocument
		if err := snap.DataTo(&doc); err != nil {
			return domain.NameMapping{}, err
		}
		doc.ID = snap.Ref.ID
		if doc.CreatedAt.IsZero() {
			doc.CreatedAt = snap.CreateTime
		}
		if doc.UpdatedAt.IsZero() {
			doc.UpdatedAt = snap.UpdateTime
		}
		return decodeNameMappingDocument(doc), nil
	}

	base := pfirestore.NewBaseRepository[domain.NameMapping](provider, nameMappingsCollection, encoder, decoder)
	return &NameMappingRepository{base: base}, nil
}

// Insert stores a new mapping document.
func (r *NameMappingRepository) Insert(ctx context.Context, mapping domain.NameMapping) error {
	if r == nil || r.base == nil {
		return errors.New("name mapping repository not initialised")
	}
	mapping.ID = strings.TrimSpace(mapping.ID)
	if mapping.ID == "" {
		return errors.New("name mapping repository: id is required")
	}

	docRef, err := r.base.DocumentRef(ctx, mapping.ID)
	if err != nil {
		return err
	}
	payload := encodeNameMappingDocument(mapping)
	if _, err := docRef.Create(ctx, payload); err != nil {
		return pfirestore.WrapError("name_mappings.insert", err)
	}
	return nil
}

// Update replaces the mapping document state.
func (r *NameMappingRepository) Update(ctx context.Context, mapping domain.NameMapping) error {
	if r == nil || r.base == nil {
		return errors.New("name mapping repository not initialised")
	}
	mapping.ID = strings.TrimSpace(mapping.ID)
	if mapping.ID == "" {
		return errors.New("name mapping repository: id is required")
	}

	docRef, err := r.base.DocumentRef(ctx, mapping.ID)
	if err != nil {
		return err
	}
	payload := encodeNameMappingDocument(mapping)
	if _, err := docRef.Set(ctx, payload); err != nil {
		return pfirestore.WrapError("name_mappings.update", err)
	}
	return nil
}

// FindByID loads a mapping by its identifier.
func (r *NameMappingRepository) FindByID(ctx context.Context, mappingID string) (domain.NameMapping, error) {
	if r == nil || r.base == nil {
		return domain.NameMapping{}, errors.New("name mapping repository not initialised")
	}
	mappingID = strings.TrimSpace(mappingID)
	if mappingID == "" {
		return domain.NameMapping{}, errors.New("name mapping repository: id is required")
	}
	doc, err := r.base.Get(ctx, mappingID)
	if err != nil {
		return domain.NameMapping{}, err
	}
	return doc.Data, nil
}

// FindByLookup returns the most recent mapping for a user and latin/locale pair.
func (r *NameMappingRepository) FindByLookup(ctx context.Context, userID string, latin string, locale string) (domain.NameMapping, error) {
	if r == nil || r.base == nil {
		return domain.NameMapping{}, errors.New("name mapping repository not initialised")
	}
	key := buildLookupKey(userID, latin, locale)

	docs, err := r.base.Query(ctx, func(q firestore.Query) firestore.Query {
		return q.Where("lookupKey", "==", key).OrderBy("updatedAt", firestore.Desc).Limit(1)
	})
	if err != nil {
		return domain.NameMapping{}, err
	}
	if len(docs) == 0 {
		return domain.NameMapping{}, pfirestore.WrapError("name_mappings.lookup", status.Error(codes.NotFound, "name mapping not found"))
	}
	return docs[0].Data, nil
}

func encodeNameMappingDocument(mapping domain.NameMapping) nameMappingDocument {
	candidates := make([]nameMappingCandidateDocument, 0, len(mapping.Candidates))
	for _, cand := range mapping.Candidates {
		candidates = append(candidates, encodeCandidateDocument(cand))
	}

	var selected *nameMappingCandidateDocument
	if mapping.SelectedCandidate != nil {
		encoded := encodeCandidateDocument(*mapping.SelectedCandidate)
		selected = &encoded
	}

	return nameMappingDocument{
		UserRef:           mapping.UserRef,
		UserID:            mapping.UserID,
		Latin:             strings.TrimSpace(mapping.Input.Latin),
		Locale:            strings.TrimSpace(mapping.Input.Locale),
		Gender:            strings.TrimSpace(mapping.Input.Gender),
		Context:           cloneStringMap(mapping.Input.Context),
		Status:            string(mapping.Status),
		Source:            strings.TrimSpace(mapping.Source),
		Candidates:        candidates,
		SelectedCandidate: selected,
		SelectedAt:        cloneTime(mapping.SelectedAt),
		ExpiresAt:         cloneTime(mapping.ExpiresAt),
		CreatedAt:         mapping.CreatedAt.UTC(),
		UpdatedAt:         mapping.UpdatedAt.UTC(),
		Metadata:          cloneMetadata(mapping.Metadata),
		LookupKey:         buildLookupKey(mapping.UserID, mapping.Input.Latin, mapping.Input.Locale),
	}
}

func encodeCandidateDocument(candidate domain.NameMappingCandidate) nameMappingCandidateDocument {
	return nameMappingCandidateDocument{
		ID:       candidate.ID,
		Display:  candidate.Kanji,
		Readings: cloneSlice(candidate.Kana),
		Score:    candidate.Score,
		Notes:    candidate.Notes,
		Metadata: cloneMetadata(candidate.Metadata),
	}
}

func decodeNameMappingDocument(doc nameMappingDocument) domain.NameMapping {
	candidates := make([]domain.NameMappingCandidate, 0, len(doc.Candidates))
	for _, cand := range doc.Candidates {
		candidates = append(candidates, decodeCandidateDocument(cand))
	}

	var selected *domain.NameMappingCandidate
	if doc.SelectedCandidate != nil {
		decoded := decodeCandidateDocument(*doc.SelectedCandidate)
		selected = &decoded
	}

	return domain.NameMapping{
		ID:      doc.ID,
		UserID:  doc.UserID,
		UserRef: doc.UserRef,
		Input: domain.NameMappingInput{
			Latin:   doc.Latin,
			Locale:  doc.Locale,
			Gender:  doc.Gender,
			Context: cloneStringMap(doc.Context),
		},
		Status:            domain.NameMappingStatus(doc.Status),
		Source:            doc.Source,
		Candidates:        candidates,
		SelectedCandidate: selected,
		SelectedAt:        cloneTime(doc.SelectedAt),
		ExpiresAt:         cloneTime(doc.ExpiresAt),
		CreatedAt:         doc.CreatedAt.UTC(),
		UpdatedAt:         doc.UpdatedAt.UTC(),
		Metadata:          cloneMetadata(doc.Metadata),
	}
}

func decodeCandidateDocument(doc nameMappingCandidateDocument) domain.NameMappingCandidate {
	return domain.NameMappingCandidate{
		ID:       doc.ID,
		Kanji:    doc.Display,
		Kana:     cloneSlice(doc.Readings),
		Score:    doc.Score,
		Notes:    doc.Notes,
		Metadata: cloneMetadata(doc.Metadata),
	}
}

type nameMappingDocument struct {
	ID                string                         `firestore:"-"`
	UserRef           string                         `firestore:"userRef"`
	UserID            string                         `firestore:"userUid"`
	Latin             string                         `firestore:"latin"`
	Locale            string                         `firestore:"locale"`
	Gender            string                         `firestore:"gender,omitempty"`
	Context           map[string]string              `firestore:"context,omitempty"`
	Status            string                         `firestore:"status"`
	Source            string                         `firestore:"source,omitempty"`
	Candidates        []nameMappingCandidateDocument `firestore:"candidates"`
	SelectedCandidate *nameMappingCandidateDocument  `firestore:"selectedCandidate,omitempty"`
	SelectedAt        *time.Time                     `firestore:"selectedAt,omitempty"`
	ExpiresAt         *time.Time                     `firestore:"expiresAt,omitempty"`
	CreatedAt         time.Time                      `firestore:"createdAt"`
	UpdatedAt         time.Time                      `firestore:"updatedAt"`
	Metadata          map[string]any                 `firestore:"metadata,omitempty"`
	LookupKey         string                         `firestore:"lookupKey"`
}

type nameMappingCandidateDocument struct {
	ID       string         `firestore:"id"`
	Display  string         `firestore:"display"`
	Readings []string       `firestore:"readings,omitempty"`
	Score    float64        `firestore:"score"`
	Notes    string         `firestore:"notes,omitempty"`
	Metadata map[string]any `firestore:"metadata,omitempty"`
}

func cloneMetadata(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}

func buildLookupKey(userID string, latin string, locale string) string {
	trimmedUser := strings.ToLower(strings.TrimSpace(userID))
	normalizedLatin := strings.ToLower(strings.Join(strings.Fields(latin), " "))
	normalizedLocale := strings.ToLower(strings.TrimSpace(locale))
	return strings.Join([]string{trimmedUser, normalizedLatin, normalizedLocale}, "|")
}

var _ repositories.NameMappingRepository = (*NameMappingRepository)(nil)
