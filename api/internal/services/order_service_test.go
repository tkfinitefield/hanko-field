package services

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

type stubOrderRepo struct {
	insertFn func(context.Context, domain.Order) error
	updateFn func(context.Context, domain.Order) error
	findFn   func(context.Context, string) (domain.Order, error)
	listFn   func(context.Context, repositories.OrderListFilter) (domain.CursorPage[domain.Order], error)
}

func (s *stubOrderRepo) Insert(ctx context.Context, order domain.Order) error {
	if s.insertFn != nil {
		return s.insertFn(ctx, order)
	}
	return nil
}

func (s *stubOrderRepo) Update(ctx context.Context, order domain.Order) error {
	if s.updateFn != nil {
		return s.updateFn(ctx, order)
	}
	return nil
}

func (s *stubOrderRepo) FindByID(ctx context.Context, orderID string) (domain.Order, error) {
	if s.findFn != nil {
		return s.findFn(ctx, orderID)
	}
	return domain.Order{}, errors.New("not implemented")
}

func (s *stubOrderRepo) List(ctx context.Context, filter repositories.OrderListFilter) (domain.CursorPage[domain.Order], error) {
	if s.listFn != nil {
		return s.listFn(ctx, filter)
	}
	return domain.CursorPage[domain.Order]{}, nil
}

type stubProductionRepo struct {
	insertFn func(context.Context, domain.OrderProductionEvent) (domain.OrderProductionEvent, error)
	listFn   func(context.Context, string) ([]domain.OrderProductionEvent, error)
}

func (s *stubProductionRepo) Insert(ctx context.Context, event domain.OrderProductionEvent) (domain.OrderProductionEvent, error) {
	if s.insertFn != nil {
		return s.insertFn(ctx, event)
	}
	return event, nil
}

func (s *stubProductionRepo) List(ctx context.Context, orderID string) ([]domain.OrderProductionEvent, error) {
	if s.listFn != nil {
		return s.listFn(ctx, orderID)
	}
	return nil, nil
}

type stubCounterRepo struct {
	nextFn func(context.Context, string, int64) (int64, error)
}

func (s *stubCounterRepo) Next(ctx context.Context, counterID string, step int64) (int64, error) {
	if s.nextFn != nil {
		return s.nextFn(ctx, counterID, step)
	}
	return 0, nil
}

func (s *stubCounterRepo) Configure(context.Context, string, repositories.CounterConfig) error {
	return nil
}

type stubInventoryService struct {
	commitFn  func(context.Context, InventoryCommitCommand) (InventoryReservation, error)
	releaseFn func(context.Context, InventoryReleaseCommand) (InventoryReservation, error)
}

func (s *stubInventoryService) ReserveStocks(context.Context, InventoryReserveCommand) (InventoryReservation, error) {
	return InventoryReservation{}, errors.New("not implemented")
}

func (s *stubInventoryService) CommitReservation(ctx context.Context, cmd InventoryCommitCommand) (InventoryReservation, error) {
	if s.commitFn != nil {
		return s.commitFn(ctx, cmd)
	}
	return InventoryReservation{}, nil
}

func (s *stubInventoryService) ReleaseReservation(ctx context.Context, cmd InventoryReleaseCommand) (InventoryReservation, error) {
	if s.releaseFn != nil {
		return s.releaseFn(ctx, cmd)
	}
	return InventoryReservation{}, nil
}

func (s *stubInventoryService) ListLowStock(context.Context, InventoryLowStockFilter) (domain.CursorPage[InventorySnapshot], error) {
	return domain.CursorPage[InventorySnapshot]{}, errors.New("not implemented")
}

func (s *stubInventoryService) ConfigureSafetyStock(context.Context, ConfigureSafetyStockCommand) (InventoryStock, error) {
	return InventoryStock{}, errors.New("not implemented")
}

type captureOrderEvents struct {
	events []OrderEvent
}

func (c *captureOrderEvents) PublishOrderEvent(_ context.Context, event OrderEvent) error {
	c.events = append(c.events, event)
	return nil
}

type stubUnitOfWork struct {
	runFn func(context.Context, func(context.Context) error) error
}

func (s *stubUnitOfWork) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	if s.runFn != nil {
		return s.runFn(ctx, fn)
	}
	return fn(ctx)
}

func TestOrderServiceCreateFromCart(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2025, 5, 1, 9, 30, 0, 0, time.UTC)
	inserted := make([]domain.Order, 0, 1)
	var committed InventoryCommitCommand
	events := &captureOrderEvents{}

	orderRepo := &stubOrderRepo{
		insertFn: func(_ context.Context, order domain.Order) error {
			inserted = append(inserted, order)
			return nil
		},
	}

	inventory := &stubInventoryService{
		commitFn: func(_ context.Context, cmd InventoryCommitCommand) (InventoryReservation, error) {
			committed = cmd
			return InventoryReservation{ID: cmd.ReservationID}, nil
		},
	}

	counters := &stubCounterRepo{
		nextFn: func(_ context.Context, counterID string, step int64) (int64, error) {
			if counterID != "orders" {
				t.Fatalf("unexpected counter id %s", counterID)
			}
			if step != 1 {
				t.Fatalf("unexpected step %d", step)
			}
			return 42, nil
		},
	}

	unit := &stubUnitOfWork{}

	svc, err := NewOrderService(OrderServiceDeps{
		Orders:     orderRepo,
		Counters:   counters,
		Inventory:  inventory,
		UnitOfWork: unit,
		Clock: func() time.Time {
			return now
		},
		IDGenerator: func() string { return "000TEST" },
		Events:      events,
	})
	if err != nil {
		t.Fatalf("new order service: %v", err)
	}

	cart := Cart{
		ID:       "cart-1",
		UserID:   "user-1",
		Currency: "JPY",
		Estimate: &CartEstimate{Subtotal: 1000, Discount: 0, Tax: 80, Shipping: 200, Total: 1280},
		Items: []CartItem{
			{ProductID: "prod-1", SKU: "SKU-1", Quantity: 2, UnitPrice: 500},
		},
	}

	order, err := svc.CreateFromCart(ctx, CreateOrderFromCartCommand{
		Cart:          cart,
		ActorID:       "user-1",
		ReservationID: "res-1",
	})
	if err != nil {
		t.Fatalf("create from cart: %v", err)
	}

	if order.ID != "ord_000TEST" {
		t.Fatalf("unexpected order id %s", order.ID)
	}
	if order.Status != domain.OrderStatusPendingPayment {
		t.Fatalf("expected status pending_payment got %s", order.Status)
	}
	if order.OrderNumber != "HF-2025-000042" {
		t.Fatalf("unexpected order number %s", order.OrderNumber)
	}
	if committed.ReservationID != "res-1" {
		t.Fatalf("expected reservation res-1 got %s", committed.ReservationID)
	}
	if len(inserted) != 1 {
		t.Fatalf("expected 1 inserted order got %d", len(inserted))
	}
	if inserted[0].Totals.Total != 1280 {
		t.Fatalf("expected total 1280 got %d", inserted[0].Totals.Total)
	}
	if v := order.Metadata["reservationId"]; v != "res-1" {
		t.Fatalf("expected metadata reservationId res-1 got %v", v)
	}
	if len(events.events) == 0 {
		t.Fatalf("expected domain event emission")
	}
}

func TestOrderServiceTransitionStatus(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)
	orderRepo := &stubOrderRepo{}
	orderRepo.findFn = func(_ context.Context, id string) (domain.Order, error) {
		return domain.Order{ID: id, Status: domain.OrderStatusPendingPayment, OrderNumber: "HF-2025-000001", Currency: "JPY"}, nil
	}
	var updated domain.Order
	orderRepo.updateFn = func(_ context.Context, order domain.Order) error {
		updated = order
		return nil
	}

	svc, err := NewOrderService(OrderServiceDeps{
		Orders:     orderRepo,
		Counters:   &stubCounterRepo{nextFn: func(context.Context, string, int64) (int64, error) { return 1, nil }},
		UnitOfWork: &stubUnitOfWork{},
		Clock:      func() time.Time { return now },
		IDGenerator: func() string {
			return "NEWID"
		},
	})
	if err != nil {
		t.Fatalf("new order service: %v", err)
	}

	order, err := svc.TransitionStatus(ctx, OrderStatusTransitionCommand{
		OrderID:      "order-1",
		TargetStatus: domain.OrderStatusPaid,
		ActorID:      "staff-1",
	})
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if order.Status != domain.OrderStatusPaid {
		t.Fatalf("expected status paid got %s", order.Status)
	}
	if updated.Status != domain.OrderStatusPaid {
		t.Fatalf("repository update not invoked with paid status")
	}
	if updated.PaidAt == nil {
		t.Fatalf("expected paidAt to be set")
	}

	if _, err := svc.TransitionStatus(ctx, OrderStatusTransitionCommand{
		OrderID:      "order-1",
		TargetStatus: domain.OrderStatusShipped,
		ActorID:      "staff-1",
	}); err == nil {
		t.Fatalf("expected invalid transition error")
	}
}

func TestOrderServiceCancelReleasesReservation(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2025, 7, 1, 8, 0, 0, 0, time.UTC)
	orderRepo := &stubOrderRepo{}
	orderRepo.findFn = func(_ context.Context, id string) (domain.Order, error) {
		return domain.Order{ID: id, Status: domain.OrderStatusPendingPayment, OrderNumber: "HF-2025-000010", Currency: "JPY"}, nil
	}
	var updated domain.Order
	orderRepo.updateFn = func(_ context.Context, order domain.Order) error {
		updated = order
		return nil
	}

	var released InventoryReleaseCommand
	inventory := &stubInventoryService{
		releaseFn: func(_ context.Context, cmd InventoryReleaseCommand) (InventoryReservation, error) {
			released = cmd
			return InventoryReservation{}, nil
		},
	}

	events := &captureOrderEvents{}

	svc, err := NewOrderService(OrderServiceDeps{
		Orders:     orderRepo,
		Counters:   &stubCounterRepo{nextFn: func(context.Context, string, int64) (int64, error) { return 10, nil }},
		Inventory:  inventory,
		UnitOfWork: &stubUnitOfWork{},
		Clock:      func() time.Time { return now },
		Events:     events,
	})
	if err != nil {
		t.Fatalf("new order service: %v", err)
	}

	order, err := svc.Cancel(ctx, CancelOrderCommand{
		OrderID:       "order-1",
		ActorID:       "user-1",
		Reason:        "changed mind",
		ReservationID: "res-77",
	})
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if order.Status != domain.OrderStatusCanceled {
		t.Fatalf("expected canceled status got %s", order.Status)
	}
	if updated.CancelReason == nil || *updated.CancelReason != "changed mind" {
		t.Fatalf("expected cancel reason propagated")
	}
	if released.ReservationID != "res-77" {
		t.Fatalf("expected reservation release res-77 got %s", released.ReservationID)
	}
	if len(events.events) == 0 {
		t.Fatalf("expected cancellation event to be published")
	}
}

func TestOrderServiceAppendProductionEventAdvancesStatus(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2025, 7, 10, 12, 0, 0, 0, time.UTC)
	orderRepo := &stubOrderRepo{}
	orderRepo.findFn = func(_ context.Context, id string) (domain.Order, error) {
		return domain.Order{ID: id, Status: domain.OrderStatusPaid, OrderNumber: "HF-2025-000020", Currency: "JPY"}, nil
	}
	var updated domain.Order
	orderRepo.updateFn = func(_ context.Context, order domain.Order) error {
		updated = order
		return nil
	}

	production := &stubProductionRepo{
		insertFn: func(_ context.Context, event domain.OrderProductionEvent) (domain.OrderProductionEvent, error) {
			return event, nil
		},
	}

	events := &captureOrderEvents{}

	svc, err := NewOrderService(OrderServiceDeps{
		Orders:     orderRepo,
		Counters:   &stubCounterRepo{nextFn: func(context.Context, string, int64) (int64, error) { return 20, nil }},
		Production: production,
		UnitOfWork: &stubUnitOfWork{},
		Clock:      func() time.Time { return now },
		Events:     events,
		IDGenerator: func() string {
			return "PEVID"
		},
	})
	if err != nil {
		t.Fatalf("new order service: %v", err)
	}

	event, err := svc.AppendProductionEvent(ctx, AppendProductionEventCommand{
		OrderID: "order-2",
		ActorID: "operator-1",
		Event:   OrderProductionEvent{Type: "engraving", Station: "CNC-01"},
	})
	if err != nil {
		t.Fatalf("append production: %v", err)
	}
	if event.ID != "ope_PEVID" {
		t.Fatalf("expected prefixed event id got %s", event.ID)
	}
	if updated.Status != domain.OrderStatusInProduction {
		t.Fatalf("expected status in_production got %s", updated.Status)
	}
	if len(events.events) == 0 {
		t.Fatalf("expected production event publication")
	}
}

func TestOrderServiceRequestInvoice(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2025, 3, 10, 8, 15, 0, 0, time.UTC)
	var updated domain.Order
	events := &captureOrderEvents{}

	orderRepo := &stubOrderRepo{
		findFn: func(context.Context, string) (domain.Order, error) {
			return domain.Order{
				ID:     "ord_123",
				UserID: "user-1",
				Status: domain.OrderStatusPaid,
				Metadata: map[string]any{
					"other": "value",
				},
			}, nil
		},
		updateFn: func(_ context.Context, order domain.Order) error {
			updated = order
			return nil
		},
	}

	svc, err := NewOrderService(OrderServiceDeps{
		Orders:     orderRepo,
		Counters:   &stubCounterRepo{},
		UnitOfWork: &stubUnitOfWork{},
		Clock: func() time.Time {
			return now
		},
		Events: events,
	})
	if err != nil {
		t.Fatalf("new order service: %v", err)
	}

	result, err := svc.RequestInvoice(ctx, RequestInvoiceCommand{
		OrderID: "ord_123",
		ActorID: "user-1",
		Notes:   " please send ",
	})
	if err != nil {
		t.Fatalf("request invoice: %v", err)
	}

	if updated.ID != "ord_123" {
		t.Fatalf("expected updated order ord_123, got %s", updated.ID)
	}
	requestedAt, ok := updated.Metadata["invoiceRequestedAt"]
	if !ok || stringify(requestedAt) == "" {
		t.Fatalf("expected invoiceRequestedAt set, got %#v", updated.Metadata)
	}
	if updated.Metadata["invoiceRequestedBy"] != "user-1" {
		t.Fatalf("expected invoiceRequestedBy user-1, got %#v", updated.Metadata["invoiceRequestedBy"])
	}
	if updated.Metadata["invoiceNotes"] != "please send" {
		t.Fatalf("expected invoiceNotes trimmed, got %#v", updated.Metadata["invoiceNotes"])
	}
	if result.Metadata["invoiceRequestedAt"] == "" {
		t.Fatalf("expected result metadata populated")
	}
	if result.UpdatedAt != now {
		t.Fatalf("expected updated time %s, got %s", now, result.UpdatedAt)
	}

	if len(events.events) != 1 {
		t.Fatalf("expected one event, got %d", len(events.events))
	}
	event := events.events[0]
	if event.Type != orderEventInvoiceRequested {
		t.Fatalf("expected event type %s, got %s", orderEventInvoiceRequested, event.Type)
	}
	if event.OrderID != "ord_123" {
		t.Fatalf("expected event order ord_123, got %s", event.OrderID)
	}
	if event.ActorID != "user-1" {
		t.Fatalf("expected actor user-1, got %s", event.ActorID)
	}
	if event.OccurredAt != now {
		t.Fatalf("expected occurred at %s, got %s", now, event.OccurredAt)
	}
}

func TestOrderServiceRequestInvoiceDuplicate(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2025, 4, 1, 7, 0, 0, 0, time.UTC)
	events := &captureOrderEvents{}

	orderRepo := &stubOrderRepo{
		findFn: func(context.Context, string) (domain.Order, error) {
			return domain.Order{
				ID:     "ord_456",
				UserID: "user-1",
				Status: domain.OrderStatusPaid,
				Metadata: map[string]any{
					"invoiceRequestedAt": now.Add(-time.Hour).Format(time.RFC3339Nano),
				},
			}, nil
		},
		updateFn: func(_ context.Context, order domain.Order) error {
			t.Fatalf("update should not be called on duplicate, got %#v", order)
			return nil
		},
	}

	svc, err := NewOrderService(OrderServiceDeps{
		Orders:     orderRepo,
		Counters:   &stubCounterRepo{},
		UnitOfWork: &stubUnitOfWork{},
		Clock: func() time.Time {
			return now
		},
		Events: events,
	})
	if err != nil {
		t.Fatalf("new order service: %v", err)
	}

	result, err := svc.RequestInvoice(ctx, RequestInvoiceCommand{
		OrderID: "ord_456",
		ActorID: "user-1",
	})
	if !errors.Is(err, ErrOrderInvoiceAlreadyRequested) {
		t.Fatalf("expected ErrOrderInvoiceAlreadyRequested, got %v", err)
	}

	if len(events.events) != 0 {
		t.Fatalf("expected no events published, got %d", len(events.events))
	}
	if stringify(result.Metadata["invoiceRequestedAt"]) == "" {
		t.Fatalf("expected metadata retained")
	}
}

func TestOrderServiceCloneForReorder(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2025, 8, 1, 14, 0, 0, 0, time.UTC)
	orderRepo := &stubOrderRepo{}
	orderRepo.findFn = func(_ context.Context, id string) (domain.Order, error) {
		return domain.Order{
			ID:          id,
			Status:      domain.OrderStatusDelivered,
			UserID:      "user-9",
			Currency:    "JPY",
			OrderNumber: "HF-2025-000030",
			Items: []domain.OrderLineItem{{
				ProductRef: "prod-9",
				SKU:        "SKU-99",
				Quantity:   1,
				UnitPrice:  1500,
				Total:      1500,
			}},
		}, nil
	}
	var inserted domain.Order
	orderRepo.insertFn = func(_ context.Context, order domain.Order) error {
		inserted = order
		return nil
	}

	svc, err := NewOrderService(OrderServiceDeps{
		Orders:      orderRepo,
		Counters:    &stubCounterRepo{nextFn: func(context.Context, string, int64) (int64, error) { return 31, nil }},
		UnitOfWork:  &stubUnitOfWork{},
		Clock:       func() time.Time { return now },
		IDGenerator: func() string { return "REORDER" },
	})
	if err != nil {
		t.Fatalf("new order service: %v", err)
	}

	reorder, err := svc.CloneForReorder(ctx, CloneForReorderCommand{
		OrderID: "order-original",
		ActorID: "user-9",
	})
	if err != nil {
		t.Fatalf("clone for reorder: %v", err)
	}
	if reorder.Status != domain.OrderStatusDraft {
		t.Fatalf("expected draft status got %s", reorder.Status)
	}
	if inserted.Metadata["reorderOf"] != "order-original" {
		t.Fatalf("expected metadata reorderOf to be order-original")
	}
	if len(reorder.Items) != 1 || reorder.Items[0].SKU != "SKU-99" {
		t.Fatalf("expected item cloned from source")
	}
}
