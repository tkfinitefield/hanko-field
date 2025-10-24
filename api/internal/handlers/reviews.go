package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/platform/httpx"
	"github.com/hanko-field/api/internal/repositories"
	"github.com/hanko-field/api/internal/services"
)

const maxReviewBodySize = 32 * 1024

// ReviewHandlers exposes endpoints for creating and retrieving user reviews.
type ReviewHandlers struct {
	authn   *auth.Authenticator
	reviews services.ReviewService
}

// NewReviewHandlers constructs a new ReviewHandlers instance.
func NewReviewHandlers(authn *auth.Authenticator, reviews services.ReviewService) *ReviewHandlers {
	return &ReviewHandlers{
		authn:   authn,
		reviews: reviews,
	}
}

// Routes registers the /reviews endpoints.
func (h *ReviewHandlers) Routes(r chi.Router) {
	if r == nil {
		return
	}
	if h.authn != nil {
		r.Use(h.authn.RequireFirebaseAuth())
	}
	r.Post("/", h.createReview)
}

func (h *ReviewHandlers) createReview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.reviews == nil {
		httpx.WriteError(ctx, w, httpx.NewError("review_service_unavailable", "review service unavailable", http.StatusServiceUnavailable))
		return
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}

	body, err := readLimitedBody(r, maxReviewBodySize)
	if err != nil {
		switch {
		case errors.Is(err, errEmptyBody):
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "request body is required", http.StatusBadRequest))
		case errors.Is(err, errBodyTooLarge):
			httpx.WriteError(ctx, w, httpx.NewError("payload_too_large", "request body exceeds allowed size", http.StatusRequestEntityTooLarge))
		default:
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		}
		return
	}

	var req createReviewRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "invalid JSON payload", http.StatusBadRequest))
		return
	}

	cmd := services.CreateReviewCommand{
		OrderID: strings.TrimSpace(req.OrderID),
		UserID:  strings.TrimSpace(identity.UID),
		Rating:  req.Rating,
		Comment: buildReviewComment(req),
		ActorID: strings.TrimSpace(identity.UID),
	}

	review, err := h.reviews.Create(ctx, cmd)
	if err != nil {
		writeReviewError(ctx, w, err)
		return
	}

	payload := createReviewResponse{
		Review: buildReviewPayload(review),
	}

	writeJSONResponse(w, http.StatusCreated, payload)
}

type createReviewRequest struct {
	OrderID string   `json:"order_id"`
	Rating  int      `json:"rating"`
	Title   string   `json:"title"`
	Body    string   `json:"body"`
	Comment string   `json:"comment"`
	Photos  []string `json:"photos"`
}

type createReviewResponse struct {
	Review reviewPayload `json:"review"`
}

type reviewPayload struct {
	ID          string              `json:"id"`
	OrderID     string              `json:"order_id"`
	UserID      string              `json:"user_id"`
	Rating      int                 `json:"rating"`
	Comment     string              `json:"comment"`
	Status      string              `json:"status"`
	ModeratedBy *string             `json:"moderated_by,omitempty"`
	ModeratedAt string              `json:"moderated_at,omitempty"`
	Reply       *reviewReplyPayload `json:"reply,omitempty"`
	CreatedAt   string              `json:"created_at"`
	UpdatedAt   string              `json:"updated_at"`
}

type reviewReplyPayload struct {
	Message   string `json:"message"`
	AuthorID  string `json:"author_id"`
	Visible   bool   `json:"visible"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func buildReviewComment(req createReviewRequest) string {
	if comment := strings.TrimSpace(req.Comment); comment != "" {
		return comment
	}

	title := strings.TrimSpace(req.Title)
	body := strings.TrimSpace(req.Body)

	parts := make([]string, 0, 2)
	if title != "" {
		parts = append(parts, title)
	}
	if body != "" {
		parts = append(parts, body)
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func buildReviewPayload(review services.Review) reviewPayload {
	payload := reviewPayload{
		ID:          review.ID,
		OrderID:     review.OrderRef,
		UserID:      review.UserRef,
		Rating:      review.Rating,
		Comment:     review.Comment,
		Status:      string(review.Status),
		ModeratedBy: cloneStringPointer(review.ModeratedBy),
		ModeratedAt: formatTime(pointerTime(review.ModeratedAt)),
		CreatedAt:   formatTime(review.CreatedAt),
		UpdatedAt:   formatTime(review.UpdatedAt),
	}

	if review.Reply != nil {
		payload.Reply = &reviewReplyPayload{
			Message:   review.Reply.Message,
			AuthorID:  review.Reply.AuthorRef,
			Visible:   review.Reply.Visible,
			CreatedAt: formatTime(review.Reply.CreatedAt),
			UpdatedAt: formatTime(review.Reply.UpdatedAt),
		}
	}

	return payload
}

func writeReviewError(ctx context.Context, w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	switch {
	case errors.Is(err, services.ErrReviewInvalidInput):
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
	case errors.Is(err, services.ErrReviewUnauthorized):
		httpx.WriteError(ctx, w, httpx.NewError("forbidden", "insufficient permissions for review", http.StatusForbidden))
	case errors.Is(err, services.ErrReviewConflict):
		httpx.WriteError(ctx, w, httpx.NewError("review_conflict", err.Error(), http.StatusConflict))
	case errors.Is(err, services.ErrReviewNotFound):
		httpx.WriteError(ctx, w, httpx.NewError("review_not_found", "review not found", http.StatusNotFound))
	default:
		var repoErr repositories.RepositoryError
		if errors.As(err, &repoErr) && repoErr.IsUnavailable() {
			httpx.WriteError(ctx, w, httpx.NewError("review_service_unavailable", "review repository unavailable", http.StatusServiceUnavailable))
			return
		}
		httpx.WriteError(ctx, w, httpx.NewError("review_error", "failed to process review request", http.StatusInternalServerError))
	}
}
