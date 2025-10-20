package services

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"sync"
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
	listFn   func(context.Context, string, domain.Pagination) (domain.CursorPage[domain.DesignVersion], error)
	findFn   func(context.Context, string, string) (domain.DesignVersion, error)
}

func (s *stubDesignVersionRepository) Append(_ context.Context, version domain.DesignVersion) error {
	s.appended = append(s.appended, version)
	return nil
}

func (s *stubDesignVersionRepository) ListByDesign(ctx context.Context, designID string, pager domain.Pagination) (domain.CursorPage[domain.DesignVersion], error) {
	if s.listFn != nil {
		return s.listFn(ctx, designID, pager)
	}
	return domain.CursorPage[domain.DesignVersion]{}, errors.New("not implemented")
}

func (s *stubDesignVersionRepository) FindByID(ctx context.Context, designID, versionID string) (domain.DesignVersion, error) {
	if s.findFn != nil {
		return s.findFn(ctx, designID, versionID)
	}
	return domain.DesignVersion{}, errors.New("not implemented")
}

type assetCopyCall struct {
	SourceBucket string
	SourceObject string
	DestBucket   string
	DestObject   string
}

type stubAssetCopier struct {
	mu    sync.Mutex
	calls []assetCopyCall
	ch    chan assetCopyCall
}

func newStubAssetCopier(buffer int) *stubAssetCopier {
	if buffer <= 0 {
		buffer = 4
	}
	return &stubAssetCopier{
		ch: make(chan assetCopyCall, buffer),
	}
}

func (s *stubAssetCopier) CopyObject(ctx context.Context, sourceBucket, sourceObject, destBucket, destObject string) error {
	call := assetCopyCall{
		SourceBucket: strings.TrimSpace(sourceBucket),
		SourceObject: strings.TrimSpace(sourceObject),
		DestBucket:   strings.TrimSpace(destBucket),
		DestObject:   strings.TrimSpace(destObject),
	}
	s.mu.Lock()
	s.calls = append(s.calls, call)
	s.mu.Unlock()
	s.ch <- call
	return nil
}

func (s *stubAssetCopier) waitForCalls(t *testing.T, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		select {
		case <-s.ch:
		case <-time.After(200 * time.Millisecond):
			t.Fatalf("timed out waiting for asset copier calls; expected %d, observed %d", n, len(s.Calls()))
		}
	}
	// Give goroutines a moment to finish appending to the calls slice.
	time.Sleep(10 * time.Millisecond)
}

func (s *stubAssetCopier) Calls() []assetCopyCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]assetCopyCall, len(s.calls))
	copy(out, s.calls)
	return out
}

type capturingAuditService struct {
	mu      sync.Mutex
	records []AuditLogRecord
}

func (s *capturingAuditService) Record(_ context.Context, record AuditLogRecord) {
	s.mu.Lock()
	s.records = append(s.records, record)
	s.mu.Unlock()
}

func (s *capturingAuditService) List(context.Context, AuditLogFilter) (domain.CursorPage[domain.AuditLogEntry], error) {
	return domain.CursorPage[domain.AuditLogEntry]{}, errors.New("not implemented")
}

func (s *capturingAuditService) Records() []AuditLogRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]AuditLogRecord, len(s.records))
	copy(out, s.records)
	return out
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

func TestDesignService_ListDesignVersions_StripsAssetsWhenExcluded(t *testing.T) {
	versions := &stubDesignVersionRepository{
		listFn: func(context.Context, string, domain.Pagination) (domain.CursorPage[domain.DesignVersion], error) {
			return domain.CursorPage[domain.DesignVersion]{
				Items: []domain.DesignVersion{
					{
						ID:       "ver_1",
						DesignID: "dsg_1",
						Version:  1,
						Snapshot: map[string]any{
							"label":  "Initial",
							"assets": map[string]any{"previewUrl": "https://cdn/ver_1.png"},
						},
					},
				},
			}, nil
		},
	}

	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      &stubDesignRepository{},
		Versions:     versions,
		AssetsBucket: "bucket",
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	page, err := svc.ListDesignVersions(context.Background(), "dsg_1", DesignVersionListFilter{
		Pagination:    Pagination{PageSize: 5},
		IncludeAssets: false,
	})
	if err != nil {
		t.Fatalf("ListDesignVersions error: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(page.Items))
	}
	if _, ok := page.Items[0].Snapshot["assets"]; ok {
		t.Fatalf("expected assets stripped from snapshot")
	}
}

func TestDesignService_ListDesignVersions_KeepsAssetsWhenIncluded(t *testing.T) {
	versions := &stubDesignVersionRepository{
		listFn: func(context.Context, string, domain.Pagination) (domain.CursorPage[domain.DesignVersion], error) {
			return domain.CursorPage[domain.DesignVersion]{
				Items: []domain.DesignVersion{
					{
						ID:       "ver_2",
						DesignID: "dsg_1",
						Version:  2,
						Snapshot: map[string]any{
							"label":  "Updated",
							"assets": map[string]any{"previewUrl": "https://cdn/ver_2.png"},
						},
					},
				},
			}, nil
		},
	}

	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      &stubDesignRepository{},
		Versions:     versions,
		AssetsBucket: "bucket",
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	page, err := svc.ListDesignVersions(context.Background(), "dsg_1", DesignVersionListFilter{
		IncludeAssets: true,
	})
	if err != nil {
		t.Fatalf("ListDesignVersions error: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(page.Items))
	}
	if _, ok := page.Items[0].Snapshot["assets"]; !ok {
		t.Fatalf("expected assets retained in snapshot")
	}
}

func TestDesignService_GetDesignVersion_StripsAssetsWhenExcluded(t *testing.T) {
	versions := &stubDesignVersionRepository{
		findFn: func(context.Context, string, string) (domain.DesignVersion, error) {
			return domain.DesignVersion{
				ID:       "ver_1",
				DesignID: "dsg_1",
				Version:  1,
				Snapshot: map[string]any{
					"label":  "Initial",
					"assets": map[string]any{"previewUrl": "https://cdn/ver_1.png"},
				},
			}, nil
		},
	}

	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      &stubDesignRepository{},
		Versions:     versions,
		AssetsBucket: "bucket",
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	version, err := svc.GetDesignVersion(context.Background(), "dsg_1", "ver_1", DesignVersionReadOptions{})
	if err != nil {
		t.Fatalf("GetDesignVersion error: %v", err)
	}
	if _, ok := version.Snapshot["assets"]; ok {
		t.Fatalf("expected assets removed when IncludeAssets=false")
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

func TestDesignService_DuplicateDesign_Success(t *testing.T) {
	fixed := time.Date(2025, 4, 10, 15, 30, 0, 0, time.UTC)
	previewPath := "assets/designs/dsg_src/previews/ver_src/preview.png"
	vectorPath := "assets/designs/dsg_src/sources/render-ver_src/design.svg"

	source := domain.Design{
		ID:         "dsg_src",
		OwnerID:    "user-1",
		Label:      "Original",
		Type:       domain.DesignTypeTyped,
		TextLines:  []string{"Original"},
		FontID:     "font-1",
		MaterialID: "material-1",
		Template:   "tmpl-1",
		Locale:     "ja-JP",
		Shape:      "round",
		SizeMM:     15,
		Source: domain.DesignSource{
			Type:      domain.DesignTypeTyped,
			RawName:   "Original",
			TextLines: []string{"Original"},
			UploadAsset: &domain.DesignAssetReference{
				AssetID:     "render-ver_src",
				Bucket:      "bucket",
				ObjectPath:  vectorPath,
				FileName:    "design.svg",
				ContentType: "image/svg+xml",
				SizeBytes:   2048,
				Checksum:    "checksum-src",
			},
		},
		Assets: domain.DesignAssets{
			SourcePath:  vectorPath,
			VectorPath:  vectorPath,
			PreviewPath: previewPath,
			PreviewURL:  "https://storage.googleapis.com/bucket/" + previewPath,
		},
		Status:           domain.DesignStatusReady,
		ThumbnailURL:     "https://storage.googleapis.com/bucket/" + previewPath,
		Version:          3,
		CurrentVersionID: "ver_src",
		Snapshot: map[string]any{
			"label":  "Original",
			"status": "ready",
			"assets": map[string]any{
				"previewPath": previewPath,
				"previewUrl":  "https://storage.googleapis.com/bucket/" + previewPath,
				"sourcePath":  vectorPath,
				"vectorPath":  vectorPath,
			},
			"source": map[string]any{
				"type":      string(domain.DesignTypeTyped),
				"rawName":   "Original",
				"textLines": []string{"Original"},
				"uploadAsset": map[string]any{
					"assetId":     "render-ver_src",
					"bucket":      "bucket",
					"objectPath":  vectorPath,
					"fileName":    "design.svg",
					"contentType": "image/svg+xml",
					"sizeBytes":   int64(2048),
					"checksum":    "checksum-src",
				},
			},
		},
		CreatedAt: fixed.Add(-24 * time.Hour),
		UpdatedAt: fixed,
	}

	repo := &stubDesignRepository{
		store: map[string]domain.Design{
			source.ID: cloneTestDesign(source),
		},
	}
	versions := &stubDesignVersionRepository{}
	idSeq := []string{"ID100", "ID101", "ID102"}

	copier := newStubAssetCopier(6)
	audit := &capturingAuditService{}

	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      repo,
		Versions:     versions,
		AssetsBucket: "bucket",
		AssetCopier:  copier,
		Audit:        audit,
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

	override := " My Copy "
	duplicate, err := svc.DuplicateDesign(context.Background(), DuplicateDesignCommand{
		SourceDesignID: source.ID,
		RequestedBy:    "user-1",
		OverrideName:   &override,
	})
	if err != nil {
		t.Fatalf("DuplicateDesign error: %v", err)
	}

	copier.waitForCalls(t, 2)
	calls := copier.Calls()
	if len(calls) != 2 {
		t.Fatalf("expected 2 asset copies, got %d", len(calls))
	}
	expectedDestinations := map[string]struct{}{
		"assets/designs/dsg_id100/previews/ver_id101/preview.png":      {},
		"assets/designs/dsg_id100/sources/render-ver_id101/design.svg": {},
	}
	for _, call := range calls {
		if call.DestBucket != "bucket" {
			t.Fatalf("unexpected destination bucket: %s", call.DestBucket)
		}
		if _, ok := expectedDestinations[call.DestObject]; !ok {
			t.Fatalf("unexpected destination object: %s", call.DestObject)
		}
	}

	if duplicate.ID != "dsg_id100" {
		t.Fatalf("unexpected duplicate id: %s", duplicate.ID)
	}
	if duplicate.Label != "My Copy" {
		t.Fatalf("expected label override to be applied, got %s", duplicate.Label)
	}
	if duplicate.Status != DesignStatusDraft {
		t.Fatalf("expected draft status, got %s", duplicate.Status)
	}
	if duplicate.Version != 1 {
		t.Fatalf("expected version 1, got %d", duplicate.Version)
	}
	if duplicate.CurrentVersionID != "ver_id101" {
		t.Fatalf("unexpected current version id: %s", duplicate.CurrentVersionID)
	}
	expectedPreviewPath := "assets/designs/dsg_id100/previews/ver_id101/preview.png"
	if duplicate.Assets.PreviewPath != expectedPreviewPath {
		t.Fatalf("unexpected preview path: %s", duplicate.Assets.PreviewPath)
	}
	expectedVectorPath := "assets/designs/dsg_id100/sources/render-ver_id101/design.svg"
	if duplicate.Assets.SourcePath != expectedVectorPath {
		t.Fatalf("unexpected source path: %s", duplicate.Assets.SourcePath)
	}
	if duplicate.Assets.VectorPath != expectedVectorPath {
		t.Fatalf("unexpected vector path: %s", duplicate.Assets.VectorPath)
	}
	expectedPreviewURL := "https://storage.googleapis.com/bucket/" + expectedPreviewPath
	if duplicate.Assets.PreviewURL != expectedPreviewURL {
		t.Fatalf("unexpected preview url: %s", duplicate.Assets.PreviewURL)
	}
	if duplicate.ThumbnailURL != expectedPreviewURL {
		t.Fatalf("unexpected thumbnail url: %s", duplicate.ThumbnailURL)
	}
	if duplicate.Source.UploadAsset == nil {
		t.Fatalf("expected upload asset to be present")
	} else {
		if duplicate.Source.UploadAsset.AssetID != "render-ver_id101" {
			t.Fatalf("unexpected upload asset id: %s", duplicate.Source.UploadAsset.AssetID)
		}
		if duplicate.Source.UploadAsset.ObjectPath != expectedVectorPath {
			t.Fatalf("unexpected upload object path: %s", duplicate.Source.UploadAsset.ObjectPath)
		}
	}
	if duplicate.Source.RawName != "My Copy" {
		t.Fatalf("expected raw name to use override, got %s", duplicate.Source.RawName)
	}

	assetsSnapshot, ok := duplicate.Snapshot["assets"].(map[string]any)
	if !ok {
		t.Fatalf("expected assets snapshot map, got %T", duplicate.Snapshot["assets"])
	}
	if assetsSnapshot["previewPath"] != expectedPreviewPath {
		t.Fatalf("snapshot preview path mismatch: %v", assetsSnapshot["previewPath"])
	}
	sourceSnapshot, ok := duplicate.Snapshot["source"].(map[string]any)
	if !ok {
		t.Fatalf("expected source snapshot map, got %T", duplicate.Snapshot["source"])
	}
	if uploadSnapshot, ok := sourceSnapshot["uploadAsset"].(map[string]any); !ok {
		t.Fatalf("expected upload asset snapshot, got %T", sourceSnapshot["uploadAsset"])
	} else if uploadSnapshot["objectPath"] != expectedVectorPath {
		t.Fatalf("snapshot upload path mismatch: %v", uploadSnapshot["objectPath"])
	}

	if len(repo.inserted) != 1 {
		t.Fatalf("expected 1 inserted design, got %d", len(repo.inserted))
	}
	if repo.inserted[0].ID != duplicate.ID {
		t.Fatalf("repository stored wrong id: %s", repo.inserted[0].ID)
	}

	if len(versions.appended) != 1 {
		t.Fatalf("expected 1 appended version, got %d", len(versions.appended))
	}
	if versions.appended[0].ID != "ver_id101" {
		t.Fatalf("unexpected version id: %s", versions.appended[0].ID)
	}
	if versions.appended[0].Snapshot["label"] != "My Copy" {
		t.Fatalf("version snapshot label mismatch: %v", versions.appended[0].Snapshot["label"])
	}
	if versions.appended[0].CreatedBy != "user-1" {
		t.Fatalf("expected version created_by to be user-1, got %s", versions.appended[0].CreatedBy)
	}

	records := audit.Records()
	if len(records) != 1 {
		t.Fatalf("expected 1 audit record, got %d", len(records))
	}
	record := records[0]
	if record.Action != "design.duplicate" {
		t.Fatalf("unexpected audit action: %s", record.Action)
	}
	if record.TargetRef != "/designs/dsg_id100" {
		t.Fatalf("unexpected audit target: %s", record.TargetRef)
	}
	if sourceID, ok := record.Metadata["sourceDesignId"]; !ok || sourceID != "dsg_src" {
		t.Fatalf("audit metadata missing source design id: %v", record.Metadata)
	}
}

func TestDesignService_DuplicateDesign_NotOwner(t *testing.T) {
	repo := &stubDesignRepository{
		store: map[string]domain.Design{
			"dsg_src": {
				ID:      "dsg_src",
				OwnerID: "user-1",
				Status:  domain.DesignStatusReady,
			},
		},
	}
	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      repo,
		Versions:     &stubDesignVersionRepository{},
		AssetsBucket: "bucket",
		IDGenerator:  func() string { return "id" },
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	_, err = svc.DuplicateDesign(context.Background(), DuplicateDesignCommand{
		SourceDesignID: "dsg_src",
		RequestedBy:    "user-2",
	})
	if !errors.Is(err, ErrDesignNotFound) {
		t.Fatalf("expected ErrDesignNotFound, got %v", err)
	}
}

func TestDesignService_DuplicateDesign_DeletedSource(t *testing.T) {
	repo := &stubDesignRepository{
		store: map[string]domain.Design{
			"dsg_src": {
				ID:      "dsg_src",
				OwnerID: "user-1",
				Status:  domain.DesignStatusDeleted,
			},
		},
	}
	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      repo,
		Versions:     &stubDesignVersionRepository{},
		AssetsBucket: "bucket",
		IDGenerator:  func() string { return "id" },
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	_, err = svc.DuplicateDesign(context.Background(), DuplicateDesignCommand{
		SourceDesignID: "dsg_src",
		RequestedBy:    "user-1",
	})
	if !errors.Is(err, ErrDesignInvalidInput) {
		t.Fatalf("expected ErrDesignInvalidInput, got %v", err)
	}
}

func TestDesignService_DuplicateDesign_InvalidInput(t *testing.T) {
	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      &stubDesignRepository{},
		Versions:     &stubDesignVersionRepository{},
		AssetsBucket: "bucket",
		IDGenerator:  func() string { return "id" },
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}
	_, err = svc.DuplicateDesign(context.Background(), DuplicateDesignCommand{})
	if !errors.Is(err, ErrDesignInvalidInput) {
		t.Fatalf("expected invalid input error, got %v", err)
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

func TestDesignService_DeleteDesign_Idempotent(t *testing.T) {
	now := time.Date(2025, 3, 12, 9, 0, 0, 0, time.UTC)
	repo := &stubDesignRepository{
		store: map[string]domain.Design{
			"dsg_001": {
				ID:        "dsg_001",
				OwnerID:   "user-1",
				Status:    domain.DesignStatusDeleted,
				UpdatedAt: now,
			},
		},
	}
	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      repo,
		Versions:     &stubDesignVersionRepository{},
		AssetsBucket: "bucket",
		Clock:        func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	err = svc.DeleteDesign(context.Background(), DeleteDesignCommand{
		DesignID:    "dsg_001",
		RequestedBy: "user-1",
		SoftDelete:  true,
	})
	if err != nil {
		t.Fatalf("DeleteDesign should be idempotent, got error: %v", err)
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
