package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/platform/httpx"
	"github.com/hanko-field/api/internal/services"
)

const (
	defaultOrderPageSize    = 20
	maxOrderPageSize        = 100
	maxOrderCancelBodySize  = 4 * 1024
	maxOrderInvoiceBodySize = 2 * 1024
	maxOrderReorderBodySize = 4 * 1024
)

var userCancellableStatuses = map[domain.OrderStatus]struct{}{
	domain.OrderStatusPendingPayment: {},
	domain.OrderStatusPaid:           {},
}

var validOrderStatuses = map[domain.OrderStatus]struct{}{
	domain.OrderStatusDraft:          {},
	domain.OrderStatusPendingPayment: {},
	domain.OrderStatusPaid:           {},
	domain.OrderStatusInProduction:   {},
	domain.OrderStatusReadyToShip:    {},
	domain.OrderStatusShipped:        {},
	domain.OrderStatusDelivered:      {},
	domain.OrderStatusCompleted:      {},
	domain.OrderStatusCanceled:       {},
}

var invoiceRequestableStatuses = map[domain.OrderStatus]struct{}{
	domain.OrderStatusPaid:         {},
	domain.OrderStatusInProduction: {},
	domain.OrderStatusReadyToShip:  {},
	domain.OrderStatusShipped:      {},
	domain.OrderStatusDelivered:    {},
	domain.OrderStatusCompleted:    {},
}

var reorderableStatuses = map[domain.OrderStatus]struct{}{
	domain.OrderStatusDelivered: {},
	domain.OrderStatusCompleted: {},
}

var invoiceDeliveryChannels = []string{"email", "dashboard"}

var productionNoteTagPattern = regexp.MustCompile(`(?is)<[^>]+>`)

type cancelOrderRequest struct {
	Reason         string         `json:"reason"`
	Metadata       map[string]any `json:"metadata"`
	ExpectedStatus string         `json:"expected_status"`
}

type requestInvoiceRequest struct {
	Notes          string `json:"notes"`
	ExpectedStatus string `json:"expected_status"`
}

type invoiceRequestResponse struct {
	Status           string   `json:"status"`
	DeliveryChannels []string `json:"delivery_channels"`
	RequestedAt      string   `json:"requested_at,omitempty"`
	Duplicate        bool     `json:"duplicate,omitempty"`
}

type reorderOrderRequest struct {
	Metadata map[string]any `json:"metadata"`
}

// OrderHandlers exposes order read-only endpoints for authenticated users.
type OrderHandlers struct {
	authn  *auth.Authenticator
	orders services.OrderService
}

// NewOrderHandlers constructs a new OrderHandlers instance.
func NewOrderHandlers(authn *auth.Authenticator, orders services.OrderService) *OrderHandlers {
	return &OrderHandlers{
		authn:  authn,
		orders: orders,
	}
}

// Routes registers the /orders endpoints.
func (h *OrderHandlers) Routes(r chi.Router) {
	if r == nil {
		return
	}
	if h.authn != nil {
		r.Use(h.authn.RequireFirebaseAuth())
	}
	r.Get("/", h.listOrders)
	r.Get("/{orderID}/production-events", h.listOrderProductionEvents)
	r.Get("/{orderID}/payments", h.listOrderPayments)
	r.Get("/{orderID}/shipments", h.listOrderShipments)
	r.Get("/{orderID}/shipments/{shipmentID}", h.getOrderShipment)
	r.Get("/{orderID}", h.getOrder)
	r.Post("/{orderID}:reorder", h.reorderOrder)
	r.Post("/{orderID}:request-invoice", h.requestInvoice)
	r.Post("/{orderID}:cancel", h.cancelOrder)
}

func (h *OrderHandlers) listOrders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.orders == nil {
		httpx.WriteError(ctx, w, httpx.NewError("order_service_unavailable", "order service unavailable", http.StatusServiceUnavailable))
		return
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}

	query := r.URL.Query()
	statusFilters := parseFilterValues(query["status"])

	var dateRange domain.RangeQuery[time.Time]
	var hasDateRange bool
	if raw := strings.TrimSpace(query.Get("created_after")); raw != "" {
		ts, err := parseTimeParam(raw)
		if err != nil {
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "created_after must be a valid RFC3339 timestamp", http.StatusBadRequest))
			return
		}
		dateRange.From = &ts
		hasDateRange = true
	}
	if raw := strings.TrimSpace(query.Get("created_before")); raw != "" {
		ts, err := parseTimeParam(raw)
		if err != nil {
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "created_before must be a valid RFC3339 timestamp", http.StatusBadRequest))
			return
		}
		dateRange.To = &ts
		hasDateRange = true
	}

	pageSize := defaultOrderPageSize
	if sizeRaw := strings.TrimSpace(query.Get("page_size")); sizeRaw != "" {
		size, err := strconv.Atoi(sizeRaw)
		if err != nil {
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "page_size must be an integer", http.StatusBadRequest))
			return
		}
		switch {
		case size <= 0:
			pageSize = defaultOrderPageSize
		case size > maxOrderPageSize:
			pageSize = maxOrderPageSize
		default:
			pageSize = size
		}
	}

	filter := services.OrderListFilter{
		UserID: strings.TrimSpace(identity.UID),
		Status: statusFilters,
		Pagination: services.Pagination{
			PageSize:  pageSize,
			PageToken: strings.TrimSpace(query.Get("page_token")),
		},
	}
	if hasDateRange {
		filter.DateRange = dateRange
	}

	page, err := h.orders.ListOrders(ctx, filter)
	if err != nil {
		writeOrderError(ctx, w, err)
		return
	}

	items := make([]orderSummaryPayload, 0, len(page.Items))
	for _, order := range page.Items {
		items = append(items, buildOrderSummary(order))
	}

	response := orderListResponse{
		Items:         items,
		NextPageToken: strings.TrimSpace(page.NextPageToken),
	}
	writeJSONResponse(w, http.StatusOK, response)
}

func (h *OrderHandlers) getOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.orders == nil {
		httpx.WriteError(ctx, w, httpx.NewError("order_service_unavailable", "order service unavailable", http.StatusServiceUnavailable))
		return
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}

	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "order id is required", http.StatusBadRequest))
		return
	}

	order, err := h.orders.GetOrder(ctx, orderID, services.OrderReadOptions{
		IncludePayments:  true,
		IncludeShipments: true,
	})
	if err != nil {
		writeOrderError(ctx, w, err)
		return
	}

	if !strings.EqualFold(strings.TrimSpace(order.UserID), strings.TrimSpace(identity.UID)) {
		httpx.WriteError(ctx, w, httpx.NewError("order_not_found", "order not found", http.StatusNotFound))
		return
	}

	payload := orderResponse{
		Order: buildOrderPayload(order),
	}
	writeJSONResponse(w, http.StatusOK, payload)
}

func (h *OrderHandlers) listOrderShipments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.orders == nil {
		httpx.WriteError(ctx, w, httpx.NewError("order_service_unavailable", "order service unavailable", http.StatusServiceUnavailable))
		return
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}

	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "order id is required", http.StatusBadRequest))
		return
	}

	order, err := h.orders.GetOrder(ctx, orderID, services.OrderReadOptions{IncludeShipments: true})
	if err != nil {
		writeOrderError(ctx, w, err)
		return
	}

	if !strings.EqualFold(strings.TrimSpace(order.UserID), strings.TrimSpace(identity.UID)) {
		httpx.WriteError(ctx, w, httpx.NewError("order_not_found", "order not found", http.StatusNotFound))
		return
	}

	shipments := buildOrderShipmentSummaries(order.Shipments)
	writeJSONResponse(w, http.StatusOK, shipments)
}

func (h *OrderHandlers) listOrderProductionEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.orders == nil {
		httpx.WriteError(ctx, w, httpx.NewError("order_service_unavailable", "order service unavailable", http.StatusServiceUnavailable))
		return
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}

	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "order id is required", http.StatusBadRequest))
		return
	}

	includeNotes := true
	if raw := strings.TrimSpace(r.URL.Query().Get("includeNotes")); raw != "" {
		flag, err := strconv.ParseBool(raw)
		if err != nil {
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "includeNotes must be a boolean", http.StatusBadRequest))
			return
		}
		includeNotes = flag
	}

	order, err := h.orders.GetOrder(ctx, orderID, services.OrderReadOptions{IncludeProductionEvents: true})
	if err != nil {
		writeOrderError(ctx, w, err)
		return
	}

	if !strings.EqualFold(strings.TrimSpace(order.UserID), strings.TrimSpace(identity.UID)) {
		httpx.WriteError(ctx, w, httpx.NewError("order_not_found", "order not found", http.StatusNotFound))
		return
	}

	timeline := buildOrderProductionTimeline(order.ProductionEvents, includeNotes)
	response := orderProductionTimelineResponse{
		Events: timeline,
	}
	writeJSONResponse(w, http.StatusOK, response)
}

func (h *OrderHandlers) getOrderShipment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.orders == nil {
		httpx.WriteError(ctx, w, httpx.NewError("order_service_unavailable", "order service unavailable", http.StatusServiceUnavailable))
		return
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}

	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "order id is required", http.StatusBadRequest))
		return
	}

	shipmentID := strings.TrimSpace(chi.URLParam(r, "shipmentID"))
	if shipmentID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "shipment id is required", http.StatusBadRequest))
		return
	}

	order, err := h.orders.GetOrder(ctx, orderID, services.OrderReadOptions{IncludeShipments: true})
	if err != nil {
		writeOrderError(ctx, w, err)
		return
	}

	if !strings.EqualFold(strings.TrimSpace(order.UserID), strings.TrimSpace(identity.UID)) {
		httpx.WriteError(ctx, w, httpx.NewError("order_not_found", "order not found", http.StatusNotFound))
		return
	}

	var target *services.Shipment
	for i := range order.Shipments {
		shipment := &order.Shipments[i]
		if strings.EqualFold(strings.TrimSpace(shipment.ID), shipmentID) {
			target = shipment
			break
		}
	}

	if target == nil {
		httpx.WriteError(ctx, w, httpx.NewError("shipment_not_found", "shipment not found", http.StatusNotFound))
		return
	}

	payload := buildOrderShipmentDetail(*target)
	writeJSONResponse(w, http.StatusOK, payload)
}

func (h *OrderHandlers) listOrderPayments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.orders == nil {
		httpx.WriteError(ctx, w, httpx.NewError("order_service_unavailable", "order service unavailable", http.StatusServiceUnavailable))
		return
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}

	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "order id is required", http.StatusBadRequest))
		return
	}

	order, err := h.orders.GetOrder(ctx, orderID, services.OrderReadOptions{
		IncludePayments: true,
	})
	if err != nil {
		writeOrderError(ctx, w, err)
		return
	}

	if !strings.EqualFold(strings.TrimSpace(order.UserID), strings.TrimSpace(identity.UID)) {
		httpx.WriteError(ctx, w, httpx.NewError("order_not_found", "order not found", http.StatusNotFound))
		return
	}

	response := orderPaymentHistoryResponse{
		Payments: buildOrderPaymentHistory(order.Payments),
	}
	writeJSONResponse(w, http.StatusOK, response)
}

func (h *OrderHandlers) reorderOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.orders == nil {
		httpx.WriteError(ctx, w, httpx.NewError("order_service_unavailable", "order service unavailable", http.StatusServiceUnavailable))
		return
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}

	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "order id is required", http.StatusBadRequest))
		return
	}

	body, err := readLimitedBody(r, maxOrderReorderBodySize)
	switch {
	case err == nil:
	case errors.Is(err, errEmptyBody):
		body = nil
	case errors.Is(err, errBodyTooLarge):
		httpx.WriteError(ctx, w, httpx.NewError("payload_too_large", "request body exceeds allowed size", http.StatusRequestEntityTooLarge))
		return
	default:
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		return
	}

	var payload reorderOrderRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &payload); err != nil {
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "invalid JSON body", http.StatusBadRequest))
			return
		}
	}

	order, err := h.orders.GetOrder(ctx, orderID, services.OrderReadOptions{})
	if err != nil {
		writeOrderError(ctx, w, err)
		return
	}

	userID := strings.TrimSpace(identity.UID)
	if !strings.EqualFold(strings.TrimSpace(order.UserID), userID) {
		httpx.WriteError(ctx, w, httpx.NewError("order_not_found", "order not found", http.StatusNotFound))
		return
	}

	statusKey := domain.OrderStatus(strings.TrimSpace(strings.ToLower(string(order.Status))))
	if _, allowed := reorderableStatuses[statusKey]; !allowed {
		httpx.WriteError(ctx, w, httpx.NewError("order_invalid_state", "reorder only allowed from delivered/completed orders", http.StatusConflict))
		return
	}

	cmd := services.CloneForReorderCommand{
		OrderID:  orderID,
		ActorID:  userID,
		Metadata: cloneMap(payload.Metadata),
	}

	result, err := h.orders.CloneForReorder(ctx, cmd)
	if err != nil {
		writeOrderError(ctx, w, err)
		return
	}

	response := orderResponse{
		Order: buildOrderPayload(result),
	}
	writeJSONResponse(w, http.StatusCreated, response)
}

func (h *OrderHandlers) requestInvoice(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.orders == nil {
		httpx.WriteError(ctx, w, httpx.NewError("order_service_unavailable", "order service unavailable", http.StatusServiceUnavailable))
		return
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}

	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "order id is required", http.StatusBadRequest))
		return
	}

	order, err := h.orders.GetOrder(ctx, orderID, services.OrderReadOptions{})
	if err != nil {
		writeOrderError(ctx, w, err)
		return
	}

	if !strings.EqualFold(strings.TrimSpace(order.UserID), strings.TrimSpace(identity.UID)) {
		httpx.WriteError(ctx, w, httpx.NewError("order_not_found", "order not found", http.StatusNotFound))
		return
	}

	statusKey := domain.OrderStatus(strings.TrimSpace(strings.ToLower(string(order.Status))))
	if _, ok := invoiceRequestableStatuses[statusKey]; !ok {
		httpx.WriteError(ctx, w, httpx.NewError("order_invalid_state", "invoice request only allowed after payment", http.StatusConflict))
		return
	}

	existingRequestedAt := extractInvoiceRequestedAt(order.Metadata)

	body, err := readLimitedBody(r, maxOrderInvoiceBodySize)
	switch {
	case err == nil:
	case errors.Is(err, errEmptyBody):
		body = nil
	case errors.Is(err, errBodyTooLarge):
		httpx.WriteError(ctx, w, httpx.NewError("payload_too_large", "request body exceeds allowed size", http.StatusRequestEntityTooLarge))
		return
	default:
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		return
	}

	var payload requestInvoiceRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &payload); err != nil {
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "invalid JSON body", http.StatusBadRequest))
			return
		}
	}

	var expectedStatus *services.OrderStatus
	if raw := strings.TrimSpace(payload.ExpectedStatus); raw != "" {
		parsed, ok := parseOrderStatus(raw)
		if !ok {
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "expected_status must be a valid order status", http.StatusBadRequest))
			return
		}
		current, ok := parseOrderStatus(string(order.Status))
		if !ok {
			current = services.OrderStatus(statusKey)
		}
		if parsed != current {
			httpx.WriteError(ctx, w, httpx.NewError("order_conflict", fmt.Sprintf("expected status %q but was %q", raw, order.Status), http.StatusConflict))
			return
		}
		expectedStatus = &parsed
	}

	note := strings.TrimSpace(payload.Notes)
	delivery := append([]string(nil), invoiceDeliveryChannels...)
	response := invoiceRequestResponse{
		DeliveryChannels: delivery,
	}

	if existingRequestedAt != "" {
		response.Status = "duplicate"
		response.Duplicate = true
		response.RequestedAt = existingRequestedAt
		writeJSONResponse(w, http.StatusAccepted, response)
		return
	}

	cmd := services.RequestInvoiceCommand{
		OrderID:        orderID,
		ActorID:        strings.TrimSpace(identity.UID),
		Notes:          note,
		ExpectedStatus: expectedStatus,
	}

	result, err := h.orders.RequestInvoice(ctx, cmd)
	if err != nil {
		if errors.Is(err, services.ErrOrderInvoiceAlreadyRequested) {
			requestedAt := extractInvoiceRequestedAt(result.Metadata)
			if requestedAt == "" {
				requestedAt = formatTime(result.UpdatedAt)
			}
			response.Status = "duplicate"
			response.Duplicate = true
			response.RequestedAt = requestedAt
			writeJSONResponse(w, http.StatusAccepted, response)
			return
		}
		writeOrderError(ctx, w, err)
		return
	}

	requestedAt := extractInvoiceRequestedAt(result.Metadata)
	if requestedAt == "" {
		requestedAt = formatTime(result.UpdatedAt)
	}

	response.Status = "queued"
	response.RequestedAt = requestedAt
	writeJSONResponse(w, http.StatusAccepted, response)
}

func (h *OrderHandlers) cancelOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.orders == nil {
		httpx.WriteError(ctx, w, httpx.NewError("order_service_unavailable", "order service unavailable", http.StatusServiceUnavailable))
		return
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}

	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "order id is required", http.StatusBadRequest))
		return
	}

	var req cancelOrderRequest
	body, err := readLimitedBody(r, maxOrderCancelBodySize)
	if err != nil {
		switch {
		case errors.Is(err, errBodyTooLarge):
			httpx.WriteError(ctx, w, httpx.NewError("payload_too_large", "request body exceeds allowed size", http.StatusRequestEntityTooLarge))
		default:
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		}
		return
	}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "invalid JSON body", http.StatusBadRequest))
			return
		}
	}

	order, err := h.orders.GetOrder(ctx, orderID, services.OrderReadOptions{})
	if err != nil {
		writeOrderError(ctx, w, err)
		return
	}

	if !strings.EqualFold(strings.TrimSpace(order.UserID), strings.TrimSpace(identity.UID)) {
		httpx.WriteError(ctx, w, httpx.NewError("order_not_found", "order not found", http.StatusNotFound))
		return
	}

	if !isUserCancellableStatus(order.Status) {
		httpx.WriteError(ctx, w, httpx.NewError("order_invalid_state", "order cannot be canceled in its current status", http.StatusConflict))
		return
	}

	reason := strings.TrimSpace(req.Reason)
	reservationID := extractReservationID(order.Metadata)

	expectedStatus := services.OrderStatus(order.Status)
	if raw := strings.TrimSpace(req.ExpectedStatus); raw != "" {
		parsed, ok := parseOrderStatus(raw)
		if !ok {
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "expected_status must be a valid order status", http.StatusBadRequest))
			return
		}
		expectedStatus = parsed
	}

	cmd := services.CancelOrderCommand{
		OrderID:        orderID,
		ActorID:        strings.TrimSpace(identity.UID),
		Reason:         reason,
		ReservationID:  reservationID,
		ExpectedStatus: &expectedStatus,
		Metadata:       cloneMap(req.Metadata),
	}

	canceled, err := h.orders.Cancel(ctx, cmd)
	if err != nil {
		writeOrderError(ctx, w, err)
		return
	}

	payload := orderResponse{
		Order: buildOrderPayload(canceled),
	}
	writeJSONResponse(w, http.StatusOK, payload)
}

type orderListResponse struct {
	Items         []orderSummaryPayload `json:"items"`
	NextPageToken string                `json:"next_page_token,omitempty"`
}

type orderSummaryPayload struct {
	ID          string `json:"id"`
	OrderNumber string `json:"order_number"`
	Status      string `json:"status"`
	Currency    string `json:"currency"`
	Total       int64  `json:"total"`
	CreatedAt   string `json:"created_at"`
}

type orderResponse struct {
	Order orderPayload `json:"order"`
}

type orderPayload struct {
	ID              string                   `json:"id"`
	OrderNumber     string                   `json:"order_number"`
	UserID          string                   `json:"user_id"`
	CartRef         string                   `json:"cart_ref,omitempty"`
	Status          string                   `json:"status"`
	Currency        string                   `json:"currency"`
	Totals          orderTotalsPayload       `json:"totals"`
	Promotion       *orderPromotionPayload   `json:"promotion,omitempty"`
	Items           []orderItemPayload       `json:"items"`
	ShippingAddress *addressPayload          `json:"shipping_address,omitempty"`
	BillingAddress  *addressPayload          `json:"billing_address,omitempty"`
	Contact         *orderContactPayload     `json:"contact,omitempty"`
	Fulfillment     *orderFulfillmentPayload `json:"fulfillment,omitempty"`
	Production      *orderProductionPayload  `json:"production,omitempty"`
	Notes           map[string]any           `json:"notes,omitempty"`
	Flags           orderFlagsPayload        `json:"flags,omitempty"`
	Audit           *orderAuditPayload       `json:"audit,omitempty"`
	Metadata        map[string]any           `json:"metadata,omitempty"`
	CreatedAt       string                   `json:"created_at"`
	UpdatedAt       string                   `json:"updated_at,omitempty"`
	PlacedAt        string                   `json:"placed_at,omitempty"`
	PaidAt          string                   `json:"paid_at,omitempty"`
	ShippedAt       string                   `json:"shipped_at,omitempty"`
	DeliveredAt     string                   `json:"delivered_at,omitempty"`
	CompletedAt     string                   `json:"completed_at,omitempty"`
	CanceledAt      string                   `json:"canceled_at,omitempty"`
	CancelReason    *string                  `json:"cancel_reason,omitempty"`
	Payments        []orderPaymentPayload    `json:"payments,omitempty"`
	Shipments       []orderShipmentPayload   `json:"shipments,omitempty"`
}

type orderTotalsPayload struct {
	Subtotal int64 `json:"subtotal"`
	Discount int64 `json:"discount"`
	Shipping int64 `json:"shipping"`
	Tax      int64 `json:"tax"`
	Fees     int64 `json:"fees"`
	Total    int64 `json:"total"`
}

type orderPromotionPayload struct {
	Code           string `json:"code"`
	DiscountAmount int64  `json:"discount_amount"`
	Applied        bool   `json:"applied"`
}

type orderItemPayload struct {
	ProductRef     string         `json:"product_ref"`
	SKU            string         `json:"sku"`
	Name           string         `json:"name,omitempty"`
	Quantity       int            `json:"quantity"`
	UnitPrice      int64          `json:"unit_price"`
	Total          int64          `json:"total"`
	Options        map[string]any `json:"options,omitempty"`
	DesignRef      *string        `json:"design_ref,omitempty"`
	DesignSnapshot map[string]any `json:"design_snapshot,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type orderContactPayload struct {
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`
}

type orderFulfillmentPayload struct {
	RequestedAt           string `json:"requested_at,omitempty"`
	EstimatedShipDate     string `json:"estimated_ship_date,omitempty"`
	EstimatedDeliveryDate string `json:"estimated_delivery_date,omitempty"`
}

type orderProductionPayload struct {
	QueueRef        *string `json:"queue_ref,omitempty"`
	AssignedStation *string `json:"assigned_station,omitempty"`
	OperatorRef     *string `json:"operator_ref,omitempty"`
	LastEventType   string  `json:"last_event_type,omitempty"`
	LastEventAt     string  `json:"last_event_at,omitempty"`
	OnHold          bool    `json:"on_hold,omitempty"`
}

type orderFlagsPayload struct {
	ManualReview bool `json:"manual_review,omitempty"`
	Gift         bool `json:"gift,omitempty"`
}

type orderAuditPayload struct {
	CreatedBy *string `json:"created_by,omitempty"`
	UpdatedBy *string `json:"updated_by,omitempty"`
}

type orderPaymentPayload struct {
	ID         string `json:"id"`
	Provider   string `json:"provider"`
	Status     string `json:"status"`
	Amount     int64  `json:"amount"`
	Currency   string `json:"currency"`
	Captured   bool   `json:"captured"`
	CapturedAt string `json:"captured_at,omitempty"`
	RefundedAt string `json:"refunded_at,omitempty"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at,omitempty"`
}

type orderProductionTimelineResponse struct {
	Events []orderProductionTimelineEvent `json:"events"`
}

type orderProductionTimelineEvent struct {
	Timestamp   string   `json:"timestamp"`
	Stage       string   `json:"stage"`
	Operator    string   `json:"operator,omitempty"`
	Notes       string   `json:"notes,omitempty"`
	Attachments []string `json:"attachments,omitempty"`
}

type orderPaymentHistoryResponse struct {
	Payments []orderPaymentHistoryEntry `json:"payments"`
}

type orderPaymentHistoryEntry struct {
	Provider       string `json:"provider"`
	TransactionID  string `json:"transaction_id"`
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
	Status         string `json:"status"`
	CapturedAt     string `json:"captured_at,omitempty"`
	RefundedAmount int64  `json:"refunded_amount"`
}

type orderShipmentPayload struct {
	ID           string                   `json:"id"`
	Carrier      string                   `json:"carrier"`
	TrackingCode string                   `json:"tracking_code,omitempty"`
	Status       string                   `json:"status"`
	Events       []orderShipmentEventData `json:"events,omitempty"`
	CreatedAt    string                   `json:"created_at"`
	UpdatedAt    string                   `json:"updated_at,omitempty"`
}

type orderShipmentSummaryPayload struct {
	ID           string                  `json:"id"`
	Carrier      string                  `json:"carrier"`
	TrackingCode string                  `json:"tracking_code,omitempty"`
	Status       string                  `json:"status"`
	CreatedAt    string                  `json:"created_at"`
	UpdatedAt    string                  `json:"updated_at,omitempty"`
	LatestEvent  *orderShipmentEventData `json:"latest_event,omitempty"`
}

type orderShipmentEventData struct {
	Status     string         `json:"status"`
	OccurredAt string         `json:"occurred_at"`
	Details    map[string]any `json:"details,omitempty"`
}

func buildOrderSummary(order services.Order) orderSummaryPayload {
	return orderSummaryPayload{
		ID:          strings.TrimSpace(order.ID),
		OrderNumber: strings.TrimSpace(order.OrderNumber),
		Status:      strings.TrimSpace(string(order.Status)),
		Currency:    strings.ToUpper(strings.TrimSpace(order.Currency)),
		Total:       order.Totals.Total,
		CreatedAt:   formatTime(order.CreatedAt),
	}
}

func buildOrderPayload(order services.Order) orderPayload {
	payload := orderPayload{
		ID:          strings.TrimSpace(order.ID),
		OrderNumber: strings.TrimSpace(order.OrderNumber),
		UserID:      strings.TrimSpace(order.UserID),
		Status:      strings.TrimSpace(string(order.Status)),
		Currency:    strings.ToUpper(strings.TrimSpace(order.Currency)),
		Totals: orderTotalsPayload{
			Subtotal: order.Totals.Subtotal,
			Discount: order.Totals.Discount,
			Shipping: order.Totals.Shipping,
			Tax:      order.Totals.Tax,
			Fees:     order.Totals.Fees,
			Total:    order.Totals.Total,
		},
		Items:        make([]orderItemPayload, 0, len(order.Items)),
		Notes:        cloneMap(order.Notes),
		Metadata:     cloneMap(order.Metadata),
		CreatedAt:    formatTime(order.CreatedAt),
		UpdatedAt:    formatTime(order.UpdatedAt),
		PlacedAt:     formatTime(pointerTime(order.PlacedAt)),
		PaidAt:       formatTime(pointerTime(order.PaidAt)),
		ShippedAt:    formatTime(pointerTime(order.ShippedAt)),
		DeliveredAt:  formatTime(pointerTime(order.DeliveredAt)),
		CompletedAt:  formatTime(pointerTime(order.CompletedAt)),
		CanceledAt:   formatTime(pointerTime(order.CanceledAt)),
		CancelReason: cloneStringPointer(order.CancelReason),
		Payments:     buildOrderPaymentPayloads(order.Payments),
		Shipments:    buildOrderShipmentPayloads(order.Shipments),
	}

	if order.CartRef != nil {
		payload.CartRef = strings.TrimSpace(*order.CartRef)
	}

	if order.Promotion != nil {
		payload.Promotion = &orderPromotionPayload{
			Code:           strings.ToUpper(strings.TrimSpace(order.Promotion.Code)),
			DiscountAmount: order.Promotion.DiscountAmount,
			Applied:        order.Promotion.Applied,
		}
	}

	for _, item := range order.Items {
		entry := orderItemPayload{
			ProductRef:     strings.TrimSpace(item.ProductRef),
			SKU:            strings.TrimSpace(item.SKU),
			Name:           strings.TrimSpace(item.Name),
			Quantity:       item.Quantity,
			UnitPrice:      item.UnitPrice,
			Total:          item.Total,
			Options:        cloneMap(item.Options),
			DesignRef:      cloneStringPointer(item.DesignRef),
			DesignSnapshot: cloneMap(item.DesignSnapshot),
			Metadata:       cloneMap(item.Metadata),
		}
		payload.Items = append(payload.Items, entry)
	}

	if len(order.Items) == 0 {
		payload.Items = []orderItemPayload{}
	}

	if order.ShippingAddress != nil {
		addr := buildAddressPayload(*order.ShippingAddress)
		payload.ShippingAddress = &addr
	}
	if order.BillingAddress != nil {
		addr := buildAddressPayload(*order.BillingAddress)
		payload.BillingAddress = &addr
	}

	if order.Contact != nil {
		payload.Contact = &orderContactPayload{
			Email: strings.TrimSpace(order.Contact.Email),
			Phone: strings.TrimSpace(order.Contact.Phone),
		}
	}

	if order.Fulfillment.RequestedAt != nil ||
		order.Fulfillment.EstimatedShipDate != nil ||
		order.Fulfillment.EstimatedDeliveryDate != nil {
		payload.Fulfillment = &orderFulfillmentPayload{
			RequestedAt:           formatTime(pointerTime(order.Fulfillment.RequestedAt)),
			EstimatedShipDate:     formatTime(pointerTime(order.Fulfillment.EstimatedShipDate)),
			EstimatedDeliveryDate: formatTime(pointerTime(order.Fulfillment.EstimatedDeliveryDate)),
		}
	}

	if order.Production.QueueRef != nil ||
		order.Production.AssignedStation != nil ||
		order.Production.OperatorRef != nil ||
		order.Production.LastEventType != "" ||
		order.Production.LastEventAt != nil ||
		order.Production.OnHold {
		payload.Production = &orderProductionPayload{
			QueueRef:        order.Production.QueueRef,
			AssignedStation: order.Production.AssignedStation,
			OperatorRef:     order.Production.OperatorRef,
			LastEventType:   strings.TrimSpace(order.Production.LastEventType),
			LastEventAt:     formatTime(pointerTime(order.Production.LastEventAt)),
			OnHold:          order.Production.OnHold,
		}
	}

	if order.Flags.ManualReview || order.Flags.Gift {
		payload.Flags = orderFlagsPayload{
			ManualReview: order.Flags.ManualReview,
			Gift:         order.Flags.Gift,
		}
	}

	if order.Audit.CreatedBy != nil || order.Audit.UpdatedBy != nil {
		payload.Audit = &orderAuditPayload{
			CreatedBy: cloneStringPointer(order.Audit.CreatedBy),
			UpdatedBy: cloneStringPointer(order.Audit.UpdatedBy),
		}
	}

	return payload
}

func buildOrderPaymentPayloads(payments []services.Payment) []orderPaymentPayload {
	if len(payments) == 0 {
		return nil
	}
	result := make([]orderPaymentPayload, 0, len(payments))
	for _, payment := range payments {
		result = append(result, orderPaymentPayload{
			ID:         strings.TrimSpace(payment.ID),
			Provider:   strings.TrimSpace(payment.Provider),
			Status:     strings.TrimSpace(payment.Status),
			Amount:     payment.Amount,
			Currency:   strings.ToUpper(strings.TrimSpace(payment.Currency)),
			Captured:   payment.Captured,
			CapturedAt: formatTime(pointerTime(payment.CapturedAt)),
			RefundedAt: formatTime(pointerTime(payment.RefundedAt)),
			CreatedAt:  formatTime(payment.CreatedAt),
			UpdatedAt:  formatTime(payment.UpdatedAt),
		})
	}
	return result
}

func buildOrderPaymentHistory(payments []services.Payment) []orderPaymentHistoryEntry {
	if len(payments) == 0 {
		return []orderPaymentHistoryEntry{}
	}

	sorted := append([]services.Payment(nil), payments...)
	slices.SortFunc(sorted, func(a, b services.Payment) int {
		switch {
		case a.CreatedAt.Before(b.CreatedAt):
			return -1
		case a.CreatedAt.After(b.CreatedAt):
			return 1
		default:
			return strings.Compare(strings.TrimSpace(a.ID), strings.TrimSpace(b.ID))
		}
	})

	result := make([]orderPaymentHistoryEntry, 0, len(sorted))
	for _, payment := range sorted {
		entry := orderPaymentHistoryEntry{
			Provider:       strings.TrimSpace(payment.Provider),
			TransactionID:  strings.TrimSpace(payment.IntentID),
			Amount:         payment.Amount,
			Currency:       strings.ToUpper(strings.TrimSpace(payment.Currency)),
			Status:         strings.TrimSpace(payment.Status),
			CapturedAt:     formatTime(pointerTime(payment.CapturedAt)),
			RefundedAmount: extractRefundedAmount(payment),
		}
		result = append(result, entry)
	}
	return result
}

func buildOrderShipmentPayloads(shipments []services.Shipment) []orderShipmentPayload {
	if len(shipments) == 0 {
		return nil
	}
	result := make([]orderShipmentPayload, 0, len(shipments))
	for _, shipment := range shipments {
		result = append(result, buildOrderShipmentDetail(shipment))
	}
	return result
}

func buildOrderShipmentSummaries(shipments []services.Shipment) []orderShipmentSummaryPayload {
	if len(shipments) == 0 {
		return []orderShipmentSummaryPayload{}
	}
	result := make([]orderShipmentSummaryPayload, 0, len(shipments))
	for _, shipment := range shipments {
		detail := buildOrderShipmentDetail(shipment)
		summary := orderShipmentSummaryPayload{
			ID:           detail.ID,
			Carrier:      detail.Carrier,
			TrackingCode: detail.TrackingCode,
			Status:       detail.Status,
			CreatedAt:    detail.CreatedAt,
			UpdatedAt:    detail.UpdatedAt,
		}
		if len(detail.Events) > 0 {
			latest := detail.Events[0]
			summary.LatestEvent = &latest
		}
		result = append(result, summary)
	}
	return result
}

func buildOrderShipmentDetail(shipment services.Shipment) orderShipmentPayload {
	payload := orderShipmentPayload{
		ID:           strings.TrimSpace(shipment.ID),
		Carrier:      strings.TrimSpace(shipment.Carrier),
		TrackingCode: strings.TrimSpace(shipment.TrackingCode),
		Status:       strings.TrimSpace(shipment.Status),
		CreatedAt:    formatTime(shipment.CreatedAt),
		UpdatedAt:    formatTime(shipment.UpdatedAt),
	}
	if events := buildShipmentEvents(shipment.Events); len(events) > 0 {
		payload.Events = events
	}
	return payload
}

func buildOrderProductionTimeline(events []services.OrderProductionEvent, includeNotes bool) []orderProductionTimelineEvent {
	if len(events) == 0 {
		return []orderProductionTimelineEvent{}
	}

	sorted := append([]services.OrderProductionEvent(nil), events...)
	slices.SortFunc(sorted, func(a, b services.OrderProductionEvent) int {
		switch {
		case a.CreatedAt.Before(b.CreatedAt):
			return -1
		case a.CreatedAt.After(b.CreatedAt):
			return 1
		default:
			return strings.Compare(strings.TrimSpace(a.ID), strings.TrimSpace(b.ID))
		}
	})

	result := make([]orderProductionTimelineEvent, 0, len(sorted))
	for _, event := range sorted {
		entry := orderProductionTimelineEvent{
			Timestamp: formatTime(event.CreatedAt),
			Stage:     strings.TrimSpace(event.Type),
		}
		if ref := event.OperatorRef; ref != nil {
			if trimmed := strings.TrimSpace(*ref); trimmed != "" {
				entry.Operator = trimmed
			}
		}
		if includeNotes {
			if note := sanitizeProductionNote(event.Note); note != "" {
				entry.Notes = note
			}
		}
		if attachments := buildProductionEventAttachments(event); len(attachments) > 0 {
			entry.Attachments = attachments
		}
		result = append(result, entry)
	}
	return result
}

func buildShipmentEvents(events []services.ShipmentEvent) []orderShipmentEventData {
	if len(events) == 0 {
		return nil
	}
	sorted := append([]services.ShipmentEvent(nil), events...)
	slices.SortFunc(sorted, func(a, b services.ShipmentEvent) int {
		switch {
		case a.OccurredAt.After(b.OccurredAt):
			return -1
		case a.OccurredAt.Before(b.OccurredAt):
			return 1
		default:
			return strings.Compare(strings.TrimSpace(a.Status), strings.TrimSpace(b.Status))
		}
	})
	result := make([]orderShipmentEventData, 0, len(sorted))
	for _, event := range sorted {
		entry := orderShipmentEventData{
			Status:     strings.TrimSpace(event.Status),
			OccurredAt: formatTime(event.OccurredAt),
		}
		if details := sanitizeShipmentEventDetails(event.Details); len(details) > 0 {
			entry.Details = details
		}
		result = append(result, entry)
	}
	return result
}

func sanitizeShipmentEventDetails(details map[string]any) map[string]any {
	if len(details) == 0 {
		return nil
	}
	sanitized := make(map[string]any, len(details))
	for key, value := range details {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		lowerKey := strings.ToLower(trimmedKey)
		if strings.Contains(lowerKey, "internal") || strings.Contains(lowerKey, "private") {
			continue
		}
		sanitized[trimmedKey] = value
	}
	if len(sanitized) == 0 {
		return nil
	}
	return sanitized
}

func sanitizeProductionNote(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	withoutTags := productionNoteTagPattern.ReplaceAllString(trimmed, "")
	sanitized := strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, withoutTags)

	result := strings.TrimSpace(sanitized)
	return result
}

func buildProductionEventAttachments(event services.OrderProductionEvent) []string {
	if event.PhotoURL == nil {
		return nil
	}
	url := strings.TrimSpace(*event.PhotoURL)
	if url == "" {
		return nil
	}
	return []string{url}
}

func extractRefundedAmount(payment services.Payment) int64 {
	if len(payment.Raw) == 0 {
		return 0
	}

	if amount := intFromAny(payment.Raw["refundedAmount"]); amount > 0 {
		return amount
	}
	if amount := intFromAny(payment.Raw["refunded_amount"]); amount > 0 {
		return amount
	}

	if latest, ok := payment.Raw["latest_charge"].(map[string]any); ok {
		if amount := intFromAny(latest["amount_refunded"]); amount > 0 {
			return amount
		}
	}

	if charges, ok := payment.Raw["charges"].(map[string]any); ok {
		if amount := intFromAny(charges["amount_refunded"]); amount > 0 {
			return amount
		}
		if data, ok := charges["data"].([]any); ok {
			for _, entry := range data {
				charge, ok := entry.(map[string]any)
				if !ok {
					continue
				}
				if amount := intFromAny(charge["amount_refunded"]); amount > 0 {
					return amount
				}
			}
		}
	}

	return 0
}

func intFromAny(value any) int64 {
	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i
		}
		if f, err := v.Float64(); err == nil {
			return int64(f)
		}
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0
		}
		if i, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
			return i
		}
	}
	return 0
}

func extractInvoiceRequestedAt(metadata map[string]any) string {
	if len(metadata) == 0 {
		return ""
	}
	if value, ok := metadata["invoiceRequestedAt"]; ok {
		if ts := stringify(value); ts != "" {
			return ts
		}
	}
	return ""
}

func writeOrderError(ctx context.Context, w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	switch {
	case errors.Is(err, services.ErrOrderInvalidInput):
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
	case errors.Is(err, services.ErrOrderNotFound):
		httpx.WriteError(ctx, w, httpx.NewError("order_not_found", "order not found", http.StatusNotFound))
	case errors.Is(err, services.ErrOrderConflict):
		httpx.WriteError(ctx, w, httpx.NewError("order_conflict", err.Error(), http.StatusConflict))
	case errors.Is(err, services.ErrOrderInvalidState):
		httpx.WriteError(ctx, w, httpx.NewError("order_invalid_state", err.Error(), http.StatusConflict))
	default:
		httpx.WriteError(ctx, w, httpx.NewError("order_error", "failed to process order request", http.StatusInternalServerError))
	}
}

func extractReservationID(metadata map[string]any) string {
	if len(metadata) == 0 {
		return ""
	}
	if value, ok := metadata["reservationId"]; ok {
		if str := stringify(value); str != "" {
			return str
		}
	}
	if value, ok := metadata["reservationID"]; ok {
		if str := stringify(value); str != "" {
			return str
		}
	}
	return ""
}

func stringify(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case []byte:
		return strings.TrimSpace(string(v))
	case time.Time:
		return formatTime(v)
	default:
		return ""
	}
}

func isUserCancellableStatus(status domain.OrderStatus) bool {
	_, ok := userCancellableStatuses[domain.OrderStatus(strings.TrimSpace(string(status)))]
	return ok
}

func parseOrderStatus(raw string) (services.OrderStatus, bool) {
	status := domain.OrderStatus(strings.TrimSpace(strings.ToLower(raw)))
	if _, ok := validOrderStatuses[status]; !ok {
		return "", false
	}
	return services.OrderStatus(status), true
}
