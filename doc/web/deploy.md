# Web Deployment (Cloud Run)

This document outlines how to build and deploy the web service to Cloud Run using Docker and Cloud Build.

## Prerequisites
- Google Cloud project with Artifact Registry and Cloud Run enabled
- Cloud Build API enabled
- Service account with Artifact Registry and Cloud Run permissions

## Build the container locally (optional)
```bash
cd web
docker build -t web:local .
```

## Deploy via Cloud Build
`web/cloudbuild.yaml` builds and deploys the service. It produces and pushes an image to Artifact Registry and runs `gcloud run deploy`.

### Configure Cloud Build trigger
- Repo: this repository
- Directory: `web/`
- Trigger on: main branch (or as preferred)
- Substitutions (adjust as needed):
  - `_REGION` (default `asia-northeast1`)
  - `_SERVICE` (default `hanko-web`)
  - `_REPOSITORY` (Artifact Registry repository name for web images, default `web`)
  - `_AR_HOST` (e.g. `asia-northeast1-docker.pkg.dev`)

### Secrets (Secret Manager)
Create the following secrets or update substitutions in `cloudbuild.yaml` to match your names:
- `web-API_BASE_URL`
- `web-FIREBASE_API_KEY`
- `web-FIREBASE_AUTH_DOMAIN`
- `web-FIREBASE_PROJECT_ID`

Cloud Build deploy step uses `--set-secrets` to mount these into Cloud Run env vars.

## Runtime configuration
- Service listens on `HANKO_WEB_PORT` (fallback to Cloud Run `PORT`, defaults to `8080`).
- Template path and public assets are passed in the ENTRYPOINT flags.
- `HANKO_WEB_ENV=prod` is set during deployment by Cloud Build.

## Rollback
- Use Cloud Run revisions to roll back to a previous image/revision from the Cloud Console or via `gcloud run services update-traffic`.

## Notes
- The Dockerfile builds Tailwind CSS using the standalone binary in an intermediate stage; no Node.js is required.
- Distroless static runtime image is used for a small, secure final container.
