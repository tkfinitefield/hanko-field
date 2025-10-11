package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

const (
	defaultTemplateSort      = domain.TemplateSortPopularity
	defaultTemplateSortOrder = domain.SortDesc
)

// CatalogServiceDeps bundles constructor inputs for the catalog service.
type CatalogServiceDeps struct {
	Catalog repositories.CatalogRepository
	Clock   func() time.Time
}

type catalogService struct {
	repo  repositories.CatalogRepository
	clock func() time.Time
}

// ErrCatalogRepositoryMissing indicates the repository dependency is absent.
var ErrCatalogRepositoryMissing = errors.New("catalog service: repository is not configured")

// NewCatalogService constructs the catalog service with the supplied dependencies.
func NewCatalogService(deps CatalogServiceDeps) (CatalogService, error) {
	if deps.Catalog == nil {
		return nil, fmt.Errorf("catalog service: catalog repository is required")
	}
	clock := deps.Clock
	if clock == nil {
		clock = time.Now
	}
	return &catalogService{
		repo:  deps.Catalog,
		clock: func() time.Time { return clock().UTC() },
	}, nil
}

func (s *catalogService) ListTemplates(ctx context.Context, filter TemplateFilter) (domain.CursorPage[TemplateSummary], error) {
	if s.repo == nil {
		return domain.CursorPage[TemplateSummary]{}, ErrCatalogRepositoryMissing
	}

	repoFilter := repositories.TemplateFilter{
		Category:      normalizeFilterPointer(filter.Category),
		Style:         normalizeFilterPointer(filter.Style),
		Tags:          normalizeTags(filter.Tags),
		OnlyPublished: filter.PublishedOnly,
		Pagination: domain.Pagination{
			PageSize:  filter.Pagination.PageSize,
			PageToken: strings.TrimSpace(filter.Pagination.PageToken),
		},
		SortBy:    normalizeTemplateSort(filter.SortBy),
		SortOrder: normalizeSortOrder(filter.SortOrder),
	}

	page, err := s.repo.ListTemplates(ctx, repoFilter)
	if err != nil {
		return domain.CursorPage[TemplateSummary]{}, err
	}
	return page, nil
}

func (s *catalogService) GetTemplate(ctx context.Context, templateID string) (Template, error) {
	if s.repo == nil {
		return Template{}, ErrCatalogRepositoryMissing
	}
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return Template{}, errors.New("catalog service: template id is required")
	}
	return s.repo.GetTemplate(ctx, templateID)
}

func (s *catalogService) UpsertTemplate(ctx context.Context, cmd UpsertTemplateCommand) (Template, error) {
	if s.repo == nil {
		return Template{}, ErrCatalogRepositoryMissing
	}

	template := cmd.Template
	template.ID = strings.TrimSpace(template.ID)
	template.Name = strings.TrimSpace(template.Name)
	template.Category = strings.TrimSpace(template.Category)
	template.Style = strings.TrimSpace(template.Style)
	template.Description = strings.TrimSpace(template.Description)
	template.Tags = normalizeTags(template.Tags)
	template.PreviewImagePath = strings.TrimSpace(template.PreviewImagePath)
	template.SVGPath = strings.TrimSpace(template.SVGPath)

	now := s.clock()
	if template.CreatedAt.IsZero() {
		template.CreatedAt = now
	} else {
		template.CreatedAt = template.CreatedAt.UTC()
	}
	template.UpdatedAt = now

	return s.repo.UpsertTemplate(ctx, template)
}

func (s *catalogService) DeleteTemplate(ctx context.Context, templateID string) error {
	if s.repo == nil {
		return ErrCatalogRepositoryMissing
	}
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return errors.New("catalog service: template id is required")
	}
	return s.repo.DeleteTemplate(ctx, templateID)
}

func (s *catalogService) ListFonts(ctx context.Context, filter FontFilter) (domain.CursorPage[FontSummary], error) {
	if s.repo == nil {
		return domain.CursorPage[FontSummary]{}, ErrCatalogRepositoryMissing
	}
	repoFilter := repositories.FontFilter{
		Writing:    normalizeFilterPointer(filter.Writing),
		Pagination: domain.Pagination{PageSize: filter.Pagination.PageSize, PageToken: strings.TrimSpace(filter.Pagination.PageToken)},
	}
	return s.repo.ListFonts(ctx, repoFilter)
}

func (s *catalogService) UpsertFont(ctx context.Context, cmd UpsertFontCommand) (FontSummary, error) {
	if s.repo == nil {
		return FontSummary{}, ErrCatalogRepositoryMissing
	}
	return s.repo.UpsertFont(ctx, cmd.Font)
}

func (s *catalogService) DeleteFont(ctx context.Context, fontID string) error {
	if s.repo == nil {
		return ErrCatalogRepositoryMissing
	}
	fontID = strings.TrimSpace(fontID)
	if fontID == "" {
		return errors.New("catalog service: font id is required")
	}
	return s.repo.DeleteFont(ctx, fontID)
}

func (s *catalogService) ListMaterials(ctx context.Context, filter MaterialFilter) (domain.CursorPage[MaterialSummary], error) {
	if s.repo == nil {
		return domain.CursorPage[MaterialSummary]{}, ErrCatalogRepositoryMissing
	}
	repoFilter := repositories.MaterialFilter{
		Texture:    normalizeFilterPointer(filter.Texture),
		IsPublic:   normalizeBoolPointer(filter.IsPublic),
		Pagination: domain.Pagination{PageSize: filter.Pagination.PageSize, PageToken: strings.TrimSpace(filter.Pagination.PageToken)},
	}
	return s.repo.ListMaterials(ctx, repoFilter)
}

func (s *catalogService) UpsertMaterial(ctx context.Context, cmd UpsertMaterialCommand) (MaterialSummary, error) {
	if s.repo == nil {
		return MaterialSummary{}, ErrCatalogRepositoryMissing
	}
	return s.repo.UpsertMaterial(ctx, cmd.Material)
}

func (s *catalogService) DeleteMaterial(ctx context.Context, materialID string) error {
	if s.repo == nil {
		return ErrCatalogRepositoryMissing
	}
	materialID = strings.TrimSpace(materialID)
	if materialID == "" {
		return errors.New("catalog service: material id is required")
	}
	return s.repo.DeleteMaterial(ctx, materialID)
}

func (s *catalogService) ListProducts(ctx context.Context, filter ProductFilter) (domain.CursorPage[ProductSummary], error) {
	if s.repo == nil {
		return domain.CursorPage[ProductSummary]{}, ErrCatalogRepositoryMissing
	}
	repoFilter := repositories.ProductFilter{
		Shape:      normalizeFilterPointer(filter.Shape),
		SizeMm:     filter.SizeMm,
		MaterialID: normalizeFilterPointer(filter.MaterialID),
		Pagination: domain.Pagination{PageSize: filter.Pagination.PageSize, PageToken: strings.TrimSpace(filter.Pagination.PageToken)},
	}
	return s.repo.ListProducts(ctx, repoFilter)
}

func (s *catalogService) UpsertProduct(ctx context.Context, cmd UpsertProductCommand) (ProductSummary, error) {
	if s.repo == nil {
		return ProductSummary{}, ErrCatalogRepositoryMissing
	}
	return s.repo.UpsertProduct(ctx, cmd.Product)
}

func (s *catalogService) DeleteProduct(ctx context.Context, productID string) error {
	if s.repo == nil {
		return ErrCatalogRepositoryMissing
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return errors.New("catalog service: product id is required")
	}
	return s.repo.DeleteProduct(ctx, productID)
}

func normalizeFilterPointer(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(tags))
	var result []string
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func normalizeTemplateSort(sortField domain.TemplateSort) domain.TemplateSort {
	switch sortField {
	case domain.TemplateSortCreatedAt, domain.TemplateSortPopularity:
		return sortField
	default:
		return defaultTemplateSort
	}
}

func normalizeSortOrder(order domain.SortOrder) domain.SortOrder {
	switch order {
	case domain.SortAsc, domain.SortDesc:
		return order
	default:
		return defaultTemplateSortOrder
	}
}

func normalizeBoolPointer(value *bool) *bool {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}
