# Build reorder flow (`/orders/:orderId/reorder`) cloning cart data and redirecting to checkout.

**Parent Section:** 8. Orders & Tracking
**Task ID:** 052

## Goal
Implement reorder flow reusing previous order data.

## Implementation Steps
1. Clone order items and design snapshot into new cart via repository call.
2. Notify user of items out of stock or changed pricing.
3. Redirect to cart/checkout with success message.
