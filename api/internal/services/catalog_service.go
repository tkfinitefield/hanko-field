package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

const (
	defaultTemplateSort      = domain.TemplateSortPopularity
	defaultTemplateSortOrder = domain.SortDesc
)

var fontSlugSanitizer = regexp.MustCompile(`[^a-z0-9]+`)

// TemplateCacheInvalidator purges CDN/cache entries referencing template assets.
type TemplateCacheInvalidator interface {
	InvalidateTemplates(ctx context.Context, templateIDs []string) error
}

// CatalogServiceDeps bundles constructor inputs for the catalog service.
type CatalogServiceDeps struct {
	Catalog repositories.CatalogRepository
	Audit   AuditLogService
	Cache   TemplateCacheInvalidator
	Clock   func() time.Time
}

type catalogService struct {
	repo  repositories.CatalogRepository
	audit AuditLogService
	cache TemplateCacheInvalidator
	clock func() time.Time
}

var (
	// ErrCatalogRepositoryMissing indicates the repository dependency is absent.
	ErrCatalogRepositoryMissing = errors.New("catalog service: repository is not configured")
	// ErrCatalogInvalidInput indicates the caller supplied invalid data to a catalog mutation.
	ErrCatalogInvalidInput = errors.New("catalog service: invalid input")
	// ErrCatalogFontConflict indicates a slug/family+weight combination already exists.
	ErrCatalogFontConflict = errors.New("catalog service: font conflict")
	// ErrCatalogFontInUse indicates the font cannot be deleted due to active dependencies.
	ErrCatalogFontInUse = errors.New("catalog service: font in use")
)

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
		audit: deps.Audit,
		cache: deps.Cache,
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
	template, err := s.repo.GetPublishedTemplate(ctx, templateID)
	if err != nil {
		return Template{}, err
	}
	return Template(template), nil
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
	actorID := strings.TrimSpace(cmd.ActorID)

	now := s.clock()

	var existing Template
	if template.ID != "" {
		current, err := s.repo.GetTemplate(ctx, template.ID)
		if err != nil && !isCatalogRepositoryNotFound(err) {
			return Template{}, err
		}
		if err == nil {
			existing = Template(current)
		}
	}

	if template.CreatedAt.IsZero() {
		if existing.CreatedAt.IsZero() {
			template.CreatedAt = now
		} else {
			template.CreatedAt = existing.CreatedAt
		}
	} else {
		template.CreatedAt = template.CreatedAt.UTC()
	}
	template.UpdatedAt = now
	template.Version = nextTemplateVersion(existing.Version)
	template.PublishedAt = resolveTemplatePublishedAt(template, existing, now)
	template.Draft = normalizeTemplateDraft(template.Draft, now, actorID)

	savedDomain, err := s.repo.UpsertTemplate(ctx, template)
	if err != nil {
		return Template{}, err
	}
	saved := Template(savedDomain)

	if err := s.appendTemplateVersion(ctx, saved, actorID, now); err != nil {
		return Template{}, err
	}

	if existing.IsPublished != saved.IsPublished {
		s.invalidateTemplates(ctx, []string{saved.ID})
	}

	action := "catalog.template.update"
	occurredAt := saved.UpdatedAt
	if existing.ID == "" {
		action = "catalog.template.create"
		occurredAt = saved.CreatedAt
	}
	s.recordTemplateAudit(ctx, action, saved, actorID, occurredAt, nil, nil)

	if existing.IsPublished != saved.IsPublished {
		diff := map[string]AuditLogDiff{
			"isPublished": {
				Before: existing.IsPublished,
				After:  saved.IsPublished,
			},
		}
		extra := map[string]any{"publishedAt": saved.PublishedAt}
		if saved.IsPublished {
			s.recordTemplateAudit(ctx, "catalog.template.publish", saved, actorID, saved.PublishedAt, diff, extra)
		} else {
			s.recordTemplateAudit(ctx, "catalog.template.unpublish", saved, actorID, occurredAt, diff, extra)
		}
	}

	return saved, nil
}

func (s *catalogService) DeleteTemplate(ctx context.Context, cmd DeleteTemplateCommand) error {
	if s.repo == nil {
		return ErrCatalogRepositoryMissing
	}
	templateID := strings.TrimSpace(cmd.TemplateID)
	if templateID == "" {
		return errors.New("catalog service: template id is required")
	}

	var existing Template
	current, err := s.repo.GetTemplate(ctx, templateID)
	if err != nil && !isCatalogRepositoryNotFound(err) {
		return err
	}
	if err == nil {
		existing = Template(current)
	}

	if err := s.repo.DeleteTemplate(ctx, templateID); err != nil {
		var repoErr repositories.RepositoryError
		if errors.As(err, &repoErr) && repoErr.IsNotFound() {
			return nil
		}
		return err
	}

	occurredAt := s.clock()
	if existing.ID != "" {
		if existing.IsPublished {
			s.invalidateTemplates(ctx, []string{existing.ID})
		}
		diff := map[string]AuditLogDiff{
			"isPublished": {Before: existing.IsPublished, After: false},
		}
		extra := map[string]any{"deleted": true}
		s.recordTemplateAudit(ctx, "catalog.template.delete", existing, strings.TrimSpace(cmd.ActorID), occurredAt, diff, extra)
	}

	return nil
}

func (s *catalogService) ListFonts(ctx context.Context, filter FontFilter) (domain.CursorPage[FontSummary], error) {
	if s.repo == nil {
		return domain.CursorPage[FontSummary]{}, ErrCatalogRepositoryMissing
	}
	repoFilter := repositories.FontFilter{
		Script:        normalizeFilterPointer(filter.Script),
		IsPremium:     normalizeBoolPointer(filter.IsPremium),
		PublishedOnly: filter.PublishedOnly,
		Pagination: domain.Pagination{
			PageSize:  filter.Pagination.PageSize,
			PageToken: strings.TrimSpace(filter.Pagination.PageToken),
		},
	}
	return s.repo.ListFonts(ctx, repoFilter)
}

func (s *catalogService) GetFont(ctx context.Context, fontID string) (Font, error) {
	if s.repo == nil {
		return Font{}, ErrCatalogRepositoryMissing
	}
	font, err := s.repo.GetPublishedFont(ctx, strings.TrimSpace(fontID))
	if err != nil {
		return Font{}, err
	}
	return Font(font), nil
}

func (s *catalogService) UpsertFont(ctx context.Context, cmd UpsertFontCommand) (FontSummary, error) {
	if s.repo == nil {
		return FontSummary{}, ErrCatalogRepositoryMissing
	}

	font := normalizeFontSummary(cmd.Font)
	originalID := strings.TrimSpace(cmd.Font.ID)
	expectedSlug := generateFontSlug(font.Family, font.Weight)
	if expectedSlug == "" {
		return FontSummary{}, fmt.Errorf("%w: family and weight are required", ErrCatalogInvalidInput)
	}
	id := strings.TrimSpace(cmd.Font.ID)
	if id == "" {
		id = expectedSlug
	} else if !strings.EqualFold(id, expectedSlug) {
		return FontSummary{}, fmt.Errorf("%w: font id %s must match derived slug %s", ErrCatalogInvalidInput, id, expectedSlug)
	}
	font.Slug = expectedSlug
	font.ID = id
	font.SupportedWeights = ensureSupportedWeight(font.SupportedWeights, font.Weight)

	if err := validateFontSummary(font); err != nil {
		return FontSummary{}, err
	}

	var existing domain.Font
	if font.ID != "" {
		current, err := s.repo.GetFont(ctx, font.ID)
		if err != nil && !isCatalogRepositoryNotFound(err) {
			return FontSummary{}, err
		}
		existing = current
	}
	isCreate := strings.TrimSpace(originalID) == ""
	if existing.ID != "" && isCreate {
		return FontSummary{}, fmt.Errorf("%w: font %s already exists", ErrCatalogFontConflict, font.ID)
	}
	now := s.clock()
	if existing.ID != "" && !existing.CreatedAt.IsZero() {
		font.CreatedAt = existing.CreatedAt
	} else if font.CreatedAt.IsZero() {
		font.CreatedAt = now
	} else {
		font.CreatedAt = font.CreatedAt.UTC()
	}
	font.UpdatedAt = now

	saved, err := s.repo.UpsertFont(ctx, domain.FontSummary(font))
	if err != nil {
		var repoErr repositories.RepositoryError
		if errors.As(err, &repoErr) && repoErr.IsConflict() {
			return FontSummary{}, fmt.Errorf("%w: %s", ErrCatalogFontConflict, err.Error())
		}
		return FontSummary{}, err
	}
	return FontSummary(saved), nil
}

func (s *catalogService) DeleteFont(ctx context.Context, fontID string) error {
	if s.repo == nil {
		return ErrCatalogRepositoryMissing
	}
	fontID = strings.TrimSpace(fontID)
	if fontID == "" {
		return fmt.Errorf("%w: font id is required", ErrCatalogInvalidInput)
	}
	if err := s.repo.DeleteFont(ctx, fontID); err != nil {
		var repoErr repositories.RepositoryError
		if errors.As(err, &repoErr) && repoErr.IsConflict() {
			return fmt.Errorf("%w: font in use", ErrCatalogFontInUse)
		}
		return err
	}
	return nil
}

func (s *catalogService) ListMaterials(ctx context.Context, filter MaterialFilter) (domain.CursorPage[MaterialSummary], error) {
	if s.repo == nil {
		return domain.CursorPage[MaterialSummary]{}, ErrCatalogRepositoryMissing
	}
	var isAvailable *bool
	if filter.OnlyAvailable {
		trueVal := true
		isAvailable = &trueVal
	}
	repoFilter := repositories.MaterialFilter{
		Category:    normalizeFilterPointer(filter.Category),
		IsAvailable: isAvailable,
		Locale:      strings.TrimSpace(filter.Locale),
		Pagination:  domain.Pagination{PageSize: filter.Pagination.PageSize, PageToken: strings.TrimSpace(filter.Pagination.PageToken)},
	}
	return s.repo.ListMaterials(ctx, repoFilter)
}

func (s *catalogService) GetMaterial(ctx context.Context, materialID string) (Material, error) {
	if s.repo == nil {
		return Material{}, ErrCatalogRepositoryMissing
	}
	materialID = strings.TrimSpace(materialID)
	if materialID == "" {
		return Material{}, errors.New("catalog service: material id is required")
	}
	material, err := s.repo.GetPublishedMaterial(ctx, materialID)
	if err != nil {
		return Material{}, err
	}
	return Material(material), nil
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

func (s *catalogService) appendTemplateVersion(ctx context.Context, template Template, actorID string, now time.Time) error {
	if s.repo == nil {
		return ErrCatalogRepositoryMissing
	}
	version := domain.TemplateVersion{
		TemplateID: template.ID,
		Version:    template.Version,
		Snapshot:   domain.Template(template),
		Draft:      domain.TemplateDraft(template.Draft),
		CreatedAt:  now,
		CreatedBy:  strings.TrimSpace(actorID),
	}
	return s.repo.AppendTemplateVersion(ctx, version)
}

func (s *catalogService) invalidateTemplates(ctx context.Context, templateIDs []string) {
	if s.cache == nil || len(templateIDs) == 0 {
		return
	}
	_ = s.cache.InvalidateTemplates(ctx, templateIDs)
}

func (s *catalogService) recordTemplateAudit(ctx context.Context, action string, template Template, actorID string, occurredAt time.Time, diff map[string]AuditLogDiff, extra map[string]any) {
	if s.audit == nil {
		return
	}
	actorID = strings.TrimSpace(actorID)
	if occurredAt.IsZero() {
		occurredAt = s.clock()
	}
	metadata := map[string]any{
		"templateId":  template.ID,
		"name":        template.Name,
		"version":     template.Version,
		"isPublished": template.IsPublished,
	}
	for k, v := range extra {
		metadata[k] = v
	}
	record := AuditLogRecord{
		Actor:      actorID,
		ActorType:  "staff",
		Action:     action,
		TargetRef:  fmt.Sprintf("/templates/%s", strings.TrimSpace(template.ID)),
		Severity:   "info",
		OccurredAt: occurredAt,
		Metadata:   metadata,
		Diff:       diff,
	}
	s.audit.Record(ctx, record)
}

func resolveTemplatePublishedAt(next Template, existing Template, now time.Time) time.Time {
	if !next.IsPublished {
		return time.Time{}
	}
	if !next.PublishedAt.IsZero() {
		return next.PublishedAt.UTC()
	}
	if existing.IsPublished && !existing.PublishedAt.IsZero() {
		return existing.PublishedAt
	}
	return now
}

func nextTemplateVersion(current int) int {
	if current < 0 {
		current = 0
	}
	return current + 1
}

func normalizeTemplateDraft(d TemplateDraft, now time.Time, actorID string) TemplateDraft {
	d.Notes = strings.TrimSpace(d.Notes)
	d.PreviewImagePath = strings.TrimSpace(d.PreviewImagePath)
	d.PreviewSVGPath = strings.TrimSpace(d.PreviewSVGPath)
	if len(d.Metadata) == 0 {
		d.Metadata = nil
	}
	if templateDraftIsEmpty(d) {
		return TemplateDraft{}
	}
	if d.UpdatedAt.IsZero() {
		d.UpdatedAt = now
	} else {
		d.UpdatedAt = d.UpdatedAt.UTC()
	}
	actorID = strings.TrimSpace(actorID)
	if strings.TrimSpace(d.UpdatedBy) == "" {
		d.UpdatedBy = actorID
	} else {
		d.UpdatedBy = strings.TrimSpace(d.UpdatedBy)
	}
	return d
}

func templateDraftIsEmpty(d TemplateDraft) bool {
	if d.Notes != "" || d.PreviewImagePath != "" || d.PreviewSVGPath != "" {
		return false
	}
	return len(d.Metadata) == 0
}

func isCatalogRepositoryNotFound(err error) bool {
	if err == nil {
		return false
	}
	var repoErr repositories.RepositoryError
	if errors.As(err, &repoErr) {
		return repoErr.IsNotFound()
	}
	return false
}

func (s *catalogService) ListProducts(ctx context.Context, filter ProductFilter) (domain.CursorPage[ProductSummary], error) {
	if s.repo == nil {
		return domain.CursorPage[ProductSummary]{}, ErrCatalogRepositoryMissing
	}
	repoFilter := repositories.ProductFilter{
		Shape:          normalizeFilterPointer(filter.Shape),
		SizeMm:         filter.SizeMm,
		MaterialID:     normalizeFilterPointer(filter.MaterialID),
		IsCustomizable: normalizeBoolPointer(filter.IsCustomizable),
		OnlyPublished:  filter.PublishedOnly,
		Pagination:     domain.Pagination{PageSize: filter.Pagination.PageSize, PageToken: strings.TrimSpace(filter.Pagination.PageToken)},
	}
	return s.repo.ListProducts(ctx, repoFilter)
}

func (s *catalogService) GetProduct(ctx context.Context, productID string) (Product, error) {
	if s.repo == nil {
		return Product{}, ErrCatalogRepositoryMissing
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return Product{}, errors.New("catalog service: product id is required")
	}
	product, err := s.repo.GetPublishedProduct(ctx, productID)
	if err != nil {
		return Product{}, err
	}
	return Product(product), nil
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

func normalizeFontSummary(font FontSummary) FontSummary {
	font.ID = strings.TrimSpace(font.ID)
	font.Slug = strings.TrimSpace(font.Slug)
	font.DisplayName = strings.TrimSpace(font.DisplayName)
	font.Family = strings.TrimSpace(font.Family)
	font.Weight = strings.ToLower(strings.TrimSpace(font.Weight))
	font.PreviewImagePath = strings.TrimSpace(font.PreviewImagePath)
	font.Scripts = normalizeTags(font.Scripts)
	font.SupportedWeights = normalizeTags(font.SupportedWeights)
	font.License.Name = strings.TrimSpace(font.License.Name)
	font.License.URL = strings.TrimSpace(font.License.URL)
	font.License.AllowedUsages = normalizeTags(font.License.AllowedUsages)
	return font
}

func ensureSupportedWeight(weights []string, weight string) []string {
	weight = strings.TrimSpace(weight)
	if weight == "" {
		return weights
	}
	if len(weights) == 0 {
		return []string{weight}
	}
	weightLower := strings.ToLower(weight)
	for _, w := range weights {
		if strings.ToLower(strings.TrimSpace(w)) == weightLower {
			return weights
		}
	}
	return append(weights, weight)
}

func validateFontSummary(font FontSummary) error {
	if strings.TrimSpace(font.DisplayName) == "" {
		return fmt.Errorf("%w: display_name is required", ErrCatalogInvalidInput)
	}
	if strings.TrimSpace(font.Family) == "" {
		return fmt.Errorf("%w: family is required", ErrCatalogInvalidInput)
	}
	if strings.TrimSpace(font.Weight) == "" {
		return fmt.Errorf("%w: weight is required", ErrCatalogInvalidInput)
	}
	if len(font.Scripts) == 0 {
		return fmt.Errorf("%w: at least one script is required", ErrCatalogInvalidInput)
	}
	if strings.TrimSpace(font.PreviewImagePath) == "" {
		return fmt.Errorf("%w: preview_image_path is required", ErrCatalogInvalidInput)
	}
	if err := validateFontLicense(font.License); err != nil {
		return err
	}
	return nil
}

func validateFontLicense(license FontLicense) error {
	if strings.TrimSpace(license.Name) == "" {
		return fmt.Errorf("%w: license.name is required", ErrCatalogInvalidInput)
	}
	if strings.TrimSpace(license.URL) == "" {
		return fmt.Errorf("%w: license.url is required", ErrCatalogInvalidInput)
	}
	parsed, err := url.Parse(license.URL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%w: license.url must be absolute", ErrCatalogInvalidInput)
	}
	if len(license.AllowedUsages) == 0 {
		return fmt.Errorf("%w: license.allowed_usages is required", ErrCatalogInvalidInput)
	}
	return nil
}

func generateFontSlug(family, weight string) string {
	normalize := func(value string) string {
		value = strings.ToLower(strings.TrimSpace(value))
		value = fontSlugSanitizer.ReplaceAllString(value, "-")
		return strings.Trim(value, "-")
	}
	familySlug := normalize(family)
	weightSlug := normalize(weight)
	if familySlug == "" || weightSlug == "" {
		return ""
	}
	return fmt.Sprintf("%s-%s", familySlug, weightSlug)
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
