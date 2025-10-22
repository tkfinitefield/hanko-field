package services

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

type fakeNameMappingRepository struct {
	existing *domain.NameMapping
	findErr  error
	inserted []domain.NameMapping
	updated  []domain.NameMapping
}

func (r *fakeNameMappingRepository) Insert(ctx context.Context, mapping domain.NameMapping) error {
	copy := mapping
	r.inserted = append(r.inserted, copy)
	r.existing = &r.inserted[len(r.inserted)-1]
	return nil
}

func (r *fakeNameMappingRepository) Update(ctx context.Context, mapping domain.NameMapping) error {
	copy := mapping
	r.updated = append(r.updated, copy)
	r.existing = &r.updated[len(r.updated)-1]
	return nil
}

func (r *fakeNameMappingRepository) FindByID(ctx context.Context, mappingID string) (domain.NameMapping, error) {
	if r.existing != nil && r.existing.ID == mappingID {
		return *r.existing, nil
	}
	return domain.NameMapping{}, nmRepoNotFoundError{}
}

func (r *fakeNameMappingRepository) FindByLookup(ctx context.Context, userID string, latin string, locale string) (domain.NameMapping, error) {
	if r.findErr != nil {
		return domain.NameMapping{}, r.findErr
	}
	if r.existing != nil {
		return *r.existing, nil
	}
	return domain.NameMapping{}, nmRepoNotFoundError{}
}

type nmRepoNotFoundError struct{}

func (nmRepoNotFoundError) Error() string       { return "not found" }
func (nmRepoNotFoundError) IsNotFound() bool    { return true }
func (nmRepoNotFoundError) IsConflict() bool    { return false }
func (nmRepoNotFoundError) IsUnavailable() bool { return false }

type stubTransliterationProvider struct {
	result TransliterationResult
	err    error
	calls  []TransliterationRequest
}

func (s *stubTransliterationProvider) Transliterate(ctx context.Context, req TransliterationRequest) (TransliterationResult, error) {
	s.calls = append(s.calls, req)
	if s.err != nil {
		return TransliterationResult{}, s.err
	}
	return s.result, nil
}

func TestNameMappingServiceConvert_GeneratesMapping(t *testing.T) {
	now := time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC)
	repo := &fakeNameMappingRepository{}
	provider := &stubTransliterationProvider{
		result: TransliterationResult{
			Provider: "external",
			Candidates: []TransliterationCandidate{
				{ID: "ext-1", Kanji: "佐藤", Kana: []string{"サトウ"}, Score: 0.95, Notes: "primary"},
			},
		},
	}
	svc, err := NewNameMappingService(NameMappingServiceDeps{
		Repository:     repo,
		Transliterator: provider,
		Clock:          func() time.Time { return now },
		IDGenerator:    func() string { return "abc123" },
	})
	if err != nil {
		t.Fatalf("expected no error constructing service: %v", err)
	}

	result, err := svc.ConvertName(context.Background(), NameConversionCommand{
		UserID: "user-1",
		Latin:  "Sato",
		Locale: "EN",
		Gender: "neutral",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "nmap_abc123" {
		t.Fatalf("expected id nmap_abc123, got %s", result.ID)
	}
	if result.Source != "external" {
		t.Fatalf("expected source external, got %s", result.Source)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(result.Candidates))
	}
	if len(repo.inserted) != 1 {
		t.Fatalf("expected insert to be called once, got %d", len(repo.inserted))
	}
	if len(repo.updated) != 0 {
		t.Fatalf("expected update not called, got %d", len(repo.updated))
	}
	if len(provider.calls) != 1 {
		t.Fatalf("expected provider to be called once, got %d", len(provider.calls))
	}
	call := provider.calls[0]
	if call.Locale != "en" {
		t.Fatalf("expected normalised locale en, got %s", call.Locale)
	}
	if call.Latin != "Sato" {
		t.Fatalf("expected latin Sato, got %s", call.Latin)
	}
	if result.CreatedAt != now {
		t.Fatalf("expected createdAt equal to now")
	}
	if result.ExpiresAt == nil {
		t.Fatalf("expected expiresAt to be set")
	}
	if result.ExpiresAt.Sub(now) <= 0 {
		t.Fatalf("expected expiresAt in future")
	}
}

func TestNameMappingServiceConvert_UsesCache(t *testing.T) {
	now := time.Now().UTC()
	existing := domain.NameMapping{
		ID:         "nmap_cached",
		UserID:     "user-1",
		UserRef:    "/users/user-1",
		Input:      domain.NameMappingInput{Latin: "Sato", Locale: "en"},
		Status:     domain.NameMappingStatusReady,
		Source:     "external",
		CreatedAt:  now.Add(-time.Hour),
		UpdatedAt:  now.Add(-30 * time.Minute),
		ExpiresAt:  pointerToTime(now.Add(2 * time.Hour)),
		Candidates: []domain.NameMappingCandidate{{ID: "c1", Kanji: "佐藤", Score: 0.9}},
	}
	repo := &fakeNameMappingRepository{existing: &existing}
	provider := &stubTransliterationProvider{}
	svc, err := NewNameMappingService(NameMappingServiceDeps{
		Repository:     repo,
		Transliterator: provider,
		Clock:          func() time.Time { return now },
		IDGenerator:    func() string { return "newid" },
	})
	if err != nil {
		t.Fatalf("expected no error constructing service: %v", err)
	}

	result, err := svc.ConvertName(context.Background(), NameConversionCommand{
		UserID: "user-1",
		Latin:  "Sato",
		Locale: "en",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != existing.ID {
		t.Fatalf("expected cached id %s, got %s", existing.ID, result.ID)
	}
	if len(provider.calls) != 0 {
		t.Fatalf("expected provider not called, got %d", len(provider.calls))
	}
	if len(repo.inserted) != 0 {
		t.Fatalf("expected insert not called")
	}
	if len(repo.updated) != 0 {
		t.Fatalf("expected update not called")
	}
}

func TestNameMappingServiceConvert_ForceRefresh(t *testing.T) {
	now := time.Now().UTC()
	created := now.Add(-48 * time.Hour)
	existing := domain.NameMapping{
		ID:         "nmap_existing",
		UserID:     "user-2",
		UserRef:    "/users/user-2",
		Input:      domain.NameMappingInput{Latin: "Lee", Locale: "en"},
		Status:     domain.NameMappingStatusReady,
		Source:     "external",
		CreatedAt:  created,
		UpdatedAt:  now.Add(-24 * time.Hour),
		ExpiresAt:  pointerToTime(now.Add(-time.Hour)),
		Candidates: []domain.NameMappingCandidate{{ID: "old", Kanji: "李", Score: 0.4}},
	}
	repo := &fakeNameMappingRepository{existing: &existing}
	provider := &stubTransliterationProvider{
		result: TransliterationResult{
			Provider:   "external",
			Candidates: []TransliterationCandidate{{ID: "new", Kanji: "麗", Score: 0.92}},
		},
	}
	svc, err := NewNameMappingService(NameMappingServiceDeps{
		Repository:     repo,
		Transliterator: provider,
		Clock:          func() time.Time { return now },
		IDGenerator:    func() string { return "unused" },
	})
	if err != nil {
		t.Fatalf("expected no error constructing service: %v", err)
	}

	result, err := svc.ConvertName(context.Background(), NameConversionCommand{
		UserID:       "user-2",
		Latin:        "Lee",
		Locale:       "EN",
		ForceRefresh: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != existing.ID {
		t.Fatalf("expected same id %s, got %s", existing.ID, result.ID)
	}
	if len(repo.updated) != 1 {
		t.Fatalf("expected update called once, got %d", len(repo.updated))
	}
	if result.CreatedAt != created {
		t.Fatalf("expected createdAt preserved, got %v", result.CreatedAt)
	}
	if len(result.Candidates) != 1 || result.Candidates[0].ID != "new" {
		t.Fatalf("expected new candidate applied")
	}
}

func TestNameMappingServiceConvert_UnsupportedLocale(t *testing.T) {
	repo := &fakeNameMappingRepository{}
	provider := &stubTransliterationProvider{
		err: ErrTransliterationUnsupportedLocale,
	}
	svc, err := NewNameMappingService(NameMappingServiceDeps{
		Repository:     repo,
		Transliterator: provider,
		Clock:          time.Now,
		IDGenerator:    func() string { return "id" },
	})
	if err != nil {
		t.Fatalf("expected no error constructing service: %v", err)
	}

	_, err = svc.ConvertName(context.Background(), NameConversionCommand{UserID: "user", Latin: "Name", Locale: "fr"})
	if !errors.Is(err, ErrNameMappingUnsupportedLocale) {
		t.Fatalf("expected unsupported locale error, got %v", err)
	}
}

func TestNameMappingServiceConvert_Fallback(t *testing.T) {
	repo := &fakeNameMappingRepository{}
	provider := &stubTransliterationProvider{
		err: ErrTransliterationUnavailable,
	}
	svc, err := NewNameMappingService(NameMappingServiceDeps{
		Repository:     repo,
		Transliterator: provider,
		Clock:          func() time.Time { return time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC) },
		IDGenerator:    func() string { return "fallback" },
	})
	if err != nil {
		t.Fatalf("expected no error constructing service: %v", err)
	}

	result, err := svc.ConvertName(context.Background(), NameConversionCommand{UserID: "user", Latin: "Nao", Locale: "en"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Source == "" {
		t.Fatalf("expected source populated for fallback")
	}
	if len(result.Candidates) == 0 {
		t.Fatalf("expected fallback candidates")
	}
	if len(repo.inserted) != 1 {
		t.Fatalf("expected insert called once, got %d", len(repo.inserted))
	}
}

func pointerToTime(t time.Time) *time.Time {
	value := t
	return &value
}

var _ repositories.NameMappingRepository = (*fakeNameMappingRepository)(nil)
