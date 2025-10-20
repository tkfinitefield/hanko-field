package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	errInvalidNotificationKey       = errors.New("user: invalid notification key")
	errProfileConflict              = errors.New("user: profile has been modified")
	errAddressRepositoryUnavailable = errors.New("user: address repository not configured")
	errAddressIDRequired            = errors.New("user: address id is required")
	errAddressNotFound              = errors.New("user: address not found")
	errInvalidAddressRecipient      = errors.New("user: invalid address recipient")
	errInvalidAddressLine1          = errors.New("user: invalid address line1")
	errInvalidAddressCity           = errors.New("user: invalid address city")
	errInvalidAddressCountry        = errors.New("user: invalid address country")
	errInvalidAddressPostalCode     = errors.New("user: invalid address postal code")
	errInvalidAddressPhone          = errors.New("user: invalid address phone")
	errPaymentRepositoryUnavailable = errors.New("user: payment method repository not configured")
	errPaymentVerifierUnavailable   = errors.New("user: payment method verifier not configured")
	errPaymentProviderRequired      = errors.New("user: payment provider is required")
	errPaymentTokenRequired         = errors.New("user: payment token is required")
	errPaymentMethodNotFound        = errors.New("user: payment method not found")
	errPaymentMethodDuplicate       = errors.New("user: payment method already exists")
	errPaymentMethodInUse           = errors.New("user: payment method has outstanding invoices")
	errFavoritesNotImplemented      = errors.New("user: favorites operations not yet implemented")
	emailMaskSuffix                 = "@hanko-field.invalid"
	notificationKeyPattern          = regexp.MustCompile(`^[a-z0-9_.-]{1,40}$`)
	addressPhonePattern             = regexp.MustCompile(`^[0-9+()\-\s]{6,20}$`)
	addressCountryPattern           = regexp.MustCompile(`^[A-Za-z]{2}$`)
	addressPostalPattern            = regexp.MustCompile(`^[0-9A-Za-z\-\s]{3,16}$`)
	auditActionProfileUpdate        = "user.profile.update"
	auditActionProfileMask          = "user.profile.mask"
	auditActionProfileActivate      = "user.profile.activate"
	auditActionProfileDeactivate    = "user.profile.deactivate"
)

var (
	// ErrUserProfileConflict indicates the profile has been modified by another concurrent actor.
	ErrUserProfileConflict = errProfileConflict
	// ErrUserInvalidDisplayName indicates the supplied display name failed validation.
	ErrUserInvalidDisplayName = errInvalidDisplayName
	// ErrUserInvalidLanguageTag indicates the supplied language or locale tag is invalid.
	ErrUserInvalidLanguageTag = errInvalidLanguageTag
	// ErrUserInvalidNotificationKey indicates a notification preference key did not meet validation rules.
	ErrUserInvalidNotificationKey = errInvalidNotificationKey
	// ErrUserAddressNotFound indicates the requested address does not exist.
	ErrUserAddressNotFound = errAddressNotFound
	// ErrUserInvalidAddressRecipient indicates the address recipient failed validation.
	ErrUserInvalidAddressRecipient = errInvalidAddressRecipient
	// ErrUserInvalidAddressLine1 indicates the primary address line failed validation.
	ErrUserInvalidAddressLine1 = errInvalidAddressLine1
	// ErrUserInvalidAddressCity indicates the city component failed validation.
	ErrUserInvalidAddressCity = errInvalidAddressCity
	// ErrUserInvalidAddressCountry indicates the country component failed validation.
	ErrUserInvalidAddressCountry = errInvalidAddressCountry
	// ErrUserInvalidAddressPostalCode indicates the postal code failed validation.
	ErrUserInvalidAddressPostalCode = errInvalidAddressPostalCode
	// ErrUserInvalidAddressPhone indicates the phone number failed validation.
	ErrUserInvalidAddressPhone = errInvalidAddressPhone
	// ErrUserPaymentMethodNotFound indicates the requested payment method does not exist.
	ErrUserPaymentMethodNotFound = errPaymentMethodNotFound
	// ErrUserPaymentMethodDuplicate indicates the PSP token already exists for the user.
	ErrUserPaymentMethodDuplicate = errPaymentMethodDuplicate
	// ErrUserPaymentMethodInUse indicates the payment method cannot be removed due to outstanding invoices.
	ErrUserPaymentMethodInUse = errPaymentMethodInUse
	// ErrUserPaymentProviderRequired indicates the provider input was empty.
	ErrUserPaymentProviderRequired = errPaymentProviderRequired
	// ErrUserPaymentTokenRequired indicates the token input was empty.
	ErrUserPaymentTokenRequired = errPaymentTokenRequired
)

// UserServiceDeps bundles the dependencies required to construct a user service instance.
type UserServiceDeps struct {
	Users           repositories.UserRepository
	Addresses       repositories.AddressRepository
	PaymentMethods  repositories.PaymentMethodRepository
	PaymentVerifier PaymentMethodVerifier
	Invoices        OutstandingInvoiceChecker
	Favorites       repositories.FavoriteRepository
	Audit           AuditLogService
	Firebase        auth.UserGetter
	Clock           func() time.Time
}

type userService struct {
	users           repositories.UserRepository
	addresses       repositories.AddressRepository
	paymentMethods  repositories.PaymentMethodRepository
	paymentVerifier PaymentMethodVerifier
	invoices        OutstandingInvoiceChecker
	favorites       repositories.FavoriteRepository
	audit           AuditLogService
	firebase        auth.UserGetter
	clock           func() time.Time
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
		users:           deps.Users,
		addresses:       deps.Addresses,
		paymentMethods:  deps.PaymentMethods,
		paymentVerifier: deps.PaymentVerifier,
		invoices:        deps.Invoices,
		favorites:       deps.Favorites,
		audit:           deps.Audit,
		firebase:        deps.Firebase,
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
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errUserIDRequired
	}
	items, err := s.addresses.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].NormalizedHash == "" {
			items[i].NormalizedHash = addressFingerprint(items[i])
		}
	}
	return items, nil
}

func (s *userService) UpsertAddress(ctx context.Context, cmd UpsertAddressCommand) (Address, error) {
	if s.addresses == nil {
		return Address{}, errAddressRepositoryUnavailable
	}
	userID := strings.TrimSpace(cmd.UserID)
	if userID == "" {
		return Address{}, errUserIDRequired
	}

	targetID := ""
	if cmd.AddressID != nil {
		targetID = strings.TrimSpace(*cmd.AddressID)
	}

	var existing Address
	var err error
	if targetID != "" {
		existing, err = s.addresses.Get(ctx, userID, targetID)
		if err != nil {
			if isNotFound(err) {
				return Address{}, errAddressNotFound
			}
			return Address{}, err
		}
	}

	addressInput, err := sanitizeAddress(cmd.Address)
	if err != nil {
		return Address{}, err
	}

	fingerprint := addressFingerprint(addressInput)

	if targetID == "" {
		if duplicate, found, err := s.addresses.FindByHash(ctx, userID, fingerprint); err != nil {
			return Address{}, err
		} else if found {
			targetID = duplicate.ID
			existing = duplicate
		}
	}

	hasAny, err := s.addresses.HasAny(ctx, userID)
	if err != nil {
		return Address{}, err
	}

	finalAddress := mergeAddress(existing, addressInput)
	finalAddress.ID = targetID
	finalAddress.NormalizedHash = fingerprint

	defaultShipping := existing.DefaultShipping
	if cmd.DefaultShipping != nil {
		defaultShipping = *cmd.DefaultShipping
	} else if targetID == "" && !hasAny {
		defaultShipping = true
	}

	defaultBilling := existing.DefaultBilling
	if cmd.DefaultBilling != nil {
		defaultBilling = *cmd.DefaultBilling
	} else if targetID == "" && !hasAny {
		defaultBilling = true
	}

	finalAddress.DefaultShipping = defaultShipping
	finalAddress.DefaultBilling = defaultBilling

	var addressIDPtr *string
	if targetID != "" {
		addressIDPtr = &targetID
	}

	saved, err := s.addresses.Upsert(ctx, userID, addressIDPtr, finalAddress)
	if err != nil {
		return Address{}, err
	}
	if saved.NormalizedHash == "" {
		saved.NormalizedHash = addressFingerprint(saved)
	}
	return saved, nil
}

func (s *userService) DeleteAddress(ctx context.Context, cmd DeleteAddressCommand) error {
	if s.addresses == nil {
		return errAddressRepositoryUnavailable
	}
	userID := strings.TrimSpace(cmd.UserID)
	addressID := strings.TrimSpace(cmd.AddressID)
	if userID == "" {
		return errUserIDRequired
	}
	if addressID == "" {
		return errAddressIDRequired
	}

	target, err := s.addresses.Get(ctx, userID, addressID)
	if err != nil {
		if isNotFound(err) {
			return errAddressNotFound
		}
		return err
	}

	if err := s.addresses.Delete(ctx, userID, addressID); err != nil {
		return err
	}

	if !(target.DefaultShipping || target.DefaultBilling) {
		return nil
	}

	addresses, err := s.addresses.List(ctx, userID)
	if err != nil {
		return err
	}

	var replacementID string
	if cmd.ReplacementID != nil {
		id := strings.TrimSpace(*cmd.ReplacementID)
		if id != "" {
			for _, addr := range addresses {
				if strings.EqualFold(addr.ID, id) {
					replacementID = addr.ID
					break
				}
			}
			if replacementID == "" {
				return errAddressNotFound
			}
		}
	}

	if replacementID == "" {
		for _, addr := range addresses {
			if strings.EqualFold(addr.ID, addressID) {
				continue
			}
			replacementID = addr.ID
			break
		}
	}

	if replacementID == "" {
		return nil
	}

	var shippingPtr, billingPtr *bool
	if target.DefaultShipping {
		val := true
		shippingPtr = &val
	}
	if target.DefaultBilling {
		val := true
		billingPtr = &val
	}

	if _, err := s.addresses.SetDefaultFlags(ctx, userID, replacementID, shippingPtr, billingPtr); err != nil {
		return err
	}

	return nil
}

func (s *userService) ListPaymentMethods(ctx context.Context, userID string) ([]PaymentMethod, error) {
	if s.paymentMethods == nil {
		return nil, errPaymentRepositoryUnavailable
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errUserIDRequired
	}
	items, err := s.paymentMethods.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	copied := append([]PaymentMethod(nil), items...)
	slices.SortStableFunc(copied, func(a, b PaymentMethod) int {
		if a.IsDefault && !b.IsDefault {
			return -1
		}
		if !a.IsDefault && b.IsDefault {
			return 1
		}
		switch {
		case a.CreatedAt.After(b.CreatedAt):
			return -1
		case a.CreatedAt.Before(b.CreatedAt):
			return 1
		default:
			return strings.Compare(a.ID, b.ID)
		}
	})
	return copied, nil
}

func (s *userService) AddPaymentMethod(ctx context.Context, cmd AddPaymentMethodCommand) (PaymentMethod, error) {
	if s.paymentMethods == nil {
		return PaymentMethod{}, errPaymentRepositoryUnavailable
	}
	if s.paymentVerifier == nil {
		return PaymentMethod{}, errPaymentVerifierUnavailable
	}

	userID := strings.TrimSpace(cmd.UserID)
	if userID == "" {
		return PaymentMethod{}, errUserIDRequired
	}

	provider := normaliseProvider(cmd.Provider)
	if provider == "" {
		return PaymentMethod{}, errPaymentProviderRequired
	}

	token := strings.TrimSpace(cmd.Token)
	if token == "" {
		return PaymentMethod{}, errPaymentTokenRequired
	}

	meta, err := s.paymentVerifier.VerifyPaymentMethod(ctx, provider, token)
	if err != nil {
		return PaymentMethod{}, err
	}

	if trimmed := strings.TrimSpace(meta.Token); trimmed != "" {
		token = trimmed
	}

	existing, err := s.paymentMethods.List(ctx, userID)
	if err != nil {
		return PaymentMethod{}, err
	}
	for _, method := range existing {
		if strings.TrimSpace(method.Token) == token {
			return PaymentMethod{}, errPaymentMethodDuplicate
		}
	}

	now := s.clock()
	method := PaymentMethod{
		Provider:  provider,
		Token:     token,
		Brand:     strings.TrimSpace(meta.Brand),
		Last4:     strings.TrimSpace(meta.Last4),
		ExpMonth:  meta.ExpMonth,
		ExpYear:   meta.ExpYear,
		IsDefault: cmd.MakeDefault,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if len(existing) == 0 {
		method.IsDefault = true
	}

	saved, err := s.paymentMethods.Insert(ctx, userID, method)
	if err != nil {
		if isConflict(err) {
			return PaymentMethod{}, errPaymentMethodDuplicate
		}
		return PaymentMethod{}, err
	}
	return saved, nil
}

func (s *userService) RemovePaymentMethod(ctx context.Context, cmd RemovePaymentMethodCommand) error {
	if s.paymentMethods == nil {
		return errPaymentRepositoryUnavailable
	}

	userID := strings.TrimSpace(cmd.UserID)
	paymentMethodID := strings.TrimSpace(cmd.PaymentMethodID)
	if userID == "" {
		return errUserIDRequired
	}
	if paymentMethodID == "" {
		return errPaymentMethodNotFound
	}

	method, err := s.paymentMethods.Get(ctx, userID, paymentMethodID)
	if err != nil {
		if isNotFound(err) {
			return errPaymentMethodNotFound
		}
		return err
	}

	if s.invoices != nil {
		blocked, err := s.invoices.HasOutstandingInvoices(ctx, userID)
		if err != nil {
			return err
		}
		if blocked {
			return errPaymentMethodInUse
		}
	}

	if err := s.paymentMethods.Delete(ctx, userID, paymentMethodID); err != nil {
		if isNotFound(err) {
			return errPaymentMethodNotFound
		}
		return err
	}

	if !method.IsDefault {
		return nil
	}

	remaining, err := s.paymentMethods.List(ctx, userID)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return nil
	}

	next, ok := selectNextDefault(remaining)
	if !ok {
		return nil
	}

	if _, err := s.paymentMethods.SetDefault(ctx, userID, next.ID); err != nil {
		if isNotFound(err) {
			return errPaymentMethodNotFound
		}
		return err
	}

	return nil
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

	if cmd.NotificationPrefs != nil || cmd.NotificationPrefsSet {
		prefs, err := normaliseNotificationPrefs(cmd.NotificationPrefs)
		if err != nil {
			return domain.UserProfile{}, nil, err
		}
		if !equalNotificationPrefs(existing.NotificationPrefs, prefs) {
			after.NotificationPrefs = prefs
			changes["notificationPrefs"] = diffValue(cloneNotificationPrefs(existing.NotificationPrefs), cloneNotificationPrefs(prefs))
		}
	}

	if cmd.AvatarAssetID != nil || cmd.AvatarAssetIDSet {
		trimmed := ""
		if cmd.AvatarAssetID != nil {
			trimmed = strings.TrimSpace(*cmd.AvatarAssetID)
		}
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
	record := AuditLogRecord{
		Actor:      strings.TrimSpace(actorID),
		Action:     action,
		TargetRef:  fmt.Sprintf("/users/%s", strings.TrimSpace(userID)),
		OccurredAt: s.clock(),
	}
	diffPayload, sensitive, metadata := splitAuditChanges(diff)
	if len(diffPayload) > 0 {
		record.Diff = diffPayload
		record.SensitiveDiffKeys = sensitive
	}
	if len(metadata) > 0 {
		if record.Metadata == nil {
			record.Metadata = make(map[string]any, len(metadata))
		}
		for k, v := range metadata {
			record.Metadata[k] = v
		}
	}
	if record.Metadata == nil {
		record.Metadata = map[string]any{}
	}
	record.Metadata["service"] = "user"
	s.audit.Record(ctx, record)
	return nil
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
			return nil, fmt.Errorf("%w %q", errInvalidNotificationKey, key)
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

func splitAuditChanges(changes map[string]any) (map[string]AuditLogDiff, []string, map[string]any) {
	if len(changes) == 0 {
		return nil, nil, nil
	}
	diff := make(map[string]AuditLogDiff)
	metadata := make(map[string]any)
	var sensitive []string
	for key, value := range changes {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		if changeMap, ok := value.(map[string]any); ok {
			diff[trimmedKey] = AuditLogDiff{
				Before: changeMap["from"],
				After:  changeMap["to"],
			}
			if isSensitiveAuditField(trimmedKey) {
				sensitive = append(sensitive, trimmedKey)
			}
			continue
		}
		metadata[trimmedKey] = value
	}
	if len(diff) == 0 {
		diff = nil
	}
	if len(metadata) == 0 {
		metadata = nil
	}
	return diff, sensitive, metadata
}

func isSensitiveAuditField(field string) bool {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "displayname", "email", "phonenumber", "notificationprefs":
		return true
	default:
		return false
	}
}

func sanitizeAddress(addr Address) (Address, error) {
	sanitized := Address{
		ID:              strings.TrimSpace(addr.ID),
		Label:           strings.TrimSpace(addr.Label),
		Recipient:       strings.TrimSpace(addr.Recipient),
		Company:         strings.TrimSpace(addr.Company),
		Line1:           strings.TrimSpace(addr.Line1),
		City:            strings.TrimSpace(addr.City),
		PostalCode:      strings.TrimSpace(addr.PostalCode),
		Country:         strings.ToUpper(strings.TrimSpace(addr.Country)),
		DefaultShipping: addr.DefaultShipping,
		DefaultBilling:  addr.DefaultBilling,
	}
	sanitized.Line2 = normalizeOptionalString(addr.Line2)
	sanitized.State = normalizeOptionalString(addr.State)
	sanitized.Phone = normalizeOptionalString(addr.Phone)
	if !addr.CreatedAt.IsZero() {
		sanitized.CreatedAt = addr.CreatedAt.UTC()
	}
	if !addr.UpdatedAt.IsZero() {
		sanitized.UpdatedAt = addr.UpdatedAt.UTC()
	}

	if sanitized.Recipient == "" {
		return Address{}, errInvalidAddressRecipient
	}
	if utf8.RuneCountInString(sanitized.Recipient) > 200 {
		return Address{}, errInvalidAddressRecipient
	}
	if sanitized.Line1 == "" {
		return Address{}, errInvalidAddressLine1
	}
	if sanitized.City == "" {
		return Address{}, errInvalidAddressCity
	}
	if sanitized.Country == "" || !addressCountryPattern.MatchString(sanitized.Country) {
		return Address{}, errInvalidAddressCountry
	}
	postal, err := canonicalisePostalCode(sanitized.Country, sanitized.PostalCode)
	if err != nil {
		return Address{}, err
	}
	sanitized.PostalCode = postal

	if sanitized.Phone != nil {
		phone := strings.TrimSpace(*sanitized.Phone)
		if phone == "" {
			sanitized.Phone = nil
		} else {
			if !addressPhonePattern.MatchString(phone) {
				return Address{}, errInvalidAddressPhone
			}
			sanitized.Phone = &phone
		}
	}

	return sanitized, nil
}

func canonicalisePostalCode(country, postal string) (string, error) {
	trimmed := strings.TrimSpace(postal)
	if trimmed == "" {
		return "", errInvalidAddressPostalCode
	}
	switch strings.ToUpper(strings.TrimSpace(country)) {
	case "JP":
		digits := strings.ReplaceAll(strings.ReplaceAll(trimmed, "-", ""), " ", "")
		if len(digits) != 7 || !allDigits(digits) {
			return "", errInvalidAddressPostalCode
		}
		return digits[:3] + "-" + digits[3:], nil
	default:
		if !addressPostalPattern.MatchString(trimmed) {
			return "", errInvalidAddressPostalCode
		}
		return trimmed, nil
	}
}

func allDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func mergeAddress(existing, input Address) Address {
	if existing.ID == "" {
		existing.CreatedAt = input.CreatedAt
	}
	result := existing
	result.Label = input.Label
	result.Recipient = input.Recipient
	result.Company = input.Company
	result.Line1 = input.Line1
	result.Line2 = input.Line2
	result.City = input.City
	result.State = input.State
	result.PostalCode = input.PostalCode
	result.Country = input.Country
	result.Phone = input.Phone
	result.UpdatedAt = input.UpdatedAt
	if result.CreatedAt.IsZero() {
		result.CreatedAt = input.CreatedAt
	}
	result.DefaultShipping = input.DefaultShipping
	result.DefaultBilling = input.DefaultBilling
	result.NormalizedHash = input.NormalizedHash
	return result
}

func addressFingerprint(addr Address) string {
	parts := []string{
		strings.ToLower(strings.TrimSpace(addr.Recipient)),
		strings.ToLower(strings.TrimSpace(addr.Company)),
		strings.ToLower(strings.TrimSpace(addr.Line1)),
		strings.ToLower(stringFromPointer(addr.Line2)),
		strings.ToLower(strings.TrimSpace(addr.City)),
		strings.ToLower(stringFromPointer(addr.State)),
		strings.ToLower(strings.TrimSpace(addr.PostalCode)),
		strings.ToLower(strings.TrimSpace(addr.Country)),
	}
	input := strings.Join(parts, "|")
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])
}

func stringFromPointer(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func normaliseProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

func selectNextDefault(methods []PaymentMethod) (PaymentMethod, bool) {
	for _, method := range methods {
		if method.IsDefault {
			return method, true
		}
	}
	if len(methods) == 0 {
		return PaymentMethod{}, false
	}
	candidate := methods[0]
	for _, method := range methods[1:] {
		if method.CreatedAt.After(candidate.CreatedAt) {
			candidate = method
			continue
		}
		if method.CreatedAt.Equal(candidate.CreatedAt) && strings.Compare(method.ID, candidate.ID) < 0 {
			candidate = method
		}
	}
	return candidate, true
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
