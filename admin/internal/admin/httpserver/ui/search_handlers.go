package ui

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"

	custommw "finitefield.org/hanko-admin/internal/admin/httpserver/middleware"
	adminsearch "finitefield.org/hanko-admin/internal/admin/search"
	searchtpl "finitefield.org/hanko-admin/internal/admin/templates/search"
)

// SearchPage renders the global search page with the initial results.
func (h *Handlers) SearchPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	params := buildSearchRequest(r)
	result, err := h.search.Search(ctx, user.Token, params.query)
	errMsg := ""
	if err != nil {
		log.Printf("search: query failed: %v", err)
		errMsg = "検索に失敗しました。時間を置いて再度お試しください。"
		result = adminsearch.ResultSet{}
	}

	table := searchtpl.TablePayload(params.state, result, errMsg)
	payload := searchtpl.BuildPageData(custommw.BasePathFromContext(ctx), params.state, table)

	templ.Handler(searchtpl.Index(payload)).ServeHTTP(w, r)
}

// SearchTable renders the result table fragment for htmx requests.
func (h *Handlers) SearchTable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	params := buildSearchRequest(r)
	result, err := h.search.Search(ctx, user.Token, params.query)
	errMsg := ""
	if err != nil {
		log.Printf("search: query failed: %v", err)
		errMsg = "検索に失敗しました。時間を置いて再度お試しください。"
		result = adminsearch.ResultSet{}
	}

	table := searchtpl.TablePayload(params.state, result, errMsg)
	component := searchtpl.Table(table)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
	}
	templ.Handler(component).ServeHTTP(w, r)
}

type searchRequest struct {
	query adminsearch.Query
	state searchtpl.QueryState
}

func buildSearchRequest(r *http.Request) searchRequest {
	values := r.URL.Query()
	rawTerm := strings.TrimSpace(values.Get("q"))
	rawScope := strings.TrimSpace(values.Get("scope"))
	rawPersona := strings.TrimSpace(values.Get("persona"))
	rawStart := strings.TrimSpace(values.Get("start"))
	rawEnd := strings.TrimSpace(values.Get("end"))
	rawLimit := strings.TrimSpace(values.Get("limit"))

	scopeValue := normaliseScope(rawScope)
	scope := toScopeEntities(scopeValue)

	var startPtr *time.Time
	if t := parseDate(rawStart); !t.IsZero() {
		startPtr = &t
	}
	var endPtr *time.Time
	if t := parseDate(rawEnd); !t.IsZero() {
		// Ensure end >= start if both provided.
		if startPtr != nil && t.Before(*startPtr) {
			adjusted := startPtr.Add(24 * time.Hour)
			endPtr = &adjusted
		} else {
			endPtr = &t
		}
	}

	limit := 20
	if rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	state := searchtpl.QueryState{
		Term:      rawTerm,
		Scope:     scopeValue,
		StartDate: normalizeDateInput(rawStart),
		EndDate:   normalizeDateInput(rawEnd),
		Persona:   rawPersona,
	}

	query := adminsearch.Query{
		Term:    rawTerm,
		Scope:   scope,
		Persona: rawPersona,
		Start:   startPtr,
		End:     endPtr,
		Limit:   limit,
	}

	return searchRequest{
		query: query,
		state: state,
	}
}

func normaliseScope(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "all", "すべて", "総合":
		return "all"
	case "orders", "order", "注文":
		return "orders"
	case "users", "user", "顧客":
		return "users"
	case "reviews", "review", "レビュー":
		return "reviews"
	default:
		return "all"
	}
}

func toScopeEntities(scope string) []adminsearch.Entity {
	switch scope {
	case "orders":
		return []adminsearch.Entity{adminsearch.EntityOrder}
	case "users":
		return []adminsearch.Entity{adminsearch.EntityUser}
	case "reviews":
		return []adminsearch.Entity{adminsearch.EntityReview}
	default:
		return nil
	}
}

func parseDate(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02",
		"2006/01/02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t
		}
	}
	return time.Time{}
}

func normalizeDateInput(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if t, err := time.Parse("2006-01-02", value); err == nil {
		return t.Format("2006-01-02")
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t.Format("2006-01-02")
	}
	return ""
}
