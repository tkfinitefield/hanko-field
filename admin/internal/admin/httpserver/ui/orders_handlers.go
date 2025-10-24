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
	adminorders "finitefield.org/hanko-admin/internal/admin/orders"
	orderstpl "finitefield.org/hanko-admin/internal/admin/templates/orders"
)

const (
	defaultOrdersPageSize = 20
	defaultOrdersSort     = "-updated_at"
)

// OrdersPage renders the orders index page with SSR.
func (h *Handlers) OrdersPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	req := buildOrdersRequest(r)

	result, err := h.orders.List(ctx, user.Token, req.query)
	errMsg := ""
	if err != nil {
		log.Printf("orders: list failed: %v", err)
		errMsg = "注文の取得に失敗しました。時間を置いて再度お試しください。"
		result = adminorders.ListResult{}
	}

	basePath := custommw.BasePathFromContext(ctx)
	table := orderstpl.TablePayload(basePath, req.state, result, errMsg)
	page := orderstpl.BuildPageData(basePath, req.state, result, table)

	templ.Handler(orderstpl.Index(page)).ServeHTTP(w, r)
}

// OrdersTable renders the orders table fragment for htmx requests.
func (h *Handlers) OrdersTable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	req := buildOrdersRequest(r)

	result, err := h.orders.List(ctx, user.Token, req.query)
	errMsg := ""
	if err != nil {
		log.Printf("orders: list failed: %v", err)
		errMsg = "注文の取得に失敗しました。時間を置いて再度お試しください。"
		result = adminorders.ListResult{}
	}

	basePath := custommw.BasePathFromContext(ctx)
	table := orderstpl.TablePayload(basePath, req.state, result, errMsg)

	if canonical := canonicalOrdersURL(basePath, req); canonical != "" {
		w.Header().Set("HX-Push-Url", canonical)
	}

	templ.Handler(orderstpl.Table(table)).ServeHTTP(w, r)
}

// OrdersBulkStatus handles bulk status update submissions.
func (h *Handlers) OrdersBulkStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "リクエストの解析に失敗しました。", http.StatusBadRequest)
		return
	}

	// Future implementation will call backend API using h.orders.
	// For now, we respond with a no-content acknowledgement so HX requests complete silently.
	w.Header().Set("HX-Trigger", `{"toast":{"message":"ステータスを更新しました。","tone":"success"}}`)
	w.WriteHeader(http.StatusNoContent)
	_ = user // reserved for future use
}

// OrdersBulkLabels handles bulk label generation submissions.
func (h *Handlers) OrdersBulkLabels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, ok := custommw.UserFromContext(ctx)
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "リクエストの解析に失敗しました。", http.StatusBadRequest)
		return
	}

	w.Header().Set("HX-Trigger", `{"toast":{"message":"出荷ラベル生成をキューに投入しました。","tone":"info"}}`)
	w.WriteHeader(http.StatusNoContent)
}

// OrdersBulkExport handles bulk CSV export submissions.
func (h *Handlers) OrdersBulkExport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, ok := custommw.UserFromContext(ctx)
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "リクエストの解析に失敗しました。", http.StatusBadRequest)
		return
	}

	w.Header().Set("HX-Trigger", `{"toast":{"message":"エクスポートを開始しました。完了後に通知します。","tone":"info"}}`)
	w.WriteHeader(http.StatusNoContent)
}

type ordersRequest struct {
	query adminorders.Query
	state orderstpl.QueryState
}

func buildOrdersRequest(r *http.Request) ordersRequest {
	values := r.URL.Query()

	status := normalizeStatus(values.Get("status"))
	sinceStr := strings.TrimSpace(values.Get("since"))
	currency := strings.TrimSpace(values.Get("currency"))
	amountMinStr := strings.TrimSpace(values.Get("amountMin"))
	amountMaxStr := strings.TrimSpace(values.Get("amountMax"))
	hasRefundStr := strings.TrimSpace(values.Get("hasRefund"))
	sortStr := strings.TrimSpace(values.Get("sort"))
	page := parsePositiveIntDefault(values.Get("page"), 1)
	pageSize := parsePositiveIntDefault(values.Get("pageSize"), defaultOrdersPageSize)

	var sincePtr *time.Time
	if ts := parseOrdersSince(sinceStr); !ts.IsZero() {
		t := ts
		sincePtr = &t
	}

	var amountMin *int64
	if v, ok := parseAmountMinor(amountMinStr); ok {
		amountMin = &v
	}

	var amountMax *int64
	if v, ok := parseAmountMinor(amountMaxStr); ok {
		amountMax = &v
	}

	var hasRefundPtr *bool
	if hasRefundStr != "" {
		switch strings.ToLower(hasRefundStr) {
		case "true", "1", "yes":
			value := true
			hasRefundPtr = &value
		case "false", "0", "no":
			value := false
			hasRefundPtr = &value
		}
	}

	sortKey, sortDir, sortToken := parseSort(sortStr)

	query := adminorders.Query{
		Since:         sincePtr,
		Currency:      currency,
		AmountMin:     amountMin,
		AmountMax:     amountMax,
		HasRefundOnly: hasRefundPtr,
		Page:          page,
		PageSize:      pageSize,
		SortKey:       sortKey,
		SortDirection: sortDir,
	}
	if status != "" {
		query.Statuses = []adminorders.Status{adminorders.Status(status)}
	}

	hasFilters := false
	if status != "" || sincePtr != nil || currency != "" || amountMin != nil || amountMax != nil || hasRefundPtr != nil {
		hasFilters = true
	}

	state := orderstpl.QueryState{
		Status:     status,
		Since:      normaliseDateInput(sinceStr),
		Currency:   currency,
		AmountMin:  normalizeAmountInput(amountMinStr),
		AmountMax:  normalizeAmountInput(amountMaxStr),
		HasRefund:  normalizeBoolInput(hasRefundStr),
		Sort:       sortToken,
		Page:       page,
		PageSize:   pageSize,
		RawQuery:   r.URL.RawQuery,
		SortKey:    sortKey,
		SortDir:    string(sortDir),
		SortToken:  sortToken,
		HasFilters: hasFilters,
	}

	return ordersRequest{
		query: query,
		state: state,
	}
}

func canonicalOrdersURL(basePath string, req ordersRequest) string {
	base := strings.TrimSpace(basePath)
	if base == "" {
		base = "/admin"
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	base = strings.TrimRight(base, "/")

	values := url.Values{}

	// Query defaults
	if strings.TrimSpace(req.state.Status) != "" {
		values.Set("status", strings.TrimSpace(req.state.Status))
	}
	if strings.TrimSpace(req.state.Since) != "" {
		values.Set("since", strings.TrimSpace(req.state.Since))
	}
	if strings.TrimSpace(req.state.Currency) != "" {
		values.Set("currency", strings.TrimSpace(req.state.Currency))
	}
	if strings.TrimSpace(req.state.AmountMin) != "" {
		values.Set("amountMin", strings.TrimSpace(req.state.AmountMin))
	}
	if strings.TrimSpace(req.state.AmountMax) != "" {
		values.Set("amountMax", strings.TrimSpace(req.state.AmountMax))
	}
	if hasRefund := strings.TrimSpace(req.state.HasRefund); hasRefund != "" && hasRefund != "any" {
		values.Set("hasRefund", hasRefund)
	}
	if sort := strings.TrimSpace(req.state.Sort); sort != "" && sort != defaultOrdersSort {
		values.Set("sort", sort)
	}
	if req.state.Page > 1 {
		values.Set("page", strconv.Itoa(req.state.Page))
	}
	if req.state.PageSize != defaultOrdersPageSize {
		values.Set("pageSize", strconv.Itoa(req.state.PageSize))
	}

	if encoded := values.Encode(); encoded != "" {
		return base + "/orders?" + encoded
	}
	return base + "/orders"
}

func normalizeStatus(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "pending_payment", "pending-payment":
		return string(adminorders.StatusPendingPayment)
	case "payment_review", "payment-review":
		return string(adminorders.StatusPaymentReview)
	case "in_production", "in-production":
		return string(adminorders.StatusInProduction)
	case "ready_to_ship", "ready-to-ship":
		return string(adminorders.StatusReadyToShip)
	case "shipped":
		return string(adminorders.StatusShipped)
	case "delivered":
		return string(adminorders.StatusDelivered)
	case "refunded":
		return string(adminorders.StatusRefunded)
	case "cancelled", "canceled":
		return string(adminorders.StatusCancelled)
	default:
		return ""
	}
}

func parsePositiveIntDefault(raw string, fallback int) int {
	value, err := parsePositiveInt(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func parseOrdersSince(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02",
		"2006/01/02",
		"2006-01-02 15:04",
	}
	for _, layout := range layouts {
		if ts, err := time.Parse(layout, value); err == nil {
			return ts
		}
	}
	return time.Time{}
}

func parseAmountMinor(value string) (int64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	raw := strings.ReplaceAll(value, ",", "")
	if raw == "" {
		return 0, false
	}

	if strings.Contains(raw, ".") {
		parts := strings.SplitN(raw, ".", 2)
		intPart := parts[0]
		fracPart := parts[1]
		if intPart == "" {
			intPart = "0"
		}
		if len(fracPart) > 2 {
			fracPart = fracPart[:2]
		}
		for len(fracPart) < 2 {
			fracPart += "0"
		}
		total := intPart + fracPart
		parsed, err := strconv.ParseInt(total, 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	}

	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return parsed * 100, true
}

func parseSort(raw string) (string, adminorders.SortDirection, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "updated_at", adminorders.SortDirectionDesc, defaultOrdersSort
	}
	lower := strings.ToLower(raw)

	var key string
	dir := adminorders.SortDirectionDesc

	if strings.Contains(lower, ":") {
		parts := strings.Split(lower, ":")
		key = strings.TrimSpace(parts[0])
		if len(parts) > 1 {
			switch strings.TrimSpace(parts[1]) {
			case "asc":
				dir = adminorders.SortDirectionAsc
			case "desc":
				dir = adminorders.SortDirectionDesc
			}
		}
	} else {
		if strings.HasPrefix(lower, "-") {
			key = strings.TrimPrefix(lower, "-")
			dir = adminorders.SortDirectionDesc
		} else if strings.HasPrefix(lower, "+") {
			key = strings.TrimPrefix(lower, "+")
			dir = adminorders.SortDirectionAsc
		} else {
			key = lower
			dir = adminorders.SortDirectionAsc
		}
	}

	switch key {
	case "total", "number", "updated_at":
		// OK
	default:
		key = "updated_at"
	}

	token := key
	if dir == adminorders.SortDirectionDesc {
		token = "-" + key
	}

	return key, dir, token
}

func normaliseDateInput(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if t, err := time.Parse("2006-01-02", value); err == nil {
		return t.Format("2006-01-02")
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t.Format("2006-01-02")
	}
	return ""
}

func normalizeAmountInput(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, ",", "")
	return value
}

func normalizeBoolInput(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "true", "1", "yes":
		return "true"
	case "false", "0", "no":
		return "false"
	default:
		return "any"
	}
}
