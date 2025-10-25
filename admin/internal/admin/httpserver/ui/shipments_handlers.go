package ui

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"

	custommw "finitefield.org/hanko-admin/internal/admin/httpserver/middleware"
	adminshipments "finitefield.org/hanko-admin/internal/admin/shipments"
	shipmentstpl "finitefield.org/hanko-admin/internal/admin/templates/shipments"
)

const (
	defaultShipmentsPageSize = 10
	dateInputLayout          = "2006-01-02"
)

type shipmentsRequest struct {
	query adminshipments.ListQuery
	state shipmentstpl.QueryState
}

type trackingRequest struct {
	query adminshipments.TrackingQuery
	state shipmentstpl.TrackingQueryState
}

func buildShipmentsRequest(r *http.Request) shipmentsRequest {
	raw := r.URL.Query()

	status := parseBatchStatus(raw.Get("status"))
	carrier := strings.TrimSpace(raw.Get("carrier"))
	facility := strings.TrimSpace(raw.Get("facility"))
	page := parsePositiveIntDefault(raw.Get("page"), 1)
	pageSize := parsePositiveIntDefault(raw.Get("pageSize"), defaultShipmentsPageSize)

	startStr := strings.TrimSpace(raw.Get("start"))
	endStr := strings.TrimSpace(raw.Get("end"))

	var start, end *time.Time
	if startStr != "" {
		if ts, err := time.Parse(dateInputLayout, startStr); err == nil {
			start = &ts
			startStr = ts.Format(dateInputLayout)
		} else {
			startStr = ""
		}
	}
	if endStr != "" {
		if ts, err := time.Parse(dateInputLayout, endStr); err == nil {
			end = &ts
			endStr = ts.Format(dateInputLayout)
		} else {
			endStr = ""
		}
	}

	selected := strings.TrimSpace(raw.Get("selected"))

	query := adminshipments.ListQuery{
		Status:   status,
		Carrier:  carrier,
		Facility: facility,
		Start:    start,
		End:      end,
		Page:     page,
		PageSize: pageSize,
		Selected: selected,
	}

	state := shipmentstpl.QueryState{
		Status:   string(status),
		Carrier:  carrier,
		Facility: facility,
		Start:    startStr,
		End:      endStr,
		Page:     page,
		PageSize: pageSize,
		Selected: selected,
		RawQuery: raw.Encode(),
	}

	return shipmentsRequest{
		query: query,
		state: state,
	}
}

func buildTrackingRequest(r *http.Request) trackingRequest {
	raw := r.URL.Query()

	status := parseTrackingStatus(raw.Get("status"))
	carrier := strings.TrimSpace(raw.Get("carrier"))
	lane := strings.TrimSpace(raw.Get("lane"))
	region := strings.TrimSpace(raw.Get("region"))
	delay := strings.TrimSpace(raw.Get("delay"))
	page := parsePositiveIntDefault(raw.Get("page"), 1)
	pageSize := parsePositiveIntDefault(raw.Get("pageSize"), 20)

	query := adminshipments.TrackingQuery{
		Status:      status,
		Carrier:     carrier,
		Lane:        lane,
		Destination: region,
		DelayWindow: delay,
		Page:        page,
		PageSize:    pageSize,
	}

	state := shipmentstpl.TrackingQueryState{
		Status:      string(status),
		Carrier:     carrier,
		Lane:        lane,
		Region:      region,
		DelayWindow: delay,
		Page:        page,
		PageSize:    pageSize,
		RawQuery:    trackingRawQuery(query),
	}

	return trackingRequest{query: query, state: state}
}

func trackingRawQuery(query adminshipments.TrackingQuery) string {
	values := url.Values{}
	if query.Status != "" {
		values.Set("status", string(query.Status))
	}
	if query.Carrier != "" {
		values.Set("carrier", query.Carrier)
	}
	if query.Lane != "" {
		values.Set("lane", query.Lane)
	}
	if query.Destination != "" {
		values.Set("region", query.Destination)
	}
	if query.DelayWindow != "" {
		values.Set("delay", query.DelayWindow)
	}
	if query.Page > 1 {
		values.Set("page", strconv.Itoa(query.Page))
	}
	if query.PageSize > 0 && query.PageSize != 20 {
		values.Set("pageSize", strconv.Itoa(query.PageSize))
	}
	return values.Encode()
}

func parseBatchStatus(value string) adminshipments.BatchStatus {
	switch strings.TrimSpace(value) {
	case string(adminshipments.BatchStatusDraft):
		return adminshipments.BatchStatusDraft
	case string(adminshipments.BatchStatusQueued):
		return adminshipments.BatchStatusQueued
	case string(adminshipments.BatchStatusRunning):
		return adminshipments.BatchStatusRunning
	case string(adminshipments.BatchStatusCompleted):
		return adminshipments.BatchStatusCompleted
	case string(adminshipments.BatchStatusFailed):
		return adminshipments.BatchStatusFailed
	default:
		return ""
	}
}

func parseTrackingStatus(value string) adminshipments.TrackingStatus {
	switch strings.TrimSpace(value) {
	case string(adminshipments.TrackingStatusInTransit):
		return adminshipments.TrackingStatusInTransit
	case string(adminshipments.TrackingStatusOutForDelivery):
		return adminshipments.TrackingStatusOutForDelivery
	case string(adminshipments.TrackingStatusLabelCreated):
		return adminshipments.TrackingStatusLabelCreated
	case string(adminshipments.TrackingStatusDelivered):
		return adminshipments.TrackingStatusDelivered
	case string(adminshipments.TrackingStatusException):
		return adminshipments.TrackingStatusException
	default:
		return ""
	}
}

// ShipmentsBatchesPage renders the batches page.
func (h *Handlers) ShipmentsBatchesPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	req := buildShipmentsRequest(r)

	result, err := h.shipments.ListBatches(ctx, user.Token, req.query)
	errMsg := ""
	if err != nil {
		log.Printf("shipments: list batches failed: %v", err)
		errMsg = "出荷バッチの取得に失敗しました。時間を置いて再度お試しください。"
		result = adminshipments.ListResult{}
	}

	basePath := custommw.BasePathFromContext(ctx)
	table := shipmentstpl.TablePayload(basePath, req.state, result, errMsg)
	selectedID := table.SelectedID
	req.state.Selected = selectedID

	drawer := shipmentstpl.DrawerPayload(adminshipments.BatchDetail{}, selectedID)
	if errMsg == "" && selectedID != "" {
		detail, detailErr := h.shipments.BatchDetail(ctx, user.Token, selectedID)
		if detailErr != nil {
			if errors.Is(detailErr, adminshipments.ErrBatchNotFound) {
				log.Printf("shipments: batch detail not found: %s", selectedID)
			} else {
				log.Printf("shipments: batch detail failed: %v", detailErr)
			}
		} else {
			drawer = shipmentstpl.DrawerPayload(detail, selectedID)
		}
	}

	page := shipmentstpl.BuildPageData(basePath, req.state, result, table, drawer)

	templ.Handler(shipmentstpl.Index(page)).ServeHTTP(w, r)
}

// ShipmentsBatchesTable renders the table fragment.
func (h *Handlers) ShipmentsBatchesTable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	req := buildShipmentsRequest(r)
	result, err := h.shipments.ListBatches(ctx, user.Token, req.query)
	errMsg := ""
	if err != nil {
		log.Printf("shipments: list batches failed: %v", err)
		errMsg = "出荷バッチの取得に失敗しました。時間を置いて再度お試しください。"
		result = adminshipments.ListResult{}
	}

	basePath := custommw.BasePathFromContext(ctx)
	table := shipmentstpl.TablePayload(basePath, req.state, result, errMsg)
	req.state.Selected = table.SelectedID

	if canonical := canonicalShipmentsURL(basePath, req.state, table.SelectedID); canonical != "" {
		w.Header().Set("HX-Push-Url", canonical)
	}
	triggerShipmentsSelect(w, table.SelectedID)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templ.Handler(shipmentstpl.Table(table)).ServeHTTP(w, r)
}

// ShipmentsTrackingPage renders the shipment tracking monitor.
func (h *Handlers) ShipmentsTrackingPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	req := buildTrackingRequest(r)
	result, err := h.shipments.ListTracking(ctx, user.Token, req.query)
	errMsg := ""
	if err != nil {
		log.Printf("shipments: list tracking failed: %v", err)
		errMsg = "配送状況の取得に失敗しました。時間を置いて再度お試しください。"
		result = adminshipments.TrackingResult{
			Summary: adminshipments.TrackingSummary{
				LastRefresh:     time.Now(),
				RefreshInterval: 30 * time.Second,
			},
		}
	}

	basePath := custommw.BasePathFromContext(ctx)
	table := shipmentstpl.TrackingTablePayload(basePath, req.state, result, errMsg)
	page := shipmentstpl.BuildTrackingPageData(basePath, req.state, result, table)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templ.Handler(shipmentstpl.TrackingIndex(page)).ServeHTTP(w, r)
}

// ShipmentsTrackingTable renders the tracking table fragment.
func (h *Handlers) ShipmentsTrackingTable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	req := buildTrackingRequest(r)
	result, err := h.shipments.ListTracking(ctx, user.Token, req.query)
	errMsg := ""
	if err != nil {
		log.Printf("shipments: list tracking failed: %v", err)
		errMsg = "配送状況の取得に失敗しました。時間を置いて再度お試しください。"
		result = adminshipments.TrackingResult{
			Summary: adminshipments.TrackingSummary{
				LastRefresh:     time.Now(),
				RefreshInterval: 30 * time.Second,
			},
		}
	}

	basePath := custommw.BasePathFromContext(ctx)
	table := shipmentstpl.TrackingTablePayload(basePath, req.state, result, errMsg)

	if canonical := canonicalTrackingURL(basePath, req.state); canonical != "" {
		w.Header().Set("HX-Push-Url", canonical)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templ.Handler(shipmentstpl.TrackingTable(table)).ServeHTTP(w, r)
}

// ShipmentsBatchDrawer renders the drawer fragment.
func (h *Handlers) ShipmentsBatchDrawer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	batchID := strings.TrimSpace(chi.URLParam(r, "batchID"))
	if batchID == "" {
		http.Error(w, "バッチIDが指定されていません。", http.StatusBadRequest)
		return
	}

	selected := strings.TrimSpace(r.URL.Query().Get("selected"))
	if selected == "" {
		selected = batchID
	}

	req := buildShipmentsRequest(r)
	req.state.Selected = selected

	detail, err := h.shipments.BatchDetail(ctx, user.Token, batchID)
	if err != nil {
		log.Printf("shipments: batch detail failed: %v", err)
		http.Error(w, "バッチ詳細の取得に失敗しました。", http.StatusBadGateway)
		return
	}

	drawer := shipmentstpl.DrawerPayload(detail, selected)
	triggerShipmentsSelect(w, selected)

	basePath := custommw.BasePathFromContext(ctx)
	if canonical := canonicalShipmentsURL(basePath, req.state, selected); canonical != "" {
		w.Header().Set("HX-Push-Url", canonical)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templ.Handler(shipmentstpl.DetailDrawer(drawer)).ServeHTTP(w, r)
}

// ShipmentsCreateBatch acknowledges batch creation requests.
func (h *Handlers) ShipmentsCreateBatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if _, ok := custommw.UserFromContext(ctx); !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	triggerToast(w, "バッチ作成リクエストを受け付けました。ラベル生成を開始します。", "info")
	w.WriteHeader(http.StatusNoContent)
}

// ShipmentsRegenerateLabels acknowledges label regeneration.
func (h *Handlers) ShipmentsRegenerateLabels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if _, ok := custommw.UserFromContext(ctx); !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "リクエストの解析に失敗しました。", http.StatusBadRequest)
		return
	}
	batchID := strings.TrimSpace(r.FormValue("batchID"))
	if batchID == "" {
		http.Error(w, "バッチが選択されていません。", http.StatusBadRequest)
		return
	}

	triggerToast(w, fmt.Sprintf("バッチ %s のラベル再生成を開始しました。", batchID), "success")
	triggerShipmentsSelect(w, batchID)
	w.WriteHeader(http.StatusNoContent)
}

// ShipmentsCreateOrderShipment acknowledges individual shipment generation.
func (h *Handlers) ShipmentsCreateOrderShipment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if _, ok := custommw.UserFromContext(ctx); !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		http.Error(w, "注文IDが指定されていません。", http.StatusBadRequest)
		return
	}

	triggerToast(w, fmt.Sprintf("注文 %s の出荷ラベル生成をキューに投入しました。", orderID), "info")
	w.WriteHeader(http.StatusNoContent)
}

func canonicalShipmentsURL(basePath string, state shipmentstpl.QueryState, selected string) string {
	values := url.Values{}
	if strings.TrimSpace(state.Status) != "" {
		values.Set("status", strings.TrimSpace(state.Status))
	}
	if strings.TrimSpace(state.Carrier) != "" {
		values.Set("carrier", strings.TrimSpace(state.Carrier))
	}
	if strings.TrimSpace(state.Facility) != "" {
		values.Set("facility", strings.TrimSpace(state.Facility))
	}
	if strings.TrimSpace(state.Start) != "" {
		values.Set("start", strings.TrimSpace(state.Start))
	}
	if strings.TrimSpace(state.End) != "" {
		values.Set("end", strings.TrimSpace(state.End))
	}
	if state.Page > 1 {
		values.Set("page", strconv.Itoa(state.Page))
	}
	if state.PageSize > 0 && state.PageSize != defaultShipmentsPageSize {
		values.Set("pageSize", strconv.Itoa(state.PageSize))
	}
	if strings.TrimSpace(selected) != "" {
		values.Set("selected", strings.TrimSpace(selected))
	}

	base := strings.TrimSpace(basePath)
	if base == "" {
		base = "/admin"
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	if len(base) > 1 {
		base = strings.TrimRight(base, "/")
	}
	path := base + "/shipments/batches"
	query := values.Encode()
	if query == "" {
		return path
	}
	return path + "?" + query
}

func canonicalTrackingURL(basePath string, state shipmentstpl.TrackingQueryState) string {
	values := url.Values{}
	if strings.TrimSpace(state.Status) != "" {
		values.Set("status", strings.TrimSpace(state.Status))
	}
	if strings.TrimSpace(state.Carrier) != "" {
		values.Set("carrier", strings.TrimSpace(state.Carrier))
	}
	if strings.TrimSpace(state.Lane) != "" {
		values.Set("lane", strings.TrimSpace(state.Lane))
	}
	if strings.TrimSpace(state.Region) != "" {
		values.Set("region", strings.TrimSpace(state.Region))
	}
	if strings.TrimSpace(state.DelayWindow) != "" {
		values.Set("delay", strings.TrimSpace(state.DelayWindow))
	}
	if state.Page > 1 {
		values.Set("page", strconv.Itoa(state.Page))
	}
	if state.PageSize > 0 && state.PageSize != 20 {
		values.Set("pageSize", strconv.Itoa(state.PageSize))
	}

	base := strings.TrimSpace(basePath)
	if base == "" {
		base = "/admin"
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	if len(base) > 1 {
		base = strings.TrimRight(base, "/")
	}
	path := base + "/shipments/tracking"
	if encoded := values.Encode(); encoded != "" {
		return path + "?" + encoded
	}
	return path
}

func triggerShipmentsSelect(w http.ResponseWriter, batchID string) {
	if strings.TrimSpace(batchID) == "" {
		return
	}

	payload := map[string]any{
		"shipments:select": map[string]string{
			"id": strings.TrimSpace(batchID),
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("shipments: marshal HX-Trigger payload failed: %v", err)
		return
	}
	headerName := "HX-Trigger-After-Swap"
	existing := strings.TrimSpace(w.Header().Get(headerName))
	if existing != "" {
		if strings.HasPrefix(existing, "[") {
			existing = strings.TrimRight(existing, "]")
			if !strings.HasSuffix(existing, "[") && existing != "[" {
				existing += ","
			}
			w.Header().Set(headerName, existing+string(data)+"]")
		} else {
			w.Header().Set(headerName, fmt.Sprintf("[%s,%s]", existing, string(data)))
		}
		return
	}
	w.Header().Set(headerName, string(data))
}
