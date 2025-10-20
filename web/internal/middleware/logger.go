package middleware

import (
	"encoding/json"
	chiMid "github.com/go-chi/chi/v5/middleware"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type logEntry struct {
	Timestamp  string `json:"ts"`
	Level      string `json:"level"`
	Message    string `json:"msg"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	Status     int    `json:"status"`
	DurationMs int64  `json:"duration_ms"`
	RemoteIP   string `json:"remote_ip,omitempty"`
	RequestID  string `json:"request_id,omitempty"`
	UserID     string `json:"user_id,omitempty"`
	HTMX       bool   `json:"htmx"`
}

var useStdoutLogger = true

// Logger emits a structured JSON log per request
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// wrap writer to capture status
		rw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)
		// build entry
		rid := chiMid.GetReqID(r.Context())
		if rid != "" {
			r = r.WithContext(WithRequestID(r.Context(), rid))
		}
		var uid string
		if u := UserFromContext(r.Context()); u != nil {
			uid = u.ID
		}
		e := logEntry{
			Timestamp:  time.Now().Format(time.RFC3339Nano),
			Level:      "info",
			Message:    "request",
			Method:     r.Method,
			Path:       r.URL.Path,
			Status:     rw.status,
			DurationMs: time.Since(start).Milliseconds(),
			RemoteIP:   clientIP(r),
			RequestID:  rid,
			UserID:     uid,
			HTMX:       IsHTMX(r.Context()),
		}
		b, _ := json.Marshal(e)
		if useStdoutLogger {
			log.Println(string(b))
		} else {
			_, _ = os.Stdout.Write(append(b, '\n'))
		}
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func clientIP(r *http.Request) string {
	// Trust X-Forwarded-For set by Cloud Run (last IP is client)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		p := strings.Split(xff, ",")
		return strings.TrimSpace(p[len(p)-1])
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i != -1 {
		return host[:i]
	}
	return host
}
