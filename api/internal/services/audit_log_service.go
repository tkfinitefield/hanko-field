package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

const (
	defaultAuditSeverity = "info"
	defaultActorType     = "unknown"
	defaultHasherPrefix  = "sha256:"
)

// AuditLogger defines the logging contract used by the audit writer service.
type AuditLogger interface {
	Warnf(format string, args ...any)
}

type auditLogService struct {
	repo     repositories.AuditLogRepository
	clock    func() time.Time
	logger   AuditLogger
	hashSalt string
}

// AuditLogServiceDeps bundles constructor inputs for the audit writer service.
type AuditLogServiceDeps struct {
	Repository repositories.AuditLogRepository
	Clock      func() time.Time
	Logger     AuditLogger
	HashSalt   string
}

// NewAuditLogService creates an audit log writer backed by the supplied repository.
func NewAuditLogService(deps AuditLogServiceDeps) (AuditLogService, error) {
	if deps.Repository == nil {
		return nil, fmt.Errorf("audit log service: repository is required")
	}

	clock := deps.Clock
	if clock == nil {
		clock = time.Now
	}

	logger := deps.Logger
	if logger == nil {
		logger = noopAuditLogger{}
	}

	return &auditLogService{
		repo:     deps.Repository,
		clock:    func() time.Time { return clock().UTC() },
		logger:   logger,
		hashSalt: deps.HashSalt,
	}, nil
}

// Record persists an audit log entry after sanitising sensitive fields. Repository failures are
// logged but do not bubble up to callers to avoid interrupting the primary mutation flow.
func (s *auditLogService) Record(ctx context.Context, record AuditLogRecord) {
	if s.repo == nil {
		return
	}
	entry := s.buildEntry(record)
	if err := s.repo.Append(ctx, entry); err != nil {
		s.logger.Warnf("audit log append failed: %v", err)
	}
}

// List delegates to the repository to retrieve paginated audit logs.
func (s *auditLogService) List(ctx context.Context, filter AuditLogFilter) (domain.CursorPage[AuditLogEntry], error) {
	if s.repo == nil {
		return domain.CursorPage[AuditLogEntry]{}, fmt.Errorf("audit log service: repository is required")
	}
	page, err := s.repo.List(ctx, repositories.AuditLogFilter{
		TargetRef:  strings.TrimSpace(filter.TargetRef),
		Actor:      strings.TrimSpace(filter.Actor),
		ActorType:  strings.TrimSpace(filter.ActorType),
		Action:     strings.TrimSpace(filter.Action),
		DateRange:  filter.DateRange,
		Pagination: domain.Pagination{PageSize: filter.Pagination.PageSize, PageToken: filter.Pagination.PageToken},
	})
	if err != nil {
		return domain.CursorPage[AuditLogEntry]{}, err
	}
	return domain.CursorPage[AuditLogEntry]{
		Items:         page.Items,
		NextPageToken: page.NextPageToken,
	}, nil
}

func (s *auditLogService) buildEntry(record AuditLogRecord) domain.AuditLogEntry {
	now := s.clock()
	occurred := record.OccurredAt
	if occurred.IsZero() {
		occurred = now
	} else {
		occurred = occurred.UTC()
	}

	entry := domain.AuditLogEntry{
		Actor:     sanitizeActor(record.Actor),
		ActorType: normalizeActorType(record.ActorType, record.Actor),
		Action:    sanitizeAction(record.Action),
		TargetRef: sanitizeTargetRef(record.TargetRef),
		Severity:  normalizeSeverity(record.Severity),
		RequestID: sanitizeText(record.RequestID, 128),
		UserAgent: sanitizeText(record.UserAgent, 256),
		CreatedAt: occurred,
	}

	meta := s.prepareMetadata(record.Metadata, record.SensitiveMetadataKeys)
	if len(meta) > 0 {
		entry.Metadata = meta
	}

	diff := s.prepareDiff(record.Diff, record.SensitiveDiffKeys)
	if len(diff) > 0 {
		entry.Diff = diff
	}

	if ip := strings.TrimSpace(record.IPAddress); ip != "" {
		entry.IPHash = defaultHasherPrefix + s.hashString(ip)
	}

	return entry
}

func (s *auditLogService) prepareMetadata(metadata map[string]any, sensitiveKeys []string) map[string]any {
	if len(metadata) == 0 {
		return nil
	}
	if len(sensitiveKeys) > 0 {
		sensitiveKeys = normaliseKeys(sensitiveKeys)
	}
	result := make(map[string]any, len(metadata))
	for key, value := range metadata {
		trimmedKey := sanitizeMetadataKey(key)
		if trimmedKey == "" {
			continue
		}
		if containsKey(sensitiveKeys, trimmedKey) {
			result[trimmedKey] = defaultHasherPrefix + s.hashAny(value)
			continue
		}
		result[trimmedKey] = sanitizeMetadataValue(value)
	}
	return result
}

func (s *auditLogService) prepareDiff(diff map[string]AuditLogDiff, sensitiveKeys []string) map[string]any {
	if len(diff) == 0 {
		return nil
	}
	if len(sensitiveKeys) > 0 {
		sensitiveKeys = normaliseKeys(sensitiveKeys)
	}

	result := make(map[string]any, len(diff))
	for key, change := range diff {
		trimmedKey := sanitizeMetadataKey(key)
		if trimmedKey == "" {
			continue
		}
		if containsKey(sensitiveKeys, trimmedKey) {
			result[trimmedKey] = map[string]any{
				"before": defaultHasherPrefix + s.hashAny(change.Before),
				"after":  defaultHasherPrefix + s.hashAny(change.After),
			}
			continue
		}
		result[trimmedKey] = map[string]any{
			"before": sanitizeDiffValue(change.Before),
			"after":  sanitizeDiffValue(change.After),
		}
	}
	return result
}

func (s *auditLogService) hashString(value string) string {
	value = strings.TrimSpace(value)
	sum := sha256.Sum256([]byte(s.hashSalt + value))
	return hex.EncodeToString(sum[:])
}

func (s *auditLogService) hashAny(value any) string {
	switch v := value.(type) {
	case string:
		return s.hashString(v)
	case fmt.Stringer:
		return s.hashString(v.String())
	case []byte:
		return s.hashString(string(v))
	default:
		if b, err := json.Marshal(v); err == nil {
			return s.hashString(string(b))
		}
		if normalized := normalizeForHash(value); normalized != nil {
			if b, err := json.Marshal(normalized); err == nil {
				return s.hashString(string(b))
			}
		}
		return s.hashString(fmt.Sprintf("%T", value))
	}
}

type noopAuditLogger struct{}

func (noopAuditLogger) Warnf(string, ...any) {}

func sanitizeActor(actor string) string {
	return sanitizeText(actor, 160)
}

func normalizeActorType(actorType string, actor string) string {
	normalized := strings.ToLower(strings.TrimSpace(actorType))
	switch normalized {
	case "user", "staff", "system", "service":
		return normalized
	}
	actor = strings.ToLower(strings.TrimSpace(actor))
	switch {
	case strings.HasPrefix(actor, "/users/"), strings.HasPrefix(actor, "user:"):
		return "user"
	case strings.HasPrefix(actor, "/staff/"), strings.HasPrefix(actor, "staff:"):
		return "staff"
	case actor == "system" || strings.HasPrefix(actor, "system:"):
		return "system"
	default:
		return defaultActorType
	}
}

func sanitizeAction(action string) string {
	return sanitizeText(action, 120)
}

func sanitizeTargetRef(target string) string {
	return sanitizeText(target, 200)
}

func normalizeSeverity(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "warn", "warning":
		return "warn"
	case "error":
		return "error"
	default:
		return defaultAuditSeverity
	}
}

func sanitizeMetadataKey(key string) string {
	return sanitizeText(strings.TrimSpace(key), 80)
}

func sanitizeMetadataValue(value any) any {
	switch v := value.(type) {
	case string:
		return sanitizeText(v, 512)
	case fmt.Stringer:
		return sanitizeText(v.String(), 512)
	default:
		return v
	}
}

func sanitizeDiffValue(value any) any {
	switch v := value.(type) {
	case string:
		return sanitizeText(v, 512)
	case fmt.Stringer:
		return sanitizeText(v.String(), 512)
	default:
		return v
	}
}

type normalizedKV struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

func normalizeForHash(value any) any {
	return normalizeValueForHash(reflect.ValueOf(value))
}

func normalizeValueForHash(v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}

	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return nil
		}
		return normalizeValueForHash(v.Elem())
	case reflect.Map:
		if v.IsNil() {
			return nil
		}
		keys := v.MapKeys()
		if len(keys) == 0 {
			return []normalizedKV{}
		}
		pairs := make([]normalizedKV, 0, len(keys))
		for _, key := range keys {
			pairs = append(pairs, normalizedKV{
				Key:   formatHashKey(key),
				Value: normalizeValueForHash(v.MapIndex(key)),
			})
		}
		sort.Slice(pairs, func(i, j int) bool { return pairs[i].Key < pairs[j].Key })
		return pairs
	case reflect.Slice:
		if v.IsNil() {
			return nil
		}
		if v.Type().Elem().Kind() == reflect.Uint8 {
			bytes := make([]byte, v.Len())
			for i := 0; i < v.Len(); i++ {
				bytes[i] = byte(v.Index(i).Uint())
			}
			return bytes
		}
		fallthrough
	case reflect.Array:
		length := v.Len()
		result := make([]any, length)
		for i := 0; i < length; i++ {
			result[i] = normalizeValueForHash(v.Index(i))
		}
		return result
	case reflect.Struct:
		t := v.Type()
		pairs := make([]normalizedKV, 0, v.NumField())
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" {
				continue // unexported
			}
			name := field.Name
			if tag := field.Tag.Get("json"); tag != "" {
				parts := strings.Split(tag, ",")
				if len(parts) > 0 && parts[0] == "-" {
					continue
				}
				if len(parts) > 0 && parts[0] != "" {
					name = parts[0]
				}
			}
			pairs = append(pairs, normalizedKV{
				Key:   name,
				Value: normalizeValueForHash(v.Field(i)),
			})
		}
		sort.Slice(pairs, func(i, j int) bool { return pairs[i].Key < pairs[j].Key })
		return pairs
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint()
	case reflect.Float32, reflect.Float64:
		return v.Float()
	case reflect.String:
		return v.String()
	default:
		if v.CanInterface() {
			return v.Interface()
		}
		return fmt.Sprintf("<unexported:%s>", v.Type().String())
	}
}

func formatHashKey(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return "<nil>"
		}
		return formatHashKey(v.Elem())
	case reflect.String:
		return v.String()
	case reflect.Bool:
		if v.Bool() {
			return "true"
		}
		return "false"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return fmt.Sprintf("%d", v.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%g", v.Float())
	default:
		if v.CanInterface() {
			return fmt.Sprintf("%#v", v.Interface())
		}
		return v.Type().String()
	}
}

func normaliseKeys(keys []string) []string {
	if len(keys) == 0 {
		return keys
	}
	unique := make(map[string]struct{}, len(keys))
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		trimmed := sanitizeMetadataKey(key)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if _, exists := unique[lower]; exists {
			continue
		}
		unique[lower] = struct{}{}
		result = append(result, lower)
	}
	return result
}

func containsKey(keys []string, candidate string) bool {
	candidate = strings.ToLower(candidate)
	for _, key := range keys {
		if key == candidate {
			return true
		}
	}
	return false
}

func sanitizeText(input string, limit int) string {
	if limit <= 0 {
		limit = 256
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	var builder strings.Builder
	for _, r := range input {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			continue
		}
		builder.WriteRune(r)
		if builder.Len() >= limit {
			break
		}
	}
	return builder.String()
}
