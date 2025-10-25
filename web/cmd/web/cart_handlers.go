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

// CartHandler renders the cart page.
func CartHandler(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	view := buildCartView(lang, r.URL.Query())
	title := i18nOrDefault(lang, "cart.title", "Cart and engraving review")
	desc := i18nOrDefault(lang, "cart.description", "Review each seal, apply promo codes, and lock in your fulfillment estimate.")

	vm := handlersPkg.PageData{
		Title:       title,
		Lang:        lang,
		Path:        r.URL.Path,
		Nav:         nav.Build(r.URL.Path),
		Breadcrumbs: nav.Breadcrumbs(r.URL.Path),
		Analytics:   handlersPkg.LoadAnalyticsFromEnv(),
		Cart:        view,
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

	renderPage(w, r, "cart", vm)
}

// CartTableFrag renders the line items table fragment.
func CartTableFrag(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	view := buildCartView(lang, r.URL.Query())
	push := "/cart"
	if view.Query != "" {
		push = push + "?" + view.Query
	}
	w.Header().Set("HX-Push-Url", push)
	renderTemplate(w, r, "frag_cart_table", view)
}

// CartEstimateFrag renders the summary + estimator fragment.
func CartEstimateFrag(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	view := buildCartView(lang, r.URL.Query())
	data := map[string]any{
		"Lang":     lang,
		"Estimate": view.Estimate,
		"Promo":    view.Promo,
		"Shipping": view.Shipping,
	}
	renderTemplate(w, r, "frag_cart_estimate", data)
}

// CartPromoModal renders the promo modal contents.
func CartPromoModal(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	view := buildCartView(lang, r.URL.Query())
	data := map[string]any{
		"Lang":  lang,
		"Promo": view.Promo,
	}
	renderTemplate(w, r, "frag_cart_promo_modal", data)
}

// CartPromoApplyHandler processes promo submissions and instructs the frontend to refresh estimates.
func CartPromoApplyHandler(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	rawCode := strings.ToUpper(strings.TrimSpace(r.FormValue("promo_code")))
	valid := isValidPromo(rawCode)

	var q url.Values
	if valid {
		q = cloneQuery(r.URL.Query())
		if q == nil {
			q = url.Values{}
		}
		q.Set("promo", rawCode)
	} else {
		q = r.URL.Query()
	}
	view := buildCartView(lang, q)

	data := map[string]any{
		"Lang":       lang,
		"Promo":      view.Promo,
		"Attempt":    rawCode,
		"StatusTone": "",
		"StatusText": "",
	}

	if valid {
		msg := ""
		if lang == "ja" {
			msg = "コードが適用されました。"
		} else {
			msg = "Promo applied."
		}
		data["StatusTone"] = "success"
		data["StatusText"] = msg

		payload := map[string]any{
			"cart:promo-applied": map[string]string{
				"code":    rawCode,
				"label":   view.Promo.ActiveLabel,
				"message": view.Promo.ActiveMessage,
			},
		}
		if raw, err := json.Marshal(payload); err == nil {
			w.Header().Set("HX-Trigger", string(raw))
		}
	} else {
		msg := ""
		if lang == "ja" {
			msg = "コードが見つかりません。"
		} else {
			msg = "We couldn’t find that code."
		}
		data["StatusTone"] = "error"
		data["StatusText"] = msg
	}

	renderTemplate(w, r, "frag_cart_promo_modal", data)
}
