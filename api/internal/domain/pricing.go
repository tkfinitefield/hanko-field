package domain

// PricingBreakdown captures the aggregated monetary results of pricing a cart.
type PricingBreakdown struct {
	Currency        string
	Subtotal        int64
	Discount        int64
	Tax             int64
	Shipping        int64
	Total           int64
	Rounding        int64
	Items           []ItemPricingBreakdown
	Discounts       []DiscountBreakdown
	Taxes           []TaxBreakdown
	ShippingDetails []ShippingBreakdown
	Metadata        map[string]any
}

// ItemPricingBreakdown stores the per-item pricing outputs after running the engine.
type ItemPricingBreakdown struct {
	ItemID   string
	Currency string
	Subtotal int64
	Discount int64
	Tax      int64
	Shipping int64
	Total    int64
	Metadata map[string]any
}

// DiscountBreakdown lists the individual discount adjustments applied to the cart.
type DiscountBreakdown struct {
	Type        string
	Code        string
	Source      string
	Description string
	Amount      int64
	Metadata    map[string]any
}

// TaxBreakdown captures individual tax components returned by the tax calculator.
type TaxBreakdown struct {
	Name         string
	Jurisdiction string
	Rate         float64
	Amount       int64
	Metadata     map[string]any
}

// ShippingBreakdown records the selected shipping option and associated costs.
type ShippingBreakdown struct {
	ServiceLevel string
	Carrier      string
	Amount       int64
	Currency     string
	EstimateDays *int
	Metadata     map[string]any
}
