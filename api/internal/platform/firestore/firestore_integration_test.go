//go:build integration

package firestore_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	pconfig "github.com/hanko-field/api/internal/platform/config"
	pfirestore "github.com/hanko-field/api/internal/platform/firestore"
)

const firestoreEmulatorImage = "gcr.io/google.com/cloudsdktool/cloud-sdk:emulators"

type sampleEntity struct {
	Name  string `firestore:"name"`
	Count int    `firestore:"count"`
}

func TestProviderAndRepositoryIntegration(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available: " + err.Error())
	}

	ensureDockerDaemon(t)

	port := freePort(t)
	endpoint := fmt.Sprintf("127.0.0.1:%d", port)
	containerID := startFirestoreEmulator(t, port)
	defer stopContainer(containerID)

	waitForEndpoint(t, endpoint, 30*time.Second)

	cfg := pconfig.FirestoreConfig{
		ProjectID:    "test-project",
		EmulatorHost: endpoint,
	}

	provider := pfirestore.NewProvider(cfg)
	t.Cleanup(func() {
		_ = provider.Close(context.Background())
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client, err := provider.Client(ctx)
	if err != nil {
		t.Fatalf("expected firestore client, got error: %v", err)
	}
	if client == nil {
		t.Fatalf("provider returned nil client")
	}

	repo := pfirestore.NewBaseRepository[sampleEntity](provider, "samples", nil, nil)

	if _, err := repo.Set(ctx, "sample-1", sampleEntity{Name: "alpha", Count: 1}); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	doc, err := repo.Get(ctx, "sample-1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if doc.ID != "sample-1" {
		t.Fatalf("expected id sample-1, got %s", doc.ID)
	}
	if doc.Data.Name != "alpha" || doc.Data.Count != 1 {
		t.Fatalf("unexpected data: %#v", doc.Data)
	}
	if doc.UpdateTime.IsZero() {
		t.Fatalf("expected update time to be set")
	}

	if _, err := repo.Update(ctx, "sample-1", []firestore.Update{{Path: "count", Value: 2}}); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	doc, err = repo.Get(ctx, "sample-1")
	if err != nil {
		t.Fatalf("get after update failed: %v", err)
	}
	if doc.Data.Count != 2 {
		t.Fatalf("expected count=2, got %d", doc.Data.Count)
	}

	docs, err := repo.Query(ctx, nil)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 document, got %d", len(docs))
	}

	if _, err := repo.Get(ctx, "missing"); err == nil {
		t.Fatalf("expected not found error")
	} else {
		type repoClassifier interface{ IsNotFound() bool }
		var cls repoClassifier
		if !errors.As(err, &cls) {
			t.Fatalf("expected repository error, got %v", err)
		}
		if !cls.IsNotFound() {
			t.Fatalf("expected not found classification")
		}
	}

	if err := provider.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		ref, err := repo.DocumentRef(ctx, "sample-1")
		if err != nil {
			return err
		}
		snap, err := tx.Get(ref)
		if err != nil {
			return err
		}
		var entity sampleEntity
		if err := snap.DataTo(&entity); err != nil {
			return err
		}
		entity.Count++
		return tx.Set(ref, entity)
	}); err != nil {
		t.Fatalf("transaction failed: %v", err)
	}

	doc, err = repo.Get(ctx, "sample-1")
	if err != nil {
		t.Fatalf("get after transaction failed: %v", err)
	}
	if doc.Data.Count != 3 {
		t.Fatalf("expected count=3 after txn, got %d", doc.Data.Count)
	}

	cancelCtx, cancelTxn := context.WithCancel(context.Background())
	cancelTxn()
	if err := provider.RunTransaction(cancelCtx, func(ctx context.Context, tx *firestore.Transaction) error {
		return nil
	}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled error, got %v", err)
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	addr, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("unable to allocate port: %v", err)
	}
	defer addr.Close()
	return addr.Addr().(*net.TCPAddr).Port
}

func startFirestoreEmulator(t *testing.T, port int) string {
	t.Helper()
	args := []string{
		"run", "-d", "--rm",
		"-p", fmt.Sprintf("%d:8080", port),
		firestoreEmulatorImage,
		"gcloud", "beta", "emulators", "firestore", "start",
		"--host-port=0.0.0.0:8080",
		"--quiet",
	}

	cmd := exec.Command("docker", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to start firestore emulator: %v - %s", err, string(out))
	}
	id := strings.TrimSpace(string(out))
	if id == "" {
		t.Fatalf("docker returned empty container id")
	}
	// Shorten the ID to match docker CLI behaviour for stop/remove commands.
	if len(id) > 12 {
		id = id[:12]
	}
	return id
}

func stopContainer(id string) {
	if id == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "stop", id)
	_ = cmd.Run()
}

func waitForEndpoint(t *testing.T, endpoint string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", endpoint, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		lastErr = err
		time.Sleep(250 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = errors.New("timeout waiting for endpoint")
	}
	t.Fatalf("emulator did not become ready: %v", lastErr)
}

func ensureDockerDaemon(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "info")
	if err := cmd.Run(); err != nil {
		t.Skip("docker daemon unavailable: " + err.Error())
	}
}
