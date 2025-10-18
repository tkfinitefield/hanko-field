package storage

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Signer represents the capability to sign arbitrary payloads for generating signed URLs.
type Signer interface {
	// Email returns the service account email used as the GoogleAccessID when signing URLs.
	Email() string
	// SignBytes signs the provided payload with the underlying private key.
	SignBytes(ctx context.Context, payload []byte) ([]byte, error)
}

// ServiceAccountSigner implements Signer backed by a service account private key.
type ServiceAccountSigner struct {
	email string
	key   *rsa.PrivateKey
}

// serviceAccountKey models the fields required from a service account JSON key file.
type serviceAccountKey struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
}

// NewServiceAccountSignerFromJSON builds a signer from a raw service account JSON key.
func NewServiceAccountSignerFromJSON(data []byte) (*ServiceAccountSigner, error) {
	if len(data) == 0 {
		return nil, errors.New("storage: service account JSON is empty")
	}

	var key serviceAccountKey
	if err := json.Unmarshal(data, &key); err != nil {
		return nil, fmt.Errorf("storage: decode service account json: %w", err)
	}

	key.PrivateKey = strings.TrimSpace(key.PrivateKey)
	if key.PrivateKey == "" {
		return nil, errors.New("storage: private_key missing in service account JSON")
	}
	key.ClientEmail = strings.TrimSpace(key.ClientEmail)
	if key.ClientEmail == "" {
		return nil, errors.New("storage: client_email missing in service account JSON")
	}

	rsaKey, err := parseRSAPrivateKey(key.PrivateKey)
	if err != nil {
		return nil, err
	}

	return &ServiceAccountSigner{email: key.ClientEmail, key: rsaKey}, nil
}

// NewServiceAccountSignerFromFile builds a signer by reading the JSON key from disk.
func NewServiceAccountSignerFromFile(path string) (*ServiceAccountSigner, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("storage: read service account file: %w", err)
	}
	return NewServiceAccountSignerFromJSON(contents)
}

// Email returns the signer service account email.
func (s *ServiceAccountSigner) Email() string {
	if s == nil {
		return ""
	}
	return s.email
}

// SignBytes applies RSA SHA256 signing over the payload.
func (s *ServiceAccountSigner) SignBytes(ctx context.Context, payload []byte) ([]byte, error) {
	if s == nil || s.key == nil {
		return nil, errors.New("storage: signer not initialised")
	}
	if len(payload) == 0 {
		return nil, errors.New("storage: payload is empty")
	}
	if ctx != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}

	digest := sha256.Sum256(payload)
	sig, err := rsa.SignPKCS1v15(rand.Reader, s.key, crypto.SHA256, digest[:])
	if err != nil {
		return nil, fmt.Errorf("storage: sign payload: %w", err)
	}
	return sig, nil
}

func parseRSAPrivateKey(pemData string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, errors.New("storage: failed to decode PEM private key")
	}

	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rsaKey, ok := key.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
		return nil, errors.New("storage: private key is not RSA")
	}

	rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("storage: parse RSA private key: %w", err)
	}
	return rsaKey, nil
}
