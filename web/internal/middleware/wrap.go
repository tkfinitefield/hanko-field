package middleware

import "net/http"

// ResponseRecorder wraps ResponseWriter and captures the status code.
type ResponseRecorder struct {
	http.ResponseWriter
	status int
}

func NewResponseRecorder(w http.ResponseWriter) *ResponseRecorder {
	return &ResponseRecorder{ResponseWriter: w, status: http.StatusOK}
}

func (rw *ResponseRecorder) WriteHeader(statusCode int) {
	rw.status = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *ResponseRecorder) Status() int { return rw.status }
