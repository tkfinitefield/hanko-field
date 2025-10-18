package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/oklog/ulid/v2"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

const (
	reviewIDPrefix          = "rev_"
	reviewEventCreated      = "review.created"
	reviewEventApproved     = "review.approved"
	reviewEventRejected     = "review.rejected"
	reviewEventReplyUpdated = "review.reply.updated"
)

var (
	// ErrReviewInvalidInput indicates validation failures for review operations.
	ErrReviewInvalidInput = errors.New("review: invalid input")
	// ErrReviewNotFound indicates a review could not be located.
	ErrReviewNotFound = errors.New("review: not found")
	// ErrReviewUnauthorized indicates the actor is not allowed to access the review.
	ErrReviewUnauthorized = errors.New("review: unauthorized")
	// ErrReviewConflict signals duplicate submissions or conflicting updates.
	ErrReviewConflict = errors.New("review: conflict")
	// ErrReviewInvalidState is returned when an invalid status transition is attempted.
	ErrReviewInvalidState = errors.New("review: invalid state transition")
)

// ReviewServiceDeps bundles collaborators required to construct a ReviewService.
type ReviewServiceDeps struct {
	Reviews              repositories.ReviewRepository
	Orders               repositories.OrderRepository
	Clock                func() time.Time
	IDGenerator          func() string
	Sanitizer            func(string) string
	ProfanityChecker     func(string) bool
	Events               ReviewEventPublisher
	AllowedOrderStatuses []domain.OrderStatus
}

type reviewService struct {
	reviews                repositories.ReviewRepository
	orders                 repositories.OrderRepository
	clock                  func() time.Time
	newID                  func() string
	sanitize               func(string) string
	isProfane              func(string) bool
	events                 ReviewEventPublisher
	allowedStatuses        map[domain.ReviewStatus]struct{}
	completedOrderStatuses map[domain.OrderStatus]struct{}
}

// NewReviewService wires dependencies into a concrete ReviewService implementation.
func NewReviewService(deps ReviewServiceDeps) (ReviewService, error) {
	if deps.Reviews == nil {
		return nil, errors.New("review service: review repository is required")
	}
	if deps.Orders == nil {
		return nil, errors.New("review service: order repository is required")
	}

	clock := deps.Clock
	if clock == nil {
		clock = time.Now
	}
	idGen := deps.IDGenerator
	if idGen == nil {
		idGen = func() string {
			return reviewIDPrefix + ulid.Make().String()
		}
	}
	sanitize := deps.Sanitizer
	if sanitize == nil {
		sanitize = sanitizeReviewText
	}
	profanity := deps.ProfanityChecker
	if profanity == nil {
		profanity = basicProfanityChecker
	}

	orderStatuses := deps.AllowedOrderStatuses
	if len(orderStatuses) == 0 {
		orderStatuses = []domain.OrderStatus{
			domain.OrderStatusCompleted,
			domain.OrderStatusDelivered,
		}
	}

	completed := make(map[domain.OrderStatus]struct{}, len(orderStatuses))
	for _, status := range orderStatuses {
		completed[status] = struct{}{}
	}

	return &reviewService{
		reviews: deps.Reviews,
		orders:  deps.Orders,
		clock: func() time.Time {
			return clock().UTC()
		},
		newID:     idGen,
		sanitize:  sanitize,
		isProfane: profanity,
		events:    deps.Events,
		allowedStatuses: map[domain.ReviewStatus]struct{}{
			domain.ReviewStatusApproved: {},
			domain.ReviewStatusRejected: {},
		},
		completedOrderStatuses: completed,
	}, nil
}

func (s *reviewService) Create(ctx context.Context, cmd CreateReviewCommand) (Review, error) {
	if err := s.validateCreateCommand(cmd); err != nil {
		return Review{}, err
	}

	order, err := s.orders.FindByID(ctx, cmd.OrderID)
	if err != nil {
		return Review{}, s.mapOrderError(err)
	}

	if order.UserID != cmd.UserID {
		return Review{}, fmt.Errorf("%w: order does not belong to user", ErrReviewInvalidInput)
	}
	if _, ok := s.completedOrderStatuses[order.Status]; !ok {
		return Review{}, fmt.Errorf("%w: order must be completed before review submission", ErrReviewInvalidInput)
	}

	if err := s.ensureNoExistingReview(ctx, cmd.OrderID); err != nil {
		return Review{}, err
	}

	now := s.now()
	review := domain.Review{
		ID:        s.newID(),
		OrderRef:  cmd.OrderID,
		UserRef:   cmd.UserID,
		Rating:    cmd.Rating,
		Comment:   s.sanitize(cmd.Comment),
		Status:    domain.ReviewStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	created, err := s.reviews.Insert(ctx, review)
	if err != nil {
		return Review{}, s.mapReviewError(err)
	}

	s.emitEvent(ctx, reviewEventCreated, created, cmd.ActorID)

	return created, nil
}

func (s *reviewService) GetByOrder(ctx context.Context, cmd GetReviewByOrderCommand) (Review, error) {
	if err := s.validateGetByOrderCommand(cmd); err != nil {
		return Review{}, err
	}

	review, err := s.reviews.FindByOrder(ctx, cmd.OrderID)
	if err != nil {
		return Review{}, s.mapReviewError(err)
	}
	if !cmd.AllowStaff && review.UserRef != cmd.ActorID {
		return Review{}, ErrReviewUnauthorized
	}
	return review, nil
}

func (s *reviewService) ListByUser(ctx context.Context, cmd ListUserReviewsCommand) (domain.CursorPage[Review], error) {
	if strings.TrimSpace(cmd.UserID) == "" {
		return domain.CursorPage[Review]{}, fmt.Errorf("%w: user id is required", ErrReviewInvalidInput)
	}
	page, err := s.reviews.ListByUser(ctx, cmd.UserID, cmd.Pagination)
	if err != nil {
		return domain.CursorPage[Review]{}, s.mapReviewError(err)
	}
	return domain.CursorPage[Review]{
		Items:         page.Items,
		NextPageToken: page.NextPageToken,
	}, nil
}

func (s *reviewService) Moderate(ctx context.Context, cmd ModerateReviewCommand) (Review, error) {
	if err := s.validateModerationCommand(cmd); err != nil {
		return Review{}, err
	}

	review, err := s.reviews.FindByID(ctx, cmd.ReviewID)
	if err != nil {
		return Review{}, s.mapReviewError(err)
	}

	if review.Status == cmd.Status {
		return review, nil
	}

	if review.Status != domain.ReviewStatusPending {
		return Review{}, fmt.Errorf("%w: cannot transition from %s to %s", ErrReviewInvalidState, review.Status, cmd.Status)
	}

	now := s.now()
	updated, err := s.reviews.UpdateStatus(ctx, cmd.ReviewID, cmd.Status, repositories.ReviewModerationUpdate{
		ModeratedBy: cmd.ActorID,
		ModeratedAt: now,
	})
	if err != nil {
		return Review{}, s.mapReviewError(err)
	}

	switch cmd.Status {
	case domain.ReviewStatusApproved:
		s.emitEvent(ctx, reviewEventApproved, updated, cmd.ActorID)
	case domain.ReviewStatusRejected:
		s.emitEvent(ctx, reviewEventRejected, updated, cmd.ActorID)
	}

	return updated, nil
}

func (s *reviewService) StoreReply(ctx context.Context, cmd StoreReviewReplyCommand) (Review, error) {
	if err := s.validateReplyCommand(cmd); err != nil {
		return Review{}, err
	}

	review, err := s.reviews.FindByID(ctx, cmd.ReviewID)
	if err != nil {
		return Review{}, s.mapReviewError(err)
	}

	if review.Status != domain.ReviewStatusApproved {
		return Review{}, fmt.Errorf("%w: replies allowed only for approved reviews", ErrReviewInvalidState)
	}

	message := s.sanitize(cmd.Message)
	if message != "" && s.isProfane(message) {
		return Review{}, fmt.Errorf("%w: reply contains profanity", ErrReviewInvalidInput)
	}

	updateAt := s.now()

	var reply *domain.ReviewReply
	if message != "" {
		createdAt := updateAt
		if review.Reply != nil && !review.Reply.CreatedAt.IsZero() {
			createdAt = review.Reply.CreatedAt
		}
		reply = &domain.ReviewReply{
			Message:   message,
			AuthorRef: cmd.ActorID,
			Visible:   cmd.Visible,
			CreatedAt: createdAt,
			UpdatedAt: updateAt,
		}
	}

	updated, err := s.reviews.UpdateReply(ctx, cmd.ReviewID, reply, updateAt)
	if err != nil {
		return Review{}, s.mapReviewError(err)
	}

	s.emitEvent(ctx, reviewEventReplyUpdated, updated, cmd.ActorID)

	return updated, nil
}

func (s *reviewService) validateCreateCommand(cmd CreateReviewCommand) error {
	if strings.TrimSpace(cmd.OrderID) == "" {
		return fmt.Errorf("%w: order id is required", ErrReviewInvalidInput)
	}
	if strings.TrimSpace(cmd.UserID) == "" {
		return fmt.Errorf("%w: user id is required", ErrReviewInvalidInput)
	}
	if strings.TrimSpace(cmd.ActorID) == "" {
		return fmt.Errorf("%w: actor id is required", ErrReviewInvalidInput)
	}
	if cmd.Rating < 1 || cmd.Rating > 5 {
		return fmt.Errorf("%w: rating must be between 1 and 5", ErrReviewInvalidInput)
	}

	comment := s.sanitize(cmd.Comment)
	if comment == "" {
		return fmt.Errorf("%w: comment is required", ErrReviewInvalidInput)
	}
	if s.isProfane(comment) {
		return fmt.Errorf("%w: comment contains profanity", ErrReviewInvalidInput)
	}
	return nil
}

func (s *reviewService) validateModerationCommand(cmd ModerateReviewCommand) error {
	if strings.TrimSpace(cmd.ReviewID) == "" {
		return fmt.Errorf("%w: review id is required", ErrReviewInvalidInput)
	}
	if strings.TrimSpace(cmd.ActorID) == "" {
		return fmt.Errorf("%w: actor id is required", ErrReviewInvalidInput)
	}
	if _, ok := s.allowedStatuses[cmd.Status]; !ok {
		return fmt.Errorf("%w: unsupported moderation status %s", ErrReviewInvalidInput, cmd.Status)
	}
	return nil
}

func (s *reviewService) validateReplyCommand(cmd StoreReviewReplyCommand) error {
	if strings.TrimSpace(cmd.ReviewID) == "" {
		return fmt.Errorf("%w: review id is required", ErrReviewInvalidInput)
	}
	if strings.TrimSpace(cmd.ActorID) == "" {
		return fmt.Errorf("%w: actor id is required", ErrReviewInvalidInput)
	}
	return nil
}

func (s *reviewService) validateGetByOrderCommand(cmd GetReviewByOrderCommand) error {
	if strings.TrimSpace(cmd.OrderID) == "" {
		return fmt.Errorf("%w: order id is required", ErrReviewInvalidInput)
	}
	if strings.TrimSpace(cmd.ActorID) == "" {
		return fmt.Errorf("%w: actor id is required", ErrReviewInvalidInput)
	}
	return nil
}

func (s *reviewService) ensureNoExistingReview(ctx context.Context, orderID string) error {
	_, err := s.reviews.FindByOrder(ctx, orderID)
	if err == nil {
		return fmt.Errorf("%w: review already exists for order", ErrReviewConflict)
	}
	var repoErr repositories.RepositoryError
	if errors.As(err, &repoErr) && repoErr.IsNotFound() {
		return nil
	}
	if err != nil {
		return s.mapReviewError(err)
	}
	return nil
}

func (s *reviewService) emitEvent(ctx context.Context, eventType string, review domain.Review, actorID string) {
	if s.events == nil {
		return
	}
	event := ReviewEvent{
		Type:       eventType,
		ReviewID:   review.ID,
		OrderID:    review.OrderRef,
		Status:     review.Status,
		ActorID:    actorID,
		OccurredAt: s.now(),
		Metadata: map[string]any{
			"userRef": review.UserRef,
		},
	}
	_ = s.events.PublishReviewEvent(ctx, event)
}

func (s *reviewService) now() time.Time {
	return s.clock()
}

func (s *reviewService) mapReviewError(err error) error {
	if err == nil {
		return nil
	}
	var repoErr repositories.RepositoryError
	if errors.As(err, &repoErr) {
		switch {
		case repoErr.IsNotFound():
			return ErrReviewNotFound
		case repoErr.IsConflict():
			return ErrReviewConflict
		}
	}
	return err
}

func (s *reviewService) mapOrderError(err error) error {
	if err == nil {
		return nil
	}
	var repoErr repositories.RepositoryError
	if errors.As(err, &repoErr) && repoErr.IsNotFound() {
		return fmt.Errorf("%w: order not found", ErrReviewInvalidInput)
	}
	return err
}

// ReviewEventPublisher emits review lifecycle events to downstream consumers.
type ReviewEventPublisher interface {
	PublishReviewEvent(ctx context.Context, event ReviewEvent) error
}

// ReviewEvent captures metadata for review lifecycle events.
type ReviewEvent struct {
	Type       string
	ReviewID   string
	OrderID    string
	Status     domain.ReviewStatus
	ActorID    string
	OccurredAt time.Time
	Metadata   map[string]any
}

var defaultProfanityTerms = map[string]struct{}{
	"ass":     {},
	"asshole": {},
	"bastard": {},
	"bitch":   {},
	"bloody":  {},
	"damn":    {},
	"fuck":    {},
	"fucker":  {},
	"fucking": {},
	"shit":    {},
	"shitty":  {},
	"slut":    {},
	"whore":   {},
}

func basicProfanityChecker(input string) bool {
	if input == "" {
		return false
	}

	normalized := strings.ToLower(input)
	words := strings.FieldsFunc(normalized, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsNumber(r))
	})

	for _, word := range words {
		if _, ok := defaultProfanityTerms[word]; ok {
			return true
		}
	}
	return false
}

// sanitizeReviewText trims whitespace, strips unsafe control characters, and normalises spacing while
// preserving intentional newlines for readability.
func sanitizeReviewText(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}

	normalized := strings.ReplaceAll(strings.ReplaceAll(trimmed, "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(normalized, "\n")
	for i, line := range lines {
		line = strings.Map(func(r rune) rune {
			if unicode.IsControl(r) && r != '\n' {
				return -1
			}
			return r
		}, line)
		lines[i] = strings.Join(strings.Fields(line), " ")
	}

	result := strings.Join(lines, "\n")
	return strings.TrimSpace(result)
}
