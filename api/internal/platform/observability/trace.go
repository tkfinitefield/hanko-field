package observability

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/hanko-field/api/internal/platform/requestctx"
)

const cloudTraceHeader = "X-Cloud-Trace-Context"

var tracer = otel.Tracer("github.com/hanko-field/api/internal/platform/observability")

// TraceMiddleware extracts Cloud Trace headers, starts a server span, and stores trace metadata on the request context.
func TraceMiddleware(projectID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			headerVal := r.Header.Get(cloudTraceHeader)
			info, remoteSpanCtx, ok := parseCloudTraceContext(headerVal)
			if ok {
				ctx = trace.ContextWithRemoteSpanContext(ctx, remoteSpanCtx)
			}

			spanName := spanNameFromRequest(r)
			ctx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))
			span.SetAttributes(standardSpanAttributes(r)...)

			spanCtx := span.SpanContext()
			info.TraceID = spanCtx.TraceID().String()
			info.SpanID = spanCtx.SpanID().String()
			info.Sampled = spanCtx.IsSampled()
			info.ProjectID = projectID

			ctx = requestctx.WithTrace(ctx, info)
			r = r.WithContext(ctx)

			if formatted := formatCloudTraceHeader(info); formatted != "" {
				w.Header().Set(cloudTraceHeader, formatted)
			}

			defer span.End()
			next.ServeHTTP(w, r)
		})
	}
}

func parseCloudTraceContext(header string) (requestctx.TraceInfo, trace.SpanContext, bool) {
	header = strings.TrimSpace(header)
	if header == "" {
		return requestctx.TraceInfo{}, trace.SpanContext{}, false
	}

	parts := strings.SplitN(header, "/", 2)
	if len(parts) != 2 {
		return requestctx.TraceInfo{}, trace.SpanContext{}, false
	}

	traceIDHex := strings.TrimSpace(parts[0])
	if len(traceIDHex) != 32 {
		return requestctx.TraceInfo{}, trace.SpanContext{}, false
	}
	traceID, err := trace.TraceIDFromHex(traceIDHex)
	if err != nil {
		return requestctx.TraceInfo{}, trace.SpanContext{}, false
	}

	spanPart := parts[1]
	optionPart := ""
	if idx := strings.Index(spanPart, ";"); idx >= 0 {
		optionPart = spanPart[idx+1:]
		spanPart = spanPart[:idx]
	}

	spanIDHex := strings.TrimSpace(spanPart)
	spanID, ok := parseSpanID(spanIDHex)
	if !ok {
		return requestctx.TraceInfo{}, trace.SpanContext{}, false
	}

	sampled := parseTraceOptions(optionPart)
	flags := trace.TraceFlags(0)
	if sampled {
		flags = trace.FlagsSampled
	}

	spanCtxConfig := trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: flags,
		Remote:     true,
	}

	return requestctx.TraceInfo{
		TraceID: traceID.String(),
		SpanID:  spanID.String(),
		Sampled: sampled,
	}, trace.NewSpanContext(spanCtxConfig), true
}

func parseSpanID(value string) (trace.SpanID, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return trace.SpanID{}, false
	}

	if len(value) <= 16 && isHex(value) {
		if len(value) < 16 {
			value = strings.Repeat("0", 16-len(value)) + value
		}
		spanID, err := trace.SpanIDFromHex(value)
		if err == nil {
			return spanID, true
		}
	}

	// Fallback attempt to parse decimal encoded span IDs.
	if num, err := strconv.ParseUint(value, 10, 64); err == nil {
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], num)
		var spanID trace.SpanID
		copy(spanID[:], buf[:])
		if spanID.IsValid() {
			return spanID, true
		}
	}

	return trace.SpanID{}, false
}

func parseTraceOptions(optionPart string) bool {
	optionPart = strings.TrimSpace(optionPart)
	if optionPart == "" {
		return false
	}
	segments := strings.Split(optionPart, ";")
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if strings.HasPrefix(segment, "o=") {
			return segment == "o=1"
		}
	}
	return false
}

func isHex(value string) bool {
	if value == "" {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func formatCloudTraceHeader(info requestctx.TraceInfo) string {
	if info.TraceID == "" || info.SpanID == "" {
		return ""
	}
	option := "0"
	if info.Sampled {
		option = "1"
	}
	return fmt.Sprintf("%s/%s;o=%s", info.TraceID, info.SpanID, option)
}

func spanNameFromRequest(r *http.Request) string {
	if r == nil {
		return "unknown"
	}
	path := r.URL.Path
	if path == "" {
		path = "/"
	}
	return fmt.Sprintf("%s %s", r.Method, path)
}

func standardSpanAttributes(r *http.Request) []attribute.KeyValue {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	attrs := []attribute.KeyValue{
		attribute.String("http.request.method", r.Method),
		attribute.String("url.scheme", scheme),
	}
	if r.URL != nil {
		if path := r.URL.Path; path != "" {
			attrs = append(attrs, attribute.String("url.path", path))
		}
		if target := r.URL.RequestURI(); target != "" {
			attrs = append(attrs, attribute.String("url.full", target))
		}
	}
	if host := r.Host; host != "" {
		attrs = append(attrs, attribute.String("server.address", host))
	}
	if ua := r.UserAgent(); ua != "" {
		attrs = append(attrs, attribute.String("user_agent.original", ua))
	}
	return attrs
}
