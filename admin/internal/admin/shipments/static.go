package shipments

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

// StaticService provides deterministic shipment batch data for local development.
type StaticService struct {
	mu      sync.RWMutex
	batches []Batch
	details map[string]BatchDetail
}

// NewStaticService seeds the static shipment data set.
func NewStaticService() *StaticService {
	now := time.Now()

	makeBatch := func(id, ref, carrier, carrierLabel, serviceLevel, facility, facilityLabel string, status BatchStatus, statusLabel, statusTone string, createdAgo time.Duration, progress, ordersTotal, ordersPending, labelsReady, labelsFailed int, slaStatus, slaTone, badgeIcon, badgeTone, badgeLabel, lastOperator string) Batch {
		if carrierLabel == "" {
			carrierLabel = carrier
		}
		if facilityLabel == "" {
			facilityLabel = facility
		}
		return Batch{
			ID:               id,
			Reference:        ref,
			Carrier:          carrier,
			CarrierLabel:     carrierLabel,
			ServiceLevel:     serviceLevel,
			Facility:         facility,
			FacilityLabel:    facilityLabel,
			Status:           status,
			StatusLabel:      statusLabel,
			StatusTone:       statusTone,
			CreatedAt:        now.Add(-createdAgo),
			OrdersTotal:      ordersTotal,
			OrdersPending:    ordersPending,
			LabelsReady:      labelsReady,
			LabelsFailed:     labelsFailed,
			ProgressPercent:  progress,
			SLAStatus:        slaStatus,
			SLATone:          slaTone,
			BadgeIcon:        badgeIcon,
			BadgeTone:        badgeTone,
			BadgeLabel:       badgeLabel,
			LabelDownloadURL: "/admin/shipments/batches/" + id + "/labels.pdf",
			ManifestURL:      "/admin/shipments/batches/" + id + "/manifest.csv",
			LastOperator:     lastOperator,
			LastUpdated:      now.Add(-time.Duration(progress) * time.Minute),
		}
	}

	var batches = []Batch{
		makeBatch("batch-2304", "Batch #2304", "yamato", "ヤマト運輸", "宅急便 (翌日)", "tokyo", "東京倉庫", BatchStatusQueued, "キュー待ち", "info", 35*time.Minute, 12, 48, 36, 12, 0, "ピッキング中", "info", "🆕", "info", "新規", "星野"),
		makeBatch("batch-2305", "Batch #2305", "sagawa", "佐川急便", "飛脚宅配便", "osaka", "大阪DC", BatchStatusRunning, "ラベル生成中", "warning", 52*time.Minute, 68, 60, 12, 48, 0, "〆切まで 20分", "warning", "⚠️", "warning", "要注意", "田中"),
		makeBatch("batch-2306", "Batch #2306", "yamato", "ヤマト運輸", "ネコポス", "tokyo", "東京倉庫", BatchStatusCompleted, "完了", "success", 2*time.Hour+15*time.Minute, 100, 80, 0, 80, 0, "完了済み", "success", "✅", "success", "完了", "渡辺"),
		makeBatch("batch-2307", "Batch #2307", "japanpost", "日本郵便", "ゆうパック", "fukuoka", "福岡サテライト", BatchStatusFailed, "エラー", "danger", 15*time.Minute, 47, 24, 24, 0, 23, "ラベル失敗 23件", "danger", "❗", "danger", "再処理必要", "松本"),
		makeBatch("batch-2308", "Batch #2308", "yamato", "ヤマト運輸", "宅急便 (タイム)", "nagoya", "名古屋センター", BatchStatusDraft, "下書き", "slate", 8*time.Minute, 0, 18, 18, 0, 0, "未送信", "default", "", "", "", "中野"),
	}

	details := map[string]BatchDetail{}
	for _, batch := range batches {
		details[batch.ID] = mockDetail(batch, now)
	}

	return &StaticService{
		batches: batches,
		details: details,
	}
}

func mockDetail(batch Batch, now time.Time) BatchDetail {
	makeOrder := func(id, number, customer, destination, serviceLevel, labelStatus, labelTone string, ago time.Duration, labelURL string) BatchOrder {
		return BatchOrder{
			OrderID:      id,
			OrderNumber:  number,
			CustomerName: customer,
			Destination:  destination,
			ServiceLevel: serviceLevel,
			LabelStatus:  labelStatus,
			LabelTone:    labelTone,
			LabelURL:     labelURL,
			CreatedAt:    now.Add(-ago),
		}
	}

	makeTimeline := func(title, description, actor, tone, icon string, ago time.Duration) TimelineEvent {
		return TimelineEvent{
			Title:       title,
			Description: description,
			Actor:       actor,
			Tone:        tone,
			Icon:        icon,
			OccurredAt:  now.Add(-ago),
		}
	}

	makePrint := func(label, actor string, count int, channel string, ago time.Duration) PrintRecord {
		return PrintRecord{
			Label:     label,
			Actor:     actor,
			Count:     count,
			PrintedAt: now.Add(-ago),
			Channel:   channel,
		}
	}

	operator := Operator{
		Name:      batch.LastOperator,
		Email:     strings.ToLower(batch.LastOperator) + "@hanko.local",
		Shift:     "日勤 (9:00 - 18:00)",
		AvatarURL: "",
	}

	jobState := "queued"
	jobLabel := "キュー待機中"
	jobTone := "info"
	progress := batch.ProgressPercent

	switch batch.Status {
	case BatchStatusDraft:
		jobState = "draft"
		jobLabel = "下書き"
		jobTone = "slate"
	case BatchStatusQueued:
		jobState = "queued"
		jobLabel = "キュー待機中"
		jobTone = "info"
	case BatchStatusRunning:
		jobState = "running"
		jobLabel = "ラベル生成中"
		jobTone = "warning"
	case BatchStatusCompleted:
		jobState = "completed"
		jobLabel = "完了"
		jobTone = "success"
		progress = 100
	case BatchStatusFailed:
		jobState = "failed"
		jobLabel = "失敗"
		jobTone = "danger"
		if progress < 5 {
			progress = 5
		}
	}

	detail := BatchDetail{
		Batch: batch,
		Orders: []BatchOrder{
			makeOrder("order-1101", "1101", "青木 里奈", "東京都世田谷区", batch.ServiceLevel, "ラベル出力済み", "success", 45*time.Minute, batch.LabelDownloadURL),
			makeOrder("order-1102", "1102", "近藤 翼", "大阪府豊中市", batch.ServiceLevel, "ラベル生成待ち", "warning", 40*time.Minute, ""),
			makeOrder("order-1103", "1103", "山田 貴子", "福岡県福岡市中央区", batch.ServiceLevel, "エラー: サイズ不一致", "danger", 32*time.Minute, ""),
		},
		Timeline: []TimelineEvent{
			makeTimeline("ラベルジョブをキューに投入", "配送ラベル生成ジョブをスケジュールしました。", operator.Name, "info", "📝", 30*time.Minute),
			makeTimeline("検品完了", "倉庫スタッフが検品完了を報告しました。", "倉庫システム", "success", "📦", 45*time.Minute),
			makeTimeline("集荷依頼送信", "キャリアに集荷依頼を送信しました。", operator.Name, "info", "📨", 50*time.Minute),
		},
		PrintHistory: []PrintRecord{
			makePrint("ラベル再出力", operator.Name, 12, "倉庫プリンタ", 15*time.Minute),
			makePrint("ラベル初回出力", "自動化ジョブ", batch.OrdersTotal, "Label API", 35*time.Minute),
		},
		Operator: operator,
		Job: JobStatus{
			State:      jobState,
			StateLabel: jobLabel,
			StateTone:  jobTone,
			Progress:   progress,
			StartedAt:  ptr(batch.CreatedAt.Add(10 * time.Minute)),
			EndedAt:    ptr(batch.CreatedAt.Add(35 * time.Minute)),
			Message:    "クラウドプリントキュー連携済み",
		},
	}

	if batch.Status == BatchStatusDraft {
		detail.Job.StartedAt = nil
		detail.Job.EndedAt = nil
		detail.Job.Message = "送信待ち。バッチを提出してラベル生成を開始します。"
		detail.PrintHistory = nil
		detail.Timeline = detail.Timeline[:1]
	}

	return detail
}

func ptr[T any](v T) *T {
	return &v
}

// ListBatches implements Service.
func (s *StaticService) ListBatches(_ context.Context, _ string, query ListQuery) (ListResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := strings.TrimSpace(string(query.Status))
	carrier := strings.TrimSpace(query.Carrier)
	facility := strings.TrimSpace(query.Facility)

	var filtered []Batch
	for _, batch := range s.batches {
		if status != "" && !strings.EqualFold(string(batch.Status), status) {
			continue
		}
		if carrier != "" && !strings.EqualFold(batch.Carrier, carrier) {
			continue
		}
		if facility != "" && !strings.EqualFold(batch.Facility, facility) {
			continue
		}
		if query.Start != nil && batch.CreatedAt.Before(*query.Start) {
			continue
		}
		if query.End != nil && batch.CreatedAt.After(*query.End) {
			continue
		}
		filtered = append(filtered, batch)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})

	total := len(filtered)
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	page := query.Page
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

	paged := append([]Batch(nil), filtered[start:end]...)

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

	summary := buildSummary(filtered)
	filters := buildFilterSummary(s.batches)

	selected := strings.TrimSpace(query.Selected)
	if selected == "" && len(paged) > 0 {
		selected = paged[0].ID
	}

	return ListResult{
		Summary:    summary,
		Batches:    paged,
		Filters:    filters,
		Pagination: Pagination{Page: page, PageSize: pageSize, TotalItems: total, NextPage: next, PrevPage: prev},
		Generated:  time.Now(),
		SelectedID: selected,
	}, nil
}

func buildSummary(batches []Batch) Summary {
	var outstanding, inProgress, warnings int
	var lastRun *time.Time
	for _, batch := range batches {
		switch batch.Status {
		case BatchStatusDraft, BatchStatusQueued:
			outstanding++
		case BatchStatusRunning:
			inProgress++
		case BatchStatusFailed:
			warnings++
		}
		if batch.Status == BatchStatusCompleted {
			if lastRun == nil || lastRun.Before(batch.CreatedAt) {
				ts := batch.CreatedAt
				lastRun = &ts
			}
		}
	}
	return Summary{
		Outstanding: outstanding,
		InProgress:  inProgress,
		Warnings:    warnings,
		LastRun:     lastRun,
	}
}

func buildFilterSummary(all []Batch) FilterSummary {
	statusCounts := map[BatchStatus]int{}
	carrierCounts := map[string]int{}
	facilityCounts := map[string]int{}

	for _, batch := range all {
		statusCounts[batch.Status]++
		carrierCounts[batch.Carrier]++
		facilityCounts[batch.Facility]++
	}

	var statuses []StatusOption
	for status, count := range statusCounts {
		statuses = append(statuses, StatusOption{
			Value: status,
			Label: statusLabel(status),
			Tone:  statusTone(status),
			Count: count,
		})
	}
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Label < statuses[j].Label
	})

	var carriers []SelectOption
	for value, count := range carrierCounts {
		carriers = append(carriers, SelectOption{
			Value: value,
			Label: carrierLabel(value),
			Count: count,
		})
	}
	sort.Slice(carriers, func(i, j int) bool {
		return carriers[i].Label < carriers[j].Label
	})

	var facilities []SelectOption
	for value, count := range facilityCounts {
		facilities = append(facilities, SelectOption{
			Value: value,
			Label: facilityLabel(value),
			Count: count,
		})
	}
	sort.Slice(facilities, func(i, j int) bool {
		return facilities[i].Label < facilities[j].Label
	})

	return FilterSummary{
		StatusOptions:   statuses,
		CarrierOptions:  carriers,
		FacilityOptions: facilities,
	}
}

func statusLabel(status BatchStatus) string {
	switch status {
	case BatchStatusDraft:
		return "下書き"
	case BatchStatusQueued:
		return "キュー待ち"
	case BatchStatusRunning:
		return "処理中"
	case BatchStatusCompleted:
		return "完了"
	case BatchStatusFailed:
		return "失敗"
	default:
		return string(status)
	}
}

func statusTone(status BatchStatus) string {
	switch status {
	case BatchStatusDraft:
		return "slate"
	case BatchStatusQueued:
		return "info"
	case BatchStatusRunning:
		return "warning"
	case BatchStatusCompleted:
		return "success"
	case BatchStatusFailed:
		return "danger"
	default:
		return "slate"
	}
}

func carrierLabel(value string) string {
	switch strings.ToLower(value) {
	case "yamato":
		return "ヤマト運輸"
	case "sagawa":
		return "佐川急便"
	case "japanpost":
		return "日本郵便"
	default:
		return strings.ToUpper(value)
	}
}

func facilityLabel(value string) string {
	switch strings.ToLower(value) {
	case "tokyo":
		return "東京倉庫"
	case "osaka":
		return "大阪DC"
	case "fukuoka":
		return "福岡サテライト"
	case "nagoya":
		return "名古屋センター"
	default:
		return strings.ToUpper(value)
	}
}

// BatchDetail implements Service.
func (s *StaticService) BatchDetail(_ context.Context, _ string, batchID string) (BatchDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	detail, ok := s.details[strings.TrimSpace(batchID)]
	if !ok {
		return BatchDetail{}, ErrBatchNotFound
	}
	return detail, nil
}
