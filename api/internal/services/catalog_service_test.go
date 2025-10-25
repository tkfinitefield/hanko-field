package services

import (
	"context"
	"errors"
	"reflect"
	"strings"
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

func TestCatalogServiceUpsertFont_ValidatesInput(t *testing.T) {
	repo := &stubCatalogRepository{}
	now := time.Date(2024, time.May, 1, 12, 0, 0, 0, time.UTC)
	svc, err := NewCatalogService(CatalogServiceDeps{Catalog: repo, Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = svc.UpsertFont(context.Background(), UpsertFontCommand{
		Font: FontSummary{
			DisplayName:      "",
			Family:           "Ryumin",
			Weight:           "Regular",
			Scripts:          []string{"kanji"},
			PreviewImagePath: "fonts/ryumin.png",
			License: FontLicense{
				Name:          "Commercial",
				URL:           "",
				AllowedUsages: []string{"app"},
			},
		},
	})
	if !errors.Is(err, ErrCatalogInvalidInput) {
		t.Fatalf("expected invalid input error, got %v", err)
	}
}

func TestCatalogServiceUpsertFont_SlugAndConflict(t *testing.T) {
	repo := &stubCatalogRepository{
		fontGet: domain.Font{
			FontSummary: domain.FontSummary{
				ID:        "tensho-regular",
				CreatedAt: time.Date(2024, time.April, 10, 0, 0, 0, 0, time.UTC),
			},
		},
	}
	now := time.Date(2024, time.May, 2, 9, 0, 0, 0, time.UTC)
	svc, err := NewCatalogService(CatalogServiceDeps{Catalog: repo, Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.UpsertFont(context.Background(), UpsertFontCommand{
		Font: FontSummary{
			DisplayName:      "Tensho Regular",
			Family:           "Tensho",
			Weight:           "Regular",
			Scripts:          []string{"kanji"},
			PreviewImagePath: "fonts/tensho.png",
			License: FontLicense{
				Name:          "Commercial",
				URL:           "https://example.com/license",
				AllowedUsages: []string{"print"},
			},
		},
	})
	if !errors.Is(err, ErrCatalogFontConflict) {
		t.Fatalf("expected conflict error, got %v", err)
	}

	repo.fontUpsertResp = domain.FontSummary{ID: "tensho-regular", CreatedAt: repo.fontGet.CreatedAt, UpdatedAt: now}
	result, err := svc.UpsertFont(context.Background(), UpsertFontCommand{
		Font: FontSummary{
			ID:               "tensho-regular",
			DisplayName:      "Tensho Regular",
			Family:           "Tensho",
			Weight:           "Regular",
			Scripts:          []string{"kanji"},
			PreviewImagePath: "fonts/tensho.png",
			SupportedWeights: []string{"400"},
			License: FontLicense{
				Name:          "Commercial",
				URL:           "https://example.com/license",
				AllowedUsages: []string{"print"},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "tensho-regular" {
		t.Fatalf("expected slug id, got %s", result.ID)
	}
	if repo.fontUpsertInput.CreatedAt != repo.fontGet.CreatedAt {
		t.Fatalf("expected CreatedAt preserved, got %v", repo.fontUpsertInput.CreatedAt)
	}
	if !containsString(repo.fontUpsertInput.SupportedWeights, "regular") {
		t.Fatalf("expected supported weights to include weight, got %#v", repo.fontUpsertInput.SupportedWeights)
	}
}

func TestCatalogServiceUpsertFont_RejectsMismatchedPathID(t *testing.T) {
	repo := &stubCatalogRepository{}
	svc, err := NewCatalogService(CatalogServiceDeps{Catalog: repo})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = svc.UpsertFont(context.Background(), UpsertFontCommand{
		Font: FontSummary{
			ID:               "tensho-regular",
			DisplayName:      "Tensho Regular",
			Family:           "Ryumin",
			Weight:           "Regular",
			Scripts:          []string{"kanji"},
			PreviewImagePath: "fonts/tensho.png",
			License: FontLicense{
				Name:          "Commercial",
				URL:           "https://example.com/license",
				AllowedUsages: []string{"print"},
			},
		},
	})
	if !errors.Is(err, ErrCatalogInvalidInput) {
		t.Fatalf("expected invalid input when path id mismatches slug, got %v", err)
	}
}

func TestCatalogServiceDeleteFont_Conflict(t *testing.T) {
	repo := &stubCatalogRepository{fontDeleteErr: stubRepositoryError{conflict: true}}
	svc, err := NewCatalogService(CatalogServiceDeps{Catalog: repo})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := svc.DeleteFont(context.Background(), " font_in_use "); !errors.Is(err, ErrCatalogFontInUse) {
		t.Fatalf("expected font in use error, got %v", err)
	}
	if repo.fontDeleteID != "font_in_use" {
		t.Fatalf("expected trimmed font id, got %q", repo.fontDeleteID)
	}
	if err := svc.DeleteFont(context.Background(), " "); !errors.Is(err, ErrCatalogInvalidInput) {
		t.Fatalf("expected invalid input error, got %v", err)
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

func TestCatalogServiceUpsertProductValidations(t *testing.T) {
	repo := &stubCatalogRepository{
		materialGet: domain.Material{MaterialSummary: domain.MaterialSummary{ID: "mat_wood"}},
	}
	svc, err := NewCatalogService(CatalogServiceDeps{Catalog: repo})
	if err != nil {
		t.Fatalf("new catalog service: %v", err)
	}
	base := Product{
		ProductSummary: domain.ProductSummary{
			ID:                "prod_round",
			SKU:               "SKU-100",
			Name:              "Round 30mm",
			Shape:             "round",
			SizesMm:           []int{30},
			DefaultMaterialID: "mat_wood",
			MaterialIDs:       []string{"mat_wood"},
			BasePrice:         5400,
			Currency:          "JPY",
			InventoryStatus:   "inventory",
			LeadTimeDays:      5,
			IsPublished:       true,
		},
	}

	t.Run("ensures default material present in list", func(t *testing.T) {
		product := base
		product.MaterialIDs = []string{"mat_bamboo"}
		if _, err := svc.UpsertProduct(context.Background(), UpsertProductCommand{Product: product}); err == nil {
			t.Fatalf("expected validation error for default material not included")
		}
	})

	t.Run("fails when template missing", func(t *testing.T) {
		product := base
		product.CompatibleTemplateIDs = []string{"tpl_missing"}
		repo.getTemplate = domain.Template{}
		repo.getErr = stubRepositoryError{notFound: true}
		if _, err := svc.UpsertProduct(context.Background(), UpsertProductCommand{Product: product}); err == nil {
			t.Fatalf("expected error when template missing")
		}
	})
}

func TestCatalogServiceUpsertProductConfiguresInventory(t *testing.T) {
	repo := &stubCatalogRepository{
		materialGet:         domain.Material{MaterialSummary: domain.MaterialSummary{ID: "mat_steel"}},
		productFindBySKUErr: stubRepositoryError{notFound: true},
	}
	inventory := &stubInventorySafetyService{}
	svc, err := NewCatalogService(CatalogServiceDeps{
		Catalog:   repo,
		Inventory: inventory,
		Clock: func() time.Time {
			return time.Date(2024, time.June, 1, 10, 0, 0, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatalf("new catalog service: %v", err)
	}
	initialStock := 20
	product := Product{
		ProductSummary: domain.ProductSummary{
			ID:                "prod_inv",
			SKU:               "SKU-INV",
			Name:              "Inventory Product",
			Shape:             "square",
			SizesMm:           []int{20},
			DefaultMaterialID: "mat_steel",
			MaterialIDs:       []string{"mat_steel"},
			BasePrice:         8800,
			Currency:          "JPY",
			InventoryStatus:   "inventory",
			IsPublished:       true,
		},
		Inventory: ProductInventorySettings{
			SafetyStock:  5,
			InitialStock: initialStock,
		},
	}
	saved, err := svc.UpsertProduct(context.Background(), UpsertProductCommand{
		Product: product,
		ActorID: "admin",
	})
	if err != nil {
		t.Fatalf("upsert product: %v", err)
	}
	if inventory.configureCmd.SKU != "SKU-INV" {
		t.Fatalf("expected inventory sku SKU-INV got %s", inventory.configureCmd.SKU)
	}
	if inventory.configureCmd.ProductRef != "/products/prod_inv" {
		t.Fatalf("expected product ref /products/prod_inv got %s", inventory.configureCmd.ProductRef)
	}
	if inventory.configureCmd.InitialOnHand == nil || *inventory.configureCmd.InitialOnHand != initialStock {
		t.Fatalf("expected initial stock pointer %d got %#v", initialStock, inventory.configureCmd.InitialOnHand)
	}
	if saved.ID != "prod_inv" || saved.BasePrice != 8800 {
		t.Fatalf("unexpected saved product %+v", saved)
	}
}

func TestCatalogServiceUpsertProductSKUConflict(t *testing.T) {
	repo := &stubCatalogRepository{
		materialGet:         domain.Material{MaterialSummary: domain.MaterialSummary{ID: "mat_brass"}},
		productFindBySKU:    domain.Product{ProductSummary: domain.ProductSummary{ID: "prod_existing", SKU: "SKU-dup"}},
		productFindBySKUErr: nil,
	}
	svc, err := NewCatalogService(CatalogServiceDeps{Catalog: repo})
	if err != nil {
		t.Fatalf("new catalog service: %v", err)
	}
	product := Product{
		ProductSummary: domain.ProductSummary{
			ID:                "prod_new",
			SKU:               "SKU-DUP",
			Name:              "Dup Product",
			Shape:             "round",
			SizesMm:           []int{18},
			DefaultMaterialID: "mat_brass",
			MaterialIDs:       []string{"mat_brass"},
			BasePrice:         3200,
			Currency:          "JPY",
			InventoryStatus:   "made_to_order",
		},
	}
	if _, err := svc.UpsertProduct(context.Background(), UpsertProductCommand{Product: product}); !errors.Is(err, ErrCatalogProductConflict) {
		t.Fatalf("expected sku conflict, got %v", err)
	}
}

func TestCatalogServiceDeleteProduct(t *testing.T) {
	repo := &stubCatalogRepository{
		productGet: domain.Product{
			ProductSummary: domain.ProductSummary{
				ID:          "prod_del",
				SKU:         "SKU-DEL",
				IsPublished: true,
				MaterialIDs: []string{"mat_wood"},
			},
		},
		materialGet: domain.Material{MaterialSummary: domain.MaterialSummary{ID: "mat_wood"}},
	}
	audit := &stubAuditLogService{}
	svc, err := NewCatalogService(CatalogServiceDeps{Catalog: repo, Audit: audit})
	if err != nil {
		t.Fatalf("new catalog service: %v", err)
	}
	if err := svc.DeleteProduct(context.Background(), DeleteProductCommand{ProductID: "prod_del", ActorID: "admin"}); err != nil {
		t.Fatalf("delete product: %v", err)
	}
	if repo.productUpsertInput.ProductSummary.ID != "prod_del" || repo.productUpsertInput.ProductSummary.IsPublished {
		t.Fatalf("expected product to be unpublished, got %+v", repo.productUpsertInput.ProductSummary)
	}
	foundStateChange := false
	for _, record := range audit.records {
		if record.Action == "catalog.product.unpublish" {
			foundStateChange = true
			break
		}
	}
	if !foundStateChange {
		t.Fatalf("expected unpublish audit record, got %#v", audit.records)
	}
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

	repo.getTemplate = domain.Template{}
	repo.getErr = stubRepositoryError{notFound: true}
	repo.deleteErr = stubRepositoryError{notFound: true}
	if err := svc.DeleteTemplate(context.Background(), DeleteTemplateCommand{TemplateID: "tpl_missing"}); err != nil {
		t.Fatalf("expected not found deletes to be idempotent, got %v", err)
	}

	if err := svc.DeleteTemplate(context.Background(), DeleteTemplateCommand{}); err == nil {
		t.Fatalf("expected error when id empty")
	}
}

func containsString(values []string, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == target {
			return true
		}
	}
	return false
}

func TestCatalogServiceUpsertMaterialSyncsInventoryAndAudits(t *testing.T) {
	repo := &stubCatalogRepository{}
	audit := &stubAuditLogService{}
	inventory := &stubInventorySafetyService{}
	now := time.Date(2024, time.June, 10, 12, 0, 0, 0, time.UTC)
	repo.materialUpsertResp = domain.MaterialSummary{
		ID:       "mat_wood",
		Name:     "Maple",
		Category: "wood",
		Inventory: domain.MaterialInventory{
			SKU:         "MAT-WOOD",
			SafetyStock: 5,
		},
		Procurement: domain.MaterialProcurement{SupplierRef: "sup-1"},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	svc, err := NewCatalogService(CatalogServiceDeps{
		Catalog:   repo,
		Audit:     audit,
		Inventory: inventory,
		Clock:     func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("new catalog service: %v", err)
	}
	result, err := svc.UpsertMaterial(context.Background(), UpsertMaterialCommand{
		ActorID: " admin ",
		Material: MaterialSummary{
			ID:       " mat_wood ",
			Name:     " Maple ",
			Category: " wood ",
			Inventory: MaterialInventory{
				SKU:         " mat-wood ",
				SafetyStock: 5,
			},
		},
	})
	if err != nil {
		t.Fatalf("upsert material: %v", err)
	}
	if repo.materialCreateInput.ID != "mat_wood" {
		t.Fatalf("expected create input with trimmed id mat_wood got %s", repo.materialCreateInput.ID)
	}
	if inventory.configureCmd.SKU != "mat-wood" {
		t.Fatalf("expected inventory sync to use sku mat-wood got %s", inventory.configureCmd.SKU)
	}
	if inventory.configureCmd.ProductRef != "/materials/mat_wood" {
		t.Fatalf("expected product ref /materials/mat_wood got %s", inventory.configureCmd.ProductRef)
	}
	if result.ID != "mat_wood" || result.Name != "Maple" {
		t.Fatalf("unexpected result %#v", result)
	}
	if len(audit.records) == 0 {
		t.Fatalf("expected audit records, got none")
	}
}

func TestCatalogServiceUpsertMaterialValidatesInput(t *testing.T) {
	repo := &stubCatalogRepository{}
	svc, err := NewCatalogService(CatalogServiceDeps{Catalog: repo})
	if err != nil {
		t.Fatalf("new catalog service: %v", err)
	}
	_, err = svc.UpsertMaterial(context.Background(), UpsertMaterialCommand{Material: MaterialSummary{}})
	if err == nil {
		t.Fatalf("expected validation error when id missing")
	}
	_, err = svc.UpsertMaterial(context.Background(), UpsertMaterialCommand{
		Material: MaterialSummary{
			ID:       "mat_stock",
			Name:     "Maple",
			Category: "wood",
			Inventory: MaterialInventory{
				SafetyStock: 3,
			},
		},
	})
	if err == nil {
		t.Fatalf("expected error when safety stock set without sku")
	}
}

func TestCatalogServiceUpsertMaterialCreateConflict(t *testing.T) {
	repo := &stubCatalogRepository{materialCreateErr: stubRepositoryError{conflict: true}}
	svc, err := NewCatalogService(CatalogServiceDeps{Catalog: repo})
	if err != nil {
		t.Fatalf("new catalog service: %v", err)
	}
	_, err = svc.UpsertMaterial(context.Background(), UpsertMaterialCommand{Material: MaterialSummary{ID: "mat_wood", Name: "Maple", Category: "wood"}})
	if err == nil || !errors.Is(err, ErrCatalogMaterialConflict) {
		t.Fatalf("expected material conflict error, got %v", err)
	}
}

func TestCatalogServiceDeleteMaterialRecordsAudit(t *testing.T) {
	repo := &stubCatalogRepository{
		materialGet: domain.Material{MaterialSummary: domain.MaterialSummary{ID: "mat_wood", Name: "Maple", Category: "wood", IsAvailable: true}},
		productListPages: map[string]domain.CursorPage[domain.ProductSummary]{
			"":     {Items: []domain.ProductSummary{{ID: "prod-1"}, {ID: "prod-2"}}, NextPageToken: "next"},
			"next": {Items: []domain.ProductSummary{{ID: "prod-3"}}},
		},
	}
	audit := &stubAuditLogService{}
	now := time.Date(2024, time.May, 1, 8, 0, 0, 0, time.UTC)
	svc, err := NewCatalogService(CatalogServiceDeps{Catalog: repo, Audit: audit, Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("new catalog service: %v", err)
	}
	if err := svc.DeleteMaterial(context.Background(), DeleteMaterialCommand{MaterialID: "mat_wood", ActorID: "admin"}); err != nil {
		t.Fatalf("delete material: %v", err)
	}
	if repo.materialDeleteID != "mat_wood" {
		t.Fatalf("expected repo delete id mat_wood got %s", repo.materialDeleteID)
	}
	if len(audit.records) == 0 {
		t.Fatalf("expected audit records on delete")
	}
	found := false
	for _, record := range audit.records {
		if record.Action == "catalog.material.products.flagged" {
			found = true
			total, _ := record.Metadata["totalProducts"].(int)
			if total != 3 {
				t.Fatalf("expected totalProducts 3 got %v", total)
			}
			products, _ := record.Metadata["products"].([]string)
			if len(products) != 3 {
				t.Fatalf("expected 3 sampled products got %d", len(products))
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected product flagged audit record")
	}
}

type stubCatalogRepository struct {
	listFilter          repositories.TemplateFilter
	listResp            domain.CursorPage[domain.TemplateSummary]
	listErr             error
	fontListFilter      repositories.FontFilter
	fontListResp        domain.CursorPage[domain.FontSummary]
	fontListErr         error
	materialListFilter  repositories.MaterialFilter
	materialListResp    domain.CursorPage[domain.MaterialSummary]
	materialListErr     error
	productListFilter   repositories.ProductFilter
	productListResp     domain.CursorPage[domain.ProductSummary]
	productListPages    map[string]domain.CursorPage[domain.ProductSummary]
	productListErr      error
	productFindBySKU    domain.Product
	productFindBySKUErr error
	productFindSKU      string
	productUpsertInput  domain.Product
	productUpsertResp   domain.Product
	productUpsertErr    error
	productDeleteID     string
	productDeleteErr    error

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

	getID               string
	getTemplate         domain.Template
	getErr              error
	fontGetID           string
	fontGet             domain.Font
	fontGetErr          error
	fontUpsertInput     domain.FontSummary
	fontUpsertResp      domain.FontSummary
	fontUpsertErr       error
	fontDeleteID        string
	fontDeleteErr       error
	materialGetID       string
	materialGet         domain.Material
	materialGetErr      error
	materialCreateInput domain.MaterialSummary
	materialCreateResp  domain.MaterialSummary
	materialCreateErr   error
	materialUpsertInput domain.MaterialSummary
	materialUpsertResp  domain.MaterialSummary
	materialUpsertErr   error
	materialDeleteID    string
	materialDeleteErr   error
	productGetID        string
	productGet          domain.Product
	productGetErr       error

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

func (s *stubCatalogRepository) UpsertFont(_ context.Context, font domain.FontSummary) (domain.FontSummary, error) {
	s.fontUpsertInput = font
	if s.fontUpsertErr != nil {
		return domain.FontSummary{}, s.fontUpsertErr
	}
	if s.fontUpsertResp.ID != "" {
		return s.fontUpsertResp, nil
	}
	return font, nil
}

func (s *stubCatalogRepository) DeleteFont(_ context.Context, fontID string) error {
	s.fontDeleteID = fontID
	return s.fontDeleteErr
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

func (s *stubCatalogRepository) CreateMaterial(_ context.Context, material domain.MaterialSummary) (domain.MaterialSummary, error) {
	s.materialCreateInput = material
	if s.materialCreateErr != nil {
		return domain.MaterialSummary{}, s.materialCreateErr
	}
	if s.materialCreateResp.ID != "" {
		return s.materialCreateResp, nil
	}
	return material, nil
}

func (s *stubCatalogRepository) UpsertMaterial(_ context.Context, material domain.MaterialSummary) (domain.MaterialSummary, error) {
	s.materialUpsertInput = material
	if s.materialUpsertErr != nil {
		return domain.MaterialSummary{}, s.materialUpsertErr
	}
	if s.materialUpsertResp.ID != "" {
		return s.materialUpsertResp, nil
	}
	return material, nil
}

func (s *stubCatalogRepository) DeleteMaterial(_ context.Context, materialID string) error {
	s.materialDeleteID = materialID
	return s.materialDeleteErr
}

func (s *stubCatalogRepository) ListProducts(_ context.Context, filter repositories.ProductFilter) (domain.CursorPage[domain.ProductSummary], error) {
	s.productListFilter = filter
	if len(s.productListPages) > 0 {
		page, ok := s.productListPages[strings.TrimSpace(filter.Pagination.PageToken)]
		if ok {
			return page, s.productListErr
		}
	}
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

func (s *stubCatalogRepository) FindProductBySKU(_ context.Context, sku string) (domain.Product, error) {
	s.productFindSKU = sku
	if s.productFindBySKUErr != nil {
		return domain.Product{}, s.productFindBySKUErr
	}
	if s.productFindBySKU.ID != "" {
		return s.productFindBySKU, nil
	}
	return domain.Product{}, stubRepositoryError{notFound: true}
}

func (s *stubCatalogRepository) UpsertProduct(_ context.Context, product domain.Product) (domain.Product, error) {
	s.productUpsertInput = product
	if s.productUpsertErr != nil {
		return domain.Product{}, s.productUpsertErr
	}
	if s.productUpsertResp.ID != "" {
		return s.productUpsertResp, nil
	}
	return product, nil
}

func (s *stubCatalogRepository) DeleteProduct(_ context.Context, productID string) error {
	s.productDeleteID = productID
	return s.productDeleteErr
}

type stubRepositoryError struct {
	notFound    bool
	conflict    bool
	unavailable bool
}

func (e stubRepositoryError) Error() string { return "catalog repository error" }

func (e stubRepositoryError) IsNotFound() bool    { return e.notFound }
func (e stubRepositoryError) IsConflict() bool    { return e.conflict }
func (e stubRepositoryError) IsUnavailable() bool { return e.unavailable }

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

type stubInventorySafetyService struct {
	configureCmd ConfigureSafetyStockCommand
	configureErr error
}

func (s *stubInventorySafetyService) ReserveStocks(context.Context, InventoryReserveCommand) (InventoryReservation, error) {
	return InventoryReservation{}, errors.New("not implemented")
}

func (s *stubInventorySafetyService) CommitReservation(context.Context, InventoryCommitCommand) (InventoryReservation, error) {
	return InventoryReservation{}, errors.New("not implemented")
}

func (s *stubInventorySafetyService) ReleaseReservation(context.Context, InventoryReleaseCommand) (InventoryReservation, error) {
	return InventoryReservation{}, errors.New("not implemented")
}

func (s *stubInventorySafetyService) ListLowStock(context.Context, InventoryLowStockFilter) (domain.CursorPage[InventorySnapshot], error) {
	return domain.CursorPage[InventorySnapshot]{}, nil
}

func (s *stubInventorySafetyService) ConfigureSafetyStock(_ context.Context, cmd ConfigureSafetyStockCommand) (InventoryStock, error) {
	s.configureCmd = cmd
	if s.configureErr != nil {
		return InventoryStock{}, s.configureErr
	}
	stock := InventoryStock{SKU: cmd.SKU, SafetyStock: cmd.SafetyStock}
	if cmd.InitialOnHand != nil {
		stock.OnHand = *cmd.InitialOnHand
	}
	return stock, nil
}
