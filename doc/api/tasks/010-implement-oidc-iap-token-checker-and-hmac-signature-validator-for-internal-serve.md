# Implement OIDC/IAP token checker and HMAC signature validator for internal/server-to-server and webhook endpoints.

**Parent Section:** 2. Core Platform Services
**Task ID:** 010

## Goal
Protect internal and webhook endpoints via Google-signed IAP/OIDC tokens or HMAC signatures per integration.

## Design
- Provide middleware `RequireOIDC(audience, issuers)` verifying Google-signed JWT (audience, issuer, expiry) with JWKS caching.
- Provide middleware `RequireHMAC(secretName)` verifying `X-Signature` built from canonical request string + timestamp, with replay protection using nonce store.
- Apply OIDC to `/internal/*`; apply HMAC to `/webhooks/*` (Stripe, PayPal, shipping, AI) using provider-specific logic.

## Steps
1. Implement JWKS fetcher with background refresh and environment-specific audiences.
2. Create canonical string builder for HMAC (method, path, body, timestamp) and store used nonces in Firestore/Redis for 5 minutes.
3. Add metrics/logging for verification outcomes.
4. Provide documentation for partner services on header expectations.
