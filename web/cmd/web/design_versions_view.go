package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"finitefield.org/hanko-web/internal/format"
)

// designVersionsNow anchors relative time descriptions for deterministic output.
var designVersionsNow = time.Date(2025, time.March, 24, 10, 30, 0, 0, time.UTC)

// DesignVersionHistoryView contains all data required to render the versions page and fragments.
type DesignVersionHistoryView struct {
	Lang          string
	DesignID      string
	ActiveAuthor  string
	ActiveRange   string
	Query         string
	TotalVersions int

	AuthorFilters []DesignVersionFilterOption
	RangeFilters  []DesignVersionFilterOption

	Versions []DesignVersionRow
	Empty    bool

	Selected DesignVersionDetail
	Timeline []DesignVersionTimelineEntry
}

// DesignVersionFilterOption describes a filter chip entry.
type DesignVersionFilterOption struct {
	ID     string
	Label  string
	Count  int
	Active bool
}

// DesignVersionRow represents one entry in the history table.
type DesignVersionRow struct {
	ID             string
	VersionLabel   string
	StatusBadge    string
	StatusTone     string
	Tag            string
	TagTone        string
	CreatedDisplay string
	CreatedAgo     string
	AuthorName     string
	AuthorRole     string
	Note           string
	DiffSummary    []DesignVersionDiffChip
	CompareURL     string
	RollbackURL    string
	Active         bool
}

// DesignVersionDiffChip summarises a change scope for quick scanning.
type DesignVersionDiffChip struct {
	Kind  string
	Label string
}

// DesignVersionDetail powers the preview + toolbar payload for the selected version.
type DesignVersionDetail struct {
	ID            string
	VersionLabel  string
	Subtitle      string
	AuthorName    string
	AuthorRole    string
	CreatedAgo    string
	CreatedExact  string
	Note          string
	StatusBadge   string
	StatusTone    string
	RollbackURL   string
	DuplicateURL  string
	DeleteURL     string
	CompareURL    string
	Preview       DesignVersionPreview
	Insights      []DesignVersionInsight
	ChangelogTags []DesignVersionDiffChip
}

// DesignVersionPreview feeds the split preview component.
type DesignVersionPreview struct {
	Before           DesignVersionPreviewPane
	After            DesignVersionPreviewPane
	LastComparedText string
	DiffSummary      []DesignVersionDiffChip
	Hints            []string
}

// DesignVersionPreviewPane describes one side of the split preview.
type DesignVersionPreviewPane struct {
	Title       string
	Description string
	Image       string
	Badge       string
	Meta        []string
}

// DesignVersionInsight highlights metrics for the selected version.
type DesignVersionInsight struct {
	Label string
	Value string
	Tone  string
	Icon  string
}

// DesignVersionTimelineEntry powers the audit drawer timeline UI.
type DesignVersionTimelineEntry struct {
	ID          string
	Title       string
	Description string
	Actor       string
	ActorRole   string
	DisplayTime string
	Relative    string
	Icon        string
	Tone        string
}

// designVersion is an internal DTO for assembling the view model.
type designVersion struct {
	ID              string
	VersionLabel    string
	StatusBadge     string
	StatusTone      string
	Tag             string
	TagTone         string
	AuthorID        string
	AuthorName      string
	AuthorRole      string
	CreatedAt       time.Time
	Note            string
	DiffSummary     []DesignVersionDiffChip
	CompareURL      string
	RollbackURL     string
	DuplicateURL    string
	DeleteURL       string
	PreviewBefore   DesignVersionPreviewPane
	PreviewAfter    DesignVersionPreviewPane
	Insights        []DesignVersionInsight
	ChangelogTags   []DesignVersionDiffChip
	TimelineEntries []DesignVersionTimelineEntry
}

// buildDesignVersionHistoryView assembles the versions view model with filtering.
func buildDesignVersionHistoryView(lang string, q url.Values) DesignVersionHistoryView {
	author := normalizeDesignVersionAuthor(strings.TrimSpace(q.Get("author")))
	dateRange := normalizeDesignVersionRange(strings.TrimSpace(q.Get("range")))
	focus := strings.TrimSpace(q.Get("focus"))

	all := designVersionMockData(lang)
	total := len(all)

	authorCounts := map[string]int{"all": total}
	for _, v := range all {
		authorCounts[v.AuthorID]++
	}

	rangeCounts := map[string]int{
		"all": total,
		"7d":  0,
		"30d": 0,
	}
	for _, v := range all {
		if designVersionsNow.Sub(v.CreatedAt) <= 7*24*time.Hour {
			rangeCounts["7d"]++
		}
		if designVersionsNow.Sub(v.CreatedAt) <= 30*24*time.Hour {
			rangeCounts["30d"]++
		}
	}

	filtered := make([]designVersion, 0, len(all))
	for _, v := range all {
		if author != "" && author != "all" && v.AuthorID != author {
			continue
		}
		switch dateRange {
		case "", "all":
			// no-op
		case "7d":
			if designVersionsNow.Sub(v.CreatedAt) > 7*24*time.Hour {
				continue
			}
		case "30d":
			if designVersionsNow.Sub(v.CreatedAt) > 30*24*time.Hour {
				continue
			}
		}
		filtered = append(filtered, v)
	}

	query := url.Values{}
	if author != "" && author != "all" {
		query.Set("author", author)
	}
	if dateRange != "" && dateRange != "all" {
		query.Set("range", dateRange)
	}
	queryStr := query.Encode()

	view := DesignVersionHistoryView{
		Lang:          lang,
		DesignID:      "df-219a",
		ActiveAuthor:  author,
		ActiveRange:   dateRange,
		Query:         queryStr,
		TotalVersions: len(filtered),
	}

	view.AuthorFilters = buildDesignVersionAuthorFilters(lang, authorCounts, author)
	view.RangeFilters = buildDesignVersionRangeFilters(lang, rangeCounts, dateRange)

	view.Empty = len(filtered) == 0
	view.Versions = buildDesignVersionRows(lang, filtered, focus)

	if selected, ok := findSelectedDesignVersion(filtered, all, focus); ok {
		view.Selected = buildDesignVersionDetail(lang, selected, all)
		view.Timeline = buildDesignVersionTimeline(selected)
	} else {
		view.Selected = DesignVersionDetail{}
		view.Timeline = nil
	}

	return view
}

func buildDesignVersionAuthorFilters(lang string, counts map[string]int, active string) []DesignVersionFilterOption {
	options := []DesignVersionFilterOption{
		{
			ID:     "all",
			Label:  dictAuthorLabel(lang, "all"),
			Count:  counts["all"],
			Active: active == "" || active == "all",
		},
	}
	order := []string{"mn-labs", "hk-sakura", "ops-auto"}
	for _, id := range order {
		count := counts[id]
		if count == 0 {
			continue
		}
		options = append(options, DesignVersionFilterOption{
			ID:     id,
			Label:  dictAuthorLabel(lang, id),
			Count:  count,
			Active: active == id,
		})
	}
	return options
}

func buildDesignVersionRangeFilters(lang string, counts map[string]int, active string) []DesignVersionFilterOption {
	options := []DesignVersionFilterOption{
		{
			ID:     "all",
			Label:  dictRangeLabel(lang, "all"),
			Count:  counts["all"],
			Active: active == "" || active == "all",
		},
		{
			ID:     "7d",
			Label:  dictRangeLabel(lang, "7d"),
			Count:  counts["7d"],
			Active: active == "7d",
		},
		{
			ID:     "30d",
			Label:  dictRangeLabel(lang, "30d"),
			Count:  counts["30d"],
			Active: active == "30d",
		},
	}
	return options
}

func buildDesignVersionRows(lang string, versions []designVersion, focus string) []DesignVersionRow {
	rows := make([]DesignVersionRow, 0, len(versions))
	for i, v := range versions {
		active := false
		if focus != "" {
			active = v.ID == focus
		} else if i == 0 {
			active = true
		}
		rows = append(rows, DesignVersionRow{
			ID:             v.ID,
			VersionLabel:   v.VersionLabel,
			StatusBadge:    v.StatusBadge,
			StatusTone:     v.StatusTone,
			Tag:            v.Tag,
			TagTone:        v.TagTone,
			CreatedDisplay: fmt.Sprintf("%s · %s", format.FmtDate(v.CreatedAt, lang), v.CreatedAt.Format("15:04")),
			CreatedAgo:     formatDesignVersionRelative(lang, designVersionsNow.Sub(v.CreatedAt)),
			AuthorName:     v.AuthorName,
			AuthorRole:     v.AuthorRole,
			Note:           v.Note,
			DiffSummary:    append([]DesignVersionDiffChip{}, v.DiffSummary...),
			CompareURL:     v.CompareURL,
			RollbackURL:    v.RollbackURL,
			Active:         active,
		})
	}
	return rows
}

func buildDesignVersionDetail(lang string, selected designVersion, all []designVersion) DesignVersionDetail {
	if selected.ID == "" {
		return DesignVersionDetail{}
	}
	before := selected
	for i, v := range all {
		if v.ID == selected.ID {
			if i+1 < len(all) {
				before = all[i+1]
			}
			break
		}
	}
	subtitle := fmt.Sprintf("%s • %s", selected.AuthorName, formatDesignVersionRelative(lang, designVersionsNow.Sub(selected.CreatedAt)))
	if lang == "ja" {
		subtitle = fmt.Sprintf("%s・%s", selected.AuthorName, formatDesignVersionRelative(lang, designVersionsNow.Sub(selected.CreatedAt)))
	}
	return DesignVersionDetail{
		ID:            selected.ID,
		VersionLabel:  selected.VersionLabel,
		Subtitle:      subtitle,
		AuthorName:    selected.AuthorName,
		AuthorRole:    selected.AuthorRole,
		CreatedAgo:    formatDesignVersionRelative(lang, designVersionsNow.Sub(selected.CreatedAt)),
		CreatedExact:  fmt.Sprintf("%s • %s", format.FmtDate(selected.CreatedAt, lang), selected.CreatedAt.Format("15:04")),
		Note:          selected.Note,
		StatusBadge:   selected.StatusBadge,
		StatusTone:    selected.StatusTone,
		RollbackURL:   selected.RollbackURL,
		DuplicateURL:  selected.DuplicateURL,
		DeleteURL:     selected.DeleteURL,
		CompareURL:    selected.CompareURL,
		Preview:       buildDesignVersionPreview(selected, before, lang),
		Insights:      append([]DesignVersionInsight{}, selected.Insights...),
		ChangelogTags: append([]DesignVersionDiffChip{}, selected.ChangelogTags...),
	}
}

func buildDesignVersionPreview(selected, before designVersion, lang string) DesignVersionPreview {
	lastCompared := fmt.Sprintf("Compared against %s", before.VersionLabel)
	if lang == "ja" {
		lastCompared = fmt.Sprintf("%s と比較済み", before.VersionLabel)
	}
	return DesignVersionPreview{
		Before:           before.PreviewAfter,
		After:            selected.PreviewAfter,
		LastComparedText: lastCompared,
		DiffSummary:      append([]DesignVersionDiffChip{}, selected.DiffSummary...),
		Hints: []string{
			i18nOrDefault(lang, "design.versions.preview.hint", "Switch canvases to inspect before/after overlays."),
		},
	}
}

func buildDesignVersionTimeline(selected designVersion) []DesignVersionTimelineEntry {
	if len(selected.TimelineEntries) == 0 {
		return nil
	}
	out := make([]DesignVersionTimelineEntry, len(selected.TimelineEntries))
	copy(out, selected.TimelineEntries)
	return out
}

func findSelectedDesignVersion(filtered, all []designVersion, focus string) (designVersion, bool) {
	if focus != "" {
		for _, v := range filtered {
			if v.ID == focus {
				return v, true
			}
		}
		for _, v := range all {
			if v.ID == focus {
				return v, true
			}
		}
	}
	if focus == "" && len(filtered) > 0 {
		return filtered[0], true
	}
	return designVersion{}, false
}

func normalizeDesignVersionAuthor(author string) string {
	if author == "" {
		return ""
	}
	switch author {
	case "all", "mn-labs", "hk-sakura", "ops-auto":
		return author
	default:
		return ""
	}
}

func normalizeDesignVersionRange(r string) string {
	switch r {
	case "", "all", "7d", "30d":
		return r
	default:
		return ""
	}
}

func dictAuthorLabel(lang, id string) string {
	switch id {
	case "all":
		if lang == "ja" {
			return "すべての担当者"
		}
		return "All authors"
	case "mn-labs":
		if lang == "ja" {
			return "皆川 (デザインラボ)"
		}
		return "Mina (Design Lab)"
	case "hk-sakura":
		if lang == "ja" {
			return "光希 (桜工房)"
		}
		return "Koki (Sakura Studio)"
	case "ops-auto":
		if lang == "ja" {
			return "自動保存"
		}
		return "Autosave agent"
	default:
		return id
	}
}

func dictRangeLabel(lang, id string) string {
	switch id {
	case "all":
		if lang == "ja" {
			return "すべて"
		}
		return "All history"
	case "7d":
		if lang == "ja" {
			return "直近7日"
		}
		return "Last 7 days"
	case "30d":
		if lang == "ja" {
			return "直近30日"
		}
		return "Last 30 days"
	default:
		return id
	}
}

func formatDesignVersionRelative(lang string, d time.Duration) string {
	if d < time.Minute {
		if lang == "ja" {
			return "たった今"
		}
		return "just now"
	}
	if d < time.Hour {
		n := int(d / time.Minute)
		if lang == "ja" {
			return fmt.Sprintf("%d分前", n)
		}
		return fmt.Sprintf("%d min ago", n)
	}
	if d < 24*time.Hour {
		n := int(d / time.Hour)
		if lang == "ja" {
			return fmt.Sprintf("%d時間前", n)
		}
		return fmt.Sprintf("%d h ago", n)
	}
	days := int(d / (24 * time.Hour))
	if days < 30 {
		if lang == "ja" {
			return fmt.Sprintf("%d日前", days)
		}
		return fmt.Sprintf("%d days ago", days)
	}
	months := days / 30
	if months < 12 {
		if lang == "ja" {
			return fmt.Sprintf("%dか月前", months)
		}
		return fmt.Sprintf("%d mo ago", months)
	}
	years := months / 12
	if lang == "ja" {
		return fmt.Sprintf("%d年前", years)
	}
	return fmt.Sprintf("%d yr ago", years)
}

func designVersionMockData(lang string) []designVersion {
	basePreview := DesignVersionPreviewPane{
		Title:       i18nOrDefault(lang, "design.versions.preview.title", "Final mockup"),
		Description: i18nOrDefault(lang, "design.versions.preview.caption", "Vector output prepared for export."),
		Image:       "https://cdn.hanko-field.app/designs/df-219a/current.svg",
		Badge:       "",
		Meta:        []string{"1,200 DPI", i18nOrDefault(lang, "design.versions.preview.meta", "Desk mockup")},
	}

	return []designVersion{
		{
			ID:           "ver-210",
			VersionLabel: "v1.12.0",
			StatusBadge:  i18nOrDefault(lang, "design.versions.status.current", "Current"),
			StatusTone:   "success",
			Tag:          i18nOrDefault(lang, "design.versions.tag.production", "In production"),
			TagTone:      "emerald",
			AuthorID:     "mn-labs",
			AuthorName:   "Mina Nagata",
			AuthorRole:   i18nOrDefault(lang, "design.versions.role.art-director", "Art Director"),
			CreatedAt:    designVersionsNow.Add(-2 * time.Hour),
			Note:         i18nOrDefault(lang, "design.versions.note.adjust-letter", "Adjusted letterspacing for vertical headline and increased outer ring contrast."),
			DiffSummary: []DesignVersionDiffChip{
				{Kind: "change", Label: i18nOrDefault(lang, "design.versions.diff.kerning", "Kerning refined")},
				{Kind: "add", Label: i18nOrDefault(lang, "design.versions.diff.grid", "Guides enabled")},
			},
			CompareURL:   "/design/editor?version=ver-210",
			RollbackURL:  "/design/versions/ver-210/rollback",
			DuplicateURL: "/design/versions/ver-210/duplicate",
			DeleteURL:    "/design/versions/ver-210/delete",
			PreviewBefore: DesignVersionPreviewPane{
				Title:       "Baseline v1.11.2",
				Description: i18nOrDefault(lang, "design.versions.preview.before", "Previous approved export"),
				Image:       "https://cdn.hanko-field.app/designs/df-219a/v1.11.2.svg",
				Badge:       i18nOrDefault(lang, "design.versions.preview.badge.previous", "Before"),
				Meta:        []string{"1,200 DPI", "Ring contrast 56%"},
			},
			PreviewAfter: DesignVersionPreviewPane{
				Title:       "v1.12.0",
				Description: i18nOrDefault(lang, "design.versions.preview.after", "Ready for engraving"),
				Image:       "https://cdn.hanko-field.app/designs/df-219a/v1.12.0.svg",
				Badge:       i18nOrDefault(lang, "design.versions.preview.badge.after", "After"),
				Meta:        []string{"1,200 DPI", "Ring contrast 68%"},
			},
			Insights: []DesignVersionInsight{
				{Label: i18nOrDefault(lang, "design.versions.insight.approval", "Approvals"), Value: "5/6 ✅", Tone: "success", Icon: "check-circle"},
				{Label: i18nOrDefault(lang, "design.versions.insight.ai", "AI assists incorporated"), Value: "2", Tone: "info", Icon: "sparkles"},
				{Label: i18nOrDefault(lang, "design.versions.insight.rollbacks", "Rollbacks in sprint"), Value: "0", Tone: "muted", Icon: "arrow-uturn-left"},
			},
			ChangelogTags: []DesignVersionDiffChip{
				{Kind: "change", Label: i18nOrDefault(lang, "design.versions.diff.letterspacing", "Letterspacing +4")},
				{Kind: "add", Label: i18nOrDefault(lang, "design.versions.diff.texture", "Texture enabled")},
			},
			TimelineEntries: []DesignVersionTimelineEntry{
				{
					ID:          "ev-730",
					Title:       i18nOrDefault(lang, "design.versions.timeline.sent", "Sent to production queue"),
					Description: i18nOrDefault(lang, "design.versions.timeline.sent.desc", "Queued for engraving batch #1241"),
					Actor:       "Automation",
					ActorRole:   "System",
					DisplayTime: "2025-03-24 09:50",
					Relative:    formatDesignVersionRelative(lang, 40*time.Minute),
					Icon:        "briefcase",
					Tone:        "info",
				},
				{
					ID:          "ev-729",
					Title:       i18nOrDefault(lang, "design.versions.timeline.approved", "Stakeholder approval"),
					Description: i18nOrDefault(lang, "design.versions.timeline.approved.desc", "Operations signed off on engraving contrast."),
					Actor:       "Nao Takahashi",
					ActorRole:   i18nOrDefault(lang, "design.versions.timeline.role.ops", "Production Ops"),
					DisplayTime: "2025-03-24 09:20",
					Relative:    formatDesignVersionRelative(lang, 1*time.Hour+10*time.Minute),
					Icon:        "check-badge",
					Tone:        "success",
				},
			},
		},
		{
			ID:           "ver-208",
			VersionLabel: "v1.11.2",
			StatusBadge:  i18nOrDefault(lang, "design.versions.status.published", "Published"),
			StatusTone:   "info",
			Tag:          i18nOrDefault(lang, "design.versions.tag.reviewed", "Reviewed"),
			TagTone:      "indigo",
			AuthorID:     "hk-sakura",
			AuthorName:   "Koki Hayashi",
			AuthorRole:   i18nOrDefault(lang, "design.versions.role.engraver", "Lead Engraver"),
			CreatedAt:    designVersionsNow.Add(-26 * time.Hour),
			Note:         i18nOrDefault(lang, "design.versions.note.stroke-balance", "Balanced stroke tapering on vertical axis and lowered seal script contrast."),
			DiffSummary: []DesignVersionDiffChip{
				{Kind: "change", Label: i18nOrDefault(lang, "design.versions.diff.stroke", "Stroke width -2")},
				{Kind: "issue", Label: i18nOrDefault(lang, "design.versions.diff.review", "Requires QA review")},
			},
			CompareURL:   "/design/editor?version=ver-208",
			RollbackURL:  "/design/versions/ver-208/rollback",
			DuplicateURL: "/design/versions/ver-208/duplicate",
			DeleteURL:    "/design/versions/ver-208/delete",
			PreviewBefore: DesignVersionPreviewPane{
				Title:       "v1.10.4",
				Description: i18nOrDefault(lang, "design.versions.preview.before", "Previous approved export"),
				Image:       "https://cdn.hanko-field.app/designs/df-219a/v1.10.4.svg",
				Badge:       i18nOrDefault(lang, "design.versions.preview.badge.previous", "Before"),
				Meta:        []string{"1,200 DPI", "Stroke contrast 62%"},
			},
			PreviewAfter: DesignVersionPreviewPane{
				Title:       "v1.11.2",
				Description: i18nOrDefault(lang, "design.versions.preview.after", "Ready for review"),
				Image:       "https://cdn.hanko-field.app/designs/df-219a/v1.11.2.svg",
				Badge:       i18nOrDefault(lang, "design.versions.preview.badge.after", "After"),
				Meta:        []string{"1,200 DPI", "Stroke contrast 58%"},
			},
			Insights: []DesignVersionInsight{
				{Label: i18nOrDefault(lang, "design.versions.insight.comments", "Comments"), Value: "8", Tone: "info", Icon: "chat-bubble-bottom-center-text"},
				{Label: i18nOrDefault(lang, "design.versions.insight.ai", "AI assists incorporated"), Value: "1", Tone: "info", Icon: "sparkles"},
			},
			ChangelogTags: []DesignVersionDiffChip{
				{Kind: "change", Label: i18nOrDefault(lang, "design.versions.diff.stroke-taper", "Stroke taper refined")},
			},
			TimelineEntries: []DesignVersionTimelineEntry{
				{
					ID:          "ev-725",
					Title:       i18nOrDefault(lang, "design.versions.timeline.reviewed", "Design review completed"),
					Description: i18nOrDefault(lang, "design.versions.timeline.reviewed.desc", "Mina signed off a11y contrast notes."),
					Actor:       "Mina Nagata",
					ActorRole:   i18nOrDefault(lang, "design.versions.role.art-director", "Art Director"),
					DisplayTime: "2025-03-23 08:10",
					Relative:    formatDesignVersionRelative(lang, 26*time.Hour+20*time.Minute),
					Icon:        "clipboard-document-check",
					Tone:        "info",
				},
			},
		},
		{
			ID:           "ver-204",
			VersionLabel: "v1.10.4",
			StatusBadge:  i18nOrDefault(lang, "design.versions.status.archived", "Archived"),
			StatusTone:   "muted",
			Tag:          i18nOrDefault(lang, "design.versions.tag.ai", "AI assist"),
			TagTone:      "sky",
			AuthorID:     "ops-auto",
			AuthorName:   i18nOrDefault(lang, "design.versions.author.auto", "Autosave service"),
			AuthorRole:   i18nOrDefault(lang, "design.versions.role.auto", "System"),
			CreatedAt:    designVersionsNow.Add(-4 * 24 * time.Hour),
			Note:         i18nOrDefault(lang, "design.versions.note.ai", "Autosave before applying AI suggestion DF-431."),
			DiffSummary: []DesignVersionDiffChip{
				{Kind: "pending", Label: i18nOrDefault(lang, "design.versions.diff.pending", "Awaiting review")},
			},
			CompareURL:    "/design/editor?version=ver-204",
			RollbackURL:   "/design/versions/ver-204/rollback",
			DuplicateURL:  "/design/versions/ver-204/duplicate",
			DeleteURL:     "/design/versions/ver-204/delete",
			PreviewBefore: basePreview,
			PreviewAfter: DesignVersionPreviewPane{
				Title:       "v1.10.4",
				Description: i18nOrDefault(lang, "design.versions.preview.after", "Ready for review"),
				Image:       "https://cdn.hanko-field.app/designs/df-219a/v1.10.4.svg",
				Badge:       i18nOrDefault(lang, "design.versions.preview.badge.after", "After"),
				Meta:        []string{"1,200 DPI", "Auto saved"},
			},
			Insights: []DesignVersionInsight{
				{Label: i18nOrDefault(lang, "design.versions.insight.ai", "AI assists incorporated"), Value: "0", Tone: "muted", Icon: "sparkles"},
			},
		},
		{
			ID:           "ver-198",
			VersionLabel: "v1.9.0",
			StatusBadge:  i18nOrDefault(lang, "design.versions.status.archived", "Archived"),
			StatusTone:   "muted",
			Tag:          i18nOrDefault(lang, "design.versions.tag.feedback", "Client feedback"),
			TagTone:      "amber",
			AuthorID:     "mn-labs",
			AuthorName:   "Mina Nagata",
			AuthorRole:   i18nOrDefault(lang, "design.versions.role.art-director", "Art Director"),
			CreatedAt:    designVersionsNow.Add(-9 * 24 * time.Hour),
			Note:         i18nOrDefault(lang, "design.versions.note.client", "Client requested alternate vertical glyph arrangement for ceremonial seal."),
			DiffSummary: []DesignVersionDiffChip{
				{Kind: "remove", Label: i18nOrDefault(lang, "design.versions.diff.frame", "Removed desk frame")},
				{Kind: "add", Label: i18nOrDefault(lang, "design.versions.diff.alt-glyph", "Added alt glyph")},
			},
			CompareURL:    "/design/editor?version=ver-198",
			RollbackURL:   "/design/versions/ver-198/rollback",
			DuplicateURL:  "/design/versions/ver-198/duplicate",
			DeleteURL:     "/design/versions/ver-198/delete",
			PreviewBefore: basePreview,
			PreviewAfter: DesignVersionPreviewPane{
				Title:       "v1.9.0",
				Description: i18nOrDefault(lang, "design.versions.preview.after", "Alternate glyph layout"),
				Image:       "https://cdn.hanko-field.app/designs/df-219a/v1.9.0.svg",
				Badge:       i18nOrDefault(lang, "design.versions.preview.badge.after", "After"),
				Meta:        []string{"900 DPI", "Alt glyph enabled"},
			},
		},
		{
			ID:           "ver-187",
			VersionLabel: "v1.7.5",
			StatusBadge:  i18nOrDefault(lang, "design.versions.status.archived", "Archived"),
			StatusTone:   "muted",
			Tag:          i18nOrDefault(lang, "design.versions.tag.milestone", "Milestone"),
			TagTone:      "slate",
			AuthorID:     "hk-sakura",
			AuthorName:   "Koki Hayashi",
			AuthorRole:   i18nOrDefault(lang, "design.versions.role.engraver", "Lead Engraver"),
			CreatedAt:    designVersionsNow.Add(-18 * 24 * time.Hour),
			Note:         i18nOrDefault(lang, "design.versions.note.export", "First export shared with client for preview."),
			DiffSummary: []DesignVersionDiffChip{
				{Kind: "add", Label: i18nOrDefault(lang, "design.versions.diff.export", "Exported mockups")},
			},
			CompareURL:    "/design/editor?version=ver-187",
			RollbackURL:   "/design/versions/ver-187/rollback",
			DuplicateURL:  "/design/versions/ver-187/duplicate",
			DeleteURL:     "/design/versions/ver-187/delete",
			PreviewBefore: basePreview,
			PreviewAfter: DesignVersionPreviewPane{
				Title:       "v1.7.5",
				Description: i18nOrDefault(lang, "design.versions.preview.after", "Draft export"),
				Image:       "https://cdn.hanko-field.app/designs/df-219a/v1.7.5.svg",
				Badge:       i18nOrDefault(lang, "design.versions.preview.badge.after", "After"),
				Meta:        []string{"900 DPI", "Client preview"},
			},
		},
		{
			ID:           "ver-160",
			VersionLabel: "v1.2.0",
			StatusBadge:  i18nOrDefault(lang, "design.versions.status.archived", "Archived"),
			StatusTone:   "muted",
			Tag:          i18nOrDefault(lang, "design.versions.tag.initial", "Initial draft"),
			TagTone:      "gray",
			AuthorID:     "mn-labs",
			AuthorName:   "Mina Nagata",
			AuthorRole:   i18nOrDefault(lang, "design.versions.role.art-director", "Art Director"),
			CreatedAt:    designVersionsNow.Add(-33 * 24 * time.Hour),
			Note:         i18nOrDefault(lang, "design.versions.note.initial", "Initial draft seeded from template tpl-ring-corporate."),
			DiffSummary: []DesignVersionDiffChip{
				{Kind: "add", Label: i18nOrDefault(lang, "design.versions.diff.initial", "Baseline established")},
			},
			CompareURL:    "/design/editor?version=ver-160",
			RollbackURL:   "/design/versions/ver-160/rollback",
			DuplicateURL:  "/design/versions/ver-160/duplicate",
			DeleteURL:     "/design/versions/ver-160/delete",
			PreviewBefore: basePreview,
			PreviewAfter: DesignVersionPreviewPane{
				Title:       "v1.2.0",
				Description: i18nOrDefault(lang, "design.versions.preview.after", "Initial baseline"),
				Image:       "https://cdn.hanko-field.app/designs/df-219a/v1.2.0.svg",
				Badge:       i18nOrDefault(lang, "design.versions.preview.badge.after", "After"),
				Meta:        []string{"600 DPI", "Template seed"},
			},
		},
	}
}
