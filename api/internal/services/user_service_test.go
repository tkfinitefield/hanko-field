package services

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	firebaseauth "firebase.google.com/go/v4/auth"
	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

type memoryUserRepo struct {
	store map[string]domain.UserProfile
	clock func() time.Time
}

type repoErr struct {
	err      error
	notFound bool
	conflict bool
}

func (e *repoErr) Error() string       { return e.err.Error() }
func (e *repoErr) Unwrap() error       { return e.err }
func (e *repoErr) IsNotFound() bool    { return e.notFound }
func (e *repoErr) IsConflict() bool    { return e.conflict }
func (e *repoErr) IsUnavailable() bool { return false }

func newMemoryUserRepo(clock func() time.Time) *memoryUserRepo {
	return &memoryUserRepo{
		store: make(map[string]domain.UserProfile),
		clock: clock,
	}
}

func (m *memoryUserRepo) FindByID(_ context.Context, userID string) (domain.UserProfile, error) {
	profile, ok := m.store[userID]
	if !ok {
		return domain.UserProfile{}, &repoErr{err: fmt.Errorf("user %s not found", userID), notFound: true}
	}
	return cloneProfile(profile), nil
}

func (m *memoryUserRepo) UpdateProfile(_ context.Context, profile domain.UserProfile) (domain.UserProfile, error) {
	stored, exists := m.store[profile.ID]
	if exists {
		if profile.LastSyncTime.IsZero() || !profile.LastSyncTime.Equal(stored.LastSyncTime) {
			return domain.UserProfile{}, &repoErr{err: errors.New("conflict"), conflict: true}
		}
		profile.CreatedAt = stored.CreatedAt
	}
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = m.clock()
	}
	profile.UpdatedAt = m.clock()
	profile.LastSyncTime = profile.UpdatedAt
	m.store[profile.ID] = cloneProfile(profile)
	return cloneProfile(profile), nil
}

type captureAuditRepo struct {
	entries []domain.AuditLogEntry
}

func (c *captureAuditRepo) Append(_ context.Context, entry domain.AuditLogEntry) error {
	c.entries = append(c.entries, entry)
	return nil
}

func (c *captureAuditRepo) List(_ context.Context, _ repositories.AuditLogFilter) (domain.CursorPage[domain.AuditLogEntry], error) {
	return domain.CursorPage[domain.AuditLogEntry]{Items: append([]domain.AuditLogEntry(nil), c.entries...)}, nil
}

type stubFirebase struct {
	records map[string]*firebaseauth.UserRecord
}

func (s *stubFirebase) GetUser(_ context.Context, uid string) (*firebaseauth.UserRecord, error) {
	record, ok := s.records[uid]
	if !ok {
		return nil, fmt.Errorf("firebase user %s not found", uid)
	}
	return record, nil
}

func TestUserServiceGetProfileSeedsFromFirebase(t *testing.T) {
	ctx := context.Background()
	current := time.Date(2024, 5, 1, 10, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		t := current
		current = current.Add(time.Second)
		return t
	}
	repo := newMemoryUserRepo(clock)
	firebase := &stubFirebase{records: map[string]*firebaseauth.UserRecord{
		"user-1": {
			UserInfo: &firebaseauth.UserInfo{
				UID:         "user-1",
				Email:       "USER@example.com",
				DisplayName: "Firebase User",
				ProviderID:  "firebase",
			},
			ProviderUserInfo: []*firebaseauth.UserInfo{
				&firebaseauth.UserInfo{
					ProviderID: "google.com",
					UID:        "google-uid",
					Email:      "user@gmail.com",
				},
			},
			CustomClaims: map[string]any{
				"role":   "staff",
				"locale": "ja-JP",
			},
		},
	}}
	audits := &captureAuditRepo{}

	svc, err := NewUserService(UserServiceDeps{
		Users:    repo,
		Audit:    audits,
		Firebase: firebase,
		Clock:    clock,
	})
	if err != nil {
		t.Fatalf("new user service: %v", err)
	}

	profile, err := svc.GetProfile(ctx, "user-1")
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}

	if profile.Email != "user@example.com" {
		t.Fatalf("expected lower-cased email, got %q", profile.Email)
	}
	if profile.Locale != "ja-JP" {
		t.Fatalf("expected locale ja-JP, got %q", profile.Locale)
	}
	if len(profile.Roles) != 2 {
		t.Fatalf("expected two roles, got %#v", profile.Roles)
	}
	if profile.LastSyncTime.IsZero() {
		t.Fatalf("expected last sync time to be set")
	}
	if len(profile.ProviderData) == 0 {
		t.Fatalf("expected provider data to be captured")
	}
}

func TestUserServiceUpdateProfileValidation(t *testing.T) {
	ctx := context.Background()
	current := time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		t := current
		current = current.Add(time.Second)
		return t
	}
	repo := newMemoryUserRepo(clock)
	firebase := &stubFirebase{records: map[string]*firebaseauth.UserRecord{}}
	audits := &captureAuditRepo{}

	svc, err := NewUserService(UserServiceDeps{
		Users:    repo,
		Audit:    audits,
		Firebase: firebase,
		Clock:    clock,
	})
	if err != nil {
		t.Fatalf("new user service: %v", err)
	}

	_, err = repo.UpdateProfile(ctx, domain.UserProfile{ID: "user-2", DisplayName: "Initial", Email: "initial@example.com", IsActive: true})
	if err != nil {
		t.Fatalf("seed profile: %v", err)
	}

	invalid := "x"
	_, err = svc.UpdateProfile(ctx, UpdateProfileCommand{
		UserID:      "user-2",
		ActorID:     "actor-1",
		DisplayName: &invalid,
	})
	if !errors.Is(err, errInvalidDisplayName) {
		t.Fatalf("expected invalid display name error, got %v", err)
	}
}

func TestUserServiceUpdateProfileSuccess(t *testing.T) {
	ctx := context.Background()
	current := time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		t := current
		current = current.Add(time.Second)
		return t
	}
	repo := newMemoryUserRepo(clock)
	firebase := &stubFirebase{records: map[string]*firebaseauth.UserRecord{}}
	audits := &captureAuditRepo{}

	svc, err := NewUserService(UserServiceDeps{
		Users:    repo,
		Audit:    audits,
		Firebase: firebase,
		Clock:    clock,
	})
	if err != nil {
		t.Fatalf("new user service: %v", err)
	}

	profile, err := repo.UpdateProfile(ctx, domain.UserProfile{ID: "user-3", DisplayName: "Initial", Email: "user3@example.com", IsActive: true})
	if err != nil {
		t.Fatalf("seed profile: %v", err)
	}

	newName := "Updated Name"
	locale := "ja-jp"
	prefs := map[string]bool{"EMAIL": true, "sms": false}
	avatar := "asset-123"

	updated, err := svc.UpdateProfile(ctx, UpdateProfileCommand{
		UserID:            "user-3",
		ActorID:           "actor-2",
		DisplayName:       &newName,
		Locale:            &locale,
		NotificationPrefs: prefs,
		AvatarAssetID:     &avatar,
		ExpectedSyncTime:  &profile.LastSyncTime,
	})
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}

	if updated.DisplayName != newName {
		t.Fatalf("expected display name updated, got %q", updated.DisplayName)
	}
	if updated.Locale != "ja-JP" {
		t.Fatalf("expected locale canonicalised, got %q", updated.Locale)
	}
	if updated.NotificationPrefs["email"] != true || updated.NotificationPrefs["sms"] != false {
		t.Fatalf("unexpected notification prefs %#v", updated.NotificationPrefs)
	}
	if updated.AvatarAssetID == nil || *updated.AvatarAssetID != avatar {
		t.Fatalf("expected avatar asset id, got %v", updated.AvatarAssetID)
	}

	if len(audits.entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(audits.entries))
	}
	entry := audits.entries[0]
	if entry.Action != auditActionProfileUpdate {
		t.Fatalf("unexpected audit action %s", entry.Action)
	}
	if _, ok := entry.Diff["displayName"]; !ok {
		t.Fatalf("expected displayName diff in audit")
	}
}

func TestUserServiceMaskProfile(t *testing.T) {
	ctx := context.Background()
	current := time.Date(2024, 7, 1, 8, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		t := current
		current = current.Add(time.Second)
		return t
	}
	repo := newMemoryUserRepo(clock)
	firebase := &stubFirebase{records: map[string]*firebaseauth.UserRecord{}}
	audits := &captureAuditRepo{}

	svc, err := NewUserService(UserServiceDeps{
		Users:    repo,
		Audit:    audits,
		Firebase: firebase,
		Clock:    clock,
	})
	if err != nil {
		t.Fatalf("new user service: %v", err)
	}

	_, err = repo.UpdateProfile(ctx, domain.UserProfile{
		ID:                "user-4",
		DisplayName:       "Mask Me",
		Email:             "mask@example.com",
		PhoneNumber:       "+819012345678",
		AvatarAssetID:     ptr("asset-old"),
		NotificationPrefs: domain.NotificationPreferences{"email": true},
		IsActive:          true,
	})
	if err != nil {
		t.Fatalf("seed profile: %v", err)
	}

	masked, err := svc.MaskProfile(ctx, MaskProfileCommand{
		UserID:  "user-4",
		ActorID: "admin-1",
		Reason:  "GDPR request",
	})
	if err != nil {
		t.Fatalf("mask profile: %v", err)
	}

	if masked.Email == "" {
		t.Fatalf("expected masked email, got empty string")
	}
	if masked.PhoneNumber != "" {
		t.Fatalf("expected phone cleared, got %q", masked.PhoneNumber)
	}
	if masked.AvatarAssetID != nil {
		t.Fatalf("expected avatar cleared")
	}
	if len(masked.NotificationPrefs) != 0 {
		t.Fatalf("expected notification prefs cleared, got %#v", masked.NotificationPrefs)
	}
	if masked.PiiMaskedAt == nil {
		t.Fatalf("expected piiMaskedAt set")
	}
	if masked.IsActive {
		t.Fatalf("expected user inactive after masking")
	}
	if len(audits.entries) != 1 || audits.entries[0].Action != auditActionProfileMask {
		t.Fatalf("expected mask audit entry")
	}
}

func TestUserServiceSetUserActiveConflict(t *testing.T) {
	ctx := context.Background()
	current := time.Date(2024, 8, 1, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		t := current
		current = current.Add(time.Second)
		return t
	}
	repo := newMemoryUserRepo(clock)
	firebase := &stubFirebase{records: map[string]*firebaseauth.UserRecord{}}
	audits := &captureAuditRepo{}

	svc, err := NewUserService(UserServiceDeps{
		Users:    repo,
		Audit:    audits,
		Firebase: firebase,
		Clock:    clock,
	})
	if err != nil {
		t.Fatalf("new user service: %v", err)
	}

	profile, err := repo.UpdateProfile(ctx, domain.UserProfile{ID: "user-5", DisplayName: "Initial", Email: "user5@example.com", IsActive: true})
	if err != nil {
		t.Fatalf("seed profile: %v", err)
	}

	expected := profile.LastSyncTime.Add(-time.Second)
	_, err = svc.UpdateProfile(ctx, UpdateProfileCommand{
		UserID:           "user-5",
		ActorID:          "actor-3",
		DisplayName:      ptr("Another"),
		ExpectedSyncTime: &expected,
	})
	if !errors.Is(err, errProfileConflict) {
		t.Fatalf("expected profile conflict, got %v", err)
	}
}

func ptr[T any](value T) *T {
	return &value
}

func cloneProfile(profile domain.UserProfile) domain.UserProfile {
	copy := profile
	if profile.NotificationPrefs != nil {
		copy.NotificationPrefs = domain.NotificationPreferences{}
		for k, v := range profile.NotificationPrefs {
			copy.NotificationPrefs[k] = v
		}
	}
	if profile.ProviderData != nil {
		copy.ProviderData = append([]domain.AuthProvider(nil), profile.ProviderData...)
	}
	if profile.AvatarAssetID != nil {
		value := *profile.AvatarAssetID
		copy.AvatarAssetID = &value
	}
	if profile.PiiMaskedAt != nil {
		t := *profile.PiiMaskedAt
		copy.PiiMaskedAt = &t
	}
	return copy
}

var _ repositories.UserRepository = (*memoryUserRepo)(nil)
var _ repositories.AuditLogRepository = (*captureAuditRepo)(nil)
