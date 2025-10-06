package observability

import "unicode"

const defaultStringLimit = 256

// sanitizeString trims unwanted characters and limits string length to avoid log injection.
func sanitizeString(value string, limit int) string {
	if limit <= 0 {
		limit = defaultStringLimit
	}

	cleaned := make([]rune, 0, len(value))
	for _, r := range value {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			continue
		}
		cleaned = append(cleaned, r)
	}
	if len(cleaned) > limit {
		cleaned = cleaned[:limit]
	}
	return string(cleaned)
}

// SanitizeRoute removes control characters and enforces length constraints on routes.
func SanitizeRoute(route string) string {
	if route == "" {
		return "/"
	}
	return sanitizeString(route, 180)
}

// SanitizeMethod removes control characters in HTTP methods.
func SanitizeMethod(method string) string {
	return sanitizeString(method, 10)
}

// SanitizeUserID limits potential identifiers to reduce PII leakage in logs.
func SanitizeUserID(uid string) string {
	if len(uid) == 0 {
		return ""
	}
	return sanitizeString(uid, 64)
}
