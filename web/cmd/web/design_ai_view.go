package main

import (
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// DesignAISuggestionsView represents the full state required to render the AI suggestions page and fragments.
type DesignAISuggestionsView struct {
	Lang               string
	DesignID           string
	ActiveStatus       string
	ActivePersona      string
	ActiveSort         string
	StatusFilters      []DesignAISegmentOption
	PersonaOptions     []DesignAISelectOption
	SortOptions        []DesignAISelectOption
	Suggestions        []DesignAISuggestionRow
	Queue              DesignAIQueueStatus
	Stats              []DesignAIStat
	Poll               DesignAIPolling
	SelectedID         string
	SelectedSuggestion DesignAISuggestionDetail
	Empty              bool
	Errors             []string
	Query              string
	LastUpdated        time.Time
}

// DesignAISegmentOption describes a segmented control option for filtering by status.
type DesignAISegmentOption struct {
	Value string
	Label string
	Count int
}

// DesignAISelectOption represents an option for the persona and sort dropdowns.
type DesignAISelectOption struct {
	Value  string
	Label  string
	Active bool
}

// DesignAISuggestionRow is the lightweight view model for the suggestions table fragment.
type DesignAISuggestionRow struct {
	ID           string
	Prompt       string
	Persona      string
	PersonaLabel string
	Status       string
	StatusLabel  string
	StatusTone   string
	ScoreDisplay string
	ScoreTone    string
	ScoreTag     string
	Tags         []string
	Thumbnail    string
	Age          string
	ETA          string
	Diff         []DesignAIDiff
	Confidence   string
}

// DesignAIDiff represents a highlighted diff item between baseline and AI suggestion.
type DesignAIDiff struct {
	Kind   string
	Label  string
	Before string
	After  string
}

// DesignAIQueueStatus summarises queue depth and processing indicators.
type DesignAIQueueStatus struct {
	Pending          int
	Processing       int
	Backlog          int
	AvgSeconds       int
	Message          string
	UpdatedAgo       string
	PollEverySeconds int
}

// DesignAIStat provides a quick analytics snapshot.
type DesignAIStat struct {
	Label string
	Value string
	Trend string
	Tone  string
}

// DesignAIPolling governs automatic polling behaviour.
type DesignAIPolling struct {
	Enabled          bool
	IntervalSeconds  int
	CountdownSeconds int
}

// DesignAISuggestionDetail powers the preview drawer.
type DesignAISuggestionDetail struct {
	ID              string
	Title           string
	Prompt          string
	PersonaLabel    string
	Status          string
	StatusLabel     string
	StatusTone      string
	ScoreDisplay    string
	ScoreTone       string
	ScoreTag        string
	Description     string
	Thumbnail       string
	GeneratedAgo    string
	Confidence      string
	Diff            []DesignAIDiff
	Insights        []DesignAIInsight
	Notes           []string
	AcceptURL       string
	RejectURL       string
	ActionsDisabled bool
}

// DesignAIInsight lists auxiliary metrics for the preview drawer.
type DesignAIInsight struct {
	Label   string
	Value   string
	Tone    string
	Tooltip string
}

type designAISuggestion struct {
	ID           string
	Prompt       string
	Persona      string
	Status       string
	Score        int
	ScoreTag     string
	ScoreTone    string
	Tags         []string
	Thumbnail    string
	GeneratedAgo string
	ETA          string
	Confidence   string
	Description  string
	Diff         []DesignAIDiff
	Insights     []DesignAIInsight
	Notes        []string
}

// buildDesignAISuggestionsView assembles the view model given query parameters.
func buildDesignAISuggestionsView(lang string, q url.Values) DesignAISuggestionsView {
	status := normalizeAISuggestionStatus(strings.TrimSpace(q.Get("status")))
	persona := normalizeAISuggestionPersona(strings.TrimSpace(q.Get("persona")))
	sortKey := normalizeAISuggestionSort(strings.TrimSpace(q.Get("sort")))
	focus := strings.TrimSpace(q.Get("focus"))

	all := designAIMockData(lang)

	statusCounts := map[string]int{}
	personaCounts := map[string]int{}
	for _, sg := range all {
		statusCounts[sg.Status]++
		personaCounts[sg.Persona]++
	}

	filtered := make([]designAISuggestion, 0, len(all))
	for _, sg := range all {
		if status != "all" {
			if status == "queued" {
				if sg.Status != "queued" && sg.Status != "processing" {
					continue
				}
			} else if sg.Status != status {
				continue
			}
		}
		if persona != "" && sg.Persona != persona {
			continue
		}
		filtered = append(filtered, sg)
	}

	if sortKey == "score" {
		sort.SliceStable(filtered, func(i, j int) bool {
			return filtered[i].Score > filtered[j].Score
		})
	} else {
		// default newest -> rely on order defined in mock data. Could reverse if needed using GeneratedAgo heuristics.
	}

	rows := make([]DesignAISuggestionRow, 0, len(filtered))
	for _, sg := range filtered {
		label, tone := aiStatusLabel(lang, sg.Status)
		rows = append(rows, DesignAISuggestionRow{
			ID:           sg.ID,
			Prompt:       sg.Prompt,
			Persona:      sg.Persona,
			PersonaLabel: personaLabel(lang, sg.Persona),
			Status:       sg.Status,
			StatusLabel:  label,
			StatusTone:   tone,
			ScoreDisplay: formatAIScore(lang, sg.Score),
			ScoreTone:    sg.ScoreTone,
			ScoreTag:     sg.ScoreTag,
			Tags:         sg.Tags,
			Thumbnail:    sg.Thumbnail,
			Age:          sg.GeneratedAgo,
			ETA:          sg.ETA,
			Diff:         sg.Diff,
			Confidence:   sg.Confidence,
		})
	}

	selectedID := ""
	var selected designAISuggestion
	selectedFound := false
	if len(filtered) > 0 {
		if focus != "" {
			for _, sg := range filtered {
				if sg.ID == focus {
					selectedID = sg.ID
					selected = sg
					selectedFound = true
					break
				}
			}
		}
		if !selectedFound {
			selectedID = filtered[0].ID
			selected = filtered[0]
			selectedFound = true
		}
	}

	detail := DesignAISuggestionDetail{}
	if selectedFound {
		detail = buildDesignAISuggestionDetail(lang, selected, selected.Status)
	}

	view := DesignAISuggestionsView{
		Lang:          lang,
		DesignID:      q.Get("designId"),
		ActiveStatus:  status,
		ActivePersona: persona,
		ActiveSort:    sortKey,
		Suggestions:   rows,
		Queue:         buildDesignAIQueueStatus(lang, statusCounts),
		Stats:         buildDesignAIStats(lang),
		Poll: DesignAIPolling{
			Enabled:          true,
			IntervalSeconds:  12,
			CountdownSeconds: 12,
		},
		SelectedID:         selectedID,
		SelectedSuggestion: detail,
		Empty:              len(rows) == 0,
		LastUpdated:        time.Now(),
	}

	view.StatusFilters = buildDesignAIStatusFilters(lang, statusCounts, status)
	view.PersonaOptions = buildDesignAIPersonaOptions(lang, personaCounts, persona)
	view.SortOptions = buildDesignAISortOptions(lang, sortKey)
	focusParam := ""
	if selectedID != "" && len(rows) > 0 {
		focusParam = selectedID
	}
	view.Query = designAIBaseQuery(status, persona, sortKey, focusParam)

	return view
}

func buildDesignAIQueueStatus(lang string, counts map[string]int) DesignAIQueueStatus {
	pending := counts["queued"]
	pending += counts["processing"]
	return DesignAIQueueStatus{
		Pending:          counts["queued"],
		Processing:       counts["processing"],
		Backlog:          pending,
		AvgSeconds:       26,
		Message:          i18nOrDefault(lang, "design.ai.queue.message", "AI queue refreshes automatically."),
		UpdatedAgo:       i18nOrDefault(lang, "design.ai.queue.updated_ago", "Refreshed 30s ago"),
		PollEverySeconds: 12,
	}
}

func buildDesignAIStats(lang string) []DesignAIStat {
	return []DesignAIStat{
		{
			Label: i18nOrDefault(lang, "design.ai.stats.adoption", "Adoption rate"),
			Value: "68%",
			Trend: "+5.2%",
			Tone:  "positive",
		},
		{
			Label: i18nOrDefault(lang, "design.ai.stats.success", "Success on first pass"),
			Value: "82%",
			Trend: "+3.4%",
			Tone:  "positive",
		},
		{
			Label: i18nOrDefault(lang, "design.ai.stats.time_saved", "Avg. time saved"),
			Value: "7m 40s",
			Trend: "–48s",
			Tone:  "neutral",
		},
		{
			Label: i18nOrDefault(lang, "design.ai.stats.rework", "Manual rework"),
			Value: "9%",
			Trend: "-1.1%",
			Tone:  "success",
		},
	}
}

func buildDesignAISuggestionDetail(lang string, sg designAISuggestion, overrideStatus string) DesignAISuggestionDetail {
	status := sg.Status
	if overrideStatus != "" {
		status = overrideStatus
	}
	label, tone := aiStatusLabel(lang, status)

	return DesignAISuggestionDetail{
		ID:              sg.ID,
		Title:           sg.Prompt,
		Prompt:          sg.Prompt,
		PersonaLabel:    personaLabel(lang, sg.Persona),
		Status:          status,
		StatusLabel:     label,
		StatusTone:      tone,
		ScoreDisplay:    formatAIScore(lang, sg.Score),
		ScoreTone:       sg.ScoreTone,
		ScoreTag:        sg.ScoreTag,
		Description:     sg.Description,
		Thumbnail:       sg.Thumbnail,
		GeneratedAgo:    sg.GeneratedAgo,
		Confidence:      sg.Confidence,
		Diff:            sg.Diff,
		Insights:        sg.Insights,
		Notes:           sg.Notes,
		AcceptURL:       "/design/ai/suggestions/" + sg.ID + "/accept",
		RejectURL:       "/design/ai/suggestions/" + sg.ID + "/reject",
		ActionsDisabled: status == "accepted" || status == "rejected",
	}
}

func buildDesignAIStatusFilters(lang string, counts map[string]int, active string) []DesignAISegmentOption {
	total := 0
	for _, c := range counts {
		total += c
	}
	filters := []DesignAISegmentOption{
		{
			Value: "all",
			Label: i18nOrDefault(lang, "design.ai.filters.all", "All"),
			Count: total,
		},
		{
			Value: "ready",
			Label: i18nOrDefault(lang, "design.ai.filters.ready", "Ready"),
			Count: counts["ready"],
		},
		{
			Value: "queued",
			Label: i18nOrDefault(lang, "design.ai.filters.queued", "Queued"),
			Count: counts["queued"] + counts["processing"],
		},
		{
			Value: "accepted",
			Label: i18nOrDefault(lang, "design.ai.filters.accepted", "Accepted"),
			Count: counts["accepted"],
		},
		{
			Value: "rejected",
			Label: i18nOrDefault(lang, "design.ai.filters.rejected", "Rejected"),
			Count: counts["rejected"],
		},
	}

	// Ensure active option exists. If not, fall back to "all".
	found := false
	for _, f := range filters {
		if f.Value == active {
			found = true
			break
		}
	}
	if !found {
		active = "all"
	}
	return filters
}

func buildDesignAIPersonaOptions(lang string, counts map[string]int, active string) []DesignAISelectOption {
	options := []DesignAISelectOption{
		{Value: "", Label: i18nOrDefault(lang, "design.ai.persona.all", "All personas"), Active: active == ""},
		{Value: "corporate", Label: i18nOrDefault(lang, "design.ai.persona.corporate", "Corporate"), Active: active == "corporate"},
		{Value: "personal", Label: i18nOrDefault(lang, "design.ai.persona.personal", "Personal"), Active: active == "personal"},
		{Value: "government", Label: i18nOrDefault(lang, "design.ai.persona.government", "Government"), Active: active == "government"},
	}
	_ = counts // reserved for future count displays
	return options
}

func buildDesignAISortOptions(lang string, active string) []DesignAISelectOption {
	options := []DesignAISelectOption{
		{Value: "newest", Label: i18nOrDefault(lang, "design.ai.sort.newest", "Newest first"), Active: active == "newest"},
		{Value: "score", Label: i18nOrDefault(lang, "design.ai.sort.score", "Highest score"), Active: active == "score"},
	}
	if active == "" {
		options[0].Active = true
	}
	return options
}

func designAIBaseQuery(status, persona, sortKey, focus string) string {
	q := url.Values{}
	if status != "" && status != "all" {
		q.Set("status", status)
	}
	if persona != "" {
		q.Set("persona", persona)
	}
	if sortKey != "" && sortKey != "newest" {
		q.Set("sort", sortKey)
	}
	if focus != "" {
		q.Set("focus", focus)
	}
	return q.Encode()
}

func designAISuggestionByID(list []designAISuggestion, id string) (designAISuggestion, bool) {
	for _, sg := range list {
		if sg.ID == id {
			return sg, true
		}
	}
	return designAISuggestion{}, false
}

func normalizeAISuggestionStatus(status string) string {
	switch status {
	case "ready", "queued", "accepted", "rejected":
		return status
	default:
		return "all"
	}
}

func normalizeAISuggestionPersona(persona string) string {
	switch persona {
	case "corporate", "personal", "government":
		return persona
	default:
		return ""
	}
}

func normalizeAISuggestionSort(sortKey string) string {
	switch sortKey {
	case "score":
		return "score"
	default:
		return "newest"
	}
}

func aiStatusLabel(lang, status string) (string, string) {
	switch status {
	case "ready":
		return i18nOrDefault(lang, "design.ai.status.ready", "Ready"), "success"
	case "queued":
		return i18nOrDefault(lang, "design.ai.status.queued", "Queued"), "warning"
	case "processing":
		return i18nOrDefault(lang, "design.ai.status.processing", "Processing"), "info"
	case "accepted":
		return i18nOrDefault(lang, "design.ai.status.accepted", "Accepted"), "success"
	case "rejected":
		return i18nOrDefault(lang, "design.ai.status.rejected", "Rejected"), "muted"
	default:
		return i18nOrDefault(lang, "design.ai.status.unknown", "Unknown"), "muted"
	}
}

func personaLabel(lang, persona string) string {
	switch persona {
	case "corporate":
		return i18nOrDefault(lang, "design.ai.persona.corporate", "Corporate")
	case "personal":
		return i18nOrDefault(lang, "design.ai.persona.personal", "Personal")
	case "government":
		return i18nOrDefault(lang, "design.ai.persona.government", "Government")
	default:
		return i18nOrDefault(lang, "design.ai.persona.unknown", "General")
	}
}

func formatAIScore(lang string, score int) string {
	if score <= 0 {
		return i18nOrDefault(lang, "design.ai.score.pending", "Pending")
	}
	return strconv.Itoa(score)
}

func designAIMockData(lang string) []designAISuggestion {
	return []designAISuggestion{
		{
			ID:           "sg-401",
			Prompt:       i18nOrDefault(lang, "design.ai.sg401.prompt", "Double-ring corporate seal with compliance annotations"),
			Persona:      "corporate",
			Status:       "ready",
			Score:        92,
			ScoreTag:     i18nOrDefault(lang, "design.ai.sg401.score_tag", "High fit"),
			ScoreTone:    "success",
			Tags:         []string{"double-ring", "compliance", "kanji"},
			Thumbnail:    "",
			GeneratedAgo: i18nOrDefault(lang, "design.ai.sg401.generated", "Generated 2 minutes ago"),
			ETA:          i18nOrDefault(lang, "design.ai.sg401.eta", "Auto-applies in 18m"),
			Confidence:   "Confidence 94%",
			Description:  i18nOrDefault(lang, "design.ai.sg401.description", "Balanced kanji strokes with refreshed inner glyph spacing to meet corporate registrability rules."),
			Diff: []DesignAIDiff{
				{Kind: "add", Label: i18nOrDefault(lang, "design.ai.sg401.diff.add", "Added compliance ring"), Before: "None", After: "Dual 0.7mm rings"},
				{Kind: "change", Label: i18nOrDefault(lang, "design.ai.sg401.diff.change", "Adjusted stroke ratio"), Before: "1.0", After: "0.86"},
				{Kind: "change", Label: i18nOrDefault(lang, "design.ai.sg401.diff.spacing", "Spacing +3% outer kanji"), Before: "+0%", After: "+3%"},
			},
			Insights: []DesignAIInsight{
				{Label: i18nOrDefault(lang, "design.ai.insight.adoption", "Projected adoption"), Value: "73%", Tone: "positive", Tooltip: i18nOrDefault(lang, "design.ai.insight.adoption.tip", "Based on similar corporate seals accepted this quarter.")},
				{Label: i18nOrDefault(lang, "design.ai.insight.engraving", "Engraving risk"), Value: "Low", Tone: "success", Tooltip: i18nOrDefault(lang, "design.ai.insight.engraving.tip", "Stroke width above 0.25mm minimum.")},
			},
			Notes: []string{
				i18nOrDefault(lang, "design.ai.sg401.note1", "Inner title uses vetted Mincho stroke set."),
				i18nOrDefault(lang, "design.ai.sg401.note2", "Queue flagged medium priority due to executive tier."),
			},
		},
		{
			ID:           "sg-402",
			Prompt:       i18nOrDefault(lang, "design.ai.sg402.prompt", "Personal square seal emphasizing given name"),
			Persona:      "personal",
			Status:       "queued",
			Score:        0,
			ScoreTag:     i18nOrDefault(lang, "design.ai.sg402.score_tag", "Queued"),
			ScoreTone:    "warning",
			Tags:         []string{"nickname", "square"},
			Thumbnail:    "",
			GeneratedAgo: i18nOrDefault(lang, "design.ai.sg402.generated", "In queue (ETA 1m)"),
			ETA:          i18nOrDefault(lang, "design.ai.sg402.eta", "Processing next"),
			Confidence:   "Confidence pending",
			Description:  i18nOrDefault(lang, "design.ai.sg402.description", "Personalized layout prioritizing given name legibility for everyday approvals."),
			Diff: []DesignAIDiff{
				{Kind: "pending", Label: i18nOrDefault(lang, "design.ai.sg402.diff.pending", "Diff pending generation"), Before: "", After: ""},
			},
			Insights: []DesignAIInsight{
				{Label: i18nOrDefault(lang, "design.ai.insight.queue_position", "Queue position"), Value: "#2", Tone: "neutral", Tooltip: i18nOrDefault(lang, "design.ai.insight.queue_position.tip", "Estimated completion under 90 seconds.")},
			},
			Notes: []string{
				i18nOrDefault(lang, "design.ai.sg402.note1", "Baseline uses uploaded signature strokes."),
			},
		},
		{
			ID:           "sg-403",
			Prompt:       i18nOrDefault(lang, "design.ai.sg403.prompt", "Government case signature with vertical script"),
			Persona:      "government",
			Status:       "processing",
			Score:        0,
			ScoreTag:     i18nOrDefault(lang, "design.ai.sg403.score_tag", "Processing"),
			ScoreTone:    "info",
			Tags:         []string{"vertical", "registrable"},
			Thumbnail:    "",
			GeneratedAgo: i18nOrDefault(lang, "design.ai.sg403.generated", "Generating now"),
			ETA:          i18nOrDefault(lang, "design.ai.sg403.eta", "Rendering preview"),
			Confidence:   "Confidence pending",
			Description:  i18nOrDefault(lang, "design.ai.sg403.description", "High-contrast vertical script tuned for registry submission and optical capture."),
			Diff: []DesignAIDiff{
				{Kind: "change", Label: i18nOrDefault(lang, "design.ai.sg403.diff.baseline", "Baseline kerning adjustments"), Before: "Auto", After: "Manual +7%"},
			},
			Insights: []DesignAIInsight{
				{Label: i18nOrDefault(lang, "design.ai.insight.clearance", "Registry clearance likelihood"), Value: "Pending", Tone: "muted", Tooltip: i18nOrDefault(lang, "design.ai.insight.clearance.tip", "Awaiting contrast validation run.")},
			},
			Notes: []string{
				i18nOrDefault(lang, "design.ai.sg403.note1", "Government persona uses stricter serif set."),
			},
		},
		{
			ID:           "sg-398",
			Prompt:       i18nOrDefault(lang, "design.ai.sg398.prompt", "Corporate fallback with bold outer ring"),
			Persona:      "corporate",
			Status:       "accepted",
			Score:        88,
			ScoreTag:     i18nOrDefault(lang, "design.ai.sg398.score_tag", "In use"),
			ScoreTone:    "success",
			Tags:         []string{"accepted", "bold"},
			Thumbnail:    "",
			GeneratedAgo: i18nOrDefault(lang, "design.ai.sg398.generated", "Accepted 1 hour ago"),
			ETA:          i18nOrDefault(lang, "design.ai.sg398.eta", "Live in editor"),
			Confidence:   "Confidence 90%",
			Description:  i18nOrDefault(lang, "design.ai.sg398.description", "Accepted variation emphasizing outer glyphs for CFO signature set."),
			Diff: []DesignAIDiff{
				{Kind: "accept", Label: i18nOrDefault(lang, "design.ai.sg398.diff.accepted", "Live variation"), Before: "", After: ""},
			},
			Insights: []DesignAIInsight{
				{Label: i18nOrDefault(lang, "design.ai.insight.time_saved", "Time saved vs manual"), Value: "6m", Tone: "success", Tooltip: i18nOrDefault(lang, "design.ai.insight.time_saved.tip", "Accepted on first iteration.")},
			},
			Notes: []string{
				i18nOrDefault(lang, "design.ai.sg398.note1", "Logged for production engraving batch #1043."),
			},
		},
		{
			ID:           "sg-359",
			Prompt:       i18nOrDefault(lang, "design.ai.sg359.prompt", "Personal script with flowing autograph line"),
			Persona:      "personal",
			Status:       "rejected",
			Score:        74,
			ScoreTag:     i18nOrDefault(lang, "design.ai.sg359.score_tag", "Rejected"),
			ScoreTone:    "muted",
			Tags:         []string{"autograph", "reflow"},
			Thumbnail:    "",
			GeneratedAgo: i18nOrDefault(lang, "design.ai.sg359.generated", "Rejected 3 hours ago"),
			ETA:          i18nOrDefault(lang, "design.ai.sg359.eta", "Archived"),
			Confidence:   "Confidence 71%",
			Description:  i18nOrDefault(lang, "design.ai.sg359.description", "Curved autograph baseline that did not meet contrast requirements."),
			Diff: []DesignAIDiff{
				{Kind: "remove", Label: i18nOrDefault(lang, "design.ai.sg359.diff.remove", "Removed autograph curve"), Before: "Enabled", After: "Disabled"},
				{Kind: "issue", Label: i18nOrDefault(lang, "design.ai.sg359.diff.issue", "Contrast violation"), Before: "1.5", After: "0.8"},
			},
			Insights: []DesignAIInsight{
				{Label: i18nOrDefault(lang, "design.ai.insight.reason", "Rejection reason"), Value: i18nOrDefault(lang, "design.ai.sg359.reason", "Low contrast vs baseline"), Tone: "warning", Tooltip: i18nOrDefault(lang, "design.ai.insight.reason.tip", "Triggered by compliance rule D-14.")},
			},
			Notes: []string{
				i18nOrDefault(lang, "design.ai.sg359.note1", "User opted for manual retouch workflow."),
			},
		},
	}
}
