package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
	defaultOrderPageSize   = 20
	maxOrderPageSize       = 100
	maxOrderCancelBodySize = 4 * 1024
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

type cancelOrderRequest struct {
	Reason         string         `json:"reason"`
	Metadata       map[string]any `json:"metadata"`
	ExpectedStatus string         `json:"expected_status"`
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
	r.Get("/{orderID}", h.getOrder)
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

type orderShipmentPayload struct {
	ID           string                   `json:"id"`
	Carrier      string                   `json:"carrier"`
	TrackingCode string                   `json:"tracking_code,omitempty"`
	Status       string                   `json:"status"`
	Events       []orderShipmentEventData `json:"events,omitempty"`
	CreatedAt    string                   `json:"created_at"`
	UpdatedAt    string                   `json:"updated_at,omitempty"`
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

func buildOrderShipmentPayloads(shipments []services.Shipment) []orderShipmentPayload {
	if len(shipments) == 0 {
		return nil
	}
	result := make([]orderShipmentPayload, 0, len(shipments))
	for _, shipment := range shipments {
		payload := orderShipmentPayload{
			ID:           strings.TrimSpace(shipment.ID),
			Carrier:      strings.TrimSpace(shipment.Carrier),
			TrackingCode: strings.TrimSpace(shipment.TrackingCode),
			Status:       strings.TrimSpace(shipment.Status),
			CreatedAt:    formatTime(shipment.CreatedAt),
			UpdatedAt:    formatTime(shipment.UpdatedAt),
		}
		if len(shipment.Events) > 0 {
			events := make([]orderShipmentEventData, 0, len(shipment.Events))
			for _, event := range shipment.Events {
				events = append(events, orderShipmentEventData{
					Status:     strings.TrimSpace(event.Status),
					OccurredAt: formatTime(event.OccurredAt),
					Details:    cloneMap(event.Details),
				})
			}
			payload.Events = events
		}
		result = append(result, payload)
	}
	return result
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
