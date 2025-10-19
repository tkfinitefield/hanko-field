package ui

import (
	"net/http"

	"github.com/a-h/templ"

	"finitefield.org/hanko-admin/internal/admin/templates/dashboard"
)

func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	component := dashboard.Index()
	templ.Handler(component).ServeHTTP(w, r)
}
