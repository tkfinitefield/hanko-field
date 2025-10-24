package shipments

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	adminshipments "finitefield.org/hanko-admin/internal/admin/shipments"
	"finitefield.org/hanko-admin/internal/admin/templates/helpers"
	"finitefield.org/hanko-admin/internal/admin/templates/partials"
)

// PageData represents the server-rendered payload for the shipments batch page.
type PageData struct {
	Title         string
	Description   string
	Breadcrumbs   []partials.Breadcrumb
	TableEndpoint string
	Query         QueryState
	Filters       Filters
	Metrics       []MetricChip
	Table         TableData
	Drawer        DrawerData
}

// QueryState captures current filter state.
type QueryState struct {
	Status   string
	Carrier  string
	Facility string
	Start    string
	End      string
	Page     int
	PageSize int
	Selected string
	RawQuery string
}

// Filters enumerates filter option data.
type Filters struct {
	Statuses   []StatusChip
	Carriers   []SelectOption
	Facilities []SelectOption
	HasActive  bool
}

// StatusChip represents a selectable status filter chip.
type StatusChip struct {
	Value  string
	Label  string
	Tone   string
	Count  int
	Active bool
}

// SelectOption represents a select option.
type SelectOption struct {
	Value  string
	Label  string
	Count  int
	Active bool
}

// MetricChip represents KPI chip data.
type MetricChip struct {
	Label   string
	Value   string
	SubText string
	Tone    string
	Icon    string
}

// TableData powers the batches table fragment.
type TableData struct {
	BasePath     string
	FragmentPath string
	Rows         []TableRow
	SelectedID   string
	EmptyMessage string
	Error        string
	Pagination   Pagination
}

// Pagination describes pagination state.
type Pagination struct {
	Page     int
	PageSize int
	Total    int
	Next     *int
	Prev     *int
	RawQuery string
}

// TableRow represents a single table row.
type TableRow struct {
	ID              string
	Reference       string
	CreatedLabel    string
	CreatedRelative string
	CarrierLabel    string
	ServiceLevel    string
	FacilityLabel   string
	OrderStats      string
	ProgressPercent int
	ProgressLabel   string
	StatusLabel     string
	StatusTone      string
	SLAStatus       string
	SLATone         string
	BadgeIcon       string
	BadgeTone       string
	BadgeLabel      string
	LabelURL        string
	ManifestURL     string
	DetailURL       string
	Selected        bool
}

// DrawerData powers the batch detail drawer.
type DrawerData struct {
	Empty           bool
	ID              string
	Reference       string
	StatusLabel     string
	StatusTone      string
	Carrier         string
	ServiceLevel    string
	Facility        string
	CreatedLabel    string
	CreatedRelative string
	Operator        OperatorData
	Job             JobData
	OrderSummary    []DrawerOrder
	Timeline        []DrawerTimeline
	PrintHistory    []DrawerPrintRecord
	LabelURL        string
	ManifestURL     string
}

// OperatorData renders operator info.
type OperatorData struct {
	Name  string
	Email string
	Shift string
}

// JobData conveys async job state.
type JobData struct {
	StateLabel string
	StateTone  string
	Message    string
	Progress   int
	StartLabel string
	EndLabel   string
}

// DrawerOrder summarises orders within the drawer.
type DrawerOrder struct {
	OrderID     string
	OrderNumber string
	Customer    string
	Destination string
	Service     string
	LabelStatus string
	LabelTone   string
	LabelURL    string
}

// DrawerTimeline represents a timeline item.
type DrawerTimeline struct {
	Title       string
	Description string
	Tone        string
	Icon        string
	Timestamp   string
}

// DrawerPrintRecord renders print history.
type DrawerPrintRecord struct {
	Label     string
	Actor     string
	Channel   string
	Count     int
	Timestamp string
}

// BuildPageData assembles the full page payload.
func BuildPageData(basePath string, state QueryState, result adminshipments.ListResult, table TableData, drawer DrawerData) PageData {
	filters := buildFilters(state, result.Filters)
	metrics := buildMetrics(result.Summary)

	return PageData{
		Title:         "出荷バッチ管理",
		Description:   "ラベル生成と一括出荷バッチの進捗をモニタリングし、異常時に迅速に対応します。",
		Breadcrumbs:   breadcrumbItems(basePath),
		TableEndpoint: joinBase(basePath, "/shipments/batches/table"),
		Query:         state,
		Filters:       filters,
		Metrics:       metrics,
		Table:         table,
		Drawer:        drawer,
	}
}

// TablePayload prepares the table fragment payload.
func TablePayload(basePath string, state QueryState, result adminshipments.ListResult, errMsg string) TableData {
	selected := strings.TrimSpace(state.Selected)
	if selected == "" {
		selected = strings.TrimSpace(result.SelectedID)
	}
	rows := buildRows(basePath, result.Batches, selected)
	hasSelected := false
	for _, row := range rows {
		if row.Selected {
			hasSelected = true
			break
		}
	}
	if !hasSelected {
		if len(rows) > 0 {
			selected = rows[0].ID
			for i := range rows {
				rows[i].Selected = rows[i].ID == selected
			}
		} else {
			selected = ""
		}
	}
	if errMsg != "" {
		return TableData{
			BasePath:     basePath,
			FragmentPath: joinBase(basePath, "/shipments/batches/table"),
			Rows:         nil,
			SelectedID:   selected,
			EmptyMessage: "出荷バッチを読み込みできませんでした。",
			Error:        errMsg,
			Pagination:   buildPagination(state, result.Pagination),
		}
	}
	emptyMsg := "一致する出荷バッチが見つかりません。フィルター条件を調整してください。"
	if len(result.Batches) == 0 {
		selected = ""
	}
	return TableData{
		BasePath:     basePath,
		FragmentPath: joinBase(basePath, "/shipments/batches/table"),
		Rows:         rows,
		SelectedID:   selected,
		EmptyMessage: emptyMsg,
		Error:        "",
		Pagination:   buildPagination(state, result.Pagination),
	}
}

// DrawerPayload converts the detail response into drawer data.
func DrawerPayload(detail adminshipments.BatchDetail, selected string) DrawerData {
	if detail.Batch.ID == "" || strings.TrimSpace(selected) == "" {
		return DrawerData{Empty: true}
	}

	if strings.TrimSpace(detail.Batch.ID) != strings.TrimSpace(selected) {
		return DrawerData{Empty: true}
	}

	created := helpers.Date(detail.Batch.CreatedAt, "2006-01-02 15:04")
	relative := helpers.Relative(detail.Batch.CreatedAt)

	operator := OperatorData{
		Name:  strings.TrimSpace(detail.Operator.Name),
		Email: strings.TrimSpace(detail.Operator.Email),
		Shift: strings.TrimSpace(detail.Operator.Shift),
	}

	job := JobData{
		StateLabel: strings.TrimSpace(detail.Job.StateLabel),
		StateTone:  strings.TrimSpace(detail.Job.StateTone),
		Message:    strings.TrimSpace(detail.Job.Message),
		Progress:   clamp(detail.Job.Progress, 0, 100),
		StartLabel: formatOptionalTime(detail.Job.StartedAt),
		EndLabel:   formatOptionalTime(detail.Job.EndedAt),
	}

	orderSummary := make([]DrawerOrder, 0, len(detail.Orders))
	for _, order := range detail.Orders {
		orderSummary = append(orderSummary, DrawerOrder{
			OrderID:     strings.TrimSpace(order.OrderID),
			OrderNumber: strings.TrimSpace(order.OrderNumber),
			Customer:    strings.TrimSpace(order.CustomerName),
			Destination: strings.TrimSpace(order.Destination),
			Service:     strings.TrimSpace(order.ServiceLevel),
			LabelStatus: strings.TrimSpace(order.LabelStatus),
			LabelTone:   strings.TrimSpace(order.LabelTone),
			LabelURL:    strings.TrimSpace(order.LabelURL),
		})
	}

	timeline := make([]DrawerTimeline, 0, len(detail.Timeline))
	for _, event := range detail.Timeline {
		timeline = append(timeline, DrawerTimeline{
			Title:       strings.TrimSpace(event.Title),
			Description: strings.TrimSpace(event.Description),
			Tone:        strings.TrimSpace(event.Tone),
			Icon:        strings.TrimSpace(event.Icon),
			Timestamp:   helpers.Date(event.OccurredAt, "01/02 15:04"),
		})
	}

	prints := make([]DrawerPrintRecord, 0, len(detail.PrintHistory))
	for _, record := range detail.PrintHistory {
		label := strings.TrimSpace(record.Label)
		if label == "" {
			label = "ラベル出力"
		}
		prints = append(prints, DrawerPrintRecord{
			Label:     label,
			Actor:     strings.TrimSpace(record.Actor),
			Channel:   strings.TrimSpace(record.Channel),
			Count:     record.Count,
			Timestamp: helpers.Date(record.PrintedAt, "01/02 15:04"),
		})
	}

	return DrawerData{
		Empty:           false,
		ID:              detail.Batch.ID,
		Reference:       strings.TrimSpace(detail.Batch.Reference),
		StatusLabel:     strings.TrimSpace(detail.Batch.StatusLabel),
		StatusTone:      strings.TrimSpace(detail.Batch.StatusTone),
		Carrier:         labelOrFallback(detail.Batch.CarrierLabel, detail.Batch.Carrier),
		ServiceLevel:    strings.TrimSpace(detail.Batch.ServiceLevel),
		Facility:        labelOrFallback(detail.Batch.FacilityLabel, detail.Batch.Facility),
		CreatedLabel:    created,
		CreatedRelative: relative,
		Operator:        operator,
		Job:             job,
		OrderSummary:    orderSummary,
		Timeline:        timeline,
		PrintHistory:    prints,
		LabelURL:        strings.TrimSpace(detail.Batch.LabelDownloadURL),
		ManifestURL:     strings.TrimSpace(detail.Batch.ManifestURL),
	}
}

func buildRows(basePath string, batches []adminshipments.Batch, selected string) []TableRow {
	if len(batches) == 0 {
		return nil
	}

	result := make([]TableRow, 0, len(batches))
	for _, batch := range batches {
		created := helpers.Date(batch.CreatedAt, "01/02 15:04")
		relative := helpers.Relative(batch.CreatedAt)
		progressLabel := fmt.Sprintf("%d%%", clamp(batch.ProgressPercent, 0, 100))
		orderStats := fmt.Sprintf("%d / %d 件", batch.OrdersTotal-batch.OrdersPending, batch.OrdersTotal)
		if batch.OrdersTotal == 0 {
			orderStats = "0 件"
		}

		detailURL := joinBase(basePath, fmt.Sprintf("/shipments/batches/%s/drawer", url.PathEscape(batch.ID)))

		result = append(result, TableRow{
			ID:              batch.ID,
			Reference:       strings.TrimSpace(batch.Reference),
			CreatedLabel:    created,
			CreatedRelative: relative,
			CarrierLabel:    labelOrFallback(batch.CarrierLabel, batch.Carrier),
			ServiceLevel:    strings.TrimSpace(batch.ServiceLevel),
			FacilityLabel:   labelOrFallback(batch.FacilityLabel, batch.Facility),
			OrderStats:      orderStats,
			ProgressPercent: clamp(batch.ProgressPercent, 0, 100),
			ProgressLabel:   progressLabel,
			StatusLabel:     strings.TrimSpace(batch.StatusLabel),
			StatusTone:      strings.TrimSpace(batch.StatusTone),
			SLAStatus:       strings.TrimSpace(batch.SLAStatus),
			SLATone:         strings.TrimSpace(batch.SLATone),
			BadgeIcon:       strings.TrimSpace(batch.BadgeIcon),
			BadgeTone:       strings.TrimSpace(batch.BadgeTone),
			BadgeLabel:      strings.TrimSpace(batch.BadgeLabel),
			LabelURL:        strings.TrimSpace(batch.LabelDownloadURL),
			ManifestURL:     strings.TrimSpace(batch.ManifestURL),
			DetailURL:       detailURL,
			Selected:        strings.TrimSpace(selected) == strings.TrimSpace(batch.ID),
		})
	}
	return result
}

func buildFilters(state QueryState, summary adminshipments.FilterSummary) Filters {
	statuses := make([]StatusChip, 0, len(summary.StatusOptions))
	for _, option := range summary.StatusOptions {
		value := strings.TrimSpace(string(option.Value))
		statuses = append(statuses, StatusChip{
			Value:  value,
			Label:  strings.TrimSpace(option.Label),
			Tone:   strings.TrimSpace(option.Tone),
			Count:  option.Count,
			Active: value == strings.TrimSpace(state.Status),
		})
	}
	sort.SliceStable(statuses, func(i, j int) bool {
		return statuses[i].Label < statuses[j].Label
	})

	carriers := make([]SelectOption, 0, len(summary.CarrierOptions))
	for _, option := range summary.CarrierOptions {
		value := strings.TrimSpace(option.Value)
		carriers = append(carriers, SelectOption{
			Value:  value,
			Label:  strings.TrimSpace(option.Label),
			Count:  option.Count,
			Active: value == strings.TrimSpace(state.Carrier),
		})
	}
	sort.SliceStable(carriers, func(i, j int) bool {
		return carriers[i].Label < carriers[j].Label
	})

	facilities := make([]SelectOption, 0, len(summary.FacilityOptions))
	for _, option := range summary.FacilityOptions {
		value := strings.TrimSpace(option.Value)
		facilities = append(facilities, SelectOption{
			Value:  value,
			Label:  strings.TrimSpace(option.Label),
			Count:  option.Count,
			Active: value == strings.TrimSpace(state.Facility),
		})
	}
	sort.SliceStable(facilities, func(i, j int) bool {
		return facilities[i].Label < facilities[j].Label
	})

	hasActive := strings.TrimSpace(state.Status) != "" ||
		strings.TrimSpace(state.Carrier) != "" ||
		strings.TrimSpace(state.Facility) != "" ||
		strings.TrimSpace(state.Start) != "" ||
		strings.TrimSpace(state.End) != ""

	return Filters{
		Statuses:   statuses,
		Carriers:   carriers,
		Facilities: facilities,
		HasActive:  hasActive,
	}
}

func buildMetrics(summary adminshipments.Summary) []MetricChip {
	lastRun := ""
	if summary.LastRun != nil && !summary.LastRun.IsZero() {
		lastRun = helpers.Relative(*summary.LastRun)
	}

	return []MetricChip{
		{
			Label:   "Outstanding",
			Value:   fmt.Sprintf("%d 件", summary.Outstanding),
			SubText: "キュー待機中・未送信",
			Tone:    "info",
			Icon:    "📦",
		},
		{
			Label:   "処理中",
			Value:   fmt.Sprintf("%d 件", summary.InProgress),
			SubText: "現在ラベル生成中",
			Tone:    "warning",
			Icon:    "⚙️",
		},
		{
			Label:   "要対応",
			Value:   fmt.Sprintf("%d 件", summary.Warnings),
			SubText: "エラーや再処理が必要",
			Tone:    "danger",
			Icon:    "🚨",
		},
		{
			Label:   "最新完了",
			Value:   firstNonEmpty(lastRun, "未実行"),
			SubText: "最新の完了バッチ",
			Tone:    "success",
			Icon:    "✅",
		},
	}
}

func buildPagination(state QueryState, pag adminshipments.Pagination) Pagination {
	return Pagination{
		Page:     pag.Page,
		PageSize: pag.PageSize,
		Total:    pag.TotalItems,
		Next:     pag.NextPage,
		Prev:     pag.PrevPage,
		RawQuery: state.RawQuery,
	}
}

func breadcrumbItems(base string) []partials.Breadcrumb {
	return []partials.Breadcrumb{
		{Label: "受注管理", Href: joinBase(base, "/orders")},
		{Label: "出荷バッチ", Href: joinBase(base, "/shipments/batches")},
	}
}

func formatOptionalTime(ts *time.Time) string {
	if ts == nil || ts.IsZero() {
		return ""
	}
	return helpers.Date(ts.In(time.Local), "2006-01-02 15:04")
}

func labelOrFallback(label, fallback string) string {
	label = strings.TrimSpace(label)
	if label != "" {
		return label
	}
	return strings.TrimSpace(fallback)
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func joinBase(base, suffix string) string {
	b := strings.TrimSpace(base)
	if b == "" {
		b = "/admin"
	}
	u := strings.TrimSpace(suffix)
	if !strings.HasPrefix(u, "/") {
		u = "/" + u
	}
	return strings.TrimRight(b, "/") + u
}
