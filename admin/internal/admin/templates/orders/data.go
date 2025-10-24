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
	StatusCell         StatusCellData
}

// StatusCellData represents the badge stack rendered within the status column.
type StatusCellData struct {
	OrderID       string
	ModalURL      string
	StatusLabel   string
	StatusTone    string
	PaymentLabel  string
	PaymentTone   string
	PaymentDue    string
	SLAStatus     string
	SLAStatusTone string
	ContainerID   string
}

// StatusModalData provides context for the status update modal.
type StatusModalData struct {
	OrderID        string
	OrderNumber    string
	CurrentStatus  string
	CurrentLabel   string
	ActionURL      string
	CSRFToken      string
	Options        []StatusModalOption
	Note           string
	NotifyCustomer bool
	Error          string
	Timeline       StatusTimelineData
}

// StatusModalOption describes a selectable status transition.
type StatusModalOption struct {
	Value          string
	Label          string
	Description    string
	Disabled       bool
	DisabledReason string
	Selected       bool
}

// StatusTimelineData captures a concise list of recent timeline entries.
type StatusTimelineData struct {
	OrderID     string
	ContainerID string
	Events      []StatusTimelineItem
}

// StatusTimelineItem represents a single timeline entry view.
type StatusTimelineItem struct {
	ID        string
	Title     string
	Body      string
	Actor     string
	Timestamp string
	Relative  string
	Tone      string
}

// StatusUpdateSuccessData bundles fragments to update after a successful transition.
type StatusUpdateSuccessData struct {
	Cell     StatusCellData
	Timeline StatusTimelineData
}

// RefundModalData describes the refund modal view model.
type RefundModalData struct {
	OrderID         string
	OrderNumber     string
	CustomerName    string
	Total           string
	Currency        string
	PaymentStatus   string
	PaymentTone     string
	Outstanding     string
	SupportsPartial bool
	ActionURL       string
	CSRFToken       string
	SelectedPayment string
	AmountInput     string
	Reason          string
	NotifyCustomer  bool
	PaymentClass    string
	AmountClass     string
	ReasonClass     string
	Error           string
	FieldErrors     map[string]string
	Payments        []RefundPaymentOptionData
	Refunds         []RefundRecordData
}

// RefundPaymentOptionData represents a selectable payment in the modal.
type RefundPaymentOptionData struct {
	ID             string
	Label          string
	Caption        string
	Status         string
	StatusTone     string
	Captured       string
	Refunded       string
	Available      string
	AvailableMinor int64
	Selected       bool
	Disabled       bool
}

// RefundRecordData summarises previously issued refunds.
type RefundRecordData struct {
	ID         string
	Status     string
	StatusTone string
	Amount     string
	Reason     string
	Actor      string
	Reference  string
	Processed  string
	Relative   string
}

// RefundFormState stores user input when re-rendering the modal.
type RefundFormState struct {
	PaymentID      string
	Amount         string
	Reason         string
	NotifyCustomer bool
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

		row.StatusCell = buildStatusCell(basePath, order)

		rows = append(rows, row)
	}
	return rows
}

func buildStatusCell(basePath string, order adminorders.Order) StatusCellData {
	return StatusCellData{
		OrderID:       order.ID,
		ModalURL:      joinBase(basePath, "/orders/"+strings.TrimSpace(order.ID)+"/modal/status"),
		StatusLabel:   order.StatusLabel,
		StatusTone:    order.StatusTone,
		PaymentLabel:  paymentLabel(order.Payment),
		PaymentTone:   paymentTone(order.Payment),
		PaymentDue:    paymentDue(order.Payment),
		SLAStatus:     order.Fulfillment.SLAStatus,
		SLAStatusTone: order.Fulfillment.SLAStatusTone,
		ContainerID:   fmt.Sprintf("order-status-%s", strings.TrimSpace(order.ID)),
	}
}

// StatusCellPayload converts an order into a status cell fragment payload.
func StatusCellPayload(basePath string, order adminorders.Order) StatusCellData {
	return buildStatusCell(basePath, order)
}

// StatusModalPayload prepares data for rendering the status update modal.
func StatusModalPayload(basePath string, modal adminorders.StatusModal, csrfToken, note string, notify bool, errMsg string) StatusModalData {
	options := make([]StatusModalOption, 0, len(modal.Choices))
	for _, choice := range modal.Choices {
		options = append(options, StatusModalOption{
			Value:          string(choice.Value),
			Label:          choice.Label,
			Description:    choice.Description,
			Disabled:       choice.Disabled,
			DisabledReason: choice.DisabledReason,
			Selected:       choice.Selected,
		})
	}

	return StatusModalData{
		OrderID:        modal.Order.ID,
		OrderNumber:    modal.Order.Number,
		CurrentStatus:  string(modal.Order.Status),
		CurrentLabel:   modal.Order.StatusLabel,
		ActionURL:      joinBase(basePath, "/orders/"+strings.TrimSpace(modal.Order.ID)+":status"),
		CSRFToken:      csrfToken,
		Options:        options,
		Note:           note,
		NotifyCustomer: notify,
		Error:          errMsg,
		Timeline:       StatusTimelinePayload(modal.Order.ID, modal.LatestTimeline),
	}
}

// StatusTimelinePayload converts timeline entries into template items.
func StatusTimelinePayload(orderID string, events []adminorders.TimelineEvent) StatusTimelineData {
	items := make([]StatusTimelineItem, 0, len(events))
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		title := strings.TrimSpace(event.Title)
		if title == "" {
			title = "ステータス更新"
		}
		body := strings.TrimSpace(event.Description)
		actor := strings.TrimSpace(event.Actor)
		if actor == "" {
			actor = "システム"
		}
		items = append(items, StatusTimelineItem{
			ID:        event.ID,
			Title:     title,
			Body:      body,
			Actor:     actor,
			Timestamp: helpers.Date(event.OccurredAt, "2006-01-02 15:04"),
			Relative:  helpers.Relative(event.OccurredAt),
			Tone:      statusToneForStatus(event.Status),
		})
	}
	return StatusTimelineData{OrderID: orderID, ContainerID: fmt.Sprintf("order-timeline-%s", strings.TrimSpace(orderID)), Events: items}
}

func statusToneForStatus(status adminorders.Status) string {
	switch status {
	case adminorders.StatusPendingPayment, adminorders.StatusPaymentReview:
		return "warning"
	case adminorders.StatusInProduction, adminorders.StatusReadyToShip, adminorders.StatusShipped:
		return "info"
	case adminorders.StatusDelivered:
		return "success"
	case adminorders.StatusRefunded, adminorders.StatusCancelled:
		return "muted"
	default:
		return "info"
	}
}

// StatusUpdateSuccessPayload creates the combined fragment payload for OOB swaps.
func StatusUpdateSuccessPayload(cell StatusCellData, timeline StatusTimelineData) StatusUpdateSuccessData {
	return StatusUpdateSuccessData{Cell: cell, Timeline: timeline}
}

// RefundModalPayload prepares the refund modal payload from service data.
func RefundModalPayload(basePath string, modal adminorders.RefundModal, csrfToken string, form RefundFormState, errMsg string, fieldErrors map[string]string) RefundModalData {
	order := modal.Order

	selected := strings.TrimSpace(form.PaymentID)
	if selected == "" {
		selected = defaultRefundPayment(modal.Payments)
	}

	amountInput := strings.TrimSpace(form.Amount)
	if amountInput == "" {
		if payment := findRefundPayment(modal.Payments, selected); payment != nil && payment.AvailableMinor > 0 {
			value := payment.AvailableMinor
			amountInput = formatMajorUnits(&value)
		}
	}

	payments := make([]RefundPaymentOptionData, 0, len(modal.Payments))
	for _, payment := range modal.Payments {
		payments = append(payments, buildRefundPaymentOption(payment, selected))
	}

	refunds := make([]RefundRecordData, 0, len(modal.ExistingRefunds))
	for _, record := range modal.ExistingRefunds {
		currency := strings.TrimSpace(record.Currency)
		if currency == "" {
			currency = strings.TrimSpace(modal.Currency)
		}
		if currency == "" {
			currency = strings.TrimSpace(order.Currency)
		}
		refunds = append(refunds, RefundRecordData{
			ID:         record.ID,
			Status:     safeText(record.Status, "処理中"),
			StatusTone: refundStatusTone(record.Status),
			Amount:     helpers.Currency(record.AmountMinor, currency),
			Reason:     strings.TrimSpace(record.Reason),
			Actor:      strings.TrimSpace(record.Actor),
			Reference:  strings.TrimSpace(record.Reference),
			Processed:  helpers.Date(record.ProcessedAt, "2006-01-02 15:04"),
			Relative:   helpers.Relative(record.ProcessedAt),
		})
	}

	var errs map[string]string
	if len(fieldErrors) > 0 {
		errs = make(map[string]string, len(fieldErrors))
		for key, value := range fieldErrors {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			errs[strings.TrimSpace(key)] = value
		}
	}

	currency := strings.TrimSpace(modal.Currency)
	if currency == "" {
		currency = strings.TrimSpace(order.Currency)
	}

	baseClass := "w-full rounded-lg border px-3 py-2 text-sm text-slate-700 shadow-sm focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-200"
	defaultClass := helpers.ClassList(baseClass, "border-slate-300")
	errorClass := helpers.ClassList(baseClass, "border-rose-400 focus:border-rose-500 focus:ring-rose-200")

	paymentClass := defaultClass
	if errs != nil && errs["paymentID"] != "" {
		paymentClass = errorClass
	}
	amountClass := defaultClass
	if errs != nil && errs["amount"] != "" {
		amountClass = errorClass
	}
	reasonClass := defaultClass
	if errs != nil && errs["reason"] != "" {
		reasonClass = errorClass
	}

	return RefundModalData{
		OrderID:         strings.TrimSpace(order.ID),
		OrderNumber:     strings.TrimSpace(order.Number),
		CustomerName:    strings.TrimSpace(order.CustomerName),
		Total:           helpers.Currency(order.TotalMinor, order.Currency),
		Currency:        currency,
		PaymentStatus:   strings.TrimSpace(order.PaymentStatus),
		PaymentTone:     strings.TrimSpace(order.PaymentTone),
		Outstanding:     strings.TrimSpace(order.OutstandingDue),
		SupportsPartial: modal.SupportsPartial,
		ActionURL:       joinBase(basePath, "/orders/"+strings.TrimSpace(order.ID)+"/payments:refund"),
		CSRFToken:       csrfToken,
		SelectedPayment: selected,
		AmountInput:     amountInput,
		Reason:          strings.TrimSpace(form.Reason),
		NotifyCustomer:  form.NotifyCustomer,
		PaymentClass:    paymentClass,
		AmountClass:     amountClass,
		ReasonClass:     reasonClass,
		Error:           strings.TrimSpace(errMsg),
		FieldErrors:     errs,
		Payments:        payments,
		Refunds:         refunds,
	}
}

func defaultRefundPayment(payments []adminorders.RefundPaymentOption) string {
	for _, payment := range payments {
		if payment.SupportsRefunds && payment.AvailableMinor > 0 {
			return payment.ID
		}
	}
	if len(payments) > 0 {
		return payments[0].ID
	}
	return ""
}

func findRefundPayment(payments []adminorders.RefundPaymentOption, id string) *adminorders.RefundPaymentOption {
	for i, payment := range payments {
		if payment.ID == id {
			return &payments[i]
		}
	}
	return nil
}

func buildRefundPaymentOption(payment adminorders.RefundPaymentOption, selected string) RefundPaymentOptionData {
	label := strings.TrimSpace(payment.Label)
	if label == "" {
		label = "支払い"
	}
	if ref := strings.TrimSpace(payment.Reference); ref != "" {
		label = label + " (" + ref + ")"
	}

	captionParts := []string{}
	if method := strings.TrimSpace(payment.Method); method != "" {
		captionParts = append(captionParts, method)
	}
	if captured := payment.CapturedMinor; captured > 0 {
		captionParts = append(captionParts, "売上 "+helpers.Currency(captured, payment.Currency))
	}
	caption := strings.Join(captionParts, " · ")

	return RefundPaymentOptionData{
		ID:             payment.ID,
		Label:          label,
		Caption:        caption,
		Status:         safeText(payment.Status, "処理中"),
		StatusTone:     strings.TrimSpace(payment.StatusTone),
		Captured:       helpers.Currency(payment.CapturedMinor, payment.Currency),
		Refunded:       helpers.Currency(payment.RefundedMinor, payment.Currency),
		Available:      helpers.Currency(payment.AvailableMinor, payment.Currency),
		AvailableMinor: payment.AvailableMinor,
		Selected:       strings.TrimSpace(payment.ID) == strings.TrimSpace(selected),
		Disabled:       !payment.SupportsRefunds || payment.AvailableMinor <= 0,
	}
}

func refundStatusTone(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "succeeded", "success", "completed":
		return "success"
	case "pending", "processing":
		return "info"
	case "failed", "error", "declined":
		return "danger"
	default:
		return "info"
	}
}

func safeText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
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
