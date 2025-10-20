package services

import (
	"context"
	"errors"
	"fmt"
	"maps"
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
	designIDPrefix     = "dsg_"
	versionIDPrefix    = "ver_"
	defaultLocale      = "ja-JP"
	defaultShape       = "round"
	minDesignSizeMM    = 6.0
	maxDesignSizeMM    = 30.0
	defaultDesignSize  = 15.0
	maxDesignTextLines = 4
	maxDesignLineChars = 32
	maxDesignLabelLen  = 120
	maxUploadSizeBytes = int64(20 * 1024 * 1024)
	previewFileName    = "preview.png"
	vectorFileName     = "design.svg"
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
	Designs      repositories.DesignRepository
	Versions     repositories.DesignVersionRepository
	Audit        AuditLogService
	Renderer     DesignRenderer
	UnitOfWork   repositories.UnitOfWork
	Clock        func() time.Time
	IDGenerator  func() string
	AssetsBucket string
	Logger       func(context.Context, string, map[string]any)
}

type designService struct {
	designs      repositories.DesignRepository
	versions     repositories.DesignVersionRepository
	audit        AuditLogService
	renderer     DesignRenderer
	unitOfWork   repositories.UnitOfWork
	clock        func() time.Time
	newID        func() string
	assetsBucket string
	logger       func(context.Context, string, map[string]any)
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

	return &designService{
		designs:      deps.Designs,
		versions:     deps.Versions,
		audit:        deps.Audit,
		renderer:     deps.Renderer,
		unitOfWork:   deps.UnitOfWork,
		clock:        func() time.Time { return clock().UTC() },
		newID:        idGen,
		assetsBucket: bucket,
		logger:       logger,
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
func (s *designService) GetDesign(ctx context.Context, designID string, _ DesignReadOptions) (Design, error) {
	if s.designs == nil {
		return Design{}, ErrDesignRepositoryUnavailable
	}
	design, err := s.designs.FindByID(ctx, designID)
	if err != nil {
		return Design{}, s.mapRepositoryError(err)
	}
	return design, nil
}

// ListDesigns returns designs owned by a user filtered by status.
func (s *designService) ListDesigns(ctx context.Context, filter DesignListFilter) (domain.CursorPage[Design], error) {
	if s.designs == nil {
		return domain.CursorPage[Design]{}, ErrDesignRepositoryUnavailable
	}
	page, err := s.designs.ListByOwner(ctx, filter.OwnerID, repositories.DesignListFilter{
		Status:     filter.Status,
		Pagination: filter.Pagination,
	})
	if err != nil {
		return domain.CursorPage[Design]{}, s.mapRepositoryError(err)
	}
	return page, nil
}

func (s *designService) UpdateDesign(context.Context, UpdateDesignCommand) (Design, error) {
	return Design{}, ErrDesignNotImplemented
}

func (s *designService) DeleteDesign(context.Context, DeleteDesignCommand) error {
	return ErrDesignNotImplemented
}

func (s *designService) DuplicateDesign(context.Context, DuplicateDesignCommand) (Design, error) {
	return Design{}, ErrDesignNotImplemented
}

func (s *designService) RequestAISuggestion(context.Context, AISuggestionRequest) (AISuggestion, error) {
	return AISuggestion{}, ErrDesignNotImplemented
}

func (s *designService) ListAISuggestions(context.Context, string, AISuggestionFilter) (domain.CursorPage[AISuggestion], error) {
	return domain.CursorPage[AISuggestion]{}, ErrDesignNotImplemented
}

func (s *designService) UpdateAISuggestionStatus(context.Context, AISuggestionStatusCommand) (AISuggestion, error) {
	return AISuggestion{}, ErrDesignNotImplemented
}

func (s *designService) RequestRegistrabilityCheck(context.Context, RegistrabilityCheckCommand) (RegistrabilityCheckResult, error) {
	return RegistrabilityCheckResult{}, ErrDesignNotImplemented
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

func cloneMetadata(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	return maps.Clone(src)
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
