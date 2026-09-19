---
id: "01-archify-backend"
title: "Repository binding and Archify backend"
status: done
wave: 1
depends_on: []
plan: "plan.md"
requirements:
  - REQ-WORKSPACES-ARCHIFY-BROWSER-001
acceptance_criteria:
  - AC-WORKSPACES-ARCHIFY-BROWSER-001.1
  - AC-WORKSPACES-ARCHIFY-BROWSER-001.2
  - AC-WORKSPACES-ARCHIFY-BROWSER-001.3
  - AC-WORKSPACES-ARCHIFY-BROWSER-001.4
  - AC-WORKSPACES-ARCHIFY-BROWSER-001.5
  - AC-WORKSPACES-ARCHIFY-BROWSER-001.6
  - AC-WORKSPACES-ARCHIFY-BROWSER-001.8
system_design:
  - ../../specs/workspaces/system-design/archify-browser.md
---

# Task 01: Repository binding and Archify backend

## Summary

Persist the minimal repository binding and implement authenticated inventory, source, validation,
render, refresh, and derived-cache APIs. Resolve every operation against one immutable SHA without
writing the registered repository checkout.

## In scope

- Repository model, DTO, SQLite/PostgreSQL schema, migration, and update support.
- Ref/path/source resolution and dirty-state projection.
- Bounded Archify subprocess runner and managed detached-checkout/render cache.
- Authenticated `/api/v1/architecture` handlers and structured errors.
- Runtime wiring, unit/integration tests, and no-write assertions.

## Out of scope

- Browser page, navigation UI, Playwright flows, source editing, or commits.
- Modifying the pending `.planning/architecture` source convention.

## Acceptance

- Repository binding round-trips with empty migration defaults and workspace authorization.
- Seven committed sources can be inventoried, validated, rendered deterministically, and served at
  one resolved SHA while source checkout and Git status remain unchanged.
- Missing/ambiguous configuration, invalid paths/refs/runtime, diagnostics, timeout, and cache
  eviction fail closed with stable error codes and no partial render.

## Verification

```bash
(cd apps/backend && go test -tags fts5 ./internal/architecture/... ./internal/task/handlers/... ./internal/task/repository/sqlite/...)
make -C apps/backend lint
```

## Files likely touched

- `apps/backend/internal/architecture/`
- `apps/backend/internal/backendapp/`
- `apps/backend/internal/task/models/models.go`
- `apps/backend/internal/task/service/service_resources.go`
- `apps/backend/internal/task/repository/sqlite/base_schema.go`
- `apps/backend/internal/task/repository/sqlite/base_migrations.go`
- `apps/backend/internal/task/repository/sqlite/repository_entity.go`
- `apps/backend/internal/task/handlers/repository_handlers.go`
- `apps/backend/internal/task/handlers/route_registration_test.go`

## Dependencies

None.

## Risks

- Repository SQL projections have many callers; every scan/insert/update path must stay aligned.
- Pending Roadmap handlers already modify repository route registration and must be preserved.
- Subprocess and temporary-artifact ownership need cancellation-safe cleanup.

## Parallelism

`sequential`

## Inputs

- Requirement sections for stable browsing, cache-only output, failures, refresh, and authorization.
- System-design sections for binding, Git resolution, cache, HTTP contracts, and security.
- ADR `2026-09-17-repository-bound-archify-workspaces`.

## Results

- Added repository-owned Archify binding fields with SQLite/PostgreSQL-compatible schema migration and authenticated repository update support.
- Added fail-closed architecture inventory, source, refresh, validation, and render endpoints pinned to one resolved commit SHA.
- Added bounded Archify execution and SHA-keyed temporary derived caches without writing the registered checkout.
- Verified seven-source inventory, validation/render behavior, repository persistence, route failure states, and unchanged source Git status.
