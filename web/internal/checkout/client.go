package checkout

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Default HTTP timeouts for PSP interactions.
const (
	defaultTimeout       = 8 * time.Second
	idempotencyHeader    = "Idempotency-Key"
	defaultConfirmTarget = "/checkout/review"
)

// Client issues checkout session and confirmation calls against the API service.
type Client struct {
	baseURL string
	http    *http.Client
}

// SessionRequest carries metadata required for PSP session creation.
type SessionRequest struct {
	Provider       string
	ReturnURL      string
	CancelURL      string
	Locale         string
	IdempotencyKey string
}

// SessionResponse mirrors the backend payload for a checkout session.
type SessionResponse struct {
	SessionID      string
	URL            string
	ClientSecret   string
	PublishableKey string
	Status         string
	Provider       string
	Amount         int64
	Currency       string
	ExpiresAt      time.Time
}

// ConfirmRequest finalizes a checkout session once PSP reports completion.
type ConfirmRequest struct {
	SessionID      string
	Provider       string
	IdempotencyKey string
}

// ConfirmResponse indicates the downstream order identifier and redirect target.
type ConfirmResponse struct {
	OrderID string
	Status  string
	NextURL string
}

// ErrMissingSessionID is returned when no session identifier is provided.
var ErrMissingSessionID = errors.New("checkout: missing session id")

// NewClient constructs an API client. When baseURL is empty, the client serves mock data.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		http: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// CreateSession invokes the backend API (or fallback) to initiate a PSP session.
func (c *Client) CreateSession(ctx context.Context, req SessionRequest) (SessionResponse, error) {
	provider := normalizeProvider(req.Provider)
	if c == nil || c.baseURL == "" {
		return fakeSessionResponse(provider), nil
	}

	body := map[string]string{
		"provider":  provider,
		"returnUrl": strings.TrimSpace(req.ReturnURL),
		"cancelUrl": strings.TrimSpace(req.CancelURL),
	}
	if req.Locale != "" {
		body["locale"] = req.Locale
	}

	endpoint, err := url.JoinPath(c.baseURL, "checkout", "session")
	if err != nil {
		return SessionResponse{}, err
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return SessionResponse{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return SessionResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set(idempotencyHeader, ensureIdempotencyKey(req.IdempotencyKey))

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return SessionResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return SessionResponse{}, fmt.Errorf("checkout: session status %d: %s", resp.StatusCode, drainError(resp.Body))
	}

	var payloadResp sessionPayload
	if err := json.NewDecoder(resp.Body).Decode(&payloadResp); err != nil {
		return SessionResponse{}, err
	}
	return payloadResp.toSessionResponse(provider), nil
}

// ConfirmSession notifies the backend that the PSP session (or saved method token) completed.
func (c *Client) ConfirmSession(ctx context.Context, req ConfirmRequest) (ConfirmResponse, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return ConfirmResponse{}, ErrMissingSessionID
	}
	provider := normalizeProvider(req.Provider)
	if c == nil || c.baseURL == "" {
		return fakeConfirmResponse(sessionID), nil
	}

	endpoint, err := url.JoinPath(c.baseURL, "checkout", "confirm")
	if err != nil {
		return ConfirmResponse{}, err
	}
	body := map[string]string{
		"sessionId": sessionID,
	}
	if provider != "" {
		body["provider"] = provider
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return ConfirmResponse{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return ConfirmResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set(idempotencyHeader, ensureIdempotencyKey(req.IdempotencyKey))

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return ConfirmResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return ConfirmResponse{}, fmt.Errorf("checkout: confirm status %d: %s", resp.StatusCode, drainError(resp.Body))
	}

	var payloadResp confirmPayload
	if err := json.NewDecoder(resp.Body).Decode(&payloadResp); err != nil {
		return ConfirmResponse{}, err
	}
	return payloadResp.toConfirmResponse(), nil
}

func (p sessionPayload) toSessionResponse(provider string) SessionResponse {
	resp := SessionResponse{
		SessionID:      strings.TrimSpace(p.SessionID),
		URL:            strings.TrimSpace(p.URL),
		ClientSecret:   strings.TrimSpace(p.ClientSecret),
		PublishableKey: strings.TrimSpace(p.PublishableKey),
		Status:         defaultString(p.Status, "requires_action"),
		Provider:       defaultString(p.Provider, provider),
		Amount:         p.Amount,
		Currency:       defaultString(p.Currency, "JPY"),
		ExpiresAt:      parseTime(p.ExpiresAt),
	}
	return resp
}

func (p confirmPayload) toConfirmResponse() ConfirmResponse {
	resp := ConfirmResponse{
		OrderID: strings.TrimSpace(p.OrderID),
		Status:  defaultString(p.Status, "pending_review"),
		NextURL: strings.TrimSpace(p.NextURL),
	}
	if resp.NextURL == "" {
		resp.NextURL = defaultConfirmTarget
	}
	return resp
}

func (c *Client) HTTPClient() *http.Client {
	if c.http == nil {
		c.http = &http.Client{Timeout: defaultTimeout}
	}
	return c.http
}

func ensureIdempotencyKey(key string) string {
	key = strings.TrimSpace(key)
	if key != "" {
		return key
	}
	return randomID("pay")
}

func normalizeProvider(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return "stripe"
	}
	return provider
}

func defaultString(val, fallback string) string {
	if strings.TrimSpace(val) == "" {
		return fallback
	}
	return strings.TrimSpace(val)
}

func drainError(r io.Reader) string {
	if r == nil {
		return ""
	}
	b, _ := io.ReadAll(io.LimitReader(r, 256))
	return strings.TrimSpace(string(b))
}

type sessionPayload struct {
	SessionID      string `json:"sessionId"`
	URL            string `json:"url"`
	ClientSecret   string `json:"clientSecret"`
	PublishableKey string `json:"publishableKey"`
	Status         string `json:"status"`
	Provider       string `json:"provider"`
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
	ExpiresAt      string `json:"expiresAt"`
}

type confirmPayload struct {
	OrderID string `json:"orderId"`
	Status  string `json:"status"`
	NextURL string `json:"nextUrl"`
}

func parseTime(val string) time.Time {
	val = strings.TrimSpace(val)
	if val == "" {
		return time.Time{}
	}
	layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05Z07:00"}
	for _, layout := range layouts {
		if ts, err := time.Parse(layout, val); err == nil {
			return ts
		}
	}
	return time.Time{}
}
