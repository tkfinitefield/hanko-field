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

type stubSuggestionRepository struct {
	mu             sync.Mutex
	store          map[string]map[string]domain.AISuggestion
	inserted       []domain.AISuggestion
	insertFn       func(context.Context, domain.AISuggestion) error
	findFn         func(context.Context, string, string) (domain.AISuggestion, error)
	updateStatusFn func(context.Context, string, string, string, map[string]any) (domain.AISuggestion, error)
}

func (s *stubSuggestionRepository) Insert(ctx context.Context, suggestion domain.AISuggestion) error {
	if s.insertFn != nil {
		return s.insertFn(ctx, suggestion)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.store == nil {
		s.store = make(map[string]map[string]domain.AISuggestion)
	}
	byDesign, ok := s.store[suggestion.DesignID]
	if !ok {
		byDesign = make(map[string]domain.AISuggestion)
		s.store[suggestion.DesignID] = byDesign
	}
	if _, exists := byDesign[suggestion.ID]; exists {
		return suggestionRepoErr{err: errors.New("conflict"), conflict: true}
	}
	clone := cloneSuggestionRecord(suggestion)
	byDesign[suggestion.ID] = clone
	s.inserted = append(s.inserted, cloneSuggestionRecord(suggestion))
	return nil
}

func (s *stubSuggestionRepository) FindByID(ctx context.Context, designID string, suggestionID string) (domain.AISuggestion, error) {
	if s.findFn != nil {
		return s.findFn(ctx, designID, suggestionID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.store == nil {
		return domain.AISuggestion{}, suggestionRepoErr{err: errors.New("not found"), notFound: true}
	}
	designSuggestions, ok := s.store[designID]
	if !ok {
		return domain.AISuggestion{}, suggestionRepoErr{err: errors.New("not found"), notFound: true}
	}
	suggestion, ok := designSuggestions[suggestionID]
	if !ok {
		return domain.AISuggestion{}, suggestionRepoErr{err: errors.New("not found"), notFound: true}
	}
	return cloneSuggestionRecord(suggestion), nil
}

func (s *stubSuggestionRepository) UpdateStatus(ctx context.Context, designID string, suggestionID string, status string, metadata map[string]any) (domain.AISuggestion, error) {
	if s.updateStatusFn != nil {
		return s.updateStatusFn(ctx, designID, suggestionID, status, metadata)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.store == nil {
		return domain.AISuggestion{}, suggestionRepoErr{err: errors.New("not found"), notFound: true}
	}
	designSuggestions, ok := s.store[designID]
	if !ok {
		return domain.AISuggestion{}, suggestionRepoErr{err: errors.New("not found"), notFound: true}
	}
	record, ok := designSuggestions[suggestionID]
	if !ok {
		return domain.AISuggestion{}, suggestionRepoErr{err: errors.New("not found"), notFound: true}
	}
	record.Status = status
	if len(metadata) > 0 {
		if record.Payload == nil {
			record.Payload = make(map[string]any)
		}
		for k, v := range metadata {
			record.Payload[k] = v
		}
	}
	record.UpdatedAt = time.Now().UTC()
	designSuggestions[suggestionID] = cloneSuggestionRecord(record)
	return cloneSuggestionRecord(record), nil
}

func (s *stubSuggestionRepository) ListByDesign(ctx context.Context, designID string, filter repositories.AISuggestionListFilter) (domain.CursorPage[domain.AISuggestion], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	page := domain.CursorPage[domain.AISuggestion]{}
	if s.store == nil {
		return page, nil
	}
	var allowed map[string]struct{}
	if len(filter.Status) > 0 {
		allowed = make(map[string]struct{}, len(filter.Status))
		for _, status := range filter.Status {
			if trimmed := strings.ToLower(strings.TrimSpace(status)); trimmed != "" {
				allowed[trimmed] = struct{}{}
			}
		}
	}
	if suggestions, ok := s.store[designID]; ok {
		for _, suggestion := range suggestions {
			if len(allowed) > 0 {
				current := strings.ToLower(strings.TrimSpace(suggestion.Status))
				if _, ok := allowed[current]; !ok {
					continue
				}
			}
			page.Items = append(page.Items, cloneSuggestionRecord(suggestion))
		}
	}
	return page, nil
}

type suggestionRepoErr struct {
	err         error
	notFound    bool
	conflict    bool
	unavailable bool
}

func (e suggestionRepoErr) Error() string {
	if e.err != nil {
		return e.err.Error()
	}
	return "suggestion repository error"
}

func (e suggestionRepoErr) IsNotFound() bool    { return e.notFound }
func (e suggestionRepoErr) IsConflict() bool    { return e.conflict }
func (e suggestionRepoErr) IsUnavailable() bool { return e.unavailable }

type stubJobDispatcher struct {
	queueFn func(context.Context, QueueAISuggestionCommand) (QueueAISuggestionResult, error)
}

func (s *stubJobDispatcher) QueueAISuggestion(ctx context.Context, cmd QueueAISuggestionCommand) (QueueAISuggestionResult, error) {
	if s.queueFn != nil {
		return s.queueFn(ctx, cmd)
	}
	return QueueAISuggestionResult{}, nil
}

func (s *stubJobDispatcher) GetAIJob(context.Context, string) (domain.AIJob, error) {
	return domain.AIJob{}, errors.New("not implemented")
}

func (s *stubJobDispatcher) CompleteAISuggestion(context.Context, CompleteAISuggestionCommand) (CompleteAISuggestionResult, error) {
	return CompleteAISuggestionResult{}, errors.New("not implemented")
}

func (s *stubJobDispatcher) GetSuggestion(context.Context, string, string) (AISuggestion, error) {
	return AISuggestion{}, errors.New("not implemented")
}

func (s *stubJobDispatcher) EnqueueRegistrabilityCheck(context.Context, RegistrabilityJobPayload) (string, error) {
	return "", errors.New("not implemented")
}

func (s *stubJobDispatcher) EnqueueStockCleanup(context.Context, StockCleanupPayload) error {
	return errors.New("not implemented")
}

type stubRegistrabilityEvaluator struct {
	mu      sync.Mutex
	calls   []RegistrabilityCheckPayload
	checkFn func(context.Context, RegistrabilityCheckPayload) (RegistrabilityAssessment, error)
}

func (s *stubRegistrabilityEvaluator) Check(ctx context.Context, payload RegistrabilityCheckPayload) (RegistrabilityAssessment, error) {
	s.mu.Lock()
	s.calls = append(s.calls, payload)
	s.mu.Unlock()
	if s.checkFn != nil {
		return s.checkFn(ctx, payload)
	}
	return RegistrabilityAssessment{}, errors.New("not implemented")
}

type stubRegistrabilityCache struct {
	mu     sync.Mutex
	store  map[string]RegistrabilityCheckResult
	getFn  func(context.Context, string) (RegistrabilityCheckResult, error)
	saveFn func(context.Context, RegistrabilityCheckResult) error
	saved  []RegistrabilityCheckResult
}

func (s *stubRegistrabilityCache) Get(ctx context.Context, designID string) (RegistrabilityCheckResult, error) {
	if s.getFn != nil {
		return s.getFn(ctx, designID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.store == nil {
		return RegistrabilityCheckResult{}, repoNotFoundError{}
	}
	record, ok := s.store[designID]
	if !ok {
		return RegistrabilityCheckResult{}, repoNotFoundError{}
	}
	return cloneRegistrabilityResult(record), nil
}

func (s *stubRegistrabilityCache) Save(ctx context.Context, result RegistrabilityCheckResult) error {
	if s.saveFn != nil {
		return s.saveFn(ctx, result)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.store == nil {
		s.store = make(map[string]RegistrabilityCheckResult)
	}
	clone := cloneRegistrabilityResult(result)
	s.store[result.DesignID] = clone
	s.saved = append(s.saved, clone)
	return nil
}

type repoNotFoundError struct{}

func (repoNotFoundError) Error() string       { return "not found" }
func (repoNotFoundError) IsNotFound() bool    { return true }
func (repoNotFoundError) IsConflict() bool    { return false }
func (repoNotFoundError) IsUnavailable() bool { return false }

func cloneRegistrabilityResult(result RegistrabilityCheckResult) RegistrabilityCheckResult {
	copy := result
	copy.Reasons = cloneStrings(result.Reasons)
	if result.Metadata != nil {
		copy.Metadata = maps.Clone(result.Metadata)
	}
	if result.ExpiresAt != nil {
		expires := result.ExpiresAt.UTC()
		copy.ExpiresAt = &expires
	}
	return copy
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

func TestDesignService_RequestAISuggestion_Success(t *testing.T) {
	now := time.Date(2025, 6, 1, 9, 0, 0, 0, time.UTC)
	designRepo := &stubDesignRepository{
		store: map[string]domain.Design{
			"dsg_123": {
				ID:        "dsg_123",
				OwnerID:   "user-1",
				Label:     "My Design",
				Type:      domain.DesignTypeTyped,
				Status:    domain.DesignStatusReady,
				TextLines: []string{"Line1", "Line2"},
				Version:   3,
				Snapshot: map[string]any{
					"label":  "My Design",
					"type":   "typed",
					"assets": map[string]any{"previewPath": "path/to/preview"},
				},
			},
		},
	}
	suggestionRepo := &stubSuggestionRepository{}

	var captured QueueAISuggestionCommand
	dispatcher := &stubJobDispatcher{
		queueFn: func(ctx context.Context, cmd QueueAISuggestionCommand) (QueueAISuggestionResult, error) {
			captured = cmd
			return QueueAISuggestionResult{
				JobID:        "aj_001",
				SuggestionID: "as_001",
				Status:       domain.AIJobStatusQueued,
				QueuedAt:     now,
			}, nil
		},
	}

	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      designRepo,
		Versions:     &stubDesignVersionRepository{},
		Suggestions:  suggestionRepo,
		Jobs:         dispatcher,
		AssetsBucket: "bucket",
		Clock:        func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	req := AISuggestionRequest{
		DesignID:       "dsg_123",
		Method:         " balance ",
		Model:          " glyph-balancer@2025-05 ",
		Prompt:         "Balance glyphs",
		Parameters:     map[string]any{"strength": 0.8},
		Metadata:       map[string]any{"channel": "app"},
		IdempotencyKey: "idem-001",
		Priority:       25,
		ActorID:        "user-1",
	}

	result, err := svc.RequestAISuggestion(context.Background(), req)
	if err != nil {
		t.Fatalf("RequestAISuggestion error: %v", err)
	}
	if result.ID != "as_7368dd0c7cc6fff7" {
		t.Fatalf("expected suggestion id as_7368dd0c7cc6fff7, got %s", result.ID)
	}
	if result.Status != string(domain.AIJobStatusQueued) {
		t.Fatalf("expected status queued, got %s", result.Status)
	}
	if result.Method != "balance" {
		t.Fatalf("expected method trimmed to balance, got %s", result.Method)
	}
	if result.Payload == nil || result.Payload["jobId"] != "aj_001" {
		t.Fatalf("expected payload to include jobId, got %+v", result.Payload)
	}

	if captured.DesignID != "dsg_123" {
		t.Fatalf("expected queue design id dsg_123, got %s", captured.DesignID)
	}
	if captured.Method != "balance" {
		t.Fatalf("expected queue method balance, got %s", captured.Method)
	}
	if captured.Model != "glyph-balancer@2025-05" {
		t.Fatalf("expected queue model trimmed, got %s", captured.Model)
	}
	if captured.IdempotencyKey != "idem-001" {
		t.Fatalf("expected idempotency key propagated, got %s", captured.IdempotencyKey)
	}
	if captured.Priority != 25 {
		t.Fatalf("expected priority propagated, got %d", captured.Priority)
	}
	if captured.SuggestionID != result.ID {
		t.Fatalf("expected suggestion id propagated, got %s", captured.SuggestionID)
	}
	if captured.Snapshot == nil || captured.Snapshot["designId"] != "dsg_123" {
		t.Fatalf("expected snapshot to include designId, got %+v", captured.Snapshot)
	}

	record, err := suggestionRepo.FindByID(context.Background(), "dsg_123", result.ID)
	if err != nil {
		t.Fatalf("FindByID error: %v", err)
	}
	if record.Payload == nil || record.Payload["jobId"] != "aj_001" {
		t.Fatalf("expected stored payload to include jobId, got %+v", record.Payload)
	}
}

func TestDesignService_RequestAISuggestion_Idempotent(t *testing.T) {
	now := time.Date(2025, 6, 2, 11, 0, 0, 0, time.UTC)
	designRepo := &stubDesignRepository{
		store: map[string]domain.Design{
			"dsg_999": {
				ID:      "dsg_999",
				OwnerID: "user-9",
				Status:  domain.DesignStatusDraft,
				Label:   "Draft",
			},
		},
	}

	existing := domain.AISuggestion{
		ID:        "as_cf12644d69a7c5ba",
		DesignID:  "dsg_999",
		Method:    "balance",
		Status:    string(domain.AIJobStatusQueued),
		Payload:   map[string]any{"jobId": "aj_existing"},
		CreatedAt: now.Add(-time.Minute),
		UpdatedAt: now.Add(-time.Minute),
	}

	suggestionRepo := &stubSuggestionRepository{
		store: map[string]map[string]domain.AISuggestion{
			"dsg_999": {
				existing.ID: cloneSuggestionRecord(existing),
			},
		},
		insertFn: func(context.Context, domain.AISuggestion) error {
			return suggestionRepoErr{err: errors.New("conflict"), conflict: true}
		},
		findFn: func(context.Context, string, string) (domain.AISuggestion, error) {
			return cloneSuggestionRecord(existing), nil
		},
	}

	dispatcher := &stubJobDispatcher{
		queueFn: func(ctx context.Context, cmd QueueAISuggestionCommand) (QueueAISuggestionResult, error) {
			return QueueAISuggestionResult{
				JobID:        "aj_existing",
				SuggestionID: existing.ID,
				Status:       domain.AIJobStatusQueued,
				QueuedAt:     now,
			}, nil
		},
	}

	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      designRepo,
		Versions:     &stubDesignVersionRepository{},
		Suggestions:  suggestionRepo,
		Jobs:         dispatcher,
		AssetsBucket: "bucket",
		Clock:        func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	result, err := svc.RequestAISuggestion(context.Background(), AISuggestionRequest{
		DesignID:       "dsg_999",
		Method:         "balance",
		Model:          "glyph-balancer@2025-05",
		ActorID:        "user-9",
		IdempotencyKey: "idem-existing",
	})
	if err != nil {
		t.Fatalf("RequestAISuggestion error: %v", err)
	}
	if result.ID != existing.ID {
		t.Fatalf("expected existing suggestion returned, got %s", result.ID)
	}
	if result.Payload["jobId"] != "aj_existing" {
		t.Fatalf("expected existing payload preserved, got %+v", result.Payload)
	}
}

func TestDesignService_RequestAISuggestion_InvalidInput(t *testing.T) {
	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      &stubDesignRepository{},
		Versions:     &stubDesignVersionRepository{},
		Suggestions:  &stubSuggestionRepository{},
		Jobs:         &stubJobDispatcher{},
		AssetsBucket: "bucket",
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}
	_, err = svc.RequestAISuggestion(context.Background(), AISuggestionRequest{
		DesignID: "",
		Method:   "balance",
		Model:    "glyph",
		ActorID:  "user",
	})
	if !errors.Is(err, ErrDesignInvalidInput) {
		t.Fatalf("expected invalid input error, got %v", err)
	}
}

func TestDesignService_ListAISuggestions_StatusFilters(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2025, 7, 1, 9, 0, 0, 0, time.UTC)

	suggestions := &stubSuggestionRepository{
		store: map[string]map[string]domain.AISuggestion{
			"dsg_filters": {
				"as_queue": {
					ID:        "as_queue",
					DesignID:  "dsg_filters",
					Status:    "queued",
					CreatedAt: now,
					UpdatedAt: now,
				},
				"as_ready": {
					ID:        "as_ready",
					DesignID:  "dsg_filters",
					Status:    "proposed",
					CreatedAt: now,
					UpdatedAt: now,
				},
				"as_reject": {
					ID:        "as_reject",
					DesignID:  "dsg_filters",
					Status:    "rejected",
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		},
	}

	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      &stubDesignRepository{},
		Versions:     &stubDesignVersionRepository{},
		Suggestions:  suggestions,
		Jobs:         &stubJobDispatcher{},
		AssetsBucket: "bucket",
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	assertIDs := func(t *testing.T, page domain.CursorPage[AISuggestion], expected ...string) {
		t.Helper()
		got := make(map[string]struct{}, len(page.Items))
		for _, item := range page.Items {
			got[item.ID] = struct{}{}
		}
		if len(got) != len(expected) {
			t.Fatalf("expected %d items, got %d", len(expected), len(got))
		}
		for _, id := range expected {
			if _, ok := got[id]; !ok {
				t.Fatalf("expected suggestion %s in result, got %+v", id, got)
			}
		}
	}

	page, err := svc.ListAISuggestions(ctx, "dsg_filters", AISuggestionFilter{
		Status: []string{"completed"},
	})
	if err != nil {
		t.Fatalf("ListAISuggestions (completed) error: %v", err)
	}
	assertIDs(t, page, "as_ready")

	page, err = svc.ListAISuggestions(ctx, "dsg_filters", AISuggestionFilter{
		Status: []string{"queued"},
	})
	if err != nil {
		t.Fatalf("ListAISuggestions (queued) error: %v", err)
	}
	assertIDs(t, page, "as_queue")

	page, err = svc.ListAISuggestions(ctx, "dsg_filters", AISuggestionFilter{
		Status: []string{"rejected"},
	})
	if err != nil {
		t.Fatalf("ListAISuggestions (rejected) error: %v", err)
	}
	assertIDs(t, page, "as_reject")

	page, err = svc.ListAISuggestions(ctx, "dsg_filters", AISuggestionFilter{
		Status: []string{"queued", "completed"},
	})
	if err != nil {
		t.Fatalf("ListAISuggestions (multi) error: %v", err)
	}
	assertIDs(t, page, "as_queue", "as_ready")
}

func TestDesignService_ListAISuggestions_InvalidInput(t *testing.T) {
	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      &stubDesignRepository{},
		Versions:     &stubDesignVersionRepository{},
		Suggestions:  &stubSuggestionRepository{},
		Jobs:         &stubJobDispatcher{},
		AssetsBucket: "bucket",
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	_, err = svc.ListAISuggestions(context.Background(), "", AISuggestionFilter{})
	if !errors.Is(err, ErrDesignInvalidInput) {
		t.Fatalf("expected invalid input error, got %v", err)
	}
}

func TestDesignService_GetAISuggestion(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	suggestions := &stubSuggestionRepository{
		store: map[string]map[string]domain.AISuggestion{
			"dsg_1": {
				"as_ok": {
					ID:        "as_ok",
					DesignID:  "dsg_1",
					Status:    "proposed",
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		},
	}

	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      &stubDesignRepository{},
		Versions:     &stubDesignVersionRepository{},
		Suggestions:  suggestions,
		Jobs:         &stubJobDispatcher{},
		AssetsBucket: "bucket",
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	result, err := svc.GetAISuggestion(ctx, "dsg_1", "as_ok")
	if err != nil {
		t.Fatalf("GetAISuggestion error: %v", err)
	}
	if result.ID != "as_ok" {
		t.Fatalf("expected as_ok, got %s", result.ID)
	}

	_, err = svc.GetAISuggestion(ctx, "dsg_1", "missing")
	if !errors.Is(err, ErrDesignNotFound) {
		t.Fatalf("expected design not found error, got %v", err)
	}

	_, err = svc.GetAISuggestion(ctx, "", "as_ok")
	if !errors.Is(err, ErrDesignInvalidInput) {
		t.Fatalf("expected invalid input error, got %v", err)
	}
}

func TestDesignService_UpdateAISuggestionStatus_Accept(t *testing.T) {
	now := time.Date(2025, 1, 4, 15, 30, 0, 0, time.UTC)
	designRepo := &stubDesignRepository{
		store: map[string]domain.Design{
			"dsg_accept": {
				ID:               "dsg_accept",
				OwnerID:          "user_accept",
				Status:           domain.DesignStatusDraft,
				Version:          2,
				CurrentVersionID: "ver_old",
				Assets: domain.DesignAssets{
					PreviewPath: "designs/dsg_accept/previews/ver_old.png",
					PreviewURL:  "https://old-preview",
				},
				ThumbnailURL: "https://old-thumb",
				Snapshot: map[string]any{
					"status":  "draft",
					"version": 2,
					"assets": map[string]any{
						"previewPath":  "designs/dsg_accept/previews/ver_old.png",
						"previewUrl":   "https://old-preview",
						"thumbnailUrl": "https://old-thumb",
					},
				},
			},
		},
	}
	suggestionRepo := &stubSuggestionRepository{
		store: map[string]map[string]domain.AISuggestion{
			"dsg_accept": {
				"as_accept": {
					ID:       "as_accept",
					DesignID: "dsg_accept",
					Status:   "proposed",
					Payload: map[string]any{
						"result": map[string]any{
							"preview": map[string]any{
								"previewUrl":   "https://new-preview",
								"thumbnailUrl": "https://new-thumb",
								"objectPath":   "designs/dsg_accept/previews/ver_new.png",
							},
						},
					},
					CreatedAt: now.Add(-2 * time.Minute),
					UpdatedAt: now.Add(-2 * time.Minute),
				},
			},
		},
	}
	versionRepo := &stubDesignVersionRepository{}
	idCalls := 0

	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      designRepo,
		Versions:     versionRepo,
		Suggestions:  suggestionRepo,
		Jobs:         &stubJobDispatcher{},
		AssetsBucket: "bucket",
		Clock:        func() time.Time { return now },
		IDGenerator: func() string {
			idCalls++
			return "ver_next"
		},
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	result, err := svc.UpdateAISuggestionStatus(context.Background(), AISuggestionStatusCommand{
		DesignID:     "dsg_accept",
		SuggestionID: "as_accept",
		Action:       "accept",
		ActorID:      "user_accept",
	})
	if err != nil {
		t.Fatalf("UpdateAISuggestionStatus error: %v", err)
	}
	if idCalls == 0 {
		t.Fatalf("expected ID generator to be invoked")
	}
	if result.Status != "accepted" {
		t.Fatalf("expected suggestion status accepted, got %s", result.Status)
	}
	payload := result.Payload
	if payload == nil {
		t.Fatalf("expected payload in suggestion result")
	}
	if status := stringFromAny(payload["status"]); status != "accepted" {
		t.Fatalf("expected payload status accepted, got %s", status)
	}
	if stringFromAny(payload["acceptedBy"]) != "user_accept" {
		t.Fatalf("expected acceptedBy user_accept")
	}
	if stringFromAny(payload["acceptedAt"]) == "" {
		t.Fatalf("expected acceptedAt timestamp")
	}
	if _, exists := payload["rejectionReason"]; exists {
		t.Fatalf("expected rejectionReason removed after accept")
	}
	if resultMap := mapFromAny(payload["result"]); len(resultMap) == 0 {
		t.Fatalf("expected result metadata present")
	} else {
		if ver := resultMap["newVersion"]; ver != float64(3) && ver != 3 {
			t.Fatalf("expected newVersion 3, got %#v", ver)
		}
		if verID := stringFromAny(resultMap["designVersionId"]); verID != "ver_ver_next" {
			t.Fatalf("expected designVersionId ver_ver_next, got %s", verID)
		}
	}

	if len(designRepo.updated) != 1 {
		t.Fatalf("expected design updated once, got %d", len(designRepo.updated))
	}
	updatedDesign := designRepo.updated[0]
	if updatedDesign.Status != DesignStatusReady {
		t.Fatalf("expected design status ready, got %s", updatedDesign.Status)
	}
	if updatedDesign.Version != 3 {
		t.Fatalf("expected design version 3, got %d", updatedDesign.Version)
	}
	if updatedDesign.CurrentVersionID != "ver_ver_next" {
		t.Fatalf("expected current version id ver_ver_next, got %s", updatedDesign.CurrentVersionID)
	}
	if updatedDesign.Assets.PreviewPath != "designs/dsg_accept/previews/ver_new.png" {
		t.Fatalf("unexpected preview path %s", updatedDesign.Assets.PreviewPath)
	}
	if updatedDesign.Assets.PreviewURL != "https://new-preview" {
		t.Fatalf("unexpected preview url %s", updatedDesign.Assets.PreviewURL)
	}
	if updatedDesign.ThumbnailURL != "https://new-thumb" {
		t.Fatalf("unexpected thumbnail %s", updatedDesign.ThumbnailURL)
	}
	if updatedDesign.UpdatedAt != now {
		t.Fatalf("expected UpdatedAt %v, got %v", now, updatedDesign.UpdatedAt)
	}

	if len(versionRepo.appended) != 1 {
		t.Fatalf("expected version appended once, got %d", len(versionRepo.appended))
	}
	appended := versionRepo.appended[0]
	if appended.Version != 3 {
		t.Fatalf("expected appended version 3, got %d", appended.Version)
	}
	if appended.CreatedBy != "user_accept" {
		t.Fatalf("expected version created by user_accept, got %s", appended.CreatedBy)
	}
	if snap := mapFromAny(appended.Snapshot); len(snap) == 0 || stringFromAny(snap["status"]) != "ready" {
		t.Fatalf("expected snapshot status ready")
	}
}

func TestDesignService_UpdateAISuggestionStatus_Reject(t *testing.T) {
	now := time.Date(2025, 1, 5, 8, 0, 0, 0, time.UTC)
	designRepo := &stubDesignRepository{
		store: map[string]domain.Design{
			"dsg_reject": {
				ID:      "dsg_reject",
				OwnerID: "user_reject",
				Status:  domain.DesignStatusReady,
				Version: 4,
				Snapshot: map[string]any{
					"status": "ready",
				},
			},
		},
	}
	suggestionRepo := &stubSuggestionRepository{
		store: map[string]map[string]domain.AISuggestion{
			"dsg_reject": {
				"as_reject": {
					ID:        "as_reject",
					DesignID:  "dsg_reject",
					Status:    "proposed",
					Payload:   map[string]any{},
					CreatedAt: now.Add(-time.Minute),
					UpdatedAt: now.Add(-time.Minute),
				},
			},
		},
	}
	versionRepo := &stubDesignVersionRepository{}

	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      designRepo,
		Versions:     versionRepo,
		Suggestions:  suggestionRepo,
		Jobs:         &stubJobDispatcher{},
		AssetsBucket: "bucket",
		Clock:        func() time.Time { return now },
		IDGenerator:  func() string { return "unused" },
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	reason := "worse_quality"
	result, err := svc.UpdateAISuggestionStatus(context.Background(), AISuggestionStatusCommand{
		DesignID:     "dsg_reject",
		SuggestionID: "as_reject",
		Action:       "reject",
		ActorID:      "user_reject",
		Reason:       &reason,
	})
	if err != nil {
		t.Fatalf("UpdateAISuggestionStatus reject error: %v", err)
	}
	if result.Status != "rejected" {
		t.Fatalf("expected suggestion status rejected, got %s", result.Status)
	}
	if len(designRepo.updated) != 0 {
		t.Fatalf("expected no design update on rejection")
	}
	if len(versionRepo.appended) != 0 {
		t.Fatalf("expected no version appended on rejection")
	}
	payload := result.Payload
	if payload == nil {
		t.Fatalf("expected payload on rejection")
	}
	if stringFromAny(payload["rejectionReason"]) != "worse_quality" {
		t.Fatalf("expected rejection reason worse_quality")
	}
	if stringFromAny(payload["rejectedBy"]) != "user_reject" {
		t.Fatalf("expected rejectedBy user_reject")
	}
	if stringFromAny(payload["rejectedAt"]) == "" {
		t.Fatalf("expected rejectedAt timestamp present")
	}
	if _, exists := payload["acceptedAt"]; exists {
		t.Fatalf("expected accepted fields cleared on rejection")
	}
}

func TestDesignService_UpdateAISuggestionStatus_InvalidStatus(t *testing.T) {
	now := time.Date(2025, 1, 6, 10, 0, 0, 0, time.UTC)
	designRepo := &stubDesignRepository{
		store: map[string]domain.Design{
			"dsg_invalid": {
				ID:      "dsg_invalid",
				OwnerID: "user_invalid",
				Status:  domain.DesignStatusDraft,
			},
		},
	}
	suggestionRepo := &stubSuggestionRepository{
		store: map[string]map[string]domain.AISuggestion{
			"dsg_invalid": {
				"as_invalid": {
					ID:       "as_invalid",
					DesignID: "dsg_invalid",
					Status:   "queued",
				},
			},
		},
	}
	versionRepo := &stubDesignVersionRepository{}

	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      designRepo,
		Versions:     versionRepo,
		Suggestions:  suggestionRepo,
		Jobs:         &stubJobDispatcher{},
		AssetsBucket: "bucket",
		Clock:        func() time.Time { return now },
		IDGenerator:  func() string { return "irrelevant" },
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	_, err = svc.UpdateAISuggestionStatus(context.Background(), AISuggestionStatusCommand{
		DesignID:     "dsg_invalid",
		SuggestionID: "as_invalid",
		Action:       "accept",
		ActorID:      "user_invalid",
	})
	if !errors.Is(err, ErrDesignInvalidInput) {
		t.Fatalf("expected invalid input error, got %v", err)
	}
}

func TestDesignService_UpdateAISuggestionStatus_Conflict(t *testing.T) {
	now := time.Date(2025, 1, 7, 11, 0, 0, 0, time.UTC)
	designRepo := &stubDesignRepository{
		store: map[string]domain.Design{
			"dsg_conflict": {
				ID:      "dsg_conflict",
				OwnerID: "user_conflict",
				Status:  domain.DesignStatusReady,
			},
		},
	}
	suggestionRepo := &stubSuggestionRepository{
		store: map[string]map[string]domain.AISuggestion{
			"dsg_conflict": {
				"as_conflict": {
					ID:       "as_conflict",
					DesignID: "dsg_conflict",
					Status:   "accepted",
				},
			},
		},
	}
	versionRepo := &stubDesignVersionRepository{}

	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      designRepo,
		Versions:     versionRepo,
		Suggestions:  suggestionRepo,
		Jobs:         &stubJobDispatcher{},
		AssetsBucket: "bucket",
		Clock:        func() time.Time { return now },
		IDGenerator:  func() string { return "irrelevant" },
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	_, err = svc.UpdateAISuggestionStatus(context.Background(), AISuggestionStatusCommand{
		DesignID:     "dsg_conflict",
		SuggestionID: "as_conflict",
		Action:       "accept",
		ActorID:      "user_conflict",
	})
	if !errors.Is(err, ErrDesignConflict) {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

func TestDesignService_UpdateAISuggestionStatus_RejectConflict(t *testing.T) {
	designRepo := &stubDesignRepository{
		store: map[string]domain.Design{
			"dsg_rej_conflict": {
				ID:      "dsg_rej_conflict",
				OwnerID: "user_conflict",
				Status:  domain.DesignStatusReady,
			},
		},
	}
	suggestionRepo := &stubSuggestionRepository{
		store: map[string]map[string]domain.AISuggestion{
			"dsg_rej_conflict": {
				"as_rejected": {
					ID:       "as_rejected",
					DesignID: "dsg_rej_conflict",
					Status:   "rejected",
				},
			},
		},
	}

	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      designRepo,
		Versions:     &stubDesignVersionRepository{},
		Suggestions:  suggestionRepo,
		Jobs:         &stubJobDispatcher{},
		AssetsBucket: "bucket",
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	_, err = svc.UpdateAISuggestionStatus(context.Background(), AISuggestionStatusCommand{
		DesignID:     "dsg_rej_conflict",
		SuggestionID: "as_rejected",
		Action:       "reject",
		ActorID:      "user_conflict",
	})
	if !errors.Is(err, ErrDesignConflict) {
		t.Fatalf("expected conflict on repeated reject, got %v", err)
	}
}

func TestDesignService_UpdateAISuggestionStatus_Unauthorized(t *testing.T) {
	designRepo := &stubDesignRepository{
		store: map[string]domain.Design{
			"dsg_owner": {
				ID:      "dsg_owner",
				OwnerID: "user_owner",
				Status:  domain.DesignStatusDraft,
			},
		},
	}
	suggestionRepo := &stubSuggestionRepository{
		store: map[string]map[string]domain.AISuggestion{
			"dsg_owner": {
				"as_owner": {
					ID:       "as_owner",
					DesignID: "dsg_owner",
					Status:   "proposed",
				},
			},
		},
	}
	versionRepo := &stubDesignVersionRepository{}

	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      designRepo,
		Versions:     versionRepo,
		Suggestions:  suggestionRepo,
		Jobs:         &stubJobDispatcher{},
		AssetsBucket: "bucket",
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	_, err = svc.UpdateAISuggestionStatus(context.Background(), AISuggestionStatusCommand{
		DesignID:     "dsg_owner",
		SuggestionID: "as_owner",
		Action:       "reject",
		ActorID:      "other_user",
	})
	if !errors.Is(err, ErrDesignNotFound) {
		t.Fatalf("expected not found error for non-owner, got %v", err)
	}
}

func TestDesignService_UpdateAISuggestionStatus_InvalidReason(t *testing.T) {
	designRepo := &stubDesignRepository{
		store: map[string]domain.Design{
			"dsg_reason": {
				ID:      "dsg_reason",
				OwnerID: "user_reason",
				Status:  domain.DesignStatusReady,
			},
		},
	}
	suggestionRepo := &stubSuggestionRepository{
		store: map[string]map[string]domain.AISuggestion{
			"dsg_reason": {
				"as_reason": {
					ID:       "as_reason",
					DesignID: "dsg_reason",
					Status:   "proposed",
				},
			},
		},
	}
	versionRepo := &stubDesignVersionRepository{}

	svc, err := NewDesignService(DesignServiceDeps{
		Designs:      designRepo,
		Versions:     versionRepo,
		Suggestions:  suggestionRepo,
		Jobs:         &stubJobDispatcher{},
		AssetsBucket: "bucket",
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	badReason := "not-supported"
	_, err = svc.UpdateAISuggestionStatus(context.Background(), AISuggestionStatusCommand{
		DesignID:     "dsg_reason",
		SuggestionID: "as_reason",
		Action:       "reject",
		ActorID:      "user_reason",
		Reason:       &badReason,
	})
	if !errors.Is(err, ErrDesignInvalidInput) {
		t.Fatalf("expected invalid input error for bad reason, got %v", err)
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

func TestDesignService_RequestRegistrabilityCheck_UsesCache(t *testing.T) {
	now := time.Date(2025, time.January, 2, 10, 0, 0, 0, time.UTC)
	expiry := now.Add(2 * time.Hour)

	repo := &stubDesignRepository{
		store: map[string]domain.Design{
			"dsg_001": {
				ID:        "dsg_001",
				OwnerID:   "user_123",
				Type:      domain.DesignTypeTyped,
				Status:    domain.DesignStatusDraft,
				TextLines: []string{"Yamada"},
				Source: domain.DesignSource{
					Type:      domain.DesignTypeTyped,
					RawName:   "Yamada",
					TextLines: []string{"Yamada"},
				},
				Locale: "ja-JP",
				Shape:  "round",
				SizeMM: 15,
			},
		},
	}

	cached := RegistrabilityCheckResult{
		DesignID:    "dsg_001",
		Status:      "pass",
		Passed:      true,
		Reasons:     []string{"ok"},
		RequestedAt: now.Add(-30 * time.Minute),
		ExpiresAt:   &expiry,
	}

	cache := &stubRegistrabilityCache{
		getFn: func(context.Context, string) (RegistrabilityCheckResult, error) {
			return cached, nil
		},
	}

	evaluator := &stubRegistrabilityEvaluator{
		checkFn: func(context.Context, RegistrabilityCheckPayload) (RegistrabilityAssessment, error) {
			t.Fatalf("expected cached result to be used")
			return RegistrabilityAssessment{}, nil
		},
	}

	svc, err := NewDesignService(DesignServiceDeps{
		Designs:             repo,
		Versions:            &stubDesignVersionRepository{},
		AssetsBucket:        "bucket",
		Clock:               func() time.Time { return now },
		Registrability:      evaluator,
		RegistrabilityCache: cache,
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	result, err := svc.RequestRegistrabilityCheck(context.Background(), RegistrabilityCheckCommand{
		DesignID: "dsg_001",
		UserID:   "user_123",
	})
	if err != nil {
		t.Fatalf("RequestRegistrabilityCheck error: %v", err)
	}

	if !result.Passed || result.Status != "pass" {
		t.Fatalf("unexpected cached result: %+v", result)
	}
	if len(evaluator.calls) != 0 {
		t.Fatalf("expected evaluator not to be called, was called %d times", len(evaluator.calls))
	}
}

func TestDesignService_RequestRegistrabilityCheck_EvaluatesAndCaches(t *testing.T) {
	now := time.Date(2025, time.January, 2, 11, 0, 0, 0, time.UTC)

	repo := &stubDesignRepository{
		store: map[string]domain.Design{
			"dsg_002": {
				ID:        "dsg_002",
				OwnerID:   "user_456",
				Type:      domain.DesignTypeTyped,
				Status:    domain.DesignStatusDraft,
				TextLines: []string{"Suzuki"},
				Source: domain.DesignSource{
					Type:      domain.DesignTypeTyped,
					RawName:   "Suzuki",
					TextLines: []string{"Suzuki"},
				},
				Locale: "ja-JP",
				Shape:  "square",
				SizeMM: 18,
			},
		},
	}

	cache := &stubRegistrabilityCache{}

	evaluator := &stubRegistrabilityEvaluator{
		checkFn: func(_ context.Context, payload RegistrabilityCheckPayload) (RegistrabilityAssessment, error) {
			if payload.DesignID != "dsg_002" {
				t.Fatalf("unexpected payload design id: %s", payload.DesignID)
			}
			if payload.Name != "Suzuki" {
				t.Fatalf("unexpected payload name: %s", payload.Name)
			}
			return RegistrabilityAssessment{
				Status:  "fail",
				Passed:  false,
				Reasons: []string{"conflict"},
			}, nil
		},
	}

	svc, err := NewDesignService(DesignServiceDeps{
		Designs:             repo,
		Versions:            &stubDesignVersionRepository{},
		AssetsBucket:        "bucket",
		Clock:               func() time.Time { return now },
		Registrability:      evaluator,
		RegistrabilityCache: cache,
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	result, err := svc.RequestRegistrabilityCheck(context.Background(), RegistrabilityCheckCommand{
		DesignID: "dsg_002",
		UserID:   "user_456",
	})
	if err != nil {
		t.Fatalf("RequestRegistrabilityCheck error: %v", err)
	}

	if result.Passed || result.Status != "fail" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Reasons) != 1 || result.Reasons[0] != "conflict" {
		t.Fatalf("unexpected reasons: %+v", result.Reasons)
	}
	if len(evaluator.calls) != 1 {
		t.Fatalf("expected evaluator to be called once, called %d times", len(evaluator.calls))
	}
	if len(cache.saved) != 1 {
		t.Fatalf("expected cache save, got %d", len(cache.saved))
	}
	saved := cache.saved[0]
	if saved.Status != "fail" || saved.Passed {
		t.Fatalf("unexpected cached value: %+v", saved)
	}
	if saved.ExpiresAt == nil {
		t.Fatalf("expected cached value to have expiry")
	}
	expectedExpiry := now.Add(defaultRegistrabilityCacheTTL)
	if saved.ExpiresAt.Sub(expectedExpiry) > time.Second || saved.ExpiresAt.Sub(expectedExpiry) < -time.Second {
		t.Fatalf("unexpected expiry: %v", saved.ExpiresAt)
	}
}

func TestDesignService_RequestRegistrabilityCheck_ExternalUnavailable(t *testing.T) {
	now := time.Date(2025, time.January, 2, 12, 0, 0, 0, time.UTC)

	repo := &stubDesignRepository{
		store: map[string]domain.Design{
			"dsg_003": {
				ID:        "dsg_003",
				OwnerID:   "user_789",
				Type:      domain.DesignTypeTyped,
				Status:    domain.DesignStatusDraft,
				TextLines: []string{"Tanaka"},
				Source: domain.DesignSource{
					Type:      domain.DesignTypeTyped,
					RawName:   "Tanaka",
					TextLines: []string{"Tanaka"},
				},
				Locale: "ja-JP",
				Shape:  "round",
				SizeMM: 16,
			},
		},
	}

	evaluator := &stubRegistrabilityEvaluator{
		checkFn: func(context.Context, RegistrabilityCheckPayload) (RegistrabilityAssessment, error) {
			return RegistrabilityAssessment{}, ErrRegistrabilityUnavailable
		},
	}

	svc, err := NewDesignService(DesignServiceDeps{
		Designs:             repo,
		Versions:            &stubDesignVersionRepository{},
		AssetsBucket:        "bucket",
		Clock:               func() time.Time { return now },
		Registrability:      evaluator,
		RegistrabilityCache: &stubRegistrabilityCache{},
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	_, err = svc.RequestRegistrabilityCheck(context.Background(), RegistrabilityCheckCommand{
		DesignID: "dsg_003",
		UserID:   "user_789",
	})
	if !errors.Is(err, ErrDesignRepositoryUnavailable) {
		t.Fatalf("expected ErrDesignRepositoryUnavailable, got %v", err)
	}
}

func TestDesignService_RequestRegistrabilityCheck_MissingName(t *testing.T) {
	now := time.Date(2025, time.January, 2, 13, 0, 0, 0, time.UTC)

	repo := &stubDesignRepository{
		store: map[string]domain.Design{
			"dsg_004": {
				ID:      "dsg_004",
				OwnerID: "user_999",
				Type:    domain.DesignTypeTyped,
				Status:  domain.DesignStatusDraft,
			},
		},
	}

	evaluator := &stubRegistrabilityEvaluator{
		checkFn: func(context.Context, RegistrabilityCheckPayload) (RegistrabilityAssessment, error) {
			t.Fatalf("evaluator should not be called")
			return RegistrabilityAssessment{}, nil
		},
	}

	svc, err := NewDesignService(DesignServiceDeps{
		Designs:             repo,
		Versions:            &stubDesignVersionRepository{},
		AssetsBucket:        "bucket",
		Clock:               func() time.Time { return now },
		Registrability:      evaluator,
		RegistrabilityCache: &stubRegistrabilityCache{},
	})
	if err != nil {
		t.Fatalf("NewDesignService error: %v", err)
	}

	_, err = svc.RequestRegistrabilityCheck(context.Background(), RegistrabilityCheckCommand{
		DesignID: "dsg_004",
		UserID:   "user_999",
	})
	if !errors.Is(err, ErrDesignInvalidInput) {
		t.Fatalf("expected ErrDesignInvalidInput, got %v", err)
	}
}

func cloneSuggestionRecord(s domain.AISuggestion) domain.AISuggestion {
	copy := s
	if s.Payload != nil {
		copy.Payload = maps.Clone(s.Payload)
	}
	if s.ExpiresAt != nil {
		expires := *s.ExpiresAt
		copy.ExpiresAt = &expires
	}
	return copy
}
