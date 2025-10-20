package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/services"
)

func TestMeHandlersGetProfile(t *testing.T) {
	now := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	lastSync := now.Add(10 * time.Minute)
	updated := now.Add(5 * time.Minute)
	avatar := "asset-123"

	profile := services.UserProfile{
		ID:                "user-1",
		DisplayName:       "Jane Doe",
		Email:             "Jane@example.com",
		PhoneNumber:       "+1-555-0100",
		PhotoURL:          "https://example.com/photo.png",
		AvatarAssetID:     &avatar,
		PreferredLanguage: "ja-JP",
		Locale:            "ja-JP",
		Roles:             []string{"user", "vip"},
		IsActive:          true,
		NotificationPrefs: domain.NotificationPreferences{"email": true, "sms": false},
		ProviderData: []domain.AuthProvider{
			{ProviderID: "password", UID: "user-1"},
			{ProviderID: "google.com", UID: "google-123", Email: "sample@gmail.com"},
		},
		CreatedAt:    now,
		UpdatedAt:    updated,
		LastSyncTime: lastSync,
	}

	svc := &stubUserService{
		getProfileFunc: func(ctx context.Context, userID string) (services.UserProfile, error) {
			if userID != "user-1" {
				return services.UserProfile{}, errors.New("not found")
			}
			return profile, nil
		},
	}

	handler := NewMeHandlers(nil, svc)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	identity := &auth.Identity{
		UID:    "user-1",
		Email:  "IDENTITY@example.com",
		Locale: "en-US",
		Roles:  []string{"user"},
	}
	req = req.WithContext(auth.WithIdentity(req.Context(), identity))

	rr := httptest.NewRecorder()
	handler.getProfile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp meResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON response: %v", err)
	}

	got := resp.Profile
	if got.ID != profile.ID {
		t.Fatalf("expected id %q, got %q", profile.ID, got.ID)
	}
	if got.DisplayName != profile.DisplayName {
		t.Fatalf("expected display name %q, got %q", profile.DisplayName, got.DisplayName)
	}
	if got.Email != "jane@example.com" {
		t.Fatalf("expected lower-cased email, got %q", got.Email)
	}
	if !reflect.DeepEqual(got.Roles, profile.Roles) {
		t.Fatalf("expected roles %v, got %v", profile.Roles, got.Roles)
	}
	if got.HasPassword != true {
		t.Fatalf("expected has_password true")
	}
	if got.AvatarAssetID == nil || *got.AvatarAssetID != avatar {
		t.Fatalf("expected avatar asset id %q, got %v", avatar, got.AvatarAssetID)
	}
	if got.CreatedAt != now.UTC().Format(time.RFC3339Nano) {
		t.Fatalf("unexpected created_at %q", got.CreatedAt)
	}
	if got.LastSyncTime != lastSync.UTC().Format(time.RFC3339Nano) {
		t.Fatalf("unexpected last_sync_time %q", got.LastSyncTime)
	}
	if got.NotificationPrefs == nil {
		t.Fatalf("expected notification prefs map, got nil")
	}
	if got.NotificationPrefs["email"] != true || got.NotificationPrefs["sms"] != false {
		t.Fatalf("unexpected notification prefs %#v", got.NotificationPrefs)
	}
	if len(got.ProviderData) != len(profile.ProviderData) {
		t.Fatalf("expected %d providers, got %d", len(profile.ProviderData), len(got.ProviderData))
	}
}

func TestMeHandlersUpdateProfile(t *testing.T) {
	now := time.Date(2024, 4, 5, 15, 0, 0, 0, time.UTC)
	lastSync := now.Add(30 * time.Minute)
	avatar := "avatar-old"
	profile := services.UserProfile{
		ID:                "user-2",
		DisplayName:       "Initial Name",
		Email:             "initial@example.com",
		PreferredLanguage: "ja-JP",
		Locale:            "ja-JP",
		IsActive:          true,
		AvatarAssetID:     &avatar,
		LastSyncTime:      lastSync,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	var captured services.UpdateProfileCommand
	updatedAvatar := "asset-789"
	svc := &stubUserService{
		getProfileFunc: func(ctx context.Context, userID string) (services.UserProfile, error) {
			return profile, nil
		},
		updateProfileFunc: func(ctx context.Context, cmd services.UpdateProfileCommand) (services.UserProfile, error) {
			captured = cmd
			updated := profile
			updated.DisplayName = strings.TrimSpace(*cmd.DisplayName)
			if cmd.PreferredLanguage != nil {
				updated.PreferredLanguage = strings.TrimSpace(*cmd.PreferredLanguage)
			}
			if cmd.Locale != nil {
				updated.Locale = strings.TrimSpace(*cmd.Locale)
			}
			if len(cmd.NotificationPrefs) > 0 {
				updated.NotificationPrefs = domain.NotificationPreferences(cmd.NotificationPrefs)
			} else {
				updated.NotificationPrefs = nil
			}
			if cmd.AvatarAssetID != nil {
				val := strings.TrimSpace(*cmd.AvatarAssetID)
				if val == "" {
					updated.AvatarAssetID = nil
				} else {
					updated.AvatarAssetID = &val
				}
			}
			newSync := profile.LastSyncTime.Add(time.Minute)
			updated.LastSyncTime = newSync
			updated.UpdatedAt = newSync
			updated.AvatarAssetID = &updatedAvatar
			return updated, nil
		},
	}

	handler := NewMeHandlers(nil, svc)

	body := map[string]any{
		"display_name":       "  Updated Name ",
		"preferred_language": "en",
		"locale":             nil,
		"notification_prefs": map[string]bool{"marketing.email": true},
		"avatar_asset_id":    " asset-789 ",
		"last_sync_time":     lastSync.Format(time.RFC3339Nano),
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/me", bytes.NewReader(payload))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{
		UID:   "user-2",
		Email: "user2@example.com",
		Roles: []string{"user"},
	}))

	rr := httptest.NewRecorder()
	handler.updateProfile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	if captured.UserID != "user-2" || captured.ActorID != "user-2" {
		t.Fatalf("expected command user and actor id user-2, got %#v", captured)
	}
	if captured.DisplayName == nil || strings.TrimSpace(*captured.DisplayName) != "Updated Name" {
		t.Fatalf("expected display name Updated Name, got %v", captured.DisplayName)
	}
	if captured.PreferredLanguage == nil || *captured.PreferredLanguage != "en" {
		t.Fatalf("expected preferred language en, got %v", captured.PreferredLanguage)
	}
	if captured.Locale == nil || *captured.Locale != "" {
		t.Fatalf("expected locale cleared, got %v", captured.Locale)
	}
	if captured.NotificationPrefs == nil || captured.NotificationPrefs["marketing.email"] != true {
		t.Fatalf("expected notification prefs with marketing.email true, got %#v", captured.NotificationPrefs)
	}
	if !captured.NotificationPrefsSet {
		t.Fatalf("expected notification prefs flag set")
	}
	if captured.AvatarAssetID == nil {
		t.Fatalf("expected avatar asset id pointer")
	}
	if !captured.AvatarAssetIDSet {
		t.Fatalf("expected avatar asset id flag set")
	}
	if captured.ExpectedSyncTime == nil || !captured.ExpectedSyncTime.Equal(lastSync) {
		t.Fatalf("expected expected sync time %s, got %v", lastSync, captured.ExpectedSyncTime)
	}

	var resp meResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON response: %v", err)
	}

	if resp.Profile.DisplayName != "Updated Name" {
		t.Fatalf("unexpected display name %q", resp.Profile.DisplayName)
	}
	if resp.Profile.PreferredLanguage != "en" {
		t.Fatalf("unexpected preferred language %q", resp.Profile.PreferredLanguage)
	}
	if resp.Profile.Locale != "" {
		t.Fatalf("expected locale cleared, got %q", resp.Profile.Locale)
	}
	if resp.Profile.AvatarAssetID == nil || *resp.Profile.AvatarAssetID != updatedAvatar {
		t.Fatalf("unexpected avatar asset id %v", resp.Profile.AvatarAssetID)
	}
	if resp.Profile.NotificationPrefs == nil || resp.Profile.NotificationPrefs["marketing.email"] != true {
		t.Fatalf("unexpected notification prefs %#v", resp.Profile.NotificationPrefs)
	}
}

func TestMeHandlersListPaymentMethods(t *testing.T) {
	svc := &stubUserService{
		listPaymentMethodsFunc: func(ctx context.Context, userID string) ([]services.PaymentMethod, error) {
			if userID != "user-1" {
				t.Fatalf("expected user-1, got %s", userID)
			}
			return []services.PaymentMethod{
				{ID: "pm_1", Provider: "stripe", Token: "pm_token", Brand: "visa", Last4: "4242", ExpMonth: 12, ExpYear: 2030, IsDefault: true},
			}, nil
		},
	}
	handler := NewMeHandlers(nil, svc)

	req := httptest.NewRequest(http.MethodGet, "/me/payment-methods", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	rr := httptest.NewRecorder()
	handler.listPaymentMethods(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var payload []paymentMethodPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload) != 1 {
		t.Fatalf("expected 1 payment method, got %d", len(payload))
	}
	if payload[0].ID != "pm_1" || payload[0].Provider != "stripe" || !payload[0].IsDefault {
		t.Fatalf("unexpected payload %#v", payload[0])
	}
}

func TestMeHandlersCreatePaymentMethod(t *testing.T) {
	var captured services.AddPaymentMethodCommand
	svc := &stubUserService{
		addPaymentMethodFunc: func(ctx context.Context, cmd services.AddPaymentMethodCommand) (services.PaymentMethod, error) {
			captured = cmd
			return services.PaymentMethod{
				ID:        "pm_new",
				Provider:  cmd.Provider,
				Token:     cmd.Token,
				IsDefault: true,
			}, nil
		},
	}
	handler := NewMeHandlers(nil, svc)

	body := `{"provider":"stripe","token":"pm_new","make_default":true}`
	req := httptest.NewRequest(http.MethodPost, "/me/payment-methods", strings.NewReader(body))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-2"}))

	rr := httptest.NewRecorder()
	handler.createPaymentMethod(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rr.Code)
	}
	if captured.UserID != "user-2" || captured.Provider != "stripe" || captured.Token != "pm_new" || !captured.MakeDefault {
		t.Fatalf("unexpected command %#v", captured)
	}
	if rr.Header().Get("Location") == "" {
		t.Fatalf("expected location header")
	}

	var payload paymentMethodPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.ID != "pm_new" || !payload.IsDefault {
		t.Fatalf("unexpected payload %#v", payload)
	}
}

func TestMeHandlersDeletePaymentMethod(t *testing.T) {
	var removed services.RemovePaymentMethodCommand
	svc := &stubUserService{
		removePaymentMethodFunc: func(ctx context.Context, cmd services.RemovePaymentMethodCommand) error {
			removed = cmd
			return nil
		},
	}
	handler := NewMeHandlers(nil, svc)

	req := httptest.NewRequest(http.MethodDelete, "/me/payment-methods/pm_123", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-3"}))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("paymentMethodID", "pm_123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rr := httptest.NewRecorder()
	handler.deletePaymentMethod(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rr.Code)
	}
	if removed.UserID != "user-3" || removed.PaymentMethodID != "pm_123" {
		t.Fatalf("unexpected remove command %#v", removed)
	}
}

func TestMeHandlersListFavorites(t *testing.T) {
	addedAt := time.Date(2024, 9, 1, 12, 0, 0, 0, time.UTC)
	svc := &stubUserService{
		listFavoritesFunc: func(ctx context.Context, userID string, pager services.Pagination) (domain.CursorPage[services.FavoriteDesign], error) {
			if userID != "user-1" {
				t.Fatalf("expected user-1, got %s", userID)
			}
			return domain.CursorPage[services.FavoriteDesign]{
				Items: []services.FavoriteDesign{
					{
						DesignID: "design-1",
						AddedAt:  addedAt,
						Design: &services.Design{
							ID:        "design-1",
							OwnerID:   "user-1",
							Status:    "draft",
							Template:  "template-a",
							Locale:    "ja-JP",
							UpdatedAt: addedAt,
						},
					},
				},
			}, nil
		},
	}
	handler := NewMeHandlers(nil, svc)

	req := httptest.NewRequest(http.MethodGet, "/me/favorites", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	rr := httptest.NewRecorder()
	handler.listFavorites(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var payload []favoritePayload
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload) != 1 {
		t.Fatalf("expected 1 favorite, got %d", len(payload))
	}
	if payload[0].DesignID != "design-1" || payload[0].AddedAt == "" {
		t.Fatalf("unexpected payload %#v", payload[0])
	}
	if payload[0].Design == nil || payload[0].Design.ID != "design-1" {
		t.Fatalf("expected design metadata, got %#v", payload[0].Design)
	}
}

func TestMeHandlersAddFavorite(t *testing.T) {
	var captured services.ToggleFavoriteCommand
	svc := &stubUserService{
		toggleFavoriteFunc: func(ctx context.Context, cmd services.ToggleFavoriteCommand) error {
			captured = cmd
			return nil
		},
	}
	handler := NewMeHandlers(nil, svc)

	req := httptest.NewRequest(http.MethodPut, "/me/favorites/design-1", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-2"}))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "design-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rr := httptest.NewRecorder()
	handler.addFavorite(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rr.Code)
	}
	if captured.UserID != "user-2" || captured.DesignID != "design-1" || !captured.Mark {
		t.Fatalf("unexpected command %#v", captured)
	}
}

func TestMeHandlersRemoveFavorite(t *testing.T) {
	var captured services.ToggleFavoriteCommand
	svc := &stubUserService{
		toggleFavoriteFunc: func(ctx context.Context, cmd services.ToggleFavoriteCommand) error {
			captured = cmd
			return nil
		},
	}
	handler := NewMeHandlers(nil, svc)

	req := httptest.NewRequest(http.MethodDelete, "/me/favorites/design-2", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-3"}))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("designID", "design-2")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rr := httptest.NewRecorder()
	handler.removeFavorite(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rr.Code)
	}
	if captured.UserID != "user-3" || captured.DesignID != "design-2" || captured.Mark {
		t.Fatalf("unexpected command %#v", captured)
	}
}

func TestMeHandlersUpdateProfileRejectsDisallowedField(t *testing.T) {
	svc := &stubUserService{
		getProfileFunc: func(ctx context.Context, userID string) (services.UserProfile, error) {
			return services.UserProfile{ID: "user-3", LastSyncTime: time.Now().UTC()}, nil
		},
	}
	handler := NewMeHandlers(nil, svc)

	body := []byte(`{"role":"admin"}`)
	req := httptest.NewRequest(http.MethodPut, "/me", bytes.NewReader(body))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{
		UID:   "user-3",
		Email: "test@example.com",
		Roles: []string{"user"},
	}))

	rr := httptest.NewRecorder()
	handler.updateProfile(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}

	var errPayload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &errPayload); err != nil {
		t.Fatalf("expected JSON error: %v", err)
	}
	if errPayload["error"] != "invalid_request" {
		t.Fatalf("expected error code invalid_request, got %v", errPayload["error"])
	}
}

func TestMeHandlersUpdateProfileClearsAvatar(t *testing.T) {
	lastSync := time.Date(2024, 5, 10, 9, 0, 0, 0, time.UTC)
	avatar := "asset-old"
	profile := services.UserProfile{
		ID:            "user-4",
		AvatarAssetID: &avatar,
		LastSyncTime:  lastSync,
		CreatedAt:     lastSync.Add(-time.Hour),
		UpdatedAt:     lastSync,
	}

	var captured services.UpdateProfileCommand
	svc := &stubUserService{
		getProfileFunc: func(ctx context.Context, userID string) (services.UserProfile, error) {
			return profile, nil
		},
		updateProfileFunc: func(ctx context.Context, cmd services.UpdateProfileCommand) (services.UserProfile, error) {
			captured = cmd
			updated := profile
			updated.AvatarAssetID = nil
			updated.NotificationPrefs = nil
			return updated, nil
		},
	}

	handler := NewMeHandlers(nil, svc)

	body := map[string]any{
		"avatar_asset_id":    nil,
		"notification_prefs": nil,
		"last_sync_time":     lastSync.Format(time.RFC3339Nano),
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/me", bytes.NewReader(payload))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{
		UID:   "user-4",
		Email: "user4@example.com",
		Roles: []string{"user"},
	}))

	rr := httptest.NewRecorder()
	handler.updateProfile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	if !captured.AvatarAssetIDSet {
		t.Fatalf("expected avatar asset id flag set")
	}
	if captured.AvatarAssetID != nil {
		t.Fatalf("expected avatar asset id cleared, got %v", captured.AvatarAssetID)
	}
	if !captured.NotificationPrefsSet {
		t.Fatalf("expected notification prefs flag set")
	}
	if captured.NotificationPrefs != nil {
		t.Fatalf("expected notification prefs cleared, got %#v", captured.NotificationPrefs)
	}

	var resp meResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON response: %v", err)
	}
	if resp.Profile.AvatarAssetID != nil {
		t.Fatalf("expected avatar_asset_id null, got %v", resp.Profile.AvatarAssetID)
	}
	if resp.Profile.NotificationPrefs == nil || len(resp.Profile.NotificationPrefs) != 0 {
		t.Fatalf("expected empty notification prefs map, got %#v", resp.Profile.NotificationPrefs)
	}
}

func TestMeHandlersUpdateProfileConflict(t *testing.T) {
	svc := &stubUserService{
		getProfileFunc: func(ctx context.Context, userID string) (services.UserProfile, error) {
			return services.UserProfile{ID: userID, LastSyncTime: time.Now().UTC()}, nil
		},
		updateProfileFunc: func(ctx context.Context, cmd services.UpdateProfileCommand) (services.UserProfile, error) {
			return services.UserProfile{}, services.ErrUserProfileConflict
		},
	}
	handler := NewMeHandlers(nil, svc)

	body := []byte(`{"display_name":"Another","last_sync_time":"2024-05-10T09:00:00Z"}`)
	req := httptest.NewRequest(http.MethodPut, "/me", bytes.NewReader(body))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{
		UID:   "user-5",
		Email: "user5@example.com",
		Roles: []string{"user"},
	}))

	rr := httptest.NewRecorder()
	handler.updateProfile(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rr.Code)
	}

	var errPayload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &errPayload); err != nil {
		t.Fatalf("expected JSON error: %v", err)
	}
	if errPayload["error"] != "profile_conflict" {
		t.Fatalf("expected error code profile_conflict, got %v", errPayload["error"])
	}
}

type stubUserService struct {
	getProfileFunc          func(ctx context.Context, userID string) (services.UserProfile, error)
	updateProfileFunc       func(ctx context.Context, cmd services.UpdateProfileCommand) (services.UserProfile, error)
	listAddressesFunc       func(ctx context.Context, userID string) ([]services.Address, error)
	upsertAddressFunc       func(ctx context.Context, cmd services.UpsertAddressCommand) (services.Address, error)
	deleteAddressFunc       func(ctx context.Context, cmd services.DeleteAddressCommand) error
	listPaymentMethodsFunc  func(ctx context.Context, userID string) ([]services.PaymentMethod, error)
	addPaymentMethodFunc    func(ctx context.Context, cmd services.AddPaymentMethodCommand) (services.PaymentMethod, error)
	removePaymentMethodFunc func(ctx context.Context, cmd services.RemovePaymentMethodCommand) error
	listFavoritesFunc       func(ctx context.Context, userID string, pager services.Pagination) (domain.CursorPage[services.FavoriteDesign], error)
	toggleFavoriteFunc      func(ctx context.Context, cmd services.ToggleFavoriteCommand) error
}

func (s *stubUserService) GetProfile(ctx context.Context, userID string) (services.UserProfile, error) {
	if s != nil && s.getProfileFunc != nil {
		return s.getProfileFunc(ctx, userID)
	}
	return services.UserProfile{}, errors.New("not implemented")
}

func (s *stubUserService) GetByUID(ctx context.Context, userID string) (services.UserProfile, error) {
	return s.GetProfile(ctx, userID)
}

func (s *stubUserService) UpdateProfile(ctx context.Context, cmd services.UpdateProfileCommand) (services.UserProfile, error) {
	if s != nil && s.updateProfileFunc != nil {
		return s.updateProfileFunc(ctx, cmd)
	}
	return services.UserProfile{}, errors.New("not implemented")
}

func (s *stubUserService) MaskProfile(ctx context.Context, cmd services.MaskProfileCommand) (services.UserProfile, error) {
	return services.UserProfile{}, errors.New("not implemented")
}

func (s *stubUserService) SetUserActive(ctx context.Context, cmd services.SetUserActiveCommand) (services.UserProfile, error) {
	return services.UserProfile{}, errors.New("not implemented")
}

func (s *stubUserService) ListAddresses(ctx context.Context, userID string) ([]services.Address, error) {
	if s != nil && s.listAddressesFunc != nil {
		return s.listAddressesFunc(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func (s *stubUserService) UpsertAddress(ctx context.Context, cmd services.UpsertAddressCommand) (services.Address, error) {
	if s != nil && s.upsertAddressFunc != nil {
		return s.upsertAddressFunc(ctx, cmd)
	}
	return services.Address{}, errors.New("not implemented")
}

func (s *stubUserService) DeleteAddress(ctx context.Context, cmd services.DeleteAddressCommand) error {
	if s != nil && s.deleteAddressFunc != nil {
		return s.deleteAddressFunc(ctx, cmd)
	}
	return errors.New("not implemented")
}

func (s *stubUserService) ListPaymentMethods(ctx context.Context, userID string) ([]services.PaymentMethod, error) {
	if s != nil && s.listPaymentMethodsFunc != nil {
		return s.listPaymentMethodsFunc(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func (s *stubUserService) AddPaymentMethod(ctx context.Context, cmd services.AddPaymentMethodCommand) (services.PaymentMethod, error) {
	if s != nil && s.addPaymentMethodFunc != nil {
		return s.addPaymentMethodFunc(ctx, cmd)
	}
	return services.PaymentMethod{}, errors.New("not implemented")
}

func (s *stubUserService) RemovePaymentMethod(ctx context.Context, cmd services.RemovePaymentMethodCommand) error {
	if s != nil && s.removePaymentMethodFunc != nil {
		return s.removePaymentMethodFunc(ctx, cmd)
	}
	return errors.New("not implemented")
}

func (s *stubUserService) ListFavorites(ctx context.Context, userID string, pager services.Pagination) (domain.CursorPage[services.FavoriteDesign], error) {
	if s != nil && s.listFavoritesFunc != nil {
		return s.listFavoritesFunc(ctx, userID, pager)
	}
	return domain.CursorPage[services.FavoriteDesign]{}, errors.New("not implemented")
}

func (s *stubUserService) ToggleFavorite(ctx context.Context, cmd services.ToggleFavoriteCommand) error {
	if s != nil && s.toggleFavoriteFunc != nil {
		return s.toggleFavoriteFunc(ctx, cmd)
	}
	return errors.New("not implemented")
}
