package search

import (
	"context"
	"sort"
	"strings"
	"time"
)

// StaticService provides deterministic global search results for local development.
type StaticService struct {
	data []staticRecord
}

type staticRecord struct {
	ID          string
	Entity      Entity
	Title       string
	Description string
	Badge       string
	BadgeTone   string
	URL         string
	Score       float64
	OccurredAt  time.Time
	Persona     string
	Metadata    []Metadata
}

// NewStaticService constructs a mock search dataset suitable for local usage.
func NewStaticService() *StaticService {
	now := time.Now()
	return &StaticService{
		data: []staticRecord{
			{
				ID:          "order-1042",
				Entity:      EntityOrder,
				Title:       "注文 #1042 / 長谷川 純",
				Description: "刻印リング（18K）- ステータス: 制作中 - 支払い済み",
				Badge:       "制作中",
				BadgeTone:   "info",
				URL:         "/admin/orders/1042",
				Score:       0.94,
				OccurredAt:  now.Add(-2 * time.Hour),
				Persona:     "operations",
				Metadata: []Metadata{
					{Key: "合計", Value: "¥32,000", Icon: "💴"},
					{Key: "配送予定", Value: now.Add(72 * time.Hour).Format("2006-01-02"), Icon: "📦"},
				},
			},
			{
				ID:          "order-1036",
				Entity:      EntityOrder,
				Title:       "注文 #1036 / 佐藤 真帆",
				Description: "ペアネックレス（シルバー）- ステータス: 出荷済み - 返金申請なし",
				Badge:       "出荷済み",
				BadgeTone:   "success",
				URL:         "/admin/orders/1036",
				Score:       0.88,
				OccurredAt:  now.Add(-26 * time.Hour),
				Persona:     "operations",
				Metadata: []Metadata{
					{Key: "合計", Value: "¥18,400", Icon: "💴"},
					{Key: "配送", Value: "ヤマト運輸 5543-2021-9921", Icon: "🚚"},
				},
			},
			{
				ID:          "order-0998",
				Entity:      EntityOrder,
				Title:       "注文 #998 / 松本 拓也",
				Description: "特注シグネット - ステータス: 支払い待ち - 承認待ち",
				Badge:       "支払い待ち",
				BadgeTone:   "warning",
				URL:         "/admin/orders/998",
				Score:       0.75,
				OccurredAt:  now.Add(-96 * time.Hour),
				Persona:     "finance",
				Metadata: []Metadata{
					{Key: "合計", Value: "¥54,800", Icon: "💴"},
					{Key: "請求書", Value: "送信済み 2024-04-01", Icon: "🧾"},
				},
			},
			{
				ID:          "user-802",
				Entity:      EntityUser,
				Title:       "ユーザー: 青木 里奈",
				Description: "アクティブ顧客。直近注文 #1036、LTV ¥86,400、MFA 有効。",
				Badge:       "顧客",
				BadgeTone:   "muted",
				URL:         "/admin/customers/802",
				Score:       0.82,
				OccurredAt:  now.Add(-6 * time.Hour),
				Persona:     "cs",
				Metadata: []Metadata{
					{Key: "メール", Value: "rina.aoki@example.com", Icon: "✉️"},
					{Key: "登録日", Value: "2023-09-12", Icon: "📅"},
				},
			},
			{
				ID:          "user-640",
				Entity:      EntityUser,
				Title:       "ユーザー: 松田 洋介",
				Description: "VIP 顧客。レビュー 4 件、返金 1 件。サポートタグ: 要フォロー。",
				Badge:       "VIP",
				BadgeTone:   "info",
				URL:         "/admin/customers/640",
				Score:       0.79,
				OccurredAt:  now.Add(-48 * time.Hour),
				Persona:     "marketing",
				Metadata: []Metadata{
					{Key: "LTV", Value: "¥124,000", Icon: "💎"},
					{Key: "最終注文", Value: "#1011 (2024-03-22)", Icon: "🛒"},
				},
			},
			{
				ID:          "review-441",
				Entity:      EntityReview,
				Title:       "レビュー #441 / 評価 ★★☆☆☆",
				Description: "「刻印が薄かったです」丁寧な謝罪と再制作対応を検討。",
				Badge:       "要対応",
				BadgeTone:   "danger",
				URL:         "/admin/reviews/441",
				Score:       0.91,
				OccurredAt:  now.Add(-12 * time.Hour),
				Persona:     "cs",
				Metadata: []Metadata{
					{Key: "注文", Value: "#1042", Icon: "🧾"},
					{Key: "作成日", Value: now.Add(-14 * time.Hour).Format("2006-01-02 15:04"), Icon: "🕒"},
				},
			},
			{
				ID:          "review-439",
				Entity:      EntityReview,
				Title:       "レビュー #439 / 評価 ★★★★★",
				Description: "「指輪の仕上がりが素晴らしい！」 SNS 共有済み。",
				Badge:       "公開中",
				BadgeTone:   "success",
				URL:         "/admin/reviews/439",
				Score:       0.73,
				OccurredAt:  now.Add(-72 * time.Hour),
				Persona:     "marketing",
				Metadata: []Metadata{
					{Key: "投稿者", Value: "青木 里奈", Icon: "🗣"},
					{Key: "モデレーション", Value: "完了", Icon: "✅"},
				},
			},
		},
	}
}

// Search returns filtered static results matching the provided query.
func (s *StaticService) Search(_ context.Context, _ string, query Query) (ResultSet, error) {
	scope := map[Entity]bool{}
	if len(query.Scope) > 0 {
		for _, entity := range query.Scope {
			scope[entity] = true
		}
	}

	term := strings.TrimSpace(strings.ToLower(query.Term))
	persona := strings.TrimSpace(strings.ToLower(query.Persona))

	var filters []func(staticRecord) bool
	if len(scope) > 0 {
		filters = append(filters, func(rec staticRecord) bool {
			return scope[rec.Entity]
		})
	}
	if term != "" {
		filters = append(filters, func(rec staticRecord) bool {
			if strings.Contains(strings.ToLower(rec.Title), term) {
				return true
			}
			return strings.Contains(strings.ToLower(rec.Description), term)
		})
	}
	if query.Start != nil {
		start := query.Start.Truncate(time.Minute)
		filters = append(filters, func(rec staticRecord) bool {
			return !rec.OccurredAt.Before(start)
		})
	}
	if query.End != nil {
		end := query.End.Truncate(time.Minute)
		filters = append(filters, func(rec staticRecord) bool {
			return !rec.OccurredAt.After(end)
		})
	}
	if persona != "" {
		filters = append(filters, func(rec staticRecord) bool {
			return strings.Contains(strings.ToLower(rec.Persona), persona)
		})
	}

	type groupState struct {
		group ResultGroup
	}

	groups := map[Entity]*groupState{}
	total := 0
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}

	for _, rec := range s.data {
		matches := true
		for _, fn := range filters {
			if !fn(rec) {
				matches = false
				break
			}
		}
		if !matches {
			continue
		}

		state, ok := groups[rec.Entity]
		if !ok {
			state = &groupState{
				group: ResultGroup{
					Entity: rec.Entity,
					Label:  labelForEntity(rec.Entity),
					Icon:   iconForEntity(rec.Entity),
				},
			}
			groups[rec.Entity] = state
		}

		state.group.Total++
		if len(state.group.Hits) >= limit {
			state.group.HasMore = true
			continue
		}

		hit := Hit{
			ID:          rec.ID,
			Entity:      rec.Entity,
			Title:       rec.Title,
			Description: rec.Description,
			Badge:       rec.Badge,
			BadgeTone:   rec.BadgeTone,
			URL:         rec.URL,
			Score:       rec.Score,
			Persona:     rec.Persona,
			Metadata:    append([]Metadata(nil), rec.Metadata...),
		}
		if !rec.OccurredAt.IsZero() {
			t := rec.OccurredAt
			hit.OccurredAt = &t
		}

		state.group.Hits = append(state.group.Hits, hit)
		total++
	}

	result := ResultSet{
		Total:    total,
		Duration: 12 * time.Millisecond,
		Groups:   make([]ResultGroup, 0, len(groups)),
	}

	for _, state := range groups {
		sort.SliceStable(state.group.Hits, func(i, j int) bool {
			if state.group.Hits[i].Score == state.group.Hits[j].Score {
				if state.group.Hits[i].OccurredAt == nil || state.group.Hits[j].OccurredAt == nil {
					return state.group.Hits[i].Title < state.group.Hits[j].Title
				}
				return state.group.Hits[i].OccurredAt.After(*state.group.Hits[j].OccurredAt)
			}
			return state.group.Hits[i].Score > state.group.Hits[j].Score
		})
		result.Groups = append(result.Groups, state.group)
	}

	sort.SliceStable(result.Groups, func(i, j int) bool {
		return result.Groups[i].Label < result.Groups[j].Label
	})

	return result, nil
}

func labelForEntity(entity Entity) string {
	switch entity {
	case EntityOrder:
		return "注文"
	case EntityUser:
		return "ユーザー"
	case EntityReview:
		return "レビュー"
	default:
		return "その他"
	}
}

func iconForEntity(entity Entity) string {
	switch entity {
	case EntityOrder:
		return "🧾"
	case EntityUser:
		return "🧑"
	case EntityReview:
		return "⭐"
	default:
		return "🔍"
	}
}
