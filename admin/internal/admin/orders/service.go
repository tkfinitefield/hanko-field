package orders

import (
	"context"
	"time"
)

// Service exposes order listing capabilities for the admin UI.
type Service interface {
	// List returns a paginated set of orders that match the provided query.
	List(ctx context.Context, token string, query Query) (ListResult, error)
}

// Status represents the canonical lifecycle state of an order.
type Status string

const (
	// StatusPendingPayment indicates the order is awaiting payment confirmation.
	StatusPendingPayment Status = "pending_payment"
	// StatusPaymentReview indicates the order is under payment review.
	StatusPaymentReview Status = "payment_review"
	// StatusInProduction indicates the order is being manufactured.
	StatusInProduction Status = "in_production"
	// StatusReadyToShip indicates the order is ready for fulfillment.
	StatusReadyToShip Status = "ready_to_ship"
	// StatusShipped indicates the order has left the warehouse.
	StatusShipped Status = "shipped"
	// StatusDelivered indicates the order was delivered to the customer.
	StatusDelivered Status = "delivered"
	// StatusCancelled indicates the order was cancelled.
	StatusCancelled Status = "cancelled"
	// StatusRefunded indicates the order was refunded.
	StatusRefunded Status = "refunded"
)

// SortDirection describes the requested sort ordering.
type SortDirection string

const (
	// SortDirectionAsc sorts ascending.
	SortDirectionAsc SortDirection = "asc"
	// SortDirectionDesc sorts descending.
	SortDirectionDesc SortDirection = "desc"
)

// Query captures filters and pagination arguments for listing orders.
type Query struct {
	Statuses      []Status
	Since         *time.Time
	Currency      string
	AmountMin     *int64
	AmountMax     *int64
	HasRefundOnly *bool
	Page          int
	PageSize      int
	SortKey       string
	SortDirection SortDirection
}

// ListResult represents a paginated orders response.
type ListResult struct {
	Orders     []Order
	Pagination Pagination
	Summary    Summary
	Filters    FilterSummary
}

// Pagination captures pagination metadata.
type Pagination struct {
	Page       int
	PageSize   int
	TotalItems int
	NextPage   *int
	PrevPage   *int
}

// Summary aggregates quick metrics for the current result set.
type Summary struct {
	TotalOrders        int
	TotalRevenueMinor  int64
	AverageLeadHours   float64
	DelayedCount       int
	RefundRequested    int
	InProductionCount  int
	FulfilledLast24h   int
	LastRefreshedAt    time.Time
	PrimaryCurrency    string
	StatusDistribution []StatusCount
}

// StatusCount captures counts per status for the filtered result set.
type StatusCount struct {
	Status Status
	Count  int
}

// FilterSummary exposes supporting data used to render filter controls.
type FilterSummary struct {
	StatusOptions   []StatusOption
	CurrencyOptions []CurrencyOption
	RefundOptions   []RefundOption
	AmountRanges    []AmountRange
}

// StatusOption represents a selectable status filter.
type StatusOption struct {
	Value       Status
	Label       string
	Count       int
	Description string
}

// CurrencyOption represents a selectable currency filter.
type CurrencyOption struct {
	Code  string
	Label string
	Count int
}

// RefundOption represents the has-refund filter choices.
type RefundOption struct {
	Value string
	Label string
}

// AmountRange suggests useful amount range shortcuts.
type AmountRange struct {
	Label string
	Min   *int64
	Max   *int64
}

// Order represents a single order row in the index table.
type Order struct {
	ID               string
	Number           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Customer         Customer
	TotalMinor       int64
	Currency         string
	Status           Status
	StatusLabel      string
	StatusTone       string
	Fulfillment      Fulfillment
	Payment          Payment
	Tags             []string
	Badges           []Badge
	ItemsSummary     string
	Notes            []string
	SalesChannel     string
	Integration      string
	HasRefundRequest bool
}

// Customer contains primary customer display information.
type Customer struct {
	ID    string
	Name  string
	Email string
	Phone string
}

// Fulfillment captures shipping and SLA metadata.
type Fulfillment struct {
	Method        string
	Carrier       string
	TrackingID    string
	PromisedDate  *time.Time
	DispatchedAt  *time.Time
	DeliveredAt   *time.Time
	SLAStatus     string
	SLAStatusTone string
}

// Payment summarises payment state for the order.
type Payment struct {
	Status        string
	StatusTone    string
	CapturedAt    *time.Time
	DueAt         *time.Time
	PastDue       bool
	PastDueReason string
}

// Badge renders a small inline badge.
type Badge struct {
	Label string
	Tone  string
	Icon  string
	Title string
}
