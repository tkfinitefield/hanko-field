package repositories

import "fmt"

// InventoryErrorCode enumerates repository error causes for inventory operations.
type InventoryErrorCode string

const (
	// InventoryErrorUnknown represents an unspecified failure.
	InventoryErrorUnknown InventoryErrorCode = "inventory_unknown"
	// InventoryErrorInsufficientStock indicates requested quantity exceeds availability.
	InventoryErrorInsufficientStock InventoryErrorCode = "inventory_insufficient_stock"
	// InventoryErrorStockNotFound indicates the SKU does not have a stock record.
	InventoryErrorStockNotFound InventoryErrorCode = "inventory_stock_not_found"
	// InventoryErrorReservationNotFound indicates the reservation document is missing.
	InventoryErrorReservationNotFound InventoryErrorCode = "inventory_reservation_not_found"
	// InventoryErrorInvalidReservationState indicates the reservation status forbids the operation.
	InventoryErrorInvalidReservationState InventoryErrorCode = "inventory_invalid_state"
)

// InventoryError wraps inventory-specific failures with machine readable codes.
type InventoryError struct {
	Op      string
	Code    InventoryErrorCode
	Message string
	Err     error
}

// Error implements the error interface.
func (e *InventoryError) Error() string {
	if e == nil {
		return ""
	}
	if e.Op != "" {
		return fmt.Sprintf("%s: %s", e.Op, e.Message)
	}
	return e.Message
}

// Unwrap exposes the underlying error, if any.
func (e *InventoryError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// NewInventoryError constructs a typed inventory error.
func NewInventoryError(code InventoryErrorCode, message string, err error) *InventoryError {
	if message == "" {
		message = string(code)
	}
	return &InventoryError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}
