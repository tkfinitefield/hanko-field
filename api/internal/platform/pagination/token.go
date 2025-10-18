package pagination

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// EncodeToken serialises the provided cursor into a base64 URL-safe page token.
func EncodeToken(cursor Cursor) (string, error) {
	if len(cursor.StartAfter) == 0 && len(cursor.StartAt) == 0 {
		return "", nil
	}
	data, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("pagination: encode token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

// DecodeToken parses the page token produced by EncodeToken back into a cursor.
func DecodeToken(token string) (Cursor, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Cursor{}, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: %v", ErrInvalidPageToken, err)
	}
	var cursor Cursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return Cursor{}, fmt.Errorf("%w: %v", ErrInvalidPageToken, err)
	}
	return cursor, nil
}
