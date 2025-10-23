package dashboard

import (
	"fmt"
	"strings"
	"time"

	admindashboard "finitefield.org/hanko-admin/internal/admin/dashboard"
	"finitefield.org/hanko-admin/internal/admin/templates/helpers"
	"finitefield.org/hanko-admin/internal/admin/templates/partials"
)

// PageData represents the full dashboard SSR payload.
type PageData struct {
	Title              string
	KPIFragment        KPIFragmentData
	AlertsFragment     AlertsFragmentData
	Activity           []ActivityItem
	KPIEndpoint        string
	AlertsEndpoint     string
	PollIntervalSecond int
}

// KPIFragmentData holds the KPI cards payload.
type KPIFragmentData struct {
	KPIs  []KPIView
	Error string
}

// KPIView is the rendered representation of a metric card.
type KPIView struct {
	ID        string
	Label     string
	Value     string
	DeltaText string
	Trend     string
	Sparkline []float64
	UpdatedAt time.Time
}

// AlertsFragmentData holds alerts list payload.
type AlertsFragmentData struct {
	Alerts []AlertView
	Error  string
}

// AlertView represents a single alert entry.
type AlertView struct {
	ID        string
	Severity  string
	Title     string
	Message   string
	ActionURL string
	Action    string
	CreatedAt time.Time
}

// ActivityItem represents a recent update displayed on the dashboard.
type ActivityItem struct {
	ID       string
	Icon     string
	Title    string
	Detail   string
	Occurred time.Time
	LinkURL  string
}

// BuildPageData prepares the template payload for SSR rendering.
func BuildPageData(basePath string, kpis []admindashboard.KPI, alerts []admindashboard.Alert, activity []admindashboard.ActivityItem) PageData {
	return PageData{
		Title:              "ダッシュボード",
		KPIFragment:        KPIFragmentPayload(kpis),
		AlertsFragment:     AlertsFragmentPayload(alerts),
		Activity:           ActivityFeedPayload(activity),
		KPIEndpoint:        joinBase(basePath, "/fragments/kpi"),
		AlertsEndpoint:     joinBase(basePath, "/fragments/alerts"),
		PollIntervalSecond: 60,
	}
}

// KPIFragmentPayload prepares KPI data for rendering.
func KPIFragmentPayload(list []admindashboard.KPI) KPIFragmentData {
	return KPIFragmentData{KPIs: toKPIViews(list)}
}

// AlertsFragmentPayload prepares alerts data for rendering.
func AlertsFragmentPayload(list []admindashboard.Alert) AlertsFragmentData {
	return AlertsFragmentData{Alerts: toAlertViews(list)}
}

// ActivityFeedPayload prepares activity items for rendering.
func ActivityFeedPayload(list []admindashboard.ActivityItem) []ActivityItem {
	return toActivityViews(list)
}

// toKPIViews converts service models.
func toKPIViews(list []admindashboard.KPI) []KPIView {
	result := make([]KPIView, 0, len(list))
	for _, item := range list {
		result = append(result, KPIView{
			ID:        item.ID,
			Label:     item.Label,
			Value:     item.Value,
			DeltaText: item.DeltaText,
			Trend:     string(item.Trend),
			Sparkline: append([]float64(nil), item.Sparkline...),
			UpdatedAt: item.UpdatedAt,
		})
	}
	return result
}

// toAlertViews converts service models.
func toAlertViews(list []admindashboard.Alert) []AlertView {
	result := make([]AlertView, 0, len(list))
	for _, item := range list {
		result = append(result, AlertView{
			ID:        item.ID,
			Severity:  item.Severity,
			Title:     item.Title,
			Message:   item.Message,
			ActionURL: item.ActionURL,
			Action:    item.Action,
			CreatedAt: item.CreatedAt,
		})
	}
	return result
}

// toActivityViews converts service models.
func toActivityViews(list []admindashboard.ActivityItem) []ActivityItem {
	result := make([]ActivityItem, 0, len(list))
	for _, item := range list {
		result = append(result, ActivityItem{
			ID:       item.ID,
			Icon:     item.Icon,
			Title:    item.Title,
			Detail:   item.Detail,
			Occurred: item.Occurred,
			LinkURL:  item.LinkURL,
		})
	}
	return result
}

func joinBase(base, suffix string) string {
	base = strings.TrimSpace(base)
	if base == "" || base == "/" {
		base = ""
	}
	if !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}
	path := base + suffix
	// Normalise multiple slashes while preserving http(s) scheme patterns (not expected here).
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func breadcrumbItems() []partials.Breadcrumb {
	return []partials.Breadcrumb{{Label: helpers.I18N("admin.dashboard.breadcrumb")}}
}

func sparklinePoints(values []float64) string {
	if len(values) == 0 {
		return ""
	}
	if len(values) == 1 {
		return "0,50 100,50"
	}

	min := values[0]
	max := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	rangeVal := max - min
	if rangeVal == 0 {
		rangeVal = 1
	}

	points := make([]string, 0, len(values))
	lastIndex := len(values) - 1
	for i, v := range values {
		x := 0.0
		if lastIndex > 0 {
			x = float64(i) / float64(lastIndex) * 100
		}
		y := 100 - ((v - min) / rangeVal * 100)
		points = append(points, fmt.Sprintf("%.1f,%.1f", x, y))
	}
	return strings.Join(points, " ")
}
