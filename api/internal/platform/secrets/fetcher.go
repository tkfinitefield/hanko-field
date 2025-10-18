package secrets

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/googleapis/gax-go/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultEnvironment  = "local"
	defaultFallbackPath = ".secrets.local"
	metricNamespace     = "github.com/hanko-field/api/internal/platform/secrets"
)

var secretManagerClientFactory = func(ctx context.Context, opts ...option.ClientOption) (*secretmanager.Client, error) {
	return secretmanager.NewClient(ctx, opts...)
}

// Fetcher resolves secret:// references using Google Secret Manager with local caching and fallbacks.
type Fetcher struct {
	client     secretManagerClient
	ownsClient bool

	logger *zap.Logger

	env           string
	defaultProjID string
	projectMap    map[string]string
	versionPins   map[string]string

	fallbackPath string
	fallbackOnce sync.Once
	fallbackVals map[string]string
	fallbackErr  error

	mu       sync.RWMutex
	cache    map[string]cacheEntry
	watchers map[string][]chan struct{}

	meter            metric.Meter
	latency          metric.Float64Histogram
	latencyEnabled   bool
	cacheHits        metric.Int64Counter
	cacheHitsEnabled bool
}

type cacheEntry struct {
	value     string
	canonical string
	version   string
	fetchedAt time.Time
	source    string
}

type secretManagerClient interface {
	AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error)
	Close() error
}

type fetcherConfig struct {
	logger       *zap.Logger
	env          string
	defaultProj  string
	projectMap   map[string]string
	fallbackPath string
	meter        metric.Meter
	client       secretManagerClient
	clientOpts   []option.ClientOption
	versionPins  map[string]string
}

// Option customises Fetcher construction.
type Option func(*fetcherConfig)

// WithLogger sets the logger used for diagnostic output.
func WithLogger(logger *zap.Logger) Option {
	return func(cfg *fetcherConfig) {
		cfg.logger = logger
	}
}

// WithEnvironment selects the environment key used to resolve per-environment project IDs.
func WithEnvironment(env string) Option {
	return func(cfg *fetcherConfig) {
		cfg.env = strings.ToLower(strings.TrimSpace(env))
	}
}

// WithDefaultProject configures the fallback project ID used when no environment-specific mapping matches.
func WithDefaultProject(projectID string) Option {
	return func(cfg *fetcherConfig) {
		cfg.defaultProj = strings.TrimSpace(projectID)
	}
}

// WithProjectMap supplies environment-specific project IDs.
func WithProjectMap(m map[string]string) Option {
	return func(cfg *fetcherConfig) {
		cfg.projectMap = copyStringMap(m)
	}
}

// WithFallbackFile overrides the path to the local fallback secrets file.
func WithFallbackFile(path string) Option {
	return func(cfg *fetcherConfig) {
		cfg.fallbackPath = strings.TrimSpace(path)
	}
}

// WithMeter injects a custom OpenTelemetry meter.
func WithMeter(m metric.Meter) Option {
	return func(cfg *fetcherConfig) {
		cfg.meter = m
	}
}

// WithSecretManagerClient injects a preconfigured Secret Manager client (primarily for tests).
func WithSecretManagerClient(client secretManagerClient) Option {
	return func(cfg *fetcherConfig) {
		cfg.client = client
	}
}

// WithClientOptions forwards Cloud client options when constructing the Secret Manager client.
func WithClientOptions(opts ...option.ClientOption) Option {
	return func(cfg *fetcherConfig) {
		cfg.clientOpts = append(cfg.clientOpts, opts...)
	}
}

// WithVersionPins sets explicit version overrides keyed by canonical secret reference.
func WithVersionPins(pins map[string]string) Option {
	return func(cfg *fetcherConfig) {
		cfg.versionPins = copyStringMap(pins)
	}
}

// NewFetcher builds a Fetcher with secret caching, metrics, and local fallback support.
func NewFetcher(ctx context.Context, opts ...Option) (*Fetcher, error) {
	cfg := fetcherConfig{}
	cfg.logger = zap.NewNop()
	cfg.env = strings.ToLower(strings.TrimSpace(os.Getenv("API_SECURITY_ENVIRONMENT")))
	if cfg.env == "" {
		cfg.env = defaultEnvironment
	}
	cfg.fallbackPath = defaultFallbackPath
	cfg.projectMap = map[string]string{}
	cfg.versionPins = map[string]string{}

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.logger == nil {
		cfg.logger = zap.NewNop()
	}

	var meter metric.Meter
	if cfg.meter != nil {
		meter = cfg.meter
	} else {
		meter = otel.GetMeterProvider().Meter(metricNamespace)
	}

	latency, latencyErr := meter.Float64Histogram(
		"secrets.fetch.latency",
		metric.WithUnit("ms"),
		metric.WithDescription("Latency in milliseconds for secret fetch attempts"),
	)
	if latencyErr != nil {
		cfg.logger.Warn("secrets: unable to register latency metric", zap.Error(latencyErr))
	}

	cacheHits, cacheErr := meter.Int64Counter(
		"secrets.fetch.cache_hits",
		metric.WithDescription("Count of cache hits when resolving secrets"),
	)
	if cacheErr != nil {
		cfg.logger.Warn("secrets: unable to register cache hit metric", zap.Error(cacheErr))
	}

	f := &Fetcher{
		logger:           cfg.logger,
		env:              cfg.env,
		defaultProjID:    cfg.defaultProj,
		projectMap:       copyStringMap(cfg.projectMap),
		versionPins:      copyStringMap(cfg.versionPins),
		fallbackPath:     cfg.fallbackPath,
		cache:            make(map[string]cacheEntry),
		watchers:         make(map[string][]chan struct{}),
		meter:            meter,
		latency:          latency,
		latencyEnabled:   latencyErr == nil,
		cacheHits:        cacheHits,
		cacheHitsEnabled: cacheErr == nil,
	}

	if cfg.client != nil {
		f.client = cfg.client
	} else {
		client, err := secretManagerClientFactory(ctx, cfg.clientOpts...)
		if err != nil {
			cfg.logger.Warn("secrets: secret manager client unavailable; operating in fallback mode", zap.Error(err))
		} else {
			f.client = client
			f.ownsClient = true
		}
	}

	return f, nil
}

// Close releases resources held by the fetcher.
func (f *Fetcher) Close() error {
	f.mu.Lock()
	for canonical, watchers := range f.watchers {
		delete(f.watchers, canonical)
		for _, ch := range watchers {
			closeSafe(ch)
		}
	}
	f.mu.Unlock()

	if f.ownsClient && f.client != nil {
		return f.client.Close()
	}
	return nil
}

// Resolve retrieves the secret value for the supplied reference, consulting cache and fallbacks as needed.
func (f *Fetcher) Resolve(ctx context.Context, ref string) (string, error) {
	start := time.Now()
	parsed, err := parseReference(ref)
	if err != nil {
		return "", err
	}

	version := f.selectVersion(parsed)
	key := cacheKey(parsed.Canonical, version)

	if value, ok := f.lookupCache(key); ok {
		f.recordCacheHit(ctx, parsed)
		f.recordLatency(ctx, time.Since(start), "cache", nil)
		return value, nil
	}

	projectID := f.projectID(parsed)
	useFallbackOnly := projectID == ""

	if !useFallbackOnly && f.client == nil {
		useFallbackOnly = true
	}

	if !useFallbackOnly {
		value, source, fetchErr := f.fetchRemote(ctx, projectID, parsed.Secret, version)
		if fetchErr == nil {
			f.storeCache(key, value, parsed.Canonical, version, source)
			f.recordLatency(ctx, time.Since(start), source, nil)
			return value, nil
		}

		if !isFallbackError(fetchErr) {
			f.recordLatency(ctx, time.Since(start), "error", fetchErr)
			return "", fmt.Errorf("secrets: fetch failed for %s: %w", parsed.Canonical, fetchErr)
		}

		f.logger.Debug("secrets: falling back to local secrets", zap.String("ref", parsed.Canonical), zap.Error(fetchErr))
	}

	value, ok := f.lookupFallback(parsed, version)
	if !ok {
		err := fmt.Errorf("secrets: fallback value not found for %s", parsed.Canonical)
		f.recordLatency(ctx, time.Since(start), "error", err)
		return "", err
	}

	f.storeCache(key, value, parsed.Canonical, version, "fallback")
	f.recordLatency(ctx, time.Since(start), "fallback", nil)
	return value, nil
}

// Invalidate clears cached values for the supplied reference and notifies subscribers.
func (f *Fetcher) Invalidate(ref string) {
	parsed, err := parseReference(ref)
	if err != nil {
		return
	}

	f.mu.Lock()
	for key, entry := range f.cache {
		if entry.canonical == parsed.Canonical {
			delete(f.cache, key)
		}
	}
	watchers := f.watchers[parsed.Canonical]
	f.mu.Unlock()

	notifyWatchers(parsed.Canonical, watchers)
}

// Subscribe registers a watcher that receives notifications when the secret invalidates.
func (f *Fetcher) Subscribe(ref string) (<-chan struct{}, func()) {
	parsed, err := parseReference(ref)
	if err != nil {
		ch := make(chan struct{})
		close(ch)
		return ch, func() {}
	}

	ch := make(chan struct{}, 1)

	f.mu.Lock()
	f.watchers[parsed.Canonical] = append(f.watchers[parsed.Canonical], ch)
	f.mu.Unlock()

	cancel := func() {
		f.mu.Lock()
		defer f.mu.Unlock()
		watchers := f.watchers[parsed.Canonical]
		for i, watcher := range watchers {
			if watcher == ch {
				watchers = append(watchers[:i], watchers[i+1:]...)
				break
			}
		}
		if len(watchers) == 0 {
			delete(f.watchers, parsed.Canonical)
		} else {
			f.watchers[parsed.Canonical] = watchers
		}
	}

	return ch, cancel
}

// Notify simulates an external rotation event, invalidating cache and notifying subscribers.
func (f *Fetcher) Notify(ref string) {
	f.Invalidate(ref)
}

func (f *Fetcher) lookupCache(key string) (string, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	entry, ok := f.cache[key]
	if !ok {
		return "", false
	}
	return entry.value, true
}

func (f *Fetcher) storeCache(key, value, canonical, version, source string) {
	f.mu.Lock()
	f.cache[key] = cacheEntry{
		value:     value,
		canonical: canonical,
		version:   version,
		fetchedAt: time.Now(),
		source:    source,
	}
	f.mu.Unlock()
}

func (f *Fetcher) fetchRemote(ctx context.Context, projectID, secretName, version string) (string, string, error) {
	if f.client == nil {
		return "", "remote", errors.New("secrets: secret manager client not configured")
	}

	resourceName := fmt.Sprintf("projects/%s/secrets/%s/versions/%s", projectID, secretName, version)
	resp, err := f.client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{Name: resourceName})
	if err != nil {
		return "", "remote", err
	}
	if resp == nil || resp.Payload == nil {
		return "", "remote", fmt.Errorf("secret manager returned empty payload for %s", resourceName)
	}
	value := string(resp.Payload.GetData())
	return value, "remote", nil
}

func (f *Fetcher) projectID(ref parsedReference) string {
	if ref.ProjectOverride != "" {
		return ref.ProjectOverride
	}
	if id, ok := f.projectMap[f.env]; ok && strings.TrimSpace(id) != "" {
		return strings.TrimSpace(id)
	}
	return strings.TrimSpace(f.defaultProjID)
}

func (f *Fetcher) selectVersion(ref parsedReference) string {
	if ref.Version != "" {
		return ref.Version
	}

	if pin, ok := f.versionPins[keyWithEnv(f.env, ref.Canonical)]; ok && strings.TrimSpace(pin) != "" {
		return strings.TrimSpace(pin)
	}
	if pin, ok := f.versionPins[ref.Canonical]; ok && strings.TrimSpace(pin) != "" {
		return strings.TrimSpace(pin)
	}

	return "latest"
}

func (f *Fetcher) lookupFallback(ref parsedReference, version string) (string, bool) {
	f.loadFallback()

	if f.fallbackErr != nil {
		f.logger.Debug("secrets: fallback load error", zap.Error(f.fallbackErr))
		return "", false
	}

	key := cacheKey(ref.Canonical, version)
	if val, ok := f.fallbackVals[key]; ok {
		return val, true
	}
	if val, ok := f.fallbackVals[ref.Canonical]; ok {
		return val, true
	}
	return "", false
}

func (f *Fetcher) loadFallback() {
	f.fallbackOnce.Do(func() {
		path := strings.TrimSpace(f.fallbackPath)
		if path == "" {
			f.fallbackVals = map[string]string{}
			return
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path
		}

		file, err := os.Open(absPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				f.fallbackVals = map[string]string{}
				return
			}
			f.fallbackErr = fmt.Errorf("secrets: unable to open fallback file %s: %w", absPath, err)
			f.fallbackVals = map[string]string{}
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		values := make(map[string]string)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if key == "" {
				continue
			}
			normalizedKey := canonicalFallbackKey(key)
			if parsed, err := parseReference(normalizedKey); err == nil {
				canonical := parsed.Canonical
				version := parsed.Version
				if version == "" {
					version = "latest"
				}
				values[canonical] = value
				values[cacheKey(canonical, version)] = value
			} else {
				values[normalizedKey] = value
			}
		}
		if err := scanner.Err(); err != nil {
			f.fallbackErr = fmt.Errorf("secrets: failed reading %s: %w", absPath, err)
		}
		f.fallbackVals = values
	})
}

func (f *Fetcher) recordLatency(ctx context.Context, d time.Duration, source string, err error) {
	if !f.latencyEnabled {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("source", source),
	}
	if err != nil {
		attrs = append(attrs, attribute.String("error", err.Error()))
	}
	f.latency.Record(ctx, float64(d)/float64(time.Millisecond), metric.WithAttributes(attrs...))
}

func (f *Fetcher) recordCacheHit(ctx context.Context, ref parsedReference) {
	refAttr := attribute.String("secret", maskReference(ref.Canonical))
	if !f.cacheHitsEnabled {
		return
	}
	f.cacheHits.Add(ctx, 1, metric.WithAttributes(refAttr))
}

func notifyWatchers(canonical string, watchers []chan struct{}) {
	for _, ch := range watchers {
		if ch == nil {
			continue
		}
		func() {
			defer func() {
				_ = recover()
			}()
			select {
			case ch <- struct{}{}:
			default:
			}
		}()
	}
}

func closeSafe(ch chan struct{}) {
	defer func() {
		_ = recover()
	}()
	close(ch)
}

type parsedReference struct {
	Raw             string
	Canonical       string
	Secret          string
	Version         string
	ProjectOverride string
}

func parseReference(ref string) (parsedReference, error) {
	if strings.TrimSpace(ref) == "" {
		return parsedReference{}, errors.New("secrets: empty reference")
	}
	u, err := url.Parse(ref)
	if err != nil {
		return parsedReference{}, fmt.Errorf("secrets: invalid reference %q: %w", ref, err)
	}
	if u.Scheme != "secret" {
		return parsedReference{}, fmt.Errorf("secrets: unsupported scheme %q", u.Scheme)
	}
	secret := strings.Trim(strings.TrimPrefix(u.Host+u.Path, "/"), "/")
	if secret == "" {
		return parsedReference{}, fmt.Errorf("secrets: missing secret name in %q", ref)
	}

	canonical := *u
	canonical.RawQuery = ""
	canonical.Fragment = ""

	values := u.Query()
	version := strings.TrimSpace(values.Get("version"))
	project := strings.TrimSpace(values.Get("project"))

	return parsedReference{
		Raw:             ref,
		Canonical:       canonical.String(),
		Secret:          secret,
		Version:         version,
		ProjectOverride: project,
	}, nil
}

func cacheKey(canonical, version string) string {
	return canonical + "#" + version
}

func keyWithEnv(env, canonical string) string {
	if strings.TrimSpace(env) == "" {
		return canonical
	}
	return env + ":" + canonical
}

func copyStringMap(src map[string]string) map[string]string {
	if src == nil {
		return map[string]string{}
	}
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func maskReference(ref string) string {
	h := sha256.Sum256([]byte(ref))
	return hex.EncodeToString(h[:8])
}

func isFallbackError(err error) bool {
	if err == nil {
		return false
	}
	switch status.Code(err) {
	case codes.PermissionDenied, codes.Unauthenticated, codes.Unavailable, codes.DeadlineExceeded:
		return true
	default:
		return false
	}
}

func canonicalFallbackKey(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "sm://") {
		return "secret://" + strings.TrimPrefix(trimmed, "sm://")
	}
	return trimmed
}
