# API Configuration Reference

The API service reads configuration through `internal/platform/config.Load`. Values resolve in the following order:

1. Built-in defaults
2. Key/value pairs from a `.env` file (default `./.env`)
3. Environment variables (`API_*`)
4. Secret references (`secret://...`) resolved via the configured Secret Manager client

Missing required values cause the loader to return a `ValidationError` containing the field names.

## Environment Variables

| Key | Default | Required | Description |
| --- | --- | --- | --- |
| `API_SERVER_PORT` | `8080` | No | TCP port for the HTTP server. |
| `API_SERVER_READ_TIMEOUT` | `15s` | No | Maximum duration for reading the request body. |
| `API_SERVER_WRITE_TIMEOUT` | `30s` | No | Maximum duration for writing responses. |
| `API_SERVER_IDLE_TIMEOUT` | `120s` | No | Keep-alive timeout for idle connections. |
| `API_FIREBASE_PROJECT_ID` | _none_ | **Yes** | Firebase project identifier. |
| `API_FIREBASE_CREDENTIALS_FILE` | _empty_ | No | Path to service account credentials for local development. |
| `API_FIRESTORE_PROJECT_ID` | defaults to Firebase project | No | Firestore project override; use when reading from a different project. |
| `API_FIRESTORE_EMULATOR_HOST` | _empty_ | No | Host for Firestore emulator (e.g. `localhost:8081`). |
| `API_STORAGE_ASSETS_BUCKET` | _none_ | **Yes** | GCS bucket for assets (design uploads, previews). |
| `API_STORAGE_LOGS_BUCKET` | _empty_ | No | Optional bucket for exported logs. |
| `API_STORAGE_EXPORTS_BUCKET` | _empty_ | No | Optional bucket for scheduled exports. |
| `API_PSP_STRIPE_API_KEY` | _empty_ | No | Stripe secret key or `secret://` reference. |
| `API_PSP_STRIPE_WEBHOOK_SECRET` | _empty_ | No | Stripe webhook signing secret or `secret://` reference. |
| `API_PSP_PAYPAL_CLIENT_ID` | _empty_ | No | PayPal client identifier. |
| `API_PSP_PAYPAL_SECRET` | _empty_ | No | PayPal client secret or `secret://` reference. |
| `API_AI_SUGGESTION_ENDPOINT` | _empty_ | No | Base URL for the AI suggestion worker. |
| `API_AI_AUTH_TOKEN` | _empty_ | No | Token for authenticating with AI workers; supports `secret://`. |
| `API_WEBHOOK_SIGNING_SECRET` | _empty_ | No | Shared secret for verifying inbound webhooks (`secret://` supported). |
| `API_WEBHOOK_ALLOWED_HOSTS` | _empty_ | No | Comma-separated allowlist for webhook source hosts. |
| `API_RATELIMIT_DEFAULT_PER_MIN` | `120` | No | Anonymous requests per minute. |
| `API_RATELIMIT_AUTH_PER_MIN` | `240` | No | Authenticated requests per minute. |
| `API_RATELIMIT_WEBHOOK_BURST` | `60` | No | Burst allowance for webhook endpoints. |
| `API_FEATURE_AISUGGESTIONS` | `false` | No | Enable AI suggestion features. |
| `API_FEATURE_PROMOTIONS` | `true` | No | Enable promotions-related flows. |
| `API_SECURITY_ENVIRONMENT` | `local` | No | Environment label (e.g., `dev`, `stg`, `prod`) used to select audience defaults. |
| `API_SECURITY_OIDC_JWKS_URL` | `https://www.googleapis.com/oauth2/v3/certs` | No | JWKS endpoint for verifying Google-signed OIDC/IAP tokens. |
| `API_SECURITY_OIDC_AUDIENCE` | _empty_ | No | Audience expected for OIDC tokens in the current environment. |
| `API_SECURITY_OIDC_AUDIENCES` | _empty_ | No | Comma-separated map (`dev=aud,stg=aud`) supplying per-environment audiences. |
| `API_SECURITY_OIDC_ISSUERS` | `https://accounts.google.com, https://cloud.google.com/iap` | No | Allowed token issuers for internal authentication. |
| `API_SECURITY_HMAC_SECRETS` | _empty_ | No | Comma-separated map (`payments/stripe=secret,shipping=secret`) resolving webhook HMAC secrets; supports `secret://` references. |
| `API_SECRET_PROJECT_IDS` | _empty_ | No | Comma-separated map (`local=project-dev,prod=project-prod`) mapping environments to Secret Manager project IDs. |
| `API_SECRET_DEFAULT_PROJECT_ID` | _empty_ | No | Fallback project ID used when no environment-specific mapping is supplied. |
| `API_SECRET_VERSION_PINS` | _empty_ | No | Comma-separated map (`secret://stripe/api=5`) forcing specific secret versions instead of `latest`. |
| `API_SECRET_FALLBACK_FILE` | `.secrets.local` | No | Path to local fallback secrets file used when Secret Manager is unavailable (developers only). |
| `API_SECURITY_HMAC_HEADER_SIGNATURE` | `X-Signature` | No | Header carrying the webhook HMAC signature. |
| `API_SECURITY_HMAC_HEADER_TIMESTAMP` | `X-Signature-Timestamp` | No | Header carrying the signature timestamp. |
| `API_SECURITY_HMAC_HEADER_NONCE` | `X-Signature-Nonce` | No | Header carrying the nonce used for replay protection. |
| `API_SECURITY_HMAC_CLOCK_SKEW` | `5m` | No | Maximum allowed difference between the request timestamp and server time. |
| `API_SECURITY_HMAC_NONCE_TTL` | `5m` | No | Duration to retain used nonces to detect replays. |

## Secret References

Any value beginning with `secret://` (or the legacy `sm://` form) is treated as a Secret Manager reference. Provide a `SecretResolver` when calling `config.Load` (e.g. via DI wiring) to fetch secrets from Google Secret Manager. When the resolver is not configured, the loader returns a `SecretError` to prevent accidental plaintext usage.

```
cfg, err := config.Load(ctx, config.WithSecretResolver(secretClient))
```

For local development, secrets can be resolved by a custom resolver that maps `secret://` identifiers to `.env` overrides or fake values.

## Secret Manager Fetcher

- The API service resolves secrets through `internal/platform/secrets.Fetcher`, which understands `secret://` URIs, caches resolved values, and exposes OpenTelemetry metrics `secrets.fetch.latency` and `secrets.fetch.cache_hits`.
- When Google Secret Manager cannot be reached (e.g. local development without ADC), the fetcher consults `.secrets.local` (override with `API_SECRET_FALLBACK_FILE`). Entries use simple `key=value` format, where keys may include query parameters to pin versions: `secret://stripe/api=sk_test_...`.
- Configure project routing with `API_SECRET_PROJECT_IDS` and `API_SECRET_DEFAULT_PROJECT_ID`; version overrides are supplied through `API_SECRET_VERSION_PINS`.
- The DI wiring enforces critical secrets via `config.WithRequiredSecrets` and aborts startup when they are missing (`config.WithPanicOnMissingSecrets`). Missing secret names are hashed before logging.

### Rotation Playbook

1. Add a new version to the target secret in Google Secret Manager and keep the current version enabled during rollout.
2. If you need immediate invalidation of cached values, publish a Pub/Sub message that identifies the rotated secret (for example, `{ "ref": "secret://stripe/api" }`) to the rotation topic.
3. Disable or destroy the previous version after confirming that dependent services read the new value (the fetcher automatically requests `latest` unless a version pin is configured).
4. No application redeploy is required; caches are repopulated on-demand and metrics expose latency anomalies during rotation.

### Pub/Sub Hot Reload

- Create a Pub/Sub topic (e.g. `secrets-rotation`) with a push subscription to the API service. The handler should call `Fetcher.Notify(ref)` with the secret reference so cached values are invalidated.
- This makes rotation proactive: publishing either manual messages or automation (Cloud Build / Terraform) triggers hot reload without waiting for cache expiry.
- When Pub/Sub is unavailable, operators can still trigger regeneration by calling an admin endpoint or forcing a process restart; caches refresh transparently.

## Dotenv Support

The loader attempts to read `./.env` (customisable with `WithEnvFile`) and merges values that are not already set in the environment. Example:

```
API_FIREBASE_PROJECT_ID=hanko-dev
API_STORAGE_ASSETS_BUCKET=hanko-dev-assets
API_SERVER_PORT=8081
```

## Integration

- `config.Load` returns a fully populated `config.Config` struct; pass the struct into the DI container (`di.NewContainer`).
- Required values: `Firebase.ProjectID`, `Firestore.ProjectID` (defaults to Firebase), and `Storage.AssetsBucket`.
- Secrets (`StripeAPIKey`, `StripeWebhookSecret`, `PayPalSecret`, `AI.AuthToken`, `Webhooks.SigningSecret`) are resolved via the injected resolver before validation.
- The default bootstrap path (`cmd/api/main.go`) registers the fetcher, enforces required secrets, and panics on start if any are missing.
