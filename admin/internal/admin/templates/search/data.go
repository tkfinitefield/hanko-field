package search

import (
	"fmt"
	"strings"
	"time"

	adminsearch "finitefield.org/hanko-admin/internal/admin/search"
	"finitefield.org/hanko-admin/internal/admin/templates/partials"
)

// PageData represents the payload for the full search page.
type PageData struct {
	Title           string
	Query           QueryState
	ScopeOptions    []ScopeOption
	PersonaOptions  []SelectOption
	TableEndpoint   string
	TableData       TableData
	Summary         Summary
	ShortcutHints   []ShortcutHint
	Breadcrumbs     []partials.Breadcrumb
	AnalyticsBadge  string
	AnalyticsTone   string
	PersonaSelected string
}

// QueryState holds query parameters for rendering the form.
type QueryState struct {
	Term      string
	Scope     string
	StartDate string
	EndDate   string
	Persona   string
}

// ScopeOption represents a segmented control item.
type ScopeOption struct {
	Value string
	Label string
	Icon  string
}

// SelectOption represents a dropdown option.
type SelectOption struct {
	Value string
	Label string
}

// Summary captures high level stats for the current query.
type Summary struct {
	TotalHits int
	Duration  string
}

// ShortcutHint documents available keyboard shortcuts.
type ShortcutHint struct {
	Keys        []string
	Description string
}

// TableData represents the payload for the results fragment.
type TableData struct {
	QueryTerm     string
	Error         string
	EmptyMessage  string
	Groups        []ResultGroupView
	Summary       Summary
	ShortcutHints []ShortcutHint
}

// ResultGroupView groups hits by entity type.
type ResultGroupView struct {
	Entity  string
	Label   string
	Icon    string
	Total   int
	HasMore bool
	Hits    []HitView
}

// HitView is a single row in the results table.
type HitView struct {
	ID          string
	Entity      string
	Title       string
	Description string
	Badge       string
	BadgeTone   string
	URL         string
	Score       float64
	OccurredAt  *time.Time
	Persona     string
	Metadata    []MetadataView
}

// MetadataView captures supplemental key/value pairs.
type MetadataView struct {
	Key   string
	Value string
	Icon  string
}

// BuildPageData composes the payload for SSR rendering.
func BuildPageData(basePath string, state QueryState, table TableData) PageData {
	return PageData{
		Title:           "横断検索",
		Query:           state,
		ScopeOptions:    defaultScopeOptions(),
		PersonaOptions:  defaultPersonaOptions(),
		TableEndpoint:   joinBase(basePath, "/search/table"),
		TableData:       table,
		Summary:         table.Summary,
		ShortcutHints:   table.ShortcutHints,
		Breadcrumbs:     breadcrumbItems(),
		AnalyticsBadge:  analyticsLabel(table.Summary.TotalHits),
		AnalyticsTone:   analyticsTone(table.Summary.TotalHits),
		PersonaSelected: state.Persona,
	}
}

// TablePayload prepares the table fragment payload.
func TablePayload(state QueryState, result adminsearch.ResultSet, errMsg string) TableData {
	payload := TableData{
		QueryTerm:     state.Term,
		Groups:        toResultGroups(result.Groups),
		Summary:       Summary{TotalHits: result.Total, Duration: formatDuration(result.Duration)},
		ShortcutHints: defaultShortcutHints(),
	}
	if errMsg != "" {
		payload.Error = errMsg
	}

	if len(payload.Groups) == 0 && payload.Error == "" {
		term := strings.TrimSpace(state.Term)
		if term == "" {
			payload.EmptyMessage = "検索キーワードを入力すると、注文・ユーザー・レビューを横断して結果が表示されます。"
		} else {
			payload.EmptyMessage = fmt.Sprintf("「%s」に一致する結果は見つかりませんでした。フィルタ条件を調整してください。", term)
		}
	}

	return payload
}

func toResultGroups(groups []adminsearch.ResultGroup) []ResultGroupView {
	result := make([]ResultGroupView, 0, len(groups))
	for _, group := range groups {
		result = append(result, ResultGroupView{
			Entity:  string(group.Entity),
			Label:   group.Label,
			Icon:    group.Icon,
			Total:   group.Total,
			HasMore: group.HasMore,
			Hits:    toHitViews(group.Hits),
		})
	}
	return result
}

func toHitViews(hits []adminsearch.Hit) []HitView {
	result := make([]HitView, 0, len(hits))
	for _, hit := range hits {
		var occurred *time.Time
		if hit.OccurredAt != nil && !hit.OccurredAt.IsZero() {
			t := *hit.OccurredAt
			occurred = &t
		}
		result = append(result, HitView{
			ID:          hit.ID,
			Entity:      string(hit.Entity),
			Title:       hit.Title,
			Description: hit.Description,
			Badge:       hit.Badge,
			BadgeTone:   hit.BadgeTone,
			URL:         hit.URL,
			Score:       hit.Score,
			OccurredAt:  occurred,
			Persona:     hit.Persona,
			Metadata:    toMetadataViews(hit.Metadata),
		})
	}
	return result
}

func toMetadataViews(list []adminsearch.Metadata) []MetadataView {
	result := make([]MetadataView, 0, len(list))
	for _, item := range list {
		result = append(result, MetadataView{
			Key:   item.Key,
			Value: item.Value,
			Icon:  item.Icon,
		})
	}
	return result
}

func defaultScopeOptions() []ScopeOption {
	return []ScopeOption{
		{Value: "all", Label: "すべて", Icon: "🔍"},
		{Value: "orders", Label: "注文", Icon: "🧾"},
		{Value: "users", Label: "ユーザー", Icon: "🧑"},
		{Value: "reviews", Label: "レビュー", Icon: "⭐"},
	}
}

func defaultPersonaOptions() []SelectOption {
	return []SelectOption{
		{Value: "", Label: "全員"},
		{Value: "operations", Label: "オペレーション"},
		{Value: "cs", Label: "CS"},
		{Value: "marketing", Label: "マーケティング"},
		{Value: "finance", Label: "ファイナンス"},
	}
}

func defaultShortcutHints() []ShortcutHint {
	return []ShortcutHint{
		{Keys: []string{"/"}, Description: "検索バーにフォーカス"},
		{Keys: []string{"↑", "↓"}, Description: "結果の移動"},
		{Keys: []string{"↵"}, Description: "選択中の結果を開く"},
	}
}

func breadcrumbItems() []partials.Breadcrumb {
	return []partials.Breadcrumb{
		{Label: "横断検索"},
	}
}

func joinBase(base, suffix string) string {
	base = strings.TrimSpace(base)
	if base == "" || base == "/" {
		base = ""
	}
	if !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}
	path := base + suffix
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "<1ms"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func analyticsLabel(total int) string {
	if total <= 0 {
		return "0 件"
	}
	return fmt.Sprintf("%d 件", total)
}

func analyticsTone(total int) string {
	if total > 0 {
		return "info"
	}
	return "muted"
}
