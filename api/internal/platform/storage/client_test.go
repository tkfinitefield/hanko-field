package storage

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/hanko-field/api/internal/platform/auth"
)

type fakeSigner struct {
	email    string
	payloads [][]byte
	err      error
}

func (f *fakeSigner) Email() string {
	return f.email
}

func (f *fakeSigner) SignBytes(_ context.Context, payload []byte) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.payloads = append(f.payloads, append([]byte(nil), payload...))
	return []byte("signed"), nil
}

func TestSignedURLUploadSuccess(t *testing.T) {
	signer := &fakeSigner{email: "test@example.iam.gserviceaccount.com"}
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	client, err := NewClient(signer, WithClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	opts := SignedURLOptions{
		Upload: &UploadOptions{
			Method:              "PUT",
			ContentType:         "image/png",
			ContentMD5:          "xN0dYbCPv0CM0k9d1u8G7g==",
			RequireMD5:          true,
			AllowedContentTypes: []string{"image/png"},
			MaxSize:             1 << 20,
			ExpiresIn:           10 * time.Minute,
		},
	}

	res, err := client.SignedURL(context.Background(), "bucket", "assets/designs/design123/sources/upload456/file.png", opts)
	if err != nil {
		t.Fatalf("SignedURL returned error: %v", err)
	}

	if res.Method != httpMethodPut {
		t.Fatalf("expected method PUT, got %s", res.Method)
	}
	expectedExpiry := now.Add(10 * time.Minute)
	if !res.ExpiresAt.Equal(expectedExpiry) {
		t.Fatalf("expected expiry %v, got %v", expectedExpiry, res.ExpiresAt)
	}
	if res.Headers["Content-Type"] != "image/png" {
		t.Fatalf("expected Content-Type header, got %v", res.Headers)
	}
	if res.Headers["Content-MD5"] != "xN0dYbCPv0CM0k9d1u8G7g==" {
		t.Fatalf("expected Content-MD5 header, got %v", res.Headers)
	}
	if res.Headers["x-goog-content-length-range"] != "0,1048576" {
		t.Fatalf("expected content length header, got %v", res.Headers)
	}

	parsed, err := url.Parse(res.URL)
	if err != nil {
		t.Fatalf("failed to parse signed URL: %v", err)
	}
	if !strings.Contains(parsed.RawQuery, "X-Goog-Signature=") {
		t.Fatalf("expected signature in query: %s", parsed.RawQuery)
	}
	if len(signer.payloads) == 0 {
		t.Fatalf("expected signer to be invoked")
	}
}

func TestSignedURLUploadRejectsInvalidContentType(t *testing.T) {
	signer := &fakeSigner{email: "test@example.iam.gserviceaccount.com"}
	client, err := NewClient(signer)
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	opts := SignedURLOptions{
		Upload: &UploadOptions{
			Method:              "PUT",
			ContentType:         "application/pdf",
			AllowedContentTypes: []string{"image/png"},
		},
	}

	_, err = client.SignedURL(context.Background(), "bucket", "object", opts)
	if !errors.Is(err, errContentTypeDenied) {
		t.Fatalf("expected errContentTypeDenied, got %v", err)
	}
}

func TestSignedURLUploadRequiresMD5(t *testing.T) {
	signer := &fakeSigner{email: "test@example.iam.gserviceaccount.com"}
	client, err := NewClient(signer)
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	opts := SignedURLOptions{
		Upload: &UploadOptions{
			Method:      "PUT",
			ContentType: "image/png",
			RequireMD5:  true,
		},
	}

	_, err = client.SignedURL(context.Background(), "bucket", "object", opts)
	if !errors.Is(err, errMD5Required) {
		t.Fatalf("expected errMD5Required, got %v", err)
	}
}

func TestSignedURLDownloadPermissionDenied(t *testing.T) {
	signer := &fakeSigner{email: "test@example.iam.gserviceaccount.com"}
	client, err := NewClient(signer)
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	opts := SignedURLOptions{
		Download: &DownloadOptions{
			OwnerID:  "owner-123",
			Identity: &auth.Identity{UID: "other-456"},
		},
	}

	_, err = client.SignedURL(context.Background(), "bucket", "object", opts)
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestSignedURLDownloadAllowsStaff(t *testing.T) {
	signer := &fakeSigner{email: "test@example.iam.gserviceaccount.com"}
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	client, err := NewClient(signer, WithClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	opts := SignedURLOptions{
		Download: &DownloadOptions{
			OwnerID:   "owner-123",
			Identity:  &auth.Identity{UID: "staff-1", Roles: []string{auth.RoleStaff}},
			ExpiresIn: 5 * time.Minute,
		},
	}

	res, err := client.SignedURL(context.Background(), "bucket", "object", opts)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.Method != httpMethodGet {
		t.Fatalf("expected GET method, got %s", res.Method)
	}
	if !res.ExpiresAt.Equal(now.Add(5 * time.Minute)) {
		t.Fatalf("unexpected expiry: %v", res.ExpiresAt)
	}
}

func TestSignedURLDownloadExpiryTooLong(t *testing.T) {
	signer := &fakeSigner{email: "test@example.iam.gserviceaccount.com"}
	client, err := NewClient(signer)
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	opts := SignedURLOptions{
		Download: &DownloadOptions{
			Identity:  &auth.Identity{UID: "owner", Roles: []string{auth.RoleUser}},
			OwnerID:   "owner",
			ExpiresIn: 30 * time.Minute,
		},
	}

	_, err = client.SignedURL(context.Background(), "bucket", "object", opts)
	if !errors.Is(err, errExpiryTooLong) {
		t.Fatalf("expected errExpiryTooLong, got %v", err)
	}
}
