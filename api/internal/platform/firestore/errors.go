package firestore

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Error implements repositories.RepositoryError for Firestore backed repositories.
type Error struct {
	op          string
	err         error
	notFound    bool
	conflict    bool
	unavailable bool
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.op != "" {
		return fmt.Sprintf("%s: %v", e.op, e.err)
	}
	return e.err.Error()
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// IsNotFound reports whether the error represents a missing document.
func (e *Error) IsNotFound() bool {
	return e != nil && e.notFound
}

// IsConflict reports whether the error represents a conflicting update.
func (e *Error) IsConflict() bool {
	return e != nil && e.conflict
}

// IsUnavailable reports whether the error represents a transient backend outage.
func (e *Error) IsUnavailable() bool {
	return e != nil && e.unavailable
}

func newError(op string, err error) *Error {
	if err == nil {
		return nil
	}

	code := status.Code(err)
	e := &Error{op: op, err: err}
	switch code {
	case codes.NotFound:
		e.notFound = true
	case codes.AlreadyExists, codes.FailedPrecondition, codes.Aborted, codes.OutOfRange:
		e.conflict = true
	case codes.Unavailable, codes.ResourceExhausted, codes.Internal:
		e.unavailable = true
	case codes.DeadlineExceeded:
		e.unavailable = true
	}
	return e
}

// WrapError annotates Firestore errors with repository semantics. Context cancellations are passed through.
func WrapError(op string, err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}

	switch status.Code(err) {
	case codes.Canceled:
		return context.Canceled
	case codes.DeadlineExceeded:
		return context.DeadlineExceeded
	}

	var repoErr *Error
	if errors.As(err, &repoErr) {
		if op != "" && repoErr.op == "" {
			repoErr.op = op
		}
		return repoErr
	}
	return newError(op, err)
}
