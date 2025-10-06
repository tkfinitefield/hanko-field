package services

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	firebaseauth "firebase.google.com/go/v4/auth"
	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/repositories"
	"golang.org/x/text/language"
)

var (
	errUserIDRequired               = errors.New("user: user id is required")
	errActorIDRequired              = errors.New("user: actor id is required")
	errInvalidDisplayName           = errors.New("user: invalid display name")
	errInvalidLanguageTag           = errors.New("user: invalid language tag")
	errProfileConflict              = errors.New("user: profile has been modified")
	errAddressRepositoryUnavailable = errors.New("user: address repository not configured")
	errPaymentMethodsNotImplemented = errors.New("user: payment method operations not yet implemented")
	errFavoritesNotImplemented      = errors.New("user: favorites operations not yet implemented")
	emailMaskSuffix                 = "@hanko-field.invalid"
	notificationKeyPattern          = regexp.MustCompile(`^[a-z0-9_.-]{1,40}$`)
	auditActionProfileUpdate        = "user.profile.update"
	auditActionProfileMask          = "user.profile.mask"
	auditActionProfileActivate      = "user.profile.activate"
	auditActionProfileDeactivate    = "user.profile.deactivate"
)

// UserServiceDeps bundles the dependencies required to construct a user service instance.
type UserServiceDeps struct {
	Users          repositories.UserRepository
	Addresses      repositories.AddressRepository
	PaymentMethods repositories.PaymentMethodRepository
	Favorites      repositories.FavoriteRepository
	Audit          repositories.AuditLogRepository
	Firebase       auth.UserGetter
	Clock          func() time.Time
}

type userService struct {
	users          repositories.UserRepository
	addresses      repositories.AddressRepository
	paymentMethods repositories.PaymentMethodRepository
	favorites      repositories.FavoriteRepository
	audit          repositories.AuditLogRepository
	firebase       auth.UserGetter
	clock          func() time.Time
}

// NewUserService wires dependencies into a concrete UserService implementation.
func NewUserService(deps UserServiceDeps) (UserService, error) {
	if deps.Users == nil {
		return nil, errors.New("user service: user repository is required")
	}
	if deps.Firebase == nil {
		return nil, errors.New("user service: firebase user getter is required")
	}

	clock := deps.Clock
	if clock == nil {
		clock = time.Now
	}

	return &userService{
		users:          deps.Users,
		addresses:      deps.Addresses,
		paymentMethods: deps.PaymentMethods,
		favorites:      deps.Favorites,
		audit:          deps.Audit,
		firebase:       deps.Firebase,
		clock: func() time.Time {
			return clock().UTC()
		},
	}, nil
}

func (s *userService) GetProfile(ctx context.Context, userID string) (UserProfile, error) {
	return s.getProfile(ctx, userID, true)
}

func (s *userService) GetByUID(ctx context.Context, userID string) (UserProfile, error) {
	return s.getProfile(ctx, userID, true)
}

func (s *userService) UpdateProfile(ctx context.Context, cmd UpdateProfileCommand) (UserProfile, error) {
	if strings.TrimSpace(cmd.UserID) == "" {
		return UserProfile{}, errUserIDRequired
	}
	profile, err := s.getProfile(ctx, cmd.UserID, true)
	if err != nil {
		return UserProfile{}, err
	}

	if cmd.ActorID == "" {
		return UserProfile{}, errActorIDRequired
	}

	if cmd.ExpectedSyncTime != nil && !profile.LastSyncTime.IsZero() && !profile.LastSyncTime.Equal(cmd.ExpectedSyncTime.UTC()) {
		return UserProfile{}, errors.Join(errProfileConflict, fmt.Errorf("expected %s got %s", cmd.ExpectedSyncTime.UTC().Format(time.RFC3339Nano), profile.LastSyncTime.Format(time.RFC3339Nano)))
	}

	updated, changes, err := s.applyProfileUpdates(profile, cmd)
	if err != nil {
		return UserProfile{}, err
	}

	if len(changes) == 0 {
		return profile, nil
	}

	updated.LastSyncTime = profile.LastSyncTime
	saved, err := s.users.UpdateProfile(ctx, updated)
	if err != nil {
		return UserProfile{}, mapConflictError(err)
	}

	if err := s.appendAudit(ctx, auditActionProfileUpdate, cmd.ActorID, saved.ID, changes); err != nil {
		return UserProfile{}, err
	}

	return saved, nil
}

func (s *userService) MaskProfile(ctx context.Context, cmd MaskProfileCommand) (UserProfile, error) {
	if strings.TrimSpace(cmd.UserID) == "" {
		return UserProfile{}, errUserIDRequired
	}
	profile, err := s.getProfile(ctx, cmd.UserID, true)
	if err != nil {
		return UserProfile{}, err
	}
	if strings.TrimSpace(cmd.ActorID) == "" {
		return UserProfile{}, errActorIDRequired
	}

	now := s.clock()

	masked := profile
	masked.LastSyncTime = profile.LastSyncTime
	masked.DisplayName = "Masked User"
	masked.Email = fmt.Sprintf("masked+%s%s", profile.ID, emailMaskSuffix)
	masked.PhoneNumber = ""
	masked.PhotoURL = ""
	masked.AvatarAssetID = nil
	masked.NotificationPrefs = nil
	masked.PreferredLanguage = ""
	masked.Locale = ""
	masked.ProviderData = nil
	masked.PiiMaskedAt = &now
	masked.IsActive = false

	saved, err := s.users.UpdateProfile(ctx, masked)
	if err != nil {
		return UserProfile{}, mapConflictError(err)
	}

	changes := map[string]any{
		"masked": true,
	}
	if cmd.Reason != "" {
		changes["reason"] = strings.TrimSpace(cmd.Reason)
	}
	changes["occurredAt"] = now.Format(time.RFC3339Nano)

	if err := s.appendAudit(ctx, auditActionProfileMask, cmd.ActorID, saved.ID, changes); err != nil {
		return UserProfile{}, err
	}

	return saved, nil
}

func (s *userService) SetUserActive(ctx context.Context, cmd SetUserActiveCommand) (UserProfile, error) {
	if strings.TrimSpace(cmd.UserID) == "" {
		return UserProfile{}, errUserIDRequired
	}
	profile, err := s.getProfile(ctx, cmd.UserID, false)
	if err != nil {
		return UserProfile{}, err
	}
	if strings.TrimSpace(cmd.ActorID) == "" {
		return UserProfile{}, errActorIDRequired
	}
	if cmd.ExpectedSyncTime != nil && !profile.LastSyncTime.IsZero() && !profile.LastSyncTime.Equal(cmd.ExpectedSyncTime.UTC()) {
		return UserProfile{}, errors.Join(errProfileConflict, fmt.Errorf("expected %s got %s", cmd.ExpectedSyncTime.UTC().Format(time.RFC3339Nano), profile.LastSyncTime.Format(time.RFC3339Nano)))
	}

	if profile.IsActive == cmd.IsActive {
		return profile, nil
	}

	updated := profile
	updated.LastSyncTime = profile.LastSyncTime
	updated.IsActive = cmd.IsActive

	saved, err := s.users.UpdateProfile(ctx, updated)
	if err != nil {
		return UserProfile{}, mapConflictError(err)
	}

	action := auditActionProfileDeactivate
	if cmd.IsActive {
		action = auditActionProfileActivate
	}

	changes := map[string]any{
		"isActive": diffValue(profile.IsActive, cmd.IsActive),
	}
	if cmd.Reason != "" {
		changes["reason"] = strings.TrimSpace(cmd.Reason)
	}

	if err := s.appendAudit(ctx, action, cmd.ActorID, saved.ID, changes); err != nil {
		return UserProfile{}, err
	}

	return saved, nil
}

func (s *userService) ListAddresses(ctx context.Context, userID string) ([]Address, error) {
	if s.addresses == nil {
		return nil, errAddressRepositoryUnavailable
	}
	return s.addresses.List(ctx, userID)
}

func (s *userService) UpsertAddress(ctx context.Context, cmd UpsertAddressCommand) (Address, error) {
	if s.addresses == nil {
		return Address{}, errAddressRepositoryUnavailable
	}
	address, err := s.addresses.Upsert(ctx, cmd.UserID, cmd.AddressID, cmd.Address, cmd.IsDefault)
	return address, err
}

func (s *userService) DeleteAddress(ctx context.Context, cmd DeleteAddressCommand) error {
	if s.addresses == nil {
		return errAddressRepositoryUnavailable
	}
	return s.addresses.Delete(ctx, cmd.UserID, cmd.AddressID)
}

func (s *userService) ListPaymentMethods(ctx context.Context, userID string) ([]PaymentMethod, error) {
	return nil, errPaymentMethodsNotImplemented
}

func (s *userService) AddPaymentMethod(ctx context.Context, cmd AddPaymentMethodCommand) (PaymentMethod, error) {
	return PaymentMethod{}, errPaymentMethodsNotImplemented
}

func (s *userService) RemovePaymentMethod(ctx context.Context, cmd RemovePaymentMethodCommand) error {
	return errPaymentMethodsNotImplemented
}

func (s *userService) ListFavorites(ctx context.Context, userID string, pager Pagination) (domain.CursorPage[FavoriteDesign], error) {
	return domain.CursorPage[FavoriteDesign]{}, errFavoritesNotImplemented
}

func (s *userService) ToggleFavorite(ctx context.Context, cmd ToggleFavoriteCommand) error {
	return errFavoritesNotImplemented
}

func (s *userService) getProfile(ctx context.Context, userID string, seed bool) (domain.UserProfile, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return domain.UserProfile{}, errUserIDRequired
	}

	profile, err := s.users.FindByID(ctx, userID)
	if err == nil {
		return profile, nil
	}
	if !seed || !isNotFound(err) {
		return domain.UserProfile{}, err
	}

	record, err := s.firebase.GetUser(ctx, userID)
	if err != nil {
		return domain.UserProfile{}, fmt.Errorf("fetch firebase user: %w", err)
	}

	now := s.clock()
	fresh := profileFromFirebase(record, now)
	fresh.ID = userID
	fresh.LastSyncTime = time.Time{}

	saved, err := s.users.UpdateProfile(ctx, fresh)
	if err != nil {
		return domain.UserProfile{}, err
	}
	return saved, nil
}

func (s *userService) applyProfileUpdates(existing domain.UserProfile, cmd UpdateProfileCommand) (domain.UserProfile, map[string]any, error) {
	after := existing
	changes := make(map[string]any)

	if cmd.DisplayName != nil {
		name := strings.TrimSpace(*cmd.DisplayName)
		if err := validateDisplayName(name); err != nil {
			return domain.UserProfile{}, nil, err
		}
		if name != existing.DisplayName {
			after.DisplayName = name
			changes["displayName"] = diffValue(existing.DisplayName, name)
		}
	}

	if cmd.PreferredLanguage != nil {
		lang := strings.TrimSpace(*cmd.PreferredLanguage)
		canonical, err := canonicaliseLanguageTag(lang)
		if err != nil {
			return domain.UserProfile{}, nil, err
		}
		if canonical != existing.PreferredLanguage {
			after.PreferredLanguage = canonical
			changes["preferredLanguage"] = diffValue(existing.PreferredLanguage, canonical)
		}
	}

	if cmd.Locale != nil {
		locale := strings.TrimSpace(*cmd.Locale)
		canonical, err := canonicaliseLanguageTag(locale)
		if err != nil {
			return domain.UserProfile{}, nil, err
		}
		if canonical != existing.Locale {
			after.Locale = canonical
			changes["locale"] = diffValue(existing.Locale, canonical)
		}
	}

	if cmd.NotificationPrefs != nil {
		prefs, err := normaliseNotificationPrefs(cmd.NotificationPrefs)
		if err != nil {
			return domain.UserProfile{}, nil, err
		}
		if !equalNotificationPrefs(existing.NotificationPrefs, prefs) {
			after.NotificationPrefs = prefs
			changes["notificationPrefs"] = diffValue(cloneNotificationPrefs(existing.NotificationPrefs), cloneNotificationPrefs(prefs))
		}
	}

	if cmd.AvatarAssetID != nil {
		trimmed := strings.TrimSpace(*cmd.AvatarAssetID)
		var newValue *string
		if trimmed != "" {
			value := trimmed
			newValue = &value
		}
		if !equalStringPointers(existing.AvatarAssetID, newValue) {
			after.AvatarAssetID = newValue
			changes["avatarAssetId"] = diffValue(pointerValue(existing.AvatarAssetID), pointerValue(newValue))
		}
	}

	return after, changes, nil
}

func (s *userService) appendAudit(ctx context.Context, action string, actorID string, userID string, diff map[string]any) error {
	if s.audit == nil {
		return nil
	}
	entry := domain.AuditLogEntry{
		ActorRef:  strings.TrimSpace(actorID),
		TargetRef: fmt.Sprintf("user:%s", strings.TrimSpace(userID)),
		Action:    action,
		Diff:      diff,
		CreatedAt: s.clock(),
	}
	return s.audit.Append(ctx, entry)
}

func validateDisplayName(name string) error {
	if name == "" {
		return errInvalidDisplayName
	}
	length := utf8.RuneCountInString(name)
	if length < 2 || length > 100 {
		return errInvalidDisplayName
	}
	return nil
}

func canonicaliseLanguageTag(tag string) (string, error) {
	tag = strings.ReplaceAll(strings.TrimSpace(tag), "_", "-")
	if tag == "" {
		return "", nil
	}
	parsed, err := language.Parse(tag)
	if err != nil {
		return "", errors.Join(errInvalidLanguageTag, err)
	}
	return parsed.String(), nil
}

func normaliseNotificationPrefs(prefs map[string]bool) (domain.NotificationPreferences, error) {
	if len(prefs) == 0 {
		return nil, nil
	}
	normalised := make(domain.NotificationPreferences)
	for key, value := range prefs {
		trimmed := strings.ToLower(strings.TrimSpace(key))
		if trimmed == "" {
			continue
		}
		if !notificationKeyPattern.MatchString(trimmed) {
			return nil, fmt.Errorf("user: invalid notification key %q", key)
		}
		normalised[trimmed] = value
	}
	if len(normalised) == 0 {
		return nil, nil
	}
	return normalised, nil
}

func equalNotificationPrefs(a, b domain.NotificationPreferences) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return maps.Equal(map[string]bool(a), map[string]bool(b))
}

func cloneNotificationPrefs(prefs domain.NotificationPreferences) domain.NotificationPreferences {
	if len(prefs) == 0 {
		return nil
	}
	clone := make(domain.NotificationPreferences, len(prefs))
	for k, v := range prefs {
		clone[k] = v
	}
	return clone
}

func equalStringPointers(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func pointerValue(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func diffValue(from, to any) map[string]any {
	return map[string]any{
		"from": from,
		"to":   to,
	}
}

func profileFromFirebase(record *firebaseauth.UserRecord, now time.Time) domain.UserProfile {
	if record == nil {
		return domain.UserProfile{}
	}

	var (
		uid         string
		displayName string
		email       string
		phone       string
		photo       string
	)

	if record.UserInfo != nil {
		uid = record.UserInfo.UID
		displayName = record.UserInfo.DisplayName
		email = record.UserInfo.Email
		phone = record.UserInfo.PhoneNumber
		photo = record.UserInfo.PhotoURL
	}

	profile := domain.UserProfile{
		ID:                strings.TrimSpace(uid),
		DisplayName:       strings.TrimSpace(displayName),
		Email:             strings.ToLower(strings.TrimSpace(email)),
		PhoneNumber:       strings.TrimSpace(phone),
		PhotoURL:          strings.TrimSpace(photo),
		PreferredLanguage: "",
		Locale:            "",
		Roles:             deriveRoles(record),
		IsActive:          true,
		NotificationPrefs: nil,
		ProviderData:      providersFromFirebase(record),
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if locale, ok := record.CustomClaims["locale"].(string); ok {
		if canonical, err := canonicaliseLanguageTag(locale); err == nil {
			profile.Locale = canonical
		}
	}
	if lang, ok := record.CustomClaims["preferredLanguage"].(string); ok {
		if canonical, err := canonicaliseLanguageTag(lang); err == nil {
			profile.PreferredLanguage = canonical
		}
	}

	return profile
}

func providersFromFirebase(record *firebaseauth.UserRecord) []domain.AuthProvider {
	if record == nil {
		return nil
	}

	var providers []domain.AuthProvider
	primary := providerFromInfo(record.UserInfo)
	if primary.ProviderID != "" {
		providers = append(providers, primary)
	}

	for _, info := range record.ProviderUserInfo {
		if info == nil {
			continue
		}
		providers = append(providers, providerFromInfo(info))
	}
	return providers
}

func providerFromInfo(info *firebaseauth.UserInfo) domain.AuthProvider {
	if info == nil {
		return domain.AuthProvider{}
	}
	return domain.AuthProvider{
		ProviderID:  strings.TrimSpace(info.ProviderID),
		UID:         strings.TrimSpace(info.UID),
		Email:       strings.ToLower(strings.TrimSpace(info.Email)),
		DisplayName: strings.TrimSpace(info.DisplayName),
		PhoneNumber: strings.TrimSpace(info.PhoneNumber),
		PhotoURL:    strings.TrimSpace(info.PhotoURL),
	}
}

func deriveRoles(record *firebaseauth.UserRecord) []string {
	roles := map[string]struct{}{auth.RoleUser: {}}
	if record == nil {
		return sortedKeys(roles)
	}

	if value, ok := record.CustomClaims["role"]; ok {
		if str, ok := value.(string); ok {
			addRole(roles, str)
		}
	}
	if raw, ok := record.CustomClaims["roles"]; ok {
		switch v := raw.(type) {
		case []any:
			for _, item := range v {
				if str, ok := item.(string); ok {
					addRole(roles, str)
				}
			}
		case []string:
			for _, str := range v {
				addRole(roles, str)
			}
		}
	}
	return sortedKeys(roles)
}

func addRole(target map[string]struct{}, role string) {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		return
	}
	target[role] = struct{}{}
}

func sortedKeys(m map[string]struct{}) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	var repoErr repositories.RepositoryError
	if errors.As(err, &repoErr) {
		return repoErr.IsNotFound()
	}
	return false
}

func isConflict(err error) bool {
	if err == nil {
		return false
	}
	var repoErr repositories.RepositoryError
	if errors.As(err, &repoErr) {
		return repoErr.IsConflict()
	}
	return false
}

func mapConflictError(err error) error {
	if isConflict(err) {
		return errors.Join(errProfileConflict, err)
	}
	return err
}
