package ui

import (
	"log"
	"net/http"

	"github.com/a-h/templ"

	custommw "finitefield.org/hanko-admin/internal/admin/httpserver/middleware"
	"finitefield.org/hanko-admin/internal/admin/profile"
	"finitefield.org/hanko-admin/internal/admin/templates/dashboard"
	profiletpl "finitefield.org/hanko-admin/internal/admin/templates/profile"
)

// Dependencies collects external services required by the UI handlers.
type Dependencies struct {
	ProfileService profile.Service
}

// Handlers exposes HTTP handlers for admin UI pages and fragments.
type Handlers struct {
	profile profile.Service
}

// NewHandlers wires the UI handler set.
func NewHandlers(deps Dependencies) *Handlers {
	service := deps.ProfileService
	if service == nil {
		service = profile.NewStaticService(nil)
	}
	return &Handlers{
		profile: service,
	}
}

// Dashboard renders the admin dashboard.
func (h *Handlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	component := dashboard.Index()
	templ.Handler(component).ServeHTTP(w, r)
}

func (h *Handlers) renderProfilePage(w http.ResponseWriter, r *http.Request) {
	user, ok := custommw.UserFromContext(r.Context())
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	state, err := h.profile.SecurityOverview(r.Context(), user.Token)
	if err != nil {
		log.Printf("profile: fetch security overview failed: %v", err)
		http.Error(w, "セキュリティ情報の取得に失敗しました。時間を置いて再度お試しください。", http.StatusBadGateway)
		return
	}

	payload := profiletpl.PageData{
		UserEmail: user.Email,
		UserName:  user.UID,
		Security:  state,
		CSRFToken: custommw.CSRFTokenFromContext(r.Context()),
	}

	component := profiletpl.Index(payload)
	templ.Handler(component).ServeHTTP(w, r)
}
