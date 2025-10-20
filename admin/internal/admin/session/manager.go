package session

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/securecookie"
)

const (
	defaultCookieName       = "admin_session"
	defaultCookiePath       = "/"
	defaultLifetime         = 12 * time.Hour
	defaultRememberLifetime = 30 * 24 * time.Hour
	defaultIdleTimeout      = 30 * time.Minute
)

// ErrExpired indicates the stored session is no longer valid due to idle or absolute expiry.
var ErrExpired = errors.New("session expired")

// ErrInvalidConfig indicates the manager was initialised with missing or invalid options.
var ErrInvalidConfig = errors.New("session: invalid config")

// User captures authenticated staff profile details persisted in the session.
type User struct {
	UID   string   `json:"uid"`
	Email string   `json:"email,omitempty"`
	Roles []string `json:"roles,omitempty"`
}

// Data represents the full persisted session payload.
type Data struct {
	ID           string          `json:"id"`
	CreatedAt    time.Time       `json:"createdAt"`
	LastActive   time.Time       `json:"lastActive"`
	ExpiresAt    time.Time       `json:"expiresAt,omitempty"`
	RememberMe   bool            `json:"rememberMe"`
	CSRFToken    string          `json:"csrfToken,omitempty"`
	User         *User           `json:"user,omitempty"`
	FeatureFlags map[string]bool `json:"featureFlags,omitempty"`
	RefreshToken string          `json:"refreshToken,omitempty"`
}

// Session holds mutable state for the current request lifecycle.
type Session struct {
	data      Data
	dirty     bool
	destroyed bool
	cfg       *Config
}

// Config controls cookie encoding and lifecycle limits for the session manager.
type Config struct {
	CookieName     string
	HashKey        []byte
	BlockKey       []byte
	CookiePath     string
	CookieDomain   string
	CookieSecure   bool
	CookieHTTPOnly *bool
	CookieSameSite http.SameSite

	IdleTimeout      time.Duration
	Lifetime         time.Duration
	RememberLifetime time.Duration
	Now              func() time.Time
}

// Manager decodes and persists session state via signed (and optionally encrypted) cookies.
type Manager struct {
	cfg      Config
	codec    *securecookie.SecureCookie
	now      func() time.Time
	httpOnly bool
}

// NewManager constructs a Manager using the provided configuration.
func NewManager(cfg Config) (*Manager, error) {
	if len(cfg.HashKey) == 0 {
		return nil, fmt.Errorf("%w: hash key is required", ErrInvalidConfig)
	}

	if cfg.CookieName == "" {
		cfg.CookieName = defaultCookieName
	}
	if cfg.CookiePath == "" {
		cfg.CookiePath = defaultCookiePath
	}
	if cfg.Lifetime <= 0 {
		cfg.Lifetime = defaultLifetime
	}
	if cfg.RememberLifetime <= 0 {
		cfg.RememberLifetime = defaultRememberLifetime
	}
	if cfg.IdleTimeout <= 0 {
		cfg.IdleTimeout = defaultIdleTimeout
	}
	if cfg.CookieSameSite == http.SameSiteDefaultMode {
		cfg.CookieSameSite = http.SameSiteLaxMode
	}
	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = time.Now
	}

	codec := securecookie.New(cfg.HashKey, cfg.BlockKey)
	codec.SetSerializer(securecookie.JSONEncoder{})

	httpOnly := true
	if cfg.CookieHTTPOnly != nil {
		httpOnly = *cfg.CookieHTTPOnly
	}

	mgr := &Manager{
		cfg:      cfg,
		codec:    codec,
		now:      nowFn,
		httpOnly: httpOnly,
	}
	return mgr, nil
}

// Load retrieves the session from the incoming request or creates a new one.
func (m *Manager) Load(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie(m.cfg.CookieName)
	if err != nil {
		return m.newSession(m.now()), nil
	}

	var stored Data
	if err := m.codec.Decode(m.cfg.CookieName, cookie.Value, &stored); err != nil {
		return m.newSession(m.now()), nil
	}

	sess := m.sessionFromData(stored)
	if expired := m.isExpired(sess, m.now()); expired {
		return nil, ErrExpired
	}
	return sess, nil
}

// Save writes the session back to the response as a cookie. Destroyed sessions clear the cookie.
func (m *Manager) Save(w http.ResponseWriter, sess *Session) error {
	if sess == nil {
		return errors.New("session: nil session")
	}

	if sess.destroyed {
		http.SetCookie(w, m.expiredCookie())
		return nil
	}

	// Mark the session as accessed for this request.
	sess.Touch(m.now())

	data := sess.snapshot()

	encoded, err := m.codec.Encode(m.cfg.CookieName, data)
	if err != nil {
		return fmt.Errorf("encode session: %w", err)
	}

	cookie := &http.Cookie{
		Name:     m.cfg.CookieName,
		Value:    encoded,
		Path:     m.cfg.CookiePath,
		Domain:   m.cfg.CookieDomain,
		Secure:   m.cfg.CookieSecure,
		HttpOnly: m.httpOnly,
		SameSite: m.cfg.CookieSameSite,
	}

	if !data.ExpiresAt.IsZero() {
		expiry := data.ExpiresAt.UTC()
		cookie.Expires = expiry
		remaining := expiry.Sub(m.now())
		if remaining <= 0 {
			cookie.MaxAge = -1
		} else {
			cookie.MaxAge = int(remaining.Round(time.Second).Seconds())
		}
	}

	http.SetCookie(w, cookie)
	return nil
}

// Destroy invalidates the session cookie immediately.
func (m *Manager) Destroy(w http.ResponseWriter) {
	http.SetCookie(w, m.expiredCookie())
}

// newSession creates a pristine session with generated identifiers.
func (m *Manager) newSession(now time.Time) *Session {
	id := mustGenerateToken(32)
	data := Data{
		ID:           id,
		CreatedAt:    now.UTC(),
		LastActive:   now.UTC(),
		FeatureFlags: make(map[string]bool),
	}
	data.ExpiresAt = m.cfg.computeExpiry(now, false)

	return &Session{
		data:  data,
		dirty: true,
		cfg:   &m.cfg,
	}
}

// New returns a new empty session instance using the manager configuration.
func (m *Manager) New() *Session {
	return m.newSession(m.now())
}

func (m *Manager) sessionFromData(d Data) *Session {
	if d.FeatureFlags == nil {
		d.FeatureFlags = make(map[string]bool)
	}
	if d.ID == "" {
		d.ID = mustGenerateToken(32)
		d.CreatedAt = m.now().UTC()
		d.LastActive = d.CreatedAt
		d.ExpiresAt = m.cfg.computeExpiry(d.CreatedAt, d.RememberMe)
	}
	return &Session{
		data: d,
		cfg:  &m.cfg,
	}
}

func (m *Manager) isExpired(sess *Session, now time.Time) bool {
	if sess == nil {
		return true
	}
	now = now.UTC()

	if !sess.data.ExpiresAt.IsZero() && now.After(sess.data.ExpiresAt.UTC()) {
		return true
	}

	if m.cfg.IdleTimeout > 0 {
		last := sess.data.LastActive
		if last.IsZero() {
			last = sess.data.CreatedAt
		}
		if !last.IsZero() && now.Sub(last) > m.cfg.IdleTimeout {
			return true
		}
	}
	return false
}

func (m *Manager) expiredCookie() *http.Cookie {
	return &http.Cookie{
		Name:     m.cfg.CookieName,
		Value:    "",
		Path:     m.cfg.CookiePath,
		Domain:   m.cfg.CookieDomain,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		Secure:   m.cfg.CookieSecure,
		HttpOnly: m.httpOnly,
		SameSite: m.cfg.CookieSameSite,
	}
}

// ID returns the stable session identifier.
func (s *Session) ID() string {
	return s.data.ID
}

// CreatedAt returns the session creation timestamp.
func (s *Session) CreatedAt() time.Time {
	return s.data.CreatedAt
}

// LastActive returns the last access timestamp.
func (s *Session) LastActive() time.Time {
	return s.data.LastActive
}

// ExpiresAt returns the absolute expiry timestamp for the session.
func (s *Session) ExpiresAt() time.Time {
	return s.data.ExpiresAt
}

// RememberMe indicates whether the session should persist beyond the default lifetime.
func (s *Session) RememberMe() bool {
	return s.data.RememberMe
}

// SetRememberMe toggles the remember-me state and adjusts expiry accordingly.
func (s *Session) SetRememberMe(remember bool) {
	if s.data.RememberMe == remember {
		return
	}
	s.data.RememberMe = remember
	s.data.ExpiresAt = s.cfg.computeExpiry(s.data.CreatedAt, remember)
	s.dirty = true
}

// EnsureCSRFToken returns the existing CSRF token or generates a new one on demand.
func (s *Session) EnsureCSRFToken() (string, error) {
	if s.data.CSRFToken != "" {
		return s.data.CSRFToken, nil
	}
	token, err := generateToken(32)
	if err != nil {
		return "", err
	}
	s.data.CSRFToken = token
	s.dirty = true
	return token, nil
}

// SetCSRFToken explicitly sets the CSRF token value.
func (s *Session) SetCSRFToken(token string) {
	if token == "" {
		return
	}
	if s.data.CSRFToken == token {
		return
	}
	s.data.CSRFToken = token
	s.dirty = true
}

// CSRFToken returns the stored CSRF token value.
func (s *Session) CSRFToken() string {
	return s.data.CSRFToken
}

// User returns the persisted user profile, if present.
func (s *Session) User() *User {
	return s.data.User
}

// SetUser updates the session user profile.
func (s *Session) SetUser(user *User) {
	// Avoid marking dirty when nothing changes.
	if equalUsers(s.data.User, user) {
		return
	}
	if user == nil {
		s.data.User = nil
		s.dirty = true
		return
	}

	copied := *user
	if copied.Roles != nil {
		copied.Roles = append([]string(nil), copied.Roles...)
	}
	s.data.User = &copied
	s.dirty = true
}

// FeatureFlags returns a copy of the stored feature flags map.
func (s *Session) FeatureFlags() map[string]bool {
	result := make(map[string]bool, len(s.data.FeatureFlags))
	for k, v := range s.data.FeatureFlags {
		result[k] = v
	}
	return result
}

// SetFeatureFlag updates a single feature flag value.
func (s *Session) SetFeatureFlag(name string, enabled bool) {
	if s.data.FeatureFlags == nil {
		s.data.FeatureFlags = make(map[string]bool)
	}
	if current, ok := s.data.FeatureFlags[name]; ok && current == enabled {
		return
	}
	s.data.FeatureFlags[name] = enabled
	s.dirty = true
}

// SetFeatureFlags replaces the stored feature flag map.
func (s *Session) SetFeatureFlags(flags map[string]bool) {
	newMap := make(map[string]bool, len(flags))
	for key, val := range flags {
		newMap[key] = val
	}
	if mapsEqual(s.data.FeatureFlags, newMap) {
		return
	}
	s.data.FeatureFlags = newMap
	s.dirty = true
}

// RefreshToken returns the stored refresh token (if any).
func (s *Session) RefreshToken() string {
	return s.data.RefreshToken
}

// SetRefreshToken updates the stored refresh token for remember-me support.
func (s *Session) SetRefreshToken(token string) {
	if s.data.RefreshToken == token {
		return
	}
	s.data.RefreshToken = token
	s.dirty = true
}

// Destroy marks the session for deletion at the end of the request.
func (s *Session) Destroy() {
	s.destroyed = true
	s.dirty = true
}

// Destroyed exposes the destroy marker.
func (s *Session) Destroyed() bool {
	return s.destroyed
}

// Touch updates the last active timestamp.
func (s *Session) Touch(now time.Time) {
	now = now.UTC()
	if now.After(s.data.LastActive) {
		s.data.LastActive = now
		s.dirty = true
	}
}

// Dirty indicates whether the session contents have changed during this request.
func (s *Session) Dirty() bool {
	return s.dirty
}

func (s *Session) snapshot() Data {
	if s.data.FeatureFlags == nil {
		s.data.FeatureFlags = make(map[string]bool)
	}
	return s.data
}

func (cfg *Config) computeExpiry(from time.Time, remember bool) time.Time {
	if cfg == nil {
		return time.Time{}
	}
	from = from.UTC()
	lifetime := cfg.Lifetime
	if remember {
		lifetime = cfg.RememberLifetime
		if lifetime <= 0 {
			lifetime = cfg.Lifetime
		}
	}
	if lifetime <= 0 {
		return time.Time{}
	}
	return from.Add(lifetime).UTC()
}

func equalUsers(a, b *User) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.UID != b.UID || a.Email != b.Email {
		return false
	}
	if len(a.Roles) != len(b.Roles) {
		return false
	}
	for i := range a.Roles {
		if a.Roles[i] != b.Roles[i] {
			return false
		}
	}
	return true
}

func mapsEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func mustGenerateToken(length int) string {
	token, err := generateToken(length)
	if err != nil {
		panic(err)
	}
	return token
}

func generateToken(length int) (string, error) {
	if length <= 0 {
		length = 32
	}
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
