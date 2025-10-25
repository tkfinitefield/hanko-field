package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	mw "finitefield.org/hanko-web/internal/middleware"
)

// CheckoutAddressView drives the `/checkout/address` page.
type CheckoutAddressView struct {
	Steps     []CheckoutStep
	Alerts    []CheckoutInlineAlert
	Addresses []CheckoutAddress

	ShippingSection CheckoutAddressSection
	BillingSection  CheckoutAddressSection

	Summary CheckoutSummary
	Support CheckoutSupportCard

	AddressCount int
	ContinueURL  string
	BackURL      string
	LastUpdated  time.Time
}

// CheckoutAddressSection represents one selection rail (shipping or billing).
type CheckoutAddressSection struct {
	Kind        string
	Title       string
	Description string
	SelectedID  string
	EmptyCopy   string
	AddLabel    string
	AddKind     string
	Hint        string
	MirrorShip  bool
	LastUsed    time.Time
}

// CheckoutInlineAlert renders inline contextual alerts.
type CheckoutInlineAlert struct {
	Tone  string
	Title string
	Body  string
	Icon  string
}

// CheckoutSummary powers the sidebar summary.
type CheckoutSummary struct {
	Estimate        CartEstimate
	Items           []CheckoutSummaryItem
	Notes           []string
	ShippingAddress *CheckoutAddress
}

// CheckoutSummaryItem is a label/value row.
type CheckoutSummaryItem struct {
	Label string
	Value string
}

// CheckoutSupportCard highlights assistance options.
type CheckoutSupportCard struct {
	Title    string
	Body     string
	CTALabel string
	CTAHref  string
}

// CheckoutAddress models a saved address entry.
type CheckoutAddress struct {
	ID              string
	Label           string
	Recipient       string
	Company         string
	Department      string
	Line1           string
	Line2           string
	City            string
	Region          string
	PostalCode      string
	Country         string
	CountryLabel    string
	Phone           string
	Email           string
	Tags            []CheckoutBadge
	DefaultShipping bool
	DefaultBilling  bool
	Instructions    string
	Kind            string
	UpdatedAt       time.Time
	Editable        bool
}

// CheckoutBadge renders annotated metadata for an address.
type CheckoutBadge struct {
	Label string
	Tone  string
}

// CheckoutAddressListView is used by the address fragment template.
type CheckoutAddressListView struct {
	Lang      string
	Section   CheckoutAddressSection
	Addresses []CheckoutAddress
}

// CheckoutAddressFormView feeds the add/edit modal.
type CheckoutAddressFormView struct {
	Lang        string
	Mode        string
	Title       string
	Subtitle    string
	Kind        string
	AddressID   string
	Values      map[string]string
	Errors      map[string]string
	Countries   []CartOption
	Prefectures []CartOption
	KindOptions []CartOption
}

func buildCheckoutAddressView(lang string, q url.Values, sessionAddresses []mw.SessionAddress, sessionShipping, sessionBilling string) CheckoutAddressView {
	shippingOverride := strings.TrimSpace(q.Get("shipping"))
	billingOverride := strings.TrimSpace(q.Get("billing"))
	if shippingOverride != "" {
		sessionShipping = shippingOverride
	}
	if billingOverride != "" {
		sessionBilling = billingOverride
	}

	addresses := mergeCheckoutAddresses(lang, sessionAddresses)
	shippingID := ensureAddressSelection(addresses, sessionShipping, func(a CheckoutAddress) bool { return a.DefaultShipping })
	billingID := ensureAddressSelection(addresses, sessionBilling, func(a CheckoutAddress) bool { return a.DefaultBilling })
	if billingID == "" {
		billingID = shippingID
	}

	shippingAddr := findCheckoutAddress(addresses, shippingID)
	billingAddr := findCheckoutAddress(addresses, billingID)

	cartView := buildCartView(lang, q)

	shippingSection := CheckoutAddressSection{
		Kind:        "shipping",
		Title:       i18nOrDefault(lang, "checkout.address.shipping.title", "Shipping address"),
		Description: i18nOrDefault(lang, "checkout.address.shipping.desc", "Select the address where we should deliver and confirm customs paperwork."),
		SelectedID:  shippingID,
		EmptyCopy:   i18nOrDefault(lang, "checkout.address.empty", "Add a new address to continue."),
		AddLabel:    i18nOrDefault(lang, "checkout.address.add", "Add address"),
		AddKind:     "shipping",
		Hint:        i18nOrDefault(lang, "checkout.address.shipping.hint", "Studio pickup still requires an address for invoices."),
	}
	if shippingAddr != nil {
		shippingSection.LastUsed = shippingAddr.UpdatedAt
	}

	billingSection := CheckoutAddressSection{
		Kind:        "billing",
		Title:       i18nOrDefault(lang, "checkout.address.billing.title", "Billing address"),
		Description: i18nOrDefault(lang, "checkout.address.billing.desc", "Invoices and card verification will reference this entity."),
		SelectedID:  billingID,
		EmptyCopy:   i18nOrDefault(lang, "checkout.address.empty", "Add a new address to continue."),
		AddLabel:    i18nOrDefault(lang, "checkout.address.add", "Add address"),
		AddKind:     "billing",
		Hint:        i18nOrDefault(lang, "checkout.address.billing.hint", "Most teams reuse the shipping contact unless finance requires otherwise."),
		MirrorShip:  billingID != "" && billingID == shippingID,
	}
	if billingAddr != nil {
		billingSection.LastUsed = billingAddr.UpdatedAt
	}

	alerts := buildCheckoutAddressAlerts(lang, q)

	summary := CheckoutSummary{
		Estimate:        cartView.Estimate,
		ShippingAddress: shippingAddr,
		Items: []CheckoutSummaryItem{
			{Label: i18nOrDefault(lang, "checkout.summary.items", "Items"), Value: fmt.Sprintf("%d", cartView.Estimate.ItemsCount)},
			{Label: i18nOrDefault(lang, "checkout.summary.weight", "Packed weight"), Value: cartView.Estimate.WeightDisplay},
			{Label: i18nOrDefault(lang, "checkout.summary.method", "Method"), Value: cartView.Estimate.MethodLabel},
		},
		Notes: []string{
			fmt.Sprintf("%s: %s", i18nOrDefault(lang, "checkout.summary.eta", "ETA"), cartView.Estimate.ETA),
		},
	}

	view := CheckoutAddressView{
		Steps:           cartSteps(lang, "shipping"),
		Alerts:          alerts,
		Addresses:       addresses,
		ShippingSection: shippingSection,
		BillingSection:  billingSection,
		Summary:         summary,
		Support: CheckoutSupportCard{
			Title:    i18nOrDefault(lang, "checkout.support.title", "Need help with delivery paperwork?"),
			Body:     i18nOrDefault(lang, "checkout.support.body", "Our concierge team can prepare customs-ready invoices, HS codes, and banking letters."),
			CTALabel: i18nOrDefault(lang, "checkout.support.cta", "Chat with concierge"),
			CTAHref:  "mailto:support@hanko-field.example",
		},
		AddressCount: len(addresses),
		ContinueURL:  "/checkout/shipping",
		BackURL:      "/cart",
		LastUpdated:  time.Now(),
	}

	return view
}

func buildCheckoutAddressAlerts(lang string, q url.Values) []CheckoutInlineAlert {
	status := strings.TrimSpace(q.Get("status"))
	if status == "" {
		return nil
	}
	alert := CheckoutInlineAlert{Icon: "information-circle"}
	switch status {
	case "geocode_error":
		alert.Tone = "error"
		if lang == "ja" {
			alert.Title = "住所を確認してください"
			alert.Body = "入力された住所を正しくジオコーディングできませんでした。番地や郵便番号に漏れがないかご確認ください。"
		} else {
			alert.Title = "We couldn’t verify this address"
			alert.Body = "Double-check the postal code and municipality before continuing."
		}
	case "saved":
		alert.Tone = "success"
		if lang == "ja" {
			alert.Title = "住所を保存しました"
			alert.Body = "新しい配送先を登録し、選択肢に追加しています。"
		} else {
			alert.Title = "Address saved"
			alert.Body = "Your new address is now available for shipping and billing selections."
		}
	case "missing_selection":
		alert.Tone = "error"
		if lang == "ja" {
			alert.Title = "配送先と請求先を選択してください"
			alert.Body = "両方の住所を選択するまで次に進めません。"
		} else {
			alert.Title = "Select shipping and billing"
			alert.Body = "Choose an address for each step before continuing."
		}
	default:
		return nil
	}
	return []CheckoutInlineAlert{alert}
}

func mergeCheckoutAddresses(lang string, sessionAddrs []mw.SessionAddress) []CheckoutAddress {
	base := mockCheckoutAddresses(lang)
	prepend := make([]CheckoutAddress, 0, len(sessionAddrs))
	for _, addr := range sessionAddrs {
		prepend = append(prepend, sessionAddressToCheckout(lang, addr))
	}
	all := append(prepend, base...)
	seen := map[string]bool{}
	deduped := make([]CheckoutAddress, 0, len(all))
	for _, addr := range all {
		if addr.ID == "" || seen[addr.ID] {
			continue
		}
		seen[addr.ID] = true
		deduped = append(deduped, addr)
	}
	sort.SliceStable(deduped, func(i, j int) bool {
		return deduped[i].UpdatedAt.After(deduped[j].UpdatedAt)
	})
	return deduped
}

func ensureAddressSelection(addresses []CheckoutAddress, requested string, prefer func(CheckoutAddress) bool) string {
	if requested != "" {
		if findCheckoutAddress(addresses, requested) != nil {
			return requested
		}
	}
	for _, addr := range addresses {
		if prefer(addr) {
			return addr.ID
		}
	}
	if len(addresses) > 0 {
		return addresses[0].ID
	}
	return ""
}

func findCheckoutAddress(addresses []CheckoutAddress, id string) *CheckoutAddress {
	if id == "" {
		return nil
	}
	for i := range addresses {
		if addresses[i].ID == id {
			return &addresses[i]
		}
	}
	return nil
}

func sessionAddressToCheckout(lang string, addr mw.SessionAddress) CheckoutAddress {
	label := addr.Label
	if label == "" {
		label = i18nOrDefault(lang, "checkout.address.custom", "Custom address")
	}
	return CheckoutAddress{
		ID:              addr.ID,
		Label:           label,
		Recipient:       addr.Recipient,
		Company:         addr.Company,
		Department:      addr.Department,
		Line1:           addr.Line1,
		Line2:           addr.Line2,
		City:            addr.City,
		Region:          addr.Region,
		PostalCode:      addr.Postal,
		Country:         addr.Country,
		CountryLabel:    countryLabelFor(lang, addr.Country),
		Phone:           addr.Phone,
		Kind:            addr.Kind,
		UpdatedAt:       addr.CreatedAt,
		DefaultShipping: false,
		DefaultBilling:  false,
		Tags:            []CheckoutBadge{{Label: i18nOrDefault(lang, "checkout.address.tag.manual", "Added"), Tone: "info"}},
		Editable:        true,
	}
}

func mockCheckoutAddresses(lang string) []CheckoutAddress {
	now := time.Now()
	return []CheckoutAddress{
		{
			ID:           "addr_tokyo",
			Label:        i18nOrDefault(lang, "checkout.address.labs", "Nihonbashi Studio"),
			Recipient:    "Mai Kato",
			Company:      "Hanko Field Lab",
			Department:   "Studio Ops",
			Line1:        "1-4-1 Nihonbashi",
			Line2:        "Coredo Muromachi 3F",
			City:         "Chuo-ku",
			Region:       "Tokyo",
			PostalCode:   "103-0027",
			Country:      "JP",
			CountryLabel: countryLabelFor(lang, "JP"),
			Phone:        "+81-3-0000-0000",
			Email:        "mai@hanko.example",
			Tags: []CheckoutBadge{
				{Label: i18nOrDefault(lang, "checkout.address.default_shipping", "Default shipping"), Tone: "success"},
				{Label: "Studio", Tone: "indigo"},
			},
			DefaultShipping: true,
			DefaultBilling:  false,
			Instructions:    i18nOrDefault(lang, "checkout.address.labs.notes", "After-hours courier pickup available"),
			Kind:            "both",
			UpdatedAt:       now.Add(-2 * time.Hour),
			Editable:        true,
		},
		{
			ID:           "addr_osaka",
			Label:        i18nOrDefault(lang, "checkout.address.office", "Osaka Finance"),
			Recipient:    "Kenji Morimoto",
			Company:      "Hanko Field Holdings",
			Department:   "Finance",
			Line1:        "2-11-5 Sonezaki",
			Line2:        "UMEDA tower 12F",
			City:         "Kita-ku",
			Region:       "Osaka",
			PostalCode:   "530-0057",
			Country:      "JP",
			CountryLabel: countryLabelFor(lang, "JP"),
			Phone:        "+81-6-0000-0000",
			Tags: []CheckoutBadge{
				{Label: i18nOrDefault(lang, "checkout.address.billing_default", "Billing"), Tone: "warning"},
			},
			DefaultShipping: false,
			DefaultBilling:  true,
			Instructions:    i18nOrDefault(lang, "checkout.address.finance.notes", "Stamp receipts and forward to AP."),
			Kind:            "billing",
			UpdatedAt:       now.Add(-6 * time.Hour),
			Editable:        true,
		},
		{
			ID:           "addr_sf",
			Label:        "Field Partner SF",
			Recipient:    "Alex Rivera",
			Company:      "Field Partners Inc.",
			Department:   "Logistics",
			Line1:        "50 3rd Street",
			Line2:        "Suite 210",
			City:         "San Francisco",
			Region:       "CA",
			PostalCode:   "94103",
			Country:      "US",
			CountryLabel: countryLabelFor(lang, "US"),
			Phone:        "+1-415-555-0112",
			Tags: []CheckoutBadge{
				{Label: i18nOrDefault(lang, "checkout.address.intl", "International"), Tone: "info"},
			},
			DefaultShipping: false,
			DefaultBilling:  false,
			Instructions:    i18nOrDefault(lang, "checkout.address.intl.notes", "Include HS code 9611.00 for customs."),
			Kind:            "shipping",
			UpdatedAt:       now.Add(-28 * time.Hour),
			Editable:        true,
		},
	}
}

func countryLabelFor(lang, code string) string {
	if code == "" {
		return ""
	}
	opts := cartCountryOptions(lang, strings.ToUpper(code))
	for _, opt := range opts {
		if strings.EqualFold(opt.Value, code) {
			return opt.Label
		}
	}
	return strings.ToUpper(code)
}

func checkoutPrefectureOptions(lang string) []CartOption {
	base := []CartOption{
		{Value: "tokyo", Label: i18nOrDefault(lang, "pref.tokyo", "Tokyo")},
		{Value: "osaka", Label: i18nOrDefault(lang, "pref.osaka", "Osaka")},
		{Value: "hokkaido", Label: i18nOrDefault(lang, "pref.hokkaido", "Hokkaido")},
		{Value: "fukuoka", Label: i18nOrDefault(lang, "pref.fukuoka", "Fukuoka")},
	}
	return append([]CartOption{{Value: "", Label: i18nOrDefault(lang, "pref.select", "Select prefecture")}}, base...)
}

func checkoutKindOptions(lang string) []CartOption {
	return []CartOption{
		{Value: "shipping", Label: i18nOrDefault(lang, "checkout.address.kind.shipping", "Shipping only")},
		{Value: "billing", Label: i18nOrDefault(lang, "checkout.address.kind.billing", "Billing only")},
		{Value: "both", Label: i18nOrDefault(lang, "checkout.address.kind.both", "Use for both")},
	}
}

var addrFallbackCounter uint64

func newSessionAddressID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err == nil {
		return "addr_" + hex.EncodeToString(b)
	}
	seed := fmt.Sprintf("%d-%d", time.Now().UnixNano(), atomic.AddUint64(&addrFallbackCounter, 1))
	sum := sha256.Sum256([]byte(seed))
	return "addr_" + hex.EncodeToString(sum[:6])
}
