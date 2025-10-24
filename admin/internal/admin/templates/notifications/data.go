package notifications

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	adminnotifications "finitefield.org/hanko-admin/internal/admin/notifications"
	"finitefield.org/hanko-admin/internal/admin/templates/helpers"
	"finitefield.org/hanko-admin/internal/admin/templates/partials"
	"github.com/a-h/templ"
)

// PageData contains the full SSR payload for the notifications page.
type PageData struct {
	Title         string
	Description   string
	Breadcrumbs   []partials.Breadcrumb
	TableEndpoint string
	Table         TableData
	Drawer        DrawerData
	Query         QueryState
	Filters       Filters
	Legend        []LegendItem
}

// QueryState represents the submitted filter params.
type QueryState struct {
	Category  string
	Severity  string
	Status    string
	StartDate string
	EndDate   string
	Search    string
	Owner     string
}

// Filters groups the available filtering controls.
type Filters struct {
	Categories []CategoryOption
	Severities []SeverityOption
	Statuses   []SelectOption
}

// CategoryOption is rendered in the segmented control.
type CategoryOption struct {
	Value  string
	Label  string
	Icon   string
	Count  int
	Active bool
}

// SeverityOption is rendered as a chip-like toggle.
type SeverityOption struct {
	Value  string
	Label  string
	Tone   string
	Count  int
	Active bool
}

// SelectOption renders <option> values.
type SelectOption struct {
	Value string
	Label string
}

// TableData represents the table fragment payload.
type TableData struct {
	Items          []TableRow
	Error          string
	EmptyMessage   string
	Total          int
	NextCursor     string
	CategoryCounts map[string]int
	SeverityCounts map[string]int
	StatusCounts   map[string]int
	SelectedID     string
}

// TableRow is a display-friendly row model.
type TableRow struct {
	ID                string
	CategoryLabel     string
	CategoryTone      string
	CategoryIcon      string
	SeverityLabel     string
	SeverityTone      string
	Title             string
	Summary           string
	StatusLabel       string
	StatusTone        string
	ResourceLabel     string
	ResourceURL       string
	ResourceKind      string
	Owner             string
	CreatedAt         time.Time
	CreatedAtRelative string
	CreatedAtTooltip  string
	Actions           []RowAction
	Attributes        templ.Attributes
}

// RowAction represents a contextual menu entry.
type RowAction struct {
	Label string
	URL   string
	Icon  string
}

// DrawerData powers the detail drawer on the right-hand side.
type DrawerData struct {
	Empty             bool
	ID                string
	Title             string
	Summary           string
	CategoryLabel     string
	CategoryTone      string
	SeverityLabel     string
	SeverityTone      string
	StatusLabel       string
	StatusTone        string
	Owner             string
	Resource          ResourceView
	CreatedAt         time.Time
	CreatedRelative   string
	AcknowledgedAt    *time.Time
	AcknowledgedLabel string
	ResolvedAt        *time.Time
	ResolvedLabel     string
	Metadata          []MetadataView
	Timeline          []TimelineEventView
	Links             []RowAction
}

// ResourceView captures the impacted entity.
type ResourceView struct {
	Label string
	URL   string
	Kind  string
}

// MetadataView renders supporting information.
type MetadataView struct {
	Label string
	Value string
	Icon  string
}

// TimelineEventView renders a historical entry.
type TimelineEventView struct {
	Title            string
	Description      string
	OccurredAt       time.Time
	OccurredRelative string
	Actor            string
	Tone             string
	Icon             string
}

// LegendItem summarises severity meanings.
type LegendItem struct {
	Label       string
	Tone        string
	Description string
	Icon        string
}

// BadgeData represents the payload for the notification badge fragment.
type BadgeData struct {
	Total          int
	Critical       int
	Warning        int
	Endpoint       string
	StreamEndpoint string
	Href           string
}

// BuildPageData assembles the SSR payload.
func BuildPageData(basePath string, state QueryState, table TableData, drawer DrawerData) PageData {
	return PageData{
		Title:         "通知センター",
		Description:   "システムアラートと例外を一元管理し、対応状況を把握します。",
		Breadcrumbs:   breadcrumbItems(),
		TableEndpoint: joinBase(basePath, "/notifications/table"),
		Table:         table,
		Drawer:        drawer,
		Query:         state,
		Filters:       buildFilters(state, table),
		Legend:        severityLegend(),
	}
}

// TablePayload prepares the table fragment payload.
func TablePayload(state QueryState, feed adminnotifications.Feed, errMsg string, selectedID string) TableData {
	categoryCounts := make(map[string]int)
	severityCounts := make(map[string]int)
	statusCounts := make(map[string]int)

	rows := make([]TableRow, 0, len(feed.Items))
	for _, item := range feed.Items {
		categoryKey := string(item.Category)
		severityKey := string(item.Severity)
		statusKey := string(item.Status)
		categoryCounts[categoryKey]++
		severityCounts[severityKey]++
		statusCounts[statusKey]++
		rows = append(rows, toTableRow(item))
	}

	payload := TableData{
		Items:          rows,
		Total:          feed.Total,
		NextCursor:     feed.NextCursor,
		CategoryCounts: categoryCounts,
		SeverityCounts: severityCounts,
		StatusCounts:   statusCounts,
		SelectedID:     strings.TrimSpace(selectedID),
	}
	if payload.SelectedID == "" && len(rows) > 0 {
		payload.SelectedID = rows[0].ID
	}

	if errMsg != "" {
		payload.Error = errMsg
	}
	if len(rows) == 0 && payload.Error == "" {
		payload.EmptyMessage = "現在アクティブな通知はありません。"
	}

	return payload
}

// DrawerPayload selects a notification for the detail drawer.
func DrawerPayload(feed adminnotifications.Feed, selectedID string) DrawerData {
	if len(feed.Items) == 0 {
		return DrawerData{Empty: true}
	}

	selectedID = strings.TrimSpace(selectedID)
	var selected *adminnotifications.Notification
	if selectedID != "" {
		for idx := range feed.Items {
			item := feed.Items[idx]
			if item.ID == selectedID {
				selected = &item
				break
			}
		}
	}
	if selected == nil {
		selected = &feed.Items[0]
	}
	return toDrawerData(*selected)
}

// BadgePayload prepares the top-bar badge fragment payload.
func BadgePayload(basePath string, count adminnotifications.BadgeCount) BadgeData {
	return BadgeData{
		Total:          count.Total,
		Critical:       count.Critical,
		Warning:        count.Warning,
		Endpoint:       joinBase(basePath, "/notifications/badge"),
		StreamEndpoint: joinBase(basePath, "/notifications/stream"),
		Href:           joinBase(basePath, "/notifications"),
	}
}

func buildFilters(state QueryState, table TableData) Filters {
	return Filters{
		Categories: categoryOptions(state.Category, table.CategoryCounts),
		Severities: severityOptions(state.Severity, table.SeverityCounts),
		Statuses:   statusOptions(state.Status, table.StatusCounts),
	}
}

func categoryOptions(selected string, counts map[string]int) []CategoryOption {
	selected = strings.TrimSpace(selected)
	options := []CategoryOption{
		{Value: "", Label: "すべて", Icon: "🔔"},
		{Value: string(adminnotifications.CategoryFailedJob), Label: "ジョブ失敗", Icon: "🛠"},
		{Value: string(adminnotifications.CategoryStockAlert), Label: "在庫アラート", Icon: "📦"},
		{Value: string(adminnotifications.CategoryShippingException), Label: "配送例外", Icon: "🚚"},
	}
	for idx := range options {
		val := options[idx].Value
		options[idx].Active = val == selected
		if val == "" {
			options[idx].Count = totalMap(counts)
		} else {
			options[idx].Count = counts[val]
		}
	}
	return options
}

func severityOptions(selected string, counts map[string]int) []SeverityOption {
	selected = strings.TrimSpace(selected)
	options := []SeverityOption{
		{Value: "", Label: "すべて", Tone: "info"},
		{Value: string(adminnotifications.SeverityCritical), Label: "クリティカル", Tone: "danger"},
		{Value: string(adminnotifications.SeverityHigh), Label: "高", Tone: "warning"},
		{Value: string(adminnotifications.SeverityMedium), Label: "中", Tone: "warning"},
		{Value: string(adminnotifications.SeverityLow), Label: "低", Tone: "info"},
	}
	for idx := range options {
		val := options[idx].Value
		options[idx].Active = val == selected
		if val == "" {
			options[idx].Count = totalMap(counts)
		} else {
			options[idx].Count = counts[val]
		}
	}
	return options
}

func statusOptions(selected string, counts map[string]int) []SelectOption {
	summary := []struct {
		Value string
		Label string
	}{
		{Value: "", Label: fmt.Sprintf("すべて (%d)", totalMap(counts))},
		{Value: string(adminnotifications.StatusOpen), Label: fmt.Sprintf("未対応 (%d)", counts[string(adminnotifications.StatusOpen)])},
		{Value: string(adminnotifications.StatusAcknowledged), Label: fmt.Sprintf("対応中 (%d)", counts[string(adminnotifications.StatusAcknowledged)])},
		{Value: string(adminnotifications.StatusResolved), Label: fmt.Sprintf("解決済み (%d)", counts[string(adminnotifications.StatusResolved)])},
		{Value: string(adminnotifications.StatusSuppressed), Label: fmt.Sprintf("ミュート (%d)", counts[string(adminnotifications.StatusSuppressed)])},
	}
	options := make([]SelectOption, 0, len(summary))
	for _, item := range summary {
		label := item.Label
		if strings.TrimSpace(item.Value) == strings.TrimSpace(selected) {
			label = "✓ " + label
		}
		options = append(options, SelectOption{Value: item.Value, Label: label})
	}
	return options
}

func severityLegend() []LegendItem {
	return []LegendItem{
		{Label: "クリティカル", Tone: "danger", Icon: "🛑", Description: "即時対応が必要な致命的障害"},
		{Label: "高", Tone: "warning", Icon: "⚠️", Description: "優先対応が必要なアラート"},
		{Label: "中/低", Tone: "info", Icon: "ℹ️", Description: "状況確認やフォローアップ推奨"},
	}
}

func breadcrumbItems() []partials.Breadcrumb {
	return []partials.Breadcrumb{{Label: "通知センター"}}
}

func joinBase(base, suffix string) string {
	base = strings.TrimSpace(base)
	if base == "" || base == "/" {
		base = ""
	}
	if !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}
	path := base + suffix
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func toTableRow(item adminnotifications.Notification) TableRow {
	attrs := templ.Attributes{
		"data-notification-row":     "true",
		"data-notification-id":      item.ID,
		"data-notification-payload": encodeDetailPayload(item),
	}
	return TableRow{
		ID:                item.ID,
		CategoryLabel:     categoryLabel(item.Category),
		CategoryTone:      categoryTone(item.Category),
		CategoryIcon:      categoryIcon(item.Category),
		SeverityLabel:     severityLabel(item.Severity),
		SeverityTone:      severityTone(item.Severity),
		Title:             item.Title,
		Summary:           item.Summary,
		StatusLabel:       statusLabel(item.Status),
		StatusTone:        statusTone(item.Status),
		ResourceLabel:     item.Resource.Label,
		ResourceURL:       item.Resource.URL,
		ResourceKind:      item.Resource.Kind,
		Owner:             item.Owner,
		CreatedAt:         item.CreatedAt,
		CreatedAtRelative: helpers.Relative(item.CreatedAt),
		CreatedAtTooltip:  helpers.Date(item.CreatedAt, "2006-01-02 15:04"),
		Actions:           rowActions(item),
		Attributes:        attrs,
	}
}

func toDrawerData(item adminnotifications.Notification) DrawerData {
	data := DrawerData{
		Empty:         false,
		ID:            item.ID,
		Title:         item.Title,
		Summary:       item.Summary,
		CategoryLabel: categoryLabel(item.Category),
		CategoryTone:  categoryTone(item.Category),
		SeverityLabel: severityLabel(item.Severity),
		SeverityTone:  severityTone(item.Severity),
		StatusLabel:   statusLabel(item.Status),
		StatusTone:    statusTone(item.Status),
		Owner:         item.Owner,
		Resource: ResourceView{
			Label: item.Resource.Label,
			URL:   item.Resource.URL,
			Kind:  resourceLabel(item.Resource.Kind),
		},
		CreatedAt:       item.CreatedAt,
		CreatedRelative: helpers.Relative(item.CreatedAt),
		Metadata:        toMetadataViews(item.Metadata),
		Timeline:        toTimelineEvents(item.Timeline),
		Links:           rowActions(item),
	}
	if item.AcknowledgedAt != nil {
		data.AcknowledgedAt = item.AcknowledgedAt
		label := helpers.Date(*item.AcknowledgedAt, "2006-01-02 15:04")
		data.AcknowledgedLabel = fmt.Sprintf("%s（%s）", item.AcknowledgedBy, label)
	}
	if item.ResolvedAt != nil {
		data.ResolvedAt = item.ResolvedAt
		data.ResolvedLabel = helpers.Date(*item.ResolvedAt, "2006-01-02 15:04")
	}
	return data
}

func toMetadataViews(list []adminnotifications.Metadata) []MetadataView {
	if len(list) == 0 {
		return nil
	}
	result := make([]MetadataView, 0, len(list))
	for _, meta := range list {
		result = append(result, MetadataView{
			Label: meta.Label,
			Value: meta.Value,
			Icon:  meta.Icon,
		})
	}
	return result
}

func toTimelineEvents(list []adminnotifications.TimelineEvent) []TimelineEventView {
	if len(list) == 0 {
		return nil
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].OccurredAt.Before(list[j].OccurredAt)
	})
	result := make([]TimelineEventView, 0, len(list))
	for _, event := range list {
		result = append(result, TimelineEventView{
			Title:            event.Title,
			Description:      event.Description,
			OccurredAt:       event.OccurredAt,
			OccurredRelative: helpers.Relative(event.OccurredAt),
			Actor:            event.Actor,
			Tone:             event.Tone,
			Icon:             event.Icon,
		})
	}
	return result
}

func rowActions(item adminnotifications.Notification) []RowAction {
	actions := make([]RowAction, 0, len(item.Links))
	for _, link := range item.Links {
		actions = append(actions, RowAction{
			Label: link.Label,
			URL:   link.URL,
			Icon:  link.Icon,
		})
	}
	return actions
}

func totalMap(counts map[string]int) int {
	total := 0
	for _, count := range counts {
		total += count
	}
	return total
}

func categoryLabel(cat adminnotifications.Category) string {
	switch cat {
	case adminnotifications.CategoryFailedJob:
		return "ジョブ失敗"
	case adminnotifications.CategoryStockAlert:
		return "在庫アラート"
	case adminnotifications.CategoryShippingException:
		return "配送例外"
	default:
		return "通知"
	}
}

func categoryTone(cat adminnotifications.Category) string {
	switch cat {
	case adminnotifications.CategoryFailedJob:
		return "danger"
	case adminnotifications.CategoryStockAlert:
		return "warning"
	case adminnotifications.CategoryShippingException:
		return "info"
	default:
		return "info"
	}
}

func categoryIcon(cat adminnotifications.Category) string {
	switch cat {
	case adminnotifications.CategoryFailedJob:
		return "🛠"
	case adminnotifications.CategoryStockAlert:
		return "📦"
	case adminnotifications.CategoryShippingException:
		return "🚚"
	default:
		return "🔔"
	}
}

func severityLabel(sev adminnotifications.Severity) string {
	switch sev {
	case adminnotifications.SeverityCritical:
		return "クリティカル"
	case adminnotifications.SeverityHigh:
		return "高"
	case adminnotifications.SeverityMedium:
		return "中"
	case adminnotifications.SeverityLow:
		return "低"
	default:
		return "通知"
	}
}

func severityTone(sev adminnotifications.Severity) string {
	switch sev {
	case adminnotifications.SeverityCritical:
		return "danger"
	case adminnotifications.SeverityHigh:
		return "warning"
	case adminnotifications.SeverityMedium:
		return "warning"
	case adminnotifications.SeverityLow:
		return "info"
	default:
		return "info"
	}
}

func statusLabel(status adminnotifications.Status) string {
	switch status {
	case adminnotifications.StatusOpen:
		return "未対応"
	case adminnotifications.StatusAcknowledged:
		return "対応中"
	case adminnotifications.StatusResolved:
		return "解決済み"
	case adminnotifications.StatusSuppressed:
		return "ミュート"
	default:
		return "不明"
	}
}

func statusTone(status adminnotifications.Status) string {
	switch status {
	case adminnotifications.StatusOpen:
		return "danger"
	case adminnotifications.StatusAcknowledged:
		return "warning"
	case adminnotifications.StatusResolved:
		return "success"
	case adminnotifications.StatusSuppressed:
		return "info"
	default:
		return "info"
	}
}

func resourceLabel(kind string) string {
	kind = strings.TrimSpace(strings.ToLower(kind))
	switch kind {
	case "job":
		return "ジョブ"
	case "sku":
		return "SKU"
	case "order":
		return "注文"
	default:
		return "リソース"
	}
}

type detailPayload struct {
	ID                string              `json:"id"`
	Title             string              `json:"title"`
	Summary           string              `json:"summary"`
	Category          string              `json:"category"`
	CategoryLabel     string              `json:"categoryLabel"`
	CategoryTone      string              `json:"categoryTone"`
	Severity          string              `json:"severity"`
	SeverityLabel     string              `json:"severityLabel"`
	SeverityTone      string              `json:"severityTone"`
	Status            string              `json:"status"`
	StatusLabel       string              `json:"statusLabel"`
	StatusTone        string              `json:"statusTone"`
	Owner             string              `json:"owner"`
	Resource          ResourceView        `json:"resource"`
	CreatedAt         string              `json:"createdAt"`
	CreatedRelative   string              `json:"createdRelative"`
	Metadata          []MetadataView      `json:"metadata"`
	Timeline          []TimelineEventView `json:"timeline"`
	Links             []RowAction         `json:"links"`
	AcknowledgedLabel string              `json:"acknowledgedLabel"`
	ResolvedLabel     string              `json:"resolvedLabel"`
}

func encodeDetailPayload(item adminnotifications.Notification) string {
	data := detailPayload{
		ID:            item.ID,
		Title:         item.Title,
		Summary:       item.Summary,
		Category:      string(item.Category),
		CategoryLabel: categoryLabel(item.Category),
		CategoryTone:  categoryTone(item.Category),
		Severity:      string(item.Severity),
		SeverityLabel: severityLabel(item.Severity),
		SeverityTone:  severityTone(item.Severity),
		Status:        string(item.Status),
		StatusLabel:   statusLabel(item.Status),
		StatusTone:    statusTone(item.Status),
		Owner:         item.Owner,
		Resource: ResourceView{
			Label: item.Resource.Label,
			URL:   item.Resource.URL,
			Kind:  resourceLabel(item.Resource.Kind),
		},
		CreatedAt:       item.CreatedAt.Format(time.RFC3339),
		CreatedRelative: helpers.Relative(item.CreatedAt),
		Metadata:        toMetadataViews(item.Metadata),
		Timeline:        toTimelineEvents(item.Timeline),
		Links:           rowActions(item),
	}
	if item.AcknowledgedAt != nil {
		data.AcknowledgedLabel = fmt.Sprintf("%s（%s）", item.AcknowledgedBy, helpers.Date(*item.AcknowledgedAt, "2006-01-02 15:04"))
	}
	if item.ResolvedAt != nil {
		data.ResolvedLabel = helpers.Date(*item.ResolvedAt, "2006-01-02 15:04")
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return "{}"
	}
	return string(bytes)
}

func badgeDisplay(total int) string {
	switch {
	case total <= 0:
		return "0"
	case total > 99:
		return "99+"
	default:
		return fmt.Sprintf("%d", total)
	}
}

func boolAttr(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func tableRowClass(selected bool) string {
	base := "hover:bg-slate-50 transition cursor-pointer"
	if selected {
		return base + " bg-brand-50"
	}
	return base
}

func categoryOptionClass(active bool) string {
	base := "inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-sm font-medium transition focus-within:ring-2 focus-within:ring-brand-500"
	if active {
		return base + " border-brand-500 bg-brand-50 text-brand-600"
	}
	return base + " border-slate-200 text-slate-600 hover:border-slate-300 hover:text-slate-900"
}

func severityOptionClass(active bool, tone string) string {
	base := "inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-sm font-medium transition focus-within:ring-2 focus-within:ring-brand-500"
	if active {
		return base + " border-brand-500 bg-brand-50 text-brand-600"
	}
	switch tone {
	case "danger":
		return base + " border-danger-200 text-danger-600 hover:border-danger-300 hover:text-danger-700"
	case "warning":
		return base + " border-warning-200 text-warning-600 hover:border-warning-300 hover:text-warning-700"
	default:
		return base + " border-slate-200 text-slate-600 hover:border-slate-300 hover:text-slate-900"
	}
}
