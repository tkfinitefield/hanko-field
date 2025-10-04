# Configure build/release pipeline for Cloud Run deployment (Dockerfile, cloudbuild.yaml, env configs).

**Parent Section:** 1. Project Setup & Tooling
**Task ID:** 007

## Goal
Configure build and release pipeline targeting Cloud Run.

## Steps
1. Create Dockerfile optimized for Go binary + static assets; ensure minimal image.
2. Set up Cloud Build or GitHub Actions workflow for build/test/deploy.
3. Configure environment variables/secrets (API base URL, Firebase config) via Secret Manager.
4. Document deployment commands and environment promotion flow.
