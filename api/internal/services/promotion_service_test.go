package services

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

func TestPromotionService_GetPublicPromotion_Success(t *testing.T) {
	now := time.Date(2024, time.June, 1, 9, 0, 0, 0, time.UTC)
	repo := &stubPromotionRepository{
		promotion: domain.Promotion{
			Code:              "SPRING10",
			Status:            "active",
			DescriptionPublic: "Spring offer",
			StartsAt:          now.Add(-time.Hour),
			EndsAt:            now.Add(2 * time.Hour),
			EligibleAudiences: []string{"new", "vip"},
		},
	}

	svc, err := NewPromotionService(PromotionServiceDeps{
		Promotions: repo,
		Clock: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatalf("NewPromotionService: %v", err)
	}

	result, err := svc.GetPublicPromotion(context.Background(), " spring10 ")
	if err != nil {
		t.Fatalf("GetPublicPromotion returned error: %v", err)
	}
	if !result.IsAvailable {
		t.Fatalf("expected promotion to be available")
	}
	if result.Code != "SPRING10" {
		t.Fatalf("expected code SPRING10 got %s", result.Code)
	}
	if result.DescriptionPublic != "Spring offer" {
		t.Fatalf("unexpected description %q", result.DescriptionPublic)
	}
	if len(result.EligibleAudiences) != 2 {
		t.Fatalf("unexpected audiences %v", result.EligibleAudiences)
	}
	if repo.lastCode != "SPRING10" {
		t.Fatalf("repository looked up wrong code %s", repo.lastCode)
	}
}

func TestPromotionService_GetPublicPromotion_NotFound(t *testing.T) {
	repo := &stubPromotionRepository{
		err: &stubPromotionRepoError{notFound: true},
	}
	svc, err := NewPromotionService(PromotionServiceDeps{Promotions: repo})
	if err != nil {
		t.Fatalf("NewPromotionService: %v", err)
	}

	_, err = svc.GetPublicPromotion(context.Background(), "MISSING")
	if !errors.Is(err, ErrPromotionNotFound) {
		t.Fatalf("expected ErrPromotionNotFound got %v", err)
	}
}

func TestPromotionService_GetPublicPromotion_UnavailableFlags(t *testing.T) {
	repo := &stubPromotionRepository{
		promotion: domain.Promotion{
			Code:         "PRIVATE",
			Status:       "active",
			InternalOnly: true,
		},
	}
	svc, err := NewPromotionService(PromotionServiceDeps{Promotions: repo})
	if err != nil {
		t.Fatalf("NewPromotionService: %v", err)
	}

	if _, err := svc.GetPublicPromotion(context.Background(), "PRIVATE"); !errors.Is(err, ErrPromotionUnavailable) {
		t.Fatalf("expected ErrPromotionUnavailable got %v", err)
	}

	repo.promotion.InternalOnly = false
	repo.promotion.RequiresAuth = true
	if _, err := svc.GetPublicPromotion(context.Background(), "PRIVATE"); !errors.Is(err, ErrPromotionUnavailable) {
		t.Fatalf("expected ErrPromotionUnavailable for requiresAuth flag got %v", err)
	}
}

func TestPromotionService_GetPublicPromotion_NotYetActive(t *testing.T) {
	now := time.Date(2024, time.July, 1, 10, 0, 0, 0, time.UTC)
	repo := &stubPromotionRepository{
		promotion: domain.Promotion{
			Code:     "LATER",
			Status:   "active",
			StartsAt: now.Add(time.Hour),
			EndsAt:   now.Add(24 * time.Hour),
		},
	}
	svc, err := NewPromotionService(PromotionServiceDeps{
		Promotions: repo,
		Clock: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatalf("NewPromotionService: %v", err)
	}

	result, err := svc.GetPublicPromotion(context.Background(), "LATER")
	if err != nil {
		t.Fatalf("GetPublicPromotion returned error: %v", err)
	}
	if result.IsAvailable {
		t.Fatalf("expected promotion to be unavailable before start")
	}
}

type stubPromotionRepository struct {
	promotion domain.Promotion
	err       error
	lastCode  string
}

func (s *stubPromotionRepository) Insert(context.Context, domain.Promotion) error {
	return errors.New("not implemented")
}

func (s *stubPromotionRepository) Update(context.Context, domain.Promotion) error {
	return errors.New("not implemented")
}

func (s *stubPromotionRepository) Delete(context.Context, string) error {
	return errors.New("not implemented")
}

func (s *stubPromotionRepository) FindByCode(_ context.Context, code string) (domain.Promotion, error) {
	s.lastCode = code
	if s.err != nil {
		return domain.Promotion{}, s.err
	}
	return s.promotion, nil
}

func (s *stubPromotionRepository) List(context.Context, repositories.PromotionListFilter) (domain.CursorPage[domain.Promotion], error) {
	return domain.CursorPage[domain.Promotion]{}, errors.New("not implemented")
}

type stubPromotionRepoError struct {
	notFound    bool
	conflict    bool
	unavailable bool
}

func (e *stubPromotionRepoError) Error() string {
	return "promotion repo error"
}

func (e *stubPromotionRepoError) IsNotFound() bool    { return e.notFound }
func (e *stubPromotionRepoError) IsConflict() bool    { return e.conflict }
func (e *stubPromotionRepoError) IsUnavailable() bool { return e.unavailable }
