package payments

import (
	"context"
	"errors"
	"strings"

	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/client"
)

// PaymentMethodDetails captures PSP-sourced metadata for a payment instrument.
type PaymentMethodDetails struct {
	Token    string
	Brand    string
	Last4    string
	ExpMonth int
	ExpYear  int
}

// StripePaymentMethodVerifier retrieves payment method metadata from Stripe.
type StripePaymentMethodVerifier struct {
	api     stripePaymentMethodAPI
	account string
}

// NewStripePaymentMethodVerifier constructs a verifier using the provided configuration.
func NewStripePaymentMethodVerifier(cfg StripeProviderConfig) (*StripePaymentMethodVerifier, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" && (cfg.Clients == nil || cfg.Clients.paymentMethods == nil) {
		return nil, errors.New("stripe: api key is required")
	}

	var api stripePaymentMethodAPI
	if cfg.Clients != nil && cfg.Clients.paymentMethods != nil {
		api = cfg.Clients.paymentMethods
	} else {
		sc := client.New(apiKey, cfg.Backends)
		api = sc.PaymentMethods
	}
	if api == nil {
		return nil, errors.New("stripe: payment methods client is nil")
	}

	return &StripePaymentMethodVerifier{
		api:     api,
		account: strings.TrimSpace(cfg.AccountID),
	}, nil
}

// Lookup fetches metadata for the provided token from Stripe.
func (v *StripePaymentMethodVerifier) Lookup(ctx context.Context, token string) (PaymentMethodDetails, error) {
	if v == nil {
		return PaymentMethodDetails{}, errors.New("stripe: verifier is nil")
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return PaymentMethodDetails{}, errors.New("stripe: payment method token is required")
	}

	params := &stripe.PaymentMethodParams{}
	params.Context = ctx
	if v.account != "" {
		params.SetStripeAccount(v.account)
	}

	pm, err := v.api.Get(token, params)
	if err != nil {
		return PaymentMethodDetails{}, err
	}

	details := PaymentMethodDetails{
		Token: token,
	}
	if pm == nil {
		return details, nil
	}
	if trimmed := strings.TrimSpace(pm.ID); trimmed != "" {
		details.Token = trimmed
	}

	if pm.Type == stripe.PaymentMethodTypeCard && pm.Card != nil {
		details.Brand = strings.ToLower(string(pm.Card.Brand))
		details.Last4 = strings.TrimSpace(pm.Card.Last4)
		details.ExpMonth = int(pm.Card.ExpMonth)
		details.ExpYear = int(pm.Card.ExpYear)
	}

	return details, nil
}
