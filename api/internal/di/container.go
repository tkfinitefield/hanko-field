package di

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/platform/config"
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
	Reviews    services.ReviewService
	Counters   services.CounterService
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
	Audit      services.AuditLogService
}

// Container wires repositories, services, and background infrastructure for runtime use.
type Container struct {
	Config       config.Config
	Repositories repositories.Registry
	Services     Services
}

// NewContainer constructs the runtime dependencies. Production wiring will provide real
// implementations, while tests can supply in-memory registries.
func NewContainer(ctx context.Context, cfg config.Config, reg repositories.Registry) (*Container, error) {
	if reg == nil {
		return nil, errors.New("repositories registry is required")
	}

	svc, err := buildServices(ctx, reg, cfg)
	if err != nil {
		return nil, err
	}

	return &Container{
		Config:       cfg,
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

func buildServices(ctx context.Context, reg repositories.Registry, cfg config.Config) (Services, error) {
	var svc Services
	if reg == nil {
		return svc, nil
	}

	if auditRepo := reg.AuditLogs(); auditRepo != nil {
		auditSvc, err := services.NewAuditLogService(services.AuditLogServiceDeps{
			Repository: auditRepo,
			Clock:      time.Now,
		})
		if err != nil {
			return Services{}, fmt.Errorf("build audit log service: %w", err)
		}
		svc.Audit = auditSvc
	}

	if usersRepo := reg.Users(); usersRepo != nil && cfg.Firebase.ProjectID != "" {
		firebase, err := auth.NewFirebaseVerifier(ctx, cfg.Firebase)
		if err != nil {
			return Services{}, fmt.Errorf("build firebase verifier: %w", err)
		}
		userSvc, err := services.NewUserService(services.UserServiceDeps{
			Users:          usersRepo,
			Addresses:      reg.Addresses(),
			PaymentMethods: reg.PaymentMethods(),
			Favorites:      reg.Favorites(),
			Designs:        reg.Designs(),
			Audit:          svc.Audit,
			Firebase:       firebase,
			Clock:          time.Now,
		})
		if err != nil {
			return Services{}, fmt.Errorf("build user service: %w", err)
		}
		svc.Users = userSvc
	}

	if inventoryRepo := reg.Inventory(); inventoryRepo != nil {
		inventorySvc, err := services.NewInventoryService(services.InventoryServiceDeps{
			Inventory: inventoryRepo,
			Clock:     time.Now,
		})
		if err != nil {
			return Services{}, fmt.Errorf("build inventory service: %w", err)
		}
		svc.Inventory = inventorySvc
	}

	if promotionsRepo := reg.Promotions(); promotionsRepo != nil {
		promotionSvc, err := services.NewPromotionService(services.PromotionServiceDeps{
			Promotions: promotionsRepo,
			Clock:      time.Now,
		})
		if err != nil {
			return Services{}, fmt.Errorf("build promotion service: %w", err)
		}
		svc.Promotions = promotionSvc
	}

	if catalogRepo := reg.Catalog(); catalogRepo != nil {
		catalogSvc, err := services.NewCatalogService(services.CatalogServiceDeps{
			Catalog:   catalogRepo,
			Audit:     svc.Audit,
			Inventory: svc.Inventory,
			Clock:     time.Now,
		})
		if err != nil {
			return Services{}, fmt.Errorf("build catalog service: %w", err)
		}
		svc.Catalog = catalogSvc
	}

	if contentRepo := reg.Content(); contentRepo != nil {
		contentSvc, err := services.NewContentService(services.ContentServiceDeps{
			Repository: contentRepo,
			Clock:      time.Now,
		})
		if err != nil {
			return Services{}, fmt.Errorf("build content service: %w", err)
		}
		svc.Content = contentSvc
	}

	counterRepo := reg.Counters()
	if counterRepo != nil {
		counterSvc, err := services.NewCounterService(services.CounterServiceDeps{
			Repository: counterRepo,
			Clock:      time.Now,
		})
		if err != nil {
			return Services{}, fmt.Errorf("build counter service: %w", err)
		}
		svc.Counters = counterSvc
	}

	if healthRepo := reg.Health(); healthRepo != nil {
		systemSvc, err := services.NewSystemService(services.SystemServiceDeps{
			HealthRepository: healthRepo,
			Clock:            time.Now,
			Build: services.BuildInfo{
				Environment: cfg.Security.Environment,
				StartedAt:   time.Now().UTC(),
			},
			Audit:    svc.Audit,
			Counters: svc.Counters,
		})
		if err != nil {
			return Services{}, fmt.Errorf("build system service: %w", err)
		}
		svc.System = systemSvc
	}

	ordersRepo := reg.Orders()
	if ordersRepo != nil && counterRepo != nil {
		orderSvc, err := services.NewOrderService(services.OrderServiceDeps{
			Orders:     ordersRepo,
			Payments:   reg.OrderPayments(),
			Shipments:  reg.OrderShipments(),
			Production: reg.OrderProductionEvents(),
			Counters:   counterRepo,
			Inventory:  svc.Inventory,
			UnitOfWork: reg,
			Clock:      time.Now,
		})
		if err != nil {
			return Services{}, fmt.Errorf("build order service: %w", err)
		}
		svc.Orders = orderSvc
	}

	if reviewRepo := reg.Reviews(); reviewRepo != nil && ordersRepo != nil {
		reviewSvc, err := services.NewReviewService(services.ReviewServiceDeps{
			Reviews: reviewRepo,
			Orders:  ordersRepo,
			Clock:   time.Now,
		})
		if err != nil {
			return Services{}, fmt.Errorf("build review service: %w", err)
		}
		svc.Reviews = reviewSvc
	}

	return svc, nil
}
