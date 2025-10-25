package shipments

import (
	"fmt"
	"net/url"
	"strings"

	adminshipments "finitefield.org/hanko-admin/internal/admin/shipments"
	"finitefield.org/hanko-admin/internal/admin/templates/helpers"
	"finitefield.org/hanko-admin/internal/admin/templates/partials"
)

// TrackingPageData represents the tracking monitor payload.
type TrackingPageData struct {
	Title         string
	Description   string
	Breadcrumbs   []partials.Breadcrumb
	TableEndpoint string
	Query         TrackingQueryState
	Filters       TrackingFilterData
	Summary       TrackingSummaryData
	Alerts        []TrackingAlertData
	Table         TrackingTableData
}

// TrackingQueryState reflects query params.
type TrackingQueryState struct {
	Status      string
	Carrier     string
	Lane        string
	Region      string
	DelayWindow string
	Page        int
	PageSize    int
	RawQuery    string
}

// TrackingFilterData enumerates filter UI data.
type TrackingFilterData struct {
	Statuses []TrackingStatusChip
	Carriers []TrackingSelectOption
	Lanes    []TrackingSelectOption
	Regions  []TrackingSelectOption
	HasAny   bool
}

// TrackingStatusChip renders selectable chips.
type TrackingStatusChip struct {
	Value  string
	Label  string
	Tone   string
	Count  int
	Active bool
}

// TrackingSelectOption renders select options.
type TrackingSelectOption struct {
	Value  string
	Label  string
	Count  int
	Active bool
}

// TrackingSummaryData powers KPI cards.
type TrackingSummaryData struct {
	ActiveLabel    string
	DelayedLabel   string
	ExceptionLabel string
	LastRefresh    string
	RefreshEvery   string
	RefreshSeconds int
}

// TrackingAlertData renders alert banner.
type TrackingAlertData struct {
	Label       string
	Description string
	Tone        string
	ActionLabel string
	ActionURL   string
}

// TrackingTableData powers the table fragment.
type TrackingTableData struct {
	BasePath     string
	FragmentPath string
	Rows         []TrackingRow
	EmptyMessage string
	Error        string
	Pagination   TrackingPagination
	AutoRefresh  bool
	RefreshEvery int
	RawQuery     string
}

// TrackingRow represents a shipment row.
type TrackingRow struct {
	ID             string
	OrderNumber    string
	OrderURL       string
	Customer       string
	Carrier        string
	Service        string
	StatusLabel    string
	StatusTone     string
	TrackingNumber string
	Destination    string
	Region         string
	Lane           string
	LastEvent      string
	LastEventTime  string
	ETA            string
	SLAStatus      string
	SLATone        string
	Exception      string
	ExceptionTone  string
	BadgeIcon      string
	DelayLabel     string
}

// TrackingPagination describes table pagination.
type TrackingPagination struct {
	Page     int
	PageSize int
	Total    int
	Next     *int
	Prev     *int
	RawQuery string
}

// BuildTrackingPageData assembles the page payload.
func BuildTrackingPageData(basePath string, state TrackingQueryState, result adminshipments.TrackingResult, table TrackingTableData) TrackingPageData {
	alerts := buildTrackingAlerts(result.Alerts, basePath)
	filters := buildTrackingFilters(state, result.Filters)
	summary := buildTrackingSummary(result.Summary)

	return TrackingPageData{
		Title:         "配送状況モニタ",
		Description:   "キャリア横断で配送状況を監視し、遅延や例外に迅速に対応します。",
		Breadcrumbs:   trackingBreadcrumbs(basePath),
		TableEndpoint: joinBase(basePath, "/shipments/tracking/table"),
		Query:         state,
		Filters:       filters,
		Summary:       summary,
		Alerts:        alerts,
		Table:         table,
	}
}

// TrackingTablePayload converts TrackingResult into table-specific data.
func TrackingTablePayload(basePath string, state TrackingQueryState, result adminshipments.TrackingResult, errMsg string) TrackingTableData {
	rows := buildTrackingRows(basePath, result.Shipments)
	pagination := TrackingPagination{
		Page:     result.Pagination.Page,
		PageSize: result.Pagination.PageSize,
		Total:    result.Pagination.TotalItems,
		Next:     result.Pagination.NextPage,
		Prev:     result.Pagination.PrevPage,
		RawQuery: state.RawQuery,
	}

	return TrackingTableData{
		BasePath:     basePath,
		FragmentPath: joinBase(basePath, "/shipments/tracking/table"),
		Rows:         rows,
		EmptyMessage: "対象の配送は見つかりませんでした。",
		Error:        errMsg,
		Pagination:   pagination,
		AutoRefresh:  true,
		RefreshEvery: int(result.Summary.RefreshInterval.Seconds()),
		RawQuery:     state.RawQuery,
	}
}

func buildTrackingAlerts(alerts []adminshipments.TrackingAlert, basePath string) []TrackingAlertData {
	var payload []TrackingAlertData
	for _, alert := range alerts {
		payload = append(payload, TrackingAlertData{
			Label:       alert.Label,
			Description: alert.Description,
			Tone:        alert.Tone,
			ActionLabel: alert.ActionLabel,
			ActionURL:   joinBase(basePath, alert.ActionURL),
		})
	}
	return payload
}

func buildTrackingFilters(state TrackingQueryState, filters adminshipments.TrackingFilters) TrackingFilterData {
	var statuses []TrackingStatusChip
	for _, option := range filters.StatusOptions {
		value := string(option.Value)
		statuses = append(statuses, TrackingStatusChip{
			Value:  value,
			Label:  option.Label,
			Tone:   option.Tone,
			Count:  option.Count,
			Active: state.Status == value,
		})
	}

	mapSelect := func(options []adminshipments.SelectOption, active string) []TrackingSelectOption {
		var result []TrackingSelectOption
		for _, option := range options {
			result = append(result, TrackingSelectOption{
				Value:  option.Value,
				Label:  option.Label,
				Count:  option.Count,
				Active: strings.EqualFold(option.Value, active),
			})
		}
		return result
	}

	carriers := mapSelect(filters.CarrierOptions, state.Carrier)
	lanes := mapSelect(filters.LaneOptions, state.Lane)
	regions := mapSelect(filters.RegionOptions, state.Region)

	return TrackingFilterData{
		Statuses: statuses,
		Carriers: carriers,
		Lanes:    lanes,
		Regions:  regions,
		HasAny:   len(statuses) > 0 || len(carriers) > 0 || len(lanes) > 0 || len(regions) > 0,
	}
}

func buildTrackingSummary(summary adminshipments.TrackingSummary) TrackingSummaryData {
	refreshSeconds := int(summary.RefreshInterval.Seconds())
	if refreshSeconds <= 0 {
		refreshSeconds = 30
	}
	return TrackingSummaryData{
		ActiveLabel:    fmt.Sprintf("%d 件の配送が進行中", summary.ActiveShipments),
		DelayedLabel:   fmt.Sprintf("%d 件が遅延リスク", summary.Delayed),
		ExceptionLabel: fmt.Sprintf("%d 件が対応待ち", summary.Exceptions),
		LastRefresh:    summary.LastRefresh.Local().Format("15:04:05"),
		RefreshEvery:   fmt.Sprintf("自動更新: %ds", refreshSeconds),
		RefreshSeconds: refreshSeconds,
	}
}

func buildTrackingRows(basePath string, shipments []adminshipments.TrackingShipment) []TrackingRow {
	rows := make([]TrackingRow, 0, len(shipments))
	for _, shipment := range shipments {
		row := TrackingRow{
			ID:             shipment.ID,
			OrderNumber:    shipment.OrderNumber,
			OrderURL:       absoluteOrderURL(basePath, shipment),
			Customer:       shipment.CustomerName,
			Carrier:        shipment.CarrierLabel,
			Service:        shipment.ServiceLevel,
			StatusLabel:    shipment.StatusLabel,
			StatusTone:     shipment.StatusTone,
			TrackingNumber: shipment.TrackingNumber,
			Destination:    shipment.Destination,
			Region:         shipment.Region,
			Lane:           shipment.Lane,
			LastEvent:      shipment.LastEvent,
			LastEventTime:  helpers.Relative(shipment.LastEventAt),
			SLAStatus:      shipment.SLAStatus,
			SLATone:        shipment.SLATone,
			Exception:      shipment.ExceptionLabel,
			ExceptionTone:  shipment.ExceptionTone,
			BadgeIcon:      shipment.AlertIcon,
		}
		if shipment.EstimatedArrival != nil {
			row.ETA = helpers.Date(*shipment.EstimatedArrival, "01/02 15:04")
		}
		if shipment.DelayMinutes > 0 {
			row.DelayLabel = fmt.Sprintf("+%d分", shipment.DelayMinutes)
		}
		rows = append(rows, row)
	}
	return rows
}

func trackingBreadcrumbs(basePath string) []partials.Breadcrumb {
	return []partials.Breadcrumb{
		{Label: "ダッシュボード", Href: joinBase(basePath, "/")},
		{Label: "出荷モニタ", Href: joinBase(basePath, "/shipments/tracking")},
	}
}

func absoluteOrderURL(basePath string, shipment adminshipments.TrackingShipment) string {
	if strings.TrimSpace(shipment.OrderURL) != "" {
		return joinBase(basePath, shipment.OrderURL)
	}
	if shipment.OrderID == "" {
		return ""
	}
	path := fmt.Sprintf("/orders/%s?tab=shipments", url.PathEscape(shipment.OrderID))
	return joinBase(basePath, path)
}

func trackingFragmentURL(path, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return path
	}
	if strings.Contains(path, "?") {
		return path + "&" + raw
	}
	return path + "?" + raw
}
