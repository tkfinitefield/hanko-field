package httpserver_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	admindashboard "finitefield.org/hanko-admin/internal/admin/dashboard"
	"finitefield.org/hanko-admin/internal/admin/httpserver/middleware"
	adminproduction "finitefield.org/hanko-admin/internal/admin/production"
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

func TestOrdersStatusUpdateFlow(t *testing.T) {
	t.Parallel()

	auth := &tokenAuthenticator{Token: "orders-token"}
	ts := testutil.NewServer(t, testutil.WithAuthenticator(auth))
	client := noRedirectClient(t)

	// Seed CSRF cookie by loading the orders page.
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/admin/orders", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	resp, err := client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	csrf := findCSRFCookie(t, client.Jar, ts.URL+"/admin")
	require.NotEmpty(t, csrf)

	// Fetch the status modal via htmx request.
	modalReq, err := http.NewRequest(http.MethodGet, ts.URL+"/admin/orders/order-1052/modal/status", nil)
	require.NoError(t, err)
	modalReq.Header.Set("Authorization", "Bearer "+auth.Token)
	modalReq.Header.Set("HX-Request", "true")
	modalReq.Header.Set("HX-Target", "modal")
	modalResp, err := client.Do(modalReq)
	require.NoError(t, err)
	body, err := io.ReadAll(modalResp.Body)
	modalResp.Body.Close()
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, modalResp.StatusCode)
	modalHTML := string(body)
	require.Contains(t, modalHTML, `hx-put="/admin/orders/order-1052:status"`)

	// Submit the status update.
	form := url.Values{}
	form.Set("status", "ready_to_ship")
	form.Set("note", "包装確認済み")
	form.Set("notifyCustomer", "true")
	updateReq, err := http.NewRequest(http.MethodPut, ts.URL+"/admin/orders/order-1052:status", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	updateReq.Header.Set("Authorization", "Bearer "+auth.Token)
	updateReq.Header.Set("HX-Request", "true")
	updateReq.Header.Set("HX-Target", "modal")
	updateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	updateReq.Header.Set("X-CSRF-Token", csrf)
	updateResp, err := client.Do(updateReq)
	require.NoError(t, err)
	updateBody, err := io.ReadAll(updateResp.Body)
	updateResp.Body.Close()
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, updateResp.StatusCode)
	require.Equal(t, `{"toast":{"message":"ステータスを更新しました。","tone":"success"},"modal:close":true}`, updateResp.Header.Get("HX-Trigger"))

	updateHTML := string(updateBody)
	require.Contains(t, updateHTML, "hx-swap-oob")
	require.Contains(t, updateHTML, "出荷待ち")
	require.Contains(t, updateHTML, "包装確認済み")
}

func TestOrdersRefundFlow(t *testing.T) {
	t.Parallel()

	auth := &tokenAuthenticator{Token: "refund-token"}
	ts := testutil.NewServer(t, testutil.WithAuthenticator(auth))
	client := noRedirectClient(t)

	// Seed CSRF cookie by loading the orders page.
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/admin/orders", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	resp, err := client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	csrf := findCSRFCookie(t, client.Jar, ts.URL+"/admin")
	require.NotEmpty(t, csrf)

	// Load the refund modal.
	modalReq, err := http.NewRequest(http.MethodGet, ts.URL+"/admin/orders/order-1052/modal/refund", nil)
	require.NoError(t, err)
	modalReq.Header.Set("Authorization", "Bearer "+auth.Token)
	modalReq.Header.Set("HX-Request", "true")
	modalReq.Header.Set("HX-Target", "modal")
	modalResp, err := client.Do(modalReq)
	require.NoError(t, err)
	modalBody, err := io.ReadAll(modalResp.Body)
	modalResp.Body.Close()
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, modalResp.StatusCode)
	modalHTML := string(modalBody)
	require.Contains(t, modalHTML, `hx-post="/admin/orders/order-1052/payments:refund"`)

	// Submit an invalid refund that exceeds the available amount.
	form := url.Values{}
	form.Set("paymentID", "pay-1052")
	form.Set("amount", "4000000") // ¥4,000,000 > ¥3,200,000 available
	form.Set("reason", "テスト返金")
	form.Set("notifyCustomer", "true")

	invalidReq, err := http.NewRequest(http.MethodPost, ts.URL+"/admin/orders/order-1052/payments:refund", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	invalidReq.Header.Set("Authorization", "Bearer "+auth.Token)
	invalidReq.Header.Set("HX-Request", "true")
	invalidReq.Header.Set("HX-Target", "modal")
	invalidReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	invalidReq.Header.Set("X-CSRF-Token", csrf)
	invalidResp, err := client.Do(invalidReq)
	require.NoError(t, err)
	invalidBody, err := io.ReadAll(invalidResp.Body)
	invalidResp.Body.Close()
	require.NoError(t, err)
	require.Equal(t, http.StatusUnprocessableEntity, invalidResp.StatusCode)
	require.Contains(t, string(invalidBody), "返金可能額を超えています。")

	// Submit a valid refund.
	form.Set("amount", "5000")
	validReq, err := http.NewRequest(http.MethodPost, ts.URL+"/admin/orders/order-1052/payments:refund", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	validReq.Header.Set("Authorization", "Bearer "+auth.Token)
	validReq.Header.Set("HX-Request", "true")
	validReq.Header.Set("HX-Target", "modal")
	validReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	validReq.Header.Set("X-CSRF-Token", csrf)
	validResp, err := client.Do(validReq)
	require.NoError(t, err)
	validBody, err := io.ReadAll(validResp.Body)
	validResp.Body.Close()
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, validResp.StatusCode)
	require.Empty(t, validBody)
	require.Equal(t, `{"toast":{"message":"返金を登録しました。","tone":"success"},"modal:close":true,"refresh:fragment":{"targets":["[data-order-payments]","[data-order-summary]"]}}`, validResp.Header.Get("HX-Trigger"))
}

func TestOrdersInvoiceIssueFlow(t *testing.T) {
	t.Parallel()

	auth := &tokenAuthenticator{Token: "invoice-token"}
	ts := testutil.NewServer(t, testutil.WithAuthenticator(auth))
	client := noRedirectClient(t)

	// Seed CSRF cookie.
	seedReq, err := http.NewRequest(http.MethodGet, ts.URL+"/admin/orders", nil)
	require.NoError(t, err)
	seedReq.Header.Set("Authorization", "Bearer "+auth.Token)
	seedResp, err := client.Do(seedReq)
	require.NoError(t, err)
	seedResp.Body.Close()
	require.Equal(t, http.StatusOK, seedResp.StatusCode)

	csrf := findCSRFCookie(t, client.Jar, ts.URL+"/admin")
	require.NotEmpty(t, csrf)

	// Load the invoice modal.
	modalReq, err := http.NewRequest(http.MethodGet, ts.URL+"/admin/orders/order-1052/modal/invoice", nil)
	require.NoError(t, err)
	modalReq.Header.Set("Authorization", "Bearer "+auth.Token)
	modalReq.Header.Set("HX-Request", "true")
	modalReq.Header.Set("HX-Target", "modal")
	modalResp, err := client.Do(modalReq)
	require.NoError(t, err)
	defer modalResp.Body.Close()
	require.Equal(t, http.StatusOK, modalResp.StatusCode)
	modalBody, err := io.ReadAll(modalResp.Body)
	require.NoError(t, err)
	require.Contains(t, string(modalBody), `hx-post="/admin/invoices:issue"`)

	// Submit invalid invoice request (invalid email).
	form := url.Values{}
	form.Set("orderID", "order-1052")
	form.Set("templateID", "invoice-standard")
	form.Set("language", "ja-JP")
	form.Set("email", "invalid-email")
	form.Set("note", "テスト領収書")

	invalidReq, err := http.NewRequest(http.MethodPost, ts.URL+"/admin/invoices:issue", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	invalidReq.Header.Set("Authorization", "Bearer "+auth.Token)
	invalidReq.Header.Set("HX-Request", "true")
	invalidReq.Header.Set("HX-Target", "modal")
	invalidReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	invalidReq.Header.Set("X-CSRF-Token", csrf)
	invalidResp, err := client.Do(invalidReq)
	require.NoError(t, err)
	defer invalidResp.Body.Close()
	require.Equal(t, http.StatusUnprocessableEntity, invalidResp.StatusCode)
	invalidBody, err := io.ReadAll(invalidResp.Body)
	require.NoError(t, err)
	require.Contains(t, string(invalidBody), "メールアドレスの形式が正しくありません")

	// Submit valid invoice request (synchronous template).
	form.Set("email", "jun.hasegawa+new@example.com")
	validReq, err := http.NewRequest(http.MethodPost, ts.URL+"/admin/invoices:issue", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	validReq.Header.Set("Authorization", "Bearer "+auth.Token)
	validReq.Header.Set("HX-Request", "true")
	validReq.Header.Set("HX-Target", "modal")
	validReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	validReq.Header.Set("X-CSRF-Token", csrf)
	validResp, err := client.Do(validReq)
	require.NoError(t, err)
	validResp.Body.Close()
	require.Equal(t, http.StatusNoContent, validResp.StatusCode)
	require.Equal(t, `{"toast":{"message":"領収書を発行しました。","tone":"success"},"modal:close":true,"refresh:fragment":{"targets":["[data-order-invoice]"]}}`, validResp.Header.Get("HX-Trigger"))

	// Load modal for asynchronous template.
	asyncModalReq, err := http.NewRequest(http.MethodGet, ts.URL+"/admin/orders/order-1050/modal/invoice", nil)
	require.NoError(t, err)
	asyncModalReq.Header.Set("Authorization", "Bearer "+auth.Token)
	asyncModalReq.Header.Set("HX-Request", "true")
	asyncModalReq.Header.Set("HX-Target", "modal")
	asyncModalResp, err := client.Do(asyncModalReq)
	require.NoError(t, err)
	defer asyncModalResp.Body.Close()
	require.Equal(t, http.StatusOK, asyncModalResp.StatusCode)

	// Submit asynchronous invoice request.
	asyncForm := url.Values{}
	asyncForm.Set("orderID", "order-1050")
	asyncForm.Set("templateID", "invoice-batch")
	asyncForm.Set("language", "ja-JP")
	asyncForm.Set("email", "maho.sato@example.com")
	asyncForm.Set("note", "バッチ請求書")

	asyncReq, err := http.NewRequest(http.MethodPost, ts.URL+"/admin/invoices:issue", strings.NewReader(asyncForm.Encode()))
	require.NoError(t, err)
	asyncReq.Header.Set("Authorization", "Bearer "+auth.Token)
	asyncReq.Header.Set("HX-Request", "true")
	asyncReq.Header.Set("HX-Target", "modal")
	asyncReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	asyncReq.Header.Set("X-CSRF-Token", csrf)
	asyncResp, err := client.Do(asyncReq)
	require.NoError(t, err)
	defer asyncResp.Body.Close()
	require.Equal(t, http.StatusOK, asyncResp.StatusCode)
	require.Equal(t, `{"toast":{"message":"領収書の生成を開始しました。","tone":"info"},"refresh:fragment":{"targets":["[data-order-invoice]"]}}`, asyncResp.Header.Get("HX-Trigger"))
	asyncBody, err := io.ReadAll(asyncResp.Body)
	require.NoError(t, err)
	require.Contains(t, string(asyncBody), "ジョブID")
	require.Contains(t, string(asyncBody), `data-invoice-job-status`)

	jobID := extractJobID(t, string(asyncBody))
	require.NotEmpty(t, jobID)

	// First poll should keep the job running.
	pollReq, err := http.NewRequest(http.MethodGet, ts.URL+"/admin/invoices/jobs/"+jobID, nil)
	require.NoError(t, err)
	pollReq.Header.Set("Authorization", "Bearer "+auth.Token)
	pollReq.Header.Set("HX-Request", "true")
	pollReq.Header.Set("HX-Target", "modal")
	pollResp, err := client.Do(pollReq)
	require.NoError(t, err)
	defer pollResp.Body.Close()
	require.Equal(t, http.StatusOK, pollResp.StatusCode)
	require.Empty(t, pollResp.Header.Get("HX-Trigger"))
	pollBody, err := io.ReadAll(pollResp.Body)
	require.NoError(t, err)
	require.Contains(t, string(pollBody), "現在のステータス")

	// Second poll should complete the job and close the modal.
	finalPollReq, err := http.NewRequest(http.MethodGet, ts.URL+"/admin/invoices/jobs/"+jobID, nil)
	require.NoError(t, err)
	finalPollReq.Header.Set("Authorization", "Bearer "+auth.Token)
	finalPollReq.Header.Set("HX-Request", "true")
	finalPollReq.Header.Set("HX-Target", "modal")
	finalPollResp, err := client.Do(finalPollReq)
	require.NoError(t, err)
	defer finalPollResp.Body.Close()
	require.Equal(t, http.StatusOK, finalPollResp.StatusCode)
	require.Equal(t, `{"toast":{"message":"領収書を発行しました。","tone":"success"},"modal:close":true,"refresh:fragment":{"targets":["[data-order-invoice]"]}}`, finalPollResp.Header.Get("HX-Trigger"))
}

func TestProductionQueuesPageRenders(t *testing.T) {
	t.Parallel()

	auth := &tokenAuthenticator{Token: "prod-board"}
	stub := &productionStub{boardResult: sampleBoardResult()}
	ts := testutil.NewServer(t, testutil.WithAuthenticator(auth), testutil.WithProductionService(stub))

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/admin/production/queues", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+auth.Token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "青山アトリエ")
	require.Contains(t, string(body), "待機")
}

func TestOrdersProductionEventSuccess(t *testing.T) {
	t.Parallel()

	auth := &tokenAuthenticator{Token: "prod-events"}
	stub := &productionStub{boardResult: sampleBoardResult()}
	ts := testutil.NewServer(t, testutil.WithAuthenticator(auth), testutil.WithProductionService(stub))
	client := noRedirectClient(t)

	seedReq, err := http.NewRequest(http.MethodGet, ts.URL+"/admin/production/queues", nil)
	require.NoError(t, err)
	seedReq.Header.Set("Authorization", "Bearer "+auth.Token)
	seedResp, err := client.Do(seedReq)
	require.NoError(t, err)
	seedResp.Body.Close()
	require.Equal(t, http.StatusOK, seedResp.StatusCode)

	csrf := findCSRFCookie(t, client.Jar, ts.URL+"/admin")
	require.NotEmpty(t, csrf)

	form := url.Values{}
	form.Set("type", "engraving")
	postReq, err := http.NewRequest(http.MethodPost, ts.URL+"/admin/orders/order-5000/production-events", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	postReq.Header.Set("Authorization", "Bearer "+auth.Token)
	postReq.Header.Set("HX-Request", "true")
	postReq.Header.Set("HX-Target", "production-board")
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("X-CSRF-Token", csrf)

	resp, err := client.Do(postReq)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.Contains(t, resp.Header.Get("HX-Trigger"), "制作ステージを更新しました。")

	require.Equal(t, "order-5000", stub.lastOrderID)
	require.Len(t, stub.appendCalls, 1)
	require.Equal(t, adminproduction.Stage("engraving"), stub.appendCalls[0].Stage)
}

func TestOrdersProductionEventHandlesErrors(t *testing.T) {
	t.Parallel()

	auth := &tokenAuthenticator{Token: "prod-events-error"}
	stub := &productionStub{boardResult: sampleBoardResult(), appendErr: adminproduction.ErrStageInvalid}
	ts := testutil.NewServer(t, testutil.WithAuthenticator(auth), testutil.WithProductionService(stub))
	client := noRedirectClient(t)

	seedReq, err := http.NewRequest(http.MethodGet, ts.URL+"/admin/production/queues", nil)
	require.NoError(t, err)
	seedReq.Header.Set("Authorization", "Bearer "+auth.Token)
	seedResp, err := client.Do(seedReq)
	require.NoError(t, err)
	seedResp.Body.Close()
	require.Equal(t, http.StatusOK, seedResp.StatusCode)
	csrf := findCSRFCookie(t, client.Jar, ts.URL+"/admin")
	require.NotEmpty(t, csrf)

	form := url.Values{}
	form.Set("type", "invalid-stage")
	postReq, err := http.NewRequest(http.MethodPost, ts.URL+"/admin/orders/order-5000/production-events", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	postReq.Header.Set("Authorization", "Bearer "+auth.Token)
	postReq.Header.Set("HX-Request", "true")
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("X-CSRF-Token", csrf)

	resp, err := client.Do(postReq)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "指定されたステージに移動できません")
}

func sampleBoardResult() adminproduction.BoardResult {
	now := time.Now()
	return adminproduction.BoardResult{
		Queue: adminproduction.Queue{
			ID:            "atelier-aoyama",
			Name:          "青山アトリエ",
			Capacity:      10,
			Load:          5,
			LeadTimeHours: 24,
		},
		Queues: []adminproduction.QueueOption{{ID: "atelier-aoyama", Label: "青山アトリエ", Active: true}},
		Summary: adminproduction.Summary{
			TotalWIP:     1,
			DueSoon:      1,
			Blocked:      0,
			AvgLeadHours: 24,
			Utilisation:  50,
			UpdatedAt:    now,
		},
		Filters: adminproduction.FilterSummary{},
		Lanes: []adminproduction.Lane{
			{
				Stage:    adminproduction.StageQueued,
				Label:    "待機",
				Capacity: adminproduction.LaneCapacity{Used: 1, Limit: 6},
				SLA:      adminproduction.SLAMeta{Label: "平均6h", Tone: "info"},
				Cards: []adminproduction.Card{
					{
						ID:            "order-5000",
						OrderNumber:   "5000",
						Stage:         adminproduction.StageQueued,
						Priority:      adminproduction.PriorityRush,
						PriorityLabel: "特急",
						PriorityTone:  "warning",
						Customer:      "テスト 顧客",
						ProductLine:   "Classic",
						Design:        "テストリング",
						PreviewURL:    "/public/static/previews/ring-classic.png",
						QueueID:       "atelier-aoyama",
						QueueName:     "青山アトリエ",
						DueAt:         now.Add(6 * time.Hour),
						DueLabel:      "残り6時間",
					},
				},
			},
		},
		Drawer:          adminproduction.Drawer{Empty: true},
		SelectedCardID:  "order-5000",
		GeneratedAt:     now,
		RefreshInterval: 30 * time.Second,
	}
}

type productionStub struct {
	boardResult  adminproduction.BoardResult
	boardErr     error
	appendResult adminproduction.AppendEventResult
	appendErr    error
	lastOrderID  string
	appendCalls  []adminproduction.AppendEventRequest
}

func (s *productionStub) Board(ctx context.Context, token string, query adminproduction.BoardQuery) (adminproduction.BoardResult, error) {
	if s.boardErr != nil {
		return adminproduction.BoardResult{}, s.boardErr
	}
	return s.boardResult, nil
}

func (s *productionStub) AppendEvent(ctx context.Context, token, orderID string, req adminproduction.AppendEventRequest) (adminproduction.AppendEventResult, error) {
	s.lastOrderID = orderID
	s.appendCalls = append(s.appendCalls, req)
	if s.appendErr != nil {
		return adminproduction.AppendEventResult{}, s.appendErr
	}
	res := s.appendResult
	if res.Card.ID == "" {
		res.Card.ID = orderID
	}
	return res, nil
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

func extractJobID(t testing.TB, body string) string {
	t.Helper()
	re := regexp.MustCompile(`job-[A-Za-z0-9\-]+`)
	match := re.FindString(body)
	return strings.TrimSpace(match)
}

func mustParseURL(t testing.TB, raw string) *url.URL {
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return u
}
