package production

import (
	"fmt"
	"strings"

	adminproduction "finitefield.org/hanko-admin/internal/admin/production"
	"finitefield.org/hanko-admin/internal/admin/templates/helpers"
	"finitefield.org/hanko-admin/internal/admin/templates/partials"
)

// WorkOrderPageData represents the SSR payload for the work order detail view.
type WorkOrderPageData struct {
	Title          string
	Breadcrumbs    []partials.Breadcrumb
	Header         WorkOrderHeader
	Overview       WorkOrderOverview
	Assets         []WorkOrderAssetCard
	Instructions   []WorkInstructionCard
	Checklist      []WorkOrderChecklistItem
	Safety         []WorkOrderNoticeCard
	Activity       []WorkOrderTimelineItem
	Tabs           []WorkOrderTab
	ActionEndpoint string
	OrderID        string
	PrimaryAction  *WorkOrderAction
}

// WorkOrderHeader powers the summary card at the top of the page.
type WorkOrderHeader struct {
	OrderNumber   string
	StageLabel    string
	StageTone     string
	PriorityLabel string
	PriorityTone  string
	Queue         string
	Responsible   string
	DueLabel      string
	DueTone       string
	PrintURL      string
	LastPrinted   string
	Customer      string
	ProductLine   string
}

// WorkOrderOverview renders materials and notes.
type WorkOrderOverview struct {
	Customer     string
	ProductLine  string
	Design       string
	Workstation  string
	Notes        []string
	CustomerNote string
	Materials    []WorkOrderMaterialRow
}

// WorkOrderMaterialRow details a material requirement.
type WorkOrderMaterialRow struct {
	Name     string
	Detail   string
	Quantity string
	Source   string
	Status   string
}

// WorkOrderAssetCard renders media cards for assets.
type WorkOrderAssetCard struct {
	ID          string
	Name        string
	Kind        string
	PreviewURL  string
	DownloadURL string
	Size        string
	UpdatedAt   string
	Description string
}

// WorkInstructionCard describes an instruction panel entry.
type WorkInstructionCard struct {
	ID          string
	Title       string
	Description string
	StageLabel  string
	Duration    string
	Tools       []string
}

// WorkOrderChecklistItem powers action buttons for progressing stages.
type WorkOrderChecklistItem struct {
	ID          string
	Label       string
	Description string
	Stage       string
	StageLabel  string
	Completed   bool
	CompletedAt string
}

// WorkOrderNoticeCard renders inline notices.
type WorkOrderNoticeCard struct {
	Icon  string
	Tone  string
	Body  string
	Title string
}

// WorkOrderTimelineItem lists activity log entries.
type WorkOrderTimelineItem struct {
	StageLabel  string
	Description string
	Actor       string
	Timestamp   string
	Tone        string
	Note        string
}

// WorkOrderTab configures the underline tabs.
type WorkOrderTab struct {
	ID     string
	Label  string
	Href   string
	Active bool
}

// WorkOrderAction represents the CTA used for quick stage updates.
type WorkOrderAction struct {
	Label      string
	Stage      string
	StageLabel string
}

// BuildWorkOrderPage converts the domain work order into template payloads.
func BuildWorkOrderPage(basePath string, work adminproduction.WorkOrder) WorkOrderPageData {
	card := work.Card
	title := fmt.Sprintf("作業指示書 #%s", fallback(card.OrderNumber, card.ID))
	header := buildWorkOrderHeader(work)
	overview := buildWorkOrderOverview(work)
	assets := buildWorkOrderAssets(work)
	instructions := buildWorkInstructions(work)
	checklist := buildWorkChecklist(work)
	notices := buildWorkNotices(work)
	activity := buildWorkTimeline(work)
	tabs := []WorkOrderTab{
		{ID: "overview", Label: "概要", Href: "#overview", Active: true},
		{ID: "assets", Label: "デザイン資産", Href: "#assets"},
		{ID: "instructions", Label: "作業手順", Href: "#instructions"},
		{ID: "activity", Label: "アクティビティ", Href: "#activity"},
	}

	actionEndpoint := joinBase(basePath, fmt.Sprintf("/orders/%s/production-events", card.ID))

	return WorkOrderPageData{
		Title:          title,
		Breadcrumbs:    workOrderBreadcrumbs(basePath, card),
		Header:         header,
		Overview:       overview,
		Assets:         assets,
		Instructions:   instructions,
		Checklist:      checklist,
		Safety:         notices,
		Activity:       activity,
		Tabs:           tabs,
		ActionEndpoint: actionEndpoint,
		OrderID:        card.ID,
		PrimaryAction:  nextChecklistAction(checklist),
	}
}

func buildWorkOrderHeader(work adminproduction.WorkOrder) WorkOrderHeader {
	card := work.Card
	return WorkOrderHeader{
		OrderNumber:   fallback(card.OrderNumber, card.ID),
		StageLabel:    adminproduction.StageLabel(card.Stage),
		StageTone:     stageTone(card.Stage),
		PriorityLabel: fallback(card.PriorityLabel, "通常"),
		PriorityTone:  fallback(card.PriorityTone, "info"),
		Queue:         fallback(card.QueueName, "制作ライン"),
		Responsible:   fallback(work.ResponsibleTeam, card.QueueName),
		DueLabel:      card.DueLabel,
		DueTone:       fallback(card.DueTone, "info"),
		PrintURL:      work.PDFURL,
		LastPrinted:   helpers.Relative(work.LastPrintedAt),
		Customer:      card.Customer,
		ProductLine:   card.ProductLine,
	}
}

func buildWorkOrderOverview(work adminproduction.WorkOrder) WorkOrderOverview {
	card := work.Card
	rows := make([]WorkOrderMaterialRow, 0, len(work.Materials))
	for _, material := range work.Materials {
		rows = append(rows, WorkOrderMaterialRow{
			Name:     material.Name,
			Detail:   material.Detail,
			Quantity: material.Quantity,
			Source:   material.Source,
			Status:   material.Status,
		})
	}
	return WorkOrderOverview{
		Customer:     card.Customer,
		ProductLine:  card.ProductLine,
		Design:       card.Design,
		Workstation:  card.Workstation,
		Notes:        append([]string(nil), card.Notes...),
		CustomerNote: strings.TrimSpace(work.CustomerNote),
		Materials:    rows,
	}
}

func buildWorkOrderAssets(work adminproduction.WorkOrder) []WorkOrderAssetCard {
	if len(work.Assets) == 0 {
		return nil
	}
	result := make([]WorkOrderAssetCard, 0, len(work.Assets))
	for _, asset := range work.Assets {
		result = append(result, WorkOrderAssetCard{
			ID:          asset.ID,
			Name:        asset.Name,
			Kind:        asset.Kind,
			PreviewURL:  asset.PreviewURL,
			DownloadURL: asset.DownloadURL,
			Size:        asset.Size,
			UpdatedAt:   helpers.Relative(asset.UpdatedAt),
			Description: asset.Description,
		})
	}
	return result
}

func buildWorkInstructions(work adminproduction.WorkOrder) []WorkInstructionCard {
	if len(work.Instructions) == 0 {
		return nil
	}
	result := make([]WorkInstructionCard, 0, len(work.Instructions))
	for _, inst := range work.Instructions {
		result = append(result, WorkInstructionCard{
			ID:          inst.ID,
			Title:       inst.Title,
			Description: inst.Description,
			StageLabel:  fallback(inst.StageLabel, adminproduction.StageLabel(inst.Stage)),
			Duration:    inst.Duration,
			Tools:       append([]string(nil), inst.Tools...),
		})
	}
	return result
}

func buildWorkChecklist(work adminproduction.WorkOrder) []WorkOrderChecklistItem {
	if len(work.Checklist) == 0 {
		return nil
	}
	items := make([]WorkOrderChecklistItem, 0, len(work.Checklist))
	for _, entry := range work.Checklist {
		items = append(items, WorkOrderChecklistItem{
			ID:          entry.ID,
			Label:       entry.Label,
			Description: entry.Description,
			Stage:       string(entry.Stage),
			StageLabel:  fallback(entry.StageLabel, adminproduction.StageLabel(entry.Stage)),
			Completed:   entry.Completed,
			CompletedAt: helpers.Relative(entry.CompletedAt),
		})
	}
	return items
}

func nextChecklistAction(items []WorkOrderChecklistItem) *WorkOrderAction {
	for _, item := range items {
		if !item.Completed {
			return &WorkOrderAction{
				Label:      fmt.Sprintf("%s完了", item.StageLabel),
				Stage:      item.Stage,
				StageLabel: item.StageLabel,
			}
		}
	}
	if len(items) == 0 {
		return nil
	}
	last := items[len(items)-1]
	return &WorkOrderAction{
		Label:      fmt.Sprintf("%sを更新", last.StageLabel),
		Stage:      last.Stage,
		StageLabel: last.StageLabel,
	}
}

func buildWorkNotices(work adminproduction.WorkOrder) []WorkOrderNoticeCard {
	if len(work.Safety) == 0 {
		return nil
	}
	alerts := make([]WorkOrderNoticeCard, 0, len(work.Safety))
	for _, notice := range work.Safety {
		alerts = append(alerts, WorkOrderNoticeCard{
			Icon:  notice.Icon,
			Tone:  fallback(notice.Tone, "info"),
			Body:  notice.Body,
			Title: notice.Title,
		})
	}
	return alerts
}

func buildWorkTimeline(work adminproduction.WorkOrder) []WorkOrderTimelineItem {
	if len(work.Activity) == 0 {
		return nil
	}
	entries := make([]WorkOrderTimelineItem, 0, len(work.Activity))
	for _, event := range work.Activity {
		entries = append(entries, WorkOrderTimelineItem{
			StageLabel:  fallback(event.StageLabel, adminproduction.StageLabel(event.Stage)),
			Description: event.Description,
			Actor:       fallback(event.Actor, "システム"),
			Timestamp:   helpers.Relative(event.OccurredAt),
			Tone:        fallback(event.Tone, "muted"),
			Note:        event.Note,
		})
	}
	return entries
}

func workOrderBreadcrumbs(base string, card adminproduction.Card) []partials.Breadcrumb {
	crumbs := breadcrumbs(base)
	orderPath := joinBase(base, fmt.Sprintf("/orders/%s", card.ID))
	crumbs = append(crumbs, partials.Breadcrumb{
		Label: fmt.Sprintf("注文 #%s", fallback(card.OrderNumber, card.ID)),
		Href:  orderPath,
	})
	crumbs = append(crumbs, partials.Breadcrumb{Label: "作業指示書"})
	return crumbs
}

func stageTone(stage adminproduction.Stage) string {
	switch stage {
	case adminproduction.StageEngraving:
		return "info"
	case adminproduction.StagePolishing:
		return "warning"
	case adminproduction.StageQC:
		return "success"
	case adminproduction.StagePacked:
		return "muted"
	default:
		return "muted"
	}
}
