package orders

import (
	"context"
	"sort"
	"strings"
	"time"
)

// StaticService provides deterministic order data suitable for local development and tests.
type StaticService struct {
	orders []Order
}

// NewStaticService returns a StaticService populated with representative orders.
func NewStaticService() *StaticService {
	now := time.Now()

	ptrTime := func(t time.Time) *time.Time {
		return &t
	}

	makeOrder := func(base Order) Order {
		// Ensure derived fields like status label/tone are populated when omitted.
		if strings.TrimSpace(base.StatusLabel) == "" {
			base.StatusLabel = defaultStatusLabel(base.Status)
		}
		if strings.TrimSpace(base.StatusTone) == "" {
			base.StatusTone = defaultStatusTone(base.Status)
		}
		return base
	}

	orders := []Order{
		makeOrder(Order{
			ID:          "order-1052",
			Number:      "1052",
			CreatedAt:   now.Add(-9 * time.Hour),
			UpdatedAt:   now.Add(-32 * time.Minute),
			Customer:    Customer{ID: "cust-8721", Name: "長谷川 純", Email: "jun.hasegawa@example.com"},
			TotalMinor:  3200000,
			Currency:    "JPY",
			Status:      StatusInProduction,
			StatusLabel: "制作中",
			Fulfillment: Fulfillment{
				Method:        "刻印工房",
				Carrier:       "工房出荷",
				PromisedDate:  ptrTime(now.Add(36 * time.Hour)),
				SLAStatus:     "制作残り 12 時間",
				SLAStatusTone: "info",
			},
			Payment: Payment{
				Status:     "支払い済み",
				StatusTone: "success",
				CapturedAt: ptrTime(now.Add(-8 * time.Hour)),
			},
			Tags:         []string{"刻印リング", "B2C"},
			Badges:       []Badge{{Label: "優先制作", Tone: "warning", Icon: "⚡"}, {Label: "VIP顧客", Tone: "info", Icon: "👑"}},
			ItemsSummary: "刻印リング（18K） × 2 / カスタム刻印",
			Notes:        []string{"刻印フォント: S-12", "納期短縮の希望あり"},
			SalesChannel: "オンラインストア",
			Integration:  "Shopify",
		}),
		makeOrder(Order{
			ID:          "order-1051",
			Number:      "1051",
			CreatedAt:   now.Add(-13 * time.Hour),
			UpdatedAt:   now.Add(-1 * time.Hour),
			Customer:    Customer{ID: "cust-8012", Name: "青木 里奈", Email: "rina.aoki@example.com"},
			TotalMinor:  1280000,
			Currency:    "JPY",
			Status:      StatusReadyToShip,
			StatusLabel: "出荷待ち",
			Fulfillment: Fulfillment{
				Method:        "宅配便",
				Carrier:       "ヤマト運輸",
				PromisedDate:  ptrTime(now.Add(18 * time.Hour)),
				SLAStatus:     "ピックアップ待ち",
				SLAStatusTone: "warning",
			},
			Payment: Payment{
				Status:     "支払い済み",
				StatusTone: "success",
				CapturedAt: ptrTime(now.Add(-10 * time.Hour)),
			},
			Tags:         []string{"ネックレス", "在庫"},
			Badges:       []Badge{{Label: "ギフト包装", Tone: "info", Icon: "🎁"}},
			ItemsSummary: "ペアネックレス（シルバー） × 1 / ギフトラッピング",
			SalesChannel: "オンラインストア",
			Integration:  "Shopify",
		}),
		makeOrder(Order{
			ID:          "order-1050",
			Number:      "1050",
			CreatedAt:   now.Add(-30 * time.Hour),
			UpdatedAt:   now.Add(-10 * time.Hour),
			Customer:    Customer{ID: "cust-7888", Name: "佐藤 真帆", Email: "maho.sato@example.com"},
			TotalMinor:  1840000,
			Currency:    "JPY",
			Status:      StatusShipped,
			StatusLabel: "発送済み",
			Fulfillment: Fulfillment{
				Method:        "宅配便",
				Carrier:       "ヤマト運輸",
				TrackingID:    "5543-2021-9921",
				DispatchedAt:  ptrTime(now.Add(-11 * time.Hour)),
				PromisedDate:  ptrTime(now.Add(12 * time.Hour)),
				SLAStatus:     "配送中",
				SLAStatusTone: "info",
			},
			Payment: Payment{
				Status:     "支払い済み",
				StatusTone: "success",
				CapturedAt: ptrTime(now.Add(-28 * time.Hour)),
			},
			Tags:         []string{"在庫", "標準"},
			Badges:       []Badge{{Label: "要配送フォロー", Tone: "warning", Icon: "📦"}},
			ItemsSummary: "カップルリング（シルバー） × 2",
			SalesChannel: "オンラインストア",
			Integration:  "Shopify",
		}),
		makeOrder(Order{
			ID:          "order-1049",
			Number:      "1049",
			CreatedAt:   now.Add(-72 * time.Hour),
			UpdatedAt:   now.Add(-20 * time.Hour),
			Customer:    Customer{ID: "cust-7420", Name: "松本 拓也", Email: "takuya.matsumoto@example.com"},
			TotalMinor:  5480000,
			Currency:    "JPY",
			Status:      StatusDelivered,
			StatusLabel: "納品済み",
			Fulfillment: Fulfillment{
				Method:        "宅配便",
				Carrier:       "佐川急便",
				TrackingID:    "3881-9932-5520",
				DispatchedAt:  ptrTime(now.Add(-36 * time.Hour)),
				DeliveredAt:   ptrTime(now.Add(-22 * time.Hour)),
				SLAStatus:     "期限内で完了",
				SLAStatusTone: "success",
			},
			Payment: Payment{
				Status:        "請求済み",
				StatusTone:    "info",
				CapturedAt:    ptrTime(now.Add(-40 * time.Hour)),
				DueAt:         ptrTime(now.Add(-16 * time.Hour)),
				PastDue:       false,
				PastDueReason: "",
			},
			Tags:         []string{"カスタム", "高額"},
			Badges:       []Badge{{Label: "制作完了", Tone: "success", Icon: "✅"}},
			ItemsSummary: "特注シグネットリング × 1 / 付属ケース",
			SalesChannel: "法人受注",
			Integration:  "電話受注",
		}),
		makeOrder(Order{
			ID:          "order-1048",
			Number:      "1048",
			CreatedAt:   now.Add(-26 * time.Hour),
			UpdatedAt:   now.Add(-2 * time.Hour),
			Customer:    Customer{ID: "cust-7011", Name: "小林 美咲", Email: "misaki.kobayashi@example.com"},
			TotalMinor:  2680000,
			Currency:    "JPY",
			Status:      StatusPaymentReview,
			StatusLabel: "支払い確認中",
			Fulfillment: Fulfillment{
				Method:        "制作前",
				PromisedDate:  ptrTime(now.Add(72 * time.Hour)),
				SLAStatus:     "支払い待ち",
				SLAStatusTone: "warning",
			},
			Payment: Payment{
				Status:        "審査中",
				StatusTone:    "warning",
				DueAt:         ptrTime(now.Add(-1 * time.Hour)),
				PastDue:       true,
				PastDueReason: "オフライン決済確認待ち",
			},
			Tags:             []string{"オフライン決済", "制作前"},
			Badges:           []Badge{{Label: "要支払いフォロー", Tone: "danger", Icon: "⚠️"}},
			ItemsSummary:     "オーダーメイド ネックレス × 1",
			SalesChannel:     "店舗受注",
			Integration:      "POS",
			HasRefundRequest: false,
		}),
		makeOrder(Order{
			ID:          "order-1047",
			Number:      "1047",
			CreatedAt:   now.Add(-48 * time.Hour),
			UpdatedAt:   now.Add(-5 * time.Hour),
			Customer:    Customer{ID: "cust-6892", Name: "Ilena Smith", Email: "ilena.smith@example.com"},
			TotalMinor:  4525000,
			Currency:    "USD",
			Status:      StatusShipped,
			StatusLabel: "発送済み",
			Fulfillment: Fulfillment{
				Method:        "国際配送",
				Carrier:       "FedEx",
				TrackingID:    "FEDEX-4488123",
				DispatchedAt:  ptrTime(now.Add(-18 * time.Hour)),
				SLAStatus:     "国際輸送中",
				SLAStatusTone: "info",
			},
			Payment: Payment{
				Status:     "支払い済み",
				StatusTone: "success",
				CapturedAt: ptrTime(now.Add(-44 * time.Hour)),
			},
			Tags:         []string{"海外", "USD"},
			Badges:       []Badge{{Label: "国際送料計算済み", Tone: "info", Icon: "🌐"}},
			ItemsSummary: "Custom Signet Ring × 1 / Gift Wrap",
			SalesChannel: "Etsy",
			Integration:  "Etsy",
		}),
		makeOrder(Order{
			ID:          "order-1046",
			Number:      "1046",
			CreatedAt:   now.Add(-6 * time.Hour),
			UpdatedAt:   now.Add(-30 * time.Minute),
			Customer:    Customer{ID: "cust-6552", Name: "田中 愛", Email: "ai.tanaka@example.com"},
			TotalMinor:  980000,
			Currency:    "JPY",
			Status:      StatusPendingPayment,
			StatusLabel: "支払い待ち",
			Fulfillment: Fulfillment{
				Method:        "制作前",
				SLAStatus:     "入金待ち",
				SLAStatusTone: "warning",
			},
			Payment: Payment{
				Status:        "未払い",
				StatusTone:    "warning",
				DueAt:         ptrTime(now.Add(12 * time.Hour)),
				PastDue:       false,
				PastDueReason: "",
			},
			Tags:         []string{"オンライン", "要フォロー"},
			Badges:       []Badge{{Label: "SMSリマインド予定", Tone: "info", Icon: "📱"}},
			ItemsSummary: "ペアブレスレット × 1",
			SalesChannel: "オンラインストア",
			Integration:  "Shopify",
		}),
		makeOrder(Order{
			ID:          "order-1045",
			Number:      "1045",
			CreatedAt:   now.Add(-96 * time.Hour),
			UpdatedAt:   now.Add(-6 * time.Hour),
			Customer:    Customer{ID: "cust-6021", Name: "鈴木 裕介", Email: "yusuke.suzuki@example.com"},
			TotalMinor:  3880000,
			Currency:    "JPY",
			Status:      StatusRefunded,
			StatusLabel: "返金済み",
			Fulfillment: Fulfillment{
				Method:        "制作なし",
				SLAStatus:     "返金完了",
				SLAStatusTone: "muted",
			},
			Payment: Payment{
				Status:     "返金済み",
				StatusTone: "info",
				CapturedAt: ptrTime(now.Add(-90 * time.Hour)),
			},
			Tags:             []string{"キャンセル"},
			Badges:           []Badge{{Label: "返金済み", Tone: "info", Icon: "↩︎"}},
			ItemsSummary:     "カスタムオーダー × 1",
			SalesChannel:     "法人受注",
			Integration:      "電話受注",
			HasRefundRequest: true,
		}),
		makeOrder(Order{
			ID:          "order-1044",
			Number:      "1044",
			CreatedAt:   now.Add(-40 * time.Hour),
			UpdatedAt:   now.Add(-3 * time.Hour),
			Customer:    Customer{ID: "cust-5777", Name: "村上 由美", Email: "yumi.murakami@example.com"},
			TotalMinor:  2150000,
			Currency:    "JPY",
			Status:      StatusInProduction,
			StatusLabel: "制作中",
			Fulfillment: Fulfillment{
				Method:        "刻印工房",
				PromisedDate:  ptrTime(now.Add(-1 * time.Hour)),
				SLAStatus:     "SLA遅延 5時間",
				SLAStatusTone: "danger",
			},
			Payment: Payment{
				Status:     "支払い済み",
				StatusTone: "success",
				CapturedAt: ptrTime(now.Add(-39 * time.Hour)),
			},
			Tags:             []string{"要フォロー", "返金申請"},
			Badges:           []Badge{{Label: "要優先対応", Tone: "danger", Icon: "🚩"}},
			ItemsSummary:     "ペンダントトップ（ゴールド） × 1",
			Notes:            []string{"顧客がSLA遅延に関する連絡済み"},
			SalesChannel:     "オンラインストア",
			Integration:      "Shopify",
			HasRefundRequest: true,
		}),
		makeOrder(Order{
			ID:          "order-1043",
			Number:      "1043",
			CreatedAt:   now.Add(-18 * time.Hour),
			UpdatedAt:   now.Add(-90 * time.Minute),
			Customer:    Customer{ID: "cust-5524", Name: "Carlos Diaz", Email: "carlos.diaz@example.com"},
			TotalMinor:  2755000,
			Currency:    "USD",
			Status:      StatusReadyToShip,
			StatusLabel: "出荷待ち",
			Fulfillment: Fulfillment{
				Method:        "国際配送",
				Carrier:       "UPS",
				PromisedDate:  ptrTime(now.Add(24 * time.Hour)),
				SLAStatus:     "輸出書類確認中",
				SLAStatusTone: "info",
			},
			Payment: Payment{
				Status:     "支払い済み",
				StatusTone: "success",
				CapturedAt: ptrTime(now.Add(-16 * time.Hour)),
			},
			Tags:         []string{"海外", "法人"},
			Badges:       []Badge{{Label: "商用インボイス必要", Tone: "warning", Icon: "📄"}},
			ItemsSummary: "Corporate Bulk Order × 5",
			SalesChannel: "Wholesale",
			Integration:  "NetSuite",
		}),
		makeOrder(Order{
			ID:          "order-1042",
			Number:      "1042",
			CreatedAt:   now.Add(-7 * 24 * time.Hour),
			UpdatedAt:   now.Add(-3 * 24 * time.Hour),
			Customer:    Customer{ID: "cust-5332", Name: "山田 直子", Email: "naoko.yamada@example.com"},
			TotalMinor:  1350000,
			Currency:    "JPY",
			Status:      StatusCancelled,
			StatusLabel: "キャンセル",
			Fulfillment: Fulfillment{
				Method:        "制作前",
				SLAStatus:     "キャンセル済み",
				SLAStatusTone: "muted",
			},
			Payment: Payment{
				Status:     "未請求",
				StatusTone: "muted",
			},
			Tags:         []string{"顧客都合"},
			Badges:       []Badge{{Label: "キャンセル", Tone: "muted", Icon: "✕"}},
			ItemsSummary: "名入れキーホルダー × 2",
			SalesChannel: "オンラインストア",
			Integration:  "Shopify",
		}),
		makeOrder(Order{
			ID:          "order-1041",
			Number:      "1041",
			CreatedAt:   now.Add(-5 * 24 * time.Hour),
			UpdatedAt:   now.Add(-18 * time.Hour),
			Customer:    Customer{ID: "cust-5200", Name: "エミリー 王", Email: "emily.wang@example.com"},
			TotalMinor:  2980000,
			Currency:    "JPY",
			Status:      StatusDelivered,
			StatusLabel: "納品済み",
			Fulfillment: Fulfillment{
				Method:        "宅配便",
				Carrier:       "ヤマト運輸",
				TrackingID:    "1182-5521-9982",
				DispatchedAt:  ptrTime(now.Add(-42 * time.Hour)),
				DeliveredAt:   ptrTime(now.Add(-20 * time.Hour)),
				SLAStatus:     "早期配達",
				SLAStatusTone: "success",
			},
			Payment: Payment{
				Status:     "支払い済み",
				StatusTone: "success",
				CapturedAt: ptrTime(now.Add(-4 * 24 * time.Hour)),
			},
			Tags:         []string{"在庫", "通常"},
			Badges:       []Badge{{Label: "レビュー依頼済み", Tone: "info", Icon: "⭐"}},
			ItemsSummary: "スターリングシルバーリング × 1 / サイズ調整",
			SalesChannel: "オンラインストア",
			Integration:  "Shopify",
		}),
	}

	return &StaticService{orders: orders}
}

// List implements the orders Service interface.
func (s *StaticService) List(_ context.Context, _ string, query Query) (ListResult, error) {
	withStatus := s.filterOrders(query, true)

	sortOrders(withStatus, query)

	total := len(withStatus)
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	page := query.Page
	if page <= 0 {
		page = 1
	}

	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	pageOrders := append([]Order(nil), withStatus[start:end]...)

	pagination := Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
	}
	if end < total {
		next := page + 1
		pagination.NextPage = &next
	}
	if page > 1 && start <= total {
		prev := page - 1
		if prev >= 1 {
			pagination.PrevPage = &prev
		}
	}

	summary := buildSummary(withStatus)

	filters := s.buildFilterSummary(query)

	return ListResult{
		Orders:     pageOrders,
		Pagination: pagination,
		Summary:    summary,
		Filters:    filters,
	}, nil
}

func (s *StaticService) filterOrders(query Query, includeStatus bool) []Order {
	results := make([]Order, 0, len(s.orders))

	statusSet := map[Status]bool{}
	if includeStatus && len(query.Statuses) > 0 {
		for _, status := range query.Statuses {
			statusSet[status] = true
		}
	}

	for _, order := range s.orders {
		if includeStatus && len(statusSet) > 0 && !statusSet[order.Status] {
			continue
		}
		if query.Since != nil && order.UpdatedAt.Before(*query.Since) {
			continue
		}
		if strings.TrimSpace(query.Currency) != "" && !strings.EqualFold(order.Currency, strings.TrimSpace(query.Currency)) {
			continue
		}
		if query.AmountMin != nil && order.TotalMinor < *query.AmountMin {
			continue
		}
		if query.AmountMax != nil && order.TotalMinor > *query.AmountMax {
			continue
		}
		if query.HasRefundOnly != nil {
			if *query.HasRefundOnly && !order.HasRefundRequest {
				continue
			}
			if !*query.HasRefundOnly && order.HasRefundRequest {
				continue
			}
		}
		results = append(results, order)
	}

	return results
}

func sortOrders(orders []Order, query Query) {
	sortKey := strings.ToLower(strings.TrimSpace(query.SortKey))
	sortDir := strings.ToLower(string(query.SortDirection))
	desc := true
	if sortDir == string(SortDirectionAsc) {
		desc = false
	}

	if sortKey == "" {
		sortKey = "updated_at"
	}

	sort.SliceStable(orders, func(i, j int) bool {
		a := orders[i]
		b := orders[j]

		var less bool
		switch sortKey {
		case "total":
			if a.TotalMinor == b.TotalMinor {
				less = a.Number < b.Number
			} else {
				less = a.TotalMinor < b.TotalMinor
			}
		case "number":
			less = a.Number < b.Number
		default: // updated_at
			less = a.UpdatedAt.Before(b.UpdatedAt)
		}

		if desc {
			return !less
		}
		return less
	})
}

func buildSummary(orders []Order) Summary {
	var revenue int64
	var leadHours float64
	var leadCount float64
	var delayed int
	var refunds int
	var inProduction int
	var fulfilled24h int

	now := time.Now()

	for _, order := range orders {
		revenue += order.TotalMinor
		if order.Fulfillment.SLAStatusTone == "danger" {
			delayed++
		}
		if order.HasRefundRequest {
			refunds++
		}
		if order.Status == StatusInProduction {
			inProduction++
		}
		if order.Fulfillment.DispatchedAt != nil {
			lead := order.Fulfillment.DispatchedAt.Sub(order.CreatedAt).Hours()
			if lead < 0 {
				lead = 0
			}
			leadHours += lead
			leadCount++
			if now.Sub(*order.Fulfillment.DispatchedAt) <= 24*time.Hour {
				fulfilled24h++
			}
		}
	}

	avgLead := 0.0
	if leadCount > 0 {
		avgLead = leadHours / leadCount
	}

	distribution := statusDistribution(orders)

	return Summary{
		TotalOrders:        len(orders),
		TotalRevenueMinor:  revenue,
		AverageLeadHours:   avgLead,
		DelayedCount:       delayed,
		RefundRequested:    refunds,
		InProductionCount:  inProduction,
		FulfilledLast24h:   fulfilled24h,
		LastRefreshedAt:    now,
		PrimaryCurrency:    primaryCurrency(orders),
		StatusDistribution: distribution,
	}
}

func statusDistribution(orders []Order) []StatusCount {
	counts := map[Status]int{}
	for _, order := range orders {
		counts[order.Status]++
	}
	allStatuses := []Status{
		StatusPendingPayment,
		StatusPaymentReview,
		StatusInProduction,
		StatusReadyToShip,
		StatusShipped,
		StatusDelivered,
		StatusRefunded,
		StatusCancelled,
	}

	result := make([]StatusCount, 0, len(allStatuses))
	for _, st := range allStatuses {
		result = append(result, StatusCount{Status: st, Count: counts[st]})
	}
	return result
}

func primaryCurrency(orders []Order) string {
	counts := map[string]int{}
	var best string
	bestCount := -1
	for _, order := range orders {
		cur := strings.ToUpper(strings.TrimSpace(order.Currency))
		if cur == "" {
			continue
		}
		counts[cur]++
		if counts[cur] > bestCount {
			best = cur
			bestCount = counts[cur]
		}
	}
	if best == "" {
		return "JPY"
	}
	return best
}

func (s *StaticService) buildFilterSummary(query Query) FilterSummary {
	withoutStatus := s.filterOrders(query, false)

	statusCounts := map[Status]int{}
	for _, order := range withoutStatus {
		statusCounts[order.Status]++
	}

	statusOptions := []StatusOption{
		{Value: StatusPendingPayment, Label: "支払い待ち"},
		{Value: StatusPaymentReview, Label: "支払い確認中"},
		{Value: StatusInProduction, Label: "制作中"},
		{Value: StatusReadyToShip, Label: "出荷待ち"},
		{Value: StatusShipped, Label: "発送済み"},
		{Value: StatusDelivered, Label: "納品済み"},
		{Value: StatusRefunded, Label: "返金済み"},
		{Value: StatusCancelled, Label: "キャンセル"},
	}
	for i := range statusOptions {
		statusOptions[i].Count = statusCounts[statusOptions[i].Value]
		statusOptions[i].Description = statusOptions[i].Label
	}

	currencyCounts := map[string]int{}
	for _, order := range withoutStatus {
		code := strings.ToUpper(strings.TrimSpace(order.Currency))
		if code == "" {
			continue
		}
		currencyCounts[code]++
	}
	currencyOptions := make([]CurrencyOption, 0, len(currencyCounts))
	for code, count := range currencyCounts {
		label := code
		if code == "JPY" {
			label = "JPY（日本円）"
		} else if code == "USD" {
			label = "USD（米ドル）"
		}
		currencyOptions = append(currencyOptions, CurrencyOption{
			Code:  code,
			Label: label,
			Count: count,
		})
	}
	sort.Slice(currencyOptions, func(i, j int) bool {
		return currencyOptions[i].Code < currencyOptions[j].Code
	})

	refundOptions := []RefundOption{
		{Value: "", Label: "すべて"},
		{Value: "true", Label: "返金申請あり"},
		{Value: "false", Label: "返金申請なし"},
	}

	amountRanges := []AmountRange{
		{Label: "¥0 - ¥10,000", Min: int64Ptr(0), Max: int64Ptr(1000000)},
		{Label: "¥10,000 - ¥30,000", Min: int64Ptr(1000000), Max: int64Ptr(3000000)},
		{Label: "¥30,000+", Min: int64Ptr(3000000), Max: nil},
	}

	return FilterSummary{
		StatusOptions:   statusOptions,
		CurrencyOptions: currencyOptions,
		RefundOptions:   refundOptions,
		AmountRanges:    amountRanges,
	}
}

func defaultStatusLabel(status Status) string {
	switch status {
	case StatusPendingPayment:
		return "支払い待ち"
	case StatusPaymentReview:
		return "支払い確認中"
	case StatusInProduction:
		return "制作中"
	case StatusReadyToShip:
		return "出荷待ち"
	case StatusShipped:
		return "発送済み"
	case StatusDelivered:
		return "納品済み"
	case StatusRefunded:
		return "返金済み"
	case StatusCancelled:
		return "キャンセル"
	default:
		return "その他"
	}
}

func defaultStatusTone(status Status) string {
	switch status {
	case StatusPendingPayment, StatusPaymentReview:
		return "warning"
	case StatusInProduction, StatusReadyToShip:
		return "info"
	case StatusShipped:
		return "info"
	case StatusDelivered:
		return "success"
	case StatusRefunded, StatusCancelled:
		return "muted"
	default:
		return "info"
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}
