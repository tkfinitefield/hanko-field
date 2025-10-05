package idempotency

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hanko-field/api/internal/platform/auth"
)

const (
	defaultHeaderName = "Idempotency-Key"
	replayHeaderName  = "X-Idempotent-Replay"
)

// Logger abstracts the logging dependency used inside the middleware.
type Logger interface {
	Printf(format string, args ...any)
}

type clockFunc func() time.Time

type middlewareConfig struct {
	headerName string
	ttl        time.Duration
	methods    map[string]struct{}
	clock      clockFunc
	logger     Logger
}

// MiddlewareOption customises middleware behaviour.
type MiddlewareOption func(*middlewareConfig)

// WithHeader overrides the header name used to extract the idempotency key.
func WithHeader(name string) MiddlewareOption {
	return func(cfg *middlewareConfig) {
		name = strings.TrimSpace(name)
		if name != "" {
			cfg.headerName = name
		}
	}
}

// WithTTL configures how long completed idempotency records are retained.
func WithTTL(ttl time.Duration) MiddlewareOption {
	return func(cfg *middlewareConfig) {
		if ttl > 0 {
			cfg.ttl = ttl
		}
	}
}

// WithMethods restricts the HTTP methods guarded by the middleware.
func WithMethods(methods ...string) MiddlewareOption {
	return func(cfg *middlewareConfig) {
		if len(methods) == 0 {
			return
		}
		cfg.methods = make(map[string]struct{}, len(methods))
		for _, method := range methods {
			method = strings.ToUpper(strings.TrimSpace(method))
			if method == "" {
				continue
			}
			cfg.methods[method] = struct{}{}
		}
	}
}

// WithLogger injects a logger for background persistence errors.
func WithLogger(logger Logger) MiddlewareOption {
	return func(cfg *middlewareConfig) {
		cfg.logger = logger
	}
}

// WithClock overrides the time source, primarily for testing.
func WithClock(clock clockFunc) MiddlewareOption {
	return func(cfg *middlewareConfig) {
		if clock != nil {
			cfg.clock = clock
		}
	}
}

// Middleware constructs an HTTP middleware enforcing idempotency semantics for mutating requests.
func Middleware(store Store, opts ...MiddlewareOption) func(http.Handler) http.Handler {
	if store == nil {
		return func(next http.Handler) http.Handler { return next }
	}

	cfg := middlewareConfig{
		headerName: defaultHeaderName,
		ttl:        DefaultTTL,
		methods: map[string]struct{}{
			http.MethodPost:   {},
			http.MethodPut:    {},
			http.MethodPatch:  {},
			http.MethodDelete: {},
		},
		clock: time.Now,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.ttl <= 0 {
		cfg.ttl = DefaultTTL
	}
	if len(cfg.methods) == 0 {
		cfg.methods = map[string]struct{}{
			http.MethodPost:   {},
			http.MethodPut:    {},
			http.MethodPatch:  {},
			http.MethodDelete: {},
		}
	}
	if cfg.clock == nil {
		cfg.clock = time.Now
	}

	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {})
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := cfg.methods[r.Method]; !ok {
				next.ServeHTTP(w, r)
				return
			}

			key := strings.TrimSpace(r.Header.Get(cfg.headerName))
			if key == "" {
				respondError(w, http.StatusBadRequest, "idempotency_key_required", "missing idempotency key header")
				return
			}

			body, err := readAndReplayBody(r)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "idempotency_read_body_failed", "unable to read request body")
				return
			}

			identity := extractRequester(r.Context())
			fingerprint := requestFingerprint(r, body, identity)
			scopedKey := scopedKey(key, identity)
			now := cfg.clock().UTC()

			reservation, err := store.Reserve(r.Context(), scopedKey, fingerprint, now, cfg.ttl)
			if err != nil {
				handleStoreError(w, cfg.logger, err)
				return
			}

			switch reservation.State {
			case ReservationStateCompleted:
				writeStoredResponse(w, reservation.Record)
				return
			case ReservationStatePending:
				respondError(w, http.StatusConflict, "idempotency_in_progress", "another request is processing this idempotency key")
				return
			case ReservationStateNew:
				// Continue to handler.
			default:
				respondError(w, http.StatusInternalServerError, "idempotency_unknown_state", "unexpected idempotency state")
				return
			}

			recorder := newResponseRecorder(w)
			next.ServeHTTP(recorder, r)

			response := Response{
				Status:  recorder.Status(),
				Headers: recorder.HeaderSnapshot(),
				Body:    recorder.Body(),
			}

			if err := store.SaveResponse(r.Context(), scopedKey, fingerprint, response, cfg.clock().UTC(), cfg.ttl); err != nil {
				if cfg.logger != nil {
					cfg.logger.Printf("idempotency: failed to persist response for key %s (identity %s): %v", key, identity, err)
				}
				if releaseErr := store.Release(r.Context(), scopedKey, fingerprint); releaseErr != nil && cfg.logger != nil {
					cfg.logger.Printf("idempotency: failed to release key %s after save failure: %v", key, releaseErr)
				}
				respondError(w, http.StatusInternalServerError, "idempotency_store_error", "unable to persist idempotency state")
				return
			}

			if err := recorder.Commit(); err != nil && cfg.logger != nil {
				cfg.logger.Printf("idempotency: failed to flush response for key %s: %v", key, err)
			}
		})
	}
}

func readAndReplayBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if err := r.Body.Close(); err != nil {
		return nil, err
	}
	r.Body = io.NopCloser(bytes.NewReader(data))
	return data, nil
}

func requestFingerprint(r *http.Request, body []byte, identity string) string {
	builder := strings.Builder{}
	builder.WriteString(strings.ToUpper(r.Method))
	builder.WriteString("|")
	builder.WriteString(r.URL.Path)
	builder.WriteString("|")
	builder.WriteString(r.URL.RawQuery)
	builder.WriteString("|")
	builder.WriteString(r.Host)
	builder.WriteString("|")
	builder.WriteString(r.Header.Get("Content-Type"))
	builder.WriteString("|")
	builder.WriteString(identity)
	builder.WriteString("|")
	builder.WriteString(hashBody(body))

	return sha256Hex([]byte(builder.String()))
}

func extractRequester(ctx context.Context) string {
	if identity, ok := auth.IdentityFromContext(ctx); ok && identity != nil && identity.UID != "" {
		return identity.UID
	}
	if svc, ok := auth.ServiceIdentityFromContext(ctx); ok && svc != nil && svc.Subject != "" {
		return svc.Subject
	}
	return "anonymous"
}

func hashBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	return sha256Hex(body)
}

func scopedKey(key, identity string) string {
	key = strings.TrimSpace(key)
	identity = strings.TrimSpace(identity)
	if identity == "" {
		identity = "anonymous"
	}
	if key == "" {
		return identity
	}
	return key + "|" + identity
}

func handleStoreError(w http.ResponseWriter, logger Logger, err error) {
	switch {
	case errors.Is(err, ErrFingerprintMismatch):
		respondError(w, http.StatusConflict, "idempotency_key_conflict", "idempotency key already used for a different request")
	default:
		if logger != nil {
			logger.Printf("idempotency: store error: %v", err)
		}
		respondError(w, http.StatusInternalServerError, "idempotency_store_error", "unable to process idempotency key")
	}
}

func writeStoredResponse(w http.ResponseWriter, record Record) {
	headers := headersFromRecord(record.ResponseHeaders)
	for key := range w.Header() {
		w.Header().Del(key)
	}
	for key, values := range headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.Header().Set(replayHeaderName, "true")

	status := record.ResponseStatus
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	if len(record.ResponseBody) > 0 {
		_, _ = w.Write(record.ResponseBody)
	}
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error":   code,
		"message": message,
	})
}

type responseRecorder struct {
	parent http.ResponseWriter
	header http.Header
	status int
	body   bytes.Buffer
}

func newResponseRecorder(parent http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		parent: parent,
		header: make(http.Header),
	}
}

func (r *responseRecorder) Header() http.Header {
	return r.header
}

func (r *responseRecorder) WriteHeader(status int) {
	if status <= 0 {
		status = http.StatusOK
	}
	r.status = status
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.body.Write(data)
}

func (r *responseRecorder) Status() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

func (r *responseRecorder) Body() []byte {
	if r.body.Len() == 0 {
		return nil
	}
	return r.body.Bytes()
}

func (r *responseRecorder) HeaderSnapshot() http.Header {
	return cloneHeader(r.header)
}

func (r *responseRecorder) Commit() error {
	dst := r.parent.Header()
	for key := range dst {
		dst.Del(key)
	}
	for key, values := range r.header {
		for _, value := range values {
			dst.Add(key, value)
		}
	}

	status := r.status
	if status == 0 {
		status = http.StatusOK
	}
	r.parent.WriteHeader(status)
	if r.body.Len() == 0 {
		return nil
	}
	_, err := r.parent.Write(r.body.Bytes())
	return err
}

func cloneHeader(src http.Header) http.Header {
	if len(src) == 0 {
		return http.Header{}
	}
	dst := make(http.Header, len(src))
	for key, values := range src {
		copied := make([]string, len(values))
		copy(copied, values)
		dst[key] = copied
	}
	return dst
}
