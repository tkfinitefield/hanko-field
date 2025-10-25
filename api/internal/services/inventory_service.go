package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

const (
	eventInventoryReserve = "inventory.reserve"
	eventInventoryCommit  = "inventory.commit"
	eventInventoryRelease = "inventory.release"

	statusReserved  = "reserved"
	statusCommitted = "committed"
	statusReleased  = "released"
)

var (
	// ErrInventoryInvalidInput signals the caller provided invalid arguments.
	ErrInventoryInvalidInput = errors.New("inventory: invalid input")
	// ErrInventoryInsufficientStock indicates the requested quantity exceeds availability.
	ErrInventoryInsufficientStock = errors.New("inventory: insufficient stock")
	// ErrInventoryReservationNotFound indicates the reservation could not be located.
	ErrInventoryReservationNotFound = errors.New("inventory: reservation not found")
	// ErrInventoryInvalidState indicates the reservation cannot transition due to its state.
	ErrInventoryInvalidState = errors.New("inventory: reservation state invalid")
)

// InventoryServiceDeps bundles the collaborators required to construct an inventory service.
type InventoryServiceDeps struct {
	Inventory   repositories.InventoryRepository
	Events      InventoryEventPublisher
	Clock       func() time.Time
	IDGenerator func() string
	Logger      func(ctx context.Context, event string, fields map[string]any)
}

type inventoryService struct {
	repo   repositories.InventoryRepository
	events InventoryEventPublisher
	clock  func() time.Time
	newID  func() string
	logger func(context.Context, string, map[string]any)
}

// NewInventoryService wires dependencies into a concrete InventoryService implementation.
func NewInventoryService(deps InventoryServiceDeps) (InventoryService, error) {
	if deps.Inventory == nil {
		return nil, errors.New("inventory service: inventory repository is required")
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

	return &inventoryService{
		repo:   deps.Inventory,
		events: deps.Events,
		clock: func() time.Time {
			return clock().UTC()
		},
		newID:  idGen,
		logger: logger,
	}, nil
}

func (s *inventoryService) ReserveStocks(ctx context.Context, cmd InventoryReserveCommand) (InventoryReservation, error) {
	if err := s.validateReserveInput(cmd); err != nil {
		return InventoryReservation{}, err
	}

	now := s.now()
	lines, err := normaliseInventoryLines(cmd.Lines)
	if err != nil {
		return InventoryReservation{}, err
	}

	reservation := InventoryReservation{
		ID:             ensureReservationID(s.newID()),
		OrderRef:       ensureOrderRef(cmd.OrderID),
		UserRef:        ensureUserRef(cmd.UserID),
		Status:         statusReserved,
		Lines:          lines,
		Reason:         strings.TrimSpace(cmd.Reason),
		IdempotencyKey: strings.TrimSpace(cmd.IdempotencyKey),
		ExpiresAt:      now.Add(cmd.TTL),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	req := repositories.InventoryReserveRequest{
		Reservation: reservation,
		Now:         now,
	}

	result, err := s.repo.Reserve(ctx, req)
	if err != nil {
		return InventoryReservation{}, s.mapRepositoryError(err)
	}

	reserveResult := result.Reservation
	if reserveResult.ID == "" {
		reserveResult = reservation
	}

	deltas := make(map[string]stockDelta)
	for _, line := range reserveResult.Lines {
		sku := strings.TrimSpace(line.SKU)
		delta := deltas[sku]
		delta.Reserved += line.Quantity
		deltas[sku] = delta
	}

	metadata := map[string]any{}
	if reason := strings.TrimSpace(reservation.Reason); reason != "" {
		metadata["reason"] = reason
	}

	s.logEventFailure(ctx, s.emitStockEvents(ctx, eventInventoryReserve, reserveResult, result.Stocks, deltas, metadata))

	return reserveResult, nil
}

func (s *inventoryService) CommitReservation(ctx context.Context, cmd InventoryCommitCommand) (InventoryReservation, error) {
	reservationID := strings.TrimSpace(cmd.ReservationID)
	if reservationID == "" {
		return InventoryReservation{}, fmt.Errorf("%w: reservation id is required", ErrInventoryInvalidInput)
	}

	now := s.now()
	req := repositories.InventoryCommitRequest{
		ReservationID: reservationID,
		OrderRef:      ensureOrderRef(cmd.OrderID),
		Now:           now,
	}

	result, err := s.repo.Commit(ctx, req)
	if err != nil {
		return InventoryReservation{}, s.mapRepositoryError(err)
	}

	metadata := map[string]any{}
	if actor := strings.TrimSpace(cmd.ActorID); actor != "" {
		metadata["actorId"] = actor
	}

	deltas := make(map[string]stockDelta)
	for _, line := range result.Reservation.Lines {
		sku := strings.TrimSpace(line.SKU)
		delta := deltas[sku]
		delta.OnHand -= line.Quantity
		delta.Reserved -= line.Quantity
		deltas[sku] = delta
	}

	s.logEventFailure(ctx, s.emitStockEvents(ctx, eventInventoryCommit, result.Reservation, result.Stocks, deltas, metadata))

	return result.Reservation, nil
}

func (s *inventoryService) ReleaseReservation(ctx context.Context, cmd InventoryReleaseCommand) (InventoryReservation, error) {
	reservationID := strings.TrimSpace(cmd.ReservationID)
	if reservationID == "" {
		return InventoryReservation{}, fmt.Errorf("%w: reservation id is required", ErrInventoryInvalidInput)
	}

	now := s.now()
	req := repositories.InventoryReleaseRequest{
		ReservationID: reservationID,
		Reason:        strings.TrimSpace(cmd.Reason),
		Now:           now,
	}

	result, err := s.repo.Release(ctx, req)
	if err != nil {
		return InventoryReservation{}, s.mapRepositoryError(err)
	}

	metadata := map[string]any{}
	if reason := strings.TrimSpace(cmd.Reason); reason != "" {
		metadata["reason"] = reason
	}
	if actor := strings.TrimSpace(cmd.ActorID); actor != "" {
		metadata["actorId"] = actor
	}

	deltas := make(map[string]stockDelta)
	for _, line := range result.Reservation.Lines {
		sku := strings.TrimSpace(line.SKU)
		delta := deltas[sku]
		delta.Reserved -= line.Quantity
		deltas[sku] = delta
	}

	s.logEventFailure(ctx, s.emitStockEvents(ctx, eventInventoryRelease, result.Reservation, result.Stocks, deltas, metadata))

	return result.Reservation, nil
}

func (s *inventoryService) ListLowStock(ctx context.Context, filter InventoryLowStockFilter) (domain.CursorPage[InventorySnapshot], error) {
	req := repositories.InventoryLowStockQuery{
		Threshold: filter.Threshold,
		PageSize:  filter.Pagination.PageSize,
		PageToken: filter.Pagination.PageToken,
	}

	page, err := s.repo.ListLowStock(ctx, req)
	if err != nil {
		return domain.CursorPage[InventorySnapshot]{}, s.mapRepositoryError(err)
	}

	snapshots := make([]InventorySnapshot, len(page.Items))
	for i, stock := range page.Items {
		snapshots[i] = InventorySnapshot{
			SKU:         stock.SKU,
			ProductRef:  stock.ProductRef,
			OnHand:      stock.OnHand,
			Reserved:    stock.Reserved,
			Available:   stock.Available,
			SafetyStock: stock.SafetyStock,
			SafetyDelta: stock.SafetyDelta,
			UpdatedAt:   stock.UpdatedAt,
		}
	}

	return domain.CursorPage[InventorySnapshot]{
		Items:         snapshots,
		NextPageToken: page.NextPageToken,
	}, nil
}

func (s *inventoryService) ConfigureSafetyStock(ctx context.Context, cmd ConfigureSafetyStockCommand) (InventoryStock, error) {
	sku := strings.TrimSpace(cmd.SKU)
	if sku == "" {
		return InventoryStock{}, fmt.Errorf("%w: sku is required", ErrInventoryInvalidInput)
	}
	productRef := strings.TrimSpace(cmd.ProductRef)
	if productRef == "" {
		return InventoryStock{}, fmt.Errorf("%w: product ref is required", ErrInventoryInvalidInput)
	}
	if cmd.SafetyStock < 0 {
		return InventoryStock{}, fmt.Errorf("%w: safety stock must be >= 0", ErrInventoryInvalidInput)
	}
	stock, err := s.repo.ConfigureSafetyStock(ctx, repositories.InventorySafetyStockConfig{
		SKU:         sku,
		ProductRef:  productRef,
		SafetyStock: cmd.SafetyStock,
		Now:         s.now(),
	})
	if err != nil {
		return InventoryStock{}, s.mapRepositoryError(err)
	}
	if s.logger != nil {
		s.logger(ctx, "inventory.configureSafety", map[string]any{
			"sku":         sku,
			"productRef":  productRef,
			"safetyStock": stock.SafetyStock,
		})
	}
	return InventoryStock(stock), nil
}

func (s *inventoryService) now() time.Time {
	return s.clock()
}

func (s *inventoryService) validateReserveInput(cmd InventoryReserveCommand) error {
	if strings.TrimSpace(cmd.OrderID) == "" {
		return fmt.Errorf("%w: order id is required", ErrInventoryInvalidInput)
	}
	if strings.TrimSpace(cmd.UserID) == "" {
		return fmt.Errorf("%w: user id is required", ErrInventoryInvalidInput)
	}
	if len(cmd.Lines) == 0 {
		return fmt.Errorf("%w: at least one line is required", ErrInventoryInvalidInput)
	}
	if cmd.TTL <= 0 {
		return fmt.Errorf("%w: ttl must be positive", ErrInventoryInvalidInput)
	}
	return nil
}

func (s *inventoryService) mapRepositoryError(err error) error {
	if err == nil {
		return nil
	}

	var invErr *repositories.InventoryError
	if errors.As(err, &invErr) {
		switch invErr.Code {
		case repositories.InventoryErrorInsufficientStock:
			return fmt.Errorf("%w: %s", ErrInventoryInsufficientStock, invErr.Message)
		case repositories.InventoryErrorReservationNotFound:
			return fmt.Errorf("%w: %s", ErrInventoryReservationNotFound, invErr.Message)
		case repositories.InventoryErrorInvalidReservationState:
			return fmt.Errorf("%w: %s", ErrInventoryInvalidState, invErr.Message)
		case repositories.InventoryErrorStockNotFound:
			return fmt.Errorf("%w: %s", ErrInventoryInvalidInput, invErr.Message)
		}
	}

	return err
}

func (s *inventoryService) emitStockEvents(ctx context.Context, eventType string, reservation InventoryReservation, stocks map[string]domain.InventoryStock, deltas map[string]stockDelta, metadata map[string]any) error {
	if s.events == nil {
		return nil
	}

	aggregated := aggregateReservationLines(reservation.Lines)

	occurredAt := reservation.UpdatedAt
	if occurredAt.IsZero() {
		occurredAt = s.now()
	}

	for sku, line := range aggregated {
		stock := stocks[sku]
		delta := deltas[sku]

		event := InventoryStockEvent{
			Type:          eventType,
			ReservationID: reservation.ID,
			OrderRef:      reservation.OrderRef,
			UserRef:       reservation.UserRef,
			SKU:           sku,
			ProductRef:    line.ProductRef,
			DeltaOnHand:   delta.OnHand,
			DeltaReserved: delta.Reserved,
			OnHand:        stock.OnHand,
			Reserved:      stock.Reserved,
			SafetyStock:   stock.SafetyStock,
			OccurredAt:    occurredAt,
		}
		if len(metadata) > 0 {
			copy := make(map[string]any, len(metadata))
			for k, v := range metadata {
				copy[k] = v
			}
			event.Metadata = copy
		}

		if err := s.events.PublishInventoryEvent(ctx, event); err != nil {
			return err
		}
	}

	return nil
}

func (s *inventoryService) logEventFailure(ctx context.Context, err error) {
	if err == nil {
		return
	}
	if s.logger != nil {
		s.logger(ctx, "inventory_event_publish_failed", map[string]any{"error": err.Error()})
	}
}

func normaliseInventoryLines(lines []InventoryLine) ([]InventoryReservationLine, error) {
	aggregated := make(map[string]*InventoryReservationLine)
	for _, line := range lines {
		sku := strings.TrimSpace(line.SKU)
		if sku == "" {
			return nil, fmt.Errorf("%w: line sku is required", ErrInventoryInvalidInput)
		}
		productID := strings.TrimSpace(line.ProductID)
		if productID == "" {
			return nil, fmt.Errorf("%w: line product id is required", ErrInventoryInvalidInput)
		}
		if line.Quantity <= 0 {
			return nil, fmt.Errorf("%w: quantity for %s must be positive", ErrInventoryInvalidInput, sku)
		}

		ref := fmt.Sprintf("/products/%s", productID)
		agg, ok := aggregated[sku]
		if !ok {
			agg = &InventoryReservationLine{ProductRef: ref, SKU: sku}
			aggregated[sku] = agg
		} else if agg.ProductRef != ref {
			return nil, fmt.Errorf("%w: conflicting product references for sku %s", ErrInventoryInvalidInput, sku)
		}
		agg.Quantity += line.Quantity
	}

	result := make([]InventoryReservationLine, 0, len(aggregated))
	for _, line := range aggregated {
		result = append(result, *line)
	}
	if len(result) > 1 {
		sort.Slice(result, func(i, j int) bool { return result[i].SKU < result[j].SKU })
	}
	return result, nil
}

func ensureReservationID(candidate string) string {
	trimmed := strings.TrimSpace(candidate)
	if trimmed == "" {
		trimmed = ulid.Make().String()
	}
	if strings.HasPrefix(trimmed, "sr_") {
		return trimmed
	}
	return "sr_" + trimmed
}

func ensureOrderRef(orderID string) string {
	trimmed := strings.TrimSpace(orderID)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "/orders/") {
		return trimmed
	}
	return "/orders/" + trimmed
}

func ensureUserRef(userID string) string {
	trimmed := strings.TrimSpace(userID)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "/users/") {
		return trimmed
	}
	return "/users/" + trimmed
}

func aggregateReservationLines(lines []InventoryReservationLine) map[string]InventoryReservationLine {
	aggregated := make(map[string]InventoryReservationLine, len(lines))
	for _, line := range lines {
		sku := strings.TrimSpace(line.SKU)
		if sku == "" {
			continue
		}
		agg := aggregated[sku]
		agg.ProductRef = strings.TrimSpace(line.ProductRef)
		agg.SKU = sku
		agg.Quantity += line.Quantity
		aggregated[sku] = agg
	}
	return aggregated
}

type stockDelta struct {
	OnHand   int
	Reserved int
}
