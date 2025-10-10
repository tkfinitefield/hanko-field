package services

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

const (
	orderEventCreated            = "order.created"
	orderEventStatusChanged      = "order.status.changed"
	orderEventProductionAppended = "order.production.event.appended"
	orderEventInvoiceRequested   = "order.invoice.requested"

	orderIDPrefix           = "ord_"
	productionEventIDPrefix = "ope_"
)

var (
	// ErrOrderInvalidInput signals the caller provided invalid data.
	ErrOrderInvalidInput = errors.New("order: invalid input")
	// ErrOrderNotFound indicates the order could not be located.
	ErrOrderNotFound = errors.New("order: not found")
	// ErrOrderInvalidState indicates an invalid status transition was attempted.
	ErrOrderInvalidState = errors.New("order: invalid status transition")
	// ErrOrderConflict indicates optimistic concurrency conflicts or duplicates.
	ErrOrderConflict = errors.New("order: conflict")

	errOrderPaymentRepositoryUnavailable    = errors.New("order: payment repository not configured")
	errOrderShipmentRepositoryUnavailable   = errors.New("order: shipment repository not configured")
	errOrderProductionRepositoryUnavailable = errors.New("order: production repository not configured")
)

var orderStateTransitions = map[string][]string{
	string(domain.OrderStatusDraft):          {string(domain.OrderStatusPendingPayment), string(domain.OrderStatusCanceled)},
	string(domain.OrderStatusPendingPayment): {string(domain.OrderStatusPaid), string(domain.OrderStatusCanceled)},
	string(domain.OrderStatusPaid):           {string(domain.OrderStatusInProduction), string(domain.OrderStatusReadyToShip), string(domain.OrderStatusCanceled)},
	string(domain.OrderStatusInProduction):   {string(domain.OrderStatusReadyToShip), string(domain.OrderStatusShipped), string(domain.OrderStatusCanceled)},
	string(domain.OrderStatusReadyToShip):    {string(domain.OrderStatusShipped), string(domain.OrderStatusCanceled)},
	string(domain.OrderStatusShipped):        {string(domain.OrderStatusDelivered)},
	string(domain.OrderStatusDelivered):      {string(domain.OrderStatusCompleted)},
}

var cancellableStatuses = []string{
	string(domain.OrderStatusDraft),
	string(domain.OrderStatusPendingPayment),
	string(domain.OrderStatusPaid),
	string(domain.OrderStatusInProduction),
	string(domain.OrderStatusReadyToShip),
}

var productionEventStatusMapping = map[string]string{
	"queued":     string(domain.OrderStatusInProduction),
	"engraving":  string(domain.OrderStatusInProduction),
	"polishing":  string(domain.OrderStatusInProduction),
	"qc":         string(domain.OrderStatusInProduction),
	"on_hold":    string(domain.OrderStatusInProduction),
	"rework":     string(domain.OrderStatusInProduction),
	"packed":     string(domain.OrderStatusReadyToShip),
	"canceled":   string(domain.OrderStatusCanceled),
	"completed":  string(domain.OrderStatusReadyToShip),
	"in_transit": string(domain.OrderStatusShipped),
}

var productionHoldEvents = map[string]bool{
	"on_hold": true,
	"rework":  true,
}

// OrderEventPublisher publishes order domain events for downstream consumers.
type OrderEventPublisher interface {
	PublishOrderEvent(ctx context.Context, event OrderEvent) error
}

// OrderEvent captures metadata for emitted order domain events.
type OrderEvent struct {
	Type           string
	OrderID        string
	OrderNumber    string
	PreviousStatus string
	CurrentStatus  string
	ActorID        string
	OccurredAt     time.Time
	Metadata       map[string]any
}

// OrderServiceDeps bundles collaborators required to construct the order service.
type OrderServiceDeps struct {
	Orders      repositories.OrderRepository
	Payments    repositories.OrderPaymentRepository
	Shipments   repositories.OrderShipmentRepository
	Production  repositories.OrderProductionEventRepository
	Counters    repositories.CounterRepository
	Inventory   InventoryService
	UnitOfWork  repositories.UnitOfWork
	Clock       func() time.Time
	IDGenerator func() string
	Events      OrderEventPublisher
	Logger      func(ctx context.Context, event string, fields map[string]any)
}

type orderService struct {
	orders     repositories.OrderRepository
	payments   repositories.OrderPaymentRepository
	shipments  repositories.OrderShipmentRepository
	production repositories.OrderProductionEventRepository
	counters   repositories.CounterRepository
	inventory  InventoryService
	unitOfWork repositories.UnitOfWork
	clock      func() time.Time
	newID      func() string
	events     OrderEventPublisher
	logger     func(context.Context, string, map[string]any)
}

// NewOrderService wires dependencies into a concrete OrderService implementation.
func NewOrderService(deps OrderServiceDeps) (OrderService, error) {
	if deps.Orders == nil {
		return nil, errors.New("order service: order repository is required")
	}
	if deps.Counters == nil {
		return nil, errors.New("order service: counter repository is required")
	}

	unit := deps.UnitOfWork
	if unit == nil {
		unit = noopUnitOfWork{}
	}

	clock := deps.Clock
	if clock == nil {
		clock = time.Now
	}

	idGen := deps.IDGenerator
	if idGen == nil {
		idGen = func() string {
			return ulid.Make().String()
		}
	}

	logger := deps.Logger
	if logger == nil {
		logger = func(context.Context, string, map[string]any) {}
	}

	return &orderService{
		orders:     deps.Orders,
		payments:   deps.Payments,
		shipments:  deps.Shipments,
		production: deps.Production,
		counters:   deps.Counters,
		inventory:  deps.Inventory,
		unitOfWork: unit,
		clock: func() time.Time {
			return clock().UTC()
		},
		newID:  idGen,
		events: deps.Events,
		logger: logger,
	}, nil
}

func (s *orderService) CreateFromCart(ctx context.Context, cmd CreateOrderFromCartCommand) (Order, error) {
	if len(cmd.Cart.Items) == 0 {
		return Order{}, fmt.Errorf("%w: cart must contain at least one item", ErrOrderInvalidInput)
	}
	userID := strings.TrimSpace(cmd.Cart.UserID)
	if userID == "" {
		return Order{}, fmt.Errorf("%w: cart user id is required", ErrOrderInvalidInput)
	}
	currency := strings.TrimSpace(cmd.Cart.Currency)
	if currency == "" {
		return Order{}, fmt.Errorf("%w: cart currency is required", ErrOrderInvalidInput)
	}

	now := s.now()

	order := Order{
		ID:              s.nextOrderID(),
		UserID:          userID,
		Status:          string(domain.OrderStatusPendingPayment),
		Currency:        currency,
		Totals:          buildOrderTotals(cmd.Cart),
		Items:           buildOrderLineItems(cmd.Cart.Items),
		ShippingAddress: cloneAddress(cmd.Cart.ShippingAddress),
		BillingAddress:  cloneAddress(cmd.Cart.BillingAddress),
		Promotion:       clonePromotion(cmd.Cart.Promotion),
		Metadata:        cloneAndMergeMetadata(cmd.Cart.Metadata, cmd.Metadata),
		CreatedAt:       now,
		UpdatedAt:       now,
		PlacedAt:        &now,
		Production:      OrderProduction{},
		Fulfillment:     OrderFulfillment{},
		Flags:           OrderFlags{},
	}

	if trimmed := strings.TrimSpace(cmd.Cart.ID); trimmed != "" {
		order.CartRef = valuePtr(trimmed)
	}

	if cmd.OrderNumber != nil && strings.TrimSpace(*cmd.OrderNumber) != "" {
		order.OrderNumber = strings.TrimSpace(*cmd.OrderNumber)
	}

	if order.OrderNumber == "" {
		number, err := s.generateOrderNumber(ctx, now)
		if err != nil {
			return Order{}, err
		}
		order.OrderNumber = number
	}

	if reservation := strings.TrimSpace(cmd.ReservationID); reservation != "" {
		order.Metadata = ensureMap(order.Metadata)
		order.Metadata["reservationId"] = reservation
	}

	if actor := strings.TrimSpace(cmd.ActorID); actor != "" {
		order.Audit.CreatedBy = valuePtr(actor)
		order.Audit.UpdatedBy = valuePtr(actor)
	}

	err := s.runInTx(ctx, func(txCtx context.Context) error {
		if cmd.ReservationID != "" && s.inventory != nil {
			if _, err := s.inventory.CommitReservation(txCtx, InventoryCommitCommand{
				ReservationID: cmd.ReservationID,
				OrderID:       order.ID,
				ActorID:       cmd.ActorID,
			}); err != nil {
				return err
			}
		}
		if err := s.orders.Insert(txCtx, domain.Order(order)); err != nil {
			return s.mapRepositoryError(err)
		}
		return nil
	})
	if err != nil {
		return Order{}, err
	}

	s.publishEvent(ctx, OrderEvent{
		Type:        orderEventCreated,
		OrderID:     order.ID,
		OrderNumber: order.OrderNumber,
		ActorID:     cmd.ActorID,
		OccurredAt:  now,
		Metadata:    maps.Clone(order.Metadata),
	})

	return order, nil
}

func (s *orderService) ListOrders(ctx context.Context, filter OrderListFilter) (domain.CursorPage[Order], error) {
	page, err := s.orders.List(ctx, filter)
	if err != nil {
		return domain.CursorPage[Order]{}, s.mapRepositoryError(err)
	}
	return page, nil
}

func (s *orderService) GetOrder(ctx context.Context, orderID string, opts OrderReadOptions) (Order, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return Order{}, fmt.Errorf("%w: order id is required", ErrOrderInvalidInput)
	}

	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return Order{}, s.mapRepositoryError(err)
	}

	if opts.IncludePayments {
		if s.payments == nil {
			return Order{}, errOrderPaymentRepositoryUnavailable
		}
		payments, err := s.payments.List(ctx, orderID)
		if err != nil {
			return Order{}, s.mapRepositoryError(err)
		}
		order.Payments = payments
	}

	if opts.IncludeShipments {
		if s.shipments == nil {
			return Order{}, errOrderShipmentRepositoryUnavailable
		}
		shipments, err := s.shipments.List(ctx, orderID)
		if err != nil {
			return Order{}, s.mapRepositoryError(err)
		}
		order.Shipments = shipments
	}

	if opts.IncludeProductionEvents {
		if s.production == nil {
			return Order{}, errOrderProductionRepositoryUnavailable
		}
		events, err := s.production.List(ctx, orderID)
		if err != nil {
			return Order{}, s.mapRepositoryError(err)
		}
		order.ProductionEvents = events
	}

	return order, nil
}

func (s *orderService) TransitionStatus(ctx context.Context, cmd OrderStatusTransitionCommand) (Order, error) {
	orderID := strings.TrimSpace(cmd.OrderID)
	target := strings.TrimSpace(cmd.TargetStatus)

	if orderID == "" {
		return Order{}, fmt.Errorf("%w: order id is required", ErrOrderInvalidInput)
	}
	if target == "" {
		return Order{}, fmt.Errorf("%w: target status is required", ErrOrderInvalidInput)
	}

	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return Order{}, s.mapRepositoryError(err)
	}

	if cmd.ExpectedStatus != nil && order.Status != strings.TrimSpace(*cmd.ExpectedStatus) {
		return Order{}, fmt.Errorf("%w: expected status %q but was %q", ErrOrderConflict, *cmd.ExpectedStatus, order.Status)
	}

	actor := strings.TrimSpace(cmd.ActorID)
	now := s.now()
	prevStatus := order.Status

	if _, err := s.applyStatusTransition(&order, target, actor, now); err != nil {
		return Order{}, err
	}

	err = s.runInTx(ctx, func(txCtx context.Context) error {
		if err := s.orders.Update(txCtx, order); err != nil {
			return s.mapRepositoryError(err)
		}
		return nil
	})
	if err != nil {
		return Order{}, err
	}

	metadata := ensureMap(cmd.Metadata)
	if cmd.Reason != "" {
		metadata = ensureMap(metadata)
		metadata["reason"] = strings.TrimSpace(cmd.Reason)
	}

	s.publishEvent(ctx, OrderEvent{
		Type:           orderEventStatusChanged,
		OrderID:        order.ID,
		OrderNumber:    order.OrderNumber,
		PreviousStatus: prevStatus,
		CurrentStatus:  order.Status,
		ActorID:        actor,
		OccurredAt:     now,
		Metadata:       metadata,
	})

	return order, nil
}

func (s *orderService) Cancel(ctx context.Context, cmd CancelOrderCommand) (Order, error) {
	orderID := strings.TrimSpace(cmd.OrderID)
	if orderID == "" {
		return Order{}, fmt.Errorf("%w: order id is required", ErrOrderInvalidInput)
	}

	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return Order{}, s.mapRepositoryError(err)
	}

	if !slices.Contains(cancellableStatuses, order.Status) {
		return Order{}, fmt.Errorf("%w: order status %q cannot be canceled", ErrOrderInvalidState, order.Status)
	}

	if cmd.ExpectedStatus != nil && order.Status != strings.TrimSpace(*cmd.ExpectedStatus) {
		return Order{}, fmt.Errorf("%w: expected status %q but was %q", ErrOrderConflict, *cmd.ExpectedStatus, order.Status)
	}

	now := s.now()
	prevStatus := order.Status
	reason := strings.TrimSpace(cmd.Reason)

	order.CancelReason = optionalString(reason)
	order.CanceledAt = &now

	if _, err := s.applyStatusTransition(&order, string(domain.OrderStatusCanceled), strings.TrimSpace(cmd.ActorID), now); err != nil {
		return Order{}, err
	}

	err = s.runInTx(ctx, func(txCtx context.Context) error {
		if err := s.orders.Update(txCtx, order); err != nil {
			return s.mapRepositoryError(err)
		}
		if s.inventory != nil && strings.TrimSpace(cmd.ReservationID) != "" {
			if _, err := s.inventory.ReleaseReservation(txCtx, InventoryReleaseCommand{
				ReservationID: cmd.ReservationID,
				Reason:        reason,
				ActorID:       cmd.ActorID,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Order{}, err
	}

	metadata := ensureMap(cmd.Metadata)
	if reason != "" {
		metadata = ensureMap(metadata)
		metadata["reason"] = reason
	}
	if cmd.ReservationID != "" {
		metadata = ensureMap(metadata)
		metadata["reservationId"] = strings.TrimSpace(cmd.ReservationID)
	}

	s.publishEvent(ctx, OrderEvent{
		Type:           orderEventStatusChanged,
		OrderID:        order.ID,
		OrderNumber:    order.OrderNumber,
		PreviousStatus: prevStatus,
		CurrentStatus:  order.Status,
		ActorID:        cmd.ActorID,
		OccurredAt:     now,
		Metadata:       metadata,
	})

	return order, nil
}

func (s *orderService) AppendProductionEvent(ctx context.Context, cmd AppendProductionEventCommand) (OrderProductionEvent, error) {
	if s.production == nil {
		return OrderProductionEvent{}, errOrderProductionRepositoryUnavailable
	}
	orderID := strings.TrimSpace(cmd.OrderID)
	if orderID == "" {
		return OrderProductionEvent{}, fmt.Errorf("%w: order id is required", ErrOrderInvalidInput)
	}

	eventType := strings.TrimSpace(cmd.Event.Type)
	if eventType == "" {
		return OrderProductionEvent{}, fmt.Errorf("%w: event type is required", ErrOrderInvalidInput)
	}

	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return OrderProductionEvent{}, s.mapRepositoryError(err)
	}

	now := s.now()
	event := cmd.Event
	event.ID = s.nextProductionEventID()
	event.OrderID = orderID
	event.Type = eventType
	event.CreatedAt = now

	if cmd.Event.OperatorRef != nil {
		ref := strings.TrimSpace(*cmd.Event.OperatorRef)
		if ref == "" {
			event.OperatorRef = nil
		} else {
			event.OperatorRef = &ref
		}
	}

	if cmd.Event.Station != "" {
		event.Station = strings.TrimSpace(cmd.Event.Station)
	}

	targetStatus, hasMapping := productionEventStatusMapping[eventType]
	onHold := productionHoldEvents[eventType]

	var inserted OrderProductionEvent
	prevStatus := order.Status

	err = s.runInTx(ctx, func(txCtx context.Context) error {
		var insertErr error
		inserted, insertErr = s.production.Insert(txCtx, domain.OrderProductionEvent(event))
		if insertErr != nil {
			return s.mapRepositoryError(insertErr)
		}

		order.Production.LastEventType = eventType
		order.Production.LastEventAt = &now
		if event.Station != "" {
			order.Production.AssignedStation = valuePtr(event.Station)
		}
		order.Production.OnHold = onHold

		if hasMapping && targetStatus != "" && targetStatus != order.Status {
			if _, err := s.applyStatusTransition(&order, targetStatus, cmd.ActorID, now); err != nil {
				return err
			}
		} else {
			order.UpdatedAt = now
			if actor := strings.TrimSpace(cmd.ActorID); actor != "" {
				order.Audit.UpdatedBy = valuePtr(actor)
			}
		}

		if err := s.orders.Update(txCtx, order); err != nil {
			return s.mapRepositoryError(err)
		}
		return nil
	})
	if err != nil {
		return OrderProductionEvent{}, err
	}

	metadata := map[string]any{
		"eventType": eventType,
	}
	if event.Station != "" {
		metadata["station"] = event.Station
	}

	if hasMapping && prevStatus != order.Status {
		s.publishEvent(ctx, OrderEvent{
			Type:           orderEventStatusChanged,
			OrderID:        order.ID,
			OrderNumber:    order.OrderNumber,
			PreviousStatus: prevStatus,
			CurrentStatus:  order.Status,
			ActorID:        cmd.ActorID,
			OccurredAt:     now,
			Metadata:       metadata,
		})
	}

	s.publishEvent(ctx, OrderEvent{
		Type:        orderEventProductionAppended,
		OrderID:     order.ID,
		OrderNumber: order.OrderNumber,
		ActorID:     cmd.ActorID,
		OccurredAt:  now,
		Metadata:    metadata,
	})

	return inserted, nil
}

func (s *orderService) RequestInvoice(ctx context.Context, cmd RequestInvoiceCommand) (Order, error) {
	orderID := strings.TrimSpace(cmd.OrderID)
	if orderID == "" {
		return Order{}, fmt.Errorf("%w: order id is required", ErrOrderInvalidInput)
	}

	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return Order{}, s.mapRepositoryError(err)
	}

	if cmd.ExpectedStatus != nil && order.Status != strings.TrimSpace(*cmd.ExpectedStatus) {
		return Order{}, fmt.Errorf("%w: expected status %q but was %q", ErrOrderConflict, *cmd.ExpectedStatus, order.Status)
	}

	now := s.now()
	order.Metadata = ensureMap(order.Metadata)
	order.Metadata["invoiceRequestedAt"] = now.UTC().Format(time.RFC3339Nano)
	if strings.TrimSpace(cmd.ActorID) != "" {
		order.Metadata["invoiceRequestedBy"] = strings.TrimSpace(cmd.ActorID)
	}
	if strings.TrimSpace(cmd.Notes) != "" {
		order.Metadata["invoiceNotes"] = strings.TrimSpace(cmd.Notes)
	}
	order.UpdatedAt = now
	if actor := strings.TrimSpace(cmd.ActorID); actor != "" {
		order.Audit.UpdatedBy = valuePtr(actor)
	}

	err = s.runInTx(ctx, func(txCtx context.Context) error {
		if err := s.orders.Update(txCtx, order); err != nil {
			return s.mapRepositoryError(err)
		}
		return nil
	})
	if err != nil {
		return Order{}, err
	}

	s.publishEvent(ctx, OrderEvent{
		Type:        orderEventInvoiceRequested,
		OrderID:     order.ID,
		OrderNumber: order.OrderNumber,
		ActorID:     cmd.ActorID,
		OccurredAt:  now,
		Metadata: map[string]any{
			"notes": cmd.Notes,
		},
	})

	return order, nil
}

func (s *orderService) CloneForReorder(ctx context.Context, cmd CloneForReorderCommand) (Order, error) {
	orderID := strings.TrimSpace(cmd.OrderID)
	if orderID == "" {
		return Order{}, fmt.Errorf("%w: order id is required", ErrOrderInvalidInput)
	}

	source, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return Order{}, s.mapRepositoryError(err)
	}

	if !slices.Contains([]string{string(domain.OrderStatusDelivered), string(domain.OrderStatusCompleted)}, source.Status) {
		return Order{}, fmt.Errorf("%w: reorder only allowed from delivered/completed orders", ErrOrderInvalidState)
	}

	now := s.now()
	reorder := Order{
		ID:              s.nextOrderID(),
		UserID:          source.UserID,
		Status:          string(domain.OrderStatusDraft),
		Currency:        source.Currency,
		Totals:          source.Totals,
		Items:           cloneOrderItems(source.Items),
		ShippingAddress: cloneAddress(source.ShippingAddress),
		BillingAddress:  cloneAddress(source.BillingAddress),
		Promotion:       nil,
		Metadata:        ensureMap(cmd.Metadata),
		CreatedAt:       now,
		UpdatedAt:       now,
		Production:      OrderProduction{},
		Fulfillment:     OrderFulfillment{},
		Flags:           source.Flags,
	}

	if reorder.Metadata == nil {
		reorder.Metadata = map[string]any{}
	}
	reorder.Metadata["reorderOf"] = source.ID
	reorder.Metadata["reorderSourceOrderNumber"] = source.OrderNumber

	if strings.TrimSpace(cmd.ActorID) != "" {
		reorder.Audit.CreatedBy = valuePtr(strings.TrimSpace(cmd.ActorID))
		reorder.Audit.UpdatedBy = valuePtr(strings.TrimSpace(cmd.ActorID))
	}

	number, err := s.generateOrderNumber(ctx, now)
	if err != nil {
		return Order{}, err
	}
	reorder.OrderNumber = number

	err = s.runInTx(ctx, func(txCtx context.Context) error {
		if err := s.orders.Insert(txCtx, domain.Order(reorder)); err != nil {
			return s.mapRepositoryError(err)
		}
		return nil
	})
	if err != nil {
		return Order{}, err
	}

	s.publishEvent(ctx, OrderEvent{
		Type:           orderEventCreated,
		OrderID:        reorder.ID,
		OrderNumber:    reorder.OrderNumber,
		PreviousStatus: string(domain.OrderStatusDraft),
		CurrentStatus:  reorder.Status,
		ActorID:        cmd.ActorID,
		OccurredAt:     now,
		Metadata:       reorder.Metadata,
	})

	return reorder, nil
}

func (s *orderService) applyStatusTransition(order *Order, target string, actor string, now time.Time) (string, error) {
	current := strings.TrimSpace(order.Status)
	target = strings.TrimSpace(target)

	if current == target {
		order.UpdatedAt = now
		if actor != "" {
			order.Audit.UpdatedBy = valuePtr(actor)
		}
		return current, nil
	}

	if !canTransition(current, target) {
		return "", fmt.Errorf("%w: %s → %s", ErrOrderInvalidState, current, target)
	}

	order.Status = target
	order.UpdatedAt = now
	s.updateTimestamps(order, target, now)

	if actor != "" {
		order.Audit.UpdatedBy = valuePtr(actor)
	}

	return current, nil
}

func (s *orderService) updateTimestamps(order *Order, status string, now time.Time) {
	switch status {
	case string(domain.OrderStatusPendingPayment):
		if order.PlacedAt == nil {
			order.PlacedAt = &now
		}
	case string(domain.OrderStatusPaid):
		order.PaidAt = &now
	case string(domain.OrderStatusShipped):
		order.ShippedAt = &now
	case string(domain.OrderStatusDelivered):
		order.DeliveredAt = &now
	case string(domain.OrderStatusCompleted):
		order.CompletedAt = &now
	case string(domain.OrderStatusCanceled):
		if order.CanceledAt == nil {
			order.CanceledAt = &now
		}
	}
}

func (s *orderService) mapRepositoryError(err error) error {
	if err == nil {
		return nil
	}

	var repoErr repositories.RepositoryError
	if errors.As(err, &repoErr) {
		switch {
		case repoErr.IsNotFound():
			return fmt.Errorf("%w: %v", ErrOrderNotFound, err)
		case repoErr.IsConflict():
			return fmt.Errorf("%w: %v", ErrOrderConflict, err)
		case repoErr.IsUnavailable():
			return fmt.Errorf("order: repository unavailable: %w", err)
		}
	}

	return err
}

func (s *orderService) generateOrderNumber(ctx context.Context, now time.Time) (string, error) {
	seq, err := s.counters.Next(ctx, "orders", 1)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("HF-%04d-%06d", now.Year(), seq), nil
}

func (s *orderService) runInTx(ctx context.Context, fn func(context.Context) error) error {
	if s.unitOfWork == nil {
		return fn(ctx)
	}
	return s.unitOfWork.RunInTx(ctx, fn)
}

func (s *orderService) now() time.Time {
	return s.clock()
}

func (s *orderService) nextOrderID() string {
	return orderIDPrefix + s.newID()
}

func (s *orderService) nextProductionEventID() string {
	return productionEventIDPrefix + s.newID()
}

func (s *orderService) publishEvent(ctx context.Context, event OrderEvent) {
	if s.events == nil {
		return
	}
	if event.Metadata != nil {
		event.Metadata = maps.Clone(event.Metadata)
	}
	if err := s.events.PublishOrderEvent(ctx, event); err != nil {
		s.logger(ctx, "order.event.publish.failed", map[string]any{
			"type":   event.Type,
			"order":  event.OrderID,
			"error":  err.Error(),
			"status": event.CurrentStatus,
		})
	}
}

type noopUnitOfWork struct{}

func (noopUnitOfWork) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func buildOrderTotals(cart Cart) OrderTotals {
	if cart.Estimate != nil {
		return OrderTotals{
			Subtotal: cart.Estimate.Subtotal,
			Discount: cart.Estimate.Discount,
			Shipping: cart.Estimate.Shipping,
			Tax:      cart.Estimate.Tax,
			Fees:     0,
			Total:    cart.Estimate.Total,
		}
	}

	var subtotal int64
	for _, item := range cart.Items {
		subtotal += item.UnitPrice * int64(item.Quantity)
	}

	return OrderTotals{
		Subtotal: subtotal,
		Discount: 0,
		Shipping: 0,
		Tax:      0,
		Fees:     0,
		Total:    subtotal,
	}
}

func buildOrderLineItems(items []CartItem) []OrderLineItem {
	lines := make([]OrderLineItem, 0, len(items))
	for _, item := range items {
		line := OrderLineItem{
			ProductRef: strings.TrimSpace(item.ProductID),
			SKU:        strings.TrimSpace(item.SKU),
			Name:       "",
			Options:    cloneMap(item.Customization),
			Quantity:   item.Quantity,
			UnitPrice:  item.UnitPrice,
			Total:      item.UnitPrice * int64(item.Quantity),
			Metadata:   cloneMap(item.Metadata),
		}
		if item.DesignRef != nil {
			if ref := strings.TrimSpace(*item.DesignRef); ref != "" {
				line.DesignRef = valuePtr(ref)
			}
		}
		lines = append(lines, line)
	}
	return lines
}

func cloneOrderItems(items []OrderLineItem) []OrderLineItem {
	cloned := make([]OrderLineItem, len(items))
	for i, item := range items {
		cloned[i] = OrderLineItem{
			ProductRef:     item.ProductRef,
			SKU:            item.SKU,
			Name:           item.Name,
			Options:        cloneMap(item.Options),
			DesignRef:      cloneStringPtr(item.DesignRef),
			DesignSnapshot: cloneMap(item.DesignSnapshot),
			Quantity:       item.Quantity,
			UnitPrice:      item.UnitPrice,
			Total:          item.Total,
			Metadata:       cloneMap(item.Metadata),
		}
	}
	return cloned
}

func clonePromotion(promo *CartPromotion) *CartPromotion {
	if promo == nil {
		return nil
	}
	cloned := *promo
	return &cloned
}

func cloneAddress(addr *Address) *Address {
	if addr == nil {
		return nil
	}
	cloned := *addr
	return &cloned
}

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	ref := *value
	return &ref
}

func cloneMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	return maps.Clone(src)
}

func cloneAndMergeMetadata(base map[string]any, extra map[string]any) map[string]any {
	if base == nil && extra == nil {
		return nil
	}
	result := cloneMap(base)
	if len(extra) == 0 {
		return result
	}
	if result == nil {
		result = map[string]any{}
	}
	for k, v := range extra {
		result[k] = v
	}
	return result
}

func ensureMap(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	return src
}

func valuePtr[T any](v T) *T {
	return &v
}

func optionalString(v string) *string {
	if v == "" {
		return nil
	}
	ref := v
	return &ref
}

func canTransition(current, target string) bool {
	if current == target {
		return true
	}
	next, ok := orderStateTransitions[current]
	if !ok {
		return false
	}
	return slices.Contains(next, target)
}
