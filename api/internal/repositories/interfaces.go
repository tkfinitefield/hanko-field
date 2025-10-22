package repositories

import (
	"context"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
)

// Registry exposes typed repository accessors and lifecycle hooks for dependency injection.
type Registry interface {
	Close(ctx context.Context) error

	Designs() DesignRepository
	DesignVersions() DesignVersionRepository
	Registrability() RegistrabilityRepository
	AISuggestions() AISuggestionRepository
	AIJobs() AIJobRepository
	Carts() CartRepository
	Inventory() InventoryRepository
	Orders() OrderRepository
	Reviews() ReviewRepository
	OrderPayments() OrderPaymentRepository
	OrderShipments() OrderShipmentRepository
	OrderProductionEvents() OrderProductionEventRepository
	Promotions() PromotionRepository
	PromotionUsage() PromotionUsageRepository
	Users() UserRepository
	Addresses() AddressRepository
	PaymentMethods() PaymentMethodRepository
	Favorites() FavoriteRepository
	Catalog() CatalogRepository
	Content() ContentRepository
	Assets() AssetRepository
	AuditLogs() AuditLogRepository
	NameMappings() NameMappingRepository
	Counters() CounterRepository
	Health() HealthRepository
	UnitOfWork
}

// RepositoryError wraps low-level persistence failures with categorisation used by services.
type RepositoryError interface {
	error
	IsNotFound() bool
	IsConflict() bool
	IsUnavailable() bool
}

// UnitOfWork allows grouping repository operations in a transactional boundary when supported.
type UnitOfWork interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// DesignRepository persists design documents and related metadata.
type DesignRepository interface {
	Insert(ctx context.Context, design domain.Design) error
	Update(ctx context.Context, design domain.Design) error
	SoftDelete(ctx context.Context, designID string, deletedAt time.Time) error
	FindByID(ctx context.Context, designID string) (domain.Design, error)
	ListByOwner(ctx context.Context, ownerID string, filter DesignListFilter) (domain.CursorPage[domain.Design], error)
}

// DesignVersionRepository handles immutable snapshots for designs.
type DesignVersionRepository interface {
	Append(ctx context.Context, version domain.DesignVersion) error
	ListByDesign(ctx context.Context, designID string, pager domain.Pagination) (domain.CursorPage[domain.DesignVersion], error)
	FindByID(ctx context.Context, designID string, versionID string) (domain.DesignVersion, error)
}

// RegistrabilityRepository persists registrability assessment results for designs.
type RegistrabilityRepository interface {
	Get(ctx context.Context, designID string) (domain.RegistrabilityCheckResult, error)
	Save(ctx context.Context, result domain.RegistrabilityCheckResult) error
}

// AISuggestionRepository stores AI suggestion records and status transitions.
type AISuggestionRepository interface {
	Insert(ctx context.Context, suggestion domain.AISuggestion) error
	FindByID(ctx context.Context, designID string, suggestionID string) (domain.AISuggestion, error)
	UpdateStatus(ctx context.Context, designID string, suggestionID string, status string, metadata map[string]any) (domain.AISuggestion, error)
	ListByDesign(ctx context.Context, designID string, filter AISuggestionListFilter) (domain.CursorPage[domain.AISuggestion], error)
}

// AIJobRepository persists AI job metadata and lifecycle state.
type AIJobRepository interface {
	Insert(ctx context.Context, job domain.AIJob) (domain.AIJob, error)
	FindByID(ctx context.Context, jobID string) (domain.AIJob, error)
	FindByIdempotencyKey(ctx context.Context, key string) (domain.AIJob, error)
	UpdateStatus(ctx context.Context, jobID string, status domain.AIJobStatus, update AIJobStatusUpdate) (domain.AIJob, error)
}

// AIJobStatusUpdate carries optional fields to mutate during a status transition.
type AIJobStatusUpdate struct {
	Payload     map[string]any
	ResultRef   *string
	Error       *domain.AIJobError
	Attempt     *domain.AIJobAttempt
	LockedBy    *string
	LockedAt    *time.Time
	CompletedAt *time.Time
	ExpiresAt   *time.Time
	Metadata    map[string]any
}

// CartRepository owns cart header + items persistence with optimistic locking guarantees.
type CartRepository interface {
	UpsertCart(ctx context.Context, cart domain.Cart) (domain.Cart, error)
	GetCart(ctx context.Context, userID string) (domain.Cart, error)
	ReplaceItems(ctx context.Context, userID string, items []domain.CartItem) (domain.Cart, error)
}

// InventoryRepository manages stock levels and reservation lifecycle with transactional guarantees.
type InventoryRepository interface {
	Reserve(ctx context.Context, req InventoryReserveRequest) (InventoryReserveResult, error)
	Commit(ctx context.Context, req InventoryCommitRequest) (InventoryCommitResult, error)
	Release(ctx context.Context, req InventoryReleaseRequest) (InventoryReleaseResult, error)
	GetReservation(ctx context.Context, reservationID string) (domain.InventoryReservation, error)
	ListLowStock(ctx context.Context, query InventoryLowStockQuery) (domain.CursorPage[domain.InventoryStock], error)
}

// InventoryReserveRequest encapsulates reservation creation metadata for the repository.
type InventoryReserveRequest struct {
	Reservation domain.InventoryReservation
	Now         time.Time
}

// InventoryReserveResult returns the saved reservation and updated stock projections.
type InventoryReserveResult struct {
	Reservation domain.InventoryReservation
	Stocks      map[string]domain.InventoryStock
}

// InventoryCommitRequest finalises a reservation and decrements on-hand counts.
type InventoryCommitRequest struct {
	ReservationID string
	OrderRef      string
	Now           time.Time
}

// InventoryCommitResult reports the updated reservation and stock metrics after commit.
type InventoryCommitResult struct {
	Reservation domain.InventoryReservation
	Stocks      map[string]domain.InventoryStock
}

// InventoryReleaseRequest restores reserved stock back to availability.
type InventoryReleaseRequest struct {
	ReservationID string
	Reason        string
	Now           time.Time
}

// InventoryReleaseResult reports the reservation and stock metrics after release.
type InventoryReleaseResult struct {
	Reservation domain.InventoryReservation
	Stocks      map[string]domain.InventoryStock
}

// InventoryLowStockQuery controls pagination and threshold filtering for low stock listings.
type InventoryLowStockQuery struct {
	Threshold int
	PageSize  int
	PageToken string
}

// OrderRepository persists order headers and provides query helpers for users and admins.
type OrderRepository interface {
	Insert(ctx context.Context, order domain.Order) error
	Update(ctx context.Context, order domain.Order) error
	FindByID(ctx context.Context, orderID string) (domain.Order, error)
	List(ctx context.Context, filter OrderListFilter) (domain.CursorPage[domain.Order], error)
}

// OrderPaymentRepository stores payment records underneath an order document.
type OrderPaymentRepository interface {
	Insert(ctx context.Context, payment domain.Payment) error
	Update(ctx context.Context, payment domain.Payment) error
	List(ctx context.Context, orderID string) ([]domain.Payment, error)
}

// OrderShipmentRepository stores fulfillment data for orders.
type OrderShipmentRepository interface {
	Insert(ctx context.Context, shipment domain.Shipment) error
	Update(ctx context.Context, shipment domain.Shipment) error
	List(ctx context.Context, orderID string) ([]domain.Shipment, error)
}

// ReviewRepository stores product reviews and their moderation meta.
type ReviewRepository interface {
	Insert(ctx context.Context, review domain.Review) (domain.Review, error)
	FindByID(ctx context.Context, reviewID string) (domain.Review, error)
	FindByOrder(ctx context.Context, orderID string) (domain.Review, error)
	ListByUser(ctx context.Context, userID string, pager domain.Pagination) (domain.CursorPage[domain.Review], error)
	UpdateStatus(ctx context.Context, reviewID string, status domain.ReviewStatus, update ReviewModerationUpdate) (domain.Review, error)
	UpdateReply(ctx context.Context, reviewID string, reply *domain.ReviewReply, updatedAt time.Time) (domain.Review, error)
}

// OrderProductionEventRepository stores production timeline events for an order.
type OrderProductionEventRepository interface {
	Insert(ctx context.Context, event domain.OrderProductionEvent) (domain.OrderProductionEvent, error)
	List(ctx context.Context, orderID string) ([]domain.OrderProductionEvent, error)
}

// PromotionRepository maintains promotion definitions and usage counters.
type PromotionRepository interface {
	Insert(ctx context.Context, promotion domain.Promotion) error
	Update(ctx context.Context, promotion domain.Promotion) error
	Delete(ctx context.Context, promotionID string) error
	FindByCode(ctx context.Context, code string) (domain.Promotion, error)
	List(ctx context.Context, filter PromotionListFilter) (domain.CursorPage[domain.Promotion], error)
}

// PromotionUsageRepository records per-user usage counts to enforce limits.
type PromotionUsageRepository interface {
	IncrementUsage(ctx context.Context, promoID string, userID string, now time.Time) (domain.PromotionUsage, error)
	RemoveUsage(ctx context.Context, promoID string, userID string) error
	ListUsage(ctx context.Context, promoID string, pager domain.Pagination) (domain.CursorPage[domain.PromotionUsage], error)
}

// UserRepository stores user profiles and supports masking/deactivation flows.
type UserRepository interface {
	FindByID(ctx context.Context, userID string) (domain.UserProfile, error)
	UpdateProfile(ctx context.Context, profile domain.UserProfile) (domain.UserProfile, error)
}

// AddressRepository stores shipping addresses per user.
type AddressRepository interface {
	List(ctx context.Context, userID string) ([]domain.Address, error)
	Upsert(ctx context.Context, userID string, addressID *string, addr domain.Address) (domain.Address, error)
	Delete(ctx context.Context, userID string, addressID string) error
	Get(ctx context.Context, userID string, addressID string) (domain.Address, error)
	FindByHash(ctx context.Context, userID string, hash string) (domain.Address, bool, error)
	HasAny(ctx context.Context, userID string) (bool, error)
	SetDefaultFlags(ctx context.Context, userID string, addressID string, shipping, billing *bool) (domain.Address, error)
}

// PaymentMethodRepository stores PSP reference tokens per user.
type PaymentMethodRepository interface {
	List(ctx context.Context, userID string) ([]domain.PaymentMethod, error)
	Insert(ctx context.Context, userID string, method domain.PaymentMethod) (domain.PaymentMethod, error)
	Delete(ctx context.Context, userID string, paymentMethodID string) error
	Get(ctx context.Context, userID string, paymentMethodID string) (domain.PaymentMethod, error)
	SetDefault(ctx context.Context, userID string, paymentMethodID string) (domain.PaymentMethod, error)
}

// FavoriteRepository tracks favorite designs per user.
type FavoriteRepository interface {
	List(ctx context.Context, userID string, pager domain.Pagination) (domain.CursorPage[domain.FavoriteDesign], error)
	Put(ctx context.Context, userID string, designID string, addedAt time.Time, limit int) (bool, error)
	Delete(ctx context.Context, userID string, designID string) error
}

// NameMappingRepository persists transliteration results and selection state.
type NameMappingRepository interface {
	Insert(ctx context.Context, mapping domain.NameMapping) error
	Update(ctx context.Context, mapping domain.NameMapping) error
	FindByID(ctx context.Context, mappingID string) (domain.NameMapping, error)
	FindByLookup(ctx context.Context, userID string, latin string, locale string) (domain.NameMapping, error)
}

// CatalogRepository bundles template/font/material/product storage with shared transactions.
type CatalogRepository interface {
	ListTemplates(ctx context.Context, filter TemplateFilter) (domain.CursorPage[domain.TemplateSummary], error)
	GetPublishedTemplate(ctx context.Context, templateID string) (domain.Template, error)
	GetTemplate(ctx context.Context, templateID string) (domain.Template, error)
	UpsertTemplate(ctx context.Context, template domain.Template) (domain.Template, error)
	DeleteTemplate(ctx context.Context, templateID string) error

	// ListFonts returns a paginated collection of fonts respecting the provided filter.
	ListFonts(ctx context.Context, filter FontFilter) (domain.CursorPage[domain.FontSummary], error)
	// GetPublishedFont retrieves a single font that is publicly published. Should return a RepositoryError with
	// IsNotFound when the font is absent or unpublished.
	GetPublishedFont(ctx context.Context, fontID string) (domain.Font, error)
	// GetFont retrieves a font regardless of publication state (for admin/internal usage).
	GetFont(ctx context.Context, fontID string) (domain.Font, error)
	UpsertFont(ctx context.Context, font domain.FontSummary) (domain.FontSummary, error)
	DeleteFont(ctx context.Context, fontID string) error

	ListMaterials(ctx context.Context, filter MaterialFilter) (domain.CursorPage[domain.MaterialSummary], error)
	GetPublishedMaterial(ctx context.Context, materialID string) (domain.Material, error)
	GetMaterial(ctx context.Context, materialID string) (domain.Material, error)
	UpsertMaterial(ctx context.Context, material domain.MaterialSummary) (domain.MaterialSummary, error)
	DeleteMaterial(ctx context.Context, materialID string) error

	ListProducts(ctx context.Context, filter ProductFilter) (domain.CursorPage[domain.ProductSummary], error)
	GetPublishedProduct(ctx context.Context, productID string) (domain.Product, error)
	GetProduct(ctx context.Context, productID string) (domain.Product, error)
	UpsertProduct(ctx context.Context, product domain.ProductSummary) (domain.ProductSummary, error)
	DeleteProduct(ctx context.Context, productID string) error
}

// ContentRepository stores CMS-managed guides and pages.
type ContentRepository interface {
	ListGuides(ctx context.Context, filter ContentGuideFilter) (domain.CursorPage[domain.ContentGuide], error)
	UpsertGuide(ctx context.Context, guide domain.ContentGuide) (domain.ContentGuide, error)
	DeleteGuide(ctx context.Context, guideID string) error
	GetGuideBySlug(ctx context.Context, slug string, locale string) (domain.ContentGuide, error)
	GetGuide(ctx context.Context, guideID string) (domain.ContentGuide, error)

	GetPage(ctx context.Context, slug string, locale string) (domain.ContentPage, error)
	UpsertPage(ctx context.Context, page domain.ContentPage) (domain.ContentPage, error)
	DeletePage(ctx context.Context, pageID string) error
}

// AssetRepository handles metadata synchronized with Cloud Storage objects.
type AssetRepository interface {
	CreateSignedUpload(ctx context.Context, cmd SignedUploadRecord) (domain.SignedAssetResponse, error)
	CreateSignedDownload(ctx context.Context, cmd SignedDownloadRecord) (domain.SignedAssetResponse, error)
	MarkUploaded(ctx context.Context, assetID string, actorID string, metadata map[string]any) error
}

// AuditLogRepository persists immutable audit trail entries.
type AuditLogRepository interface {
	Append(ctx context.Context, entry domain.AuditLogEntry) error
	List(ctx context.Context, filter AuditLogFilter) (domain.CursorPage[domain.AuditLogEntry], error)
}

// CounterRepository provides transaction-safe sequence numbers.
type CounterRepository interface {
	Next(ctx context.Context, counterID string, step int64) (int64, error)
	Configure(ctx context.Context, counterID string, cfg CounterConfig) error
}

// HealthRepository exposes status of downstream dependencies for health checks.
type HealthRepository interface {
	Collect(ctx context.Context) (domain.SystemHealthReport, error)
}

// Filter DTOs shared across repositories ------------------------------------

type DesignListFilter struct {
	Status       []string
	Types        []string
	UpdatedAfter *time.Time
	Pagination   domain.Pagination
}

type AISuggestionListFilter struct {
	Status     []string
	Pagination domain.Pagination
}

type OrderListFilter struct {
	UserID     string
	Status     []string
	DateRange  domain.RangeQuery[time.Time]
	Pagination domain.Pagination
}

type PromotionListFilter struct {
	Status     []string
	Pagination domain.Pagination
}

// ReviewModerationUpdate carries moderation metadata for status transitions.
type ReviewModerationUpdate struct {
	ModeratedBy string
	ModeratedAt time.Time
}

type TemplateFilter struct {
	Category      *string
	Style         *string
	Tags          []string
	OnlyPublished bool
	SortBy        domain.TemplateSort
	SortOrder     domain.SortOrder
	Pagination    domain.Pagination
}

type FontFilter struct {
	Script        *string
	IsPremium     *bool
	PublishedOnly bool
	Pagination    domain.Pagination
}

type MaterialFilter struct {
	Category    *string
	IsAvailable *bool
	Locale      string
	Pagination  domain.Pagination
}

type ProductFilter struct {
	Shape          *string
	SizeMm         *int
	MaterialID     *string
	IsCustomizable *bool
	OnlyPublished  bool
	Pagination     domain.Pagination
}

type ContentGuideFilter struct {
	Category       *string
	Slug           *string
	Locale         *string
	FallbackLocale string
	Status         []string
	OnlyPublished  bool
	Pagination     domain.Pagination
}

type AuditLogFilter struct {
	TargetRef  string
	Actor      string
	ActorType  string
	Action     string
	DateRange  domain.RangeQuery[time.Time]
	Pagination domain.Pagination
}

type SignedUploadRecord struct {
	ActorID     string
	DesignID    *string
	Kind        string
	Purpose     string
	FileName    string
	ContentType string
	SizeBytes   int64
}

type SignedDownloadRecord struct {
	ActorID string
	AssetID string
}

// CounterConfig customises increment behaviour and bounds for a counter.
type CounterConfig struct {
	Step         int64
	MaxValue     *int64
	InitialValue *int64
}
