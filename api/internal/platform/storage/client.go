package storage

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/hanko-field/api/internal/platform/auth"
)

const (
	defaultSignedURLExpiry     = 15 * time.Minute
	maxDownloadSignedURLExpiry = 15 * time.Minute
)

var (
	errNoSigner           = errors.New("storage: signer is required")
	errInvalidOptions     = errors.New("storage: either upload or download options must be provided")
	errBothIntents        = errors.New("storage: upload and download options cannot be used together")
	errInvalidBucket      = errors.New("storage: bucket name is required")
	errInvalidObject      = errors.New("storage: object name is required")
	errMethodNotAllowed   = errors.New("storage: HTTP method not allowed for intent")
	errContentTypeMissing = errors.New("storage: content type is required for uploads")
	errContentTypeDenied  = errors.New("storage: content type not allowed")
	errMD5Required        = errors.New("storage: content MD5 is required for uploads")
	errMD5Invalid         = errors.New("storage: content MD5 must be base64 encoded")
	errExpiryTooLong      = errors.New("storage: expiry exceeds permitted maximum")
)

// Client generates signed URLs backed by a Signer.
type Client struct {
	signer Signer
	scheme storage.SigningScheme
	now    func() time.Time
}

// ClientOption customises client behaviour.
type ClientOption func(*Client)

// WithSigningScheme overrides the signing scheme (defaults to V4).
func WithSigningScheme(scheme storage.SigningScheme) ClientOption {
	return func(c *Client) {
		if scheme != 0 {
			c.scheme = scheme
		}
	}
}

// WithClock injects a custom clock (useful for tests).
func WithClock(clock func() time.Time) ClientOption {
	return func(c *Client) {
		if clock != nil {
			c.now = clock
		}
	}
}

// NewClient constructs a new storage signed URL client.
func NewClient(signer Signer, opts ...ClientOption) (*Client, error) {
	if signer == nil || strings.TrimSpace(signer.Email()) == "" {
		return nil, errNoSigner
	}

	client := &Client{
		signer: signer,
		scheme: storage.SigningSchemeV4,
		now:    time.Now,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(client)
		}
	}
	return client, nil
}

// SignedURLOptions capture configuration for upload or download signed URLs.
type SignedURLOptions struct {
	Upload   *UploadOptions
	Download *DownloadOptions
	Query    map[string]string
}

// UploadOptions control upload-specific validation.
type UploadOptions struct {
	Method              string
	ContentType         string
	ContentMD5          string
	RequireMD5          bool
	AllowedMethods      []string
	AllowedContentTypes []string
	MaxSize             int64
	ExpiresIn           time.Duration
	AdditionalHeaders   map[string]string
}

// DownloadOptions control download-specific validation and response behaviour.
type DownloadOptions struct {
	Method         string
	ExpiresIn      time.Duration
	Disposition    string
	CacheControl   string
	ResponseType   string
	OwnerID        string
	Identity       *auth.Identity
	AllowAnonymous bool
}

// SignedURLResult describes the generated signed URL details.
type SignedURLResult struct {
	URL       string
	Method    string
	ExpiresAt time.Time
	Headers   map[string]string
}

// SignedURL creates a signed URL according to the provided options.
func (c *Client) SignedURL(ctx context.Context, bucket, object string, opts SignedURLOptions) (SignedURLResult, error) {
	if c == nil {
		return SignedURLResult{}, errNoSigner
	}
	if ctx == nil {
		return SignedURLResult{}, errors.New("storage: context is required")
	}
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return SignedURLResult{}, errInvalidBucket
	}
	object = strings.TrimSpace(object)
	if object == "" {
		return SignedURLResult{}, errInvalidObject
	}
	if opts.Upload == nil && opts.Download == nil {
		return SignedURLResult{}, errInvalidOptions
	}
	if opts.Upload != nil && opts.Download != nil {
		return SignedURLResult{}, errBothIntents
	}

	googleAccessID := c.signer.Email()
	if googleAccessID == "" {
		return SignedURLResult{}, errNoSigner
	}

	var (
		result    SignedURLResult
		urlOpts   = storage.SignedURLOptions{GoogleAccessID: googleAccessID, Scheme: c.scheme}
		headers   = make(map[string]string)
		extHeader []string
		expiry    time.Duration
	)

	if opts.Upload != nil {
		method, err := normaliseUploadMethod(opts.Upload.Method)
		if err != nil {
			return SignedURLResult{}, err
		}

		contentType := strings.TrimSpace(opts.Upload.ContentType)
		if contentType == "" {
			return SignedURLResult{}, errContentTypeMissing
		}
		if len(opts.Upload.AllowedContentTypes) > 0 && !contentTypeAllowed(contentType, opts.Upload.AllowedContentTypes) {
			return SignedURLResult{}, errContentTypeDenied
		}

		expiry = opts.Upload.ExpiresIn
		if expiry <= 0 {
			expiry = defaultSignedURLExpiry
		}

		if opts.Upload.RequireMD5 && strings.TrimSpace(opts.Upload.ContentMD5) == "" {
			return SignedURLResult{}, errMD5Required
		}
		if strings.TrimSpace(opts.Upload.ContentMD5) != "" {
			if _, err := base64.StdEncoding.DecodeString(opts.Upload.ContentMD5); err != nil {
				return SignedURLResult{}, errMD5Invalid
			}
		}

		urlOpts.Method = method
		urlOpts.ContentType = contentType
		urlOpts.MD5 = strings.TrimSpace(opts.Upload.ContentMD5)

		headers["Content-Type"] = contentType
		if urlOpts.MD5 != "" {
			headers["Content-MD5"] = urlOpts.MD5
		}

		if opts.Upload.MaxSize > 0 {
			sizeHeader := fmt.Sprintf("0,%d", opts.Upload.MaxSize)
			headerLine := fmt.Sprintf("x-goog-content-length-range:%s", sizeHeader)
			extHeader = append(extHeader, headerLine)
			headers["x-goog-content-length-range"] = sizeHeader
		}

		if len(opts.Upload.AdditionalHeaders) > 0 {
			keys := make([]string, 0, len(opts.Upload.AdditionalHeaders))
			for k := range opts.Upload.AdditionalHeaders {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, key := range keys {
				value := strings.TrimSpace(opts.Upload.AdditionalHeaders[key])
				if value == "" {
					continue
				}
				canonical := strings.ToLower(strings.TrimSpace(key))
				extHeader = append(extHeader, fmt.Sprintf("%s:%s", canonical, value))
				headers[key] = value
			}
		}

		expiryTime := c.now().Add(expiry)
		urlOpts.Expires = expiryTime
		urlOpts.Headers = extHeader
		urlOpts.SignBytes = func(payload []byte) ([]byte, error) {
			return c.signer.SignBytes(ctx, payload)
		}
		if len(opts.Query) > 0 {
			urlOpts.QueryParameters = mapToURLValues(opts.Query)
		}

		signedURL, err := storage.SignedURL(bucket, object, &urlOpts)
		if err != nil {
			return SignedURLResult{}, fmt.Errorf("storage: sign upload url: %w", err)
		}

		result.URL = signedURL
		result.Method = method
		result.ExpiresAt = expiryTime
		result.Headers = headers
		return result, nil
	}

	download := opts.Download
	method := strings.ToUpper(strings.TrimSpace(download.Method))
	if method == "" {
		method = httpMethodGet
	}
	if method != httpMethodGet && method != httpMethodHead {
		return SignedURLResult{}, errMethodNotAllowed
	}

	expiry = download.ExpiresIn
	if expiry <= 0 {
		expiry = 5 * time.Minute
	}
	if expiry > maxDownloadSignedURLExpiry {
		return SignedURLResult{}, errExpiryTooLong
	}

	if err := AuthorizeDownload(download.Identity, download.OwnerID, download.AllowAnonymous); err != nil {
		return SignedURLResult{}, err
	}

	urlOpts.Method = method
	urlOpts.SignBytes = func(payload []byte) ([]byte, error) {
		return c.signer.SignBytes(ctx, payload)
	}

	queryValues := map[string]string{}
	if download.Disposition != "" {
		queryValues["response-content-disposition"] = download.Disposition
	}
	if download.CacheControl != "" {
		queryValues["response-cache-control"] = download.CacheControl
	}
	if download.ResponseType != "" {
		queryValues["response-content-type"] = download.ResponseType
	}
	for key, value := range opts.Query {
		if _, exists := queryValues[key]; exists {
			continue
		}
		queryValues[key] = value
	}
	if len(queryValues) > 0 {
		urlOpts.QueryParameters = mapToURLValues(queryValues)
	}

	expiryTime := c.now().Add(expiry)
	urlOpts.Expires = expiryTime

	signedURL, err := storage.SignedURL(bucket, object, &urlOpts)
	if err != nil {
		return SignedURLResult{}, fmt.Errorf("storage: sign download url: %w", err)
	}

	result.URL = signedURL
	result.Method = method
	result.ExpiresAt = expiryTime
	if len(headers) > 0 {
		result.Headers = headers
	}
	return result, nil
}

const (
	httpMethodPut  = "PUT"
	httpMethodPost = "POST"
	httpMethodGet  = "GET"
	httpMethodHead = "HEAD"
)

func normaliseUploadMethod(method string) (string, error) {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = httpMethodPut
	}
	switch method {
	case httpMethodPut, httpMethodPost:
		return method, nil
	default:
		return "", errMethodNotAllowed
	}
}

func contentTypeAllowed(contentType string, allowed []string) bool {
	normalized := strings.ToLower(strings.TrimSpace(contentType))
	for _, candidate := range allowed {
		candidate = strings.ToLower(strings.TrimSpace(candidate))
		if candidate == "" {
			continue
		}
		if candidate == "*" {
			return true
		}
		if strings.HasSuffix(candidate, "/*") {
			prefix := strings.TrimSuffix(candidate, "*")
			if strings.HasPrefix(normalized, strings.TrimSuffix(prefix, "/")) {
				return true
			}
			continue
		}
		if normalized == candidate {
			return true
		}
	}
	return false
}

func mapToURLValues(values map[string]string) url.Values {
	out := make(url.Values, len(values))
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := values[key]
		out.Add(key, value)
	}
	return out
}
