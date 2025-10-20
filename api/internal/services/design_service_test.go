package services

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

type stubDesignRepository struct {
	inserted []domain.Design
}

func (s *stubDesignRepository) Insert(_ context.Context, design domain.Design) error {
	s.inserted = append(s.inserted, design)
	return nil
}

func (s *stubDesignRepository) Update(context.Context, domain.Design) error {
	return errors.New("not implemented")
}

func (s *stubDesignRepository) SoftDelete(context.Context, string, time.Time) error {
	return errors.New("not implemented")
}

func (s *stubDesignRepository) FindByID(context.Context, string) (domain.Design, error) {
	return domain.Design{}, errors.New("not implemented")
}

func (s *stubDesignRepository) ListByOwner(context.Context, string, repositories.DesignListFilter) (domain.CursorPage[domain.Design], error) {
	return domain.CursorPage[domain.Design]{}, errors.New("not implemented")
}

type stubDesignVersionRepository struct {
	appended []domain.DesignVersion
}

func (s *stubDesignVersionRepository) Append(_ context.Context, version domain.DesignVersion) error {
	s.appended = append(s.appended, version)
	return nil
}

func (s *stubDesignVersionRepository) ListByDesign(context.Context, string, domain.Pagination) (domain.CursorPage[domain.DesignVersion], error) {
	return domain.CursorPage[domain.DesignVersion]{}, errors.New("not implemented")
}

func TestDesignService_CreateDesignTyped(t *testing.T) {
	repo := &stubDesignRepository{}
	versions := &stubDesignVersionRepository{}

	fixed := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	idSeq := []string{"ID001", "ID002", "ID003"}

	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      repo,
		Versions:     versions,
		AssetsBucket: "bucket",
		Clock:        func() time.Time { return fixed },
		IDGenerator: func() string {
			if len(idSeq) == 0 {
				return "id"
			}
			id := idSeq[0]
			idSeq = idSeq[1:]
			return id
		},
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	cmd := CreateDesignCommand{
		OwnerID:    "user-123",
		ActorID:    "user-123",
		Label:      " My Seal ",
		Type:       DesignTypeTyped,
		TextLines:  []string{" 太郎 ", "花子"},
		FontID:     "font-abc",
		MaterialID: "material-xyz",
		TemplateID: "tmpl-1",
		Locale:     "ja-JP",
		Shape:      "round",
		SizeMM:     18.0,
	}

	design, err := svc.CreateDesign(context.Background(), cmd)
	if err != nil {
		t.Fatalf("CreateDesign error: %v", err)
	}

	if design.ID != "dsg_id001" {
		t.Fatalf("unexpected design id: %s", design.ID)
	}
	if design.CurrentVersionID != "ver_id002" {
		t.Fatalf("unexpected version id: %s", design.CurrentVersionID)
	}
	if design.Version != 1 {
		t.Fatalf("expected version 1, got %d", design.Version)
	}
	expectedPreview := "assets/designs/dsg_id001/previews/ver_id002/preview.png"
	if design.Assets.PreviewPath != expectedPreview {
		t.Fatalf("unexpected preview path: %s", design.Assets.PreviewPath)
	}
	expectedVector := "assets/designs/dsg_id001/sources/render-ver_id002/design.svg"
	if design.Assets.VectorPath != expectedVector {
		t.Fatalf("unexpected vector path: %s", design.Assets.VectorPath)
	}
	if design.Assets.SourcePath != expectedVector {
		t.Fatalf("expected source path to equal vector path, got %s", design.Assets.SourcePath)
	}
	expectedURL := "https://storage.googleapis.com/bucket/" + expectedPreview
	if design.Assets.PreviewURL != expectedURL {
		t.Fatalf("unexpected preview url: %s", design.Assets.PreviewURL)
	}
	if got, want := len(design.TextLines), 2; got != want {
		t.Fatalf("expected %d text lines, got %d", want, got)
	}
	if design.TextLines[0] != "太郎" {
		t.Fatalf("text line not trimmed: %v", design.TextLines)
	}
	if !design.CreatedAt.Equal(fixed) || !design.UpdatedAt.Equal(fixed) {
		t.Fatalf("unexpected timestamps: %v / %v", design.CreatedAt, design.UpdatedAt)
	}

	if len(repo.inserted) != 1 {
		t.Fatalf("expected 1 inserted design, got %d", len(repo.inserted))
	}
	if repo.inserted[0].ID != design.ID {
		t.Fatalf("repository stored different design id: %s", repo.inserted[0].ID)
	}
	if len(versions.appended) != 1 {
		t.Fatalf("expected 1 appended version, got %d", len(versions.appended))
	}
	if versions.appended[0].DesignID != design.ID {
		t.Fatalf("version design id mismatch: %s", versions.appended[0].DesignID)
	}
	if versions.appended[0].Version != 1 {
		t.Fatalf("expected version number 1, got %d", versions.appended[0].Version)
	}
	if versions.appended[0].Snapshot["type"] != string(DesignTypeTyped) {
		t.Fatalf("expected snapshot type typed, got %v", versions.appended[0].Snapshot["type"])
	}
}

func TestDesignService_CreateDesignUploaded(t *testing.T) {
	repo := &stubDesignRepository{}
	versions := &stubDesignVersionRepository{}

	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      repo,
		Versions:     versions,
		AssetsBucket: "bucket",
		Clock:        func() time.Time { return time.Unix(0, 0).UTC() },
		IDGenerator: func() string {
			return "asset"
		},
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	cmd := CreateDesignCommand{
		OwnerID: "user-1",
		ActorID: "user-1",
		Type:    DesignTypeUploaded,
		Upload: &DesignAssetInput{
			AssetID:     "upload-1",
			FileName:    "source.png",
			ContentType: "image/png",
			SizeBytes:   2048,
		},
	}

	design, err := svc.CreateDesign(context.Background(), cmd)
	if err != nil {
		t.Fatalf("CreateDesign error: %v", err)
	}

	expectedSource := "assets/designs/dsg_asset/sources/upload-1/source.png"
	if design.Assets.SourcePath != expectedSource {
		t.Fatalf("unexpected source path: %s", design.Assets.SourcePath)
	}
	if design.Source.UploadAsset == nil {
		t.Fatalf("expected upload asset metadata")
	}
	if design.Source.UploadAsset.ObjectPath != expectedSource {
		t.Fatalf("unexpected upload asset object path: %s", design.Source.UploadAsset.ObjectPath)
	}
	if design.Assets.VectorPath != "" {
		t.Fatalf("expected empty vector path for uploaded design")
	}
}

func TestDesignService_CreateDesignInvalidInput(t *testing.T) {
	repo := &stubDesignRepository{}
	versions := &stubDesignVersionRepository{}

	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      repo,
		Versions:     versions,
		AssetsBucket: "bucket",
		Clock:        time.Now,
		IDGenerator:  func() string { return "x" },
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	_, err = svc.CreateDesign(context.Background(), CreateDesignCommand{
		OwnerID:   "user-1",
		ActorID:   "user-1",
		Type:      DesignTypeTyped,
		TextLines: []string{"禁止文字死"},
		FontID:    "font-1",
	})
	if !errors.Is(err, ErrDesignInvalidInput) {
		t.Fatalf("expected ErrDesignInvalidInput, got %v", err)
	}
	if len(repo.inserted) != 0 {
		t.Fatalf("expected no designs inserted on validation failure")
	}
}
