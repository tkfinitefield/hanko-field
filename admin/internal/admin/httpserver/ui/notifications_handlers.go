package ui

import (
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"

	custommw "finitefield.org/hanko-admin/internal/admin/httpserver/middleware"
	adminnotifications "finitefield.org/hanko-admin/internal/admin/notifications"
	notificationstpl "finitefield.org/hanko-admin/internal/admin/templates/notifications"
)

// NotificationsPage renders the notifications index page.
func (h *Handlers) NotificationsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	params := buildNotificationsRequest(r)

	feed, err := h.notifications.List(ctx, user.Token, params.query)
	errMsg := ""
	if err != nil {
		log.Printf("notifications: list failed: %v", err)
		errMsg = "通知の取得に失敗しました。時間を置いて再度お試しください。"
		feed = adminnotifications.Feed{}
	}

	table := notificationstpl.TablePayload(params.state, feed, errMsg, params.selectedID)
	drawer := notificationstpl.DrawerPayload(feed, table.SelectedID)
	payload := notificationstpl.BuildPageData(custommw.BasePathFromContext(ctx), params.state, table, drawer)

	templ.Handler(notificationstpl.Index(payload)).ServeHTTP(w, r)
}

// NotificationsTable renders the table fragment for htmx updates.
func (h *Handlers) NotificationsTable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	params := buildNotificationsRequest(r)

	feed, err := h.notifications.List(ctx, user.Token, params.query)
	errMsg := ""
	if err != nil {
		log.Printf("notifications: list failed: %v", err)
		errMsg = "通知の取得に失敗しました。時間を置いて再度お試しください。"
		feed = adminnotifications.Feed{}
	}

	table := notificationstpl.TablePayload(params.state, feed, errMsg, params.selectedID)
	params.selectedID = table.SelectedID
	component := notificationstpl.Table(table)

	if canonical := canonicalNotificationsURL(custommw.BasePathFromContext(ctx), params); canonical != "" {
		w.Header().Set("HX-Push-Url", canonical)
	}

	templ.Handler(component).ServeHTTP(w, r)
}

// NotificationsBadge renders the top-bar badge fragment.
func (h *Handlers) NotificationsBadge(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	count, err := h.notifications.Badge(ctx, user.Token)
	if err != nil {
		log.Printf("notifications: badge failed: %v", err)
		count = adminnotifications.BadgeCount{}
	}

	payload := notificationstpl.BadgePayload(custommw.BasePathFromContext(ctx), count)
	templ.Handler(notificationstpl.Badge(payload)).ServeHTTP(w, r)
}

type notificationsRequest struct {
	query      adminnotifications.Query
	state      notificationstpl.QueryState
	selectedID string
}

func buildNotificationsRequest(r *http.Request) notificationsRequest {
	values := r.URL.Query()
	rawCategory := strings.TrimSpace(values.Get("category"))
	rawSeverity := strings.TrimSpace(values.Get("severity"))
	rawStatus := strings.TrimSpace(values.Get("status"))
	rawSearch := strings.TrimSpace(values.Get("q"))
	rawStart := strings.TrimSpace(values.Get("start"))
	rawEnd := strings.TrimSpace(values.Get("end"))
	rawSelected := strings.TrimSpace(values.Get("selected"))
	rawLimit := strings.TrimSpace(values.Get("limit"))

	category := normaliseCategory(rawCategory)
	severity := normaliseSeverity(rawSeverity)
	status := normaliseStatus(rawStatus)

	var startPtr *time.Time
	if ts := parseDate(rawStart); !ts.IsZero() {
		startPtr = &ts
	}
	var endPtr *time.Time
	if ts := parseDate(rawEnd); !ts.IsZero() {
		if startPtr != nil && ts.Before(*startPtr) {
			adjusted := startPtr.Add(24 * time.Hour)
			endPtr = &adjusted
		} else {
			endPtr = &ts
		}
	}

	limit := 0
	if rawLimit != "" {
		if parsed, err := parsePositiveInt(rawLimit); err == nil {
			limit = parsed
		}
	}

	state := notificationstpl.QueryState{
		Category:  category,
		Severity:  severity,
		Status:    status,
		Search:    rawSearch,
		StartDate: normalizeDateInput(rawStart),
		EndDate:   normalizeDateInput(rawEnd),
	}

	query := adminnotifications.Query{
		Search: rawSearch,
		Start:  startPtr,
		End:    endPtr,
		Limit:  limit,
	}
	if category != "" {
		query.Categories = []adminnotifications.Category{adminnotifications.Category(category)}
	}
	if severity != "" {
		query.Severities = []adminnotifications.Severity{adminnotifications.Severity(severity)}
	}
	if status != "" {
		query.Statuses = []adminnotifications.Status{adminnotifications.Status(status)}
	}

	return notificationsRequest{
		query:      query,
		state:      state,
		selectedID: rawSelected,
	}
}

func canonicalNotificationsURL(basePath string, req notificationsRequest) string {
	basePath = strings.TrimSpace(basePath)
	if basePath == "" {
		basePath = "/admin"
	}
	values := url.Values{}
	if req.state.Category != "" {
		values.Set("category", req.state.Category)
	}
	if req.state.Severity != "" {
		values.Set("severity", req.state.Severity)
	}
	if req.state.Status != "" {
		values.Set("status", req.state.Status)
	}
	if req.state.Search != "" {
		values.Set("q", req.state.Search)
	}
	if req.state.StartDate != "" {
		values.Set("start", req.state.StartDate)
	}
	if req.state.EndDate != "" {
		values.Set("end", req.state.EndDate)
	}
	if strings.TrimSpace(req.selectedID) != "" {
		values.Set("selected", strings.TrimSpace(req.selectedID))
	}
	if len(values) == 0 {
		return joinRoute(basePath, "/notifications")
	}
	return joinRoute(basePath, "/notifications") + "?" + values.Encode()
}

func joinRoute(base, suffix string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "/admin"
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	if len(base) > 1 && strings.HasSuffix(base, "/") {
		base = strings.TrimRight(base, "/")
	}

	raw := strings.TrimSpace(suffix)
	if raw == "" {
		return base
	}
	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}
	if base == "/" {
		return raw
	}
	return base + raw
}

func normaliseCategory(value string) string {
	switch strings.TrimSpace(value) {
	case string(adminnotifications.CategoryFailedJob):
		return string(adminnotifications.CategoryFailedJob)
	case string(adminnotifications.CategoryStockAlert):
		return string(adminnotifications.CategoryStockAlert)
	case string(adminnotifications.CategoryShippingException):
		return string(adminnotifications.CategoryShippingException)
	default:
		return ""
	}
}

func normaliseSeverity(value string) string {
	switch strings.TrimSpace(value) {
	case string(adminnotifications.SeverityCritical):
		return string(adminnotifications.SeverityCritical)
	case string(adminnotifications.SeverityHigh):
		return string(adminnotifications.SeverityHigh)
	case string(adminnotifications.SeverityMedium):
		return string(adminnotifications.SeverityMedium)
	case string(adminnotifications.SeverityLow):
		return string(adminnotifications.SeverityLow)
	default:
		return ""
	}
}

func normaliseStatus(value string) string {
	switch strings.TrimSpace(value) {
	case string(adminnotifications.StatusOpen):
		return string(adminnotifications.StatusOpen)
	case string(adminnotifications.StatusAcknowledged):
		return string(adminnotifications.StatusAcknowledged)
	case string(adminnotifications.StatusResolved):
		return string(adminnotifications.StatusResolved)
	case string(adminnotifications.StatusSuppressed):
		return string(adminnotifications.StatusSuppressed)
	default:
		return ""
	}
}

func parsePositiveInt(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	if parsed < 0 {
		parsed = 0
	}
	return parsed, nil
}
