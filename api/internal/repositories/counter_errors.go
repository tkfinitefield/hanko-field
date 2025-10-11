package repositories

import "fmt"

// CounterErrorCode enumerates failure reasons for counter operations.
type CounterErrorCode string

const (
	// CounterErrorUnknown represents an unspecified failure.
	CounterErrorUnknown CounterErrorCode = "counter_unknown"
	// CounterErrorInvalidInput indicates the caller supplied invalid arguments.
	CounterErrorInvalidInput CounterErrorCode = "counter_invalid_input"
	// CounterErrorExhausted indicates the counter cannot be incremented further due to a configured max value.
	CounterErrorExhausted CounterErrorCode = "counter_exhausted"
)

// CounterError wraps counter-specific failures with machine readable codes.
type CounterError struct {
	Op      string
	Code    CounterErrorCode
	Message string
	Err     error
}

// Error implements the error interface.
func (e *CounterError) Error() string {
	if e == nil {
		return ""
	}
	if e.Op != "" {
		return fmt.Sprintf("%s: %s", e.Op, e.Message)
	}
	return e.Message
}

// Unwrap exposes the underlying error, if any.
func (e *CounterError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// NewCounterError constructs a typed counter error.
func NewCounterError(code CounterErrorCode, message string, err error) *CounterError {
	if message == "" {
		message = string(code)
	}
	return &CounterError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}
