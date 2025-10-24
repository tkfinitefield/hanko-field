package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/services"
)

type stubOrderService struct {
	createFn     func(context.Context, services.CreateOrderFromCartCommand) (services.Order, error)
	listFn       func(context.Context, services.OrderListFilter) (domain.CursorPage[services.Order], error)
	getFn        func(context.Context, string, services.OrderReadOptions) (services.Order, error)
	transitionFn func(context.Context, services.OrderStatusTransitionCommand) (services.Order, error)
	cancelFn     func(context.Context, services.CancelOrderCommand) (services.Order, error)
	appendFn     func(context.Context, services.AppendProductionEventCommand) (services.OrderProductionEvent, error)
	invoiceFn    func(context.Context, services.RequestInvoiceCommand) (services.Order, error)
	reorderFn    func(context.Context, services.CloneForReorderCommand) (services.Order, error)
}

func (s *stubOrderService) CreateFromCart(ctx context.Context, cmd services.CreateOrderFromCartCommand) (services.Order, error) {
	if s.createFn != nil {
		return s.createFn(ctx, cmd)
	}
	return services.Order{}, errors.New("not implemented")
}

func (s *stubOrderService) ListOrders(ctx context.Context, filter services.OrderListFilter) (domain.CursorPage[services.Order], error) {
	if s.listFn != nil {
		return s.listFn(ctx, filter)
	}
	return domain.CursorPage[services.Order]{}, nil
}

func (s *stubOrderService) GetOrder(ctx context.Context, orderID string, opts services.OrderReadOptions) (services.Order, error) {
	if s.getFn != nil {
		return s.getFn(ctx, orderID, opts)
	}
	return services.Order{}, errors.New("not implemented")
}

func (s *stubOrderService) TransitionStatus(ctx context.Context, cmd services.OrderStatusTransitionCommand) (services.Order, error) {
	if s.transitionFn != nil {
		return s.transitionFn(ctx, cmd)
	}
	return services.Order{}, errors.New("not implemented")
}

func (s *stubOrderService) Cancel(ctx context.Context, cmd services.CancelOrderCommand) (services.Order, error) {
	if s.cancelFn != nil {
		return s.cancelFn(ctx, cmd)
	}
	return services.Order{}, errors.New("not implemented")
}

func (s *stubOrderService) AppendProductionEvent(ctx context.Context, cmd services.AppendProductionEventCommand) (services.OrderProductionEvent, error) {
	if s.appendFn != nil {
		return s.appendFn(ctx, cmd)
	}
	return services.OrderProductionEvent{}, errors.New("not implemented")
}

func (s *stubOrderService) RequestInvoice(ctx context.Context, cmd services.RequestInvoiceCommand) (services.Order, error) {
	if s.invoiceFn != nil {
		return s.invoiceFn(ctx, cmd)
	}
	return services.Order{}, errors.New("not implemented")
}

func (s *stubOrderService) CloneForReorder(ctx context.Context, cmd services.CloneForReorderCommand) (services.Order, error) {
	if s.reorderFn != nil {
		return s.reorderFn(ctx, cmd)
	}
	return services.Order{}, errors.New("not implemented")
}

func TestOrderHandlersListOrdersSuccess(t *testing.T) {
	now := time.Date(2024, 3, 15, 9, 30, 0, 0, time.UTC)
	fromExpected := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	toExpected := time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)

	var capturedFilter services.OrderListFilter
	service := &stubOrderService{
		listFn: func(ctx context.Context, filter services.OrderListFilter) (domain.CursorPage[services.Order], error) {
			capturedFilter = filter
			return domain.CursorPage[services.Order]{
				Items: []services.Order{
					{
						ID:          "ord_123",
						OrderNumber: "HF-2024-000123",
						UserID:      "user-1",
						Status:      domain.OrderStatusPaid,
						Currency:    "jpy",
						Totals: services.OrderTotals{
							Subtotal: 1000,
							Tax:      100,
							Shipping: 200,
							Total:    1300,
						},
						CreatedAt: now,
					},
				},
				NextPageToken: "tok-next",
			}, nil
		},
	}

	handler := NewOrderHandlers(nil, service)
	router := chi.NewRouter()
	router.Route("/orders", handler.Routes)

	req := httptest.NewRequest(http.MethodGet, "/orders?status=paid,status=shipped&page_size=10&page_token=tok123&created_after=2024-03-01T00:00:00Z&created_before=2024-04-01T00:00:00Z", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	if capturedFilter.UserID != "user-1" {
		t.Fatalf("expected filter user user-1, got %s", capturedFilter.UserID)
	}
	if capturedFilter.Pagination.PageSize != 10 {
		t.Fatalf("expected page size 10, got %d", capturedFilter.Pagination.PageSize)
	}
	if capturedFilter.Pagination.PageToken != "tok123" {
		t.Fatalf("expected page token tok123, got %s", capturedFilter.Pagination.PageToken)
	}
	if capturedFilter.DateRange.From == nil || !capturedFilter.DateRange.From.Equal(fromExpected) {
		t.Fatalf("expected date range from %s, got %#v", fromExpected.Format(time.RFC3339Nano), capturedFilter.DateRange.From)
	}
	if capturedFilter.DateRange.To == nil || !capturedFilter.DateRange.To.Equal(toExpected) {
		t.Fatalf("expected date range to %s, got %#v", toExpected.Format(time.RFC3339Nano), capturedFilter.DateRange.To)
	}
	if len(capturedFilter.Status) != 2 {
		t.Fatalf("expected 2 status filters, got %d", len(capturedFilter.Status))
	}

	var resp orderListResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 order, got %d", len(resp.Items))
	}
	order := resp.Items[0]
	if order.ID != "ord_123" || order.OrderNumber != "HF-2024-000123" {
		t.Fatalf("unexpected order summary: %#v", order)
	}
	if order.Currency != "JPY" {
		t.Fatalf("expected currency uppercased, got %s", order.Currency)
	}
	if order.Total != 1300 {
		t.Fatalf("expected total 1300, got %d", order.Total)
	}
	if resp.NextPageToken != "tok-next" {
		t.Fatalf("expected next page token tok-next, got %s", resp.NextPageToken)
	}

	if capturedFilter.DateRange.To != nil && !capturedFilter.DateRange.To.After(*capturedFilter.DateRange.From) {
		t.Fatalf("expected range to be valid")
	}
}

func TestOrderHandlersListOrdersInvalidPageSize(t *testing.T) {
	handler := NewOrderHandlers(nil, &stubOrderService{})
	router := chi.NewRouter()
	router.Route("/orders", handler.Routes)

	req := httptest.NewRequest(http.MethodGet, "/orders?page_size=abc", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestOrderHandlersListOrdersInvalidDate(t *testing.T) {
	handler := NewOrderHandlers(nil, &stubOrderService{})
	router := chi.NewRouter()
	router.Route("/orders", handler.Routes)

	req := httptest.NewRequest(http.MethodGet, "/orders?created_after=not-a-date", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestOrderHandlersListOrdersUnauthenticated(t *testing.T) {
	handler := NewOrderHandlers(nil, &stubOrderService{})
	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	rr := httptest.NewRecorder()
	handler.listOrders(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestOrderHandlersListOrdersServiceUnavailable(t *testing.T) {
	handler := NewOrderHandlers(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	rr := httptest.NewRecorder()

	handler.listOrders(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rr.Code)
	}
}

func TestOrderHandlersGetOrderSuccess(t *testing.T) {
	now := time.Date(2024, 5, 12, 10, 0, 0, 0, time.UTC)
	placedAt := now.Add(-2 * time.Hour)
	paidAt := now.Add(-90 * time.Minute)
	shippedAt := now.Add(24 * time.Hour)
	deliveredAt := now.Add(72 * time.Hour)
	completedAt := now.Add(96 * time.Hour)
	canceledAt := now.Add(120 * time.Hour)
	capturedAt := now.Add(-30 * time.Minute)
	lastEventAt := now.Add(-15 * time.Minute)
	requestedAt := now.Add(-3 * time.Hour)
	shipDate := now.Add(24 * time.Hour)
	deliveryDate := now.Add(72 * time.Hour)
	cancelReason := "customer request"
	cartRef := "cart-123"
	queueRef := "queue-a"
	station := "station-7"
	operator := "op-55"
	createdBy := "system"
	updatedBy := "admin"

	service := &stubOrderService{
		getFn: func(ctx context.Context, orderID string, opts services.OrderReadOptions) (services.Order, error) {
			if orderID != "ord_123" {
				t.Fatalf("unexpected order id %s", orderID)
			}
			if !opts.IncludePayments || !opts.IncludeShipments {
				t.Fatalf("expected handler to request payments and shipments")
			}
			return services.Order{
				ID:          "ord_123",
				OrderNumber: "HF-2024-000123",
				UserID:      "user-1",
				CartRef:     &cartRef,
				Status:      domain.OrderStatusShipped,
				Currency:    "jpy",
				Totals: services.OrderTotals{
					Subtotal: 1000,
					Discount: 50,
					Shipping: 200,
					Tax:      100,
					Fees:     0,
					Total:    1250,
				},
				Promotion: &services.CartPromotion{
					Code:           "SPRING",
					DiscountAmount: 50,
					Applied:        true,
				},
				Items: []services.OrderLineItem{
					{
						ProductRef: "prod-1",
						SKU:        "SKU-1",
						Name:       "Personal Seal",
						Options: map[string]any{
							"color": "red",
						},
						Quantity:  1,
						UnitPrice: 1250,
						Total:     1250,
						Metadata: map[string]any{
							"gift_wrap": true,
						},
					},
				},
				ShippingAddress: &services.Address{
					ID:         "addr-ship",
					Recipient:  "Hanako",
					Line1:      "1-2-3 Marunouchi",
					City:       "Tokyo",
					PostalCode: "100-0001",
					Country:    "JP",
					CreatedAt:  now,
					UpdatedAt:  now,
				},
				BillingAddress: &services.Address{
					ID:         "addr-bill",
					Recipient:  "Hanako",
					Line1:      "1-2-3 Marunouchi",
					City:       "Tokyo",
					PostalCode: "100-0001",
					Country:    "JP",
					CreatedAt:  now,
					UpdatedAt:  now,
				},
				Contact: &services.OrderContact{
					Email: "hanako@example.com",
					Phone: "+81-3-1234-5678",
				},
				Fulfillment: services.OrderFulfillment{
					RequestedAt:           &requestedAt,
					EstimatedShipDate:     &shipDate,
					EstimatedDeliveryDate: &deliveryDate,
				},
				Production: services.OrderProduction{
					QueueRef:        &queueRef,
					AssignedStation: &station,
					OperatorRef:     &operator,
					LastEventType:   "engraving",
					LastEventAt:     &lastEventAt,
					OnHold:          false,
				},
				Notes: map[string]any{
					"note": "gift message",
				},
				Flags: services.OrderFlags{
					Gift: true,
				},
				Audit: services.OrderAudit{
					CreatedBy: &createdBy,
					UpdatedBy: &updatedBy,
				},
				Metadata: map[string]any{
					"channel": "app",
				},
				CreatedAt:    now,
				UpdatedAt:    now,
				PlacedAt:     &placedAt,
				PaidAt:       &paidAt,
				ShippedAt:    &shippedAt,
				DeliveredAt:  &deliveredAt,
				CompletedAt:  &completedAt,
				CanceledAt:   &canceledAt,
				CancelReason: &cancelReason,
				Payments: []services.Payment{
					{
						ID:         "pay_1",
						OrderID:    "ord_123",
						Provider:   "stripe",
						Status:     "succeeded",
						Amount:     1250,
						Currency:   "jpy",
						Captured:   true,
						CapturedAt: &capturedAt,
						CreatedAt:  now,
						UpdatedAt:  now,
					},
				},
				Shipments: []services.Shipment{
					{
						ID:           "shp_1",
						OrderID:      "ord_123",
						Carrier:      "yamato",
						TrackingCode: "TRK123",
						Status:       "in_transit",
						Events: []services.ShipmentEvent{
							{
								Status:     "picked_up",
								OccurredAt: now,
								Details: map[string]any{
									"location": "Tokyo DC",
								},
							},
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
			}, nil
		},
	}

	handler := NewOrderHandlers(nil, service)
	router := chi.NewRouter()
	router.Route("/orders", handler.Routes)

	req := httptest.NewRequest(http.MethodGet, "/orders/ord_123", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp orderResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	payload := resp.Order
	if payload.ID != "ord_123" {
		t.Fatalf("expected order id ord_123, got %s", payload.ID)
	}
	if payload.UserID != "user-1" {
		t.Fatalf("expected user id user-1, got %s", payload.UserID)
	}
	if payload.Currency != "JPY" {
		t.Fatalf("expected currency uppercase, got %s", payload.Currency)
	}
	if payload.Totals.Total != 1250 || payload.Totals.Discount != 50 {
		t.Fatalf("unexpected totals %#v", payload.Totals)
	}
	if payload.Promotion == nil || payload.Promotion.Code != "SPRING" {
		t.Fatalf("expected promotion payload, got %#v", payload.Promotion)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(payload.Items))
	}
	if payload.Items[0].Metadata["gift_wrap"] != true {
		t.Fatalf("expected item metadata preserved")
	}
	if payload.ShippingAddress == nil || payload.ShippingAddress.ID != "addr-ship" {
		t.Fatalf("expected shipping address, got %#v", payload.ShippingAddress)
	}
	if payload.Contact == nil || payload.Contact.Email != "hanako@example.com" {
		t.Fatalf("expected contact info, got %#v", payload.Contact)
	}
	if payload.Fulfillment == nil || payload.Fulfillment.EstimatedShipDate == "" {
		t.Fatalf("expected fulfillment payload, got %#v", payload.Fulfillment)
	}
	if payload.Production == nil || payload.Production.QueueRef == nil || *payload.Production.QueueRef != "queue-a" {
		t.Fatalf("expected production payload, got %#v", payload.Production)
	}
	if payload.Flags.Gift != true {
		t.Fatalf("expected gift flag true")
	}
	if payload.Audit == nil || payload.Audit.CreatedBy == nil || *payload.Audit.CreatedBy != "system" {
		t.Fatalf("expected audit payload, got %#v", payload.Audit)
	}
	if len(payload.Payments) != 1 || payload.Payments[0].ID != "pay_1" {
		t.Fatalf("expected payment payload, got %#v", payload.Payments)
	}
	if len(payload.Shipments) != 1 || payload.Shipments[0].ID != "shp_1" {
		t.Fatalf("expected shipment payload, got %#v", payload.Shipments)
	}
	if len(payload.Shipments[0].Events) != 1 || payload.Shipments[0].Events[0].Status != "picked_up" {
		t.Fatalf("expected shipment events, got %#v", payload.Shipments[0].Events)
	}
	if payload.CancelReason == nil || *payload.CancelReason != cancelReason {
		t.Fatalf("expected cancel reason %s, got %#v", cancelReason, payload.CancelReason)
	}
	if payload.CartRef != cartRef {
		t.Fatalf("expected cart ref %s, got %s", cartRef, payload.CartRef)
	}
	if payload.PlacedAt == "" || payload.PaidAt == "" || payload.ShippedAt == "" ||
		payload.DeliveredAt == "" || payload.CompletedAt == "" || payload.CanceledAt == "" {
		t.Fatalf("expected lifecycle timestamps to be populated")
	}
}

func TestOrderHandlersGetOrderEnforcesOwnership(t *testing.T) {
	service := &stubOrderService{
		getFn: func(ctx context.Context, orderID string, opts services.OrderReadOptions) (services.Order, error) {
			return services.Order{
				ID:     "ord_456",
				UserID: "other-user",
			}, nil
		},
	}

	handler := NewOrderHandlers(nil, service)
	req := httptest.NewRequest(http.MethodGet, "/orders/ord_456", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	rr := httptest.NewRecorder()

	router := chi.NewRouter()
	router.Route("/orders", handler.Routes)
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestOrderHandlersGetOrderNotFound(t *testing.T) {
	service := &stubOrderService{
		getFn: func(ctx context.Context, orderID string, opts services.OrderReadOptions) (services.Order, error) {
			return services.Order{}, services.ErrOrderNotFound
		},
	}

	handler := NewOrderHandlers(nil, service)
	req := httptest.NewRequest(http.MethodGet, "/orders/ord_missing", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	rr := httptest.NewRecorder()

	router := chi.NewRouter()
	router.Route("/orders", handler.Routes)
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestOrderHandlersGetOrderInvalidRequest(t *testing.T) {
	handler := NewOrderHandlers(nil, &stubOrderService{})
	req := httptest.NewRequest(http.MethodGet, "/orders/", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	rr := httptest.NewRecorder()
	handler.getOrder(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestOrderHandlersGetOrderServiceUnavailable(t *testing.T) {
	handler := NewOrderHandlers(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/orders/ord_1", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	rr := httptest.NewRecorder()
	handler.getOrder(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rr.Code)
	}
}

func TestOrderHandlersCancelSuccess(t *testing.T) {
	now := time.Date(2024, 6, 10, 12, 0, 0, 0, time.UTC)
	reason := "changed mind"

	service := &stubOrderService{
		getFn: func(ctx context.Context, orderID string, opts services.OrderReadOptions) (services.Order, error) {
			if orderID != "ord_123" {
				t.Fatalf("unexpected order id %s", orderID)
			}
			return services.Order{
				ID:       "ord_123",
				UserID:   "user-1",
				Status:   domain.OrderStatusPaid,
				Metadata: map[string]any{"reservationId": "res-123"},
			}, nil
		},
		cancelFn: func(ctx context.Context, cmd services.CancelOrderCommand) (services.Order, error) {
			if cmd.OrderID != "ord_123" {
				t.Fatalf("unexpected cancel order id %s", cmd.OrderID)
			}
			if cmd.ActorID != "user-1" {
				t.Fatalf("expected actor user-1 got %s", cmd.ActorID)
			}
			if cmd.ReservationID != "res-123" {
				t.Fatalf("expected reservation res-123 got %s", cmd.ReservationID)
			}
			if cmd.Reason != reason {
				t.Fatalf("expected reason %s got %s", reason, cmd.Reason)
			}
			if cmd.ExpectedStatus == nil || *cmd.ExpectedStatus != services.OrderStatus(domain.OrderStatusPaid) {
				t.Fatalf("expected status pointer paid, got %#v", cmd.ExpectedStatus)
			}
			if cmd.Metadata == nil || cmd.Metadata["channel"] != "app" {
				t.Fatalf("expected metadata channel app, got %#v", cmd.Metadata)
			}
			return services.Order{
				ID:           "ord_123",
				UserID:       "user-1",
				Status:       domain.OrderStatusCanceled,
				Metadata:     map[string]any{"reservationId": "res-123"},
				CancelReason: &reason,
				CanceledAt:   &now,
			}, nil
		},
	}

	handler := NewOrderHandlers(nil, service)
	router := chi.NewRouter()
	router.Route("/orders", handler.Routes)

	body := `{"reason":"changed mind","metadata":{"channel":"app"}}`
	req := httptest.NewRequest(http.MethodPost, "/orders/ord_123:cancel", bytes.NewBufferString(body))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp orderResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	payload := resp.Order
	if payload.Status != string(domain.OrderStatusCanceled) {
		t.Fatalf("expected status canceled, got %s", payload.Status)
	}
	if payload.CancelReason == nil || *payload.CancelReason != reason {
		t.Fatalf("expected cancel reason %s got %#v", reason, payload.CancelReason)
	}
	if payload.Metadata["reservationId"] != "res-123" {
		t.Fatalf("expected reservation metadata, got %#v", payload.Metadata)
	}
}

func TestOrderHandlersCancelRequiresOwnership(t *testing.T) {
	service := &stubOrderService{
		getFn: func(ctx context.Context, orderID string, opts services.OrderReadOptions) (services.Order, error) {
			return services.Order{
				ID:     orderID,
				UserID: "other-user",
				Status: domain.OrderStatusPendingPayment,
			}, nil
		},
		cancelFn: func(ctx context.Context, cmd services.CancelOrderCommand) (services.Order, error) {
			t.Fatalf("cancel should not be called")
			return services.Order{}, nil
		},
	}

	handler := NewOrderHandlers(nil, service)
	router := chi.NewRouter()
	router.Route("/orders", handler.Routes)

	req := httptest.NewRequest(http.MethodPost, "/orders/ord_987:cancel", bytes.NewBufferString(`{"reason":"change"}`))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestOrderHandlersCancelRejectsStatus(t *testing.T) {
	var cancelCalled bool
	service := &stubOrderService{
		getFn: func(ctx context.Context, orderID string, opts services.OrderReadOptions) (services.Order, error) {
			return services.Order{
				ID:     orderID,
				UserID: "user-1",
				Status: domain.OrderStatusInProduction,
			}, nil
		},
		cancelFn: func(ctx context.Context, cmd services.CancelOrderCommand) (services.Order, error) {
			cancelCalled = true
			return services.Order{}, nil
		},
	}

	handler := NewOrderHandlers(nil, service)
	router := chi.NewRouter()
	router.Route("/orders", handler.Routes)

	req := httptest.NewRequest(http.MethodPost, "/orders/ord_555:cancel", bytes.NewBufferString(`{}`))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if cancelCalled {
		t.Fatalf("cancel should not be invoked")
	}
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rr.Code)
	}
}

func TestOrderHandlersCancelInvalidJSON(t *testing.T) {
	var getCalled bool
	service := &stubOrderService{
		getFn: func(ctx context.Context, orderID string, opts services.OrderReadOptions) (services.Order, error) {
			getCalled = true
			return services.Order{}, nil
		},
	}

	handler := NewOrderHandlers(nil, service)
	router := chi.NewRouter()
	router.Route("/orders", handler.Routes)

	req := httptest.NewRequest(http.MethodPost, "/orders/ord_111:cancel", bytes.NewBufferString(`{"reason":`))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if getCalled {
		t.Fatalf("expected to reject before fetching order")
	}
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestOrderHandlersCancelPropagatesServiceError(t *testing.T) {
	service := &stubOrderService{
		getFn: func(ctx context.Context, orderID string, opts services.OrderReadOptions) (services.Order, error) {
			return services.Order{
				ID:     orderID,
				UserID: "user-1",
				Status: domain.OrderStatusPaid,
			}, nil
		},
		cancelFn: func(ctx context.Context, cmd services.CancelOrderCommand) (services.Order, error) {
			return services.Order{}, services.ErrOrderConflict
		},
	}

	handler := NewOrderHandlers(nil, service)
	router := chi.NewRouter()
	router.Route("/orders", handler.Routes)

	req := httptest.NewRequest(http.MethodPost, "/orders/ord_222:cancel", strings.NewReader(`{"reason":"dup"}`))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rr.Code)
	}
}

func TestOrderHandlersRequestInvoiceSuccess(t *testing.T) {
	now := time.Date(2025, 1, 15, 9, 45, 0, 0, time.UTC)
	var captured services.RequestInvoiceCommand

	service := &stubOrderService{
		getFn: func(ctx context.Context, orderID string, opts services.OrderReadOptions) (services.Order, error) {
			return services.Order{
				ID:       orderID,
				UserID:   "user-1",
				Status:   domain.OrderStatusPaid,
				Metadata: map[string]any{},
			}, nil
		},
		invoiceFn: func(ctx context.Context, cmd services.RequestInvoiceCommand) (services.Order, error) {
			captured = cmd
			return services.Order{
				ID:     cmd.OrderID,
				UserID: "user-1",
				Status: domain.OrderStatusPaid,
				Metadata: map[string]any{
					"invoiceRequestedAt": now.Format(time.RFC3339Nano),
				},
				UpdatedAt: now,
			}, nil
		},
	}

	handler := NewOrderHandlers(nil, service)
	router := chi.NewRouter()
	router.Route("/orders", handler.Routes)

	req := httptest.NewRequest(http.MethodPost, "/orders/ord_123:request-invoice", strings.NewReader(`{"notes":" please send ","expected_status":"paid"}`))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", rr.Code)
	}

	if captured.OrderID != "ord_123" {
		t.Fatalf("expected command order id ord_123, got %s", captured.OrderID)
	}
	if captured.ActorID != "user-1" {
		t.Fatalf("expected actor user-1, got %s", captured.ActorID)
	}
	if captured.Notes != "please send" {
		t.Fatalf("expected trimmed notes, got %q", captured.Notes)
	}
	if captured.ExpectedStatus == nil || *captured.ExpectedStatus != services.OrderStatus(domain.OrderStatusPaid) {
		t.Fatalf("expected expected status paid, got %#v", captured.ExpectedStatus)
	}

	var resp invoiceRequestResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "queued" {
		t.Fatalf("expected status queued, got %s", resp.Status)
	}
	if resp.Duplicate {
		t.Fatalf("expected duplicate false")
	}
	if resp.RequestedAt != now.Format(time.RFC3339Nano) {
		t.Fatalf("expected requested_at %s, got %s", now.Format(time.RFC3339Nano), resp.RequestedAt)
	}
	if len(resp.DeliveryChannels) != len(invoiceDeliveryChannels) {
		t.Fatalf("expected delivery channels length %d, got %d", len(invoiceDeliveryChannels), len(resp.DeliveryChannels))
	}
	want := map[string]struct{}{"email": {}, "dashboard": {}}
	for _, channel := range resp.DeliveryChannels {
		if _, ok := want[channel]; !ok {
			t.Fatalf("unexpected delivery channel %s", channel)
		}
	}
}

func TestOrderHandlersRequestInvoiceDuplicateSkipsService(t *testing.T) {
	now := time.Date(2025, 2, 20, 11, 0, 0, 0, time.UTC)
	service := &stubOrderService{
		getFn: func(ctx context.Context, orderID string, opts services.OrderReadOptions) (services.Order, error) {
			return services.Order{
				ID:     orderID,
				UserID: "user-1",
				Status: domain.OrderStatusPaid,
				Metadata: map[string]any{
					"invoiceRequestedAt": now.Format(time.RFC3339Nano),
				},
			}, nil
		},
		invoiceFn: func(ctx context.Context, cmd services.RequestInvoiceCommand) (services.Order, error) {
			t.Fatalf("request invoice should not be called on duplicate")
			return services.Order{}, nil
		},
	}

	handler := NewOrderHandlers(nil, service)
	router := chi.NewRouter()
	router.Route("/orders", handler.Routes)

	req := httptest.NewRequest(http.MethodPost, "/orders/ord_777:request-invoice", strings.NewReader(`{}`))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", rr.Code)
	}

	var resp invoiceRequestResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Duplicate {
		t.Fatalf("expected duplicate true")
	}
	if resp.Status != "duplicate" {
		t.Fatalf("expected status duplicate, got %s", resp.Status)
	}
	if resp.RequestedAt != now.Format(time.RFC3339Nano) {
		t.Fatalf("expected requested_at %s, got %s", now.Format(time.RFC3339Nano), resp.RequestedAt)
	}
}

func TestOrderHandlersRequestInvoiceRequiresPaidStatus(t *testing.T) {
	service := &stubOrderService{
		getFn: func(ctx context.Context, orderID string, opts services.OrderReadOptions) (services.Order, error) {
			return services.Order{
				ID:     orderID,
				UserID: "user-1",
				Status: domain.OrderStatusPendingPayment,
			}, nil
		},
		invoiceFn: func(ctx context.Context, cmd services.RequestInvoiceCommand) (services.Order, error) {
			t.Fatalf("request invoice should not be invoked")
			return services.Order{}, nil
		},
	}

	handler := NewOrderHandlers(nil, service)
	router := chi.NewRouter()
	router.Route("/orders", handler.Routes)

	req := httptest.NewRequest(http.MethodPost, "/orders/ord_999:request-invoice", strings.NewReader(`{"notes":"now"}`))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rr.Code)
	}
}

func TestOrderHandlersRequestInvoiceRequiresOwnership(t *testing.T) {
	service := &stubOrderService{
		getFn: func(ctx context.Context, orderID string, opts services.OrderReadOptions) (services.Order, error) {
			return services.Order{
				ID:     orderID,
				UserID: "other-user",
				Status: domain.OrderStatusPaid,
			}, nil
		},
		invoiceFn: func(ctx context.Context, cmd services.RequestInvoiceCommand) (services.Order, error) {
			t.Fatalf("request invoice should not be invoked")
			return services.Order{}, nil
		},
	}

	handler := NewOrderHandlers(nil, service)
	router := chi.NewRouter()
	router.Route("/orders", handler.Routes)

	req := httptest.NewRequest(http.MethodPost, "/orders/ord_111:request-invoice", strings.NewReader(`{"notes":"now"}`))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestOrderHandlersRequestInvoiceInvalidJSON(t *testing.T) {
	service := &stubOrderService{
		getFn: func(ctx context.Context, orderID string, opts services.OrderReadOptions) (services.Order, error) {
			return services.Order{
				ID:     orderID,
				UserID: "user-1",
				Status: domain.OrderStatusPaid,
			}, nil
		},
		invoiceFn: func(ctx context.Context, cmd services.RequestInvoiceCommand) (services.Order, error) {
			t.Fatalf("request invoice should not be invoked")
			return services.Order{}, nil
		},
	}

	handler := NewOrderHandlers(nil, service)
	router := chi.NewRouter()
	router.Route("/orders", handler.Routes)

	req := httptest.NewRequest(http.MethodPost, "/orders/ord_222:request-invoice", strings.NewReader(`{"notes":`))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestOrderHandlersRequestInvoiceExpectedStatusMismatch(t *testing.T) {
	service := &stubOrderService{
		getFn: func(ctx context.Context, orderID string, opts services.OrderReadOptions) (services.Order, error) {
			return services.Order{
				ID:     orderID,
				UserID: "user-1",
				Status: domain.OrderStatusPaid,
			}, nil
		},
		invoiceFn: func(ctx context.Context, cmd services.RequestInvoiceCommand) (services.Order, error) {
			t.Fatalf("request invoice should not be invoked")
			return services.Order{}, nil
		},
	}

	handler := NewOrderHandlers(nil, service)
	router := chi.NewRouter()
	router.Route("/orders", handler.Routes)

	req := httptest.NewRequest(http.MethodPost, "/orders/ord_333:request-invoice", strings.NewReader(`{"expected_status":"shipped"}`))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rr.Code)
	}
}
