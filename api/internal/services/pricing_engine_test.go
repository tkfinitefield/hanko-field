package services

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
)

func TestCartPricingEngine_FullPipeline(t *testing.T) {
	ctx := context.Background()

	promo := &fakePromotionService{
		results: map[string]PromotionValidationResult{
			"PROMO10": {Code: "PROMO10", Eligible: true, DiscountAmount: 700, Reason: "seasonal"},
		},
	}
	taxCalc := &fakeTaxCalculator{quote: TaxQuote{Amount: 800, Breakdown: []TaxBreakdown{{Name: "VAT", Amount: 800}}}}
	ship := &fakeShippingEstimator{quote: ShippingQuote{Amount: 950, Breakdown: []ShippingBreakdown{{ServiceLevel: "standard", Amount: 950}}}}
	inventory := &fakeInventoryAvailability{}
	rule := &fakeItemDiscountRule{
		name: "loyalty",
		fn: func(item CartItem, subtotal int64) int64 {
			if item.SKU == "SKU-001" {
				return 200 * int64(item.Quantity)
			}
			return 0
		},
	}

	engine, err := NewCartPricingEngine(CartPricingEngineDeps{
		Promotion: promo,
		Tax:       taxCalc,
		Shipping:  ship,
		Inventory: inventory,
		ItemRules: []ItemDiscountRule{rule},
		CacheTTL:  time.Hour,
		Now:       func() time.Time { return time.Date(2024, 10, 10, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("NewCartPricingEngine error: %v", err)
	}

	promoCode := "promo10"
	cart := Cart{
		ID:       "cart_123",
		UserID:   "user_1",
		Currency: "JPY",
		Items: []CartItem{
			{
				ID:               "item_1",
				ProductID:        "prod_a",
				SKU:              "SKU-001",
				Quantity:         2,
				UnitPrice:        4000,
				Currency:         "JPY",
				WeightGrams:      200,
				RequiresShipping: true,
			},
			{
				ID:               "item_2",
				ProductID:        "prod_b",
				SKU:              "SKU-002",
				Quantity:         1,
				UnitPrice:        2000,
				Currency:         "JPY",
				WeightGrams:      100,
				RequiresShipping: true,
			},
		},
		ShippingAddress: &Address{Country: "JP", PostalCode: "100-0001"},
	}

	result, err := engine.Calculate(ctx, PriceCartCommand{Cart: cart, PromotionCode: &promoCode})
	if err != nil {
		t.Fatalf("Calculate error: %v", err)
	}

	expectEstimate := CartEstimate{Subtotal: 10000, Discount: 1100, Tax: 800, Shipping: 950, Total: 10650}
	if result.Estimate != expectEstimate {
		t.Fatalf("Estimate mismatch: want %+v, got %+v", expectEstimate, result.Estimate)
	}

	if len(result.Breakdown.Discounts) != 2 {
		t.Fatalf("expected 2 discount breakdowns, got %d", len(result.Breakdown.Discounts))
	}

	itemTotals := result.Breakdown.Items
	if len(itemTotals) != 2 {
		t.Fatalf("expected 2 item breakdowns, got %d", len(itemTotals))
	}
	for _, item := range itemTotals {
		if item.Total <= 0 {
			t.Fatalf("expected positive item total, got %+v", item)
		}
	}
	if itemTotals[0].Total+itemTotals[1].Total != result.Breakdown.Total {
		t.Fatalf("item totals should match cart total, got %d vs %d", itemTotals[0].Total+itemTotals[1].Total, result.Breakdown.Total)
	}
	if taxCalc.lastRequest.ShippingAmount != result.Estimate.Shipping {
		t.Fatalf("expected shipping amount %d in tax request, got %d", result.Estimate.Shipping, taxCalc.lastRequest.ShippingAmount)
	}
	if taxCalc.lastRequest.PromotionCode == nil || ship.lastRequest.PromotionCode == nil {
		t.Fatalf("expected promotion code to be forwarded on valid promo")
	}

	if ship.calls != 1 {
		t.Fatalf("expected shipping estimator to be called once, got %d", ship.calls)
	}
	if promo.calls != 1 {
		t.Fatalf("expected promotion service to be called once, got %d", promo.calls)
	}
	if !inventory.validated {
		t.Fatalf("expected inventory validation to run")
	}

	// Second call should use cache for shipping.
	_, err = engine.Calculate(ctx, PriceCartCommand{Cart: cart, PromotionCode: &promoCode})
	if err != nil {
		t.Fatalf("second Calculate error: %v", err)
	}
	if ship.calls != 1 {
		t.Fatalf("expected shipping estimator cache to prevent second call, got %d", ship.calls)
	}

	// Changing the promotion code should invalidate the cache because tote pricing changes.
	promo.results["PROMO20"] = PromotionValidationResult{Code: "PROMO20", Eligible: true, DiscountAmount: 1500, Reason: "vip"}
	newPromo := "promo20"
	_, err = engine.Calculate(ctx, PriceCartCommand{Cart: cart, PromotionCode: &newPromo})
	if err != nil {
		t.Fatalf("third Calculate error (promo change): %v", err)
	}
	if ship.calls != 2 {
		t.Fatalf("expected shipping estimator to be called on promo change, got %d", ship.calls)
	}

	// Repeating with the same code should reuse the cached quote.
	_, err = engine.Calculate(ctx, PriceCartCommand{Cart: cart, PromotionCode: &newPromo})
	if err != nil {
		t.Fatalf("fourth Calculate error (cached promo): %v", err)
	}
	if ship.calls != 2 {
		t.Fatalf("expected cached shipping quote for repeated promo, got %d", ship.calls)
	}

	// Bypass cache to force another call.
	_, err = engine.Calculate(ctx, PriceCartCommand{Cart: cart, PromotionCode: &newPromo, BypassShippingCache: true})
	if err != nil {
		t.Fatalf("fifth Calculate error (bypass): %v", err)
	}
	if ship.calls != 3 {
		t.Fatalf("expected shipping estimator to be called three times after bypass, got %d", ship.calls)
	}

	// Promotion larger than subtotal should clamp and still align totals.
	bigPromo := "promo_big"
	promo.results["PROMO_BIG"] = PromotionValidationResult{Code: "PROMO_BIG", Eligible: true, DiscountAmount: 50_000, Reason: "full"}
	resBig, err := engine.Calculate(ctx, PriceCartCommand{Cart: cart, PromotionCode: &bigPromo})
	if err != nil {
		t.Fatalf("big promo Calculate error: %v", err)
	}
	if resBig.Breakdown.Discount != 10000 {
		t.Fatalf("expected clamp to subtotal (10000), got %d", resBig.Breakdown.Discount)
	}
	itemDiscountAccum := int64(0)
	for _, d := range resBig.Breakdown.Discounts {
		if d.Type == "item" {
			itemDiscountAccum += d.Amount
		}
	}
	expectedPromoShare := resBig.Breakdown.Discount - itemDiscountAccum
	if expectedPromoShare < 0 {
		expectedPromoShare = 0
	}
	if got := resBig.Breakdown.Discounts[len(resBig.Breakdown.Discounts)-1].Amount; got != expectedPromoShare {
		t.Fatalf("promotion breakdown amount mismatch: got %d want %d", got, expectedPromoShare)
	}

	// Promotion with zero discount should still be forwarded to shipping/tax.
	promo.results["SHIPFREE"] = PromotionValidationResult{Code: "SHIPFREE", Eligible: true, DiscountAmount: 0, Reason: "free-shipping"}
	shipFree := "shipfree"
	_, err = engine.Calculate(ctx, PriceCartCommand{Cart: cart, PromotionCode: &shipFree, BypassShippingCache: true})
	if err != nil {
		t.Fatalf("shipfree Calculate error: %v", err)
	}
	if ship.lastRequest.PromotionCode == nil || taxCalc.lastRequest.PromotionCode == nil {
		t.Fatalf("expected zero-discount promo to be forwarded")
	}

	invalidPromo := "invalid"
	_, err = engine.Calculate(ctx, PriceCartCommand{Cart: cart, PromotionCode: &invalidPromo, BypassShippingCache: true})
	if err != nil {
		t.Fatalf("invalid promo Calculate error: %v", err)
	}
	if ship.lastRequest.PromotionCode != nil {
		t.Fatalf("expected shipping estimator to receive nil promotion for rejected code")
	}
	if taxCalc.lastRequest.PromotionCode != nil {
		t.Fatalf("expected tax calculator to receive nil promotion for rejected code")
	}
}

func TestCartPricingEngine_CurrencyMismatch(t *testing.T) {
	promo := &fakePromotionService{}
	engine, err := NewCartPricingEngine(CartPricingEngineDeps{Promotion: promo})
	if err != nil {
		t.Fatalf("NewCartPricingEngine error: %v", err)
	}

	cart := Cart{
		Currency: "JPY",
		Items: []CartItem{
			{ID: "item1", SKU: "A", Quantity: 1, UnitPrice: 100, Currency: "JPY"},
			{ID: "item2", SKU: "B", Quantity: 1, UnitPrice: 100, Currency: "USD"},
		},
	}

	_, err = engine.Calculate(context.Background(), PriceCartCommand{Cart: cart})
	if !errors.Is(err, ErrCartPricingCurrencyMismatch) {
		t.Fatalf("expected ErrCartPricingCurrencyMismatch, got %v", err)
	}
}

func TestCartPricingEngine_EmptyCart(t *testing.T) {
	promo := &fakePromotionService{}
	engine, err := NewCartPricingEngine(CartPricingEngineDeps{Promotion: promo})
	if err != nil {
		t.Fatalf("NewCartPricingEngine error: %v", err)
	}

	cart := Cart{Currency: "JPY"}
	result, err := engine.Calculate(context.Background(), PriceCartCommand{Cart: cart})
	if err != nil {
		t.Fatalf("Calculate error: %v", err)
	}

	empty := CartEstimate{}
	if result.Estimate != empty {
		t.Fatalf("expected zero estimate, got %+v", result.Estimate)
	}
}

func TestCartPricingEngine_ItemRuleClampDistribution(t *testing.T) {
	ctx := context.Background()
	promo := &fakePromotionService{}
	ruleA := &fakeItemDiscountRule{name: "ruleA", fn: func(item CartItem, subtotal int64) int64 {
		return 80
	}}
	ruleB := &fakeItemDiscountRule{name: "ruleB", fn: func(item CartItem, subtotal int64) int64 {
		return 80
	}}
	engine, err := NewCartPricingEngine(CartPricingEngineDeps{
		Promotion: promo,
		ItemRules: []ItemDiscountRule{ruleA, ruleB},
	})
	if err != nil {
		t.Fatalf("NewCartPricingEngine error: %v", err)
	}

	cart := Cart{
		Currency: "JPY",
		Items: []CartItem{{
			ID:        "i1",
			SKU:       "sku",
			Quantity:  1,
			UnitPrice: 100,
			Currency:  "JPY",
		}},
	}

	result, err := engine.Calculate(ctx, PriceCartCommand{Cart: cart})
	if err != nil {
		t.Fatalf("Calculate error: %v", err)
	}

	if got := result.Breakdown.Discount; got != 100 {
		t.Fatalf("expected total discount 100, got %d", got)
	}
	if len(result.Breakdown.Items) != 1 {
		t.Fatalf("expected one item breakdown, got %d", len(result.Breakdown.Items))
	}
	if itemDiscount := result.Breakdown.Items[0].Discount; itemDiscount != 100 {
		t.Fatalf("expected item discount 100, got %d", itemDiscount)
	}

	var itemDiscountTotal int64
	for _, d := range result.Breakdown.Discounts {
		if d.Type == "item" {
			itemDiscountTotal += d.Amount
		}
	}
	if itemDiscountTotal != result.Breakdown.Discount {
		t.Fatalf("item discount breakdown (%d) should equal total discount (%d)", itemDiscountTotal, result.Breakdown.Discount)
	}
	for _, d := range result.Breakdown.Discounts {
		if d.Type == "item" && d.Amount <= 0 {
			t.Fatalf("expected positive item rule allocation, got %+v", d)
		}
	}

	promo.results = map[string]PromotionValidationResult{"PROMO_ZERO": {Code: "PROMO_ZERO", Eligible: true, DiscountAmount: 200}}
	promoCode := "promo_zero"
	resPromo, err := engine.Calculate(ctx, PriceCartCommand{Cart: cart, PromotionCode: &promoCode})
	if err != nil {
		t.Fatalf("Calculate with promo error: %v", err)
	}
	if resPromo.Breakdown.Discount != 100 {
		t.Fatalf("expected total discount to remain 100, got %d", resPromo.Breakdown.Discount)
	}
	foundPromo := false
	for _, d := range resPromo.Breakdown.Discounts {
		if d.Type == "promotion" {
			foundPromo = true
			if d.Amount != 0 {
				t.Fatalf("expected promo breakdown amount 0 after clamp, got %d", d.Amount)
			}
		}
	}
	if !foundPromo {
		t.Fatalf("expected promo breakdown entry")
	}
}

type fakePromotionService struct {
	results map[string]PromotionValidationResult
	calls   int
}

func (f *fakePromotionService) GetPublicPromotion(context.Context, string) (PromotionPublic, error) {
	panic("unexpected call")
}

func (f *fakePromotionService) ValidatePromotion(ctx context.Context, cmd ValidatePromotionCommand) (PromotionValidationResult, error) {
	f.calls++
	if f.results == nil {
		return PromotionValidationResult{Code: cmd.Code, Eligible: false}, nil
	}
	if res, ok := f.results[strings.ToUpper(cmd.Code)]; ok {
		return res, nil
	}
	return PromotionValidationResult{Code: cmd.Code, Eligible: false}, nil
}

func (f *fakePromotionService) ListPromotions(context.Context, PromotionListFilter) (domain.CursorPage[Promotion], error) {
	panic("unexpected call")
}

func (f *fakePromotionService) CreatePromotion(context.Context, UpsertPromotionCommand) (Promotion, error) {
	panic("unexpected call")
}

func (f *fakePromotionService) UpdatePromotion(context.Context, UpsertPromotionCommand) (Promotion, error) {
	panic("unexpected call")
}

func (f *fakePromotionService) DeletePromotion(context.Context, string) error {
	panic("unexpected call")
}

func (f *fakePromotionService) ListPromotionUsage(context.Context, PromotionUsageFilter) (domain.CursorPage[PromotionUsage], error) {
	panic("unexpected call")
}

type fakeTaxCalculator struct {
	quote       TaxQuote
	lastRequest TaxCalculationRequest
}

func (f *fakeTaxCalculator) CalculateTax(ctx context.Context, req TaxCalculationRequest) (TaxQuote, error) {
	f.lastRequest = req
	return f.quote, nil
}

type fakeShippingEstimator struct {
	quote       ShippingQuote
	calls       int
	lastRequest ShippingEstimateRequest
}

func (f *fakeShippingEstimator) EstimateShipping(ctx context.Context, req ShippingEstimateRequest) (ShippingQuote, error) {
	f.calls++
	f.lastRequest = req
	return f.quote, nil
}

type fakeInventoryAvailability struct {
	validated bool
}

func (f *fakeInventoryAvailability) ValidateAvailability(ctx context.Context, lines []InventoryLine) error {
	f.validated = true
	if len(lines) == 0 {
		return errors.New("expected lines")
	}
	return nil
}

type fakeItemDiscountRule struct {
	name string
	fn   func(item CartItem, subtotal int64) int64
}

func (f *fakeItemDiscountRule) Name() string { return f.name }

func (f *fakeItemDiscountRule) Apply(ctx context.Context, item CartItem, subtotal int64) (ItemDiscountResult, error) {
	amount := f.fn(item, subtotal)
	if amount < 0 {
		return ItemDiscountResult{}, errors.New("negative discount")
	}
	return ItemDiscountResult{Amount: amount}, nil
}
func TestCartPricingEngine_PromoOverflowClamp(t *testing.T) {
	ctx := context.Background()
	promo := &fakePromotionService{
		results: map[string]PromotionValidationResult{
			"MEGA": {Code: "MEGA", Eligible: true, DiscountAmount: math.MaxInt64, Reason: "cap"},
		},
	}
	engine, err := NewCartPricingEngine(CartPricingEngineDeps{Promotion: promo})
	if err != nil {
		t.Fatalf("NewCartPricingEngine error: %v", err)
	}

	cart := Cart{
		Currency: "JPY",
		Items: []CartItem{{
			ID:        "item",
			SKU:       "SKU",
			Quantity:  1,
			UnitPrice: 1000,
			Currency:  "JPY",
		}},
	}

	code := "mega"
	result, err := engine.Calculate(ctx, PriceCartCommand{Cart: cart, PromotionCode: &code})
	if err != nil {
		t.Fatalf("Calculate error: %v", err)
	}
	if result.Estimate.Total < 0 {
		t.Fatalf("expected non-negative total, got %d", result.Estimate.Total)
	}
	if result.Breakdown.Discount != result.Breakdown.Subtotal {
		t.Fatalf("expected discount to equal subtotal after clamp, got %d vs %d", result.Breakdown.Discount, result.Breakdown.Subtotal)
	}
}
func TestCartPricingEngine_TaxWeightAfterPromo(t *testing.T) {
	ctx := context.Background()
	promo := &fakePromotionService{
		results: map[string]PromotionValidationResult{"HALF": {Code: "HALF", Eligible: true, DiscountAmount: 1, Reason: "round"}},
	}
	tax := &fakeTaxCalculator{quote: TaxQuote{Amount: 2}}
	svc, err := NewCartPricingEngine(CartPricingEngineDeps{
		Promotion: promo,
		Tax:       tax,
	})
	if err != nil {
		t.Fatalf("NewCartPricingEngine error: %v", err)
	}

	cart := Cart{
		Currency: "JPY",
		Items: []CartItem{
			{ID: "a", SKU: "a", Quantity: 1, UnitPrice: 1, Currency: "JPY", RequiresShipping: false},
			{ID: "b", SKU: "b", Quantity: 1, UnitPrice: 1, Currency: "JPY", RequiresShipping: false},
		},
	}

	code := "half"
	res, err := svc.Calculate(ctx, PriceCartCommand{Cart: cart, PromotionCode: &code})
	if err != nil {
		t.Fatalf("Calculate error: %v", err)
	}
	if len(res.Breakdown.Items) != 2 {
		t.Fatalf("expected two items, got %d", len(res.Breakdown.Items))
	}
	if res.Breakdown.Items[0].Tax != 0 {
		t.Fatalf("expected first item tax to be zero, got %d", res.Breakdown.Items[0].Tax)
	}
	if res.Breakdown.Items[1].Tax != 2 {
		t.Fatalf("expected second item tax to absorb total tax, got %d", res.Breakdown.Items[1].Tax)
	}
}
