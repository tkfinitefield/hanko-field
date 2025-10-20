package payments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/client"
)

// StripeLogger defines the logging contract for Stripe provider operations.
type StripeLogger func(ctx context.Context, event string, fields map[string]any)

type stripeSessionAPI interface {
	New(params *stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error)
}

type stripePaymentIntentAPI interface {
	Confirm(id string, params *stripe.PaymentIntentConfirmParams) (*stripe.PaymentIntent, error)
	Capture(id string, params *stripe.PaymentIntentCaptureParams) (*stripe.PaymentIntent, error)
	Get(id string, params *stripe.PaymentIntentParams) (*stripe.PaymentIntent, error)
}

type stripeRefundAPI interface {
	New(params *stripe.RefundParams) (*stripe.Refund, error)
}

type stripePaymentMethodAPI interface {
	Get(id string, params *stripe.PaymentMethodParams) (*stripe.PaymentMethod, error)
}

type stripeClients struct {
	sessions       stripeSessionAPI
	intents        stripePaymentIntentAPI
	refunds        stripeRefundAPI
	paymentMethods stripePaymentMethodAPI
}

// StripeProviderConfig configures the StripeProvider.
type StripeProviderConfig struct {
	APIKey    string
	AccountID string
	Backends  *stripe.Backends
	Logger    StripeLogger
	Clock     func() time.Time
	Clients   *stripeClients
}

// StripeProvider implements the Provider interface using Stripe APIs.
type StripeProvider struct {
	api     stripeClients
	account string
	clock   func() time.Time
	logger  StripeLogger
}

// NewStripeProvider constructs a Stripe Provider using the given configuration.
func NewStripeProvider(cfg StripeProviderConfig) (*StripeProvider, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" && cfg.Clients == nil {
		return nil, errors.New("stripe: api key is required")
	}

	var clients stripeClients
	if cfg.Clients != nil {
		clients = *cfg.Clients
	} else {
		sc := client.New(apiKey, cfg.Backends)
		clients = stripeClients{
			sessions:       sc.CheckoutSessions,
			intents:        sc.PaymentIntents,
			refunds:        sc.Refunds,
			paymentMethods: sc.PaymentMethods,
		}
	}

	if clients.sessions == nil || clients.intents == nil || clients.refunds == nil || clients.paymentMethods == nil {
		return nil, errors.New("stripe: incomplete client configuration")
	}

	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}

	logger := cfg.Logger
	if logger == nil {
		logger = func(context.Context, string, map[string]any) {}
	}

	return &StripeProvider{
		api:     clients,
		account: strings.TrimSpace(cfg.AccountID),
		clock: func() time.Time {
			return clock().UTC()
		},
		logger: logger,
	}, nil
}

// CreateCheckoutSession creates a Stripe Checkout session.
func (p *StripeProvider) CreateCheckoutSession(ctx context.Context, req CheckoutSessionRequest) (CheckoutSession, error) {
	if p == nil {
		return CheckoutSession{}, errors.New("stripe: provider is nil")
	}

	params := &stripe.CheckoutSessionParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(req.SuccessURL),
		CancelURL:  stripe.String(req.CancelURL),
	}

	params.Context = ctx
	if key := strings.TrimSpace(req.IdempotencyKey); key != "" {
		params.SetIdempotencyKey(key)
	}
	if p.account != "" {
		params.SetStripeAccount(p.account)
	}
	if req.CustomerID != "" {
		params.Customer = stripe.String(req.CustomerID)
	}
	if req.Locale != "" {
		params.Locale = stripe.String(strings.ReplaceAll(strings.ToLower(req.Locale), "_", "-"))
	}
	if len(req.Metadata) > 0 {
		params.Metadata = make(map[string]string, len(req.Metadata))
		for k, v := range req.Metadata {
			params.Metadata[k] = v
		}
	}

	if req.AllowPromotion {
		params.AllowPromotionCodes = stripe.Bool(true)
	}

	lineItems := make([]*stripe.CheckoutSessionLineItemParams, 0, len(req.Items))
	for _, item := range req.Items {
		line := &stripe.CheckoutSessionLineItemParams{
			Quantity: stripe.Int64(max64(item.Quantity, 1)),
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency:   stripe.String(strings.ToLower(defaultString(item.Currency, req.Currency))),
				UnitAmount: stripe.Int64(item.Amount),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name: stripe.String(item.Name),
				},
			},
		}
		if item.Description != "" {
			line.PriceData.ProductData.Description = stripe.String(item.Description)
		}
		if item.SKU != "" {
			line.PriceData.ProductData.Metadata = map[string]string{
				"sku": item.SKU,
			}
		}
		lineItems = append(lineItems, line)
	}

	if len(lineItems) == 0 {
		lineItems = append(lineItems, &stripe.CheckoutSessionLineItemParams{
			Quantity: stripe.Int64(1),
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency:   stripe.String(strings.ToLower(req.Currency)),
				UnitAmount: stripe.Int64(req.Amount),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name: stripe.String("Order"),
				},
			},
		})
	}

	params.LineItems = lineItems
	params.PaymentIntentData = &stripe.CheckoutSessionPaymentIntentDataParams{}
	if len(req.Metadata) > 0 {
		params.PaymentIntentData.Metadata = make(map[string]string, len(req.Metadata))
		for k, v := range req.Metadata {
			params.PaymentIntentData.Metadata[k] = v
		}
	}

	session, err := p.api.sessions.New(params)
	if err != nil {
		return CheckoutSession{}, fmt.Errorf("stripe: create checkout session: %w", err)
	}

	intentID := ""
	if session.PaymentIntent != nil {
		intentID = session.PaymentIntent.ID
	}

	p.logger(ctx, "payments.stripe.session.created", map[string]any{
		"sessionId":     session.ID,
		"paymentIntent": intentID,
		"currency":      session.Currency,
	})

	expiresAt := p.clock().Add(30 * time.Minute)
	if session.ExpiresAt != 0 {
		expiresAt = time.Unix(session.ExpiresAt, 0).UTC()
	}

	raw := map[string]any{}
	if data, err := json.Marshal(session); err == nil {
		_ = json.Unmarshal(data, &raw)
	} else {
		raw["session"] = session
	}

	return CheckoutSession{
		ID:           session.ID,
		Provider:     "stripe",
		ClientSecret: session.ClientSecret,
		RedirectURL:  session.URL,
		IntentID:     intentID,
		ExpiresAt:    expiresAt,
		Raw:          raw,
	}, nil
}

// Confirm confirms a Stripe Payment Intent.
func (p *StripeProvider) Confirm(ctx context.Context, req ConfirmRequest) (PaymentDetails, error) {
	if p == nil {
		return PaymentDetails{}, errors.New("stripe: provider is nil")
	}
	params := &stripe.PaymentIntentConfirmParams{}
	params.Context = ctx
	if key := strings.TrimSpace(req.IdempotencyKey); key != "" {
		params.SetIdempotencyKey(key)
	}
	if p.account != "" {
		params.SetStripeAccount(p.account)
	}
	if len(req.Metadata) > 0 {
		params.Metadata = make(map[string]string, len(req.Metadata))
		for k, v := range req.Metadata {
			params.Metadata[k] = v
		}
	}
	intent, err := p.api.intents.Confirm(req.IntentID, params)
	if err != nil {
		return PaymentDetails{}, fmt.Errorf("stripe: confirm payment intent: %w", err)
	}
	p.logger(ctx, "payments.stripe.intent.confirmed", map[string]any{
		"paymentIntent": intent.ID,
		"status":        intent.Status,
	})
	return stripePaymentDetails(intent), nil
}

// Capture captures a Stripe Payment Intent.
func (p *StripeProvider) Capture(ctx context.Context, req CaptureRequest) (PaymentDetails, error) {
	if p == nil {
		return PaymentDetails{}, errors.New("stripe: provider is nil")
	}
	params := &stripe.PaymentIntentCaptureParams{}
	params.Context = ctx
	if key := strings.TrimSpace(req.IdempotencyKey); key != "" {
		params.SetIdempotencyKey(key)
	}
	if p.account != "" {
		params.SetStripeAccount(p.account)
	}
	if req.Amount != nil {
		params.AmountToCapture = stripe.Int64(*req.Amount)
	}
	intent, err := p.api.intents.Capture(req.IntentID, params)
	if err != nil {
		return PaymentDetails{}, fmt.Errorf("stripe: capture payment intent: %w", err)
	}
	p.logger(ctx, "payments.stripe.intent.captured", map[string]any{
		"paymentIntent":  intent.ID,
		"amountReceived": intent.AmountReceived,
	})
	return stripePaymentDetails(intent), nil
}

// Refund creates a refund for the provided Payment Intent.
func (p *StripeProvider) Refund(ctx context.Context, req RefundRequest) (PaymentDetails, error) {
	if p == nil {
		return PaymentDetails{}, errors.New("stripe: provider is nil")
	}
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(req.IntentID),
	}
	params.Context = ctx
	if key := strings.TrimSpace(req.IdempotencyKey); key != "" {
		params.SetIdempotencyKey(key)
	}
	if p.account != "" {
		params.SetStripeAccount(p.account)
	}
	if req.Amount != nil {
		params.Amount = stripe.Int64(*req.Amount)
	}
	if reason := mapStripeRefundReason(req.Reason); reason != "" {
		params.Reason = stripe.String(reason)
	}
	if len(req.Metadata) > 0 {
		params.Metadata = make(map[string]string, len(req.Metadata))
		for k, v := range req.Metadata {
			params.Metadata[k] = v
		}
	}
	if _, err := p.api.refunds.New(params); err != nil {
		return PaymentDetails{}, fmt.Errorf("stripe: refund payment intent: %w", err)
	}
	p.logger(ctx, "payments.stripe.intent.refunded", map[string]any{
		"paymentIntent": req.IntentID,
	})
	return p.LookupPayment(ctx, LookupRequest{IntentID: req.IntentID})
}

// LookupPayment retrieves a Stripe Payment Intent.
func (p *StripeProvider) LookupPayment(ctx context.Context, req LookupRequest) (PaymentDetails, error) {
	if p == nil {
		return PaymentDetails{}, errors.New("stripe: provider is nil")
	}
	params := &stripe.PaymentIntentParams{}
	params.Context = ctx
	if p.account != "" {
		params.SetStripeAccount(p.account)
	}
	intent, err := p.api.intents.Get(req.IntentID, params)
	if err != nil {
		return PaymentDetails{}, fmt.Errorf("stripe: lookup payment intent: %w", err)
	}
	return stripePaymentDetails(intent), nil
}

func stripePaymentDetails(intent *stripe.PaymentIntent) PaymentDetails {
	if intent == nil {
		return PaymentDetails{}
	}

	status := StatusPending
	switch intent.Status {
	case stripe.PaymentIntentStatusSucceeded:
		status = StatusSucceeded
	case stripe.PaymentIntentStatusCanceled:
		status = StatusFailed
	case stripe.PaymentIntentStatusRequiresPaymentMethod, stripe.PaymentIntentStatusProcessing, stripe.PaymentIntentStatusRequiresAction, stripe.PaymentIntentStatusRequiresConfirmation, stripe.PaymentIntentStatusRequiresCapture:
		status = StatusPending
	}

	var capturedAt *time.Time
	var refundedAt *time.Time
	captured := intent.Status == stripe.PaymentIntentStatusSucceeded

	if charge := intent.LatestCharge; charge != nil {
		if charge.Paid || charge.Captured {
			t := time.Unix(charge.Created, 0).UTC()
			capturedAt = &t
			captured = true
		}
		if charge.Refunded || charge.AmountRefunded > 0 {
			t := time.Unix(charge.Created, 0).UTC()
			refundedAt = &t
			if charge.AmountRefunded >= charge.Amount && charge.Amount > 0 {
				status = StatusRefunded
			}
		}
	}

	if intent.Status == stripe.PaymentIntentStatusSucceeded && status != StatusRefunded {
		status = StatusSucceeded
	}

	currency := strings.ToUpper(string(intent.Currency))
	if currency == "" && intent.LatestCharge != nil {
		currency = strings.ToUpper(string(intent.LatestCharge.Currency))
	}

	raw := map[string]any{}
	if data, err := json.Marshal(intent); err == nil {
		_ = json.Unmarshal(data, &raw)
	} else {
		raw["payment_intent"] = intent
	}

	return PaymentDetails{
		Provider:   "stripe",
		IntentID:   intent.ID,
		Status:     status,
		Amount:     intent.Amount,
		Currency:   currency,
		Captured:   captured,
		CapturedAt: capturedAt,
		RefundedAt: refundedAt,
		Raw:        raw,
	}
}

func mapStripeRefundReason(reason string) string {
	switch strings.ToLower(strings.TrimSpace(reason)) {
	case string(stripe.RefundReasonDuplicate):
		return string(stripe.RefundReasonDuplicate)
	case string(stripe.RefundReasonFraudulent):
		return string(stripe.RefundReasonFraudulent)
	case string(stripe.RefundReasonRequestedByCustomer):
		return string(stripe.RefundReasonRequestedByCustomer)
	default:
		return ""
	}
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
