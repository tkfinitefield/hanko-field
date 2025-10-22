package ui

import (
	"log"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"

	custommw "finitefield.org/hanko-admin/internal/admin/httpserver/middleware"
	"finitefield.org/hanko-admin/internal/admin/profile"
	profiletpl "finitefield.org/hanko-admin/internal/admin/templates/profile"
)

// ProfilePage renders the main profile/security dashboard.
func (h *Handlers) ProfilePage(w http.ResponseWriter, r *http.Request) {
	h.renderProfilePage(w, r)
}

// MFATOTPStart displays the enrollment modal with QR code/secret.
func (h *Handlers) MFATOTPStart(w http.ResponseWriter, r *http.Request) {
	user, ok := custommw.UserFromContext(r.Context())
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	enrollment, err := h.profile.StartTOTPEnrollment(r.Context(), user.Token)
	if err != nil {
		log.Printf("profile: start totp enrollment failed: %v", err)
		http.Error(w, "MFA登録の開始に失敗しました。後ほどお試しください。", http.StatusBadGateway)
		return
	}

	data := profiletpl.TOTPModalData{
		Enrollment: enrollment,
		CSRFToken:  custommw.CSRFTokenFromContext(r.Context()),
	}
	templ.Handler(profiletpl.TOTPModal(data)).ServeHTTP(w, r)
}

// MFATOTPConfirm finalises TOTP enrollment.
func (h *Handlers) MFATOTPConfirm(w http.ResponseWriter, r *http.Request) {
	user, ok := custommw.UserFromContext(r.Context())
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームの解析に失敗しました。", http.StatusBadRequest)
		return
	}
	code := strings.TrimSpace(r.PostFormValue("code"))
	if code == "" {
		enrollment, _ := h.profile.StartTOTPEnrollment(r.Context(), user.Token)
		data := profiletpl.TOTPModalData{
			Enrollment: enrollment,
			CSRFToken:  custommw.CSRFTokenFromContext(r.Context()),
			Error:      "認証コードを入力してください。",
		}
		templ.Handler(profiletpl.TOTPModal(data)).ServeHTTP(w, r)
		return
	}

	state, err := h.profile.ConfirmTOTPEnrollment(r.Context(), user.Token, code)
	if err != nil {
		log.Printf("profile: confirm totp enrollment failed: %v", err)
		enrollment, _ := h.profile.StartTOTPEnrollment(r.Context(), user.Token)
		data := profiletpl.TOTPModalData{
			Enrollment: enrollment,
			CSRFToken:  custommw.CSRFTokenFromContext(r.Context()),
			Error:      "コードが正しくないか、期限切れです。再度お試しください。",
		}
		templ.Handler(profiletpl.TOTPModal(data)).ServeHTTP(w, r)
		return
	}

	payload := profiletpl.MFAUpdateData{
		Security:  state,
		CSRFToken: custommw.CSRFTokenFromContext(r.Context()),
		Message:   "Authenticator アプリを有効化しました。",
	}
	templ.Handler(profiletpl.MFAUpdate(payload)).ServeHTTP(w, r)
}

// EmailMFAEnable toggles email-based MFA.
func (h *Handlers) EmailMFAEnable(w http.ResponseWriter, r *http.Request) {
	user, ok := custommw.UserFromContext(r.Context())
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	state, err := h.profile.EnableEmailMFA(r.Context(), user.Token)
	if err != nil {
		log.Printf("profile: enable email mfa failed: %v", err)
		http.Error(w, "メール認証の有効化に失敗しました。", http.StatusBadGateway)
		return
	}

	payload := profiletpl.MFAUpdateData{
		Security:  state,
		CSRFToken: custommw.CSRFTokenFromContext(r.Context()),
		Message:   "メールによる MFA を有効化しました。",
	}
	templ.Handler(profiletpl.MFAUpdate(payload)).ServeHTTP(w, r)
}

// DisableMFA removes MFA factors.
func (h *Handlers) DisableMFA(w http.ResponseWriter, r *http.Request) {
	user, ok := custommw.UserFromContext(r.Context())
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	state, err := h.profile.DisableMFA(r.Context(), user.Token)
	if err != nil {
		log.Printf("profile: disable mfa failed: %v", err)
		http.Error(w, "MFAの無効化に失敗しました。", http.StatusBadGateway)
		return
	}

	payload := profiletpl.MFAUpdateData{
		Security:  state,
		CSRFToken: custommw.CSRFTokenFromContext(r.Context()),
		Message:   "MFA を無効化しました。",
	}
	templ.Handler(profiletpl.MFAUpdate(payload)).ServeHTTP(w, r)
}

// NewAPIKeyForm renders the creation form modal.
func (h *Handlers) NewAPIKeyForm(w http.ResponseWriter, r *http.Request) {
	data := profiletpl.APIKeyFormData{
		CSRFToken: custommw.CSRFTokenFromContext(r.Context()),
	}
	templ.Handler(profiletpl.APIKeyFormModal(data)).ServeHTTP(w, r)
}

// CreateAPIKey issues a new key and displays the secret once.
func (h *Handlers) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	user, ok := custommw.UserFromContext(r.Context())
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームの解析に失敗しました。", http.StatusBadRequest)
		return
	}

	label := strings.TrimSpace(r.PostFormValue("label"))
	if label == "" {
		data := profiletpl.APIKeyFormData{
			CSRFToken: custommw.CSRFTokenFromContext(r.Context()),
			Error:     "APIキーのラベルを入力してください。",
		}
		templ.Handler(profiletpl.APIKeyFormModal(data)).ServeHTTP(w, r)
		return
	}

	secret, err := h.profile.CreateAPIKey(r.Context(), user.Token, profile.CreateAPIKeyRequest{Label: label})
	if err != nil {
		log.Printf("profile: create api key failed: %v", err)
		data := profiletpl.APIKeyFormData{
			CSRFToken: custommw.CSRFTokenFromContext(r.Context()),
			Error:     "APIキーの発行に失敗しました。時間を置いて再度お試しください。",
			Label:     label,
		}
		templ.Handler(profiletpl.APIKeyFormModal(data)).ServeHTTP(w, r)
		return
	}

	state, err := h.profile.SecurityOverview(r.Context(), user.Token)
	if err != nil {
		log.Printf("profile: refresh security overview failed: %v", err)
	}

	payload := profiletpl.APIKeyUpdateData{
		Security:  state,
		CSRFToken: custommw.CSRFTokenFromContext(r.Context()),
		Secret:    secret,
		Message:   "新しい API キーを発行しました。シークレットはこの画面でのみ表示されます。",
	}
	templ.Handler(profiletpl.APIKeyUpdate(payload)).ServeHTTP(w, r)
}

// RevokeAPIKey revokes the selected key.
func (h *Handlers) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	user, ok := custommw.UserFromContext(r.Context())
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	keyID := strings.TrimSpace(chi.URLParam(r, "keyID"))
	if keyID == "" {
		http.Error(w, "APIキーが指定されていません。", http.StatusBadRequest)
		return
	}

	state, err := h.profile.RevokeAPIKey(r.Context(), user.Token, keyID)
	if err != nil {
		log.Printf("profile: revoke api key failed: %v", err)
		http.Error(w, "APIキーの失効に失敗しました。", http.StatusBadGateway)
		return
	}

	payload := profiletpl.APIKeyUpdateData{
		Security:  state,
		CSRFToken: custommw.CSRFTokenFromContext(r.Context()),
		Message:   "選択した API キーを失効させました。",
	}
	templ.Handler(profiletpl.APIKeyUpdate(payload)).ServeHTTP(w, r)
}

// RevokeSession terminates an active session.
func (h *Handlers) RevokeSession(w http.ResponseWriter, r *http.Request) {
	user, ok := custommw.UserFromContext(r.Context())
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	sessionID := strings.TrimSpace(chi.URLParam(r, "sessionID"))
	if sessionID == "" {
		http.Error(w, "セッションが指定されていません。", http.StatusBadRequest)
		return
	}

	state, err := h.profile.RevokeSession(r.Context(), user.Token, sessionID)
	if err != nil {
		log.Printf("profile: revoke session failed: %v", err)
		http.Error(w, "セッションの失効に失敗しました。", http.StatusBadGateway)
		return
	}

	payload := profiletpl.SessionUpdateData{
		Security:  state,
		CSRFToken: custommw.CSRFTokenFromContext(r.Context()),
		Message:   "セッションを失効させました。",
	}
	templ.Handler(profiletpl.SessionUpdate(payload)).ServeHTTP(w, r)
}
