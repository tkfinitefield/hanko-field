package production

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// StaticService provides deterministic production data for local development and tests.
type StaticService struct {
	mu           sync.RWMutex
	queues       map[string]Queue
	cards        map[string]*cardRecord
	laneDefs     []laneDefinition
	defaultQueue string
}

type cardRecord struct {
	card     Card
	timeline []ProductionEvent
}

type counter map[string]int

type laneDefinition struct {
	stage       Stage
	label       string
	description string
	capacity    int
	slaLabel    string
	slaTone     string
}

// NewStaticService returns a production service seeded with representative data.
func NewStaticService() *StaticService {
	svc := &StaticService{
		queues: make(map[string]Queue),
		cards:  make(map[string]*cardRecord),
		laneDefs: []laneDefinition{
			{stage: StageQueued, label: "待機", description: "支給待ち / 図面確認", capacity: 10, slaLabel: "平均6h", slaTone: "info"},
			{stage: StageEngraving, label: "刻印", description: "CNC + ハンドエングレーブ", capacity: 8, slaLabel: "平均9h", slaTone: "info"},
			{stage: StagePolishing, label: "研磨", description: "仕上げ・石留め調整", capacity: 8, slaLabel: "平均5h", slaTone: "warning"},
			{stage: StageQC, label: "検品", description: "寸法/SLA チェック", capacity: 6, slaLabel: "平均3h", slaTone: "success"},
			{stage: StagePacked, label: "梱包", description: "付属品セット / 梱包", capacity: 6, slaLabel: "平均2h", slaTone: "success"},
		},
	}
	svc.seed()
	return svc
}

// Board implements Service.
func (s *StaticService) Board(_ context.Context, _ string, query BoardQuery) (BoardResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	queueID := strings.TrimSpace(query.QueueID)
	if queueID == "" {
		queueID = s.defaultQueue
	}

	queue, ok := s.queues[queueID]
	if !ok {
		return BoardResult{}, ErrQueueNotFound
	}

	allRecords := s.queueRecords(queueID)
	filtered := filterRecords(allRecords, query)

	lanes := s.buildLanes(filtered)
	summary := s.buildSummary(queue, filtered)
	filters := s.buildFilters(allRecords, query)
	queueOptions := s.queueOptions(queueID)
	selectedID, drawer := s.buildDrawer(filtered, query.Selected)

	return BoardResult{
		Queue:           queue,
		Queues:          queueOptions,
		Summary:         summary,
		Filters:         filters,
		Lanes:           lanes,
		Drawer:          drawer,
		SelectedCardID:  selectedID,
		GeneratedAt:     time.Now(),
		RefreshInterval: 30 * time.Second,
	}, nil
}

// AppendEvent implements Service.
func (s *StaticService) AppendEvent(_ context.Context, _ string, orderID string, req AppendEventRequest) (AppendEventResult, error) {
	stage := Stage(strings.TrimSpace(string(req.Stage)))
	if !isValidStage(stage) {
		return AppendEventResult{}, ErrStageInvalid
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.cards[strings.TrimSpace(orderID)]
	if !ok {
		return AppendEventResult{}, ErrCardNotFound
	}

	now := time.Now()
	event := ProductionEvent{
		ID:          fmt.Sprintf("evt-%s-%d", record.card.ID, now.UnixNano()),
		Stage:       stage,
		StageLabel:  StageLabel(stage),
		Type:        fmt.Sprintf("%s.progress", stage),
		Description: fmt.Sprintf("%s へ移動", StageLabel(stage)),
		Actor:       coalesce(req.ActorRef, "工房オペレーター"),
		Station:     coalesce(req.Station, record.card.Workstation),
		Tone:        "info",
		OccurredAt:  now,
		Note:        strings.TrimSpace(req.Note),
	}
	record.timeline = append([]ProductionEvent{event}, record.timeline...)

	record.card.Stage = stage
	record.card.LastEvent = event
	record.card.Workstation = event.Station
	record.card.Blocked = false
	record.card.BlockedReason = ""
	record.card.Notes = appendUnique(record.card.Notes, event.Note)
	record.card.Timeline = append([]ProductionEvent(nil), record.timeline...)

	return AppendEventResult{
		Event: event,
		Card:  cloneCard(record.card),
	}, nil
}

func (s *StaticService) seed() {
	now := time.Now()

	s.queues["atelier-aoyama"] = Queue{
		ID:            "atelier-aoyama",
		Name:          "青山アトリエ",
		Description:   "リング刻印ライン / 表参道工房",
		Location:      "東京都港区",
		Shift:         "08:00-22:00",
		Capacity:      28,
		Load:          0,
		Utilisation:   58,
		LeadTimeHours: 36,
		Notes:         []string{"VIP優先ライン常設", "CNC 2台 + レーザー1台"},
	}
	s.queues["atelier-kyoto"] = Queue{
		ID:            "atelier-kyoto",
		Name:          "京都スタジオ",
		Description:   "和彫り / 仕上げ特化ライン",
		Location:      "京都府京都市",
		Shift:         "09:00-19:00",
		Capacity:      18,
		Load:          0,
		Utilisation:   44,
		LeadTimeHours: 40,
		Notes:         []string{"彫金士3名常駐", "QC 兼任体制"},
	}
	s.defaultQueue = "atelier-aoyama"

	cards := []*cardRecord{
		newCardRecord(Card{
			ID:            "order-1052",
			OrderNumber:   "1052",
			Stage:         StageEngraving,
			Priority:      PriorityRush,
			PriorityLabel: "特急",
			PriorityTone:  "warning",
			Customer:      "長谷川 純",
			ProductLine:   "Classic Ring",
			Design:        "18K カスタム刻印リング",
			PreviewURL:    "/public/static/previews/ring-classic.png",
			PreviewAlt:    "Classic Ring Preview",
			QueueID:       "atelier-aoyama",
			QueueName:     "青山アトリエ",
			Workstation:   "CNC-02",
			Assignees: []Assignee{
				{Name: "木村 遼", Initials: "RK", Role: "刻印"},
				{Name: "星野 彩", Initials: "AH", Role: "段取り"},
			},
			Flags:      []CardFlag{{Label: "VIP", Tone: "info", Icon: "👑"}},
			DueAt:      now.Add(20 * time.Hour),
			DueLabel:   "残り20時間",
			DueTone:    "warning",
			Notes:      []string{"フォント: S-12", "ダイヤ加飾"},
			Blocked:    false,
			AgingHours: 18,
		}, []ProductionEvent{
			{ID: "evt-1052-1", Stage: StageQueued, StageLabel: StageLabel(StageQueued), Type: "queued", Description: "支給待ち", Actor: "自動割当", OccurredAt: now.Add(-26 * time.Hour)},
			{ID: "evt-1052-2", Stage: StageEngraving, StageLabel: StageLabel(StageEngraving), Type: "engraving.start", Description: "刻印開始", Actor: "木村 遼", Station: "CNC-02", OccurredAt: now.Add(-2 * time.Hour)},
		}),
		newCardRecord(Card{
			ID:            "order-1060",
			OrderNumber:   "1060",
			Stage:         StageQueued,
			Priority:      PriorityNormal,
			PriorityLabel: "通常",
			PriorityTone:  "info",
			Customer:      "山本 遥",
			ProductLine:   "Signet",
			Design:        "サインリング スクエア",
			PreviewURL:    "/public/static/previews/signet.png",
			PreviewAlt:    "Signet Ring",
			QueueID:       "atelier-aoyama",
			QueueName:     "青山アトリエ",
			Workstation:   "準備中",
			Assignees:     []Assignee{{Name: "益田 拓", Initials: "TM", Role: "図面確認"}},
			Flags:         []CardFlag{{Label: "素材待ち", Tone: "danger", Icon: "⛔"}},
			DueAt:         now.Add(48 * time.Hour),
			DueLabel:      "残り2日",
			Notes:         []string{"ロゴデータ差し替え待ち"},
			Blocked:       true,
			BlockedReason: "素材支給待ち",
			AgingHours:    6,
		}, []ProductionEvent{
			{ID: "evt-1060-1", Stage: StageQueued, StageLabel: StageLabel(StageQueued), Type: "queued", Description: "支給待ち", Actor: "益田 拓", OccurredAt: now.Add(-6 * time.Hour), Note: "素材調達中"},
		}),
		newCardRecord(Card{
			ID:            "order-1041",
			OrderNumber:   "1041",
			Stage:         StagePolishing,
			Priority:      PriorityRush,
			PriorityLabel: "特急",
			PriorityTone:  "warning",
			Customer:      "李 美咲",
			ProductLine:   "Aurora",
			Design:        "グラデーションバングル",
			PreviewURL:    "/public/static/previews/bangle.png",
			PreviewAlt:    "Aurora Bangle",
			QueueID:       "atelier-aoyama",
			QueueName:     "青山アトリエ",
			Workstation:   "POL-01",
			Assignees:     []Assignee{{Name: "原田 琴", Initials: "KH", Role: "研磨"}},
			Flags:         []CardFlag{{Label: "QC要注意", Tone: "warning", Icon: "⚠"}},
			DueAt:         now.Add(12 * time.Hour),
			DueLabel:      "残り12時間",
			DueTone:       "danger",
			Notes:         []string{"内側に小傷あり"},
			AgingHours:    27,
		}, []ProductionEvent{
			{ID: "evt-1041-1", Stage: StageEngraving, StageLabel: StageLabel(StageEngraving), Type: "engraving.complete", Description: "刻印完了", Actor: "北原 悠", OccurredAt: now.Add(-15 * time.Hour)},
			{ID: "evt-1041-2", Stage: StagePolishing, StageLabel: StageLabel(StagePolishing), Type: "polishing.start", Description: "研磨開始", Actor: "原田 琴", Station: "POL-01", OccurredAt: now.Add(-4 * time.Hour)},
		}),
		newCardRecord(Card{
			ID:            "order-1033",
			OrderNumber:   "1033",
			Stage:         StageQC,
			Priority:      PriorityNormal,
			PriorityLabel: "通常",
			PriorityTone:  "info",
			Customer:      "フィリップ 仁",
			ProductLine:   "Heritage",
			Design:        "ペアリング",
			PreviewURL:    "/public/static/previews/pair.png",
			PreviewAlt:    "Pair Ring",
			QueueID:       "atelier-aoyama",
			QueueName:     "青山アトリエ",
			Workstation:   "QC-02",
			Assignees: []Assignee{
				{Name: "宮川 光", Initials: "HM", Role: "QC"},
				{Name: "鈴木 亮", Initials: "RS", Role: "梱包"},
			},
			Flags:      []CardFlag{{Label: "刻印差異", Tone: "warning", Icon: "✏"}},
			DueAt:      now.Add(6 * time.Hour),
			DueLabel:   "残り6時間",
			Notes:      []string{"サイズ#10/#12"},
			AgingHours: 30,
		}, []ProductionEvent{
			{ID: "evt-1033-1", Stage: StagePolishing, StageLabel: StageLabel(StagePolishing), Type: "polishing.complete", Description: "研磨完了", Actor: "土屋 凛", OccurredAt: now.Add(-8 * time.Hour)},
			{ID: "evt-1033-2", Stage: StageQC, StageLabel: StageLabel(StageQC), Type: "qc.start", Description: "検品中", Actor: "宮川 光", Station: "QC-02", OccurredAt: now.Add(-1 * time.Hour)},
		}),
		newCardRecord(Card{
			ID:            "order-1025",
			OrderNumber:   "1025",
			Stage:         StagePacked,
			Priority:      PriorityNormal,
			PriorityLabel: "通常",
			PriorityTone:  "success",
			Customer:      "杉山 桃子",
			ProductLine:   "Brilliant",
			Design:        "ハーフエタニティ",
			PreviewURL:    "/public/static/previews/eternity.png",
			PreviewAlt:    "Eternity Ring",
			QueueID:       "atelier-aoyama",
			QueueName:     "青山アトリエ",
			Workstation:   "PACK-01",
			Assignees:     []Assignee{{Name: "鈴木 亮", Initials: "RS", Role: "梱包"}},
			Flags:         []CardFlag{{Label: "ラッピング指定", Tone: "info", Icon: "🎀"}},
			DueAt:         now.Add(3 * time.Hour),
			DueLabel:      "本日出荷",
			Notes:         []string{"カード同梱"},
			AgingHours:    34,
		}, []ProductionEvent{
			{ID: "evt-1025-1", Stage: StageQC, StageLabel: StageLabel(StageQC), Type: "qc.pass", Description: "QC合格", Actor: "宮川 光", OccurredAt: now.Add(-5 * time.Hour)},
			{ID: "evt-1025-2", Stage: StagePacked, StageLabel: StageLabel(StagePacked), Type: "packing.start", Description: "梱包中", Actor: "鈴木 亮", Station: "PACK-01", OccurredAt: now.Add(-1 * time.Hour)},
		}),
		newCardRecord(Card{
			ID:            "order-1071",
			OrderNumber:   "1071",
			Stage:         StageEngraving,
			Priority:      PriorityHold,
			PriorityLabel: "保留",
			PriorityTone:  "danger",
			Customer:      "アレックス 中島",
			ProductLine:   "Monogram",
			Design:        "K18 シグネット",
			PreviewURL:    "/public/static/previews/monogram.png",
			PreviewAlt:    "Monogram Ring",
			QueueID:       "atelier-kyoto",
			QueueName:     "京都スタジオ",
			Workstation:   "HAND-01",
			Assignees:     []Assignee{{Name: "辻村 慎", Initials: "ST", Role: "手彫り"}},
			Flags:         []CardFlag{{Label: "校正待ち", Tone: "danger", Icon: "✉"}},
			DueAt:         now.Add(72 * time.Hour),
			DueLabel:      "残り3日",
			Notes:         []string{"校了次第再開"},
			Blocked:       true,
			BlockedReason: "モノグラム校正待ち",
			AgingHours:    5,
		}, []ProductionEvent{
			{ID: "evt-1071-1", Stage: StageQueued, StageLabel: StageLabel(StageQueued), Type: "queued", Description: "京都工房待機", Actor: "自動割当", OccurredAt: now.Add(-8 * time.Hour)},
			{ID: "evt-1071-2", Stage: StageEngraving, StageLabel: StageLabel(StageEngraving), Type: "engraving.paused", Description: "校正待ち", Actor: "辻村 慎", OccurredAt: now.Add(-2 * time.Hour), Note: "モノグラム修正要"},
		}),
	}

	for _, record := range cards {
		timeline := record.timeline
		if len(timeline) > 0 {
			record.card.LastEvent = timeline[0]
		}
		record.card.Timeline = append([]ProductionEvent(nil), timeline...)
		s.cards[record.card.ID] = record
		if queue, ok := s.queues[record.card.QueueID]; ok {
			queue.Load++
			s.queues[record.card.QueueID] = queue
		}
	}
}

func (s *StaticService) queueRecords(queueID string) []*cardRecord {
	records := make([]*cardRecord, 0, len(s.cards))
	for _, record := range s.cards {
		if record.card.QueueID != queueID {
			continue
		}
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].card.AgingHours > records[j].card.AgingHours
	})
	return records
}

func filterRecords(records []*cardRecord, query BoardQuery) []*cardRecord {
	var result []*cardRecord
	for _, record := range records {
		card := record.card
		if query.Priority != "" && string(card.Priority) != query.Priority {
			continue
		}
		if query.ProductLine != "" && !strings.EqualFold(card.ProductLine, query.ProductLine) {
			continue
		}
		if query.Workstation != "" && !strings.EqualFold(card.Workstation, query.Workstation) {
			continue
		}
		result = append(result, record)
	}
	return result
}

func (s *StaticService) buildLanes(records []*cardRecord) []Lane {
	lanes := make([]Lane, 0, len(s.laneDefs))
	for _, def := range s.laneDefs {
		laneRecords := make([]*cardRecord, 0)
		for _, record := range records {
			if record.card.Stage == def.stage {
				laneRecords = append(laneRecords, record)
			}
		}
		sort.SliceStable(laneRecords, func(i, j int) bool {
			if laneRecords[i].card.Priority != laneRecords[j].card.Priority {
				if laneRecords[i].card.Priority == PriorityRush {
					return true
				}
				if laneRecords[j].card.Priority == PriorityRush {
					return false
				}
			}
			if !laneRecords[i].card.DueAt.Equal(laneRecords[j].card.DueAt) {
				return laneRecords[i].card.DueAt.Before(laneRecords[j].card.DueAt)
			}
			return laneRecords[i].card.OrderNumber < laneRecords[j].card.OrderNumber
		})

		cards := make([]Card, 0, len(laneRecords))
		for _, record := range laneRecords {
			card := cloneCard(record.card)
			card.Timeline = append([]ProductionEvent(nil), record.timeline...)
			cards = append(cards, card)
		}

		lanes = append(lanes, Lane{
			Stage:       def.stage,
			Label:       def.label,
			Description: def.description,
			Capacity:    LaneCapacity{Used: len(cards), Limit: def.capacity},
			SLA:         SLAMeta{Label: def.slaLabel, Tone: def.slaTone},
			Cards:       cards,
		})
	}
	return lanes
}

func (s *StaticService) buildSummary(queue Queue, records []*cardRecord) Summary {
	var dueSoon, blocked int
	now := time.Now()
	for _, record := range records {
		if record.card.Blocked {
			blocked++
		}
		if record.card.DueAt.Sub(now) <= 24*time.Hour {
			dueSoon++
		}
	}
	utilisation := 0
	if queue.Capacity > 0 {
		utilisation = int(float64(queue.Load) / float64(queue.Capacity) * 100)
	}
	return Summary{
		TotalWIP:     len(records),
		DueSoon:      dueSoon,
		Blocked:      blocked,
		AvgLeadHours: queue.LeadTimeHours,
		Utilisation:  utilisation,
		UpdatedAt:    time.Now(),
	}
}

func (s *StaticService) buildFilters(records []*cardRecord, query BoardQuery) FilterSummary {
	countProduct := counter{}
	countPriority := counter{}
	countWorkstation := counter{}

	for _, record := range records {
		countProduct[record.card.ProductLine]++
		countPriority[string(record.card.Priority)]++
		ws := strings.TrimSpace(record.card.Workstation)
		if ws == "" {
			ws = "未割当"
		}
		countWorkstation[ws]++
	}

	priorities := buildFilterOptions(countPriority, query.Priority)
	for i := range priorities {
		priorities[i].Label = priorityDisplay(priorities[i].Value)
	}

	return FilterSummary{
		ProductLines: buildFilterOptions(countProduct, query.ProductLine),
		Priorities:   priorities,
		Workstations: buildFilterOptions(countWorkstation, query.Workstation),
	}
}

func buildFilterOptions(c counter, active string) []FilterOption {
	options := make([]FilterOption, 0, len(c))
	keys := make([]string, 0, len(c))
	for key := range c {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		options = append(options, FilterOption{
			Value:  key,
			Label:  key,
			Count:  c[key],
			Active: strings.EqualFold(key, active),
		})
	}
	return options
}

func priorityDisplay(value string) string {
	switch value {
	case string(PriorityRush):
		return "特急"
	case string(PriorityHold):
		return "保留"
	case string(PriorityNormal):
		fallthrough
	default:
		return "通常"
	}
}

func (s *StaticService) queueOptions(active string) []QueueOption {
	options := make([]QueueOption, 0, len(s.queues))
	for _, queue := range s.queues {
		options = append(options, QueueOption{
			ID:       queue.ID,
			Label:    queue.Name,
			Sublabel: queue.Location,
			Load:     fmt.Sprintf("%d枚進行", queue.Load),
			Active:   queue.ID == active,
		})
	}
	sort.Slice(options, func(i, j int) bool {
		return options[i].Label < options[j].Label
	})
	return options
}

func (s *StaticService) buildDrawer(records []*cardRecord, selected string) (string, Drawer) {
	if len(records) == 0 {
		return "", Drawer{Empty: true}
	}

	var target *cardRecord
	if selected != "" {
		for _, record := range records {
			if record.card.ID == selected {
				target = record
				break
			}
		}
	}
	if target == nil {
		target = records[0]
	}

	card := target.card
	timeline := make([]ProductionEvent, len(target.timeline))
	copy(timeline, target.timeline)

	drawer := Drawer{
		Card: DrawerCard{
			ID:            card.ID,
			OrderNumber:   card.OrderNumber,
			Customer:      card.Customer,
			PriorityLabel: card.PriorityLabel,
			PriorityTone:  card.PriorityTone,
			Stage:         card.Stage,
			StageLabel:    StageLabel(card.Stage),
			ProductLine:   card.ProductLine,
			QueueName:     card.QueueName,
			Workstation:   card.Workstation,
			PreviewURL:    card.PreviewURL,
			PreviewAlt:    card.PreviewAlt,
			DueLabel:      card.DueLabel,
			Notes:         append([]string(nil), card.Notes...),
			Flags:         cloneFlags(card.Flags),
			Assignees:     cloneAssignees(card.Assignees),
			LastUpdated:   card.LastEvent.OccurredAt,
		},
		Timeline: timeline,
		Details: []DrawerDetail{
			{Label: "ステージ", Value: StageLabel(card.Stage)},
			{Label: "ライン", Value: card.QueueName},
			{Label: "ステーション", Value: card.Workstation},
		},
	}

	return card.ID, drawer
}

func newCardRecord(card Card, timeline []ProductionEvent) *cardRecord {
	return &cardRecord{card: card, timeline: timeline}
}

func cloneCard(card Card) Card {
	clone := card
	clone.Assignees = cloneAssignees(card.Assignees)
	clone.Flags = cloneFlags(card.Flags)
	clone.Notes = append([]string(nil), card.Notes...)
	clone.Timeline = append([]ProductionEvent(nil), card.Timeline...)
	return clone
}

func cloneAssignees(src []Assignee) []Assignee {
	out := make([]Assignee, len(src))
	copy(out, src)
	return out
}

func cloneFlags(src []CardFlag) []CardFlag {
	out := make([]CardFlag, len(src))
	copy(out, src)
	return out
}

func appendUnique(list []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return list
	}
	for _, existing := range list {
		if existing == value {
			return list
		}
	}
	return append(list, value)
}

func isValidStage(stage Stage) bool {
	switch stage {
	case StageQueued, StageEngraving, StagePolishing, StageQC, StagePacked:
		return true
	default:
		return false
	}
}

func coalesce(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
