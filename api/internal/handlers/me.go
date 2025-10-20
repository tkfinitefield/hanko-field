package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	firebaseauth "firebase.google.com/go/v4/auth"
	"github.com/go-chi/chi/v5"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/platform/httpx"
	"github.com/hanko-field/api/internal/repositories"
	"github.com/hanko-field/api/internal/services"
)

const maxProfileBodySize = 64 * 1024

var (
	errBodyTooLarge     = errors.New("request body too large")
	errEmptyBody        = errors.New("request body is required")
	errNoEditableFields = errors.New("no editable fields provided")
)

// MeHandlers exposes authenticated profile endpoints for the current user.
type MeHandlers struct {
	authn *auth.Authenticator
	users services.UserService
}

// NewMeHandlers constructs handlers enforcing Firebase authentication before invoking the user service.
func NewMeHandlers(authn *auth.Authenticator, users services.UserService) *MeHandlers {
	return &MeHandlers{
		authn: authn,
		users: users,
	}
}

// Routes wires the /me endpoints onto the provided router.
func (h *MeHandlers) Routes(r chi.Router) {
	if r == nil {
		return
	}
	if h.authn != nil {
		r.Use(h.authn.RequireFirebaseAuth())
	}
	r.Get("/", h.getProfile)
	r.Put("/", h.updateProfile)
	r.Route("/addresses", h.addressRoutes)
	r.Route("/payment-methods", h.paymentMethodRoutes)
	r.Route("/favorites", h.favoriteRoutes)
}

func (h *MeHandlers) getProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.users == nil {
		httpx.WriteError(ctx, w, httpx.NewError("profile_service_unavailable", "profile service is unavailable", http.StatusServiceUnavailable))
		return
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}

	profile, err := h.users.GetProfile(ctx, identity.UID)
	if err != nil {
		writeUserProfileError(ctx, w, err)
		return
	}

	record, _ := identity.User(ctx)

	payload := meResponse{Profile: buildProfilePayload(profile, identity, record)}
	writeJSONResponse(w, http.StatusOK, payload)
}

func (h *MeHandlers) updateProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.users == nil {
		httpx.WriteError(ctx, w, httpx.NewError("profile_service_unavailable", "profile service is unavailable", http.StatusServiceUnavailable))
		return
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}

	body, err := readLimitedBody(r, maxProfileBodySize)
	if err != nil {
		switch {
		case errors.Is(err, errBodyTooLarge):
			httpx.WriteError(ctx, w, httpx.NewError("payload_too_large", "request body exceeds allowed size", http.StatusRequestEntityTooLarge))
		default:
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		}
		return
	}

	updateReq, err := parseUpdateProfileRequest(body)
	if err != nil {
		switch {
		case errors.Is(err, errEmptyBody), errors.Is(err, errNoEditableFields):
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		default:
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		}
		return
	}

	cmd := services.UpdateProfileCommand{
		UserID:  identity.UID,
		ActorID: identity.UID,
	}
	if updateReq.hasDisplayName {
		cmd.DisplayName = updateReq.displayName
	}
	if updateReq.hasPreferredLanguage {
		cmd.PreferredLanguage = updateReq.preferredLanguage
	}
	if updateReq.hasLocale {
		cmd.Locale = updateReq.locale
	}
	if updateReq.hasNotificationPrefs {
		cmd.NotificationPrefsSet = true
		cmd.NotificationPrefs = cloneNotificationPrefs(updateReq.notificationPrefs)
	}
	if updateReq.hasAvatarAssetID {
		cmd.AvatarAssetIDSet = true
		cmd.AvatarAssetID = cloneStringPointer(updateReq.avatarAssetID)
	}
	if updateReq.expectedSync != nil {
		cmd.ExpectedSyncTime = updateReq.expectedSync
	}

	updated, err := h.users.UpdateProfile(ctx, cmd)
	if err != nil {
		writeUserProfileError(ctx, w, err)
		return
	}

	record, _ := identity.User(ctx)

	payload := meResponse{Profile: buildProfilePayload(updated, identity, record)}
	writeJSONResponse(w, http.StatusOK, payload)
}

type meResponse struct {
	Profile meProfilePayload `json:"profile"`
}

type meProfilePayload struct {
	ID                string            `json:"id"`
	DisplayName       string            `json:"display_name"`
	Email             string            `json:"email"`
	EmailVerified     bool              `json:"email_verified"`
	PhoneNumber       string            `json:"phone_number,omitempty"`
	PhotoURL          string            `json:"photo_url,omitempty"`
	AvatarAssetID     *string           `json:"avatar_asset_id"`
	PreferredLanguage string            `json:"preferred_language,omitempty"`
	Locale            string            `json:"locale,omitempty"`
	Roles             []string          `json:"roles"`
	IsActive          bool              `json:"is_active"`
	HasPassword       bool              `json:"has_password"`
	NotificationPrefs map[string]bool   `json:"notification_prefs"`
	ProviderData      []providerPayload `json:"provider_data,omitempty"`
	CreatedAt         string            `json:"created_at,omitempty"`
	UpdatedAt         string            `json:"updated_at,omitempty"`
	OnboardedAt       string            `json:"onboarded_at,omitempty"`
	PiiMaskedAt       string            `json:"pii_masked_at,omitempty"`
	LastSyncTime      string            `json:"last_sync_time,omitempty"`
}

type providerPayload struct {
	ProviderID  string `json:"provider_id"`
	UID         string `json:"uid"`
	Email       string `json:"email,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
	PhotoURL    string `json:"photo_url,omitempty"`
}

type updateProfileRequest struct {
	displayName          *string
	preferredLanguage    *string
	locale               *string
	notificationPrefs    map[string]bool
	avatarAssetID        *string
	expectedSync         *time.Time
	hasDisplayName       bool
	hasPreferredLanguage bool
	hasLocale            bool
	hasNotificationPrefs bool
	hasAvatarAssetID     bool
}

func readLimitedBody(r *http.Request, limit int64) ([]byte, error) {
	if r == nil || r.Body == nil {
		return nil, errEmptyBody
	}
	if limit <= 0 {
		limit = maxProfileBodySize
	}
	reader := io.LimitReader(r.Body, limit+1)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, errEmptyBody
	}
	if int64(len(data)) > limit {
		return nil, errBodyTooLarge
	}
	return data, nil
}

func parseUpdateProfileRequest(data []byte) (updateProfileRequest, error) {
	var req updateProfileRequest
	if len(strings.TrimSpace(string(data))) == 0 {
		return req, errEmptyBody
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return req, fmt.Errorf("invalid JSON payload: %w", err)
	}
	if len(raw) == 0 {
		return req, errNoEditableFields
	}

	updateFields := 0
	for key, value := range raw {
		switch key {
		case "display_name":
			if isJSONNull(value) {
				return req, errors.New("display_name must not be null")
			}
			var name string
			if err := json.Unmarshal(value, &name); err != nil {
				return req, errors.New("display_name must be a string")
			}
			req.displayName = &name
			req.hasDisplayName = true
			updateFields++
		case "preferred_language":
			if isJSONNull(value) {
				empty := ""
				req.preferredLanguage = &empty
			} else {
				var lang string
				if err := json.Unmarshal(value, &lang); err != nil {
					return req, errors.New("preferred_language must be a string")
				}
				req.preferredLanguage = &lang
			}
			req.hasPreferredLanguage = true
			updateFields++
		case "locale":
			if isJSONNull(value) {
				empty := ""
				req.locale = &empty
			} else {
				var locale string
				if err := json.Unmarshal(value, &locale); err != nil {
					return req, errors.New("locale must be a string")
				}
				req.locale = &locale
			}
			req.hasLocale = true
			updateFields++
		case "notification_prefs":
			req.hasNotificationPrefs = true
			updateFields++
			if isJSONNull(value) {
				req.notificationPrefs = nil
				continue
			}
			var prefs map[string]bool
			if err := json.Unmarshal(value, &prefs); err != nil {
				return req, errors.New("notification_prefs must be an object with boolean values")
			}
			if prefs == nil {
				prefs = map[string]bool{}
			}
			req.notificationPrefs = prefs
		case "avatar_asset_id":
			req.hasAvatarAssetID = true
			updateFields++
			if isJSONNull(value) {
				req.avatarAssetID = nil
				continue
			}
			var asset string
			if err := json.Unmarshal(value, &asset); err != nil {
				return req, errors.New("avatar_asset_id must be a string or null")
			}
			req.avatarAssetID = &asset
		case "last_sync_time":
			if isJSONNull(value) {
				return req, errors.New("last_sync_time must be a string")
			}
			var ts string
			if err := json.Unmarshal(value, &ts); err != nil {
				return req, errors.New("last_sync_time must be a string")
			}
			parsed, err := parseRFC3339(ts)
			if err != nil {
				return req, fmt.Errorf("last_sync_time must be RFC3339 timestamp: %w", err)
			}
			req.expectedSync = &parsed
		default:
			return req, fmt.Errorf("field %q is not editable", key)
		}
	}

	if updateFields == 0 {
		return req, errNoEditableFields
	}

	return req, nil
}

func isJSONNull(value json.RawMessage) bool {
	return strings.EqualFold(strings.TrimSpace(string(value)), "null")
}

func parseRFC3339(value string) (time.Time, error) {
	layouts := []string{time.RFC3339Nano, time.RFC3339}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid time %q", value)
}

func buildProfilePayload(profile services.UserProfile, identity *auth.Identity, record *firebaseauth.UserRecord) meProfilePayload {
	email := strings.TrimSpace(strings.ToLower(profile.Email))
	if email == "" && identity != nil {
		email = strings.TrimSpace(strings.ToLower(identity.Email))
	}

	locale := strings.TrimSpace(profile.Locale)
	if locale == "" && identity != nil {
		locale = strings.TrimSpace(identity.Locale)
	}

	preferredLanguage := strings.TrimSpace(profile.PreferredLanguage)

	roles := slices.Clone(profile.Roles)
	if len(roles) == 0 && identity != nil {
		roles = slices.Clone(identity.Roles)
	}
	if len(roles) == 0 {
		roles = []string{}
	}

	notificationPrefs := cloneNotificationPrefs(profile.NotificationPrefs)
	if notificationPrefs == nil {
		notificationPrefs = map[string]bool{}
	}
	providers := providerPayloads(profile.ProviderData)
	if len(providers) == 0 && record != nil {
		providers = providerPayloads(providersFromRecord(record))
	}

	emailVerified := false
	if record != nil {
		emailVerified = record.EmailVerified
	}

	hasPassword := deriveHasPassword(profile.ProviderData, record)

	onboardedAt := formatTime(profile.CreatedAt)
	if meta := userCreationTime(record); !meta.IsZero() {
		onboardedAt = formatTime(meta)
	}

	return meProfilePayload{
		ID:                strings.TrimSpace(profile.ID),
		DisplayName:       profile.DisplayName,
		Email:             email,
		EmailVerified:     emailVerified,
		PhoneNumber:       profile.PhoneNumber,
		PhotoURL:          profile.PhotoURL,
		AvatarAssetID:     profile.AvatarAssetID,
		PreferredLanguage: preferredLanguage,
		Locale:            locale,
		Roles:             roles,
		IsActive:          profile.IsActive,
		HasPassword:       hasPassword,
		NotificationPrefs: notificationPrefs,
		ProviderData:      providers,
		CreatedAt:         formatTime(profile.CreatedAt),
		UpdatedAt:         formatTime(profile.UpdatedAt),
		OnboardedAt:       onboardedAt,
		PiiMaskedAt:       formatTime(pointerTime(profile.PiiMaskedAt)),
		LastSyncTime:      formatTime(profile.LastSyncTime),
	}
}

func deriveHasPassword(providers []domain.AuthProvider, record *firebaseauth.UserRecord) bool {
	for _, provider := range providers {
		if strings.EqualFold(strings.TrimSpace(provider.ProviderID), "password") {
			return true
		}
	}
	if record == nil {
		return false
	}
	if info := record.UserInfo; info != nil && strings.EqualFold(strings.TrimSpace(info.ProviderID), "password") {
		return true
	}
	for _, info := range record.ProviderUserInfo {
		if info != nil && strings.EqualFold(strings.TrimSpace(info.ProviderID), "password") {
			return true
		}
	}
	return false
}

func userCreationTime(record *firebaseauth.UserRecord) time.Time {
	if record == nil || record.UserMetadata == nil {
		return time.Time{}
	}
	timestamp := record.UserMetadata.CreationTimestamp
	if timestamp <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(timestamp).UTC()
}

func providerPayloads(providers []domain.AuthProvider) []providerPayload {
	if len(providers) == 0 {
		return nil
	}
	payload := make([]providerPayload, 0, len(providers))
	for _, provider := range providers {
		payload = append(payload, providerPayload{
			ProviderID:  provider.ProviderID,
			UID:         provider.UID,
			Email:       provider.Email,
			DisplayName: provider.DisplayName,
			PhoneNumber: provider.PhoneNumber,
			PhotoURL:    provider.PhotoURL,
		})
	}
	return payload
}

func providersFromRecord(record *firebaseauth.UserRecord) []domain.AuthProvider {
	if record == nil {
		return nil
	}
	type seenKey struct {
		provider string
		uid      string
	}
	seen := make(map[seenKey]struct{})
	appendProvider := func(info *firebaseauth.UserInfo, providers *[]domain.AuthProvider) {
		if info == nil {
			return
		}
		key := seenKey{
			provider: strings.TrimSpace(info.ProviderID),
			uid:      strings.TrimSpace(info.UID),
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		*providers = append(*providers, domain.AuthProvider{
			ProviderID:  key.provider,
			UID:         key.uid,
			Email:       strings.TrimSpace(strings.ToLower(info.Email)),
			DisplayName: strings.TrimSpace(info.DisplayName),
			PhoneNumber: strings.TrimSpace(info.PhoneNumber),
			PhotoURL:    strings.TrimSpace(info.PhotoURL),
		})
	}

	var providers []domain.AuthProvider
	appendProvider(record.UserInfo, &providers)
	for _, info := range record.ProviderUserInfo {
		appendProvider(info, &providers)
	}
	return providers
}

func cloneNotificationPrefs(prefs map[string]bool) map[string]bool {
	if prefs == nil {
		return nil
	}
	if len(prefs) == 0 {
		return map[string]bool{}
	}
	cloned := make(map[string]bool, len(prefs))
	for key, value := range prefs {
		cloned[key] = value
	}
	return cloned
}

func pointerTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func writeJSONResponse(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func writeUserProfileError(ctx context.Context, w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	switch {
	case errors.Is(err, services.ErrUserProfileConflict):
		httpx.WriteError(ctx, w, httpx.NewError("profile_conflict", "profile has changed; refresh and retry", http.StatusConflict))
		return
	case errors.Is(err, services.ErrUserInvalidDisplayName),
		errors.Is(err, services.ErrUserInvalidLanguageTag),
		errors.Is(err, services.ErrUserInvalidNotificationKey):
		httpx.WriteError(ctx, w, httpx.NewError("invalid_profile_field", err.Error(), http.StatusBadRequest))
		return
	}

	var repoErr repositories.RepositoryError
	if errors.As(err, &repoErr) {
		switch {
		case repoErr.IsNotFound():
			httpx.WriteError(ctx, w, httpx.NewError("profile_not_found", "profile not found", http.StatusNotFound))
			return
		case repoErr.IsUnavailable():
			httpx.WriteError(ctx, w, httpx.NewError("profile_service_unavailable", "profile repository unavailable", http.StatusServiceUnavailable))
			return
		default:
			httpx.WriteError(ctx, w, httpx.NewError("profile_error", err.Error(), http.StatusInternalServerError))
			return
		}
	}

	httpx.WriteError(ctx, w, httpx.NewError("profile_error", err.Error(), http.StatusInternalServerError))
}
