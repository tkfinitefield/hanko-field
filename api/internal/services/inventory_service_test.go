package services

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

type stubInventoryRepo struct {
	reserveFn   func(ctx context.Context, req repositories.InventoryReserveRequest) (repositories.InventoryReserveResult, error)
	commitFn    func(ctx context.Context, req repositories.InventoryCommitRequest) (repositories.InventoryCommitResult, error)
	releaseFn   func(ctx context.Context, req repositories.InventoryReleaseRequest) (repositories.InventoryReleaseResult, error)
	listFn      func(ctx context.Context, query repositories.InventoryLowStockQuery) (domain.CursorPage[domain.InventoryStock], error)
	configureFn func(ctx context.Context, cfg repositories.InventorySafetyStockConfig) (domain.InventoryStock, error)
}

func (s *stubInventoryRepo) Reserve(ctx context.Context, req repositories.InventoryReserveRequest) (repositories.InventoryReserveResult, error) {
	if s.reserveFn != nil {
		return s.reserveFn(ctx, req)
	}
	return repositories.InventoryReserveResult{}, nil
}

func (s *stubInventoryRepo) Commit(ctx context.Context, req repositories.InventoryCommitRequest) (repositories.InventoryCommitResult, error) {
	if s.commitFn != nil {
		return s.commitFn(ctx, req)
	}
	return repositories.InventoryCommitResult{}, nil
}

func (s *stubInventoryRepo) Release(ctx context.Context, req repositories.InventoryReleaseRequest) (repositories.InventoryReleaseResult, error) {
	if s.releaseFn != nil {
		return s.releaseFn(ctx, req)
	}
	return repositories.InventoryReleaseResult{}, nil
}

func (s *stubInventoryRepo) GetReservation(ctx context.Context, reservationID string) (domain.InventoryReservation, error) {
	return domain.InventoryReservation{}, errors.New("not implemented")
}

func (s *stubInventoryRepo) ListLowStock(ctx context.Context, query repositories.InventoryLowStockQuery) (domain.CursorPage[domain.InventoryStock], error) {
	if s.listFn != nil {
		return s.listFn(ctx, query)
	}
	return domain.CursorPage[domain.InventoryStock]{}, nil
}

func (s *stubInventoryRepo) ConfigureSafetyStock(ctx context.Context, cfg repositories.InventorySafetyStockConfig) (domain.InventoryStock, error) {
	if s.configureFn != nil {
		return s.configureFn(ctx, cfg)
	}
	return domain.InventoryStock{}, errors.New("not implemented")
}

type captureInventoryEvents struct {
	events []InventoryStockEvent
}

func (c *captureInventoryEvents) PublishInventoryEvent(_ context.Context, event InventoryStockEvent) error {
	c.events = append(c.events, event)
	return nil
}

func TestInventoryServiceReserveAggregatesLinesAndEmitsEvents(t *testing.T) {
	now := time.Date(2025, 5, 1, 9, 0, 0, 0, time.UTC)
	repo := &stubInventoryRepo{}
	events := &captureInventoryEvents{}
	repo.reserveFn = func(_ context.Context, req repositories.InventoryReserveRequest) (repositories.InventoryReserveResult, error) {
		if len(req.Reservation.Lines) != 1 {
			t.Fatalf("expected aggregated single line, got %d", len(req.Reservation.Lines))
		}
		line := req.Reservation.Lines[0]
		if line.Quantity != 3 {
			t.Fatalf("expected quantity 3, got %d", line.Quantity)
		}
		if line.ProductRef != "/products/prod-1" {
			t.Fatalf("unexpected product ref %s", line.ProductRef)
		}
		return repositories.InventoryReserveResult{
			Reservation: req.Reservation,
			Stocks: map[string]domain.InventoryStock{
				"SKU-1": {
					SKU:         "SKU-1",
					ProductRef:  "/products/prod-1",
					OnHand:      10,
					Reserved:    3,
					Available:   7,
					SafetyStock: 2,
					SafetyDelta: 5,
					UpdatedAt:   req.Now,
				},
			},
		}, nil
	}

	svc, err := NewInventoryService(InventoryServiceDeps{
		Inventory: repo,
		Events:    events,
		Clock: func() time.Time {
			return now
		},
		IDGenerator: func() string { return "testid" },
	})
	if err != nil {
		t.Fatalf("new inventory service: %v", err)
	}

	ctx := context.Background()
	reservation, err := svc.ReserveStocks(ctx, InventoryReserveCommand{
		OrderID: "order-1",
		UserID:  "user-1",
		TTL:     time.Minute,
		Reason:  "checkout",
		Lines: []InventoryLine{
			{ProductID: "prod-1", SKU: "SKU-1", Quantity: 1},
			{ProductID: "prod-1", SKU: "SKU-1", Quantity: 2},
		},
	})
	if err != nil {
		t.Fatalf("reserve stocks: %v", err)
	}
	if reservation.ID != "sr_testid" {
		t.Fatalf("expected reservation id sr_testid, got %s", reservation.ID)
	}
	if len(events.events) != 1 {
		t.Fatalf("expected single event, got %d", len(events.events))
	}
	event := events.events[0]
	if event.Type != eventInventoryReserve {
		t.Fatalf("unexpected event type %s", event.Type)
	}
	if event.DeltaReserved != 3 || event.DeltaOnHand != 0 {
		t.Fatalf("unexpected deltas %+v", event)
	}
	if reason, ok := event.Metadata["reason"].(string); !ok || reason != "checkout" {
		t.Fatalf("expected metadata reason checkout, got %#v", event.Metadata["reason"])
	}
}

func TestInventoryServiceReserveValidatesInput(t *testing.T) {
	repo := &stubInventoryRepo{}
	svc, err := NewInventoryService(InventoryServiceDeps{
		Inventory: repo,
		Clock:     func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		t.Fatalf("new inventory service: %v", err)
	}

	_, err = svc.ReserveStocks(context.Background(), InventoryReserveCommand{})
	if err == nil || !errors.Is(err, ErrInventoryInvalidInput) {
		t.Fatalf("expected invalid input error, got %v", err)
	}
}

func TestInventoryServiceReserveMapsInsufficientStock(t *testing.T) {
	now := time.Now().UTC()
	repo := &stubInventoryRepo{}
	repo.reserveFn = func(ctx context.Context, req repositories.InventoryReserveRequest) (repositories.InventoryReserveResult, error) {
		return repositories.InventoryReserveResult{}, repositories.NewInventoryError(repositories.InventoryErrorInsufficientStock, "only 1 remaining", nil)
	}

	svc, err := NewInventoryService(InventoryServiceDeps{
		Inventory: repo,
		Clock:     func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("new inventory service: %v", err)
	}

	_, err = svc.ReserveStocks(context.Background(), InventoryReserveCommand{
		OrderID: "order-1",
		UserID:  "user-1",
		TTL:     time.Minute,
		Lines:   []InventoryLine{{ProductID: "prod-1", SKU: "SKU-1", Quantity: 2}},
	})
	if err == nil || !errors.Is(err, ErrInventoryInsufficientStock) {
		t.Fatalf("expected insufficient stock error, got %v", err)
	}
}

func TestInventoryServiceCommitEmitsEvents(t *testing.T) {
	now := time.Now().UTC()
	repo := &stubInventoryRepo{}
	events := &captureInventoryEvents{}
	repo.commitFn = func(ctx context.Context, req repositories.InventoryCommitRequest) (repositories.InventoryCommitResult, error) {
		if req.OrderRef != "/orders/order-1" {
			t.Fatalf("expected order ref /orders/order-1, got %s", req.OrderRef)
		}
		return repositories.InventoryCommitResult{
			Reservation: InventoryReservation{
				ID:        req.ReservationID,
				OrderRef:  "/orders/order-1",
				UserRef:   "/users/user-1",
				Status:    statusCommitted,
				Lines:     []InventoryReservationLine{{ProductRef: "/products/prod-1", SKU: "SKU-1", Quantity: 2}},
				UpdatedAt: req.Now,
			},
			Stocks: map[string]domain.InventoryStock{
				"SKU-1": {SKU: "SKU-1", ProductRef: "/products/prod-1", OnHand: 5, Reserved: 0, SafetyStock: 2, SafetyDelta: 3, UpdatedAt: req.Now},
			},
		}, nil
	}

	svc, err := NewInventoryService(InventoryServiceDeps{
		Inventory: repo,
		Events:    events,
		Clock: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatalf("new inventory service: %v", err)
	}

	reservation, err := svc.CommitReservation(context.Background(), InventoryCommitCommand{
		ReservationID: "sr_test",
		OrderID:       "order-1",
		ActorID:       "staff-1",
	})
	if err != nil {
		t.Fatalf("commit reservation: %v", err)
	}
	if reservation.Status != statusCommitted {
		t.Fatalf("expected committed status, got %s", reservation.Status)
	}
	if len(events.events) != 1 {
		t.Fatalf("expected one event, got %d", len(events.events))
	}
	event := events.events[0]
	if event.DeltaOnHand != -2 || event.DeltaReserved != -2 {
		t.Fatalf("unexpected deltas %+v", event)
	}
	if actor, ok := event.Metadata["actorId"].(string); !ok || actor != "staff-1" {
		t.Fatalf("expected actor metadata staff-1, got %#v", event.Metadata["actorId"])
	}
}

func TestInventoryServiceListLowStock(t *testing.T) {
	repo := &stubInventoryRepo{}
	repo.listFn = func(ctx context.Context, query repositories.InventoryLowStockQuery) (domain.CursorPage[domain.InventoryStock], error) {
		return domain.CursorPage[domain.InventoryStock]{
			Items: []domain.InventoryStock{{
				SKU:         "SKU-1",
				ProductRef:  "/products/prod-1",
				OnHand:      4,
				Reserved:    2,
				Available:   2,
				SafetyStock: 3,
				SafetyDelta: -1,
			}},
			NextPageToken: "token",
		}, nil
	}

	svc, err := NewInventoryService(InventoryServiceDeps{
		Inventory: repo,
	})
	if err != nil {
		t.Fatalf("new inventory service: %v", err)
	}

	page, err := svc.ListLowStock(context.Background(), InventoryLowStockFilter{})
	if err != nil {
		t.Fatalf("list low stock: %v", err)
	}
	if page.NextPageToken != "token" {
		t.Fatalf("expected token next page, got %s", page.NextPageToken)
	}
	if len(page.Items) != 1 || page.Items[0].SafetyDelta != -1 {
		t.Fatalf("unexpected page contents: %+v", page.Items)
	}
}

func TestInventoryServiceConfigureSafetyStock(t *testing.T) {
	repo := &stubInventoryRepo{}
	repo.configureFn = func(ctx context.Context, cfg repositories.InventorySafetyStockConfig) (domain.InventoryStock, error) {
		if cfg.SKU != "MAT-001" {
			t.Fatalf("expected sku MAT-001 got %s", cfg.SKU)
		}
		if cfg.ProductRef != "/materials/mat_001" {
			t.Fatalf("expected product ref /materials/mat_001 got %s", cfg.ProductRef)
		}
		if cfg.SafetyStock != 6 {
			t.Fatalf("expected safety stock 6 got %d", cfg.SafetyStock)
		}
		return domain.InventoryStock{SKU: cfg.SKU, ProductRef: cfg.ProductRef, SafetyStock: cfg.SafetyStock}, nil
	}
	var logged map[string]any
	svc, err := NewInventoryService(InventoryServiceDeps{
		Inventory: repo,
		Logger: func(_ context.Context, _ string, fields map[string]any) {
			logged = fields
		},
	})
	if err != nil {
		t.Fatalf("new inventory service: %v", err)
	}
	stock, err := svc.ConfigureSafetyStock(context.Background(), ConfigureSafetyStockCommand{
		SKU:         " MAT-001 ",
		ProductRef:  " /materials/mat_001 ",
		SafetyStock: 6,
	})
	if err != nil {
		t.Fatalf("configure safety stock: %v", err)
	}
	if stock.SKU != "MAT-001" || stock.SafetyStock != 6 {
		t.Fatalf("unexpected stock %+v", stock)
	}
	if logged == nil || logged["sku"] != "MAT-001" {
		t.Fatalf("expected logger fields recorded, got %#v", logged)
	}
}

func TestInventoryServiceConfigureSafetyStockValidatesInput(t *testing.T) {
	repo := &stubInventoryRepo{}
	svc, err := NewInventoryService(InventoryServiceDeps{Inventory: repo})
	if err != nil {
		t.Fatalf("new inventory service: %v", err)
	}
	if _, err := svc.ConfigureSafetyStock(context.Background(), ConfigureSafetyStockCommand{}); err == nil {
		t.Fatalf("expected error when inputs missing")
	}
	if _, err := svc.ConfigureSafetyStock(context.Background(), ConfigureSafetyStockCommand{SKU: "MAT-1", SafetyStock: -1, ProductRef: "/materials/mat_1"}); err == nil {
		t.Fatalf("expected error for negative safety stock")
	}
}
