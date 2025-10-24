package orders

import (
	"fmt"
	"math"
	"strings"
	"time"

	adminorders "finitefield.org/hanko-admin/internal/admin/orders"
	"finitefield.org/hanko-admin/internal/admin/templates/helpers"
	"finitefield.org/hanko-admin/internal/admin/templates/partials"
)

// PageData represents the payload for the orders index page.
type PageData struct {
	Title         string
	Description   string
	Breadcrumbs   []partials.Breadcrumb
	TableEndpoint string
	Query         QueryState
	Filters       Filters
	Table         TableData
	Metrics       []MetricCard
	LastUpdated   string
	LastRelative  string
}

// QueryState captures current filter and view state.
type QueryState struct {
	Status     string
	Since      string
	Currency   string
	AmountMin  string
	AmountMax  string
	HasRefund  string
	Sort       string
	SortKey    string
	SortDir    string
	SortToken  string
	Page       int
	PageSize   int
	RawQuery   string
	HasFilters bool
}

// Filters encapsulates filter control data.
type Filters struct {
	StatusOptions   []StatusFilterOption
	CurrencyOptions []SelectOption
	RefundOptions   []SelectOption
	AmountPresets   []AmountPreset
	HasActive       bool
}

// StatusFilterOption represents a status dropdown option.
type StatusFilterOption struct {
	Value  string
	Label  string
	Count  int
	Active bool
}

// SelectOption represents a select menu option.
type SelectOption struct {
	Value    string
	Label    string
	Selected bool
}

// AmountPreset represents a quick amount range shortcut.
type AmountPreset struct {
	Label   string
	Min     string
	Max     string
	Active  bool
	Encoded string
}

// TableData contains the fragment payload for the orders table.
type TableData struct {
	BasePath     string
	FragmentPath string
	HxTarget     string
	HxSwap       string
	Rows         []TableRow
	Error        string
	EmptyMessage string
	Pagination   Pagination
	Sort         SortState
}

// Pagination describes pagination metadata.
type Pagination struct {
	Page     int
	PageSize int
	Total    int
	TotalPtr *int
	Next     *int
	Prev     *int
}

// SortState describes current sort for header controls.
type SortState struct {
	Active       string
	BasePath     string
	FragmentPath string
	RawQuery     string
	Param        string
	PageParam    string
	HxTarget     string
	HxSwap       string
	HxPushURL    bool
}

// TableRow represents a single table row.
type TableRow struct {
	Index              int
	ID                 string
	CheckboxID         string
	Number             string
	URL                string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	UpdatedLabel       string
	UpdatedRelative    string
	CustomerName       string
	CustomerEmail      string
	CustomerPhone      string
	CustomerMeta       string
	Total              string
	Currency           string
	StatusLabel        string
	StatusTone         string
	SLAStatus          string
	SLAStatusTone      string
	PaymentLabel       string
	PaymentTone        string
	PaymentDue         string
	Badges             []BadgeView
	Tags               []string
	HasRefundRequest   bool
	Notes              []string
	ItemsSummary       string
	SalesChannel       string
	Integration        string
	PromisedAtLabel    string
	PromisedAtRelative string
}

// BadgeView describes a badge to render for a row.
type BadgeView struct {
	Label string
	Tone  string
	Icon  string
	Title string
}

// MetricCard represents a summary metric card.
type MetricCard struct {
	Key         string
	Label       string
	Value       string
	SubText     string
	Tone        string
	Icon        string
	Description string
}

// BuildPageData assembles the full SSR payload for the orders page.
func BuildPageData(basePath string, state QueryState, result adminorders.ListResult, table TableData) PageData {
	filters := buildFilters(state, result.Filters)
	metrics := buildMetrics(result.Summary)
	lastUpdated := ""
	lastRelative := ""
	if !result.Summary.LastRefreshedAt.IsZero() {
		lastUpdated = helpers.Date(result.Summary.LastRefreshedAt, "2006-01-02 15:04")
		lastRelative = helpers.Relative(result.Summary.LastRefreshedAt)
	}

	return PageData{
		Title:         "注文一覧",
		Description:   "全チャネルの注文状況を一元管理し、進捗やSLA遅延を把握します。",
		Breadcrumbs:   breadcrumbItems(),
		TableEndpoint: joinBase(basePath, "/orders/table"),
		Query:         state,
		Filters:       filters,
		Table:         table,
		Metrics:       metrics,
		LastUpdated:   lastUpdated,
		LastRelative:  lastRelative,
	}
}

// TablePayload prepares the table fragment data.
func TablePayload(basePath string, state QueryState, result adminorders.ListResult, errMsg string) TableData {
	rows := toTableRows(basePath, result.Orders)
	empty := ""
	if errMsg == "" && len(rows) == 0 {
		empty = "条件に一致する注文はありません。フィルタを調整してください。"
	}

	pagination := toPagination(result.Pagination)

	return TableData{
		BasePath:     joinBase(basePath, "/orders"),
		FragmentPath: joinBase(basePath, "/orders/table"),
		HxTarget:     "#orders-table",
		HxSwap:       "outerHTML",
		Rows:         rows,
		Error:        errMsg,
		EmptyMessage: empty,
		Pagination:   pagination,
		Sort: SortState{
			Active:       state.Sort,
			BasePath:     joinBase(basePath, "/orders"),
			FragmentPath: joinBase(basePath, "/orders/table"),
			RawQuery:     state.RawQuery,
			Param:        "sort",
			PageParam:    "page",
			HxTarget:     "#orders-table",
			HxSwap:       "outerHTML",
			HxPushURL:    true,
		},
	}
}

func toPagination(p adminorders.Pagination) Pagination {
	totalPtr := (*int)(nil)
	if p.TotalItems >= 0 {
		value := p.TotalItems
		totalPtr = &value
	}

	return Pagination{
		Page:     p.Page,
		PageSize: p.PageSize,
		Total:    p.TotalItems,
		TotalPtr: totalPtr,
		Next:     p.NextPage,
		Prev:     p.PrevPage,
	}
}

func toTableRows(basePath string, orders []adminorders.Order) []TableRow {
	rows := make([]TableRow, 0, len(orders))
	for index, order := range orders {
		row := TableRow{
			Index:            index,
			ID:               order.ID,
			CheckboxID:       fmt.Sprintf("order-%02d", index),
			Number:           order.Number,
			URL:              joinBase(basePath, "/orders/"+strings.TrimSpace(order.Number)),
			CreatedAt:        order.CreatedAt,
			UpdatedAt:        order.UpdatedAt,
			UpdatedLabel:     helpers.Date(order.UpdatedAt, "2006-01-02 15:04"),
			UpdatedRelative:  helpers.Relative(order.UpdatedAt),
			CustomerName:     order.Customer.Name,
			CustomerEmail:    order.Customer.Email,
			CustomerPhone:    order.Customer.Phone,
			Total:            helpers.Currency(order.TotalMinor, order.Currency),
			Currency:         order.Currency,
			StatusLabel:      order.StatusLabel,
			StatusTone:       order.StatusTone,
			SLAStatus:        order.Fulfillment.SLAStatus,
			SLAStatusTone:    order.Fulfillment.SLAStatusTone,
			PaymentLabel:     paymentLabel(order.Payment),
			PaymentTone:      paymentTone(order.Payment),
			PaymentDue:       paymentDue(order.Payment),
			Badges:           toBadgeViews(order.Badges),
			Tags:             append([]string(nil), order.Tags...),
			HasRefundRequest: order.HasRefundRequest,
			Notes:            append([]string(nil), order.Notes...),
			ItemsSummary:     order.ItemsSummary,
			SalesChannel:     order.SalesChannel,
			Integration:      order.Integration,
		}

		if order.Customer.Email != "" && order.SalesChannel != "" {
			row.CustomerMeta = fmt.Sprintf("%s · %s", order.SalesChannel, order.Customer.Email)
		} else if order.SalesChannel != "" {
			row.CustomerMeta = order.SalesChannel
		} else {
			row.CustomerMeta = order.Customer.Email
		}

		if order.Fulfillment.PromisedDate != nil {
			row.PromisedAtLabel = helpers.Date(*order.Fulfillment.PromisedDate, "2006-01-02 15:04")
			row.PromisedAtRelative = helpers.Relative(*order.Fulfillment.PromisedDate)
		}

		rows = append(rows, row)
	}
	return rows
}

func paymentLabel(p adminorders.Payment) string {
	if strings.TrimSpace(p.Status) != "" {
		return strings.TrimSpace(p.Status)
	}
	if p.PastDue {
		return "期限超過"
	}
	return "未設定"
}

func paymentTone(p adminorders.Payment) string {
	if strings.TrimSpace(p.StatusTone) != "" {
		return strings.TrimSpace(p.StatusTone)
	}
	if p.PastDue {
		return "danger"
	}
	return "info"
}

func paymentDue(p adminorders.Payment) string {
	if p.DueAt != nil {
		return helpers.Date(*p.DueAt, "2006-01-02 15:04")
	}
	return ""
}

func toBadgeViews(badges []adminorders.Badge) []BadgeView {
	result := make([]BadgeView, 0, len(badges))
	for _, badge := range badges {
		result = append(result, BadgeView{
			Label: badge.Label,
			Tone:  badge.Tone,
			Icon:  badge.Icon,
			Title: badge.Title,
		})
	}
	return result
}

func buildFilters(state QueryState, summary adminorders.FilterSummary) Filters {
	statusOptions := make([]StatusFilterOption, 0, len(summary.StatusOptions)+1)
	statusOptions = append(statusOptions, StatusFilterOption{
		Value:  "",
		Label:  "すべて",
		Count:  totalStatusCount(summary.StatusOptions),
		Active: strings.TrimSpace(state.Status) == "",
	})
	for _, option := range summary.StatusOptions {
		statusOptions = append(statusOptions, StatusFilterOption{
			Value:  string(option.Value),
			Label:  option.Label,
			Count:  option.Count,
			Active: strings.EqualFold(state.Status, string(option.Value)),
		})
	}

	currencyOptions := []SelectOption{
		{Value: "", Label: "すべて", Selected: strings.TrimSpace(state.Currency) == ""},
	}
	for _, option := range summary.CurrencyOptions {
		currencyOptions = append(currencyOptions, SelectOption{
			Value:    option.Code,
			Label:    option.Label,
			Selected: strings.EqualFold(state.Currency, option.Code),
		})
	}

	refundOptions := make([]SelectOption, 0, len(summary.RefundOptions))
	currentRefund := strings.TrimSpace(state.HasRefund)
	for _, option := range summary.RefundOptions {
		value := strings.TrimSpace(option.Value)
		if value == "" {
			value = ""
		}
		label := option.Label
		selected := false
		switch value {
		case "":
			selected = currentRefund == "" || currentRefund == "any"
		case "true":
			selected = currentRefund == "true"
		case "false":
			selected = currentRefund == "false"
		default:
			selected = currentRefund == value
		}
		refundOptions = append(refundOptions, SelectOption{
			Value:    value,
			Label:    label,
			Selected: selected,
		})
	}

	amountPresets := make([]AmountPreset, 0, len(summary.AmountRanges))
	for _, preset := range summary.AmountRanges {
		min := formatMajorUnits(preset.Min)
		max := formatMajorUnits(preset.Max)
		active := false
		if strings.TrimSpace(state.AmountMin) == min && strings.TrimSpace(state.AmountMax) == max {
			active = true
		}
		amountPresets = append(amountPresets, AmountPreset{
			Label:   preset.Label,
			Min:     min,
			Max:     max,
			Active:  active,
			Encoded: encodeAmountRange(min, max),
		})
	}

	hasActive := state.HasFilters

	return Filters{
		StatusOptions:   statusOptions,
		CurrencyOptions: currencyOptions,
		RefundOptions:   refundOptions,
		AmountPresets:   amountPresets,
		HasActive:       hasActive,
	}
}

func buildMetrics(summary adminorders.Summary) []MetricCard {
	totalOrders := MetricCard{
		Key:         "total_orders",
		Label:       "対象の注文",
		Value:       fmt.Sprintf("%d 件", summary.TotalOrders),
		SubText:     fmt.Sprintf("直近24時間で %d 件発送", summary.FulfilledLast24h),
		Tone:        "info",
		Icon:        "📦",
		Description: "条件に一致した注文数です。",
	}

	totalRevenue := MetricCard{
		Key:         "total_revenue",
		Label:       "合計売上",
		Value:       helpers.Currency(summary.TotalRevenueMinor, summary.PrimaryCurrency),
		SubText:     fmt.Sprintf("平均リードタイム %.1f 時間", summary.AverageLeadHours),
		Tone:        "success",
		Icon:        "💴",
		Description: "表示中の注文に対する売上概算値です。",
	}

	inProduction := MetricCard{
		Key:         "in_production",
		Label:       "制作中",
		Value:       fmt.Sprintf("%d 件", summary.InProductionCount),
		SubText:     fmt.Sprintf("遅延 %d 件 / 返金申請 %d 件", summary.DelayedCount, summary.RefundRequested),
		Tone:        "warning",
		Icon:        "🛠",
		Description: "制作中件数と遅延状況の概要です。",
	}

	return []MetricCard{totalOrders, totalRevenue, inProduction}
}

func totalStatusCount(options []adminorders.StatusOption) int {
	total := 0
	for _, option := range options {
		total += option.Count
	}
	return total
}

func formatMajorUnits(value *int64) string {
	if value == nil {
		return ""
	}
	major := float64(*value) / 100.0
	if math.Mod(major, 1.0) == 0 {
		return fmt.Sprintf("%.0f", major)
	}
	return fmt.Sprintf("%.2f", major)
}

func encodeAmountRange(min, max string) string {
	return strings.TrimSpace(min) + ":" + strings.TrimSpace(max)
}

func breadcrumbItems() []partials.Breadcrumb {
	return []partials.Breadcrumb{
		{Label: "受注管理", Href: ""},
		{Label: "注文一覧", Href: ""},
	}
}

func joinBase(base, suffix string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "/admin"
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	if base != "/" {
		base = strings.TrimRight(base, "/")
	}
	if !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}
	return base + suffix
}
