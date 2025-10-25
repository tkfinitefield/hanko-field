package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	defaultDesignShareFormat = "png"
	defaultDesignShareSize   = "original"
	maxDesignShareExpiryDays = 7
)

// DesignShareView drives the design share modal content.
type DesignShareView struct {
	Lang       string
	DesignID   string
	DesignName string
	Owner      string
	LastSaved  string
	Hint       string

	FormatOptions []DesignShareOption
	SizeOptions   []DesignShareOption

	SelectedFormat string
	SelectedSize   string
	Watermark      bool

	WatermarkLabel       string
	WatermarkDescription string

	ExpiryValue string
	ExpiryMin   string
	ExpiryMax   string
	ExpiryHelp  string

	Preview        DesignSharePreview
	Link           DesignShareLink
	HasLink        bool
	Alerts         []DesignShareAlert
	AnalyticsEvent string
	Summary        string
}

// DesignShareOption renders a select/radio entry.
type DesignShareOption struct {
	ID          string
	Label       string
	Description string
	Secondary   string
	Active      bool
}

// DesignSharePreview powers the left-side preview card.
type DesignSharePreview struct {
	Title          string
	Subtitle       string
	Badge          string
	FormatLabel    string
	SizeLabel      string
	Watermark      bool
	WatermarkLabel string
}

// DesignShareLink contains the generated signed URL details.
type DesignShareLink struct {
	Available       bool
	ShareURL        string
	DownloadURL     string
	Format          string
	FormatLabel     string
	Size            string
	SizeLabel       string
	Watermark       bool
	AssetID         string
	Method          string
	Headers         map[string]string
	FileSize        string
	ExpiresAt       time.Time
	ExpiresDisplay  string
	ExpiresRelative string
	EmbedCode       string
	GeneratedAt     time.Time
	GeneratedCopy   string
	ShortCode       string
}

// DesignShareAlert surfaces inline notices under the header.
type DesignShareAlert struct {
	Tone        string
	Title       string
	Description string
	Icon        string
}

type designShareForm struct {
	DesignID  string
	Format    string
	Size      string
	Watermark bool
	Expiry    time.Time
}

type shareOptionSeed struct {
	ID            string
	LabelJA       string
	LabelEN       string
	DescriptionJA string
	DescriptionEN string
	NotesJA       string
	NotesEN       string
}

var (
	shareFormatCatalog = []shareOptionSeed{
		{
			ID:            "png",
			LabelJA:       "PNG（透過）",
			LabelEN:       "PNG (transparent)",
			DescriptionJA: "ICC プロファイル付き / 背景透過",
			DescriptionEN: "Embedded ICC profile / transparent",
			NotesJA:       "ICC プロファイル埋め込み / 背景透過",
			NotesEN:       "Embedded ICC profile / alpha channel",
		},
		{
			ID:            "svg",
			LabelJA:       "SVG（ベクター）",
			LabelEN:       "SVG (vector)",
			DescriptionJA: "編集可能なパスを保持 / 彫刻向け",
			DescriptionEN: "Keeps editable paths for engraving",
			NotesJA:       "パス保持 / 彫刻・大判印刷",
			NotesEN:       "Live paths for engraving / large format",
		},
		{
			ID:            "pdf",
			LabelJA:       "PDF（校正用）",
			LabelEN:       "PDF (proof)",
			DescriptionJA: "校正コメント欄付きシート",
			DescriptionEN: "Proof sheet with comment gutter",
			NotesJA:       "校正コメント欄付き / ドキュメント共有",
			NotesEN:       "Proof sheet with comment gutter",
		},
	}

	shareSizeCatalog = []shareOptionSeed{
		{
			ID:            "original",
			LabelJA:       "原寸（1200px）",
			LabelEN:       "Original (1200px)",
			DescriptionJA: "最大解像度 / 印刷登録用",
			DescriptionEN: "Maximum resolution for print",
			NotesJA:       "フル解像度 / 提出用",
			NotesEN:       "Full resolution / approvals",
		},
		{
			ID:            "medium",
			LabelJA:       "ミドル（800px）",
			LabelEN:       "Medium (800px)",
			DescriptionJA: "スライド・メール共有向け",
			DescriptionEN: "Ideal for slides and emails",
			NotesJA:       "メール添付 / スライド",
			NotesEN:       "Email + decks",
		},
		{
			ID:            "thumbnail",
			LabelJA:       "サムネ（320px）",
			LabelEN:       "Thumbnail (320px)",
			DescriptionJA: "チャット・リスト向け軽量版",
			DescriptionEN: "Lightweight preview for chat",
			NotesJA:       "チャット / 一覧",
			NotesEN:       "Chat previews",
		},
	}
)

var (
	designIDPattern = regexp.MustCompile(`^df-[a-z0-9-]{4,}$`)
	cdnBase         = strings.TrimSuffix(strings.TrimSpace(os.Getenv("HANKO_WEB_CDN_BASE")), "/")
	shareBase       = strings.TrimSuffix(strings.TrimSpace(os.Getenv("HANKO_WEB_SHARE_BASE")), "/")
)

func defaultDesignShareForm(now time.Time) designShareForm {
	now = toShareZone(now)
	defExpiry := now.Add(72 * time.Hour)
	defExpiry = normalizeShareDate(defExpiry)
	return designShareForm{
		DesignID:  demoDesignID,
		Format:    defaultDesignShareFormat,
		Size:      defaultDesignShareSize,
		Watermark: true,
		Expiry:    defExpiry,
	}
}

func parseDesignShareForm(lang string, form url.Values, now time.Time) (designShareForm, []DesignShareAlert) {
	state := defaultDesignShareForm(now)
	var alerts []DesignShareAlert

	if id := strings.TrimSpace(form.Get("design_id")); id != "" {
		if normalized, ok := normalizeDesignID(id); ok {
			state.DesignID = normalized
		} else {
			alerts = append(alerts, DesignShareAlert{
				Tone:        "danger",
				Title:       editorCopy(lang, "デザイン ID が無効です。", "Design ID is invalid."),
				Description: editorCopy(lang, "もう一度共有リンクを開き直してください。", "Reload the share dialog and try again."),
				Icon:        "exclamation-triangle",
			})
		}
	}

	format := strings.TrimSpace(strings.ToLower(form.Get("format")))
	if format == "" {
		format = state.Format
	}
	if !shareFormatAllowed(format) {
		alerts = append(alerts, DesignShareAlert{
			Tone:        "danger",
			Title:       editorCopy(lang, "形式が無効です。", "Selected format is not available."),
			Description: editorCopy(lang, "PNG / SVG / PDF から選択してください。", "Choose between PNG, SVG, or PDF exports."),
			Icon:        "exclamation-triangle",
		})
	} else {
		state.Format = format
	}

	size := strings.TrimSpace(strings.ToLower(form.Get("size")))
	if size == "" {
		size = state.Size
	}
	if !shareSizeAllowed(size) {
		alerts = append(alerts, DesignShareAlert{
			Tone:        "danger",
			Title:       editorCopy(lang, "サイズを選択してください。", "Select a download size."),
			Description: editorCopy(lang, "原寸 / ミドル / サムネイル から選べます。", "Pick from original, medium, or thumbnail renderings."),
			Icon:        "exclamation-triangle",
		})
	} else {
		state.Size = size
	}

	state.Watermark = form.Get("watermark") != ""

	if raw := strings.TrimSpace(form.Get("expiry")); raw != "" {
		if parsed, err := time.ParseInLocation("2006-01-02", raw, shareLocation()); err == nil {
			parsed = normalizeShareDate(parsed)
			min, max := designShareExpiryBounds(now)
			if parsed.Before(min) || parsed.After(max) {
				alerts = append(alerts, DesignShareAlert{
					Tone:        "danger",
					Title:       editorCopy(lang, "期限が範囲外です。", "Expiry is outside the allowed window."),
					Description: editorCopy(lang, "本日から7日以内の日付を選択してください。", "Pick a date within the next 7 days."),
					Icon:        "clock",
				})
			} else {
				state.Expiry = parsed
			}
		} else {
			alerts = append(alerts, DesignShareAlert{
				Tone:        "danger",
				Title:       editorCopy(lang, "期限を解析できませんでした。", "Could not read the expiry date."),
				Description: editorCopy(lang, "YYYY-MM-DD 形式で入力してください。", "Use YYYY-MM-DD format."),
				Icon:        "clock",
			})
		}
	}

	return state, alerts
}

func buildDesignShareView(lang string, form designShareForm, link *DesignShareLink, alerts []DesignShareAlert, now time.Time) DesignShareView {
	now = toShareZone(now)
	min, max := designShareExpiryBounds(now)

	view := DesignShareView{
		Lang:           lang,
		DesignID:       form.DesignID,
		DesignName:     editorCopy(lang, "国際配送向けバイリンガル印影", "Bilingual customs seal"),
		Owner:          editorCopy(lang, "宮崎 愛子", "Aiko Miyazaki"),
		LastSaved:      editorCopy(lang, "最終保存: 2 分前", "Last saved 2 minutes ago"),
		Hint:           editorCopy(lang, "共有リンクは選択した期限で自動無効化されます。", "Shared links expire automatically on the selected date."),
		FormatOptions:  shareFormatOptions(lang, form.Format),
		SizeOptions:    shareSizeOptions(lang, form.Size),
		SelectedFormat: form.Format,
		SelectedSize:   form.Size,
		Watermark:      form.Watermark,
		WatermarkLabel: editorCopy(lang, "透かしを入れる", "Enable watermark"),
		WatermarkDescription: editorCopy(
			lang,
			"透かしは未承認のプレビューが無断利用されないよう保護します。",
			"Watermark protects unapproved previews from being reused elsewhere.",
		),
		ExpiryValue: form.Expiry.Format("2006-01-02"),
		ExpiryMin:   min.Format("2006-01-02"),
		ExpiryMax:   max.Format("2006-01-02"),
		ExpiryHelp: editorCopy(
			lang,
			"期限(日本時間)を過ぎると共有リンクはアクセス不可になります。",
			"Links become inaccessible immediately after the selected JST date.",
		),
		Preview:        buildDesignSharePreview(lang, form),
		AnalyticsEvent: "design_share",
		Summary:        fmt.Sprintf("%s · %s · %s", shareFormatLabel(lang, form.Format), shareSizeLabel(lang, form.Size), shareWatermarkCopy(lang, form.Watermark)),
	}

	view.Alerts = append(view.Alerts, alerts...)
	if !form.Watermark {
		view.Alerts = append(view.Alerts, DesignShareAlert{
			Tone:        "warning",
			Title:       editorCopy(lang, "透かし無効で共有します。", "Watermark disabled for this share."),
			Description: editorCopy(lang, "承認前の共有では透かしを推奨しています。", "Consider keeping the watermark enabled until the mark is approved."),
			Icon:        "information-circle",
		})
	}

	if link != nil && link.Available {
		view.Link = *link
		view.HasLink = true
		view.Alerts = append(view.Alerts, designShareLinkAlerts(lang, *link, now)...)
	}

	return view
}

func buildDesignSharePreview(lang string, form designShareForm) DesignSharePreview {
	return DesignSharePreview{
		Title:          editorCopy(lang, "共有プレビュー", "Share preview"),
		Subtitle:       editorCopy(lang, "背景と透かしの状態を確認できます。", "Double-check the exported tone and watermark."),
		Badge:          strings.ToUpper(form.Format) + " · " + shareSizeBadge(form.Size),
		FormatLabel:    shareFormatLabel(lang, form.Format),
		SizeLabel:      shareSizeLabel(lang, form.Size),
		Watermark:      form.Watermark,
		WatermarkLabel: shareWatermarkCopy(lang, form.Watermark),
	}
}

func shareFormatOptions(lang, selected string) []DesignShareOption {
	opts := make([]DesignShareOption, len(shareFormatCatalog))
	for i, seed := range shareFormatCatalog {
		opts[i] = DesignShareOption{
			ID:          seed.ID,
			Label:       editorCopy(lang, seed.LabelJA, seed.LabelEN),
			Description: editorCopy(lang, seed.DescriptionJA, seed.DescriptionEN),
			Secondary:   editorCopy(lang, seed.NotesJA, seed.NotesEN),
			Active:      seed.ID == selected,
		}
	}
	return opts
}

func shareSizeOptions(lang, selected string) []DesignShareOption {
	opts := make([]DesignShareOption, len(shareSizeCatalog))
	for i, seed := range shareSizeCatalog {
		opts[i] = DesignShareOption{
			ID:          seed.ID,
			Label:       editorCopy(lang, seed.LabelJA, seed.LabelEN),
			Description: editorCopy(lang, seed.DescriptionJA, seed.DescriptionEN),
			Secondary:   editorCopy(lang, seed.NotesJA, seed.NotesEN),
			Active:      seed.ID == selected,
		}
	}
	return opts
}

func issueDesignShareLink(ctx context.Context, lang, designID string, form designShareForm, now time.Time) (DesignShareLink, error) {
	select {
	case <-ctx.Done():
		return DesignShareLink{}, ctx.Err()
	default:
	}
	now = toShareZone(now)
	expires := shareExpiryCutoff(form.Expiry)
	token, err := randomShareToken(20)
	if err != nil {
		return DesignShareLink{}, fmt.Errorf("issue share link token: %w", err)
	}
	short, err := randomShareToken(6)
	if err != nil {
		return DesignShareLink{}, fmt.Errorf("issue share shortcode: %w", err)
	}
	path := fmt.Sprintf("designs/%s/export/%s/%s.%s", designID, form.Size, watermarkSlug(form.Watermark), form.Format)
	downloadURL := fmt.Sprintf("%s/%s?token=%s", designShareCDNBase(), path, token)
	shareURL := fmt.Sprintf("%s/d/%s?design=%s", designSharePortalBase(), short, designID)
	fileSize := shareApproxFileSize(form.Format, form.Size)
	embed := fmt.Sprintf(`<iframe src="%s" title="Hanko Field design preview" loading="lazy" style="width:100%%;max-width:560px;height:420px;border:0;border-radius:20px;"></iframe>`, shareURL)

	link := DesignShareLink{
		Available:       true,
		ShareURL:        shareURL,
		DownloadURL:     downloadURL,
		Format:          form.Format,
		FormatLabel:     shareFormatLabel(lang, form.Format),
		Size:            form.Size,
		SizeLabel:       shareSizeLabel(lang, form.Size),
		Watermark:       form.Watermark,
		AssetID:         fmt.Sprintf("%s-%s-%s", designID, form.Size, strings.ToUpper(form.Format)),
		Method:          http.MethodGet,
		Headers:         map[string]string{"X-Hanko-Signed": "true"},
		FileSize:        fileSize,
		ExpiresAt:       expires,
		ExpiresDisplay:  formatShareDate(expires, lang),
		ExpiresRelative: relativeShareExpiry(expires, now, lang),
		EmbedCode:       embed,
		GeneratedAt:     now,
		GeneratedCopy:   editorCopy(lang, "共有リンクを発行しました。", "Share link is ready."),
		ShortCode:       short,
	}
	return link, nil
}

func shareFormatAllowed(id string) bool {
	for _, seed := range shareFormatCatalog {
		if seed.ID == id {
			return true
		}
	}
	return false
}

func shareSizeAllowed(id string) bool {
	for _, seed := range shareSizeCatalog {
		if seed.ID == id {
			return true
		}
	}
	return false
}

func shareFormatLabel(lang, id string) string {
	for _, seed := range shareFormatCatalog {
		if seed.ID == id {
			return editorCopy(lang, seed.LabelJA, seed.LabelEN)
		}
	}
	return strings.ToUpper(id)
}

func shareSizeLabel(lang, id string) string {
	for _, seed := range shareSizeCatalog {
		if seed.ID == id {
			return editorCopy(lang, seed.LabelJA, seed.LabelEN)
		}
	}
	return strings.ToUpper(id)
}

func shareSizeBadge(id string) string {
	switch id {
	case "original":
		return "ORG"
	case "medium":
		return "MID"
	case "thumbnail":
		return "THM"
	default:
		if len(id) >= 3 {
			return strings.ToUpper(id[:3])
		}
		return strings.ToUpper(id)
	}
}

func shareWatermarkCopy(lang string, enabled bool) string {
	if enabled {
		return editorCopy(lang, "透かしあり", "Watermark on")
	}
	return editorCopy(lang, "透かしなし", "Watermark off")
}

func shareApproxFileSize(format, size string) string {
	base := map[string]float64{
		"png": 2.3,
		"svg": 0.4,
		"pdf": 1.8,
	}
	mult := map[string]float64{
		"original":  1.0,
		"medium":    0.7,
		"thumbnail": 0.35,
	}
	b := base[strings.ToLower(format)]
	if b == 0 {
		b = 1.2
	}
	m := mult[strings.ToLower(size)]
	if m == 0 {
		m = 1.0
	}
	return fmt.Sprintf("~%.1f MB", b*m)
}

func shareExpiryCutoff(date time.Time) time.Time {
	date = toShareZone(date)
	y, m, d := date.Date()
	return time.Date(y, m, d, 23, 0, 0, 0, shareLocation())
}

func formatShareDate(ts time.Time, lang string) string {
	ts = toShareZone(ts)
	if lang == "ja" {
		return ts.Format("2006年01月02日 15:04 JST")
	}
	return ts.Format("02 Jan 2006 15:04 JST")
}

func relativeShareExpiry(target, now time.Time, lang string) string {
	if target.IsZero() {
		return ""
	}
	diff := target.Sub(now)
	if diff <= 0 {
		return editorCopy(lang, "期限切れ", "Expired")
	}
	days := int(diff.Hours()) / 24
	hours := int(math.Mod(diff.Hours(), 24))
	minutes := int(math.Mod(diff.Minutes(), 60))
	switch {
	case days > 0 && hours > 0:
		if lang == "ja" {
			return fmt.Sprintf("%d日 %d時間後に失効", days, hours)
		}
		return fmt.Sprintf("Expires in %dd %dh", days, hours)
	case days > 0:
		if lang == "ja" {
			return fmt.Sprintf("%d日後に失効", days)
		}
		return fmt.Sprintf("Expires in %d days", days)
	case hours > 0:
		if lang == "ja" {
			return fmt.Sprintf("%d時間後に失効", hours)
		}
		return fmt.Sprintf("Expires in %d hours", hours)
	default:
		if minutes <= 0 {
			minutes = 1
		}
		if lang == "ja" {
			return fmt.Sprintf("%d分後に失効", minutes)
		}
		return fmt.Sprintf("Expires in %d min", minutes)
	}
}

func designShareExpiryBounds(now time.Time) (time.Time, time.Time) {
	now = toShareZone(now)
	min := normalizeShareDate(now)
	max := normalizeShareDate(now.Add(maxDesignShareExpiryDays * 24 * time.Hour))
	return min, max
}

func normalizeShareDate(ts time.Time) time.Time {
	ts = toShareZone(ts)
	y, m, d := ts.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, shareLocation())
}

func toShareZone(ts time.Time) time.Time {
	loc := shareLocation()
	if loc == nil {
		return ts
	}
	return ts.In(loc)
}

func shareLocation() *time.Location {
	if jstLocation == nil {
		return time.FixedZone("JST", 9*60*60)
	}
	return jstLocation
}

func watermarkSlug(enabled bool) string {
	if enabled {
		return "wm"
	}
	return "clean"
}

func designShareLinkAlerts(lang string, link DesignShareLink, now time.Time) []DesignShareAlert {
	if link.ExpiresAt.IsZero() {
		return nil
	}
	var alerts []DesignShareAlert
	diff := link.ExpiresAt.Sub(now)
	if diff <= 0 {
		alerts = append(alerts, DesignShareAlert{
			Tone:        "danger",
			Title:       editorCopy(lang, "共有リンクは期限切れです。", "Share link has expired."),
			Description: editorCopy(lang, "再発行してから共有を更新してください。", "Generate a new link before sharing again."),
			Icon:        "x-circle",
		})
	} else if diff < 24*time.Hour {
		alerts = append(alerts, DesignShareAlert{
			Tone:        "warning",
			Title:       editorCopy(lang, "まもなく期限を迎えます。", "Link will expire soon."),
			Description: editorCopy(lang, "24 時間以内に失効します。必要であれば延長してください。", "Expires within 24 hours. Extend the expiry if approval is pending."),
			Icon:        "clock",
		})
	}
	return alerts
}

func randomShareToken(n int) (string, error) {
	if n <= 0 {
		n = 8
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("random: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func designShareCDNBase() string {
	if cdnBase != "" {
		return cdnBase
	}
	return "https://cdn.hanko-field.app"
}

func designSharePortalBase() string {
	if shareBase != "" {
		return shareBase
	}
	return "https://share.hanko-field.jp"
}

func normalizeDesignID(id string) (string, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", false
	}
	if designIDPattern.MatchString(id) {
		return id, true
	}
	return "", false
}
