package services

import (
	"context"
	"errors"
	"strings"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

const defaultContentLocale = "ja"

// ContentServiceDeps groups constructor parameters for the content service.
type ContentServiceDeps struct {
	Repository    repositories.ContentRepository
	Clock         func() time.Time
	DefaultLocale string
}

type contentService struct {
	repo          repositories.ContentRepository
	clock         func() time.Time
	defaultLocale string
}

// ErrContentRepositoryMissing signals that the content repository dependency is absent.
var ErrContentRepositoryMissing = errors.New("content service: content repository is not configured")

// NewContentService constructs the content service with the supplied dependencies.
func NewContentService(deps ContentServiceDeps) (ContentService, error) {
	if deps.Repository == nil {
		return nil, ErrContentRepositoryMissing
	}
	clock := deps.Clock
	if clock == nil {
		clock = time.Now
	}
	defaultLocale := strings.TrimSpace(deps.DefaultLocale)
	if defaultLocale == "" {
		defaultLocale = defaultContentLocale
	}
	return &contentService{
		repo:          deps.Repository,
		clock:         func() time.Time { return clock().UTC() },
		defaultLocale: normalizeLocaleValue(defaultLocale),
	}, nil
}

func (s *contentService) ListGuides(ctx context.Context, filter ContentGuideFilter) (domain.CursorPage[ContentGuide], error) {
	if s.repo == nil {
		return domain.CursorPage[ContentGuide]{}, ErrContentRepositoryMissing
	}

	requestedLocale := normalizeLocalePointer(filter.Locale)
	if requestedLocale == "" {
		requestedLocale = s.defaultLocale
	}

	fallback := normalizeLocaleValue(filter.FallbackLocale)
	if fallback == "" {
		fallback = s.defaultLocale
	}

	repoFilter := repositories.ContentGuideFilter{
		Category:       normalizeFilterPointer(filter.Category),
		Slug:           normalizeFilterPointer(filter.Slug),
		Locale:         pointerIfNotEmpty(requestedLocale),
		FallbackLocale: fallback,
		Status:         normalizeStatusSlice(filter.Status),
		OnlyPublished:  filter.PublishedOnly,
		Pagination: domain.Pagination{
			PageSize:  filter.Pagination.PageSize,
			PageToken: strings.TrimSpace(filter.Pagination.PageToken),
		},
	}

	page, err := s.repo.ListGuides(ctx, repoFilter)
	if err != nil {
		return domain.CursorPage[ContentGuide]{}, err
	}

	result := domain.CursorPage[ContentGuide]{
		Items:         make([]ContentGuide, 0, len(page.Items)),
		NextPageToken: page.NextPageToken,
	}

	for _, guide := range page.Items {
		normalized := normalizeContentGuide(guide, requestedLocale, fallback, s.defaultLocale)
		result.Items = append(result.Items, ContentGuide(normalized))
	}

	return result, nil
}

func (s *contentService) GetGuideBySlug(ctx context.Context, slug string, locale string) (ContentGuide, error) {
	if s.repo == nil {
		return ContentGuide{}, ErrContentRepositoryMissing
	}

	slug = strings.TrimSpace(slug)
	if slug == "" {
		return ContentGuide{}, errors.New("content service: slug is required")
	}

	requested := normalizeLocaleValue(locale)
	if requested == "" {
		requested = s.defaultLocale
	}

	guide, err := s.repo.GetGuideBySlug(ctx, slug, requested)
	if err != nil && requested != s.defaultLocale && isRepositoryNotFound(err) {
		guide, err = s.repo.GetGuideBySlug(ctx, slug, s.defaultLocale)
	}
	if err != nil {
		return ContentGuide{}, err
	}

	return ContentGuide(normalizeContentGuide(guide, requested, s.defaultLocale, s.defaultLocale)), nil
}

func (s *contentService) GetGuide(ctx context.Context, guideID string) (ContentGuide, error) {
	if s.repo == nil {
		return ContentGuide{}, ErrContentRepositoryMissing
	}
	guideID = strings.TrimSpace(guideID)
	if guideID == "" {
		return ContentGuide{}, errors.New("content service: guide id is required")
	}
	guide, err := s.repo.GetGuide(ctx, guideID)
	if err != nil {
		return ContentGuide{}, err
	}
	return ContentGuide(normalizeContentGuide(guide, "", "", s.defaultLocale)), nil
}

func (s *contentService) UpsertGuide(ctx context.Context, cmd UpsertContentGuideCommand) (ContentGuide, error) {
	if s.repo == nil {
		return ContentGuide{}, ErrContentRepositoryMissing
	}

	guide := cmd.Guide
	guide.ID = strings.TrimSpace(guide.ID)
	guide.Slug = strings.TrimSpace(guide.Slug)
	guide.Locale = normalizeLocaleValue(guide.Locale)
	guide.Category = strings.TrimSpace(guide.Category)
	guide.Title = strings.TrimSpace(guide.Title)
	guide.Summary = strings.TrimSpace(guide.Summary)
	guide.HeroImage = strings.TrimSpace(guide.HeroImage)
	guide.Tags = normalizeStringSlice(guide.Tags)
	guide.Status = strings.TrimSpace(guide.Status)
	if guide.Locale == "" {
		guide.Locale = s.defaultLocale
	}

	now := s.clock()
	if guide.CreatedAt.IsZero() {
		guide.CreatedAt = now
	} else {
		guide.CreatedAt = guide.CreatedAt.UTC()
	}
	guide.UpdatedAt = now
	if !guide.PublishedAt.IsZero() {
		guide.PublishedAt = guide.PublishedAt.UTC()
	}

	saved, err := s.repo.UpsertGuide(ctx, domain.ContentGuide(guide))
	if err != nil {
		return ContentGuide{}, err
	}
	return ContentGuide(normalizeContentGuide(saved, guide.Locale, "", s.defaultLocale)), nil
}

func (s *contentService) DeleteGuide(ctx context.Context, guideID string) error {
	if s.repo == nil {
		return ErrContentRepositoryMissing
	}
	guideID = strings.TrimSpace(guideID)
	if guideID == "" {
		return errors.New("content service: guide id is required")
	}
	return s.repo.DeleteGuide(ctx, guideID)
}

func (s *contentService) GetPage(ctx context.Context, slug string, locale string) (ContentPage, error) {
	if s.repo == nil {
		return ContentPage{}, ErrContentRepositoryMissing
	}
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return ContentPage{}, errors.New("content service: slug is required")
	}
	locale = normalizeLocaleValue(locale)
	if locale == "" {
		locale = s.defaultLocale
	}
	page, err := s.repo.GetPage(ctx, slug, locale)
	if err != nil && locale != s.defaultLocale && isRepositoryNotFound(err) {
		page, err = s.repo.GetPage(ctx, slug, s.defaultLocale)
	}
	if err != nil {
		return ContentPage{}, err
	}
	return ContentPage(page), nil
}

func (s *contentService) UpsertPage(ctx context.Context, cmd UpsertContentPageCommand) (ContentPage, error) {
	if s.repo == nil {
		return ContentPage{}, ErrContentRepositoryMissing
	}
	page := cmd.Page
	page.ID = strings.TrimSpace(page.ID)
	page.Slug = strings.TrimSpace(page.Slug)
	page.Locale = normalizeLocaleValue(page.Locale)
	if page.Locale == "" {
		page.Locale = s.defaultLocale
	}
	page.Title = strings.TrimSpace(page.Title)
	page.Status = strings.TrimSpace(page.Status)
	updated := s.clock().UTC()
	page.UpdatedAt = updated

	saved, err := s.repo.UpsertPage(ctx, domain.ContentPage(page))
	if err != nil {
		return ContentPage{}, err
	}
	return ContentPage(saved), nil
}

func normalizeContentGuide(guide domain.ContentGuide, requestedLocale, fallbackLocale, defaultLocale string) domain.ContentGuide {
	guide.Slug = strings.TrimSpace(guide.Slug)
	guide.Locale = normalizeLocaleValue(guide.Locale)
	requestedLocale = normalizeLocaleValue(requestedLocale)
	fallbackLocale = normalizeLocaleValue(fallbackLocale)
	defaultLocale = normalizeLocaleValue(defaultLocale)

	if guide.Locale == "" {
		switch {
		case requestedLocale != "":
			guide.Locale = requestedLocale
		case fallbackLocale != "":
			guide.Locale = fallbackLocale
		default:
			guide.Locale = defaultLocale
		}
	}

	guide.Category = strings.TrimSpace(guide.Category)
	guide.Title = strings.TrimSpace(guide.Title)
	guide.Summary = strings.TrimSpace(guide.Summary)
	guide.BodyHTML = strings.TrimSpace(guide.BodyHTML)
	guide.HeroImage = strings.TrimSpace(guide.HeroImage)
	guide.Tags = normalizeStringSlice(guide.Tags)
	guide.Status = strings.TrimSpace(guide.Status)
	if !guide.IsPublished && guide.Status != "" {
		guide.IsPublished = strings.EqualFold(guide.Status, "published")
	}

	if !guide.PublishedAt.IsZero() {
		guide.PublishedAt = guide.PublishedAt.UTC()
	}
	if !guide.CreatedAt.IsZero() {
		guide.CreatedAt = guide.CreatedAt.UTC()
	}
	if !guide.UpdatedAt.IsZero() {
		guide.UpdatedAt = guide.UpdatedAt.UTC()
	}
	return guide
}

func normalizeLocalePointer(value *string) string {
	if value == nil {
		return ""
	}
	return normalizeLocaleValue(*value)
}

func pointerIfNotEmpty(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizeStatusSlice(statuses []string) []string {
	if len(statuses) == 0 {
		return nil
	}
	result := make([]string, 0, len(statuses))
	seen := make(map[string]struct{}, len(statuses))
	for _, status := range statuses {
		trimmed := strings.TrimSpace(status)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		result = append(result, lower)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func normalizeStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func normalizeLocaleValue(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	normalized := strings.ReplaceAll(trimmed, "_", "-")
	return strings.ToLower(normalized)
}

func isRepositoryNotFound(err error) bool {
	if err == nil {
		return false
	}
	var repoErr repositories.RepositoryError
	if errors.As(err, &repoErr) {
		return repoErr.IsNotFound()
	}
	return false
}
