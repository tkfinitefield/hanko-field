package profile

import (
	"time"

	"finitefield.org/hanko-admin/internal/admin/profile"
	"finitefield.org/hanko-admin/internal/admin/templates/helpers"
	"finitefield.org/hanko-admin/internal/admin/templates/partials"
)

// PageData drives the main profile page.
type PageData struct {
	UserEmail string
	UserName  string
	Security  *profile.SecurityState
	CSRFToken string
	Flash     string
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
