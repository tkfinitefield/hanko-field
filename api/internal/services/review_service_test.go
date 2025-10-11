package services

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"slices"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

func TestReviewServiceCreateSanitizesAndEmitsEvent(t *testing.T) {
	now := time.Date(2025, 5, 20, 10, 30, 0, 0, time.UTC)
	repo := newMemoryReviewRepo()
	orderRepo := &stubOrderRepository{
		orders: map[string]domain.Order{
			"order-1": {
				ID:     "order-1",
				UserID: "user-1",
				Status: domain.OrderStatusCompleted,
			},
		},
	}
	events := &captureReviewEvents{}

	svc, err := NewReviewService(ReviewServiceDeps{
		Reviews: repo,
		Orders:  orderRepo,
		Clock: func() time.Time {
			return now
		},
		IDGenerator: func() string { return "rev_test" },
		Events:      events,
	})
	if err != nil {
		t.Fatalf("new review service: %v", err)
	}

	ctx := context.Background()
	review, err := svc.Create(ctx, CreateReviewCommand{
		OrderID: "order-1",
		UserID:  "user-1",
		Rating:  5,
		Comment: "  Great\nproduct  ",
		ActorID: "user-1",
	})
	if err != nil {
		t.Fatalf("create review: %v", err)
	}

	if review.ID != "rev_test" {
		t.Fatalf("expected review id rev_test, got %s", review.ID)
	}
	if review.Comment != "Great\nproduct" {
		t.Fatalf("expected sanitized comment with newline preserved, got %q", review.Comment)
	}
	if review.Status != domain.ReviewStatusPending {
		t.Fatalf("expected status pending, got %s", review.Status)
	}

	if len(events.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events.events))
	}
	event := events.events[0]
	if event.Type != reviewEventCreated {
		t.Fatalf("expected event type %s, got %s", reviewEventCreated, event.Type)
	}
	if event.ReviewID != review.ID {
		t.Fatalf("expected review id %s, got %s", review.ID, event.ReviewID)
	}
	if event.OccurredAt != now {
		t.Fatalf("expected occurred at %s, got %s", now, event.OccurredAt)
	}
}

func TestReviewServiceCreatePreventsDuplicateReviews(t *testing.T) {
	now := time.Date(2025, 5, 20, 11, 0, 0, 0, time.UTC)
	repo := newMemoryReviewRepo()
	existing := domain.Review{
		ID:        "rev_existing",
		OrderRef:  "order-1",
		UserRef:   "user-1",
		Rating:    4,
		Comment:   "existing",
		Status:    domain.ReviewStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if _, err := repo.Insert(context.Background(), existing); err != nil {
		t.Fatalf("seed insert: %v", err)
	}

	orderRepo := &stubOrderRepository{
		orders: map[string]domain.Order{
			"order-1": {
				ID:     "order-1",
				UserID: "user-1",
				Status: domain.OrderStatusCompleted,
			},
		},
	}

	svc, err := NewReviewService(ReviewServiceDeps{
		Reviews: repo,
		Orders:  orderRepo,
		Clock: func() time.Time {
			return now
		},
		IDGenerator: func() string { return "rev_new" },
	})
	if err != nil {
		t.Fatalf("new review service: %v", err)
	}

	_, err = svc.Create(context.Background(), CreateReviewCommand{
		OrderID: "order-1",
		UserID:  "user-1",
		Rating:  5,
		Comment: "duplicate",
		ActorID: "user-1",
	})
	if !errors.Is(err, ErrReviewConflict) {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

func TestReviewServiceCreateRejectsProfanity(t *testing.T) {
	now := time.Date(2025, 5, 20, 11, 30, 0, 0, time.UTC)
	repo := newMemoryReviewRepo()
	orderRepo := &stubOrderRepository{
		orders: map[string]domain.Order{
			"order-1": {
				ID:     "order-1",
				UserID: "user-1",
				Status: domain.OrderStatusCompleted,
			},
		},
	}

	svc, err := NewReviewService(ReviewServiceDeps{
		Reviews: repo,
		Orders:  orderRepo,
		Clock: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatalf("new review service: %v", err)
	}

	_, err = svc.Create(context.Background(), CreateReviewCommand{
		OrderID: "order-1",
		UserID:  "user-1",
		Rating:  4,
		Comment: "This product is shit",
		ActorID: "user-1",
	})
	if !errors.Is(err, ErrReviewInvalidInput) {
		t.Fatalf("expected invalid input error for profanity, got %v", err)
	}
}

func TestReviewServiceModerateTransitionsAndEmitsEvent(t *testing.T) {
	now := time.Date(2025, 5, 20, 12, 0, 0, 0, time.UTC)
	repo := newMemoryReviewRepo()
	review := domain.Review{
		ID:        "rev_pending",
		OrderRef:  "order-1",
		UserRef:   "user-1",
		Rating:    5,
		Comment:   "great",
		Status:    domain.ReviewStatusPending,
		CreatedAt: now.Add(-time.Hour),
		UpdatedAt: now.Add(-time.Hour),
	}
	if _, err := repo.Insert(context.Background(), review); err != nil {
		t.Fatalf("seed insert: %v", err)
	}

	orderRepo := &stubOrderRepository{}
	events := &captureReviewEvents{}

	svc, err := NewReviewService(ReviewServiceDeps{
		Reviews: repo,
		Orders:  orderRepo,
		Clock: func() time.Time {
			return now
		},
		Events: events,
	})
	if err != nil {
		t.Fatalf("new review service: %v", err)
	}

	ctx := context.Background()
	approved, err := svc.Moderate(ctx, ModerateReviewCommand{
		ReviewID: "rev_pending",
		ActorID:  "moderator-1",
		Status:   domain.ReviewStatusApproved,
	})
	if err != nil {
		t.Fatalf("moderate: %v", err)
	}

	if approved.Status != domain.ReviewStatusApproved {
		t.Fatalf("expected approved status, got %s", approved.Status)
	}
	if approved.ModeratedBy == nil || *approved.ModeratedBy != "moderator-1" {
		t.Fatalf("expected moderated by moderator-1, got %v", approved.ModeratedBy)
	}
	if approved.ModeratedAt == nil || approved.ModeratedAt.IsZero() {
		t.Fatalf("expected moderated at set")
	}

	if len(events.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events.events))
	}
	if events.events[0].Type != reviewEventApproved {
		t.Fatalf("expected event type %s, got %s", reviewEventApproved, events.events[0].Type)
	}

	_, err = svc.Moderate(ctx, ModerateReviewCommand{
		ReviewID: "rev_pending",
		ActorID:  "moderator-2",
		Status:   domain.ReviewStatusRejected,
	})
	if !errors.Is(err, ErrReviewInvalidState) {
		t.Fatalf("expected invalid state error, got %v", err)
	}
}

func TestReviewServiceGetByOrderAuthorization(t *testing.T) {
	now := time.Date(2025, 5, 20, 12, 30, 0, 0, time.UTC)
	repo := newMemoryReviewRepo()
	review := domain.Review{
		ID:        "rev_auth",
		OrderRef:  "order-1",
		UserRef:   "user-1",
		Rating:    5,
		Comment:   "great",
		Status:    domain.ReviewStatusApproved,
		CreatedAt: now.Add(-time.Hour),
		UpdatedAt: now.Add(-time.Hour),
	}
	if _, err := repo.Insert(context.Background(), review); err != nil {
		t.Fatalf("seed insert: %v", err)
	}

	orderRepo := &stubOrderRepository{}
	svc, err := NewReviewService(ReviewServiceDeps{
		Reviews: repo,
		Orders:  orderRepo,
		Clock: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatalf("new review service: %v", err)
	}

	// Owner can access
	result, err := svc.GetByOrder(context.Background(), GetReviewByOrderCommand{
		OrderID: "order-1",
		ActorID: "user-1",
	})
	if err != nil {
		t.Fatalf("get by order: %v", err)
	}
	if result.ID != "rev_auth" {
		t.Fatalf("expected review rev_auth, got %s", result.ID)
	}

	// Other user blocked
	_, err = svc.GetByOrder(context.Background(), GetReviewByOrderCommand{
		OrderID: "order-1",
		ActorID: "user-2",
	})
	if !errors.Is(err, ErrReviewUnauthorized) {
		t.Fatalf("expected unauthorized error, got %v", err)
	}

	// Staff override allowed
	result, err = svc.GetByOrder(context.Background(), GetReviewByOrderCommand{
		OrderID:    "order-1",
		ActorID:    "admin-1",
		AllowStaff: true,
	})
	if err != nil {
		t.Fatalf("staff get by order: %v", err)
	}
	if result.ID != "rev_auth" {
		t.Fatalf("expected review rev_auth for staff, got %s", result.ID)
	}
}

func TestReviewServiceStoreReplySanitizesAndClears(t *testing.T) {
	now := time.Date(2025, 5, 20, 13, 0, 0, 0, time.UTC)
	repo := newMemoryReviewRepo()
	review := domain.Review{
		ID:        "rev_approved",
		OrderRef:  "order-1",
		UserRef:   "user-1",
		Rating:    5,
		Comment:   "great",
		Status:    domain.ReviewStatusApproved,
		CreatedAt: now.Add(-2 * time.Hour),
		UpdatedAt: now.Add(-2 * time.Hour),
		ModeratedBy: func() *string {
			v := "moderator"
			return &v
		}(),
		ModeratedAt: func() *time.Time {
			v := now.Add(-time.Hour)
			return &v
		}(),
	}
	if _, err := repo.Insert(context.Background(), review); err != nil {
		t.Fatalf("seed insert: %v", err)
	}

	orderRepo := &stubOrderRepository{}
	events := &captureReviewEvents{}

	svc, err := NewReviewService(ReviewServiceDeps{
		Reviews: repo,
		Orders:  orderRepo,
		Clock: func() time.Time {
			return now
		},
		Events: events,
	})
	if err != nil {
		t.Fatalf("new review service: %v", err)
	}

	ctx := context.Background()
	withReply, err := svc.StoreReply(ctx, StoreReviewReplyCommand{
		ReviewID: "rev_approved",
		ActorID:  "staff-1",
		Message:  "Thanks\nfor your feedback!",
		Visible:  true,
	})
	if err != nil {
		t.Fatalf("store reply: %v", err)
	}
	if withReply.Reply == nil {
		t.Fatalf("expected reply to be set")
	}
	if withReply.Reply.Message != "Thanks\nfor your feedback!" {
		t.Fatalf("expected sanitized message with newline preserved, got %q", withReply.Reply.Message)
	}
	if withReply.Reply.AuthorRef != "staff-1" {
		t.Fatalf("expected author staff-1, got %s", withReply.Reply.AuthorRef)
	}
	if !withReply.Reply.Visible {
		t.Fatalf("expected reply visible")
	}
	if withReply.Reply.CreatedAt.IsZero() || withReply.Reply.UpdatedAt.IsZero() {
		t.Fatalf("expected reply timestamps to be set")
	}

	if len(events.events) == 0 || events.events[len(events.events)-1].Type != reviewEventReplyUpdated {
		t.Fatalf("expected reply updated event")
	}
	events.events = nil

	cleared, err := svc.StoreReply(ctx, StoreReviewReplyCommand{
		ReviewID: "rev_approved",
		ActorID:  "staff-1",
		Message:  "",
		Visible:  false,
	})
	if err != nil {
		t.Fatalf("clear reply: %v", err)
	}
	if cleared.Reply != nil {
		t.Fatalf("expected reply cleared, got %+v", cleared.Reply)
	}
	if len(events.events) != 1 || events.events[0].Type != reviewEventReplyUpdated {
		t.Fatalf("expected reply updated event on clear")
	}
}

// --- test doubles -----------------------------------------------------------------

type captureReviewEvents struct {
	events []ReviewEvent
}

func (c *captureReviewEvents) PublishReviewEvent(_ context.Context, event ReviewEvent) error {
	c.events = append(c.events, event)
	return nil
}

type memoryReviewRepo struct {
	reviews map[string]domain.Review
	byOrder map[string]string
}

func newMemoryReviewRepo() *memoryReviewRepo {
	return &memoryReviewRepo{
		reviews: make(map[string]domain.Review),
		byOrder: make(map[string]string),
	}
}

func (m *memoryReviewRepo) Insert(_ context.Context, review domain.Review) (domain.Review, error) {
	if _, exists := m.byOrder[review.OrderRef]; exists {
		return domain.Review{}, repoError{message: "duplicate", conflict: true}
	}
	m.reviews[review.ID] = copyReview(review)
	m.byOrder[review.OrderRef] = review.ID
	return copyReview(review), nil
}

func (m *memoryReviewRepo) FindByID(_ context.Context, reviewID string) (domain.Review, error) {
	review, ok := m.reviews[reviewID]
	if !ok {
		return domain.Review{}, repoError{message: "not found", notFound: true}
	}
	return copyReview(review), nil
}

func (m *memoryReviewRepo) FindByOrder(_ context.Context, orderID string) (domain.Review, error) {
	reviewID, ok := m.byOrder[orderID]
	if !ok {
		return domain.Review{}, repoError{message: "not found", notFound: true}
	}
	return copyReview(m.reviews[reviewID]), nil
}

func (m *memoryReviewRepo) ListByUser(_ context.Context, userID string, pager domain.Pagination) (domain.CursorPage[domain.Review], error) {
	var results []domain.Review
	for _, review := range m.reviews {
		if review.UserRef == userID {
			results = append(results, copyReview(review))
		}
	}

	slices.SortFunc(results, func(a, b domain.Review) int {
		switch {
		case a.CreatedAt.After(b.CreatedAt):
			return -1
		case a.CreatedAt.Before(b.CreatedAt):
			return 1
		default:
			return strings.Compare(a.ID, b.ID)
		}
	})

	start := 0
	if token, err := strconv.Atoi(pager.PageToken); err == nil {
		switch {
		case token < 0:
			start = 0
		case token >= len(results):
			start = len(results)
		default:
			start = token
		}
	}

	pageSize := pager.PageSize
	remaining := len(results) - start
	if remaining < 0 {
		remaining = 0
	}
	if pageSize <= 0 || pageSize > remaining {
		pageSize = remaining
	}

	end := start + pageSize
	if end > len(results) {
		end = len(results)
	}

	var pageItems []domain.Review
	if start < len(results) {
		pageItems = results[start:end]
	}

	nextToken := ""
	if end < len(results) {
		nextToken = strconv.Itoa(end)
	}

	return domain.CursorPage[domain.Review]{
		Items:         pageItems,
		NextPageToken: nextToken,
	}, nil
}

func (m *memoryReviewRepo) UpdateStatus(_ context.Context, reviewID string, status domain.ReviewStatus, update repositories.ReviewModerationUpdate) (domain.Review, error) {
	review, ok := m.reviews[reviewID]
	if !ok {
		return domain.Review{}, repoError{message: "not found", notFound: true}
	}
	review.Status = status
	review.UpdatedAt = update.ModeratedAt
	review.ModeratedAt = &update.ModeratedAt
	review.ModeratedBy = &update.ModeratedBy
	m.reviews[reviewID] = copyReview(review)
	return copyReview(review), nil
}

func (m *memoryReviewRepo) UpdateReply(_ context.Context, reviewID string, reply *domain.ReviewReply, updatedAt time.Time) (domain.Review, error) {
	review, ok := m.reviews[reviewID]
	if !ok {
		return domain.Review{}, repoError{message: "not found", notFound: true}
	}
	if reply != nil {
		cp := *reply
		review.Reply = &cp
	} else {
		review.Reply = nil
	}
	review.UpdatedAt = updatedAt
	m.reviews[reviewID] = copyReview(review)
	return copyReview(review), nil
}

type repoError struct {
	message  string
	notFound bool
	conflict bool
	unavail  bool
}

func (e repoError) Error() string {
	return e.message
}

func (e repoError) IsNotFound() bool {
	return e.notFound
}

func (e repoError) IsConflict() bool {
	return e.conflict
}

func (e repoError) IsUnavailable() bool {
	return e.unavail
}

type stubOrderRepository struct {
	orders map[string]domain.Order
}

func (s *stubOrderRepository) Insert(context.Context, domain.Order) error {
	return errors.New("not implemented")
}

func (s *stubOrderRepository) Update(context.Context, domain.Order) error {
	return errors.New("not implemented")
}

func (s *stubOrderRepository) FindByID(_ context.Context, orderID string) (domain.Order, error) {
	if s.orders == nil {
		return domain.Order{}, repoError{message: "not found", notFound: true}
	}
	order, ok := s.orders[orderID]
	if !ok {
		return domain.Order{}, repoError{message: "not found", notFound: true}
	}
	return order, nil
}

func (s *stubOrderRepository) List(context.Context, repositories.OrderListFilter) (domain.CursorPage[domain.Order], error) {
	return domain.CursorPage[domain.Order]{}, errors.New("not implemented")
}

func copyReview(in domain.Review) domain.Review {
	out := in
	if in.Reply != nil {
		replyCopy := *in.Reply
		out.Reply = &replyCopy
	}
	if in.ModeratedBy != nil {
		v := *in.ModeratedBy
		out.ModeratedBy = &v
	}
	if in.ModeratedAt != nil {
		ts := *in.ModeratedAt
		out.ModeratedAt = &ts
	}
	return out
}
