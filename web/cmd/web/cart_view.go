package main

import (
	"fmt"
	"math"
	"net/url"
	"sort"
	"strings"
	"time"

	"finitefield.org/hanko-web/internal/format"
)

const cartTaxRate = 0.10

// CartView aggregates all data needed for the cart page and fragments.
type CartView struct {
	Lang        string
	Steps       []CheckoutStep
	Items       []CartItem
	Empty       bool
	Alerts      []CartAlert
	Promo       CartPromoView
	Estimate    CartEstimate
	Shipping    CartShippingEstimator
	CrossSell   []Product
	Query       string
	LastUpdated time.Time
}

// CheckoutStep represents a stepper entry for the checkout flow.
type CheckoutStep struct {
	Key         string
	Label       string
	Description string
	Href        string
	Active      bool
	Completed   bool
}

// CartAlert renders contextual notices above the table.
type CartAlert struct {
	Tone  string
	Title string
	Body  string
	Icon  string
}

// CartItem represents a line item in the cart table.
type CartItem struct {
	ID          string
	Name        string
	Subtitle    string
	Image       string
	Badge       string
	BadgeTone   string
	Options     []CartItemOption
	Quantity    int
	MinQty      int
	MaxQty      int
	UnitPrice   int64
	LineTotal   int64
	SKU         string
	Material    string
	Shape       string
	Size        string
	ETA         string
	Notes       []string
	InStock     bool
	WeightGrams int
}

// CartItemOption lists a chosen option or engraving metadata.
type CartItemOption struct {
	Label string
	Value string
}

// CartEstimate summarizes totals and estimator metadata.
type CartEstimate struct {
	Currency      string
	Subtotal      int64
	Discount      int64
	Shipping      int64
	Tax           int64
	Total         int64
	ItemsCount    int
	WeightGrams   int
	WeightDisplay string
	MethodID      string
	MethodLabel   string
	ETA           string
	Country       string
	PostalCode    string
	PromoCode     string
	PromoLabel    string
	PromoTone     string
	UpdatedAt     time.Time
}

// CartShippingEstimator powers the shipping estimator form.
type CartShippingEstimator struct {
	Country       string
	PostalCode    string
	Method        string
	WeightDisplay string
	Methods       []CartShippingMethod
	Countries     []CartOption
	Notes         []string
}

// CartShippingMethod lists selectable options and their quotes.
type CartShippingMethod struct {
	ID          string
	Label       string
	ETA         string
	Description string
	Cost        int64
	Selected    bool
	Tone        string
}

// CartOption represents a simple select option.
type CartOption struct {
	Value string
	Label string
}

// CartPromoView captures the active promo banner plus suggestions.
type CartPromoView struct {
	ActiveCode    string
	ActiveLabel   string
	ActiveMessage string
	ActiveTone    string
	Suggestions   []CartPromoSuggestion
	Placeholder   string
}

// CartPromoSuggestion showcases available promo codes.
type CartPromoSuggestion struct {
	Code        string
	Label       string
	Description string
	Tone        string
}

// buildCartView assembles the page view from query parameters.
func buildCartView(lang string, q url.Values) CartView {
	now := time.Now()
	promo := normalizePromoCode(q.Get("promo"))
	method := normalizeCartShippingMethod(q.Get("method"))
	country := strings.ToUpper(strings.TrimSpace(q.Get("country")))
	if country == "" {
		country = "JP"
	}
	postal := strings.TrimSpace(q.Get("postal"))
	if postal == "" {
		postal = defaultPostalForCountry(country)
	}
	empty := strings.TrimSpace(q.Get("empty")) == "1"

	items := mockCartItems(lang)
	if empty {
		items = nil
	}
	subtotal, quantity, weight := calcCartTotals(items)

	shippingOptions := buildCartShippingMethods(method, country, weight)
	if promo == "FREESHIP" {
		for i := range shippingOptions {
			shippingOptions[i].Cost = 0
		}
	}
	shippingCost := shippingOptionsCost(shippingOptions)

	discount := estimateCartDiscount(subtotal, promo)
	if discount > subtotal {
		discount = subtotal
	}
	taxable := subtotal - discount + shippingCost
	if taxable < 0 {
		taxable = 0
	}
	tax := int64(math.Round(float64(taxable) * cartTaxRate))
	total := taxable + tax

	estimate := CartEstimate{
		Currency:      "JPY",
		Subtotal:      subtotal,
		Discount:      discount,
		Shipping:      shippingCost,
		Tax:           tax,
		Total:         total,
		ItemsCount:    quantity,
		WeightGrams:   weight,
		WeightDisplay: formatCartWeight(weight),
		MethodID:      activeShippingID(shippingOptions),
		MethodLabel:   activeShippingLabel(shippingOptions),
		ETA:           activeShippingETA(shippingOptions),
		Country:       country,
		PostalCode:    postal,
		PromoCode:     promo,
		PromoLabel:    promoLabel(lang, promo),
		PromoTone:     promoTone(promo),
		UpdatedAt:     now,
	}

	shipping := CartShippingEstimator{
		Country:       country,
		PostalCode:    postal,
		Method:        estimate.MethodID,
		WeightDisplay: estimate.WeightDisplay,
		Methods:       shippingOptions,
		Countries:     cartCountryOptions(lang, country),
		Notes: []string{
			i18nOrDefault(lang, "cart.shipping.note", "Lead time reflects engraving and QA (1-2 business days)."),
		},
	}

	promoView := buildCartPromoView(lang, promo, discount)

	steps := cartSteps(lang, "cart")
	alerts := buildCartAlerts(lang, items)
	crossSell := buildCartCrossSell(lang)

	query := q.Encode()

	return CartView{
		Lang:        lang,
		Steps:       steps,
		Items:       items,
		Empty:       len(items) == 0,
		Alerts:      alerts,
		Promo:       promoView,
		Estimate:    estimate,
		Shipping:    shipping,
		CrossSell:   crossSell,
		Query:       query,
		LastUpdated: now,
	}
}

func calcCartTotals(items []CartItem) (subtotal int64, quantity int, weight int) {
	for i := range items {
		line := items[i].UnitPrice * int64(items[i].Quantity)
		items[i].LineTotal = line
		subtotal += line
		quantity += items[i].Quantity
		weight += items[i].WeightGrams * items[i].Quantity
	}
	return
}

func formatCartWeight(weight int) string {
	if weight <= 0 {
		return "-"
	}
	if weight < 1000 {
		return fmt.Sprintf("%d g", weight)
	}
	return fmt.Sprintf("%.1f kg", float64(weight)/1000.0)
}

func defaultPostalForCountry(country string) string {
	switch strings.ToUpper(strings.TrimSpace(country)) {
	case "JP":
		return "100-0001"
	case "US":
		return "94107"
	case "SG":
		return "049910"
	case "AU":
		return "2000"
	default:
		return ""
	}
}

func normalizePromoCode(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	switch code {
	case "HANKO10", "STUDIO15", "WELCOME500", "FREESHIP":
		return code
	default:
		return ""
	}
}

func isValidPromo(code string) bool {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "HANKO10", "STUDIO15", "WELCOME500", "FREESHIP":
		return true
	default:
		return false
	}
}

func normalizeCartShippingMethod(method string) string {
	method = strings.ToLower(strings.TrimSpace(method))
	switch method {
	case "express":
		return "express"
	case "pickup":
		return "pickup"
	default:
		return "standard"
	}
}

func estimateCartDiscount(subtotal int64, promo string) int64 {
	if subtotal <= 0 {
		return 0
	}
	switch promo {
	case "HANKO10":
		return int64(math.Round(float64(subtotal) * 0.10))
	case "STUDIO15":
		return 1500
	case "WELCOME500":
		return 500
	default:
		return 0
	}
}

func promoLabel(lang, promo string) string {
	switch promo {
	case "HANKO10":
		if lang == "ja" {
			return "ハンドカットシリーズ10%オフ"
		}
		return "10% off handcrafted seals"
	case "STUDIO15":
		if lang == "ja" {
			return "スタジオクレジット ¥1,500"
		}
		return "¥1,500 design studio credit"
	case "WELCOME500":
		if lang == "ja" {
			return "初回注文 ¥500 オフ"
		}
		return "¥500 off your first order"
	case "FREESHIP":
		if lang == "ja" {
			return "通常配送が無料"
		}
		return "Free standard shipping"
	default:
		return ""
	}
}

func promoTone(promo string) string {
	if promo == "" {
		return ""
	}
	return "success"
}

func buildCartPromoView(lang, promo string, discount int64) CartPromoView {
	message := ""
	tone := ""
	if promo != "" {
		switch promo {
		case "HANKO10":
			if lang == "ja" {
				message = fmt.Sprintf("ハンドカット印鑑が10%%オフになりました（-%s）。", formatCurrency(discount, lang))
			} else {
				message = fmt.Sprintf("Hand-carved seals are now 10%% off (−%s).", formatCurrency(discount, lang))
			}
		case "STUDIO15":
			if lang == "ja" {
				message = "スタジオクレジットが適用され、¥1,500の割引になりました。"
			} else {
				message = "Studio credit applied — ¥1,500 saved."
			}
		case "WELCOME500":
			if lang == "ja" {
				message = "ようこそ！初回注文から¥500オフになりました。"
			} else {
				message = "Welcome! ¥500 has been taken off this order."
			}
		case "FREESHIP":
			if lang == "ja" {
				message = "通常配送が無料になりました。"
			} else {
				message = "Standard shipping is now free."
			}
		}
		if message != "" {
			tone = "success"
		}
	}
	suggestions := []CartPromoSuggestion{
		{Code: "HANKO10", Label: "10% Hand-carved", Description: i18nOrDefault(lang, "cart.promo.handcut", "Save on classic wood bodies."), Tone: "info"},
		{Code: "STUDIO15", Label: "¥1,500 Studio credit", Description: i18nOrDefault(lang, "cart.promo.studio", "Applies after AI tweaks or engraving updates."), Tone: "info"},
		{Code: "FREESHIP", Label: "Free shipping", Description: i18nOrDefault(lang, "cart.promo.ship", "Zero cost on standard JP delivery."), Tone: "success"},
	}
	return CartPromoView{
		ActiveCode:    promo,
		ActiveLabel:   promoLabel(lang, promo),
		ActiveMessage: message,
		ActiveTone:    tone,
		Suggestions:   suggestions,
		Placeholder: func() string {
			if lang == "ja" {
				return "コードを入力"
			}
			return "Enter promo code"
		}(),
	}
}

func cartSteps(lang, active string) []CheckoutStep {
	labels := map[string][2]string{
		"cart":     {"カート", "Cart"},
		"shipping": {"配送", "Shipping"},
		"payment":  {"支払い", "Payment"},
		"review":   {"確認", "Review"},
	}
	descJa := map[string]string{
		"cart":     "内容を確認し、数量やメモを調整。",
		"shipping": "配送先と請求先を指定。",
		"payment":  "安全な決済情報を入力。",
		"review":   "刻印プレビューを最終確認。",
	}
	descEn := map[string]string{
		"cart":     "Review quantities and engraving notes.",
		"shipping": "Confirm fulfillment and billing addresses.",
		"payment":  "Enter payment securely.",
		"review":   "Approve proof before engraving.",
	}
	order := []string{"cart", "shipping", "payment", "review"}
	steps := make([]CheckoutStep, 0, len(order))
	for i, key := range order {
		lbl := labels[key][1]
		desc := descEn[key]
		if lang == "ja" {
			lbl = labels[key][0]
			if v, ok := descJa[key]; ok {
				desc = v
			}
		}
		steps = append(steps, CheckoutStep{
			Key:         key,
			Label:       lbl,
			Description: desc,
			Href:        stepHrefFor(key),
			Active:      key == active,
			Completed:   i < indexOf(order, active),
		})
	}
	return steps
}

func stepHrefFor(key string) string {
	switch key {
	case "cart":
		return "/cart"
	case "shipping":
		return "/checkout/address"
	case "payment":
		return "/checkout/payment"
	case "review":
		return "/checkout/review"
	default:
		return "/cart"
	}
}

func indexOf(list []string, key string) int {
	for i, v := range list {
		if v == key {
			return i
		}
	}
	return len(list)
}

func buildCartAlerts(lang string, items []CartItem) []CartAlert {
	if len(items) == 0 {
		return nil
	}
	alert := CartAlert{
		Tone: "info",
		Icon: "clock",
	}
	if lang == "ja" {
		alert.Title = "最終チェック"
		alert.Body = "発送前にスタッフが各デザインを再確認します。修正点があればメールでご案内します。"
	} else {
		alert.Title = "Pre-flight QA"
		alert.Body = "Our studio double-checks each seal before fulfillment and will email if adjustments are needed."
	}
	return []CartAlert{alert}
}

func buildCartCrossSell(lang string) []Product {
	all := productData(lang)
	if len(all) == 0 {
		return nil
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].PriceJPY == all[j].PriceJPY {
			return all[i].ID < all[j].ID
		}
		return all[i].PriceJPY < all[j].PriceJPY
	})
	var selected []Product
	seenMaterial := map[string]bool{}
	for _, p := range all {
		if seenMaterial[p.Material] {
			continue
		}
		selected = append(selected, p)
		seenMaterial[p.Material] = true
		if len(selected) == 4 {
			return selected
		}
	}
	if len(selected) < 4 {
		for _, p := range all {
			if len(selected) == 4 {
				break
			}
			duplicate := false
			for _, chosen := range selected {
				if chosen.ID == p.ID {
					duplicate = true
					break
				}
			}
			if duplicate {
				continue
			}
			selected = append(selected, p)
		}
	}
	return selected
}

func cartCountryOptions(lang, active string) []CartOption {
	opts := []CartOption{
		{Value: "JP", Label: i18nOrDefault(lang, "cart.country.jp", "Japan")},
		{Value: "US", Label: i18nOrDefault(lang, "cart.country.us", "United States")},
		{Value: "SG", Label: i18nOrDefault(lang, "cart.country.sg", "Singapore")},
		{Value: "AU", Label: i18nOrDefault(lang, "cart.country.au", "Australia")},
	}
	for i := range opts {
		if opts[i].Value == active {
			// keep order but maybe highlight in template
		}
	}
	return opts
}

func buildCartShippingMethods(method, country string, weight int) []CartShippingMethod {
	base := []CartShippingMethod{
		{
			ID:          "standard",
			Label:       "Standard courier",
			ETA:         "2-3 business days",
			Description: "Tracked Yamato or JP Post delivery nationwide.",
			Cost:        550,
			Tone:        "default",
		},
		{
			ID:          "express",
			Label:       "Express (JP + Intl)",
			ETA:         "1-2 business days",
			Description: "Priority handling + overnight pickup window.",
			Cost:        1100,
			Tone:        "info",
		},
		{
			ID:          "pickup",
			Label:       "Studio pickup",
			ETA:         "Ready in 4 hours",
			Description: "Collect from Nihonbashi studio. ID required.",
			Cost:        0,
			Tone:        "success",
		},
	}
	if country != "JP" {
		base[0].ETA = "4-6 business days"
		base[1].ETA = "2-4 business days"
		base[0].Description = "Air mail with tracking. Duties billed separately."
		base[1].Description = "FedEx International Priority."
		if weight > 1500 {
			base[1].Cost = 2400
		} else {
			base[1].Cost = 1800
		}
		base[0].Cost = 1200
		base[2].Description = "Not available outside Japan"
	}
	for i := range base {
		if base[i].ID == method {
			base[i].Selected = true
		}
		if country != "JP" && base[i].ID == "pickup" {
			base[i].Tone = "warning"
		}
	}
	// Ensure at least one is selected
	selected := false
	for _, opt := range base {
		if opt.Selected {
			selected = true
			break
		}
	}
	if !selected {
		base[0].Selected = true
	}
	return base
}

func shippingOptionsCost(options []CartShippingMethod) int64 {
	for _, opt := range options {
		if opt.Selected {
			return opt.Cost
		}
	}
	if len(options) == 0 {
		return 0
	}
	return options[0].Cost
}

func activeShippingID(options []CartShippingMethod) string {
	for _, opt := range options {
		if opt.Selected {
			return opt.ID
		}
	}
	if len(options) > 0 {
		return options[0].ID
	}
	return "standard"
}

func activeShippingLabel(options []CartShippingMethod) string {
	for _, opt := range options {
		if opt.Selected {
			return opt.Label
		}
	}
	if len(options) > 0 {
		return options[0].Label
	}
	return ""
}

func activeShippingETA(options []CartShippingMethod) string {
	for _, opt := range options {
		if opt.Selected {
			return opt.ETA
		}
	}
	if len(options) > 0 {
		return options[0].ETA
	}
	return ""
}

func mockCartItems(lang string) []CartItem {
	title := func(en, ja string) string {
		if lang == "ja" && ja != "" {
			return ja
		}
		return en
	}
	return []CartItem{
		{
			ID:        "cart-classic-round",
			Name:      title("Hinoki signature seal", "柘植（つげ）認印 15mm"),
			Subtitle:  title("Hand-carved / Round 15 mm", "手彫り / 丸形 15 mm"),
			Image:     "https://placehold.co/600x600?text=Hinoki",
			Badge:     title("Workshop favorite", "工房おすすめ"),
			BadgeTone: "info",
			Options: []CartItemOption{
				{Label: title("Material", "素材"), Value: materialLabel(lang, "wood")},
				{Label: title("Engraving", "刻印"), Value: "佐藤 花子"},
				{Label: title("Font", "書体"), Value: "Kaisho"},
			},
			Quantity:    2,
			MinQty:      1,
			MaxQty:      10,
			UnitPrice:   2480,
			SKU:         "WOOD-R15",
			Material:    "wood",
			Shape:       "round",
			Size:        "medium",
			ETA:         title("Ships in 2 days", "2営業日以内に発送"),
			Notes:       []string{title("Includes velvet case", "ベルベットケース付き")},
			InStock:     true,
			WeightGrams: 110,
		},
		{
			ID:        "cart-metal-square",
			Name:      title("Stainless corporate seal", "ステンレス社判 18mm"),
			Subtitle:  title("Square / Metal / Laser assist", "角印 / 金属 / レーザー補助"),
			Image:     "https://placehold.co/600x600?text=Steel",
			Badge:     title("Rush eligible", "特急対応可"),
			BadgeTone: "success",
			Options: []CartItemOption{
				{Label: "Template", Value: "HF-DX203"},
				{Label: title("Line weight", "線の太さ"), Value: "1.6 pt"},
			},
			Quantity:    1,
			MinQty:      1,
			MaxQty:      5,
			UnitPrice:   6200,
			SKU:         "MET-SQ18",
			Material:    "metal",
			Shape:       "square",
			Size:        "large",
			ETA:         title("Ships tomorrow", "翌営業日発送"),
			Notes:       []string{title("Includes digital proof", "デジタル校正付き")},
			InStock:     true,
			WeightGrams: 280,
		},
		{
			ID:        "cart-case-add-on",
			Name:      title("Washi storage duo", "和紙ケースセット"),
			Subtitle:  title("Protective travel cases", "印鑑用保護ケース"),
			Image:     "https://placehold.co/600x600?text=Case",
			Badge:     title("Limited", "限定"),
			BadgeTone: "warning",
			Options: []CartItemOption{
				{Label: title("Color", "カラー"), Value: title("Indigo / Vermilion", "藍 / 朱")},
			},
			Quantity:    1,
			MinQty:      1,
			MaxQty:      3,
			UnitPrice:   1800,
			SKU:         "CASE-SET",
			Material:    "accessory",
			Shape:       "rect",
			Size:        "small",
			ETA:         title("Ships with stamps", "同梱発送"),
			Notes:       []string{title("Protects up to 2 seals", "印鑑2本まで収納可")},
			InStock:     true,
			WeightGrams: 90,
		},
	}
}

func formatCurrency(amount int64, lang string) string {
	if amount <= 0 {
		return ""
	}
	return format.FmtCurrency(amount, "JPY", lang)
}
