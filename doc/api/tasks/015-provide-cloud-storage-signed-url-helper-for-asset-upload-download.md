# Provide Cloud Storage signed URL helper for asset upload/download.

**Parent Section:** 2. Core Platform Services
**Task ID:** 015

## Goal
Generate signed upload and download URLs for assets while enforcing security policies.

## Design
- Package `internal/platform/storage` wrapping GCS client with helper `SignedURL(ctx, bucket, object, opts)`.
- Upload options: allowed methods (PUT/POST), content type whitelist, maximum size, MD5 check.
- Download options: short-lived (<=15 min), response headers for caching, optional response disposition.

## Steps
1. Implement signer using service account credentials; support emulator (fake signer) for tests.
2. Define path builders per asset type (`designs/{{designId}}/source.png`, `invoices/{{orderId}}.pdf`).
3. Validate caller permissions (ownership, staff role) before generating download URLs.
4. Write unit tests verifying signature generation and rejection paths.
