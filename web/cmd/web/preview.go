package main

import (
	"fmt"
	"html/template"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DesignPreviewView aggregates all state required to render the design preview page and fragment.
type DesignPreviewView struct {
	DesignID            string
	Lang                string
	Title               string
	Subtitle            string
	Hint                string
	SelectedBackground  string
	SelectedDPI         int
	ActiveFrame         string
	ShowGrid            bool
	Query               string
	BackgroundOptions   []DesignPreviewBackgroundOption
	FrameOptions        []DesignPreviewFrameOption
	DPIOtions           []DesignPreviewDPIOption
	Preview             DesignPreviewImage
	Metadata            DesignPreviewMetadata
	InfoBar             []DesignPreviewInfo
	Downloads           []DesignPreviewDownload
	ShareActions        []DesignPreviewShareAction
	Tips                []DesignPreviewTip
	Snackbars           []DesignPreviewSnackbar
	Coachmark           DesignPreviewCoachmark
	LastRegeneratedCopy string
}

// DesignPreviewBackgroundOption represents a selectable preview backdrop/material option.
type DesignPreviewBackgroundOption struct {
	ID          string
	Label       string
	Description string
	Swatch      string
	Accent      string
	Active      bool
}

// DesignPreviewFrameOption toggles the mockup frame (document, envelope, desk, etc).
type DesignPreviewFrameOption struct {
	ID          string
	Label       string
	Description string
	Active      bool
}

// DesignPreviewDPIOption exposes downloadable resolution targets.
type DesignPreviewDPIOption struct {
	Value       int
	Label       string
	Description string
	Badge       string
	Active      bool
}

// DesignPreviewImage describes the mockup viewport rendering with overlays.
type DesignPreviewImage struct {
	BackgroundID     string
	BackgroundLabel  string
	BackgroundStyle  template.CSS
	CanvasTone       string
	FrameLabel       string
	FrameDescription string
	MockupNote       string
	Grid             bool
	ZoomPercent      int
	Measurements     []DesignPreviewMeasurement
	Metrics          []DesignPreviewMetric
	Badge            string
	Elevation        string
	PrimaryGlyph     string
	SecondaryGlyph   string
	FrameClass       string
}

// DesignPreviewMeasurement annotates an overlay dimension marker.
type DesignPreviewMeasurement struct {
	Label string
	Value string
	Class string
	Axis  string
}

// DesignPreviewMetric highlights numeric stats under the viewport.
type DesignPreviewMetric struct {
	Label string
	Value string
	Tone  string
}

// DesignPreviewMetadata feeds the info bar.
type DesignPreviewMetadata struct {
	Version           string
	Owner             string
	OwnerRole         string
	LastSavedExact    time.Time
	LastSavedRelative string
	LastSavedDisplay  string
	CurrentBackground string
	CurrentResolution string
	CurrentFrame      string
}

// DesignPreviewInfo is rendered in the metadata info bar.
type DesignPreviewInfo struct {
	Label string
	Value string
	Sub   string
	Icon  string
}

// DesignPreviewDownload models the download actions for PNG/SVG.
type DesignPreviewDownload struct {
	Format      string
	Label       string
	Description string
	URL         string
	Icon        string
	Secondary   string
	Primary     bool
}

// DesignPreviewShareAction drives quick share buttons.
type DesignPreviewShareAction struct {
	ID          string
	Label       string
	Description string
	Icon        string
	Href        string
	Variant     string
	Tooltip     string
}

// DesignPreviewTip represents inline guidance items.
type DesignPreviewTip struct {
	ID          string
	Title       string
	Description string
	Icon        string
}

// DesignPreviewSnackbar is displayed in the action footer host.
type DesignPreviewSnackbar struct {
	ID          string
	Message     string
	Tone        string
	ActionLabel string
	ActionHref  string
}

// DesignPreviewCoachmark explains gestures near the viewport.
type DesignPreviewCoachmark struct {
	Title   string
	Message string
	Icon    string
}

// buildDesignPreviewView assembles the design preview view model from query/form parameters.
func buildDesignPreviewView(lang string, values url.Values) DesignPreviewView {
	bg := normalizePreviewBackground(strings.TrimSpace(values.Get("bg")))
	frame := normalizePreviewFrame(strings.TrimSpace(values.Get("frame")))
	dpi := normalizePreviewDPI(values.Get("dpi"))
	showGrid := parsePreviewGrid(values.Get("grid"))

	view := DesignPreviewView{
		DesignID:           demoDesignID,
		Lang:               lang,
		Title:              editorCopy(lang, "デザインプレビュー", "Design preview"),
		Subtitle:           editorCopy(lang, "最終的な印影とモックアップを確認してエクスポートします。", "Review the final impression in context before export."),
		Hint:               editorCopy(lang, "背景や解像度を切り替えて最適な出力を選んでください。", "Swap backgrounds and resolution to find the best export target."),
		SelectedBackground: bg,
		SelectedDPI:        dpi,
		ActiveFrame:        frame,
		ShowGrid:           showGrid,
	}

	query := url.Values{}
	query.Set("bg", bg)
	query.Set("dpi", strconv.Itoa(dpi))
	if frame != "document" {
		query.Set("frame", frame)
	}
	if showGrid {
		query.Set("grid", "1")
	}
	view.Query = query.Encode()

	view.BackgroundOptions = buildPreviewBackgroundOptions(lang, bg)
	view.FrameOptions = buildPreviewFrameOptions(lang, frame)
	view.DPIOtions = buildPreviewDPIOptions(lang, dpi)
	view.Preview = buildPreviewImage(lang, bg, frame, dpi, showGrid)
	view.Metadata = buildPreviewMetadata(lang, bg, frame, dpi)
	view.InfoBar = buildPreviewInfoBar(lang, view.Metadata)
	view.Downloads = buildPreviewDownloads(lang, view.DesignID, bg, frame, dpi)
	view.ShareActions = buildPreviewShare(lang, view.DesignID, bg, frame, dpi)
	view.Tips = buildPreviewTips(lang, frame)
	view.Snackbars = buildPreviewSnackbars(lang, bg, frame, dpi, showGrid)
	view.Coachmark = buildPreviewCoachmark(lang, frame)
	view.LastRegeneratedCopy = editorCopy(lang, "最終プレビューは 45 秒前に更新されました。", "Last render refreshed 45 seconds ago.")

	return view
}

func normalizePreviewBackground(id string) string {
	switch id {
	case "washi", "wood", "transparent":
		return id
	default:
		return "washi"
	}
}

func normalizePreviewFrame(id string) string {
	switch id {
	case "document", "envelope", "desk":
		return id
	default:
		return "document"
	}
}

func normalizePreviewDPI(raw string) int {
	if raw == "" {
		return 600
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		return 600
	}
	switch val {
	case 300, 600, 1200:
		return val
	default:
		return 600
	}
}

func parsePreviewGrid(raw string) bool {
	if raw == "" {
		return false
	}
	switch strings.ToLower(raw) {
	case "1", "true", "on", "yes":
		return true
	default:
		return false
	}
}

func buildPreviewBackgroundOptions(lang, active string) []DesignPreviewBackgroundOption {
	base := []DesignPreviewBackgroundOption{
		{
			ID:          "washi",
			Label:       editorCopy(lang, "和紙", "Washi"),
			Description: editorCopy(lang, "柔らかな繊維感の背景。実印の撮影に近い質感です。", "Soft fibrous sheet, matching studio product shots."),
			Swatch:      "#f6f0e4",
			Accent:      "#f0d9b5",
		},
		{
			ID:          "wood",
			Label:       editorCopy(lang, "木製台座", "Wood block"),
			Description: editorCopy(lang, "温かみのある檜の台座。立体感を強調します。", "Warm hinoki base that emphasises depth."),
			Swatch:      "#c9986a",
			Accent:      "#915f2f",
		},
		{
			ID:          "transparent",
			Label:       editorCopy(lang, "透過", "Transparent"),
			Description: editorCopy(lang, "背景なし。透過PNGの仕上がりを確認できます。", "No backdrop—preview the transparent PNG export."),
			Swatch:      "#ced3da",
			Accent:      "#9ba3b1",
		},
	}
	for i := range base {
		base[i].Active = base[i].ID == active
	}
	return base
}

func buildPreviewFrameOptions(lang, active string) []DesignPreviewFrameOption {
	options := []DesignPreviewFrameOption{
		{
			ID:          "document",
			Label:       editorCopy(lang, "A4 用紙", "A4 document"),
			Description: editorCopy(lang, "契約書上のレイアウトを確認します。", "See placement on a contract sheet."),
		},
		{
			ID:          "envelope",
			Label:       editorCopy(lang, "長形3号封筒", "Business envelope"),
			Description: editorCopy(lang, "郵送時のバランスを確認します。", "Check balance for outbound mail."),
		},
		{
			ID:          "desk",
			Label:       editorCopy(lang, "ワークデスク", "Workspace desk"),
			Description: editorCopy(lang, "備品と並べたライフスタイル演出。", "Lifestyle shot alongside stationery."),
		},
	}
	for i := range options {
		options[i].Active = options[i].ID == active
	}
	return options
}

func buildPreviewDPIOptions(lang string, active int) []DesignPreviewDPIOption {
	options := []DesignPreviewDPIOption{
		{
			Value:       300,
			Label:       "300 DPI",
			Description: editorCopy(lang, "社内資料向けの軽量サイズ。", "Lightweight for internal docs."),
			Badge:       editorCopy(lang, "速い", "Fastest"),
		},
		{
			Value:       600,
			Label:       "600 DPI",
			Description: editorCopy(lang, "正式な提出や印刷に推奨。", "Recommended for submission & print."),
			Badge:       editorCopy(lang, "推奨", "Recommended"),
		},
		{
			Value:       1200,
			Label:       "1200 DPI",
			Description: editorCopy(lang, "彫刻データや大型プリント向け。", "For engraving and large-format use."),
			Badge:       editorCopy(lang, "最高品質", "Hi-fidelity"),
		},
	}
	for i := range options {
		options[i].Active = options[i].Value == active
	}
	return options
}

func buildPreviewImage(lang, bg, frame string, dpi int, showGrid bool) DesignPreviewImage {
	label := map[string]string{
		"washi":       editorCopy(lang, "和紙背景", "Washi texture"),
		"wood":        editorCopy(lang, "木製背景", "Wood base"),
		"transparent": editorCopy(lang, "透過背景", "Transparent backplate"),
	}[bg]
	if label == "" {
		label = editorCopy(lang, "背景", "Background")
	}

	style := previewBackgroundStyle(bg)
	frameLabel := map[string]string{
		"document": editorCopy(lang, "契約書", "Contract sheet"),
		"envelope": editorCopy(lang, "封筒レイアウト", "Envelope layout"),
		"desk":     editorCopy(lang, "デスク演出", "Desk scene"),
	}[frame]
	frameDesc := map[string]string{
		"document": editorCopy(lang, "印影を右下余白で表示", "Positioned along the lower margin."),
		"envelope": editorCopy(lang, "宛名と証印の間隔 18mm", "18 mm spacing to recipient block."),
		"desk":     editorCopy(lang, "備品と柔らかい光で演出", "Styled with soft daylight and props."),
	}[frame]
	mockupNote := editorCopy(lang, "ズームとドラッグで細部を確認できます。", "Zoom and drag to inspect details.")
	if frame == "envelope" {
		mockupNote = editorCopy(lang, "封筒の差し込み角度に合わせた傾斜プレビュー。", "Preview tilted to match envelope insertion.")
	}

	measurements := []DesignPreviewMeasurement{
		{
			Label: editorCopy(lang, "直径", "Diameter"),
			Value: measurementFor("round", "medium"),
			Class: "top-4 left-1/2 -translate-x-1/2",
			Axis:  "horizontal",
		},
		{
			Label: editorCopy(lang, "外枠幅", "Border width"),
			Value: editorCopy(lang, "0.8 mm", "0.8 mm"),
			Class: "right-4 top-1/2 -translate-y-1/2",
			Axis:  "vertical",
		},
		{
			Label: editorCopy(lang, "余白", "Padding"),
			Value: editorCopy(lang, "2.4 mm", "2.4 mm"),
			Class: "bottom-5 left-1/2 -translate-x-1/2",
			Axis:  "horizontal",
		},
	}

	stats := []DesignPreviewMetric{
		{
			Label: editorCopy(lang, "実寸比", "Scale"),
			Value: editorCopy(lang, "100% (実寸)", "100% actual size"),
			Tone:  "muted",
		},
		{
			Label: editorCopy(lang, "推奨出力", "Suggested output"),
			Value: fmt.Sprintf("%d dpi · %s", dpi, editorCopy(lang, "PNG", "PNG")),
			Tone:  "info",
		},
		{
			Label: editorCopy(lang, "シャープネス", "Sharpness"),
			Value: editorCopy(lang, "自動補正済み", "Auto enhanced"),
			Tone:  "success",
		},
	}

	return DesignPreviewImage{
		BackgroundID:     bg,
		BackgroundLabel:  label,
		BackgroundStyle:  template.CSS(style),
		CanvasTone:       previewCanvasTone(bg),
		FrameLabel:       frameLabel,
		FrameDescription: frameDesc,
		MockupNote:       mockupNote,
		Grid:             showGrid,
		ZoomPercent:      previewZoomForFrame(frame),
		Measurements:     measurements,
		Metrics:          stats,
		Badge:            fmt.Sprintf("%d DPI", dpi),
		Elevation:        previewElevation(frame),
		PrimaryGlyph:     editorCopy(lang, "藤原", "FUJIWARA"),
		SecondaryGlyph:   editorCopy(lang, "印章局", "Hanko Bureau"),
		FrameClass:       previewFrameClass(frame),
	}
}

func previewBackgroundStyle(bg string) string {
	switch bg {
	case "wood":
		return "background: linear-gradient(135deg, #d9aa72 0%, #c48a4a 45%, #b97736 100%);"
	case "transparent":
		return "background-image: linear-gradient(45deg, #eef1f6 25%, transparent 25%), linear-gradient(-45deg, #eef1f6 25%, transparent 25%), linear-gradient(45deg, transparent 75%, #eef1f6 75%), linear-gradient(-45deg, transparent 75%, #eef1f6 75%);background-size: 24px 24px;background-position: 0 0, 0 12px, 12px -12px, -12px 0;"
	default: // washi
		return "background: radial-gradient(circle at 25% 25%, #fff5da, #f8e7c1 45%, #f3ddae 100%);"
	}
}

func previewCanvasTone(bg string) string {
	switch bg {
	case "wood":
		return "#faf6f0"
	case "transparent":
		return "#ffffff"
	default:
		return "#ffffff"
	}
}

func previewFrameClass(frame string) string {
	switch frame {
	case "envelope":
		return "rotate-[-4deg]"
	case "desk":
		return "rotate-[2deg]"
	default:
		return ""
	}
}

func previewElevation(frame string) string {
	switch frame {
	case "desk":
		return "shadow-xl shadow-slate-900/30"
	case "envelope":
		return "shadow-lg shadow-slate-900/20"
	default:
		return "shadow-lg shadow-slate-900/15"
	}
}

func previewZoomForFrame(frame string) int {
	switch frame {
	case "desk":
		return 88
	case "envelope":
		return 94
	default:
		return 100
	}
}

func buildPreviewMetadata(lang, bg, frame string, dpi int) DesignPreviewMetadata {
	now := time.Now().In(jstLocation)
	lastSaved := now.Add(-5 * time.Minute)
	lastDisplay := lastSaved.Format("2006-01-02 15:04")
	relative := editorCopy(lang, "5分前に保存", "Saved 5 minutes ago")
	backgroundLabel := map[string]string{
		"washi":       editorCopy(lang, "和紙の背景", "Washi backplate"),
		"wood":        editorCopy(lang, "木製台座", "Wood block"),
		"transparent": editorCopy(lang, "透過背景", "Transparent"),
	}[bg]
	frameLabel := map[string]string{
		"document": editorCopy(lang, "契約書表示", "Contract sheet"),
		"envelope": editorCopy(lang, "封筒表示", "Envelope shot"),
		"desk":     editorCopy(lang, "デスク演出", "Desktop scene"),
	}[frame]
	return DesignPreviewMetadata{
		Version:           "v1.4.2",
		Owner:             editorCopy(lang, "宮崎 愛子", "Aiko Miyazaki"),
		OwnerRole:         editorCopy(lang, "ブランドマネージャー", "Brand manager"),
		LastSavedExact:    lastSaved,
		LastSavedRelative: relative,
		LastSavedDisplay:  lastDisplay + " JST",
		CurrentBackground: backgroundLabel,
		CurrentResolution: fmt.Sprintf("%d DPI", dpi),
		CurrentFrame:      frameLabel,
	}
}

func buildPreviewInfoBar(lang string, meta DesignPreviewMetadata) []DesignPreviewInfo {
	return []DesignPreviewInfo{
		{
			Label: editorCopy(lang, "バージョン", "Version"),
			Value: meta.Version,
			Sub:   editorCopy(lang, "ワークスペース共有", "Shared workspace"),
			Icon:  "tag",
		},
		{
			Label: editorCopy(lang, "最終更新", "Last saved"),
			Value: meta.LastSavedRelative,
			Sub:   meta.LastSavedDisplay,
			Icon:  "clock",
		},
		{
			Label: editorCopy(lang, "担当者", "Owner"),
			Value: meta.Owner,
			Sub:   meta.OwnerRole,
			Icon:  "user-circle",
		},
	}
}

func buildPreviewDownloads(lang, designID, bg, frame string, dpi int) []DesignPreviewDownload {
	base := fmt.Sprintf("https://cdn.hanko-field.app/designs/%s/%s/%s", designID, bg, frame)
	token := previewSignedToken(dpi)
	return []DesignPreviewDownload{
		{
			Format:      "png",
			Label:       editorCopy(lang, "PNG をダウンロード", "Download PNG"),
			Description: editorCopy(lang, "透過背景・ICCプロファイル埋め込み済み。", "Transparent background with embedded ICC profile."),
			URL:         fmt.Sprintf("%s-%ddpi.png?%s", base, dpi, token),
			Icon:        "arrow-down-tray",
			Secondary:   editorCopy(lang, "約 2.3 MB", "~2.3 MB"),
			Primary:     true,
		},
		{
			Format:      "svg",
			Label:       editorCopy(lang, "SVG をダウンロード", "Download SVG"),
			Description: editorCopy(lang, "ベクターデータ。レーザー彫刻・大判出力向け。", "Vector export for engraving or large-format print."),
			URL:         fmt.Sprintf("%s-master.svg?%s", base, token),
			Icon:        "sparkles",
			Secondary:   editorCopy(lang, "ライブパス保持", "Maintains live paths"),
			Primary:     false,
		},
	}
}

func previewSignedToken(dpi int) string {
	exp := time.Now().Add(15 * time.Minute).Unix()
	hash := 5381
	hash = ((hash << 5) + hash) + dpi
	return fmt.Sprintf("exp=%d&sig=%x", exp, hash&math.MaxInt32)
}

func buildPreviewShare(lang, designID, bg, frame string, dpi int) []DesignPreviewShareAction {
	query := url.Values{}
	query.Set("bg", bg)
	query.Set("frame", frame)
	query.Set("dpi", strconv.Itoa(dpi))
	shareURL := fmt.Sprintf("https://app.hanko-field.jp/design/%s/preview?%s", designID, query.Encode())
	return []DesignPreviewShareAction{
		{
			ID:          "copy-link",
			Label:       editorCopy(lang, "リンクをコピー", "Copy link"),
			Description: editorCopy(lang, "クリップボードに共有リンクをコピーします。", "Copies the share link to clipboard."),
			Icon:        "link",
			Href:        shareURL,
			Variant:     "secondary",
			Tooltip:     editorCopy(lang, "キーボードショートカット: L", "Keyboard shortcut: L"),
		},
		{
			ID:          "send-workspace",
			Label:       editorCopy(lang, "ワークスペースに送る", "Send to workspace"),
			Description: editorCopy(lang, "チームボードにプレビューを固定します。", "Pin the preview to your workspace board."),
			Icon:        "paper-airplane",
			Href:        shareURL + "&channel=workspace",
			Variant:     "primary",
			Tooltip:     editorCopy(lang, "承認者へ通知されます", "Notifies approvers"),
		},
	}
}

func buildPreviewTips(lang, frame string) []DesignPreviewTip {
	tips := []DesignPreviewTip{
		{
			ID:          "gesture",
			Title:       editorCopy(lang, "ピンチでズーム", "Pinch to zoom"),
			Description: editorCopy(lang, "タッチパッドやホイールでも細部を確認できます。", "Use touchpad or wheel to inspect the seal texture."),
			Icon:        "magnifying-glass",
		},
		{
			ID:          "frame",
			Title:       editorCopy(lang, "フレームを切り替え", "Swap mockup frame"),
			Description: editorCopy(lang, "用途にあわせて用紙・封筒・机のシーンを比較します。", "Compare contract, envelope, and desk scenarios."),
			Icon:        "rectangle-group",
		},
	}
	if frame == "desk" {
		tips = append(tips, DesignPreviewTip{
			ID:          "lighting",
			Title:       editorCopy(lang, "ライティング", "Lighting"),
			Description: editorCopy(lang, "逆光で撮影すると凹凸が際立ちます。", "Backlighting accentuates carved depth."),
			Icon:        "sun",
		})
	}
	return tips
}

func buildPreviewSnackbars(lang, bg, frame string, dpi int, showGrid bool) []DesignPreviewSnackbar {
	message := editorCopy(lang, "プレビューは保存済みのデザインを使用しています。", "Preview uses your saved design state.")
	snack := DesignPreviewSnackbar{
		ID:      "preview-sync",
		Message: message,
		Tone:    "info",
	}
	out := []DesignPreviewSnackbar{snack}
	if showGrid {
		vals := url.Values{}
		vals.Set("bg", bg)
		vals.Set("dpi", strconv.Itoa(dpi))
		if frame != "document" {
			vals.Set("frame", frame)
		}
		vals.Set("grid", "0")
		href := "/design/preview"
		if encoded := vals.Encode(); encoded != "" {
			href = href + "?" + encoded
		}
		out = append(out, DesignPreviewSnackbar{
			ID:          "grid-enabled",
			Message:     editorCopy(lang, "製図ガイドがオンになっています。", "Measurement grid enabled."),
			Tone:        "success",
			ActionLabel: editorCopy(lang, "オフにする", "Disable"),
			ActionHref:  href,
		})
	}
	return out
}

func buildPreviewCoachmark(lang, frame string) DesignPreviewCoachmark {
	note := editorCopy(lang, "スペースキーを押しながらドラッグするとパンできます。", "Hold space to pan the mockup.")
	if frame == "envelope" {
		note = editorCopy(lang, "封筒プレビューでは左右の余白が強調表示されます。", "Envelope mode highlights side margins.")
	}
	return DesignPreviewCoachmark{
		Title:   editorCopy(lang, "操作ヒント", "Gesture hint"),
		Message: note,
		Icon:    "hand-raised",
	}
}
