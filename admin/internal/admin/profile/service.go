package profile

import (
	"context"
	"errors"
	"time"
)

// ErrNotConfigured indicates that the profile service dependency has not been wired.
var ErrNotConfigured = errors.New("profile service not configured")

// Service exposes security-related operations for the currently authenticated staff user.
type Service interface {
	// SecurityOverview retrieves MFA status, API keys, and active sessions for the caller.
	SecurityOverview(ctx context.Context, token string) (*SecurityState, error)
	// StartTOTPEnrollment begins a new TOTP enrollment flow and returns secrets to display.
	StartTOTPEnrollment(ctx context.Context, token string) (*TOTPEnrollment, error)
	// ConfirmTOTPEnrollment finishes the enrollment by validating the provided OTP code.
	ConfirmTOTPEnrollment(ctx context.Context, token, code string) (*SecurityState, error)
	// EnableEmailMFA enables email-based MFA for the caller.
	EnableEmailMFA(ctx context.Context, token string) (*SecurityState, error)
	// DisableMFA disables all MFA factors for the caller.
	DisableMFA(ctx context.Context, token string) (*SecurityState, error)
	// CreateAPIKey issues a new API key tied to the caller.
	CreateAPIKey(ctx context.Context, token string, req CreateAPIKeyRequest) (*APIKeySecret, error)
	// RevokeAPIKey revokes the API key identified by keyID.
	RevokeAPIKey(ctx context.Context, token, keyID string) (*SecurityState, error)
	// RevokeSession terminates the provided session.
	RevokeSession(ctx context.Context, token, sessionID string) (*SecurityState, error)
}

// SecurityState contains the data required to render the profile security dashboard.
type SecurityState struct {
	UserEmail string    `json:"userEmail"`
	UserName  string    `json:"userName"`
	Phone     string    `json:"phone"`
	MFA       MFAState  `json:"mfa"`
	APIKeys   []APIKey  `json:"apiKeys"`
	Sessions  []Session `json:"sessions"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// MFAState captures MFA configuration for the user.
type MFAState struct {
	Enabled         bool               `json:"enabled"`
	PrimaryMethod   MFAMethodKind      `json:"primaryMethod"`
	Methods         []MFAMethod        `json:"methods"`
	RecoveryCodes   []string           `json:"recoveryCodes"`
	LastConfirmed   *time.Time         `json:"lastConfirmed"`
	PendingEnroll   *PendingEnrollment `json:"pending"`
	TOTPEnforced    bool               `json:"totpEnforced"`
	EmailEnforced   bool               `json:"emailEnforced"`
	CanDisableMFA   bool               `json:"canDisable"`
	CanAddNewFactor bool               `json:"canAddNew"`
}

// MFAMethod represents an enrolled MFA factor.
type MFAMethod struct {
	ID         string        `json:"id"`
	Kind       MFAMethodKind `json:"kind"`
	Label      string        `json:"label"`
	CreatedAt  time.Time     `json:"createdAt"`
	LastUsedAt *time.Time    `json:"lastUsedAt"`
	Verified   bool          `json:"verified"`
	Default    bool          `json:"default"`
}

// PendingEnrollment signals that the backend requires the user to finish an outstanding MFA setup.
type PendingEnrollment struct {
	Kind   MFAMethodKind `json:"kind"`
	Issued time.Time     `json:"issuedAt"`
}

// MFAMethodKind enumerates supported MFA factor types.
type MFAMethodKind string

const (
	// MFAMethodTOTP represents authenticator app / time-based one-time password.
	MFAMethodTOTP MFAMethodKind = "totp"
	// MFAMethodEmail represents email one-time password.
	MFAMethodEmail MFAMethodKind = "email"
)

// TOTPEnrollment returns artifacts for a new authenticator app enrollment.
type TOTPEnrollment struct {
	Issuer        string     `json:"issuer"`
	AccountName   string     `json:"account"`
	Secret        string     `json:"secret"`
	URI           string     `json:"otpauthUrl"`
	QRCodePNG     string     `json:"qrCode"`
	RecoveryCodes []string   `json:"recoveryCodes"`
	ExpiresAt     *time.Time `json:"expiresAt"`
}

// CreateAPIKeyRequest describes an API key creation payload.
type CreateAPIKeyRequest struct {
	Label     string     `json:"label"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	Roles     []string   `json:"roles,omitempty"`
	Scopes    []string   `json:"scopes,omitempty"`
}

// APIKeySecret contains details of a freshly issued API key, including the raw secret.
type APIKeySecret struct {
	ID        string     `json:"id"`
	Label     string     `json:"label"`
	Secret    string     `json:"secret"`
	CreatedAt time.Time  `json:"createdAt"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

// APIKeyStatus captures the lifecycle state for an API key.
type APIKeyStatus string

const (
	APIKeyStatusActive  APIKeyStatus = "active"
	APIKeyStatusRevoked APIKeyStatus = "revoked"
	APIKeyStatusExpired APIKeyStatus = "expired"
)

// APIKey summarizes an issued API key (without the secret).
type APIKey struct {
	ID        string       `json:"id"`
	Label     string       `json:"label"`
	Status    APIKeyStatus `json:"status"`
	CreatedAt time.Time    `json:"createdAt"`
	LastUsed  *time.Time   `json:"lastUsedAt"`
	ExpiresAt *time.Time   `json:"expiresAt"`
}

// Session represents an authenticated admin session.
type Session struct {
	ID         string    `json:"id"`
	UserAgent  string    `json:"userAgent"`
	IPAddress  string    `json:"ipAddress"`
	Location   string    `json:"location"`
	CreatedAt  time.Time `json:"createdAt"`
	LastSeenAt time.Time `json:"lastSeenAt"`
	Current    bool      `json:"current"`
}
