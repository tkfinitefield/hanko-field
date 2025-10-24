package orders

import (
	"context"
	"fmt"
	"net/mail"
	"sort"
	"strings"
	"sync"
	"time"
)

// StaticService provides deterministic order data suitable for local development and tests.
type StaticService struct {
	mu               sync.RWMutex
	orders           []Order
	timelines        map[string][]TimelineEvent
	invoiceJobs      map[string]*invoiceJobState
	exportJobs       map[string]*exportJobState
	invoiceTemplates []InvoiceTemplate
	invoiceLanguages []InvoiceLanguage
	audit            AuditLogger
}

// NewStaticService returns a StaticService populated with representative orders.
func NewStaticService() *StaticService {
	now := time.Now()

	ptrTime := func(t time.Time) *time.Time {
		return &t
	}

	makePaymentDetail := func(id, provider, method, last4, reference, status, tone, currency string, authorized, captured, refunded int64, capturedAt *time.Time) PaymentDetail {
		if strings.TrimSpace(provider) == "" {
			provider = "Stripe"
		}
		if strings.TrimSpace(method) == "" {
			method = "クレジットカード"
		}
		if strings.TrimSpace(reference) == "" {
			reference = id
		}
		if strings.TrimSpace(status) == "" {
			status = "支払い済み"
		}
		if strings.TrimSpace(currency) == "" {
			currency = "JPY"
		}
		if authorized < captured {
			authorized = captured
		}
		available := captured - refunded
		if available < 0 {
			available = 0
		}
		if refunded < 0 {
			refunded = 0
		}
		return PaymentDetail{
			ID:               strings.TrimSpace(id),
			Provider:         strings.TrimSpace(provider),
			Method:           strings.TrimSpace(method),
			Last4:            strings.TrimSpace(last4),
			Reference:        strings.TrimSpace(reference),
			Status:           strings.TrimSpace(status),
			StatusTone:       strings.TrimSpace(tone),
			Currency:         strings.TrimSpace(currency),
			AmountAuthorized: authorized,
			AmountCaptured:   captured,
			AmountRefunded:   refunded,
			AmountAvailable:  available,
			CapturedAt:       capturedAt,
		}
	}

	makeRefundRecord := func(id, paymentID string, amount int64, currency, reason, status, actor string, processed time.Time) RefundRecord {
		if strings.TrimSpace(id) == "" {
			id = fmt.Sprintf("refund_%s_%d", paymentID, processed.Unix())
		}
		if processed.IsZero() {
			processed = time.Now()
		}
		if strings.TrimSpace(status) == "" {
			status = "succeeded"
		}
		if strings.TrimSpace(currency) == "" {
			currency = "JPY"
		}
		if strings.TrimSpace(actor) == "" {
			actor = "オペレーター"
		}
		return RefundRecord{
			ID:          strings.TrimSpace(id),
			PaymentID:   strings.TrimSpace(paymentID),
			AmountMinor: amount,
			Currency:    strings.TrimSpace(currency),
			Reason:      strings.TrimSpace(reason),
			Status:      strings.TrimSpace(status),
			ProcessedAt: processed,
			Actor:       strings.TrimSpace(actor),
			Reference:   fmt.Sprintf("%s-ref", strings.TrimSpace(id)),
		}
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
			Payments: []PaymentDetail{
				makePaymentDetail(
					"pay-1052",
					"Stripe",
					"クレジットカード",
					"4242",
					"pay_1052",
					"支払い済み",
					"success",
					"JPY",
					3200000,
					3200000,
					0,
					ptrTime(now.Add(-8*time.Hour)),
				),
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
			Payments: []PaymentDetail{
				makePaymentDetail(
					"pay-1051",
					"Stripe",
					"クレジットカード",
					"1881",
					"pay_1051",
					"支払い済み",
					"success",
					"JPY",
					1280000,
					1280000,
					0,
					ptrTime(now.Add(-10*time.Hour)),
				),
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
			Payments: []PaymentDetail{
				makePaymentDetail(
					"pay-1050",
					"Stripe",
					"クレジットカード",
					"5210",
					"pay_1050",
					"支払い済み",
					"success",
					"JPY",
					1840000,
					1840000,
					0,
					ptrTime(now.Add(-28*time.Hour)),
				),
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
			Payments: []PaymentDetail{
				makePaymentDetail(
					"pay-1049",
					"Stripe",
					"銀行振込",
					"",
					"pay_1049",
					"請求済み",
					"info",
					"JPY",
					5480000,
					5480000,
					0,
					ptrTime(now.Add(-40*time.Hour)),
				),
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
			Payments: []PaymentDetail{
				makePaymentDetail(
					"pay-1048",
					"オフライン決済",
					"銀行振込",
					"",
					"pay_1048",
					"審査中",
					"warning",
					"JPY",
					2680000,
					0,
					0,
					nil,
				),
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
			Payments: []PaymentDetail{
				makePaymentDetail(
					"pay-1047",
					"Stripe",
					"クレジットカード",
					"7788",
					"pay_1047",
					"支払い済み",
					"success",
					"USD",
					4525000,
					4525000,
					625000,
					ptrTime(now.Add(-44*time.Hour)),
				),
			},
			Refunds: []RefundRecord{
				makeRefundRecord(
					"refund-1047-1",
					"pay-1047",
					625000,
					"USD",
					"サイズ再調整の差額返金",
					"succeeded",
					"support@hanko.example",
					now.Add(-12*time.Hour),
				),
			},
			Tags:             []string{"海外", "USD"},
			Badges:           []Badge{{Label: "国際送料計算済み", Tone: "info", Icon: "🌐"}},
			ItemsSummary:     "Custom Signet Ring × 1 / Gift Wrap",
			SalesChannel:     "Etsy",
			Integration:      "Etsy",
			HasRefundRequest: true,
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
			Payments: []PaymentDetail{
				makePaymentDetail(
					"pay-1046",
					"Stripe",
					"クレジットカード",
					"3005",
					"pay_1046",
					"未払い",
					"warning",
					"JPY",
					980000,
					0,
					0,
					nil,
				),
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
			Payments: []PaymentDetail{
				makePaymentDetail(
					"pay-1045",
					"Stripe",
					"クレジットカード",
					"9900",
					"pay_1045",
					"返金済み",
					"info",
					"JPY",
					3880000,
					3880000,
					3880000,
					ptrTime(now.Add(-90*time.Hour)),
				),
			},
			Refunds: []RefundRecord{
				makeRefundRecord(
					"refund-1045-1",
					"pay-1045",
					3880000,
					"JPY",
					"顧客キャンセルによる全額返金",
					"succeeded",
					"finance@hanko.example",
					now.Add(-6*time.Hour),
				),
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
			Payments: []PaymentDetail{
				makePaymentDetail(
					"pay-1044",
					"Stripe",
					"クレジットカード",
					"5561",
					"pay_1044",
					"支払い済み",
					"success",
					"JPY",
					2150000,
					2150000,
					500000,
					ptrTime(now.Add(-39*time.Hour)),
				),
			},
			Refunds: []RefundRecord{
				makeRefundRecord(
					"refund-1044-1",
					"pay-1044",
					500000,
					"JPY",
					"SLA遅延による補償",
					"processing",
					"support@hanko.example",
					now.Add(-2*time.Hour),
				),
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
			Payments: []PaymentDetail{
				makePaymentDetail(
					"pay-1043",
					"Stripe",
					"クレジットカード",
					"4431",
					"pay_1043",
					"支払い済み",
					"success",
					"USD",
					2755000,
					2755000,
					0,
					ptrTime(now.Add(-16*time.Hour)),
				),
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
			Payments: []PaymentDetail{
				makePaymentDetail(
					"pay-1042",
					"Stripe",
					"クレジットカード",
					"2210",
					"pay_1042",
					"未請求",
					"muted",
					"JPY",
					1350000,
					0,
					0,
					nil,
				),
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
			Payments: []PaymentDetail{
				makePaymentDetail(
					"pay-1041",
					"Stripe",
					"クレジットカード",
					"6622",
					"pay_1041",
					"支払い済み",
					"success",
					"JPY",
					2980000,
					2980000,
					0,
					ptrTime(now.Add(-4*24*time.Hour)),
				),
			},
			Tags:         []string{"在庫", "通常"},
			Badges:       []Badge{{Label: "レビュー依頼済み", Tone: "info", Icon: "⭐"}},
			ItemsSummary: "スターリングシルバーリング × 1 / サイズ調整",
			SalesChannel: "オンラインストア",
			Integration:  "Shopify",
		}),
	}

	for i := range orders {
		switch orders[i].ID {
		case "order-1052":
			orders[i].Invoices = []InvoiceRecord{
				{
					ID:            "inv-order-1052-001",
					Number:        "INV-2025-001",
					Status:        "発行済み",
					StatusTone:    "success",
					IssuedAt:      now.Add(-6 * time.Hour),
					DeliveryEmail: "jun.hasegawa@example.com",
					Note:          "制作開始時に送付",
					Actor:         "finance@hanko.example",
					PDFURL:        "https://storage.example.com/invoices/INV-2025-001.pdf",
					TemplateID:    "invoice-standard",
					Language:      "ja-JP",
					CreatedAt:     now.Add(-6 * time.Hour),
					UpdatedAt:     now.Add(-6 * time.Hour),
				},
			}
		case "order-1050":
			orders[i].Invoices = []InvoiceRecord{
				{
					ID:            "inv-order-1050-001",
					Number:        "INV-2025-045",
					Status:        "発行準備中",
					StatusTone:    "info",
					DeliveryEmail: "maho.sato@example.com",
					Note:          "法人請求書 (バッチ)",
					Actor:         "ops@hanko.example",
					TemplateID:    "invoice-batch",
					Language:      "ja-JP",
					JobID:         "job-invoice-1050-1",
					CreatedAt:     now.Add(-10 * time.Minute),
					UpdatedAt:     now.Add(-10 * time.Minute),
				},
			}
		case "order-1047":
			orders[i].Invoices = []InvoiceRecord{
				{
					ID:            "inv-order-1047-001",
					Number:        "INV-2025-019",
					Status:        "発行済み",
					StatusTone:    "success",
					IssuedAt:      now.Add(-12 * time.Hour),
					DeliveryEmail: "ilena.smith@example.com",
					Note:          "International shipment invoice",
					Actor:         "finance@hanko.example",
					PDFURL:        "https://storage.example.com/invoices/INV-2025-019.pdf",
					TemplateID:    "invoice-standard",
					Language:      "en-US",
					CreatedAt:     now.Add(-12 * time.Hour),
					UpdatedAt:     now.Add(-12 * time.Hour),
				},
			}
		case "order-1045":
			orders[i].Invoices = []InvoiceRecord{
				{
					ID:            "inv-order-1045-001",
					Number:        "INV-2024-312",
					Status:        "キャンセル済み",
					StatusTone:    "muted",
					IssuedAt:      now.Add(-80 * time.Hour),
					DeliveryEmail: "yusuke.suzuki@example.com",
					Note:          "返金対応のため無効化済み",
					Actor:         "finance@hanko.example",
					PDFURL:        "",
					TemplateID:    "invoice-detailed",
					Language:      "ja-JP",
					CreatedAt:     now.Add(-81 * time.Hour),
					UpdatedAt:     now.Add(-80 * time.Hour),
				},
			}
		}
	}

	templates := []InvoiceTemplate{
		{
			ID:          "invoice-standard",
			Label:       "標準テンプレート（日本語）",
			Description: "一般的な個人向けフォーマット。税込表示と注文番号を含みます。",
			Default:     true,
		},
		{
			ID:          "invoice-detailed",
			Label:       "明細付きテンプレート",
			Description: "ラインアイテムと備考を細かく記載します。",
			Default:     false,
		},
		{
			ID:          "invoice-batch",
			Label:       "法人請求（バッチ生成）",
			Description: "大量発行向けテンプレート。生成に数分かかる場合があります。",
			Default:     false,
		},
	}

	languages := []InvoiceLanguage{
		{Code: "ja-JP", Label: "日本語", Default: true},
		{Code: "en-US", Label: "English", Default: false},
	}

	invoiceJobs := make(map[string]*invoiceJobState)
	for _, order := range orders {
		for _, invoice := range order.Invoices {
			jobID := strings.TrimSpace(invoice.JobID)
			if jobID == "" {
				continue
			}
			state := &invoiceJobState{
				OrderID:   order.ID,
				InvoiceID: invoice.ID,
				Job: InvoiceJob{
					ID:          jobID,
					Status:      "キュー投入済み",
					StatusTone:  "info",
					SubmittedAt: invoice.CreatedAt,
					Message:     "バッチ処理でPDFを生成中です（通常1-2分程度）。",
				},
				Completed: false,
			}
			invoiceJobs[jobID] = state
		}
	}

	timelines := make(map[string][]TimelineEvent, len(orders))
	for _, order := range orders {
		timelines[order.ID] = seedTimeline(order)
	}

	exportJobs := make(map[string]*exportJobState)

	return &StaticService{
		orders:           orders,
		timelines:        timelines,
		invoiceJobs:      invoiceJobs,
		exportJobs:       exportJobs,
		invoiceTemplates: templates,
		invoiceLanguages: languages,
		audit:            noopAuditLogger{},
	}
}

var exportAllowedFields = map[ExportFormat][]string{
	ExportFormatCSV: {
		"order_id",
		"order_number",
		"status",
		"total_minor",
		"currency",
		"customer_name",
		"created_at",
	},
	ExportFormatPDF: {
		"order_number",
		"customer_name",
		"items",
		"total_minor",
		"currency",
		"fulfillment",
	},
}

func (s *StaticService) snapshotOrders() []Order {
	s.mu.RLock()
	defer s.mu.RUnlock()
	copy := make([]Order, len(s.orders))
	for i, order := range s.orders {
		copy[i] = cloneOrder(order)
	}
	return copy
}

func cloneOrder(order Order) Order {
	result := order
	if len(order.Tags) > 0 {
		result.Tags = append([]string(nil), order.Tags...)
	}
	if len(order.Badges) > 0 {
		result.Badges = append([]Badge(nil), order.Badges...)
	}
	if len(order.Notes) > 0 {
		result.Notes = append([]string(nil), order.Notes...)
	}
	if len(order.Payments) > 0 {
		result.Payments = append([]PaymentDetail(nil), order.Payments...)
	}
	if len(order.Refunds) > 0 {
		result.Refunds = append([]RefundRecord(nil), order.Refunds...)
	}
	if len(order.Invoices) > 0 {
		result.Invoices = append([]InvoiceRecord(nil), order.Invoices...)
	}
	return result
}

func cloneTimeline(events []TimelineEvent) []TimelineEvent {
	if len(events) == 0 {
		return nil
	}
	cloned := make([]TimelineEvent, len(events))
	copy(cloned, events)
	return cloned
}

func cloneExportJob(job ExportJob) ExportJob {
	copied := job
	if job.CompletedAt != nil {
		ts := *job.CompletedAt
		copied.CompletedAt = &ts
	}
	if len(job.Fields) > 0 {
		copied.Fields = append([]string(nil), job.Fields...)
	}
	return copied
}

func normaliseExportFormat(format ExportFormat) ExportFormat {
	switch strings.ToLower(strings.TrimSpace(string(format))) {
	case string(ExportFormatPDF):
		return ExportFormatPDF
	default:
		return ExportFormatCSV
	}
}

func exportQueuedMessage(format ExportFormat) string {
	switch format {
	case ExportFormatPDF:
		return "印刷用PDFを生成キューに投入しました。準備ができ次第ダウンロードできます。"
	default:
		return "CSVエクスポートを準備しています。完了後にダウンロードリンクが表示されます。"
	}
}

func exportProcessingMessage(format ExportFormat) string {
	switch format {
	case ExportFormatPDF:
		return "PDFを生成しています。完了までしばらくお待ちください。"
	default:
		return "CSVを書き出しています。完了までしばらくお待ちください。"
	}
}

func exportCompletedMessage(format ExportFormat) string {
	switch format {
	case ExportFormatPDF:
		return "PDFの生成が完了しました。ダウンロードリンクから印刷できます。"
	default:
		return "CSVエクスポートが完了しました。ダウンロードリンクから取得できます。"
	}
}

func buildExportDownloadURL(format ExportFormat, jobID string) string {
	safeID := strings.TrimSpace(jobID)
	if safeID == "" {
		return ""
	}
	safeID = strings.ReplaceAll(safeID, " ", "_")
	switch format {
	case ExportFormatPDF:
		return fmt.Sprintf("https://storage.example.com/orders/exports/%s.pdf", safeID)
	default:
		return fmt.Sprintf("https://storage.example.com/orders/exports/%s.csv", safeID)
	}
}

// List implements the orders Service interface.
func (s *StaticService) List(_ context.Context, _ string, query Query) (ListResult, error) {
	orders := s.snapshotOrders()
	withStatus := filterOrders(orders, query, true)

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

	filters := buildFilterSummary(orders, query)

	return ListResult{
		Orders:     pageOrders,
		Pagination: pagination,
		Summary:    summary,
		Filters:    filters,
	}, nil
}

// StatusModal assembles modal data for the specified order.
func (s *StaticService) StatusModal(_ context.Context, _ string, orderID string) (StatusModal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, order := s.findOrderLocked(orderID)
	if order == nil {
		return StatusModal{}, ErrOrderNotFound
	}

	orderCopy := cloneOrder(*order)
	choices := buildStatusChoices(orderCopy.Status)
	events := cloneTimeline(s.timelines[orderID])
	if len(events) > 5 {
		events = events[len(events)-5:]
	}

	return StatusModal{
		Order:          orderCopy,
		Choices:        choices,
		LatestTimeline: events,
	}, nil
}

// RefundModal assembles refund modal data for the specified order.
func (s *StaticService) RefundModal(_ context.Context, _ string, orderID string) (RefundModal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, order := s.findOrderLocked(orderID)
	if order == nil {
		return RefundModal{}, ErrOrderNotFound
	}

	cloned := cloneOrder(*order)

	options := make([]RefundPaymentOption, 0, len(cloned.Payments))
	for _, payment := range cloned.Payments {
		options = append(options, toRefundPaymentOption(payment))
	}

	supportsPartial := false
	for _, option := range options {
		if option.AvailableMinor > 0 {
			supportsPartial = true
			break
		}
	}

	existing := append([]RefundRecord(nil), cloned.Refunds...)

	outstanding := ""
	if cloned.Payment.PastDue {
		outstanding = strings.TrimSpace(cloned.Payment.PastDueReason)
		if outstanding == "" {
			outstanding = "支払い確認中"
		}
	}

	summary := RefundOrderSummary{
		ID:             cloned.ID,
		Number:         cloned.Number,
		CustomerName:   cloned.Customer.Name,
		TotalMinor:     cloned.TotalMinor,
		Currency:       cloned.Currency,
		PaymentStatus:  cloned.Payment.Status,
		PaymentTone:    cloned.Payment.StatusTone,
		OutstandingDue: outstanding,
	}

	return RefundModal{
		Order:           summary,
		Payments:        options,
		ExistingRefunds: existing,
		SupportsPartial: supportsPartial,
		Currency:        cloned.Currency,
	}, nil
}

// InvoiceModal assembles invoice modal data for the specified order.
func (s *StaticService) InvoiceModal(_ context.Context, _ string, orderID string) (InvoiceModal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, order := s.findOrderLocked(orderID)
	if order == nil {
		return InvoiceModal{}, ErrOrderNotFound
	}

	cloned := cloneOrder(*order)

	recent := append([]InvoiceRecord(nil), cloned.Invoices...)
	sort.SliceStable(recent, func(i, j int) bool {
		return recent[i].UpdatedAt.After(recent[j].UpdatedAt)
	})

	templates := append([]InvoiceTemplate(nil), s.invoiceTemplates...)
	languages := append([]InvoiceLanguage(nil), s.invoiceLanguages...)

	defaultTemplate := defaultInvoiceTemplateID(templates)
	defaultLanguage := defaultInvoiceLanguageCode(languages)

	summary := InvoiceOrderSummary{
		ID:            cloned.ID,
		Number:        cloned.Number,
		CustomerName:  cloned.Customer.Name,
		CustomerEmail: cloned.Customer.Email,
		Currency:      cloned.Currency,
		TotalMinor:    cloned.TotalMinor,
	}

	return InvoiceModal{
		Order:           summary,
		Templates:       templates,
		Languages:       languages,
		SuggestedEmail:  strings.TrimSpace(cloned.Customer.Email),
		RecentInvoices:  recent,
		DefaultTemplate: defaultTemplate,
		DefaultLanguage: defaultLanguage,
	}, nil
}

// IssueInvoice records a simulated invoice issuance for development usage.
func (s *StaticService) IssueInvoice(_ context.Context, _ string, req InvoiceIssueRequest) (InvoiceIssueResult, error) {
	orderID := strings.TrimSpace(req.OrderID)
	templateID := strings.TrimSpace(req.TemplateID)
	email := strings.TrimSpace(req.DeliveryEmail)
	language := strings.TrimSpace(req.Language)
	note := strings.TrimSpace(req.Note)

	fieldErrors := map[string]string{}
	if orderID == "" {
		fieldErrors["orderID"] = "注文IDが不正です。"
	}
	if templateID == "" {
		fieldErrors["templateID"] = "テンプレートを選択してください。"
	}
	if email == "" {
		fieldErrors["email"] = "送付先メールアドレスを入力してください。"
	} else if !isValidEmail(email) {
		fieldErrors["email"] = "メールアドレスの形式が正しくありません。"
	}

	if len(fieldErrors) > 0 {
		return InvoiceIssueResult{}, &InvoiceValidationError{
			Message:     "入力内容を確認してください。",
			FieldErrors: fieldErrors,
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx, order := s.findOrderLocked(orderID)
	if order == nil {
		return InvoiceIssueResult{}, ErrOrderNotFound
	}

	template, ok := s.templateByIDLocked(templateID)
	if !ok {
		return InvoiceIssueResult{}, &InvoiceValidationError{
			Message:     "選択したテンプレートが見つかりません。",
			FieldErrors: map[string]string{"templateID": "選択したテンプレートが見つかりません。"},
		}
	}

	language = s.normaliseLanguageLocked(language)

	now := time.Now()
	counter := len(order.Invoices) + 1
	invoiceID := fmt.Sprintf("inv-%s-%d", orderID, now.UnixNano())
	invoiceNumber := fmt.Sprintf("INV-%d%02d-%03d", now.Year(), int(now.Month()), counter)
	actor := strings.TrimSpace(req.ActorEmail)
	if actor == "" {
		actor = strings.TrimSpace(req.ActorID)
	}
	if actor == "" {
		actor = "オペレーター"
	}

	invoice := InvoiceRecord{
		ID:            invoiceID,
		Number:        invoiceNumber,
		Status:        "発行済み",
		StatusTone:    "success",
		IssuedAt:      now,
		DeliveryEmail: email,
		Note:          note,
		Actor:         actor,
		PDFURL:        buildInvoicePDFURL(invoiceNumber),
		TemplateID:    template.ID,
		Language:      language,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	var job *InvoiceJob
	if strings.EqualFold(template.ID, "invoice-batch") {
		jobID := fmt.Sprintf("job-%s-%d", orderID, now.UnixNano())
		job = &InvoiceJob{
			ID:          jobID,
			Status:      "キュー投入済み",
			StatusTone:  "info",
			SubmittedAt: now,
			Message:     "請求書を生成しています。完了まで少し時間がかかる場合があります。",
		}
		invoice.Status = "発行準備中"
		invoice.StatusTone = "info"
		invoice.IssuedAt = time.Time{}
		invoice.PDFURL = ""
		invoice.JobID = jobID
	}

	order.Invoices = append([]InvoiceRecord{invoice}, order.Invoices...)
	s.orders[idx] = *order

	storedInvoice := order.Invoices[0]

	if job != nil {
		s.invoiceJobs[job.ID] = &invoiceJobState{
			OrderID:   orderID,
			InvoiceID: storedInvoice.ID,
			Job:       *job,
			Completed: false,
			Attempts:  0,
		}
	}

	return InvoiceIssueResult{
		OrderID: orderID,
		Invoice: storedInvoice,
		Job:     job,
	}, nil
}

// InvoiceJobStatus returns the latest status for an asynchronous invoice issuance job.
func (s *StaticService) InvoiceJobStatus(_ context.Context, _ string, jobID string) (InvoiceJobStatus, error) {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return InvoiceJobStatus{}, ErrInvoiceJobNotFound
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.invoiceJobs[jobID]
	if !ok {
		return InvoiceJobStatus{}, ErrInvoiceJobNotFound
	}

	idx, order := s.findOrderLocked(state.OrderID)
	if order == nil {
		delete(s.invoiceJobs, jobID)
		return InvoiceJobStatus{}, ErrOrderNotFound
	}

	var invoice *InvoiceRecord
	for i := range order.Invoices {
		if order.Invoices[i].ID == state.InvoiceID {
			invoice = &order.Invoices[i]
			break
		}
	}
	if invoice == nil {
		delete(s.invoiceJobs, jobID)
		return InvoiceJobStatus{}, ErrInvoiceJobNotFound
	}

	if !state.Completed {
		state.Attempts++
		if state.Attempts >= 2 {
			state.Completed = true
			state.Job.Status = "発行済み"
			state.Job.StatusTone = "success"
			state.Job.Message = "請求書の生成が完了しました。"
			now := time.Now()
			if invoice.IssuedAt.IsZero() {
				invoice.IssuedAt = now
			}
			invoice.Status = "発行済み"
			invoice.StatusTone = "success"
			invoice.PDFURL = buildInvoicePDFURL(invoice.Number)
			invoice.JobID = ""
			invoice.UpdatedAt = now
			s.orders[idx] = *order
			delete(s.invoiceJobs, jobID)
		} else {
			state.Job.Status = "処理中"
			state.Job.StatusTone = "info"
			state.Job.Message = "PDFを生成しています。完了までお待ちください。"
		}
	}

	result := InvoiceJobStatus{
		OrderID: state.OrderID,
		Invoice: *invoice,
		Job:     state.Job,
		Done:    state.Completed,
	}

	if !state.Completed {
		s.invoiceJobs[jobID] = state
	}

	return result, nil
}

// StartBulkExport records a simulated export job for development usage.
func (s *StaticService) StartBulkExport(_ context.Context, _ string, req BulkExportRequest) (ExportJob, error) {
	format := normaliseExportFormat(req.Format)

	fields, ok := exportAllowedFields[format]
	if !ok {
		return ExportJob{}, ErrExportFormatNotAllowed
	}

	targetIDs := make(map[string]struct{})
	for _, raw := range req.OrderIDs {
		if trimmed := strings.TrimSpace(raw); trimmed != "" {
			targetIDs[trimmed] = struct{}{}
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	orders := make([]Order, len(s.orders))
	copy(orders, s.orders)

	filtered := filterOrders(orders, req.Query, true)
	if len(targetIDs) > 0 {
		subset := make([]Order, 0, len(filtered))
		for _, order := range filtered {
			if _, ok := targetIDs[order.ID]; ok {
				subset = append(subset, order)
			}
		}
		filtered = subset
	}

	total := len(filtered)
	if total == 0 {
		return ExportJob{}, ErrExportNoOrders
	}

	now := time.Now()
	jobID := fmt.Sprintf("export-%d", now.UnixNano())

	job := ExportJob{
		ID:              jobID,
		Format:          format,
		Status:          "キュー投入済み",
		StatusTone:      "info",
		Message:         exportQueuedMessage(format),
		SubmittedAt:     now,
		Progress:        0,
		ProcessedOrders: 0,
		TotalOrders:     total,
		Fields:          append([]string(nil), fields...),
	}

	state := &exportJobState{
		Job:       job,
		Completed: false,
		Attempts:  0,
	}

	if s.exportJobs == nil {
		s.exportJobs = make(map[string]*exportJobState)
	}
	s.exportJobs[jobID] = state

	return cloneExportJob(job), nil
}

// ListExportJobs returns the set of known export jobs.
func (s *StaticService) ListExportJobs(_ context.Context, _ string) ([]ExportJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.exportJobs) == 0 {
		return nil, nil
	}

	jobs := make([]ExportJob, 0, len(s.exportJobs))
	for _, state := range s.exportJobs {
		jobs = append(jobs, cloneExportJob(state.Job))
	}

	sort.SliceStable(jobs, func(i, j int) bool {
		return jobs[i].SubmittedAt.After(jobs[j].SubmittedAt)
	})

	return jobs, nil
}

// ExportJobStatus reports the simulated progress of an export job.
func (s *StaticService) ExportJobStatus(_ context.Context, _ string, jobID string) (ExportJobStatus, error) {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return ExportJobStatus{}, ErrExportJobNotFound
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.exportJobs[jobID]
	if !ok {
		return ExportJobStatus{}, ErrExportJobNotFound
	}

	if !state.Completed {
		state.Attempts++

		total := state.Job.TotalOrders
		step := total / 3
		if step < 1 {
			step = 1
		}
		state.Job.ProcessedOrders += step
		if state.Job.ProcessedOrders > total {
			state.Job.ProcessedOrders = total
		}

		if total > 0 {
			state.Job.Progress = (state.Job.ProcessedOrders * 100) / total
		} else {
			state.Job.Progress = 100
		}
		if state.Job.Progress > 100 {
			state.Job.Progress = 100
		}

		if state.Attempts >= 3 || state.Job.ProcessedOrders >= total {
			now := time.Now()
			state.Completed = true
			state.Job.Status = "完了"
			state.Job.StatusTone = "success"
			state.Job.Message = exportCompletedMessage(state.Job.Format)
			state.Job.ProcessedOrders = total
			state.Job.Progress = 100
			state.Job.CompletedAt = &now
			state.Job.DownloadURL = buildExportDownloadURL(state.Job.Format, jobID)
		} else {
			state.Job.Status = "処理中"
			state.Job.StatusTone = "info"
			state.Job.Message = exportProcessingMessage(state.Job.Format)
		}
	}

	result := ExportJobStatus{
		Job:  cloneExportJob(state.Job),
		Done: state.Completed,
	}

	s.exportJobs[jobID] = state

	return result, nil
}

func defaultInvoiceTemplateID(templates []InvoiceTemplate) string {
	for _, tpl := range templates {
		if tpl.Default && strings.TrimSpace(tpl.ID) != "" {
			return strings.TrimSpace(tpl.ID)
		}
	}
	if len(templates) > 0 {
		return strings.TrimSpace(templates[0].ID)
	}
	return ""
}

func defaultInvoiceLanguageCode(languages []InvoiceLanguage) string {
	for _, lang := range languages {
		if lang.Default && strings.TrimSpace(lang.Code) != "" {
			return strings.TrimSpace(lang.Code)
		}
	}
	if len(languages) > 0 {
		return strings.TrimSpace(languages[0].Code)
	}
	return ""
}

func (s *StaticService) templateByIDLocked(id string) (InvoiceTemplate, bool) {
	target := strings.TrimSpace(id)
	for _, tpl := range s.invoiceTemplates {
		if strings.EqualFold(strings.TrimSpace(tpl.ID), target) {
			return tpl, true
		}
	}
	return InvoiceTemplate{}, false
}

func (s *StaticService) normaliseLanguageLocked(code string) string {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return defaultInvoiceLanguageCode(s.invoiceLanguages)
	}
	for _, lang := range s.invoiceLanguages {
		if strings.EqualFold(strings.TrimSpace(lang.Code), trimmed) {
			return strings.TrimSpace(lang.Code)
		}
	}
	return defaultInvoiceLanguageCode(s.invoiceLanguages)
}

func isValidEmail(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	if _, err := mail.ParseAddress(value); err != nil {
		return false
	}
	return true
}

func buildInvoicePDFURL(number string) string {
	safe := strings.TrimSpace(number)
	if safe == "" {
		return ""
	}
	safe = strings.ReplaceAll(safe, " ", "_")
	return fmt.Sprintf("https://storage.example.com/invoices/%s.pdf", safe)
}

// UpdateStatus mutates the order status with optimistic local data for development use.
func (s *StaticService) UpdateStatus(ctx context.Context, _ string, orderID string, req StatusUpdateRequest) (StatusUpdateResult, error) {
	requested := strings.TrimSpace(string(req.Status))
	if requested == "" {
		return StatusUpdateResult{}, &StatusTransitionError{From: Status(""), To: req.Status, Reason: "ステータスを選択してください"}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx, order := s.findOrderLocked(orderID)
	if order == nil {
		return StatusUpdateResult{}, ErrOrderNotFound
	}

	current := order.Status
	if current == req.Status {
		return StatusUpdateResult{}, &StatusTransitionError{From: current, To: req.Status, Reason: "すでにこのステータスです"}
	}
	if !isTransitionAllowed(current, req.Status) {
		reason := fmt.Sprintf("「%s」から「%s」への変更は許可されていません", defaultStatusLabel(current), defaultStatusLabel(req.Status))
		return StatusUpdateResult{}, &StatusTransitionError{From: current, To: req.Status, Reason: reason}
	}

	note := strings.TrimSpace(req.Note)
	now := time.Now()

	order.Status = req.Status
	order.StatusLabel = defaultStatusLabel(req.Status)
	order.StatusTone = defaultStatusTone(req.Status)
	order.UpdatedAt = now

	if note != "" {
		formatted := note
		actor := strings.TrimSpace(req.ActorEmail)
		if actor != "" {
			formatted = actor + ": " + note
		}
		order.Notes = append([]string{formatted}, order.Notes...)
	}

	if req.Status == StatusRefunded {
		order.HasRefundRequest = true
		order.Payment.Status = "返金済み"
		order.Payment.StatusTone = "info"
	}

	switch req.Status {
	case StatusInProduction:
		order.Fulfillment.SLAStatus = "制作進行中"
		order.Fulfillment.SLAStatusTone = "info"
	case StatusReadyToShip:
		order.Fulfillment.SLAStatus = "集荷待ち"
		order.Fulfillment.SLAStatusTone = "info"
	case StatusShipped:
		order.Fulfillment.DispatchedAt = timePtr(now)
		order.Fulfillment.SLAStatus = "配送中"
		order.Fulfillment.SLAStatusTone = "info"
	case StatusDelivered:
		order.Fulfillment.DeliveredAt = timePtr(now)
		order.Fulfillment.SLAStatus = "納品済み"
		order.Fulfillment.SLAStatusTone = "success"
	case StatusCancelled:
		order.Fulfillment.SLAStatus = "キャンセル済み"
		order.Fulfillment.SLAStatusTone = "muted"
	}

	s.orders[idx] = *order

	actor := strings.TrimSpace(req.ActorEmail)
	if actor == "" {
		actor = "オペレーター"
	}

	description := buildTimelineDescription(note, req.NotifyCustomer)
	event := TimelineEvent{
		ID:          fmt.Sprintf("%s-%d", orderID, now.UnixNano()),
		Status:      req.Status,
		Title:       fmt.Sprintf("ステータスを「%s」に更新", defaultStatusLabel(req.Status)),
		Description: description,
		Actor:       actor,
		OccurredAt:  now,
	}
	s.timelines[orderID] = append(s.timelines[orderID], event)

	if s.audit != nil {
		_ = s.audit.Record(ctx, AuditLogEntry{
			OrderID:     order.ID,
			OrderNumber: order.Number,
			Action:      "orders.status.transition",
			ActorID:     strings.TrimSpace(req.ActorID),
			ActorEmail:  strings.TrimSpace(req.ActorEmail),
			FromStatus:  current,
			ToStatus:    req.Status,
			Note:        note,
			OccurredAt:  now,
		})
	}

	updated := cloneOrder(*order)
	timeline := cloneTimeline(s.timelines[orderID])

	return StatusUpdateResult{Order: updated, Timeline: timeline}, nil
}

// SubmitRefund mutates the payment state with a simulated refund.
func (s *StaticService) SubmitRefund(ctx context.Context, _ string, orderID string, req RefundRequest) (RefundResult, error) {
	paymentID := strings.TrimSpace(req.PaymentID)
	if paymentID == "" {
		return RefundResult{}, &RefundValidationError{
			Message:     "返金対象の支払いを選択してください。",
			FieldErrors: map[string]string{"paymentID": "返金対象の支払いを選択してください。"},
		}
	}
	if req.AmountMinor <= 0 {
		return RefundResult{}, &RefundValidationError{
			Message:     "返金金額を正しく入力してください。",
			FieldErrors: map[string]string{"amount": "1円以上の金額を入力してください。"},
		}
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return RefundResult{}, &RefundValidationError{
			Message:     "返金理由を入力してください。",
			FieldErrors: map[string]string{"reason": "返金理由を入力してください。"},
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx, order := s.findOrderLocked(orderID)
	if order == nil {
		return RefundResult{}, ErrOrderNotFound
	}

	payment := findPayment(order, paymentID)
	if payment == nil {
		return RefundResult{}, ErrPaymentNotFound
	}

	if payment.AmountCaptured <= 0 {
		return RefundResult{}, &RefundValidationError{
			Message:     "この支払いはまだ確定していないため返金できません。",
			FieldErrors: map[string]string{"paymentID": "この支払いは返金できません。"},
		}
	}
	if payment.AmountAvailable <= 0 {
		return RefundResult{}, &RefundValidationError{
			Message:     "返金可能な金額がありません。",
			FieldErrors: map[string]string{"amount": "返金可能な金額がありません。"},
		}
	}
	if req.AmountMinor > payment.AmountAvailable {
		return RefundResult{}, &RefundValidationError{
			Message:     "返金可能額を超えています。",
			FieldErrors: map[string]string{"amount": "返金可能額を超えています。"},
		}
	}

	now := time.Now()
	payment.AmountRefunded += req.AmountMinor
	if payment.AmountRefunded > payment.AmountCaptured {
		payment.AmountRefunded = payment.AmountCaptured
	}
	payment.AmountAvailable = payment.AmountCaptured - payment.AmountRefunded
	if payment.AmountAvailable < 0 {
		payment.AmountAvailable = 0
	}

	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		if payment.Currency != "" {
			currency = payment.Currency
		} else {
			currency = order.Currency
		}
	}

	actor := strings.TrimSpace(req.ActorEmail)
	if actor == "" {
		actor = "オペレーター"
	}

	refID := fmt.Sprintf("refund_%s_%d", payment.ID, now.UnixNano())
	refund := RefundRecord{
		ID:          refID,
		PaymentID:   payment.ID,
		AmountMinor: req.AmountMinor,
		Currency:    currency,
		Reason:      reason,
		Status:      "succeeded",
		ProcessedAt: now,
		Actor:       actor,
		Reference:   refID,
	}
	order.Refunds = append([]RefundRecord{refund}, order.Refunds...)

	order.HasRefundRequest = true
	order.Payment.StatusTone = "info"
	if payment.AmountAvailable == 0 {
		order.Payment.Status = "返金済み"
	} else {
		order.Payment.Status = "一部返金"
	}
	order.UpdatedAt = now

	if reason != "" {
		formatted := reason
		if actor != "" {
			formatted = actor + ": " + reason
		}
		order.Notes = append([]string{formatted}, order.Notes...)
	}

	if s.timelines != nil {
		description := fmt.Sprintf("%s を返金 (%s)", formatMinorAmount(req.AmountMinor, currency), reason)
		event := TimelineEvent{
			ID:          fmt.Sprintf("%s-refund-%d", orderID, now.UnixNano()),
			Status:      order.Status,
			Title:       "返金を登録",
			Description: strings.TrimSpace(description),
			Actor:       actor,
			OccurredAt:  now,
		}
		s.timelines[orderID] = append(s.timelines[orderID], event)
	}

	paymentOption := toRefundPaymentOption(*payment)
	paymentOptions := make([]RefundPaymentOption, 0, len(order.Payments))
	for _, p := range order.Payments {
		paymentOptions = append(paymentOptions, toRefundPaymentOption(p))
	}

	s.orders[idx] = *order

	return RefundResult{
		Refund:   refund,
		Payment:  paymentOption,
		Payments: paymentOptions,
	}, nil
}

type invoiceJobState struct {
	OrderID   string
	InvoiceID string
	Job       InvoiceJob
	Completed bool
	Attempts  int
}

type exportJobState struct {
	Job       ExportJob
	Completed bool
	Attempts  int
}

func (s *StaticService) findOrderLocked(orderID string) (int, *Order) {
	for i := range s.orders {
		if s.orders[i].ID == orderID {
			return i, &s.orders[i]
		}
	}
	return -1, nil
}

func findPayment(order *Order, paymentID string) *PaymentDetail {
	if order == nil {
		return nil
	}
	for i := range order.Payments {
		if order.Payments[i].ID == paymentID {
			return &order.Payments[i]
		}
	}
	return nil
}

func toRefundPaymentOption(payment PaymentDetail) RefundPaymentOption {
	return RefundPaymentOption{
		ID:              payment.ID,
		Label:           buildPaymentLabel(payment),
		Method:          payment.Method,
		Reference:       payment.Reference,
		Status:          payment.Status,
		StatusTone:      payment.StatusTone,
		Currency:        payment.Currency,
		CapturedMinor:   payment.AmountCaptured,
		RefundedMinor:   payment.AmountRefunded,
		AvailableMinor:  payment.AmountAvailable,
		CapturedAt:      payment.CapturedAt,
		SupportsRefunds: payment.AmountAvailable > 0,
	}
}

func buildPaymentLabel(payment PaymentDetail) string {
	parts := []string{}
	if trimmed := strings.TrimSpace(payment.Provider); trimmed != "" {
		parts = append(parts, trimmed)
	}
	if method := strings.TrimSpace(payment.Method); method != "" {
		parts = append(parts, method)
	}
	if last4 := strings.TrimSpace(payment.Last4); last4 != "" {
		if !strings.HasPrefix(last4, "****") && len(last4) <= 4 {
			parts = append(parts, "****"+last4)
		} else {
			parts = append(parts, last4)
		}
	}
	if len(parts) == 0 {
		return "支払い"
	}
	return strings.Join(parts, " ")
}

func formatMinorAmount(amount int64, currency string) string {
	sign := ""
	if amount < 0 {
		sign = "-"
		amount = -amount
	}
	major := amount / 100
	minor := amount % 100
	code := strings.ToUpper(strings.TrimSpace(currency))
	if code == "" {
		return fmt.Sprintf("%s%d.%02d", sign, major, minor)
	}
	return fmt.Sprintf("%s%d.%02d %s", sign, major, minor, code)
}

func buildStatusChoices(current Status) []StatusTransitionOption {
	allowed := map[Status]bool{}
	for _, next := range statusTransitionGraph[current] {
		allowed[next] = true
	}

	choices := make([]StatusTransitionOption, 0, len(orderedStatuses()))
	for _, candidate := range orderedStatuses() {
		choice := StatusTransitionOption{
			Value:       candidate,
			Label:       defaultStatusLabel(candidate),
			Description: StatusDescription(candidate),
			Selected:    candidate == current,
		}
		if candidate == current {
			choice.Disabled = true
			choice.DisabledReason = "現在のステータスです"
		} else if !allowed[candidate] {
			choice.Disabled = true
			choice.DisabledReason = fmt.Sprintf("「%s」から「%s」へは遷移できません", defaultStatusLabel(current), defaultStatusLabel(candidate))
		}
		choices = append(choices, choice)
	}
	return choices
}

func isTransitionAllowed(from, to Status) bool {
	if from == to {
		return false
	}
	for _, candidate := range statusTransitionGraph[from] {
		if candidate == to {
			return true
		}
	}
	return false
}

func buildTimelineDescription(note string, notify bool) string {
	parts := []string{}
	trimmed := strings.TrimSpace(note)
	if trimmed != "" {
		parts = append(parts, trimmed)
	}
	if notify {
		parts = append(parts, "顧客に通知を送信しました")
	}
	return strings.Join(parts, " / ")
}

var statusTransitionGraph = map[Status][]Status{
	StatusPendingPayment: {StatusPaymentReview, StatusInProduction, StatusCancelled},
	StatusPaymentReview:  {StatusInProduction, StatusCancelled},
	StatusInProduction:   {StatusReadyToShip, StatusCancelled},
	StatusReadyToShip:    {StatusShipped, StatusCancelled},
	StatusShipped:        {StatusDelivered, StatusRefunded},
	StatusDelivered:      {StatusRefunded},
	StatusCancelled:      {StatusRefunded},
	StatusRefunded:       {},
}

func filterOrders(orders []Order, query Query, includeStatus bool) []Order {
	results := make([]Order, 0, len(orders))

	statusSet := map[Status]bool{}
	if includeStatus && len(query.Statuses) > 0 {
		for _, status := range query.Statuses {
			statusSet[status] = true
		}
	}

	trimmedCurrency := strings.TrimSpace(query.Currency)

	for _, order := range orders {
		if includeStatus && len(statusSet) > 0 && !statusSet[order.Status] {
			continue
		}
		if query.Since != nil && order.UpdatedAt.Before(*query.Since) {
			continue
		}
		if trimmedCurrency != "" && !strings.EqualFold(order.Currency, trimmedCurrency) {
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
		case "status":
			ra := statusSortRank(a.Status)
			rb := statusSortRank(b.Status)
			if ra == rb {
				less = strings.Compare(strings.ToLower(strings.TrimSpace(a.StatusLabel)), strings.ToLower(strings.TrimSpace(b.StatusLabel))) < 0
			} else {
				less = ra < rb
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
	allStatuses := orderedStatuses()

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

func buildFilterSummary(orders []Order, query Query) FilterSummary {
	withoutStatus := filterOrders(orders, query, false)

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

func seedTimeline(order Order) []TimelineEvent {
	statuses := orderedStatuses()
	index := len(statuses) - 1
	for i, st := range statuses {
		if st == order.Status {
			index = i
			break
		}
	}

	base := order.CreatedAt
	if base.IsZero() {
		base = time.Now().Add(-48 * time.Hour)
	}
	step := 3 * time.Hour
	current := base

	events := make([]TimelineEvent, 0, index+2)
	events = append(events, TimelineEvent{
		ID:          fmt.Sprintf("%s-created", order.ID),
		Status:      StatusPendingPayment,
		Title:       "注文を作成",
		Description: fmt.Sprintf("注文 #%s を受け付けました", strings.TrimSpace(order.Number)),
		Actor:       "システム",
		OccurredAt:  base,
	})

	for i := 0; i <= index && i < len(statuses); i++ {
		current = current.Add(step)
		status := statuses[i]
		events = append(events, TimelineEvent{
			ID:          fmt.Sprintf("%s-%s-%d", order.ID, status, i),
			Status:      status,
			Title:       fmt.Sprintf("ステータスを「%s」に更新", defaultStatusLabel(status)),
			Description: StatusDescription(status),
			Actor:       "システム",
			OccurredAt:  current,
		})
	}

	if len(events) > 0 {
		final := &events[len(events)-1]
		if !order.UpdatedAt.IsZero() {
			final.OccurredAt = order.UpdatedAt
		}
	}

	return events
}

type noopAuditLogger struct{}

func (noopAuditLogger) Record(_ context.Context, _ AuditLogEntry) error { return nil }

func int64Ptr(value int64) *int64 {
	return &value
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func orderedStatuses() []Status {
	return []Status{
		StatusPendingPayment,
		StatusPaymentReview,
		StatusInProduction,
		StatusReadyToShip,
		StatusShipped,
		StatusDelivered,
		StatusRefunded,
		StatusCancelled,
	}
}

func statusSortRank(status Status) int {
	for idx, candidate := range orderedStatuses() {
		if candidate == status {
			return idx
		}
	}
	return len(orderedStatuses())
}
