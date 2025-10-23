package ui

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"

	admindashboard "finitefield.org/hanko-admin/internal/admin/dashboard"
	custommw "finitefield.org/hanko-admin/internal/admin/httpserver/middleware"
	"finitefield.org/hanko-admin/internal/admin/profile"
	dashboardtpl "finitefield.org/hanko-admin/internal/admin/templates/dashboard"
	profiletpl "finitefield.org/hanko-admin/internal/admin/templates/profile"
)

// Dependencies collects external services required by the UI handlers.
type Dependencies struct {
	DashboardService admindashboard.Service
	ProfileService   profile.Service
}

// Handlers exposes HTTP handlers for admin UI pages and fragments.
type Handlers struct {
	dashboard admindashboard.Service
	profile   profile.Service
}

// NewHandlers wires the UI handler set.
func NewHandlers(deps Dependencies) *Handlers {
	profileService := deps.ProfileService
	if profileService == nil {
		profileService = profile.NewStaticService(nil)
	}
	dashboardService := deps.DashboardService
	if dashboardService == nil {
		dashboardService = admindashboard.NewStaticService()
	}
	return &Handlers{
		dashboard: dashboardService,
		profile:   profileService,
	}
}

// Dashboard renders the admin dashboard.
func (h *Handlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	since := parseSince(r.URL.Query().Get("since"))

	kpis, err := h.dashboard.FetchKPIs(ctx, user.Token, since)
	kpiFragment := dashboardtpl.KPIFragmentPayload(kpis)
	if err != nil {
		log.Printf("dashboard: fetch kpis failed: %v", err)
		kpiFragment.Error = "KPIの取得に失敗しました。時間を置いて再度お試しください。"
	}

	alerts, err := h.dashboard.FetchAlerts(ctx, user.Token, 0)
	alertsFragment := dashboardtpl.AlertsFragmentPayload(alerts)
	if err != nil {
		log.Printf("dashboard: fetch alerts failed: %v", err)
		alertsFragment.Error = "アラートの取得に失敗しました。"
	}

	activity, err := h.dashboard.FetchActivity(ctx, user.Token, 0)
	if err != nil {
		log.Printf("dashboard: fetch activity failed: %v", err)
		activity = nil
	}

	data := dashboardtpl.BuildPageData(custommw.BasePathFromContext(ctx), kpis, alerts, activity)
	data.KPIFragment = kpiFragment
	data.AlertsFragment = alertsFragment

	templ.Handler(dashboardtpl.Index(data)).ServeHTTP(w, r)
}

// DashboardKPIs serves the KPI fragment for htmx requests.
func (h *Handlers) DashboardKPIs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	since := parseSince(r.URL.Query().Get("since"))
	limit := parseLimit(r.URL.Query().Get("limit"), 0)

	kpis, err := h.dashboard.FetchKPIs(ctx, user.Token, since)
	if limit > 0 && len(kpis) > limit {
		kpis = kpis[:limit]
	}

	payload := dashboardtpl.KPIFragmentPayload(kpis)
	if err != nil {
		log.Printf("dashboard: fetch kpis failed: %v", err)
		payload.Error = "KPIの取得に失敗しました。時間を置いて再度お試しください。"
	}

	templ.Handler(dashboardtpl.KPIFragment(payload)).ServeHTTP(w, r)
}

// DashboardAlerts serves the alerts fragment for htmx requests.
func (h *Handlers) DashboardAlerts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	limit := parseLimit(r.URL.Query().Get("limit"), 0)

	alerts, err := h.dashboard.FetchAlerts(ctx, user.Token, limit)
	payload := dashboardtpl.AlertsFragmentPayload(alerts)
	if err != nil {
		log.Printf("dashboard: fetch alerts failed: %v", err)
		payload.Error = "アラートの取得に失敗しました。"
	}

	templ.Handler(dashboardtpl.AlertsFragment(payload)).ServeHTTP(w, r)
}

func parseSince(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	if ts, err := time.Parse(time.RFC3339, raw); err == nil {
		return &ts
	}
	return nil
}

func parseLimit(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

func (h *Handlers) renderProfilePage(w http.ResponseWriter, r *http.Request) {
	user, ok := custommw.UserFromContext(r.Context())
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	state, err := h.profile.SecurityOverview(r.Context(), user.Token)
	if err != nil {
		log.Printf("profile: fetch security overview failed: %v", err)
		http.Error(w, "セキュリティ情報の取得に失敗しました。時間を置いて再度お試しください。", http.StatusBadGateway)
		return
	}

	payload := profiletpl.PageData{
		UserEmail: user.Email,
		UserName:  user.UID,
		Security:  state,
		CSRFToken: custommw.CSRFTokenFromContext(r.Context()),
	}

	component := profiletpl.Index(payload)
	templ.Handler(component).ServeHTTP(w, r)
}
