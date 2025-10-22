package profile_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"finitefield.org/hanko-admin/internal/admin/profile"
)

func TestHTTPServiceSecurityOverview(t *testing.T) {
	t.Parallel()

	var receivedAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/staff/security/profile", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		receivedAuth = r.Header.Get("Authorization")

		state := profile.SecurityState{
			UserEmail: "staff@example.com",
			Phone:     "+81-90-0000-0000",
			UpdatedAt: time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC),
			MFA: profile.MFAState{
				Enabled:       true,
				PrimaryMethod: profile.MFAMethodTOTP,
				Methods: []profile.MFAMethod{
					{
						ID:        "factor-1",
						Kind:      profile.MFAMethodTOTP,
						Label:     "Authenticator App",
						CreatedAt: time.Date(2024, 12, 30, 12, 0, 0, 0, time.UTC),
						Verified:  true,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(state)
	}))
	t.Cleanup(ts.Close)

	svc, err := profile.NewHTTPService(ts.URL, ts.Client())
	require.NoError(t, err)

	state, err := svc.SecurityOverview(context.Background(), "test-token")
	require.NoError(t, err)
	require.NotNil(t, state)
	require.Equal(t, "Bearer test-token", receivedAuth)
	require.Equal(t, "staff@example.com", state.UserEmail)
	require.True(t, state.MFA.Enabled)
}

func TestHTTPServiceCreateAPIKey(t *testing.T) {
	t.Parallel()

	var payload profile.CreateAPIKeyRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/staff/security/api-keys", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		defer r.Body.Close()
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))

		resp := profile.APIKeySecret{
			ID:        "key-1",
			Label:     payload.Label,
			Secret:    "abcd1234",
			CreatedAt: time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(ts.Close)

	svc, err := profile.NewHTTPService(ts.URL, ts.Client())
	require.NoError(t, err)

	secret, err := svc.CreateAPIKey(context.Background(), "token", profile.CreateAPIKeyRequest{Label: "Automation"})
	require.NoError(t, err)
	require.Equal(t, "Automation", secret.Label)
	require.Equal(t, "abcd1234", secret.Secret)
	require.Equal(t, "Automation", payload.Label)
}
