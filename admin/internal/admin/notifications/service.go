package notifications

import (
	"context"
	"errors"
	"time"
)

// ErrNotConfigured indicates the notifications service dependency has not been provided.
var ErrNotConfigured = errors.New("notifications service not configured")

// Service defines access to the notifications feed and badge counts.
type Service interface {
	// List returns notifications for the given query filters.
	List(ctx context.Context, token string, query Query) (Feed, error)
	// Badge summarises counts for display in the top bar badge.
	Badge(ctx context.Context, token string) (BadgeCount, error)
}

// Category identifies the origin of a notification.
type Category string

const (
	// CategoryFailedJob represents background job failures.
	CategoryFailedJob Category = "failed_job"
	// CategoryStockAlert represents low or out-of-stock alerts.
	CategoryStockAlert Category = "stock_alert"
	// CategoryShippingException represents shipping related issues.
	CategoryShippingException Category = "shipping_exception"
)

// Severity classifies the urgency of a notification.
type Severity string

const (
	// SeverityCritical indicates immediate action is required.
	SeverityCritical Severity = "critical"
	// SeverityHigh indicates a high-priority item.
	SeverityHigh Severity = "high"
	// SeverityMedium indicates a medium-priority item.
	SeverityMedium Severity = "medium"
	// SeverityLow indicates informational items.
	SeverityLow Severity = "low"
)

// Status describes the current lifecycle stage of a notification.
type Status string

const (
	// StatusOpen indicates a fresh, unassigned notification.
	StatusOpen Status = "open"
	// StatusAcknowledged indicates someone is actively investigating.
	StatusAcknowledged Status = "acknowledged"
	// StatusResolved indicates the underlying issue has been remediated.
	StatusResolved Status = "resolved"
	// StatusSuppressed indicates the alert has been silenced temporarily.
	StatusSuppressed Status = "suppressed"
)

// Query captures filter arguments for listing notifications.
type Query struct {
	Categories []Category
	Severities []Severity
	Statuses   []Status
	Owner      string
	Search     string
	Start      *time.Time
	End        *time.Time
	Limit      int
	Cursor     string
}

// Feed represents a paginated notification feed response.
type Feed struct {
	Items      []Notification
	Total      int
	NextCursor string
	Counts     CountSummary
}

// CountSummary aggregates totals used for badges and filters.
type CountSummary struct {
	Total        int
	Open         int
	Acknowledged int
	Resolved     int
	Suppressed   int
	Critical     int
	Warning      int
	Info         int
}

// BadgeCount represents counts surfaced in the top bar notification badge.
type BadgeCount struct {
	Total    int
	Critical int
	Warning  int
}

// Notification stores a single feed entry.
type Notification struct {
	ID             string
	Category       Category
	Severity       Severity
	Status         Status
	Title          string
	Summary        string
	Resource       ResourceRef
	CreatedAt      time.Time
	Owner          string
	AcknowledgedBy string
	AcknowledgedAt *time.Time
	ResolvedAt     *time.Time
	Links          []Link
	Metadata       []Metadata
	Timeline       []TimelineEvent
}

// ResourceRef identifies an impacted entity.
type ResourceRef struct {
	Kind       string
	Identifier string
	Label      string
	URL        string
}

// Link represents a related action shortcut.
type Link struct {
	Label string
	URL   string
	Icon  string
}

// Metadata captures supporting key/value info.
type Metadata struct {
	Label string
	Value string
	Icon  string
}

// TimelineEvent describes an update within the investigation history.
type TimelineEvent struct {
	Title       string
	Description string
	OccurredAt  time.Time
	Actor       string
	Tone        string
	Icon        string
}
