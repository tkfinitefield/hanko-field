package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const sessionCookieName = "HANKO_WEB_SESSION"

type SessionData struct {
	ID        string        `json:"id"`
	UserID    string        `json:"uid,omitempty"`
	Locale    string        `json:"locale,omitempty"`
	CartID    string        `json:"cart,omitempty"`
	Checkout  CheckoutState `json:"checkout,omitempty"`
	CSRFToken string        `json:"csrf,omitempty"`
	CreatedAt time.Time     `json:"createdAt"`
	UpdatedAt time.Time     `json:"updatedAt"`
	// internal dirty flag; not serialized
	dirty bool `json:"-"`
}

// CheckoutState stores lightweight checkout progress scoped to the session cookie.
type CheckoutState struct {
	ShippingAddressID string           `json:"ship,omitempty"`
	BillingAddressID  string           `json:"bill,omitempty"`
	Addresses         []SessionAddress `json:"addresses,omitempty"`
}

// SessionAddress is a simplified address record persisted inside the signed session cookie.
type SessionAddress struct {
	ID         string    `json:"id"`
	Label      string    `json:"label,omitempty"`
	Recipient  string    `json:"recipient,omitempty"`
	Company    string    `json:"company,omitempty"`
	Department string    `json:"department,omitempty"`
	Line1      string    `json:"line1,omitempty"`
	Line2      string    `json:"line2,omitempty"`
	City       string    `json:"city,omitempty"`
	Region     string    `json:"region,omitempty"`
	Postal     string    `json:"postal,omitempty"`
	Country    string    `json:"country,omitempty"`
	Phone      string    `json:"phone,omitempty"`
	Kind       string    `json:"kind,omitempty"`
	Notes      string    `json:"notes,omitempty"`
	CreatedAt  time.Time `json:"createdAt,omitempty"`
}

var sessionSignKey []byte
var sessionSecure bool

func init() {
	// signing key: prefer env var; if absent, generate a process-ephemeral one (dev only)
	key := os.Getenv("HANKO_WEB_SESSION_SIGNING_KEY")
	if key == "" {
		sessionSignKey = make([]byte, 32)
		if _, err := rand.Read(sessionSignKey); err != nil {
			log.Printf("session: failed to generate signing key: %v", err)
			sessionSignKey = []byte("insecure-dev-key-please-set-HANKO_WEB_SESSION_SIGNING_KEY")
		}
		log.Printf("session: using ephemeral signing key (dev). Set HANKO_WEB_SESSION_SIGNING_KEY for production.")
	} else {
		sessionSignKey = []byte(key)
	}
	// mark cookies secure in prod (when HANKO_WEB_ENV=prod)
	sessionSecure = strings.ToLower(os.Getenv("HANKO_WEB_ENV")) == "prod"
}

// Session loads or initializes a session and stores it in request context.
func Session(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sd, fromCookie := readSessionCookie(r)
		if sd.ID == "" {
			sd.ID = randID()
			sd.CreatedAt = time.Now().UTC()
			sd.UpdatedAt = sd.CreatedAt
			sd.CSRFToken = newCSRFToken()
			sd.dirty = true
		}
		// attach to context
		ctx := contextWithSession(r, sd)
		// proceed
		rw := NewResponseRecorder(w)
		// ensure cookie is set just before first write if needed
		rw.SetBeforeWrite(func(w http.ResponseWriter) {
			if sd.dirty || !fromCookie {
				writeSessionCookie(w, r, sd)
			}
		})
		next.ServeHTTP(rw, r.WithContext(ctx))
		// If nothing was written yet (e.g., HEAD), persist cookie now
		if !rw.wrote && (sd.dirty || !fromCookie) {
			writeSessionCookie(w, r, sd)
		}
	})
}

// context helpers
func contextWithSession(r *http.Request, s *SessionData) context.Context {
	ctx := context.WithValue(r.Context(), ctxKeySession, s)
	// if user id present, also attach user to context
	if s.UserID != "" {
		ctx = WithUser(ctx, &User{ID: s.UserID})
	}
	return ctx
}

// GetSession returns session data from context
func GetSession(r *http.Request) *SessionData {
	if v := r.Context().Value(ctxKeySession); v != nil {
		if sd, ok := v.(*SessionData); ok {
			return sd
		}
	}
	return &SessionData{}
}

// MarkDirty flags the session for writing at end of request
func (s *SessionData) MarkDirty() { s.dirty = true; s.UpdatedAt = time.Now().UTC() }

// readSessionCookie parses and verifies the session cookie
func readSessionCookie(r *http.Request) (*SessionData, bool) {
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		return &SessionData{}, false
	}
	parts := strings.Split(c.Value, ".")
	if len(parts) != 2 {
		return &SessionData{}, false
	}
	payloadB, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return &SessionData{}, false
	}
	sigB, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return &SessionData{}, false
	}
	mac := hmac.New(sha256.New, sessionSignKey)
	mac.Write(payloadB)
	if !hmac.Equal(sigB, mac.Sum(nil)) {
		return &SessionData{}, false
	}
	var sd SessionData
	if err := json.Unmarshal(payloadB, &sd); err != nil {
		return &SessionData{}, false
	}
	return &sd, true
}

func writeSessionCookie(w http.ResponseWriter, r *http.Request, sd *SessionData) {
	b, _ := json.Marshal(sd)
	payload := base64.RawURLEncoding.EncodeToString(b)
	mac := hmac.New(sha256.New, sessionSignKey)
	mac.Write(b)
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	val := payload + "." + sig
	// httpOnly to prevent JS access
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    val,
		Path:     "/",
		HttpOnly: true,
		Secure:   sessionSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
	})
}

// RegenerateID assigns a new session ID and CSRF token to prevent fixation after auth.
func (s *SessionData) RegenerateID() {
	s.ID = randID()
	s.CSRFToken = newCSRFToken()
	s.MarkDirty()
}

// helpers
func randID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
