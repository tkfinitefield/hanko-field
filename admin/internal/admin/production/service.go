package production

import (
	"context"
	"errors"
	"time"
)

// Service exposes production queue data and workflows for the admin UI.
type Service interface {
	// Board returns the current state of the selected production queue with filters applied.
	Board(ctx context.Context, token string, query BoardQuery) (BoardResult, error)
	// AppendEvent appends a production workflow event for the specified order/card.
	AppendEvent(ctx context.Context, token, orderID string, req AppendEventRequest) (AppendEventResult, error)
}

var (
	// ErrQueueNotFound indicates the requested queue does not exist.
	ErrQueueNotFound = errors.New("production queue not found")
	// ErrCardNotFound indicates the requested order/card is unknown.
	ErrCardNotFound = errors.New("production order not found")
	// ErrStageInvalid indicates the requested stage is unsupported.
	ErrStageInvalid = errors.New("production stage is invalid")
)

// Stage represents a workflow step on the production board.
type Stage string

const (
	StageQueued    Stage = "queued"
	StageEngraving Stage = "engraving"
	StagePolishing Stage = "polishing"
	StageQC        Stage = "qc"
	StagePacked    Stage = "packed"
)

// Priority represents the urgency of a card.
type Priority string

const (
	PriorityNormal Priority = "normal"
	PriorityRush   Priority = "rush"
	PriorityHold   Priority = "hold"
)

// BoardQuery captures filters applied to the kanban board.
type BoardQuery struct {
	QueueID     string
	Priority    string
	ProductLine string
	Workstation string
	Selected    string
}

// BoardResult describes the production board snapshot rendered for the UI.
type BoardResult struct {
	Queue           Queue
	Queues          []QueueOption
	Summary         Summary
	Filters         FilterSummary
	Lanes           []Lane
	Drawer          Drawer
	SelectedCardID  string
	GeneratedAt     time.Time
	RefreshInterval time.Duration
}

// Queue provides metadata about a specific production queue/workshop.
type Queue struct {
	ID            string
	Name          string
	Description   string
	Location      string
	Shift         string
	Capacity      int
	Load          int
	Utilisation   float64
	LeadTimeHours int
	Notes         []string
}

// QueueOption powers the queue selector combobox.
type QueueOption struct {
	ID       string
	Label    string
	Sublabel string
	Load     string
	Active   bool
}

// Summary aggregates WIP metrics for the current filters.
type Summary struct {
	TotalWIP     int
	DueSoon      int
	Blocked      int
	AvgLeadHours int
	Utilisation  int
	UpdatedAt    time.Time
}

// FilterSummary enumerates available filter options per facet.
type FilterSummary struct {
	ProductLines []FilterOption
	Priorities   []FilterOption
	Workstations []FilterOption
}

// FilterOption represents a selectable filter chip/option.
type FilterOption struct {
	Value  string
	Label  string
	Count  int
	Active bool
}

// Lane represents a single stage column on the board.
type Lane struct {
	Stage       Stage
	Label       string
	Description string
	Capacity    LaneCapacity
	SLA         SLAMeta
	Cards       []Card
}

// LaneCapacity reports usage/limit stats for the column.
type LaneCapacity struct {
	Used  int
	Limit int
}

// SLAMeta reflects SLA/aging info per stage.
type SLAMeta struct {
	Label string
	Tone  string
}

// Card models a single order card on the board.
type Card struct {
	ID            string
	OrderNumber   string
	Stage         Stage
	Priority      Priority
	PriorityLabel string
	PriorityTone  string
	Customer      string
	ProductLine   string
	Design        string
	PreviewURL    string
	PreviewAlt    string
	QueueID       string
	QueueName     string
	Workstation   string
	Assignees     []Assignee
	Flags         []CardFlag
	DueAt         time.Time
	DueLabel      string
	DueTone       string
	Notes         []string
	Blocked       bool
	BlockedReason string
	AgingHours    int
	LastEvent     ProductionEvent
	Timeline      []ProductionEvent
}

// CardFlag highlights blockers/warnings on a card.
type CardFlag struct {
	Label string
	Tone  string
	Icon  string
}

// Assignee lists operators currently owning the card.
type Assignee struct {
	Name      string
	AvatarURL string
	Initials  string
	Role      string
}

// Drawer contains data for the detail inspector panel.
type Drawer struct {
	Empty    bool
	Card     DrawerCard
	Timeline []ProductionEvent
	Details  []DrawerDetail
}

// DrawerCard summarises the selected card inside the drawer.
type DrawerCard struct {
	ID            string
	OrderNumber   string
	Customer      string
	PriorityLabel string
	PriorityTone  string
	Stage         Stage
	StageLabel    string
	ProductLine   string
	QueueName     string
	Workstation   string
	PreviewURL    string
	PreviewAlt    string
	DueLabel      string
	Notes         []string
	Flags         []CardFlag
	Assignees     []Assignee
	LastUpdated   time.Time
}

// DrawerDetail renders supplemental metadata rows.
type DrawerDetail struct {
	Label string
	Value string
}

// ProductionEvent stores timeline events for a card.
type ProductionEvent struct {
	ID          string
	Stage       Stage
	StageLabel  string
	Type        string
	Description string
	Actor       string
	ActorAvatar string
	Station     string
	Tone        string
	OccurredAt  time.Time
	Note        string
}

// AppendEventRequest captures inputs when changing a card stage via DnD.
type AppendEventRequest struct {
	Stage    Stage
	Note     string
	Station  string
	ActorID  string
	ActorRef string
}

// AppendEventResult returns the persisted event and updated card snapshot.
type AppendEventResult struct {
	Event ProductionEvent
	Card  Card
}

// StageLabel returns a japanese-friendly label for the stage.
func StageLabel(stage Stage) string {
	switch stage {
	case StageQueued:
		return "待機"
	case StageEngraving:
		return "刻印"
	case StagePolishing:
		return "研磨"
	case StageQC:
		return "検品"
	case StagePacked:
		return "梱包"
	default:
		return string(stage)
	}
}
