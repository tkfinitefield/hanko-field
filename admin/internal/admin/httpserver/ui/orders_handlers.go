package ui

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"

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

// OrdersStatusModal renders the status update modal for a specific order.
func (h *Handlers) OrdersStatusModal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		http.Error(w, "注文IDが不正です。", http.StatusBadRequest)
		return
	}

	modal, err := h.orders.StatusModal(ctx, user.Token, orderID)
	if err != nil {
		if errors.Is(err, adminorders.ErrOrderNotFound) {
			http.Error(w, "指定された注文が見つかりません。", http.StatusNotFound)
			return
		}
		log.Printf("orders: fetch status modal failed: %v", err)
		http.Error(w, "ステータス更新モーダルの取得に失敗しました。", http.StatusBadGateway)
		return
	}

	basePath := custommw.BasePathFromContext(ctx)
	csrf := custommw.CSRFTokenFromContext(ctx)
	data := orderstpl.StatusModalPayload(basePath, modal, csrf, "", true, "")

	templ.Handler(orderstpl.StatusModal(data)).ServeHTTP(w, r)
}

// OrdersStatusUpdate handles status transition submissions for a single order.
func (h *Handlers) OrdersStatusUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		http.Error(w, "注文IDが不正です。", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "リクエストの解析に失敗しました。", http.StatusBadRequest)
		return
	}

	statusValue := strings.TrimSpace(r.FormValue("status"))
	note := strings.TrimSpace(r.FormValue("note"))
	notify := parseCheckbox(r.FormValue("notifyCustomer"))

	updateReq := adminorders.StatusUpdateRequest{
		Status:         adminorders.Status(statusValue),
		Note:           note,
		NotifyCustomer: notify,
		ActorID:        user.UID,
		ActorEmail:     user.Email,
	}

	result, err := h.orders.UpdateStatus(ctx, user.Token, orderID, updateReq)
	if err != nil {
		var transitionErr *adminorders.StatusTransitionError
		if errors.Is(err, adminorders.ErrOrderNotFound) {
			http.Error(w, "指定された注文が見つかりません。", http.StatusNotFound)
			return
		}
		if errors.As(err, &transitionErr) {
			modal, modalErr := h.orders.StatusModal(ctx, user.Token, orderID)
			if modalErr != nil {
				log.Printf("orders: reload status modal after error failed: %v", modalErr)
				http.Error(w, "ステータス更新に失敗しました。", http.StatusBadGateway)
				return
			}
			basePath := custommw.BasePathFromContext(ctx)
			csrf := custommw.CSRFTokenFromContext(ctx)
			message := transitionErr.Reason
			if strings.TrimSpace(message) == "" {
				message = "ステータスを変更できません。"
			}
			data := orderstpl.StatusModalPayload(basePath, modal, csrf, note, notify, message)
			w.WriteHeader(http.StatusUnprocessableEntity)
			templ.Handler(orderstpl.StatusModal(data)).ServeHTTP(w, r)
			return
		}
		log.Printf("orders: status update failed: %v", err)
		http.Error(w, "ステータス更新に失敗しました。", http.StatusBadGateway)
		return
	}

	basePath := custommw.BasePathFromContext(ctx)
	cell := orderstpl.StatusCellPayload(basePath, result.Order)
	timeline := orderstpl.StatusTimelinePayload(result.Order.ID, result.Timeline)
	success := orderstpl.StatusUpdateSuccessPayload(cell, timeline)

	w.Header().Set("HX-Trigger", `{"toast":{"message":"ステータスを更新しました。","tone":"success"},"modal:close":true}`)
	templ.Handler(orderstpl.StatusUpdateSuccess(success)).ServeHTTP(w, r)
}

// OrdersRefundModal renders the refund modal for a specific order.
func (h *Handlers) OrdersRefundModal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		http.Error(w, "注文IDが不正です。", http.StatusBadRequest)
		return
	}

	modal, err := h.orders.RefundModal(ctx, user.Token, orderID)
	if err != nil {
		if errors.Is(err, adminorders.ErrOrderNotFound) {
			http.Error(w, "指定された注文が見つかりません。", http.StatusNotFound)
			return
		}
		log.Printf("orders: fetch refund modal failed: %v", err)
		http.Error(w, "返金モーダルの取得に失敗しました。", http.StatusBadGateway)
		return
	}

	basePath := custommw.BasePathFromContext(ctx)
	csrf := custommw.CSRFTokenFromContext(ctx)
	data := orderstpl.RefundModalPayload(basePath, modal, csrf, orderstpl.RefundFormState{
		PaymentID:      "",
		Amount:         "",
		Reason:         "",
		NotifyCustomer: true,
	}, "", nil)

	templ.Handler(orderstpl.RefundModal(data)).ServeHTTP(w, r)
}

// OrdersSubmitRefund processes refund submissions for a specific order payment.
func (h *Handlers) OrdersSubmitRefund(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		http.Error(w, "注文IDが不正です。", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "リクエストの解析に失敗しました。", http.StatusBadRequest)
		return
	}

	paymentID := strings.TrimSpace(r.FormValue("paymentID"))
	amountStr := strings.TrimSpace(r.FormValue("amount"))
	reason := strings.TrimSpace(r.FormValue("reason"))
	notify := parseCheckbox(r.FormValue("notifyCustomer"))

	formState := orderstpl.RefundFormState{
		PaymentID:      paymentID,
		Amount:         amountStr,
		Reason:         reason,
		NotifyCustomer: notify,
	}

	fieldErrors := map[string]string{}
	if paymentID == "" {
		fieldErrors["paymentID"] = "返金対象の支払いを選択してください。"
	}

	amountMinor, parsed := parseAmountMinor(amountStr)
	if !parsed {
		fieldErrors["amount"] = "返金金額を正しく入力してください。"
	}

	if strings.TrimSpace(reason) == "" {
		fieldErrors["reason"] = "返金理由を入力してください。"
	}

	if len(fieldErrors) > 0 {
		h.renderRefundModalError(w, r, user, orderID, formState, "入力内容を確認してください。", fieldErrors, http.StatusUnprocessableEntity)
		return
	}

	req := adminorders.RefundRequest{
		PaymentID:      paymentID,
		AmountMinor:    amountMinor,
		Reason:         reason,
		NotifyCustomer: notify,
		ActorID:        user.UID,
		ActorEmail:     user.Email,
	}

	_, err := h.orders.SubmitRefund(ctx, user.Token, orderID, req)
	if err != nil {
		if errors.Is(err, adminorders.ErrOrderNotFound) {
			http.Error(w, "指定された注文が見つかりません。", http.StatusNotFound)
			return
		}
		if errors.Is(err, adminorders.ErrPaymentNotFound) {
			fieldErrors["paymentID"] = "選択した支払いが見つかりません。"
			h.renderRefundModalError(w, r, user, orderID, formState, "返金対象の支払いが存在しません。", fieldErrors, http.StatusUnprocessableEntity)
			return
		}
		var valErr *adminorders.RefundValidationError
		if errors.As(err, &valErr) {
			fieldErrors = mergeFieldErrors(fieldErrors, valErr.FieldErrors)
			message := strings.TrimSpace(valErr.Message)
			if message == "" {
				message = "返金処理に失敗しました。入力内容を確認してください。"
			}
			h.renderRefundModalError(w, r, user, orderID, formState, message, fieldErrors, http.StatusUnprocessableEntity)
			return
		}
		if errors.Is(err, adminorders.ErrRefundFailed) {
			h.renderRefundModalError(w, r, user, orderID, formState, "返金処理に失敗しました。時間を置いて再度お試しください。", fieldErrors, http.StatusUnprocessableEntity)
			return
		}
		log.Printf("orders: submit refund failed: %v", err)
		http.Error(w, "返金処理に失敗しました。", http.StatusBadGateway)
		return
	}

	w.Header().Set("HX-Trigger", `{"toast":{"message":"返金を登録しました。","tone":"success"},"modal:close":true,"refresh:fragment":{"targets":["[data-order-payments]","[data-order-summary]"]}}`)
	w.WriteHeader(http.StatusNoContent)
}

// OrdersInvoiceModal renders the invoice issuance modal for a specific order.
func (h *Handlers) OrdersInvoiceModal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		http.Error(w, "注文IDが不正です。", http.StatusBadRequest)
		return
	}

	modal, err := h.orders.InvoiceModal(ctx, user.Token, orderID)
	if err != nil {
		if errors.Is(err, adminorders.ErrOrderNotFound) {
			http.Error(w, "指定された注文が見つかりません。", http.StatusNotFound)
			return
		}
		log.Printf("orders: fetch invoice modal failed: %v", err)
		http.Error(w, "領収書モーダルの取得に失敗しました。", http.StatusBadGateway)
		return
	}

	basePath := custommw.BasePathFromContext(ctx)
	csrf := custommw.CSRFTokenFromContext(ctx)
	data := orderstpl.InvoiceModalPayload(basePath, modal, csrf, orderstpl.InvoiceFormState{}, "", nil)

	templ.Handler(orderstpl.InvoiceModal(data)).ServeHTTP(w, r)
}

// InvoicesIssue processes invoice issuance requests.
func (h *Handlers) InvoicesIssue(w http.ResponseWriter, r *http.Request) {
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

	orderID := strings.TrimSpace(r.FormValue("orderID"))
	templateID := strings.TrimSpace(r.FormValue("templateID"))
	language := strings.TrimSpace(r.FormValue("language"))
	email := strings.TrimSpace(r.FormValue("email"))
	note := strings.TrimSpace(r.FormValue("note"))

	if orderID == "" {
		http.Error(w, "注文IDが不正です。", http.StatusBadRequest)
		return
	}

	formState := orderstpl.InvoiceFormState{
		TemplateID: templateID,
		Language:   language,
		Email:      email,
		Note:       note,
	}

	request := adminorders.InvoiceIssueRequest{
		OrderID:       orderID,
		TemplateID:    templateID,
		Language:      language,
		DeliveryEmail: email,
		Note:          note,
		ActorID:       user.UID,
		ActorEmail:    user.Email,
	}

	result, err := h.orders.IssueInvoice(ctx, user.Token, request)
	if err != nil {
		if errors.Is(err, adminorders.ErrOrderNotFound) {
			http.Error(w, "指定された注文が見つかりません。", http.StatusNotFound)
			return
		}
		var valErr *adminorders.InvoiceValidationError
		if errors.As(err, &valErr) {
			h.renderInvoiceModalError(w, r, user, orderID, formState, valErr.Message, valErr.FieldErrors, http.StatusUnprocessableEntity)
			return
		}
		if errors.Is(err, adminorders.ErrInvoiceTemplateNotFound) {
			h.renderInvoiceModalError(w, r, user, orderID, formState, "選択したテンプレートが見つかりません。", map[string]string{"templateID": "選択したテンプレートが見つかりません。"}, http.StatusUnprocessableEntity)
			return
		}
		log.Printf("orders: issue invoice failed: %v", err)
		http.Error(w, "領収書の発行に失敗しました。", http.StatusBadGateway)
		return
	}

	basePath := custommw.BasePathFromContext(ctx)
	if result.Job != nil {
		modal, modalErr := h.orders.InvoiceModal(ctx, user.Token, orderID)
		if modalErr != nil {
			log.Printf("orders: reload invoice modal after enqueue failed: %v", modalErr)
			http.Error(w, "領収書モーダルの再取得に失敗しました。", http.StatusBadGateway)
			return
		}
		pollURL := joinBasePath(basePath, "/invoices/jobs/"+url.PathEscape(result.Job.ID))
		payload := orderstpl.InvoiceModalJobPayload(modal, *result.Job, result.Invoice, pollURL)
		w.Header().Set("HX-Trigger", `{"toast":{"message":"領収書の生成を開始しました。","tone":"info"},"refresh:fragment":{"targets":["[data-order-invoice]"]}}`)
		templ.Handler(orderstpl.InvoiceModal(payload)).ServeHTTP(w, r)
		return
	}

	w.Header().Set("HX-Trigger", `{"toast":{"message":"領収書を発行しました。","tone":"success"},"modal:close":true,"refresh:fragment":{"targets":["[data-order-invoice]"]}}`)
	w.WriteHeader(http.StatusNoContent)
}

// InvoiceJobStatus polls the status of an invoice issuance job.
func (h *Handlers) InvoiceJobStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	jobID := strings.TrimSpace(chi.URLParam(r, "jobID"))
	if jobID == "" {
		http.Error(w, "ジョブIDが不正です。", http.StatusBadRequest)
		return
	}

	statusResult, err := h.orders.InvoiceJobStatus(ctx, user.Token, jobID)
	if err != nil {
		switch {
		case errors.Is(err, adminorders.ErrInvoiceJobNotFound):
			http.Error(w, "指定されたジョブが見つかりません。", http.StatusNotFound)
			return
		case errors.Is(err, adminorders.ErrOrderNotFound):
			http.Error(w, "指定された注文が見つかりません。", http.StatusNotFound)
			return
		default:
			log.Printf("orders: invoice job status failed: %v", err)
			http.Error(w, "ジョブの状態取得に失敗しました。", http.StatusBadGateway)
			return
		}
	}

	pollURL := r.URL.Path
	statusData := orderstpl.InvoiceJobStatusFragmentPayload(statusResult, pollURL)
	if statusData.Done {
		w.Header().Set("HX-Trigger", `{"toast":{"message":"領収書を発行しました。","tone":"success"},"modal:close":true,"refresh:fragment":{"targets":["[data-order-invoice]"]}}`)
	}

	templ.Handler(orderstpl.InvoiceJobStatusFragment(statusData)).ServeHTTP(w, r)
}

func (h *Handlers) renderInvoiceModalError(w http.ResponseWriter, r *http.Request, user *custommw.User, orderID string, form orderstpl.InvoiceFormState, message string, fieldErrors map[string]string, status int) {
	ctx := r.Context()
	modal, err := h.orders.InvoiceModal(ctx, user.Token, orderID)
	if err != nil {
		if errors.Is(err, adminorders.ErrOrderNotFound) {
			http.Error(w, "指定された注文が見つかりません。", http.StatusNotFound)
			return
		}
		log.Printf("orders: reload invoice modal after error failed: %v", err)
		http.Error(w, "領収書モーダルの再取得に失敗しました。", http.StatusBadGateway)
		return
	}

	basePath := custommw.BasePathFromContext(ctx)
	csrf := custommw.CSRFTokenFromContext(ctx)
	data := orderstpl.InvoiceModalPayload(basePath, modal, csrf, form, strings.TrimSpace(message), fieldErrors)

	if status > 0 {
		w.WriteHeader(status)
	} else {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}
	templ.Handler(orderstpl.InvoiceModal(data)).ServeHTTP(w, r)
}

func (h *Handlers) renderRefundModalError(w http.ResponseWriter, r *http.Request, user *custommw.User, orderID string, form orderstpl.RefundFormState, message string, fieldErrors map[string]string, status int) {
	ctx := r.Context()
	modal, err := h.orders.RefundModal(ctx, user.Token, orderID)
	if err != nil {
		if errors.Is(err, adminorders.ErrOrderNotFound) {
			http.Error(w, "指定された注文が見つかりません。", http.StatusNotFound)
			return
		}
		log.Printf("orders: reload refund modal after error failed: %v", err)
		http.Error(w, "返金モーダルの再取得に失敗しました。", http.StatusBadGateway)
		return
	}

	basePath := custommw.BasePathFromContext(ctx)
	csrf := custommw.CSRFTokenFromContext(ctx)
	data := orderstpl.RefundModalPayload(basePath, modal, csrf, form, message, fieldErrors)

	if status > 0 {
		w.WriteHeader(status)
	} else {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}
	templ.Handler(orderstpl.RefundModal(data)).ServeHTTP(w, r)
}

func mergeFieldErrors(base map[string]string, extra map[string]string) map[string]string {
	result := map[string]string{}
	for key, value := range base {
		result[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	for key, value := range extra {
		k := strings.TrimSpace(key)
		v := strings.TrimSpace(value)
		if v == "" {
			continue
		}
		result[k] = v
	}
	return result
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

func parseCheckbox(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
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
	case "total", "number", "updated_at", "status":
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

func joinBasePath(basePath, suffix string) string {
	base := strings.TrimSpace(basePath)
	if base == "" {
		base = "/admin"
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	base = strings.TrimRight(base, "/")
	path := strings.TrimSpace(suffix)
	if path == "" {
		return base
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}
