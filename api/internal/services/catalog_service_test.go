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

func TestCatalogServiceGetTemplate(t *testing.T) {
	stubRepo := &stubCatalogRepository{
		getTemplate: domain.Template{
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
		if stubRepo.getID != "tpl_001" {
			t.Fatalf("expected repository to receive trimmed id tpl_001, got %q", stubRepo.getID)
		}
	})
}

func TestCatalogServiceDeleteTemplate(t *testing.T) {
	stubRepo := &stubCatalogRepository{}
	svc, err := NewCatalogService(CatalogServiceDeps{Catalog: stubRepo})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := svc.DeleteTemplate(context.Background(), " tpl_001 "); err != nil {
		t.Fatalf("unexpected error deleting: %v", err)
	}
	if stubRepo.deletedID != "tpl_001" {
		t.Fatalf("expected trimmed id, got %q", stubRepo.deletedID)
	}

	if err := svc.DeleteTemplate(context.Background(), " "); err == nil {
		t.Fatalf("expected error when id empty")
	}
}

type stubCatalogRepository struct {
	listFilter repositories.TemplateFilter
	listResp   domain.CursorPage[domain.TemplateSummary]
	listErr    error

	getID       string
	getTemplate domain.Template
	getErr      error

	upsertInput  domain.Template
	upsertResult domain.Template
	upsertErr    error

	deletedID string
	deleteErr error
}

func (s *stubCatalogRepository) ListTemplates(_ context.Context, filter repositories.TemplateFilter) (domain.CursorPage[domain.TemplateSummary], error) {
	s.listFilter = filter
	return s.listResp, s.listErr
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

func (s *stubCatalogRepository) ListFonts(context.Context, repositories.FontFilter) (domain.CursorPage[domain.FontSummary], error) {
	return domain.CursorPage[domain.FontSummary]{}, errors.New("not implemented")
}

func (s *stubCatalogRepository) UpsertFont(context.Context, domain.FontSummary) (domain.FontSummary, error) {
	return domain.FontSummary{}, errors.New("not implemented")
}

func (s *stubCatalogRepository) DeleteFont(context.Context, string) error {
	return errors.New("not implemented")
}

func (s *stubCatalogRepository) ListMaterials(context.Context, repositories.MaterialFilter) (domain.CursorPage[domain.MaterialSummary], error) {
	return domain.CursorPage[domain.MaterialSummary]{}, errors.New("not implemented")
}

func (s *stubCatalogRepository) UpsertMaterial(context.Context, domain.MaterialSummary) (domain.MaterialSummary, error) {
	return domain.MaterialSummary{}, errors.New("not implemented")
}

func (s *stubCatalogRepository) DeleteMaterial(context.Context, string) error {
	return errors.New("not implemented")
}

func (s *stubCatalogRepository) ListProducts(context.Context, repositories.ProductFilter) (domain.CursorPage[domain.ProductSummary], error) {
	return domain.CursorPage[domain.ProductSummary]{}, errors.New("not implemented")
}

func (s *stubCatalogRepository) UpsertProduct(context.Context, domain.ProductSummary) (domain.ProductSummary, error) {
	return domain.ProductSummary{}, errors.New("not implemented")
}

func (s *stubCatalogRepository) DeleteProduct(context.Context, string) error {
	return errors.New("not implemented")
}
