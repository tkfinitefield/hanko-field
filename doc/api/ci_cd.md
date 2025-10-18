# API CI/CD Pipeline

The API pipeline is implemented in `.github/workflows/api.yml` and split across three jobs: `checks`, `build-and-deploy-staging`, and `promote-production`. The workflow governs pull-request validation, automatic staging deployment on `main`, and a gated promotion to production.

## Pull Request Validation (`checks`)
- Trigger: any PR that touches `api/**` or the workflow itself.
- Tool bootstrap: runs `make deps` to install `gofumpt`, `golangci-lint`, `staticcheck`, `govulncheck`, and `gocovmerge` into `api/bin` with module caching via `actions/setup-go`.
- Formatting & linting: `make fmt-check` reports unformatted files without mutating the working tree. `make lint` runs both golangci-lint and staticcheck.
- Unit tests: `go test ./... -coverprofile=coverage.unit.out -covermode=atomic`.
- Firestore emulator integration tests: launches the Cloud SDK Firestore emulator in Docker, sets `FIRESTORE_EMULATOR_HOST=127.0.0.1:8080`, and re-runs `go test` with the `integration` build tag (`coverage.integration.out`).
- Coverage: merges unit + integration coverage with `./bin/gocovmerge` and uploads `api/coverage.out` as an artifact for PR review or Codecov ingestion.

## Main Branch Deploy (`build-and-deploy-staging`)
- Trigger: push to `main` touching `api/**` or the workflow.
- Auth: `google-github-actions/auth@v2` and `setup-gcloud@v2` use the `GCP_SA_KEY` service account JSON secret.
- Build & publish: builds the multi-stage `api/Dockerfile` and tags the image as `${GAR_LOCATION}-docker.pkg.dev/${GCP_PROJECT_ID}/${GAR_REPOSITORY}/api:${GITHUB_SHA}`. Artifact Registry auth is configured via `gcloud auth configure-docker`.
- Metadata capture: records the current Cloud Run revision (for rollback) and emits `deployment-metadata.json` alongside `previous_revision.txt` as workflow artifacts.
- Deployment: `gcloud run deploy` updates the staging service with labels `commit=${GITHUB_SHA}, env=staging` and the `staging` traffic tag. Environment variables include `GOOGLE_CLOUD_PROJECT` for runtime configuration.
- Smoke testing: hits `${STAGING_SMOKE_URL or service URL}/healthz` up to five times. Failure fails the job and blocks promotion.
- Notification: optional Slack webhook (`SLACK_WEBHOOK_URL`) posts a deployment summary when configured.

## Production Promotion (`promote-production`)
- Trigger: runs after staging as part of the same `main` pipeline. It uses the GitHub `production` environment so deployment pauses until a reviewer approves the promotion.
- Deployment: reuses the staged image URI (`needs.build-and-deploy-staging.outputs.image-uri`) and deploys via `gcloud run deploy ... --traffic=100` with production labels.
- Smoke testing & notification: identical health check at `${PRODUCTION_SMOKE_URL or service URL}/healthz` and optional Slack notification.
- Rollback context: captures the previous production revision and uploads it as an artifact for quick rollback.

## Required Secrets & Environments
| Secret | Purpose |
| --- | --- |
| `GCP_PROJECT_ID` | Google Cloud project hosting Cloud Run and Artifact Registry. |
| `GAR_LOCATION` | Artifact Registry location prefix (e.g. `asia-northeast1`). |
| `GAR_REPOSITORY` | Artifact Registry repository name for API images. |
| `CLOUD_RUN_REGION` | Cloud Run region (e.g. `asia-northeast1`). |
| `CLOUD_RUN_SERVICE` | Cloud Run service name (staging and production share logical name with tags). |
| `GCP_SA_KEY` | Service account JSON with Artifact Registry + Cloud Run permissions (deploy, read revisions). |
| `SLACK_WEBHOOK_URL` (optional) | Slack incoming webhook for deployment status notifications. |
| `STAGING_SMOKE_URL` (optional) | Override URL for staging smoke test if service URL differs. |
| `PRODUCTION_SMOKE_URL` (optional) | Override URL for production smoke test. |

Recommended GitHub environment protection rules:
- `staging` (no approval required) to populate runtime variables or secrets.
- `production` requiring at least one approval before `promote-production` executes, fulfilling the manual promotion gate.

## Rollback Procedure
1. Download the `api-staging-deploy-*` or `api-production-deploy-*` artifact from the failing workflow run to retrieve `previous_revision.txt`.
2. Perform rollback:
   ```bash
   gcloud run services update-traffic ${CLOUD_RUN_SERVICE} \
     --platform=managed \
     --region=${CLOUD_RUN_REGION} \
     --project=${GCP_PROJECT_ID} \
     --to-revisions=${PREVIOUS_REVISION}=100
   ```
3. Re-run the smoke test endpoint to confirm recovery, and post status in Slack / incident channel.

## Local CI Helpers
- `make fmt-check`, `make lint`, `make test`, `make test-integration`, and `make cover` mirror pipeline steps.
- `make docker-build` / `make docker-run` build the same multi-stage image used in CI.
- Developers can reuse the Firestore emulator locally with Docker:
  ```bash
  docker run --rm -p 8080:8080 gcr.io/google.com/cloudsdktool/cloud-sdk:emulators \
    gcloud beta emulators firestore start --host-port=0.0.0.0:8080 --quiet
  FIRESTORE_EMULATOR_HOST=127.0.0.1:8080 GOOGLE_CLOUD_PROJECT=hanko-field-dev \
    go test -tags=integration ./...
  ```

## Promotion Notes
- Manual promotion happens through GitHub's environment approval UI. Approvers should review staging smoke results, Cloud Run logs, and Slack notifications before approving.
- If a production hotfix is needed without a new merge, re-run the latest successful `main` workflow and approve the paused `promote-production` job, or manually dispatch the workflow referencing the desired commit.
