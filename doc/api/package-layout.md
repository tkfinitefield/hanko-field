# API Package Layout Proposal

This layout defines where new code should live as we implement API v1. Each directory maps to a specific layer from ADR 0001.

```
api/
├── cmd/
│   └── api/                # main entrypoint, reads config, constructs DI container, starts HTTP server
├── internal/
│   ├── handlers/           # HTTP handlers (chi), webhook adapters, background job triggers
│   ├── services/           # business logic interfaces + command DTOs
│   ├── repositories/       # persistence interfaces, Firestore/Storage adapters
│   ├── domain/             # shared value objects used by services + repositories
│   ├── jobs/               # long-running/background processors (AI worker, cleanup)
│   ├── platform/           # firebase auth, firestore client factory, storage signed URL helpers, logging
│   └── di/                 # dependency injection container + wire provider sets
├── test/
│   ├── integration/        # emulator-backed repository and handler tests
│   └── fixtures/           # sample payloads, golden files
└── tools/
    └── scripts/            # development tooling (schema validation, seeders)
```

## Guidelines

- Handlers depend only on `services` interfaces and transport types; they must not import repositories directly.
- Services depend on `domain`, `repositories`, and `platform` abstractions but never on handler packages.
- Repositories may import `domain` and `platform` helpers, but not services or handlers.
- Shared constants, feature flags, and configuration structs live in `internal/platform/config` to avoid circular dependencies.
- Background jobs reuse services rather than calling repositories directly to preserve business rules.
- CLI tools (migrations, backfills) live under `cmd/` with their own mains but reuse the DI container for consistency.
