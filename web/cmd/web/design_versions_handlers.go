package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	handlersPkg "finitefield.org/hanko-web/internal/handlers"
	mw "finitefield.org/hanko-web/internal/middleware"
	"finitefield.org/hanko-web/internal/nav"
)

// DesignVersionsHandler renders the design version history page.
func DesignVersionsHandler(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	view := buildDesignVersionHistoryView(lang, r.URL.Query())

	title := i18nOrDefault(lang, "design.versions.title", "Version history")
	desc := i18nOrDefault(lang, "design.versions.description", "Inspect every saved version, compare diffs, and safely roll back the design editor.")

	vm := handlersPkg.PageData{
		Title:          title,
		Lang:           lang,
		Path:           r.URL.Path,
		Nav:            nav.Build(r.URL.Path),
		Breadcrumbs:    nav.Breadcrumbs(r.URL.Path),
		Analytics:      handlersPkg.LoadAnalyticsFromEnv(),
		DesignVersions: view,
	}

	brand := i18nOrDefault(lang, "brand.name", "Hanko Field")
	vm.SEO.Title = fmt.Sprintf("%s | %s", title, brand)
	vm.SEO.Description = desc
	vm.SEO.Canonical = absoluteURL(r)
	vm.SEO.OG.URL = vm.SEO.Canonical
	vm.SEO.OG.SiteName = brand
	vm.SEO.OG.Type = "website"
	vm.SEO.OG.Title = vm.SEO.Title
	vm.SEO.OG.Description = vm.SEO.Description
	vm.SEO.Twitter.Card = "summary_large_image"
	vm.SEO.Alternates = buildAlternates(r)

	renderPage(w, r, "design_versions", vm)
}

// DesignVersionsTableFrag renders the history table fragment (filters + rows).
func DesignVersionsTableFrag(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	view := buildDesignVersionHistoryView(lang, r.URL.Query())
	push := "/design/versions"
	if view.Query != "" {
		push = push + "?" + view.Query
	}
	w.Header().Set("HX-Push-Url", push)
	renderTemplate(w, r, "frag_design_versions_table", view)
}

// DesignVersionsPreviewFrag renders the split preview + insights for a selected version.
func DesignVersionsPreviewFrag(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	versionID := strings.TrimSpace(r.URL.Query().Get("version"))
	if versionID == "" {
		http.Error(w, "missing version id", http.StatusBadRequest)
		return
	}
	q := cloneQuery(r.URL.Query())
	q.Set("focus", versionID)
	view := buildDesignVersionHistoryView(lang, q)
	if view.Selected.ID == "" {
		http.NotFound(w, r)
		return
	}
	data := map[string]any{
		"Lang":     lang,
		"Detail":   view.Selected,
		"Timeline": view.Timeline,
		"Query":    view.Query,
	}
	renderTemplate(w, r, "frag_design_versions_preview", data)
}

// DesignVersionRollbackModal renders a confirmation modal for rollback.
func DesignVersionRollbackModal(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	versionID := strings.TrimSpace(chi.URLParam(r, "versionID"))
	if versionID == "" {
		http.Error(w, "missing version id", http.StatusBadRequest)
		return
	}
	q := cloneQuery(r.URL.Query())
	q.Set("focus", versionID)
	view := buildDesignVersionHistoryView(lang, q)
	if view.Selected.ID == "" {
		http.NotFound(w, r)
		return
	}
	formAuthor := view.ActiveAuthor
	if formAuthor == "" {
		formAuthor = "all"
	}
	formRange := view.ActiveRange
	if formRange == "" {
		formRange = "all"
	}
	data := map[string]any{
		"Lang":       lang,
		"Detail":     view.Selected,
		"Query":      view.Query,
		"VersionID":  versionID,
		"FormAuthor": formAuthor,
		"FormRange":  formRange,
	}
	renderTemplate(w, r, "frag_design_versions_rollback_modal", data)
}

// DesignVersionRollbackHandler handles rollback POST requests.
func DesignVersionRollbackHandler(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	versionID := strings.TrimSpace(chi.URLParam(r, "versionID"))
	if versionID == "" {
		http.Error(w, "missing version id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	q := cloneQuery(r.URL.Query())
	for k, vv := range r.Form {
		for _, v := range vv {
			q.Set(k, v)
		}
	}
	if q.Get("version") == "" {
		q.Set("version", versionID)
	}
	q.Set("focus", versionID)
	view := buildDesignVersionHistoryView(lang, q)
	if view.Selected.ID == "" {
		http.NotFound(w, r)
		return
	}

	payload := map[string]any{
		"design-versions:rolled-back": map[string]string{
			"id":      versionID,
			"version": view.Selected.VersionLabel,
			"query":   view.Query,
		},
		"design-versions:refresh-table": map[string]string{
			"query": view.Query,
		},
	}
	if raw, err := json.Marshal(payload); err == nil {
		w.Header().Set("HX-Trigger", string(raw))
	}

	data := map[string]any{
		"Lang":     lang,
		"Detail":   view.Selected,
		"Timeline": view.Timeline,
		"Query":    view.Query,
	}
	renderTemplate(w, r, "frag_design_versions_preview", data)
}

func cloneQuery(values url.Values) url.Values {
	cp := url.Values{}
	for k, vv := range values {
		for _, v := range vv {
			cp.Add(k, v)
		}
	}
	return cp
}
