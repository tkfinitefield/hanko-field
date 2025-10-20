package httpserver_test

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"finitefield.org/hanko-admin/internal/admin/httpserver/middleware"
	"finitefield.org/hanko-admin/internal/admin/testutil"
)

func TestDashboardRedirectsWithoutAuth(t *testing.T) {
	t.Parallel()

	ts := testutil.NewServer(t)

	client := noRedirectClient(t)

	resp, err := client.Get(ts.URL + "/admin")
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })

	require.Equal(t, http.StatusFound, resp.StatusCode)
	location := resp.Header.Get("Location")
	require.NotEmpty(t, location)
	loc, err := url.Parse(location)
	require.NoError(t, err)
	require.Equal(t, "/admin/login", loc.Path)
	q := loc.Query()
	require.Equal(t, middleware.ReasonMissingToken, q.Get("reason"))
	require.Equal(t, "/admin", q.Get("next"))
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

func TestLoginSuccessFlow(t *testing.T) {
	t.Parallel()

	auth := &tokenAuthenticator{Token: "valid-token"}
	ts := testutil.NewServer(t, testutil.WithAuthenticator(auth))

	client := noRedirectClient(t)

	seedLoginCSRF(t, client, ts.URL+"/admin/login")
	csrf := findCSRFCookie(t, client.Jar, ts.URL+"/admin/login")
	require.NotEmpty(t, csrf)

	form := url.Values{}
	form.Set("email", "tester@example.com")
	form.Set("id_token", auth.Token)
	form.Set("remember", "1")
	form.Set("csrf_token", csrf)

	resp, err := client.PostForm(ts.URL+"/admin/login", form)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	require.Equal(t, "/admin", resp.Header.Get("Location"))

	cookies := client.Jar.Cookies(mustParseURL(t, ts.URL+"/admin"))
	var authCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "Authorization" {
			authCookie = c
			break
		}
	}
	require.NotNil(t, authCookie)
	require.Equal(t, "Bearer "+auth.Token, authCookie.Value)

	resp, err = client.Get(ts.URL + "/admin")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestLoginHandlesInvalidToken(t *testing.T) {
	t.Parallel()

	auth := &tokenAuthenticator{Token: "expected-token"}
	ts := testutil.NewServer(t, testutil.WithAuthenticator(auth))

	client := noRedirectClient(t)
	seedLoginCSRF(t, client, ts.URL+"/admin/login")
	csrf := findCSRFCookie(t, client.Jar, ts.URL+"/admin/login")
	require.NotEmpty(t, csrf)

	form := url.Values{}
	form.Set("email", "tester@example.com")
	form.Set("id_token", "wrong-token")
	form.Set("csrf_token", csrf)

	resp, err := client.PostForm(ts.URL+"/admin/login", form)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "認証に失敗しました")
}

func TestLogoutClearsSession(t *testing.T) {
	t.Parallel()

	auth := &tokenAuthenticator{Token: "logout-token"}
	ts := testutil.NewServer(t, testutil.WithAuthenticator(auth))

	client := noRedirectClient(t)
	seedLoginCSRF(t, client, ts.URL+"/admin/login")
	csrf := findCSRFCookie(t, client.Jar, ts.URL+"/admin/login")
	require.NotEmpty(t, csrf)

	form := url.Values{}
	form.Set("id_token", auth.Token)
	form.Set("csrf_token", csrf)

	resp, err := client.PostForm(ts.URL+"/admin/login", form)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)

	resp, err = client.Get(ts.URL + "/admin/logout")
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	loc := resp.Header.Get("Location")
	require.NotEmpty(t, loc)
	mapped, err := url.Parse(loc)
	require.NoError(t, err)
	require.Equal(t, "/admin/login", mapped.Path)
	require.Equal(t, "logged_out", mapped.Query().Get("status"))

	resp, err = client.Get(ts.URL + "/admin")
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusFound, resp.StatusCode)

	reloc, err := url.Parse(resp.Header.Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "/admin/login", reloc.Path)
	require.Equal(t, middleware.ReasonMissingToken, reloc.Query().Get("reason"))
}

func TestLoginRejectsExternalNextParameter(t *testing.T) {
	t.Parallel()

	auth := &tokenAuthenticator{Token: "safe-token"}
	ts := testutil.NewServer(t, testutil.WithAuthenticator(auth))

	client := noRedirectClient(t)
	seedLoginCSRF(t, client, ts.URL+"/admin/login")
	csrf := findCSRFCookie(t, client.Jar, ts.URL+"/admin/login")
	require.NotEmpty(t, csrf)

	form := url.Values{}
	form.Set("id_token", auth.Token)
	form.Set("csrf_token", csrf)
	form.Set("next", "http://evil.example/phish")

	resp, err := client.PostForm(ts.URL+"/admin/login", form)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	require.Equal(t, "/admin", resp.Header.Get("Location"))

	// Ensure encoded double slash is also rejected.
	form.Set("next", "%2f%2fevil.example/another")
	resp, err = client.PostForm(ts.URL+"/admin/login", form)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	require.Equal(t, "/admin", resp.Header.Get("Location"))
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

func noRedirectClient(t testing.TB) *http.Client {
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	t.Cleanup(func() {
		client.CloseIdleConnections()
	})
	return client
}

func seedLoginCSRF(t testing.TB, client *http.Client, loginURL string) {
	resp, err := client.Get(loginURL)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
}

func findCSRFCookie(t testing.TB, jar http.CookieJar, rawURL string) string {
	u := mustParseURL(t, rawURL)
	cookies := jar.Cookies(u)
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			return c.Value
		}
	}
	return ""
}

func mustParseURL(t testing.TB, raw string) *url.URL {
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return u
}
