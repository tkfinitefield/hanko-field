package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	mw "finitefield.org/hanko-web/internal/middleware"
)

// DesignShareModal renders the design share modal with default selections.
func DesignShareModal(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	now := time.Now()
	form := defaultDesignShareForm(now)
	if design := strings.TrimSpace(r.URL.Query().Get("design")); design != "" {
		if normalized, ok := normalizeDesignID(design); ok {
			form.DesignID = normalized
		}
	}
	view := buildDesignShareView(lang, form, nil, nil, now)
	renderTemplate(w, r, "frag_design_share_modal", view)
}

// DesignShareLinkHandler handles link creation/regeneration requests from the modal form.
func DesignShareLinkHandler(w http.ResponseWriter, r *http.Request) {
	lang := mw.Lang(r)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form submission", http.StatusBadRequest)
		return
	}
	now := time.Now()
	state, alerts := parseDesignShareForm(lang, r.PostForm, now)

	var link *DesignShareLink
	if len(alerts) == 0 {
		issued, err := issueDesignShareLink(r.Context(), lang, state.DesignID, state, now)
		if err != nil {
			alerts = append(alerts, DesignShareAlert{
				Tone:        "danger",
				Title:       editorCopy(lang, "共有リンクの発行に失敗しました。", "Unable to issue share link."),
				Description: err.Error(),
				Icon:        "exclamation-circle",
			})
		} else {
			link = &issued
			trigger := map[string]any{
				"design-share:link-issued": map[string]any{
					"format":    issued.Format,
					"size":      issued.Size,
					"watermark": issued.Watermark,
				},
			}
			if raw, err := json.Marshal(trigger); err == nil {
				w.Header().Set("HX-Trigger", string(raw))
			}
		}
	}

	view := buildDesignShareView(lang, state, link, alerts, now)
	renderTemplate(w, r, "frag_design_share_modal", view)
}
