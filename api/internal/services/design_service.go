package services

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"path"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/platform/storage"
	"github.com/hanko-field/api/internal/repositories"
)

var (
	// ErrDesignInvalidInput indicates the caller provided invalid arguments.
	ErrDesignInvalidInput = errors.New("design: invalid input")
	// ErrDesignNotFound indicates the requested design does not exist.
	ErrDesignNotFound = errors.New("design: not found")
	// ErrDesignConflict indicates the operation would conflict with existing state.
	ErrDesignConflict = errors.New("design: conflict")
	// ErrDesignRepositoryUnavailable signals that persistence dependencies are unavailable.
	ErrDesignRepositoryUnavailable = errors.New("design: repository unavailable")
	// ErrDesignRendererUnavailable indicates the renderer dependency was required but missing.
	ErrDesignRendererUnavailable = errors.New("design: renderer unavailable")
	// ErrDesignNotImplemented is returned for unimplemented operations.
	ErrDesignNotImplemented = errors.New("design: not implemented")
)

const (
	designIDPrefix                = "dsg_"
	versionIDPrefix               = "ver_"
	defaultLocale                 = "ja-JP"
	defaultShape                  = "round"
	defaultRegistrabilityCacheTTL = 24 * time.Hour
	minDesignSizeMM               = 6.0
	maxDesignSizeMM               = 30.0
	defaultDesignSize             = 15.0
	maxDesignTextLines            = 4
	maxDesignLineChars            = 32
	maxDesignLabelLen             = 120
	maxUploadSizeBytes            = int64(20 * 1024 * 1024)
	previewFileName               = "preview.png"
	vectorFileName                = "design.svg"
)

var (
	allowedUploadContentTypes = map[string]struct{}{
		"image/png":       {},
		"image/jpeg":      {},
		"image/jpg":       {},
		"image/svg+xml":   {},
		"application/pdf": {},
	}
	bannedDesignWords = []string{"死", "殺", "fuck"}
)

type createDesignParams struct {
	ownerID      string
	actorID      string
	label        string
	designType   DesignType
	textLines    []string
	fontID       string
	materialID   string
	templateID   string
	locale       string
	shape        string
	sizeMM       float64
	rawName      string
	kanjiValue   *string
	kanjiMapping *string
	uploadAsset  *DesignAssetReference
	logoAsset    *DesignAssetReference
	snapshot     map[string]any
	metadata     map[string]any
	idempotency  string
}

type assetPlan struct {
	sourcePath   string
	vectorPath   string
	previewPath  string
	previewURL   string
	thumbnailURL string
	uploadAsset  *DesignAssetReference
	logoAsset    *DesignAssetReference
}

// DesignRenderer renders the initial assets for a design variant.
type DesignRenderer interface {
	Render(ctx context.Context, req RenderDesignRequest) (RenderDesignResult, error)
}

// RenderDesignRequest describes the payload provided to the renderer dependency.
type RenderDesignRequest struct {
	DesignID     string
	VersionID    string
	Type         DesignType
	TextLines    []string
	FontID       string
	TemplateID   string
	Locale       string
	MaterialID   string
	OutputBucket string
	VectorPath   string
	PreviewPath  string
}

// RenderDesignResult reports the generated asset locations.
type RenderDesignResult struct {
	VectorPath   string
	PreviewPath  string
	PreviewURL   string
	ThumbnailURL string
}

// DesignServiceDeps wires dependencies for the design service implementation.
type DesignServiceDeps struct {
	Designs             repositories.DesignRepository
	Versions            repositories.DesignVersionRepository
	Audit               AuditLogService
	Renderer            DesignRenderer
	AssetCopier         AssetCopier
	Suggestions         repositories.AISuggestionRepository
	Jobs                BackgroundJobDispatcher
	UnitOfWork          repositories.UnitOfWork
	Clock               func() time.Time
	IDGenerator         func() string
	AssetsBucket        string
	Logger              func(context.Context, string, map[string]any)
	Registrability      RegistrabilityEvaluator
	RegistrabilityCache repositories.RegistrabilityRepository
	RegistrabilityTTL   time.Duration
}

type designService struct {
	designs        repositories.DesignRepository
	versions       repositories.DesignVersionRepository
	audit          AuditLogService
	renderer       DesignRenderer
	assetCopier    AssetCopier
	suggestions    repositories.AISuggestionRepository
	jobs           BackgroundJobDispatcher
	unitOfWork     repositories.UnitOfWork
	clock          func() time.Time
	newID          func() string
	assetsBucket   string
	logger         func(context.Context, string, map[string]any)
	registrability RegistrabilityEvaluator
	regCache       repositories.RegistrabilityRepository
	regTTL         time.Duration
}

// NewDesignService constructs a DesignService backed by the provided dependencies.
func NewDesignService(deps DesignServiceDeps) (DesignService, error) {
	if deps.Designs == nil {
		return nil, errors.New("design service: designs repository is required")
	}
	if deps.Versions == nil {
		return nil, errors.New("design service: versions repository is required")
	}
	bucket := strings.TrimSpace(deps.AssetsBucket)
	if bucket == "" {
		return nil, errors.New("design service: assets bucket is required")
	}

	clock := deps.Clock
	if clock == nil {
		clock = time.Now
	}

	idGen := deps.IDGenerator
	if idGen == nil {
		idGen = func() string { return ulid.Make().String() }
	}

	logger := deps.Logger
	if logger == nil {
		logger = func(context.Context, string, map[string]any) {}
	}

	regTTL := deps.RegistrabilityTTL
	if regTTL <= 0 {
		regTTL = defaultRegistrabilityCacheTTL
	}

	return &designService{
		designs:        deps.Designs,
		versions:       deps.Versions,
		audit:          deps.Audit,
		renderer:       deps.Renderer,
		assetCopier:    deps.AssetCopier,
		suggestions:    deps.Suggestions,
		jobs:           deps.Jobs,
		unitOfWork:     deps.UnitOfWork,
		clock:          func() time.Time { return clock().UTC() },
		newID:          idGen,
		assetsBucket:   bucket,
		logger:         logger,
		registrability: deps.Registrability,
		regCache:       deps.RegistrabilityCache,
		regTTL:         regTTL,
	}, nil
}

// CreateDesign validates the payload, renders initial assets, and persists the design + initial version.
func (s *designService) CreateDesign(ctx context.Context, cmd CreateDesignCommand) (Design, error) {
	params, err := s.prepareCreateParams(cmd)
	if err != nil {
		return Design{}, err
	}

	now := s.now()
	designID := s.nextDesignID()
	versionNumber := 1
	versionID := s.nextVersionID()

	label := params.label
	if label == "" {
		label = defaultDesignLabel(designID, params.designType, params.textLines)
	}

	assetPlan, err := s.planAssets(ctx, designID, versionID, params)
	if err != nil {
		return Design{}, err
	}

	snapshot := buildDesignSnapshot(label, params, assetPlan)

	design := Design{
		ID:         designID,
		OwnerID:    params.ownerID,
		Label:      label,
		Type:       params.designType,
		TextLines:  cloneStrings(params.textLines),
		FontID:     params.fontID,
		MaterialID: params.materialID,
		Template:   params.templateID,
		Locale:     params.locale,
		Shape:      params.shape,
		SizeMM:     params.sizeMM,
		Source: DesignSource{
			Type:        params.designType,
			RawName:     params.rawName,
			TextLines:   cloneStrings(params.textLines),
			UploadAsset: cloneAssetReference(assetPlan.uploadAsset),
			LogoAsset:   cloneAssetReference(assetPlan.logoAsset),
		},
		Assets: DesignAssets{
			SourcePath:  assetPlan.sourcePath,
			VectorPath:  assetPlan.vectorPath,
			PreviewPath: assetPlan.previewPath,
			PreviewURL:  assetPlan.previewURL,
		},
		Status:           DesignStatusDraft,
		ThumbnailURL:     assetPlan.thumbnailURL,
		Version:          versionNumber,
		CurrentVersionID: versionID,
		Snapshot:         snapshot,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	version := DesignVersion{
		ID:        versionID,
		DesignID:  designID,
		Version:   versionNumber,
		Snapshot:  cloneSnapshot(snapshot),
		CreatedAt: now,
		CreatedBy: params.actorID,
	}

	if err := s.runInTx(ctx, func(txCtx context.Context) error {
		if err := s.designs.Insert(txCtx, design); err != nil {
			return s.mapRepositoryError(err)
		}
		if err := s.versions.Append(txCtx, version); err != nil {
			return s.mapRepositoryError(err)
		}
		return nil
	}); err != nil {
		return Design{}, err
	}

	s.recordAudit(ctx, design, params)

	return design, nil
}

// GetDesign fetches a single design by ID.
func (s *designService) GetDesign(ctx context.Context, designID string, opts DesignReadOptions) (Design, error) {
	if s.designs == nil {
		return Design{}, ErrDesignRepositoryUnavailable
	}
	design, err := s.designs.FindByID(ctx, designID)
	if err != nil {
		return Design{}, s.mapRepositoryError(err)
	}
	if opts.IncludeVersions {
		if s.versions == nil {
			return Design{}, ErrDesignRepositoryUnavailable
		}
		page, err := s.versions.ListByDesign(ctx, design.ID, domain.Pagination{})
		if err != nil {
			return Design{}, s.mapRepositoryError(err)
		}
		if len(page.Items) > 0 {
			design.Versions = make([]domain.DesignVersion, len(page.Items))
			copy(design.Versions, page.Items)
		}
	}
	return design, nil
}

// ListDesigns returns designs owned by a user filtered by status.
func (s *designService) ListDesigns(ctx context.Context, filter DesignListFilter) (domain.CursorPage[Design], error) {
	if s.designs == nil {
		return domain.CursorPage[Design]{}, ErrDesignRepositoryUnavailable
	}
	page, err := s.designs.ListByOwner(ctx, filter.OwnerID, repositories.DesignListFilter{
		Status:       filter.Status,
		Types:        filter.Types,
		UpdatedAfter: filter.UpdatedAfter,
		Pagination:   filter.Pagination,
	})
	if err != nil {
		return domain.CursorPage[Design]{}, s.mapRepositoryError(err)
	}
	return page, nil
}

func (s *designService) ListDesignVersions(ctx context.Context, designID string, filter DesignVersionListFilter) (domain.CursorPage[DesignVersion], error) {
	if s.versions == nil {
		return domain.CursorPage[DesignVersion]{}, ErrDesignRepositoryUnavailable
	}

	designID = strings.TrimSpace(designID)
	if designID == "" {
		return domain.CursorPage[DesignVersion]{}, fmt.Errorf("%w: design_id is required", ErrDesignInvalidInput)
	}

	page, err := s.versions.ListByDesign(ctx, designID, domain.Pagination(filter.Pagination))
	if err != nil {
		return domain.CursorPage[DesignVersion]{}, s.mapRepositoryError(err)
	}

	if !filter.IncludeAssets && len(page.Items) > 0 {
		for i, version := range page.Items {
			page.Items[i].Snapshot = stripSnapshotAssets(version.Snapshot)
		}
	}

	return page, nil
}

func (s *designService) GetDesignVersion(ctx context.Context, designID, versionID string, opts DesignVersionReadOptions) (DesignVersion, error) {
	if s.versions == nil {
		return DesignVersion{}, ErrDesignRepositoryUnavailable
	}

	designID = strings.TrimSpace(designID)
	if designID == "" {
		return DesignVersion{}, fmt.Errorf("%w: design_id is required", ErrDesignInvalidInput)
	}
	versionID = strings.TrimSpace(versionID)
	if versionID == "" {
		return DesignVersion{}, fmt.Errorf("%w: version_id is required", ErrDesignInvalidInput)
	}

	version, err := s.versions.FindByID(ctx, designID, versionID)
	if err != nil {
		return DesignVersion{}, s.mapRepositoryError(err)
	}
	if !opts.IncludeAssets {
		version.Snapshot = stripSnapshotAssets(version.Snapshot)
	}
	return version, nil
}

func (s *designService) UpdateDesign(ctx context.Context, cmd UpdateDesignCommand) (Design, error) {
	if s.designs == nil || s.versions == nil {
		return Design{}, ErrDesignRepositoryUnavailable
	}

	designID := strings.TrimSpace(cmd.DesignID)
	if designID == "" {
		return Design{}, fmt.Errorf("%w: design_id is required", ErrDesignInvalidInput)
	}
	actorID := strings.TrimSpace(cmd.UpdatedBy)
	if actorID == "" {
		return Design{}, fmt.Errorf("%w: updated_by is required", ErrDesignInvalidInput)
	}

	design, err := s.designs.FindByID(ctx, designID)
	if err != nil {
		return Design{}, s.mapRepositoryError(err)
	}

	if cmd.ExpectedUpdatedAt != nil {
		expected := cmd.ExpectedUpdatedAt.UTC()
		if design.UpdatedAt.IsZero() || !design.UpdatedAt.UTC().Equal(expected) {
			return Design{}, ErrDesignConflict
		}
	}

	if !designEditable(design.Status) {
		return Design{}, fmt.Errorf("%w: design status %q does not allow updates", ErrDesignInvalidInput, design.Status)
	}

	updated := design

	if cmd.Label != nil {
		updated.Label = strings.TrimSpace(*cmd.Label)
	}
	if updated.Label == "" {
		updated.Label = defaultDesignLabel(updated.ID, updated.Type, updated.TextLines)
	}

	if cmd.Status != nil {
		nextStatus, err := normalizeDesignStatus(*cmd.Status)
		if err != nil {
			return Design{}, err
		}
		if !designStatusUserMutable(nextStatus) {
			return Design{}, fmt.Errorf("%w: status %q is not allowed", ErrDesignInvalidInput, nextStatus)
		}
		updated.Status = nextStatus
	}

	if cmd.ThumbnailURL != nil {
		updated.ThumbnailURL = strings.TrimSpace(*cmd.ThumbnailURL)
	}

	var snapshot map[string]any
	if len(cmd.Snapshot) > 0 {
		snapshot = cloneSnapshot(cmd.Snapshot)
	} else {
		snapshot = cloneSnapshot(design.Snapshot)
	}
	updated.Snapshot = snapshot

	now := s.now()
	newVersionID := s.nextVersionID()
	updated.Version = design.Version + 1
	updated.CurrentVersionID = newVersionID
	updated.UpdatedAt = now

	version := domain.DesignVersion{
		ID:        newVersionID,
		DesignID:  designID,
		Version:   updated.Version,
		Snapshot:  cloneSnapshot(snapshot),
		CreatedAt: now,
		CreatedBy: actorID,
	}

	if err := s.runInTx(ctx, func(txCtx context.Context) error {
		if err := s.designs.Update(txCtx, updated); err != nil {
			return s.mapRepositoryError(err)
		}
		if err := s.versions.Append(txCtx, version); err != nil {
			return s.mapRepositoryError(err)
		}
		return nil
	}); err != nil {
		return Design{}, err
	}

	s.recordAuditUpdate(ctx, design, updated, actorID)

	return updated, nil
}

func (s *designService) DeleteDesign(ctx context.Context, cmd DeleteDesignCommand) error {
	if s.designs == nil {
		return ErrDesignRepositoryUnavailable
	}

	designID := strings.TrimSpace(cmd.DesignID)
	if designID == "" {
		return fmt.Errorf("%w: design_id is required", ErrDesignInvalidInput)
	}
	actorID := strings.TrimSpace(cmd.RequestedBy)
	if actorID == "" {
		return fmt.Errorf("%w: requested_by is required", ErrDesignInvalidInput)
	}
	if !cmd.SoftDelete {
		return ErrDesignNotImplemented
	}

	design, err := s.designs.FindByID(ctx, designID)
	if err != nil {
		return s.mapRepositoryError(err)
	}

	if cmd.ExpectedUpdatedAt != nil {
		expected := cmd.ExpectedUpdatedAt.UTC()
		if design.UpdatedAt.IsZero() || !design.UpdatedAt.UTC().Equal(expected) {
			return ErrDesignConflict
		}
	}

	if design.Status == DesignStatusDeleted {
		return nil
	}

	if !designDeletable(design.Status) {
		return fmt.Errorf("%w: design status %q cannot be deleted", ErrDesignInvalidInput, design.Status)
	}

	if design.Status == DesignStatusDeleted {
		return nil
	}

	now := s.now()
	if err := s.designs.SoftDelete(ctx, designID, now); err != nil {
		return s.mapRepositoryError(err)
	}

	s.recordAuditDelete(ctx, design, actorID, now)
	return nil
}

func (s *designService) DuplicateDesign(ctx context.Context, cmd DuplicateDesignCommand) (Design, error) {
	if s.designs == nil || s.versions == nil {
		return Design{}, ErrDesignRepositoryUnavailable
	}

	sourceID := strings.TrimSpace(cmd.SourceDesignID)
	if sourceID == "" {
		return Design{}, fmt.Errorf("%w: source_design_id is required", ErrDesignInvalidInput)
	}
	actorID := strings.TrimSpace(cmd.RequestedBy)
	if actorID == "" {
		return Design{}, fmt.Errorf("%w: requested_by is required", ErrDesignInvalidInput)
	}

	source, err := s.designs.FindByID(ctx, sourceID)
	if err != nil {
		return Design{}, s.mapRepositoryError(err)
	}

	ownerID := strings.TrimSpace(source.OwnerID)
	if ownerID == "" {
		return Design{}, fmt.Errorf("%w: source design owner missing", ErrDesignInvalidInput)
	}
	if !strings.EqualFold(ownerID, actorID) {
		return Design{}, ErrDesignNotFound
	}
	if source.Status == DesignStatusDeleted {
		return Design{}, fmt.Errorf("%w: source design status %q cannot be duplicated", ErrDesignInvalidInput, source.Status)
	}

	now := s.now()
	newDesignID := s.nextDesignID()
	versionID := s.nextVersionID()

	label := strings.TrimSpace(source.Label)
	if cmd.OverrideName != nil {
		if trimmed := strings.TrimSpace(*cmd.OverrideName); trimmed != "" {
			label = trimmed
		}
	}
	if label == "" {
		label = defaultDesignLabel(newDesignID, source.Type, source.TextLines)
	}
	if len(label) > maxDesignLabelLen {
		label = label[:maxDesignLabelLen]
	}

	textLines := cloneStrings(source.TextLines)
	if len(textLines) == 0 && len(source.Source.TextLines) > 0 {
		textLines = cloneStrings(source.Source.TextLines)
	}

	assets, uploadAsset, logoAsset, tasks, err := s.prepareDuplicateAssets(source, newDesignID, versionID)
	if err != nil {
		return Design{}, err
	}

	rawName := strings.TrimSpace(source.Source.RawName)
	if cmd.OverrideName != nil {
		if trimmed := strings.TrimSpace(*cmd.OverrideName); trimmed != "" {
			rawName = trimmed
		}
	}
	if rawName == "" {
		rawName = label
	}

	sourceLines := cloneStrings(source.Source.TextLines)
	if len(sourceLines) == 0 && len(textLines) > 0 {
		sourceLines = cloneStrings(textLines)
	}

	snapshot := s.buildDuplicateSnapshot(source, label, textLines, assets, uploadAsset, logoAsset, rawName)

	thumbnailURL := strings.TrimSpace(assets.PreviewURL)
	if thumbnailURL == "" {
		thumbnailURL = strings.TrimSpace(source.ThumbnailURL)
	}

	design := Design{
		ID:         newDesignID,
		OwnerID:    ownerID,
		Label:      label,
		Type:       source.Type,
		TextLines:  cloneStrings(textLines),
		FontID:     source.FontID,
		MaterialID: source.MaterialID,
		Template:   source.Template,
		Locale:     source.Locale,
		Shape:      source.Shape,
		SizeMM:     source.SizeMM,
		Source: DesignSource{
			Type:        firstNonEmptyDesignType(source.Source.Type, source.Type),
			RawName:     rawName,
			TextLines:   sourceLines,
			UploadAsset: cloneAssetReference(uploadAsset),
			LogoAsset:   cloneAssetReference(logoAsset),
		},
		Assets:           assets,
		Status:           DesignStatusDraft,
		ThumbnailURL:     thumbnailURL,
		Version:          1,
		CurrentVersionID: versionID,
		Snapshot:         snapshot,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	version := DesignVersion{
		ID:        versionID,
		DesignID:  newDesignID,
		Version:   1,
		Snapshot:  cloneSnapshot(snapshot),
		CreatedAt: now,
		CreatedBy: actorID,
	}

	if err := s.runInTx(ctx, func(txCtx context.Context) error {
		if err := s.designs.Insert(txCtx, design); err != nil {
			return s.mapRepositoryError(err)
		}
		if err := s.versions.Append(txCtx, version); err != nil {
			return s.mapRepositoryError(err)
		}
		return nil
	}); err != nil {
		return Design{}, err
	}

	s.recordAuditDuplicate(ctx, source, design, actorID)
	s.copyAssetsAsync(ctx, tasks)

	return design, nil
}

type assetCopyTask struct {
	sourceBucket string
	sourceObject string
	destBucket   string
	destObject   string
}

func (s *designService) prepareDuplicateAssets(source Design, newDesignID, versionID string) (DesignAssets, *DesignAssetReference, *DesignAssetReference, []assetCopyTask, error) {
	assets := DesignAssets{}
	tasks := make([]assetCopyTask, 0, 4)

	previewPath := strings.TrimSpace(source.Assets.PreviewPath)
	if previewPath != "" {
		fileName := fileNameFromPath(previewPath, previewFileName)
		newPreviewPath, err := storage.BuildObjectPath(storage.PurposePreview, storage.PathParams{
			DesignID:  newDesignID,
			VersionID: versionID,
			FileName:  fileName,
		})
		if err != nil {
			return DesignAssets{}, nil, nil, nil, fmt.Errorf("%w: %v", ErrDesignInvalidInput, err)
		}
		assets.PreviewPath = newPreviewPath
		assets.PreviewURL = buildBucketURL(s.assetsBucket, newPreviewPath)
		tasks = appendCopyTask(tasks, assetCopyTask{
			sourceBucket: s.assetsBucket,
			sourceObject: previewPath,
			destBucket:   s.assetsBucket,
			destObject:   newPreviewPath,
		})
	} else if url := strings.TrimSpace(source.Assets.PreviewURL); url != "" {
		assets.PreviewURL = url
	}

	var (
		uploadAsset *DesignAssetReference
		logoAsset   *DesignAssetReference
	)

	// Vector assets are generated for typed designs and should be copied alongside the master asset.
	originalUpload := cloneAssetReference(source.Source.UploadAsset)
	vectorPath := strings.TrimSpace(source.Assets.VectorPath)
	var newVectorPath string
	if vectorPath != "" {
		vectorFile := fileNameFromPath(vectorPath, vectorFileName)
		pathParams := storage.PathParams{
			DesignID: newDesignID,
			UploadID: "render-" + versionID,
			FileName: vectorFile,
		}
		computed, err := storage.BuildObjectPath(storage.PurposeDesignMaster, pathParams)
		if err != nil {
			return DesignAssets{}, nil, nil, nil, fmt.Errorf("%w: %v", ErrDesignInvalidInput, err)
		}
		newVectorPath = computed
		assets.VectorPath = newVectorPath
	}

	sourcePath := strings.TrimSpace(source.Assets.SourcePath)
	if newVectorPath != "" && (originalUpload == nil || strings.TrimSpace(originalUpload.ObjectPath) == vectorPath) {
		bucket := firstNonEmptyString(assetBucket(originalUpload), s.assetsBucket)
		if bucket == "" {
			bucket = s.assetsBucket
		}
		fileName := fileNameFromPath(vectorPath, vectorFileName)
		uploadAsset = &DesignAssetReference{
			AssetID:     "render-" + versionID,
			Bucket:      bucket,
			ObjectPath:  newVectorPath,
			FileName:    fileName,
			ContentType: firstNonEmptyString(assetContentType(originalUpload), "image/svg+xml"),
			SizeBytes:   0,
			Checksum:    "",
		}
		if originalUpload != nil {
			uploadAsset.SizeBytes = originalUpload.SizeBytes
			uploadAsset.Checksum = strings.TrimSpace(originalUpload.Checksum)
		}
		assets.SourcePath = newVectorPath
		tasks = appendCopyTask(tasks, assetCopyTask{
			sourceBucket: firstNonEmptyString(assetBucket(originalUpload), s.assetsBucket),
			sourceObject: firstNonEmptyString(assetObjectPath(originalUpload), vectorPath),
			destBucket:   bucket,
			destObject:   newVectorPath,
		})
	} else if originalUpload != nil && strings.TrimSpace(originalUpload.ObjectPath) != "" {
		fileName := firstNonEmptyString(originalUpload.FileName, fileNameFromPath(originalUpload.ObjectPath, "source"))
		newUploadID := "upload-" + s.newID()
		newSourcePath, err := storage.BuildObjectPath(storage.PurposeDesignMaster, storage.PathParams{
			DesignID: newDesignID,
			UploadID: newUploadID,
			FileName: fileName,
		})
		if err != nil {
			return DesignAssets{}, nil, nil, nil, fmt.Errorf("%w: %v", ErrDesignInvalidInput, err)
		}
		bucket := firstNonEmptyString(originalUpload.Bucket, s.assetsBucket)
		if bucket == "" {
			bucket = s.assetsBucket
		}
		uploadAsset = &DesignAssetReference{
			AssetID:     newUploadID,
			Bucket:      bucket,
			ObjectPath:  newSourcePath,
			FileName:    fileName,
			ContentType: originalUpload.ContentType,
			SizeBytes:   originalUpload.SizeBytes,
			Checksum:    strings.TrimSpace(originalUpload.Checksum),
		}
		assets.SourcePath = newSourcePath
		tasks = appendCopyTask(tasks, assetCopyTask{
			sourceBucket: firstNonEmptyString(assetBucket(originalUpload), s.assetsBucket),
			sourceObject: originalUpload.ObjectPath,
			destBucket:   bucket,
			destObject:   newSourcePath,
		})
	} else if sourcePath != "" {
		fileName := fileNameFromPath(sourcePath, "source")
		newUploadID := "upload-" + s.newID()
		newSourcePath, err := storage.BuildObjectPath(storage.PurposeDesignMaster, storage.PathParams{
			DesignID: newDesignID,
			UploadID: newUploadID,
			FileName: fileName,
		})
		if err != nil {
			return DesignAssets{}, nil, nil, nil, fmt.Errorf("%w: %v", ErrDesignInvalidInput, err)
		}
		assets.SourcePath = newSourcePath
		bucket := s.assetsBucket
		uploadAsset = &DesignAssetReference{
			AssetID:    newUploadID,
			Bucket:     bucket,
			ObjectPath: newSourcePath,
			FileName:   fileName,
		}
		tasks = appendCopyTask(tasks, assetCopyTask{
			sourceBucket: s.assetsBucket,
			sourceObject: sourcePath,
			destBucket:   bucket,
			destObject:   newSourcePath,
		})
	}

	// If vector path exists and wasn't handled above, ensure copy task is queued.
	if newVectorPath != "" && (len(tasks) == 0 || tasks[len(tasks)-1].destObject != newVectorPath) {
		tasks = appendCopyTask(tasks, assetCopyTask{
			sourceBucket: firstNonEmptyString(assetBucket(originalUpload), s.assetsBucket),
			sourceObject: firstNonEmptyString(assetObjectPath(originalUpload), vectorPath),
			destBucket:   firstNonEmptyString(assetBucket(uploadAsset), s.assetsBucket),
			destObject:   newVectorPath,
		})
	}

	// Logo assets are optional; copy if present.
	if originalLogo := source.Source.LogoAsset; originalLogo != nil && strings.TrimSpace(originalLogo.ObjectPath) != "" {
		fileName := firstNonEmptyString(originalLogo.FileName, fileNameFromPath(originalLogo.ObjectPath, "logo"))
		newLogoID := "logo-" + s.newID()
		newLogoPath, err := storage.BuildObjectPath(storage.PurposeDesignMaster, storage.PathParams{
			DesignID: newDesignID,
			UploadID: newLogoID,
			FileName: fileName,
		})
		if err != nil {
			return DesignAssets{}, nil, nil, nil, fmt.Errorf("%w: %v", ErrDesignInvalidInput, err)
		}
		bucket := firstNonEmptyString(assetBucket(originalLogo), s.assetsBucket)
		if bucket == "" {
			bucket = s.assetsBucket
		}
		logoAsset = &DesignAssetReference{
			AssetID:     newLogoID,
			Bucket:      bucket,
			ObjectPath:  newLogoPath,
			FileName:    fileName,
			ContentType: originalLogo.ContentType,
			SizeBytes:   originalLogo.SizeBytes,
			Checksum:    strings.TrimSpace(originalLogo.Checksum),
		}
		tasks = appendCopyTask(tasks, assetCopyTask{
			sourceBucket: firstNonEmptyString(assetBucket(originalLogo), s.assetsBucket),
			sourceObject: assetObjectPath(originalLogo),
			destBucket:   bucket,
			destObject:   newLogoPath,
		})
	}

	return assets, uploadAsset, logoAsset, tasks, nil
}

func (s *designService) buildDuplicateSnapshot(source Design, label string, textLines []string, assets DesignAssets, uploadAsset, logoAsset *DesignAssetReference, rawName string) map[string]any {
	snapshot := cloneSnapshot(source.Snapshot)
	if snapshot == nil {
		snapshot = make(map[string]any)
	}
	snapshot["label"] = label
	snapshot["status"] = string(DesignStatusDraft)
	if len(textLines) > 0 {
		snapshot["textLines"] = cloneStrings(textLines)
	} else {
		delete(snapshot, "textLines")
	}
	snapshot["type"] = string(source.Type)

	assetsSnapshot := map[string]any{}
	if assets.SourcePath != "" {
		assetsSnapshot["sourcePath"] = assets.SourcePath
	}
	if assets.VectorPath != "" {
		assetsSnapshot["vectorPath"] = assets.VectorPath
	}
	if assets.PreviewPath != "" {
		assetsSnapshot["previewPath"] = assets.PreviewPath
	}
	if assets.PreviewURL != "" {
		assetsSnapshot["previewUrl"] = assets.PreviewURL
	}
	if len(assetsSnapshot) > 0 {
		snapshot["assets"] = assetsSnapshot
	} else {
		delete(snapshot, "assets")
	}

	sourceSnapshot := map[string]any{}
	if existing, ok := snapshot["source"].(map[string]any); ok && len(existing) > 0 {
		sourceSnapshot = maps.Clone(existing)
	}
	sourceSnapshot["type"] = string(firstNonEmptyDesignType(source.Source.Type, source.Type))
	sourceSnapshot["rawName"] = rawName
	if len(textLines) > 0 {
		sourceSnapshot["textLines"] = cloneStrings(textLines)
	} else {
		delete(sourceSnapshot, "textLines")
	}
	if uploadAsset != nil {
		sourceSnapshot["uploadAsset"] = assetReferenceSnapshot(uploadAsset)
	} else {
		delete(sourceSnapshot, "uploadAsset")
	}
	if logoAsset != nil {
		sourceSnapshot["logoAsset"] = assetReferenceSnapshot(logoAsset)
	} else {
		delete(sourceSnapshot, "logoAsset")
	}
	snapshot["source"] = sourceSnapshot
	return snapshot
}

func (s *designService) copyAssetsAsync(ctx context.Context, tasks []assetCopyTask) {
	if len(tasks) == 0 || s.assetCopier == nil {
		return
	}
	background := context.WithoutCancel(ctx)
	go func(copyTasks []assetCopyTask) {
		for _, task := range copyTasks {
			if err := s.assetCopier.CopyObject(background, task.sourceBucket, task.sourceObject, task.destBucket, task.destObject); err != nil {
				if s.logger != nil {
					s.logger(background, "design.asset_copy_failed", map[string]any{
						"sourceBucket": task.sourceBucket,
						"sourceObject": task.sourceObject,
						"destBucket":   task.destBucket,
						"destObject":   task.destObject,
						"error":        err.Error(),
					})
				}
			}
		}
	}(append([]assetCopyTask(nil), tasks...))
}

func appendCopyTask(tasks []assetCopyTask, task assetCopyTask) []assetCopyTask {
	sourceBucket := strings.TrimSpace(task.sourceBucket)
	sourceObject := strings.TrimSpace(task.sourceObject)
	destBucket := strings.TrimSpace(task.destBucket)
	destObject := strings.TrimSpace(task.destObject)
	if sourceBucket == "" || sourceObject == "" || destBucket == "" || destObject == "" {
		return tasks
	}
	if sourceBucket == destBucket && sourceObject == destObject {
		return tasks
	}
	normalised := assetCopyTask{
		sourceBucket: sourceBucket,
		sourceObject: sourceObject,
		destBucket:   destBucket,
		destObject:   destObject,
	}
	for _, existing := range tasks {
		if existing == normalised {
			return tasks
		}
	}
	return append(tasks, normalised)
}

func fileNameFromPath(objectPath, fallback string) string {
	trimmed := strings.TrimSpace(objectPath)
	if trimmed == "" {
		return fallback
	}
	base := path.Base(trimmed)
	if base == "." || base == "/" || base == "" {
		return fallback
	}
	return base
}

func firstNonEmptyDesignType(values ...DesignType) DesignType {
	for _, value := range values {
		if trimmed := strings.TrimSpace(string(value)); trimmed != "" {
			return value
		}
	}
	return ""
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func assetBucket(ref *DesignAssetReference) string {
	if ref == nil {
		return ""
	}
	return ref.Bucket
}

func assetObjectPath(ref *DesignAssetReference) string {
	if ref == nil {
		return ""
	}
	return ref.ObjectPath
}

func assetContentType(ref *DesignAssetReference) string {
	if ref == nil {
		return ""
	}
	return ref.ContentType
}

func (s *designService) RequestAISuggestion(ctx context.Context, cmd AISuggestionRequest) (AISuggestion, error) {
	if s.designs == nil {
		return AISuggestion{}, ErrDesignRepositoryUnavailable
	}
	if s.jobs == nil || s.suggestions == nil {
		return AISuggestion{}, ErrDesignNotImplemented
	}

	designID := strings.TrimSpace(cmd.DesignID)
	if designID == "" {
		return AISuggestion{}, fmt.Errorf("%w: design_id is required", ErrDesignInvalidInput)
	}
	method := strings.TrimSpace(cmd.Method)
	if method == "" {
		return AISuggestion{}, fmt.Errorf("%w: method is required", ErrDesignInvalidInput)
	}
	model := strings.TrimSpace(cmd.Model)
	if model == "" {
		return AISuggestion{}, fmt.Errorf("%w: model is required", ErrDesignInvalidInput)
	}
	actorID := strings.TrimSpace(cmd.ActorID)
	if actorID == "" {
		return AISuggestion{}, fmt.Errorf("%w: actor_id is required", ErrDesignInvalidInput)
	}

	design, err := s.designs.FindByID(ctx, designID)
	if err != nil {
		return AISuggestion{}, s.mapRepositoryError(err)
	}

	ownerID := strings.TrimSpace(design.OwnerID)
	if ownerID == "" {
		return AISuggestion{}, fmt.Errorf("%w: design owner missing", ErrDesignInvalidInput)
	}
	if !strings.EqualFold(ownerID, actorID) {
		return AISuggestion{}, ErrDesignNotFound
	}
	if design.Status == DesignStatusDeleted {
		return AISuggestion{}, fmt.Errorf("%w: design status %q cannot request ai suggestions", ErrDesignInvalidInput, design.Status)
	}

	snapshot := stripSnapshotAssets(design.Snapshot)
	if snapshot == nil {
		snapshot = make(map[string]any)
	}
	if len(snapshot) == 0 {
		if label := strings.TrimSpace(design.Label); label != "" {
			snapshot["label"] = label
		}
		if len(design.TextLines) > 0 {
			snapshot["textLines"] = cloneStrings(design.TextLines)
		}
		if design.Type != "" {
			snapshot["type"] = string(design.Type)
		}
		if design.Version > 0 {
			snapshot["version"] = design.Version
		}
		if design.Status != "" {
			snapshot["status"] = string(design.Status)
		}
	}
	if _, ok := snapshot["designId"]; !ok {
		snapshot["designId"] = design.ID
	}

	parameters := cloneMetadata(cmd.Parameters)
	metadata := cloneMetadata(cmd.Metadata)
	idempotencyKey := strings.TrimSpace(cmd.IdempotencyKey)

	suggestionID := ensureSuggestionID(s.newID())
	if idempotencyKey != "" {
		suggestionID = ensureSuggestionID(suggestionIDFromKey(idempotencyKey))
	}

	now := s.now()
	payload := map[string]any{
		"model":       model,
		"requestedBy": actorID,
		"queuedAt":    now.Format(time.RFC3339Nano),
	}
	if idempotencyKey != "" {
		payload["idempotencyKey"] = idempotencyKey
	}
	if prompt := strings.TrimSpace(cmd.Prompt); prompt != "" {
		payload["prompt"] = prompt
	}
	if len(parameters) > 0 {
		payload["parameters"] = cloneMetadata(parameters)
	}
	if len(metadata) > 0 {
		payload["metadata"] = cloneMetadata(metadata)
	}

	suggestion := AISuggestion{
		ID:        suggestionID,
		DesignID:  design.ID,
		Method:    method,
		Status:    string(domain.AIJobStatusQueued),
		Payload:   payload,
		CreatedAt: now,
		UpdatedAt: now,
	}

	insertedNew := true
	if err := s.suggestions.Insert(ctx, suggestion); err != nil {
		var repoErr repositories.RepositoryError
		if errors.As(err, &repoErr) && repoErr.IsConflict() {
			insertedNew = false
			existing, findErr := s.suggestions.FindByID(ctx, design.ID, suggestionID)
			if findErr != nil {
				return AISuggestion{}, s.mapRepositoryError(findErr)
			}
			suggestion = existing
		} else {
			return AISuggestion{}, s.mapRepositoryError(err)
		}
	}

	queueResult, err := s.jobs.QueueAISuggestion(ctx, QueueAISuggestionCommand{
		SuggestionID:   suggestionID,
		DesignID:       design.ID,
		Method:         method,
		Model:          model,
		Prompt:         strings.TrimSpace(cmd.Prompt),
		Snapshot:       snapshot,
		Parameters:     parameters,
		Metadata:       metadata,
		IdempotencyKey: idempotencyKey,
		Priority:       cmd.Priority,
		RequestedBy:    actorID,
	})
	if err != nil {
		if insertedNew {
			failPayload := cloneMetadata(suggestion.Payload)
			if failPayload == nil {
				failPayload = make(map[string]any)
			}
			failPayload["jobDispatchError"] = err.Error()
			failPayload["jobDispatchFailedAt"] = now.Format(time.RFC3339Nano)
			if _, updateErr := s.suggestions.UpdateStatus(ctx, design.ID, suggestionID, string(domain.AIJobStatusFailed), failPayload); updateErr != nil && s.logger != nil {
				s.logger(ctx, "design.ai_suggestion_dispatch_fail_update", map[string]any{
					"designId":     design.ID,
					"suggestionId": suggestionID,
					"error":        updateErr.Error(),
				})
			}
		}
		return AISuggestion{}, s.mapAIError(err)
	}

	current, err := s.suggestions.FindByID(ctx, design.ID, suggestionID)
	if err != nil {
		return AISuggestion{}, s.mapRepositoryError(err)
	}
	payloadUpdate := cloneMetadata(current.Payload)
	if payloadUpdate == nil {
		payloadUpdate = make(map[string]any)
	}
	payloadUpdate["jobId"] = queueResult.JobID
	if len(parameters) > 0 {
		if _, exists := payloadUpdate["parameters"]; !exists {
			payloadUpdate["parameters"] = cloneMetadata(parameters)
		}
	}
	if len(metadata) > 0 {
		if _, exists := payloadUpdate["metadata"]; !exists {
			payloadUpdate["metadata"] = cloneMetadata(metadata)
		}
	}
	statusForUpdate := current.Status
	if strings.TrimSpace(statusForUpdate) == "" {
		statusForUpdate = string(queueResult.Status)
	}
	updatedSuggestion, updateErr := s.suggestions.UpdateStatus(ctx, design.ID, suggestionID, statusForUpdate, payloadUpdate)
	if updateErr != nil {
		var repoErr repositories.RepositoryError
		if errors.As(updateErr, &repoErr) && repoErr.IsConflict() {
			updatedSuggestion, err = s.suggestions.FindByID(ctx, design.ID, suggestionID)
			if err != nil {
				return AISuggestion{}, s.mapRepositoryError(err)
			}
		} else {
			return AISuggestion{}, s.mapRepositoryError(updateErr)
		}
	} else {
		suggestion = updatedSuggestion
	}

	return suggestion, nil
}

func (s *designService) ListAISuggestions(ctx context.Context, designID string, filter AISuggestionFilter) (domain.CursorPage[AISuggestion], error) {
	if s.suggestions == nil {
		return domain.CursorPage[AISuggestion]{}, ErrDesignNotImplemented
	}

	designID = strings.TrimSpace(designID)
	if designID == "" {
		return domain.CursorPage[AISuggestion]{}, fmt.Errorf("%w: design_id is required", ErrDesignInvalidInput)
	}

	statusFilters := normalizeSuggestionStatusFilters(filter.Status)

	query := repositories.AISuggestionListFilter{
		Pagination: filter.Pagination,
	}
	if len(statusFilters) > 0 {
		query.Status = expandSuggestionStatusFilters(statusFilters)
	}

	page, err := s.suggestions.ListByDesign(ctx, designID, query)
	if err != nil {
		return domain.CursorPage[AISuggestion]{}, s.mapRepositoryError(err)
	}

	return page, nil
}

func (s *designService) GetAISuggestion(ctx context.Context, designID string, suggestionID string) (AISuggestion, error) {
	if s.suggestions == nil {
		return AISuggestion{}, ErrDesignNotImplemented
	}

	designID = strings.TrimSpace(designID)
	if designID == "" {
		return AISuggestion{}, fmt.Errorf("%w: design_id is required", ErrDesignInvalidInput)
	}
	suggestionID = strings.TrimSpace(suggestionID)
	if suggestionID == "" {
		return AISuggestion{}, fmt.Errorf("%w: suggestion_id is required", ErrDesignInvalidInput)
	}

	suggestion, err := s.suggestions.FindByID(ctx, designID, suggestionID)
	if err != nil {
		return AISuggestion{}, s.mapRepositoryError(err)
	}
	return suggestion, nil
}

func (s *designService) UpdateAISuggestionStatus(ctx context.Context, cmd AISuggestionStatusCommand) (AISuggestion, error) {
	if s.suggestions == nil {
		return AISuggestion{}, ErrDesignNotImplemented
	}

	designID := strings.TrimSpace(cmd.DesignID)
	if designID == "" {
		return AISuggestion{}, fmt.Errorf("%w: design_id is required", ErrDesignInvalidInput)
	}
	suggestionID := strings.TrimSpace(cmd.SuggestionID)
	if suggestionID == "" {
		return AISuggestion{}, fmt.Errorf("%w: suggestion_id is required", ErrDesignInvalidInput)
	}

	action := strings.ToLower(strings.TrimSpace(cmd.Action))
	if action == "" {
		return AISuggestion{}, fmt.Errorf("%w: action is required", ErrDesignInvalidInput)
	}
	if action != "accept" && action != "reject" {
		return AISuggestion{}, fmt.Errorf("%w: unsupported action %q", ErrDesignInvalidInput, action)
	}

	actorID := strings.TrimSpace(cmd.ActorID)
	if actorID == "" {
		return AISuggestion{}, fmt.Errorf("%w: actor_id is required", ErrDesignInvalidInput)
	}

	design, err := s.designs.FindByID(ctx, designID)
	if err != nil {
		return AISuggestion{}, s.mapRepositoryError(err)
	}

	ownerID := strings.TrimSpace(design.OwnerID)
	if ownerID == "" {
		return AISuggestion{}, fmt.Errorf("%w: design owner missing", ErrDesignInvalidInput)
	}
	if !strings.EqualFold(ownerID, actorID) {
		return AISuggestion{}, ErrDesignNotFound
	}
	if design.Status == DesignStatusDeleted {
		return AISuggestion{}, fmt.Errorf("%w: design status %q cannot update suggestions", ErrDesignInvalidInput, design.Status)
	}
	if action == "accept" && design.Status != DesignStatusDraft && design.Status != DesignStatusReady {
		return AISuggestion{}, fmt.Errorf("%w: design status %q cannot accept suggestions", ErrDesignInvalidInput, design.Status)
	}

	suggestion, err := s.suggestions.FindByID(ctx, designID, suggestionID)
	if err != nil {
		return AISuggestion{}, s.mapRepositoryError(err)
	}

	currentStatus := strings.ToLower(strings.TrimSpace(suggestion.Status))
	category := suggestionStatusCategory(currentStatus)

	switch action {
	case "accept":
		if currentStatus == "accepted" || currentStatus == "applied" {
			return AISuggestion{}, ErrDesignConflict
		}
		if category == "rejected" {
			return AISuggestion{}, ErrDesignConflict
		}
		if category != "completed" {
			return AISuggestion{}, fmt.Errorf("%w: suggestion status %q cannot be accepted", ErrDesignInvalidInput, suggestion.Status)
		}
	case "reject":
		if category == "rejected" || currentStatus == "accepted" || currentStatus == "applied" {
			return AISuggestion{}, ErrDesignConflict
		}
		if category != "completed" {
			return AISuggestion{}, fmt.Errorf("%w: suggestion status %q cannot be rejected", ErrDesignInvalidInput, suggestion.Status)
		}
	}

	now := s.now()
	payloadUpdate := cloneMetadata(suggestion.Payload)
	if payloadUpdate == nil {
		payloadUpdate = make(map[string]any)
	}

	var (
		nextStatus      = action
		updatedDesign   Design
		newVersion      domain.DesignVersion
		applySuggestion bool
	)

	switch action {
	case "accept":
		payloadUpdate["acceptedAt"] = now.Format(time.RFC3339Nano)
		payloadUpdate["acceptedBy"] = actorID
		delete(payloadUpdate, "rejectionReason")
		delete(payloadUpdate, "rejectedAt")
		delete(payloadUpdate, "rejectedBy")

		updatedDesign = design
		if updatedDesign.Status == DesignStatusDraft {
			updatedDesign.Status = DesignStatusReady
		}

		snapshot := cloneSnapshot(design.Snapshot)
		if snapshot == nil {
			snapshot = make(map[string]any)
		}

		preview := extractSuggestionPreviewInfo(payloadUpdate)

		if preview.ObjectPath != "" {
			updatedDesign.Assets.PreviewPath = preview.ObjectPath
		}
		if preview.PreviewURL != "" {
			updatedDesign.Assets.PreviewURL = preview.PreviewURL
		}
		if preview.ThumbnailURL != "" {
			updatedDesign.ThumbnailURL = preview.ThumbnailURL
		} else if updatedDesign.ThumbnailURL == "" && preview.PreviewURL != "" {
			updatedDesign.ThumbnailURL = preview.PreviewURL
		}

		assetsSnapshot := mapFromAny(snapshot["assets"])
		if assetsSnapshot == nil {
			assetsSnapshot = make(map[string]any)
		}
		if preview.ObjectPath != "" {
			assetsSnapshot["previewPath"] = preview.ObjectPath
		}
		if preview.PreviewURL != "" {
			assetsSnapshot["previewUrl"] = preview.PreviewURL
		}
		if preview.ThumbnailURL != "" {
			assetsSnapshot["thumbnailUrl"] = preview.ThumbnailURL
		}
		if len(assetsSnapshot) > 0 {
			snapshot["assets"] = assetsSnapshot
		}
		snapshot["status"] = string(updatedDesign.Status)

		newVersionID := s.nextVersionID()
		updatedDesign.Version = design.Version + 1
		updatedDesign.CurrentVersionID = newVersionID
		updatedDesign.UpdatedAt = now
		snapshot["version"] = updatedDesign.Version
		updatedDesign.Snapshot = snapshot

		result := mapFromAny(payloadUpdate["result"])
		if result == nil {
			result = make(map[string]any)
		}
		result["newVersion"] = updatedDesign.Version
		result["designVersionId"] = updatedDesign.CurrentVersionID
		result["appliedAt"] = now.Format(time.RFC3339Nano)
		payloadUpdate["result"] = result

		newVersion = domain.DesignVersion{
			ID:        newVersionID,
			DesignID:  design.ID,
			Version:   updatedDesign.Version,
			Snapshot:  cloneSnapshot(snapshot),
			CreatedAt: now,
			CreatedBy: actorID,
		}

		nextStatus = "accepted"
		applySuggestion = true

	case "reject":
		var reasonValue string
		if cmd.Reason != nil {
			trimmed := strings.ToLower(strings.TrimSpace(*cmd.Reason))
			if trimmed != "" {
				if _, ok := suggestionRejectionReasons[trimmed]; !ok {
					return AISuggestion{}, fmt.Errorf("%w: unsupported rejection reason %q", ErrDesignInvalidInput, trimmed)
				}
				reasonValue = trimmed
			}
		}

		payloadUpdate["rejectedAt"] = now.Format(time.RFC3339Nano)
		payloadUpdate["rejectedBy"] = actorID
		if reasonValue != "" {
			payloadUpdate["rejectionReason"] = reasonValue
		} else {
			delete(payloadUpdate, "rejectionReason")
		}
		delete(payloadUpdate, "acceptedAt")
		delete(payloadUpdate, "acceptedBy")
		nextStatus = "rejected"
	}

	payloadUpdate["status"] = nextStatus

	if err := s.runInTx(ctx, func(txCtx context.Context) error {
		if applySuggestion {
			if err := s.designs.Update(txCtx, updatedDesign); err != nil {
				return s.mapRepositoryError(err)
			}
			if err := s.versions.Append(txCtx, newVersion); err != nil {
				return s.mapRepositoryError(err)
			}
		}

		updatedSuggestion, err := s.suggestions.UpdateStatus(txCtx, designID, suggestionID, nextStatus, cloneMetadata(payloadUpdate))
		if err != nil {
			return s.mapRepositoryError(err)
		}
		suggestion = updatedSuggestion
		return nil
	}); err != nil {
		return AISuggestion{}, err
	}

	if applySuggestion {
		s.recordAuditUpdate(ctx, design, updatedDesign, actorID)
	}

	return suggestion, nil
}

func (s *designService) RequestRegistrabilityCheck(ctx context.Context, cmd RegistrabilityCheckCommand) (RegistrabilityCheckResult, error) {
	if s.designs == nil {
		return RegistrabilityCheckResult{}, ErrDesignRepositoryUnavailable
	}

	designID := strings.TrimSpace(cmd.DesignID)
	if designID == "" {
		return RegistrabilityCheckResult{}, fmt.Errorf("%w: design_id is required", ErrDesignInvalidInput)
	}
	userID := strings.TrimSpace(cmd.UserID)
	if userID == "" {
		return RegistrabilityCheckResult{}, fmt.Errorf("%w: user_id is required", ErrDesignInvalidInput)
	}

	design, err := s.designs.FindByID(ctx, designID)
	if err != nil {
		return RegistrabilityCheckResult{}, s.mapRepositoryError(err)
	}

	ownerID := strings.TrimSpace(design.OwnerID)
	if ownerID == "" || !strings.EqualFold(ownerID, userID) {
		return RegistrabilityCheckResult{}, ErrDesignNotFound
	}

	if design.Status == DesignStatusDeleted {
		return RegistrabilityCheckResult{}, fmt.Errorf("%w: design status %q cannot be evaluated", ErrDesignInvalidInput, design.Status)
	}

	if result, ok := s.cachedRegistrability(ctx, designID); ok {
		return result, nil
	}

	if s.registrability == nil {
		return RegistrabilityCheckResult{}, ErrDesignNotImplemented
	}

	payload, err := s.buildRegistrabilityPayload(cmd, design)
	if err != nil {
		return RegistrabilityCheckResult{}, err
	}

	assessment, err := s.registrability.Check(ctx, payload)
	if err != nil {
		return RegistrabilityCheckResult{}, s.mapRegistrabilityError(err)
	}

	now := s.now()
	result := RegistrabilityCheckResult{
		DesignID:    designID,
		Status:      strings.TrimSpace(assessment.Status),
		Passed:      assessment.Passed,
		Score:       assessment.Score,
		Reasons:     cloneStrings(assessment.Reasons),
		RequestedAt: now,
		Metadata:    cloneSnapshot(assessment.Metadata),
	}
	if assessment.ExpiresAt != nil && !assessment.ExpiresAt.IsZero() {
		expiry := assessment.ExpiresAt.UTC()
		result.ExpiresAt = &expiry
	} else if s.regTTL > 0 {
		expiry := now.Add(s.regTTL)
		result.ExpiresAt = &expiry
	}

	s.cacheRegistrability(ctx, result)

	return result, nil
}

func (s *designService) cachedRegistrability(ctx context.Context, designID string) (RegistrabilityCheckResult, bool) {
	if s.regCache == nil {
		return RegistrabilityCheckResult{}, false
	}
	result, err := s.regCache.Get(ctx, designID)
	if err != nil {
		if !isRepoNotFound(err) && err != context.Canceled && err != context.DeadlineExceeded {
			s.logger(ctx, "design.registrability.cache_read_failed", map[string]any{
				"designId": designID,
				"error":    err.Error(),
			})
		}
		return RegistrabilityCheckResult{}, false
	}
	result = normalizeRegistrabilityResult(result)
	now := s.now()
	if !s.registrabilityResultFresh(result, now) {
		return RegistrabilityCheckResult{}, false
	}
	return result, true
}

func (s *designService) cacheRegistrability(ctx context.Context, result RegistrabilityCheckResult) {
	if s.regCache == nil {
		return
	}
	result = normalizeRegistrabilityResult(result)
	if err := s.regCache.Save(ctx, result); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		s.logger(ctx, "design.registrability.cache_write_failed", map[string]any{
			"designId": result.DesignID,
			"error":    err.Error(),
		})
	}
}

func (s *designService) buildRegistrabilityPayload(cmd RegistrabilityCheckCommand, design Design) (RegistrabilityCheckPayload, error) {
	textLines := cloneStrings(design.Source.TextLines)
	if len(textLines) == 0 {
		textLines = cloneStrings(design.TextLines)
	}

	name := strings.TrimSpace(design.Source.RawName)
	if name == "" && len(textLines) > 0 {
		name = strings.Join(textLines, "")
	}
	if name == "" {
		return RegistrabilityCheckPayload{}, fmt.Errorf("%w: design metadata missing name", ErrDesignInvalidInput)
	}

	designType := firstNonEmptyDesignType(design.Source.Type, design.Type)
	if strings.TrimSpace(string(designType)) == "" {
		return RegistrabilityCheckPayload{}, fmt.Errorf("%w: design type is required", ErrDesignInvalidInput)
	}

	locale := strings.TrimSpace(cmd.Locale)
	if locale == "" {
		locale = strings.TrimSpace(design.Locale)
		if locale == "" {
			locale = defaultLocale
		}
	}

	metadata := make(map[string]any)
	if trimmed := strings.TrimSpace(design.Shape); trimmed != "" {
		metadata["shape"] = trimmed
	}
	if design.SizeMM > 0 {
		metadata["sizeMm"] = design.SizeMM
	}

	return RegistrabilityCheckPayload{
		DesignID:   design.ID,
		Name:       name,
		TextLines:  textLines,
		Type:       designType,
		Locale:     locale,
		MaterialID: strings.TrimSpace(design.MaterialID),
		TemplateID: strings.TrimSpace(design.Template),
		Metadata:   metadata,
	}, nil
}

func (s *designService) registrabilityResultFresh(result RegistrabilityCheckResult, now time.Time) bool {
	if strings.TrimSpace(result.DesignID) == "" {
		return false
	}

	if result.ExpiresAt != nil && !result.ExpiresAt.IsZero() {
		if now.After(result.ExpiresAt.UTC()) {
			return false
		}
		return true
	}

	if s.regTTL <= 0 {
		return true
	}

	ref := result.RequestedAt
	if ref.IsZero() {
		ref = now
	}
	return now.Before(ref.Add(s.regTTL))
}

func (s *designService) mapRegistrabilityError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ErrRegistrabilityInvalidInput):
		return fmt.Errorf("%w: %v", ErrDesignInvalidInput, err)
	case errors.Is(err, ErrRegistrabilityUnavailable), errors.Is(err, ErrRegistrabilityRateLimited):
		return fmt.Errorf("%w: %v", ErrDesignRepositoryUnavailable, err)
	default:
		return fmt.Errorf("%w: %v", ErrDesignRepositoryUnavailable, err)
	}
}

func normalizeRegistrabilityResult(result RegistrabilityCheckResult) RegistrabilityCheckResult {
	result.Status = strings.TrimSpace(result.Status)
	result.Reasons = cloneStrings(result.Reasons)
	if len(result.Metadata) > 0 {
		result.Metadata = cloneSnapshot(result.Metadata)
	}
	if result.ExpiresAt != nil && !result.ExpiresAt.IsZero() {
		expiry := result.ExpiresAt.UTC()
		result.ExpiresAt = &expiry
	}
	return result
}

func (s *designService) runInTx(ctx context.Context, fn func(context.Context) error) error {
	if s.unitOfWork == nil {
		return fn(ctx)
	}
	return s.unitOfWork.RunInTx(ctx, fn)
}

func (s *designService) now() time.Time {
	return s.clock()
}

func (s *designService) nextDesignID() string {
	return designIDPrefix + strings.ToLower(strings.TrimSpace(s.newID()))
}

func (s *designService) nextVersionID() string {
	return versionIDPrefix + strings.ToLower(strings.TrimSpace(s.newID()))
}

func (s *designService) prepareCreateParams(cmd CreateDesignCommand) (createDesignParams, error) {
	ownerID := strings.TrimSpace(cmd.OwnerID)
	if ownerID == "" {
		return createDesignParams{}, fmt.Errorf("%w: owner_id is required", ErrDesignInvalidInput)
	}

	designType := DesignType(strings.TrimSpace(string(cmd.Type)))
	if designType == "" {
		designType = DesignTypeTyped
	}
	if designType != DesignTypeTyped && designType != DesignTypeUploaded && designType != DesignTypeLogo {
		return createDesignParams{}, fmt.Errorf("%w: unsupported type %q", ErrDesignInvalidInput, designType)
	}

	lines, err := sanitizeTextLines(cmd.TextLines, designType)
	if err != nil {
		return createDesignParams{}, err
	}

	fontID := strings.TrimSpace(cmd.FontID)
	if designType == DesignTypeTyped && fontID == "" {
		return createDesignParams{}, fmt.Errorf("%w: font_id is required for typed designs", ErrDesignInvalidInput)
	}

	materialID := strings.TrimSpace(cmd.MaterialID)
	templateID := strings.TrimSpace(cmd.TemplateID)
	locale := strings.TrimSpace(cmd.Locale)
	if locale == "" {
		locale = defaultLocale
	}

	shape := strings.ToLower(strings.TrimSpace(cmd.Shape))
	if shape == "" {
		shape = defaultShape
	}
	if shape != "round" && shape != "square" {
		return createDesignParams{}, fmt.Errorf("%w: unsupported shape %q", ErrDesignInvalidInput, shape)
	}

	sizeMM := cmd.SizeMM
	if sizeMM <= 0 {
		sizeMM = defaultDesignSize
	}
	if sizeMM < minDesignSizeMM || sizeMM > maxDesignSizeMM {
		return createDesignParams{}, fmt.Errorf("%w: size_mm must be between %.1f and %.1f", ErrDesignInvalidInput, minDesignSizeMM, maxDesignSizeMM)
	}

	label := strings.TrimSpace(cmd.Label)
	if len(label) > maxDesignLabelLen {
		label = label[:maxDesignLabelLen]
	}

	rawName := strings.TrimSpace(cmd.RawName)
	if rawName == "" && len(lines) > 0 {
		rawName = strings.Join(lines, "")
	}

	uploadAsset, err := s.normalizeAssetInput(cmd.Upload, designType == DesignTypeTyped)
	if err != nil {
		return createDesignParams{}, err
	}
	logoAsset, err := s.normalizeAssetInput(cmd.Logo, designType != DesignTypeLogo)
	if err != nil {
		return createDesignParams{}, err
	}

	if designType == DesignTypeUploaded && uploadAsset == nil {
		return createDesignParams{}, fmt.Errorf("%w: upload asset is required for uploaded designs", ErrDesignInvalidInput)
	}
	if designType == DesignTypeLogo && logoAsset == nil {
		return createDesignParams{}, fmt.Errorf("%w: logo asset is required for logo designs", ErrDesignInvalidInput)
	}

	snapshot := cloneSnapshot(cmd.Snapshot)
	metadata := cloneMetadata(cmd.Metadata)

	actorID := strings.TrimSpace(cmd.ActorID)
	if actorID == "" {
		actorID = ownerID
	}

	return createDesignParams{
		ownerID:      ownerID,
		actorID:      actorID,
		label:        label,
		designType:   designType,
		textLines:    lines,
		fontID:       fontID,
		materialID:   materialID,
		templateID:   templateID,
		locale:       locale,
		shape:        shape,
		sizeMM:       sizeMM,
		rawName:      rawName,
		kanjiValue:   cmd.KanjiValue,
		kanjiMapping: cmd.KanjiMappingRef,
		uploadAsset:  uploadAsset,
		logoAsset:    logoAsset,
		snapshot:     snapshot,
		metadata:     metadata,
		idempotency:  strings.TrimSpace(cmd.IdempotencyKey),
	}, nil
}

func (s *designService) planAssets(ctx context.Context, designID, versionID string, params createDesignParams) (assetPlan, error) {
	sourceAsset := cloneAssetReference(params.uploadAsset)
	logoAsset := cloneAssetReference(params.logoAsset)

	// Determine source path for master asset.
	if sourceAsset != nil {
		if sourceAsset.ObjectPath == "" {
			uploadID := sourceAsset.AssetID
			if uploadID == "" {
				uploadID = "upload-" + s.newID()
				sourceAsset.AssetID = uploadID
			}
			fileName := sourceAsset.FileName
			if fileName == "" {
				fileName = "source"
			}
			objectPath, err := storage.BuildObjectPath(storage.PurposeDesignMaster, storage.PathParams{
				DesignID: designID,
				UploadID: uploadID,
				FileName: fileName,
			})
			if err != nil {
				return assetPlan{}, fmt.Errorf("%w: %v", ErrDesignInvalidInput, err)
			}
			sourceAsset.ObjectPath = objectPath
		}
		if sourceAsset.Bucket == "" {
			sourceAsset.Bucket = s.assetsBucket
		}
	}

	vectorPath, previewPath := "", ""
	// Default preview location for any design.
	defaultPreviewPath, err := storage.BuildObjectPath(storage.PurposePreview, storage.PathParams{
		DesignID:  designID,
		VersionID: versionID,
		FileName:  previewFileName,
	})
	if err != nil {
		return assetPlan{}, fmt.Errorf("%w: %v", ErrDesignInvalidInput, err)
	}
	previewPath = defaultPreviewPath

	if params.designType == DesignTypeTyped {
		vectorUploadID := "render-" + versionID
		vector, err := storage.BuildObjectPath(storage.PurposeDesignMaster, storage.PathParams{
			DesignID: designID,
			UploadID: vectorUploadID,
			FileName: vectorFileName,
		})
		if err != nil {
			return assetPlan{}, fmt.Errorf("%w: %v", ErrDesignInvalidInput, err)
		}
		vectorPath = vector
		if sourceAsset == nil {
			sourceAsset = &DesignAssetReference{
				AssetID:     vectorUploadID,
				Bucket:      s.assetsBucket,
				ObjectPath:  vectorPath,
				FileName:    vectorFileName,
				ContentType: "image/svg+xml",
			}
		}

		if s.renderer != nil {
			_, err := s.renderer.Render(ctx, RenderDesignRequest{
				DesignID:     designID,
				VersionID:    versionID,
				Type:         params.designType,
				TextLines:    params.textLines,
				FontID:       params.fontID,
				TemplateID:   params.templateID,
				Locale:       params.locale,
				MaterialID:   params.materialID,
				OutputBucket: s.assetsBucket,
				VectorPath:   vectorPath,
				PreviewPath:  previewPath,
			})
			if err != nil {
				return assetPlan{}, err
			}
		}
	}

	previewURL := buildBucketURL(s.assetsBucket, previewPath)
	thumbnailURL := previewURL

	plan := assetPlan{
		sourcePath:   "",
		vectorPath:   vectorPath,
		previewPath:  previewPath,
		previewURL:   previewURL,
		thumbnailURL: thumbnailURL,
		uploadAsset:  sourceAsset,
		logoAsset:    logoAsset,
	}
	if sourceAsset != nil {
		plan.sourcePath = sourceAsset.ObjectPath
	}
	return plan, nil
}

func (s *designService) normalizeAssetInput(input *DesignAssetInput, optional bool) (*DesignAssetReference, error) {
	if input == nil {
		if optional {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: asset metadata is required", ErrDesignInvalidInput)
	}

	contentType := strings.ToLower(strings.TrimSpace(input.ContentType))
	if contentType == "" {
		return nil, fmt.Errorf("%w: asset content_type is required", ErrDesignInvalidInput)
	}
	if _, ok := allowedUploadContentTypes[contentType]; !ok {
		return nil, fmt.Errorf("%w: content_type %q not allowed", ErrDesignInvalidInput, contentType)
	}
	if input.SizeBytes <= 0 {
		return nil, fmt.Errorf("%w: asset size_bytes must be positive", ErrDesignInvalidInput)
	}
	if input.SizeBytes > maxUploadSizeBytes {
		return nil, fmt.Errorf("%w: asset exceeds maximum size (%d bytes)", ErrDesignInvalidInput, maxUploadSizeBytes)
	}

	ref := &DesignAssetReference{
		AssetID:     strings.TrimSpace(input.AssetID),
		Bucket:      strings.TrimSpace(input.Bucket),
		ObjectPath:  strings.TrimSpace(input.ObjectPath),
		FileName:    strings.TrimSpace(input.FileName),
		ContentType: contentType,
		SizeBytes:   input.SizeBytes,
		Checksum:    strings.TrimSpace(input.Checksum),
	}
	if ref.FileName == "" {
		ref.FileName = "source"
	}
	if ref.Bucket == "" {
		ref.Bucket = s.assetsBucket
	}
	return ref, nil
}

func (s *designService) mapRepositoryError(err error) error {
	if err == nil {
		return nil
	}
	var repoErr repositories.RepositoryError
	if errors.As(err, &repoErr) {
		switch {
		case repoErr.IsNotFound():
			return fmt.Errorf("%w: %v", ErrDesignNotFound, err)
		case repoErr.IsConflict():
			return fmt.Errorf("%w: %v", ErrDesignConflict, err)
		case repoErr.IsUnavailable():
			return fmt.Errorf("%w: %v", ErrDesignRepositoryUnavailable, err)
		}
	}
	return err
}

func (s *designService) mapAIError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ErrAIInvalidInput):
		return fmt.Errorf("%w: %v", ErrDesignInvalidInput, err)
	case errors.Is(err, ErrAIJobNotFound), errors.Is(err, ErrAISuggestionNotFound):
		return fmt.Errorf("%w: %v", ErrDesignNotFound, err)
	}
	var repoErr repositories.RepositoryError
	if errors.As(err, &repoErr) {
		switch {
		case repoErr.IsConflict():
			return fmt.Errorf("%w: %v", ErrDesignConflict, err)
		case repoErr.IsUnavailable():
			return fmt.Errorf("%w: %v", ErrDesignRepositoryUnavailable, err)
		case repoErr.IsNotFound():
			return fmt.Errorf("%w: %v", ErrDesignNotFound, err)
		}
	}
	return err
}

func designEditable(status DesignStatus) bool {
	switch status {
	case DesignStatusDraft, DesignStatusReady:
		return true
	default:
		return false
	}
}

func designDeletable(status DesignStatus) bool {
	switch status {
	case DesignStatusDraft, DesignStatusReady:
		return true
	default:
		return false
	}
}

func designStatusUserMutable(status DesignStatus) bool {
	switch status {
	case DesignStatusDraft, DesignStatusReady:
		return true
	default:
		return false
	}
}

func normalizeDesignStatus(value string) (DesignStatus, error) {
	status := DesignStatus(strings.ToLower(strings.TrimSpace(value)))
	if status == "" {
		return "", fmt.Errorf("%w: status is required", ErrDesignInvalidInput)
	}
	switch status {
	case DesignStatusDraft, DesignStatusReady, DesignStatusOrdered, DesignStatusLocked, DesignStatusDeleted:
		return status, nil
	default:
		return "", fmt.Errorf("%w: unsupported status %q", ErrDesignInvalidInput, status)
	}
}

func mapsEqual(a, b map[string]any) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}

func (s *designService) recordAuditUpdate(ctx context.Context, before, after Design, actorID string) {
	if s.audit == nil {
		return
	}
	diff := map[string]AuditLogDiff{}
	if !strings.EqualFold(before.Label, after.Label) {
		diff["label"] = AuditLogDiff{Before: before.Label, After: after.Label}
	}
	if before.Status != after.Status {
		diff["status"] = AuditLogDiff{Before: string(before.Status), After: string(after.Status)}
	}
	if strings.TrimSpace(before.ThumbnailURL) != strings.TrimSpace(after.ThumbnailURL) {
		diff["thumbnailUrl"] = AuditLogDiff{Before: before.ThumbnailURL, After: after.ThumbnailURL}
	}
	metadata := map[string]any{
		"version":          after.Version,
		"currentVersionId": after.CurrentVersionID,
		"snapshotUpdated":  !mapsEqual(before.Snapshot, after.Snapshot),
	}
	record := AuditLogRecord{
		Actor:      actorID,
		ActorType:  "user",
		Action:     "design.update",
		TargetRef:  fmt.Sprintf("/designs/%s", after.ID),
		Severity:   "info",
		OccurredAt: after.UpdatedAt,
		Metadata:   metadata,
		Diff:       diff,
	}
	s.audit.Record(ctx, record)
}

func (s *designService) recordAuditDelete(ctx context.Context, design Design, actorID string, deletedAt time.Time) {
	if s.audit == nil {
		return
	}
	diff := map[string]AuditLogDiff{
		"status": {
			Before: string(design.Status),
			After:  string(DesignStatusDeleted),
		},
	}
	metadata := map[string]any{
		"deletedAt": deletedAt,
		"soft":      true,
	}
	record := AuditLogRecord{
		Actor:      actorID,
		ActorType:  "user",
		Action:     "design.delete",
		TargetRef:  fmt.Sprintf("/designs/%s", design.ID),
		Severity:   "info",
		OccurredAt: deletedAt,
		Metadata:   metadata,
		Diff:       diff,
	}
	s.audit.Record(ctx, record)
}

func (s *designService) recordAudit(ctx context.Context, design Design, params createDesignParams) {
	if s.audit == nil {
		return
	}
	metadata := map[string]any{
		"type":       string(design.Type),
		"status":     string(design.Status),
		"textLines":  cloneStrings(design.TextLines),
		"materialId": design.MaterialID,
		"templateId": design.Template,
	}
	record := AuditLogRecord{
		Actor:      params.actorID,
		ActorType:  "user",
		Action:     "design.create",
		TargetRef:  fmt.Sprintf("/designs/%s", design.ID),
		Severity:   "info",
		OccurredAt: design.CreatedAt,
		Metadata:   metadata,
	}
	s.audit.Record(ctx, record)
}

func (s *designService) recordAuditDuplicate(ctx context.Context, source, duplicate Design, actorID string) {
	if s.audit == nil {
		return
	}
	metadata := map[string]any{
		"sourceDesignId": source.ID,
		"newVersionId":   duplicate.CurrentVersionID,
		"type":           string(duplicate.Type),
	}
	record := AuditLogRecord{
		Actor:      actorID,
		ActorType:  "user",
		Action:     "design.duplicate",
		TargetRef:  fmt.Sprintf("/designs/%s", duplicate.ID),
		Severity:   "info",
		OccurredAt: duplicate.CreatedAt,
		Metadata:   metadata,
	}
	s.audit.Record(ctx, record)
}

func buildDesignSnapshot(label string, params createDesignParams, plan assetPlan) map[string]any {
	snapshot := cloneSnapshot(params.snapshot)
	if snapshot == nil {
		snapshot = make(map[string]any)
	}
	snapshot["label"] = label
	snapshot["type"] = string(params.designType)
	snapshot["status"] = string(DesignStatusDraft)
	snapshot["textLines"] = cloneStrings(params.textLines)
	if params.fontID != "" {
		snapshot["fontId"] = params.fontID
	}
	if params.materialID != "" {
		snapshot["materialId"] = params.materialID
	}
	if params.templateID != "" {
		snapshot["templateId"] = params.templateID
	}
	snapshot["locale"] = params.locale
	snapshot["shape"] = params.shape
	snapshot["sizeMm"] = params.sizeMM

	source := map[string]any{
		"type":      string(params.designType),
		"rawName":   params.rawName,
		"textLines": cloneStrings(params.textLines),
	}
	if params.kanjiValue != nil || params.kanjiMapping != nil {
		kanji := make(map[string]any)
		if params.kanjiValue != nil {
			kanji["value"] = strings.TrimSpace(*params.kanjiValue)
		}
		if params.kanjiMapping != nil {
			kanji["mappingRef"] = strings.TrimSpace(*params.kanjiMapping)
		}
		source["kanji"] = kanji
	}
	if plan.uploadAsset != nil {
		source["uploadAsset"] = assetReferenceSnapshot(plan.uploadAsset)
	}
	if plan.logoAsset != nil {
		source["logoAsset"] = assetReferenceSnapshot(plan.logoAsset)
	}
	snapshot["source"] = source

	assets := map[string]any{
		"previewPath": plan.previewPath,
		"previewUrl":  plan.previewURL,
	}
	if plan.sourcePath != "" {
		assets["sourcePath"] = plan.sourcePath
	}
	if plan.vectorPath != "" {
		assets["vectorPath"] = plan.vectorPath
	}
	snapshot["assets"] = assets

	if len(params.metadata) > 0 {
		snapshot["metadata"] = cloneMetadata(params.metadata)
	}
	return snapshot
}

func sanitizeTextLines(lines []string, designType DesignType) ([]string, error) {
	if len(lines) == 0 {
		if designType == DesignTypeTyped {
			return nil, fmt.Errorf("%w: text_lines are required for typed designs", ErrDesignInvalidInput)
		}
		return nil, nil
	}
	if len(lines) > maxDesignTextLines {
		return nil, fmt.Errorf("%w: maximum of %d text lines supported", ErrDesignInvalidInput, maxDesignTextLines)
	}

	sanitized := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if len([]rune(trimmed)) > maxDesignLineChars {
			return nil, fmt.Errorf("%w: text line exceeds %d characters", ErrDesignInvalidInput, maxDesignLineChars)
		}
		if containsBannedWords(trimmed) {
			return nil, fmt.Errorf("%w: text contains prohibited content", ErrDesignInvalidInput)
		}
		sanitized = append(sanitized, trimmed)
	}
	if designType == DesignTypeTyped && len(sanitized) == 0 {
		return nil, fmt.Errorf("%w: at least one non-empty text line required", ErrDesignInvalidInput)
	}
	return sanitized, nil
}

func containsBannedWords(value string) bool {
	if value == "" {
		return false
	}
	lower := strings.ToLower(value)
	for _, banned := range bannedDesignWords {
		trimmed := strings.TrimSpace(banned)
		if trimmed == "" {
			continue
		}
		if strings.Contains(lower, strings.ToLower(trimmed)) || strings.Contains(value, trimmed) {
			return true
		}
	}
	return false
}

func defaultDesignLabel(designID string, designType DesignType, lines []string) string {
	if len(lines) > 0 {
		label := strings.Join(lines, " ")
		if len(label) > maxDesignLabelLen {
			return label[:maxDesignLabelLen]
		}
		return label
	}
	suffix := designID
	if len(suffix) > 6 {
		suffix = suffix[len(suffix)-6:]
	}
	return fmt.Sprintf("Design %s", strings.ToUpper(suffix))
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return slices.Clone(values)
}

func cloneSnapshot(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	return maps.Clone(src)
}

func stripSnapshotAssets(snapshot map[string]any) map[string]any {
	if len(snapshot) == 0 {
		return nil
	}
	copy := cloneSnapshot(snapshot)
	if copy == nil {
		return nil
	}
	delete(copy, "assets")
	if len(copy) == 0 {
		return nil
	}
	return copy
}

func cloneMetadata(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	return maps.Clone(src)
}

func suggestionIDFromKey(key string) string {
	sum := sha1.Sum([]byte(strings.ToLower(strings.TrimSpace(key))))
	return hex.EncodeToString(sum[:8])
}

func normalizeSuggestionStatusFilters(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if trimmed := strings.ToLower(strings.TrimSpace(value)); trimmed != "" {
			if _, exists := seen[trimmed]; exists {
				continue
			}
			seen[trimmed] = struct{}{}
			normalized = append(normalized, trimmed)
		}
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func expandSuggestionStatusFilters(filters []string) []string {
	if len(filters) == 0 {
		return nil
	}
	expanded := make([]string, 0, len(filters))
	seen := make(map[string]struct{}, len(filters)*2)
	add := func(values ...string) {
		for _, value := range values {
			trimmed := strings.ToLower(strings.TrimSpace(value))
			if trimmed == "" {
				continue
			}
			if _, exists := seen[trimmed]; exists {
				continue
			}
			seen[trimmed] = struct{}{}
			expanded = append(expanded, trimmed)
		}
	}
	for _, filter := range filters {
		switch filter {
		case "queued":
			add("queued", "pending", "in_progress")
		case "completed":
			add("proposed", "accepted", "applied", "succeeded", "completed")
		case "rejected":
			add("rejected", "expired", "failed", "canceled")
		default:
			add(filter)
		}
	}
	if len(expanded) == 0 {
		return nil
	}
	return expanded
}

var suggestionRejectionReasons = map[string]struct{}{
	"not_needed":      {},
	"worse_quality":   {},
	"user_preference": {},
	"invalid":         {},
	"other":           {},
}

func suggestionStatusCategory(status string) string {
	current := strings.ToLower(strings.TrimSpace(status))
	switch current {
	case "queued", "pending", "in_progress":
		return "queued"
	case "proposed", "accepted", "applied", "succeeded", "completed":
		return "completed"
	case "rejected", "expired", "failed", "canceled":
		return "rejected"
	case "":
		return "queued"
	default:
		return current
	}
}

type suggestionPreviewInfo struct {
	PreviewURL   string
	Bucket       string
	ObjectPath   string
	ThumbnailURL string
}

func extractSuggestionPreviewInfo(payload map[string]any) suggestionPreviewInfo {
	info := suggestionPreviewInfo{}
	if len(payload) == 0 {
		return info
	}
	sources := []map[string]any{}
	if result := mapFromAny(payload["result"]); len(result) > 0 {
		sources = append(sources, result)
	}
	sources = append(sources, payload)
	for _, src := range sources {
		if len(src) == 0 {
			continue
		}
		if previewMap := mapFromAny(src["preview"]); len(previewMap) > 0 {
			if info.PreviewURL == "" {
				info.PreviewURL = firstNonEmptyString(
					stringFromAny(previewMap["previewUrl"]),
					stringFromAny(previewMap["signedPreviewUrl"]),
					stringFromAny(previewMap["signedUrl"]),
				)
			}
			if info.Bucket == "" {
				info.Bucket = stringFromAny(previewMap["bucket"])
			}
			if info.ObjectPath == "" {
				info.ObjectPath = stringFromAny(previewMap["objectPath"])
			}
			if info.ThumbnailURL == "" {
				info.ThumbnailURL = stringFromAny(previewMap["thumbnailUrl"])
			}
		}
		if info.PreviewURL == "" {
			info.PreviewURL = firstNonEmptyString(info.PreviewURL, stringFromAny(src["previewUrl"]))
		}
		if info.Bucket == "" {
			info.Bucket = stringFromAny(src["bucket"])
		}
		if info.ObjectPath == "" {
			info.ObjectPath = stringFromAny(src["objectPath"])
		}
		if info.ThumbnailURL == "" {
			info.ThumbnailURL = stringFromAny(src["thumbnailUrl"])
		}
		if info.PreviewURL != "" && (info.ObjectPath != "" || info.ThumbnailURL != "") {
			break
		}
	}
	if info.PreviewURL == "" && info.ThumbnailURL != "" {
		info.PreviewURL = info.ThumbnailURL
	}
	return info
}

func mapFromAny(value any) map[string]any {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case map[string]any:
		if len(v) == 0 {
			return nil
		}
		return v
	default:
		return nil
	}
}

func stringFromAny(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

func cloneAssetReference(ref *DesignAssetReference) *DesignAssetReference {
	if ref == nil {
		return nil
	}
	copy := *ref
	return &copy
}

func assetReferenceSnapshot(ref *DesignAssetReference) map[string]any {
	if ref == nil {
		return nil
	}
	snapshot := map[string]any{
		"assetId":     ref.AssetID,
		"bucket":      ref.Bucket,
		"objectPath":  ref.ObjectPath,
		"fileName":    ref.FileName,
		"contentType": ref.ContentType,
		"sizeBytes":   ref.SizeBytes,
	}
	if ref.Checksum != "" {
		snapshot["checksum"] = ref.Checksum
	}
	return snapshot
}

func buildBucketURL(bucket, object string) string {
	if strings.TrimSpace(bucket) == "" || strings.TrimSpace(object) == "" {
		return ""
	}
	object = strings.TrimPrefix(object, "/")
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, object)
}
