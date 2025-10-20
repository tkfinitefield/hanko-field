package middleware

import (
	"encoding/json"
	"net/http"
)

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, r *http.Request, code int, msg string) {
	if IsHTMX(r.Context()) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(errorResponse{Error: msg})
		return
	}
	http.Error(w, msg, code)
}
