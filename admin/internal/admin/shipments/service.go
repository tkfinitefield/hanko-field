package shipments

import (
	"context"
	"errors"
	"time"
)

// Service exposes shipment batch data for the admin UI.
type Service interface {
	// ListBatches returns batches filtered by the provided query arguments.
	ListBatches(ctx context.Context, token string, query ListQuery) (ListResult, error)
	// BatchDetail returns extended information for a specific batch.
	BatchDetail(ctx context.Context, token, batchID string) (BatchDetail, error)
	// ListTracking returns in-flight shipment tracking records.
	ListTracking(ctx context.Context, token string, query TrackingQuery) (TrackingResult, error)
}

// ErrBatchNotFound indicates the requested batch does not exist.
var ErrBatchNotFound = errors.New("shipment batch not found")

// ListQuery captures filters and pagination arguments.
type ListQuery struct {
	Status   BatchStatus
	Carrier  string
	Facility string
	Start    *time.Time
	End      *time.Time
	Selected string
	Page     int
	PageSize int
}

// ListResult represents a paginated list response.
type ListResult struct {
	Summary    Summary
	Batches    []Batch
	Filters    FilterSummary
	Pagination Pagination
	Generated  time.Time
	SelectedID string
}

// Summary aggregates KPI chips.
type Summary struct {
	Outstanding int
	InProgress  int
	Warnings    int
	LastRun     *time.Time
}

// FilterSummary enumerates available filter choices.
type FilterSummary struct {
	StatusOptions   []StatusOption
	CarrierOptions  []SelectOption
	FacilityOptions []SelectOption
}

// StatusOption represents a status chip.
type StatusOption struct {
	Value BatchStatus
	Label string
	Tone  string
	Count int
}

// SelectOption represents a select dropdown option.
type SelectOption struct {
	Value string
	Label string
	Count int
}

// Pagination describes pagination metadata.
type Pagination struct {
	Page       int
	PageSize   int
	TotalItems int
	NextPage   *int
	PrevPage   *int
}

// BatchStatus captures the lifecycle of a batch.
type BatchStatus string

const (
	// BatchStatusDraft indicates the batch is staged but not yet submitted.
	BatchStatusDraft BatchStatus = "draft"
	// BatchStatusQueued indicates the batch has been enqueued for processing.
	BatchStatusQueued BatchStatus = "queued"
	// BatchStatusRunning indicates labels are being generated.
	BatchStatusRunning BatchStatus = "running"
	// BatchStatusCompleted indicates the batch finished successfully.
	BatchStatusCompleted BatchStatus = "completed"
	// BatchStatusFailed indicates the batch failed and requires attention.
	BatchStatusFailed BatchStatus = "failed"
)

// Batch represents a shipment batch row in the table.
type Batch struct {
	ID               string
	Reference        string
	CreatedAt        time.Time
	ScheduledAt      *time.Time
	Carrier          string
	CarrierLabel     string
	ServiceLevel     string
	Facility         string
	FacilityLabel    string
	Status           BatchStatus
	StatusLabel      string
	StatusTone       string
	OrdersTotal      int
	OrdersPending    int
	LabelsReady      int
	LabelsFailed     int
	ProgressPercent  int
	SLAStatus        string
	SLATone          string
	BadgeIcon        string
	BadgeTone        string
	BadgeLabel       string
	LabelDownloadURL string
	ManifestURL      string
	LastOperator     string
	LastUpdated      time.Time
}

// BatchDetail contains extended batch information for the drawer view.
type BatchDetail struct {
	Batch        Batch
	Orders       []BatchOrder
	Timeline     []TimelineEvent
	PrintHistory []PrintRecord
	Operator     Operator
	Job          JobStatus
}

// BatchOrder summarises an order within the batch.
type BatchOrder struct {
	OrderID      string
	OrderNumber  string
	CustomerName string
	Destination  string
	ServiceLevel string
	LabelStatus  string
	LabelTone    string
	LabelURL     string
	CreatedAt    time.Time
}

// TimelineEvent describes batch progress events.
type TimelineEvent struct {
	Title       string
	Description string
	OccurredAt  time.Time
	Actor       string
	Tone        string
	Icon        string
}

// PrintRecord captures label print history.
type PrintRecord struct {
	Label     string
	Actor     string
	Count     int
	PrintedAt time.Time
	Channel   string
}

// Operator identifies the staff member responsible.
type Operator struct {
	Name      string
	Email     string
	Shift     string
	AvatarURL string
}

// JobStatus summarises asynchronous job progress.
type JobStatus struct {
	State      string
	StateLabel string
	StateTone  string
	Progress   int
	StartedAt  *time.Time
	EndedAt    *time.Time
	Message    string
}

// TrackingQuery captures shipment monitoring filters.
type TrackingQuery struct {
	Status        TrackingStatus
	Carrier       string
	Lane          string
	Destination   string
	DelayWindow   string
	Page          int
	PageSize      int
	AutoRefresh   bool
	LastRefreshed *time.Time
}

// TrackingResult summarises shipment tracking data.
type TrackingResult struct {
	Summary    TrackingSummary
	Shipments  []TrackingShipment
	Filters    TrackingFilters
	Pagination Pagination
	Generated  time.Time
	Alerts     []TrackingAlert
}

// TrackingSummary powers KPI chips.
type TrackingSummary struct {
	ActiveShipments int
	Delayed         int
	Exceptions      int
	LastRefresh     time.Time
	RefreshInterval time.Duration
}

// TrackingFilters captures filter options for the monitor.
type TrackingFilters struct {
	StatusOptions  []TrackingStatusOption
	CarrierOptions []SelectOption
	LaneOptions    []SelectOption
	RegionOptions  []SelectOption
}

// TrackingStatusOption represents a status filter option.
type TrackingStatusOption struct {
	Value TrackingStatus
	Label string
	Tone  string
	Count int
}

// TrackingShipment represents a single shipment row.
type TrackingShipment struct {
	ID               string
	OrderID          string
	OrderNumber      string
	CustomerName     string
	Carrier          string
	CarrierLabel     string
	Status           TrackingStatus
	StatusLabel      string
	StatusTone       string
	TrackingNumber   string
	ServiceLevel     string
	Destination      string
	Region           string
	Lane             string
	LastEvent        string
	LastEventAt      time.Time
	EstimatedArrival *time.Time
	DelayMinutes     int
	SLAStatus        string
	SLATone          string
	ExceptionLabel   string
	ExceptionTone    string
	AlertIcon        string
	OrderURL         string
}

// TrackingAlert represents alert banner entries.
type TrackingAlert struct {
	Label       string
	Description string
	Tone        string
	ActionLabel string
	ActionURL   string
}

// TrackingStatus enumerates shipment tracking states.
type TrackingStatus string

const (
	TrackingStatusLabelCreated   TrackingStatus = "label_created"
	TrackingStatusInTransit      TrackingStatus = "in_transit"
	TrackingStatusOutForDelivery TrackingStatus = "out_for_delivery"
	TrackingStatusDelivered      TrackingStatus = "delivered"
	TrackingStatusException      TrackingStatus = "exception"
)
