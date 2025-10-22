package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"finitefield.org/hanko-web/internal/format"
	handlersPkg "finitefield.org/hanko-web/internal/handlers"
	"finitefield.org/hanko-web/internal/i18n"
	mw "finitefield.org/hanko-web/internal/middleware"
	"finitefield.org/hanko-web/internal/nav"
	"finitefield.org/hanko-web/internal/seo"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var (
	templatesDir = "templates"
	publicDir    = "public"
	localesDir   = "locales"
	// devMode is set in main() based on env: HANKO_WEB_DEV (preferred) or DEV (fallback)
	devMode    bool
	tmplCache  *template.Template
	i18nBundle *i18n.Bundle
	// per-page cache in production to avoid reparse on each request
	pageTmplCache = map[string]*template.Template{}
	pageTmplMu    sync.RWMutex
)

var (
	shapeLabels = map[string]map[string]string{
		"round":  {"en": "Round", "ja": "丸型"},
		"square": {"en": "Square", "ja": "角型"},
		"rect":   {"en": "Rect", "ja": "長方形"},
	}
	sizeLabels = map[string]map[string]string{
		"small":  {"en": "Small", "ja": "小"},
		"medium": {"en": "Medium", "ja": "中"},
		"large":  {"en": "Large", "ja": "大"},
	}
	materialLabels = map[string]map[string]string{
		"wood":   {"en": "Hinoki Cypress", "ja": "檜（ひのき）"},
		"rubber": {"en": "Eco Rubber", "ja": "エコゴム"},
		"metal":  {"en": "Stainless Steel", "ja": "ステンレス"},
	}
	sizeDimensions = map[string]map[string]string{
		"round": {
			"small":  "12 mm",
			"medium": "15 mm",
			"large":  "18 mm",
		},
		"square": {
			"small":  "12 mm",
			"medium": "15 mm",
			"large":  "18 mm",
		},
		"rect": {
			"small":  "14 × 40 mm",
			"medium": "18 × 50 mm",
			"large":  "21 × 60 mm",
		},
	}
)

func localizedLabel(table map[string]map[string]string, lang, key, fallback string) string {
	if key == "" {
		return fallback
	}
	if row, ok := table[key]; ok {
		if v, ok := row[lang]; ok && v != "" {
			return v
		}
		if v, ok := row["en"]; ok && v != "" {
			return v
		}
	}
	if fallback != "" {
		return fallback
	}
	return key
}

func shapeLabel(lang, key string) string {
	return localizedLabel(shapeLabels, lang, key, key)
}

func sizeLabel(lang, key string) string {
	return localizedLabel(sizeLabels, lang, key, key)
}

func materialLabel(lang, key string) string {
	return localizedLabel(materialLabels, lang, key, key)
}

func measurementFor(shape, size string) string {
	if m, ok := sizeDimensions[shape]; ok {
		if v, ok := m[size]; ok && v != "" {
			return v
		}
	}
	return ""
}

func weightFor(material, size string) string {
	base := map[string]int{"small": 22, "medium": 28, "large": 34}
	w, ok := base[size]
	if !ok {
		w = 26
	}
	switch material {
	case "metal":
		w += 12
	case "rubber":
		w -= 6
	case "wood":
		w += 0
	}
	if w < 12 {
		w = 12
	}
	return fmt.Sprintf("%d g", w)
}

func titleCaseASCII(s string) string {
	if s == "" {
		return s
	}
	lower := strings.ToLower(s)
	b := []byte(lower)
	if len(b) > 0 && b[0] >= 'a' && b[0] <= 'z' {
		b[0] = b[0] - ('a' - 'A')
	}
	return string(b)
}

func defaultMediaFor(prod Product) []ProductMedia {
	base := fmt.Sprintf("%s %s", materialLabel("en", prod.Material), shapeLabel("en", prod.Shape))
	entries := []struct {
		ID    string
		Label string
	}{
		{"hero", "Hero"},
		{"angle", "Angle"},
		{"detail", "Detail"},
		{"packaging", "Packaging"},
	}
	out := make([]ProductMedia, 0, len(entries))
	for _, e := range entries {
		text := fmt.Sprintf("%s %s", base, e.Label)
		src := fmt.Sprintf("https://placehold.co/1200x900?text=%s", url.QueryEscape(text))
		thumb := fmt.Sprintf("https://placehold.co/240x180?text=%s", url.QueryEscape(e.Label))
		out = append(out, ProductMedia{
			ID:        e.ID,
			Kind:      "image",
			Src:       src,
			Thumbnail: thumb,
			Alt:       text,
			Width:     1200,
			Height:    900,
		})
	}
	return out
}

func defaultOptions(lang string, prod Product) []ProductOptionGroup {
	var (
		inkLabel, inkHelp, vermillionLabel, vermillionSub, blackLabel, blackSub, blueLabel, blueSub string
		caseLabel, caseHelp, noneLabel, noneSub, snapLabel, snapSub, hingedLabel, hingedSub         string
	)
	if lang == "ja" {
		inkLabel = "朱肉カラー"
		inkHelp = "書類や用途に合わせて色を選べます。"
		vermillionLabel = "朱色"
		vermillionSub = "伝統色・書類向け"
		blackLabel = "黒"
		blackSub = "コントラストが高く鮮明"
		blueLabel = "藍"
		blueSub = "公的書類や銀行向け"
		caseLabel = "ケース"
		caseHelp = "贈り物や保管用に人気です。"
		noneLabel = "ケースなし (標準)"
		noneSub = ""
		snapLabel = "スナップケース"
		snapSub = "合皮 / クッション付き"
		hingedLabel = "ヒンジ付木箱"
		hingedSub = "檜製 / 乾燥剤付き"
	} else {
		inkLabel = "Ink color"
		inkHelp = "Choose the ink pad color that ships with your stamp."
		vermillionLabel = "Vermillion"
		vermillionSub = "Traditional, everyday documents"
		blackLabel = "Black"
		blackSub = "High contrast for crisp impressions"
		blueLabel = "Blue"
		blueSub = "Preferred for official filings"
		caseLabel = "Case"
		caseHelp = "Optional protective cases for gifting or travel."
		noneLabel = "No case (standard)"
		noneSub = ""
		snapLabel = "Snap case"
		snapSub = "Faux leather with cushion insert"
		hingedLabel = "Hinged wooden case"
		hingedSub = "Hinoki with desiccant pocket"
	}
	return []ProductOptionGroup{
		{
			Name:     "ink_color",
			Label:    inkLabel,
			Required: true,
			Help:     inkHelp,
			Choices: []ProductOptionChoice{
				{Value: "vermillion", Label: vermillionLabel, Subtitle: vermillionSub, Selected: true},
				{Value: "black", Label: blackLabel, Subtitle: blackSub},
				{Value: "blue", Label: blueLabel, Subtitle: blueSub},
			},
		},
		{
			Name:  "case",
			Label: caseLabel,
			Help:  caseHelp,
			Choices: []ProductOptionChoice{
				{Value: "none", Label: noneLabel, Subtitle: noneSub, Selected: true},
				{Value: "snap", Label: snapLabel, Subtitle: snapSub},
				{Value: "hinged", Label: hingedLabel, Subtitle: hingedSub},
			},
		},
	}
}

func defaultSpecs(lang string, prod Product) []ProductSpec {
	dimension := measurementFor(prod.Shape, prod.Size)
	mat := materialLabel(lang, prod.Material)
	shapeName := shapeLabel(lang, prod.Shape)
	sizeName := sizeLabel(lang, prod.Size)
	weight := weightFor(prod.Material, prod.Size)

	var (
		materialLabelText string
		shapeLabelText    string
		sizeLabelText     string
		finishLabel       string
		weightLabel       string
		originLabel       string
		includesLabel     string
		finishValue       string
		originValue       string
		includesValue     string
	)

	switch lang {
	case "ja":
		materialLabelText = "材質"
		shapeLabelText = "形状"
		sizeLabelText = "サイズ"
		finishLabel = "仕上げ"
		weightLabel = "重量"
		originLabel = "生産"
		includesLabel = "付属品"
		originValue = "静岡県の工房で加工"
		includesValue = "収納チューブ、取扱説明カード、乾燥剤"
	default:
		materialLabelText = "Material"
		shapeLabelText = "Shape"
		sizeLabelText = "Impression size"
		finishLabel = "Finish"
		weightLabel = "Weight"
		originLabel = "Production"
		includesLabel = "Included"
		originValue = "Crafted in Shizuoka, Japan"
		includesValue = "Protective tube, care card, moisture pack"
	}

	switch prod.Material {
	case "wood":
		if lang == "ja" {
			finishValue = "蜜蝋オイル仕上げ / ソフトタッチグリップ"
		} else {
			finishValue = "Hand-rubbed tung & beeswax oil, soft-touch grip"
		}
	case "rubber":
		if lang == "ja" {
			finishValue = "再生ゴムマウント / 滑り止めグリップ"
		} else {
			finishValue = "Recycled rubber mount with anti-slip grip"
		}
	case "metal":
		if lang == "ja" {
			finishValue = "ヘアライン加工ステンレス / 断熱スリーブ"
		} else {
			finishValue = "Brushed stainless body with thermal sleeve"
		}
	default:
		finishValue = ""
	}

	specs := []ProductSpec{
		{Label: materialLabelText, Value: mat},
		{Label: shapeLabelText, Value: fmt.Sprintf("%s / %s", shapeName, sizeName)},
	}
	if dimension != "" {
		specs = append(specs, ProductSpec{Label: sizeLabelText, Value: dimension})
	}
	specs = append(specs,
		ProductSpec{Label: finishLabel, Value: finishValue},
		ProductSpec{Label: weightLabel, Value: weight},
		ProductSpec{Label: originLabel, Value: originValue},
		ProductSpec{Label: includesLabel, Value: includesValue},
	)
	return specs
}

func defaultFAQ(lang string) []ProductFAQ {
	if lang == "ja" {
		return []ProductFAQ{
			{
				Question: "オリジナルの印影データを入稿できますか？",
				Answer:   "はい。AI/EPS/SVG または 600dpi 以上の PNG をアップロードすると自動で調整し、仕上がりイメージをメールでお送りします。",
			},
			{
				Question: "木製ボディのお手入れ方法は？",
				Answer:   "乾いた柔らかい布で軽く拭き、年に数回 蜜蝋や木製用オイルを薄く塗ってください。直射日光と高温多湿を避けると長持ちします。",
			},
		}
	}
	return []ProductFAQ{
		{
			Question: "Can I submit my own artwork?",
			Answer:   "Yes. Upload AI/EPS/SVG or a 600dpi PNG during checkout and our team will align and proof it. You'll receive a confirmation preview before we engrave.",
		},
		{
			Question: "How should I care for the wooden body?",
			Answer:   "Wipe with a dry cloth after use and apply a light layer of beeswax or wood conditioner a few times a year. Keep away from direct sun and high humidity.",
		},
	}
}

func defaultReviewSummary(lang, id string) ReviewSummary {
	link := fmt.Sprintf("/products/%s/reviews", strings.ToLower(id))
	breakdown := []RatingBreakdown{
		{Stars: 5, Percent: 74},
		{Stars: 4, Percent: 21},
		{Stars: 3, Percent: 3},
		{Stars: 2, Percent: 1},
		{Stars: 1, Percent: 1},
	}
	switch lang {
	case "ja":
		return ReviewSummary{
			Average:    4.7,
			Count:      128,
			Highlights: []string{"押し心地が軽い", "印影がくっきり出る", "ギフトに喜ばれる木箱付き"},
			Breakdown:  breakdown,
			Latest: []ReviewSnippet{
				{Author: "Mika", Rating: 5, Quote: "丸みのある木製グリップが手に馴染み、スムーズに押せます。", Date: time.Date(2025, time.January, 4, 0, 0, 0, 0, time.UTC), Locale: lang},
				{Author: "Kenji", Rating: 5, Quote: "印影がぶれずに揃い、銀行書類でも問題なく使えました。", Date: time.Date(2024, time.December, 18, 0, 0, 0, 0, time.UTC), Locale: lang},
				{Author: "Naoko", Rating: 4.5, Quote: "檜の香りが良く、贈り物にも最適です。ケース付きにして正解でした。", Date: time.Date(2024, time.November, 2, 0, 0, 0, 0, time.UTC), Locale: lang},
			},
			Link: link,
		}
	default:
		return ReviewSummary{
			Average:    4.7,
			Count:      128,
			Highlights: []string{"Light, balanced handle", "Crisp impressions every time", "Gift-ready packaging"},
			Breakdown:  breakdown,
			Latest: []ReviewSnippet{
				{Author: "Alicia", Rating: 5, Quote: "The hinoki body feels warm in hand and stamps evenly without effort.", Date: time.Date(2025, time.January, 8, 0, 0, 0, 0, time.UTC), Locale: lang},
				{Author: "David", Rating: 5, Quote: "Sharp, clean impressions on legal envelopes and matte paper alike.", Date: time.Date(2024, time.December, 20, 0, 0, 0, 0, time.UTC), Locale: lang},
				{Author: "Sofia", Rating: 4.5, Quote: "Love the natural scent and the hinged case made gifting easy.", Date: time.Date(2024, time.November, 12, 0, 0, 0, 0, time.UTC), Locale: lang},
			},
			Link: link,
		}
	}
}

func recommendedProducts(lang string, current Product) []Product {
	out := make([]Product, 0, 4)
	seen := map[string]bool{current.ID: true}
	add := func(p Product) {
		if seen[p.ID] {
			return
		}
		if len(out) >= 4 {
			return
		}
		seen[p.ID] = true
		out = append(out, p)
	}
	all := productData(lang)
	for _, p := range all {
		if p.Material == current.Material && p.ID != current.ID {
			add(p)
			if len(out) >= 4 {
				return out
			}
		}
	}
	for _, p := range all {
		if p.Shape == current.Shape && p.ID != current.ID {
			add(p)
			if len(out) >= 4 {
				return out
			}
		}
	}
	for _, p := range all {
		if p.ID != current.ID {
			add(p)
			if len(out) >= 4 {
				return out
			}
		}
	}
	return out
}

func productDetailFor(lang, id string) (*ProductDetail, bool) {
	prod, ok := findProduct(lang, id)
	if !ok {
		return nil, false
	}

	subtitleParts := []string{}
	if v := materialLabel(lang, prod.Material); v != "" {
		subtitleParts = append(subtitleParts, v)
	}
	if v := shapeLabel(lang, prod.Shape); v != "" {
		subtitleParts = append(subtitleParts, v)
	}
	if v := sizeLabel(lang, prod.Size); v != "" {
		subtitleParts = append(subtitleParts, v)
	}
	subtitle := strings.Join(subtitleParts, " • ")
	if subtitle == "" {
		subtitle = prod.Name
	}

	availability := "in-stock"
	availabilityLabel := ""
	availabilityTone := "success"
	shipping := ""
	if prod.InStock {
		if lang == "ja" {
			availabilityLabel = "在庫あり"
			shipping = "3営業日以内に出荷"
		} else {
			availabilityLabel = "In stock"
			shipping = "Ships in 3 business days"
		}
	} else {
		availability = "backorder"
		availabilityTone = "warning"
		if lang == "ja" {
			availabilityLabel = "予約受付中"
			shipping = "次回ロット：2週間前後"
		} else {
			availabilityLabel = "Backorder"
			shipping = "Next batch ships in ~2 weeks"
		}
	}

	if prod.Material == "metal" {
		if lang == "ja" {
			shipping = "5営業日以内に出荷"
		} else {
			shipping = "Ships in 5 business days"
		}
	}

	priceNote := ""
	if prod.Sale {
		if lang == "ja" {
			priceNote = "今だけセール価格（通常価格より¥400お得）"
		} else {
			priceNote = "Limited sale pricing – ¥400 off regular price"
		}
	} else {
		if lang == "ja" {
			priceNote = "彫刻費・税込価格"
		} else {
			priceNote = "Pricing includes engraving and tax"
		}
	}

	detail := ProductDetail{
		Product:           prod,
		Subtitle:          subtitle,
		Availability:      availability,
		AvailabilityLabel: availabilityLabel,
		AvailabilityTone:  availabilityTone,
		ShippingEstimate:  shipping,
		PriceNote:         priceNote,
		Media:             defaultMediaFor(prod),
		Options:           defaultOptions(lang, prod),
		Specs:             defaultSpecs(lang, prod),
		FAQ:               defaultFAQ(lang),
		Reviews:           defaultReviewSummary(lang, id),
		Recommended:       recommendedProducts(lang, prod),
		MinQty:            1,
		MaxQty:            10,
	}

	switch prod.Material {
	case "wood":
		if lang == "ja" {
			detail.Lead = "静岡産の檜を削り出した伝統的な丸型はんこ。"
			detail.Description = []string{
				"職人が乾燥させた檜素材にレーザーと手仕上げで彫刻し、軽い押し心地とムラのない印影を実現しました。",
				"湿度を調整する乾燥剤とケアカードを同梱。使用後に軽く拭くだけで香りと艶を保てます。",
			}
			detail.Highlights = []string{
				"檜の芳香と手になじむラウンド形状",
				"二段彫りで細字もくっきり再現",
				"保管用チューブとケアカード付き",
			}
		} else {
			detail.Lead = "Classic round stamp carved from sustainably sourced hinoki cypress."
			detail.Description = []string{
				"Kiln-dried hinoki blanks are laser engraved and hand-finished to deliver balanced pressure and consistent impressions.",
				"A moisture-balanced storage tube and care guide keep the wood conditioned between uses, preserving its subtle aroma.",
			}
			detail.Highlights = []string{
				"Aromatic hinoki finish that feels warm in hand",
				"Dual-depth engraving keeps edges crisp over time",
				"Ships with storage tube, desiccant, and care card",
			}
		}
	case "rubber":
		if lang == "ja" {
			detail.Lead = "高耐久の再生ゴムを使用した業務用スタンプ。大量の押印作業にも最適です。"
			detail.Description = []string{
				"再生ゴムの印面は細かな文字も潰れにくく、滑り止め付きのグリップで長時間の押印でも疲れにくい設計です。",
				"付属のスナップケースに収納すれば持ち運びも安心。インクパッドと同時購入で当日発送に対応します。",
			}
			detail.Highlights = []string{
				"再生素材を活用したエコ設計",
				"連続捺印でも滑りにくいグリップ",
				"ケースとセットで当日出荷に対応",
			}
		} else {
			detail.Lead = "Durable recycled rubber stamp engineered for high-volume office workflows."
			detail.Description = []string{
				"The recycled rubber die resists wear while the textured anti-slip grip keeps every impression aligned, even during bulk tasks.",
				"Pair it with the snap case to protect the die between site visits, and bundle ink refills for same-day dispatch.",
			}
			detail.Highlights = []string{
				"Eco-friendly recycled rubber construction",
				"Anti-slip grip stays steady during rapid stamping",
				"Same-day shipping when bundled with ink refills",
			}
		}
	case "metal":
		if lang == "ja" {
			detail.Lead = "精密に削り出したステンレスボディで、美観と耐久性を両立した角型はんこ。"
			detail.Description = []string{
				"CNC 加工のステンレスに断熱スリーブを組み合わせ、滑らかな押し心地と適度な重量感を実現。公的書類に最適です。",
				"専用の木製ケースと合わせれば、ギフトとしても映える高級感。社判やロゴにも対応します。",
			}
			detail.Highlights = []string{
				"ヘアライン仕上げのステンレスボディ",
				"断熱スリーブで冬でも冷たくなりにくい",
				"企業ロゴや角印に最適な高精細彫刻",
			}
		} else {
			detail.Lead = "Precision-milled stainless steel stamp that balances heft with a thermal grip."
			detail.Description = []string{
				"CNC machining delivers a perfectly flat impression surface while the thermal sleeve keeps the housing comfortable in colder offices.",
				"Pair with the hinoki hinged case for presentation-ready gifting or executive desk sets.",
			}
			detail.Highlights = []string{
				"Brushed stainless body with subtle chamfers",
				"Thermal sleeve prevents cold-touch discomfort",
				"Ideal for corporate logos and executive seals",
			}
		}
	}

	switch id {
	case "P-1000":
		detail.Reviews.Average = 4.8
		detail.Reviews.Count = 182
		// Add swatch and demo video to gallery
		swatch := ProductMedia{
			ID:        "swatch-hinoki",
			Kind:      "image",
			Src:       "https://placehold.co/900x900?text=Hinoki+Grain",
			Thumbnail: "https://placehold.co/200x200?text=Swatch",
			Alt:       "Hinoki grain close-up",
			Width:     900,
			Height:    900,
		}
		video := ProductMedia{
			ID:     "demo",
			Kind:   "video",
			Src:    "https://storage.googleapis.com/coverr-main/mp4/Mt_Baker.mp4",
			Poster: "https://placehold.co/1200x900?text=Stamp+Demo",
			Alt:    "Demonstration of stamping on kraft paper",
			Width:  1280,
			Height: 720,
		}
		detail.Media = append([]ProductMedia{swatch, video}, detail.Media...)
	case "P-1009":
		detail.Availability = "low-stock"
		detail.AvailabilityTone = "warning"
		if lang == "ja" {
			detail.AvailabilityLabel = "残りわずか"
			detail.ShippingEstimate = "13時までのご注文で当日出荷"
			detail.PriceNote = "数量限定セール：通常価格より¥400引き"
		} else {
			detail.AvailabilityLabel = "Low stock"
			detail.ShippingEstimate = "Ships same day if ordered before 1PM JST"
			detail.PriceNote = "Limited stock sale: ¥400 below regular price"
		}
		detail.Reviews.Average = 4.6
	}

	if detail.Lead == "" {
		if lang == "ja" {
			detail.Lead = fmt.Sprintf("%sの%sを採用したカスタムはんこ。", materialLabel(lang, prod.Material), shapeLabel(lang, prod.Shape))
		} else {
			detail.Lead = fmt.Sprintf("Custom %s stamp featuring our %s profile.", materialLabel("en", prod.Material), shapeLabel("en", prod.Shape))
		}
	}
	if len(detail.Description) == 0 {
		if lang == "ja" {
			detail.Description = []string{
				"レーザーと手仕上げで彫刻した印面が、にじみのない印影を実現します。",
				"付属のケアカードを参考に、定期的なメンテナンスで長く美しく使えます。",
			}
		} else {
			detail.Description = []string{
				"Laser-engraved and hand-finished to deliver balanced pressure and crisp impressions.",
				"Follow the included care guide to keep the stamp performing beautifully for years.",
			}
		}
	}
	if len(detail.Highlights) == 0 {
		if lang == "ja" {
			detail.Highlights = []string{"手仕上げの滑らかな押し心地", "メンテナンスが簡単", "ギフトにも喜ばれるパッケージ"}
		} else {
			detail.Highlights = []string{"Hand-finished for smooth stamping", "Easy upkeep with included guide", "Gift-ready packaging"}
		}
	}
	if detail.DefaultMedia == "" && len(detail.Media) > 0 {
		detail.DefaultMedia = detail.Media[0].ID
	}

	detail.ReviewStars = ratingStars(detail.Reviews.Average)

	return &detail, true
}

func mediaByID(detail *ProductDetail, mediaID string) (ProductMedia, bool) {
	for _, m := range detail.Media {
		if m.ID == mediaID {
			return m, true
		}
	}
	return ProductMedia{}, false
}

func ratingStars(avg float64) []string {
	stars := make([]string, 5)
	remaining := avg
	for i := 0; i < 5; i++ {
		if remaining >= 0.75 {
			stars[i] = "full"
		} else if remaining >= 0.25 {
			stars[i] = "half"
		} else {
			stars[i] = "empty"
		}
		remaining -= 1
		if remaining < 0 {
			remaining = 0
		}
	}
	return stars
}

func main() {
	// Flags/environment
	var (
		addr     string
		tmplPath string
		pubPath  string
	)
	// Port resolution: prefer HANKO_WEB_PORT, then Cloud Run's PORT, else 8080
	port := os.Getenv("HANKO_WEB_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}
	flag.StringVar(&addr, "addr", ":"+port, "HTTP listen address")
	flag.StringVar(&tmplPath, "templates", templatesDir, "templates directory")
	flag.StringVar(&pubPath, "public", publicDir, "public assets directory")
	flag.StringVar(&localesDir, "locales", localesDir, "locales directory")
	flag.Parse()

	templatesDir = tmplPath
	publicDir = pubPath

	// Dev mode: prefer HANKO_WEB_DEV, fallback to DEV
	devMode = os.Getenv("HANKO_WEB_DEV") != "" || os.Getenv("DEV") != ""

	// Load i18n bundle
	sup := []string{"ja", "en"}
	if v := os.Getenv("HANKO_WEB_LOCALES"); v != "" {
		sup = strings.Split(v, ",")
		for i := range sup {
			sup[i] = strings.TrimSpace(sup[i])
		}
	}
	var err error
	i18nBundle, err = i18n.Load(localesDir, "ja", sup)
	if err != nil {
		log.Fatalf("i18n load failed: %v", err)
	}

	if !devMode {
		// Parse templates once in production
		tc, err := parseTemplates()
		if err != nil {
			log.Fatalf("parse templates: %v", err)
		}
		tmplCache = tc
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	// If deployed behind a trusted reverse proxy/load balancer, RealIP will use
	// X-Forwarded-For to determine the client IP. Ensure only trusted proxies
	// can set these headers in production environments.
	r.Use(middleware.RealIP)
	r.Use(mw.HTMX)
	r.Use(mw.Session)
	if i18nBundle != nil {
		r.Use(mw.Locale(i18nBundle))
	}
	r.Use(mw.Auth)
	r.Use(mw.CSRF)
	r.Use(mw.VaryLocale)
	r.Use(mw.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Use(middleware.Timeout(30 * time.Second))

	// Health check
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Static assets under /assets/ (with Cache-Control and ETag)
	assetsRoot := filepath.Join(publicDir, "assets")
	assets := http.StripPrefix("/assets", mw.AssetsWithCache(assetsRoot))
	r.Handle("/assets/*", assets)

	// Home page
	r.Get("/", HomeHandler)
	// Top-level pages
	r.Get("/shop", ShopHandler)
	r.Get("/products/{productID}", ProductDetailHandler)
	// Shop results fragment (htmx)
	r.Get("/shop/table", ShopTableFrag)
	r.Get("/products/{productID}/gallery/modal", ProductGalleryModalFrag)
	r.Get("/products/{productID}/gallery", ProductGalleryFrag)
	r.Get("/products/{productID}/reviews/snippets", ProductReviewsSnippetFrag)
	r.Get("/products/{productID}/tabs/{tab}", ProductTabFrag)
	r.Get("/templates", TemplatesHandler)
	r.Get("/guides", GuidesHandler)
	r.Post("/cart/items", CartItemCreateHandler)
	r.Get("/account", AccountHandler)
	// Fragment endpoints (htmx)
	r.Get("/frags/compare/sku-table", CompareSKUTableFrag)
	r.Get("/frags/guides/latest", LatestGuidesFrag)
	// Modal demo fragment (htmx)
	r.Get("/modals/demo", DemoModalHandler)

	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("web listening on %s (devMode=%v)", addr, devMode)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %v", err)
	}
}

func tmplFuncMapFor(getT func() *template.Template) template.FuncMap {
	return template.FuncMap{
		"now":  time.Now,
		"nowf": func(layout string) string { return time.Now().Format(layout) },
		"tlang": func(lang, key string) string {
			if i18nBundle == nil {
				return key
			}
			return i18nBundle.T(lang, key)
		},
		"fmtDate":       func(ts time.Time, lang string) string { return format.FmtDate(ts, lang) },
		"fmtMoney":      func(amount int64, currency, lang string) string { return format.FmtCurrency(amount, currency, lang) },
		"materialLabel": materialLabel,
		"shapeLabel":    shapeLabel,
		"sizeLabel":     sizeLabel,
		"ratingStars":   ratingStars,
		"seq": func(n int) []int {
			if n < 0 {
				n = 0
			}
			s := make([]int, n)
			for i := range s {
				s[i] = i
			}
			return s
		},
		// dict builds a string-keyed map for component props
		"dict": func(v ...any) map[string]any {
			m := map[string]any{}
			for i := 0; i+1 < len(v); i += 2 {
				k := fmt.Sprint(v[i])
				m[k] = v[i+1]
			}
			return m
		},
		// list returns a slice of the arguments
		"list": func(v ...any) []any { return v },
		// safe marks a string as trusted HTML. Use sparingly.
		"safe": func(s string) template.HTML { return template.HTML(s) },
		// slot executes another template by name, passing data, and returns trusted HTML
		"slot": func(name string, data any) template.HTML {
			t := getT()
			if t == nil || name == "" {
				return ""
			}
			var buf bytes.Buffer
			if err := t.ExecuteTemplate(&buf, name, data); err != nil {
				// render an HTML comment with the error to aid debugging without breaking page
				return template.HTML("<!-- slot '" + template.HTMLEscapeString(name) + "' error: " + template.HTMLEscapeString(err.Error()) + " -->")
			}
			return template.HTML(buf.String())
		},
	}
}

func parseTemplates() (*template.Template, error) {
	// create root template and bind funcMap that can access it
	root := template.New("_root")
	funcMap := tmplFuncMapFor(func() *template.Template { return root })
	root = root.Funcs(funcMap)
	// Recursively discover and parse all .tmpl files. Note: ParseGlob doesn't support **.
	var files []string
	if err := filepath.WalkDir(templatesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".tmpl") {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no templates found under %s", templatesDir)
	}
	return root.ParseFiles(files...)
}

// parsePageTemplates builds a template set with the shared layout/partials and one page.
func parsePageTemplates(page string) (*template.Template, error) {
	root := template.New("_root")
	funcMap := tmplFuncMapFor(func() *template.Template { return root })
	root = root.Funcs(funcMap)
	var files []string
	// layouts
	_ = filepath.WalkDir(filepath.Join(templatesDir, "layouts"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".tmpl") {
			files = append(files, path)
		}
		return nil
	})
	// partials
	_ = filepath.WalkDir(filepath.Join(templatesDir, "partials"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".tmpl") {
			files = append(files, path)
		}
		return nil
	})
	// page
	files = append(files, filepath.Join(templatesDir, "pages", page+".tmpl"))
	return root.ParseFiles(files...)
}

// renderTemplate executes a named template (partial/fragment) without the base layout.
func renderTemplate(w http.ResponseWriter, r *http.Request, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var t *template.Template
	if devMode {
		tc, err := parseTemplates()
		if err != nil {
			http.Error(w, fmt.Sprintf("template parse error: %v", err), http.StatusInternalServerError)
			return
		}
		t = tc
	} else {
		t = tmplCache
	}
	if t == nil {
		http.Error(w, "template not initialized", http.StatusInternalServerError)
		return
	}
	if err := t.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, fmt.Sprintf("template exec error: %v", err), http.StatusInternalServerError)
		return
	}
}

// render executes the base layout. In dev mode, templates are reparsed on each request.
func render(w http.ResponseWriter, r *http.Request, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var t *template.Template
	if devMode {
		tc, err := parseTemplates()
		if err != nil {
			http.Error(w, fmt.Sprintf("template parse error: %v", err), http.StatusInternalServerError)
			return
		}
		t = tc
	} else {
		t = tmplCache
	}
	if t == nil {
		http.Error(w, "template not initialized", http.StatusInternalServerError)
		return
	}
	if err := t.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, fmt.Sprintf("template exec error: %v", err), http.StatusInternalServerError)
		return
	}
}

// renderPage executes the base layout with page-specific content definitions.
func renderPage(w http.ResponseWriter, r *http.Request, page string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var t *template.Template
	if devMode {
		var err error
		t, err = parsePageTemplates(page)
		if err != nil {
			http.Error(w, fmt.Sprintf("template parse error: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		pageTmplMu.RLock()
		t = pageTmplCache[page]
		pageTmplMu.RUnlock()
		if t == nil {
			var err error
			t, err = parsePageTemplates(page)
			if err != nil {
				http.Error(w, fmt.Sprintf("template parse error: %v", err), http.StatusInternalServerError)
				return
			}
			pageTmplMu.Lock()
			pageTmplCache[page] = t
			pageTmplMu.Unlock()
		}
	}
	if t == nil {
		http.Error(w, "template not initialized", http.StatusInternalServerError)
		return
	}
	if err := t.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, fmt.Sprintf("template exec error: %v", err), http.StatusInternalServerError)
		return
	}
}

// HomeHandler renders the landing page.
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	vm := handlersPkg.BuildHomeData(lang)
	// augment common layout data
	vm.Path = r.URL.Path
	vm.Nav = nav.Build(vm.Path)
	vm.Breadcrumbs = nav.Breadcrumbs(vm.Path)
	vm.Analytics = handlersPkg.LoadAnalyticsFromEnv()
	if i18nBundle != nil {
		vm.SEO.Title = i18nBundle.T(lang, "home.seo.title")
		vm.SEO.Description = i18nBundle.T(lang, "home.seo.description")
	}
	// Canonical + OG URL/Site
	vm.SEO.Canonical = absoluteURL(r)
	vm.SEO.OG.URL = vm.SEO.Canonical
	vm.SEO.OG.SiteName = i18nOrDefault(lang, "brand.name", "Hanko Field")
	vm.SEO.Alternates = buildAlternates(r)
	// Default JSON-LD (Organization + WebSite)
	siteURL := siteBaseURL(r)
	org := seo.Organization(i18nOrDefault(lang, "brand.name", "Hanko Field"), siteURL, "")
	ws := seo.WebSite(i18nOrDefault(lang, "brand.name", "Hanko Field"), siteURL, siteURL+"/search?q=")
	vm.SEO.JSONLD = []string{seo.JSON(org), seo.JSON(ws)}
	// Add representative Product + Articles JSON-LD
	// Product
	prod := seo.Product(i18nOrDefault(lang, "home.compare.col.name", "Name")+": Classic Round", i18nOrDefault(lang, "home.seo.description", "Custom stamps and seals"), siteURL+"/shop", "", "T-100")
	vm.SEO.JSONLD = append(vm.SEO.JSONLD, seo.JSON(prod))
	// Articles (latest guides)
	art1 := seo.Article(i18nOrDefault(lang, "home.guides.title", "Latest Guides")+": Materials", siteURL+"/guides/materials", "", "Hanko Field", "2025-01-10")
	art2 := seo.Article(i18nOrDefault(lang, "home.guides.title", "Latest Guides")+": Design Basics", siteURL+"/guides/design-basics", "", "Hanko Field", "2025-01-05")
	vm.SEO.JSONLD = append(vm.SEO.JSONLD, seo.JSON(art1), seo.JSON(art2))
	renderPage(w, r, "home", vm)
}

// Generic page handlers
func ShopHandler(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	vm := handlersPkg.PageData{Title: "Shop", Lang: lang}
	vm.Path = r.URL.Path
	vm.Nav = nav.Build(vm.Path)
	vm.Breadcrumbs = nav.Breadcrumbs(vm.Path)
	vm.Analytics = handlersPkg.LoadAnalyticsFromEnv()
	vm.SEO.Canonical = absoluteURL(r)
	vm.SEO.OG.URL = vm.SEO.Canonical
	vm.SEO.OG.SiteName = i18nOrDefault(lang, "brand.name", "Hanko Field")
	vm.SEO.Alternates = buildAlternates(r)
	// Build initial results server-side to avoid extra HTMX request on first load
	vm.Shop = buildShopProps(lang, r.URL.Query())
	renderPage(w, r, "shop", vm)
}

// --- Product detail page ---

func ProductDetailHandler(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	productID := chi.URLParam(r, "productID")
	detail, ok := productDetailFor(lang, productID)
	if !ok {
		http.NotFound(w, r)
		return
	}

	vm := handlersPkg.PageData{Title: detail.Product.Name, Lang: lang}
	vm.Path = r.URL.Path
	vm.Nav = nav.Build(vm.Path)
	vm.Breadcrumbs = nav.Breadcrumbs(vm.Path)
	vm.Analytics = handlersPkg.LoadAnalyticsFromEnv()
	vm.Product = detail

	brand := i18nOrDefault(lang, "brand.name", "Hanko Field")
	if detail.Product.Name != "" {
		vm.SEO.Title = fmt.Sprintf("%s | %s", detail.Product.Name, brand)
	} else {
		vm.SEO.Title = brand
	}
	if detail.Lead != "" {
		vm.SEO.Description = detail.Lead
	} else if len(detail.Description) > 0 {
		vm.SEO.Description = detail.Description[0]
	}
	vm.SEO.Canonical = absoluteURL(r)
	vm.SEO.OG.URL = vm.SEO.Canonical
	vm.SEO.OG.SiteName = brand
	vm.SEO.OG.Type = "product"
	vm.SEO.Alternates = buildAlternates(r)
	vm.SEO.OG.Title = vm.SEO.Title
	vm.SEO.OG.Description = vm.SEO.Description
	vm.SEO.Twitter.Card = "summary_large_image"

	if mediaID := detail.DefaultMedia; mediaID != "" {
		if media, ok := mediaByID(detail, mediaID); ok {
			vm.SEO.OG.Image = media.Src
			vm.SEO.Twitter.Image = media.Src
		}
	}
	if vm.SEO.OG.Image == "" && len(detail.Media) > 0 {
		vm.SEO.OG.Image = detail.Media[0].Src
		vm.SEO.Twitter.Image = detail.Media[0].Src
	}

	siteURL := siteBaseURL(r)
	productSchema := seo.Product(detail.Product.Name, vm.SEO.Description, vm.SEO.Canonical, vm.SEO.OG.Image, detail.Product.ID)
	crumbs := make([]seo.BreadcrumbItem, 0, len(vm.Breadcrumbs))
	for _, c := range vm.Breadcrumbs {
		label := c.Label
		if label == "" && c.LabelKey != "" {
			label = i18nOrDefault(lang, c.LabelKey, c.LabelKey)
		}
		if label == "" {
			label = c.Href
		}
		crumbs = append(crumbs, seo.BreadcrumbItem{
			Name: label,
			Item: siteURL + c.Href,
		})
	}
	vm.SEO.JSONLD = []string{
		seo.JSON(productSchema),
		seo.JSON(seo.BreadcrumbList(crumbs)),
	}

	renderPage(w, r, "product_detail", vm)
}

func ProductGalleryFrag(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	productID := chi.URLParam(r, "productID")
	detail, ok := productDetailFor(lang, productID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	activeID := strings.TrimSpace(r.URL.Query().Get("media"))
	if activeID == "" {
		activeID = detail.DefaultMedia
	}
	if activeID == "" && len(detail.Media) > 0 {
		activeID = detail.Media[0].ID
	}
	active, _ := mediaByID(detail, activeID)
	props := map[string]any{
		"Lang":        lang,
		"Detail":      detail,
		"MediaList":   detail.Media,
		"Active":      active.ID,
		"ActiveMedia": active,
	}
	renderTemplate(w, r, "frag_product_gallery", props)
}

func ProductGalleryModalFrag(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	productID := chi.URLParam(r, "productID")
	detail, ok := productDetailFor(lang, productID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	mediaID := strings.TrimSpace(r.URL.Query().Get("media"))
	if mediaID == "" {
		mediaID = detail.DefaultMedia
	}
	media, ok := mediaByID(detail, mediaID)
	if !ok && len(detail.Media) > 0 {
		media = detail.Media[0]
		ok = true
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
	props := map[string]any{
		"Lang":   lang,
		"Detail": detail,
		"Media":  media,
	}
	renderTemplate(w, r, "frag_product_gallery_modal", props)
}

func ProductReviewsSnippetFrag(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	productID := chi.URLParam(r, "productID")
	detail, ok := productDetailFor(lang, productID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	props := map[string]any{
		"Lang":    lang,
		"Detail":  detail,
		"Reviews": detail.Reviews,
		"Product": detail.Product,
		"Stars":   detail.ReviewStars,
	}
	renderTemplate(w, r, "frag_product_reviews_snippet", props)
}

func ProductTabFrag(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	productID := chi.URLParam(r, "productID")
	detail, ok := productDetailFor(lang, productID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	tab := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "tab")))
	switch tab {
	case "description", "specs", "reviews", "faq":
	default:
		tab = "description"
	}
	props := map[string]any{
		"Lang":   lang,
		"Tab":    tab,
		"Detail": detail,
	}
	renderTemplate(w, r, "frag_product_tab", props)
}

func CartItemCreateHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}
	lang := mw.Lang(r)
	productID := strings.TrimSpace(r.FormValue("product_id"))
	detail, _ := productDetailFor(lang, productID)
	productName := strings.TrimSpace(r.FormValue("product_name"))
	if detail != nil {
		productName = detail.Product.Name
		if productID == "" {
			productID = detail.Product.ID
		}
	}
	if productName == "" {
		if lang == "ja" {
			productName = "商品"
		} else {
			productName = "Item"
		}
	}
	qty := 1
	if v := strings.TrimSpace(r.FormValue("quantity")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			qty = n
		}
	}
	props := map[string]any{
		"Lang":        lang,
		"ProductID":   productID,
		"ProductName": productName,
		"Quantity":    qty,
	}
	w.Header().Set("HX-Trigger", "cart:add")
	renderTemplate(w, r, "frag_cart_add_confirmation", props)
}

// --- Shop listing (fragment) ---

// Product represents a shop list item.
type Product struct {
	ID           string
	Name         string
	Material     string // wood, rubber, metal
	Shape        string // round, square, rect
	Size         string // small, medium, large
	PriceJPY     int64  // price in JPY (for fmtMoney)
	CompareAtJPY int64  // optional list price for sales
	Sale         bool
	InStock      bool
}

// ProductMedia represents a media asset shown in the gallery.
type ProductMedia struct {
	ID        string
	Kind      string // image or video
	Src       string
	Thumbnail string
	Alt       string
	Width     int
	Height    int
	Poster    string // optional for videos
}

// ProductOptionChoice models a selectable option value.
type ProductOptionChoice struct {
	Value    string
	Label    string
	Subtitle string
	Selected bool
}

// ProductOptionGroup models a group of options (e.g. material finish).
type ProductOptionGroup struct {
	Name     string
	Label    string
	Required bool
	Help     string
	Choices  []ProductOptionChoice
}

// ProductSpec represents a key/value specification entry.
type ProductSpec struct {
	Label string
	Value string
}

// ProductFAQ represents a question/answer pair for the FAQ tab.
type ProductFAQ struct {
	Question string
	Answer   string
}

// ReviewSnippet represents a short review quote.
type ReviewSnippet struct {
	Author string
	Rating float64
	Quote  string
	Date   time.Time
	Locale string
}

// RatingBreakdown shows the percentage share for a star rating bucket.
type RatingBreakdown struct {
	Stars   int
	Percent int
}

// ReviewSummary aggregates review data for a product.
type ReviewSummary struct {
	Average    float64
	Count      int
	Highlights []string
	Breakdown  []RatingBreakdown
	Latest     []ReviewSnippet
	Link       string
}

// ProductDetail aggregates all data needed for the product detail view.
type ProductDetail struct {
	Product           Product
	Subtitle          string
	Description       []string
	Highlights        []string
	Lead              string
	Availability      string // e.g. "in-stock", "backorder"
	AvailabilityLabel string
	AvailabilityTone  string
	ShippingEstimate  string
	PriceNote         string
	DefaultMedia      string
	Media             []ProductMedia
	Options           []ProductOptionGroup
	Specs             []ProductSpec
	FAQ               []ProductFAQ
	Reviews           ReviewSummary
	ReviewStars       []string
	Recommended       []Product
	MinQty            int
	MaxQty            int
}

// productData returns a seed catalog suitable for demo listing and filtering.
func productData(lang string) []Product {
	mats := []string{"wood", "rubber", "metal"}
	shapesKeys := []string{"round", "square", "rect"}
	sizesKeys := []string{"small", "medium", "large"}
	var out []Product
	id := 1000
	base := map[string]int64{"wood": 1800, "rubber": 1200, "metal": 2400} // in JPY
	for _, m := range mats {
		for _, sh := range shapesKeys {
			for i, sz := range sizesKeys {
				price := base[m] + int64(i*300)
				sale := (m == "metal" && sh == "rect") || (m == "rubber" && sz == "small")
				name := ""
				if lang == "ja" {
					name = fmt.Sprintf("%s %s", shapeLabel(lang, sh), sizeLabel(lang, sz))
				} else {
					name = fmt.Sprintf("%s %s", shapeLabel(lang, sh), sizeLabel(lang, sz))
				}
				compare := int64(0)
				if sale {
					compare = price + 400
				}
				out = append(out, Product{
					ID:           fmt.Sprintf("P-%d", id),
					Name:         name,
					Material:     m,
					Shape:        sh,
					Size:         sz,
					PriceJPY:     price,
					CompareAtJPY: compare,
					Sale:         sale,
					InStock:      true,
				})
				id++
			}
		}
	}
	return out
}

func findProduct(lang, id string) (Product, bool) {
	for _, p := range productData(lang) {
		if p.ID == id {
			return p, true
		}
	}
	return Product{}, false
}

// buildShopProps constructs view props for shop results from query params.
func buildShopProps(lang string, q url.Values) map[string]any {
	material := strings.TrimSpace(strings.ToLower(q.Get("material")))
	shape := strings.TrimSpace(strings.ToLower(q.Get("shape")))
	size := strings.TrimSpace(strings.ToLower(q.Get("size")))
	saleOnly := strings.TrimSpace(strings.ToLower(q.Get("sale")))
	priceMinStr := strings.TrimSpace(q.Get("price_min"))
	priceMaxStr := strings.TrimSpace(q.Get("price_max"))
	pageStr := strings.TrimSpace(q.Get("page"))
	perStr := strings.TrimSpace(q.Get("per"))

	// Parse numeric params (ignore invalid values rather than converting to 0)
	priceMin, priceMinOK := 0, false
	if priceMinStr != "" {
		if v, err := strconv.Atoi(priceMinStr); err == nil && v >= 0 {
			priceMin = v
			priceMinOK = true
		}
	}
	priceMax, priceMaxOK := 0, false
	if priceMaxStr != "" {
		if v, err := strconv.Atoi(priceMaxStr); err == nil && v >= 0 {
			priceMax = v
			priceMaxOK = true
		}
	}
	page := 1
	if pageStr != "" {
		if v, err := strconv.Atoi(pageStr); err == nil && v > 0 {
			page = v
		}
	}
	per := 9
	if perStr != "" {
		if v, err := strconv.Atoi(perStr); err == nil {
			if v < 1 {
				v = 9
			}
			if v > 48 {
				v = 48
			}
			per = v
		}
	}

	// Filter products
	var filtered []Product
	for _, p := range productData(lang) {
		if material != "" && material != "all" && p.Material != material {
			continue
		}
		if shape != "" && shape != "all" && p.Shape != shape {
			continue
		}
		if size != "" && size != "all" && p.Size != size {
			continue
		}
		if (saleOnly == "1" || saleOnly == "true") && !p.Sale {
			continue
		}
		if priceMinOK && p.PriceJPY < int64(priceMin) {
			continue
		}
		if priceMaxOK && p.PriceJPY > int64(priceMax) {
			continue
		}
		filtered = append(filtered, p)
	}

	total := len(filtered)
	start := (page - 1) * per
	if start > total {
		start = total
	}
	end := start + per
	if end > total {
		end = total
	}
	window := filtered[start:end]

	// Build base query string excluding page for pager links
	base := make(url.Values)
	if material != "" {
		base.Set("material", material)
	}
	if shape != "" {
		base.Set("shape", shape)
	}
	if size != "" {
		base.Set("size", size)
	}
	if saleOnly != "" {
		base.Set("sale", saleOnly)
	}
	if priceMinOK {
		base.Set("price_min", strconv.Itoa(priceMin))
	}
	if priceMaxOK {
		base.Set("price_max", strconv.Itoa(priceMax))
	}
	base.Set("per", strconv.Itoa(per))

	hasPrev := page > 1
	hasNext := end < total

	props := map[string]any{
		"Products": window,
		"Lang":     lang,
		"Total":    total,
		"Page":     page,
		"Per":      per,
		"HasPrev":  hasPrev,
		"HasNext":  hasNext,
		"Prev":     page - 1,
		"Next":     page + 1,
		"BaseQS":   base.Encode(),
	}
	return props
}

// ShopTableFrag renders the product grid with applied filters and pagination.
func ShopTableFrag(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	props := buildShopProps(lang, r.URL.Query())
	// Push URL to the full page (/shop?...) to avoid landing on the fragment URL when reloading or sharing
	if baseQS, ok := props["BaseQS"].(string); ok {
		if page, ok2 := props["Page"].(int); ok2 {
			push := "/shop?" + baseQS + "&page=" + strconv.Itoa(page)
			w.Header().Set("HX-Push-Url", push)
		}
	}
	renderTemplate(w, r, "frag_shop_table", props)
}

func TemplatesHandler(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	vm := handlersPkg.PageData{Title: "Templates", Lang: lang}
	vm.Path = r.URL.Path
	vm.Nav = nav.Build(vm.Path)
	vm.Breadcrumbs = nav.Breadcrumbs(vm.Path)
	vm.Analytics = handlersPkg.LoadAnalyticsFromEnv()
	vm.SEO.Canonical = absoluteURL(r)
	vm.SEO.OG.URL = vm.SEO.Canonical
	vm.SEO.OG.SiteName = i18nOrDefault(lang, "brand.name", "Hanko Field")
	vm.SEO.Alternates = buildAlternates(r)
	renderPage(w, r, "templates", vm)
}

func GuidesHandler(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	vm := handlersPkg.PageData{Title: "Guides", Lang: lang}
	vm.Path = r.URL.Path
	vm.Nav = nav.Build(vm.Path)
	vm.Breadcrumbs = nav.Breadcrumbs(vm.Path)
	vm.Analytics = handlersPkg.LoadAnalyticsFromEnv()
	vm.SEO.Canonical = absoluteURL(r)
	vm.SEO.OG.URL = vm.SEO.Canonical
	vm.SEO.OG.SiteName = i18nOrDefault(lang, "brand.name", "Hanko Field")
	vm.SEO.Alternates = buildAlternates(r)
	renderPage(w, r, "guides", vm)
}

func AccountHandler(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	vm := handlersPkg.PageData{Title: "Account", Lang: lang}
	vm.Path = r.URL.Path
	vm.Nav = nav.Build(vm.Path)
	vm.Breadcrumbs = nav.Breadcrumbs(vm.Path)
	vm.Analytics = handlersPkg.LoadAnalyticsFromEnv()
	vm.SEO.Canonical = absoluteURL(r)
	vm.SEO.OG.URL = vm.SEO.Canonical
	vm.SEO.OG.SiteName = i18nOrDefault(lang, "brand.name", "Hanko Field")
	vm.SEO.Alternates = buildAlternates(r)
	renderPage(w, r, "account", vm)
}

// DemoModalHandler returns a demo modal fragment for HTMX insertion.
func DemoModalHandler(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	_ = lang // reserved for future i18n of title/buttons
	props := map[string]any{
		"ID":    "demo-modal",
		"Title": "Demo Modal",
		"Body":  "This is a shared modal opened via HTMX. Press ESC or click the overlay to close.",
		// No FooterTmpl provided → default Close button with data-modal-close
	}
	renderTemplate(w, r, "c_modal", props)
}

// absoluteURL builds an absolute URL for the current request path, using X-Forwarded-Proto if present.
func absoluteURL(r *http.Request) string {
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := r.Host
	if host == "" {
		host = "localhost"
	}
	return scheme + "://" + host + r.URL.Path
}

// siteBaseURL returns the base site URL (scheme+host) inferred from the request.
func siteBaseURL(r *http.Request) string {
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := r.Host
	if host == "" {
		host = "localhost"
	}
	return scheme + "://" + host
}

// buildAlternates produces hreflang alternates for supported languages using the current path.
func buildAlternates(r *http.Request) []struct{ Href, Hreflang string } {
	var out []struct{ Href, Hreflang string }
	if i18nBundle == nil {
		return out
	}
	base := siteBaseURL(r)
	path := r.URL.Path
	supported := i18nBundle.Supported()
	for _, l := range supported {
		href := base + path + "?hl=" + l
		out = append(out, struct{ Href, Hreflang string }{Href: href, Hreflang: l})
	}
	// x-default points to fallback
	out = append(out, struct{ Href, Hreflang string }{Href: base + path, Hreflang: "x-default"})
	return out
}

func i18nOrDefault(lang, key, def string) string {
	if i18nBundle == nil {
		return def
	}
	v := i18nBundle.T(lang, key)
	if v == "" || v == key {
		return def
	}
	return v
}

// --- Fragments and supporting types ---

// SKU represents a simple product option for comparison.
type SKU struct {
	ID    string
	Name  string
	Shape string
	Size  string
	Price string // display price (e.g., "$12")
}

// skuData returns the canonical list of SKUs for comparison.
func skuData(lang string) []SKU {
	// Static seed data; in the future fetch from API/DB.
	// Translate names lightly depending on lang.
	round := map[string]string{"en": "Classic Round", "ja": "丸型クラシック"}
	square := map[string]string{"en": "Square Logo", "ja": "角形ロゴ"}
	rect := map[string]string{"en": "Business Seal", "ja": "ビジネス印"}
	tl := func(m map[string]string, l string) string {
		if v, ok := m[l]; ok {
			return v
		}
		return m["en"]
	}
	return []SKU{
		{ID: "T-100", Name: tl(round, lang), Shape: "round", Size: "small", Price: "$12"},
		{ID: "T-105", Name: tl(round, lang), Shape: "round", Size: "medium", Price: "$14"},
		{ID: "T-110", Name: tl(round, lang), Shape: "round", Size: "large", Price: "$18"},
		{ID: "T-220", Name: tl(square, lang), Shape: "square", Size: "small", Price: "$16"},
		{ID: "T-225", Name: tl(square, lang), Shape: "square", Size: "medium", Price: "$18"},
		{ID: "T-310", Name: tl(rect, lang), Shape: "rect", Size: "large", Price: "$24"},
	}
}

// CompareSKUTableFrag renders the SKU comparison table with optional filters.
func CompareSKUTableFrag(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	shape := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("shape")))
	size := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("size")))

	// Build rows with filters applied
	cols := []map[string]any{
		{"Key": "id", "Label": "ID", "Align": "left"},
		{"Key": "name", "Label": i18nOrDefault(lang, "home.compare.col.name", "Name"), "Align": "left"},
		{"Key": "shape", "Label": i18nOrDefault(lang, "home.compare.col.shape", "Shape"), "Align": "left"},
		{"Key": "size", "Label": i18nOrDefault(lang, "home.compare.col.size", "Size"), "Align": "left"},
		{"Key": "price", "Label": i18nOrDefault(lang, "home.compare.col.price", "Price"), "Align": "right"},
	}
	var rows []map[string]string
	for _, s := range skuData(lang) {
		if shape != "" && s.Shape != shape {
			continue
		}
		if size != "" && s.Size != size {
			continue
		}
		rows = append(rows, map[string]string{"id": s.ID, "name": s.Name, "shape": s.Shape, "size": s.Size, "price": s.Price})
	}

	// ETag simple hash of inputs
	etag := etagFor("sku:", lang, shape, size)
	if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.Header().Set("ETag", etag)

	props := map[string]any{
		"Columns": cols,
		"Rows":    rows,
		"Shape":   shape,
		"Size":    size,
		"Lang":    lang,
	}
	renderTemplate(w, r, "frag_compare_sku_table", props)
}

// LatestGuidesFrag renders a small set of localized guide cards.
func LatestGuidesFrag(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	type Guide struct{ Title, URL, Excerpt, Date string }
	// Seed localized guide list
	var guides []Guide
	if lang == "ja" {
		guides = []Guide{
			{Title: "はんこ素材の選び方", URL: "/guides/materials", Excerpt: "用途別に最適な素材を解説します。", Date: "2025-01-10"},
			{Title: "印影デザインの基本", URL: "/guides/design-basics", Excerpt: "読みやすさと個性のバランスを学びます。", Date: "2025-01-05"},
			{Title: "サイズ比較ガイド", URL: "/guides/size-guide", Excerpt: "丸・角・楕円のサイズ感を比較。", Date: "2024-12-20"},
		}
	} else {
		guides = []Guide{
			{Title: "How to Choose Materials", URL: "/guides/materials", Excerpt: "Pick the right material for your use.", Date: "2025-01-10"},
			{Title: "Seal Design Basics", URL: "/guides/design-basics", Excerpt: "Balance legibility with personality.", Date: "2025-01-05"},
			{Title: "Size Comparison Guide", URL: "/guides/size-guide", Excerpt: "Compare round, square, and rectangular.", Date: "2024-12-20"},
		}
	}

	// Simple 2-minute cache
	etag := etagFor("guides:", lang)
	if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=120")
	w.Header().Set("ETag", etag)

	props := map[string]any{"Guides": guides}
	renderTemplate(w, r, "frag_guides_latest", props)
}

// etagFor builds a weak pseudo-ETag from inputs.
func etagFor(prefix string, parts ...string) string {
	// very small non-crypto hash
	h := 1469598103934665603 ^ uint64(len(prefix))
	for _, s := range parts {
		for i := 0; i < len(s); i++ {
			h ^= uint64(s[i])
			h *= 1099511628211
		}
	}
	return fmt.Sprintf("W/\"%s%x\"", prefix, h)
}
