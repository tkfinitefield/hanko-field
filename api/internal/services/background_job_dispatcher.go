package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

const (
	defaultSuggestionPriority = 50
	jobEventQueued            = "ai.job.queued"
	jobEventCompleted         = "ai.job.completed"
	jobEventFailed            = "ai.job.failed"
)

var (
	// ErrAIInvalidInput indicates required fields were missing from the command.
	ErrAIInvalidInput = errors.New("ai: invalid input")
	// ErrAIJobNotFound indicates the requested AI job could not be located.
	ErrAIJobNotFound = errors.New("ai: job not found")
	// ErrAISuggestionNotFound indicates the requested AI suggestion does not exist.
	ErrAISuggestionNotFound = errors.New("ai: suggestion not found")
)

// SuggestionJobPublisher publishes suggestion job messages to the background queue.
type SuggestionJobPublisher interface {
	PublishSuggestionJob(ctx context.Context, message SuggestionJobMessage) (string, error)
}

// SuggestionJobMessage is the payload delivered to background workers via Pub/Sub.
type SuggestionJobMessage struct {
	JobID          string    `json:"jobId"`
	SuggestionID   string    `json:"suggestionId"`
	DesignID       string    `json:"designId"`
	Method         string    `json:"method"`
	Model          string    `json:"model"`
	QueuedAt       time.Time `json:"queuedAt"`
	IdempotencyKey string    `json:"idempotencyKey,omitempty"`
}

// BackgroundJobDispatcherDeps enumerates collaborators required to construct the dispatcher.
type BackgroundJobDispatcherDeps struct {
	Jobs        repositories.AIJobRepository
	Suggestions repositories.AISuggestionRepository
	Publisher   SuggestionJobPublisher
	Clock       func() time.Time
	IDGenerator func() string
	Logger      func(ctx context.Context, event string, fields map[string]any)
}

type backgroundJobDispatcher struct {
	jobs        repositories.AIJobRepository
	suggestions repositories.AISuggestionRepository
	publisher   SuggestionJobPublisher
	clock       func() time.Time
	newID       func() string
	logger      func(context.Context, string, map[string]any)
}

// NewBackgroundJobDispatcher wires dependencies into a BackgroundJobDispatcher implementation.
func NewBackgroundJobDispatcher(deps BackgroundJobDispatcherDeps) (BackgroundJobDispatcher, error) {
	if deps.Jobs == nil {
		return nil, errors.New("background job dispatcher: job repository is required")
	}
	if deps.Suggestions == nil {
		return nil, errors.New("background job dispatcher: suggestion repository is required")
	}
	if deps.Publisher == nil {
		return nil, errors.New("background job dispatcher: publisher is required")
	}

	clock := deps.Clock
	if clock == nil {
		clock = time.Now
	}
	idGen := deps.IDGenerator
	if idGen == nil {
		idGen = func() string {
			return ulid.Make().String()
		}
	}
	logger := deps.Logger
	if logger == nil {
		logger = func(context.Context, string, map[string]any) {}
	}

	return &backgroundJobDispatcher{
		jobs:        deps.Jobs,
		suggestions: deps.Suggestions,
		publisher:   deps.Publisher,
		clock: func() time.Time {
			return clock().UTC()
		},
		newID:  idGen,
		logger: logger,
	}, nil
}

func (d *backgroundJobDispatcher) QueueAISuggestion(ctx context.Context, cmd QueueAISuggestionCommand) (QueueAISuggestionResult, error) {
	if err := d.validateQueueCommand(cmd); err != nil {
		return QueueAISuggestionResult{}, err
	}

	now := d.now()
	jobID := ensureJobID(d.newID())
	suggestionID := ensureSuggestionID(cmd.SuggestionID)

	payload := d.buildJobPayload(cmd, suggestionID, now)
	if key := strings.TrimSpace(cmd.IdempotencyKey); key != "" {
		if existing, err := d.jobs.FindByIdempotencyKey(ctx, key); err == nil && existing.ID != "" {
			return QueueAISuggestionResult{
				JobID:        existing.ID,
				SuggestionID: extractSuggestionID(existing.Payload),
				Status:       existing.Status,
				QueuedAt:     existing.CreatedAt,
			}, nil
		} else if err != nil {
			if !isRepoNotFound(err) {
				return QueueAISuggestionResult{}, err
			}
		}
	}

	job := domain.AIJob{
		ID:       jobID,
		Kind:     domain.AIJobKindDesignSuggestion,
		Status:   domain.AIJobStatusQueued,
		Priority: determinePriority(cmd.Priority),
		Payload:  payload,
		Attempt: domain.AIJobAttempt{
			Count: 0,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	inserted, err := d.jobs.Insert(ctx, job)
	if err != nil {
		return QueueAISuggestionResult{}, err
	}

	msg := SuggestionJobMessage{
		JobID:          inserted.ID,
		SuggestionID:   suggestionID,
		DesignID:       strings.TrimSpace(cmd.DesignID),
		Method:         strings.TrimSpace(cmd.Method),
		Model:          strings.TrimSpace(cmd.Model),
		QueuedAt:       inserted.CreatedAt,
		IdempotencyKey: strings.TrimSpace(cmd.IdempotencyKey),
	}

	if _, err := d.publisher.PublishSuggestionJob(ctx, msg); err != nil {
		updateErr := d.failJob(ctx, inserted.ID, err, now, payload)
		if updateErr != nil {
			d.logFailure(ctx, "publish_failure_update", map[string]any{
				"jobId": inserted.ID,
				"error": updateErr.Error(),
			})
		}
		return QueueAISuggestionResult{}, fmt.Errorf("publish suggestion job: %w", err)
	}

	d.logEvent(ctx, jobEventQueued, map[string]any{
		"jobId":        inserted.ID,
		"suggestionId": suggestionID,
		"designId":     cmd.DesignID,
		"method":       cmd.Method,
	})

	return QueueAISuggestionResult{
		JobID:        inserted.ID,
		SuggestionID: suggestionID,
		Status:       inserted.Status,
		QueuedAt:     inserted.CreatedAt,
	}, nil
}

func (d *backgroundJobDispatcher) GetAIJob(ctx context.Context, jobID string) (domain.AIJob, error) {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return domain.AIJob{}, fmt.Errorf("%w: job id is required", ErrAIInvalidInput)
	}

	job, err := d.jobs.FindByID(ctx, jobID)
	if err != nil {
		if isRepoNotFound(err) {
			return domain.AIJob{}, ErrAIJobNotFound
		}
		return domain.AIJob{}, err
	}
	return job, nil
}

func (d *backgroundJobDispatcher) CompleteAISuggestion(ctx context.Context, cmd CompleteAISuggestionCommand) (CompleteAISuggestionResult, error) {
	jobID := strings.TrimSpace(cmd.JobID)
	if jobID == "" {
		return CompleteAISuggestionResult{}, fmt.Errorf("%w: job id is required", ErrAIInvalidInput)
	}

	job, err := d.jobs.FindByID(ctx, jobID)
	if err != nil {
		if isRepoNotFound(err) {
			return CompleteAISuggestionResult{}, ErrAIJobNotFound
		}
		return CompleteAISuggestionResult{}, err
	}

	now := d.now()
	payload := mergePayload(job.Payload, cmd.Outputs, cmd.Metadata)

	if cmd.Error != nil {
		update := repositories.AIJobStatusUpdate{
			Error:       cmd.Error,
			Payload:     payload,
			CompletedAt: &now,
			Metadata:    copyMap(cmd.Metadata),
		}
		updated, err := d.jobs.UpdateStatus(ctx, job.ID, domain.AIJobStatusFailed, update)
		if err != nil {
			return CompleteAISuggestionResult{}, err
		}
		d.logEvent(ctx, jobEventFailed, map[string]any{
			"jobId": job.ID,
			"code":  cmd.Error.Code,
		})
		return CompleteAISuggestionResult{Job: updated}, nil
	}

	suggestion := cmd.Suggestion
	if strings.TrimSpace(suggestion.ID) == "" {
		suggestion.ID = extractSuggestionID(payload)
	}
	suggestion.DesignID = ensureDesignID(suggestion.DesignID, payload)
	suggestion.Method = strings.TrimSpace(suggestion.Method)
	if suggestion.Method == "" {
		if method, ok := payload["method"].(string); ok {
			suggestion.Method = strings.TrimSpace(method)
		}
	}
	if suggestion.CreatedAt.IsZero() {
		suggestion.CreatedAt = now
	}
	if suggestion.UpdatedAt.IsZero() {
		suggestion.UpdatedAt = now
	}
	basePayload := copyMap(job.Payload)
	if suggestion.Payload == nil {
		suggestion.Payload = basePayload
	} else if len(basePayload) > 0 {
		for k, v := range basePayload {
			if _, exists := suggestion.Payload[k]; !exists {
				suggestion.Payload[k] = v
			}
		}
	}
	if suggestion.Payload == nil {
		suggestion.Payload = make(map[string]any)
	}
	suggestion.Payload = mergePayload(suggestion.Payload, cmd.Outputs, cmd.Metadata)
	suggestion.Status = ensureSuggestionStatus(suggestion.Status)

	stored, err := d.suggestions.UpdateStatus(ctx, suggestion.DesignID, suggestion.ID, suggestion.Status, copyMap(suggestion.Payload))
	if err != nil {
		if isRepoNotFound(err) {
			if err := d.suggestions.Insert(ctx, suggestion); err != nil {
				return CompleteAISuggestionResult{}, err
			}
		} else {
			return CompleteAISuggestionResult{}, err
		}
	} else {
		suggestion = stored
	}

	resultRef := fmt.Sprintf("/designs/%s/aiSuggestions/%s", suggestion.DesignID, suggestion.ID)
	update := repositories.AIJobStatusUpdate{
		ResultRef:   &resultRef,
		Payload:     payload,
		CompletedAt: &now,
		Metadata:    copyMap(cmd.Metadata),
	}
	updated, err := d.jobs.UpdateStatus(ctx, job.ID, domain.AIJobStatusSucceeded, update)
	if err != nil {
		return CompleteAISuggestionResult{}, err
	}

	d.logEvent(ctx, jobEventCompleted, map[string]any{
		"jobId":        updated.ID,
		"suggestionId": suggestion.ID,
	})

	return CompleteAISuggestionResult{
		Job:        updated,
		Suggestion: &suggestion,
	}, nil
}

func (d *backgroundJobDispatcher) GetSuggestion(ctx context.Context, designID string, suggestionID string) (AISuggestion, error) {
	designID = strings.TrimSpace(designID)
	suggestionID = strings.TrimSpace(suggestionID)
	if designID == "" || suggestionID == "" {
		return AISuggestion{}, fmt.Errorf("%w: design id and suggestion id are required", ErrAIInvalidInput)
	}

	suggestion, err := d.suggestions.FindByID(ctx, designID, suggestionID)
	if err != nil {
		if isRepoNotFound(err) {
			return AISuggestion{}, ErrAISuggestionNotFound
		}
		return AISuggestion{}, err
	}
	return suggestion, nil
}

func (d *backgroundJobDispatcher) EnqueueRegistrabilityCheck(context.Context, RegistrabilityJobPayload) (string, error) {
	return "", errors.New("registrability job dispatch: not implemented")
}

func (d *backgroundJobDispatcher) EnqueueStockCleanup(context.Context, StockCleanupPayload) error {
	return errors.New("stock cleanup dispatch: not implemented")
}

func (d *backgroundJobDispatcher) validateQueueCommand(cmd QueueAISuggestionCommand) error {
	if strings.TrimSpace(cmd.DesignID) == "" {
		return fmt.Errorf("%w: design id is required", ErrAIInvalidInput)
	}
	if strings.TrimSpace(cmd.Method) == "" {
		return fmt.Errorf("%w: method is required", ErrAIInvalidInput)
	}
	if strings.TrimSpace(cmd.Model) == "" {
		return fmt.Errorf("%w: model is required", ErrAIInvalidInput)
	}
	if len(cmd.Snapshot) == 0 {
		return fmt.Errorf("%w: snapshot is required", ErrAIInvalidInput)
	}
	return nil
}

func (d *backgroundJobDispatcher) buildJobPayload(cmd QueueAISuggestionCommand, suggestionID string, now time.Time) map[string]any {
	payload := copyMap(cmd.Metadata)
	if payload == nil {
		payload = make(map[string]any)
	}
	payload["designId"] = strings.TrimSpace(cmd.DesignID)
	payload["method"] = strings.TrimSpace(cmd.Method)
	payload["model"] = strings.TrimSpace(cmd.Model)
	if prompt := strings.TrimSpace(cmd.Prompt); prompt != "" {
		payload["prompt"] = prompt
	}
	if len(cmd.Parameters) > 0 {
		payload["parameters"] = copyMap(cmd.Parameters)
	}
	if len(cmd.Snapshot) > 0 {
		snapshotCopy := make(map[string]any, len(cmd.Snapshot))
		for k, v := range cmd.Snapshot {
			snapshotCopy[k] = v
		}
		payload["snapshot"] = snapshotCopy
	}
	payload["suggestionId"] = suggestionID
	payload["queuedAt"] = now.Format(time.RFC3339Nano)
	if key := strings.TrimSpace(cmd.IdempotencyKey); key != "" {
		payload["idempotencyKey"] = key
	}
	if requestedBy := strings.TrimSpace(cmd.RequestedBy); requestedBy != "" {
		payload["requestedBy"] = requestedBy
	}
	return payload
}

func (d *backgroundJobDispatcher) failJob(ctx context.Context, jobID string, publishErr error, now time.Time, payload map[string]any) error {
	errDetails := &domain.AIJobError{
		Code:      "publish_error",
		Message:   publishErr.Error(),
		Retryable: true,
	}
	update := repositories.AIJobStatusUpdate{
		Error:       errDetails,
		Payload:     payload,
		CompletedAt: &now,
	}
	_, err := d.jobs.UpdateStatus(ctx, jobID, domain.AIJobStatusFailed, update)
	return err
}

func (d *backgroundJobDispatcher) logFailure(ctx context.Context, event string, fields map[string]any) {
	if d.logger != nil {
		d.logger(ctx, event, fields)
	}
}

func (d *backgroundJobDispatcher) logEvent(ctx context.Context, event string, fields map[string]any) {
	if d.logger != nil {
		d.logger(ctx, event, fields)
	}
}

func (d *backgroundJobDispatcher) now() time.Time {
	return d.clock()
}

func determinePriority(priority int) int {
	if priority <= 0 {
		return defaultSuggestionPriority
	}
	if priority > 100 {
		return 100
	}
	return priority
}

func ensureJobID(candidate string) string {
	trimmed := strings.TrimSpace(candidate)
	if trimmed == "" {
		trimmed = ulid.Make().String()
	}
	if strings.HasPrefix(trimmed, "aj_") {
		return trimmed
	}
	return "aj_" + trimmed
}

func ensureSuggestionID(candidate string) string {
	trimmed := strings.TrimSpace(candidate)
	if trimmed == "" {
		trimmed = ulid.Make().String()
	}
	if strings.HasPrefix(trimmed, "as_") {
		return trimmed
	}
	return "as_" + trimmed
}

func ensureDesignID(current string, payload map[string]any) string {
	current = strings.TrimSpace(current)
	if current != "" {
		return current
	}
	if payload != nil {
		if designID, ok := payload["designId"].(string); ok {
			return strings.TrimSpace(designID)
		}
	}
	return ""
}

func extractSuggestionID(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	if id, ok := payload["suggestionId"].(string); ok {
		return strings.TrimSpace(id)
	}
	return ""
}

func ensureSuggestionStatus(status string) string {
	trimmed := strings.TrimSpace(status)
	if trimmed == "" {
		return "proposed"
	}
	return trimmed
}

func mergePayload(base map[string]any, outputs map[string]any, metadata map[string]any) map[string]any {
	result := copyMap(base)
	if result == nil {
		result = make(map[string]any)
	}
	if len(outputs) > 0 {
		outputCopy := make(map[string]any, len(outputs))
		for k, v := range outputs {
			outputCopy[k] = v
		}
		result["result"] = outputCopy
	}
	if len(metadata) > 0 {
		metaCopy := copyMap(metadata)
		if len(metaCopy) > 0 {
			result["metadata"] = metaCopy
		}
	}
	return result
}

func copyMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func isRepoNotFound(err error) bool {
	var repoErr repositories.RepositoryError
	if errors.As(err, &repoErr) {
		return repoErr.IsNotFound()
	}
	return false
}
