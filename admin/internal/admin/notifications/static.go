package notifications

import (
	"context"
	"sort"
	"strings"
	"time"
)

// StaticService provides canned responses for development, previews, and tests.
type StaticService struct {
	Notifications []Notification
}

// NewStaticService builds a StaticService populated with representative notifications.
func NewStaticService() *StaticService {
	now := time.Now()

	jobFailure := Notification{
		ID:       "job-failure-1923",
		Category: CategoryFailedJob,
		Severity: SeverityCritical,
		Status:   StatusOpen,
		Title:    "バッチジョブ inventory-sync が連続3回失敗しました",
		Summary:  "在庫同期ジョブが Cloud Tasks で失敗しました。API レスポンス: 504 Gateway Timeout。",
		Resource: ResourceRef{
			Kind:       "job",
			Identifier: "inventory-sync",
			Label:      "inventory-sync / 05:00 バッチ",
			URL:        "/admin/system/tasks/jobs/inventory-sync",
		},
		CreatedAt: now.Add(-25 * time.Minute),
		Owner:     "倉橋",
		Links: []Link{
			{Label: "再実行", URL: "/admin/system/tasks/jobs/inventory-sync/retry", Icon: "⟳"},
			{Label: "Runbook", URL: "https://runbooks.hanko.local/jobs/inventory-sync"},
		},
		Metadata: []Metadata{
			{Label: "エラーコード", Value: "504"},
			{Label: "リトライ回数", Value: "3/5"},
		},
		Timeline: []TimelineEvent{
			{
				Title:       "ジョブが失敗しました",
				Description: "Cloud Scheduler からの起動後 180 秒でタイムアウトしました。",
				OccurredAt:  now.Add(-25 * time.Minute),
				Actor:       "system",
				Tone:        "danger",
				Icon:        "🛑",
			},
			{
				Title:       "自動リトライ試行",
				Description: "バックオフポリシーに従って 2 回の再試行を実施しました。",
				OccurredAt:  now.Add(-19 * time.Minute),
				Actor:       "system",
				Tone:        "warning",
				Icon:        "⏳",
			},
		},
	}

	stockAlert := Notification{
		ID:       "stock-alert-kf-2024-gd",
		Category: CategoryStockAlert,
		Severity: SeverityHigh,
		Status:   StatusAcknowledged,
		Title:    "SKU KF-2024-GD の在庫がしきい値を下回っています",
		Summary:  "残数 12。今週の販売予測は 38 のため、早急な補充が必要です。",
		Resource: ResourceRef{
			Kind:       "sku",
			Identifier: "KF-2024-GD",
			Label:      "KF-2024-GD（刻印 ゴールド）",
			URL:        "/admin/catalog/products?sku=KF-2024-GD",
		},
		CreatedAt:      now.Add(-2 * time.Hour),
		Owner:          "阿部",
		AcknowledgedBy: "阿部",
		AcknowledgedAt: ptrTime(now.Add(-95 * time.Minute)),
		Links: []Link{
			{Label: "在庫を補充", URL: "/admin/catalog/products/KF-2024-GD/restock", Icon: "➕"},
			{Label: "需要予測を見る", URL: "/admin/catalog/products/KF-2024-GD/forecast"},
		},
		Metadata: []Metadata{
			{Label: "現在庫", Value: "12"},
			{Label: "安全在庫", Value: "30"},
			{Label: "補充リードタイム", Value: "7日"},
		},
		Timeline: []TimelineEvent{
			{
				Title:       "在庫アラート発火",
				Description: "残数 15 を下回ったため警告を発火しました。",
				OccurredAt:  now.Add(-2 * time.Hour),
				Actor:       "inventory-service",
				Tone:        "warning",
				Icon:        "⚠️",
			},
			{
				Title:       "担当者がステータスを確認",
				Description: "阿部さんが在庫補充を進めるため購買チームに連絡しました。",
				OccurredAt:  now.Add(-90 * time.Minute),
				Actor:       "阿部",
				Tone:        "info",
				Icon:        "👤",
			},
		},
	}

	shippingException := Notification{
		ID:       "shipping-delay-ops-1048",
		Category: CategoryShippingException,
		Severity: SeverityMedium,
		Status:   StatusOpen,
		Title:    "注文 #1048 の配送が遅延ステータスになりました",
		Summary:  "ヤマト運輸の集荷がスキャンされず 12 時間が経過しています。顧客への連絡が推奨されます。",
		Resource: ResourceRef{
			Kind:       "order",
			Identifier: "1048",
			Label:      "注文 #1048",
			URL:        "/admin/orders/1048",
		},
		CreatedAt: now.Add(-3 * time.Hour),
		Owner:     "未割当",
		Links: []Link{
			{Label: "注文を開く", URL: "/admin/orders/1048"},
			{Label: "配送状況を確認", URL: "/admin/shipments/tracking?order=1048"},
		},
		Metadata: []Metadata{
			{Label: "キャリア", Value: "ヤマト運輸"},
			{Label: "配送種別", Value: "翌日配達"},
			{Label: "遅延時間", Value: "12時間"},
		},
		Timeline: []TimelineEvent{
			{
				Title:       "遅延検知",
				Description: "スキャン欠落を検知し配送例外に移行しました。",
				OccurredAt:  now.Add(-3 * time.Hour),
				Actor:       "logistics-monitor",
				Tone:        "warning",
				Icon:        "🚚",
			},
		},
	}

	resolvedNotification := Notification{
		ID:       "shipping-exception-closed-1033",
		Category: CategoryShippingException,
		Severity: SeverityLow,
		Status:   StatusResolved,
		Title:    "注文 #1033 の配送例外をクローズしました",
		Summary:  "再配達が完了し顧客へ連絡済みです。",
		Resource: ResourceRef{
			Kind:       "order",
			Identifier: "1033",
			Label:      "注文 #1033",
			URL:        "/admin/orders/1033",
		},
		CreatedAt:  now.Add(-27 * time.Hour),
		ResolvedAt: ptrTime(now.Add(-2 * time.Hour)),
		Owner:      "サポートチーム",
		Metadata: []Metadata{
			{Label: "顧客満足度", Value: "4 / 5"},
		},
		Timeline: []TimelineEvent{
			{
				Title:       "お客様へ再送手配済み",
				Description: "ヤマト運輸と連携し再配達を手配しました。",
				OccurredAt:  now.Add(-22 * time.Hour),
				Actor:       "佐藤（CS）",
				Tone:        "info",
				Icon:        "📞",
			},
			{
				Title:       "配送完了",
				Description: "顧客から受領確認が取れたためクローズしました。",
				OccurredAt:  now.Add(-2 * time.Hour),
				Actor:       "佐藤（CS）",
				Tone:        "success",
				Icon:        "✅",
			},
		},
	}

	suppressed := Notification{
		ID:       "job-failure-suppressed-1301",
		Category: CategoryFailedJob,
		Severity: SeverityMedium,
		Status:   StatusSuppressed,
		Title:    "ジョブ reporting-delta がメンテナンス中のため通知をミュートしています",
		Summary:  "メンテナンスウィンドウ中につき自動通知を一時停止しています。",
		Resource: ResourceRef{
			Kind:       "job",
			Identifier: "reporting-delta",
			Label:      "reporting-delta / 30分インクリメント",
			URL:        "/admin/system/tasks/jobs/reporting-delta",
		},
		CreatedAt: now.Add(-6 * time.Hour),
		Owner:     "プラットフォームチーム",
		Metadata: []Metadata{
			{Label: "再開予定", Value: now.Add(2 * time.Hour).Format("15:04")},
		},
	}

	return &StaticService{
		Notifications: []Notification{
			jobFailure,
			stockAlert,
			shippingException,
			resolvedNotification,
			suppressed,
		},
	}
}

// List returns notifications filtered by the query parameters.
func (s *StaticService) List(ctx context.Context, token string, query Query) (Feed, error) {
	items := filterNotifications(s.notifications(), query)
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})

	counts := summariseCounts(items)

	total := len(items)
	limit := query.Limit
	if limit <= 0 {
		limit = total
	}
	if limit > total {
		limit = total
	}
	page := items[:limit]

	nextCursor := ""
	if limit < total {
		nextCursor = items[limit].ID
	}

	return Feed{
		Items:      page,
		Total:      total,
		NextCursor: nextCursor,
		Counts:     counts,
	}, nil
}

// Badge returns aggregate counts for the top-bar badge.
func (s *StaticService) Badge(ctx context.Context, token string) (BadgeCount, error) {
	items := s.notifications()
	active := make([]Notification, 0, len(items))
	for _, item := range items {
		if isActiveStatus(item.Status) {
			active = append(active, item)
		}
	}

	result := BadgeCount{}
	result.Total = len(active)
	for _, item := range active {
		switch item.Severity {
		case SeverityCritical:
			result.Critical++
		case SeverityHigh, SeverityMedium:
			result.Warning++
		}
	}
	return result, nil
}

func (s *StaticService) notifications() []Notification {
	if len(s.Notifications) == 0 {
		return []Notification{}
	}
	cloned := make([]Notification, len(s.Notifications))
	copy(cloned, s.Notifications)
	return cloned
}

func filterNotifications(list []Notification, query Query) []Notification {
	categories := toCategorySet(query.Categories)
	severities := toSeveritySet(query.Severities)
	statuses := toStatusSet(query.Statuses)
	owner := strings.TrimSpace(query.Owner)
	search := strings.ToLower(strings.TrimSpace(query.Search))
	var start time.Time
	var end time.Time
	if query.Start != nil {
		start = (*query.Start).Add(-1 * time.Second)
	}
	if query.End != nil {
		end = (*query.End).Add(1 * time.Second)
	}

	result := make([]Notification, 0, len(list))
	for _, item := range list {
		if len(categories) > 0 && !categories[item.Category] {
			continue
		}
		if len(severities) > 0 && !severities[item.Severity] {
			continue
		}
		if len(statuses) > 0 && !statuses[item.Status] {
			continue
		}
		if !ownerMatches(owner, item.Owner) {
			continue
		}
		if !withinRange(start, end, item.CreatedAt) {
			continue
		}
		if search != "" && !matchesSearch(search, item) {
			continue
		}
		result = append(result, item)
	}
	return result
}

func summariseCounts(list []Notification) CountSummary {
	var summary CountSummary
	summary.Total = len(list)
	for _, item := range list {
		switch item.Status {
		case StatusOpen:
			summary.Open++
		case StatusAcknowledged:
			summary.Acknowledged++
		case StatusResolved:
			summary.Resolved++
		case StatusSuppressed:
			summary.Suppressed++
		}
		if !isActiveStatus(item.Status) {
			continue
		}
		switch item.Severity {
		case SeverityCritical:
			summary.Critical++
		case SeverityHigh, SeverityMedium:
			summary.Warning++
		default:
			summary.Info++
		}
	}
	return summary
}

func toCategorySet(values []Category) map[Category]bool {
	if len(values) == 0 {
		return nil
	}
	set := make(map[Category]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func toSeveritySet(values []Severity) map[Severity]bool {
	if len(values) == 0 {
		return nil
	}
	set := make(map[Severity]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func toStatusSet(values []Status) map[Status]bool {
	if len(values) == 0 {
		return nil
	}
	set := make(map[Status]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func ownerMatches(filter, value string) bool {
	filter = strings.TrimSpace(strings.ToLower(filter))
	if filter == "" {
		return true
	}
	value = strings.TrimSpace(strings.ToLower(value))
	return value == filter
}

func withinRange(start, end, value time.Time) bool {
	if !start.IsZero() && value.Before(start) {
		return false
	}
	if !end.IsZero() && value.After(end) {
		return false
	}
	return true
}

func matchesSearch(query string, item Notification) bool {
	fields := []string{
		item.ID,
		item.Title,
		item.Summary,
		item.Owner,
		item.Resource.Label,
		item.Resource.Identifier,
	}
	for _, meta := range item.Metadata {
		fields = append(fields, meta.Label, meta.Value)
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}

func isActiveStatus(status Status) bool {
	return status == StatusOpen || status == StatusAcknowledged
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
