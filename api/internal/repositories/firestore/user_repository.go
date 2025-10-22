package firestore

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	domain "github.com/hanko-field/api/internal/domain"
	pfirestore "github.com/hanko-field/api/internal/platform/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const userCollection = "users"

// UserRepository persists user profiles in Firestore using optimistic locking.
type UserRepository struct {
	base     *pfirestore.BaseRepository[userDocument]
	provider *pfirestore.Provider
}

// NewUserRepository constructs a Firestore-backed user repository.
func NewUserRepository(provider *pfirestore.Provider) (*UserRepository, error) {
	if provider == nil {
		return nil, errors.New("user repository requires firestore provider")
	}

	base := pfirestore.NewBaseRepository[userDocument](provider, userCollection, nil, nil)
	return &UserRepository{base: base, provider: provider}, nil
}

// FindByID loads the user profile by UID.
func (r *UserRepository) FindByID(ctx context.Context, userID string) (domain.UserProfile, error) {
	if r == nil || r.base == nil {
		return domain.UserProfile{}, errors.New("user repository not initialised")
	}
	if strings.TrimSpace(userID) == "" {
		return domain.UserProfile{}, errors.New("user id is required")
	}

	doc, err := r.base.Get(ctx, userID)
	if err != nil {
		return domain.UserProfile{}, err
	}

	profile := toDomainProfile(doc.Data)
	profile.ID = doc.ID
	profile.LastSyncTime = doc.UpdateTime
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = doc.CreateTime
	}
	if profile.UpdatedAt.IsZero() {
		profile.UpdatedAt = doc.UpdateTime
	}
	return profile, nil
}

// UpdateProfile upserts the user profile. When LastSyncTime is set the mutation will
// enforce optimistic locking using Firestore's update time precondition.
func (r *UserRepository) UpdateProfile(ctx context.Context, profile domain.UserProfile) (domain.UserProfile, error) {
	if r == nil || r.base == nil {
		return domain.UserProfile{}, errors.New("user repository not initialised")
	}
	if strings.TrimSpace(profile.ID) == "" {
		return domain.UserProfile{}, errors.New("profile id is required")
	}

	now := time.Now().UTC()
	doc := fromDomainProfile(profile, now)

	if profile.LastSyncTime.IsZero() {
		result, err := r.base.Set(ctx, profile.ID, doc)
		if err != nil {
			return domain.UserProfile{}, err
		}
		saved := toDomainProfile(doc)
		saved.ID = profile.ID
		saved.LastSyncTime = result.UpdateTime
		return saved, nil
	}

	if r.provider == nil {
		return domain.UserProfile{}, errors.New("user repository provider unavailable")
	}

	docID := profile.ID
	if err := r.provider.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		docRef, err := r.base.DocumentRef(ctx, docID)
		if err != nil {
			return err
		}
		snap, err := tx.Get(docRef)
		if err != nil {
			return err
		}
		if !snap.UpdateTime.Equal(profile.LastSyncTime) {
			return status.Error(codes.Aborted, "user profile stale update")
		}
		return tx.Set(docRef, doc)
	}); err != nil {
		return domain.UserProfile{}, err
	}

	latest, err := r.base.Get(ctx, docID)
	if err != nil {
		return domain.UserProfile{}, err
	}
	saved := toDomainProfile(latest.Data)
	saved.ID = latest.ID
	saved.LastSyncTime = latest.UpdateTime
	if saved.CreatedAt.IsZero() {
		saved.CreatedAt = latest.CreateTime
	}
	if saved.UpdatedAt.IsZero() {
		saved.UpdatedAt = latest.UpdateTime
	}
	return saved, nil
}

type userDocument struct {
	UID               string             `firestore:"uid"`
	DisplayName       string             `firestore:"displayName"`
	Email             string             `firestore:"email"`
	PhoneNumber       string             `firestore:"phoneNumber"`
	PhotoURL          string             `firestore:"photoURL"`
	AvatarAssetID     *string            `firestore:"avatarAssetId"`
	PreferredLanguage string             `firestore:"preferredLanguage"`
	Locale            string             `firestore:"locale"`
	Roles             []string           `firestore:"roles"`
	IsActive          bool               `firestore:"isActive"`
	NotificationPrefs map[string]bool    `firestore:"notificationPrefs"`
	ProviderData      []providerDocument `firestore:"providerData"`
	CreatedAt         time.Time          `firestore:"createdAt"`
	UpdatedAt         time.Time          `firestore:"updatedAt"`
	PiiMaskedAt       *time.Time         `firestore:"piiMaskedAt,omitempty"`
	NameMappingRef    *string            `firestore:"nameMappingRef,omitempty"`
}

type providerDocument struct {
	ProviderID  string `firestore:"providerId"`
	UID         string `firestore:"uid"`
	Email       string `firestore:"email,omitempty"`
	DisplayName string `firestore:"displayName,omitempty"`
	PhoneNumber string `firestore:"phoneNumber,omitempty"`
	PhotoURL    string `firestore:"photoURL,omitempty"`
}

func toDomainProfile(doc userDocument) domain.UserProfile {
	profile := domain.UserProfile{
		DisplayName:       doc.DisplayName,
		Email:             strings.TrimSpace(doc.Email),
		PhoneNumber:       strings.TrimSpace(doc.PhoneNumber),
		PhotoURL:          strings.TrimSpace(doc.PhotoURL),
		AvatarAssetID:     doc.AvatarAssetID,
		PreferredLanguage: strings.TrimSpace(doc.PreferredLanguage),
		Locale:            strings.TrimSpace(doc.Locale),
		Roles:             cloneStringSlice(doc.Roles),
		IsActive:          doc.IsActive,
		NotificationPrefs: cloneNotificationPrefs(doc.NotificationPrefs),
		ProviderData:      toDomainProviders(doc.ProviderData),
		CreatedAt:         doc.CreatedAt,
		UpdatedAt:         doc.UpdatedAt,
		PiiMaskedAt:       doc.PiiMaskedAt,
		NameMappingRef:    doc.NameMappingRef,
	}
	if profile.NotificationPrefs == nil {
		profile.NotificationPrefs = domain.NotificationPreferences{}
	}
	return profile
}

func fromDomainProfile(profile domain.UserProfile, now time.Time) userDocument {
	doc := userDocument{
		UID:               profile.ID,
		DisplayName:       strings.TrimSpace(profile.DisplayName),
		Email:             strings.ToLower(strings.TrimSpace(profile.Email)),
		PhoneNumber:       strings.TrimSpace(profile.PhoneNumber),
		PhotoURL:          strings.TrimSpace(profile.PhotoURL),
		AvatarAssetID:     profile.AvatarAssetID,
		PreferredLanguage: strings.TrimSpace(profile.PreferredLanguage),
		Locale:            strings.TrimSpace(profile.Locale),
		Roles:             normaliseRoles(profile.Roles),
		IsActive:          true,
		NotificationPrefs: map[string]bool{},
		ProviderData:      fromDomainProviders(profile.ProviderData),
		CreatedAt:         profile.CreatedAt,
		UpdatedAt:         now,
		PiiMaskedAt:       profile.PiiMaskedAt,
	}
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = now
	}
	if len(profile.NotificationPrefs) > 0 {
		for k, v := range profile.NotificationPrefs {
			key := strings.TrimSpace(k)
			if key == "" {
				continue
			}
			doc.NotificationPrefs[key] = v
		}
	} else {
		doc.NotificationPrefs = nil
	}
	if !profile.IsActive {
		doc.IsActive = false
	}
	if doc.Roles == nil {
		doc.Roles = []string{}
	}
	if profile.NameMappingRef != nil {
		trimmed := strings.TrimSpace(*profile.NameMappingRef)
		if trimmed != "" {
			value := trimmed
			doc.NameMappingRef = &value
		}
	}
	return doc
}

func toDomainProviders(docs []providerDocument) []domain.AuthProvider {
	if len(docs) == 0 {
		return nil
	}
	providers := make([]domain.AuthProvider, 0, len(docs))
	for _, p := range docs {
		providers = append(providers, domain.AuthProvider{
			ProviderID:  strings.TrimSpace(p.ProviderID),
			UID:         strings.TrimSpace(p.UID),
			Email:       strings.TrimSpace(p.Email),
			DisplayName: strings.TrimSpace(p.DisplayName),
			PhoneNumber: strings.TrimSpace(p.PhoneNumber),
			PhotoURL:    strings.TrimSpace(p.PhotoURL),
		})
	}
	return providers
}

func fromDomainProviders(providers []domain.AuthProvider) []providerDocument {
	if len(providers) == 0 {
		return nil
	}
	docs := make([]providerDocument, 0, len(providers))
	for _, p := range providers {
		docs = append(docs, providerDocument{
			ProviderID:  strings.TrimSpace(p.ProviderID),
			UID:         strings.TrimSpace(p.UID),
			Email:       strings.TrimSpace(p.Email),
			DisplayName: strings.TrimSpace(p.DisplayName),
			PhoneNumber: strings.TrimSpace(p.PhoneNumber),
			PhotoURL:    strings.TrimSpace(p.PhotoURL),
		})
	}
	return docs
}

func cloneNotificationPrefs(prefs map[string]bool) domain.NotificationPreferences {
	if len(prefs) == 0 {
		return domain.NotificationPreferences{}
	}
	cloned := make(domain.NotificationPreferences, len(prefs))
	for k, v := range prefs {
		cloned[k] = v
	}
	return cloned
}

func cloneStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func normaliseRoles(roles []string) []string {
	if len(roles) == 0 {
		return nil
	}
	uniq := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		trimmed := strings.ToLower(strings.TrimSpace(role))
		if trimmed == "" {
			continue
		}
		uniq[trimmed] = struct{}{}
	}
	if len(uniq) == 0 {
		return nil
	}
	normalised := make([]string, 0, len(uniq))
	for role := range uniq {
		normalised = append(normalised, role)
	}
	sort.Strings(normalised)
	return normalised
}

// Ensure the concrete type satisfies the repository interface.
var _ interface {
	FindByID(context.Context, string) (domain.UserProfile, error)
	UpdateProfile(context.Context, domain.UserProfile) (domain.UserProfile, error)
} = (*UserRepository)(nil)

// CollectionName exposes the Firestore collection for migration tooling.
func (r *UserRepository) CollectionName() string {
	return userCollection
}

// DocumentPath constructs the document path for the provided user id.
func (r *UserRepository) DocumentPath(userID string) string {
	return fmt.Sprintf("%s/%s", userCollection, strings.TrimSpace(userID))
}
