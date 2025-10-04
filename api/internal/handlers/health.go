package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

var startTime = time.Now()

// health responds with a simple status payload for monitoring and readiness checks.
func health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	payload := map[string]any{
		"status":    "ok",
		"uptime":    time.Since(startTime).String(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
