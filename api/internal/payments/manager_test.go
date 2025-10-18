package payments

import (
	"context"
	"errors"
	"testing"
)

type fakeProvider struct {
	lastOp  string
	session CheckoutSession
	payment PaymentDetails
	err     error
}

func (f *fakeProvider) CreateCheckoutSession(ctx context.Context, req CheckoutSessionRequest) (CheckoutSession, error) {
	f.lastOp = "create"
	return f.session, f.err
}

func (f *fakeProvider) Confirm(ctx context.Context, req ConfirmRequest) (PaymentDetails, error) {
	f.lastOp = "confirm"
	return f.payment, f.err
}

func (f *fakeProvider) Capture(ctx context.Context, req CaptureRequest) (PaymentDetails, error) {
	f.lastOp = "capture"
	return f.payment, f.err
}

func (f *fakeProvider) Refund(ctx context.Context, req RefundRequest) (PaymentDetails, error) {
	f.lastOp = "refund"
	return f.payment, f.err
}

func (f *fakeProvider) LookupPayment(ctx context.Context, req LookupRequest) (PaymentDetails, error) {
	f.lastOp = "lookup"
	return f.payment, f.err
}

func TestManagerCreateCheckoutSessionUsesPreferredProvider(t *testing.T) {
	ctx := context.Background()
	stripe := &fakeProvider{session: CheckoutSession{ID: "sess_stripe"}}
	paypal := &fakeProvider{session: CheckoutSession{ID: "sess_paypal"}}

	mgr, err := NewManager(map[string]Provider{
		"stripe": stripe,
		"paypal": paypal,
	})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	session, err := mgr.CreateCheckoutSession(ctx, PaymentContext{PreferredProvider: "paypal"}, CheckoutSessionRequest{Currency: "USD"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if session.Provider != "paypal" {
		t.Fatalf("expected provider 'paypal', got %q", session.Provider)
	}
	if paypal.lastOp != "create" {
		t.Fatalf("expected paypal provider to handle call")
	}
	if stripe.lastOp != "" {
		t.Fatalf("expected stripe provider to remain unused")
	}
}

func TestManagerRoutesByCurrency(t *testing.T) {
	ctx := context.Background()
	stripe := &fakeProvider{session: CheckoutSession{ID: "sess_stripe"}}
	paypal := &fakeProvider{session: CheckoutSession{ID: "sess_paypal"}}

	mgr, err := NewManager(
		map[string]Provider{
			"stripe": stripe,
			"paypal": paypal,
		},
		WithCurrencyRoutes(map[string]string{"JPY": "paypal"}),
	)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	session, err := mgr.CreateCheckoutSession(ctx, PaymentContext{Currency: "JPY"}, CheckoutSessionRequest{Currency: "JPY"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if session.Provider != "paypal" {
		t.Fatalf("expected provider 'paypal', got %q", session.Provider)
	}
	if paypal.lastOp != "create" {
		t.Fatalf("expected paypal provider to handle call")
	}
}

func TestManagerFallsBackToDefault(t *testing.T) {
	ctx := context.Background()
	stripe := &fakeProvider{payment: PaymentDetails{Provider: "stripe"}}

	mgr, err := NewManager(map[string]Provider{"stripe": stripe})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	details, err := mgr.Capture(ctx, PaymentContext{}, CaptureRequest{IntentID: "pi_123"})
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	if stripe.lastOp != "capture" {
		t.Fatalf("expected capture to invoke default provider")
	}
	if details.Provider != "stripe" {
		t.Fatalf("unexpected provider in details: %q", details.Provider)
	}
}

func TestManagerUnsupportedProvider(t *testing.T) {
	ctx := context.Background()
	mgr, err := NewManager(map[string]Provider{"stripe": &fakeProvider{}, "paypal": &fakeProvider{}}, WithDefaultProvider(""))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	_, err = mgr.CreateCheckoutSession(ctx, PaymentContext{PreferredProvider: "unknown"}, CheckoutSessionRequest{Currency: "USD"})
	if !errors.Is(err, ErrUnsupportedProvider) {
		t.Fatalf("expected ErrUnsupportedProvider, got %v", err)
	}
}

func TestNewManagerValidatesProviders(t *testing.T) {
	if _, err := NewManager(map[string]Provider{"bad": nil}); err == nil {
		t.Fatalf("expected error for nil provider")
	}
	if _, err := NewManager(nil); err == nil {
		t.Fatalf("expected error when providers empty")
	}
}
