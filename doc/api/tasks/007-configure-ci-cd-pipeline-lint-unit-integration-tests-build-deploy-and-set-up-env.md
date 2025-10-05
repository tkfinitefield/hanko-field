# Configure CI/CD pipeline (lint, unit/integration tests, build, deploy) and set up environment promotion workflow.

**Parent Section:** 1. Project & Environment Setup
**Task ID:** 007

## Goal
Automate linting, testing, building, and deployment of the API with promotion gates and rollback strategy.

## Pipeline Outline
- Pull Request: run `make fmt-check`, `golangci-lint`, `go test ./...`, and Firestore emulator tests; publish coverage.
- Merge to main: build container image, push to Artifact Registry, run integration tests, deploy to staging Cloud Run.
- Promotion: manual approval to deploy to production, run smoke tests post-deploy, notify Slack channel.
- Rollback: stored previous revision and scripted `gcloud run services update --revision` command.

## Steps
1. Implement CI workflows (e.g., GitHub Actions) under `.github/workflows/api.yml` with reusable commands.
2. Configure caching for Go modules and build artifacts to keep pipelines fast.
3. Inject required secrets (service account JSON, signing keys) securely via CI secret store.
4. Implement CD job using `gcloud run deploy` or Cloud Deploy; include canary or traffic splitting if required.
5. Document pipeline in `doc/api/ci_cd.md` including promotion flow and rollback instructions.

## Acceptance Criteria
- PR pipeline blocks merge on lint/test failures.
- Staging deployment occurs automatically after main branch merge.
- Production deploy requires approval and logs deployment metadata.
- Post-deploy smoke tests confirm health endpoints before marking job success.

---

## CI/CD Implementation Summary (2025-10-05)

- GitHub Actions workflow `.github/workflows/api.yml` now covers PR validation (`checks`), staging deploy (`build-and-deploy-staging`), and gated production promotion (`promote-production`).
- PR validation runs `make fmt-check`, `make lint`, full unit tests with coverage, spins up a Firestore emulator in Docker for `-tags=integration` tests, and uploads the merged coverage profile.
- Main branch deployments build the multi-stage `api/Dockerfile`, push to Artifact Registry (`${GAR_LOCATION}-docker.pkg.dev/${GCP_PROJECT_ID}/${GAR_REPOSITORY}/api:${GITHUB_SHA}`), capture the prior revision, deploy to Cloud Run staging, execute smoke tests, and emit metadata + Slack notifications when configured.
- Production promotion reuses the staged image after GitHub environment approval, deploys with 100% traffic, runs health checks, and stores rollback revision info.
- `doc/api/ci_cd.md` documents workflow details, secrets (`GCP_PROJECT_ID`, `GAR_LOCATION`, etc.), promotion approvals, smoke tests, and rollback via `gcloud run services update-traffic`.
- `api/Makefile`, `api/Taskfile.yml`, `.dockerignore`, and `api/Dockerfile` were updated to mirror pipeline commands and support local integration test + container workflows.

## Follow-ups
- [ ] Configure GitHub `staging` and `production` environments with required secrets and (for production) reviewer approvals.
- [ ] Seed GitHub secrets (`GCP_PROJECT_ID`, `GAR_LOCATION`, `GAR_REPOSITORY`, `CLOUD_RUN_REGION`, `CLOUD_RUN_SERVICE`, `GCP_SA_KEY`, optional smoke URLs / Slack).
- [ ] Populate integration tests under the `integration` build tag to exercise Firestore emulator paths as implementation progresses.
