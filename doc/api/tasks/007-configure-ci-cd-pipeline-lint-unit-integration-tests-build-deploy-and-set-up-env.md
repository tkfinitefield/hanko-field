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
