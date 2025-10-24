package httpserver_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	admindashboard "finitefield.org/hanko-admin/internal/admin/dashboard"
	"finitefield.org/hanko-admin/internal/admin/httpserver/middleware"
	"finitefield.org/hanko-admin/internal/admin/profile"
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
	now := time.Now()
	stub := &dashboardStub{
		kpis: []admindashboard.KPI{
			{ID: "revenue", Label: "日次売上", Value: "¥123,000", DeltaText: "+12%", Trend: admindashboard.TrendUp, Sparkline: []float64{120, 135, 140}, UpdatedAt: now},
		},
		alerts: []admindashboard.Alert{
			{ID: "inventory", Severity: "warning", Title: "在庫警告", Message: "SKU在庫が閾値を下回りました", ActionURL: "/admin/catalog/products", Action: "確認", CreatedAt: now.Add(-30 * time.Minute)},
		},
		activity: []admindashboard.ActivityItem{
			{ID: "order", Icon: "📦", Title: "注文 #1001 を出荷しました", Detail: "山田様", Occurred: now.Add(-10 * time.Minute)},
		},
	}

	ts := testutil.NewServer(t, testutil.WithAuthenticator(auth), testutil.WithDashboardService(stub))

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
	require.Greater(t, doc.Find("#dashboard-kpi article").Length(), 0, "dashboard should render KPI cards")
	require.Greater(t, doc.Find("#dashboard-alerts li").Length(), 0, "dashboard should render alerts list")
	require.Equal(t, 1, doc.Find("[data-dashboard-refresh]").Length(), "refresh control should be present")
	require.Contains(t, doc.Find("aside").Text(), "注文 #1001")
}

func TestDashboardKPIFragmentProvidesCards(t *testing.T) {
	t.Parallel()

	auth := &tokenAuthenticator{Token: "kpi-token"}
	now := time.Now()
	stub := &dashboardStub{
		kpis: []admindashboard.KPI{
			{ID: "orders", Label: "注文数", Value: "128", DeltaText: "+8件", Trend: admindashboard.TrendUp, Sparkline: []float64{10, 12, 15}, UpdatedAt: now},
		},
	}

	ts := testutil.NewServer(t, testutil.WithAuthenticator(auth), testutil.WithDashboardService(stub))

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/admin/fragments/kpi?limit=1", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+auth.Token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	doc := testutil.ParseHTML(t, body)
	require.Equal(t, 1, doc.Find("article").Length())
	require.Contains(t, doc.Text(), "注文数")
}

func TestDashboardKPIsHandlesServiceError(t *testing.T) {
	t.Parallel()

	auth := &tokenAuthenticator{Token: "kpi-error"}
	stub := &dashboardStub{kpiErr: errors.New("backend down")}

	ts := testutil.NewServer(t, testutil.WithAuthenticator(auth), testutil.WithDashboardService(stub))

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/admin/fragments/kpi", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+auth.Token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, string(body), "KPIの取得に失敗しました")
}

func TestDashboardAlertsFragmentProvidesList(t *testing.T) {
	t.Parallel()

	auth := &tokenAuthenticator{Token: "alert-token"}
	now := time.Now()
	stub := &dashboardStub{
		alerts: []admindashboard.Alert{
			{ID: "delay", Severity: "danger", Title: "配送遅延", Message: "2件が遅延中", ActionURL: "/admin/shipments/tracking", Action: "確認", CreatedAt: now},
		},
	}

	ts := testutil.NewServer(t, testutil.WithAuthenticator(auth), testutil.WithDashboardService(stub))

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/admin/fragments/alerts", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+auth.Token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	doc := testutil.ParseHTML(t, body)
	require.GreaterOrEqual(t, doc.Find("li").Length(), 1)
	require.Contains(t, doc.Text(), "配送遅延")
}

func TestDashboardAlertsHandlesServiceError(t *testing.T) {
	t.Parallel()

	auth := &tokenAuthenticator{Token: "alert-error"}
	stub := &dashboardStub{alertsErr: errors.New("timeout")}

	ts := testutil.NewServer(t, testutil.WithAuthenticator(auth), testutil.WithDashboardService(stub))

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/admin/fragments/alerts", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+auth.Token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "アラートの取得に失敗しました")
}

func TestProfilePageRenders(t *testing.T) {
	t.Parallel()

	auth := &tokenAuthenticator{Token: "secure-token"}
	service := &profileStub{
		state: &profile.SecurityState{
			UserEmail: "staff@example.com",
			MFA:       profile.MFAState{Enabled: true},
			APIKeys: []profile.APIKey{
				{ID: "key-1", Label: "Automation", Status: profile.APIKeyStatusActive, CreatedAt: time.Now()},
			},
			Sessions: []profile.Session{
				{ID: "sess-1", UserAgent: "Chrome", IPAddress: "127.0.0.1", CreatedAt: time.Now(), LastSeenAt: time.Now(), Current: true},
			},
		},
	}

	ts := testutil.NewServer(t, testutil.WithAuthenticator(auth), testutil.WithProfileService(service))

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/admin/profile", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+auth.Token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	doc := testutil.ParseHTML(t, body)
	require.Contains(t, doc.Find("title").First().Text(), "admin.profile.title")
	require.Contains(t, doc.Find("body").Text(), "API キー")
}

func TestProfileTabsFragmentHTMX(t *testing.T) {
	t.Parallel()

	auth := &tokenAuthenticator{Token: "tab-token"}
	service := &profileStub{
		state: &profile.SecurityState{
			UserEmail: "staff@example.com",
			Sessions: []profile.Session{
				{ID: "sess-2", UserAgent: "Safari", IPAddress: "203.0.113.10", CreatedAt: time.Now(), LastSeenAt: time.Now(), Current: false},
			},
		},
	}

	ts := testutil.NewServer(t, testutil.WithAuthenticator(auth), testutil.WithProfileService(service))

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/admin/profile?tab=sessions", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "profile-tabs")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	html := string(body)

	require.NotContains(t, html, "<html")
	require.Contains(t, html, `id="profile-tabs"`)
	require.Contains(t, html, "アクティブセッション")
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

type dashboardStub struct {
	kpis        []admindashboard.KPI
	alerts      []admindashboard.Alert
	activity    []admindashboard.ActivityItem
	kpiErr      error
	alertsErr   error
	activityErr error
}

func (s *dashboardStub) FetchKPIs(ctx context.Context, token string, since *time.Time) ([]admindashboard.KPI, error) {
	if s.kpiErr != nil {
		return nil, s.kpiErr
	}
	if since != nil {
		filtered := make([]admindashboard.KPI, 0, len(s.kpis))
		for _, k := range s.kpis {
			if k.UpdatedAt.After(*since) || k.UpdatedAt.Equal(*since) {
				filtered = append(filtered, k)
			}
		}
		return append([]admindashboard.KPI(nil), filtered...), nil
	}
	return append([]admindashboard.KPI(nil), s.kpis...), nil
}

func (s *dashboardStub) FetchAlerts(ctx context.Context, token string, limit int) ([]admindashboard.Alert, error) {
	if s.alertsErr != nil {
		return nil, s.alertsErr
	}
	alerts := append([]admindashboard.Alert(nil), s.alerts...)
	if limit > 0 && len(alerts) > limit {
		alerts = alerts[:limit]
	}
	return alerts, nil
}

func (s *dashboardStub) FetchActivity(ctx context.Context, token string, limit int) ([]admindashboard.ActivityItem, error) {
	if s.activityErr != nil {
		return nil, s.activityErr
	}
	items := append([]admindashboard.ActivityItem(nil), s.activity...)
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

type profileStub struct {
	state      *profile.SecurityState
	enrollment *profile.TOTPEnrollment
	secret     *profile.APIKeySecret
}

func (s *profileStub) SecurityOverview(ctx context.Context, token string) (*profile.SecurityState, error) {
	return s.state, nil
}

func (s *profileStub) StartTOTPEnrollment(ctx context.Context, token string) (*profile.TOTPEnrollment, error) {
	if s.enrollment != nil {
		return s.enrollment, nil
	}
	return &profile.TOTPEnrollment{Secret: "SECRET"}, nil
}

func (s *profileStub) ConfirmTOTPEnrollment(ctx context.Context, token, code string) (*profile.SecurityState, error) {
	return s.state, nil
}

func (s *profileStub) EnableEmailMFA(ctx context.Context, token string) (*profile.SecurityState, error) {
	return s.state, nil
}

func (s *profileStub) DisableMFA(ctx context.Context, token string) (*profile.SecurityState, error) {
	return s.state, nil
}

func (s *profileStub) CreateAPIKey(ctx context.Context, token string, req profile.CreateAPIKeyRequest) (*profile.APIKeySecret, error) {
	if s.secret != nil {
		return s.secret, nil
	}
	return &profile.APIKeySecret{ID: "key-2", Label: req.Label, Secret: "secret"}, nil
}

func (s *profileStub) RevokeAPIKey(ctx context.Context, token, keyID string) (*profile.SecurityState, error) {
	return s.state, nil
}

func (s *profileStub) RevokeSession(ctx context.Context, token, sessionID string) (*profile.SecurityState, error) {
	return s.state, nil
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
