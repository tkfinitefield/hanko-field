# Model navigation taxonomy and RBAC visibility rules; document mapping between sidebar groups and user roles. ✅

**Parent Section:** 0. Planning & Architecture
**Task ID:** 003

## Goal
Model sidebar navigation, route grouping, and permission visibility rules for staff vs admin roles.

## Activities
- Extract navigation groups from design (Dashboard, Orders, Catalog, Content, Marketing, Customers, System).
- Define role matrix specifying which sections appear for each role (`admin`, `ops`, `support`, `marketing`).
- Document URL ownership and breadcrumb structure for each leaf.
- Encode mapping in configuration (Go map or JSON) consumed by template renderer.

## Acceptance Criteria
- Sidebar renders correct sections per role in staging test.
- Documentation lists routes, required role(s), and description.
- RBAC logic tested via unit tests verifying hidden sections for unauthorized roles.
