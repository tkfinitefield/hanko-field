package production

import (
	"encoding/json"
	"fmt"
	"strings"

	adminproduction "finitefield.org/hanko-admin/internal/admin/production"
	"finitefield.org/hanko-admin/internal/admin/templates/helpers"
	"finitefield.org/hanko-admin/internal/admin/templates/partials"
)

const (
	boardContainerID = "production-board-shell"
	boardTargetID    = "production-board"
	drawerTargetID   = "production-drawer"
)

// PageData represents the SSR payload for the production kanban page.
type PageData struct {
	Title       string
	Description string
	Breadcrumbs []partials.Breadcrumb
	Error       string
	QueueForm   QueueSelector
	Summary     []SummaryChip
	Filters     FilterBar
	Board       BoardData
}

// QueryState captures the active filters for the board.
type QueryState struct {
	Queue       string
	Priority    string
	ProductLine string
	Workstation string
	Selected    string
	RawQuery    string
}

// QueueSelector powers the queue combobox in the header.
type QueueSelector struct {
	Endpoint string
	Options  []QueueOption
	Selected string
}

// QueueOption represents an individual queue choice.
type QueueOption struct {
	ID       string
	Label    string
	Sublabel string
	Load     string
	Active   bool
}

// SummaryChip renders a KPI chip.
type SummaryChip struct {
	Label   string
	Value   string
	Tone    string
	SubText string
	Icon    string
}

// FilterBar describes the swimlane filter controls.
type FilterBar struct {
	Endpoint     string
	ProductLines []FilterOption
	Priorities   []FilterOption
	Workstations []FilterOption
	HasActive    bool
	Query        QueryState
}

// FilterOption renders a selectable chip or option.
type FilterOption struct {
	Value  string
	Label  string
	Count  int
	Active bool
}

// BoardData powers the kanban columns + drawer fragment.
type BoardData struct {
	ContainerID      string
	BoardID          string
	DrawerID         string
	FragmentPath     string
	RefreshURL       string
	RefreshInterval  int
	Query            QueryState
	Queue            QueueMeta
	Summary          []SummaryChip
	Lanes            []LaneData
	Drawer           DrawerData
	Error            string
	LastUpdatedLabel string
}

// QueueMeta summarises the selected queue.
type QueueMeta struct {
	Name        string
	Description string
	Location    string
	Shift       string
	Notes       []string
}

// LaneData renders a single stage column.
type LaneData struct {
	Stage           string
	Label           string
	Description     string
	CapacityLabel   string
	CapacityPercent int
	SLALabel        string
	SLATone         string
	Cards           []CardData
	EmptyMessage    string
}

// CardData represents a card on the board.
type CardData struct {
	ID            string
	OrderNumber   string
	Customer      string
	PriorityLabel string
	PriorityTone  string
	ProductLine   string
	Design        string
	DueLabel      string
	DueTone       string
	Flags         []FlagData
	Assignees     []AssigneeData
	Workstation   string
	Stage         string
	StageLabel    string
	PreviewURL    string
	Notes         []string
	Blocked       bool
	BlockedReason string
	Payload       string
	Endpoint      string
	Selected      bool
}

// FlagData renders a badge on the card/drawer.
type FlagData struct {
	Label string
	Tone  string
	Icon  string
}

// AssigneeData renders assignee avatars.
type AssigneeData struct {
	Initials string
	Name     string
	Role     string
}

// DrawerData powers the side inspector.
type DrawerData struct {
	Empty    bool
	Card     DrawerCard
	Timeline []DrawerTimeline
	Details  []DrawerDetail
}

// DrawerCard summarises the selected card in the drawer.
type DrawerCard struct {
	ID            string
	OrderNumber   string
	Customer      string
	StageLabel    string
	StageTone     string
	PriorityLabel string
	PriorityTone  string
	ProductLine   string
	Queue         string
	Workstation   string
	PreviewURL    string
	DueLabel      string
	Notes         []string
	Flags         []FlagData
	Assignees     []AssigneeData
}

// DrawerTimeline renders production events in the drawer.
type DrawerTimeline struct {
	StageLabel  string
	Description string
	Timestamp   string
	Actor       string
	Tone        string
	Note        string
}

// DrawerDetail renders metadata rows.
type DrawerDetail struct {
	Label string
	Value string
}

// BuildPageData assembles the full page payload.
func BuildPageData(basePath string, state QueryState, result adminproduction.BoardResult, errMsg string) PageData {
	queues := buildQueueSelector(basePath, state, result.Queues)
	summary := buildSummaryChips(result.Summary)
	filters := buildFilterBar(basePath, state, result.Filters)
	board := BuildBoard(basePath, state, result, errMsg)

	return PageData{
		Title:       "制作カンバン",
		Description: "工房の進行状況をリアルタイムで追跡し、ステージ間のハンドオフを管理します。",
		Breadcrumbs: breadcrumbs(basePath),
		Error:       errMsg,
		QueueForm:   queues,
		Summary:     summary,
		Filters:     filters,
		Board:       board,
	}
}

// BuildBoard returns the kanban fragment payload.
func BuildBoard(basePath string, state QueryState, result adminproduction.BoardResult, errMsg string) BoardData {
	fragment := joinBase(basePath, "/production/queues/board")
	refreshURL := fragment
	if strings.TrimSpace(state.RawQuery) != "" {
		refreshURL = fmt.Sprintf("%s?%s", fragment, state.RawQuery)
	}

	lanes := buildLanes(basePath, state, result.Lanes)
	drawer := buildDrawer(result.Drawer)
	queue := buildQueueMeta(result.Queue)
	summary := buildSummaryChips(result.Summary)

	refreshInterval := int(result.RefreshInterval.Seconds())
	if refreshInterval <= 0 {
		refreshInterval = 30
	}

	lastUpdated := ""
	if !result.GeneratedAt.IsZero() {
		lastUpdated = result.GeneratedAt.Format("15:04")
	}

	return BoardData{
		ContainerID:      boardContainerID,
		BoardID:          boardTargetID,
		DrawerID:         drawerTargetID,
		FragmentPath:     fragment,
		RefreshURL:       refreshURL,
		RefreshInterval:  refreshInterval,
		Query:            state,
		Queue:            queue,
		Summary:          summary,
		Lanes:            lanes,
		Drawer:           drawer,
		Error:            errMsg,
		LastUpdatedLabel: lastUpdated,
	}
}

func buildQueueSelector(basePath string, state QueryState, options []adminproduction.QueueOption) QueueSelector {
	endpoint := joinBase(basePath, "/production/queues/board")
	selector := QueueSelector{Endpoint: endpoint, Selected: state.Queue}
	hasActive := false
	for _, opt := range options {
		active := opt.Active
		if !active && selector.Selected != "" {
			active = opt.ID == selector.Selected
		}
		if active {
			hasActive = true
			selector.Selected = opt.ID
		}
		selector.Options = append(selector.Options, QueueOption{
			ID:       opt.ID,
			Label:    opt.Label,
			Sublabel: opt.Sublabel,
			Load:     opt.Load,
			Active:   active,
		})
	}
	if !hasActive && len(selector.Options) > 0 {
		selector.Options[0].Active = true
		selector.Selected = selector.Options[0].ID
	}
	return selector
}

func buildSummaryChips(summary adminproduction.Summary) []SummaryChip {
	return []SummaryChip{
		{Label: "WIP", Value: fmt.Sprintf("%d", summary.TotalWIP), Tone: "info", SubText: "現在の制作数", Icon: "🛠"},
		{Label: "納期迫る", Value: fmt.Sprintf("%d", summary.DueSoon), Tone: "warning", SubText: "24時間以内", Icon: "⏱"},
		{Label: "ブロック", Value: fmt.Sprintf("%d", summary.Blocked), Tone: "danger", SubText: "要対応", Icon: "🚧"},
	}
}

func buildFilterBar(basePath string, state QueryState, filters adminproduction.FilterSummary) FilterBar {
	bar := FilterBar{
		Endpoint: joinBase(basePath, "/production/queues/board"),
		Query:    state,
	}
	for _, opt := range filters.ProductLines {
		bar.ProductLines = append(bar.ProductLines, FilterOption{
			Value:  opt.Value,
			Label:  opt.Label,
			Count:  opt.Count,
			Active: opt.Active,
		})
		if opt.Active {
			bar.HasActive = true
		}
	}
	for _, opt := range filters.Priorities {
		bar.Priorities = append(bar.Priorities, FilterOption{
			Value:  opt.Value,
			Label:  opt.Label,
			Count:  opt.Count,
			Active: opt.Active,
		})
		if opt.Active {
			bar.HasActive = true
		}
	}
	for _, opt := range filters.Workstations {
		bar.Workstations = append(bar.Workstations, FilterOption{
			Value:  opt.Value,
			Label:  opt.Label,
			Count:  opt.Count,
			Active: opt.Active,
		})
		if opt.Active {
			bar.HasActive = true
		}
	}
	return bar
}

func buildQueueMeta(queue adminproduction.Queue) QueueMeta {
	return QueueMeta{
		Name:        fallback(queue.Name, "工房"),
		Description: queue.Description,
		Location:    queue.Location,
		Shift:       queue.Shift,
		Notes:       append([]string(nil), queue.Notes...),
	}
}

func buildLanes(basePath string, state QueryState, lanes []adminproduction.Lane) []LaneData {
	result := make([]LaneData, 0, len(lanes))
	active := strings.TrimSpace(state.Selected)
	for _, lane := range lanes {
		data := LaneData{
			Stage:         string(lane.Stage),
			Label:         fallback(lane.Label, adminproduction.StageLabel(lane.Stage)),
			Description:   lane.Description,
			CapacityLabel: fmt.Sprintf("%d/%d", lane.Capacity.Used, max(1, lane.Capacity.Limit)),
			SLALabel:      lane.SLA.Label,
			SLATone:       lane.SLA.Tone,
		}
		if lane.Capacity.Limit > 0 {
			data.CapacityPercent = (lane.Capacity.Used * 100) / lane.Capacity.Limit
		}
		data.Cards = buildCards(basePath, lane.Cards, active)
		if len(data.Cards) == 0 {
			data.EmptyMessage = "カードがありません"
		}
		result = append(result, data)
	}
	return result
}

func buildCards(basePath string, cards []adminproduction.Card, selected string) []CardData {
	result := make([]CardData, 0, len(cards))
	endpointBase := joinBase(basePath, "/orders")
	active := strings.TrimSpace(selected)
	for _, card := range cards {
		payload := buildCardPayload(card)
		cardData := CardData{
			ID:            card.ID,
			OrderNumber:   card.OrderNumber,
			Customer:      card.Customer,
			PriorityLabel: fallback(card.PriorityLabel, "通常"),
			PriorityTone:  fallback(card.PriorityTone, "info"),
			ProductLine:   card.ProductLine,
			Design:        card.Design,
			DueLabel:      card.DueLabel,
			DueTone:       card.DueTone,
			Flags:         buildFlags(card.Flags),
			Assignees:     buildAssignees(card.Assignees),
			Workstation:   card.Workstation,
			Stage:         string(card.Stage),
			StageLabel:    adminproduction.StageLabel(card.Stage),
			PreviewURL:    card.PreviewURL,
			Notes:         append([]string(nil), card.Notes...),
			Blocked:       card.Blocked,
			BlockedReason: card.BlockedReason,
			Payload:       payload,
			Endpoint:      fmt.Sprintf("%s/%s/production-events", endpointBase, card.ID),
			Selected:      card.ID == active,
		}
		if cardData.Selected {
			active = card.ID
		}
		result = append(result, cardData)
	}
	if active == "" && len(result) > 0 {
		result[0].Selected = true
	}
	return result
}

func buildFlags(flags []adminproduction.CardFlag) []FlagData {
	result := make([]FlagData, 0, len(flags))
	for _, flag := range flags {
		result = append(result, FlagData{Label: flag.Label, Tone: flag.Tone, Icon: flag.Icon})
	}
	return result
}

func buildAssignees(assignees []adminproduction.Assignee) []AssigneeData {
	result := make([]AssigneeData, 0, len(assignees))
	for _, assignee := range assignees {
		result = append(result, AssigneeData{Initials: assignee.Initials, Name: assignee.Name, Role: assignee.Role})
	}
	return result
}

func buildDrawer(drawer adminproduction.Drawer) DrawerData {
	if drawer.Empty {
		return DrawerData{Empty: true}
	}
	card := drawer.Card
	data := DrawerData{
		Card: DrawerCard{
			ID:            card.ID,
			OrderNumber:   card.OrderNumber,
			Customer:      card.Customer,
			StageLabel:    card.StageLabel,
			StageTone:     "info",
			PriorityLabel: card.PriorityLabel,
			PriorityTone:  card.PriorityTone,
			ProductLine:   card.ProductLine,
			Queue:         card.QueueName,
			Workstation:   card.Workstation,
			PreviewURL:    card.PreviewURL,
			DueLabel:      card.DueLabel,
			Notes:         append([]string(nil), card.Notes...),
			Flags:         buildFlags(card.Flags),
			Assignees:     buildAssignees(card.Assignees),
		},
	}
	for _, row := range drawer.Details {
		data.Details = append(data.Details, DrawerDetail{Label: row.Label, Value: row.Value})
	}
	for _, event := range drawer.Timeline {
		data.Timeline = append(data.Timeline, DrawerTimeline{
			StageLabel:  event.StageLabel,
			Description: event.Description,
			Timestamp:   helpers.Date(event.OccurredAt, "2006-01-02 15:04"),
			Actor:       fallback(event.Actor, "工房"),
			Tone:        fallback(event.Tone, "info"),
			Note:        event.Note,
		})
	}
	return data
}

func buildCardPayload(card adminproduction.Card) string {
	type payload struct {
		ID           string                            `json:"id"`
		OrderNumber  string                            `json:"orderNumber"`
		Customer     string                            `json:"customer"`
		Priority     string                            `json:"priorityLabel"`
		PriorityTone string                            `json:"priorityTone"`
		Stage        string                            `json:"stage"`
		StageLabel   string                            `json:"stageLabel"`
		ProductLine  string                            `json:"productLine"`
		Queue        string                            `json:"queue"`
		Workstation  string                            `json:"workstation"`
		Notes        []string                          `json:"notes"`
		Flags        []FlagData                        `json:"flags"`
		Assignees    []AssigneeData                    `json:"assignees"`
		Timeline     []adminproduction.ProductionEvent `json:"timeline"`
		DueLabel     string                            `json:"dueLabel"`
	}

	body := payload{
		ID:           card.ID,
		OrderNumber:  card.OrderNumber,
		Customer:     card.Customer,
		Priority:     card.PriorityLabel,
		PriorityTone: card.PriorityTone,
		Stage:        string(card.Stage),
		StageLabel:   adminproduction.StageLabel(card.Stage),
		ProductLine:  card.ProductLine,
		Queue:        card.QueueName,
		Workstation:  card.Workstation,
		Notes:        append([]string(nil), card.Notes...),
		Flags:        buildFlags(card.Flags),
		Assignees:    buildAssignees(card.Assignees),
		Timeline:     append([]adminproduction.ProductionEvent(nil), card.Timeline...),
		DueLabel:     card.DueLabel,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func breadcrumbs(base string) []partials.Breadcrumb {
	return []partials.Breadcrumb{
		{Label: "受注管理", Href: joinBase(base, "/orders")},
		{Label: "制作カンバン", Href: joinBase(base, "/production/queues")},
	}
}

func joinBase(base, suffix string) string {
	if strings.TrimSpace(base) == "" {
		base = "/admin"
	}
	if !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}
	if base == "/" {
		return suffix
	}
	return strings.TrimRight(base, "/") + suffix
}

func fallback(value, alt string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return alt
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
