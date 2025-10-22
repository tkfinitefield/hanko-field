package services

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

type captureSuggestionPublisher struct {
	mu       sync.Mutex
	messages []SuggestionJobMessage
}

func (c *captureSuggestionPublisher) PublishSuggestionJob(ctx context.Context, msg SuggestionJobMessage) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = append(c.messages, msg)
	return "pub-1", nil
}

func (c *captureSuggestionPublisher) LastMessage() (SuggestionJobMessage, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.messages) == 0 {
		return SuggestionJobMessage{}, false
	}
	return c.messages[len(c.messages)-1], true
}

type inMemoryAIJobRepo struct {
	mu   sync.Mutex
	jobs map[string]domain.AIJob
}

func newInMemoryAIJobRepo() *inMemoryAIJobRepo {
	return &inMemoryAIJobRepo{
		jobs: make(map[string]domain.AIJob),
	}
}

func (r *inMemoryAIJobRepo) Insert(_ context.Context, job domain.AIJob) (domain.AIJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := cloneJob(job)
	r.jobs[job.ID] = copy
	return cloneJob(copy), nil
}

func (r *inMemoryAIJobRepo) FindByID(_ context.Context, jobID string) (domain.AIJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if job, ok := r.jobs[jobID]; ok {
		return cloneJob(job), nil
	}
	return domain.AIJob{}, &jobRepoErr{notFound: true, msg: "job not found"}
}

func (r *inMemoryAIJobRepo) FindByIdempotencyKey(_ context.Context, key string) (domain.AIJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, job := range r.jobs {
		if job.Payload != nil {
			if k, ok := job.Payload["idempotencyKey"].(string); ok && k == key {
				return cloneJob(job), nil
			}
		}
	}
	return domain.AIJob{}, &jobRepoErr{notFound: true, msg: "job not found"}
}

func (r *inMemoryAIJobRepo) UpdateStatus(_ context.Context, jobID string, status domain.AIJobStatus, update repositories.AIJobStatusUpdate) (domain.AIJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	job, ok := r.jobs[jobID]
	if !ok {
		return domain.AIJob{}, &jobRepoErr{notFound: true, msg: "job not found"}
	}
	job.Status = status
	if update.Payload != nil {
		job.Payload = cloneMapAny(update.Payload)
	}
	if update.ResultRef != nil {
		job.ResultRef = update.ResultRef
	}
	if update.Error != nil {
		errCopy := *update.Error
		job.Error = &errCopy
	}
	if update.CompletedAt != nil {
		job.CompletedAt = update.CompletedAt
		job.UpdatedAt = *update.CompletedAt
	} else {
		job.UpdatedAt = time.Now().UTC()
	}
	r.jobs[jobID] = job
	return cloneJob(job), nil
}

type inMemorySuggestionRepo struct {
	mu          sync.Mutex
	suggestions map[string]map[string]domain.AISuggestion
}

func newInMemorySuggestionRepo() *inMemorySuggestionRepo {
	return &inMemorySuggestionRepo{
		suggestions: make(map[string]map[string]domain.AISuggestion),
	}
}

func (r *inMemorySuggestionRepo) Insert(_ context.Context, suggestion domain.AISuggestion) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.suggestions[suggestion.DesignID]; !ok {
		r.suggestions[suggestion.DesignID] = make(map[string]domain.AISuggestion)
	}
	r.suggestions[suggestion.DesignID][suggestion.ID] = cloneSuggestion(suggestion)
	return nil
}

func (r *inMemorySuggestionRepo) FindByID(_ context.Context, designID string, suggestionID string) (domain.AISuggestion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if suggestions, ok := r.suggestions[designID]; ok {
		if suggestion, ok := suggestions[suggestionID]; ok {
			return cloneSuggestion(suggestion), nil
		}
	}
	return domain.AISuggestion{}, &jobRepoErr{notFound: true, msg: "suggestion not found"}
}

func (r *inMemorySuggestionRepo) UpdateStatus(_ context.Context, designID string, suggestionID string, status string, metadata map[string]any) (domain.AISuggestion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if suggestions, ok := r.suggestions[designID]; ok {
		if suggestion, ok := suggestions[suggestionID]; ok {
			suggestion.Status = status
			if metadata != nil {
				if suggestion.Payload == nil {
					suggestion.Payload = make(map[string]any)
				}
				suggestion.Payload["metadata"] = cloneMapAny(metadata)
			}
			suggestions[suggestionID] = suggestion
			return cloneSuggestion(suggestion), nil
		}
	}
	return domain.AISuggestion{}, &jobRepoErr{notFound: true, msg: "suggestion not found"}
}

func (r *inMemorySuggestionRepo) ListByDesign(_ context.Context, designID string, filter repositories.AISuggestionListFilter) (domain.CursorPage[domain.AISuggestion], error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	page := domain.CursorPage[domain.AISuggestion]{}
	var allowed map[string]struct{}
	if len(filter.Status) > 0 {
		allowed = make(map[string]struct{}, len(filter.Status))
		for _, status := range filter.Status {
			if trimmed := strings.ToLower(strings.TrimSpace(status)); trimmed != "" {
				allowed[trimmed] = struct{}{}
			}
		}
	}
	if suggestions, ok := r.suggestions[designID]; ok {
		for _, suggestion := range suggestions {
			if len(allowed) > 0 {
				current := strings.ToLower(strings.TrimSpace(suggestion.Status))
				if _, ok := allowed[current]; !ok {
					continue
				}
			}
			page.Items = append(page.Items, cloneSuggestion(suggestion))
		}
	}
	return page, nil
}

type jobRepoErr struct {
	notFound    bool
	unavailable bool
	msg         string
}

func (e *jobRepoErr) Error() string {
	if e.msg != "" {
		return e.msg
	}
	return "repository error"
}

func (e *jobRepoErr) IsNotFound() bool    { return e.notFound }
func (e *jobRepoErr) IsConflict() bool    { return false }
func (e *jobRepoErr) IsUnavailable() bool { return e.unavailable }

func TestBackgroundJobDispatcherQueueCreatesJobAndPublishes(t *testing.T) {
	ctx := context.Background()
	jobRepo := newInMemoryAIJobRepo()
	suggestionRepo := newInMemorySuggestionRepo()
	publisher := &captureSuggestionPublisher{}

	dispatcher, err := NewBackgroundJobDispatcher(BackgroundJobDispatcherDeps{
		Jobs:        jobRepo,
		Suggestions: suggestionRepo,
		Publisher:   publisher,
		Clock: func() time.Time {
			return time.Date(2025, 5, 6, 9, 0, 0, 0, time.UTC)
		},
		IDGenerator: func() string { return "ABC123456789" },
	})
	if err != nil {
		t.Fatalf("NewBackgroundJobDispatcher: %v", err)
	}

	result, err := dispatcher.QueueAISuggestion(ctx, QueueAISuggestionCommand{
		DesignID:       "design-1",
		Method:         "balance",
		Model:          "glyph-balancer@2025-05",
		Prompt:         "Balance glyphs",
		Snapshot:       map[string]any{"version": 3},
		Parameters:     map[string]any{"strength": 0.8},
		Metadata:       map[string]any{"requestedBy": "user-1"},
		IdempotencyKey: "idem-123",
		RequestedBy:    "user-1",
		Priority:       10,
	})
	if err != nil {
		t.Fatalf("QueueAISuggestion: %v", err)
	}
	if result.JobID == "" || result.SuggestionID == "" {
		t.Fatalf("expected job and suggestion IDs, got %+v", result)
	}
	if result.Status != domain.AIJobStatusQueued {
		t.Fatalf("expected status queued, got %s", result.Status)
	}

	job, err := jobRepo.FindByID(ctx, result.JobID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if job.Status != domain.AIJobStatusQueued {
		t.Fatalf("expected job queued, got %s", job.Status)
	}
	if job.Payload == nil {
		t.Fatalf("expected payload stored")
	}
	if idKey, ok := job.Payload["idempotencyKey"].(string); !ok || idKey != "idem-123" {
		t.Fatalf("expected idempotency key stored, got %v", job.Payload["idempotencyKey"])
	}

	msg, ok := publisher.LastMessage()
	if !ok {
		t.Fatalf("expected published message")
	}
	if msg.JobID != result.JobID {
		t.Fatalf("expected message job ID %s, got %s", result.JobID, msg.JobID)
	}
	if msg.SuggestionID != result.SuggestionID {
		t.Fatalf("expected message suggestion ID %s, got %s", result.SuggestionID, msg.SuggestionID)
	}
	if msg.IdempotencyKey != "idem-123" {
		t.Fatalf("expected idempotency key propagated, got %s", msg.IdempotencyKey)
	}
}

func TestBackgroundJobDispatcherQueueIsIdempotent(t *testing.T) {
	ctx := context.Background()
	jobRepo := newInMemoryAIJobRepo()
	suggestionRepo := newInMemorySuggestionRepo()
	publisher := &captureSuggestionPublisher{}

	jobRepo.Insert(ctx, domain.AIJob{
		ID:        "aj_existing",
		Status:    domain.AIJobStatusQueued,
		Kind:      domain.AIJobKindDesignSuggestion,
		Payload:   map[string]any{"idempotencyKey": "idem-dup", "suggestionId": "as_existing"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	})

	dispatcher, err := NewBackgroundJobDispatcher(BackgroundJobDispatcherDeps{
		Jobs:        jobRepo,
		Suggestions: suggestionRepo,
		Publisher:   publisher,
	})
	if err != nil {
		t.Fatalf("NewBackgroundJobDispatcher: %v", err)
	}

	result, err := dispatcher.QueueAISuggestion(ctx, QueueAISuggestionCommand{
		DesignID:       "design-1",
		Method:         "balance",
		Model:          "glyph-balancer@2025-05",
		Snapshot:       map[string]any{"version": 3},
		IdempotencyKey: "idem-dup",
	})
	if err != nil {
		t.Fatalf("QueueAISuggestion: %v", err)
	}
	if result.JobID != "aj_existing" {
		t.Fatalf("expected existing job returned, got %s", result.JobID)
	}
	if len(publisher.messages) != 0 {
		t.Fatalf("expected no new publish, got %d", len(publisher.messages))
	}
}

func TestBackgroundJobDispatcherCompleteSuccessPersistsSuggestion(t *testing.T) {
	ctx := context.Background()
	jobRepo := newInMemoryAIJobRepo()
	suggestionRepo := newInMemorySuggestionRepo()
	publisher := &captureSuggestionPublisher{}

	now := time.Date(2025, 5, 6, 9, 0, 0, 0, time.UTC)
	jobRepo.Insert(ctx, domain.AIJob{
		ID:        "aj_job",
		Status:    domain.AIJobStatusQueued,
		Kind:      domain.AIJobKindDesignSuggestion,
		Payload:   map[string]any{"designId": "design-1", "suggestionId": "as_job", "method": "balance"},
		CreatedAt: now,
		UpdatedAt: now,
	})

	dispatcher, err := NewBackgroundJobDispatcher(BackgroundJobDispatcherDeps{
		Jobs:        jobRepo,
		Suggestions: suggestionRepo,
		Publisher:   publisher,
		Clock:       func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewBackgroundJobDispatcher: %v", err)
	}

	result, err := dispatcher.CompleteAISuggestion(ctx, CompleteAISuggestionCommand{
		JobID: "aj_job",
		Suggestion: AISuggestion{
			ID:       "as_job",
			DesignID: "design-1",
			Method:   "balance",
			Status:   "proposed",
			Payload:  map[string]any{"scores": map[string]any{"quality": 0.9}},
		},
		Outputs: map[string]any{
			"score": 0.92,
			"preview": map[string]any{
				"url": "https://example.com/preview.png",
			},
		},
		Metadata: map[string]any{"worker": "ai-worker-1"},
	})
	if err != nil {
		t.Fatalf("CompleteAISuggestion: %v", err)
	}
	if result.Suggestion == nil {
		t.Fatalf("expected suggestion in result")
	}
	if result.Job.Status != domain.AIJobStatusSucceeded {
		t.Fatalf("expected job status succeeded, got %s", result.Job.Status)
	}

	suggestion, err := suggestionRepo.FindByID(ctx, "design-1", "as_job")
	if err != nil {
		t.Fatalf("FindByID suggestion: %v", err)
	}
	if suggestion.Payload == nil {
		t.Fatalf("expected suggestion payload")
	}
	if score, ok := suggestion.Payload["result"].(map[string]any)["score"].(float64); !ok || score != 0.92 {
		t.Fatalf("expected merged score in payload, got %v", suggestion.Payload["result"])
	}
}

func TestBackgroundJobDispatcherCompleteFailureUpdatesJob(t *testing.T) {
	ctx := context.Background()
	jobRepo := newInMemoryAIJobRepo()
	suggestionRepo := newInMemorySuggestionRepo()
	publisher := &captureSuggestionPublisher{}

	jobRepo.Insert(ctx, domain.AIJob{
		ID:        "aj_fail",
		Status:    domain.AIJobStatusQueued,
		Kind:      domain.AIJobKindDesignSuggestion,
		Payload:   map[string]any{"designId": "design-1"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	})

	dispatcher, err := NewBackgroundJobDispatcher(BackgroundJobDispatcherDeps{
		Jobs:        jobRepo,
		Suggestions: suggestionRepo,
		Publisher:   publisher,
	})
	if err != nil {
		t.Fatalf("NewBackgroundJobDispatcher: %v", err)
	}

	result, err := dispatcher.CompleteAISuggestion(ctx, CompleteAISuggestionCommand{
		JobID: "aj_fail",
		Error: &domain.AIJobError{
			Code:      "worker_timeout",
			Message:   "Timed out",
			Retryable: true,
		},
	})
	if err != nil {
		t.Fatalf("CompleteAISuggestion failure: %v", err)
	}
	if result.Suggestion != nil {
		t.Fatalf("expected no suggestion on failure")
	}
	job, err := jobRepo.FindByID(ctx, "aj_fail")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if job.Status != domain.AIJobStatusFailed {
		t.Fatalf("expected failed status, got %s", job.Status)
	}
	if job.Error == nil || job.Error.Code != "worker_timeout" {
		t.Fatalf("expected job error recorded, got %+v", job.Error)
	}
}

func cloneJob(job domain.AIJob) domain.AIJob {
	clone := job
	if job.Payload != nil {
		clone.Payload = cloneMapAny(job.Payload)
	}
	if job.Error != nil {
		errCopy := *job.Error
		clone.Error = &errCopy
	}
	return clone
}

func cloneSuggestion(s domain.AISuggestion) domain.AISuggestion {
	clone := s
	if s.Payload != nil {
		clone.Payload = cloneMapAny(s.Payload)
	}
	if s.ExpiresAt != nil {
		ts := *s.ExpiresAt
		clone.ExpiresAt = &ts
	}
	return clone
}

func cloneMapAny(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		switch typed := v.(type) {
		case map[string]any:
			out[k] = cloneMapAny(typed)
		default:
			out[k] = typed
		}
	}
	return out
}
