# Infrastructure Provisioning

Infrastructure for the API is managed via Terraform under `infra/terraform`. The configuration provisions:

- Cloud Run service (`api-service`) with configurable scaling, ingress, and VPC connector
- Firestore (native mode) with TTL on `stockReservations.expiresAt` and composite indexes for order queries
- Pub/Sub topics/subscriptions for AI jobs, webhook retries, and export automation
- Cloud Storage buckets (`design_assets`, `ai_suggestions`, `exports`, `invoices`) with retention/versioning policies
- Cloud Scheduler jobs that invoke internal maintenance endpoints
- Secret Manager secrets for PSP, AI, and webhook credentials
- Service accounts with least-privilege IAM bindings for runtime, scheduler, and worker roles

## Structure

```
infra/terraform/
├── backend.tf                # Remote state backend (update bucket before use)
├── main.tf                   # Root module wiring
├── provider.tf               # Google provider configuration
├── variables.tf              # Shared variables
├── modules/
│   ├── cloud_run_service/
│   ├── cloud_scheduler/
│   ├── firestore/
│   ├── pubsub/
│   ├── secret_manager/
│   ├── service_accounts/
│   └── storage_buckets/
└── envs/
    ├── dev/terraform.tfvars
    ├── stg/terraform.tfvars
    └── prod/terraform.tfvars
```

Each environment (`dev`, `stg`, `prod`) provides overrides for project IDs, images, scaling limits, and scheduler endpoints. The root module derives names following the pattern `hanko-field[-env]-resource` to keep resources grouped per environment.

## Usage

1. Update `backend.tf` with the Terraform state bucket created by the platform team.
2. Authenticate with Google Cloud (`gcloud auth application-default login`) and set the desired project (`gcloud config set project hanko-field-dev`).
3. Select the environment tfvars file:

   ```bash
   cd infra/terraform
   terraform init -backend-config="bucket=hanko-field-terraform-state" -backend-config="prefix=api/dev"
   terraform plan -var-file=envs/dev/terraform.tfvars
   terraform apply -var-file=envs/dev/terraform.tfvars
   ```

4. Promote changes to staging and production by switching the tfvars file. Enable `-var cloud_run_image` overrides if deploying a specific revision.
5. Populate Secret Manager secrets using CI or the console after the first apply:

   ```bash
   gcloud secrets versions add hanko-field-dev-stripe_api_key --data-file=secrets/dev/stripe.key
   ```

## Naming & Outputs

Terraform outputs the Cloud Run URL, bucket names, secret IDs, and service account emails—these feed into application configuration (see `doc/api/configuration.md`). Reference the outputs for CI pipelines and environment variable injection.

## Drift Detection

- Run `terraform plan` in CI nightly for each environment and alert on drift.
- Enable [Terraform Cloud/Google Cloud Storage] state locking to prevent concurrent applies.
- Audit IAM modifications regularly; service account bindings are managed exclusively through Terraform.

## Audit Log Retention & Export

- `/auditLogs` in Firestore is append-only; entries are hashed for IP/PII values by the API service before persistence.
- Cloud Scheduler triggers `export-audit-logs` once per night, invoking a Cloud Run job that batches the previous day's documents into BigQuery dataset `ops_audit_logs.audit_events` (partitioned by `createdAt`).
- A monthly Dataflow template copies the same range to Cloud Storage (`gs://hanko-field-exports/audit-logs/YYYY/MM/`) with a 7-year bucket retention policy for long-term compliance.
- Ops should monitor the Scheduler and Dataflow jobs (Stackdriver alerts are configured) and re-run the export job manually when replaying missed windows after incidents.

## Additional Notes

- Pub/Sub subscriptions default to pull with customizable ack deadlines and optional push/Dead Letter Queue configuration.
- Firestore indexes included represent critical query patterns (`userRef+createdAt`, `status+updatedAt`). Add more in Terraform as new endpoints require.
- Cloud Scheduler jobs authenticate via OIDC using the `svc-api-scheduler` service account.
- Update retention/versioning policies in `variables.tf` for bucket-specific compliance requirements.
