package main

import (
	"bytes"
	"fmt"
	"html/template"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type DesignEditorFont struct {
	ID          string
	Name        string
	Script      string
	Sample      string
	Category    string
	Description string
}

type DesignEditorTemplate struct {
	ID          string
	Name        string
	Shape       string
	Size        string
	Preview     string
	Description string
	Badge       string
	UseCase     string
}

type KanjiMappingCandidate struct {
	ID         string
	Kanji      string
	Reading    string
	Notes      string
	Confidence int
	Variant    string
	Primary    bool
}

type DesignEditorToast struct {
	ID    string
	Kind  string
	Title string
	Body  string
}

type DesignEditorState struct {
	Lang         string
	Mode         string
	Name         string
	TemplateID   string
	FontID       string
	StrokeWeight int
	Tracking     int
	Contrast     int
	Rotation     int
	Grain        int
	ShowGuides   bool
	ShowOutline  bool
	Errors       map[string]string
	Validation   []string
	StatusLabel  string
	StatusTone   string
	LastSaved    string
	DraftSaved   bool
	AIStatus     string
}

type DesignEditorPreview struct {
	SVG          template.HTML
	TemplateName string
	FontName     string
	Stats        []DesignPreviewStat
	Issues       []DesignPreviewIssue
	UpdatedLabel string
	Mode         string
}

type DesignPreviewStat struct {
	Label string
	Value string
}

type DesignPreviewIssue struct {
	Level   string
	Message string
}

type DesignEditorView struct {
	Lang             string
	State            DesignEditorState
	Fonts            []DesignEditorFont
	Templates        []DesignEditorTemplate
	SelectedFont     DesignEditorFont
	SelectedTemplate DesignEditorTemplate
	Preview          DesignEditorPreview
	Toasts           []DesignEditorToast
	Query            string
	NamePlaceholder  string
}

func designEditorFonts(lang string) []DesignEditorFont {
	return []DesignEditorFont{
		{
			ID:          "jp-mincho",
			Name:        editorCopy(lang, "和文明朝", "Mincho Traditional"),
			Script:      editorCopy(lang, "明朝体", "Mincho"),
			Sample:      editorCopy(lang, "山田太郎", "Mincho Sample"),
			Category:    "serif",
			Description: editorCopy(lang, "公的印鑑で最も多く使われるバランスの整った書体です。", "Balanced strokes commonly accepted for registrable corporate seals."),
		},
		{
			ID:          "jp-gothic",
			Name:        editorCopy(lang, "和文ゴシック", "Gothic Modern"),
			Script:      editorCopy(lang, "ゴシック体", "Gothic"),
			Sample:      editorCopy(lang, "有限会社フィールド", "Gothic Sample"),
			Category:    "sans",
			Description: editorCopy(lang, "視認性を重視した角張った線で、部署印や確認印に適しています。", "Sharp, legible strokes ideal for departmental and confirmation seals."),
		},
		{
			ID:          "seal-tenkyu",
			Name:        editorCopy(lang, "篆書・天久", "Seal Script Tenkyu"),
			Script:      editorCopy(lang, "篆書体", "Seal Script"),
			Sample:      editorCopy(lang, "藤原", "Tenkyu Sample"),
			Category:    "traditional",
			Description: editorCopy(lang, "伝統的な篆書体をベースにした細身の書体。由緒ある印影に。", "Elegant seal script inspired by traditional tenkyo carving for ceremonial use."),
		},
	}
}

func designEditorTemplates(lang string) []DesignEditorTemplate {
	return []DesignEditorTemplate{
		{
			ID:          "tpl-ring-corporate",
			Name:        editorCopy(lang, "丸印・法人代表", "Corporate Round"),
			Shape:       editorCopy(lang, "丸型", "Round"),
			Size:        "18 mm",
			Preview:     "/assets/previews/tpl-ring-corporate.svg",
			Description: editorCopy(lang, "外周に社名、中心に役職を置いた汎用的なレイアウト。", "Outer ring for company name with title centered for broad corporate use."),
			Badge:       editorCopy(lang, "推奨", "Recommended"),
			UseCase:     editorCopy(lang, "法人代表印", "Representative seal"),
		},
		{
			ID:          "tpl-square-brand",
			Name:        editorCopy(lang, "角印・ブランド", "Square Brand Mark"),
			Shape:       editorCopy(lang, "角型", "Square"),
			Size:        "21 mm",
			Preview:     "/assets/previews/tpl-square-brand.svg",
			Description: editorCopy(lang, "筆勢を活かした四角レイアウト。ロゴや屋号向け。", "Dynamic square layout that leaves room for logotypes or shop signatures."),
			Badge:       editorCopy(lang, "人気", "Popular"),
			UseCase:     editorCopy(lang, "ブランド・店舗印", "Brand seal"),
		},
		{
			ID:          "tpl-rect-ledger",
			Name:        editorCopy(lang, "長方印・帳票", "Ledger Rectangular"),
			Shape:       editorCopy(lang, "長方形", "Rectangular"),
			Size:        "21 × 60 mm",
			Preview:     "/assets/previews/tpl-rect-ledger.svg",
			Description: editorCopy(lang, "行揃え済みの縦書きレイアウト。領収書や契約書に。", "Pre-aligned vertical layout for ledgers, receipts, and document workflows."),
			Badge:       "",
			UseCase:     editorCopy(lang, "会計・書類", "Accounting & ledgers"),
		},
	}
}

func defaultDesignEditorState(lang string) DesignEditorState {
	return DesignEditorState{
		Lang:         lang,
		Mode:         "text",
		StrokeWeight: 58,
		Tracking:     44,
		Contrast:     70,
		Rotation:     50,
		Grain:        24,
		ShowGuides:   true,
		ShowOutline:  false,
		StatusLabel:  editorCopy(lang, "自動保存 待機中", "Autosave ready"),
		StatusTone:   "muted",
	}
}

func editorCopy(lang, ja, en string) string {
	if lang == "ja" {
		return ja
	}
	return en
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func readSlider(form url.Values, key string, fallback, min, max int) int {
	if form == nil {
		return fallback
	}
	val := strings.TrimSpace(form.Get(key))
	if val == "" {
		return fallback
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return clampInt(n, min, max)
}

func boolFromForm(form url.Values, key string, fallback bool) bool {
	if form == nil {
		return fallback
	}
	vals, ok := form[key]
	if !ok || len(vals) == 0 {
		return fallback
	}
	v := strings.TrimSpace(strings.ToLower(vals[len(vals)-1]))
	if v == "" {
		return true
	}
	switch v {
	case "1", "true", "on", "yes":
		return true
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
}

func appendUnique(list []string, s string) []string {
	if s == "" {
		return list
	}
	for _, existing := range list {
		if existing == s {
			return list
		}
	}
	return append(list, s)
}

var (
	legacyKanjiReplacer = strings.NewReplacer(
		"辺", "邊",
		"沢", "澤",
		"斉", "齊",
		"斎", "齋",
		"広", "廣",
		"崎", "﨑",
		"国", "國",
		"高", "髙",
	)
	simplifiedKanjiReplacer = strings.NewReplacer(
		"邊", "辺",
		"澤", "沢",
		"齊", "斉",
		"齋", "斎",
		"廣", "広",
		"﨑", "崎",
		"國", "国",
		"髙", "高",
	)
)

func kanjiMappingCandidates(lang, name string) []KanjiMappingCandidate {
	cleaned := strings.TrimSpace(name)
	if cleaned == "" {
		return nil
	}

	base, reading, noteHint := mapNameToKanji(cleaned, lang)

	primary := KanjiMappingCandidate{
		ID:         "registry",
		Kanji:      base,
		Reading:    reading,
		Variant:    editorCopy(lang, "戸籍標準", "Registry preferred"),
		Confidence: 94,
		Primary:    true,
		Notes:      editorCopy(lang, fmt.Sprintf("入力「%s」に最も近い候補です。", cleaned), fmt.Sprintf("Closest registry-safe spelling for %q.", cleaned)),
	}
	if noteHint != "" {
		primary.Notes = noteHint
	}

	candidates := []KanjiMappingCandidate{primary}

	if legacy := legacyKanjiReplacer.Replace(base); legacy != "" && legacy != base {
		candidates = append(candidates, KanjiMappingCandidate{
			ID:         "legacy",
			Kanji:      legacy,
			Reading:    reading,
			Variant:    editorCopy(lang, "旧字体", "Legacy glyphs"),
			Confidence: 82,
			Notes:      editorCopy(lang, "旧字体を含むため自治体によっては再提出が必要です。", "Contains legacy characters. Some municipalities may request resubmission."),
		})
	}

	if simplified := simplifiedKanjiReplacer.Replace(base); simplified != "" && simplified != base {
		candidates = append(candidates, KanjiMappingCandidate{
			ID:         "modern",
			Kanji:      simplified,
			Reading:    reading,
			Variant:    editorCopy(lang, "新字体", "Modernised glyphs"),
			Confidence: 76,
			Notes:      editorCopy(lang, "彫刻機で再現しやすいように文字を簡略化しています。", "Normalized characters for higher legibility with machine engraving."),
		})
	}

	return candidates
}

func mapNameToKanji(name, lang string) (kanji, reading, hint string) {
	if containsKanji(name) {
		return name, "", ""
	}

	norm := normalizeNameKey(name)
	if mapped, ok := knownNameMappings[norm]; ok {
		return mapped.Kanji, mapped.Reading, ""
	}

	// Fall back to the most common sample and communicate placeholder state.
	sample := editorCopy(lang, "山田太郎", "山田太郎")
	sampleReading := "やまだ たろう"
	hint = editorCopy(lang,
		"API 連携前のため漢字候補をサンプル表示しています。正式版では入力内容から即時変換されます。",
		"Until the API integration is ready we display a representative sample. Production will convert your input automatically.")
	return sample, sampleReading, hint
}

func containsKanji(s string) bool {
	for _, r := range s {
		if unicode.In(r, unicode.Han) {
			return true
		}
	}
	return false
}

func normalizeNameKey(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case unicode.In(r, unicode.Hiragana, unicode.Katakana):
			b.WriteRune(r)
		case unicode.In(r, unicode.Han):
			// Preserve kanji for lookups (remove whitespace/punctuation only).
			b.WriteRune(r)
		}
	}
	return b.String()
}

type nameMappingSample struct {
	Kanji   string
	Reading string
}

var knownNameMappings = map[string]nameMappingSample{
	"yamadataro":   {Kanji: "山田太郎", Reading: "やまだ たろう"},
	"suzukihanako": {Kanji: "鈴木花子", Reading: "すずき はなこ"},
	"tanakashoji":  {Kanji: "田中商事", Reading: "たなか しょうじ"},
	"hankofield":   {Kanji: "判子フィールド", Reading: "はんこ ふぃーるど"},
	"saito":        {Kanji: "斎藤", Reading: "さいとう"},
	"kato":         {Kanji: "加藤", Reading: "かとう"},
}

func findDesignEditorFont(fonts []DesignEditorFont, id string) (DesignEditorFont, bool) {
	for _, f := range fonts {
		if f.ID == id {
			return f, true
		}
	}
	if len(fonts) == 0 {
		return DesignEditorFont{}, false
	}
	return fonts[0], false
}

func findDesignEditorTemplate(templates []DesignEditorTemplate, id string) (DesignEditorTemplate, bool) {
	for _, t := range templates {
		if t.ID == id {
			return t, true
		}
	}
	if len(templates) == 0 {
		return DesignEditorTemplate{}, false
	}
	return templates[0], false
}

func designEditorStateValues(state DesignEditorState) url.Values {
	v := url.Values{}
	v.Set("mode", state.Mode)
	v.Set("name", state.Name)
	v.Set("template", state.TemplateID)
	v.Set("font", state.FontID)
	v.Set("stroke_weight", strconv.Itoa(state.StrokeWeight))
	v.Set("tracking", strconv.Itoa(state.Tracking))
	v.Set("contrast", strconv.Itoa(state.Contrast))
	v.Set("rotation", strconv.Itoa(state.Rotation))
	v.Set("grain", strconv.Itoa(state.Grain))
	if state.ShowGuides {
		v.Set("guides", "1")
	}
	if state.ShowOutline {
		v.Set("outline", "1")
	}
	return v
}

func buildDesignEditorView(lang string, form url.Values) DesignEditorView {
	state := defaultDesignEditorState(lang)
	if form == nil {
		form = url.Values{}
	}

	if mode := strings.TrimSpace(form.Get("mode")); mode != "" {
		switch mode {
		case "text", "upload", "logo":
			state.Mode = mode
		default:
			state.Mode = "text"
		}
	}

	state.Name = strings.TrimSpace(form.Get("name"))

	fonts := designEditorFonts(lang)
	fontID := strings.TrimSpace(form.Get("font"))
	selectedFont, ok := findDesignEditorFont(fonts, fontID)
	if !ok {
		fontID = selectedFont.ID
	}
	state.FontID = fontID

	templates := designEditorTemplates(lang)
	templateID := strings.TrimSpace(form.Get("template"))
	selectedTemplate, ok := findDesignEditorTemplate(templates, templateID)
	if !ok {
		templateID = selectedTemplate.ID
	}
	state.TemplateID = templateID

	state.StrokeWeight = readSlider(form, "stroke_weight", state.StrokeWeight, 0, 100)
	state.Tracking = readSlider(form, "tracking", state.Tracking, 0, 100)
	state.Contrast = readSlider(form, "contrast", state.Contrast, 0, 100)
	state.Rotation = readSlider(form, "rotation", state.Rotation, 0, 100)
	state.Grain = readSlider(form, "grain", state.Grain, 0, 100)
	state.ShowGuides = boolFromForm(form, "guides", state.ShowGuides)
	state.ShowOutline = boolFromForm(form, "outline", state.ShowOutline)

	action := strings.TrimSpace(form.Get("action"))
	errors := map[string]string{}
	var toasts []DesignEditorToast

	now := time.Now().In(jstLocation)
	nowLabel := now.Format("15:04")
	state.StatusLabel = editorCopy(lang, "自動保存 待機中", "Autosave ready")
	state.StatusTone = "muted"

	if action == "save" || action == "draft" {
		if state.Name == "" {
			errors["name"] = editorCopy(lang, "保存するにはデザイン名を入力してください。", "Enter a seal name before saving.")
		} else if runeCount := len([]rune(state.Name)); runeCount > 16 {
			errors["name"] = editorCopy(lang, "16文字以内で入力してください。", "Use 16 or fewer characters for registry compliance.")
		}
		if len(errors) == 0 {
			if action == "save" {
				state.StatusLabel = editorCopy(lang, "数秒前に保存", "Saved just now")
				state.StatusTone = "success"
				state.LastSaved = nowLabel
				toasts = append(toasts, DesignEditorToast{
					ID:    fmt.Sprintf("toast-%d", time.Now().UnixNano()),
					Kind:  "success",
					Title: editorCopy(lang, "デザインを保存しました", "Design saved"),
					Body:  editorCopy(lang, "最新のレイアウトをワークスペースに同期しました。", "Latest layout synced to your workspace."),
				})
			} else {
				state.StatusLabel = editorCopy(lang, "下書きを更新しました", "Draft saved")
				state.StatusTone = "info"
				state.LastSaved = nowLabel
				state.DraftSaved = true
				toasts = append(toasts, DesignEditorToast{
					ID:    fmt.Sprintf("toast-%d", time.Now().UnixNano()),
					Kind:  "info",
					Title: editorCopy(lang, "下書きを保存しました", "Draft updated"),
					Body:  editorCopy(lang, "続きはいつでも再開できます。", "Come back anytime to keep iterating."),
				})
			}
		} else {
			state.StatusLabel = editorCopy(lang, "入力エラーを解消してください", "Resolve form errors")
			state.StatusTone = "error"
			toasts = append(toasts, DesignEditorToast{
				ID:    fmt.Sprintf("toast-%d", time.Now().UnixNano()),
				Kind:  "error",
				Title: editorCopy(lang, "保存できませんでした", "Could not save"),
				Body:  errors["name"],
			})
		}
	}

	if action == "ai" {
		state.AIStatus = fmt.Sprintf(editorCopy(lang, "AIアシスト待機中 — %s", "AI assistant queued — %s"), nowLabel)
		toasts = append(toasts, DesignEditorToast{
			ID:    fmt.Sprintf("toast-%d", time.Now().UnixNano()),
			Kind:  "info",
			Title: editorCopy(lang, "AI レイアウトを生成中", "AI layout request queued"),
			Body:  editorCopy(lang, "数秒以内にプレビューへ反映されます。", "Expect preview updates within a few seconds."),
		})
	}

	if len(errors) > 0 {
		state.Errors = errors
	} else {
		state.Errors = nil
	}

	if state.Name != "" && len([]rune(state.Name)) < 2 {
		state.Validation = appendUnique(state.Validation, editorCopy(lang, "2文字以上で入力すると読みやすさが向上します。", "Use at least two characters for balanced carving."))
	}
	if state.Tracking > 82 {
		state.Validation = appendUnique(state.Validation, editorCopy(lang, "字間が広めです。80%以下に調整すると登録が通りやすくなります。", "Spacing is wide; trim below 80% for registry approval."))
	}
	if state.StrokeWeight > 88 {
		state.Validation = appendUnique(state.Validation, editorCopy(lang, "線が太めです。85%以下がおすすめです。", "Strokes look heavy; aim for ≤85% to avoid ink bleed."))
	}
	if state.Contrast < 25 {
		state.Validation = appendUnique(state.Validation, editorCopy(lang, "コントラストが低めです。押し跡が薄くなる恐れがあります。", "Low contrast may produce faint impressions."))
	}

	preview := buildDesignEditorPreview(state, selectedFont, selectedTemplate, lang)
	values := designEditorStateValues(state)
	placeholder := editorCopy(lang, "例: 山田太郎／合同会社フィールド", "e.g. Taro Yamada / Field Studio LLC")

	return DesignEditorView{
		Lang:             lang,
		State:            state,
		Fonts:            fonts,
		Templates:        templates,
		SelectedFont:     selectedFont,
		SelectedTemplate: selectedTemplate,
		Preview:          preview,
		Toasts:           toasts,
		Query:            values.Encode(),
		NamePlaceholder:  placeholder,
	}
}

func buildDesignEditorPreview(state DesignEditorState, font DesignEditorFont, tpl DesignEditorTemplate, lang string) DesignEditorPreview {
	displayName := strings.TrimSpace(state.Name)
	if displayName == "" {
		displayName = editorCopy(lang, "山田太郎", "Hanko Field")
	}

	strokeWidth := 6.0 + (float64(state.StrokeWeight)/100.0)*7.0
	tracking := float64(state.Tracking)
	letterSpacing := 2.2 + (tracking-50.0)/50.0*5.5
	letterSpacing = math.Max(0.8, letterSpacing)
	fontSize := 104.0 - (tracking-50.0)/50.0*18.0
	fontSize = math.Max(72, fontSize)
	rotation := (float64(state.Rotation) - 50.0) / 50.0 * 12.0
	var fillColor string
	switch {
	case state.Contrast < 25:
		fillColor = "#4b5563"
	case state.Contrast > 80:
		fillColor = "#0f172a"
	default:
		fillColor = "#111827"
	}
	grainAlpha := 0.18 + (float64(state.Grain)/100.0)*0.28

	var buf bytes.Buffer
	buf.WriteString(`<svg role="img" aria-label="`)
	buf.WriteString(template.HTMLEscapeString(displayName))
	buf.WriteString(` preview" viewBox="0 0 320 320" xmlns="http://www.w3.org/2000/svg">`)
	buf.WriteString(fmt.Sprintf(`<defs><filter id="editor-grain" x="-10%%" y="-10%%" width="120%%" height="120%%"><feTurbulence type="fractalNoise" baseFrequency="0.9" numOctaves="3" stitchTiles="stitch"/><feColorMatrix type="matrix" values="0 0 0 0 0  0 0 0 0 0  0 0 0 0 0  0 0 0 %.2f 0"/></filter></defs>`, grainAlpha))
	buf.WriteString(`<rect width="320" height="320" rx="20" fill="#f9fafb"/>`)
	buf.WriteString(`<circle cx="160" cy="160" r="132" fill="none" stroke="#d1d5db" stroke-dasharray="6 6" stroke-width="1" opacity="0.5"/>`)
	if state.ShowOutline {
		buf.WriteString(`<circle cx="160" cy="160" r="142" fill="none" stroke="#6366f1" stroke-dasharray="8 6" stroke-width="1.5" opacity="0.32"/>`)
	}
	buf.WriteString(fmt.Sprintf(`<circle cx="160" cy="160" r="124" fill="#fff" stroke="#111827" stroke-width="%.2f" filter="url(#editor-grain)"/>`, strokeWidth))
	if state.ShowGuides {
		buf.WriteString(`<line x1="160" y1="28" x2="160" y2="292" stroke="#818cf8" stroke-dasharray="10 6" stroke-width="1" opacity="0.25"/>`)
		buf.WriteString(`<line x1="28" y1="160" x2="292" y2="160" stroke="#818cf8" stroke-dasharray="10 6" stroke-width="1" opacity="0.25"/>`)
	}
	fontFamily := template.HTMLEscapeString(strings.ReplaceAll(font.Name, `"`, ""))
	buf.WriteString(fmt.Sprintf(`<g transform="translate(160,160) rotate(%.2f)">`, rotation))
	buf.WriteString(fmt.Sprintf(`<text text-anchor="middle" dominant-baseline="central" font-size="%.1f" letter-spacing="%.2f" font-family="%s" fill="%s">%s</text>`, fontSize, letterSpacing, fontFamily, fillColor, template.HTMLEscapeString(displayName)))
	buf.WriteString(`</g>`)
	buf.WriteString(`</svg>`)

	stats := []DesignPreviewStat{
		{Label: editorCopy(lang, "線幅", "Stroke"), Value: fmt.Sprintf("%d%%", state.StrokeWeight)},
		{Label: editorCopy(lang, "字間", "Spacing"), Value: fmt.Sprintf("%d%%", state.Tracking)},
		{Label: editorCopy(lang, "コントラスト", "Contrast"), Value: fmt.Sprintf("%d%%", state.Contrast)},
	}
	rotDegrees := (float64(state.Rotation) - 50.0) / 50.0 * 12.0
	stats = append(stats, DesignPreviewStat{
		Label: editorCopy(lang, "回転", "Rotation"),
		Value: fmt.Sprintf("%+.1f°", rotDegrees),
	})

	issues := make([]DesignPreviewIssue, 0, len(state.Validation)+2)
	for _, msg := range state.Validation {
		issues = append(issues, DesignPreviewIssue{Level: "notice", Message: msg})
	}
	if state.AIStatus != "" {
		issues = append(issues, DesignPreviewIssue{Level: "info", Message: state.AIStatus})
	}
	if state.ShowGuides {
		issues = append(issues, DesignPreviewIssue{Level: "info", Message: editorCopy(lang, "ガイドを表示してレイアウトの中心を維持しています。", "Guides enabled to keep alignment centered.")})
	}

	updated := fmt.Sprintf(editorCopy(lang, "更新: %s", "Updated %s"), time.Now().In(jstLocation).Format("15:04"))

	return DesignEditorPreview{
		SVG:          template.HTML(buf.String()),
		TemplateName: tpl.Name,
		FontName:     font.Name,
		Stats:        stats,
		Issues:       issues,
		UpdatedLabel: updated,
		Mode:         state.Mode,
	}
}
