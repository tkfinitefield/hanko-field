# Document navigation map (routes, tabs, nested flows) and deep link handling for app sections.

**Parent Section:** 0. Planning & Architecture
**Task ID:** 004

## Goal
Produce comprehensive navigation schema with tabs, nested stacks, and deep link handling.

## Activities
- Define route table with strongly typed arguments for each screen.
- Diagram tab navigator structure and back-stack preservation rules.
- Specify deep link URIs (e.g., `hanko://orders/{id}`) and entry handling including authentication guards.
- Document navigation guards (auth, onboarding completion) and fallback flows.
