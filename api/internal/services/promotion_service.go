package services

import (
	"context"
	"strings"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

// PromotionServiceDeps bundles dependencies required to construct a PromotionService implementation.
type PromotionServiceDeps struct {
	Promotions repositories.PromotionRepository
	Clock      func() time.Time
}

type promotionService struct {
	repo  repositories.PromotionRepository
	clock func() time.Time
}

// NewPromotionService wires a PromotionService backed by the provided repositories.
func NewPromotionService(deps PromotionServiceDeps) (PromotionService, error) {
	if deps.Promotions == nil {
		return nil, ErrPromotionRepositoryMissing
	}
	clock := deps.Clock
	if clock == nil {
		clock = time.Now
	}
	return &promotionService{
		repo:  deps.Promotions,
		clock: func() time.Time { return clock().UTC() },
	}, nil
}

func (s *promotionService) GetPublicPromotion(ctx context.Context, code string) (PromotionPublic, error) {
	if s == nil || s.repo == nil {
		return PromotionPublic{}, ErrPromotionRepositoryMissing
	}

	normalized := strings.ToUpper(strings.TrimSpace(code))
	if normalized == "" {
		return PromotionPublic{}, ErrPromotionInvalidCode
	}

	promotion, err := s.repo.FindByCode(ctx, normalized)
	if err != nil {
		if repoErr, ok := err.(repositories.RepositoryError); ok {
			switch {
			case repoErr.IsNotFound():
				return PromotionPublic{}, ErrPromotionNotFound
			case repoErr.IsUnavailable():
				return PromotionPublic{}, ErrPromotionRepositoryMissing
			}
		}
		return PromotionPublic{}, err
	}

	if promotion.InternalOnly || promotion.RequiresAuth {
		return PromotionPublic{}, ErrPromotionUnavailable
	}

	now := s.clock()
	available := normalizePromotionStatus(promotion.Status) == "active"
	if !promotion.StartsAt.IsZero() && now.Before(promotion.StartsAt) {
		available = false
	}
	if !promotion.EndsAt.IsZero() && now.After(promotion.EndsAt) {
		available = false
	}

	result := PromotionPublic{
		Code:              promotion.Code,
		IsAvailable:       available,
		DescriptionPublic: strings.TrimSpace(promotion.DescriptionPublic),
		EligibleAudiences: cloneStringSlice(promotion.EligibleAudiences),
	}
	if !promotion.StartsAt.IsZero() {
		result.StartsAt = promotion.StartsAt.UTC()
	}
	if !promotion.EndsAt.IsZero() {
		result.EndsAt = promotion.EndsAt.UTC()
	}
	return result, nil
}

func (s *promotionService) ValidatePromotion(context.Context, ValidatePromotionCommand) (PromotionValidationResult, error) {
	return PromotionValidationResult{}, ErrPromotionOperationUnsupported
}

func (s *promotionService) ListPromotions(context.Context, PromotionListFilter) (domain.CursorPage[Promotion], error) {
	return domain.CursorPage[Promotion]{}, ErrPromotionOperationUnsupported
}

func (s *promotionService) CreatePromotion(context.Context, UpsertPromotionCommand) (Promotion, error) {
	return Promotion{}, ErrPromotionOperationUnsupported
}

func (s *promotionService) UpdatePromotion(context.Context, UpsertPromotionCommand) (Promotion, error) {
	return Promotion{}, ErrPromotionOperationUnsupported
}

func (s *promotionService) DeletePromotion(context.Context, string) error {
	return ErrPromotionOperationUnsupported
}

func (s *promotionService) ListPromotionUsage(context.Context, PromotionUsageFilter) (domain.CursorPage[PromotionUsage], error) {
	return domain.CursorPage[PromotionUsage]{}, ErrPromotionOperationUnsupported
}

func normalizePromotionStatus(status string) string {
	return strings.ToLower(strings.TrimSpace(status))
}

func cloneStringSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	for i, item := range in {
		out[i] = strings.TrimSpace(item)
	}
	return out
}
