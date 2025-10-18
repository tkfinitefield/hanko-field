package firestore

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
)

const (
	defaultTxAttempts = 5
	defaultTxTimeout  = 15 * time.Second
)

// TxFunc is executed within a Firestore transaction.
type TxFunc func(ctx context.Context, tx *firestore.Transaction) error

// TxOption customises transaction behaviour.
type TxOption func(*txConfig)

type txConfig struct {
	attempts int
	timeout  time.Duration
}

// WithTxAttempts overrides the retry attempts for a transaction.
func WithTxAttempts(attempts int) TxOption {
	return func(cfg *txConfig) {
		if attempts > 0 {
			cfg.attempts = attempts
		}
	}
}

// WithTxTimeout sets a timeout for the transaction context.
func WithTxTimeout(timeout time.Duration) TxOption {
	return func(cfg *txConfig) {
		if timeout > 0 {
			cfg.timeout = timeout
		}
	}
}

// RunTransaction executes fn within a transaction on the provided client.
func RunTransaction(ctx context.Context, client *firestore.Client, fn TxFunc, opts ...TxOption) error {
	if client == nil {
		return WrapError("transaction", errors.New("firestore: client is nil"))
	}
	if fn == nil {
		return WrapError("transaction", errors.New("firestore: transaction function is nil"))
	}

	cfg := txConfig{attempts: defaultTxAttempts, timeout: defaultTxTimeout}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	txnCtx := ctx
	var cancel context.CancelFunc
	if cfg.timeout > 0 {
		deadline, hasDeadline := ctx.Deadline()
		if !hasDeadline || time.Until(deadline) > cfg.timeout {
			txnCtx, cancel = context.WithTimeout(ctx, cfg.timeout)
		}
	}
	if cancel != nil {
		defer cancel()
	}

	firestoreOpts := make([]firestore.TransactionOption, 0, 1)
	if cfg.attempts > 0 {
		firestoreOpts = append(firestoreOpts, firestore.MaxAttempts(cfg.attempts))
	}

	err := client.RunTransaction(txnCtx, func(ctx context.Context, tx *firestore.Transaction) error {
		return fn(ctx, tx)
	}, firestoreOpts...)

	return WrapError("transaction", err)
}
