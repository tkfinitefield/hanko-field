package dashboard

import (
	"context"
	"errors"
	"time"
)

// ErrNotConfigured indicates the dashboard service dependency has not been provided.
var ErrNotConfigured = errors.New("dashboard service not configured")

// Service exposes data retrieval for dashboard fragments.
type Service interface {
	// FetchKPIs returns summary metrics for the dashboard.
	FetchKPIs(ctx context.Context, token string, since *time.Time) ([]KPI, error)
	// FetchAlerts returns the most recent operational alerts.
	FetchAlerts(ctx context.Context, token string, limit int) ([]Alert, error)
	// FetchActivity returns recent activity items for the feed rail.
	FetchActivity(ctx context.Context, token string, limit int) ([]ActivityItem, error)
}

// KPI represents a dashboard metric card.
type KPI struct {
	ID        string
	Label     string
	Value     string
	DeltaText string
	Trend     Trend
	Sparkline []float64
	UpdatedAt time.Time
}

// Trend describes the direction of a KPI delta.
type Trend string

const (
	// TrendFlat indicates no significant change.
	TrendFlat Trend = "flat"
	// TrendUp indicates a positive change.
	TrendUp Trend = "up"
	// TrendDown indicates a negative change.
	TrendDown Trend = "down"
)

// Alert captures a dashboard alert entry.
type Alert struct {
	ID        string
	Severity  string
	Title     string
	Message   string
	ActionURL string
	Action    string
	CreatedAt time.Time
}

// ActivityItem represents a recent event for the activity feed.
type ActivityItem struct {
	ID        string
	Icon      string
	Title     string
	Detail    string
	Occurred  time.Time
	LinkURL   string
	LinkLabel string
}
