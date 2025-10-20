package httpserver_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"finitefield.org/hanko-admin/internal/admin/httpserver/middleware"
	"finitefield.org/hanko-admin/internal/admin/testutil"
)

func TestDashboardRedirectsWithoutAuth(t *testing.T) {
	t.Parallel()

	ts := testutil.NewServer(t)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(ts.URL + "/admin")
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })

	require.Equal(t, http.StatusFound, resp.StatusCode)
	require.Equal(t, "/admin/login", resp.Header.Get("Location"))
}

func TestDashboardRendersForAuthenticatedUser(t *testing.T) {
	t.Parallel()

	auth := &tokenAuthenticator{Token: "test-token"}
	ts := testutil.NewServer(t, testutil.WithAuthenticator(auth))

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/admin", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+auth.Token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	doc := testutil.ParseHTML(t, body)

	require.Equal(t, "ダッシュボード | Hanko Admin", doc.Find("title").First().Text())
	require.Equal(t, "admin.dashboard.title", doc.Find("h1").First().Text())
	require.Greater(t, doc.Find("table").Length(), 0, "dashboard should render metrics table")
}

type tokenAuthenticator struct {
	Token string
}

func (t *tokenAuthenticator) Authenticate(_ *http.Request, token string) (*middleware.User, error) {
	if token != t.Token {
		return nil, middleware.ErrUnauthorized
	}
	return &middleware.User{
		UID:   "tester",
		Email: "tester@example.com",
		Token: token,
		Roles: []string{"admin"},
	}, nil
}
