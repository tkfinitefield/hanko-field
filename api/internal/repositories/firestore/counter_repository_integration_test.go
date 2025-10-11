//go:build integration

package firestore

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"sync"
	"testing"
	"time"

	pconfig "github.com/hanko-field/api/internal/platform/config"
	pfirestore "github.com/hanko-field/api/internal/platform/firestore"
	"github.com/hanko-field/api/internal/repositories"
)

func TestCounterRepositoryIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test skipped in short mode")
	}

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available: " + err.Error())
	}

	ensureDockerDaemon(t)

	port := freePort(t)
	endpoint := fmt.Sprintf("127.0.0.1:%d", port)
	containerID := startFirestoreEmulator(t, port)
	t.Cleanup(func() { stopContainer(containerID) })

	waitForEndpoint(t, endpoint, 30*time.Second)

	cfg := pconfig.FirestoreConfig{
		ProjectID:    "counter-test",
		EmulatorHost: endpoint,
	}

	provider := pfirestore.NewProvider(cfg)
	t.Cleanup(func() {
		_ = provider.Close(context.Background())
	})

	repo, err := NewCounterRepository(provider)
	if err != nil {
		t.Fatalf("new counter repository: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	const workers = 16
	results := make([]int64, workers)
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func(idx int) {
			defer wg.Done()
			value, err := repo.Next(ctx, "orders:global", 1)
			if err != nil {
				t.Errorf("next(%d): %v", idx, err)
				return
			}
			results[idx] = value
		}(i)
	}

	wg.Wait()

	for _, val := range results {
		if val == 0 {
			t.Fatalf("expected counter increments to succeed, got zero values: %+v", results)
		}
	}

	sort.Slice(results, func(i, j int) bool { return results[i] < results[j] })
	for i, val := range results {
		expected := int64(i + 1)
		if val != expected {
			t.Fatalf("expected sequence %d at position %d, got %d", expected, i, val)
		}
	}

	// Configure a bounded counter and assert exhaustion.
	max := int64(3)
	start := int64(0)
	if err := repo.Configure(ctx, "invoices:regional", repositories.CounterConfig{
		Step:         1,
		MaxValue:     &max,
		InitialValue: &start,
	}); err != nil {
		t.Fatalf("configure counter: %v", err)
	}

	for i := int64(1); i <= max; i++ {
		value, err := repo.Next(ctx, "invoices:regional", 0)
		if err != nil {
			t.Fatalf("next bounded %d: %v", i, err)
		}
		if value != i {
			t.Fatalf("expected bounded counter %d got %d", i, value)
		}
	}

	_, err = repo.Next(ctx, "invoices:regional", 0)
	if err == nil {
		t.Fatalf("expected exhaustion error")
	}
	var counterErr *repositories.CounterError
	if !errors.As(err, &counterErr) {
		t.Fatalf("expected counter error, got %T %v", err, err)
	}
	if counterErr.Code != repositories.CounterErrorExhausted {
		t.Fatalf("expected exhausted code, got %s", counterErr.Code)
	}
}
