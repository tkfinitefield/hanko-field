package services

import (
	"context"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

// Type aliases expose domain models to the services package without reversing dependency direction.
type (
	Pagination                = domain.Pagination
	SortOrder                 = domain.SortOrder
	Design                    = domain.Design
	DesignVersion             = domain.DesignVersion
	AISuggestion              = domain.AISuggestion
	Cart                      = domain.Cart
	CartPromotion             = domain.CartPromotion
	CartItem                  = domain.CartItem
	CartEstimate              = domain.CartEstimate
	PricingBreakdown          = domain.PricingBreakdown
	ItemPricingBreakdown      = domain.ItemPricingBreakdown
	DiscountBreakdown         = domain.DiscountBreakdown
	TaxBreakdown              = domain.TaxBreakdown
	ShippingBreakdown         = domain.ShippingBreakdown
	CheckoutSession           = domain.CheckoutSession
	Order                     = domain.Order
	OrderTotals               = domain.OrderTotals
	OrderLineItem             = domain.OrderLineItem
	OrderProductionEvent      = domain.OrderProductionEvent
	OrderProductionQC         = domain.OrderProductionQC
	OrderStatus               = domain.OrderStatus
	OrderContact              = domain.OrderContact
	OrderFulfillment          = domain.OrderFulfillment
	OrderProduction           = domain.OrderProduction
	OrderFlags                = domain.OrderFlags
	OrderAudit                = domain.OrderAudit
	Payment                   = domain.Payment
	Shipment                  = domain.Shipment
	ShipmentEvent             = domain.ShipmentEvent
	Review                    = domain.Review
	ReviewReply               = domain.ReviewReply
	ReviewStatus              = domain.ReviewStatus
	Promotion                 = domain.Promotion
	PromotionValidationResult = domain.PromotionValidationResult
	RegistrabilityCheckResult = domain.RegistrabilityCheckResult
	Address                   = domain.Address
	UserProfile               = domain.UserProfile
	FavoriteDesign            = domain.FavoriteDesign
	InventoryReservationLine  = domain.InventoryReservationLine
	InventoryReservation      = domain.InventoryReservation
	InventorySnapshot         = domain.InventorySnapshot
	InventoryStock            = domain.InventoryStock
	InventoryStockEvent       = domain.InventoryStockEvent
	ContentPage               = domain.ContentPage
	ContentGuide              = domain.ContentGuide
	TemplateSummary           = domain.TemplateSummary
	FontSummary               = domain.FontSummary
	MaterialSummary           = domain.MaterialSummary
	ProductSummary            = domain.ProductSummary
	SystemHealthReport        = domain.SystemHealthReport
	AuditLogEntry             = domain.AuditLogEntry
	SignedAssetResponse       = domain.SignedAssetResponse
	PromotionUsage            = domain.PromotionUsage
	PaymentMethod             = domain.PaymentMethod
)

// DesignService orchestrates design lifecycle operations, coordinating repositories,
// validation, and asynchronous AI suggestion workflows.
type DesignService interface {
	CreateDesign(ctx context.Context, cmd CreateDesignCommand) (Design, error)
	GetDesign(ctx context.Context, designID string, opts DesignReadOptions) (Design, error)
	ListDesigns(ctx context.Context, filter DesignListFilter) (domain.CursorPage[Design], error)
	UpdateDesign(ctx context.Context, cmd UpdateDesignCommand) (Design, error)
	DeleteDesign(ctx context.Context, cmd DeleteDesignCommand) error
	DuplicateDesign(ctx context.Context, cmd DuplicateDesignCommand) (Design, error)
	RequestAISuggestion(ctx context.Context, cmd AISuggestionRequest) (AISuggestion, error)
	ListAISuggestions(ctx context.Context, designID string, filter AISuggestionFilter) (domain.CursorPage[AISuggestion], error)
	UpdateAISuggestionStatus(ctx context.Context, cmd AISuggestionStatusCommand) (AISuggestion, error)
	RequestRegistrabilityCheck(ctx context.Context, cmd RegistrabilityCheckCommand) (RegistrabilityCheckResult, error)
}

// CartService manages mutable cart state and estimates while enforcing inventory rules.
type CartService interface {
	GetOrCreateCart(ctx context.Context, userID string) (Cart, error)
	AddOrUpdateItem(ctx context.Context, cmd UpsertCartItemCommand) (Cart, error)
	RemoveItem(ctx context.Context, cmd RemoveCartItemCommand) (Cart, error)
	Estimate(ctx context.Context, userID string) (CartEstimate, error)
	ApplyPromotion(ctx context.Context, cmd CartPromotionCommand) (Cart, error)
	RemovePromotion(ctx context.Context, userID string) (Cart, error)
	ClearCart(ctx context.Context, userID string) error
}

// CheckoutService coordinates PSP session creation and client confirmations.
type CheckoutService interface {
	CreateCheckoutSession(ctx context.Context, cmd CreateCheckoutSessionCommand) (CheckoutSession, error)
	ConfirmClientCompletion(ctx context.Context, cmd ConfirmCheckoutCommand) error
}

// OrderService encapsulates order read/write flows including cancellation and reorders.
type OrderService interface {
	CreateFromCart(ctx context.Context, cmd CreateOrderFromCartCommand) (Order, error)
	ListOrders(ctx context.Context, filter OrderListFilter) (domain.CursorPage[Order], error)
	GetOrder(ctx context.Context, orderID string, opts OrderReadOptions) (Order, error)
	TransitionStatus(ctx context.Context, cmd OrderStatusTransitionCommand) (Order, error)
	Cancel(ctx context.Context, cmd CancelOrderCommand) (Order, error)
	AppendProductionEvent(ctx context.Context, cmd AppendProductionEventCommand) (OrderProductionEvent, error)
	RequestInvoice(ctx context.Context, cmd RequestInvoiceCommand) (Order, error)
	CloneForReorder(ctx context.Context, cmd CloneForReorderCommand) (Order, error)
}

// PaymentService handles idempotent PSP webhook processing and admin adjustments.
type PaymentService interface {
	RecordWebhookEvent(ctx context.Context, cmd PaymentWebhookCommand) error
	ManualCapture(ctx context.Context, cmd PaymentManualCaptureCommand) (Payment, error)
	ManualRefund(ctx context.Context, cmd PaymentManualRefundCommand) (Payment, error)
	ListPayments(ctx context.Context, orderID string) ([]Payment, error)
}

// ShipmentService orchestrates shipment creation, updates, and webhook ingestion.
type ShipmentService interface {
	CreateShipment(ctx context.Context, cmd CreateShipmentCommand) (Shipment, error)
	UpdateShipmentStatus(ctx context.Context, cmd UpdateShipmentCommand) (Shipment, error)
	ListShipments(ctx context.Context, orderID string) ([]Shipment, error)
	RecordCarrierEvent(ctx context.Context, cmd ShipmentEventCommand) error
}

// ReviewService coordinates review lifecycle and moderation workflows.
type ReviewService interface {
	Create(ctx context.Context, cmd CreateReviewCommand) (Review, error)
	GetByOrder(ctx context.Context, orderID string) (Review, error)
	ListByUser(ctx context.Context, cmd ListUserReviewsCommand) (domain.CursorPage[Review], error)
	Moderate(ctx context.Context, cmd ModerateReviewCommand) (Review, error)
	StoreReply(ctx context.Context, cmd StoreReviewReplyCommand) (Review, error)
}

// PromotionService exposes promotion lifecycle and validation operations.
type PromotionService interface {
	GetPublicPromotion(ctx context.Context, code string) (PromotionValidationResult, error)
	ValidatePromotion(ctx context.Context, cmd ValidatePromotionCommand) (PromotionValidationResult, error)
	ListPromotions(ctx context.Context, filter PromotionListFilter) (domain.CursorPage[Promotion], error)
	CreatePromotion(ctx context.Context, cmd UpsertPromotionCommand) (Promotion, error)
	UpdatePromotion(ctx context.Context, cmd UpsertPromotionCommand) (Promotion, error)
	DeletePromotion(ctx context.Context, promoID string) error
	ListPromotionUsage(ctx context.Context, filter PromotionUsageFilter) (domain.CursorPage[PromotionUsage], error)
}

// UserService manages profile, address, payment method, and favorite surfaces.
type UserService interface {
	GetProfile(ctx context.Context, userID string) (UserProfile, error)
	GetByUID(ctx context.Context, userID string) (UserProfile, error)
	UpdateProfile(ctx context.Context, cmd UpdateProfileCommand) (UserProfile, error)
	MaskProfile(ctx context.Context, cmd MaskProfileCommand) (UserProfile, error)
	SetUserActive(ctx context.Context, cmd SetUserActiveCommand) (UserProfile, error)
	ListAddresses(ctx context.Context, userID string) ([]Address, error)
	UpsertAddress(ctx context.Context, cmd UpsertAddressCommand) (Address, error)
	DeleteAddress(ctx context.Context, cmd DeleteAddressCommand) error
	ListPaymentMethods(ctx context.Context, userID string) ([]PaymentMethod, error)
	AddPaymentMethod(ctx context.Context, cmd AddPaymentMethodCommand) (PaymentMethod, error)
	RemovePaymentMethod(ctx context.Context, cmd RemovePaymentMethodCommand) error
	ListFavorites(ctx context.Context, userID string, pager Pagination) (domain.CursorPage[FavoriteDesign], error)
	ToggleFavorite(ctx context.Context, cmd ToggleFavoriteCommand) error
}

// InventoryEventPublisher accepts inventory stock change notifications for downstream processing.
type InventoryEventPublisher interface {
	PublishInventoryEvent(ctx context.Context, event InventoryStockEvent) error
}

// InventoryService centralizes stock reservation, commit, and release workflows.
type InventoryService interface {
	ReserveStocks(ctx context.Context, cmd InventoryReserveCommand) (InventoryReservation, error)
	CommitReservation(ctx context.Context, cmd InventoryCommitCommand) (InventoryReservation, error)
	ReleaseReservation(ctx context.Context, cmd InventoryReleaseCommand) (InventoryReservation, error)
	ListLowStock(ctx context.Context, filter InventoryLowStockFilter) (domain.CursorPage[InventorySnapshot], error)
}

// ContentService provides read/write access to CMS content for public and admin usage.
type ContentService interface {
	ListGuides(ctx context.Context, filter ContentGuideFilter) (domain.CursorPage[ContentGuide], error)
	GetGuide(ctx context.Context, guideID string) (ContentGuide, error)
	UpsertGuide(ctx context.Context, cmd UpsertContentGuideCommand) (ContentGuide, error)
	DeleteGuide(ctx context.Context, guideID string) error
	GetPage(ctx context.Context, slug string, locale string) (ContentPage, error)
	UpsertPage(ctx context.Context, cmd UpsertContentPageCommand) (ContentPage, error)
}

// CatalogService manages templates, fonts, materials, and products for admin-facing operations.
type CatalogService interface {
	ListTemplates(ctx context.Context, filter TemplateFilter) (domain.CursorPage[TemplateSummary], error)
	UpsertTemplate(ctx context.Context, cmd UpsertTemplateCommand) (TemplateSummary, error)
	DeleteTemplate(ctx context.Context, templateID string) error
	ListFonts(ctx context.Context, filter FontFilter) (domain.CursorPage[FontSummary], error)
	UpsertFont(ctx context.Context, cmd UpsertFontCommand) (FontSummary, error)
	DeleteFont(ctx context.Context, fontID string) error
	ListMaterials(ctx context.Context, filter MaterialFilter) (domain.CursorPage[MaterialSummary], error)
	UpsertMaterial(ctx context.Context, cmd UpsertMaterialCommand) (MaterialSummary, error)
	DeleteMaterial(ctx context.Context, materialID string) error
	ListProducts(ctx context.Context, filter ProductFilter) (domain.CursorPage[ProductSummary], error)
	UpsertProduct(ctx context.Context, cmd UpsertProductCommand) (ProductSummary, error)
	DeleteProduct(ctx context.Context, productID string) error
}

// AssetService issues signed URLs and coordinates storage metadata syncing.
type AssetService interface {
	IssueSignedUpload(ctx context.Context, cmd SignedUploadCommand) (SignedAssetResponse, error)
	IssueSignedDownload(ctx context.Context, cmd SignedDownloadCommand) (SignedAssetResponse, error)
}

// SystemService aggregates utility endpoints (health checks, audit logs, counters).
type SystemService interface {
	HealthReport(ctx context.Context) (SystemHealthReport, error)
	ListAuditLogs(ctx context.Context, filter AuditLogFilter) (domain.CursorPage[AuditLogEntry], error)
	NextCounterValue(ctx context.Context, cmd CounterCommand) (int64, error)
}

// AuditLogService centralizes immutable audit log persistence and retrieval.
type AuditLogService interface {
	Record(ctx context.Context, record AuditLogRecord)
	List(ctx context.Context, filter AuditLogFilter) (domain.CursorPage[AuditLogEntry], error)
}

// BackgroundJobDispatcher schedules asynchronous processing such as AI jobs, cleanup tasks, and notifications.
type BackgroundJobDispatcher interface {
	QueueAISuggestion(ctx context.Context, cmd QueueAISuggestionCommand) (QueueAISuggestionResult, error)
	GetAIJob(ctx context.Context, jobID string) (domain.AIJob, error)
	CompleteAISuggestion(ctx context.Context, cmd CompleteAISuggestionCommand) (CompleteAISuggestionResult, error)
	GetSuggestion(ctx context.Context, designID string, suggestionID string) (AISuggestion, error)
	EnqueueRegistrabilityCheck(ctx context.Context, payload RegistrabilityJobPayload) (string, error)
	EnqueueStockCleanup(ctx context.Context, payload StockCleanupPayload) error
}

// ErrorTranslator converts repository or platform errors into domain-aware sentinel errors.
type ErrorTranslator interface {
	Translate(err error) error
}

// DomainError represents a structured error with stable codes for transport across layers.
type DomainError interface {
	error
	Code() string
	SafeMessage() string
}

// Command and DTO definitions ------------------------------------------------

type CreateDesignCommand struct {
	OwnerID  string
	Template string
	Locale   string
	Snapshot map[string]any
}

type DesignReadOptions struct {
	IncludeVersions    bool
	IncludeSuggestions bool
}

type DesignListFilter struct {
	OwnerID string
	Status  []string
	Pagination
}

type UpdateDesignCommand struct {
	DesignID  string
	Snapshot  map[string]any
	Status    string
	UpdatedBy string
}

type DeleteDesignCommand struct {
	DesignID    string
	RequestedBy string
	SoftDelete  bool
}

type DuplicateDesignCommand struct {
	SourceDesignID string
	RequestedBy    string
	OverrideName   *string
}

type AISuggestionRequest struct {
	DesignID string
	Method   string
	Model    string
	Metadata map[string]any
}

type AISuggestionFilter struct {
	Status     []string
	Pagination Pagination
}

type AISuggestionStatusCommand struct {
	DesignID     string
	SuggestionID string
	Action       string
	ActorID      string
}

type RegistrabilityCheckCommand struct {
	DesignID string
	UserID   string
	Locale   string
}

type UpsertCartItemCommand struct {
	UserID        string
	ItemID        *string
	ProductID     string
	SKU           string
	Quantity      int
	Customization map[string]any
	DesignID      *string
}

type RemoveCartItemCommand struct {
	UserID string
	ItemID string
}

type CartPromotionCommand struct {
	UserID string
	Code   string
	Source string
}

type CreateCheckoutSessionCommand struct {
	UserID     string
	CartID     string
	SuccessURL string
	CancelURL  string
	PSP        string
	Metadata   map[string]string
}

type ConfirmCheckoutCommand struct {
	UserID    string
	OrderID   string
	SessionID string
}

type OrderListFilter = repositories.OrderListFilter

type OrderReadOptions struct {
	IncludePayments         bool
	IncludeShipments        bool
	IncludeProductionEvents bool
}

type CreateOrderFromCartCommand struct {
	Cart           Cart
	ActorID        string
	ReservationID  string
	OrderNumber    *string
	Metadata       map[string]any
	ExpectedStatus *OrderStatus
}

type OrderStatusTransitionCommand struct {
	OrderID        string
	TargetStatus   OrderStatus
	ActorID        string
	Reason         string
	ExpectedStatus *OrderStatus
	Metadata       map[string]any
}

type CancelOrderCommand struct {
	OrderID        string
	ActorID        string
	Reason         string
	ReservationID  string
	ExpectedStatus *OrderStatus
	Metadata       map[string]any
}

type RequestInvoiceCommand struct {
	OrderID        string
	ActorID        string
	Notes          string
	ExpectedStatus *OrderStatus
}

type AppendProductionEventCommand struct {
	OrderID string
	Event   OrderProductionEvent
	ActorID string
}

type CloneForReorderCommand struct {
	OrderID  string
	ActorID  string
	Metadata map[string]any
}

type PaymentWebhookCommand struct {
	Provider string
	Payload  []byte
	Headers  map[string]string
}

type PaymentManualCaptureCommand struct {
	OrderID   string
	PaymentID string
	ActorID   string
}

type PaymentManualRefundCommand struct {
	OrderID   string
	PaymentID string
	ActorID   string
	Amount    *int64
	Reason    string
}

type CreateShipmentCommand struct {
	OrderID   string
	Carrier   string
	Items     []ShipmentItem
	CreatedBy string
}

type ShipmentItem struct {
	LineItemSKU string
	Quantity    int
}

type UpdateShipmentCommand struct {
	OrderID      string
	ShipmentID   string
	Status       string
	TrackingCode *string
	ActorID      string
}

type ShipmentEventCommand struct {
	OrderID    string
	ShipmentID string
	Carrier    string
	Event      ShipmentEvent
}

type CreateReviewCommand struct {
	OrderID string
	UserID  string
	Rating  int
	Comment string
	ActorID string
}

type ListUserReviewsCommand struct {
	UserID     string
	Pagination Pagination
}

type ModerateReviewCommand struct {
	ReviewID string
	ActorID  string
	Status   ReviewStatus
}

type StoreReviewReplyCommand struct {
	ReviewID string
	ActorID  string
	Message  string
	Visible  bool
}

type ValidatePromotionCommand struct {
	Code    string
	UserID  *string
	CartID  *string
	OrderID *string
}

type PromotionListFilter struct {
	Status     []string
	Pagination Pagination
}

type UpsertPromotionCommand struct {
	Promotion Promotion
	ActorID   string
}

type PromotionUsageFilter struct {
	PromotionID string
	Pagination  Pagination
}

type UpdateProfileCommand struct {
	UserID            string
	ActorID           string
	DisplayName       *string
	PreferredLanguage *string
	Locale            *string
	NotificationPrefs map[string]bool
	AvatarAssetID     *string
	ExpectedSyncTime  *time.Time
}

type MaskProfileCommand struct {
	UserID   string
	ActorID  string
	Reason   string
	Occurred time.Time
}

type SetUserActiveCommand struct {
	UserID           string
	ActorID          string
	IsActive         bool
	Reason           string
	ExpectedSyncTime *time.Time
}

type UpsertAddressCommand struct {
	UserID    string
	AddressID *string
	Address   Address
	IsDefault bool
}

type DeleteAddressCommand struct {
	UserID    string
	AddressID string
}

type AddPaymentMethodCommand struct {
	UserID    string
	Provider  string
	Reference string
	Token     string
}

type RemovePaymentMethodCommand struct {
	UserID          string
	PaymentMethodID string
}

type ToggleFavoriteCommand struct {
	UserID   string
	DesignID string
	Mark     bool
}

type InventoryReserveCommand struct {
	OrderID        string
	UserID         string
	Lines          []InventoryLine
	TTL            time.Duration
	Reason         string
	IdempotencyKey string
}

type InventoryCommitCommand struct {
	ReservationID string
	OrderID       string
	ActorID       string
}

type InventoryReleaseCommand struct {
	ReservationID string
	Reason        string
	ActorID       string
}

type InventoryLine struct {
	ProductID string
	SKU       string
	Quantity  int
}

type InventoryLowStockFilter struct {
	Threshold  int
	Pagination Pagination
}

type ContentGuideFilter struct {
	Category   *string
	Locale     *string
	Status     []string
	Pagination Pagination
}

type UpsertContentGuideCommand struct {
	Guide   ContentGuide
	ActorID string
}

type UpsertContentPageCommand struct {
	Page    ContentPage
	ActorID string
}

type TemplateFilter struct {
	Shape      *string
	Writing    *string
	IsPublic   *bool
	Pagination Pagination
}

type UpsertTemplateCommand struct {
	Template TemplateSummary
	ActorID  string
}

type FontFilter struct {
	Writing    *string
	Pagination Pagination
}

type UpsertFontCommand struct {
	Font    FontSummary
	ActorID string
}

type MaterialFilter struct {
	Texture    *string
	IsPublic   *bool
	Pagination Pagination
}

type UpsertMaterialCommand struct {
	Material MaterialSummary
	ActorID  string
}

type ProductFilter struct {
	Shape      *string
	SizeMm     *int
	MaterialID *string
	Pagination Pagination
}

type UpsertProductCommand struct {
	Product ProductSummary
	ActorID string
}

type SignedUploadCommand struct {
	ActorID     string
	DesignID    *string
	Kind        string
	Purpose     string
	FileName    string
	ContentType string
	SizeBytes   int64
}

type SignedDownloadCommand struct {
	ActorID string
	AssetID string
}

// AuditLogRecord defines the payload accepted by the audit writer service.
type AuditLogRecord struct {
	Actor                 string
	ActorType             string
	Action                string
	TargetRef             string
	Severity              string
	RequestID             string
	OccurredAt            time.Time
	Metadata              map[string]any
	Diff                  map[string]AuditLogDiff
	SensitiveMetadataKeys []string
	SensitiveDiffKeys     []string
	IPAddress             string
	UserAgent             string
}

// AuditLogDiff captures before/after values for tracked fields.
type AuditLogDiff struct {
	Before any
	After  any
}

type AuditLogFilter struct {
	TargetRef  string
	Actor      string
	ActorType  string
	Action     string
	DateRange  domain.RangeQuery[time.Time]
	Pagination Pagination
}

type CounterCommand struct {
	CounterID string
	Step      int64
}

// QueueAISuggestionCommand packages inputs for an AI suggestion job request.
type QueueAISuggestionCommand struct {
	DesignID       string
	Method         string
	Model          string
	Prompt         string
	Snapshot       map[string]any
	Parameters     map[string]any
	Metadata       map[string]any
	IdempotencyKey string
	Priority       int
	RequestedBy    string
}

// QueueAISuggestionResult reports identifiers generated for a queued AI job.
type QueueAISuggestionResult struct {
	JobID        string
	SuggestionID string
	Status       domain.AIJobStatus
	QueuedAt     time.Time
}

// CompleteAISuggestionCommand encapsulates AI worker outputs for persisting suggestion results.
type CompleteAISuggestionCommand struct {
	JobID      string
	Suggestion AISuggestion
	Error      *domain.AIJobError
	Outputs    map[string]any
	Metadata   map[string]any
}

// CompleteAISuggestionResult returns the persisted job and optional suggestion outputs.
type CompleteAISuggestionResult struct {
	Job        domain.AIJob
	Suggestion *AISuggestion
}

type RegistrabilityJobPayload struct {
	RequestID string
	DesignID  string
	Locale    string
}

type StockCleanupPayload struct {
	ReservationIDs []string
}
