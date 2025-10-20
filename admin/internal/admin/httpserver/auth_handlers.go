package httpserver

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/a-h/templ"

	custommw "finitefield.org/hanko-admin/internal/admin/httpserver/middleware"
	appsession "finitefield.org/hanko-admin/internal/admin/session"
	"finitefield.org/hanko-admin/internal/admin/templates/auth"
)

const tokenCookieName = "Authorization"

type authHandlers struct {
	authenticator custommw.Authenticator
	basePath      string
	loginPath     string
}

func newAuthHandlers(authenticator custommw.Authenticator, basePath, loginPath string) *authHandlers {
	if authenticator == nil {
		panic("auth: authenticator is required")
	}
	if strings.TrimSpace(basePath) == "" {
		basePath = "/"
	}
	if strings.TrimSpace(loginPath) == "" {
		if basePath == "/" {
			loginPath = "/login"
		} else {
			loginPath = strings.TrimRight(basePath, "/") + "/login"
		}
	}
	return &authHandlers{
		authenticator: authenticator,
		basePath:      basePath,
		loginPath:     loginPath,
	}
}

func (h *authHandlers) LoginForm(w http.ResponseWriter, r *http.Request) {
	if h.isAuthenticated(r) && !forceLogin(r) {
		target := h.redirectTarget(r.URL.Query().Get("next"))
		http.Redirect(w, r, target, http.StatusFound)
		return
	}

	data := h.buildLoginPageData(r, nil)
	h.renderLoginPage(w, r, data, http.StatusOK)
}

func (h *authHandlers) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		state := &loginFormState{Error: "フォームの送信に失敗しました。もう一度お試しください。"}
		data := h.buildLoginPageData(r, state)
		h.renderLoginPage(w, r, data, http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(r.PostFormValue("email"))
	recordedNext := r.PostFormValue("next")
	remember := parseCheckbox(r.PostFormValue("remember"))
	token := strings.TrimSpace(r.PostFormValue("id_token"))
	refreshToken := strings.TrimSpace(r.PostFormValue("refresh_token"))

	state := &loginFormState{
		Email:    email,
		Remember: remember,
		Next:     recordedNext,
	}

	if token == "" {
		state.Error = "IDトークンまたはパスワードを入力してください。"
		data := h.buildLoginPageData(r, state)
		h.renderLoginPage(w, r, data, http.StatusBadRequest)
		return
	}

	user, err := h.authenticator.Authenticate(r, token)
	if err != nil || user == nil {
		log.Printf("admin login failed: %v", err)
		state.Error = h.errorMessageFor(err)
		data := h.buildLoginPageData(r, state)
		h.renderLoginPage(w, r, data, http.StatusUnauthorized)
		return
	}

	sess, _ := custommw.SessionFromContext(r.Context())
	if sess != nil {
		if user.Email == "" {
			user.Email = email
		}
		sess.SetUser(&appsession.User{
			UID:   user.UID,
			Email: user.Email,
			Roles: append([]string(nil), user.Roles...),
		})
		sess.SetFeatureFlags(user.FeatureFlags)
		sess.SetRememberMe(remember)
		if refreshToken != "" {
			sess.SetRefreshToken(refreshToken)
		}
	}

	issuedToken := token
	if user.Token != "" {
		issuedToken = user.Token
	}
	h.setAuthCookie(w, r, issuedToken, remember)

	target := h.redirectTarget(recordedNext)
	if custommw.IsHTMXRequest(r.Context()) {
		w.Header().Set("HX-Redirect", target)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (h *authHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	if sess, ok := custommw.SessionFromContext(r.Context()); ok && sess != nil {
		sess.Destroy()
	}
	h.clearAuthCookie(w)

	redirect := h.loginURLWithParams(map[string]string{
		"status": "logged_out",
	})

	if custommw.IsHTMXRequest(r.Context()) {
		w.Header().Set("HX-Redirect", redirect)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

type loginFormState struct {
	Email    string
	Remember bool
	Next     string
	Error    string
	Message  string
}

func (h *authHandlers) buildLoginPageData(r *http.Request, state *loginFormState) auth.LoginPageData {
	q := url.Values{}
	if r.URL != nil {
		q = r.URL.Query()
	}

	next := ""
	if state != nil && state.Next != "" {
		next = h.normalizeNext(state.Next)
	} else {
		next = h.normalizeNext(q.Get("next"))
	}

	message := ""
	if state != nil && strings.TrimSpace(state.Message) != "" {
		message = state.Message
	} else {
		message = h.messageForQuery(q)
	}

	errorText := ""
	if state != nil {
		errorText = state.Error
	}

	remember := false
	if state != nil {
		remember = state.Remember
	} else if sess, ok := custommw.SessionFromContext(r.Context()); ok && sess != nil {
		remember = sess.RememberMe()
	}

	email := ""
	if state != nil {
		email = state.Email
	} else {
		email = strings.TrimSpace(q.Get("email"))
	}

	return auth.LoginPageData{
		Email:     email,
		Message:   message,
		Error:     errorText,
		Remember:  remember,
		Next:      next,
		LoginPath: h.loginPath,
		BasePath:  h.basePath,
		CSRFToken: custommw.CSRFTokenFromContext(r.Context()),
	}
}

func (h *authHandlers) renderLoginPage(w http.ResponseWriter, r *http.Request, data auth.LoginPageData, status int) {
	if status != http.StatusOK {
		w.WriteHeader(status)
	}
	templ.Handler(auth.LoginPage(data)).ServeHTTP(w, r)
}

func (h *authHandlers) isAuthenticated(r *http.Request) bool {
	sess, ok := custommw.SessionFromContext(r.Context())
	if !ok || sess == nil {
		return false
	}
	user := sess.User()
	return user != nil && strings.TrimSpace(user.UID) != ""
}

func (h *authHandlers) errorMessageFor(err error) string {
	if err == nil {
		return "不明なエラーが発生しました。"
	}
	var authErr *custommw.AuthError
	if errors.As(err, &authErr) {
		switch authErr.Reason {
		case custommw.ReasonTokenExpired:
			return "セッションの有効期限が切れました。再度ログインしてください。"
		case custommw.ReasonMissingToken:
			return "認証情報が不足しています。もう一度確認してください。"
		default:
			return "認証に失敗しました。入力内容をご確認ください。"
		}
	}
	if errors.Is(err, custommw.ErrUnauthorized) {
		return "認証に失敗しました。入力内容をご確認ください。"
	}
	return "ログインに失敗しました。時間をおいて再度お試しください。"
}

func (h *authHandlers) messageForQuery(q url.Values) string {
	if q == nil {
		return ""
	}
	if status := q.Get("status"); status == "logged_out" {
		return "ログアウトしました。"
	}
	reason := q.Get("reason")
	switch reason {
	case custommw.ReasonTokenExpired, "expired":
		return "セッションの有効期限が切れました。再度ログインしてください。"
	case custommw.ReasonMissingToken:
		return "ログインが必要です。"
	case custommw.ReasonTokenInvalid:
		return "ログイン情報が無効です。再度お試しください。"
	default:
		return ""
	}
}

func (h *authHandlers) redirectTarget(raw string) string {
	next := h.normalizeNext(raw)
	if next != "" {
		return next
	}
	if strings.TrimSpace(h.basePath) == "" {
		return "/"
	}
	return h.basePath
}

func (h *authHandlers) setAuthCookie(w http.ResponseWriter, r *http.Request, token string, remember bool) {
	if strings.TrimSpace(token) == "" {
		h.clearAuthCookie(w)
		return
	}
	value := token
	if !strings.HasPrefix(strings.ToLower(token), "bearer ") {
		value = "Bearer " + token
	}
	cookie := &http.Cookie{
		Name:     tokenCookieName,
		Value:    value,
		Path:     h.cookiePath(),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	}
	if remember {
		if sess, ok := custommw.SessionFromContext(r.Context()); ok && sess != nil {
			if expiry := sess.ExpiresAt(); !expiry.IsZero() {
				expiry = expiry.UTC()
				cookie.Expires = expiry
				if remaining := time.Until(expiry); remaining > 0 {
					cookie.MaxAge = int(remaining.Round(time.Second).Seconds())
				}
			}
		}
	}
	http.SetCookie(w, cookie)
}

func (h *authHandlers) clearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     tokenCookieName,
		Value:    "",
		Path:     h.cookiePath(),
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *authHandlers) cookiePath() string {
	if strings.TrimSpace(h.basePath) == "" {
		return "/"
	}
	return h.basePath
}

func (h *authHandlers) loginURLWithParams(params map[string]string) string {
	parsed, err := url.Parse(h.loginPath)
	if err != nil {
		return h.loginPath
	}
	q := parsed.Query()
	for key, val := range params {
		if strings.TrimSpace(val) == "" {
			continue
		}
		q.Set(key, val)
	}
	parsed.RawQuery = q.Encode()
	return parsed.String()
}

func parseCheckbox(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "on", "yes":
		return true
	default:
		return false
	}
}

func forceLogin(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	flag := strings.TrimSpace(r.URL.Query().Get("force"))
	if flag == "" {
		return false
	}
	switch strings.ToLower(flag) {
	case "1", "true", "yes", "force":
		return true
	default:
		return false
	}
}

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	trim := func(p string) string {
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		for len(p) > 1 && strings.HasSuffix(p, "/") {
			p = strings.TrimSuffix(p, "/")
		}
		return p
	}
	return trim(a) == trim(b)
}

func (h *authHandlers) normalizeNext(raw string) string {
	sanitized := sanitizeNextTarget(h.basePath, raw)
	if sanitized == "" {
		return ""
	}

	if h.loginPath != "" {
		if samePath(pathOnly(sanitized), h.loginPath) {
			return ""
		}
	}
	return sanitized
}

func sanitizeNextTarget(basePath, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if parsed.Scheme != "" || parsed.Host != "" {
		return ""
	}

	pathValue := parsed.Path
	if pathValue == "" {
		pathValue = "/"
	}

	unescaped, err := url.PathUnescape(pathValue)
	if err != nil {
		return ""
	}
	if strings.Contains(unescaped, "\\") {
		return ""
	}

	cleaned := path.Clean(unescaped)
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	if strings.HasPrefix(cleaned, "//") {
		return ""
	}

	normalisedBase := normalizeBase(basePath)
	if normalisedBase != "/" && !hasSafePrefix(cleaned, normalisedBase) {
		return ""
	}

	target := cleaned
	if parsed.RawQuery != "" {
		target += "?" + parsed.RawQuery
	}
	if parsed.Fragment != "" {
		target += "#" + parsed.Fragment
	}
	return target
}

func normalizeBase(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return "/"
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	if len(base) > 1 && strings.HasSuffix(base, "/") {
		base = strings.TrimRight(base, "/")
	}
	return base
}

func hasSafePrefix(pathValue, base string) bool {
	if base == "/" {
		return strings.HasPrefix(pathValue, "/")
	}
	if !strings.HasPrefix(pathValue, base) {
		return false
	}
	if len(pathValue) == len(base) {
		return true
	}
	return pathValue[len(base)] == '/'
}

func pathOnly(raw string) string {
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return parsed.Path
}
