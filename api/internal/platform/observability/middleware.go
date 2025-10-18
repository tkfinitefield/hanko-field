package observability

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/platform/httpx"
	"github.com/hanko-field/api/internal/platform/requestctx"
)

// InjectLoggerMiddleware stores the provided logger on the request context to make it accessible downstream.
func InjectLoggerMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := requestctx.WithLogger(r.Context(), logger)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequestLoggerMiddleware logs request start and completion with structured fields suitable for Cloud Logging.
func RequestLoggerMiddleware(projectID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			baseLogger := requestctx.Logger(ctx)
			traceInfo, _ := requestctx.Trace(ctx)
			requestID := middleware.GetReqID(ctx)
			userID := sanitizedUserID(ctx)
			route := routePattern(r)
			method := SanitizeMethod(r.Method)
			logger := WithRequestFields(baseLogger,
				zap.String("request_id", requestID),
				zap.String("method", method),
				zap.String("route", SanitizeRoute(route)),
				zap.String("trace_id", traceInfo.TraceID),
				zap.String("user_id", userID),
			)
			if traceResource := loggingTraceResource(traceInfo); traceResource != "" {
				logger = logger.With(zap.String("logging.googleapis.com/trace", traceResource))
			}
			if ip := realIP(r); ip != "" {
				logger = logger.With(zap.String("remote_ip", ip))
			}

			ctx = requestctx.WithLogger(ctx, logger)
			r = r.WithContext(ctx)

			recorder := newResponseRecorder(w)
			start := time.Now()
			logger.Info("request started")

			var panicked bool
			defer func() {
				latency := time.Since(start)
				status := recorder.Status()
				if panicked && status < http.StatusInternalServerError {
					status = http.StatusInternalServerError
				}

				span := trace.SpanFromContext(ctx)
				if span != nil {
					attrs := semconvStatusAttributes(status)
					if route != "" {
						attrs = append(attrs, semconv.HTTPRoute(SanitizeRoute(route)))
					}
					span.SetAttributes(attrs...)
					setSpanStatus(span, status)
				}

				fields := []zap.Field{
					zap.Int("status", status),
					zap.Duration("latency", latency),
					zap.Int64("bytes", recorder.BytesWritten()),
				}

				if panicked || status >= http.StatusInternalServerError {
					logger.Error("request completed", fields...)
				} else if status >= http.StatusBadRequest {
					logger.Warn("request completed", fields...)
				} else {
					logger.Info("request completed", fields...)
				}
			}()

			defer func() {
				if rec := recover(); rec != nil {
					panicked = true
					panic(rec)
				}
			}()

			next.ServeHTTP(recorder, r)
		})
	}
}

// RecoveryMiddleware captures panics, logs the stack trace, and returns a JSON error response.
func RecoveryMiddleware(fallback *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					ctx := r.Context()
					logger := requestctx.Logger(ctx)
					if logger == nil || logger == requestctx.NoopLogger() {
						logger = fallback
						if logger == nil {
							logger = requestctx.NoopLogger()
						}
					}
					logger.Error("panic recovered",
						zap.Any("panic", rec),
						zap.ByteString("stack", debug.Stack()),
					)

					httpx.WriteError(ctx, w, httpx.NewError("internal_server_error", "internal server error", http.StatusInternalServerError))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

func sanitizedUserID(ctx context.Context) string {
	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil {
		return ""
	}
	return SanitizeUserID(identity.UID)
}

func routePattern(r *http.Request) string {
	if r == nil {
		return "/"
	}
	if ctx := chi.RouteContext(r.Context()); ctx != nil {
		if pattern := ctx.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	if r.URL != nil && r.URL.Path != "" {
		return r.URL.Path
	}
	return "/"
}

func realIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	addr := strings.TrimSpace(r.RemoteAddr)
	if addr == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(addr); err == nil {
		addr = host
	}
	return sanitizeString(addr, 64)
}

func loggingTraceResource(info requestctx.TraceInfo) string {
	if info.ProjectID == "" || info.TraceID == "" {
		return ""
	}
	return fmt.Sprintf("projects/%s/traces/%s", info.ProjectID, info.TraceID)
}

func semconvStatusAttributes(status int) []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.HTTPResponseStatusCode(status),
	}
}

func setSpanStatus(span trace.Span, status int) {
	if span == nil {
		return
	}
	if status >= http.StatusInternalServerError {
		span.SetStatus(codes.Error, http.StatusText(status))
		return
	}
	span.SetStatus(codes.Ok, http.StatusText(status))
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{ResponseWriter: w, status: http.StatusOK}
}

func (r *responseRecorder) WriteHeader(status int) {
	if status < 100 {
		status = http.StatusOK
	}
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.bytes += int64(n)
	return n, err
}

func (r *responseRecorder) Status() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

func (r *responseRecorder) BytesWritten() int64 {
	return r.bytes
}
