package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/services"
)

func TestReviewHandlersCreateSuccess(t *testing.T) {
	now := time.Date(2024, 7, 15, 10, 30, 0, 0, time.UTC)
	review := services.Review{
		ID:        "rev_123",
		OrderRef:  "order-123",
		UserRef:   "user-1",
		Rating:    5,
		Comment:   "Great\n\nLoved it",
		Status:    domain.ReviewStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	var captured services.CreateReviewCommand
	service := &stubReviewService{
		createFunc: func(ctx context.Context, cmd services.CreateReviewCommand) (services.Review, error) {
			captured = cmd
			return review, nil
		},
	}

	handler := NewReviewHandlers(nil, service)
	router := NewRouter(WithReviewRoutes(handler.Routes))

	body := bytes.NewBufferString(`{"order_id":" order-123 ","rating":5,"title":" Great ","body":" Loved it "}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reviews", body)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: " user-1 "}))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.Code)
	}
	if captured.OrderID != "order-123" {
		t.Fatalf("expected order id trimmed, got %s", captured.OrderID)
	}
	if captured.UserID != "user-1" || captured.ActorID != "user-1" {
		t.Fatalf("expected user identity propagated, got user=%s actor=%s", captured.UserID, captured.ActorID)
	}
	expectedComment := "Great\n\nLoved it"
	if captured.Comment != expectedComment {
		t.Fatalf("expected comment %q, got %q", expectedComment, captured.Comment)
	}
	if captured.Rating != 5 {
		t.Fatalf("expected rating 5, got %d", captured.Rating)
	}

	var payload createReviewResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Review.ID != review.ID {
		t.Fatalf("expected review id %s, got %s", review.ID, payload.Review.ID)
	}
	if payload.Review.Status != string(domain.ReviewStatusPending) {
		t.Fatalf("expected status pending, got %s", payload.Review.Status)
	}
	if payload.Review.Comment != review.Comment {
		t.Fatalf("expected comment %q, got %q", review.Comment, payload.Review.Comment)
	}
	if payload.Review.CreatedAt != formatTime(now) {
		t.Fatalf("expected created_at %s, got %s", formatTime(now), payload.Review.CreatedAt)
	}
	if payload.Review.UpdatedAt != formatTime(now) {
		t.Fatalf("expected updated_at %s, got %s", formatTime(now), payload.Review.UpdatedAt)
	}
	if payload.Review.Reply != nil {
		t.Fatalf("expected no reply payload, got %#v", payload.Review.Reply)
	}
}

func TestReviewHandlersCreateInvalidJSON(t *testing.T) {
	handler := NewReviewHandlers(nil, &stubReviewService{})
	router := NewRouter(WithReviewRoutes(handler.Routes))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/reviews", strings.NewReader("{bad json}"))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}
}

func TestReviewHandlersCreateUnauthenticated(t *testing.T) {
	handler := NewReviewHandlers(nil, &stubReviewService{})

	req := httptest.NewRequest(http.MethodPost, "/reviews", bytes.NewBufferString(`{"order_id":"order-1","rating":4,"comment":"nice"}`))
	resp := httptest.NewRecorder()

	handler.createReview(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", resp.Code)
	}
}

func TestReviewHandlersCreateServiceErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{name: "invalid input", err: services.ErrReviewInvalidInput, expected: http.StatusBadRequest},
		{name: "conflict", err: services.ErrReviewConflict, expected: http.StatusConflict},
		{name: "unauthorized", err: services.ErrReviewUnauthorized, expected: http.StatusForbidden},
		{name: "not found", err: services.ErrReviewNotFound, expected: http.StatusNotFound},
		{name: "repository unavailable", err: newRepositoryError(false, false, true), expected: http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &stubReviewService{
				createFunc: func(ctx context.Context, cmd services.CreateReviewCommand) (services.Review, error) {
					return services.Review{}, tt.err
				},
			}

			handler := NewReviewHandlers(nil, service)
			router := NewRouter(WithReviewRoutes(handler.Routes))

			req := httptest.NewRequest(http.MethodPost, "/api/v1/reviews", bytes.NewBufferString(`{"order_id":"order-1","rating":4,"comment":"nice"}`))
			req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			if resp.Code != tt.expected {
				t.Fatalf("expected status %d, got %d", tt.expected, resp.Code)
			}
		})
	}
}

func TestReviewHandlersServiceUnavailable(t *testing.T) {
	handler := NewReviewHandlers(nil, nil)
	router := NewRouter(WithReviewRoutes(handler.Routes))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/reviews", bytes.NewBufferString(`{"order_id":"order-1","rating":4,"comment":"nice"}`))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", resp.Code)
	}
}

func TestReviewHandlersListReviewsSuccess(t *testing.T) {
	now := time.Date(2024, 7, 15, 9, 0, 0, 0, time.UTC)
	moderatedAt := now.Add(time.Hour)

	reviews := []services.Review{
		{
			ID:       "rev_approved",
			OrderRef: "order-1",
			UserRef:  "user-1",
			Rating:   5,
			Comment:  "Great product",
			Status:   domain.ReviewStatusApproved,
			Reply: &domain.ReviewReply{
				Message:   "Thanks!",
				Visible:   true,
				CreatedAt: now.Add(30 * time.Minute),
				UpdatedAt: now.Add(45 * time.Minute),
			},
			ModeratedAt: &moderatedAt,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:        "rev_rejected",
			OrderRef:  "order-2",
			UserRef:   "user-1",
			Rating:    2,
			Comment:   "Not so great",
			Status:    domain.ReviewStatusRejected,
			CreatedAt: now.Add(-time.Hour),
			UpdatedAt: now.Add(-30 * time.Minute),
		},
	}

	var capturedCmd services.ListUserReviewsCommand
	service := &stubReviewService{
		listByUserFunc: func(ctx context.Context, cmd services.ListUserReviewsCommand) (domain.CursorPage[services.Review], error) {
			capturedCmd = cmd
			return domain.CursorPage[services.Review]{
				Items:         reviews,
				NextPageToken: " 10 ",
			}, nil
		},
	}

	handler := NewReviewHandlers(nil, service)
	router := NewRouter(WithReviewRoutes(handler.Routes))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reviews", nil)
	req.URL.RawQuery = "page_size=200&page_token=%205%20"
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: " user-1 "}))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	if capturedCmd.UserID != "user-1" {
		t.Fatalf("expected user id trimmed, got %s", capturedCmd.UserID)
	}
	if capturedCmd.Pagination.PageSize != maxReviewPageSize {
		t.Fatalf("expected page size clamped to %d, got %d", maxReviewPageSize, capturedCmd.Pagination.PageSize)
	}
	if capturedCmd.Pagination.PageToken != "5" {
		t.Fatalf("expected page token trimmed to 5, got %s", capturedCmd.Pagination.PageToken)
	}

	var payload reviewListResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.NextPageToken != "10" {
		t.Fatalf("expected next page token 10, got %q", payload.NextPageToken)
	}
	if len(payload.Items) != 2 {
		t.Fatalf("expected 2 reviews, got %d", len(payload.Items))
	}

	first := payload.Items[0]
	if first.ID != "rev_approved" || first.OrderID != "order-1" {
		t.Fatalf("unexpected review payload: %#v", first)
	}
	if first.Body != "Great product" {
		t.Fatalf("expected body preserved, got %q", first.Body)
	}
	if first.Moderation.Status != string(domain.ReviewStatusApproved) {
		t.Fatalf("expected moderation status approved, got %s", first.Moderation.Status)
	}
	if first.Moderation.ModeratedAt != formatTime(moderatedAt) {
		t.Fatalf("expected moderated_at %s, got %s", formatTime(moderatedAt), first.Moderation.ModeratedAt)
	}
	if first.Reply == nil || first.Reply.Message != "Thanks!" {
		t.Fatalf("expected visible reply, got %#v", first.Reply)
	}

	second := payload.Items[1]
	if second.Status != string(domain.ReviewStatusRejected) {
		t.Fatalf("expected rejected status, got %s", second.Status)
	}
	if second.Body != "" {
		t.Fatalf("expected rejected body hidden, got %q", second.Body)
	}
	if second.Reply != nil {
		t.Fatalf("expected no reply for rejected review, got %#v", second.Reply)
	}
}

func TestReviewHandlersListReviewsFilterByOrder(t *testing.T) {
	now := time.Date(2024, 7, 15, 12, 0, 0, 0, time.UTC)
	review := services.Review{
		ID:        "rev_by_order",
		OrderRef:  "order-99",
		UserRef:   "user-2",
		Rating:    4,
		Comment:   "Solid",
		Status:    domain.ReviewStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	var captured services.GetReviewByOrderCommand
	service := &stubReviewService{
		getByOrderFunc: func(ctx context.Context, cmd services.GetReviewByOrderCommand) (services.Review, error) {
			captured = cmd
			return review, nil
		},
	}

	handler := NewReviewHandlers(nil, service)
	router := NewRouter(WithReviewRoutes(handler.Routes))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reviews", nil)
	req.URL.RawQuery = "orderId=%20order-99%20"
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: " user-2 "}))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	if captured.OrderID != "order-99" {
		t.Fatalf("expected order id trimmed, got %s", captured.OrderID)
	}
	if captured.ActorID != "user-2" {
		t.Fatalf("expected actor id user-2, got %s", captured.ActorID)
	}

	var payload reviewListResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected single review, got %d", len(payload.Items))
	}
	item := payload.Items[0]
	if item.ID != "rev_by_order" || item.OrderID != "order-99" {
		t.Fatalf("unexpected payload %#v", item)
	}
	if item.Body != "Solid" {
		t.Fatalf("expected body Solid, got %q", item.Body)
	}
}

func TestReviewHandlersListReviewsOrderNotFound(t *testing.T) {
	service := &stubReviewService{
		getByOrderFunc: func(ctx context.Context, cmd services.GetReviewByOrderCommand) (services.Review, error) {
			return services.Review{}, services.ErrReviewNotFound
		},
	}

	handler := NewReviewHandlers(nil, service)
	router := NewRouter(WithReviewRoutes(handler.Routes))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reviews", nil)
	req.URL.RawQuery = "orderId=order-missing"
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-3"}))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}

	var payload reviewListResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(payload.Items) != 0 {
		t.Fatalf("expected empty result, got %d", len(payload.Items))
	}
}

func TestReviewHandlersListReviewsUnauthenticated(t *testing.T) {
	handler := NewReviewHandlers(nil, &stubReviewService{})
	router := NewRouter(WithReviewRoutes(handler.Routes))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reviews", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", resp.Code)
	}
}

type stubReviewService struct {
	createFunc     func(ctx context.Context, cmd services.CreateReviewCommand) (services.Review, error)
	getByOrderFunc func(ctx context.Context, cmd services.GetReviewByOrderCommand) (services.Review, error)
	listByUserFunc func(ctx context.Context, cmd services.ListUserReviewsCommand) (domain.CursorPage[services.Review], error)
	moderateFunc   func(ctx context.Context, cmd services.ModerateReviewCommand) (services.Review, error)
	storeReplyFunc func(ctx context.Context, cmd services.StoreReviewReplyCommand) (services.Review, error)
}

func (s *stubReviewService) Create(ctx context.Context, cmd services.CreateReviewCommand) (services.Review, error) {
	if s == nil || s.createFunc == nil {
		return services.Review{}, nil
	}
	return s.createFunc(ctx, cmd)
}

func (s *stubReviewService) GetByOrder(ctx context.Context, cmd services.GetReviewByOrderCommand) (services.Review, error) {
	if s == nil || s.getByOrderFunc == nil {
		return services.Review{}, nil
	}
	return s.getByOrderFunc(ctx, cmd)
}

func (s *stubReviewService) ListByUser(ctx context.Context, cmd services.ListUserReviewsCommand) (domain.CursorPage[services.Review], error) {
	if s == nil || s.listByUserFunc == nil {
		return domain.CursorPage[services.Review]{}, nil
	}
	return s.listByUserFunc(ctx, cmd)
}

func (s *stubReviewService) Moderate(ctx context.Context, cmd services.ModerateReviewCommand) (services.Review, error) {
	if s == nil || s.moderateFunc == nil {
		return services.Review{}, nil
	}
	return s.moderateFunc(ctx, cmd)
}

func (s *stubReviewService) StoreReply(ctx context.Context, cmd services.StoreReviewReplyCommand) (services.Review, error) {
	if s == nil || s.storeReplyFunc == nil {
		return services.Review{}, nil
	}
	return s.storeReplyFunc(ctx, cmd)
}
