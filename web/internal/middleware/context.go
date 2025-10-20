package middleware

import (
	"context"
)

// context keys are unexported to avoid collisions
type ctxKey string

const (
    ctxKeyRequestID ctxKey = "req_id"
    ctxKeyIsHTMX    ctxKey = "is_htmx"
    ctxKeySession   ctxKey = "session"
    ctxKeyUser      ctxKey = "user"
    ctxKeyLocaleFB  ctxKey = "locale_fallback"
)

// WithRequestID stores request id in context
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyRequestID, id)
}

// RequestID gets request id from context
func RequestID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyRequestID).(string)
	return v, ok
}

// WithHTMX marks request as HTMX
func WithHTMX(ctx context.Context, is bool) context.Context {
	return context.WithValue(ctx, ctxKeyIsHTMX, is)
}

// IsHTMX returns whether this is an htmx request
func IsHTMX(ctx context.Context) bool {
	v, _ := ctx.Value(ctxKeyIsHTMX).(bool)
	return v
}

// User represents authenticated user info
type User struct {
	ID    string `json:"id"`
	Email string `json:"email,omitempty"`
}

// WithUser stores user in context
func WithUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, ctxKeyUser, u)
}

// UserFromContext returns user if present
func UserFromContext(ctx context.Context) *User {
	if v := ctx.Value(ctxKeyUser); v != nil {
		if u, ok := v.(*User); ok {
			return u
		}
	}
	return nil
}
