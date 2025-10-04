package di

import (
	"context"
	"errors"

	"github.com/hanko-field/api/internal/repositories"
	"github.com/hanko-field/api/internal/services"
)

// Services bundles the service-layer contracts that handlers rely upon. Concrete implementations
// are assembled via dependency injection in NewContainer.
type Services struct {
	Design     services.DesignService
	Cart       services.CartService
	Checkout   services.CheckoutService
	Orders     services.OrderService
	Payments   services.PaymentService
	Shipments  services.ShipmentService
	Promotions services.PromotionService
	Users      services.UserService
	Inventory  services.InventoryService
	Content    services.ContentService
	Catalog    services.CatalogService
	Assets     services.AssetService
	System     services.SystemService
	Jobs       services.BackgroundJobDispatcher
	Errors     services.ErrorTranslator
}

// Container wires repositories, services, and background infrastructure for runtime use.
type Container struct {
	Repositories repositories.Registry
	Services     Services
}

// NewContainer constructs the runtime dependencies. In production this will be generated via
// google/wire using provider sets declared in the di package. Tests can supply fake repositories.
func NewContainer(ctx context.Context, reg repositories.Registry, overrides ...Option) (*Container, error) {
	if reg == nil {
		return nil, errors.New("repositories registry is required")
	}

	cfg := options{ctx: ctx}
	for _, opt := range overrides {
		opt(&cfg)
	}

	svc, err := buildServices(cfg, reg)
	if err != nil {
		return nil, err
	}

	return &Container{
		Repositories: reg,
		Services:     svc,
	}, nil
}

// Close releases resources such as repository clients, background workers, or caches.
func (c *Container) Close(ctx context.Context) error {
	if c == nil || c.Repositories == nil {
		return nil
	}
	return c.Repositories.Close(ctx)
}

// Option configures the container bootstrap for custom wiring (e.g., mocks in tests).
type Option func(*options)

type options struct {
	ctx context.Context
}

func buildServices(cfg options, reg repositories.Registry) (Services, error) {
	// Placeholder wiring; concrete implementation will compose services via constructors once
	// repositories are implemented. During planning we return zero-valued interfaces so the
	// package compiles and tests can inject their own doubles.
	return Services{}, nil
}
