//go:build integration

package firestore

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"sort"
	"strings"
	"testing"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	pconfig "github.com/hanko-field/api/internal/platform/config"
	pfirestore "github.com/hanko-field/api/internal/platform/firestore"
	"github.com/hanko-field/api/internal/repositories"
)

func TestInventoryRepositoryIntegration(t *testing.T) {
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
		ProjectID:    "inventory-test",
		EmulatorHost: endpoint,
	}

	provider := pfirestore.NewProvider(cfg)
	t.Cleanup(func() {
		_ = provider.Close(context.Background())
	})

	repo, err := NewInventoryRepository(provider)
	if err != nil {
		t.Fatalf("new inventory repository: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := provider.Client(ctx)
	if err != nil {
		t.Fatalf("provider client: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	seedStock := map[string]any{
		"sku":         "SKU-001",
		"productRef":  "/products/prod_001",
		"onHand":      5,
		"reserved":    0,
		"available":   5,
		"safetyStock": 3,
		"safetyDelta": 2,
		"updatedAt":   now,
	}

	if _, err := client.Collection(inventoryCollection).Doc("SKU-001").Set(ctx, seedStock); err != nil {
		t.Fatalf("seed stock: %v", err)
	}

	reservation := domain.InventoryReservation{
		ID:       "sr_test_1",
		OrderRef: "/orders/o_test_1",
		UserRef:  "/users/u_test",
		Lines: []domain.InventoryReservationLine{
			{ProductRef: "/products/prod_001", SKU: "SKU-001", Quantity: 3},
		},
		ExpiresAt: now.Add(30 * time.Minute),
		CreatedAt: now,
		UpdatedAt: now,
	}

	reserveResult, err := repo.Reserve(ctx, repositories.InventoryReserveRequest{
		Reservation: reservation,
		Now:         now,
	})
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if reserveResult.Reservation.Status != reservationStatusReserved {
		t.Fatalf("expected reserved status, got %s", reserveResult.Reservation.Status)
	}
	stock, ok := reserveResult.Stocks["SKU-001"]
	if !ok {
		t.Fatalf("reserve result missing stock")
	}
	if stock.Reserved != 3 {
		t.Fatalf("expected reserved=3 got %d", stock.Reserved)
	}

	var invErr *repositories.InventoryError

	_, err = repo.Reserve(ctx, repositories.InventoryReserveRequest{
		Reservation: reservation,
		Now:         now.Add(time.Second),
	})
	if err == nil {
		t.Fatalf("expected duplicate reservation error")
	}
	if !errors.As(err, &invErr) || invErr.Code != repositories.InventoryErrorInvalidReservationState {
		t.Fatalf("expected invalid reservation state for duplicate, got %v", err)
	}

	_, err = repo.Reserve(ctx, repositories.InventoryReserveRequest{
		Reservation: domain.InventoryReservation{
			ID:        "sr_test_2",
			OrderRef:  "/orders/o_test_2",
			UserRef:   "/users/u_test",
			Lines:     []domain.InventoryReservationLine{{ProductRef: "/products/prod_001", SKU: "SKU-001", Quantity: 3}},
			ExpiresAt: now.Add(30 * time.Minute),
			CreatedAt: now,
			UpdatedAt: now,
		},
		Now: now,
	})
	if err == nil {
		t.Fatalf("expected insufficient stock error")
	}
	invErr = nil
	if !errors.As(err, &invErr) {
		t.Fatalf("expected inventory error, got %T %v", err, err)
	}
	if invErr.Code != repositories.InventoryErrorInsufficientStock {
		t.Fatalf("expected insufficient stock code, got %s", invErr.Code)
	}

	commitResult, err := repo.Commit(ctx, repositories.InventoryCommitRequest{
		ReservationID: reservation.ID,
		OrderRef:      reservation.OrderRef,
		Now:           now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	stock = commitResult.Stocks["SKU-001"]
	if stock.OnHand != 2 || stock.Reserved != 0 {
		t.Fatalf("unexpected stock after commit: %+v", stock)
	}
	if commitResult.Reservation.Status != reservationStatusCommitted {
		t.Fatalf("expected committed status, got %s", commitResult.Reservation.Status)
	}

	releaseReservation := domain.InventoryReservation{
		ID:        "sr_test_release",
		OrderRef:  "/orders/o_test_release",
		UserRef:   "/users/u_test",
		Lines:     []domain.InventoryReservationLine{{ProductRef: "/products/prod_001", SKU: "SKU-001", Quantity: 1}},
		ExpiresAt: now.Add(10 * time.Minute),
		CreatedAt: now,
		UpdatedAt: now,
	}

	relReserve, err := repo.Reserve(ctx, repositories.InventoryReserveRequest{
		Reservation: releaseReservation,
		Now:         now,
	})
	if err != nil {
		t.Fatalf("reserve for release: %v", err)
	}
	if relReserve.Stocks["SKU-001"].Reserved != 1 {
		t.Fatalf("expected reserved 1 after second reserve")
	}

	releaseResult, err := repo.Release(ctx, repositories.InventoryReleaseRequest{
		ReservationID: releaseReservation.ID,
		Reason:        "checkout_cancelled",
		Now:           now.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	stock = releaseResult.Stocks["SKU-001"]
	if stock.Reserved != 0 {
		t.Fatalf("expected reserved 0 after release, got %d", stock.Reserved)
	}
	if releaseResult.Reservation.Status != reservationStatusReleased {
		t.Fatalf("expected released status, got %s", releaseResult.Reservation.Status)
	}

	lowPage, err := repo.ListLowStock(ctx, repositories.InventoryLowStockQuery{Threshold: 0, PageSize: 10})
	if err != nil {
		t.Fatalf("list low stock: %v", err)
	}
	sort.SliceStable(lowPage.Items, func(i, j int) bool { return lowPage.Items[i].SKU < lowPage.Items[j].SKU })
	if len(lowPage.Items) != 1 {
		t.Fatalf("expected 1 low stock item, got %d", len(lowPage.Items))
	}
	if lowPage.Items[0].SafetyDelta >= 0 {
		t.Fatalf("expected negative safety delta, got %d", lowPage.Items[0].SafetyDelta)
	}

	configured, err := repo.ConfigureSafetyStock(ctx, repositories.InventorySafetyStockConfig{
		SKU:         "SKU-001",
		ProductRef:  "/materials/mat_wood",
		SafetyStock: 8,
		Now:         now.Add(5 * time.Minute),
	})
	if err != nil {
		t.Fatalf("configure safety stock existing: %v", err)
	}
	if configured.SafetyStock != 8 {
		t.Fatalf("expected updated safety stock 8 got %d", configured.SafetyStock)
	}
	created, err := repo.ConfigureSafetyStock(ctx, repositories.InventorySafetyStockConfig{
		SKU:         "MAT-002",
		ProductRef:  "/materials/mat_002",
		SafetyStock: 4,
		Now:         now.Add(6 * time.Minute),
	})
	if err != nil {
		t.Fatalf("configure safety stock new: %v", err)
	}
	if created.SKU != "MAT-002" || created.SafetyStock != 4 {
		t.Fatalf("expected new stock for MAT-002, got %+v", created)
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
	if len(id) > 12 {
		id = id[:12]
	}
	return id
}

func ensureDockerDaemon(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "info")
	if err := cmd.Run(); err != nil {
		t.Fatalf("docker daemon not available: %v", err)
	}
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
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", endpoint, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("firestore emulator at %s did not become ready within %s", endpoint, timeout)
}

const firestoreEmulatorImage = "gcr.io/google.com/cloudsdktool/cloud-sdk:emulators"
