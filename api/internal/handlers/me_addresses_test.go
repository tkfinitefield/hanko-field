package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/services"
)

func TestMeHandlersListAddresses(t *testing.T) {
	handler := NewMeHandlers(nil, &stubUserService{
		listAddressesFunc: func(ctx context.Context, userID string) ([]services.Address, error) {
			return []services.Address{
				{
					ID:              "addr-1",
					Recipient:       "Test User",
					Line1:           "1-2-3",
					City:            "Chiyoda",
					PostalCode:      "100-0001",
					Country:         "JP",
					DefaultShipping: true,
					DefaultBilling:  true,
				},
			}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	rr := httptest.NewRecorder()
	router := chi.NewRouter()
	router.Route("/", handler.addressRoutes)
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var payload []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON response: %v", err)
	}
	if len(payload) != 1 {
		t.Fatalf("expected 1 address, got %d", len(payload))
	}
	if payload[0]["id"] != "addr-1" {
		t.Fatalf("unexpected id %v", payload[0]["id"])
	}
}

func TestMeHandlersCreateAddress(t *testing.T) {
	var captured services.UpsertAddressCommand
	handler := NewMeHandlers(nil, &stubUserService{
		upsertAddressFunc: func(ctx context.Context, cmd services.UpsertAddressCommand) (services.Address, error) {
			captured = cmd
			saved := cmd.Address
			saved.ID = "addr-2"
			return saved, nil
		},
	})

	body := []byte(`{"recipient":"Tester","line1":"1-2-3","city":"Shibuya","postal_code":"1500001","country":"jp","default_shipping":true}`)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-2"}))

	rr := httptest.NewRecorder()
	router := chi.NewRouter()
	router.Route("/", handler.addressRoutes)
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rr.Code)
	}
	if captured.UserID != "user-2" {
		t.Fatalf("expected user id user-2, got %s", captured.UserID)
	}
	if captured.DefaultShipping == nil || !*captured.DefaultShipping {
		t.Fatalf("expected default shipping flag set")
	}
}

func TestMeHandlersDeleteAddress(t *testing.T) {
	var captured services.DeleteAddressCommand
	handler := NewMeHandlers(nil, &stubUserService{
		deleteAddressFunc: func(ctx context.Context, cmd services.DeleteAddressCommand) error {
			captured = cmd
			return nil
		},
	})

	req := httptest.NewRequest(http.MethodDelete, "/addr-3?replacement_id=addr-4", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-3"}))

	rr := httptest.NewRecorder()
	router := chi.NewRouter()
	router.Route("/", handler.addressRoutes)
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rr.Code)
	}
	if captured.AddressID != "addr-3" {
		t.Fatalf("expected address id addr-3, got %s", captured.AddressID)
	}
	if captured.ReplacementID == nil || *captured.ReplacementID != "addr-4" {
		t.Fatalf("expected replacement id addr-4, got %v", captured.ReplacementID)
	}
}
