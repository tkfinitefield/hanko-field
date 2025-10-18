package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

type stubAuditRepo struct {
	entries   []domain.AuditLogEntry
	appendErr error

	listFilter repositories.AuditLogFilter
	listResp   domain.CursorPage[domain.AuditLogEntry]
	listErr    error
}

func (s *stubAuditRepo) Append(_ context.Context, entry domain.AuditLogEntry) error {
	s.entries = append(s.entries, entry)
	return s.appendErr
}

func (s *stubAuditRepo) List(_ context.Context, filter repositories.AuditLogFilter) (domain.CursorPage[domain.AuditLogEntry], error) {
	s.listFilter = filter
	return s.listResp, s.listErr
}

type captureAuditLogger struct {
	warnings []string
}

func (c *captureAuditLogger) Warnf(format string, args ...any) {
	c.warnings = append(c.warnings, strings.TrimSpace(format))
}

func TestAuditLogServiceRecordSanitizesAndHashes(t *testing.T) {
	repo := &stubAuditRepo{}
	logger := &captureAuditLogger{}
	fixed := time.Date(2024, 5, 5, 12, 0, 0, 0, time.UTC)

	svc, err := NewAuditLogService(AuditLogServiceDeps{
		Repository: repo,
		Clock: func() time.Time {
			return fixed
		},
		Logger:   logger,
		HashSalt: "pepper:",
	})
	if err != nil {
		t.Fatalf("new audit log service: %v", err)
	}

	record := AuditLogRecord{
		Actor:                 "  /users/user-1  ",
		Action:                " user.profile.update ",
		ActorType:             "",
		TargetRef:             " /users/user-1 ",
		Severity:              "Warn",
		RequestID:             " req-123 ",
		OccurredAt:            fixed.Add(-time.Minute),
		Metadata:              map[string]any{"email": "User@example.com", "reason": "Manual Update"},
		SensitiveMetadataKeys: []string{"email"},
		Diff: map[string]AuditLogDiff{
			"displayName": {Before: "Old Name", After: "New Name"},
			"timezone":    {Before: "Asia/Tokyo", After: "Asia/Osaka"},
		},
		SensitiveDiffKeys: []string{"displayName"},
		IPAddress:         "203.0.113.42 ",
		UserAgent:         "TestAgent\r\n",
	}

	svc.Record(context.Background(), record)

	if len(repo.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(repo.entries))
	}
	entry := repo.entries[0]

	if entry.Actor != "/users/user-1" {
		t.Fatalf("unexpected actor: %q", entry.Actor)
	}
	if entry.ActorType != "user" {
		t.Fatalf("expected actor type user, got %q", entry.ActorType)
	}
	if entry.TargetRef != "/users/user-1" {
		t.Fatalf("unexpected target ref: %q", entry.TargetRef)
	}
	if entry.Severity != "warn" {
		t.Fatalf("unexpected severity: %q", entry.Severity)
	}
	if entry.RequestID != "req-123" {
		t.Fatalf("expected trimmed request id, got %q", entry.RequestID)
	}
	if entry.UserAgent != "TestAgent" {
		t.Fatalf("expected sanitized user agent, got %q", entry.UserAgent)
	}
	expectedTime := fixed.Add(-time.Minute)
	if !entry.CreatedAt.Equal(expectedTime) {
		t.Fatalf("expected CreatedAt %s, got %s", expectedTime.Format(time.RFC3339Nano), entry.CreatedAt.Format(time.RFC3339Nano))
	}
	if entry.IPHash == "" || !strings.HasPrefix(entry.IPHash, defaultHasherPrefix) {
		t.Fatalf("expected hashed ip, got %q", entry.IPHash)
	}

	email, ok := entry.Metadata["email"].(string)
	if !ok || !strings.HasPrefix(email, defaultHasherPrefix) {
		t.Fatalf("expected hashed email, got %#v", entry.Metadata["email"])
	}
	if reason, ok := entry.Metadata["reason"].(string); !ok || reason != "Manual Update" {
		t.Fatalf("expected metadata reason to be preserved, got %#v", entry.Metadata["reason"])
	}

	display := entry.Diff["displayName"].(map[string]any)
	if before := display["before"].(string); !strings.HasPrefix(before, defaultHasherPrefix) {
		t.Fatalf("expected hashed diff before, got %q", before)
	}
	if after := display["after"].(string); !strings.HasPrefix(after, defaultHasherPrefix) {
		t.Fatalf("expected hashed diff after, got %q", after)
	}

	timezone := entry.Diff["timezone"].(map[string]any)
	if timezone["before"] != "Asia/Tokyo" || timezone["after"] != "Asia/Osaka" {
		t.Fatalf("expected diff preserved, got %#v", timezone)
	}

	if len(logger.warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", logger.warnings)
	}
}

func TestAuditLogServiceRecordLogsOnFailure(t *testing.T) {
	repo := &stubAuditRepo{appendErr: errors.New("boom")}
	logger := &captureAuditLogger{}

	svc, err := NewAuditLogService(AuditLogServiceDeps{
		Repository: repo,
		Logger:     logger,
	})
	if err != nil {
		t.Fatalf("new audit log service: %v", err)
	}

	svc.Record(context.Background(), AuditLogRecord{
		Actor:     "system",
		Action:    "test.action",
		TargetRef: "resource:1",
	})

	if len(logger.warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(logger.warnings))
	}
	if len(repo.entries) != 1 {
		t.Fatalf("expected append invoked once, got %d", len(repo.entries))
	}
}

func TestAuditLogServiceListDelegates(t *testing.T) {
	repo := &stubAuditRepo{
		listResp: domain.CursorPage[domain.AuditLogEntry]{
			Items: []domain.AuditLogEntry{
				{ID: "log-1"},
			},
			NextPageToken: "next-token",
		},
	}

	svc, err := NewAuditLogService(AuditLogServiceDeps{
		Repository: repo,
	})
	if err != nil {
		t.Fatalf("new audit log service: %v", err)
	}

	page, err := svc.List(context.Background(), AuditLogFilter{
		TargetRef:  " /orders/123 ",
		Actor:      " user:1 ",
		ActorType:  " Staff ",
		Action:     " ORDER_UPDATE ",
		Pagination: Pagination{PageSize: 25, PageToken: " token "},
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if page.NextPageToken != "next-token" || len(page.Items) != 1 || page.Items[0].ID != "log-1" {
		t.Fatalf("unexpected page response: %#v", page)
	}

	if repo.listFilter.TargetRef != "/orders/123" {
		t.Fatalf("expected trimmed target ref, got %q", repo.listFilter.TargetRef)
	}
	if repo.listFilter.Actor != "user:1" {
		t.Fatalf("expected trimmed actor, got %q", repo.listFilter.Actor)
	}
	if repo.listFilter.ActorType != "Staff" {
		t.Fatalf("expected actor type preserved case, got %q", repo.listFilter.ActorType)
	}
	if repo.listFilter.Action != "ORDER_UPDATE" {
		t.Fatalf("expected action preserved, got %q", repo.listFilter.Action)
	}
	if repo.listFilter.Pagination.PageSize != 25 {
		t.Fatalf("expected page size 25, got %d", repo.listFilter.Pagination.PageSize)
	}
	if repo.listFilter.Pagination.PageToken != " token " {
		t.Fatalf("expected page token untouched, got %q", repo.listFilter.Pagination.PageToken)
	}
}

func TestAuditLogServiceHashAnyProducesStableHashes(t *testing.T) {
	repo := &stubAuditRepo{}
	service, err := NewAuditLogService(AuditLogServiceDeps{
		Repository: repo,
	})
	if err != nil {
		t.Fatalf("new audit log service: %v", err)
	}
	impl := service.(*auditLogService)

	t1 := time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 6, 2, 10, 0, 0, 0, time.UTC)

	first := map[time.Time]string{
		t1: "alpha",
		t2: "bravo",
	}
	second := map[time.Time]string{
		t2: "bravo",
		t1: "alpha",
	}

	hash1 := impl.hashAny(first)
	hash2 := impl.hashAny(second)

	if hash1 != hash2 {
		t.Fatalf("expected stable hash, got %q and %q", hash1, hash2)
	}
}
