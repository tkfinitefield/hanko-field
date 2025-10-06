package secrets

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/googleapis/gax-go/v2"
	"go.uber.org/zap"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestResolveCachesRemoteSecret(t *testing.T) {
	ctx := context.Background()

	client := newFakeSecretClient()
	resource := "projects/test/secrets/stripe_api_key/versions/latest"
	client.values[resource] = "remote-secret"

	fetcher, err := NewFetcher(ctx,
		WithSecretManagerClient(client),
		WithDefaultProject("test"),
		WithLogger(zap.NewNop()),
	)
	if err != nil {
		t.Fatalf("NewFetcher returned error: %v", err)
	}
	defer fetcher.Close()

	got, err := fetcher.Resolve(ctx, "secret://stripe_api_key")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if got != "remote-secret" {
		t.Fatalf("expected remote-secret, got %s", got)
	}

	got, err = fetcher.Resolve(ctx, "secret://stripe_api_key")
	if err != nil {
		t.Fatalf("Resolve second call returned error: %v", err)
	}
	if got != "remote-secret" {
		t.Fatalf("expected cached remote-secret, got %s", got)
	}

	if calls := client.callCount(resource); calls != 1 {
		t.Fatalf("expected remote fetch once, got %d", calls)
	}
}

func TestResolveFallsBackWhenSecretManagerUnavailable(t *testing.T) {
	ctx := context.Background()

	dir := t.TempDir()
	fallbackPath := filepath.Join(dir, ".secrets.local")
	if err := os.WriteFile(fallbackPath, []byte("secret://stripe_api_key=local-secret\n"), 0o600); err != nil {
		t.Fatalf("failed writing fallback file: %v", err)
	}

	client := newFakeSecretClient()
	resource := "projects/test/secrets/stripe_api_key/versions/latest"
	client.errors[resource] = status.Error(codes.PermissionDenied, "denied")

	fetcher, err := NewFetcher(ctx,
		WithSecretManagerClient(client),
		WithDefaultProject("test"),
		WithFallbackFile(fallbackPath),
	)
	if err != nil {
		t.Fatalf("NewFetcher returned error: %v", err)
	}
	defer fetcher.Close()

	got, err := fetcher.Resolve(ctx, "secret://stripe_api_key")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if got != "local-secret" {
		t.Fatalf("expected fallback secret local-secret, got %s", got)
	}
}

func TestInvalidateNotifiesSubscribers(t *testing.T) {
	ctx := context.Background()

	client := newFakeSecretClient()
	resource := "projects/test/secrets/stripe_api_key/versions/latest"
	client.values[resource] = "remote-secret"

	fetcher, err := NewFetcher(ctx,
		WithSecretManagerClient(client),
		WithDefaultProject("test"),
	)
	if err != nil {
		t.Fatalf("NewFetcher error: %v", err)
	}
	defer fetcher.Close()

	if _, err := fetcher.Resolve(ctx, "secret://stripe_api_key"); err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	ch, cancel := fetcher.Subscribe("secret://stripe_api_key")
	defer cancel()

	fetcher.Invalidate("secret://stripe_api_key")

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("expected invalidation notification")
	}
}

func TestResolveUsesVersionPins(t *testing.T) {
	ctx := context.Background()

	client := newFakeSecretClient()
	resourcePinned := "projects/test/secrets/stripe_api_key/versions/5"
	client.values[resourcePinned] = "version-5"

	fetcher, err := NewFetcher(ctx,
		WithSecretManagerClient(client),
		WithDefaultProject("test"),
		WithVersionPins(map[string]string{
			"secret://stripe_api_key": "5",
		}),
	)
	if err != nil {
		t.Fatalf("NewFetcher error: %v", err)
	}
	defer fetcher.Close()

	got, err := fetcher.Resolve(ctx, "secret://stripe_api_key")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if got != "version-5" {
		t.Fatalf("expected version-5, got %s", got)
	}
	if calls := client.callCount(resourcePinned); calls != 1 {
		t.Fatalf("expected fetch of version 5, got %d calls", calls)
	}
}

func TestResolveDoesNotFallbackOnNotFound(t *testing.T) {
	ctx := context.Background()

	dir := t.TempDir()
	fallbackPath := filepath.Join(dir, ".secrets.local")
	if err := os.WriteFile(fallbackPath, []byte("secret://stripe_api_key=local-secret\n"), 0o600); err != nil {
		t.Fatalf("failed writing fallback file: %v", err)
	}

	client := newFakeSecretClient()
	resource := "projects/test/secrets/stripe_api_key/versions/latest"
	client.errors[resource] = status.Error(codes.NotFound, "missing")

	fetcher, err := NewFetcher(ctx,
		WithSecretManagerClient(client),
		WithDefaultProject("test"),
		WithFallbackFile(fallbackPath),
	)
	if err != nil {
		t.Fatalf("NewFetcher returned error: %v", err)
	}
	defer fetcher.Close()

	_, err = fetcher.Resolve(ctx, "secret://stripe_api_key")
	if err == nil {
		t.Fatal("expected error when secret is missing")
	}
}

func TestNewFetcherWithoutCredentialsUsesFallback(t *testing.T) {
	ctx := context.Background()

	originalFactory := secretManagerClientFactory
	secretManagerClientFactory = func(context.Context, ...option.ClientOption) (*secretmanager.Client, error) {
		return nil, errors.New("no credentials")
	}
	t.Cleanup(func() {
		secretManagerClientFactory = originalFactory
	})

	dir := t.TempDir()
	fallbackPath := filepath.Join(dir, ".secrets.local")
	if err := os.WriteFile(fallbackPath, []byte("secret://stripe_api_key=local-secret\n"), 0o600); err != nil {
		t.Fatalf("failed writing fallback file: %v", err)
	}

	fetcher, err := NewFetcher(ctx, WithFallbackFile(fallbackPath))
	if err != nil {
		t.Fatalf("NewFetcher returned error: %v", err)
	}
	defer fetcher.Close()

	value, err := fetcher.Resolve(ctx, "secret://stripe_api_key")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if value != "local-secret" {
		t.Fatalf("expected local secret, got %s", value)
	}
}

type fakeSecretClient struct {
	mu      sync.Mutex
	values  map[string]string
	errors  map[string]error
	counter map[string]int
}

func newFakeSecretClient() *fakeSecretClient {
	return &fakeSecretClient{
		values:  make(map[string]string),
		errors:  make(map[string]error),
		counter: make(map[string]int),
	}
}

func (f *fakeSecretClient) AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, _ ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	name := req.GetName()
	f.counter[name]++

	if err, ok := f.errors[name]; ok && err != nil {
		return nil, err
	}
	if value, ok := f.values[name]; ok {
		return &secretmanagerpb.AccessSecretVersionResponse{
			Payload: &secretmanagerpb.SecretPayload{Data: []byte(value)},
		}, nil
	}
	return nil, status.Error(codes.NotFound, "not found")
}

func (f *fakeSecretClient) Close() error {
	return nil
}

func (f *fakeSecretClient) callCount(name string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.counter[name]
}
