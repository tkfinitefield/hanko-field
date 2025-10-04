# Confirm scope, success criteria, and sequencing for API v1 with stakeholders based on `doc/api/api_design.md`.

**Parent Section:** 0. Planning & Alignment
**Task ID:** 001

## Goal
Align stakeholders on the MVP scope of API v1 described in `doc/api/api_design.md`, enumerate assumptions, and confirm delivery order across public, user, admin, webhook, and internal surfaces.

## Outputs
- Decision log capturing agreed endpoints, deferred items, and open questions.
- Release roadmap with milestones (alpha, beta, GA) and acceptance criteria.
- RACI for API ownership (product, tech lead, QA, ops) stored in shared workspace.

## Activities
1. Facilitate kickoff workshop reviewing API surfaces and business flows (checkout, production, operations).
2. Document agreed endpoint list, priorities, and sequencing in product wiki.
3. Record cross-team dependencies (Firestore schema decisions, PSP contracts, AI models) with owners and deadlines.
4. Circulate sign-off memo summarizing plan; collect stakeholder approvals.

## Acceptance Criteria
- All teams understand build/test order and success criteria.
- Risks and assumptions catalogued with mitigation owners.
- Backlog in tracker re-prioritised to match approved plan.

## Follow-up
- Review plan in weekly delivery syncs and update document on scope changes.
- Feed sequencing into CI/CD and infrastructure schedules.

---

## Stakeholder Agreement Summary (2025-03-31)

### Decision Log
- ✅ MVP scope covers Public catalogue reads, Authenticated user profile/design/cart/order flows, Stripe-based checkout, core admin catalogue + order ops, essential webhooks, and internal stock reservation + checkout commit services.
- ✅ Deferred to Beta: admin CMS bulk import/export, advanced production queue analytics, reviews moderation responses, BigQuery export job, PayPal webhook handling.
- ✅ Deferred to GA: multi-carrier shipping webhooks beyond Yamato, AI worker autoscaling policies, PSP-agnostic payment method vaulting, system diagnostics endpoints.
- ✅ Firestore remains single-region (asia-northeast1) for v1 with daily export; global multi-region resilience flagged as GA stretch.
- ❓ Open questions: PSP agreement finalization date, AI model SLA for suggestion latency, dedicated ops runbook for incident response.

### Release Roadmap & Success Criteria
- **Alpha (internal dogfood, Target: 2025-05-15)**
  - Surfaces: Public catalogue, Authenticated design builder (create/list/update), Cart + Estimate, Checkout (Stripe test mode), Internal reserve/commit, basic admin product management.
  - Criteria: End-to-end order from staff test account succeeds with manual shipment update; idempotency + error logging verified in Cloud Logging; Ops checklist reviewed.
- **Beta (invite users, Target: 2025-07-01)**
  - Adds: Promotions apply/remove, Order cancel/invoice request, Admin promotions + order status transitions, Reviews creation, Stripe live mode, PayPal webhook stub, production queue assignment.
  - Criteria: 95% of beta orders process without manual intervention; latency p95 < 400ms for read endpoints; Stripe + PayPal reconciliation reports match tracker.
- **GA (public launch, Target: 2025-09-10)**
  - Adds: Multi-carrier shipping webhooks, AI suggestion accept/reject loop, Reviews moderation, audit log export, PayPal full support, BigQuery export job.
  - Criteria: SLA 99.5% uptime (30d); AI suggestions round-trip < 90s p95; webhook retry backlog < 10 pending; no P1 incidents during launch week.

### Dependencies & Owners
- Firestore schema freeze (doc/db/schema) — Data Eng (M. Sato) by 2025-04-12.
- Stripe live credentials + webhook secrets — Finance/Legal (Y. Tanaka) by 2025-04-30.
- AI suggestion model contract — AI vendor (contracting via Product, R. Suzuki) by 2025-05-10.
- Cloud Run infrastructure baseline (runtime, secrets) — DevOps (K. Watanabe) by 2025-04-20.
- Incident comms playbook draft — QA/Ops (H. Kimura) by 2025-05-05.

### RACI for API v1 Delivery

| Workstream | Product (R. Suzuki) | Tech Lead (K. Nakamura) | Backend Eng (TBD squad) | QA Lead (H. Kimura) | DevOps (K. Watanabe) |
| --- | --- | --- | --- | --- | --- |
| Scope alignment & backlog | A | R | C | C | I |
| API design finalization | C | A | R | C | I |
| Implementation & reviews | I | C | R | C | C |
| Testing strategy & execution | C | C | R | A | C |
| Release readiness & go/no-go | R | A | C | C | C |
| Operations & monitoring | C | C | C | C | A |

### Risk & Mitigation Register
- PSP integration delays → Mitigation: maintain Stripe-first path, simulate PayPal via sandbox until contract signed; review status weekly.
- AI suggestions latency impacting UX → Mitigation: async polling with graceful fallbacks; add feature flag per environment.
- Firestore quota spikes from idempotency storage → Mitigation: configure TTL indexes, monitor metrics dashboard prior to beta.
- Webhook delivery failures → Mitigation: implement exponential retry with dead-letter queue ahead of beta; rehearse incident playbook.

### Next Actions
- Schedule weekly delivery sync (Product + Tech + QA) starting 2025-04-02.
- Open tracker tickets aligning backlog to phase scope; tag `alpha`, `beta`, `ga` accordingly.
- Draft sign-off memo and circulate to stakeholders for digital approval by 2025-04-05.
