package firestore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/hanko-field/api/internal/platform/config"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultDialTimeout = 10 * time.Second
	envEmulatorHost    = "FIRESTORE_EMULATOR_HOST"
	envGoogleProjectID = "GOOGLE_CLOUD_PROJECT"
)

var ErrProviderClosed = errors.New("firestore: provider is closed")

type initResult struct {
	client *firestore.Client
	err    error
}

// Provider lazily initialises a shared Firestore client instance.
type Provider struct {
	cfg         config.FirestoreConfig
	dialTimeout time.Duration
	clientOpts  []option.ClientOption

	stateMu sync.Mutex
	initCh  chan initResult
	client  *firestore.Client

	closed atomic.Bool
}

// ProviderOption customises the Provider behaviour.
type ProviderOption func(*Provider)

// WithDialTimeout overrides the timeout used when creating the client.
func WithDialTimeout(timeout time.Duration) ProviderOption {
	return func(p *Provider) {
		if timeout > 0 {
			p.dialTimeout = timeout
		}
	}
}

// WithClientOptions appends client options applied during initialisation.
func WithClientOptions(opts ...option.ClientOption) ProviderOption {
	return func(p *Provider) {
		if len(opts) > 0 {
			p.clientOpts = append(p.clientOpts, opts...)
		}
	}
}

// NewProvider constructs a Provider using the supplied configuration.
func NewProvider(cfg config.FirestoreConfig, opts ...ProviderOption) *Provider {
	provider := &Provider{
		cfg:         cfg,
		dialTimeout: defaultDialTimeout,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(provider)
		}
	}
	return provider
}

// Client returns the lazily initialised Firestore client.
func (p *Provider) Client(ctx context.Context) (*firestore.Client, error) {
	if ctx == nil {
		return nil, errors.New("firestore: context is required")
	}

	for {
		if p.closed.Load() {
			return nil, ErrProviderClosed
		}

		p.stateMu.Lock()
		if p.client != nil {
			client := p.client
			p.stateMu.Unlock()
			return client, nil
		}
		if p.closed.Load() {
			p.stateMu.Unlock()
			return nil, ErrProviderClosed
		}
		if waitCh := p.initCh; waitCh != nil {
			p.stateMu.Unlock()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case res := <-waitCh:
				if res.err != nil {
					return nil, res.err
				}
				if p.closed.Load() {
					return nil, ErrProviderClosed
				}
				return res.client, nil
			}
		}

		waitCh := make(chan initResult, 1)
		p.initCh = waitCh
		p.stateMu.Unlock()

		client, err := p.createClient(ctx)

		p.stateMu.Lock()
		if err != nil {
			p.client = nil
			p.initCh = nil
			p.stateMu.Unlock()
			waitCh <- initResult{err: err}
			close(waitCh)
			return nil, err
		}
		p.client = client
		p.initCh = nil
		p.stateMu.Unlock()

		waitCh <- initResult{client: client}
		close(waitCh)

		if p.closed.Load() {
			return nil, ErrProviderClosed
		}
		return client, nil
	}
}

func (p *Provider) createClient(ctx context.Context) (*firestore.Client, error) {
	ctxWithTimeout := ctx
	var cancel context.CancelFunc
	if p.dialTimeout > 0 {
		ctxWithTimeout, cancel = context.WithTimeout(ctx, p.dialTimeout)
	}
	if cancel != nil {
		defer cancel()
	}

	projectID := strings.TrimSpace(p.cfg.ProjectID)
	if projectID == "" {
		projectID = strings.TrimSpace(os.Getenv(envGoogleProjectID))
	}
	if projectID == "" {
		return nil, errors.New("firestore: project id is required")
	}

	host := p.emulatorHost()
	opts := append([]option.ClientOption(nil), p.clientOpts...)
	if host != "" {
		if os.Getenv(envEmulatorHost) == "" {
			_ = os.Setenv(envEmulatorHost, host)
		}
		opts = append(opts,
			option.WithoutAuthentication(),
			option.WithEndpoint(host),
			option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		)
	}

	client, err := firestore.NewClient(ctxWithTimeout, projectID, opts...)
	if err != nil {
		return nil, fmt.Errorf("firestore: create client: %w", err)
	}
	return client, nil
}

// Close releases the underlying Firestore client. The Provider cannot be reused afterwards.
func (p *Provider) Close(ctx context.Context) error {
	if p == nil {
		return nil
	}
	if p.closed.Load() {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var client *firestore.Client

	for {
		if p.closed.Load() {
			return nil
		}

		p.stateMu.Lock()
		if p.closed.Load() {
			p.stateMu.Unlock()
			return nil
		}
		if waitCh := p.initCh; waitCh != nil {
			p.stateMu.Unlock()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-waitCh:
				continue
			}
		}

		p.closed.Store(true)
		client = p.client
		p.client = nil
		p.stateMu.Unlock()
		break
	}

	if client == nil {
		return nil
	}

	done := make(chan error, 1)
	go func() {
		done <- client.Close()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// RunTransaction executes fn inside a Firestore transaction using the provider's client.
func (p *Provider) RunTransaction(ctx context.Context, fn TxFunc, opts ...TxOption) error {
	client, err := p.Client(ctx)
	if err != nil {
		return err
	}
	return RunTransaction(ctx, client, fn, opts...)
}

func (p *Provider) emulatorHost() string {
	if trimmed := strings.TrimSpace(p.cfg.EmulatorHost); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(os.Getenv(envEmulatorHost))
}
