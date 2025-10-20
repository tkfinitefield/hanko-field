package middleware

import (
    "net/http"
    "sync"
)

// ResponseRecorder wraps ResponseWriter and captures the status code.
// It supports a beforeWrite hook executed once just before the first write.
type ResponseRecorder struct {
    http.ResponseWriter
    status int
    wrote  bool
    mu     sync.Mutex
    once   sync.Once
    before func(http.ResponseWriter)
}

func NewResponseRecorder(w http.ResponseWriter) *ResponseRecorder {
    return &ResponseRecorder{ResponseWriter: w, status: http.StatusOK}
}

func (rw *ResponseRecorder) onWrite() {
    rw.once.Do(func() {
        if rw.before != nil {
            rw.before(rw.ResponseWriter)
        }
    })
}

func (rw *ResponseRecorder) WriteHeader(statusCode int) {
    rw.mu.Lock()
    defer rw.mu.Unlock()
    if rw.wrote {
        return
    }
    rw.onWrite()
    rw.status = statusCode
    rw.wrote = true
    rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *ResponseRecorder) Write(p []byte) (int, error) {
    rw.mu.Lock()
    defer rw.mu.Unlock()
    if !rw.wrote {
        rw.onWrite()
        rw.status = http.StatusOK
        rw.wrote = true
    }
    return rw.ResponseWriter.Write(p)
}

func (rw *ResponseRecorder) Status() int { return rw.status }

// SetBeforeWrite sets a hook executed once just before headers/body are first written.
func (rw *ResponseRecorder) SetBeforeWrite(f func(http.ResponseWriter)) { rw.before = f }
