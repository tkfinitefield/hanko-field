package jobs

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/pubsub/pstest"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/hanko-field/api/internal/services"
)

func TestPubSubSuggestionPublisherPublishesMessage(t *testing.T) {
	ctx := context.Background()
	srv := pstest.NewServer()
	defer srv.Close()

	client, err := pubsub.NewClient(ctx, "test-project",
		option.WithEndpoint(srv.Addr),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		t.Fatalf("pubsub.NewClient: %v", err)
	}
	defer func() {
		_ = client.Close()
	}()

	topic, err := client.CreateTopic(ctx, "ai-jobs")
	if err != nil {
		t.Fatalf("CreateTopic: %v", err)
	}

	publisher, err := NewPubSubSuggestionPublisher(topic)
	if err != nil {
		t.Fatalf("NewPubSubSuggestionPublisher: %v", err)
	}

	queuedAt := time.Date(2025, 5, 6, 9, 0, 0, 0, time.UTC)
	msg := services.SuggestionJobMessage{
		JobID:          "aj_test",
		SuggestionID:   "as_test",
		DesignID:       "design-1",
		Method:         "balance",
		Model:          "glyph-balancer@2025-05",
		QueuedAt:       queuedAt,
		IdempotencyKey: "idem-123",
	}

	if _, err := publisher.PublishSuggestionJob(ctx, msg); err != nil {
		t.Fatalf("PublishSuggestionJob: %v", err)
	}

	messages := srv.Messages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	var payload services.SuggestionJobMessage
	if err := json.Unmarshal(messages[0].Data, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.JobID != msg.JobID || payload.SuggestionID != msg.SuggestionID {
		t.Fatalf("unexpected payload %#v", payload)
	}
	if attr := messages[0].Attributes["idempotencyKey"]; attr != "idem-123" {
		t.Fatalf("expected idempotency key attribute, got %q", attr)
	}
	if _, ok := messages[0].Attributes["prompt"]; ok {
		t.Fatalf("prompt attribute should not be present")
	}
}
