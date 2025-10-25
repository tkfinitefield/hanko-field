package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"finitefield.org/hanko-web/internal/format"
	mw "finitefield.org/hanko-web/internal/middleware"
)

// CheckoutPaymentView drives the `/checkout/payment` page.
type CheckoutPaymentView struct {
	Steps            []CheckoutStep
	Alerts           []CheckoutInlineAlert
	Summary          CheckoutSummary
	SavedMethods     []CheckoutSavedMethod
	SelectedMethodID string
	PaymentState     CheckoutPaymentState
	Wallets          []CheckoutPaymentWallet
	TrustBadges      []CheckoutTrustBadge
	Support          CheckoutSupportCard
	BackURL          string
	ContinueURL      string
	ReturnURL        string
	CancelURL        string
	LastUpdated      time.Time
}

// CheckoutSavedMethod represents a stored payment instrument.
type CheckoutSavedMethod struct {
	ID        string
	Label     string
	Subtitle  string
	Brand     string
	Last4     string
	Expiry    string
	Badge     string
	BadgeTone string
	Default   bool
	Editable  bool
	UpdatedAt time.Time
}

// CheckoutPaymentState captures the last-known PSP session metadata.
type CheckoutPaymentState struct {
	Provider       string
	SessionID      string
	Status         string
	LastError      string
	DeclineCode    string
	RedirectURL    string
	ClientSecret   string
	PublishableKey string
	UpdatedAt      time.Time
}

// CheckoutTrustBadge displays security assurances below the form.
type CheckoutTrustBadge struct {
	Icon  string
	Label string
	Body  string
}

// CheckoutPaymentWallet lists available wallet options for the PSP element.
type CheckoutPaymentWallet struct {
	ID          string
	Label       string
	Icon        string
	Description string
}

func buildCheckoutPaymentView(lang string, q url.Values, sess *mw.SessionData, baseURL string) CheckoutPaymentView {
	addresses := mergeCheckoutAddresses(lang, sess.Checkout.Addresses)
	shippingAddr := findCheckoutAddress(addresses, sess.Checkout.ShippingAddressID)

	country := strings.ToUpper(strings.TrimSpace(q.Get("country")))
	postal := strings.TrimSpace(q.Get("postal"))
	if shippingAddr != nil {
		if country == "" {
			country = strings.ToUpper(strings.TrimSpace(shippingAddr.Country))
		}
		if postal == "" {
			postal = strings.TrimSpace(shippingAddr.PostalCode)
		}
	}
	if country == "" {
		country = "JP"
	}
	if postal == "" {
		postal = defaultPostalForCountry(country)
	}

	method := normalizeCartShippingMethod(sess.Checkout.ShippingMethodID)
	promo := normalizePromoCode(q.Get("promo"))

	query := url.Values{}
	query.Set("country", country)
	if postal != "" {
		query.Set("postal", postal)
	}
	if method != "" {
		query.Set("method", method)
	}
	if promo != "" {
		query.Set("promo", promo)
	}

	cartView := buildCartView(lang, query)
	summary := CheckoutSummary{
		Estimate: cartView.Estimate,
		Items: []CheckoutSummaryItem{
			{Label: i18nOrDefault(lang, "checkout.summary.items", "Items"), Value: fmt.Sprintf("%d", cartView.Estimate.ItemsCount)},
			{Label: i18nOrDefault(lang, "checkout.summary.weight", "Packed weight"), Value: cartView.Estimate.WeightDisplay},
			{Label: i18nOrDefault(lang, "checkout.summary.method", "Method"), Value: cartView.Estimate.MethodLabel},
		},
		Notes: []string{
			fmt.Sprintf("%s: %s", i18nOrDefault(lang, "checkout.payment.due_today", "Due today"), format.FmtCurrency(cartView.Estimate.Total, cartView.Estimate.Currency, lang)),
			i18nOrDefault(lang, "checkout.payment.invoice_note", "We’ll email an itemized receipt once the payment intent succeeds."),
		},
		ShippingAddress: shippingAddr,
	}

	saved := mockCheckoutSavedMethods(lang)
	requestedMethod := strings.TrimSpace(q.Get("method_id"))
	selected := selectCheckoutPaymentMethod(saved, requestedMethod, sess.Checkout.PaymentMethodID)
	if selected != "" && sess.Checkout.PaymentMethodID != selected {
		sess.Checkout.PaymentMethodID = selected
		sess.MarkDirty()
	}

	provider := strings.TrimSpace(q.Get("provider"))
	if provider == "" {
		provider = "stripe"
	}
	state := CheckoutPaymentState{
		Provider:    provider,
		SessionID:   strings.TrimSpace(q.Get("session_id")),
		Status:      strings.TrimSpace(q.Get("status")),
		LastError:   strings.TrimSpace(q.Get("error")),
		DeclineCode: strings.TrimSpace(q.Get("decline")),
		UpdatedAt:   time.Now(),
	}
	if state.Status == "" {
		state.Status = "idle"
	}

	view := CheckoutPaymentView{
		Steps:            cartSteps(lang, "payment"),
		Summary:          summary,
		SavedMethods:     saved,
		SelectedMethodID: selected,
		PaymentState:     state,
		Wallets:          buildCheckoutWallets(lang),
		TrustBadges:      buildCheckoutTrustBadges(lang),
		Support: CheckoutSupportCard{
			Title:    i18nOrDefault(lang, "checkout.payment.support.title", "Need invoice terms or PO assistance?"),
			Body:     i18nOrDefault(lang, "checkout.payment.support.body", "Finance ops can issue pro-forma invoices, PO references, and receipt copies within one business day."),
			CTALabel: i18nOrDefault(lang, "checkout.support.cta", "Chat with concierge"),
			CTAHref:  "mailto:support@hanko-field.example",
		},
		BackURL:     "/checkout/shipping",
		ContinueURL: "/checkout/review",
		ReturnURL:   checkoutRedirectURL(baseURL, "success", provider),
		CancelURL:   checkoutRedirectURL(baseURL, "cancelled", provider),
		LastUpdated: cartView.Estimate.UpdatedAt,
	}
	view.Alerts = buildCheckoutPaymentAlerts(lang, view, state.Status, state.LastError)
	return view
}

func selectCheckoutPaymentMethod(methods []CheckoutSavedMethod, requested, current string) string {
	if requested != "" && hasSavedMethod(methods, requested) {
		return requested
	}
	if current != "" && hasSavedMethod(methods, current) {
		return current
	}
	for _, m := range methods {
		if m.Default {
			return m.ID
		}
	}
	if len(methods) > 0 {
		return methods[0].ID
	}
	return ""
}

func hasSavedMethod(methods []CheckoutSavedMethod, id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	for _, m := range methods {
		if m.ID == id {
			return true
		}
	}
	return false
}

func buildCheckoutPaymentAlerts(lang string, view CheckoutPaymentView, status, errMsg string) []CheckoutInlineAlert {
	var alerts []CheckoutInlineAlert
	if view.Summary.ShippingAddress == nil {
		alerts = append(alerts, CheckoutInlineAlert{
			Tone:  "error",
			Icon:  "exclamation-triangle",
			Title: i18nOrDefault(lang, "checkout.payment.alert.no_address", "Select a shipping address to continue."),
			Body:  i18nOrDefault(lang, "checkout.payment.alert.no_address.body", "Add or confirm an address in the previous step so taxes and duties stay accurate."),
		})
	}
	if strings.TrimSpace(view.Summary.Estimate.MethodLabel) == "" {
		alerts = append(alerts, CheckoutInlineAlert{
			Tone:  "error",
			Icon:  "shopping-bag",
			Title: i18nOrDefault(lang, "checkout.payment.alert.no_method", "Choose a shipping method before paying."),
			Body:  i18nOrDefault(lang, "checkout.payment.alert.no_method.body", "Return to the shipping step to confirm carrier and delivery window."),
		})
	}
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "cancelled":
		alerts = append(alerts, CheckoutInlineAlert{
			Tone:  "info",
			Icon:  "arrow-uturn-left",
			Title: i18nOrDefault(lang, "checkout.payment.alert.cancelled", "Payment was cancelled"),
			Body:  i18nOrDefault(lang, "checkout.payment.alert.cancelled.body", "You can restart the session below or pick another method."),
		})
	case "declined", "error":
		body := errMsg
		if body == "" {
			if lang == "ja" {
				body = "カード会社から支払いが拒否されました。別の手段をお試しください。"
			} else {
				body = "The processor declined this attempt. Try another method or contact your card issuer."
			}
		}
		alerts = append(alerts, CheckoutInlineAlert{
			Tone:  "error",
			Icon:  "x-circle",
			Title: i18nOrDefault(lang, "checkout.payment.alert.declined", "Payment declined"),
			Body:  body,
		})
	case "success", "confirmed", "paid":
		alerts = append(alerts, CheckoutInlineAlert{
			Tone:  "success",
			Icon:  "check-circle",
			Title: i18nOrDefault(lang, "checkout.payment.alert.success", "Payment confirmed"),
			Body:  i18nOrDefault(lang, "checkout.payment.alert.success.body", "We’ve captured your payment intent. Continue to review and final submission."),
		})
	}
	return alerts
}

func mockCheckoutSavedMethods(lang string) []CheckoutSavedMethod {
	now := time.Now()
	return []CheckoutSavedMethod{
		{
			ID:        "pm_card_visa_ops",
			Label:     i18nOrDefault(lang, "checkout.payment.method.ops", "Studio corporate card"),
			Subtitle:  "Visa •••• 4242 · exp 12/26",
			Brand:     "visa",
			Last4:     "4242",
			Expiry:    "12/26",
			Badge:     i18nOrDefault(lang, "checkout.payment.method.primary", "Default"),
			BadgeTone: "success",
			Default:   true,
			Editable:  true,
			UpdatedAt: now.Add(-6 * time.Hour),
		},
		{
			ID:        "pm_card_holdings",
			Label:     i18nOrDefault(lang, "checkout.payment.method.finance", "Holdings AMEX (Finance)"),
			Subtitle:  "Amex •••• 0005 · exp 04/27",
			Brand:     "amex",
			Last4:     "0005",
			Expiry:    "04/27",
			Badge:     i18nOrDefault(lang, "checkout.payment.method.requires_pin", "Requires MFA"),
			BadgeTone: "warning",
			Default:   false,
			Editable:  false,
			UpdatedAt: now.Add(-72 * time.Hour),
		},
	}
}

func buildCheckoutTrustBadges(lang string) []CheckoutTrustBadge {
	return []CheckoutTrustBadge{
		{Icon: "shield-check", Label: i18nOrDefault(lang, "checkout.payment.badge.tls", "TLS 1.3"), Body: i18nOrDefault(lang, "checkout.payment.badge.tls.body", "256-bit encryption")},
		{Icon: "layers", Label: "PCI DSS", Body: i18nOrDefault(lang, "checkout.payment.badge.pci", "Audited annually")},
		{Icon: "sparkles", Label: i18nOrDefault(lang, "checkout.payment.badge.auth", "3D Secure"), Body: i18nOrDefault(lang, "checkout.payment.badge.auth.body", "Risk-based SCA")},
	}
}

func buildCheckoutWallets(lang string) []CheckoutPaymentWallet {
	return []CheckoutPaymentWallet{
		{ID: "apple_pay", Label: "Apple Pay", Icon: "signal", Description: i18nOrDefault(lang, "checkout.payment.wallet.apple", "Works on Safari/iOS")},
		{ID: "google_pay", Label: "Google Pay", Icon: "arrows-right-left", Description: i18nOrDefault(lang, "checkout.payment.wallet.google", "Tap to approve via Android/Chrome")},
		{ID: "paypal", Label: "PayPal", Icon: "user-group", Description: i18nOrDefault(lang, "checkout.payment.wallet.paypal", "Corporate accounts supported")},
	}
}

func checkoutRedirectURL(base, status, provider string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		base = "https://example.com"
	}
	status = strings.TrimSpace(status)
	if status == "" {
		status = "success"
	}
	provider = strings.TrimSpace(provider)
	params := url.Values{}
	params.Set("status", status)
	if provider != "" {
		params.Set("provider", provider)
	}
	return base + "/checkout/payment?" + params.Encode()
}
