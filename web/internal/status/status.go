package status

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Summary captures an overview of the platform status and recent incidents.
type Summary struct {
	State      string
	StateLabel string
	UpdatedAt  time.Time
	Components []Component
	Incidents  []Incident
}

// Component represents the status of an individual subsystem.
type Component struct {
	Name   string
	Status string
}

// Incident describes a status incident with optional updates.
type Incident struct {
	ID         string
	Title      string
	Status     string
	Impact     string
	StartedAt  time.Time
	ResolvedAt time.Time
	Updates    []IncidentUpdate
}

// IncidentUpdate captures a timeline entry for an incident.
type IncidentUpdate struct {
	Timestamp time.Time
	Status    string
	Body      string
}

// Client fetches status summaries from an external endpoint with local fallbacks.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient builds a status client with the provided base URL. When baseURL is empty,
// the client will exclusively serve fallback data.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimSpace(baseURL),
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

var (
	cacheMu      sync.RWMutex
	summaryCache = map[string]statusCacheEntry{}
	cacheTTL     = 2 * time.Minute
)

type statusCacheEntry struct {
	summary Summary
	expires time.Time
}

// SetCacheTTL configures the cache duration (primarily for tests).
func SetCacheTTL(d time.Duration) {
	if d <= 0 {
		d = time.Minute
	}
	cacheTTL = d
}

// FetchSummary returns a localized status summary, prioritizing cached values,
// then remote data, and finally local fallback content.
func (c *Client) FetchSummary(ctx context.Context, lang string) (Summary, error) {
	lang = normalizeLang(lang)
	if summary, ok := cachedSummary(lang); ok {
		return cloneSummary(summary), nil
	}

	var summary Summary
	var err error
	if c != nil && c.baseURL != "" {
		summary, err = c.fetchRemote(ctx, lang)
		if err != nil && !errors.Is(err, ErrNotFound) {
			// ignore and fall back below
			summary = Summary{}
		}
	}
	if summary.State == "" {
		summary = fallbackSummary(lang)
	}
	storeSummary(lang, summary)
	return cloneSummary(summary), nil
}

// ErrNotFound indicates the status endpoint could not locate resources for the given locale.
var ErrNotFound = errors.New("status: not found")

func (c *Client) fetchRemote(ctx context.Context, lang string) (Summary, error) {
	endpoint := c.baseURL
	if endpoint == "" {
		return Summary{}, ErrNotFound
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Summary{}, err
	}
	req.Header.Set("Accept", "application/json")
	if lang != "" {
		req.Header.Set("Accept-Language", lang)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return Summary{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return Summary{}, ErrNotFound
	}
	if resp.StatusCode >= 400 {
		return Summary{}, fmt.Errorf("status: remote status %d", resp.StatusCode)
	}

	var payload remoteSummary
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Summary{}, err
	}
	return mapRemoteSummary(payload), nil
}

func mapRemoteSummary(raw remoteSummary) Summary {
	summary := Summary{
		State:      strings.TrimSpace(raw.State),
		StateLabel: strings.TrimSpace(raw.StateLabel),
		UpdatedAt:  parseTime(raw.UpdatedAt),
	}
	for _, c := range raw.Components {
		summary.Components = append(summary.Components, Component{
			Name:   strings.TrimSpace(c.Name),
			Status: strings.TrimSpace(c.Status),
		})
	}
	for _, inc := range raw.Incidents {
		item := Incident{
			ID:         strings.TrimSpace(inc.ID),
			Title:      strings.TrimSpace(inc.Title),
			Status:     strings.TrimSpace(inc.Status),
			Impact:     strings.TrimSpace(inc.Impact),
			StartedAt:  parseTime(inc.StartedAt),
			ResolvedAt: parseTime(inc.ResolvedAt),
		}
		for _, upd := range inc.Updates {
			item.Updates = append(item.Updates, IncidentUpdate{
				Timestamp: parseTime(upd.Timestamp),
				Status:    strings.TrimSpace(upd.Status),
				Body:      strings.TrimSpace(upd.Body),
			})
		}
		summary.Incidents = append(summary.Incidents, item)
	}
	return summary
}

type remoteSummary struct {
	State      string            `json:"state"`
	StateLabel string            `json:"state_label"`
	UpdatedAt  string            `json:"updated_at"`
	Components []remoteComponent `json:"components"`
	Incidents  []remoteIncident  `json:"incidents"`
}

type remoteComponent struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type remoteIncident struct {
	ID         string                 `json:"id"`
	Title      string                 `json:"title"`
	Status     string                 `json:"status"`
	Impact     string                 `json:"impact"`
	StartedAt  string                 `json:"started_at"`
	ResolvedAt string                 `json:"resolved_at"`
	Updates    []remoteIncidentUpdate `json:"updates"`
}

type remoteIncidentUpdate struct {
	Timestamp string `json:"timestamp"`
	Status    string `json:"status"`
	Body      string `json:"body"`
}

func cachedSummary(lang string) (Summary, bool) {
	cacheMu.RLock()
	entry, ok := summaryCache[lang]
	cacheMu.RUnlock()
	if !ok || time.Now().After(entry.expires) {
		return Summary{}, false
	}
	return cloneSummary(entry.summary), true
}

func storeSummary(lang string, summary Summary) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	summaryCache[lang] = statusCacheEntry{
		summary: cloneSummary(summary),
		expires: time.Now().Add(cacheTTL),
	}
}

func cloneSummary(src Summary) Summary {
	cp := Summary{
		State:      src.State,
		StateLabel: src.StateLabel,
		UpdatedAt:  src.UpdatedAt,
	}
	if len(src.Components) > 0 {
		cp.Components = make([]Component, len(src.Components))
		copy(cp.Components, src.Components)
	}
	if len(src.Incidents) > 0 {
		cp.Incidents = make([]Incident, len(src.Incidents))
		for i, inc := range src.Incidents {
			cp.Incidents[i] = Incident{
				ID:         inc.ID,
				Title:      inc.Title,
				Status:     inc.Status,
				Impact:     inc.Impact,
				StartedAt:  inc.StartedAt,
				ResolvedAt: inc.ResolvedAt,
			}
			if len(inc.Updates) > 0 {
				cp.Incidents[i].Updates = make([]IncidentUpdate, len(inc.Updates))
				copy(cp.Incidents[i].Updates, inc.Updates)
			}
		}
	}
	return cp
}

func fallbackSummary(lang string) Summary {
	switch lang {
	case "ja":
		return jaFallback
	default:
		return enFallback
	}
}

var enFallback = Summary{
	State:      "operational",
	StateLabel: "All systems operational",
	UpdatedAt:  time.Date(2025, 1, 18, 3, 45, 0, 0, time.UTC),
	Components: []Component{
		{Name: "Web", Status: "operational"},
		{Name: "API", Status: "operational"},
		{Name: "Admin Console", Status: "operational"},
		{Name: "Realtime Notifications", Status: "operational"},
	},
	Incidents: []Incident{
		{
			ID:         "sched-maint-2025-01-15",
			Title:      "Scheduled maintenance: document stamping service",
			Status:     "completed",
			Impact:     "maintenance",
			StartedAt:  time.Date(2025, 1, 15, 2, 0, 0, 0, time.UTC),
			ResolvedAt: time.Date(2025, 1, 15, 3, 30, 0, 0, time.UTC),
			Updates: []IncidentUpdate{
				{
					Timestamp: time.Date(2025, 1, 14, 21, 0, 0, 0, time.UTC),
					Status:    "scheduled",
					Body:      "We will perform routine maintenance on the document stamping service. Downtime is expected to last up to 30 minutes.",
				},
				{
					Timestamp: time.Date(2025, 1, 15, 2, 5, 0, 0, time.UTC),
					Status:    "in_progress",
					Body:      "Maintenance is underway. Users may notice intermittent errors when generating new seals.",
				},
				{
					Timestamp: time.Date(2025, 1, 15, 3, 12, 0, 0, time.UTC),
					Status:    "completed",
					Body:      "Maintenance completed successfully. All systems are back to normal.",
				},
			},
		},
		{
			ID:         "incident-2025-01-10",
			Title:      "Delay in Cloud Storage uploads",
			Status:     "resolved",
			Impact:     "minor",
			StartedAt:  time.Date(2025, 1, 10, 10, 12, 0, 0, time.UTC),
			ResolvedAt: time.Date(2025, 1, 10, 11, 25, 0, 0, time.UTC),
			Updates: []IncidentUpdate{
				{
					Timestamp: time.Date(2025, 1, 10, 10, 20, 0, 0, time.UTC),
					Status:    "investigating",
					Body:      "We are investigating increased latency when uploading assets to Cloud Storage. Existing assets remain accessible.",
				},
				{
					Timestamp: time.Date(2025, 1, 10, 10, 52, 0, 0, time.UTC),
					Status:    "mitigating",
					Body:      "Identified a networking issue within the Tokyo region. We routed traffic to an alternate zone while the provider applies a fix.",
				},
				{
					Timestamp: time.Date(2025, 1, 10, 11, 25, 0, 0, time.UTC),
					Status:    "resolved",
					Body:      "Service is fully restored. Upload latency has returned to normal levels.",
				},
			},
		},
	},
}

var jaFallback = Summary{
	State:      "operational",
	StateLabel: "全サービス正常稼働中",
	UpdatedAt:  time.Date(2025, 1, 18, 12, 45, 0, 0, time.FixedZone("JST", 9*60*60)),
	Components: []Component{
		{Name: "ウェブ", Status: "operational"},
		{Name: "API", Status: "operational"},
		{Name: "管理コンソール", Status: "operational"},
		{Name: "リアルタイム通知", Status: "operational"},
	},
	Incidents: []Incident{
		{
			ID:         "sched-maint-2025-01-15",
			Title:      "定期メンテナンス：ドキュメント押印サービス",
			Status:     "completed",
			Impact:     "maintenance",
			StartedAt:  time.Date(2025, 1, 15, 11, 0, 0, 0, time.FixedZone("JST", 9*60*60)),
			ResolvedAt: time.Date(2025, 1, 15, 12, 30, 0, 0, time.FixedZone("JST", 9*60*60)),
			Updates: []IncidentUpdate{
				{
					Timestamp: time.Date(2025, 1, 14, 18, 0, 0, 0, time.FixedZone("JST", 9*60*60)),
					Status:    "scheduled",
					Body:      "ドキュメント押印サービスの定期メンテナンスを実施します。最大30分ほど断続的な停止が発生する見込みです。",
				},
				{
					Timestamp: time.Date(2025, 1, 15, 11, 5, 0, 0, time.FixedZone("JST", 9*60*60)),
					Status:    "in_progress",
					Body:      "メンテナンス作業を開始しました。新規の印影生成がしづらい状態になる場合があります。",
				},
				{
					Timestamp: time.Date(2025, 1, 15, 12, 12, 0, 0, time.FixedZone("JST", 9*60*60)),
					Status:    "completed",
					Body:      "メンテナンスが完了しました。現在は通常どおり利用できます。",
				},
			},
		},
		{
			ID:         "incident-2025-01-10",
			Title:      "クラウドストレージへのアップロード遅延",
			Status:     "resolved",
			Impact:     "minor",
			StartedAt:  time.Date(2025, 1, 10, 19, 12, 0, 0, time.FixedZone("JST", 9*60*60)),
			ResolvedAt: time.Date(2025, 1, 10, 20, 25, 0, 0, time.FixedZone("JST", 9*60*60)),
			Updates: []IncidentUpdate{
				{
					Timestamp: time.Date(2025, 1, 10, 19, 20, 0, 0, time.FixedZone("JST", 9*60*60)),
					Status:    "investigating",
					Body:      "クラウドストレージへのアップロード遅延を調査しています。既存のファイル閲覧には影響ありません。",
				},
				{
					Timestamp: time.Date(2025, 1, 10, 19, 52, 0, 0, time.FixedZone("JST", 9*60*60)),
					Status:    "mitigating",
					Body:      "東京リージョンのネットワーク遅延を特定し、プロバイダの修正を待つ間は別ゾーンへ迂回させています。",
				},
				{
					Timestamp: time.Date(2025, 1, 10, 20, 25, 0, 0, time.FixedZone("JST", 9*60*60)),
					Status:    "resolved",
					Body:      "サービスは復旧しました。アップロード時間は平常値に戻っています。",
				},
			},
		},
	},
}

func parseTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t
		}
	}
	return time.Time{}
}

func normalizeLang(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" {
		return "ja"
	}
	switch lang {
	case "ja", "en":
		return lang
	}
	if strings.HasPrefix(lang, "ja") {
		return "ja"
	}
	return "en"
}
