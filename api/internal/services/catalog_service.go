package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

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
	Catalog   repositories.CatalogRepository
	Audit     AuditLogService
	Cache     TemplateCacheInvalidator
	Inventory InventoryService
	Clock     func() time.Time
}

type catalogService struct {
	repo      repositories.CatalogRepository
	audit     AuditLogService
	cache     TemplateCacheInvalidator
	inventory InventoryService
	clock     func() time.Time
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
	// ErrCatalogMaterialConflict indicates a duplicate material ID creation attempt.
	ErrCatalogMaterialConflict = errors.New("catalog service: material conflict")
	// ErrCatalogProductConflict indicates SKU collisions or other product constraints.
	ErrCatalogProductConflict = errors.New("catalog service: product conflict")
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
		repo:      deps.Catalog,
		audit:     deps.Audit,
		cache:     deps.Cache,
		inventory: deps.Inventory,
		clock:     func() time.Time { return clock().UTC() },
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
	material := normalizeMaterialSummary(cmd.Material)
	actorID := strings.TrimSpace(cmd.ActorID)
	if err := validateMaterialInput(material); err != nil {
		return MaterialSummary{}, fmt.Errorf("%w: %s", ErrCatalogInvalidInput, err)
	}
	if material.ID == "" {
		return MaterialSummary{}, fmt.Errorf("%w: material id is required", ErrCatalogInvalidInput)
	}
	now := s.clock()
	var existing Material
	current, err := s.repo.GetMaterial(ctx, material.ID)
	if err != nil && !isCatalogRepositoryNotFound(err) {
		return MaterialSummary{}, err
	}
	if err == nil {
		existing = Material(current)
	}
	var saved MaterialSummary
	if existing.ID == "" {
		material.CreatedAt = now
		material.UpdatedAt = now
		saved, err = s.createMaterial(ctx, material)
		if err != nil {
			return MaterialSummary{}, err
		}
	} else {
		if material.CreatedAt.IsZero() {
			material.CreatedAt = existing.CreatedAt
		} else {
			material.CreatedAt = material.CreatedAt.UTC()
		}
		material.UpdatedAt = now
		saved, err = s.updateMaterial(ctx, material)
		if err != nil {
			return MaterialSummary{}, err
		}
	}
	if err := s.syncMaterialSafetyStock(ctx, saved); err != nil {
		return MaterialSummary{}, err
	}
	s.recordMaterialMutations(ctx, existing, saved, actorID)
	return saved, nil
}

func (s *catalogService) DeleteMaterial(ctx context.Context, cmd DeleteMaterialCommand) error {
	if s.repo == nil {
		return ErrCatalogRepositoryMissing
	}
	materialID := strings.TrimSpace(cmd.MaterialID)
	if materialID == "" {
		return errors.New("catalog service: material id is required")
	}
	var existing Material
	current, err := s.repo.GetMaterial(ctx, materialID)
	if err != nil && !isCatalogRepositoryNotFound(err) {
		return err
	}
	if err == nil {
		existing = Material(current)
	}
	if err := s.repo.DeleteMaterial(ctx, materialID); err != nil {
		var repoErr repositories.RepositoryError
		if errors.As(err, &repoErr) && repoErr.IsNotFound() {
			return nil
		}
		return err
	}
	if existing.ID != "" {
		diff := map[string]AuditLogDiff{
			"isAvailable": {Before: existing.IsAvailable, After: false},
		}
		s.recordMaterialAudit(ctx, "catalog.material.delete", existing.MaterialSummary, strings.TrimSpace(cmd.ActorID), s.clock(), diff, map[string]any{"deleted": true})
		s.flagMaterialProducts(ctx, materialID, cmd.ActorID, "material_deleted")
	}
	return nil
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

func (s *catalogService) syncMaterialSafetyStock(ctx context.Context, material MaterialSummary) error {
	if s.inventory == nil {
		return nil
	}
	sku := strings.TrimSpace(material.Inventory.SKU)
	if sku == "" {
		return nil
	}
	_, err := s.inventory.ConfigureSafetyStock(ctx, ConfigureSafetyStockCommand{
		SKU:         sku,
		ProductRef:  materialTargetRef(material.ID),
		SafetyStock: material.Inventory.SafetyStock,
	})
	return err
}

func (s *catalogService) createMaterial(ctx context.Context, material MaterialSummary) (MaterialSummary, error) {
	savedDomain, err := s.repo.CreateMaterial(ctx, domain.MaterialSummary(material))
	if err != nil {
		var repoErr repositories.RepositoryError
		if errors.As(err, &repoErr) && repoErr.IsConflict() {
			return MaterialSummary{}, ErrCatalogMaterialConflict
		}
		return MaterialSummary{}, err
	}
	return MaterialSummary(savedDomain), nil
}

func (s *catalogService) updateMaterial(ctx context.Context, material MaterialSummary) (MaterialSummary, error) {
	savedDomain, err := s.repo.UpsertMaterial(ctx, domain.MaterialSummary(material))
	if err != nil {
		return MaterialSummary{}, err
	}
	return MaterialSummary(savedDomain), nil
}

func (s *catalogService) recordMaterialMutations(ctx context.Context, existing Material, saved MaterialSummary, actorID string) {
	action := "catalog.material.update"
	occurredAt := saved.UpdatedAt
	if existing.ID == "" {
		action = "catalog.material.create"
		occurredAt = saved.CreatedAt
	}
	diff := map[string]AuditLogDiff{}
	if existing.ID != "" {
		if existing.IsAvailable != saved.IsAvailable {
			diff["isAvailable"] = AuditLogDiff{Before: existing.IsAvailable, After: saved.IsAvailable}
		}
		if strings.TrimSpace(existing.Inventory.SKU) != strings.TrimSpace(saved.Inventory.SKU) {
			diff["inventory.sku"] = AuditLogDiff{Before: strings.TrimSpace(existing.Inventory.SKU), After: strings.TrimSpace(saved.Inventory.SKU)}
		}
		if existing.Inventory.SafetyStock != saved.Inventory.SafetyStock {
			diff["inventory.safetyStock"] = AuditLogDiff{Before: existing.Inventory.SafetyStock, After: saved.Inventory.SafetyStock}
		}
	}
	if len(diff) == 0 {
		diff = nil
	}
	s.recordMaterialAudit(ctx, action, saved, actorID, occurredAt, diff, nil)
	if existing.ID == "" && saved.IsAvailable {
		s.recordMaterialAudit(ctx, "catalog.material.publish", saved, actorID, occurredAt, nil, nil)
	}
	if existing.ID != "" && existing.IsAvailable != saved.IsAvailable {
		stateAction := "catalog.material.publish"
		if saved.IsAvailable {
			s.recordMaterialAudit(ctx, stateAction, saved, actorID, occurredAt, nil, nil)
		} else {
			s.recordMaterialAudit(ctx, "catalog.material.unpublish", saved, actorID, occurredAt, nil, nil)
			s.flagMaterialProducts(ctx, saved.ID, actorID, "material_unavailable")
		}
	}
}

func (s *catalogService) flagMaterialProducts(ctx context.Context, materialID string, actorID string, reason string) {
	if s.audit == nil || s.repo == nil {
		return
	}
	materialID = strings.TrimSpace(materialID)
	if materialID == "" {
		return
	}
	filterID := materialID
	pageToken := ""
	total := 0
	sampled := make([]string, 0, 50)
	for {
		page, err := s.repo.ListProducts(ctx, repositories.ProductFilter{
			MaterialID: &filterID,
			Pagination: domain.Pagination{PageSize: 100, PageToken: pageToken},
		})
		if err != nil {
			return
		}
		if len(page.Items) == 0 {
			break
		}
		for _, product := range page.Items {
			total++
			if len(sampled) < 50 {
				sampled = append(sampled, product.ID)
			}
		}
		if strings.TrimSpace(page.NextPageToken) == "" {
			break
		}
		pageToken = page.NextPageToken
	}
	if total == 0 {
		return
	}
	metadata := map[string]any{
		"materialId":      materialID,
		"products":        sampled,
		"totalProducts":   total,
		"productsSampled": len(sampled),
	}
	if total == len(sampled) {
		delete(metadata, "productsSampled")
	}
	if strings.TrimSpace(reason) != "" {
		metadata["reason"] = strings.TrimSpace(reason)
	}
	s.audit.Record(ctx, AuditLogRecord{
		Actor:      strings.TrimSpace(actorID),
		ActorType:  "staff",
		Action:     "catalog.material.products.flagged",
		TargetRef:  materialTargetRef(materialID),
		Severity:   "warning",
		OccurredAt: s.clock(),
		Metadata:   metadata,
	})
}

func (s *catalogService) recordMaterialAudit(ctx context.Context, action string, material MaterialSummary, actorID string, occurredAt time.Time, diff map[string]AuditLogDiff, metadata map[string]any) {
	if s.audit == nil {
		return
	}
	actorID = strings.TrimSpace(actorID)
	if occurredAt.IsZero() {
		occurredAt = s.clock()
	}
	meta := map[string]any{
		"materialId":   material.ID,
		"category":     material.Category,
		"inventorySku": strings.TrimSpace(material.Inventory.SKU),
		"supplierRef":  strings.TrimSpace(material.Procurement.SupplierRef),
	}
	for k, v := range metadata {
		if v != nil {
			meta[k] = v
		}
	}
	s.audit.Record(ctx, AuditLogRecord{
		Actor:      actorID,
		ActorType:  "staff",
		Action:     action,
		TargetRef:  materialTargetRef(material.ID),
		Severity:   "info",
		OccurredAt: occurredAt,
		Metadata:   meta,
		Diff:       diff,
	})
}

func materialTargetRef(materialID string) string {
	trimmed := strings.TrimSpace(materialID)
	if trimmed == "" {
		return ""
	}
	return fmt.Sprintf("/materials/%s", trimmed)
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

func normalizeMaterialSummary(material MaterialSummary) MaterialSummary {
	material.ID = strings.TrimSpace(material.ID)
	material.Name = strings.TrimSpace(material.Name)
	material.Description = strings.TrimSpace(material.Description)
	material.Category = strings.TrimSpace(material.Category)
	material.Grain = strings.TrimSpace(material.Grain)
	material.Color = strings.TrimSpace(material.Color)
	material.PreviewImagePath = strings.TrimSpace(material.PreviewImagePath)
	material.DefaultLocale = strings.TrimSpace(material.DefaultLocale)
	material.Translations = normalizeMaterialTranslations(material.Translations)
	material.Procurement = normalizeMaterialProcurement(material.Procurement)
	material.Inventory = normalizeMaterialInventory(material.Inventory)
	return material
}

func normalizeMaterialTranslations(translations map[string]MaterialTranslation) map[string]MaterialTranslation {
	if len(translations) == 0 {
		return nil
	}
	normalized := make(map[string]MaterialTranslation, len(translations))
	for key, translation := range translations {
		normalized[strings.TrimSpace(key)] = MaterialTranslation{
			Locale:      strings.TrimSpace(translation.Locale),
			Name:        strings.TrimSpace(translation.Name),
			Description: strings.TrimSpace(translation.Description),
		}
	}
	return normalized
}

func normalizeMaterialProcurement(info MaterialProcurement) MaterialProcurement {
	info.SupplierRef = strings.TrimSpace(info.SupplierRef)
	info.SupplierName = strings.TrimSpace(info.SupplierName)
	info.ContactEmail = strings.TrimSpace(info.ContactEmail)
	info.ContactPhone = strings.TrimSpace(info.ContactPhone)
	info.Currency = strings.ToUpper(strings.TrimSpace(info.Currency))
	info.Notes = strings.TrimSpace(info.Notes)
	return info
}

func normalizeMaterialInventory(info MaterialInventory) MaterialInventory {
	info.SKU = strings.TrimSpace(info.SKU)
	info.Warehouse = strings.TrimSpace(info.Warehouse)
	return info
}

func validateMaterialInput(material MaterialSummary) error {
	if material.ID == "" {
		return errors.New("material id is required")
	}
	if strings.TrimSpace(material.Name) == "" {
		return errors.New("material name is required")
	}
	if strings.TrimSpace(material.Category) == "" {
		return errors.New("material category is required")
	}
	if material.Inventory.SafetyStock < 0 {
		return errors.New("inventory safety stock must be >= 0")
	}
	if material.Inventory.ReorderPoint < 0 {
		return errors.New("inventory reorder point must be >= 0")
	}
	if material.Inventory.ReorderQuantity < 0 {
		return errors.New("inventory reorder quantity must be >= 0")
	}
	if material.Inventory.SKU == "" {
		if material.Inventory.SafetyStock > 0 || material.Inventory.ReorderPoint > 0 || material.Inventory.ReorderQuantity > 0 {
			return errors.New("inventory sku is required when safety thresholds are configured")
		}
	}
	if material.Procurement.UnitCostCents < 0 {
		return errors.New("procurement unit cost must be >= 0")
	}
	if material.Procurement.MinimumOrderQuantity < 0 {
		return errors.New("procurement minimum order quantity must be >= 0")
	}
	if material.Procurement.LeadTimeDays < 0 {
		return errors.New("procurement lead time must be >= 0")
	}
	if cur := strings.TrimSpace(material.Procurement.Currency); cur != "" && len(cur) != 3 {
		return errors.New("procurement currency must be a 3-letter ISO code")
	}
	return nil
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

func (s *catalogService) UpsertProduct(ctx context.Context, cmd UpsertProductCommand) (Product, error) {
	if s.repo == nil {
		return Product{}, ErrCatalogRepositoryMissing
	}
	product := normalizeProduct(cmd.Product)
	actorID := strings.TrimSpace(cmd.ActorID)
	if err := s.validateProductInput(ctx, product); err != nil {
		return Product{}, fmt.Errorf("%w: %s", ErrCatalogInvalidInput, err)
	}

	var existing Product
	current, err := s.repo.GetProduct(ctx, product.ID)
	if err != nil && !isCatalogRepositoryNotFound(err) {
		return Product{}, err
	}
	if err == nil {
		existing = Product(current)
	}

	now := s.clock()
	if existing.ID == "" {
		if product.CreatedAt.IsZero() {
			product.CreatedAt = now
		} else {
			product.CreatedAt = product.CreatedAt.UTC()
		}
	} else {
		if product.CreatedAt.IsZero() {
			product.CreatedAt = existing.CreatedAt
		} else {
			product.CreatedAt = product.CreatedAt.UTC()
		}
	}
	product.UpdatedAt = now

	if err := s.ensureProductSKUUnique(ctx, product, existing); err != nil {
		return Product{}, err
	}

	savedDomain, err := s.repo.UpsertProduct(ctx, domain.Product(product))
	if err != nil {
		return Product{}, err
	}
	saved := Product(savedDomain)

	if err := s.configureProductInventory(ctx, saved); err != nil {
		return Product{}, err
	}

	s.recordProductAudit(ctx, existing, saved, actorID)

	return saved, nil
}

func (s *catalogService) DeleteProduct(ctx context.Context, cmd DeleteProductCommand) error {
	if s.repo == nil {
		return ErrCatalogRepositoryMissing
	}
	productID := strings.TrimSpace(cmd.ProductID)
	if productID == "" {
		return errors.New("catalog service: product id is required")
	}
	current, err := s.repo.GetProduct(ctx, productID)
	if err != nil {
		if isCatalogRepositoryNotFound(err) {
			return nil
		}
		return err
	}
	existing := Product(current)
	if existing.ID == "" || !existing.IsPublished {
		return nil
	}
	before := existing
	existing.IsPublished = false
	existing.UpdatedAt = s.clock()
	savedDomain, err := s.repo.UpsertProduct(ctx, domain.Product(existing))
	if err != nil {
		return err
	}
	s.recordProductAudit(ctx, before, Product(savedDomain), strings.TrimSpace(cmd.ActorID))
	return nil
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

func normalizeProduct(product Product) Product {
	product.ID = strings.TrimSpace(product.ID)
	product.SKU = strings.ToUpper(strings.TrimSpace(product.SKU))
	product.Name = strings.TrimSpace(product.Name)
	product.Description = strings.TrimSpace(product.Description)
	product.Shape = strings.TrimSpace(product.Shape)
	product.DefaultMaterialID = strings.TrimSpace(product.DefaultMaterialID)
	product.InventoryStatus = strings.TrimSpace(product.InventoryStatus)
	product.Currency = strings.ToUpper(strings.TrimSpace(product.Currency))
	product.MaterialIDs = normalizeStringList(product.MaterialIDs)
	product.CompatibleTemplateIDs = normalizeStringList(product.CompatibleTemplateIDs)
	product.ImagePaths = normalizeStringList(product.ImagePaths)
	product.SizesMm = normalizeSizeList(product.SizesMm)
	product.PriceTiers = normalizeProductPriceTiers(product.PriceTiers)
	product.Variants = normalizeProductVariants(product.Variants)
	if product.Inventory.InitialStock < 0 {
		product.Inventory.InitialStock = 0
	}
	if product.Inventory.SafetyStock < 0 {
		product.Inventory.SafetyStock = 0
	}
	return product
}

func normalizeProductPriceTiers(tiers []ProductPriceTier) []ProductPriceTier {
	if len(tiers) == 0 {
		return nil
	}
	seen := make(map[int]struct{}, len(tiers))
	filtered := make([]ProductPriceTier, 0, len(tiers))
	for _, tier := range tiers {
		if tier.MinQuantity <= 0 || tier.UnitPrice < 0 {
			continue
		}
		if _, ok := seen[tier.MinQuantity]; ok {
			continue
		}
		seen[tier.MinQuantity] = struct{}{}
		filtered = append(filtered, ProductPriceTier{
			MinQuantity: tier.MinQuantity,
			UnitPrice:   tier.UnitPrice,
		})
	}
	if len(filtered) == 0 {
		return nil
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].MinQuantity < filtered[j].MinQuantity
	})
	return filtered
}

func normalizeProductVariants(variants []ProductVariant) []ProductVariant {
	if len(variants) == 0 {
		return nil
	}
	normalized := make([]ProductVariant, 0, len(variants))
	for _, variant := range variants {
		name := strings.TrimSpace(variant.Name)
		label := strings.TrimSpace(variant.Label)
		if name == "" && len(variant.Options) == 0 {
			continue
		}
		seen := make(map[string]struct{}, len(variant.Options))
		options := make([]ProductVariantOption, 0, len(variant.Options))
		for _, option := range variant.Options {
			value := strings.TrimSpace(option.Value)
			if value == "" {
				continue
			}
			key := strings.ToLower(value)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			options = append(options, ProductVariantOption{
				Value:        value,
				Label:        strings.TrimSpace(option.Label),
				PriceDelta:   option.PriceDelta,
				ImagePath:    strings.TrimSpace(option.ImagePath),
				IsDefault:    option.IsDefault,
				Availability: strings.TrimSpace(option.Availability),
			})
		}
		if len(options) == 0 {
			continue
		}
		normalized = append(normalized, ProductVariant{
			Name:    name,
			Label:   label,
			Options: options,
		})
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func normalizeSizeList(values []int) []int {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[int]struct{}, len(values))
	result := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	if len(result) == 0 {
		return nil
	}
	sort.Ints(result)
	return result
}

func stringSliceContains(values []string, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}

func isValidCurrency(code string) bool {
	if len(code) != 3 {
		return false
	}
	for _, r := range code {
		if !unicode.IsUpper(r) || !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func (s *catalogService) validateProductInput(ctx context.Context, product Product) error {
	if product.ID == "" {
		return errors.New("product id is required")
	}
	if product.SKU == "" {
		return errors.New("sku is required")
	}
	if product.BasePrice < 0 {
		return errors.New("base price must be >= 0")
	}
	if !isValidCurrency(product.Currency) {
		return errors.New("currency must be a 3-letter uppercase ISO code")
	}
	if len(product.SizesMm) == 0 {
		return errors.New("at least one size must be provided")
	}
	if len(product.MaterialIDs) == 0 {
		return errors.New("at least one material id must be provided")
	}
	if product.DefaultMaterialID == "" {
		return errors.New("default material id is required")
	}
	if !stringSliceContains(product.MaterialIDs, product.DefaultMaterialID) {
		return fmt.Errorf("default material %s must be included in material_ids", product.DefaultMaterialID)
	}
	if product.LeadTimeDays < 0 {
		return errors.New("lead time days must be >= 0")
	}
	if product.Inventory.SafetyStock < 0 {
		return errors.New("inventory safety stock must be >= 0")
	}
	if product.Inventory.InitialStock < 0 {
		return errors.New("inventory initial stock must be >= 0")
	}
	if err := validatePriceTiersInput(product.PriceTiers); err != nil {
		return err
	}
	if err := validateProductVariantsInput(product.Variants); err != nil {
		return err
	}
	if err := s.ensureMaterialsExist(ctx, product.MaterialIDs); err != nil {
		return err
	}
	if err := s.ensureTemplatesExist(ctx, product.CompatibleTemplateIDs); err != nil {
		return err
	}
	return nil
}

func validatePriceTiersInput(tiers []ProductPriceTier) error {
	prev := 0
	for i, tier := range tiers {
		if tier.MinQuantity <= 0 {
			return fmt.Errorf("price tier %d min quantity must be > 0", i)
		}
		if tier.UnitPrice < 0 {
			return fmt.Errorf("price tier %d unit price must be >= 0", i)
		}
		if prev > 0 && tier.MinQuantity <= prev {
			return fmt.Errorf("price tiers must be in ascending order")
		}
		prev = tier.MinQuantity
	}
	return nil
}

func validateProductVariantsInput(variants []ProductVariant) error {
	for _, variant := range variants {
		if strings.TrimSpace(variant.Name) == "" {
			return errors.New("variant name is required")
		}
		if len(variant.Options) == 0 {
			return fmt.Errorf("variant %s must include at least one option", variant.Name)
		}
	}
	return nil
}

func (s *catalogService) ensureMaterialsExist(ctx context.Context, ids []string) error {
	if s.repo == nil || len(ids) == 0 {
		return nil
	}
	checked := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		materialID := strings.TrimSpace(id)
		if materialID == "" {
			continue
		}
		if _, ok := checked[materialID]; ok {
			continue
		}
		checked[materialID] = struct{}{}
		if _, err := s.repo.GetMaterial(ctx, materialID); err != nil {
			if isCatalogRepositoryNotFound(err) {
				return fmt.Errorf("material %s not found", materialID)
			}
			return err
		}
	}
	return nil
}

func (s *catalogService) ensureTemplatesExist(ctx context.Context, ids []string) error {
	if s.repo == nil || len(ids) == 0 {
		return nil
	}
	checked := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		templateID := strings.TrimSpace(id)
		if templateID == "" {
			continue
		}
		if _, ok := checked[templateID]; ok {
			continue
		}
		checked[templateID] = struct{}{}
		if _, err := s.repo.GetTemplate(ctx, templateID); err != nil {
			if isCatalogRepositoryNotFound(err) {
				return fmt.Errorf("template %s not found", templateID)
			}
			return err
		}
	}
	return nil
}

func (s *catalogService) ensureProductSKUUnique(ctx context.Context, candidate Product, existing Product) error {
	if s.repo == nil || strings.TrimSpace(candidate.SKU) == "" {
		return nil
	}
	found, err := s.repo.FindProductBySKU(ctx, candidate.SKU)
	if err != nil {
		if isCatalogRepositoryNotFound(err) {
			return nil
		}
		return err
	}
	if found.ID == "" {
		return nil
	}
	if existing.ID != "" && found.ID == existing.ID {
		return nil
	}
	if found.ID != candidate.ID {
		return ErrCatalogProductConflict
	}
	return nil
}

func (s *catalogService) configureProductInventory(ctx context.Context, product Product) error {
	if s.inventory == nil {
		return nil
	}
	sku := strings.TrimSpace(product.SKU)
	if sku == "" {
		return nil
	}
	if product.Inventory.SafetyStock == 0 && product.Inventory.InitialStock == 0 {
		return nil
	}
	var initial *int
	if product.Inventory.InitialStock > 0 {
		value := product.Inventory.InitialStock
		initial = &value
	}
	_, err := s.inventory.ConfigureSafetyStock(ctx, ConfigureSafetyStockCommand{
		SKU:           sku,
		ProductRef:    productTargetRef(product.ID),
		SafetyStock:   product.Inventory.SafetyStock,
		InitialOnHand: initial,
	})
	return err
}

func productTargetRef(productID string) string {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return ""
	}
	return fmt.Sprintf("/products/%s", productID)
}

func (s *catalogService) recordProductAudit(ctx context.Context, before Product, after Product, actorID string) {
	if s.audit == nil {
		return
	}
	actorID = strings.TrimSpace(actorID)
	action := "catalog.product.update"
	occurredAt := after.UpdatedAt
	if before.ID == "" {
		action = "catalog.product.create"
		occurredAt = after.CreatedAt
	}
	diff := map[string]AuditLogDiff{}
	if before.IsPublished != after.IsPublished {
		diff["isPublished"] = AuditLogDiff{Before: before.IsPublished, After: after.IsPublished}
	}
	if before.BasePrice != after.BasePrice {
		diff["basePrice"] = AuditLogDiff{Before: before.BasePrice, After: after.BasePrice}
	}
	if before.Currency != after.Currency {
		diff["currency"] = AuditLogDiff{Before: before.Currency, After: after.Currency}
	}
	if before.InventoryStatus != after.InventoryStatus {
		diff["inventoryStatus"] = AuditLogDiff{Before: before.InventoryStatus, After: after.InventoryStatus}
	}
	if before.LeadTimeDays != after.LeadTimeDays {
		diff["leadTimeDays"] = AuditLogDiff{Before: before.LeadTimeDays, After: after.LeadTimeDays}
	}
	if len(diff) == 0 {
		diff = nil
	}
	metadata := map[string]any{
		"productId":   after.ID,
		"sku":         after.SKU,
		"shape":       after.Shape,
		"materials":   after.MaterialIDs,
		"templates":   after.CompatibleTemplateIDs,
		"leadTime":    after.LeadTimeDays,
		"isPublished": after.IsPublished,
	}
	s.audit.Record(ctx, AuditLogRecord{
		Actor:      actorID,
		ActorType:  "staff",
		Action:     action,
		TargetRef:  productTargetRef(after.ID),
		Severity:   "info",
		OccurredAt: occurredAt,
		Metadata:   metadata,
		Diff:       diff,
	})

	if before.IsPublished != after.IsPublished {
		stateAction := "catalog.product.unpublish"
		if after.IsPublished {
			stateAction = "catalog.product.publish"
		}
		stateDiff := map[string]AuditLogDiff{
			"isPublished": {Before: before.IsPublished, After: after.IsPublished},
		}
		s.audit.Record(ctx, AuditLogRecord{
			Actor:      actorID,
			ActorType:  "staff",
			Action:     stateAction,
			TargetRef:  productTargetRef(after.ID),
			Severity:   "info",
			OccurredAt: occurredAt,
			Metadata:   metadata,
			Diff:       stateDiff,
		})
	}
}
