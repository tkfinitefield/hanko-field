package services

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sort"
	"strings"
	"testing"
	"time"

	firebaseauth "firebase.google.com/go/v4/auth"
	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type memoryUserRepo struct {
	store map[string]domain.UserProfile
	clock func() time.Time
}

type memoryAddressRepo struct {
	store map[string]map[string]domain.Address
	clock func() time.Time
	seq   int
}

type memoryPaymentMethodRepo struct {
	store map[string]map[string]domain.PaymentMethod
	clock func() time.Time
	seq   int
}

type memoryFavoriteRepo struct {
	store map[string]map[string]domain.FavoriteDesign
}

type memoryDesignRepo struct {
	store map[string]domain.Design
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

func newMemoryAddressRepo(clock func() time.Time) *memoryAddressRepo {
	return &memoryAddressRepo{
		store: make(map[string]map[string]domain.Address),
		clock: clock,
	}
}

func newMemoryPaymentMethodRepo(clock func() time.Time) *memoryPaymentMethodRepo {
	return &memoryPaymentMethodRepo{
		store: make(map[string]map[string]domain.PaymentMethod),
		clock: clock,
	}
}

func newMemoryFavoriteRepo() *memoryFavoriteRepo {
	return &memoryFavoriteRepo{
		store: make(map[string]map[string]domain.FavoriteDesign),
	}
}

func newMemoryDesignRepo() *memoryDesignRepo {
	return &memoryDesignRepo{
		store: make(map[string]domain.Design),
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

func (m *memoryAddressRepo) List(_ context.Context, userID string) ([]domain.Address, error) {
	if m.store == nil {
		m.store = make(map[string]map[string]domain.Address)
	}
	entries := m.store[userID]
	result := make([]domain.Address, 0, len(entries))
	for _, addr := range entries {
		result = append(result, addr)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	return result, nil
}

func (m *memoryAddressRepo) Upsert(_ context.Context, userID string, addressID *string, addr domain.Address) (domain.Address, error) {
	if m.store == nil {
		m.store = make(map[string]map[string]domain.Address)
	}
	bucket := m.store[userID]
	if bucket == nil {
		bucket = make(map[string]domain.Address)
		m.store[userID] = bucket
	}

	id := ""
	if addressID != nil {
		id = strings.TrimSpace(*addressID)
	}
	if id == "" {
		if strings.TrimSpace(addr.ID) != "" {
			id = strings.TrimSpace(addr.ID)
		} else {
			m.seq++
			id = fmt.Sprintf("addr-%d", m.seq)
		}
	}

	existing, found := bucket[id]
	if !found {
		if addr.CreatedAt.IsZero() {
			addr.CreatedAt = m.clock().UTC()
		} else {
			addr.CreatedAt = addr.CreatedAt.UTC()
		}
	} else {
		if addr.CreatedAt.IsZero() {
			addr.CreatedAt = existing.CreatedAt
		}
	}
	addr.ID = id
	addr.UpdatedAt = m.clock().UTC()
	if addr.NormalizedHash == "" {
		addr.NormalizedHash = addressFingerprint(addr)
	}

	bucket[id] = addr

	if addr.DefaultShipping {
		for key, other := range bucket {
			if key == id {
				continue
			}
			if other.DefaultShipping {
				other.DefaultShipping = false
				bucket[key] = other
			}
		}
	}
	if addr.DefaultBilling {
		for key, other := range bucket {
			if key == id {
				continue
			}
			if other.DefaultBilling {
				other.DefaultBilling = false
				bucket[key] = other
			}
		}
	}

	return addr, nil
}

func (m *memoryAddressRepo) Delete(_ context.Context, userID string, addressID string) error {
	if m.store == nil {
		return nil
	}
	if bucket := m.store[userID]; bucket != nil {
		delete(bucket, addressID)
	}
	return nil
}

func (m *memoryAddressRepo) Get(_ context.Context, userID string, addressID string) (domain.Address, error) {
	if bucket := m.store[userID]; bucket != nil {
		if addr, ok := bucket[addressID]; ok {
			return addr, nil
		}
	}
	return domain.Address{}, errors.New("not found")
}

func (m *memoryAddressRepo) FindByHash(_ context.Context, userID string, hash string) (domain.Address, bool, error) {
	bucket := m.store[userID]
	if bucket == nil {
		return domain.Address{}, false, nil
	}
	for _, addr := range bucket {
		if addr.NormalizedHash == hash {
			return addr, true, nil
		}
	}
	return domain.Address{}, false, nil
}

func (m *memoryAddressRepo) HasAny(_ context.Context, userID string) (bool, error) {
	bucket := m.store[userID]
	return bucket != nil && len(bucket) > 0, nil
}

func (m *memoryAddressRepo) SetDefaultFlags(_ context.Context, userID string, addressID string, shipping, billing *bool) (domain.Address, error) {
	bucket := m.store[userID]
	if bucket == nil {
		return domain.Address{}, errors.New("not found")
	}
	addr, ok := bucket[addressID]
	if !ok {
		return domain.Address{}, errors.New("not found")
	}
	if shipping != nil && *shipping {
		for key, other := range bucket {
			if key == addressID {
				continue
			}
			if other.DefaultShipping {
				other.DefaultShipping = false
				bucket[key] = other
			}
		}
		addr.DefaultShipping = true
	}
	if billing != nil && *billing {
		for key, other := range bucket {
			if key == addressID {
				continue
			}
			if other.DefaultBilling {
				other.DefaultBilling = false
				bucket[key] = other
			}
		}
		addr.DefaultBilling = true
	}
	addr.UpdatedAt = m.clock().UTC()
	bucket[addressID] = addr
	return addr, nil
}

func (m *memoryPaymentMethodRepo) List(_ context.Context, userID string) ([]domain.PaymentMethod, error) {
	if m.store == nil {
		return nil, nil
	}
	bucket := m.store[userID]
	result := make([]domain.PaymentMethod, 0, len(bucket))
	for _, pm := range bucket {
		result = append(result, clonePaymentMethod(pm))
	}
	sort.Slice(result, func(i, j int) bool {
		switch {
		case result[i].CreatedAt.After(result[j].CreatedAt):
			return true
		case result[i].CreatedAt.Before(result[j].CreatedAt):
			return false
		default:
			return result[i].ID < result[j].ID
		}
	})
	return result, nil
}

func (m *memoryPaymentMethodRepo) Insert(_ context.Context, userID string, method domain.PaymentMethod) (domain.PaymentMethod, error) {
	if m.store == nil {
		m.store = make(map[string]map[string]domain.PaymentMethod)
	}
	bucket := m.store[userID]
	if bucket == nil {
		bucket = make(map[string]domain.PaymentMethod)
		m.store[userID] = bucket
	}

	token := strings.TrimSpace(method.Token)
	for _, existing := range bucket {
		if strings.TrimSpace(existing.Token) == token {
			return domain.PaymentMethod{}, &repoErr{err: errors.New("duplicate token"), conflict: true}
		}
	}

	id := strings.TrimSpace(method.ID)
	if id == "" {
		m.seq++
		id = fmt.Sprintf("pm-%d", m.seq)
	}

	now := m.clock().UTC()
	if method.CreatedAt.IsZero() {
		method.CreatedAt = now
	} else {
		method.CreatedAt = method.CreatedAt.UTC()
	}
	method.UpdatedAt = now
	method.ID = id

	if method.IsDefault {
		for key, other := range bucket {
			if other.IsDefault {
				other.IsDefault = false
				bucket[key] = other
			}
		}
	}

	bucket[id] = clonePaymentMethod(method)
	return clonePaymentMethod(method), nil
}

func (m *memoryPaymentMethodRepo) Delete(_ context.Context, userID string, paymentMethodID string) error {
	if m.store == nil {
		return nil
	}
	if bucket := m.store[userID]; bucket != nil {
		delete(bucket, paymentMethodID)
	}
	return nil
}

func (m *memoryPaymentMethodRepo) Get(_ context.Context, userID string, paymentMethodID string) (domain.PaymentMethod, error) {
	if m.store == nil {
		return domain.PaymentMethod{}, errors.New("not found")
	}
	if bucket := m.store[userID]; bucket != nil {
		if pm, ok := bucket[paymentMethodID]; ok {
			return clonePaymentMethod(pm), nil
		}
	}
	return domain.PaymentMethod{}, errors.New("not found")
}

func (m *memoryPaymentMethodRepo) SetDefault(_ context.Context, userID string, paymentMethodID string) (domain.PaymentMethod, error) {
	bucket := m.store[userID]
	if bucket == nil {
		return domain.PaymentMethod{}, errors.New("not found")
	}
	target, ok := bucket[paymentMethodID]
	if !ok {
		return domain.PaymentMethod{}, errors.New("not found")
	}
	for key, pm := range bucket {
		if key == paymentMethodID {
			continue
		}
		if pm.IsDefault {
			pm.IsDefault = false
			pm.UpdatedAt = m.clock().UTC()
			bucket[key] = pm
		}
	}
	target.IsDefault = true
	target.UpdatedAt = m.clock().UTC()
	bucket[paymentMethodID] = target
	return clonePaymentMethod(target), nil
}

func clonePaymentMethod(method domain.PaymentMethod) domain.PaymentMethod {
	copied := method
	return copied
}

func (m *memoryFavoriteRepo) List(_ context.Context, userID string, pager domain.Pagination) (domain.CursorPage[domain.FavoriteDesign], error) {
	bucket := m.store[userID]
	if bucket == nil {
		return domain.CursorPage[domain.FavoriteDesign]{}, nil
	}
	items := make([]domain.FavoriteDesign, 0, len(bucket))
	for _, fav := range bucket {
		items = append(items, fav)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].AddedAt.Equal(items[j].AddedAt) {
			return items[i].DesignID > items[j].DesignID
		}
		return items[i].AddedAt.After(items[j].AddedAt)
	})
	if pager.PageSize > 0 && pager.PageSize < len(items) {
		items = items[:pager.PageSize]
	}
	result := make([]domain.FavoriteDesign, len(items))
	copy(result, items)
	return domain.CursorPage[domain.FavoriteDesign]{
		Items: result,
	}, nil
}

func (m *memoryFavoriteRepo) Put(_ context.Context, userID string, designID string, addedAt time.Time, limit int) (bool, error) {
	if m.store == nil {
		m.store = make(map[string]map[string]domain.FavoriteDesign)
	}
	bucket := m.store[userID]
	if bucket == nil {
		bucket = make(map[string]domain.FavoriteDesign)
		m.store[userID] = bucket
	}
	if _, ok := bucket[designID]; ok {
		return false, nil
	}
	if limit > 0 && len(bucket) >= limit {
		return false, status.Error(codes.FailedPrecondition, "favorite limit reached")
	}
	bucket[designID] = domain.FavoriteDesign{DesignID: designID, AddedAt: addedAt.UTC()}
	return true, nil
}

func (m *memoryFavoriteRepo) Delete(_ context.Context, userID string, designID string) error {
	if m.store == nil {
		return nil
	}
	if bucket := m.store[userID]; bucket != nil {
		delete(bucket, designID)
	}
	return nil
}

func (m *memoryDesignRepo) Insert(_ context.Context, design domain.Design) error {
	if m.store == nil {
		m.store = make(map[string]domain.Design)
	}
	m.store[design.ID] = cloneDesign(design)
	return nil
}

func (m *memoryDesignRepo) Update(_ context.Context, design domain.Design) error {
	if m.store == nil {
		return errors.New("not found")
	}
	m.store[design.ID] = cloneDesign(design)
	return nil
}

func (m *memoryDesignRepo) SoftDelete(_ context.Context, designID string, _ time.Time) error {
	if m.store != nil {
		delete(m.store, designID)
	}
	return nil
}

func (m *memoryDesignRepo) FindByID(_ context.Context, designID string) (domain.Design, error) {
	design, ok := m.store[designID]
	if !ok {
		return domain.Design{}, &repoErr{err: fmt.Errorf("design %s not found", designID), notFound: true}
	}
	return cloneDesign(design), nil
}

func (m *memoryDesignRepo) ListByOwner(_ context.Context, ownerID string, filter repositories.DesignListFilter) (domain.CursorPage[domain.Design], error) {
	items := make([]domain.Design, 0)
	for _, design := range m.store {
		if design.OwnerID == ownerID {
			items = append(items, cloneDesign(design))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	limit := filter.Pagination.PageSize
	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}
	if limit < len(items) {
		items = items[:limit]
	}
	return domain.CursorPage[domain.Design]{Items: items}, nil
}

func cloneDesign(design domain.Design) domain.Design {
	copy := design
	if design.Snapshot != nil {
		copy.Snapshot = maps.Clone(design.Snapshot)
	}
	return copy
}

type captureAuditService struct {
	records []AuditLogRecord
}

func (c *captureAuditService) Record(_ context.Context, record AuditLogRecord) {
	c.records = append(c.records, record)
}

func (c *captureAuditService) List(_ context.Context, _ AuditLogFilter) (domain.CursorPage[domain.AuditLogEntry], error) {
	return domain.CursorPage[domain.AuditLogEntry]{}, nil
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

type stubPaymentVerifier struct {
	verifyFunc func(ctx context.Context, provider string, token string) (PaymentMethodMetadata, error)
}

func (s *stubPaymentVerifier) VerifyPaymentMethod(ctx context.Context, provider string, token string) (PaymentMethodMetadata, error) {
	if s != nil && s.verifyFunc != nil {
		return s.verifyFunc(ctx, provider, token)
	}
	return PaymentMethodMetadata{}, errors.New("not implemented")
}

type stubInvoiceChecker struct {
	checkFunc func(ctx context.Context, userID string) (bool, error)
}

func (s *stubInvoiceChecker) HasOutstandingInvoices(ctx context.Context, userID string) (bool, error) {
	if s != nil && s.checkFunc != nil {
		return s.checkFunc(ctx, userID)
	}
	return false, nil
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
	audits := &captureAuditService{}

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

func TestUserServiceUpsertAddressCreatesDefaults(t *testing.T) {
	ctx := context.Background()
	current := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		ts := current
		current = current.Add(time.Second)
		return ts
	}

	userRepo := newMemoryUserRepo(clock)
	addressRepo := newMemoryAddressRepo(clock)
	firebase := &stubFirebase{records: map[string]*firebaseauth.UserRecord{}}

	svc, err := NewUserService(UserServiceDeps{
		Users:     userRepo,
		Addresses: addressRepo,
		Firebase:  firebase,
		Clock:     clock,
	})
	if err != nil {
		t.Fatalf("new user service: %v", err)
	}

	saved, err := svc.UpsertAddress(ctx, UpsertAddressCommand{
		UserID: "user-a",
		Address: Address{
			Recipient:  "Hanako",
			Line1:      "1-2-3",
			City:       "Chiyoda",
			PostalCode: "1000001",
			Country:    "jp",
		},
	})
	if err != nil {
		t.Fatalf("upsert address: %v", err)
	}
	if !saved.DefaultShipping || !saved.DefaultBilling {
		t.Fatalf("expected defaults set, got shipping=%v billing=%v", saved.DefaultShipping, saved.DefaultBilling)
	}
}

func TestUserServiceUpsertAddressDeduplicates(t *testing.T) {
	ctx := context.Background()
	current := time.Date(2024, 6, 5, 9, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		ts := current
		current = current.Add(time.Second)
		return ts
	}

	userRepo := newMemoryUserRepo(clock)
	addressRepo := newMemoryAddressRepo(clock)
	firebase := &stubFirebase{records: map[string]*firebaseauth.UserRecord{}}

	svc, err := NewUserService(UserServiceDeps{
		Users:     userRepo,
		Addresses: addressRepo,
		Firebase:  firebase,
		Clock:     clock,
	})
	if err != nil {
		t.Fatalf("new user service: %v", err)
	}

	first, err := svc.UpsertAddress(ctx, UpsertAddressCommand{
		UserID: "user-b",
		Address: Address{
			Recipient:  "Taro",
			Line1:      "4-5-6",
			City:       "Minato",
			PostalCode: "1050001",
			Country:    "JP",
		},
	})
	if err != nil {
		t.Fatalf("upsert initial: %v", err)
	}

	second, err := svc.UpsertAddress(ctx, UpsertAddressCommand{
		UserID: "user-b",
		Address: Address{
			Recipient:  "taro",
			Line1:      "4-5-6",
			City:       "minato",
			PostalCode: "105-0001",
			Country:    "jp",
		},
	})
	if err != nil {
		t.Fatalf("upsert duplicate: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected duplicate to reuse id %s, got %s", first.ID, second.ID)
	}
}

func TestUserServiceDeleteAddressPromotesDefault(t *testing.T) {
	ctx := context.Background()
	current := time.Date(2024, 7, 1, 8, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		ts := current
		current = current.Add(time.Second)
		return ts
	}

	userRepo := newMemoryUserRepo(clock)
	addressRepo := newMemoryAddressRepo(clock)
	firebase := &stubFirebase{records: map[string]*firebaseauth.UserRecord{}}

	svc, err := NewUserService(UserServiceDeps{
		Users:     userRepo,
		Addresses: addressRepo,
		Firebase:  firebase,
		Clock:     clock,
	})
	if err != nil {
		t.Fatalf("new user service: %v", err)
	}

	first, err := svc.UpsertAddress(ctx, UpsertAddressCommand{
		UserID: "user-c",
		Address: Address{
			Recipient:  "Ichiro",
			Line1:      "7-8-9",
			City:       "Nagoya",
			PostalCode: "4600001",
			Country:    "JP",
		},
	})
	if err != nil {
		t.Fatalf("upsert first: %v", err)
	}

	second, err := svc.UpsertAddress(ctx, UpsertAddressCommand{
		UserID: "user-c",
		Address: Address{
			Recipient:  "Jiro",
			Line1:      "10-11-12",
			City:       "Osaka",
			PostalCode: "5300001",
			Country:    "JP",
		},
	})
	if err != nil {
		t.Fatalf("upsert second: %v", err)
	}

	if err := svc.DeleteAddress(ctx, DeleteAddressCommand{UserID: "user-c", AddressID: first.ID}); err != nil {
		t.Fatalf("delete address: %v", err)
	}

	addresses, err := svc.ListAddresses(ctx, "user-c")
	if err != nil {
		t.Fatalf("list addresses: %v", err)
	}
	if len(addresses) != 1 {
		t.Fatalf("expected 1 address, got %d", len(addresses))
	}
	if !addresses[0].DefaultShipping || !addresses[0].DefaultBilling {
		t.Fatalf("expected remaining address to be default, got shipping=%v billing=%v", addresses[0].DefaultShipping, addresses[0].DefaultBilling)
	}
	if addresses[0].ID != second.ID {
		t.Fatalf("expected remaining address %s, got %s", second.ID, addresses[0].ID)
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
	audits := &captureAuditService{}

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
	audits := &captureAuditService{}

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

	if len(audits.records) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(audits.records))
	}
	entry := audits.records[0]
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
	audits := &captureAuditService{}

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
	if len(audits.records) != 1 || audits.records[0].Action != auditActionProfileMask {
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
	audits := &captureAuditService{}

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

func TestUserServicePaymentMethodsLifecycle(t *testing.T) {
	ctx := context.Background()
	current := time.Date(2024, 9, 1, 9, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		now := current
		current = current.Add(time.Second)
		return now
	}

	userRepo := newMemoryUserRepo(clock)
	paymentRepo := newMemoryPaymentMethodRepo(clock)
	firebase := &stubFirebase{records: map[string]*firebaseauth.UserRecord{}}
	verifier := &stubPaymentVerifier{
		verifyFunc: func(ctx context.Context, provider string, token string) (PaymentMethodMetadata, error) {
			return PaymentMethodMetadata{
				Token:    token,
				Brand:    "visa",
				Last4:    token[len(token)-4:],
				ExpMonth: 12,
				ExpYear:  2030,
			}, nil
		},
	}

	svc, err := NewUserService(UserServiceDeps{
		Users:           userRepo,
		PaymentMethods:  paymentRepo,
		PaymentVerifier: verifier,
		Firebase:        firebase,
		Clock:           clock,
	})
	if err != nil {
		t.Fatalf("new user service: %v", err)
	}

	first, err := svc.AddPaymentMethod(ctx, AddPaymentMethodCommand{
		UserID:   "user-pay",
		Provider: "Stripe",
		Token:    "pm_1000",
	})
	if err != nil {
		t.Fatalf("add payment method: %v", err)
	}
	if !first.IsDefault {
		t.Fatalf("expected first method to be default")
	}
	if first.Brand != "visa" || first.Last4 != "1000" {
		t.Fatalf("expected metadata populated, got brand=%s last4=%s", first.Brand, first.Last4)
	}

	second, err := svc.AddPaymentMethod(ctx, AddPaymentMethodCommand{
		UserID:   "user-pay",
		Provider: "stripe",
		Token:    "pm_2000",
	})
	if err != nil {
		t.Fatalf("add second payment method: %v", err)
	}
	if second.IsDefault {
		t.Fatalf("expected second method not default by default")
	}

	third, err := svc.AddPaymentMethod(ctx, AddPaymentMethodCommand{
		UserID:      "user-pay",
		Provider:    "stripe",
		Token:       "pm_3000",
		MakeDefault: true,
	})
	if err != nil {
		t.Fatalf("add third payment method: %v", err)
	}
	if !third.IsDefault {
		t.Fatalf("expected make_default to mark card as default")
	}

	methods, err := svc.ListPaymentMethods(ctx, "user-pay")
	if err != nil {
		t.Fatalf("list payment methods: %v", err)
	}
	if len(methods) != 3 {
		t.Fatalf("expected 3 methods, got %d", len(methods))
	}
	if methods[0].ID != third.ID {
		t.Fatalf("expected default method first, got %s", methods[0].ID)
	}
}

func TestUserServiceAddPaymentMethodDuplicate(t *testing.T) {
	ctx := context.Background()
	current := time.Date(2024, 9, 2, 9, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		now := current
		current = current.Add(time.Second)
		return now
	}

	userRepo := newMemoryUserRepo(clock)
	paymentRepo := newMemoryPaymentMethodRepo(clock)
	firebase := &stubFirebase{records: map[string]*firebaseauth.UserRecord{}}
	verifier := &stubPaymentVerifier{
		verifyFunc: func(ctx context.Context, provider string, token string) (PaymentMethodMetadata, error) {
			return PaymentMethodMetadata{Token: token}, nil
		},
	}

	svc, err := NewUserService(UserServiceDeps{
		Users:           userRepo,
		PaymentMethods:  paymentRepo,
		PaymentVerifier: verifier,
		Firebase:        firebase,
		Clock:           clock,
	})
	if err != nil {
		t.Fatalf("new user service: %v", err)
	}

	if _, err := svc.AddPaymentMethod(ctx, AddPaymentMethodCommand{
		UserID:   "user-dup",
		Provider: "stripe",
		Token:    "pm_dup",
	}); err != nil {
		t.Fatalf("seed payment method: %v", err)
	}

	_, err = svc.AddPaymentMethod(ctx, AddPaymentMethodCommand{
		UserID:   "user-dup",
		Provider: "stripe",
		Token:    "pm_dup",
	})
	if !errors.Is(err, errPaymentMethodDuplicate) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestUserServiceRemovePaymentMethodReassignsDefault(t *testing.T) {
	ctx := context.Background()
	current := time.Date(2024, 9, 3, 9, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		now := current
		current = current.Add(time.Second)
		return now
	}

	userRepo := newMemoryUserRepo(clock)
	paymentRepo := newMemoryPaymentMethodRepo(clock)
	firebase := &stubFirebase{records: map[string]*firebaseauth.UserRecord{}}
	verifier := &stubPaymentVerifier{
		verifyFunc: func(ctx context.Context, provider string, token string) (PaymentMethodMetadata, error) {
			return PaymentMethodMetadata{Token: token}, nil
		},
	}

	svc, err := NewUserService(UserServiceDeps{
		Users:           userRepo,
		PaymentMethods:  paymentRepo,
		PaymentVerifier: verifier,
		Firebase:        firebase,
		Clock:           clock,
	})
	if err != nil {
		t.Fatalf("new user service: %v", err)
	}

	first, err := svc.AddPaymentMethod(ctx, AddPaymentMethodCommand{UserID: "user-rem", Provider: "stripe", Token: "pm_one"})
	if err != nil {
		t.Fatalf("add first: %v", err)
	}
	second, err := svc.AddPaymentMethod(ctx, AddPaymentMethodCommand{UserID: "user-rem", Provider: "stripe", Token: "pm_two"})
	if err != nil {
		t.Fatalf("add second: %v", err)
	}
	third, err := svc.AddPaymentMethod(ctx, AddPaymentMethodCommand{UserID: "user-rem", Provider: "stripe", Token: "pm_three", MakeDefault: true})
	if err != nil {
		t.Fatalf("add third: %v", err)
	}

	if err := svc.RemovePaymentMethod(ctx, RemovePaymentMethodCommand{
		UserID:          "user-rem",
		PaymentMethodID: third.ID,
	}); err != nil {
		t.Fatalf("remove default: %v", err)
	}

	methods, err := svc.ListPaymentMethods(ctx, "user-rem")
	if err != nil {
		t.Fatalf("list methods: %v", err)
	}
	if len(methods) != 2 {
		t.Fatalf("expected 2 methods, got %d", len(methods))
	}
	if !methods[0].IsDefault {
		t.Fatalf("expected default reassigned, got %+v", methods[0])
	}
	if methods[0].ID != first.ID && methods[0].ID != second.ID {
		t.Fatalf("unexpected default id %s", methods[0].ID)
	}
}

func TestUserServiceRemovePaymentMethodBlockedByInvoices(t *testing.T) {
	ctx := context.Background()
	current := time.Date(2024, 9, 4, 9, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		now := current
		current = current.Add(time.Second)
		return now
	}

	userRepo := newMemoryUserRepo(clock)
	paymentRepo := newMemoryPaymentMethodRepo(clock)
	firebase := &stubFirebase{records: map[string]*firebaseauth.UserRecord{}}
	verifier := &stubPaymentVerifier{
		verifyFunc: func(ctx context.Context, provider string, token string) (PaymentMethodMetadata, error) {
			return PaymentMethodMetadata{Token: token}, nil
		},
	}

	svc, err := NewUserService(UserServiceDeps{
		Users:           userRepo,
		PaymentMethods:  paymentRepo,
		PaymentVerifier: verifier,
		Invoices: &stubInvoiceChecker{
			checkFunc: func(ctx context.Context, userID string) (bool, error) {
				return true, nil
			},
		},
		Firebase: firebase,
		Clock:    clock,
	})
	if err != nil {
		t.Fatalf("new user service: %v", err)
	}

	first, err := svc.AddPaymentMethod(ctx, AddPaymentMethodCommand{UserID: "user-inv", Provider: "stripe", Token: "pm_inv"})
	if err != nil {
		t.Fatalf("add method: %v", err)
	}

	err = svc.RemovePaymentMethod(ctx, RemovePaymentMethodCommand{
		UserID:          "user-inv",
		PaymentMethodID: first.ID,
	})
	if !errors.Is(err, errPaymentMethodInUse) {
		t.Fatalf("expected in-use error, got %v", err)
	}
}

func ptr[T any](value T) *T {
	return &value
}

func TestUserServiceToggleFavoriteAddAndList(t *testing.T) {
	ctx := context.Background()
	current := time.Date(2024, 10, 1, 11, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		now := current
		current = current.Add(time.Second)
		return now
	}

	userRepo := newMemoryUserRepo(clock)
	favoriteRepo := newMemoryFavoriteRepo()
	designRepo := newMemoryDesignRepo()
	firebase := &stubFirebase{records: map[string]*firebaseauth.UserRecord{}}

	if err := designRepo.Insert(ctx, domain.Design{ID: "design-1", OwnerID: "user-fav", Status: "draft", UpdatedAt: clock()}); err != nil {
		t.Fatalf("insert design: %v", err)
	}

	svc, err := NewUserService(UserServiceDeps{
		Users:     userRepo,
		Favorites: favoriteRepo,
		Designs:   designRepo,
		Firebase:  firebase,
		Clock:     clock,
	})
	if err != nil {
		t.Fatalf("new user service: %v", err)
	}

	if err := svc.ToggleFavorite(ctx, ToggleFavoriteCommand{UserID: "user-fav", DesignID: "design-1", Mark: true}); err != nil {
		t.Fatalf("toggle favorite add: %v", err)
	}

	page, err := svc.ListFavorites(ctx, "user-fav", Pagination{})
	if err != nil {
		t.Fatalf("list favorites: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected 1 favorite, got %d", len(page.Items))
	}
	fav := page.Items[0]
	if fav.DesignID != "design-1" {
		t.Fatalf("unexpected design id %s", fav.DesignID)
	}
	if fav.Design == nil || fav.Design.ID != "design-1" {
		t.Fatalf("expected design metadata included, got %+v", fav.Design)
	}

	// idempotent add should not error or duplicate
	if err := svc.ToggleFavorite(ctx, ToggleFavoriteCommand{UserID: "user-fav", DesignID: "design-1", Mark: true}); err != nil {
		t.Fatalf("toggle favorite idempotent add: %v", err)
	}
	page, err = svc.ListFavorites(ctx, "user-fav", Pagination{})
	if err != nil {
		t.Fatalf("list favorites: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected 1 favorite after duplicate add, got %d", len(page.Items))
	}

	// removal should succeed silently
	if err := svc.ToggleFavorite(ctx, ToggleFavoriteCommand{UserID: "user-fav", DesignID: "design-1", Mark: false}); err != nil {
		t.Fatalf("toggle favorite remove: %v", err)
	}
	page, err = svc.ListFavorites(ctx, "user-fav", Pagination{})
	if err != nil {
		t.Fatalf("list favorites: %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("expected favorites cleared, got %d", len(page.Items))
	}
}

func TestUserServiceToggleFavoriteLimit(t *testing.T) {
	ctx := context.Background()
	current := time.Date(2024, 10, 2, 9, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		now := current
		current = current.Add(time.Minute)
		return now
	}

	userRepo := newMemoryUserRepo(clock)
	favoriteRepo := newMemoryFavoriteRepo()
	designRepo := newMemoryDesignRepo()
	firebase := &stubFirebase{records: map[string]*firebaseauth.UserRecord{}}

	for i := 0; i < 200; i++ {
		designID := fmt.Sprintf("design-%03d", i)
		designRepo.Insert(ctx, domain.Design{ID: designID, OwnerID: "user-limit", Status: "draft", UpdatedAt: clock()})
		_, _ = favoriteRepo.Put(ctx, "user-limit", designID, clock(), 200)
	}

	svc, err := NewUserService(UserServiceDeps{
		Users:     userRepo,
		Favorites: favoriteRepo,
		Designs:   designRepo,
		Firebase:  firebase,
		Clock:     clock,
	})
	if err != nil {
		t.Fatalf("new user service: %v", err)
	}

	designRepo.Insert(ctx, domain.Design{ID: "design-extra", OwnerID: "user-limit", Status: "draft", UpdatedAt: clock()})

	err = svc.ToggleFavorite(ctx, ToggleFavoriteCommand{UserID: "user-limit", DesignID: "design-extra", Mark: true})
	if !errors.Is(err, ErrUserFavoriteLimitExceeded) {
		t.Fatalf("expected favorite limit error, got %v", err)
	}
}

func TestUserServiceToggleFavoriteShareable(t *testing.T) {
	ctx := context.Background()
	clock := func() time.Time { return time.Date(2024, 10, 3, 8, 0, 0, 0, time.UTC) }

	userRepo := newMemoryUserRepo(clock)
	favoriteRepo := newMemoryFavoriteRepo()
	designRepo := newMemoryDesignRepo()
	firebase := &stubFirebase{records: map[string]*firebaseauth.UserRecord{}}

	designRepo.Insert(ctx, domain.Design{ID: "shared-design", OwnerID: "other", Status: "ready", UpdatedAt: clock()})
	designRepo.Insert(ctx, domain.Design{ID: "private-design", OwnerID: "other", Status: "draft", UpdatedAt: clock()})

	svc, err := NewUserService(UserServiceDeps{
		Users:     userRepo,
		Favorites: favoriteRepo,
		Designs:   designRepo,
		Firebase:  firebase,
		Clock:     clock,
	})
	if err != nil {
		t.Fatalf("new user service: %v", err)
	}

	if err := svc.ToggleFavorite(ctx, ToggleFavoriteCommand{UserID: "user-share", DesignID: "shared-design", Mark: true}); err != nil {
		t.Fatalf("toggle favorite shareable: %v", err)
	}

	err = svc.ToggleFavorite(ctx, ToggleFavoriteCommand{UserID: "user-share", DesignID: "private-design", Mark: true})
	if !errors.Is(err, ErrUserFavoriteDesignForbidden) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
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
var _ AuditLogService = (*captureAuditService)(nil)
