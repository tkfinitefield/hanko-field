package httpx

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/hanko-field/api/internal/platform/requestctx"
)

// Error represents the canonical JSON error envelope returned by the API.
type Error struct {
	Code      string
	Message   string
	Status    int
	RequestID string
	TraceID   string
	Details   map[string]any
}

// NewError constructs a new Error with the provided parameters.
func NewError(code, message string, status int) Error {
	if status == 0 {
		status = http.StatusInternalServerError
	}
	return Error{
		Code:    sanitize(code, 80),
		Message: sanitize(message, 512),
		Status:  status,
	}
}

// WithRequestID sets the request identifier on the error payload.
func (e Error) WithRequestID(id string) Error {
	e.RequestID = sanitize(id, 80)
	return e
}

// WithTraceID sets the trace identifier on the error payload.
func (e Error) WithTraceID(id string) Error {
	e.TraceID = sanitize(id, 64)
	return e
}

// WithDetails attaches additional JSON-serialisable metadata.
func (e Error) WithDetails(details map[string]any) Error {
	if len(details) == 0 {
		return e
	}
	copyDetails := make(map[string]any, len(details))
	for k, v := range details {
		copyDetails[k] = v
	}
	e.Details = copyDetails
	return e
}

// WriteError writes the structured error as JSON to the provided response writer.
func WriteError(ctx context.Context, w http.ResponseWriter, err Error) {
	status := err.Status
	if status == 0 {
		status = http.StatusInternalServerError
	}

	requestID := err.RequestID
	if requestID == "" {
		requestID = sanitize(middleware.GetReqID(ctx), 80)
	}

	traceID := err.TraceID
	if traceID == "" {
		traceID = sanitize(requestctx.TraceID(ctx), 64)
	}

	payload := map[string]any{
		"error":   err.Code,
		"message": err.Message,
		"status":  status,
	}

	if requestID != "" {
		payload["request_id"] = requestID
	}
	if traceID != "" {
		payload["trace_id"] = traceID
	}
	for k, v := range err.Details {
		payload[k] = v
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func sanitize(value string, limit int) string {
	if limit <= 0 {
		limit = 256
	}
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.TrimSpace(value)
	if len(value) > limit {
		value = value[:limit]
	}
	return value
}
