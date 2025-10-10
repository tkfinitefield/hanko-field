package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/pubsub"

	"github.com/hanko-field/api/internal/services"
)

// PubSubSuggestionPublisher publishes AI suggestion jobs to a Pub/Sub topic.
type PubSubSuggestionPublisher struct {
	topic   *pubsub.Topic
	marshal func(any) ([]byte, error)
}

// NewPubSubSuggestionPublisher constructs a Pub/Sub backed suggestion job publisher.
func NewPubSubSuggestionPublisher(topic *pubsub.Topic) (*PubSubSuggestionPublisher, error) {
	if topic == nil {
		return nil, errors.New("pubsub suggestion publisher: topic is required")
	}
	return &PubSubSuggestionPublisher{
		topic:   topic,
		marshal: json.Marshal,
	}, nil
}

// PublishSuggestionJob enqueues a suggestion job message on the configured topic.
func (p *PubSubSuggestionPublisher) PublishSuggestionJob(ctx context.Context, message services.SuggestionJobMessage) (string, error) {
	if p == nil || p.topic == nil {
		return "", errors.New("pubsub suggestion publisher: not initialised")
	}

	data, err := p.marshal(message)
	if err != nil {
		return "", fmt.Errorf("marshal suggestion job: %w", err)
	}

	attrs := make(map[string]string)
	setAttr(attrs, "jobId", message.JobID)
	setAttr(attrs, "suggestionId", message.SuggestionID)
	setAttr(attrs, "designId", message.DesignID)
	setAttr(attrs, "method", message.Method)
	setAttr(attrs, "model", message.Model)
	if key := strings.TrimSpace(message.IdempotencyKey); key != "" {
		attrs["idempotencyKey"] = key
	}

	result := p.topic.Publish(ctx, &pubsub.Message{
		Data:       data,
		Attributes: attrs,
	})

	id, err := result.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("publish suggestion job: %w", err)
	}
	return id, nil
}

func setAttr(attrs map[string]string, key string, value string) {
	if v := strings.TrimSpace(value); v != "" {
		attrs[key] = v
	}
}
