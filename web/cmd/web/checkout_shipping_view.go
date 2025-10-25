package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	mw "finitefield.org/hanko-web/internal/middleware"
)

// CheckoutShippingView drives the `/checkout/shipping` page.
type CheckoutShippingView struct {
	Steps          []CheckoutStep
	Alerts         []CheckoutInlineAlert
	Summary        CheckoutSummary
	Options        CheckoutShippingOptionsView
	Support        CheckoutSupportCard
	ShippingStatus CheckoutShippingStatus

	ShippingAddress *CheckoutAddress
	SelectedMethod  string
	ContinueURL     string
	BackURL         string
	LastUpdated     time.Time
}

// CheckoutShippingStatus contains lightweight state flags.
type CheckoutShippingStatus struct {
	MissingAddress bool
}

// CheckoutShippingOptionsView powers the shipping options + comparison fragment.
type CheckoutShippingOptionsView struct {
	Lang          string
	Country       string
	CountryLabel  string
	PostalCode    string
	WeightDisplay string
	Query         string

	Methods    []CheckoutShippingOption
	Notes      []string
	Comparison CheckoutShippingComparison
}

// CheckoutShippingOption renders one selectable rate card.
type CheckoutShippingOption struct {
	ID          string
	Label       string
	Carrier     string
	Badge       string
	BadgeTone   string
	Description string
	ETA         string
	Window      string
	Cost        int64
	Currency    string
	Tone        string
	Selected    bool
	Disabled    bool
	Warning     string
	Highlights  []string
}

// CheckoutShippingComparison summarises relative cost vs speed.
type CheckoutShippingComparison struct {
	CountryLabel  string
	WeightDisplay string
	Entries       []CheckoutShippingComparisonEntry
	UpdatedAt     time.Time
}

// CheckoutShippingComparisonEntry is one row in the comparison chart.
type CheckoutShippingComparisonEntry struct {
	Carrier    string
	Service    string
	Cost       int64
	Currency   string
	ETA        string
	SpeedScore int
	CostScore  int
	Badge      string
	BadgeTone  string
}

func buildCheckoutShippingView(lang string, q url.Values, sess *mw.SessionData) CheckoutShippingView {
	addresses := mergeCheckoutAddresses(lang, sess.Checkout.Addresses)
	shippingAddr := findCheckoutAddress(addresses, sess.Checkout.ShippingAddressID)

	requestedMethodRaw := strings.TrimSpace(q.Get("method"))
	requestedMethod := ""
	if requestedMethodRaw != "" {
		requestedMethod = normalizeCartShippingMethod(requestedMethodRaw)
	}
	sessionMethod := ""
	if sess.Checkout.ShippingMethodID != "" {
		sessionMethod = normalizeCartShippingMethod(sess.Checkout.ShippingMethodID)
	}
	chosenMethod := requestedMethod
	if chosenMethod == "" {
		chosenMethod = sessionMethod
	}

	country := strings.ToUpper(strings.TrimSpace(q.Get("country")))
	postal := strings.TrimSpace(q.Get("postal"))
	promo := normalizePromoCode(q.Get("promo"))

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

	query := url.Values{}
	query.Set("country", country)
	if postal != "" {
		query.Set("postal", postal)
	}
	if chosenMethod != "" {
		query.Set("method", chosenMethod)
	}
	if promo != "" {
		query.Set("promo", promo)
	}

	cartView := buildCartView(lang, query)
	options := buildCheckoutShippingOptions(lang, country, cartView.Shipping.Methods, chosenMethod)
	comparison := buildCheckoutShippingComparison(lang, country, cartView.Shipping.WeightDisplay, cartView.Shipping.Methods, cartView.Estimate.UpdatedAt)
	optionsView := CheckoutShippingOptionsView{
		Lang:          lang,
		Country:       country,
		CountryLabel:  labelForCountry(lang, country),
		PostalCode:    postal,
		WeightDisplay: cartView.Shipping.WeightDisplay,
		Query:         query.Encode(),
		Methods:       options,
		Notes:         cartView.Shipping.Notes,
		Comparison:    comparison,
	}

	summary := CheckoutSummary{
		Estimate: cartView.Estimate,
		Items: []CheckoutSummaryItem{
			{Label: i18nOrDefault(lang, "checkout.summary.items", "Items"), Value: fmt.Sprintf("%d", cartView.Estimate.ItemsCount)},
			{Label: i18nOrDefault(lang, "checkout.summary.weight", "Packed weight"), Value: cartView.Estimate.WeightDisplay},
			{Label: i18nOrDefault(lang, "checkout.summary.method", "Method"), Value: cartView.Estimate.MethodLabel},
		},
		Notes: []string{
			fmt.Sprintf("%s: %s", i18nOrDefault(lang, "checkout.summary.eta", "ETA"), cartView.Estimate.ETA),
		},
		ShippingAddress: shippingAddr,
	}

	alerts := buildCheckoutShippingAlerts(lang, shippingAddr, country)

	view := CheckoutShippingView{
		Steps:   cartSteps(lang, "shipping"),
		Alerts:  alerts,
		Summary: summary,
		Options: optionsView,
		Support: CheckoutSupportCard{
			Title:    i18nOrDefault(lang, "checkout.support.title", "Need help with delivery paperwork?"),
			Body:     i18nOrDefault(lang, "checkout.support.body", "Our concierge team can prepare customs-ready invoices, HS codes, and banking letters."),
			CTALabel: i18nOrDefault(lang, "checkout.support.cta", "Chat with concierge"),
			CTAHref:  "mailto:support@hanko-field.example",
		},
		ShippingStatus: CheckoutShippingStatus{
			MissingAddress: shippingAddr == nil,
		},
		ShippingAddress: shippingAddr,
		SelectedMethod:  chosenMethod,
		ContinueURL:     "/checkout/payment",
		BackURL:         "/checkout/address",
		LastUpdated:     cartView.Estimate.UpdatedAt,
	}
	return view
}

func buildCheckoutShippingAlerts(lang string, shippingAddr *CheckoutAddress, country string) []CheckoutInlineAlert {
	var alerts []CheckoutInlineAlert
	if shippingAddr == nil {
		alerts = append(alerts, CheckoutInlineAlert{
			Tone:  "error",
			Icon:  "exclamation-triangle",
			Title: i18nOrDefault(lang, "checkout.shipping.alert.no_address", "Select a shipping address to continue."),
			Body:  i18nOrDefault(lang, "checkout.shipping.alert.no_address.body", "Add or choose an address in the previous step so we can calculate duties and carriers."),
		})
	}
	if country != "JP" {
		alerts = append(alerts, CheckoutInlineAlert{
			Tone:  "info",
			Icon:  "globe-europe-africa",
			Title: i18nOrDefault(lang, "checkout.shipping.alert.intl.title", "International delivery"),
			Body:  i18nOrDefault(lang, "checkout.shipping.alert.intl.body", "Transit estimates exclude customs clearance. Duties and taxes are settled upon arrival."),
		})
	}
	return alerts
}

func buildCheckoutShippingOptions(lang, country string, methods []CartShippingMethod, selected string) []CheckoutShippingOption {
	opts := make([]CheckoutShippingOption, 0, len(methods))
	for _, method := range methods {
		meta := shippingMethodMeta(lang, method.ID)
		opt := CheckoutShippingOption{
			ID:          method.ID,
			Label:       meta.Label,
			Carrier:     meta.Carrier,
			Badge:       meta.Badge,
			BadgeTone:   meta.BadgeTone,
			Description: method.Description,
			ETA:         method.ETA,
			Window:      meta.Window,
			Cost:        method.Cost,
			Currency:    "JPY",
			Tone:        meta.Tone,
			Highlights:  meta.Highlights,
		}
		if method.ID == selected {
			opt.Selected = true
		}
		if meta.RequiresDomestic && country != "JP" {
			opt.Disabled = true
			opt.Warning = i18nOrDefault(lang, "checkout.shipping.option.pickup.unavailable", "Pickup is limited to Japan-based orders.")
		}
		opts = append(opts, opt)
	}
	return opts
}

type shippingMethodMetadata struct {
	Label            string
	Carrier          string
	Badge            string
	BadgeTone        string
	Window           string
	Tone             string
	Highlights       []string
	RequiresDomestic bool
}

func shippingMethodMeta(lang, id string) shippingMethodMetadata {
	switch id {
	case "express":
		return shippingMethodMetadata{
			Label:   localized(lang, "エクスプレス配送", "Express courier"),
			Carrier: "FedEx Priority",
			Badge: func() string {
				if lang == "ja" {
					return "最速"
				}
				return "Fastest"
			}(),
			BadgeTone: "info",
			Window: func() string {
				if lang == "ja" {
					return "翌営業日集荷、時刻指定"
				}
				return "Next-day pickup with timed delivery"
			}(),
			Tone: "indigo",
			Highlights: []string{
				localized(lang, "国際配送も同料金帯", "Flat international lanes"),
				localized(lang, "ライブ追跡リンク", "Live tracking updates"),
			},
		}
	case "pickup":
		return shippingMethodMetadata{
			Label:   localized(lang, "スタジオ受取", "Studio pickup"),
			Carrier: "Hanko Field Nihonbashi",
			Badge: func() string {
				if lang == "ja" {
					return "来店"
				}
				return "In-person"
			}(),
			BadgeTone: "warning",
			Window: func() string {
				if lang == "ja" {
					return "4時間後に受け取り可"
				}
				return "Ready in 4 hours"
			}(),
			Tone:             "gray",
			Highlights:       []string{localized(lang, "本人確認書類が必要", "Government ID required"), localized(lang, "スタッフと受取確認", "Concierge handoff")},
			RequiresDomestic: true,
		}
	default:
		return shippingMethodMetadata{
			Label:   localized(lang, "スタンダード宅配", "Standard courier"),
			Carrier: "Yamato · JP Post",
			Badge: func() string {
				if lang == "ja" {
					return "おすすめ"
				}
				return "Best value"
			}(),
			BadgeTone: "success",
			Window: func() string {
				if lang == "ja" {
					return "2-3営業日で全国配送"
				}
				return "Nationwide in 2-3 business days"
			}(),
			Tone: "emerald",
			Highlights: []string{
				localized(lang, "時間帯指定可", "Time window selection"),
				localized(lang, "再配送料不要", "No re-delivery fee"),
			},
		}
	}
}

func localized(lang, ja, en string) string {
	if lang == "ja" {
		if ja != "" {
			return ja
		}
		if en != "" {
			return en
		}
		return ja
	}
	if en != "" {
		return en
	}
	if ja != "" {
		return ja
	}
	return ""
}

func buildCheckoutShippingComparison(lang, country, weight string, methods []CartShippingMethod, updated time.Time) CheckoutShippingComparison {
	entries := make([]CheckoutShippingComparisonEntry, 0, len(methods))
	for _, method := range methods {
		meta := shippingMethodMeta(lang, method.ID)
		entry := CheckoutShippingComparisonEntry{
			Carrier:    meta.Carrier,
			Service:    meta.Label,
			Cost:       method.Cost,
			Currency:   "JPY",
			ETA:        method.ETA,
			Badge:      meta.Badge,
			BadgeTone:  meta.BadgeTone,
			SpeedScore: 3,
			CostScore:  3,
		}
		switch method.ID {
		case "express":
			entry.SpeedScore = 5
			entry.CostScore = 2
		case "pickup":
			entry.SpeedScore = 4
			entry.CostScore = 5
		default:
			entry.SpeedScore = 3
			entry.CostScore = 4
		}
		if meta.RequiresDomestic && country != "JP" {
			entry.BadgeTone = "warning"
			entry.Badge = localized(lang, "国内限定", "JP only")
		}
		entries = append(entries, entry)
	}
	return CheckoutShippingComparison{
		CountryLabel:  labelForCountry(lang, country),
		WeightDisplay: weight,
		Entries:       entries,
		UpdatedAt:     updated,
	}
}

func labelForCountry(lang, code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	for _, opt := range cartCountryOptions(lang, code) {
		if opt.Value == code {
			return opt.Label
		}
	}
	if code == "" {
		if lang == "ja" {
			return "指定なし"
		}
		return "Unspecified"
	}
	return code
}
