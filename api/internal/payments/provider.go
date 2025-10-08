package payments

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Status enumerates the normalised payment states shared across providers.
type Status string

const (
	// StatusPending indicates the payment is awaiting customer action or PSP confirmation.
	StatusPending Status = "pending"
	// StatusSucceeded indicates the PSP reports the payment as successfully captured.
	StatusSucceeded Status = "succeeded"
	// StatusFailed indicates the PSP reports a failure and no further action is possible.
	StatusFailed Status = "failed"
	// StatusRefunded indicates the payment has been refunded (partially or fully).
	StatusRefunded Status = "refunded"
)

// ErrUnsupportedProvider is returned when the manager cannot locate a provider.
var ErrUnsupportedProvider = errors.New("payments: unsupported provider")

// CheckoutLineItem describes a single line item to include in a checkout session.
type CheckoutLineItem struct {
	Name        string
	Description string
	SKU         string
	Quantity    int64
	Amount      int64
	Currency    string
}

// CheckoutSessionRequest captures the payload required to create a checkout session.
type CheckoutSessionRequest struct {
	Amount         int64
	Currency       string
	CustomerID     string
	SuccessURL     string
	CancelURL      string
	Locale         string
	Metadata       map[string]string
	IdempotencyKey string
	Items          []CheckoutLineItem
	AllowPromotion bool
}

// CheckoutSession represents the PSP session returned to the client.
type CheckoutSession struct {
	ID           string
	Provider     string
	ClientSecret string
	RedirectURL  string
	IntentID     string
	ExpiresAt    time.Time
	Raw          map[string]any
}

// ConfirmRequest contains the data required to confirm a PSP session when applicable.
type ConfirmRequest struct {
	IntentID       string
	PaymentID      string
	IdempotencyKey string
	Metadata       map[string]string
}

// CaptureRequest defines a capture attempt, optionally for a partial amount.
type CaptureRequest struct {
	IntentID       string
	Amount         *int64
	IdempotencyKey string
	Metadata       map[string]string
}

// RefundRequest defines a PSP refund attempt.
type RefundRequest struct {
	IntentID       string
	Amount         *int64
	Reason         string
	IdempotencyKey string
	Metadata       map[string]string
}

// LookupRequest returns provider specific payment details for reconciliation.
type LookupRequest struct {
	IntentID string
}

// PaymentDetails normalises PSP specific fields for storage.
type PaymentDetails struct {
	Provider   string
	IntentID   string
	Status     Status
	Amount     int64
	Currency   string
	Captured   bool
	CapturedAt *time.Time
	RefundedAt *time.Time
	Raw        map[string]any
}

// Provider defines the contract for PSP adapters to implement.
type Provider interface {
	CreateCheckoutSession(ctx context.Context, req CheckoutSessionRequest) (CheckoutSession, error)
	Confirm(ctx context.Context, req ConfirmRequest) (PaymentDetails, error)
	Capture(ctx context.Context, req CaptureRequest) (PaymentDetails, error)
	Refund(ctx context.Context, req RefundRequest) (PaymentDetails, error)
	LookupPayment(ctx context.Context, req LookupRequest) (PaymentDetails, error)
}

// Manager coordinates provider selection and exposes the aggregated interface.
type Manager struct {
	providers       map[string]Provider
	defaultProvider string
	currencyRoutes  map[string]string
}

// ManagerOption configures optional behaviour when building a Manager.
type ManagerOption func(*Manager)

// WithDefaultProvider overrides the default provider for currencies without explicit routing.
func WithDefaultProvider(provider string) ManagerOption {
	return func(m *Manager) {
		m.defaultProvider = provider
	}
}

// WithCurrencyRoutes configures static currency to provider mappings.
func WithCurrencyRoutes(routes map[string]string) ManagerOption {
	return func(m *Manager) {
		if len(routes) == 0 {
			return
		}
		if m.currencyRoutes == nil {
			m.currencyRoutes = make(map[string]string, len(routes))
		}
		for k, v := range routes {
			m.currencyRoutes[strings.ToUpper(strings.TrimSpace(k))] = strings.TrimSpace(v)
		}
	}
}

// NewManager constructs a Manager over the supplied providers.
func NewManager(providers map[string]Provider, opts ...ManagerOption) (*Manager, error) {
	if len(providers) == 0 {
		return nil, errors.New("payments: at least one provider is required")
	}
	copyMap := make(map[string]Provider, len(providers))
	for k, v := range providers {
		key := strings.TrimSpace(strings.ToLower(k))
		if key == "" || v == nil {
			return nil, fmt.Errorf("payments: invalid provider registration for key %q", k)
		}
		copyMap[key] = v
	}
	m := &Manager{
		providers: copyMap,
	}
	if _, ok := copyMap["stripe"]; ok {
		m.defaultProvider = "stripe"
	}
	for _, opt := range opts {
		opt(m)
	}
	return m, nil
}

// PaymentContext defines the hints available when selecting a provider.
type PaymentContext struct {
	PreferredProvider string
	Currency          string
	Metadata          map[string]string
}

func (m *Manager) resolveProvider(ctx PaymentContext) (string, Provider, error) {
	if m == nil {
		return "", nil, errors.New("payments: manager is nil")
	}
	if len(m.providers) == 0 {
		return "", nil, errors.New("payments: no providers registered")
	}
	if provider := strings.TrimSpace(strings.ToLower(ctx.PreferredProvider)); provider != "" {
		if p, ok := m.providers[provider]; ok {
			return provider, p, nil
		}
	}
	currency := strings.ToUpper(strings.TrimSpace(ctx.Currency))
	if currency != "" && m.currencyRoutes != nil {
		if providerKey, ok := m.currencyRoutes[currency]; ok {
			provider := strings.TrimSpace(strings.ToLower(providerKey))
			if p, ok := m.providers[provider]; ok {
				return provider, p, nil
			}
		}
	}
	if def := strings.TrimSpace(strings.ToLower(m.defaultProvider)); def != "" {
		if p, ok := m.providers[def]; ok {
			return def, p, nil
		}
	}
	if len(m.providers) == 1 {
		for key, p := range m.providers {
			return key, p, nil
		}
	}
	return "", nil, ErrUnsupportedProvider
}

// CreateCheckoutSession delegates to the resolved provider.
func (m *Manager) CreateCheckoutSession(ctx context.Context, paymentCtx PaymentContext, req CheckoutSessionRequest) (CheckoutSession, error) {
	key, provider, err := m.resolveProvider(paymentCtx)
	if err != nil {
		return CheckoutSession{}, err
	}
	session, err := provider.CreateCheckoutSession(ctx, req)
	if err != nil {
		return CheckoutSession{}, err
	}
	session.Provider = key
	return session, nil
}

// Confirm delegates to the resolved provider.
func (m *Manager) Confirm(ctx context.Context, paymentCtx PaymentContext, req ConfirmRequest) (PaymentDetails, error) {
	_, provider, err := m.resolveProvider(paymentCtx)
	if err != nil {
		return PaymentDetails{}, err
	}
	return provider.Confirm(ctx, req)
}

// Capture delegates to the resolved provider.
func (m *Manager) Capture(ctx context.Context, paymentCtx PaymentContext, req CaptureRequest) (PaymentDetails, error) {
	_, provider, err := m.resolveProvider(paymentCtx)
	if err != nil {
		return PaymentDetails{}, err
	}
	return provider.Capture(ctx, req)
}

// Refund delegates to the resolved provider.
func (m *Manager) Refund(ctx context.Context, paymentCtx PaymentContext, req RefundRequest) (PaymentDetails, error) {
	_, provider, err := m.resolveProvider(paymentCtx)
	if err != nil {
		return PaymentDetails{}, err
	}
	return provider.Refund(ctx, req)
}

// LookupPayment delegates to the resolved provider.
func (m *Manager) LookupPayment(ctx context.Context, paymentCtx PaymentContext, req LookupRequest) (PaymentDetails, error) {
	_, provider, err := m.resolveProvider(paymentCtx)
	if err != nil {
		return PaymentDetails{}, err
	}
	return provider.LookupPayment(ctx, req)
}
