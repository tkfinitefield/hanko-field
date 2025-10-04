package domain

import (
	"time"
)

// Pagination defines standard cursor-based paging inputs for list operations.
type Pagination struct {
	PageSize  int
	PageToken string
}

// SortOrder indicates ascending or descending ordering for list queries.
type SortOrder string

const (
	// SortAsc sorts results in ascending order.
	SortAsc SortOrder = "asc"
	// SortDesc sorts results in descending order.
	SortDesc SortOrder = "desc"
)

// RangeQuery represents inclusive range filters for numeric or timestamp fields.
type RangeQuery[T comparable] struct {
	From *T
	To   *T
}

// Design encapsulates user-created seal design metadata shared across layers.
type Design struct {
	ID        string
	OwnerID   string
	Status    string
	Template  string
	Locale    string
	Snapshot  map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DesignVersion stores historical snapshots for audits and reverts.
type DesignVersion struct {
	ID        string
	DesignID  string
	Snapshot  map[string]any
	CreatedAt time.Time
	CreatedBy string
}

// AISuggestion represents AI generated variants for a design.
type AISuggestion struct {
	ID        string
	DesignID  string
	Method    string
	Status    string
	Payload   map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt *time.Time
}

// Cart aggregates the mutable shopping cart state for a user.
type Cart struct {
	ID        string
	UserID    string
	Currency  string
	Promotion *CartPromotion
	Items     []CartItem
	Estimate  *CartEstimate
	UpdatedAt time.Time
}

// CartPromotion captures the applied promotion snapshot.
type CartPromotion struct {
	Code           string
	DiscountAmount int64
	Applied        bool
}

// CartItem stores a single SKU entry within a cart.
type CartItem struct {
	ID            string
	ProductID     string
	SKU           string
	Quantity      int
	UnitPrice     int64
	Currency      string
	Customization map[string]any
	DesignRef     *string
	Estimates     map[string]int64
	AddedAt       time.Time
	UpdatedAt     *time.Time
}

// CartEstimate summarizes totals calculated for the cart.
type CartEstimate struct {
	Subtotal int64
	Discount int64
	Tax      int64
	Shipping int64
	Total    int64
}

// CheckoutSession represents PSP checkout session metadata stored by services.
type CheckoutSession struct {
	SessionID    string
	PSP          string
	ClientSecret string
	RedirectURL  string
	ExpiresAt    time.Time
}

// Order captures order headers returned to handlers/services.
type Order struct {
	ID          string
	OrderNumber string
	UserID      string
	Status      string
	Currency    string
	Totals      OrderTotals
	Promotion   *CartPromotion
	Items       []OrderLineItem
	CreatedAt   time.Time
	UpdatedAt   time.Time
	PaidAt      *time.Time
	ShippedAt   *time.Time
}

// OrderTotals holds rolled-up monetary fields in the smallest currency unit.
type OrderTotals struct {
	Subtotal int64
	Discount int64
	Shipping int64
	Tax      int64
	Fees     int64
	Total    int64
}

// OrderLineItem mirrors cart items at the time of checkout.
type OrderLineItem struct {
	ProductID string
	SKU       string
	Name      string
	Quantity  int
	UnitPrice int64
	Total     int64
}

// Payment encapsulates payment status and PSP references for an order.
type Payment struct {
	ID         string
	OrderID    string
	Provider   string
	Status     string
	Amount     int64
	Currency   string
	Captured   bool
	CapturedAt *time.Time
	RefundedAt *time.Time
	Raw        map[string]any
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Shipment represents fulfilment records for an order.
type Shipment struct {
	ID           string
	OrderID      string
	Carrier      string
	TrackingCode string
	Status       string
	Events       []ShipmentEvent
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ShipmentEvent stores timestamped updates from carriers or operations.
type ShipmentEvent struct {
	Status     string
	OccurredAt time.Time
	Details    map[string]any
}

// Promotion describes promotional rules persisted by admin services.
type Promotion struct {
	ID          string
	Code        string
	Name        string
	Description string
	Status      string
	StartsAt    time.Time
	EndsAt      time.Time
	UsageLimit  *int
	Metadata    map[string]any
}

// PromotionValidationResult is returned when a promotion is evaluated for a cart or order.
type PromotionValidationResult struct {
	Code           string
	Eligible       bool
	Reason         string
	DiscountAmount int64
}

// RegistrabilityCheckResult stores outcomes from external name seal registrability checks.
type RegistrabilityCheckResult struct {
	DesignID    string
	Passed      bool
	Reasons     []string
	RequestedAt time.Time
}

// Address represents postal address structures shared by user and order layers.
type Address struct {
	Recipient  string
	Line1      string
	Line2      *string
	City       string
	State      *string
	PostalCode string
	Country    string
	Phone      *string
}

// UserProfile captures editable user profile fields.
type UserProfile struct {
	ID             string
	DisplayName    string
	Persona        string
	PreferredLang  string
	Country        string
	MarketingOptIn bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// FavoriteDesign ties a user to a design ID for fast lookups.
type FavoriteDesign struct {
	DesignID string
	AddedAt  time.Time
}

// PromotionUsage aggregates per-user promotion usage metrics.
type PromotionUsage struct {
	UserID   string
	Times    int
	LastUsed time.Time
}

// PaymentMethod stores PSP-backed payment references without sensitive card data.
type PaymentMethod struct {
	ID        string
	Provider  string
	Reference string
	Brand     string
	Last4     string
	ExpMonth  int
	ExpYear   int
	CreatedAt time.Time
}

// InventoryReservation holds temporary or committed stock reservations.
type InventoryReservation struct {
	ID        string
	OrderID   string
	ProductID string
	Quantity  int
	Status    string
	ExpiresAt *time.Time
	CreatedAt time.Time
}

// InventorySnapshot exposes aggregated stock levels for admin surfaces.
type InventorySnapshot struct {
	ProductID string
	Available int
	Reserved  int
	Threshold int
}

// ContentPage describes CMS-managed content accessible via public endpoints.
type ContentPage struct {
	ID        string
	Slug      string
	Locale    string
	Title     string
	BodyRef   string
	Status    string
	UpdatedAt time.Time
}

// ContentGuide captures localized guide metadata for CMS flows.
type ContentGuide struct {
	ID        string
	Slug      string
	Locale    string
	Category  string
	Title     string
	Summary   string
	BodyRef   string
	Status    string
	UpdatedAt time.Time
}

// TemplateSummary describes catalog templates for listing endpoints.
type TemplateSummary struct {
	ID       string
	Name     string
	Shape    string
	Writing  string
	IsPublic bool
	Sort     int
}

// FontSummary captures metadata required by rendering services.
type FontSummary struct {
	ID         string
	Family     string
	Subfamily  string
	Writing    string
	PreviewURL string
}

// MaterialSummary stores material metadata for product configuration.
type MaterialSummary struct {
	ID       string
	Name     string
	Texture  string
	IsPublic bool
}

// ProductSummary represents public-facing product information.
type ProductSummary struct {
	ID         string
	SKU        string
	TemplateID string
	MaterialID string
	FontID     string
	Shape      string
	SizeMm     int
	IsPublic   bool
	Price      int64
}

// SystemHealthReport aggregates dependency status for health endpoints.
type SystemHealthReport struct {
	Status  string
	Checks  map[string]string
	Version string
	Uptime  time.Duration
}

// AuditLogEntry stores normalized audit information for admin use.
type AuditLogEntry struct {
	ID        string
	ActorRef  string
	TargetRef string
	Action    string
	Diff      map[string]any
	CreatedAt time.Time
}

// SignedAssetResponse returns signed URL payloads for upload/download flows.
type SignedAssetResponse struct {
	AssetID   string
	URL       string
	ExpiresAt time.Time
	Method    string
	Headers   map[string]string
}

// CursorPage packages list results with an encoded next token.
type CursorPage[T any] struct {
	Items         []T
	NextPageToken string
}
