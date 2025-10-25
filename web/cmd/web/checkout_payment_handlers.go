package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"finitefield.org/hanko-web/internal/checkout"
	handlersPkg "finitefield.org/hanko-web/internal/handlers"
	mw "finitefield.org/hanko-web/internal/middleware"
	"finitefield.org/hanko-web/internal/nav"
)

var checkoutAPIClient = checkout.NewClient(os.Getenv("HANKO_WEB_API_BASE_URL"))

// CheckoutPaymentHandler renders the PSP payment step.
func CheckoutPaymentHandler(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	sess := mw.GetSession(r)
	baseURL := siteBaseURL(r)
	view := buildCheckoutPaymentView(lang, r.URL.Query(), sess, baseURL)

	title := i18nOrDefault(lang, "checkout.payment.title", "Checkout · Payment")
	desc := i18nOrDefault(lang, "checkout.payment.desc", "Securely enter payment details or jump to Stripe to complete checkout.")

	vm := handlersPkg.PageData{
		Title:       title,
		Lang:        lang,
		Path:        r.URL.Path,
		Nav:         nav.Build(r.URL.Path),
		Breadcrumbs: nav.Breadcrumbs(r.URL.Path),
		Analytics:   handlersPkg.LoadAnalyticsFromEnv(),
		Checkout:    view,
	}

	brand := i18nOrDefault(lang, "brand.name", "Hanko Field")
	vm.SEO.Title = title + " | " + brand
	vm.SEO.Description = desc
	vm.SEO.Canonical = absoluteURL(r)
	vm.SEO.OG.URL = vm.SEO.Canonical
	vm.SEO.OG.SiteName = brand
	vm.SEO.OG.Title = vm.SEO.Title
	vm.SEO.OG.Description = vm.SEO.Description
	vm.SEO.OG.Type = "website"
	vm.SEO.Twitter.Card = "summary_large_image"
	vm.SEO.Alternates = buildAlternates(r)
	vm.SEO.Robots = "noindex, nofollow"

	renderPage(w, r, "checkout_payment", vm)
}

// CheckoutPaymentSessionHandler initiates a PSP session via the backend API.
func CheckoutPaymentSessionHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	lang := mw.Lang(r)
	provider := strings.TrimSpace(r.FormValue("provider"))
	if provider == "" {
		provider = "stripe"
	}

	req := checkout.SessionRequest{
		Provider:  provider,
		ReturnURL: checkoutRedirectURL(siteBaseURL(r), "success", provider),
		CancelURL: checkoutRedirectURL(siteBaseURL(r), "cancelled", provider),
		Locale:    lang,
	}

	resp, err := checkoutAPIClient.CreateSession(r.Context(), req)
	if err != nil {
		renderCheckoutPaymentStatus(w, r, lang, "error", paymentErrorTitle(lang), err.Error(), http.StatusBadGateway)
		return
	}

	payload := map[string]any{
		"checkout:payment:session": map[string]any{
			"sessionId":      resp.SessionID,
			"redirectUrl":    resp.URL,
			"clientSecret":   resp.ClientSecret,
			"publishableKey": resp.PublishableKey,
			"provider":       resp.Provider,
			"amount":         resp.Amount,
			"currency":       resp.Currency,
		},
	}
	if raw, err := json.Marshal(payload); err == nil {
		w.Header().Set("HX-Trigger", string(raw))
	}

	title := i18nOrDefault(lang, "checkout.payment.session.ready", "Stripe session ready")
	body := i18nOrDefault(lang, "checkout.payment.session.ready.body", "We’re redirecting you or loading the payment element.")
	renderCheckoutPaymentStatus(w, r, lang, "success", title, body, http.StatusOK)
}

// CheckoutPaymentConfirmHandler notifies the backend that payment completed so we can advance to review.
func CheckoutPaymentConfirmHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	lang := mw.Lang(r)
	sessionID := strings.TrimSpace(r.FormValue("session_id"))
	methodID := strings.TrimSpace(r.FormValue("method_id"))
	provider := strings.TrimSpace(r.FormValue("provider"))
	if sessionID == "" && methodID != "" {
		sessionID = "pm_" + methodID
	}
	if sessionID == "" {
		renderCheckoutPaymentStatus(w, r, lang, "error", paymentErrorTitle(lang), missingSessionCopy(lang), http.StatusUnprocessableEntity)
		return
	}

	resp, err := checkoutAPIClient.ConfirmSession(r.Context(), checkout.ConfirmRequest{
		SessionID: sessionID,
		Provider:  provider,
	})
	if err != nil {
		renderCheckoutPaymentStatus(w, r, lang, "error", paymentErrorTitle(lang), err.Error(), http.StatusBadGateway)
		return
	}

	nextURL := strings.TrimSpace(resp.NextURL)
	if nextURL == "" {
		nextURL = "/checkout/review"
	}

	payload := map[string]any{
		"checkout:payment:success": map[string]string{
			"orderId": resp.OrderID,
			"status":  resp.Status,
			"nextUrl": nextURL,
		},
	}
	if raw, err := json.Marshal(payload); err == nil {
		w.Header().Set("HX-Trigger", string(raw))
	}

	title := i18nOrDefault(lang, "checkout.payment.confirmed", "Payment confirmed")
	body := i18nOrDefault(lang, "checkout.payment.confirmed.body", "Taking you to the review step…")

	if r.Header.Get("HX-Request") != "true" {
		http.Redirect(w, r, nextURL, http.StatusSeeOther)
		return
	}

	renderCheckoutPaymentStatus(w, r, lang, "success", title, body, http.StatusOK)
}

func renderCheckoutPaymentStatus(w http.ResponseWriter, r *http.Request, lang, tone, title, body string, code int) {
	data := map[string]any{
		"Tone":  tone,
		"Title": title,
		"Body":  body,
		"Icon":  "information-circle",
	}
	if tone == "error" {
		data["Icon"] = "exclamation-triangle"
	} else if tone == "success" {
		data["Icon"] = "check-circle"
	}

	w.WriteHeader(code)
	renderTemplate(w, r, "c_inline_alert", data)
}

func paymentErrorTitle(lang string) string {
	if lang == "ja" {
		return "セッションを開始できません"
	}
	return "Unable to start session"
}

func missingSessionCopy(lang string) string {
	if lang == "ja" {
		return "決済セッションを特定できませんでした。再度お試しください。"
	}
	return "We couldn’t locate the payment session. Please try again."
}
