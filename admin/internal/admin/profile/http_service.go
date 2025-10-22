package profile

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// HTTPClient matches the subset of http.Client used by HTTPService.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// HTTPService implements Service backed by REST endpoints exposed by the backend API.
type HTTPService struct {
	base   *url.URL
	client HTTPClient
}

// NewHTTPService constructs a Service that talks to the backend secrets/security API.
func NewHTTPService(baseURL string, client HTTPClient) (*HTTPService, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, errors.New("profile: base URL is required")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("profile: parse base URL: %w", err)
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPService{
		base:   parsed,
		client: client,
	}, nil
}

// SecurityOverview retrieves the current security state for the caller.
func (s *HTTPService) SecurityOverview(ctx context.Context, token string) (*SecurityState, error) {
	req, err := s.newRequest(ctx, http.MethodGet, "/staff/security/profile", nil, token)
	if err != nil {
		return nil, err
	}
	resp, err := s.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, s.errorFromResponse(resp)
	}

	var payload SecurityState
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("profile: decode security overview: %w", err)
	}
	return &payload, nil
}

// StartTOTPEnrollment starts a new TOTP enrollment session.
func (s *HTTPService) StartTOTPEnrollment(ctx context.Context, token string) (*TOTPEnrollment, error) {
	req, err := s.newRequest(ctx, http.MethodPost, "/staff/security/mfa/totp:start", nil, token)
	if err != nil {
		return nil, err
	}
	resp, err := s.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, s.errorFromResponse(resp)
	}

	var payload TOTPEnrollment
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("profile: decode totp enrollment: %w", err)
	}
	return &payload, nil
}

// ConfirmTOTPEnrollment finalises the TOTP enrollment with the provided code.
func (s *HTTPService) ConfirmTOTPEnrollment(ctx context.Context, token, code string) (*SecurityState, error) {
	body := map[string]string{"code": strings.TrimSpace(code)}
	req, err := s.newJSONRequest(ctx, http.MethodPost, "/staff/security/mfa/totp:confirm", body, token)
	if err != nil {
		return nil, err
	}
	resp, err := s.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, s.errorFromResponse(resp)
	}

	var payload SecurityState
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("profile: decode totp confirmation: %w", err)
	}
	return &payload, nil
}

// EnableEmailMFA enables email-based MFA.
func (s *HTTPService) EnableEmailMFA(ctx context.Context, token string) (*SecurityState, error) {
	req, err := s.newRequest(ctx, http.MethodPost, "/staff/security/mfa/email:enable", nil, token)
	if err != nil {
		return nil, err
	}
	resp, err := s.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, s.errorFromResponse(resp)
	}

	var payload SecurityState
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("profile: decode email mfa enable: %w", err)
	}
	return &payload, nil
}

// DisableMFA disables MFA factors for the caller.
func (s *HTTPService) DisableMFA(ctx context.Context, token string) (*SecurityState, error) {
	req, err := s.newRequest(ctx, http.MethodDelete, "/staff/security/mfa", nil, token)
	if err != nil {
		return nil, err
	}
	resp, err := s.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, s.errorFromResponse(resp)
	}

	var payload SecurityState
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("profile: decode mfa disable: %w", err)
	}
	return &payload, nil
}

// CreateAPIKey issues a new API key.
func (s *HTTPService) CreateAPIKey(ctx context.Context, token string, reqBody CreateAPIKeyRequest) (*APIKeySecret, error) {
	req, err := s.newJSONRequest(ctx, http.MethodPost, "/staff/security/api-keys", reqBody, token)
	if err != nil {
		return nil, err
	}
	resp, err := s.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, s.errorFromResponse(resp)
	}

	var payload APIKeySecret
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("profile: decode api key secret: %w", err)
	}
	return &payload, nil
}

// RevokeAPIKey revokes an API key and returns updated security state.
func (s *HTTPService) RevokeAPIKey(ctx context.Context, token, keyID string) (*SecurityState, error) {
	escaped := path.Join("/staff/security/api-keys", url.PathEscape(strings.TrimSpace(keyID)))
	req, err := s.newRequest(ctx, http.MethodDelete, escaped, nil, token)
	if err != nil {
		return nil, err
	}
	resp, err := s.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, s.errorFromResponse(resp)
	}

	var payload SecurityState
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("profile: decode api key revoke: %w", err)
	}
	return &payload, nil
}

// RevokeSession revokes an active session.
func (s *HTTPService) RevokeSession(ctx context.Context, token, sessionID string) (*SecurityState, error) {
	endpoint := path.Join("/staff/security/sessions", url.PathEscape(strings.TrimSpace(sessionID))+":revoke")
	req, err := s.newRequest(ctx, http.MethodPost, endpoint, nil, token)
	if err != nil {
		return nil, err
	}
	resp, err := s.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, s.errorFromResponse(resp)
	}

	var payload SecurityState
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("profile: decode session revoke: %w", err)
	}
	return &payload, nil
}

func (s *HTTPService) do(req *http.Request) (*http.Response, error) {
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("profile: request failed: %w", err)
	}
	return resp, nil
}

func (s *HTTPService) newRequest(ctx context.Context, method, endpoint string, body io.Reader, token string) (*http.Request, error) {
	urlStr := s.resolve(endpoint)
	req, err := http.NewRequestWithContext(ctx, method, urlStr, body)
	if err != nil {
		return nil, fmt.Errorf("profile: build request: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil && (method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch) {
		// Default to JSON unless caller set otherwise.
		if req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}
	}
	return req, nil
}

func (s *HTTPService) newJSONRequest(ctx context.Context, method, endpoint string, payload any, token string) (*http.Request, error) {
	var buf bytes.Buffer
	if payload != nil {
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(payload); err != nil {
			return nil, fmt.Errorf("profile: encode payload: %w", err)
		}
	}
	req, err := s.newRequest(ctx, method, endpoint, &buf, token)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (s *HTTPService) resolve(endpoint string) string {
	if endpoint == "" {
		return s.base.String()
	}
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return endpoint
	}
	trimmed := strings.TrimPrefix(endpoint, "/")
	ref := &url.URL{Path: trimmed}
	return s.base.ResolveReference(ref).String()
}

func (s *HTTPService) errorFromResponse(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	_ = resp.Body.Close()

	type errorPayload struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	var payload errorPayload
	if len(body) > 0 {
		if err := json.Unmarshal(body, &payload); err == nil && payload.Message != "" {
			return fmt.Errorf("profile: backend error (%s): %s", strings.TrimSpace(payload.Code), payload.Message)
		}
	}
	if len(body) > 0 {
		return fmt.Errorf("profile: backend error (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return fmt.Errorf("profile: backend error (%d): %s", resp.StatusCode, http.StatusText(resp.StatusCode))
}

// StaticService returns a no-op service for tests when backend integration is not configured.
type StaticService struct {
	State         *SecurityState
	Enrollment    *TOTPEnrollment
	NewKey        *APIKeySecret
	LastOperation *SecurityState
}

// NewStaticService constructs a StaticService with helpful defaults.
func NewStaticService(state *SecurityState) *StaticService {
	if state == nil {
		state = &SecurityState{
			UpdatedAt: time.Now(),
		}
	}
	return &StaticService{State: state}
}

// SecurityOverview returns the configured static state.
func (s *StaticService) SecurityOverview(ctx context.Context, token string) (*SecurityState, error) {
	return s.State, nil
}

// StartTOTPEnrollment returns the configured enrollment payload.
func (s *StaticService) StartTOTPEnrollment(ctx context.Context, token string) (*TOTPEnrollment, error) {
	if s.Enrollment == nil {
		return &TOTPEnrollment{
			Issuer:      "Hanko Field",
			AccountName: "staff@example.com",
			Secret:      "FAKE-SECRET",
		}, nil
	}
	return s.Enrollment, nil
}

// ConfirmTOTPEnrollment returns the stored state.
func (s *StaticService) ConfirmTOTPEnrollment(ctx context.Context, token, code string) (*SecurityState, error) {
	return s.State, nil
}

// EnableEmailMFA returns the stored state.
func (s *StaticService) EnableEmailMFA(ctx context.Context, token string) (*SecurityState, error) {
	return s.State, nil
}

// DisableMFA returns the stored state.
func (s *StaticService) DisableMFA(ctx context.Context, token string) (*SecurityState, error) {
	return s.State, nil
}

// CreateAPIKey returns the configured new key.
func (s *StaticService) CreateAPIKey(ctx context.Context, token string, req CreateAPIKeyRequest) (*APIKeySecret, error) {
	if s.NewKey == nil {
		return &APIKeySecret{
			ID:        "fake",
			Label:     req.Label,
			Secret:    "FAKE-SECRET",
			CreatedAt: time.Now(),
		}, nil
	}
	return s.NewKey, nil
}

// RevokeAPIKey returns the stored state.
func (s *StaticService) RevokeAPIKey(ctx context.Context, token, keyID string) (*SecurityState, error) {
	return s.State, nil
}

// RevokeSession returns the stored state.
func (s *StaticService) RevokeSession(ctx context.Context, token, sessionID string) (*SecurityState, error) {
	return s.State, nil
}
