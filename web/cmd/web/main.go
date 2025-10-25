package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"finitefield.org/hanko-web/internal/cms"
	"finitefield.org/hanko-web/internal/format"
	handlersPkg "finitefield.org/hanko-web/internal/handlers"
	"finitefield.org/hanko-web/internal/i18n"
	mw "finitefield.org/hanko-web/internal/middleware"
	"finitefield.org/hanko-web/internal/nav"
	"finitefield.org/hanko-web/internal/seo"
	"finitefield.org/hanko-web/internal/status"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"golang.org/x/net/html"
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

var (
	cmsClient    *cms.Client
	statusClient *status.Client

	guidePersonaOrder  = []string{"maker", "manager", "newcomer", "creative"}
	guideCategoryOrder = []string{"howto", "culture", "policy", "faq", "news", "other"}

	guideHTMLPolicy = func() *bluemonday.Policy {
		p := bluemonday.UGCPolicy()
		p.AllowAttrs("class").OnElements("p", "pre", "code", "span", "a", "table", "thead", "tbody", "tr", "th", "td", "h2", "h3", "h4", "blockquote", "figure", "figcaption", "img", "ul", "ol", "li")
		p.AllowAttrs("id").OnElements("h2", "h3", "h4")
		p.AllowAttrs("href", "title", "target", "rel").OnElements("a")
		p.AllowAttrs("src", "alt", "title", "width", "height", "loading").OnElements("img")
		p.AllowURLSchemes("http", "https", "mailto", "tel")
		return p
	}()

	markdownRenderer = goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Linkify, extension.Strikethrough, extension.Table),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(
			gmhtml.WithHardWraps(),
			gmhtml.WithXHTML(),
		),
	)

	contentRenderCache = struct {
		mu    sync.RWMutex
		items map[string]renderedContentEntry
	}{
		items: map[string]renderedContentEntry{},
	}

	contentRenderTTL = 5 * time.Minute
	jstLocation      = time.FixedZone("JST", 9*60*60)
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

	cmsClient = cms.NewClient(os.Getenv("HANKO_WEB_API_BASE_URL"))
	contentDirEnv := strings.TrimSpace(os.Getenv("HANKO_WEB_CONTENT_DIR"))
	if contentDirEnv == "" {
		base := filepath.Dir(filepath.Clean(templatesDir))
		if base == "." || base == "" {
			contentDirEnv = "content"
		} else {
			contentDirEnv = filepath.Join(base, "content")
		}
	}
	if cmsClient != nil {
		cmsClient.SetContentDir(contentDirEnv)
	}
	statusClient = status.NewClient(os.Getenv("HANKO_WEB_STATUS_URL"))

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
	r.Get("/design/new", DesignNewHandler)
	r.Get("/design/editor", DesignEditorHandler)
	r.MethodFunc(http.MethodGet, "/design/editor/form", DesignEditorFormFrag)
	r.MethodFunc(http.MethodPost, "/design/editor/form", DesignEditorFormFrag)
	r.Get("/design/editor/preview", DesignEditorPreviewFrag)
	r.Get("/design/preview", DesignPreviewHandler)
	r.Get("/design/preview/image", DesignPreviewImageFrag)
	r.Get("/design/share/modal", DesignShareModal)
	r.MethodFunc(http.MethodPost, "/design/share/link", DesignShareLinkHandler)
	r.Get("/design/versions", DesignVersionsHandler)
	r.Get("/design/versions/table", DesignVersionsTableFrag)
	r.Get("/design/versions/preview", DesignVersionsPreviewFrag)
	r.Get("/design/versions/{versionID}/rollback/modal", DesignVersionRollbackModal)
	r.MethodFunc(http.MethodPost, "/design/versions/{versionID}/rollback", DesignVersionRollbackHandler)
	r.Get("/design/ai", DesignAISuggestionsHandler)
	r.Get("/design/ai/table", DesignAISuggestionTableFrag)
	r.Get("/design/ai/preview", DesignAISuggestionPreviewFrag)
	r.MethodFunc(http.MethodPost, "/design/ai/suggestions/{suggestionID}/accept", DesignAISuggestionAcceptHandler)
	r.MethodFunc(http.MethodPost, "/design/ai/suggestions/{suggestionID}/reject", DesignAISuggestionRejectHandler)
	// Design editor modal endpoints (new + legacy aliases)
	r.Get("/modal/pick/font", ModalPickFont)
	r.Get("/modal/pick/template", ModalPickTemplate)
	r.MethodFunc(http.MethodGet, "/modal/kanji-map", ModalKanjiMap)
	r.MethodFunc(http.MethodPost, "/modal/kanji-map", ModalKanjiMap)
	// Legacy paths kept temporarily for backwards compatibility during migration.
	r.Get("/design/editor/fonts/modal", ModalPickFont)
	r.Get("/design/editor/templates/modal", ModalPickTemplate)
	r.Get("/design/editor/kanji/modal", ModalKanjiMap)
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
	r.Get("/templates/table", TemplatesTableFrag)
	r.Get("/templates/{templateID}", TemplateDetailHandler)
	r.Get("/guides", GuidesHandler)
	r.Get("/guides/{slug}", GuideDetailHandler)
	r.Get("/content/{slug}", ContentPageHandler)
	r.Get("/legal/{slug}", LegalPageHandler)
	r.Get("/status", StatusHandler)
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

func DesignNewHandler(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	title := i18nOrDefault(lang, "design.new.seo.title", "Choose your starting point")
	desc := i18nOrDefault(lang, "design.new.seo.description", "Pick how you want to begin your seal design and continue to the editor with tailored guidance.")

	vm := handlersPkg.PageData{Title: title, Lang: lang}
	vm.Path = r.URL.Path
	vm.Nav = nav.Build(vm.Path)
	vm.Breadcrumbs = nav.Breadcrumbs(vm.Path)
	vm.Analytics = handlersPkg.LoadAnalyticsFromEnv()
	vm.Design = map[string]any{
		"Selection": buildDesignSelectionProps(lang),
	}

	brand := i18nOrDefault(lang, "brand.name", "Hanko Field")
	vm.SEO.Title = fmt.Sprintf("%s | %s", title, brand)
	vm.SEO.Description = desc
	vm.SEO.Canonical = absoluteURL(r)
	vm.SEO.OG.URL = vm.SEO.Canonical
	vm.SEO.OG.SiteName = brand
	vm.SEO.OG.Type = "website"
	vm.SEO.OG.Title = vm.SEO.Title
	vm.SEO.OG.Description = vm.SEO.Description
	vm.SEO.Twitter.Card = "summary_large_image"
	vm.SEO.Alternates = buildAlternates(r)

	renderPage(w, r, "design_new", vm)
}

func DesignEditorHandler(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	view := buildDesignEditorView(lang, r.URL.Query())

	pageTitle := editorCopy(lang, "デザインエディタ", "Design editor")
	desc := editorCopy(lang, "左右分割の編集フォームとライブプレビューで印影を整えます。", "Dual-pane editor with form controls and live seal preview.")

	vm := handlersPkg.PageData{Title: pageTitle, Lang: lang}
	vm.Path = r.URL.Path
	vm.Nav = nav.Build(vm.Path)
	vm.Breadcrumbs = nav.Breadcrumbs(vm.Path)
	vm.Analytics = handlersPkg.LoadAnalyticsFromEnv()
	vm.Design = view

	brand := i18nOrDefault(lang, "brand.name", "Hanko Field")
	vm.SEO.Title = fmt.Sprintf("%s | %s", pageTitle, brand)
	vm.SEO.Description = desc
	vm.SEO.Canonical = absoluteURL(r)
	vm.SEO.OG.URL = vm.SEO.Canonical
	vm.SEO.OG.SiteName = brand
	vm.SEO.OG.Type = "website"
	vm.SEO.OG.Title = vm.SEO.Title
	vm.SEO.OG.Description = vm.SEO.Description
	vm.SEO.Twitter.Card = "summary_large_image"
	vm.SEO.Alternates = buildAlternates(r)

	renderPage(w, r, "design_editor", vm)
}

func DesignEditorFormFrag(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("parse form: %v", err), http.StatusBadRequest)
		return
	}
	view := buildDesignEditorView(lang, r.Form)
	push := "/design/editor"
	if view.Query != "" {
		push = push + "?" + view.Query
	}
	w.Header().Set("HX-Push-Url", push)
	triggerPayload := map[string]any{
		"editor:state-updated": map[string]string{
			"query": view.Query,
		},
	}
	if payload, err := json.Marshal(triggerPayload); err == nil {
		w.Header().Set("HX-Trigger", string(payload))
	}
	renderTemplate(w, r, "frag_design_editor_form", view)
}

func DesignEditorPreviewFrag(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	_ = r.ParseForm()
	view := buildDesignEditorView(lang, r.Form)
	view.Toasts = nil
	renderTemplate(w, r, "frag_design_editor_preview", view)
}

// DesignPreviewHandler renders the final design preview page with background and export controls.
func DesignPreviewHandler(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	view := buildDesignPreviewView(lang, r.URL.Query())

	vm := handlersPkg.PageData{Title: view.Title, Lang: lang}
	vm.Path = r.URL.Path
	vm.Nav = nav.Build(vm.Path)
	vm.Breadcrumbs = nav.Breadcrumbs(vm.Path)
	vm.Analytics = handlersPkg.LoadAnalyticsFromEnv()
	vm.DesignPreview = view

	brand := i18nOrDefault(lang, "brand.name", "Hanko Field")
	vm.SEO.Title = fmt.Sprintf("%s | %s", view.Title, brand)
	vm.SEO.Description = editorCopy(lang, "最高解像度の印影とモックアップをエクスポートします。", "Export the final high-resolution seal impression with contextual mockups.")
	vm.SEO.Canonical = absoluteURL(r)
	vm.SEO.OG.URL = vm.SEO.Canonical
	vm.SEO.OG.SiteName = brand
	vm.SEO.OG.Type = "website"
	vm.SEO.OG.Title = vm.SEO.Title
	vm.SEO.OG.Description = vm.SEO.Description
	vm.SEO.Twitter.Card = "summary_large_image"
	vm.SEO.Alternates = buildAlternates(r)

	renderPage(w, r, "design_preview", vm)
}

// DesignPreviewImageFrag returns the dynamic preview fragment responding to background/DPI controls.
func DesignPreviewImageFrag(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form submission", http.StatusBadRequest)
		return
	}
	view := buildDesignPreviewView(lang, r.Form)

	push := "/design/preview"
	if view.Query != "" {
		push = push + "?" + view.Query
	}
	w.Header().Set("HX-Push-Url", push)

	trigger := map[string]any{
		"design-preview:updated": map[string]any{
			"background": view.SelectedBackground,
			"dpi":        view.SelectedDPI,
			"frame":      view.ActiveFrame,
			"grid":       view.ShowGrid,
		},
	}
	if raw, err := json.Marshal(trigger); err == nil {
		w.Header().Set("HX-Trigger", string(raw))
	}
	renderTemplate(w, r, "frag_design_preview_image", view)
}

// ModalPickFont renders the font selection modal fragment.
func ModalPickFont(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	fonts := designEditorFonts(lang)
	selected := strings.TrimSpace(r.URL.Query().Get("font"))
	if len(fonts) > 0 {
		if f, ok := findDesignEditorFont(fonts, selected); ok {
			selected = f.ID
		} else {
			selected = fonts[0].ID
		}
	}
	data := map[string]any{
		"Lang":     lang,
		"Fonts":    fonts,
		"Selected": selected,
	}
	renderTemplate(w, r, "frag_design_editor_fonts_modal", data)
}

// ModalPickTemplate renders the template selection modal fragment.
func ModalPickTemplate(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	templates := designEditorTemplates(lang)
	selected := strings.TrimSpace(r.URL.Query().Get("template"))
	if len(templates) > 0 {
		if t, ok := findDesignEditorTemplate(templates, selected); ok {
			selected = t.ID
		} else {
			selected = templates[0].ID
		}
	}
	data := map[string]any{
		"Lang":      lang,
		"Templates": templates,
		"Selected":  selected,
	}
	renderTemplate(w, r, "frag_design_editor_templates_modal", data)
}

// ModalKanjiMap renders the kanji mapping helper modal. Accepts GET (query) and POST (form) requests.
func ModalKanjiMap(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.Form.Get("name"))
	// Fall back to query param when invoked via GET.
	if name == "" {
		name = strings.TrimSpace(r.URL.Query().Get("name"))
	}

	candidates := kanjiMappingCandidates(lang, name)
	data := map[string]any{
		"Lang":        lang,
		"Name":        name,
		"Candidates":  candidates,
		"HasName":     name != "",
		"HasResults":  len(candidates) > 0,
		"LastUpdated": time.Now(),
	}
	renderTemplate(w, r, "frag_design_editor_kanji_modal", data)
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

// --- Design creation selection data ---

type DesignModeOption struct {
	ID          string
	Icon        string
	Title       string
	Description string
	Badge       string
	Tags        []string
	TagsCSV     string
}

type DesignFilterChip struct {
	ID     string
	Label  string
	Active bool
}

type DesignFeature struct {
	Icon        string
	Title       string
	Description string
}

func buildDesignSelectionProps(lang string) map[string]any {
	options := []DesignModeOption{
		{
			ID:          "text",
			Icon:        "document-text",
			Title:       i18nOrDefault(lang, "design.new.option.text.title", "Type your text"),
			Description: i18nOrDefault(lang, "design.new.option.text.description", "Structured layout with grid, kerning, and engraving rules applied automatically."),
			Badge:       i18nOrDefault(lang, "design.new.option.text.badge", "Most popular"),
			Tags:        []string{"all", "business", "personal"},
		},
		{
			ID:          "upload",
			Icon:        "cloud-upload",
			Title:       i18nOrDefault(lang, "design.new.option.upload.title", "Upload an image"),
			Description: i18nOrDefault(lang, "design.new.option.upload.description", "Use an existing seal scan or artwork. We clean up edges and prepare it for engraving."),
			Badge:       "",
			Tags:        []string{"all", "personal", "legacy"},
		},
		{
			ID:          "logo",
			Icon:        "stamp",
			Title:       i18nOrDefault(lang, "design.new.option.logo.title", "Engrave a logo"),
			Description: i18nOrDefault(lang, "design.new.option.logo.description", "Import vector artwork for precision engraving with compliance checks."),
			Badge:       i18nOrDefault(lang, "design.new.option.logo.badge", "New"),
			Tags:        []string{"all", "business", "export"},
		},
	}

	for i := range options {
		options[i].TagsCSV = strings.Join(options[i].Tags, ",")
	}

	filters := []DesignFilterChip{
		{
			ID:     "all",
			Label:  i18nOrDefault(lang, "design.new.filters.all", "All"),
			Active: true,
		},
		{
			ID:    "business",
			Label: i18nOrDefault(lang, "design.new.filters.business", "Business"),
		},
		{
			ID:    "personal",
			Label: i18nOrDefault(lang, "design.new.filters.personal", "Personal"),
		},
		{
			ID:    "export",
			Label: i18nOrDefault(lang, "design.new.filters.export", "Export"),
		},
		{
			ID:    "legacy",
			Label: i18nOrDefault(lang, "design.new.filters.legacy", "Legacy"),
		},
	}

	features := []DesignFeature{
		{
			Icon:        "sparkles",
			Title:       i18nOrDefault(lang, "design.new.feature.ai.title", "AI-assisted layout"),
			Description: i18nOrDefault(lang, "design.new.feature.ai.description", "We align strokes, balance spacing, and preview drag adjustments instantly."),
		},
		{
			Icon:        "layers",
			Title:       i18nOrDefault(lang, "design.new.feature.templates.title", "Template recommendations"),
			Description: i18nOrDefault(lang, "design.new.feature.templates.description", "Start from proven registrable templates tailored to your use case."),
		},
		{
			Icon:        "shield-check",
			Title:       i18nOrDefault(lang, "design.new.feature.compliance.title", "Compliance guardrails"),
			Description: i18nOrDefault(lang, "design.new.feature.compliance.description", "We flag contrast, size, and text issues before you send to production."),
		},
	}

	help := map[string]string{
		"Title":       i18nOrDefault(lang, "design.new.help.title", "Need help choosing?"),
		"Description": i18nOrDefault(lang, "design.new.help.description", "Watch the four-minute guided tour to see each mode in action."),
		"Href":        "/guides/design-basics",
		"Label":       i18nOrDefault(lang, "design.new.help.label", "Open tutorial"),
	}

	return map[string]any{
		"Lang":           lang,
		"Options":        options,
		"Filters":        filters,
		"Features":       features,
		"Help":           help,
		"DefaultMode":    "text",
		"EditorBasePath": "/design/editor",
		"AnalyticsEvent": "design_mode_select",
	}
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
	brand := i18nOrDefault(lang, "brand.name", "Hanko Field")
	vm.Templates = buildTemplateListProps(lang, r.URL.Query())
	vm.SEO.Title = fmt.Sprintf("%s | %s", i18nOrDefault(lang, "nav.templates", "Templates"), brand)
	vm.SEO.Description = "Browse tested seal layouts with script, shape, and registrability filters. Preview any template before starting a design."
	vm.SEO.Canonical = absoluteURL(r)
	vm.SEO.OG.URL = vm.SEO.Canonical
	vm.SEO.OG.SiteName = brand
	vm.SEO.OG.Title = vm.SEO.Title
	vm.SEO.OG.Description = vm.SEO.Description
	vm.SEO.OG.Type = "website"
	vm.SEO.Twitter.Card = "summary_large_image"
	vm.SEO.Alternates = buildAlternates(r)
	renderPage(w, r, "templates", vm)
}

func TemplateDetailHandler(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	templateID := chi.URLParam(r, "templateID")
	detail, ok := templateDetailFor(lang, templateID)
	if !ok {
		http.NotFound(w, r)
		return
	}

	if mw.IsHTMX(r.Context()) {
		props := map[string]any{
			"Lang":   lang,
			"Detail": detail,
		}
		renderTemplate(w, r, "frag_template_drawer", props)
		return
	}

	vm := handlersPkg.PageData{Title: detail.Template.Name, Lang: lang}
	vm.Path = r.URL.Path
	vm.Nav = nav.Build(vm.Path)
	vm.Breadcrumbs = nav.Breadcrumbs(vm.Path)
	vm.Analytics = handlersPkg.LoadAnalyticsFromEnv()
	vm.Template = detail

	brand := i18nOrDefault(lang, "brand.name", "Hanko Field")
	pageTitle := detail.Template.Name
	if pageTitle == "" {
		pageTitle = i18nOrDefault(lang, "nav.templates", "Templates")
	}
	vm.SEO.Title = fmt.Sprintf("%s | %s", pageTitle, brand)
	if detail.Lead != "" {
		vm.SEO.Description = detail.Lead
	} else if len(detail.Summary) > 0 {
		vm.SEO.Description = detail.Summary[0]
	} else {
		vm.SEO.Description = "Template detail overview."
	}
	vm.SEO.Canonical = absoluteURL(r)
	vm.SEO.OG.URL = vm.SEO.Canonical
	vm.SEO.OG.SiteName = brand
	vm.SEO.OG.Type = "article"
	vm.SEO.OG.Title = vm.SEO.Title
	vm.SEO.OG.Description = vm.SEO.Description
	vm.SEO.Twitter.Card = "summary_large_image"
	vm.SEO.Alternates = buildAlternates(r)

	renderPage(w, r, "template_detail", vm)
}

func TemplatesTableFrag(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	props := buildTemplateListProps(lang, r.URL.Query())
	if baseQS, ok := props["BaseQS"].(string); ok {
		if page, ok2 := props["Page"].(int); ok2 {
			push := "/templates"
			query := baseQS
			if page > 1 {
				if query != "" {
					query += "&"
				}
				query += "page=" + strconv.Itoa(page)
			}
			if query != "" {
				push = "/templates?" + query
			}
			w.Header().Set("HX-Push-Url", push)
		}
	}
	renderTemplate(w, r, "frag_template_table", props)
}

func GuidesHandler(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)

	title := i18nOrDefault(lang, "guides.title", "Guides")
	subtitle := i18nOrDefault(lang, "guides.subtitle", "Step-by-step guides to master Hanko Field.")

	vm := handlersPkg.PageData{Title: title, Lang: lang}
	vm.Path = r.URL.Path
	vm.Nav = nav.Build(vm.Path)
	vm.Breadcrumbs = nav.Breadcrumbs(vm.Path)
	vm.Analytics = handlersPkg.LoadAnalyticsFromEnv()
	vm.SEO.Canonical = absoluteURL(r)
	vm.SEO.Alternates = buildAlternates(r)
	vm.SEO.OG.URL = vm.SEO.Canonical

	brand := i18nOrDefault(lang, "brand.name", "Hanko Field")
	vm.SEO.Title = fmt.Sprintf("%s | %s", title, brand)
	vm.SEO.Description = subtitle
	vm.SEO.OG.SiteName = brand
	vm.SEO.OG.Type = "website"
	vm.SEO.OG.Title = vm.SEO.Title
	vm.SEO.OG.Description = vm.SEO.Description
	vm.SEO.Twitter.Card = "summary_large_image"

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	allGuides, err := cmsClient.ListGuides(ctx, cms.ListGuidesOptions{Lang: lang})
	if err != nil {
		log.Printf("guides: list: %v", err)
	}

	personaParam := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("persona")))
	topicParam := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("topic")))
	if topicParam == "" {
		topicParam = strings.ToLower(strings.TrimSpace(r.URL.Query().Get("category")))
	}
	searchParam := strings.TrimSpace(r.URL.Query().Get("q"))

	filtered := filterGuidesForView(allGuides, personaParam, topicParam, searchParam)

	cards := make([]GuideCard, 0, len(filtered))
	for _, g := range filtered {
		cards = append(cards, guideCardFromGuide(g, lang))
	}
	if len(cards) > 0 && vm.SEO.OG.Image == "" {
		vm.SEO.OG.Image = cards[0].HeroImageURL
		vm.SEO.Twitter.Image = vm.SEO.OG.Image
	}

	personaOptions := buildPersonaOptions(r, lang, allGuides, personaParam)
	topicOptions := buildTopicOptions(r, lang, allGuides, topicParam)
	activeFilters, clearAllURL := buildActiveGuideFilters(r, lang, personaParam, topicParam, searchParam)

	listVM := GuidesListViewModel{
		Items:          cards,
		PersonaOptions: personaOptions,
		TopicOptions:   topicOptions,
		ActivePersona:  personaParam,
		ActiveTopic:    topicParam,
		Search:         searchParam,
		Total:          len(filtered),
		ActiveFilters:  activeFilters,
		ClearAllURL:    clearAllURL,
		SubscribeCopy:  i18nOrDefault(lang, "guides.subscribe.copy", "Stay updated with new guides each month."),
		SubscribeCTA:   i18nOrDefault(lang, "guides.subscribe.cta", "Subscribe"),
		SubscribeURL:   "/newsletter",
		Empty:          len(filtered) == 0,
	}
	vm.Guides = listVM

	if len(cards) > 0 {
		vm.SEO.JSONLD = append(vm.SEO.JSONLD, seo.JSON(buildGuidesItemList(siteBaseURL(r), cards)))
	}

	renderPage(w, r, "guides", vm)
}

func GuideDetailHandler(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	slug := strings.TrimSpace(chi.URLParam(r, "slug"))
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	guide, err := cmsClient.GetGuide(ctx, slug, lang)
	if err != nil {
		if errors.Is(err, cms.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		log.Printf("guides: detail: %v", err)
		http.Error(w, "guides unavailable", http.StatusBadGateway)
		return
	}

	vm := handlersPkg.PageData{Title: guide.Title, Lang: lang}
	vm.Path = r.URL.Path
	vm.Nav = nav.Build(vm.Path)
	vm.Breadcrumbs = nav.Breadcrumbs(vm.Path)
	vm.Analytics = handlersPkg.LoadAnalyticsFromEnv()
	vm.SEO.Canonical = absoluteURL(r)
	vm.SEO.Alternates = buildAlternates(r)
	vm.SEO.OG.URL = vm.SEO.Canonical

	brand := i18nOrDefault(lang, "brand.name", "Hanko Field")
	title := guide.SEO.MetaTitle
	if title == "" {
		title = fmt.Sprintf("%s | %s", guide.Title, brand)
	}
	description := guide.SEO.MetaDescription
	if description == "" {
		if guide.Summary != "" {
			description = guide.Summary
		} else {
			description = i18nOrDefault(lang, "guides.detail.description_fallback", "In-depth guide from Hanko Field.")
		}
	}

	vm.SEO.Title = title
	vm.SEO.Description = description
	vm.SEO.OG.SiteName = brand
	vm.SEO.OG.Type = "article"
	vm.SEO.OG.Title = title
	vm.SEO.OG.Description = description
	vm.SEO.Twitter.Card = "summary_large_image"

	hero := guideHeroURL(guide.Title, guide.HeroImageURL)
	if guide.SEO.OGImage != "" {
		vm.SEO.OG.Image = guide.SEO.OGImage
	} else {
		vm.SEO.OG.Image = hero
	}
	vm.SEO.Twitter.Image = vm.SEO.OG.Image

	bodyHTML, toc := renderGuideBody(guide.Body)
	detailVM := GuideDetailViewModel{
		Title:         guide.Title,
		Summary:       guide.Summary,
		Body:          template.HTML(bodyHTML),
		HeroImageURL:  hero,
		CategoryLabel: guideCategoryLabel(lang, guide.Category),
		Tags:          append([]string(nil), guide.Tags...),
		TOC:           toc,
		ShareLinks:    buildGuideShareLinks(lang, vm.SEO.Canonical, guide.Title),
		Sources:       append([]string(nil), guide.Sources...),
		Author:        guide.Author,
	}
	if len(guide.Personas) > 0 {
		labels := make([]string, 0, len(guide.Personas))
		for _, p := range guide.Personas {
			labels = append(labels, guidePersonaLabel(lang, p))
		}
		detailVM.Personas = labels
	}
	if !guide.PublishAt.IsZero() {
		detailVM.PublishDate = format.FmtDate(guide.PublishAt, lang)
	}
	detailVM.PublishISO = isoOr(guide.PublishAt, guide.UpdatedAt)
	detailVM.ReadTime = guideReadingTimeLabel(lang, guide.ReadingTimeMinutes)
	detailVM.DownloadURL = firstPDF(detailVM.Sources)

	relatedPool, err := cmsClient.ListGuides(ctx, cms.ListGuidesOptions{Lang: lang})
	if err != nil {
		log.Printf("guides: related list: %v", err)
	}
	if len(relatedPool) > 0 {
		detailVM.Related = buildRelatedGuides(guide, relatedPool, lang)
	}

	vm.Guide = detailVM

	articleSchema := seo.Article(guide.Title, vm.SEO.Canonical, vm.SEO.OG.Image, guide.Author.Name, detailVM.PublishISO)
	vm.SEO.JSONLD = append(vm.SEO.JSONLD, seo.JSON(articleSchema))
	breadcrumbs := []seo.BreadcrumbItem{
		{Name: i18nOrDefault(lang, "nav.home", "Home"), Item: siteBaseURL(r)},
		{Name: i18nOrDefault(lang, "guides.title", "Guides"), Item: siteBaseURL(r) + "/guides"},
		{Name: guide.Title, Item: vm.SEO.Canonical},
	}
	vm.SEO.JSONLD = append(vm.SEO.JSONLD, seo.JSON(seo.BreadcrumbList(breadcrumbs)))

	renderPage(w, r, "guide_detail", vm)
}

func filterGuidesForView(all []cms.Guide, persona, category, search string) []cms.Guide {
	persona = strings.ToLower(strings.TrimSpace(persona))
	category = strings.ToLower(strings.TrimSpace(category))
	search = strings.ToLower(strings.TrimSpace(search))
	if persona == "" && category == "" && search == "" {
		out := make([]cms.Guide, len(all))
		copy(out, all)
		return out
	}
	out := make([]cms.Guide, 0, len(all))
	for _, g := range all {
		if category != "" && strings.ToLower(g.Category) != category {
			continue
		}
		if persona != "" && !containsFold(g.Personas, persona) {
			continue
		}
		if search != "" {
			hay := strings.ToLower(g.Title + " " + g.Summary + " " + strings.Join(g.Tags, " "))
			if !strings.Contains(hay, search) {
				continue
			}
		}
		out = append(out, g)
	}
	return out
}

func guideCardFromGuide(g cms.Guide, lang string) GuideCard {
	card := GuideCard{
		Slug:          g.Slug,
		Title:         g.Title,
		Summary:       g.Summary,
		Href:          "/guides/" + g.Slug,
		CategoryLabel: guideCategoryLabel(lang, g.Category),
		HeroImageURL:  guideHeroURL(g.Title, g.HeroImageURL),
		ReadTime:      guideReadingTimeLabel(lang, g.ReadingTimeMinutes),
		Tags:          append([]string(nil), g.Tags...),
	}
	if !g.PublishAt.IsZero() {
		card.PublishDate = format.FmtDate(g.PublishAt, lang)
		card.PublishISO = g.PublishAt.Format(time.RFC3339)
	} else if !g.UpdatedAt.IsZero() {
		card.PublishISO = g.UpdatedAt.Format(time.RFC3339)
	}
	return card
}

func guideHeroURL(title, existing string) string {
	if strings.TrimSpace(existing) != "" {
		return existing
	}
	text := strings.TrimSpace(title)
	if text == "" {
		text = "Guide"
	}
	return "https://placehold.co/960x600?text=" + url.QueryEscape(text)
}

func guideCategoryLabel(lang, category string) string {
	key := strings.ToLower(strings.TrimSpace(category))
	if key == "" {
		key = "other"
	}
	return i18nOrDefault(lang, "guides.category."+key, guideFallbackLabel(key))
}

func guidePersonaLabel(lang, persona string) string {
	key := strings.ToLower(strings.TrimSpace(persona))
	if key == "" {
		return i18nOrDefault(lang, "guides.persona.maker", "Maker")
	}
	return i18nOrDefault(lang, "guides.persona."+key, guideFallbackLabel(key))
}

func guideFallbackLabel(value string) string {
	value = strings.ReplaceAll(value, "-", " ")
	value = strings.ReplaceAll(value, "_", " ")
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Fields(value)
	for i, part := range parts {
		parts[i] = titleCaseASCII(part)
	}
	return strings.Join(parts, " ")
}

func guideReadingTimeLabel(lang string, minutes int) string {
	if minutes <= 0 {
		return ""
	}
	pattern := i18nOrDefault(lang, "guides.reading_time", "%d min read")
	return fmt.Sprintf(pattern, minutes)
}

func buildPersonaOptions(r *http.Request, lang string, guides []cms.Guide, current string) []GuideFilterOption {
	counts := map[string]int{}
	for _, g := range guides {
		for _, p := range g.Personas {
			key := strings.ToLower(strings.TrimSpace(p))
			if key != "" {
				counts[key]++
			}
		}
	}
	options := make([]GuideFilterOption, 0, len(counts)+1)
	labelAll := i18nOrDefault(lang, "guides.filters.persona_all", i18nOrDefault(lang, "filter.all", "All"))
	options = append(options, GuideFilterOption{
		Value:    "",
		Label:    labelAll,
		Href:     buildGuideFilterURL(r, lang, map[string]string{"persona": ""}, []string{"page"}),
		Selected: current == "",
		Count:    len(guides),
	})
	seen := map[string]bool{}
	for _, key := range guidePersonaOrder {
		if count := counts[key]; count > 0 {
			options = append(options, GuideFilterOption{
				Value:    key,
				Label:    guidePersonaLabel(lang, key),
				Href:     buildGuideFilterURL(r, lang, map[string]string{"persona": key}, []string{"page"}),
				Selected: current == key,
				Count:    count,
			})
			seen[key] = true
		}
	}
	extra := make([]string, 0)
	for key := range counts {
		if !seen[key] {
			extra = append(extra, key)
		}
	}
	sort.Strings(extra)
	for _, key := range extra {
		options = append(options, GuideFilterOption{
			Value:    key,
			Label:    guidePersonaLabel(lang, key),
			Href:     buildGuideFilterURL(r, lang, map[string]string{"persona": key}, []string{"page"}),
			Selected: current == key,
			Count:    counts[key],
		})
	}
	return options
}

func buildTopicOptions(r *http.Request, lang string, guides []cms.Guide, current string) []GuideFilterOption {
	counts := map[string]int{}
	for _, g := range guides {
		key := strings.ToLower(strings.TrimSpace(g.Category))
		if key != "" {
			counts[key]++
		}
	}
	options := make([]GuideFilterOption, 0, len(counts)+1)
	labelAll := i18nOrDefault(lang, "guides.filters.topic_all", i18nOrDefault(lang, "filter.all", "All"))
	options = append(options, GuideFilterOption{
		Value:    "",
		Label:    labelAll,
		Href:     buildGuideFilterURL(r, lang, map[string]string{"topic": "", "category": ""}, []string{"page"}),
		Selected: current == "",
		Count:    len(guides),
	})
	seen := map[string]bool{}
	for _, key := range guideCategoryOrder {
		if count := counts[key]; count > 0 {
			options = append(options, GuideFilterOption{
				Value:    key,
				Label:    guideCategoryLabel(lang, key),
				Href:     buildGuideFilterURL(r, lang, map[string]string{"topic": key, "category": key}, []string{"page"}),
				Selected: current == key,
				Count:    count,
			})
			seen[key] = true
		}
	}
	extra := make([]string, 0)
	for key := range counts {
		if !seen[key] {
			extra = append(extra, key)
		}
	}
	sort.Strings(extra)
	for _, key := range extra {
		options = append(options, GuideFilterOption{
			Value:    key,
			Label:    guideCategoryLabel(lang, key),
			Href:     buildGuideFilterURL(r, lang, map[string]string{"topic": key, "category": key}, []string{"page"}),
			Selected: current == key,
			Count:    counts[key],
		})
	}
	return options
}

func buildActiveGuideFilters(r *http.Request, lang, persona, topic, search string) ([]GuideActiveFilter, string) {
	filters := make([]GuideActiveFilter, 0, 3)
	if persona != "" {
		label := fmt.Sprintf("%s: %s", i18nOrDefault(lang, "guides.filters.persona_label", "Persona"), guidePersonaLabel(lang, persona))
		href := buildGuideFilterURL(r, lang, map[string]string{"persona": ""}, []string{"page"})
		filters = append(filters, GuideActiveFilter{Label: label, Href: href})
	}
	if topic != "" {
		label := fmt.Sprintf("%s: %s", i18nOrDefault(lang, "guides.filters.topic_label", "Topic"), guideCategoryLabel(lang, topic))
		href := buildGuideFilterURL(r, lang, map[string]string{"topic": "", "category": ""}, []string{"page"})
		filters = append(filters, GuideActiveFilter{Label: label, Href: href})
	}
	if strings.TrimSpace(search) != "" {
		label := fmt.Sprintf("%s: %s", i18nOrDefault(lang, "guides.filters.search_label", "Search"), search)
		href := buildGuideFilterURL(r, lang, map[string]string{"q": ""}, []string{"page"})
		filters = append(filters, GuideActiveFilter{Label: label, Href: href})
	}
	clear := buildGuideFilterURL(r, lang, map[string]string{
		"persona":  "",
		"topic":    "",
		"category": "",
		"q":        "",
	}, []string{"page"})
	return filters, clear
}

func buildGuideFilterURL(r *http.Request, lang string, updates map[string]string, clears []string) string {
	q := url.Values{}
	for key, vals := range r.URL.Query() {
		for _, v := range vals {
			q.Add(key, v)
		}
	}
	for _, key := range clears {
		q.Del(key)
	}
	for key, value := range updates {
		if strings.TrimSpace(value) == "" {
			q.Del(key)
		} else {
			q.Set(key, value)
		}
	}
	if lang != "" {
		q.Set("hl", lang)
	}
	encoded := q.Encode()
	if encoded == "" {
		return r.URL.Path
	}
	return r.URL.Path + "?" + encoded
}

func buildGuidesItemList(base string, cards []GuideCard) map[string]any {
	items := make([]map[string]any, 0, len(cards))
	for i, card := range cards {
		items = append(items, map[string]any{
			"@type":    "ListItem",
			"position": i + 1,
			"name":     card.Title,
			"url":      base + card.Href,
		})
	}
	return map[string]any{
		"@context":        "https://schema.org",
		"@type":           "ItemList",
		"itemListElement": items,
	}
}

func buildGuideShareLinks(lang, canonical, title string) []GuideShareLink {
	canonical = strings.TrimSpace(canonical)
	if canonical == "" {
		return nil
	}
	encodedURL := url.QueryEscape(canonical)
	encodedTitle := url.QueryEscape(title)
	return []GuideShareLink{
		{
			Icon:  "x",
			Label: i18nOrDefault(lang, "guides.share.x", "Share on X"),
			Href:  fmt.Sprintf("https://twitter.com/share?url=%s&text=%s", encodedURL, encodedTitle),
		},
		{
			Icon:  "facebook",
			Label: i18nOrDefault(lang, "guides.share.facebook", "Share on Facebook"),
			Href:  fmt.Sprintf("https://www.facebook.com/sharer/sharer.php?u=%s", encodedURL),
		},
		{
			Icon:  "line",
			Label: i18nOrDefault(lang, "guides.share.line", "Share on LINE"),
			Href:  fmt.Sprintf("https://social-plugins.line.me/lineit/share?url=%s", encodedURL),
		},
	}
}

func buildRelatedGuides(current cms.Guide, pool []cms.Guide, lang string) []GuideCard {
	if len(pool) == 0 {
		return nil
	}
	related := make([]cms.Guide, 0, 3)
	for _, g := range pool {
		if g.Slug == current.Slug {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(g.Category), strings.TrimSpace(current.Category)) {
			related = append(related, g)
		}
		if len(related) == 3 {
			break
		}
	}
	if len(related) < 3 {
		for _, g := range pool {
			if g.Slug == current.Slug || containsGuideSlug(related, g.Slug) {
				continue
			}
			related = append(related, g)
			if len(related) == 3 {
				break
			}
		}
	}
	cards := make([]GuideCard, 0, len(related))
	for _, g := range related {
		cards = append(cards, guideCardFromGuide(g, lang))
	}
	return cards
}

func containsGuideSlug(list []cms.Guide, slug string) bool {
	for _, g := range list {
		if g.Slug == slug {
			return true
		}
	}
	return false
}

func renderGuideBody(body string) (string, []TOCEntry) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", nil
	}
	sanitized := guideHTMLPolicy.Sanitize(body)
	nodes, err := html.ParseFragment(strings.NewReader(sanitized), nil)
	if err != nil {
		return sanitized, nil
	}
	headings := make([]TOCEntry, 0, 8)
	seen := map[string]struct{}{}
	for _, node := range nodes {
		collectHeadings(node, &headings, seen)
	}
	var buf bytes.Buffer
	for _, node := range nodes {
		if err := html.Render(&buf, node); err != nil {
			continue
		}
	}
	return buf.String(), headings
}

func collectHeadings(n *html.Node, headings *[]TOCEntry, seen map[string]struct{}) {
	if n.Type == html.ElementNode && (n.Data == "h2" || n.Data == "h3" || n.Data == "h4") {
		text := strings.TrimSpace(nodeText(n))
		if text != "" {
			id := extractHeadingID(n, text, len(*headings), seen)
			level := headingLevel(n.Data)
			*headings = append(*headings, TOCEntry{ID: id, Text: text, Level: level})
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectHeadings(c, headings, seen)
	}
}

func extractHeadingID(n *html.Node, text string, index int, seen map[string]struct{}) string {
	id := ""
	for i := range n.Attr {
		if n.Attr[i].Key == "id" {
			id = strings.TrimSpace(n.Attr[i].Val)
			break
		}
	}
	if id == "" {
		id = headingIDFromText(text, index)
	}
	original := id
	if original == "" {
		original = fmt.Sprintf("section-%d", index+1)
	}
	counter := 1
	for {
		if id != "" {
			if _, exists := seen[id]; !exists {
				break
			}
		}
		counter++
		id = fmt.Sprintf("%s-%d", original, counter)
		if _, exists := seen[id]; !exists {
			break
		}
	}
	seen[id] = struct{}{}
	updated := false
	for i := range n.Attr {
		if n.Attr[i].Key == "id" {
			n.Attr[i].Val = id
			updated = true
			break
		}
	}
	if !updated {
		n.Attr = append(n.Attr, html.Attribute{Key: "id", Val: id})
	}
	return id
}

func headingIDFromText(text string, index int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Sprintf("section-%d", index+1)
	}
	var b strings.Builder
	var lastHyphen bool
	lower := strings.ToLower(text)
	for _, r := range lower {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastHyphen = false
		case r == ' ' || r == '-' || r == '_' || r == '\t' || r == '\n' || r == '\r':
			if !lastHyphen && b.Len() > 0 {
				b.WriteRune('-')
				lastHyphen = true
			}
		default:
			if unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana) {
				b.WriteRune(r)
				lastHyphen = false
			}
		}
	}
	id := strings.Trim(b.String(), "-")
	if id == "" {
		id = fmt.Sprintf("section-%d", index+1)
	}
	return id
}

func headingLevel(tag string) int {
	switch tag {
	case "h2":
		return 2
	case "h3":
		return 3
	case "h4":
		return 4
	default:
		return 0
	}
}

func nodeText(n *html.Node) string {
	if n == nil {
		return ""
	}
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(nodeText(c))
	}
	return sb.String()
}

func containsFold(list []string, needle string) bool {
	needle = strings.ToLower(strings.TrimSpace(needle))
	if needle == "" {
		return false
	}
	for _, v := range list {
		if strings.ToLower(strings.TrimSpace(v)) == needle {
			return true
		}
	}
	return false
}

func firstPDF(sources []string) string {
	for _, src := range sources {
		s := strings.TrimSpace(src)
		if strings.HasSuffix(strings.ToLower(s), ".pdf") {
			return s
		}
	}
	if len(sources) > 0 {
		return sources[0]
	}
	return ""
}

func isoOr(primary, secondary time.Time) string {
	if !primary.IsZero() {
		return primary.Format(time.RFC3339)
	}
	if !secondary.IsZero() {
		return secondary.Format(time.RFC3339)
	}
	return ""
}

func computeContentETag(page cms.ContentPage, slug string) string {
	updated := ""
	if !page.UpdatedAt.IsZero() {
		updated = page.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	effective := ""
	if !page.EffectiveDate.IsZero() {
		effective = page.EffectiveDate.UTC().Format(time.RFC3339)
	}
	return etagFor("content:", page.Kind, page.Lang, slug, page.Version, page.Format, updated, effective, page.Body)
}

func contentCacheKey(page cms.ContentPage, slug, etag string) string {
	return strings.Join([]string{page.Kind, page.Lang, slug, etag}, "|")
}

func cachedRenderedContent(key string) (template.HTML, []TOCEntry, bool) {
	if key == "" {
		return "", nil, false
	}
	contentRenderCache.mu.RLock()
	entry, ok := contentRenderCache.items[key]
	contentRenderCache.mu.RUnlock()
	if !ok || time.Now().After(entry.expires) {
		return "", nil, false
	}
	tocCopy := make([]TOCEntry, len(entry.toc))
	copy(tocCopy, entry.toc)
	return entry.body, tocCopy, true
}

func storeRenderedContent(key string, body template.HTML, toc []TOCEntry) {
	if key == "" {
		return
	}
	var tocCopy []TOCEntry
	if len(toc) > 0 {
		tocCopy = make([]TOCEntry, len(toc))
		copy(tocCopy, toc)
	}
	contentRenderCache.mu.Lock()
	contentRenderCache.items[key] = renderedContentEntry{
		body:    body,
		toc:     tocCopy,
		expires: time.Now().Add(contentRenderTTL),
	}
	contentRenderCache.mu.Unlock()
}

func renderContentBody(page cms.ContentPage) (template.HTML, []TOCEntry, error) {
	format := strings.ToLower(strings.TrimSpace(page.Format))
	switch format {
	case "", "markdown", "md":
		htmlBody, toc, err := renderMarkdownContent(page.Body)
		return template.HTML(htmlBody), toc, err
	default:
		htmlBody, toc := renderGuideBody(page.Body)
		return template.HTML(htmlBody), toc, nil
	}
}

func renderMarkdownContent(markdown string) (string, []TOCEntry, error) {
	markdown = strings.TrimSpace(markdown)
	if markdown == "" {
		return "", nil, nil
	}
	var buf bytes.Buffer
	if err := markdownRenderer.Convert([]byte(markdown), &buf); err != nil {
		return "", nil, err
	}
	htmlStr := guideHTMLPolicy.Sanitize(buf.String())
	nodes, err := html.ParseFragment(strings.NewReader(htmlStr), nil)
	if err != nil {
		return htmlStr, nil, err
	}
	headings := make([]TOCEntry, 0, 8)
	seen := map[string]struct{}{}
	for _, node := range nodes {
		collectHeadings(node, &headings, seen)
	}
	var out bytes.Buffer
	for _, node := range nodes {
		if err := html.Render(&out, node); err != nil {
			continue
		}
	}
	return out.String(), headings, nil
}

func buildContentPageView(lang string, page cms.ContentPage, body template.HTML, toc []TOCEntry, fallbackSummary, defaultIcon string) ContentPageViewModel {
	header := ContentHeaderView{
		Icon:    valueOr(page.Icon, defaultIcon),
		Title:   page.Title,
		Summary: valueOr(page.Summary, fallbackSummary),
	}
	if eff, effISO := displayDate(page.EffectiveDate, lang); eff != "" {
		header.Effective = eff
		header.EffectiveISO = effISO
	}
	if upd, updISO := displayDate(page.UpdatedAt, lang); upd != "" {
		header.Updated = upd
		header.UpdatedISO = updISO
	}

	banner := buildContentBanner(lang, page.Banner)

	var version *ContentVersionView
	if page.Version != "" || header.Updated != "" || page.DownloadURL != "" {
		version = &ContentVersionView{
			Version:       page.Version,
			Updated:       header.Updated,
			UpdatedISO:    header.UpdatedISO,
			DownloadLabel: page.DownloadLabel,
			DownloadURL:   page.DownloadURL,
		}
		if version.DownloadURL != "" && version.DownloadLabel == "" {
			version.DownloadLabel = i18nOrDefault(lang, "content.download", "Download PDF")
		}
	}

	return ContentPageViewModel{
		Header:  header,
		Banner:  banner,
		Body:    body,
		TOC:     toc,
		Version: version,
	}
}

func buildContentBanner(lang string, banner *cms.ContentBanner) *AlertBannerView {
	if banner == nil {
		return nil
	}
	variant := strings.ToLower(strings.TrimSpace(banner.Variant))
	bg, border, text, icon := alertPalette(variant)
	return &AlertBannerView{
		Variant:    variant,
		Title:      banner.Title,
		Message:    banner.Message,
		LinkText:   banner.LinkText,
		LinkURL:    banner.LinkURL,
		Icon:       icon,
		Background: bg,
		Border:     border,
		Text:       text,
	}
}

func alertPalette(variant string) (bg, border, text, icon string) {
	switch variant {
	case "success", "positive", "ok":
		return "bg-emerald-50", "border-emerald-200", "text-emerald-900", "check-circle"
	case "warning", "caution":
		return "bg-amber-50", "border-amber-200", "text-amber-900", "exclamation-triangle"
	case "danger", "error", "critical":
		return "bg-rose-50", "border-rose-200", "text-rose-900", "exclamation-circle"
	default:
		return "bg-sky-50", "border-sky-200", "text-sky-900", "information-circle"
	}
}

func displayDate(t time.Time, lang string) (string, string) {
	if t.IsZero() {
		return "", ""
	}
	iso := t.UTC().Format(time.RFC3339)
	switch strings.ToLower(lang) {
	case "ja":
		return t.In(jstLocation).Format("2006-01-02"), iso
	default:
		return t.In(time.UTC).Format("Jan 2, 2006"), iso
	}
}

func displayDateTime(t time.Time, lang string) (string, string) {
	if t.IsZero() {
		return "", ""
	}
	iso := t.UTC().Format(time.RFC3339)
	switch strings.ToLower(lang) {
	case "ja":
		return t.In(jstLocation).Format("2006-01-02 15:04 MST"), iso
	default:
		return t.In(time.UTC).Format("Jan 2, 2006 15:04 MST"), iso
	}
}

func laterTime(a, b time.Time) time.Time {
	if a.IsZero() {
		return b
	}
	if b.IsZero() {
		return a
	}
	if a.After(b) {
		return a
	}
	return b
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func serveStaticPage(w http.ResponseWriter, r *http.Request, kind, templateName, defaultIcon, summaryKey, summaryFallback string) {
	if cmsClient == nil {
		http.Error(w, "content unavailable", http.StatusServiceUnavailable)
		return
	}
	lang := mw.Lang(r)
	slug := strings.TrimSpace(chi.URLParam(r, "slug"))
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	page, err := cmsClient.GetContentPage(ctx, kind, slug, lang)
	if err != nil {
		if errors.Is(err, cms.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		log.Printf("%s: fetch %s: %v", kind, slug, err)
		http.Error(w, "content unavailable", http.StatusBadGateway)
		return
	}

	etag := computeContentETag(page, slug)
	if etag != "" {
		w.Header().Set("ETag", etag)
		if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	w.Header().Set("Cache-Control", "public, max-age=600")
	if lm := laterTime(page.UpdatedAt, page.EffectiveDate); !lm.IsZero() {
		w.Header().Set("Last-Modified", lm.UTC().Format(http.TimeFormat))
	}

	cacheKey := contentCacheKey(page, slug, etag)
	body, toc, cached := cachedRenderedContent(cacheKey)
	if !cached {
		rendered, tocEntries, renderErr := renderContentBody(page)
		if renderErr != nil {
			log.Printf("%s: render %s: %v", kind, slug, renderErr)
		}
		body = rendered
		toc = tocEntries
		storeRenderedContent(cacheKey, body, toc)
	}

	vm := handlersPkg.PageData{Title: page.Title, Lang: lang}
	vm.Path = r.URL.Path
	vm.Nav = nav.Build(vm.Path)
	vm.Breadcrumbs = nav.Breadcrumbs(vm.Path)
	vm.Analytics = handlersPkg.LoadAnalyticsFromEnv()
	vm.SEO.Canonical = absoluteURL(r)
	vm.SEO.OG.URL = vm.SEO.Canonical
	vm.SEO.Alternates = buildAlternates(r)

	brand := i18nOrDefault(lang, "brand.name", "Hanko Field")
	if page.SEO.Title != "" {
		vm.SEO.Title = page.SEO.Title
	} else {
		vm.SEO.Title = fmt.Sprintf("%s | %s", page.Title, brand)
	}
	if page.SEO.Description != "" {
		vm.SEO.Description = page.SEO.Description
	} else if page.Summary != "" {
		vm.SEO.Description = page.Summary
	} else {
		vm.SEO.Description = i18nOrDefault(lang, "content.seo.description", "Static resources from Hanko Field.")
	}
	vm.SEO.OG.SiteName = brand
	vm.SEO.OG.Type = "article"
	vm.SEO.OG.Title = vm.SEO.Title
	vm.SEO.OG.Description = vm.SEO.Description
	if page.SEO.OGImage != "" {
		vm.SEO.OG.Image = page.SEO.OGImage
		vm.SEO.Twitter.Image = page.SEO.OGImage
	}
	vm.SEO.Twitter.Card = "summary_large_image"

	defaultSummary := i18nOrDefault(lang, summaryKey, summaryFallback)
	view := buildContentPageView(lang, page, body, toc, defaultSummary, defaultIcon)
	vm.Content = view
	renderPage(w, r, templateName, vm)
}

func ContentPageHandler(w http.ResponseWriter, r *http.Request) {
	serveStaticPage(w, r, "content", "content", "document-text", "content.default_summary", "Company announcements and resources.")
}

func LegalPageHandler(w http.ResponseWriter, r *http.Request) {
	serveStaticPage(w, r, "legal", "legal", "scale", "legal.default_summary", "Policies, terms of service, and compliance information.")
}

func StatusHandler(w http.ResponseWriter, r *http.Request) {
	if statusClient == nil {
		statusClient = status.NewClient("")
	}
	lang := mw.Lang(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	summary, err := statusClient.FetchSummary(ctx, lang)
	if err != nil {
		log.Printf("status: fetch: %v", err)
	}

	etag := computeStatusETag(summary, lang)
	if etag != "" {
		w.Header().Set("ETag", etag)
		if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	w.Header().Set("Cache-Control", "public, max-age=60")
	if !summary.UpdatedAt.IsZero() {
		w.Header().Set("Last-Modified", summary.UpdatedAt.UTC().Format(http.TimeFormat))
	}

	stateLabel := statusStateLabel(lang, summary.State)
	pageTitle := i18nOrDefault(lang, "status.title", "System Status")
	if stateLabel != "" {
		pageTitle = fmt.Sprintf("%s - %s", pageTitle, stateLabel)
	}

	vm := handlersPkg.PageData{
		Title: pageTitle,
		Lang:  lang,
	}
	vm.Path = r.URL.Path
	vm.Nav = nav.Build(vm.Path)
	vm.Breadcrumbs = nav.Breadcrumbs(vm.Path)
	vm.Analytics = handlersPkg.LoadAnalyticsFromEnv()
	vm.SEO.Canonical = absoluteURL(r)
	vm.SEO.OG.URL = vm.SEO.Canonical
	vm.SEO.Alternates = buildAlternates(r)

	brand := i18nOrDefault(lang, "brand.name", "Hanko Field")
	vm.SEO.Title = fmt.Sprintf("%s | %s", i18nOrDefault(lang, "status.title", "System Status"), brand)
	vm.SEO.Description = i18nOrDefault(lang, "status.summary", "Real-time uptime and incident history for Hanko Field services.")
	vm.SEO.OG.SiteName = brand
	vm.SEO.OG.Type = "website"
	vm.SEO.OG.Title = vm.SEO.Title
	vm.SEO.OG.Description = vm.SEO.Description
	vm.SEO.Twitter.Card = "summary_large_image"

	view := buildStatusPageView(lang, summary)
	vm.Status = view
	renderPage(w, r, "status", vm)
}

func computeStatusETag(summary status.Summary, lang string) string {
	var parts []string
	parts = append(parts, lang, summary.State, summary.StateLabel, summary.UpdatedAt.UTC().Format(time.RFC3339Nano))
	for _, comp := range summary.Components {
		parts = append(parts, comp.Name, comp.Status)
	}
	for _, incident := range summary.Incidents {
		parts = append(parts, incident.ID, incident.Status, incident.Impact, incident.StartedAt.UTC().Format(time.RFC3339Nano), incident.ResolvedAt.UTC().Format(time.RFC3339Nano))
		for _, upd := range incident.Updates {
			parts = append(parts, upd.Status, upd.Timestamp.UTC().Format(time.RFC3339Nano), upd.Body)
		}
	}
	return etagFor("status:", parts...)
}

func buildStatusPageView(lang string, summary status.Summary) StatusPageViewModel {
	stateLabel := summary.StateLabel
	if stateLabel == "" {
		stateLabel = statusStateLabel(lang, summary.State)
	}
	badge, badgeText := statusStateVariant(summary.State)
	updated, updatedISO := displayDateTime(summary.UpdatedAt, lang)
	overview := StatusOverviewView{
		State:      summary.State,
		StateLabel: stateLabel,
		Badge:      badge,
		BadgeText:  badgeText,
		Updated:    updated,
		UpdatedISO: updatedISO,
	}

	components := make([]StatusComponentView, 0, len(summary.Components))
	for _, comp := range summary.Components {
		label := statusStateLabel(lang, comp.Status)
		cBadge, cText := statusComponentVariant(comp.Status)
		components = append(components, StatusComponentView{
			Name:        comp.Name,
			Status:      comp.Status,
			StatusLabel: label,
			Badge:       cBadge,
			BadgeText:   cText,
		})
	}

	incidents := make([]StatusIncidentView, 0, len(summary.Incidents))
	for _, inc := range summary.Incidents {
		statusLabel := statusTimelineLabel(lang, inc.Status)
		statusBadge, statusText := statusIncidentVariant(inc.Status)
		impactLabel := statusImpactLabel(lang, inc.Impact)
		impactBadge, impactText := statusImpactVariant(inc.Impact)
		started, startedISO := displayDateTime(inc.StartedAt, lang)
		resolved, resolvedISO := displayDateTime(inc.ResolvedAt, lang)
		updates := make([]StatusIncidentUpdateView, 0, len(inc.Updates))
		for _, upd := range inc.Updates {
			ts, tsISO := displayDateTime(upd.Timestamp, lang)
			updates = append(updates, StatusIncidentUpdateView{
				Timestamp:    ts,
				TimestampISO: tsISO,
				Status:       upd.Status,
				StatusLabel:  statusTimelineLabel(lang, upd.Status),
				Body:         upd.Body,
			})
		}
		incidents = append(incidents, StatusIncidentView{
			Title:       inc.Title,
			Status:      inc.Status,
			StatusLabel: statusLabel,
			StatusBadge: statusBadge,
			StatusText:  statusText,
			Impact:      inc.Impact,
			ImpactLabel: impactLabel,
			ImpactBadge: impactBadge,
			ImpactText:  impactText,
			Started:     started,
			StartedISO:  startedISO,
			Resolved:    resolved,
			ResolvedISO: resolvedISO,
			Updates:     updates,
		})
	}

	header := ContentHeaderView{
		Icon:    "signal",
		Title:   i18nOrDefault(lang, "status.title", "System Status"),
		Summary: i18nOrDefault(lang, "status.summary", "Real-time uptime and incident history for Hanko Field services."),
	}
	if overview.Updated != "" {
		header.Updated = overview.Updated
		header.UpdatedISO = overview.UpdatedISO
	}

	var banner *AlertBannerView
	if overview.State != "" {
		bg, border, text, icon := statusStatePalette(overview.State)
		banner = &AlertBannerView{
			Variant:    overview.State,
			Title:      overview.StateLabel,
			Message:    i18nOrDefault(lang, "status.banner.message", "See component details and historical incidents below."),
			Background: bg,
			Border:     border,
			Text:       text,
			Icon:       icon,
		}
	}

	return StatusPageViewModel{
		Header:     header,
		Banner:     banner,
		Overview:   overview,
		Components: components,
		Incidents:  incidents,
	}
}

func statusStateLabel(lang, state string) string {
	state = strings.ToLower(strings.TrimSpace(state))
	switch state {
	case "operational":
		return i18nOrDefault(lang, "status.state.operational", "Operational")
	case "degraded":
		return i18nOrDefault(lang, "status.state.degraded", "Degraded performance")
	case "partial_outage", "partial-outage":
		return i18nOrDefault(lang, "status.state.partial_outage", "Partial outage")
	case "outage", "major_outage":
		return i18nOrDefault(lang, "status.state.outage", "Major outage")
	case "maintenance", "scheduled", "maintenance_mode":
		return i18nOrDefault(lang, "status.state.maintenance", "Maintenance")
	default:
		return titleCase(state)
	}
}

func statusTimelineLabel(lang, state string) string {
	state = strings.ToLower(strings.TrimSpace(state))
	switch state {
	case "investigating":
		return i18nOrDefault(lang, "status.timeline.investigating", "Investigating")
	case "identified", "mitigating":
		return i18nOrDefault(lang, "status.timeline.mitigating", "Mitigating")
	case "in_progress", "in-progress":
		return i18nOrDefault(lang, "status.timeline.in_progress", "In progress")
	case "monitoring":
		return i18nOrDefault(lang, "status.timeline.monitoring", "Monitoring")
	case "resolved", "completed":
		return i18nOrDefault(lang, "status.timeline.resolved", "Resolved")
	case "scheduled":
		return i18nOrDefault(lang, "status.timeline.scheduled", "Scheduled")
	default:
		return titleCase(state)
	}
}

func statusImpactLabel(lang, impact string) string {
	impact = strings.ToLower(strings.TrimSpace(impact))
	switch impact {
	case "minor":
		return i18nOrDefault(lang, "status.impact.minor", "Minor impact")
	case "major":
		return i18nOrDefault(lang, "status.impact.major", "Major impact")
	case "critical":
		return i18nOrDefault(lang, "status.impact.critical", "Critical impact")
	case "maintenance":
		return i18nOrDefault(lang, "status.impact.maintenance", "Scheduled maintenance")
	default:
		return titleCase(impact)
	}
}

func statusStatePalette(state string) (bg, border, text, icon string) {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "operational":
		return "bg-emerald-50", "border-emerald-200", "text-emerald-900", "check-circle"
	case "degraded":
		return "bg-amber-50", "border-amber-200", "text-amber-900", "exclamation-triangle"
	case "partial_outage", "partial-outage":
		return "bg-orange-50", "border-orange-200", "text-orange-900", "exclamation-circle"
	case "outage", "major_outage":
		return "bg-rose-50", "border-rose-200", "text-rose-900", "x-circle"
	case "maintenance":
		return "bg-sky-50", "border-sky-200", "text-sky-900", "wrench-screwdriver"
	default:
		return "bg-slate-50", "border-slate-200", "text-slate-900", "information-circle"
	}
}

func statusStateVariant(state string) (badge, text string) {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "operational":
		return "bg-emerald-100 text-emerald-800", "text-emerald-900"
	case "degraded":
		return "bg-amber-100 text-amber-800", "text-amber-900"
	case "partial_outage", "partial-outage":
		return "bg-orange-100 text-orange-800", "text-orange-900"
	case "outage", "major_outage":
		return "bg-rose-100 text-rose-800", "text-rose-900"
	case "maintenance":
		return "bg-sky-100 text-sky-800", "text-sky-900"
	default:
		return "bg-slate-100 text-slate-800", "text-slate-900"
	}
}

func statusComponentVariant(state string) (badge, text string) {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "operational":
		return "bg-emerald-100 text-emerald-800", "text-emerald-900"
	case "degraded":
		return "bg-amber-100 text-amber-800", "text-amber-900"
	case "partial_outage", "partial-outage":
		return "bg-orange-100 text-orange-800", "text-orange-900"
	case "outage", "major_outage":
		return "bg-rose-100 text-rose-800", "text-rose-900"
	case "maintenance":
		return "bg-sky-100 text-sky-800", "text-sky-900"
	default:
		return "bg-slate-100 text-slate-800", "text-slate-900"
	}
}

func statusIncidentVariant(state string) (badge, text string) {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "resolved":
		return "bg-emerald-100 text-emerald-800", "text-emerald-900"
	case "monitoring":
		return "bg-sky-100 text-sky-800", "text-sky-900"
	case "investigating", "mitigating", "identified":
		return "bg-amber-100 text-amber-800", "text-amber-900"
	case "completed":
		return "bg-emerald-100 text-emerald-800", "text-emerald-900"
	case "in_progress", "in-progress":
		return "bg-indigo-100 text-indigo-800", "text-indigo-900"
	case "scheduled":
		return "bg-slate-100 text-slate-800", "text-slate-900"
	default:
		return "bg-slate-100 text-slate-800", "text-slate-900"
	}
}

func statusImpactVariant(impact string) (badge, text string) {
	switch strings.ToLower(strings.TrimSpace(impact)) {
	case "minor":
		return "bg-amber-100 text-amber-800", "text-amber-900"
	case "major":
		return "bg-orange-100 text-orange-800", "text-orange-900"
	case "critical":
		return "bg-rose-100 text-rose-800", "text-rose-900"
	case "maintenance":
		return "bg-sky-100 text-sky-800", "text-sky-900"
	default:
		return "bg-slate-100 text-slate-800", "text-slate-900"
	}
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

// --- Template catalog data & helpers ---

type TemplateMeta struct {
	ID                  string
	Name                string
	Summary             string
	Script              string
	ScriptLabel         string
	Shape               string
	ShapeLabel          string
	Registrability      string
	RegistrabilityLabel string
	Category            string
	CategoryLabel       string
	Style               string
	StyleLabel          string
	Locale              string
	LocaleLabel         string
	Badge               string
	PrimarySize         string
	Preview             TemplatePreview
	Usage               int
	Favorites           int
	Updated             time.Time
	Tags                []string
}

type TemplatePreview struct {
	Background string
	Glyph      string
	GlyphClass string
	RingClass  string
}

type TemplateDetail struct {
	Template         TemplateMeta
	Lead             string
	Summary          []string
	RecommendedSizes []TemplateSize
	Constraints      []TemplateConstraint
	PreviewShots     []TemplatePreviewShot
	Stats            TemplateStats
	Metadata         []TemplateMetadataEntry
	UseCases         []string
	RecommendedInks  []string
	Actions          []TemplateAction
	Related          []TemplateMeta
}

type TemplateSize struct {
	Label      string
	Dimensions string
	Usage      string
}

type TemplateConstraint struct {
	Label       string
	Description string
}

type TemplatePreviewShot struct {
	Label      string
	Background string
	Glyph      string
	GlyphClass string
	RingClass  string
	Caption    string
}

type TemplateStats struct {
	Usage         int
	Favorites     int
	Satisfaction  int
	AvgTurnaround string
	LastUpdated   time.Time
}

type TemplateMetadataEntry struct {
	Label string
	Value string
	Hint  string
}

type TemplateAction struct {
	Label   string
	Href    string
	Variant string
}

type FilterOption struct {
	Value    string
	Label    string
	Count    int
	Selected bool
}

type GuideCard struct {
	Slug          string
	Title         string
	Summary       string
	Href          string
	CategoryLabel string
	HeroImageURL  string
	ReadTime      string
	PublishDate   string
	PublishISO    string
	Tags          []string
}

type GuideFilterOption struct {
	Value    string
	Label    string
	Href     string
	Selected bool
	Count    int
}

type GuideActiveFilter struct {
	Label string
	Href  string
}

type GuidesListViewModel struct {
	Items          []GuideCard
	PersonaOptions []GuideFilterOption
	TopicOptions   []GuideFilterOption
	ActivePersona  string
	ActiveTopic    string
	Search         string
	Total          int
	ActiveFilters  []GuideActiveFilter
	ClearAllURL    string
	SubscribeCopy  string
	SubscribeCTA   string
	SubscribeURL   string
	Empty          bool
}

type GuideShareLink struct {
	Icon  string
	Label string
	Href  string
}

type TOCEntry struct {
	ID    string
	Text  string
	Level int
}

type GuideDetailViewModel struct {
	Title         string
	Summary       string
	Body          template.HTML
	HeroImageURL  string
	PublishDate   string
	PublishISO    string
	ReadTime      string
	CategoryLabel string
	Tags          []string
	Personas      []string
	TOC           []TOCEntry
	ShareLinks    []GuideShareLink
	Related       []GuideCard
	Sources       []string
	DownloadURL   string
	Author        cms.Author
}

type renderedContentEntry struct {
	body    template.HTML
	toc     []TOCEntry
	expires time.Time
}

type ContentHeaderView struct {
	Icon         string
	Title        string
	Summary      string
	Effective    string
	EffectiveISO string
	Updated      string
	UpdatedISO   string
}

type ContentVersionView struct {
	Version       string
	Updated       string
	UpdatedISO    string
	DownloadLabel string
	DownloadURL   string
}

type AlertBannerView struct {
	Variant    string
	Title      string
	Message    string
	LinkText   string
	LinkURL    string
	Icon       string
	Background string
	Border     string
	Text       string
}

type ContentPageViewModel struct {
	Header  ContentHeaderView
	Banner  *AlertBannerView
	Body    template.HTML
	TOC     []TOCEntry
	Version *ContentVersionView
}

type StatusOverviewView struct {
	State      string
	StateLabel string
	Badge      string
	BadgeText  string
	Updated    string
	UpdatedISO string
}

type StatusComponentView struct {
	Name        string
	Status      string
	StatusLabel string
	Badge       string
	BadgeText   string
}

type StatusIncidentUpdateView struct {
	Timestamp    string
	TimestampISO string
	Status       string
	StatusLabel  string
	Body         string
}

type StatusIncidentView struct {
	Title       string
	Status      string
	StatusLabel string
	StatusBadge string
	StatusText  string
	Impact      string
	ImpactLabel string
	ImpactBadge string
	ImpactText  string
	Started     string
	StartedISO  string
	Resolved    string
	ResolvedISO string
	Updates     []StatusIncidentUpdateView
}

type StatusPageViewModel struct {
	Header     ContentHeaderView
	Banner     *AlertBannerView
	Overview   StatusOverviewView
	Components []StatusComponentView
	Incidents  []StatusIncidentView
}

var (
	templateScriptLabels = map[string]string{
		"kanji":    "Kanji",
		"kana":     "Kana",
		"alphabet": "Alphabet",
		"mixed":    "Mixed Script",
	}
	templateScriptOrder = []string{"kanji", "kana", "alphabet", "mixed"}

	templateShapeLabels = map[string]string{
		"round":  "Round",
		"square": "Square",
		"rect":   "Rectangular",
	}
	templateShapeOrder = []string{"round", "square", "rect"}

	templateRegistrabilityLabels = map[string]string{
		"official":   "Registrable",
		"individual": "Personal Use",
		"internal":   "Internal",
		"informal":   "Informal",
	}
	templateRegistrabilityOrder = []string{"official", "individual", "internal", "informal"}

	templateCategoryLabels = map[string]string{
		"corporate":     "Corporate",
		"brand":         "Brand",
		"personal":      "Personal",
		"operations":    "Operations",
		"international": "International",
		"events":        "Events",
	}
	templateCategoryOrder = []string{"corporate", "brand", "personal", "operations", "international", "events"}

	templateStyleLabels = map[string]string{
		"modern":  "Modern",
		"classic": "Classic",
		"minimal": "Minimal",
		"playful": "Playful",
		"bold":    "Bold",
		"tech":    "Tech",
	}
	templateStyleOrder = []string{"modern", "classic", "minimal", "bold", "playful", "tech"}

	templateLocaleLabels = map[string]string{
		"ja":     "Japanese",
		"en":     "English",
		"hybrid": "Dual Locale",
	}
	templateLocaleOrder = []string{"ja", "en", "hybrid"}

	templateCatalogSeed = []TemplateMeta{
		{
			ID:             "tpl-ring-corporate",
			Name:           "Corporate Ring",
			Summary:        "Compliant round seal tuned for legal filings and corporate paperwork.",
			Script:         "kanji",
			Shape:          "round",
			Registrability: "official",
			Category:       "corporate",
			Style:          "modern",
			Locale:         "ja",
			Badge:          "Popular",
			PrimarySize:    "18mm round",
			Preview: TemplatePreview{
				Background: "bg-gradient-to-br from-indigo-500 via-purple-500 to-pink-500",
				Glyph:      "CR",
				GlyphClass: "text-white text-2xl font-semibold tracking-[0.3em]",
				RingClass:  "ring-4 ring-white/60",
			},
			Usage:     1286,
			Favorites: 412,
			Updated:   time.Date(2024, time.December, 18, 0, 0, 0, 0, time.UTC),
			Tags:      []string{"compliance", "registration", "corporate"},
		},
		{
			ID:             "tpl-square-brand",
			Name:           "Brand Square Mark",
			Summary:        "Offset corner square template for packaging and signage.",
			Script:         "alphabet",
			Shape:          "square",
			Registrability: "informal",
			Category:       "brand",
			Style:          "bold",
			Locale:         "en",
			Badge:          "Trending",
			PrimarySize:    "30mm square",
			Preview: TemplatePreview{
				Background: "bg-gradient-to-br from-orange-500 via-amber-400 to-yellow-300",
				Glyph:      "BR",
				GlyphClass: "text-white text-2xl font-semibold tracking-[0.2em]",
				RingClass:  "ring-4 ring-white/50",
			},
			Usage:     956,
			Favorites: 508,
			Updated:   time.Date(2024, time.November, 20, 0, 0, 0, 0, time.UTC),
			Tags:      []string{"packaging", "stores", "campaign"},
		},
		{
			ID:             "tpl-rect-ledger",
			Name:           "Ledger Header",
			Summary:        "Rectangular ledger stamp with aligned columns for internal operations.",
			Script:         "kanji",
			Shape:          "rect",
			Registrability: "internal",
			Category:       "operations",
			Style:          "minimal",
			Locale:         "ja",
			Badge:          "New",
			PrimarySize:    "55x18mm",
			Preview: TemplatePreview{
				Background: "bg-gradient-to-br from-slate-900 via-slate-700 to-slate-500",
				Glyph:      "LG",
				GlyphClass: "text-teal-200 text-xl font-semibold tracking-[0.2em]",
				RingClass:  "ring-2 ring-slate-400/40",
			},
			Usage:     468,
			Favorites: 184,
			Updated:   time.Date(2025, time.January, 4, 0, 0, 0, 0, time.UTC),
			Tags:      []string{"ledger", "backoffice", "finance"},
		},
		{
			ID:             "tpl-round-personal",
			Name:           "Personal Script Round",
			Summary:        "Gentle double ring layout for signature hanko and personal seals.",
			Script:         "kana",
			Shape:          "round",
			Registrability: "individual",
			Category:       "personal",
			Style:          "classic",
			Locale:         "ja",
			Badge:          "Classic",
			PrimarySize:    "16.5mm round",
			Preview: TemplatePreview{
				Background: "bg-gradient-to-br from-rose-500 via-fuchsia-500 to-violet-500",
				Glyph:      "PS",
				GlyphClass: "text-white text-xl font-semibold tracking-[0.25em]",
				RingClass:  "ring-4 ring-white/40",
			},
			Usage:     782,
			Favorites: 365,
			Updated:   time.Date(2024, time.September, 12, 0, 0, 0, 0, time.UTC),
			Tags:      []string{"signature", "gift", "calligraphy"},
		},
		{
			ID:             "tpl-square-tech",
			Name:           "Tech Grid Square",
			Summary:        "Angular square template for product authentication and hardware packaging.",
			Script:         "alphabet",
			Shape:          "square",
			Registrability: "official",
			Category:       "corporate",
			Style:          "tech",
			Locale:         "en",
			Badge:          "New",
			PrimarySize:    "28mm square",
			Preview: TemplatePreview{
				Background: "bg-gradient-to-br from-cyan-500 via-blue-500 to-indigo-500",
				Glyph:      "TX",
				GlyphClass: "text-white text-2xl font-semibold tracking-[0.25em]",
				RingClass:  "ring-4 ring-white/45",
			},
			Usage:     512,
			Favorites: 230,
			Updated:   time.Date(2024, time.December, 5, 0, 0, 0, 0, time.UTC),
			Tags:      []string{"hardware", "certification", "qr-ready"},
		},
		{
			ID:             "tpl-rect-export",
			Name:           "Export Manifest",
			Summary:        "Dual-language rectangular seal for customs and logistics paperwork.",
			Script:         "mixed",
			Shape:          "rect",
			Registrability: "official",
			Category:       "international",
			Style:          "modern",
			Locale:         "hybrid",
			Badge:          "Dual",
			PrimarySize:    "60x20mm",
			Preview: TemplatePreview{
				Background: "bg-gradient-to-br from-emerald-500 via-teal-500 to-cyan-400",
				Glyph:      "EX",
				GlyphClass: "text-white text-xl font-semibold tracking-[0.3em]",
				RingClass:  "ring-2 ring-white/50",
			},
			Usage:     624,
			Favorites: 298,
			Updated:   time.Date(2024, time.October, 22, 0, 0, 0, 0, time.UTC),
			Tags:      []string{"shipping", "customs", "documentation"},
		},
	}

	templateDetailSeed = map[string]TemplateDetail{
		"tpl-ring-corporate": {
			Lead: "Compliant round seal tuned for corporate registration and notarized filings.",
			Summary: []string{
				"Double-ring layout maintains legal framing with comfortable counter space.",
				"Optimized letterforms prevent fill-in during repeated vermilion impressions.",
			},
			RecommendedSizes: []TemplateSize{
				{Label: "18mm default", Dimensions: "18mm outer / 12mm inner", Usage: "Company registration and tax office submissions"},
				{Label: "21mm expansion", Dimensions: "21mm outer / 14mm inner", Usage: "Executive authorizations and embossed stationery"},
			},
			Constraints: []TemplateConstraint{
				{Label: "Stroke width", Description: "Set primary strokes to at least 0.45mm to pass registry inspections."},
				{Label: "Outer ring", Description: "Keep outer band between 1.1mm and 1.3mm for brass milling stability."},
			},
			PreviewShots: []TemplatePreviewShot{
				{Label: "Positive impression", Background: "bg-gradient-to-br from-indigo-600 via-purple-600 to-rose-500", Glyph: "CR", GlyphClass: "text-white text-3xl font-semibold tracking-[0.35em]", RingClass: "ring-4 ring-white/60", Caption: "Vermilion ink on 90gsm washi stock"},
				{Label: "Negative master", Background: "bg-white", Glyph: "CR", GlyphClass: "text-indigo-600 text-3xl font-semibold tracking-[0.35em]", RingClass: "ring-2 ring-indigo-200", Caption: "Laser-etched brass master for duplication"},
			},
			Stats: TemplateStats{
				Usage:         1286,
				Favorites:     412,
				Satisfaction:  96,
				AvgTurnaround: "48h engraving lead time",
				LastUpdated:   time.Date(2024, time.December, 1, 0, 0, 0, 0, time.UTC),
			},
			Metadata: []TemplateMetadataEntry{
				{Label: "Compatible materials", Value: "Brass, hardened steel, reinforced resin", Hint: "Validated for 3,500 impressions"},
				{Label: "Grid system", Value: "Radial 12 axis grid", Hint: "Letter rotation locked at 30 degree increments"},
				{Label: "Vector delivery", Value: "SVG + DXF outlines", Hint: "Includes outlined fallback for cross-platform export"},
			},
			UseCases: []string{
				"Company registration filings",
				"Corporate contracts and invoices",
				"Board meeting minutes embossing",
			},
			RecommendedInks: []string{"Vermilion", "Deep navy", "Carbon black"},
			Actions: []TemplateAction{
				{Label: "Start design", Href: "/design?template=tpl-ring-corporate", Variant: "primary"},
				{Label: "Download guide", Href: "/guides/corporate-ring", Variant: "outline"},
			},
		},
		"tpl-square-brand": {
			Lead: "Offset square layout for bold retail campaigns and packaging sleeves.",
			Summary: []string{
				"Supports alternate border weights for foil or ink applications.",
				"Corner anchor grid keeps logotype balanced in responsive placements.",
			},
			RecommendedSizes: []TemplateSize{
				{Label: "30mm base", Dimensions: "30mm x 30mm", Usage: "Packaging seals and swing tags"},
				{Label: "40mm storefront", Dimensions: "40mm x 40mm", Usage: "Signage decals and window vinyl"},
				{Label: "24mm compact", Dimensions: "24mm x 24mm", Usage: "Product authentication labels"},
			},
			Constraints: []TemplateConstraint{
				{Label: "Corner radius", Description: "Maintain 2.4mm corner cuts for consistent die stamping."},
				{Label: "Safe area", Description: "Keep typography inside 18mm square to preserve breathing room."},
			},
			PreviewShots: []TemplatePreviewShot{
				{Label: "Foil mock", Background: "bg-gradient-to-br from-orange-500 via-amber-400 to-yellow-300", Glyph: "BR", GlyphClass: "text-white text-3xl font-semibold tracking-[0.25em]", RingClass: "ring-4 ring-white/45", Caption: "Gold foil on textured cardstock"},
				{Label: "Ink transfer", Background: "bg-slate-900", Glyph: "BR", GlyphClass: "text-amber-300 text-3xl font-semibold tracking-[0.25em]", RingClass: "ring-2 ring-amber-400/60", Caption: "Soya ink on kraft label stock"},
			},
			Stats: TemplateStats{
				Usage:         956,
				Favorites:     508,
				Satisfaction:  94,
				AvgTurnaround: "72h vector refinement",
				LastUpdated:   time.Date(2024, time.November, 20, 0, 0, 0, 0, time.UTC),
			},
			Metadata: []TemplateMetadataEntry{
				{Label: "Typography", Value: "Inter Bold with outline layer", Hint: "Includes uppercase and numeric alternates"},
				{Label: "Grid", Value: "8x8 modular grid", Hint: "Corner offsets locked to 4mm increments"},
				{Label: "Export", Value: "SVG, PDF, PNG @ 1200px", Hint: "Includes transparent and solid backgrounds"},
			},
			UseCases: []string{
				"Limited edition packaging seals",
				"Storefront vinyl applications",
				"Pop-up event collateral",
			},
			RecommendedInks: []string{"Carmine", "Gold", "Midnight blue"},
			Actions: []TemplateAction{
				{Label: "Start design", Href: "/design?template=tpl-square-brand", Variant: "primary"},
				{Label: "Duplicate to workspace", Href: "/workspace/new?source=tpl-square-brand", Variant: "outline"},
			},
		},
		"tpl-rect-ledger": {
			Lead: "Ledger header with aligned columns for accounting and internal ops.",
			Summary: []string{
				"Column guides speed daily stamping on invoices and delivery receipts.",
				"Baseline grid tuned for carbon copy paper to avoid ghosting.",
			},
			RecommendedSizes: []TemplateSize{
				{Label: "55x18mm primary", Dimensions: "55mm x 18mm", Usage: "Delivery dockets and purchase orders"},
				{Label: "60x20mm extended", Dimensions: "60mm x 20mm", Usage: "Receiving inspection sheets"},
			},
			Constraints: []TemplateConstraint{
				{Label: "Column gutters", Description: "Gutter width must remain 3mm to preserve handwriting space."},
				{Label: "Serial block", Description: "Keep serial label at 8pt to align with printed ledger boxes."},
			},
			PreviewShots: []TemplatePreviewShot{
				{Label: "Ink ledger", Background: "bg-gradient-to-br from-slate-900 via-slate-700 to-slate-500", Glyph: "LG", GlyphClass: "text-teal-200 text-2xl font-semibold tracking-[0.25em]", RingClass: "ring-2 ring-teal-200/40", Caption: "Oil-based ink on NCR receipt pad"},
				{Label: "Embossed brass", Background: "bg-white", Glyph: "LG", GlyphClass: "text-slate-700 text-2xl font-semibold tracking-[0.25em]", RingClass: "ring-2 ring-slate-400/60", Caption: "Embossed brass plate for repetitive use"},
			},
			Stats: TemplateStats{
				Usage:         468,
				Favorites:     184,
				Satisfaction:  92,
				AvgTurnaround: "36h milling window",
				LastUpdated:   time.Date(2025, time.January, 4, 0, 0, 0, 0, time.UTC),
			},
			Metadata: []TemplateMetadataEntry{
				{Label: "Columns", Value: "Date, department, signature", Hint: "Editable text layers for localization"},
				{Label: "Alignment", Value: "Left baseline grid", Hint: "Matches A5 ledger baseline"},
				{Label: "Inking", Value: "Low bleed rubber", Hint: "Ideal for duplicate forms"},
			},
			UseCases: []string{
				"Accounting ledgers",
				"Inventory receiving",
				"Warehouse dispatch slips",
			},
			RecommendedInks: []string{"Graphite", "Deep green"},
			Actions: []TemplateAction{
				{Label: "Start design", Href: "/design?template=tpl-rect-ledger", Variant: "primary"},
				{Label: "Download sample CSV", Href: "/downloads/tpl-rect-ledger.csv", Variant: "outline"},
			},
		},
		"tpl-round-personal": {
			Lead: "Soft double-ring signature template for personal correspondence.",
			Summary: []string{
				"Rounded letterforms pair with hand-script signatures and diaries.",
				"Inner monogram cap height matches 13mm resin blanks for hobby engraving.",
			},
			RecommendedSizes: []TemplateSize{
				{Label: "16.5mm default", Dimensions: "16.5mm outer / 11mm inner", Usage: "Daily signature hanko"},
				{Label: "13.5mm compact", Dimensions: "13.5mm outer / 9mm inner", Usage: "Travel cases and notebooks"},
			},
			Constraints: []TemplateConstraint{
				{Label: "Calligraphic stroke", Description: "Maintain 0.35mm stroke minimum to prevent ink pooling."},
				{Label: "Monogram spacing", Description: "Keep initials centered with 0.8mm breathing room."},
			},
			PreviewShots: []TemplatePreviewShot{
				{Label: "Vermilion impression", Background: "bg-gradient-to-br from-rose-500 via-fuchsia-500 to-violet-500", Glyph: "PS", GlyphClass: "text-white text-3xl font-semibold tracking-[0.3em]", RingClass: "ring-4 ring-white/40", Caption: "Water-based ink on diary paper"},
				{Label: "Emboss preview", Background: "bg-white", Glyph: "PS", GlyphClass: "text-rose-500 text-3xl font-semibold tracking-[0.3em]", RingClass: "ring-2 ring-rose-300/60", Caption: "Resin die emboss test"},
			},
			Stats: TemplateStats{
				Usage:         782,
				Favorites:     365,
				Satisfaction:  97,
				AvgTurnaround: "24h laser prep",
				LastUpdated:   time.Date(2024, time.September, 12, 0, 0, 0, 0, time.UTC),
			},
			Metadata: []TemplateMetadataEntry{
				{Label: "Script model", Value: "Rounded gothic with alternates", Hint: "Includes three monogram sets"},
				{Label: "Outer band", Value: "0.95mm", Hint: "Balanced for resin and wood handles"},
				{Label: "Packaging", Value: "Gift-ready PDF insert", Hint: "Includes personalization instructions"},
			},
			UseCases: []string{
				"Personal correspondence",
				"Journal stamping",
				"Gift personalization",
			},
			RecommendedInks: []string{"Vermilion", "Cherry blossom", "Slate"},
			Actions: []TemplateAction{
				{Label: "Start design", Href: "/design?template=tpl-round-personal", Variant: "primary"},
				{Label: "Share preview", Href: "/share?template=tpl-round-personal", Variant: "outline"},
			},
		},
		"tpl-square-tech": {
			Lead: "Angular grid for high-tech authentication labels and device packaging.",
			Summary: []string{
				"Diagonal lattice allows microtext and QR anchoring inside safe zones.",
				"Outlined glyph retains clarity on anodized aluminum and matte vinyl.",
			},
			RecommendedSizes: []TemplateSize{
				{Label: "28mm base", Dimensions: "28mm x 28mm", Usage: "Device packaging seals"},
				{Label: "18mm badge", Dimensions: "18mm x 18mm", Usage: "Component certification tags"},
			},
			Constraints: []TemplateConstraint{
				{Label: "Microtext band", Description: "Reserve 1.6mm band at edges for serial microtext."},
				{Label: "QR anchor", Description: "Keep anchor triangle aligned to 45 degree grid for scanning accuracy."},
			},
			PreviewShots: []TemplatePreviewShot{
				{Label: "Holographic mock", Background: "bg-gradient-to-br from-cyan-500 via-blue-500 to-indigo-500", Glyph: "TX", GlyphClass: "text-white text-3xl font-semibold tracking-[0.3em]", RingClass: "ring-4 ring-white/45", Caption: "Holographic foil on matte box"},
				{Label: "Tamper label", Background: "bg-slate-900", Glyph: "TX", GlyphClass: "text-cyan-300 text-3xl font-semibold tracking-[0.3em]", RingClass: "ring-2 ring-cyan-400/60", Caption: "Security sticker application"},
			},
			Stats: TemplateStats{
				Usage:         512,
				Favorites:     230,
				Satisfaction:  93,
				AvgTurnaround: "72h engraving and proof",
				LastUpdated:   time.Date(2024, time.December, 5, 0, 0, 0, 0, time.UTC),
			},
			Metadata: []TemplateMetadataEntry{
				{Label: "Grid system", Value: "12x12 diagonal grid", Hint: "Supports 45 degree anchors"},
				{Label: "Format", Value: "SVG + EPS with live stroke", Hint: "Ideal for CNC milling"},
				{Label: "Security layer", Value: "Optional microtext ring", Hint: "Editable for per-batch serials"},
			},
			UseCases: []string{
				"Hardware compliance tags",
				"Tamper-evident seals",
				"Smart device packaging",
			},
			RecommendedInks: []string{"Cyan", "Silver", "Jet black"},
			Actions: []TemplateAction{
				{Label: "Start design", Href: "/design?template=tpl-square-tech", Variant: "primary"},
				{Label: "Request approval kit", Href: "/inbox/new?subject=tpl-square-tech", Variant: "outline"},
			},
		},
		"tpl-rect-export": {
			Lead: "Dual-language export manifest template for customs and logistics workflows.",
			Summary: []string{
				"Bilingual layout balances Latin and Kanji scripts with equal emphasis.",
				"Grid aligns with A4 shipping forms and accepts barcode overlays.",
			},
			RecommendedSizes: []TemplateSize{
				{Label: "60x20mm default", Dimensions: "60mm x 20mm", Usage: "Export documentation primary seal"},
				{Label: "70x22mm extended", Dimensions: "70mm x 22mm", Usage: "International certificates of origin"},
			},
			Constraints: []TemplateConstraint{
				{Label: "Language pairing", Description: "Maintain Japanese script on left and English on right with center divider."},
				{Label: "Barcode clearance", Description: "Leave 6mm bottom margin for barcode overlays."},
			},
			PreviewShots: []TemplatePreviewShot{
				{Label: "Manifest proof", Background: "bg-gradient-to-br from-emerald-500 via-teal-500 to-cyan-400", Glyph: "EX", GlyphClass: "text-white text-3xl font-semibold tracking-[0.3em]", RingClass: "ring-2 ring-white/60", Caption: "Logistics manifest sample"},
				{Label: "Steel die", Background: "bg-white", Glyph: "EX", GlyphClass: "text-emerald-600 text-3xl font-semibold tracking-[0.3em]", RingClass: "ring-2 ring-emerald-200/60", Caption: "Steel die finishing preview"},
			},
			Stats: TemplateStats{
				Usage:         624,
				Favorites:     298,
				Satisfaction:  95,
				AvgTurnaround: "96h bilingual QA",
				LastUpdated:   time.Date(2024, time.October, 22, 0, 0, 0, 0, time.UTC),
			},
			Metadata: []TemplateMetadataEntry{
				{Label: "Languages", Value: "Japanese / English", Hint: "Editable text layers for localization"},
				{Label: "Document fit", Value: "A4 landscape grid", Hint: "Aligns with customs manifest layout"},
				{Label: "Export bundle", Value: "SVG + PDF + DOCX", Hint: "Includes bilingual instructions"},
			},
			UseCases: []string{
				"International shipping",
				"Customs declarations",
				"Freight forwarding documents",
			},
			RecommendedInks: []string{"Emerald", "Navy", "Charcoal"},
			Actions: []TemplateAction{
				{Label: "Start design", Href: "/design?template=tpl-rect-export", Variant: "primary"},
				{Label: "Share bilingual brief", Href: "/share?template=tpl-rect-export", Variant: "outline"},
			},
		},
	}
)

func templateData(lang string) []TemplateMeta {
	out := make([]TemplateMeta, len(templateCatalogSeed))
	copy(out, templateCatalogSeed)
	for i := range out {
		decorateTemplateMeta(&out[i])
	}
	return out
}

func templateDetailFor(lang, id string) (*TemplateDetail, bool) {
	data := templateData(lang)
	var meta TemplateMeta
	for _, tpl := range data {
		if tpl.ID == id {
			meta = tpl
			break
		}
	}
	if meta.ID == "" {
		return nil, false
	}
	seed, ok := templateDetailSeed[id]
	if !ok {
		return nil, false
	}
	detail := seed
	detail.Template = meta
	if detail.Stats.Usage == 0 {
		detail.Stats.Usage = meta.Usage
	}
	if detail.Stats.Favorites == 0 {
		detail.Stats.Favorites = meta.Favorites
	}
	if detail.Stats.LastUpdated.IsZero() {
		detail.Stats.LastUpdated = meta.Updated
	}
	detail.Related = selectRelatedTemplates(meta, data)
	return &detail, true
}

func buildTemplateListProps(lang string, q url.Values) map[string]any {
	normalize := func(v string) string {
		return strings.ToLower(strings.TrimSpace(v))
	}
	script := normalize(q.Get("script"))
	shape := normalize(q.Get("shape"))
	registrability := normalize(q.Get("registrability"))
	category := normalize(q.Get("category"))
	style := normalize(q.Get("style"))
	locale := normalize(q.Get("locale"))
	search := strings.TrimSpace(q.Get("q"))
	page := 1
	if v := strings.TrimSpace(q.Get("page")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	per := 9
	if v := strings.TrimSpace(q.Get("per")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			if n < 3 {
				n = 3
			}
			if n > 24 {
				n = 24
			}
			per = n
		}
	}

	data := templateData(lang)
	var filtered []TemplateMeta
	searchLower := strings.ToLower(search)
	for _, tpl := range data {
		if script != "" && script != "all" && tpl.Script != script {
			continue
		}
		if shape != "" && shape != "all" && tpl.Shape != shape {
			continue
		}
		if registrability != "" && registrability != "all" && tpl.Registrability != registrability {
			continue
		}
		if category != "" && category != "all" && tpl.Category != category {
			continue
		}
		if style != "" && style != "all" && tpl.Style != style {
			continue
		}
		if locale != "" && locale != "all" && tpl.Locale != locale {
			continue
		}
		if searchLower != "" {
			haystack := strings.ToLower(tpl.Name + " " + tpl.Summary + " " + strings.Join(tpl.Tags, " "))
			if !strings.Contains(haystack, searchLower) {
				continue
			}
		}
		filtered = append(filtered, tpl)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].Badge == "New" && filtered[j].Badge != "New" {
			return true
		}
		if filtered[j].Badge == "New" && filtered[i].Badge != "New" {
			return false
		}
		if filtered[i].Usage == filtered[j].Usage {
			return filtered[i].Updated.After(filtered[j].Updated)
		}
		return filtered[i].Usage > filtered[j].Usage
	})

	total := len(filtered)
	if page < 1 {
		page = 1
	}
	start := (page - 1) * per
	if start > total {
		start = total
	}
	end := start + per
	if end > total {
		end = total
	}
	window := filtered[start:end]
	hasMore := end < total

	base := make(url.Values)
	if script != "" {
		base.Set("script", script)
	}
	if shape != "" {
		base.Set("shape", shape)
	}
	if registrability != "" {
		base.Set("registrability", registrability)
	}
	if category != "" {
		base.Set("category", category)
	}
	if style != "" {
		base.Set("style", style)
	}
	if locale != "" {
		base.Set("locale", locale)
	}
	if search != "" {
		base.Set("q", search)
	}
	base.Set("per", strconv.Itoa(per))

	options := map[string]any{
		"Scripts":        buildFilterOptions(data, script, "All scripts", templateScriptLabels, templateScriptOrder, func(t TemplateMeta) string { return t.Script }),
		"Shapes":         buildFilterOptions(data, shape, "All shapes", templateShapeLabels, templateShapeOrder, func(t TemplateMeta) string { return t.Shape }),
		"Registrability": buildFilterOptions(data, registrability, "All registrability", templateRegistrabilityLabels, templateRegistrabilityOrder, func(t TemplateMeta) string { return t.Registrability }),
		"Categories":     buildFilterOptions(data, category, "All categories", templateCategoryLabels, templateCategoryOrder, func(t TemplateMeta) string { return t.Category }),
		"Styles":         buildFilterOptions(data, style, "All styles", templateStyleLabels, templateStyleOrder, func(t TemplateMeta) string { return t.Style }),
		"Locales":        buildFilterOptions(data, locale, "All locales", templateLocaleLabels, templateLocaleOrder, func(t TemplateMeta) string { return t.Locale }),
	}

	nextURL := ""
	if hasMore {
		nextQS := base.Encode()
		if nextQS != "" {
			nextQS += "&"
		}
		nextQS += "page=" + strconv.Itoa(page+1)
		nextURL = "/templates/table?" + nextQS
	}

	props := map[string]any{
		"Lang":        lang,
		"Templates":   window,
		"Total":       total,
		"CountLoaded": end,
		"Page":        page,
		"Per":         per,
		"HasMore":     hasMore,
		"NextPage":    page + 1,
		"NextURL":     nextURL,
		"Filters": map[string]string{
			"Script":         script,
			"Shape":          shape,
			"Registrability": registrability,
			"Category":       category,
			"Style":          style,
			"Locale":         locale,
			"Query":          search,
		},
		"Options": options,
		"BaseQS":  base.Encode(),
		"Append":  page > 1,
		"Empty":   total == 0,
	}
	return props
}

func decorateTemplateMeta(t *TemplateMeta) {
	t.ScriptLabel = labelFor(templateScriptLabels, t.Script)
	t.ShapeLabel = labelFor(templateShapeLabels, t.Shape)
	t.RegistrabilityLabel = labelFor(templateRegistrabilityLabels, t.Registrability)
	t.CategoryLabel = labelFor(templateCategoryLabels, t.Category)
	t.StyleLabel = labelFor(templateStyleLabels, t.Style)
	t.LocaleLabel = labelFor(templateLocaleLabels, t.Locale)
}

func buildFilterOptions(items []TemplateMeta, current, allLabel string, labels map[string]string, order []string, getter func(TemplateMeta) string) []FilterOption {
	counts := make(map[string]int)
	for _, item := range items {
		key := getter(item)
		counts[key]++
	}

	opts := make([]FilterOption, 0, len(counts)+1)
	opts = append(opts, FilterOption{
		Value:    "",
		Label:    allLabel,
		Count:    len(items),
		Selected: current == "" || current == "all",
	})

	seen := make(map[string]struct{})
	for _, key := range order {
		if counts[key] == 0 {
			continue
		}
		label := labelFor(labels, key)
		opts = append(opts, FilterOption{
			Value:    key,
			Label:    label,
			Count:    counts[key],
			Selected: current == key,
		})
		seen[key] = struct{}{}
	}

	var leftover []string
	for key := range counts {
		if _, ok := seen[key]; ok {
			continue
		}
		leftover = append(leftover, key)
	}
	sort.Strings(leftover)
	for _, key := range leftover {
		label := labelFor(labels, key)
		opts = append(opts, FilterOption{
			Value:    key,
			Label:    label,
			Count:    counts[key],
			Selected: current == key,
		})
	}
	return opts
}

func selectRelatedTemplates(current TemplateMeta, data []TemplateMeta) []TemplateMeta {
	related := make([]TemplateMeta, 0, 3)
	seen := map[string]struct{}{current.ID: {}}
	for _, tpl := range data {
		if tpl.ID == current.ID {
			continue
		}
		if tpl.Category == current.Category {
			related = append(related, tpl)
			seen[tpl.ID] = struct{}{}
			if len(related) == 3 {
				return related
			}
		}
	}
	for _, tpl := range data {
		if len(related) == 3 {
			break
		}
		if _, ok := seen[tpl.ID]; ok {
			continue
		}
		related = append(related, tpl)
		seen[tpl.ID] = struct{}{}
	}
	return related
}

func labelFor(m map[string]string, key string) string {
	if v, ok := m[key]; ok && v != "" {
		return v
	}
	return titleCase(key)
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	cleaned := strings.ReplaceAll(strings.ReplaceAll(s, "-", " "), "_", " ")
	parts := strings.Fields(cleaned)
	for i, p := range parts {
		lower := strings.ToLower(p)
		if len(lower) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(lower[:1]) + lower[1:]
	}
	return strings.Join(parts, " ")
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
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	guides, err := cmsClient.ListGuides(ctx, cms.ListGuidesOptions{Lang: lang, Limit: 3})
	if err != nil {
		log.Printf("guides: latest: %v", err)
	}
	cards := make([]GuideCard, 0, len(guides))
	hashParts := []string{lang}
	for _, g := range guides {
		cards = append(cards, guideCardFromGuide(g, lang))
		hashParts = append(hashParts, g.Slug)
		if !g.PublishAt.IsZero() {
			hashParts = append(hashParts, g.PublishAt.Format(time.RFC3339))
		}
	}

	etag := etagFor("guides:", hashParts...)
	if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=120")
	w.Header().Set("ETag", etag)

	props := map[string]any{
		"Guides": cards,
		"Lang":   lang,
	}
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
