package handlers

import (
	"finitefield.org/hanko-web/internal/nav"
)

// PageData is a generic view model for simple pages using the shared layout.
type PageData struct {
	Title     string
	Lang      string
	SEO       SEOData
	Analytics Analytics

	Path        string
	Nav         []nav.RenderedItem
	Breadcrumbs []nav.Crumb

	// Optional per-page view model payloads
	Shop      any
	Product   any
	Templates any
	Template  any
	Guides    any
	Guide     any
	Content   any
	Status    any
	Design    any
	DesignAI  any
}
