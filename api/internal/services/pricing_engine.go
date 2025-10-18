package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	// ErrCartPricingInvalidInput signals bad request data such as missing cart items or negative prices.
	ErrCartPricingInvalidInput = errors.New("cart pricing: invalid input")
	// ErrCartPricingCurrencyMismatch is returned when items use multiple currencies.
	ErrCartPricingCurrencyMismatch = errors.New("cart pricing: currency mismatch")
)

type CartPricingEngine struct {
	promotion PromotionService
	tax       TaxCalculator
	shipping  ShippingEstimator
	inventory InventoryAvailabilityService
	itemRules []ItemDiscountRule
	now       func() time.Time
	logger    func(context.Context, string, map[string]any)
	cache     *shippingQuoteCache
}

type CartPricingEngineDeps struct {
	Promotion PromotionService
	Tax       TaxCalculator
	Shipping  ShippingEstimator
	Inventory InventoryAvailabilityService
	ItemRules []ItemDiscountRule
	CacheTTL  time.Duration
	Now       func() time.Time
	Logger    func(context.Context, string, map[string]any)
}

func NewCartPricingEngine(deps CartPricingEngineDeps) (*CartPricingEngine, error) {
	if deps.Promotion == nil {
		return nil, errors.New("cart pricing engine: promotion service is required")
	}
	if len(deps.ItemRules) == 0 {
		deps.ItemRules = nil
	}
	ttl := deps.CacheTTL
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	logger := deps.Logger
	if logger == nil {
		logger = func(context.Context, string, map[string]any) {}
	}

	engine := &CartPricingEngine{
		promotion: deps.Promotion,
		tax:       deps.Tax,
		shipping:  deps.Shipping,
		inventory: deps.Inventory,
		itemRules: deps.ItemRules,
		now: func() time.Time {
			return now().UTC()
		},
		logger: logger,
		cache:  newShippingQuoteCache(ttl, func() time.Time { return now().UTC() }),
	}

	return engine, nil
}

type PriceCartCommand struct {
	Cart                Cart
	PromotionCode       *string
	ShippingAddress     *Address
	BillingAddress      *Address
	BypassShippingCache bool
}

type PriceCartResult struct {
	Breakdown PricingBreakdown
	Estimate  CartEstimate
}

type ItemDiscountRule interface {
	Name() string
	Apply(ctx context.Context, item CartItem, subtotal int64) (ItemDiscountResult, error)
}

type ItemDiscountResult struct {
	Amount      int64
	Description string
	Metadata    map[string]any
}

type TaxCalculator interface {
	CalculateTax(ctx context.Context, req TaxCalculationRequest) (TaxQuote, error)
}

type TaxCalculationRequest struct {
	Currency        string
	Items           []TaxableItem
	CartSubtotal    int64
	DiscountTotal   int64
	ShippingAmount  int64
	BillingAddress  *Address
	ShippingAddress *Address
	PromotionCode   *string
}

type TaxableItem struct {
	ItemID   string
	SKU      string
	Quantity int
	Subtotal int64
	Discount int64
	TaxCode  string
}

type TaxQuote struct {
	Amount    int64
	Breakdown []TaxBreakdown
}

type ShippingEstimator interface {
	EstimateShipping(ctx context.Context, req ShippingEstimateRequest) (ShippingQuote, error)
}

type ShippingEstimateRequest struct {
	Currency        string
	Items           []ShippableItem
	ShippingAddress *Address
	CartSubtotal    int64
	DiscountTotal   int64
	PromotionCode   *string
}

type ShippableItem struct {
	ItemID           string
	SKU              string
	Quantity         int
	WeightGrams      int
	RequiresShipping bool
}

type ShippingQuote struct {
	Amount    int64
	Breakdown []ShippingBreakdown
}

type InventoryAvailabilityService interface {
	ValidateAvailability(ctx context.Context, lines []InventoryLine) error
}

func (e *CartPricingEngine) Calculate(ctx context.Context, cmd PriceCartCommand) (PriceCartResult, error) {
	if err := e.validateCartInput(cmd); err != nil {
		return PriceCartResult{}, err
	}

	cart := cmd.Cart
	if cmd.ShippingAddress != nil {
		cart.ShippingAddress = cmd.ShippingAddress
	}
	if cmd.BillingAddress != nil {
		cart.BillingAddress = cmd.BillingAddress
	}

	currency, err := ensureSingleCurrency(cart)
	if err != nil {
		return PriceCartResult{}, err
	}

	if e.inventory != nil && len(cart.Items) > 0 {
		lines := make([]InventoryLine, 0, len(cart.Items))
		for _, item := range cart.Items {
			lines = append(lines, InventoryLine{ProductID: item.ProductID, SKU: item.SKU, Quantity: item.Quantity})
		}
		if err := e.inventory.ValidateAvailability(ctx, lines); err != nil {
			return PriceCartResult{}, err
		}
	}

	promotionCode := resolvePromotionCode(cart, cmd.PromotionCode)

	itemBreakdowns := make([]ItemPricingBreakdown, 0, len(cart.Items))
	itemDiscountTotals := make(map[string]int64)
	var subtotal int64
	taxWeights := make([]int64, 0, len(cart.Items))
	shippingWeights := make([]int64, 0, len(cart.Items))

	for _, item := range cart.Items {
		quantity := int64(item.Quantity)
		if item.UnitPrice > 0 && quantity > 0 {
			if item.UnitPrice > math.MaxInt64/quantity {
				return PriceCartResult{}, fmt.Errorf("%w: item %s subtotal overflow", ErrCartPricingInvalidInput, item.ID)
			}
		}

		lineSubtotal := item.UnitPrice * quantity
		if lineSubtotal < 0 {
			return PriceCartResult{}, fmt.Errorf("%w: negative line subtotal for item %s", ErrCartPricingInvalidInput, item.ID)
		}

		discountTotal := int64(0)
		perRuleContribution := make(map[string]int64)
		if len(e.itemRules) > 0 {
			for _, rule := range e.itemRules {
				result, ruleErr := rule.Apply(ctx, item, lineSubtotal)
				if ruleErr != nil {
					return PriceCartResult{}, ruleErr
				}
				if result.Amount < 0 {
					return PriceCartResult{}, fmt.Errorf("%w: rule %s produced negative discount", ErrCartPricingInvalidInput, rule.Name())
				}
				discountTotal += result.Amount
				perRuleContribution[rule.Name()] += result.Amount
			}
		}
		if discountTotal > lineSubtotal && len(perRuleContribution) > 0 {
			e.logger(ctx, "pricing_rule_clamped", map[string]any{"itemId": item.ID, "rule": "item", "subtotal": lineSubtotal, "discount": discountTotal})
			perRuleContribution = scaleDiscountAllocations(perRuleContribution, lineSubtotal)
		}

		discountTotal = 0
		for name, amount := range perRuleContribution {
			itemDiscountTotals[name] += amount
			discountTotal += amount
		}

		net := lineSubtotal - discountTotal
		if net < 0 {
			net = 0
		}

		if lineSubtotal > 0 && subtotal > math.MaxInt64-lineSubtotal {
			return PriceCartResult{}, fmt.Errorf("%w: cart subtotal overflow", ErrCartPricingInvalidInput)
		}
		subtotal += lineSubtotal

		taxWeights = append(taxWeights, net)
		shippingWeight := int64(0)
		if item.RequiresShipping {
			weight := int64(item.WeightGrams)
			if weight > 0 && quantity > 0 && weight > math.MaxInt64/quantity {
				shippingWeight = math.MaxInt64
			} else {
				shippingWeight = weight * quantity
			}
			if shippingWeight <= 0 {
				shippingWeight = net
			}
		}
		shippingWeights = append(shippingWeights, shippingWeight)

		itemBreakdowns = append(itemBreakdowns, ItemPricingBreakdown{
			ItemID:   item.ID,
			Currency: currency,
			Subtotal: lineSubtotal,
			Discount: discountTotal,
			Metadata: map[string]any{"quantity": item.Quantity, "unitPrice": item.UnitPrice, "weightGrams": item.WeightGrams},
		})
	}

	totalItemDiscount := sumInt64Map(itemDiscountTotals)

	promoDiscount, promoBreakdown, promoApplied, err := e.applyPromotion(ctx, cart, promotionCode)
	if err != nil {
		return PriceCartResult{}, err
	}
	if !promoApplied {
		promotionCode = nil
	}

	maxPromo := subtotal - totalItemDiscount
	if maxPromo < 0 {
		maxPromo = 0
	}
	if promoDiscount > maxPromo {
		e.logger(ctx, "pricing_discount_clamped", map[string]any{"subtotal": subtotal, "discount": totalItemDiscount + promoDiscount})
		promoDiscount = maxPromo
	}

	totalDiscount := totalItemDiscount + promoDiscount

	if len(promoBreakdown) > 0 {
		promoBreakdown[0].Amount = promoDiscount
	}
	if !promoApplied {
		promotionCode = nil
	}

	promoAlloc := allocateByWeight(promoDiscount, taxWeights)
	for idx := range itemBreakdowns {
		itemBreakdowns[idx].Discount += promoAlloc[idx]
	}

	// Recompute tax weights using net amounts after promotions to keep allocations aligned.
	for idx := range taxWeights {
		if idx < len(itemBreakdowns) {
			net := itemBreakdowns[idx].Subtotal - itemBreakdowns[idx].Discount
			if net < 0 {
				net = 0
			}
			taxWeights[idx] = net
		} else {
			taxWeights[idx] = 0
		}
	}

	netSubtotal := subtotal - totalDiscount
	if netSubtotal < 0 {
		netSubtotal = 0
	}

	shippingAmount, shippingBreakdown, err := e.calculateShipping(ctx, currency, cart, subtotal, totalDiscount, promotionCode, cmd.BypassShippingCache)
	if err != nil {
		return PriceCartResult{}, err
	}

	taxAmount, taxBreakdown, err := e.calculateTax(ctx, currency, cart, itemBreakdowns, subtotal, totalDiscount, shippingAmount, promotionCode)
	if err != nil {
		return PriceCartResult{}, err
	}

	distributeTaxAndShipping(itemBreakdowns, taxWeights, shippingWeights, taxAmount, shippingAmount)

	total := netSubtotal + taxAmount + shippingAmount
	if total < 0 {
		total = 0
	}

	discounts := buildDiscountBreakdowns(itemDiscountTotals, promoBreakdown)

	metadata := map[string]any{"netSubtotal": netSubtotal}
	if promotionCode != nil {
		metadata["promotionCode"] = *promotionCode
	}

	breakdown := PricingBreakdown{
		Currency:        currency,
		Subtotal:        subtotal,
		Discount:        totalDiscount,
		Tax:             taxAmount,
		Shipping:        shippingAmount,
		Total:           total,
		Rounding:        0,
		Items:           itemBreakdowns,
		Discounts:       discounts,
		Taxes:           taxBreakdown,
		ShippingDetails: shippingBreakdown,
		Metadata:        metadata,
	}

	estimate := CartEstimate{
		Subtotal: subtotal,
		Discount: totalDiscount,
		Tax:      taxAmount,
		Shipping: shippingAmount,
		Total:    total,
	}

	return PriceCartResult{Breakdown: breakdown, Estimate: estimate}, nil
}

func (e *CartPricingEngine) validateCartInput(cmd PriceCartCommand) error {
	cartCurrency := strings.TrimSpace(cmd.Cart.Currency)
	if len(cmd.Cart.Items) == 0 {
		if cartCurrency == "" {
			return fmt.Errorf("%w: cart currency required when no items provided", ErrCartPricingInvalidInput)
		}
		return nil
	}

	for _, item := range cmd.Cart.Items {
		if item.Quantity <= 0 {
			return fmt.Errorf("%w: item %s quantity must be positive", ErrCartPricingInvalidInput, item.ID)
		}
		if item.UnitPrice < 0 {
			return fmt.Errorf("%w: item %s unit price cannot be negative", ErrCartPricingInvalidInput, item.ID)
		}
		if strings.TrimSpace(item.Currency) == "" && cartCurrency == "" {
			return fmt.Errorf("%w: item %s currency missing", ErrCartPricingInvalidInput, item.ID)
		}
		if cartCurrency == "" {
			cartCurrency = strings.TrimSpace(item.Currency)
		}
	}
	return nil
}

func ensureSingleCurrency(cart Cart) (string, error) {
	base := strings.ToUpper(strings.TrimSpace(cart.Currency))
	if base == "" {
		if len(cart.Items) == 0 {
			return "", ErrCartPricingCurrencyMismatch
		}
		base = strings.ToUpper(strings.TrimSpace(cart.Items[0].Currency))
		if base == "" {
			return "", ErrCartPricingCurrencyMismatch
		}
	}
	for _, item := range cart.Items {
		itemCurrency := strings.ToUpper(strings.TrimSpace(item.Currency))
		if itemCurrency == "" {
			itemCurrency = base
		}
		if itemCurrency != base {
			return "", ErrCartPricingCurrencyMismatch
		}
	}
	return base, nil
}

func resolvePromotionCode(cart Cart, override *string) *string {
	if override != nil {
		trimmed := strings.TrimSpace(*override)
		if trimmed == "" {
			return nil
		}
		return stringPtr(strings.ToUpper(trimmed))
	}
	if cart.Promotion != nil {
		trimmed := strings.TrimSpace(cart.Promotion.Code)
		if trimmed == "" {
			return nil
		}
		return stringPtr(strings.ToUpper(trimmed))
	}
	return nil
}

func (e *CartPricingEngine) applyPromotion(ctx context.Context, cart Cart, promoCode *string) (int64, []DiscountBreakdown, bool, error) {
	if promoCode == nil {
		return 0, nil, false, nil
	}
	cmd := ValidatePromotionCommand{Code: *promoCode}
	if cart.UserID != "" {
		cmd.UserID = &cart.UserID
	}
	if cart.ID != "" {
		cmd.CartID = &cart.ID
	}

	result, err := e.promotion.ValidatePromotion(ctx, cmd)
	if err != nil {
		return 0, nil, false, err
	}
	if !result.Eligible {
		return 0, nil, false, nil
	}

	discount := result.DiscountAmount
	if discount < 0 {
		discount = 0
	}
	breakdown := DiscountBreakdown{
		Type:        "promotion",
		Code:        result.Code,
		Source:      "promotion_service",
		Description: result.Reason,
		Amount:      discount,
	}
	return discount, []DiscountBreakdown{breakdown}, true, nil
}

func (e *CartPricingEngine) calculateTax(ctx context.Context, currency string, cart Cart, items []ItemPricingBreakdown, cartSubtotal, discountTotal, shippingAmount int64, promoCode *string) (int64, []TaxBreakdown, error) {
	if e.tax == nil {
		return 0, nil, nil
	}
	reqItems := make([]TaxableItem, 0, len(items))
	for idx, item := range items {
		cartItem := cart.Items[idx]
		reqItems = append(reqItems, TaxableItem{
			ItemID:   item.ItemID,
			SKU:      cartItem.SKU,
			Quantity: cartItem.Quantity,
			Subtotal: item.Subtotal,
			Discount: item.Discount,
			TaxCode:  cartItem.TaxCode,
		})
	}

	req := TaxCalculationRequest{
		Currency:        currency,
		Items:           reqItems,
		CartSubtotal:    cartSubtotal,
		DiscountTotal:   discountTotal,
		ShippingAmount:  shippingAmount,
		BillingAddress:  cart.BillingAddress,
		ShippingAddress: cart.ShippingAddress,
		PromotionCode:   promoCode,
	}

	quote, err := e.tax.CalculateTax(ctx, req)
	if err != nil {
		return 0, nil, err
	}
	if quote.Amount < 0 {
		return 0, nil, fmt.Errorf("%w: tax amount cannot be negative", ErrCartPricingInvalidInput)
	}
	return quote.Amount, quote.Breakdown, nil
}

func (e *CartPricingEngine) calculateShipping(ctx context.Context, currency string, cart Cart, subtotal int64, discount int64, promoCode *string, bypassCache bool) (int64, []ShippingBreakdown, error) {
	if e.shipping == nil {
		return 0, nil, nil
	}
	if cart.ShippingAddress == nil {
		return 0, nil, nil
	}

	shippable := make([]ShippableItem, 0, len(cart.Items))
	requiresShipment := false
	totalWeight := int64(0)
	for _, item := range cart.Items {
		if !item.RequiresShipping {
			continue
		}
		requiresShipment = true
		weight := int64(item.WeightGrams)
		quantity := int64(item.Quantity)
		if weight > 0 && quantity > 0 && weight > math.MaxInt64/quantity {
			weight = math.MaxInt64
		} else {
			weight *= quantity
		}
		if weight > 0 && totalWeight > math.MaxInt64-weight {
			totalWeight = math.MaxInt64
		} else {
			totalWeight += weight
		}
		shippable = append(shippable, ShippableItem{
			ItemID:           item.ID,
			SKU:              item.SKU,
			Quantity:         item.Quantity,
			WeightGrams:      item.WeightGrams,
			RequiresShipping: item.RequiresShipping,
		})
	}

	if !requiresShipment {
		return 0, nil, nil
	}

	promo := ""
	if promoCode != nil {
		promo = *promoCode
	}

	cacheKey := buildShippingCacheKey(cart.ShippingAddress, currency, totalWeight, subtotal, discount, promo, shippable)
	if !bypassCache {
		if quote, ok := e.cache.Get(cacheKey); ok {
			return quote.Amount, quote.Breakdown, nil
		}
	}

	req := ShippingEstimateRequest{
		Currency:        currency,
		Items:           shippable,
		ShippingAddress: cart.ShippingAddress,
		CartSubtotal:    subtotal,
		DiscountTotal:   discount,
		PromotionCode:   promoCode,
	}

	quote, err := e.shipping.EstimateShipping(ctx, req)
	if err != nil {
		return 0, nil, err
	}
	if quote.Amount < 0 {
		return 0, nil, fmt.Errorf("%w: shipping amount cannot be negative", ErrCartPricingInvalidInput)
	}

	e.cache.Put(cacheKey, quote)
	return quote.Amount, quote.Breakdown, nil
}

func distributeTaxAndShipping(items []ItemPricingBreakdown, taxWeights, shippingWeights []int64, taxAmount, shippingAmount int64) {
	if len(items) != len(taxWeights) || len(items) != len(shippingWeights) {
		return
	}
	taxAlloc := allocateByWeight(taxAmount, taxWeights)
	shipAlloc := allocateByWeight(shippingAmount, shippingWeights)
	for idx := range items {
		items[idx].Tax = taxAlloc[idx]
		items[idx].Shipping = shipAlloc[idx]
		items[idx].Total = items[idx].Subtotal - items[idx].Discount + taxAlloc[idx] + shipAlloc[idx]
		if items[idx].Total < 0 {
			items[idx].Total = 0
		}
	}
}

func allocateByWeight(amount int64, weights []int64) []int64 {
	if len(weights) == 0 {
		return nil
	}
	allocations := make([]int64, len(weights))
	if amount == 0 {
		return allocations
	}
	totalWeight := int64(0)
	for _, w := range weights {
		if w > 0 {
			totalWeight += w
		}
	}
	if totalWeight == 0 {
		// distribute evenly if all zero
		base := amount / int64(len(weights))
		remainder := amount % int64(len(weights))
		for i := range weights {
			allocations[i] = base
			if remainder > 0 {
				allocations[i]++
				remainder--
			}
		}
		return allocations
	}

	remainderPairs := make([]struct {
		idx       int
		remainder int64
	}, len(weights))

	distributed := int64(0)
	for i, w := range weights {
		if w < 0 {
			w = 0
		}
		share := (amount * w) / totalWeight
		allocations[i] = share
		distributed += share
		remainderPairs[i] = struct {
			idx       int
			remainder int64
		}{idx: i, remainder: (amount * w) % totalWeight}
	}

	remainder := amount - distributed
	if remainder <= 0 {
		return allocations
	}

	sort.SliceStable(remainderPairs, func(i, j int) bool {
		if remainderPairs[i].remainder == remainderPairs[j].remainder {
			return remainderPairs[i].idx < remainderPairs[j].idx
		}
		return remainderPairs[i].remainder > remainderPairs[j].remainder
	})

	for _, entry := range remainderPairs {
		if remainder == 0 {
			break
		}
		allocations[entry.idx]++
		remainder--
	}

	return allocations
}

func scaleDiscountAllocations(contrib map[string]int64, target int64) map[string]int64 {
	if len(contrib) == 0 {
		return contrib
	}
	if target <= 0 {
		for name := range contrib {
			contrib[name] = 0
		}
		return contrib
	}
	names := make([]string, 0, len(contrib))
	for name := range contrib {
		names = append(names, name)
	}
	sort.Strings(names)
	weights := make([]int64, len(names))
	for i, name := range names {
		amt := contrib[name]
		if amt < 0 {
			amt = 0
		}
		weights[i] = amt
	}
	totalWeight := int64(0)
	for _, w := range weights {
		totalWeight += w
	}
	if totalWeight == 0 {
		for _, name := range names {
			contrib[name] = 0
		}
		return contrib
	}
	alloc := allocateByWeight(target, weights)
	for i, name := range names {
		contrib[name] = alloc[i]
	}
	return contrib
}

func buildDiscountBreakdowns(itemTotals map[string]int64, promo []DiscountBreakdown) []DiscountBreakdown {
	result := make([]DiscountBreakdown, 0, len(itemTotals)+len(promo))
	for name, amount := range itemTotals {
		if amount <= 0 {
			continue
		}
		result = append(result, DiscountBreakdown{
			Type:        "item",
			Source:      name,
			Description: fmt.Sprintf("%s discount", name),
			Amount:      amount,
		})
	}
	if len(promo) > 0 {
		result = append(result, promo...)
	}
	if len(result) <= 1 {
		return result
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Type == result[j].Type {
			return result[i].Source < result[j].Source
		}
		return result[i].Type < result[j].Type
	})
	return result
}

func sumInt64Map(values map[string]int64) int64 {
	var total int64
	for _, v := range values {
		total += v
	}
	return total
}

func stringPtr(value string) *string {
	v := value
	return &v
}

type shippingQuoteCache struct {
	ttl time.Duration
	now func() time.Time
	mu  sync.RWMutex
	m   map[string]shippingCacheEntry
}

type shippingCacheEntry struct {
	quote   ShippingQuote
	expires time.Time
}

func newShippingQuoteCache(ttl time.Duration, now func() time.Time) *shippingQuoteCache {
	return &shippingQuoteCache{
		ttl: ttl,
		now: now,
		m:   make(map[string]shippingCacheEntry),
	}
}

func (c *shippingQuoteCache) Get(key string) (ShippingQuote, bool) {
	c.mu.RLock()
	entry, ok := c.m[key]
	c.mu.RUnlock()
	if !ok {
		return ShippingQuote{}, false
	}
	if c.now().After(entry.expires) {
		c.mu.Lock()
		delete(c.m, key)
		c.mu.Unlock()
		return ShippingQuote{}, false
	}
	return entry.quote, true
}

func (c *shippingQuoteCache) Put(key string, quote ShippingQuote) {
	c.mu.Lock()
	c.m[key] = shippingCacheEntry{quote: quote, expires: c.now().Add(c.ttl)}
	c.mu.Unlock()
}

func buildShippingCacheKey(addr *Address, currency string, totalWeight, subtotal, discount int64, promo string, items []ShippableItem) string {
	baseParts := []string{currency, fmt.Sprintf("%d", totalWeight), fmt.Sprintf("%d", subtotal), fmt.Sprintf("%d", discount), strings.ToUpper(strings.TrimSpace(promo))}
	if addr != nil {
		state := ""
		if addr.State != nil {
			state = *addr.State
		}
		baseParts = append([]string{
			strings.ToUpper(strings.TrimSpace(addr.Country)),
			strings.ToUpper(strings.TrimSpace(addr.PostalCode)),
			strings.ToUpper(strings.TrimSpace(state)),
		}, baseParts...)
	}

	itemParts := make([]string, len(items))
	for i, item := range items {
		itemParts[i] = strings.Join([]string{
			strings.ToUpper(strings.TrimSpace(item.SKU)),
			fmt.Sprintf("%d", item.Quantity),
			fmt.Sprintf("%d", item.WeightGrams),
		}, ",")
	}
	if len(itemParts) > 0 {
		sort.Strings(itemParts)
		baseParts = append(baseParts, strings.Join(itemParts, ";"))
	}

	return strings.Join(baseParts, "|")
}
