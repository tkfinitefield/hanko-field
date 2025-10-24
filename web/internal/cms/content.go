package cms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// ContentPage represents a localized static page sourced from the CMS or local markdown.
type ContentPage struct {
	Kind          string
	Slug          string
	Lang          string
	Title         string
	Summary       string
	Body          string
	Format        string // "markdown" (default) or "html"
	EffectiveDate time.Time
	UpdatedAt     time.Time
	Version       string
	DownloadLabel string
	DownloadURL   string
	SourceURL     string
	Icon          string
	Banner        *ContentBanner
	SEO           ContentSEO
}

// ContentSEO holds optional metadata overrides for static pages.
type ContentSEO struct {
	Title       string
	Description string
	OGImage     string
}

// ContentBanner models an optional banner/alert displayed above the body.
type ContentBanner struct {
	Variant  string
	Title    string
	Message  string
	LinkText string
	LinkURL  string
}

type contentFrontMatter struct {
	Title         string                    `yaml:"title"`
	Summary       string                    `yaml:"summary"`
	Lang          string                    `yaml:"lang"`
	Format        string                    `yaml:"format"`
	EffectiveDate string                    `yaml:"effective_date"`
	UpdatedAt     string                    `yaml:"updated_at"`
	Version       string                    `yaml:"version"`
	DownloadLabel string                    `yaml:"download_label"`
	DownloadURL   string                    `yaml:"download_url"`
	SourceURL     string                    `yaml:"source_url"`
	Icon          string                    `yaml:"icon"`
	SEO           contentFrontMatterSEO     `yaml:"seo"`
	Banner        *contentFrontMatterBanner `yaml:"banner"`
}

type contentFrontMatterSEO struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	OGImage     string `yaml:"og_image"`
}

type contentFrontMatterBanner struct {
	Variant  string `yaml:"variant"`
	Title    string `yaml:"title"`
	Message  string `yaml:"message"`
	LinkText string `yaml:"link_text"`
	LinkURL  string `yaml:"link_url"`
}

const (
	defaultContentFormat = "markdown"
	defaultContentDir    = "content"
)

var (
	contentCache = struct {
		mu    sync.RWMutex
		items map[string]contentCacheEntry
	}{
		items: map[string]contentCacheEntry{},
	}
	contentCacheTTL = time.Minute * 5
)

type contentCacheEntry struct {
	page    ContentPage
	expires time.Time
}

// SetContentCacheDuration allows overriding the in-memory cache duration (primarily for tests).
func SetContentCacheDuration(d time.Duration) {
	if d <= 0 {
		d = time.Minute
	}
	contentCacheTTL = d
}

// SetContentDir configures the fallback directory for markdown pages.
func (c *Client) SetContentDir(dir string) {
	if c == nil {
		return
	}
	dir = strings.TrimSpace(dir)
	if dir == "" {
		dir = defaultContentDir
	}
	c.contentDir = dir
}

// ContentDir returns the configured fallback directory.
func (c *Client) ContentDir() string {
	if c == nil || strings.TrimSpace(c.contentDir) == "" {
		return defaultContentDir
	}
	return c.contentDir
}

// GetContentPage fetches a localized static page, consulting the remote CMS when configured,
// otherwise falling back to local markdown.
func (c *Client) GetContentPage(ctx context.Context, kind, slug, lang string) (ContentPage, error) {
	kind = strings.TrimSpace(strings.ToLower(kind))
	if kind == "" {
		kind = "content"
	}
	slug = sanitizeSlug(slug)
	if slug == "" {
		return ContentPage{}, ErrNotFound
	}
	lang = normalizeLang(lang)

	cacheKey := strings.Join([]string{kind, lang, slug}, "|")
	if page, ok := cachedContent(cacheKey); ok {
		return cloneContentPage(page), nil
	}

	page, err := c.fetchContentPage(ctx, kind, slug, lang)
	if err != nil {
		return ContentPage{}, err
	}
	storeContent(cacheKey, page)
	return cloneContentPage(page), nil
}

func (c *Client) fetchContentPage(ctx context.Context, kind, slug, lang string) (ContentPage, error) {
	// Remote fetch (optional)
	if c != nil && c.baseURL != "" {
		if page, err := c.fetchContentPageRemote(ctx, kind, slug, lang); err == nil {
			return page, nil
		} else if !errors.Is(err, ErrNotFound) {
			// Log and continue to fallback
			// logging lives here to avoid import cycle with main; it's acceptable for cms to stay quiet.
		}
	}
	return fallbackContentPage(c.ContentDir(), kind, slug, lang)
}

func (c *Client) fetchContentPageRemote(ctx context.Context, kind, slug, lang string) (ContentPage, error) {
	base := strings.TrimSpace(c.baseURL)
	endpoint, err := url.JoinPath(strings.TrimRight(base, "/"), "content", kind, slug)
	if err != nil {
		return ContentPage{}, err
	}
	if c.http == nil {
		c.http = &http.Client{Timeout: 5 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ContentPage{}, err
	}
	q := req.URL.Query()
	if lang != "" {
		q.Set("lang", lang)
	}
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return ContentPage{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return ContentPage{}, ErrNotFound
	}
	if resp.StatusCode >= 400 {
		return ContentPage{}, fmt.Errorf("cms: content remote status %d", resp.StatusCode)
	}

	var payload struct {
		Kind          string    `json:"kind"`
		Slug          string    `json:"slug"`
		Lang          string    `json:"lang"`
		Title         string    `json:"title"`
		Summary       string    `json:"summary"`
		Body          string    `json:"body"`
		Format        string    `json:"format"`
		EffectiveDate time.Time `json:"effective_date"`
		UpdatedAt     time.Time `json:"updated_at"`
		Version       string    `json:"version"`
		DownloadLabel string    `json:"download_label"`
		DownloadURL   string    `json:"download_url"`
		SourceURL     string    `json:"source_url"`
		Icon          string    `json:"icon"`
		SEO           struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			OGImage     string `json:"og_image"`
		} `json:"seo"`
		Banner struct {
			Variant  string `json:"variant"`
			Title    string `json:"title"`
			Message  string `json:"message"`
			LinkText string `json:"link_text"`
			LinkURL  string `json:"link_url"`
		} `json:"banner"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return ContentPage{}, err
	}
	if strings.TrimSpace(payload.Body) == "" {
		return ContentPage{}, fmt.Errorf("cms: empty body for %s/%s", kind, slug)
	}
	page := ContentPage{
		Kind:          firstNonEmpty(payload.Kind, kind),
		Slug:          firstNonEmpty(payload.Slug, slug),
		Lang:          firstNonEmpty(payload.Lang, lang),
		Title:         payload.Title,
		Summary:       payload.Summary,
		Body:          payload.Body,
		Format:        firstNonEmpty(payload.Format, defaultContentFormat),
		EffectiveDate: payload.EffectiveDate,
		UpdatedAt:     payload.UpdatedAt,
		Version:       payload.Version,
		DownloadLabel: payload.DownloadLabel,
		DownloadURL:   payload.DownloadURL,
		SourceURL:     payload.SourceURL,
		Icon:          payload.Icon,
		SEO: ContentSEO{
			Title:       payload.SEO.Title,
			Description: payload.SEO.Description,
			OGImage:     payload.SEO.OGImage,
		},
	}
	if payload.Banner.Title != "" || payload.Banner.Message != "" {
		page.Banner = &ContentBanner{
			Variant:  payload.Banner.Variant,
			Title:    payload.Banner.Title,
			Message:  payload.Banner.Message,
			LinkText: payload.Banner.LinkText,
			LinkURL:  payload.Banner.LinkURL,
		}
	}
	return page, nil
}

func fallbackContentPage(contentDir, kind, slug, lang string) (ContentPage, error) {
	if strings.TrimSpace(contentDir) == "" {
		contentDir = defaultContentDir
	}
	priority := []string{lang}
	if lang != "en" {
		priority = append(priority, "en")
	}
	if lang != "ja" {
		priority = append(priority, "ja")
	}
	for _, candidate := range priority {
		page, err := readContentMarkdown(contentDir, kind, slug, candidate)
		if err == nil {
			return page, nil
		}
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if errors.Is(err, ErrNotFound) {
			continue
		}
		// For other errors (parse issues), stop early.
		return ContentPage{}, err
	}
	return ContentPage{}, ErrNotFound
}

func readContentMarkdown(contentDir, kind, slug, lang string) (ContentPage, error) {
	if slug == "" {
		return ContentPage{}, ErrNotFound
	}
	segments := []string{contentDir, kind, lang}
	path := filepath.Join(segments...)
	file := filepath.Join(path, slug+".md")

	data, err := os.ReadFile(file)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ContentPage{}, ErrNotFound
		}
		return ContentPage{}, err
	}
	info, statErr := os.Stat(file)
	if statErr != nil {
		info = nil
	}
	fm, body := splitFrontMatter(string(data))
	front := contentFrontMatter{}
	if strings.TrimSpace(fm) != "" {
		if err := yaml.Unmarshal([]byte(fm), &front); err != nil {
			return ContentPage{}, fmt.Errorf("cms: parse front matter %s: %w", file, err)
		}
	}
	page := ContentPage{
		Kind:          kind,
		Slug:          slug,
		Lang:          firstNonEmpty(strings.TrimSpace(front.Lang), lang),
		Title:         strings.TrimSpace(front.Title),
		Summary:       strings.TrimSpace(front.Summary),
		Body:          body,
		Format:        strings.TrimSpace(front.Format),
		Version:       strings.TrimSpace(front.Version),
		DownloadLabel: strings.TrimSpace(front.DownloadLabel),
		DownloadURL:   strings.TrimSpace(front.DownloadURL),
		SourceURL:     strings.TrimSpace(front.SourceURL),
		Icon:          strings.TrimSpace(front.Icon),
		SEO: ContentSEO{
			Title:       strings.TrimSpace(front.SEO.Title),
			Description: strings.TrimSpace(front.SEO.Description),
			OGImage:     strings.TrimSpace(front.SEO.OGImage),
		},
	}
	if page.Format == "" {
		page.Format = defaultContentFormat
	}
	if front.Banner != nil {
		page.Banner = &ContentBanner{
			Variant:  strings.TrimSpace(front.Banner.Variant),
			Title:    strings.TrimSpace(front.Banner.Title),
			Message:  strings.TrimSpace(front.Banner.Message),
			LinkText: strings.TrimSpace(front.Banner.LinkText),
			LinkURL:  strings.TrimSpace(front.Banner.LinkURL),
		}
	}
	page.EffectiveDate = parseContentDate(front.EffectiveDate)
	page.UpdatedAt = parseContentDate(front.UpdatedAt)
	if page.UpdatedAt.IsZero() && info != nil {
		page.UpdatedAt = info.ModTime()
	}
	if page.Title == "" {
		// fall back to slug prettified
		page.Title = prettifySlug(slug)
	}
	return page, nil
}

func splitFrontMatter(input string) (string, string) {
	input = strings.TrimLeft(input, "\ufeff")
	lines := strings.Split(input, "\n")
	if len(lines) == 0 {
		return "", ""
	}
	if strings.TrimSpace(lines[0]) != "---" {
		return "", input
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			fm := strings.Join(lines[1:i], "\n")
			body := strings.Join(lines[i+1:], "\n")
			return fm, strings.TrimLeft(body, "\n\r")
		}
	}
	return "", input
}

func parseContentDate(v string) time.Time {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02",
		"2006/01/02",
		"2006-1-2",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, v); err == nil {
			return t
		}
	}
	return time.Time{}
}

func prettifySlug(slug string) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return slug
	}
	parts := strings.Split(slug, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		runes[0] = asciiUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}

func sanitizeSlug(slug string) string {
	slug = strings.TrimSpace(strings.ToLower(slug))
	slug = strings.Trim(slug, "/")
	if slug == "" {
		return ""
	}
	if strings.Contains(slug, "..") {
		return ""
	}
	if strings.ContainsRune(slug, os.PathSeparator) {
		return ""
	}
	return slug
}

func cachedContent(key string) (ContentPage, bool) {
	now := time.Now()
	contentCache.mu.RLock()
	entry, ok := contentCache.items[key]
	contentCache.mu.RUnlock()
	if !ok || now.After(entry.expires) {
		return ContentPage{}, false
	}
	return cloneContentPage(entry.page), true
}

func storeContent(key string, page ContentPage) {
	contentCache.mu.Lock()
	defer contentCache.mu.Unlock()
	entry := contentCacheEntry{
		page:    cloneContentPage(page),
		expires: time.Now().Add(contentCacheTTL),
	}
	contentCache.items[key] = entry
}

func cloneContentPage(src ContentPage) ContentPage {
	cp := src
	if src.Banner != nil {
		b := *src.Banner
		cp.Banner = &b
	}
	return cp
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func asciiUpper(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - ('a' - 'A')
	}
	return r
}
