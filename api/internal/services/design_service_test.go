package services

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

type stubDesignRepository struct {
	inserted    []domain.Design
	updated     []domain.Design
	softDeleted []string
	store       map[string]domain.Design
}

func (s *stubDesignRepository) Insert(_ context.Context, design domain.Design) error {
	if s.store == nil {
		s.store = make(map[string]domain.Design)
	}
	s.inserted = append(s.inserted, design)
	s.store[design.ID] = cloneTestDesign(design)
	return nil
}

func (s *stubDesignRepository) Update(_ context.Context, design domain.Design) error {
	if s.store == nil {
		return errors.New("not found")
	}
	if _, ok := s.store[design.ID]; !ok {
		return errors.New("not found")
	}
	s.updated = append(s.updated, design)
	s.store[design.ID] = cloneTestDesign(design)
	return nil
}

func (s *stubDesignRepository) SoftDelete(_ context.Context, designID string, deletedAt time.Time) error {
	if s.store == nil {
		return errors.New("not found")
	}
	design, ok := s.store[designID]
	if !ok {
		return errors.New("not found")
	}
	design.Status = domain.DesignStatusDeleted
	design.UpdatedAt = deletedAt.UTC()
	s.store[designID] = design
	s.softDeleted = append(s.softDeleted, designID)
	return nil
}

func (s *stubDesignRepository) FindByID(_ context.Context, designID string) (domain.Design, error) {
	if s.store == nil {
		return domain.Design{}, errors.New("not found")
	}
	design, ok := s.store[designID]
	if !ok {
		return domain.Design{}, errors.New("not found")
	}
	return cloneTestDesign(design), nil
}

func (s *stubDesignRepository) ListByOwner(_ context.Context, ownerID string, _ repositories.DesignListFilter) (domain.CursorPage[domain.Design], error) {
	items := make([]domain.Design, 0)
	for _, design := range s.store {
		if design.OwnerID == ownerID {
			items = append(items, cloneTestDesign(design))
		}
	}
	return domain.CursorPage[domain.Design]{Items: items}, nil
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

func TestDesignService_CreateDesignUploadedKeepsObjectPath(t *testing.T) {
	repo := &stubDesignRepository{}
	versions := &stubDesignVersionRepository{}

	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      repo,
		Versions:     versions,
		AssetsBucket: "bucket-default",
		Clock:        func() time.Time { return time.Unix(0, 0).UTC() },
		IDGenerator:  func() string { return "seq" },
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	cmd := CreateDesignCommand{
		OwnerID: "user-1",
		ActorID: "user-1",
		Type:    DesignTypeUploaded,
		Upload: &DesignAssetInput{
			AssetID:     "upload-123",
			Bucket:      "custom-bucket",
			ObjectPath:  "uploads/user-1/upload-123/source.png",
			FileName:    "source.png",
			ContentType: "image/png",
			SizeBytes:   1024,
		},
	}

	design, err := svc.CreateDesign(context.Background(), cmd)
	if err != nil {
		t.Fatalf("CreateDesign error: %v", err)
	}

	if design.Assets.SourcePath != "uploads/user-1/upload-123/source.png" {
		t.Fatalf("expected original object path preserved, got %s", design.Assets.SourcePath)
	}
	if design.Source.UploadAsset == nil || design.Source.UploadAsset.Bucket != "custom-bucket" {
		t.Fatalf("expected bucket metadata preserved, got %+v", design.Source.UploadAsset)
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

func TestDesignService_UpdateDesign_Success(t *testing.T) {
	updatedAt := time.Date(2025, 3, 10, 12, 0, 0, 0, time.UTC)
	repo := &stubDesignRepository{
		store: map[string]domain.Design{
			"dsg_001": {
				ID:               "dsg_001",
				OwnerID:          "user-1",
				Label:            "Original",
				Type:             domain.DesignTypeTyped,
				Status:           domain.DesignStatusDraft,
				Version:          1,
				CurrentVersionID: "ver_old",
				Snapshot:         map[string]any{"label": "Original"},
				UpdatedAt:        updatedAt,
			},
		},
	}
	versions := &stubDesignVersionRepository{}
	nextVersionID := "VERSION2"
	now := time.Date(2025, 3, 11, 9, 30, 0, 0, time.UTC)

	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      repo,
		Versions:     versions,
		AssetsBucket: "bucket",
		Clock:        func() time.Time { return now },
		IDGenerator: func() string {
			id := nextVersionID
			nextVersionID = "unused"
			return id
		},
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	label := "Updated"
	status := "ready"
	snapshot := map[string]any{"label": "Updated"}
	expected := updatedAt

	design, err := svc.UpdateDesign(context.Background(), UpdateDesignCommand{
		DesignID:          "dsg_001",
		UpdatedBy:         "user-1",
		Label:             &label,
		Status:            &status,
		Snapshot:          snapshot,
		ExpectedUpdatedAt: &expected,
	})
	if err != nil {
		t.Fatalf("UpdateDesign error: %v", err)
	}
	if design.Label != "Updated" {
		t.Fatalf("expected label Updated, got %s", design.Label)
	}
	if design.Status != DesignStatusReady {
		t.Fatalf("expected status ready, got %s", design.Status)
	}
	if design.Version != 2 {
		t.Fatalf("expected version 2, got %d", design.Version)
	}
	if design.CurrentVersionID != "ver_version2" {
		t.Fatalf("unexpected current version id: %s", design.CurrentVersionID)
	}
	if !design.UpdatedAt.Equal(now) {
		t.Fatalf("expected updatedAt %v, got %v", now, design.UpdatedAt)
	}
	if len(repo.updated) != 1 {
		t.Fatalf("expected repository update captured")
	}
	if len(versions.appended) != 1 || versions.appended[0].DesignID != "dsg_001" {
		t.Fatalf("expected version append recorded")
	}
	if versions.appended[0].Version != 2 {
		t.Fatalf("expected appended version 2, got %d", versions.appended[0].Version)
	}
}

func TestDesignService_UpdateDesign_Conflict(t *testing.T) {
	repo := &stubDesignRepository{
		store: map[string]domain.Design{
			"dsg_001": {
				ID:        "dsg_001",
				OwnerID:   "user-1",
				Label:     "Original",
				Status:    domain.DesignStatusDraft,
				Version:   1,
				UpdatedAt: time.Date(2025, 3, 10, 12, 0, 0, 0, time.UTC),
			},
		},
	}
	versions := &stubDesignVersionRepository{}
	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      repo,
		Versions:     versions,
		AssetsBucket: "bucket",
		Clock:        time.Now,
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	expected := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	status := "ready"

	_, err = svc.UpdateDesign(context.Background(), UpdateDesignCommand{
		DesignID:          "dsg_001",
		UpdatedBy:         "user-1",
		Status:            &status,
		ExpectedUpdatedAt: &expected,
	})
	if !errors.Is(err, ErrDesignConflict) {
		t.Fatalf("expected design conflict, got %v", err)
	}
}

func TestDesignService_DeleteDesign_Soft(t *testing.T) {
	updatedAt := time.Date(2025, 3, 10, 12, 0, 0, 0, time.UTC)
	repo := &stubDesignRepository{
		store: map[string]domain.Design{
			"dsg_001": {
				ID:        "dsg_001",
				OwnerID:   "user-1",
				Status:    domain.DesignStatusDraft,
				UpdatedAt: updatedAt,
			},
		},
	}
	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      repo,
		Versions:     &stubDesignVersionRepository{},
		AssetsBucket: "bucket",
		Clock: func() time.Time {
			return time.Date(2025, 3, 11, 8, 0, 0, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	expected := updatedAt
	err = svc.DeleteDesign(context.Background(), DeleteDesignCommand{
		DesignID:          "dsg_001",
		RequestedBy:       "user-1",
		SoftDelete:        true,
		ExpectedUpdatedAt: &expected,
	})
	if err != nil {
		t.Fatalf("DeleteDesign error: %v", err)
	}
	if len(repo.softDeleted) != 1 || repo.softDeleted[0] != "dsg_001" {
		t.Fatalf("expected soft delete recorded")
	}
	record, ok := repo.store["dsg_001"]
	if !ok || record.Status != domain.DesignStatusDeleted {
		t.Fatalf("expected design status deleted, got %v", record.Status)
	}
}

func cloneTestDesign(design domain.Design) domain.Design {
	copy := design
	if design.Snapshot != nil {
		copy.Snapshot = maps.Clone(design.Snapshot)
	}
	if len(design.TextLines) > 0 {
		copy.TextLines = slices.Clone(design.TextLines)
	}
	if len(design.Versions) > 0 {
		copy.Versions = make([]domain.DesignVersion, len(design.Versions))
		for i, version := range design.Versions {
			copy.Versions[i] = version
			if version.Snapshot != nil {
				copy.Versions[i].Snapshot = maps.Clone(version.Snapshot)
			}
		}
	}
	return copy
}
