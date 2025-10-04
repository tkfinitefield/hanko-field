# Define MVVM architecture conventions (view/widget, view-model, repository layers) and directory structure.

**Parent Section:** 0. Planning & Architecture
**Task ID:** 002

## Goal
Establish MVVM + Riverpod architecture conventions and directory layout.

## Decisions
- Folder structure (`lib/modules/<feature>/{view,view_model,repository}`, shared layers, test directories).
- Provider base classes (`Notifier`, `AsyncNotifier`, provider families) and naming conventions.
- Error/loading state modeling (sealed classes or custom state classes) and how view models expose state.
- Repository interfaces vs service clients and how to mock them.

## Deliverables
- Architecture guideline in `doc/app/architecture.md` with diagrams.
- Sample feature scaffold demonstrating conventions.
- Code review checklist enforcing architecture rules.
