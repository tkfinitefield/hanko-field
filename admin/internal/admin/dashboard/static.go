package dashboard

import (
	"context"
	"time"
)

// StaticService provides canned responses for development and tests.
type StaticService struct {
	KPIs     []KPI
	Alerts   []Alert
	Activity []ActivityItem
}

// NewStaticService returns a StaticService populated with sample data if none supplied.
func NewStaticService() *StaticService {
	now := time.Now()
	defaultKPIs := []KPI{
		{
			ID:        "revenue",
			Label:     "日次売上",
			Value:     "¥3,420,000",
			DeltaText: "+12% vs 昨日",
			Trend:     TrendUp,
			Sparkline: []float64{120, 140, 160, 180, 190, 210, 230},
			UpdatedAt: now,
		},
		{
			ID:        "orders",
			Label:     "注文数",
			Value:     "128",
			DeltaText: "+8件",
			Trend:     TrendUp,
			Sparkline: []float64{24, 20, 18, 22, 25, 19, 28},
			UpdatedAt: now,
		},
		{
			ID:        "csat",
			Label:     "CSAT",
			Value:     "4.8 / 5",
			DeltaText: "-0.1 vs 先週",
			Trend:     TrendDown,
			Sparkline: []float64{4.2, 4.4, 4.7, 4.9, 5.0, 4.8, 4.8},
			UpdatedAt: now,
		},
		{
			ID:        "production",
			Label:     "制作WIP",
			Value:     "36",
			DeltaText: "4件待ち",
			Trend:     TrendFlat,
			Sparkline: []float64{30, 32, 35, 34, 33, 36, 36},
			UpdatedAt: now,
		},
	}

	defaultAlerts := []Alert{
		{
			ID:        "alert-low-inventory",
			Severity:  "warning",
			Title:     "在庫残りわずか",
			Message:   "人気SKU「KF-2024-GD」の残数が 12 個になりました。",
			ActionURL: "/admin/catalog/products?sku=KF-2024-GD",
			Action:    "在庫を補充",
			CreatedAt: now.Add(-30 * time.Minute),
		},
		{
			ID:        "alert-shipment-delay",
			Severity:  "danger",
			Title:     "配送遅延アラート",
			Message:   "ヤマト運輸 6 件が遅延ステータスです。お客様への連絡が必要です。",
			ActionURL: "/admin/shipments/tracking?status=delayed",
			Action:    "詳細を確認",
			CreatedAt: now.Add(-2 * time.Hour),
		},
		{
			ID:        "alert-support",
			Severity:  "info",
			Title:     "サポート未返信",
			Message:   "高優先度の問い合わせが 3 件あります。",
			ActionURL: "/admin/notifications?category=support",
			Action:    "チケットを開く",
			CreatedAt: now.Add(-5 * time.Hour),
		},
	}

	defaultActivity := []ActivityItem{
		{
			ID:       "activity-order",
			Icon:     "📦",
			Title:    "注文 #1045 を出荷しました",
			Detail:   "佐藤様 / ヤマト運輸（翌日配達）",
			Occurred: now.Add(-12 * time.Minute),
			LinkURL:  "/admin/orders/1045",
		},
		{
			ID:       "activity-production",
			Icon:     "🛠",
			Title:    "制作ライン - 彫刻完了",
			Detail:   "オーダー #1038 の彫刻工程が完了しました。",
			Occurred: now.Add(-45 * time.Minute),
			LinkURL:  "/admin/production/queues",
		},
		{
			ID:       "activity-review",
			Icon:     "⭐️",
			Title:    "レビュー承認待ち",
			Detail:   "4.5 ★ / 署名刻印リング",
			Occurred: now.Add(-2 * time.Hour),
			LinkURL:  "/admin/reviews?moderation=pending",
		},
	}

	return &StaticService{
		KPIs:     defaultKPIs,
		Alerts:   defaultAlerts,
		Activity: defaultActivity,
	}
}

// FetchKPIs returns configured KPI cards.
func (s *StaticService) FetchKPIs(ctx context.Context, token string, since *time.Time) ([]KPI, error) {
	if len(s.KPIs) == 0 {
		return []KPI{}, nil
	}
	return s.KPIs, nil
}

// FetchAlerts returns configured alert entries.
func (s *StaticService) FetchAlerts(ctx context.Context, token string, limit int) ([]Alert, error) {
	if limit > 0 && len(s.Alerts) > limit {
		return s.Alerts[:limit], nil
	}
	return s.Alerts, nil
}

// FetchActivity returns configured activity feed entries.
func (s *StaticService) FetchActivity(ctx context.Context, token string, limit int) ([]ActivityItem, error) {
	if limit > 0 && len(s.Activity) > limit {
		return s.Activity[:limit], nil
	}
	return s.Activity, nil
}
