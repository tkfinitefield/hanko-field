package checkout

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

func fakeSessionResponse(provider string) SessionResponse {
	now := time.Now().UTC()
	id := randomID("sess")
	pi := randomID("pi")
	return SessionResponse{
		SessionID:      id,
		URL:            fmt.Sprintf("https://checkout.stripe.com/c/pay/%s", strings.TrimPrefix(id, "sess_")),
		ClientSecret:   fmt.Sprintf("%s_secret_%s", pi, randomID("key")),
		PublishableKey: "pk_test_51FakeExampleKey", // demo key for client-side mounts
		Status:         "requires_action",
		Provider:       provider,
		Amount:         482000,
		Currency:       "JPY",
		ExpiresAt:      now.Add(15 * time.Minute),
	}
}

func fakeConfirmResponse(sessionID string) ConfirmResponse {
	orderID := randomID("ord")
	return ConfirmResponse{
		OrderID: orderID,
		Status:  "pending_review",
		NextURL: fmt.Sprintf("/checkout/review?order=%s", strings.TrimPrefix(orderID, "ord_")),
	}
}

func randomID(prefix string) string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err == nil {
		return fmt.Sprintf("%s_%s", strings.TrimSpace(prefix), hex.EncodeToString(b))
	}
	ts := time.Now().UnixNano()
	return fmt.Sprintf("%s_%d", strings.TrimSpace(prefix), ts)
}
