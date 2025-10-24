package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	handlersPkg "finitefield.org/hanko-web/internal/handlers"
	mw "finitefield.org/hanko-web/internal/middleware"
	"finitefield.org/hanko-web/internal/nav"
)

// DesignAISuggestionsHandler renders the AI suggestions gallery page.
func DesignAISuggestionsHandler(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	vm := handlersPkg.PageData{
		Title: i18nOrDefault(lang, "design.ai.title", "AI suggestions"),
		Lang:  lang,
		Path:  r.URL.Path,
		Nav:   nav.Build(r.URL.Path),
	}
	vm.Breadcrumbs = nav.Breadcrumbs(r.URL.Path)
	vm.Analytics = handlersPkg.LoadAnalyticsFromEnv()
	view := buildDesignAISuggestionsView(lang, r.URL.Query())
	vm.DesignAI = view

	brand := i18nOrDefault(lang, "brand.name", "Hanko Field")
	vm.SEO.Title = i18nOrDefault(lang, "design.ai.seo.title", "AI suggestions gallery") + " | " + brand
	vm.SEO.Description = i18nOrDefault(lang, "design.ai.seo.description", "Review, compare, and accept AI-generated seal layouts with live diff highlights.")
	vm.SEO.Canonical = absoluteURL(r)
	vm.SEO.OG.URL = vm.SEO.Canonical
	vm.SEO.OG.SiteName = brand
	vm.SEO.OG.Type = "website"
	vm.SEO.OG.Title = vm.SEO.Title
	vm.SEO.OG.Description = vm.SEO.Description
	vm.SEO.Twitter.Card = "summary_large_image"
	vm.SEO.Alternates = buildAlternates(r)

	renderPage(w, r, "design_ai", vm)
}

// DesignAISuggestionTableFrag renders the suggestion table fragment.
func DesignAISuggestionTableFrag(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	view := buildDesignAISuggestionsView(lang, r.URL.Query())
	push := "/design/ai"
	if view.Query != "" {
		push = push + "?" + view.Query
	}
	w.Header().Set("HX-Push-Url", push)
	renderTemplate(w, r, "frag_design_ai_table", view)
}

// DesignAISuggestionPreviewFrag renders the preview drawer fragment for a given suggestion.
func DesignAISuggestionPreviewFrag(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	suggestion, ok := designAISuggestionByID(designAIMockData(lang), id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	detail := buildDesignAISuggestionDetail(lang, suggestion, suggestion.Status)
	renderTemplate(w, r, "frag_design_ai_preview", detail)
}

// DesignAISuggestionAcceptHandler handles accepting an AI suggestion.
func DesignAISuggestionAcceptHandler(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	id := strings.TrimSpace(chi.URLParam(r, "suggestionID"))
	if id == "" {
		http.Error(w, "missing suggestion id", http.StatusBadRequest)
		return
	}
	suggestion, ok := designAISuggestionByID(designAIMockData(lang), id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	detail := buildDesignAISuggestionDetail(lang, suggestion, "accepted")
	detail.ActionsDisabled = true
	detail.Notes = append([]string{i18nOrDefault(lang, "design.ai.accept.note", "Applied to design editor preview immediately.")}, detail.Notes...)

	payload := map[string]any{
		"design-ai:suggestion-accepted": map[string]string{
			"id":    id,
			"label": detail.StatusLabel,
			"tone":  detail.StatusTone,
		},
	}
	if raw, err := json.Marshal(payload); err == nil {
		w.Header().Set("HX-Trigger", string(raw))
	}
	renderTemplate(w, r, "frag_design_ai_preview", detail)
}

// DesignAISuggestionRejectHandler handles rejecting an AI suggestion.
func DesignAISuggestionRejectHandler(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	id := strings.TrimSpace(chi.URLParam(r, "suggestionID"))
	if id == "" {
		http.Error(w, "missing suggestion id", http.StatusBadRequest)
		return
	}
	suggestion, ok := designAISuggestionByID(designAIMockData(lang), id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	detail := buildDesignAISuggestionDetail(lang, suggestion, "rejected")
	detail.ActionsDisabled = true
	detail.Notes = append([]string{i18nOrDefault(lang, "design.ai.reject.note", "Suggestion archived and removed from active queue.")}, detail.Notes...)

	payload := map[string]any{
		"design-ai:suggestion-rejected": map[string]string{
			"id":    id,
			"label": detail.StatusLabel,
			"tone":  detail.StatusTone,
		},
	}
	if raw, err := json.Marshal(payload); err == nil {
		w.Header().Set("HX-Trigger", string(raw))
	}
	renderTemplate(w, r, "frag_design_ai_preview", detail)
}
