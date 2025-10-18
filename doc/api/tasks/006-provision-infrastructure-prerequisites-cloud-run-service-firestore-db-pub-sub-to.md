# Provision infrastructure prerequisites (Cloud Run service, Firestore DB, Pub/Sub topics, Cloud Storage buckets, Cloud Scheduler jobs) in IaC or documented scripts.

**Parent Section:** 1. Project & Environment Setup
**Task ID:** 006

## Goal
Provision all Google Cloud resources required by the API using repeatable Infrastructure as Code (IaC).

## Targets
- Cloud Run service for API (`api-service`) with VPC connector and minimum instances.
- Firestore (native mode) with composite indexes and TTL policies documented.
- Pub/Sub topics/subscriptions for AI jobs, webhook retries, export jobs.
- Cloud Storage buckets: `design-assets`, `ai-suggestions`, `exports`, `invoices` with retention + IAM policies.
- Cloud Scheduler jobs targeting internal maintenance endpoints (`cleanup-reservations`, `stock-safety-notify`).
- Secret Manager entries for PSP keys, HMAC secrets, webhook signing secrets.
- Service accounts with least privilege (runtime, scheduler invoker, job workers).

## Steps
1. Choose IaC tool (Terraform recommended) and scaffold modules per resource category.
2. Encode environment-specific variables (dev/stg/prod) and remote state storage.
3. Configure IAM bindings mapping service accounts to resources (Firestore, Pub/Sub, Storage, Cloud Run Invoker).
4. Implement output artifacts consumed by application config (bucket names, topic IDs).
5. Document bootstrap and promotion procedure in `doc/api/infrastructure.md`.

## Acceptance Criteria
- Running IaC plan/apply from clean checkout provisions required resources.
- Resource naming consistent with naming convention document.
- Terraform state secured, version controlled modules reviewed, and CI integrates drift detection.

---

## Provisioning Summary (2025-04-01)

### Deliverables
- ✅ Scaffolded Terraform configuration in `infra/terraform/` with modules for service accounts, Cloud Run, Firestore, Pub/Sub, Storage, Cloud Scheduler, and Secret Manager.
- ✅ Environment tfvars for `dev`, `stg`, and `prod` capturing project IDs, container images, scaling, and scheduler endpoints.
- ✅ Root module outputs bucket names, service accounts, topic IDs, and Cloud Run URL for downstream configuration.
- ✅ Infrastructure runbook added at `doc/api/infrastructure.md` covering backend setup, apply commands, and operational notes.
- ✅ Updated `.gitignore` to exclude Terraform state artefacts.

### Highlights
- Naming convention derives from `hanko-field[-env]-resource` ensuring per-environment isolation while aligning with storage/secrets naming.
- Firestore module enables TTL on `stockReservations` and adds core composite indexes required by order queries.
- Cloud Scheduler jobs authenticate via OIDC using the dedicated scheduler service account provisioned in Terraform.
- Secret Manager secrets are created but require manual version uploads after apply, keeping sensitive data out of git.

### Next Actions
- Configure remote state bucket and lock settings before the first apply.
- Integrate Terraform plan in CI (nightly drift detection + merge validation).
- Extend modules to include VPC connector provisioning and logging sinks as networking decisions finalize.
