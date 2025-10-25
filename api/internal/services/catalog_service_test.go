package services

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

func TestNewCatalogService(t *testing.T) {
	t.Run("requires repository", func(t *testing.T) {
		if _, err := NewCatalogService(CatalogServiceDeps{}); err == nil {
			t.Fatalf("expected error when repository missing")
		}
	})

	t.Run("uses provided clock", func(t *testing.T) {
		stubRepo := &stubCatalogRepository{}
		now := time.Date(2024, time.April, 1, 15, 4, 5, 0, time.UTC)
		svc, err := NewCatalogService(CatalogServiceDeps{
			Catalog: stubRepo,
			Clock: func() time.Time {
				return now
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		template := Template{
			TemplateSummary: domain.TemplateSummary{
				ID:               "tpl_001",
				Name:             "  Classic ",
				Category:         " round ",
				Style:            " serif ",
				Description:      "  sample ",
				Tags:             []string{" Bold ", "bold", "  "},
				PreviewImagePath: " previews/tpl_001.png ",
				IsPublished:      true,
			},
			SVGPath: " svg/tpl_001.svg ",
		}
		_, err = svc.UpsertTemplate(context.Background(), UpsertTemplateCommand{Template: template})
		if err != nil {
			t.Fatalf("unexpected error upserting: %v", err)
		}

		if stubRepo.upsertInput.CreatedAt != now {
			t.Fatalf("expected CreatedAt to default to clock value, got %v", stubRepo.upsertInput.CreatedAt)
		}
		if stubRepo.upsertInput.UpdatedAt != now {
			t.Fatalf("expected UpdatedAt to use clock value, got %v", stubRepo.upsertInput.UpdatedAt)
		}
		if got, want := stubRepo.upsertInput.Name, "Classic"; got != want {
			t.Fatalf("expected trimmed name %q, got %q", want, got)
		}
		if got, want := stubRepo.upsertInput.Category, "round"; got != want {
			t.Fatalf("expected trimmed category %q, got %q", want, got)
		}
		if got, want := stubRepo.upsertInput.Style, "serif"; got != want {
			t.Fatalf("expected trimmed style %q, got %q", want, got)
		}
		if got, want := stubRepo.upsertInput.Description, "sample"; got != want {
			t.Fatalf("expected trimmed description %q, got %q", want, got)
		}
		if !reflect.DeepEqual(stubRepo.upsertInput.Tags, []string{"Bold"}) {
			t.Fatalf("expected normalised tags [Bold], got %#v", stubRepo.upsertInput.Tags)
		}
		if got, want := stubRepo.upsertInput.PreviewImagePath, "previews/tpl_001.png"; got != want {
			t.Fatalf("expected trimmed preview path %q, got %q", want, got)
		}
		if got, want := stubRepo.upsertInput.SVGPath, "svg/tpl_001.svg"; got != want {
			t.Fatalf("expected trimmed svg path %q, got %q", want, got)
		}
		if stubRepo.upsertInput.Version != 1 {
			t.Fatalf("expected version 1, got %d", stubRepo.upsertInput.Version)
		}
		if stubRepo.upsertInput.PublishedAt != now {
			t.Fatalf("expected publishedAt to use clock, got %v", stubRepo.upsertInput.PublishedAt)
		}
		if len(stubRepo.templateVersions) != 1 {
			t.Fatalf("expected a single template version append, got %d", len(stubRepo.templateVersions))
		}
		if stubRepo.templateVersions[0].Version != 1 {
			t.Fatalf("expected appended version 1, got %d", stubRepo.templateVersions[0].Version)
		}
	})
}

func TestCatalogServiceListTemplates(t *testing.T) {
	stubRepo := &stubCatalogRepository{
		listResp: domain.CursorPage[domain.TemplateSummary]{
			Items: []domain.TemplateSummary{
				{ID: "tpl_001"},
			},
			NextPageToken: "next",
		},
	}
	svc, err := NewCatalogService(CatalogServiceDeps{
		Catalog: stubRepo,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	category := "  category "
	style := " "
	filter := TemplateFilter{
		Category:      &category,
		Style:         &style,
		Tags:          []string{" modern ", "Modern", ""},
		SortBy:        "unknown",
		SortOrder:     "invalid",
		PublishedOnly: true,
		Pagination: Pagination{
			PageSize:  25,
			PageToken: " token ",
		},
	}

	page, err := svc.ListTemplates(context.Background(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(page, stubRepo.listResp) {
		t.Fatalf("expected repository page %v, got %v", stubRepo.listResp, page)
	}

	if stubRepo.listFilter.Category == nil || *stubRepo.listFilter.Category != "category" {
		t.Fatalf("expected category pointer trimmed, got %#v", stubRepo.listFilter.Category)
	}
	if stubRepo.listFilter.Style != nil {
		t.Fatalf("expected empty style to be cleared")
	}
	if !reflect.DeepEqual(stubRepo.listFilter.Tags, []string{"modern"}) {
		t.Fatalf("expected normalised tags [modern], got %#v", stubRepo.listFilter.Tags)
	}
	if stubRepo.listFilter.SortBy != domain.TemplateSortPopularity {
		t.Fatalf("expected default sort by popularity, got %v", stubRepo.listFilter.SortBy)
	}
	if stubRepo.listFilter.SortOrder != domain.SortDesc {
		t.Fatalf("expected default sort order desc, got %v", stubRepo.listFilter.SortOrder)
	}
	if stubRepo.listFilter.Pagination.PageToken != "token" {
		t.Fatalf("expected trimmed page token, got %q", stubRepo.listFilter.Pagination.PageToken)
	}
	if !stubRepo.listFilter.OnlyPublished {
		t.Fatalf("expected published flag to be propagated")
	}
}

func TestCatalogServiceUpsertTemplate_ExistingTemplateVersioning(t *testing.T) {
	existingCreatedAt := time.Date(2024, time.February, 2, 10, 0, 0, 0, time.UTC)
	existingPublishedAt := time.Date(2024, time.February, 5, 10, 0, 0, 0, time.UTC)
	repo := &stubCatalogRepository{
		getTemplate: domain.Template{
			TemplateSummary: domain.TemplateSummary{
				ID:          "tpl_777",
				Name:        "Existing",
				IsPublished: true,
				Version:     2,
				CreatedAt:   existingCreatedAt,
				UpdatedAt:   existingPublishedAt,
				PublishedAt: existingPublishedAt,
			},
			SVGPath: "svg/existing.svg",
		},
	}
	cache := &stubTemplateCache{}
	audit := &stubAuditLogService{}
	now := time.Date(2024, time.March, 1, 12, 0, 0, 0, time.UTC)
	svc, err := NewCatalogService(CatalogServiceDeps{
		Catalog: repo,
		Cache:   cache,
		Audit:   audit,
		Clock: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	template := Template{
		TemplateSummary: domain.TemplateSummary{
			ID:          "tpl_777",
			Name:        "Existing",
			Description: "Updated copy",
			IsPublished: false,
		},
		SVGPath: " svg/new.svg ",
		Draft: TemplateDraft{
			Notes: "  adjust kerning  ",
		},
	}
	actor := "admin-777"
	saved, err := svc.UpsertTemplate(context.Background(), UpsertTemplateCommand{Template: template, ActorID: actor})
	if err != nil {
		t.Fatalf("unexpected error upserting: %v", err)
	}
	if saved.Version != 3 {
		t.Fatalf("expected version to increment to 3, got %d", saved.Version)
	}
	if !saved.CreatedAt.Equal(existingCreatedAt) {
		t.Fatalf("expected createdAt to remain unchanged, got %v", saved.CreatedAt)
	}
	if !saved.PublishedAt.IsZero() {
		t.Fatalf("expected publishedAt reset when unpublishing, got %v", saved.PublishedAt)
	}
	if len(repo.templateVersions) != 1 {
		t.Fatalf("expected version append, got %d", len(repo.templateVersions))
	}
	if repo.templateVersions[0].CreatedBy != actor {
		t.Fatalf("expected version created by %s, got %s", actor, repo.templateVersions[0].CreatedBy)
	}
	if got := repo.upsertInput.Draft.Notes; got != "adjust kerning" {
		t.Fatalf("expected trimmed draft notes, got %q", got)
	}
	if repo.upsertInput.Draft.UpdatedBy != actor {
		t.Fatalf("expected draft updatedBy %s, got %s", actor, repo.upsertInput.Draft.UpdatedBy)
	}
	if len(cache.calls) != 1 {
		t.Fatalf("expected cache invalidation, got %d", len(cache.calls))
	}
	if cache.calls[0][0] != "tpl_777" {
		t.Fatalf("expected invalidated template tpl_777, got %#v", cache.calls)
	}
	if len(audit.records) < 2 {
		t.Fatalf("expected audit records for update + unpublish, got %d", len(audit.records))
	}
	if audit.records[0].Action != "catalog.template.update" {
		t.Fatalf("expected first audit action update, got %s", audit.records[0].Action)
	}
	if audit.records[1].Action != "catalog.template.unpublish" {
		t.Fatalf("expected unpublish audit action, got %s", audit.records[1].Action)
	}
}

func TestCatalogServiceGetTemplate(t *testing.T) {
	stubRepo := &stubCatalogRepository{
		getPublished: domain.Template{
			TemplateSummary: domain.TemplateSummary{ID: "tpl_001"},
			SVGPath:         "svg/path.svg",
		},
	}
	svc, err := NewCatalogService(CatalogServiceDeps{Catalog: stubRepo})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("validates id", func(t *testing.T) {
		if _, err := svc.GetTemplate(context.Background(), " "); err == nil {
			t.Fatalf("expected error when id empty")
		}
	})

	t.Run("delegates to repository", func(t *testing.T) {
		template, err := svc.GetTemplate(context.Background(), " tpl_001 ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if template.ID != "tpl_001" {
			t.Fatalf("expected template id tpl_001, got %s", template.ID)
		}
		if stubRepo.getPublishedID != "tpl_001" {
			t.Fatalf("expected repository to receive trimmed id tpl_001, got %q", stubRepo.getPublishedID)
		}
	})
}

func TestCatalogServiceGetFont(t *testing.T) {
	stubRepo := &stubCatalogRepository{
		fontGetPublished: domain.Font{
			FontSummary: domain.FontSummary{ID: "font_001", IsPublished: true},
		},
	}
	svc, err := NewCatalogService(CatalogServiceDeps{Catalog: stubRepo})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	font, err := svc.GetFont(context.Background(), " font_001 ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if font.ID != "font_001" {
		t.Fatalf("expected font id font_001, got %s", font.ID)
	}
	if stubRepo.fontGetPublishedID != "font_001" {
		t.Fatalf("expected repository to receive trimmed id font_001, got %q", stubRepo.fontGetPublishedID)
	}
}

func TestCatalogServiceListProducts(t *testing.T) {
	stubRepo := &stubCatalogRepository{
		productListResp: domain.CursorPage[domain.ProductSummary]{
			Items: []domain.ProductSummary{{ID: "prod_001"}},
		},
	}
	svc, err := NewCatalogService(CatalogServiceDeps{Catalog: stubRepo})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	shape := " Round "
	material := " mat_wood "
	size := 45
	customizable := true
	filter := ProductFilter{
		Shape:          &shape,
		SizeMm:         &size,
		MaterialID:     &material,
		IsCustomizable: &customizable,
		PublishedOnly:  true,
		Pagination: Pagination{
			PageSize:  50,
			PageToken: " token ",
		},
	}

	page, err := svc.ListProducts(context.Background(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(page, stubRepo.productListResp) {
		t.Fatalf("expected repository response, got %#v", page)
	}

	if stubRepo.productListFilter.Shape == nil || *stubRepo.productListFilter.Shape != "Round" {
		t.Fatalf("expected trimmed shape Round got %#v", stubRepo.productListFilter.Shape)
	}
	if stubRepo.productListFilter.SizeMm == nil || *stubRepo.productListFilter.SizeMm != 45 {
		t.Fatalf("expected size 45 got %#v", stubRepo.productListFilter.SizeMm)
	}
	if stubRepo.productListFilter.MaterialID == nil || *stubRepo.productListFilter.MaterialID != "mat_wood" {
		t.Fatalf("expected trimmed material mat_wood got %#v", stubRepo.productListFilter.MaterialID)
	}
	if stubRepo.productListFilter.IsCustomizable == nil || !*stubRepo.productListFilter.IsCustomizable {
		t.Fatalf("expected customizable flag true got %#v", stubRepo.productListFilter.IsCustomizable)
	}
	if !stubRepo.productListFilter.OnlyPublished {
		t.Fatalf("expected repository filter to request only published products")
	}
	if stubRepo.productListFilter.Pagination.PageSize != 50 {
		t.Fatalf("expected page size 50 got %d", stubRepo.productListFilter.Pagination.PageSize)
	}
	if stubRepo.productListFilter.Pagination.PageToken != "token" {
		t.Fatalf("expected trimmed page token got %q", stubRepo.productListFilter.Pagination.PageToken)
	}
}

func TestCatalogServiceGetProduct(t *testing.T) {
	stubRepo := &stubCatalogRepository{
		productGetPublished: domain.Product{
			ProductSummary: domain.ProductSummary{ID: "prod_001"},
		},
	}
	svc, err := NewCatalogService(CatalogServiceDeps{Catalog: stubRepo})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("validates id", func(t *testing.T) {
		if _, err := svc.GetProduct(context.Background(), " "); err == nil {
			t.Fatalf("expected error when product id empty")
		}
	})

	t.Run("delegates to repository", func(t *testing.T) {
		product, err := svc.GetProduct(context.Background(), " prod_001 ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if product.ID != "prod_001" {
			t.Fatalf("expected product id prod_001 got %s", product.ID)
		}
		if stubRepo.productGetPublishedID != "prod_001" {
			t.Fatalf("expected repository to receive trimmed id prod_001 got %q", stubRepo.productGetPublishedID)
		}
	})
}

func TestCatalogServiceDeleteTemplate(t *testing.T) {
	repo := &stubCatalogRepository{
		getTemplate: domain.Template{
			TemplateSummary: domain.TemplateSummary{
				ID:          "tpl_001",
				IsPublished: true,
			},
		},
	}
	cache := &stubTemplateCache{}
	audit := &stubAuditLogService{}
	now := time.Date(2024, time.March, 2, 8, 0, 0, 0, time.UTC)
	svc, err := NewCatalogService(CatalogServiceDeps{
		Catalog: repo,
		Cache:   cache,
		Audit:   audit,
		Clock: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd := DeleteTemplateCommand{TemplateID: " tpl_001 ", ActorID: "admin"}
	if err := svc.DeleteTemplate(context.Background(), cmd); err != nil {
		t.Fatalf("unexpected error deleting: %v", err)
	}
	if repo.deletedID != "tpl_001" {
		t.Fatalf("expected trimmed id, got %q", repo.deletedID)
	}
	if len(cache.calls) != 1 {
		t.Fatalf("expected cache invalidation, got %d", len(cache.calls))
	}
	if len(audit.records) == 0 || audit.records[len(audit.records)-1].Action != "catalog.template.delete" {
		t.Fatalf("expected delete audit action, got %#v", audit.records)
	}

	if err := svc.DeleteTemplate(context.Background(), DeleteTemplateCommand{}); err == nil {
		t.Fatalf("expected error when id empty")
	}
}

type stubCatalogRepository struct {
	listFilter         repositories.TemplateFilter
	listResp           domain.CursorPage[domain.TemplateSummary]
	listErr            error
	fontListFilter     repositories.FontFilter
	fontListResp       domain.CursorPage[domain.FontSummary]
	fontListErr        error
	materialListFilter repositories.MaterialFilter
	materialListResp   domain.CursorPage[domain.MaterialSummary]
	materialListErr    error
	productListFilter  repositories.ProductFilter
	productListResp    domain.CursorPage[domain.ProductSummary]
	productListErr     error

	getPublishedID          string
	getPublished            domain.Template
	getPublishedErr         error
	fontGetPublishedID      string
	fontGetPublished        domain.Font
	fontGetPublishedErr     error
	materialGetPublishedID  string
	materialGetPublished    domain.Material
	materialGetPublishedErr error
	productGetPublishedID   string
	productGetPublished     domain.Product
	productGetPublishedErr  error

	getID          string
	getTemplate    domain.Template
	getErr         error
	fontGetID      string
	fontGet        domain.Font
	fontGetErr     error
	materialGetID  string
	materialGet    domain.Material
	materialGetErr error
	productGetID   string
	productGet     domain.Product
	productGetErr  error

	upsertInput  domain.Template
	upsertResult domain.Template
	upsertErr    error

	deletedID string
	deleteErr error

	templateVersions []domain.TemplateVersion
	appendVersionErr error
}

func (s *stubCatalogRepository) ListTemplates(_ context.Context, filter repositories.TemplateFilter) (domain.CursorPage[domain.TemplateSummary], error) {
	s.listFilter = filter
	return s.listResp, s.listErr
}

func (s *stubCatalogRepository) GetPublishedTemplate(_ context.Context, templateID string) (domain.Template, error) {
	s.getPublishedID = templateID
	if s.getPublishedErr != nil {
		return domain.Template{}, s.getPublishedErr
	}
	if s.getPublished.ID != "" {
		return s.getPublished, nil
	}
	if s.getTemplate.ID != "" {
		return s.getTemplate, nil
	}
	return domain.Template{}, nil
}

func (s *stubCatalogRepository) GetTemplate(_ context.Context, templateID string) (domain.Template, error) {
	s.getID = templateID
	return s.getTemplate, s.getErr
}

func (s *stubCatalogRepository) UpsertTemplate(_ context.Context, template domain.Template) (domain.Template, error) {
	s.upsertInput = template
	if s.upsertErr != nil {
		return domain.Template{}, s.upsertErr
	}
	if s.upsertResult.ID != "" {
		return s.upsertResult, nil
	}
	return template, nil
}

func (s *stubCatalogRepository) DeleteTemplate(_ context.Context, templateID string) error {
	s.deletedID = templateID
	return s.deleteErr
}

func (s *stubCatalogRepository) AppendTemplateVersion(_ context.Context, version domain.TemplateVersion) error {
	s.templateVersions = append(s.templateVersions, version)
	return s.appendVersionErr
}

func (s *stubCatalogRepository) ListFonts(_ context.Context, filter repositories.FontFilter) (domain.CursorPage[domain.FontSummary], error) {
	s.fontListFilter = filter
	return s.fontListResp, s.fontListErr
}

func (s *stubCatalogRepository) GetPublishedFont(_ context.Context, fontID string) (domain.Font, error) {
	s.fontGetPublishedID = fontID
	if s.fontGetPublishedErr != nil {
		return domain.Font{}, s.fontGetPublishedErr
	}
	if s.fontGetPublished.ID != "" {
		return s.fontGetPublished, nil
	}
	if s.fontGet.ID != "" {
		return s.fontGet, nil
	}
	return domain.Font{}, nil
}

func (s *stubCatalogRepository) GetFont(_ context.Context, fontID string) (domain.Font, error) {
	s.fontGetID = fontID
	return s.fontGet, s.fontGetErr
}

func (s *stubCatalogRepository) UpsertFont(context.Context, domain.FontSummary) (domain.FontSummary, error) {
	return domain.FontSummary{}, errors.New("not implemented")
}

func (s *stubCatalogRepository) DeleteFont(context.Context, string) error {
	return errors.New("not implemented")
}

func (s *stubCatalogRepository) ListMaterials(_ context.Context, filter repositories.MaterialFilter) (domain.CursorPage[domain.MaterialSummary], error) {
	s.materialListFilter = filter
	return s.materialListResp, s.materialListErr
}

func (s *stubCatalogRepository) GetPublishedMaterial(_ context.Context, materialID string) (domain.Material, error) {
	s.materialGetPublishedID = materialID
	if s.materialGetPublishedErr != nil {
		return domain.Material{}, s.materialGetPublishedErr
	}
	if s.materialGetPublished.ID != "" {
		return s.materialGetPublished, nil
	}
	if s.materialGet.ID != "" {
		return s.materialGet, nil
	}
	return domain.Material{}, nil
}

func (s *stubCatalogRepository) GetMaterial(_ context.Context, materialID string) (domain.Material, error) {
	s.materialGetID = materialID
	return s.materialGet, s.materialGetErr
}

func (s *stubCatalogRepository) UpsertMaterial(context.Context, domain.MaterialSummary) (domain.MaterialSummary, error) {
	return domain.MaterialSummary{}, errors.New("not implemented")
}

func (s *stubCatalogRepository) DeleteMaterial(context.Context, string) error {
	return errors.New("not implemented")
}

func (s *stubCatalogRepository) ListProducts(_ context.Context, filter repositories.ProductFilter) (domain.CursorPage[domain.ProductSummary], error) {
	s.productListFilter = filter
	return s.productListResp, s.productListErr
}

func (s *stubCatalogRepository) GetPublishedProduct(_ context.Context, productID string) (domain.Product, error) {
	s.productGetPublishedID = productID
	if s.productGetPublishedErr != nil {
		return domain.Product{}, s.productGetPublishedErr
	}
	if s.productGetPublished.ID != "" {
		return s.productGetPublished, nil
	}
	if s.productGet.ID != "" {
		return s.productGet, nil
	}
	return domain.Product{}, nil
}

func (s *stubCatalogRepository) GetProduct(_ context.Context, productID string) (domain.Product, error) {
	s.productGetID = productID
	return s.productGet, s.productGetErr
}

func (s *stubCatalogRepository) UpsertProduct(context.Context, domain.ProductSummary) (domain.ProductSummary, error) {
	return domain.ProductSummary{}, errors.New("not implemented")
}

func (s *stubCatalogRepository) DeleteProduct(context.Context, string) error {
	return errors.New("not implemented")
}

type stubTemplateCache struct {
	calls [][]string
}

func (s *stubTemplateCache) InvalidateTemplates(_ context.Context, ids []string) error {
	clone := append([]string(nil), ids...)
	s.calls = append(s.calls, clone)
	return nil
}

type stubAuditLogService struct {
	records []AuditLogRecord
}

func (s *stubAuditLogService) Record(_ context.Context, record AuditLogRecord) {
	s.records = append(s.records, record)
}

func (s *stubAuditLogService) List(context.Context, AuditLogFilter) (domain.CursorPage[domain.AuditLogEntry], error) {
	return domain.CursorPage[domain.AuditLogEntry]{}, nil
}
