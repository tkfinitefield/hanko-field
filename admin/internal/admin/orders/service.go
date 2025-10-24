package orders

import (
	"context"
	"errors"
	"strings"
	"time"
)

// Service exposes order listing capabilities for the admin UI.
type Service interface {
	// List returns a paginated set of orders that match the provided query.
	List(ctx context.Context, token string, query Query) (ListResult, error)

	// StatusModal loads metadata required to render the status update modal for an order.
	StatusModal(ctx context.Context, token, orderID string) (StatusModal, error)

	// UpdateStatus attempts to transition an order to the provided status and returns the updated order state.
	UpdateStatus(ctx context.Context, token, orderID string, req StatusUpdateRequest) (StatusUpdateResult, error)

	// RefundModal loads metadata required to render the refund modal for an order.
	RefundModal(ctx context.Context, token, orderID string) (RefundModal, error)

	// SubmitRefund attempts to create a refund for the specified order payment.
	SubmitRefund(ctx context.Context, token, orderID string, req RefundRequest) (RefundResult, error)

	// InvoiceModal loads metadata required to render the invoice issuance modal for an order.
	InvoiceModal(ctx context.Context, token, orderID string) (InvoiceModal, error)

	// IssueInvoice attempts to issue an invoice for the specified order using the provided parameters.
	IssueInvoice(ctx context.Context, token string, req InvoiceIssueRequest) (InvoiceIssueResult, error)

	// InvoiceJobStatus returns the progress of an asynchronous invoice issuance job.
	InvoiceJobStatus(ctx context.Context, token, jobID string) (InvoiceJobStatus, error)

	// StartBulkExport enqueues a background export or print job.
	StartBulkExport(ctx context.Context, token string, req BulkExportRequest) (ExportJob, error)

	// ListExportJobs returns the known export jobs scoped to the current user.
	ListExportJobs(ctx context.Context, token string) ([]ExportJob, error)

	// ExportJobStatus returns the latest status of a bulk export job.
	ExportJobStatus(ctx context.Context, token, jobID string) (ExportJobStatus, error)
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

var (
	// ErrOrderNotFound is returned when an order does not exist.
	ErrOrderNotFound = errors.New("order not found")
	// ErrInvalidTransition is returned when a requested status change is not permitted.
	ErrInvalidTransition = errors.New("invalid status transition")
	// ErrPaymentNotFound indicates the provided payment reference does not exist for the order.
	ErrPaymentNotFound = errors.New("payment not found")
	// ErrRefundFailed indicates the PSP refund attempt failed for reasons other than validation.
	ErrRefundFailed = errors.New("refund failed")
	// ErrInvoiceTemplateNotFound indicates the requested invoice template does not exist.
	ErrInvoiceTemplateNotFound = errors.New("invoice template not found")
	// ErrInvoiceJobNotFound indicates the requested invoice issuance job does not exist.
	ErrInvoiceJobNotFound = errors.New("invoice job not found")
	// ErrExportJobNotFound indicates the requested export job does not exist.
	ErrExportJobNotFound = errors.New("export job not found")
	// ErrExportFormatNotAllowed indicates the requested export format is not supported.
	ErrExportFormatNotAllowed = errors.New("export format not allowed")
	// ErrExportNoOrders indicates there were no orders available for export.
	ErrExportNoOrders = errors.New("no orders to export")
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
	Payments         []PaymentDetail
	Refunds          []RefundRecord
	Invoices         []InvoiceRecord
}

// StatusModal represents data necessary to render the status update modal.
type StatusModal struct {
	Order          Order
	Choices        []StatusTransitionOption
	LatestTimeline []TimelineEvent
}

// StatusTransitionOption describes an available status transition choice.
type StatusTransitionOption struct {
	Value          Status
	Label          string
	Description    string
	Disabled       bool
	DisabledReason string
	Selected       bool
}

// StatusUpdateRequest encapsulates desired transition parameters.
type StatusUpdateRequest struct {
	Status         Status
	Note           string
	NotifyCustomer bool
	ActorID        string
	ActorEmail     string
}

// StatusUpdateResult returns updated order state with enriched fragments.
type StatusUpdateResult struct {
	Order    Order
	Timeline []TimelineEvent
}

// TimelineEvent captures a single history entry for an order.
type TimelineEvent struct {
	ID          string
	Status      Status
	Title       string
	Description string
	Actor       string
	OccurredAt  time.Time
}

// AuditLogger records audit trail entries for order operations.
type AuditLogger interface {
	Record(ctx context.Context, entry AuditLogEntry) error
}

// AuditLogEntry describes a structured audit record for an order status change.
type AuditLogEntry struct {
	OrderID     string
	OrderNumber string
	Action      string
	ActorID     string
	ActorEmail  string
	FromStatus  Status
	ToStatus    Status
	Note        string
	OccurredAt  time.Time
}

// StatusTransitionError represents a validation failure for a requested status change.
type StatusTransitionError struct {
	From   Status
	To     Status
	Reason string
}

// Error implements the error interface.
func (e *StatusTransitionError) Error() string {
	if e == nil {
		return ErrInvalidTransition.Error()
	}
	reason := e.Reason
	if strings.TrimSpace(reason) == "" {
		reason = "transition not permitted"
	}
	return "order status transition from " + string(e.From) + " to " + string(e.To) + ": " + reason
}

// StatusDescription returns a human friendly description for a status value.
func StatusDescription(status Status) string {
	switch status {
	case StatusPendingPayment:
		return "支払い確認を待っています"
	case StatusPaymentReview:
		return "決済チームが支払いを確認中です"
	case StatusInProduction:
		return "制作工程で作業が進行中です"
	case StatusReadyToShip:
		return "出荷準備が完了し、集荷待ちです"
	case StatusShipped:
		return "配送業者に引き渡され輸送中です"
	case StatusDelivered:
		return "お客様への納品が完了しました"
	case StatusRefunded:
		return "返金処理が完了しました"
	case StatusCancelled:
		return "注文はキャンセルされました"
	default:
		return ""
	}
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

// PaymentDetail represents a single payment attempt that can be refunded.
type PaymentDetail struct {
	ID               string
	Provider         string
	Method           string
	Last4            string
	Reference        string
	Status           string
	StatusTone       string
	Currency         string
	AmountAuthorized int64
	AmountCaptured   int64
	AmountRefunded   int64
	AmountAvailable  int64
	CapturedAt       *time.Time
	ExpiresAt        *time.Time
}

// Badge renders a small inline badge.
type Badge struct {
	Label string
	Tone  string
	Icon  string
	Title string
}

// RefundModal provides information required to render the refund UI for an order.
type RefundModal struct {
	Order           RefundOrderSummary
	Payments        []RefundPaymentOption
	ExistingRefunds []RefundRecord
	SupportsPartial bool
	Currency        string
}

// RefundOrderSummary gives contextual order details for the refund modal.
type RefundOrderSummary struct {
	ID             string
	Number         string
	CustomerName   string
	TotalMinor     int64
	Currency       string
	PaymentStatus  string
	PaymentTone    string
	OutstandingDue string
}

// RefundPaymentOption represents a selectable payment source to refund against.
type RefundPaymentOption struct {
	ID              string
	Label           string
	Method          string
	Reference       string
	Status          string
	StatusTone      string
	Currency        string
	CapturedMinor   int64
	RefundedMinor   int64
	AvailableMinor  int64
	CapturedAt      *time.Time
	SupportsRefunds bool
}

// RefundRecord describes an existing refund associated with an order.
type RefundRecord struct {
	ID          string
	PaymentID   string
	AmountMinor int64
	Currency    string
	Reason      string
	Status      string
	ProcessedAt time.Time
	Actor       string
	Reference   string
}

// RefundRequest contains parameters for creating a refund.
type RefundRequest struct {
	PaymentID      string
	AmountMinor    int64
	Currency       string
	Reason         string
	NotifyCustomer bool
	ActorID        string
	ActorEmail     string
}

// RefundResult returns information about the processed refund and updated payment state.
type RefundResult struct {
	Refund   RefundRecord
	Payment  RefundPaymentOption
	Payments []RefundPaymentOption
}

// RefundValidationError indicates validation issues with the refund request.
type RefundValidationError struct {
	Message     string
	FieldErrors map[string]string
}

// Error implements the error interface.
func (e *RefundValidationError) Error() string {
	if e == nil {
		return "invalid refund request"
	}
	msg := strings.TrimSpace(e.Message)
	if msg == "" {
		msg = "invalid refund request"
	}
	return msg
}

// InvoiceModal provides information required to render the invoice issuance modal.
type InvoiceModal struct {
	Order           InvoiceOrderSummary
	Templates       []InvoiceTemplate
	Languages       []InvoiceLanguage
	SuggestedEmail  string
	RecentInvoices  []InvoiceRecord
	DefaultTemplate string
	DefaultLanguage string
}

// InvoiceOrderSummary captures primary order details needed when issuing invoices.
type InvoiceOrderSummary struct {
	ID            string
	Number        string
	CustomerName  string
	CustomerEmail string
	Currency      string
	TotalMinor    int64
}

// InvoiceTemplate represents a selectable template option.
type InvoiceTemplate struct {
	ID          string
	Label       string
	Description string
	Default     bool
}

// InvoiceLanguage represents a selectable language option.
type InvoiceLanguage struct {
	Code    string
	Label   string
	Default bool
}

// InvoiceRecord describes an issued (or pending) invoice.
type InvoiceRecord struct {
	ID            string
	Number        string
	Status        string
	StatusTone    string
	IssuedAt      time.Time
	DeliveryEmail string
	Note          string
	Actor         string
	PDFURL        string
	TemplateID    string
	Language      string
	JobID         string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// InvoiceIssueRequest contains parameters required to issue an invoice.
type InvoiceIssueRequest struct {
	OrderID       string
	TemplateID    string
	Language      string
	DeliveryEmail string
	Note          string
	ActorID       string
	ActorEmail    string
}

// InvoiceIssueResult returns information about a newly issued invoice.
type InvoiceIssueResult struct {
	OrderID string
	Invoice InvoiceRecord
	Job     *InvoiceJob
}

// InvoiceJob represents an asynchronous invoice generation job.
type InvoiceJob struct {
	ID          string
	Status      string
	StatusTone  string
	SubmittedAt time.Time
	Message     string
}

// InvoiceJobStatus reports the latest status of an asynchronous invoice job.
type InvoiceJobStatus struct {
	OrderID string
	Invoice InvoiceRecord
	Job     InvoiceJob
	Done    bool
}

// ExportFormat declares the supported export output formats.
type ExportFormat string

const (
	// ExportFormatCSV streams CSV data for spreadsheets.
	ExportFormatCSV ExportFormat = "csv"
	// ExportFormatPDF renders printable PDF bundles.
	ExportFormatPDF ExportFormat = "pdf"
)

// BulkExportRequest captures parameters for starting a background export or print job.
type BulkExportRequest struct {
	Format     ExportFormat
	OrderIDs   []string
	Query      Query
	ActorID    string
	ActorEmail string
}

// ExportJob stores metadata about an export job visible in the UI.
type ExportJob struct {
	ID              string
	Format          ExportFormat
	Status          string
	StatusTone      string
	Message         string
	SubmittedAt     time.Time
	CompletedAt     *time.Time
	Progress        int
	ProcessedOrders int
	TotalOrders     int
	Fields          []string
	DownloadURL     string
}

// ExportJobStatus reports the current status for a background export job.
type ExportJobStatus struct {
	Job  ExportJob
	Done bool
}

// InvoiceValidationError indicates validation issues for invoice issuance.
type InvoiceValidationError struct {
	Message     string
	FieldErrors map[string]string
}

// Error implements the error interface.
func (e *InvoiceValidationError) Error() string {
	if e == nil {
		return "invalid invoice request"
	}
	msg := strings.TrimSpace(e.Message)
	if msg == "" {
		msg = "invalid invoice request"
	}
	return msg
}
