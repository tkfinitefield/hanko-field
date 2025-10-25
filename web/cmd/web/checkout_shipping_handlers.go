package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	handlersPkg "finitefield.org/hanko-web/internal/handlers"
	mw "finitefield.org/hanko-web/internal/middleware"
	"finitefield.org/hanko-web/internal/nav"
)

// CheckoutShippingHandler renders the shipping method selection step.
func CheckoutShippingHandler(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	sess := mw.GetSession(r)
	view := buildCheckoutShippingView(lang, r.URL.Query(), sess)

	title := i18nOrDefault(lang, "checkout.shipping.title", "Checkout · Shipping method")
	desc := i18nOrDefault(lang, "checkout.shipping.desc", "Compare carrier speeds and pick how we should dispatch your seals.")

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

	renderPage(w, r, "checkout_shipping", vm)
}

// CheckoutShippingTableFrag renders the shipping options + comparison fragment.
func CheckoutShippingTableFrag(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	sess := mw.GetSession(r)

	var q url.Values
	switch r.Method {
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		q = r.Form
	default:
		q = r.URL.Query()
	}

	view := buildCheckoutShippingView(lang, q, sess)
	push := "/checkout/shipping"
	if view.Options.Query != "" {
		push = push + "?" + view.Options.Query
	}
	w.Header().Set("HX-Push-Url", push)

	trigger := map[string]any{
		"checkout:shipping:refresh": map[string]string{
			"query":  view.Options.Query,
			"method": view.SelectedMethod,
		},
	}
	if raw, err := json.Marshal(trigger); err == nil {
		w.Header().Set("HX-Trigger", string(raw))
	}

	data := map[string]any{
		"Lang":     lang,
		"Options":  view.Options,
		"Selected": view.SelectedMethod,
	}
	renderTemplate(w, r, "frag_checkout_shipping_table", data)
}

// CheckoutShippingSummaryFrag refreshes the sidebar summary when estimates change.
func CheckoutShippingSummaryFrag(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	sess := mw.GetSession(r)
	view := buildCheckoutShippingView(lang, r.URL.Query(), sess)
	data := map[string]any{
		"Lang":    lang,
		"Summary": view.Summary,
		"Support": view.Support,
	}
	renderTemplate(w, r, "frag_checkout_summary", data)
}

// CheckoutShippingSubmitHandler persists the selected method then advances to payment.
func CheckoutShippingSubmitHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	lang := mw.Lang(r)
	sess := mw.GetSession(r)
	method := normalizeCartShippingMethod(r.FormValue("shipping_method"))

	view := buildCheckoutShippingView(lang, url.Values{
		"method": {method},
	}, sess)
	if view.ShippingAddress == nil {
		renderCheckoutActionError(w, r, lang, i18nOrDefault(lang, "checkout.shipping.error.address", "Select an address before choosing a method."))
		return
	}
	if !isShippingMethodSelectable(view.Options.Methods, method) {
		renderCheckoutActionError(w, r, lang, i18nOrDefault(lang, "checkout.shipping.error.method", "That method is unavailable for your destination."))
		return
	}

	sess.Checkout.ShippingMethodID = method
	sess.MarkDirty()

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/checkout/payment")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, "/checkout/payment", http.StatusSeeOther)
}

func isShippingMethodSelectable(options []CheckoutShippingOption, method string) bool {
	method = strings.TrimSpace(method)
	if method == "" {
		return false
	}
	for _, opt := range options {
		if opt.ID == method && !opt.Disabled {
			return true
		}
	}
	return false
}
