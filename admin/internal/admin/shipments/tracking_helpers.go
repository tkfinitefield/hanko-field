package shipments

import (
	"fmt"
	"strings"
	"time"
)

const (
	slaDelayWarningMinutes = 60
	slaDelayBreachMinutes  = 180
)

func filterTrackingShipments(shipments []TrackingShipment, query TrackingQuery) []TrackingShipment {
	status := strings.TrimSpace(string(query.Status))
	carrier := strings.TrimSpace(query.Carrier)
	lane := strings.TrimSpace(query.Lane)
	region := strings.TrimSpace(query.Destination)
	delay := strings.TrimSpace(query.DelayWindow)

	var filtered []TrackingShipment
	for _, shipment := range shipments {
		if status != "" && string(shipment.Status) != status {
			continue
		}
		if carrier != "" && !strings.EqualFold(shipment.Carrier, carrier) {
			continue
		}
		if lane != "" && shipment.Lane != lane {
			continue
		}
		if region != "" && !strings.EqualFold(shipment.Region, region) {
			continue
		}
		if delay != "" {
			switch delay {
			case "breach":
				if shipment.SLATone != "danger" {
					continue
				}
			case "delayed":
				if shipment.DelayMinutes < 30 && shipment.SLATone != "warning" {
					continue
				}
			}
		}
		filtered = append(filtered, shipment)
	}
	return filtered
}

func paginateTrackingShipments(shipments []TrackingShipment, page, pageSize int) ([]TrackingShipment, Pagination) {
	total := len(shipments)
	if pageSize <= 0 {
		pageSize = 20
	}
	if page < 1 {
		page = 1
	}

	start := (page - 1) * pageSize
	if start >= total {
		start = 0
		page = 1
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	paged := append([]TrackingShipment(nil), shipments[start:end]...)

	var next, prev *int
	if end < total {
		nextPage := page + 1
		next = &nextPage
	}
	if start > 0 {
		prevPage := page - 1
		if prevPage < 1 {
			prevPage = 1
		}
		prev = &prevPage
	}

	return paged, Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		NextPage:   next,
		PrevPage:   prev,
	}
}

func trackingSummary(filtered []TrackingShipment, lastRefresh time.Time, refreshInterval time.Duration) TrackingSummary {
	if lastRefresh.IsZero() {
		lastRefresh = time.Now()
	}
	if refreshInterval <= 0 {
		refreshInterval = 30 * time.Second
	}
	return TrackingSummary{
		ActiveShipments: countActiveShipments(filtered),
		Delayed:         countDelayedShipments(filtered),
		Exceptions:      countExceptionShipments(filtered),
		LastRefresh:     lastRefresh,
		RefreshInterval: refreshInterval,
	}
}

func cloneTrackingFilters(filters TrackingFilters) TrackingFilters {
	return TrackingFilters{
		StatusOptions:  append([]TrackingStatusOption(nil), filters.StatusOptions...),
		CarrierOptions: append([]SelectOption(nil), filters.CarrierOptions...),
		LaneOptions:    append([]SelectOption(nil), filters.LaneOptions...),
		RegionOptions:  append([]SelectOption(nil), filters.RegionOptions...),
	}
}

func normalizeTrackingShipment(sh TrackingShipment) TrackingShipment {
	if sh.ID == "" {
		sh.ID = firstNonEmpty(sh.TrackingNumber, sh.OrderID)
	}
	if sh.CarrierLabel == "" {
		sh.CarrierLabel = carrierLabel(sh.Carrier)
	}
	if sh.StatusLabel == "" {
		sh.StatusLabel = trackingStatusLabel(sh.Status)
	}
	if sh.StatusTone == "" {
		sh.StatusTone = trackingStatusTone(sh.Status)
	}
	if sh.SLAStatus == "" || sh.SLATone == "" {
		status, tone := deriveSLAFromShipment(sh)
		sh.SLAStatus = status
		sh.SLATone = tone
	}
	if sh.ExceptionLabel == "" && sh.Status == TrackingStatusException {
		sh.ExceptionLabel = "要対応"
	}
	if sh.ExceptionTone == "" && sh.ExceptionLabel != "" {
		sh.ExceptionTone = "danger"
	}
	if sh.AlertIcon == "" {
		switch sh.ExceptionTone {
		case "danger":
			sh.AlertIcon = "⚠️"
		default:
			if sh.SLATone == "warning" {
				sh.AlertIcon = "🚨"
			}
		}
	}
	if sh.OrderURL == "" && sh.OrderID != "" {
		sh.OrderURL = fmt.Sprintf("/admin/orders/%s?tab=shipments", sh.OrderID)
	}
	return sh
}

func deriveSLAFromShipment(sh TrackingShipment) (string, string) {
	switch {
	case sh.Status == TrackingStatusDelivered:
		return "完了", "success"
	case sh.Status == TrackingStatusException:
		return "SLA逸脱", "danger"
	case sh.DelayMinutes >= slaDelayBreachMinutes:
		return "SLA逸脱", "danger"
	case sh.DelayMinutes >= slaDelayWarningMinutes:
		return "遅延リスク", "warning"
	default:
		return "SLA内", "success"
	}
}

func trackingStatusLabel(status TrackingStatus) string {
	switch status {
	case TrackingStatusLabelCreated:
		return "集荷待ち"
	case TrackingStatusInTransit:
		return "中継輸送中"
	case TrackingStatusOutForDelivery:
		return "配達中"
	case TrackingStatusDelivered:
		return "配達完了"
	case TrackingStatusException:
		return "要対応"
	default:
		return string(status)
	}
}

func trackingStatusTone(status TrackingStatus) string {
	switch status {
	case TrackingStatusLabelCreated:
		return "slate"
	case TrackingStatusInTransit:
		return "info"
	case TrackingStatusOutForDelivery:
		return "warning"
	case TrackingStatusDelivered:
		return "success"
	case TrackingStatusException:
		return "danger"
	default:
		return "slate"
	}
}
