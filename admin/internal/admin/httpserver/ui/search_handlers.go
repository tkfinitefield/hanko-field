package ui

import (
	"log"
	"net/http"
	"net/url"
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

	if canonical := canonicalSearchURL(custommw.BasePathFromContext(ctx), params); canonical != "" {
		w.Header().Set("HX-Push-Url", canonical)
	}

	templ.Handler(component).ServeHTTP(w, r)
}

type searchRequest struct {
	query         adminsearch.Query
	state         searchtpl.QueryState
	limitExplicit bool
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

	const defaultLimit = 20
	limit := defaultLimit
	limitExplicit := false
	if rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 {
			limit = parsed
			if parsed != defaultLimit {
				limitExplicit = true
			}
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
		query:         query,
		state:         state,
		limitExplicit: limitExplicit || rawLimit != "",
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

func canonicalSearchURL(basePath string, req searchRequest) string {
	base := strings.TrimSpace(basePath)
	if base == "" || base == "/" {
		base = ""
	} else {
		if !strings.HasPrefix(base, "/") {
			base = "/" + base
		}
		base = strings.TrimRight(base, "/")
	}

	path := base + "/search"
	values := url.Values{}

	if term := strings.TrimSpace(req.state.Term); term != "" {
		values.Set("q", term)
	}
	if scope := strings.TrimSpace(req.state.Scope); scope != "" && scope != "all" {
		values.Set("scope", scope)
	}
	if start := strings.TrimSpace(req.state.StartDate); start != "" {
		values.Set("start", start)
	}
	if end := strings.TrimSpace(req.state.EndDate); end != "" {
		values.Set("end", end)
	}
	if persona := strings.TrimSpace(req.state.Persona); persona != "" {
		values.Set("persona", persona)
	}
	if req.limitExplicit && req.query.Limit > 0 {
		values.Set("limit", strconv.Itoa(req.query.Limit))
	}

	if encoded := values.Encode(); encoded != "" {
		return path + "?" + encoded
	}

	return path
}
