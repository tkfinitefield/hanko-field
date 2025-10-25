package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	handlersPkg "finitefield.org/hanko-web/internal/handlers"
	mw "finitefield.org/hanko-web/internal/middleware"
	"finitefield.org/hanko-web/internal/nav"
)

// CheckoutAddressHandler renders the shipping/billing address selection page.
func CheckoutAddressHandler(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	sess := mw.GetSession(r)
	view := buildCheckoutAddressView(lang, r.URL.Query(), sess.Checkout.Addresses, sess.Checkout.ShippingAddressID, sess.Checkout.BillingAddressID)

	title := i18nOrDefault(lang, "checkout.address.title", "Checkout · Addresses")
	desc := i18nOrDefault(lang, "checkout.address.desc", "Confirm shipping and billing addresses before picking a shipping method.")

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

	renderPage(w, r, "checkout_address", vm)
}

// CheckoutAddressListFrag re-renders either the shipping or billing list fragment.
func CheckoutAddressListFrag(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	sess := mw.GetSession(r)
	view := buildCheckoutAddressView(lang, r.URL.Query(), sess.Checkout.Addresses, sess.Checkout.ShippingAddressID, sess.Checkout.BillingAddressID)

	kind := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("kind")))
	section := view.ShippingSection
	if kind == "billing" {
		section = view.BillingSection
	}

	data := CheckoutAddressListView{
		Lang:      lang,
		Section:   section,
		Addresses: view.Addresses,
	}
	renderTemplate(w, r, "frag_checkout_address_list", data)
}

// CheckoutAddressFormModal renders or processes the address modal form.
func CheckoutAddressFormModal(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	sess := mw.GetSession(r)
	switch r.Method {
	case http.MethodGet:
		form := buildCheckoutAddressFormView(lang, r.URL.Query(), sess, r.URL.Query().Get("id"))
		renderTemplate(w, r, "frag_checkout_address_form", form)
		return
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		input := parseCheckoutAddressForm(r.Form)
		errors := validateCheckoutAddressForm(input, lang)
		if len(errors) > 0 {
			form := buildCheckoutAddressFormViewFromInput(lang, input)
			form.Errors = errors
			renderTemplate(w, r, "frag_checkout_address_form", form)
			return
		}
		saved := mw.SessionAddress{
			ID:         input.ID,
			Label:      input.Label,
			Recipient:  input.Recipient,
			Company:    input.Company,
			Department: input.Department,
			Line1:      input.Line1,
			Line2:      input.Line2,
			City:       input.City,
			Region:     input.Region,
			Postal:     input.Postal,
			Country:    strings.ToUpper(input.Country),
			Phone:      input.Phone,
			Kind:       input.Kind,
			Notes:      input.Notes,
			CreatedAt:  time.Now().UTC(),
		}
		if saved.ID == "" {
			saved.ID = newSessionAddressID()
		}
		upsertSessionAddress(sess, saved)
		applyAddressSelectionForKind(sess, saved.ID, input.Kind)

		payload := map[string]any{
			"checkout:address:saved": map[string]string{
				"id":   saved.ID,
				"kind": input.Kind,
			},
		}
		if raw, err := json.Marshal(payload); err == nil {
			w.Header().Set("HX-Trigger", string(raw))
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<div class="p-6 text-center text-sm text-emerald-700">` +
			i18nOrDefault(lang, "checkout.address.saved", "Address saved. One moment...") +
			`</div>`))
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// CheckoutAddressSubmitHandler persists selected shipping/billing addresses and redirects to shipping step.
func CheckoutAddressSubmitHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	lang := mw.Lang(r)
	sess := mw.GetSession(r)

	shippingID := strings.TrimSpace(r.FormValue("shipping_id"))
	billingID := strings.TrimSpace(r.FormValue("billing_id"))

	addresses := mergeCheckoutAddresses(lang, sess.Checkout.Addresses)
	shippingAddr := findCheckoutAddress(addresses, shippingID)
	billingAddr := findCheckoutAddress(addresses, billingID)

	if shippingAddr == nil || billingAddr == nil {
		renderCheckoutActionError(w, r, lang, i18nOrDefault(lang, "checkout.address.error.select", "Select both shipping and billing addresses."))
		return
	}

	sess.Checkout.ShippingAddressID = shippingAddr.ID
	sess.Checkout.BillingAddressID = billingAddr.ID
	sess.MarkDirty()

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/checkout/shipping")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, "/checkout/shipping", http.StatusSeeOther)
}

func buildCheckoutAddressFormView(lang string, q url.Values, sess *mw.SessionData, id string) CheckoutAddressFormView {
	kind := strings.ToLower(strings.TrimSpace(q.Get("kind")))
	if kind == "" {
		kind = "shipping"
	}
	form := CheckoutAddressFormView{
		Lang:        lang,
		Mode:        "new",
		Title:       i18nOrDefault(lang, "checkout.address.modal.new", "Add address"),
		Subtitle:    i18nOrDefault(lang, "checkout.address.modal.copy", "Saved addresses sync across shipping and billing."),
		Kind:        kind,
		AddressID:   "",
		Values:      map[string]string{"country": strings.ToUpper(q.Get("country"))},
		Countries:   cartCountryOptions(lang, strings.ToUpper(q.Get("country"))),
		Prefectures: checkoutPrefectureOptions(lang),
		KindOptions: checkoutKindOptions(lang),
	}
	if id != "" {
		all := mergeCheckoutAddresses(lang, sess.Checkout.Addresses)
		if addr := findCheckoutAddress(all, id); addr != nil {
			form.Mode = "edit"
			form.Title = i18nOrDefault(lang, "checkout.address.modal.edit", "Edit address")
			form.AddressID = addr.ID
			form.Values = checkoutAddressToValues(addr)
			form.Kind = detectKindFromAddress(addr, kind)
		}
	}
	return form
}

func buildCheckoutAddressFormViewFromInput(lang string, input checkoutAddressFormInput) CheckoutAddressFormView {
	mode := "new"
	if input.ID != "" {
		mode = "edit"
	}
	return CheckoutAddressFormView{
		Lang:        lang,
		Mode:        mode,
		Title:       i18nOrDefault(lang, "checkout.address.modal.new", "Add address"),
		Subtitle:    i18nOrDefault(lang, "checkout.address.modal.copy", "Saved addresses sync across shipping and billing."),
		Kind:        input.Kind,
		AddressID:   input.ID,
		Values:      input.toValues(),
		Countries:   cartCountryOptions(lang, strings.ToUpper(input.Country)),
		Prefectures: checkoutPrefectureOptions(lang),
		KindOptions: checkoutKindOptions(lang),
	}
}

func checkoutAddressToValues(addr *CheckoutAddress) map[string]string {
	return map[string]string{
		"id":         addr.ID,
		"label":      addr.Label,
		"recipient":  addr.Recipient,
		"company":    addr.Company,
		"department": addr.Department,
		"line1":      addr.Line1,
		"line2":      addr.Line2,
		"city":       addr.City,
		"region":     addr.Region,
		"postal":     addr.PostalCode,
		"country":    addr.Country,
		"phone":      addr.Phone,
		"notes":      addr.Instructions,
	}
}

type checkoutAddressFormInput struct {
	ID         string
	Label      string
	Recipient  string
	Company    string
	Department string
	Line1      string
	Line2      string
	City       string
	Region     string
	Postal     string
	Country    string
	Phone      string
	Kind       string
	Notes      string
}

func (in checkoutAddressFormInput) toValues() map[string]string {
	return map[string]string{
		"id":         in.ID,
		"label":      in.Label,
		"recipient":  in.Recipient,
		"company":    in.Company,
		"department": in.Department,
		"line1":      in.Line1,
		"line2":      in.Line2,
		"city":       in.City,
		"region":     in.Region,
		"postal":     in.Postal,
		"country":    in.Country,
		"phone":      in.Phone,
		"notes":      in.Notes,
	}
}

func parseCheckoutAddressForm(values url.Values) checkoutAddressFormInput {
	kind := strings.ToLower(strings.TrimSpace(values.Get("kind")))
	if kind == "" {
		kind = "shipping"
	}
	return checkoutAddressFormInput{
		ID:         strings.TrimSpace(values.Get("id")),
		Label:      strings.TrimSpace(values.Get("label")),
		Recipient:  strings.TrimSpace(values.Get("recipient")),
		Company:    strings.TrimSpace(values.Get("company")),
		Department: strings.TrimSpace(values.Get("department")),
		Line1:      strings.TrimSpace(values.Get("line1")),
		Line2:      strings.TrimSpace(values.Get("line2")),
		City:       strings.TrimSpace(values.Get("city")),
		Region:     strings.TrimSpace(values.Get("region")),
		Postal:     strings.TrimSpace(values.Get("postal")),
		Country:    strings.ToUpper(strings.TrimSpace(values.Get("country"))),
		Phone:      strings.TrimSpace(values.Get("phone")),
		Kind:       kind,
		Notes:      strings.TrimSpace(values.Get("notes")),
	}
}

func validateCheckoutAddressForm(input checkoutAddressFormInput, lang string) map[string]string {
	errors := map[string]string{}
	if input.Label == "" {
		errors["label"] = i18nOrDefault(lang, "checkout.address.error.label", "Enter a label so teammates recognize it.")
	}
	if input.Recipient == "" {
		errors["recipient"] = i18nOrDefault(lang, "checkout.address.error.recipient", "Recipient is required.")
	}
	if input.Line1 == "" {
		errors["line1"] = i18nOrDefault(lang, "checkout.address.error.line1", "Street address is required.")
	}
	if input.City == "" {
		errors["city"] = i18nOrDefault(lang, "checkout.address.error.city", "City is required.")
	}
	if input.Postal == "" {
		errors["postal"] = i18nOrDefault(lang, "checkout.address.error.postal", "Postal code is required.")
	}
	if input.Country == "" {
		errors["country"] = i18nOrDefault(lang, "checkout.address.error.country", "Select a country.")
	}
	switch input.Kind {
	case "shipping", "billing", "both":
	default:
		errors["kind"] = i18nOrDefault(lang, "checkout.address.error.kind", "Choose how this address should be used.")
	}
	return errors
}

func detectKindFromAddress(addr *CheckoutAddress, fallback string) string {
	switch addr.Kind {
	case "shipping", "billing", "both":
		return addr.Kind
	}
	if fallback != "" {
		return fallback
	}
	return "shipping"
}

func upsertSessionAddress(sess *mw.SessionData, addr mw.SessionAddress) {
	addr.CreatedAt = addr.CreatedAt.UTC()
	updated := false
	for i := range sess.Checkout.Addresses {
		if sess.Checkout.Addresses[i].ID == addr.ID {
			sess.Checkout.Addresses[i] = addr
			updated = true
			break
		}
	}
	if !updated {
		sess.Checkout.Addresses = append([]mw.SessionAddress{addr}, sess.Checkout.Addresses...)
		if len(sess.Checkout.Addresses) > 6 {
			sess.Checkout.Addresses = sess.Checkout.Addresses[:6]
		}
	}
	sess.MarkDirty()
}

func applyAddressSelectionForKind(sess *mw.SessionData, id, kind string) {
	switch kind {
	case "billing":
		sess.Checkout.BillingAddressID = id
	case "both":
		sess.Checkout.ShippingAddressID = id
		sess.Checkout.BillingAddressID = id
	default:
		sess.Checkout.ShippingAddressID = id
	}
	sess.MarkDirty()
}

func renderCheckoutActionError(w http.ResponseWriter, r *http.Request, lang, msg string) {
	if r.Header.Get("HX-Request") == "true" {
		data := map[string]any{
			"Tone":  "error",
			"Title": msg,
			"Body":  "",
			"Icon":  "exclamation-triangle",
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		renderTemplate(w, r, "c_inline_alert", data)
		return
	}
	http.Redirect(w, r, "/checkout/address?status=missing_selection", http.StatusSeeOther)
}
