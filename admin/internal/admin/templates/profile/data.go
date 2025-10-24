package profile

import (
	"sort"
	"strings"
	"time"

	"finitefield.org/hanko-admin/internal/admin/profile"
	"finitefield.org/hanko-admin/internal/admin/templates/helpers"
	"finitefield.org/hanko-admin/internal/admin/templates/partials"
)

// PageData drives the main profile page.
type PageData struct {
	UserEmail    string
	UserName     string
	DisplayName  string
	Roles        []string
	LastLogin    *time.Time
	Security     *profile.SecurityState
	FeatureFlags []FeatureFlagEntry
	ActiveTab    string
	CSRFToken    string
	Flash        string
}

// MFAUpdateData renders updated MFA section + flash message.
type MFAUpdateData struct {
	Security  *profile.SecurityState
	CSRFToken string
	Message   string
}

// APIKeyUpdateData refreshes API key section and optionally shows secret modal.
type APIKeyUpdateData struct {
	Security  *profile.SecurityState
	Secret    *profile.APIKeySecret
	CSRFToken string
	Message   string
}

// SessionUpdateData refreshes sessions table.
type SessionUpdateData struct {
	Security  *profile.SecurityState
	CSRFToken string
	Message   string
}

// TOTPModalData holds data for authenticator enrollment modal.
type TOTPModalData struct {
	Enrollment *profile.TOTPEnrollment
	CSRFToken  string
	Error      string
}

// APIKeyFormData renders the new API key form modal.
type APIKeyFormData struct {
	CSRFToken string
	Error     string
	Label     string
}

// AlertContent describes a static callout card.
type AlertContent struct {
	Title    string
	Body     string
	LinkHref string
	LinkText string
}

// FeatureFlagEntry captures the enabled/disabled state for a named feature flag.
type FeatureFlagEntry struct {
	Key     string
	Enabled bool
}

func breadcrumbItems() []partials.Breadcrumb {
	return []partials.Breadcrumb{
		{Label: "プロフィール"},
	}
}

func securityAlerts() []AlertContent {
	return []AlertContent{
		{
			Title:    "セキュリティベストプラクティス",
			Body:     "Authenticator アプリとメール MFA を併用し、復旧コードは必ず安全な場所に保管してください。",
			LinkHref: "https://cloud.google.com/security/best-practices-for-enterprises",
			LinkText: "ガイドを開く",
		},
		{
			Title:    "API キー権限",
			Body:     "API キーは最小権限で発行し、定期的にローテーションを行ってください。",
			LinkHref: "https://firebase.google.com/docs/projects/api-keys#best-practices",
			LinkText: "API キー運用",
		},
	}
}

// FeatureFlagsFromMap converts a feature flag map into a deterministic slice for rendering.
func FeatureFlagsFromMap(flags map[string]bool) []FeatureFlagEntry {
	if len(flags) == 0 {
		return nil
	}
	entries := make([]FeatureFlagEntry, 0, len(flags))
	for key, enabled := range flags {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		entries = append(entries, FeatureFlagEntry{Key: trimmed, Enabled: enabled})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Key < entries[j].Key
	})
	return entries
}

// MostRecentSessionAt returns the most recent session timestamp from the security state.
func MostRecentSessionAt(state *profile.SecurityState) *time.Time {
	if state == nil {
		return nil
	}
	var latest *time.Time
	for _, session := range state.Sessions {
		ts := session.LastSeenAt
		if ts.IsZero() {
			ts = session.CreatedAt
		}
		if ts.IsZero() {
			continue
		}
		candidate := ts
		if latest == nil || candidate.After(*latest) {
			latest = &candidate
		}
	}
	return latest
}

// AvatarInitial derives the initial used for avatar placeholders.
func AvatarInitial(name, email, fallback string) string {
	candidate := strings.TrimSpace(name)
	if candidate == "" {
		candidate = strings.TrimSpace(email)
	}
	if candidate == "" {
		candidate = strings.TrimSpace(fallback)
	}
	if candidate == "" {
		return "?"
	}
	runes := []rune(strings.ToUpper(candidate))
	return string(runes[0])
}

func formatTimestamp(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return helpers.Relative(t)
}

func hasMFAMethod(state *profile.SecurityState, kind profile.MFAMethodKind) bool {
	if state == nil {
		return false
	}
	for _, method := range state.MFA.Methods {
		if method.Kind == kind {
			return true
		}
	}
	return false
}
