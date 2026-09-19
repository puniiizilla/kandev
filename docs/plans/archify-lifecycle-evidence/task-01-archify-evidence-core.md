---
id: "01-archify-evidence-core"
title: "Persist SHA-bound capture and comparison evidence"
status: complete
wave: 1
depends_on: []
plan: "plan.md"
requirements:
  - REQ-TASKS-ARCHIFY-EVIDENCE-002
  - REQ-TASKS-ARCHIFY-EVIDENCE-003
acceptance_criteria:
  - AC-TASKS-ARCHIFY-EVIDENCE-002.3
  - AC-TASKS-ARCHIFY-EVIDENCE-002.4
  - AC-TASKS-ARCHIFY-EVIDENCE-002.5
  - AC-TASKS-ARCHIFY-EVIDENCE-003.1
system_design:
  - ../../specs/tasks/system-design/archify-lifecycle-evidence.md
---

# Task 01: Persist SHA-bound capture and comparison evidence

## Summary

Build the task evidence schema and revision-explicit Archify capture/compare adapter. This work
order establishes reproducible evidence primitives without attaching them to workflow transitions
or rendering UI.

## In scope

- SQLite and PostgreSQL schema/migrations for current evidence and audit events.
- Repository models and CRUD with idempotency, bounds, cascade, and stale-token checks.
- Revision-explicit source inventory, validate, render, and `compare architecture` operations.
- Receipt projection, runtime contract hash, diagram add/remove summary, and disposable cache.
- Unit and repository integration tests.

## Out of scope

- Relevance policy, lifecycle hooks, HTTP handlers, WebSocket events, or UI.
- A Kandev semantic diff implementation.
- Source edits or repository writes.

## Acceptance

- Given pinned base/head SHAs, the service persists hashes and bounded Archify receipts and can
  regenerate missing derived artifacts without storing source JSON.
- Invalid bindings, SHAs, validation, comparison, receipts, timeouts, or output bounds produce the
  specified failure states without mutating the repository.
- SQLite and PostgreSQL enforce the same identity, cascade, and idempotency contract.

## Verification

```bash
(cd apps/backend && go test ./internal/architecture ./internal/task/repository/sqlite -run 'Architecture|Archify')
(cd apps/backend && go test ./internal/task/models)
(cd apps/backend && golangci-lint run ./internal/architecture/... ./internal/task/repository/sqlite/... ./internal/task/models/...)
git diff --check -- apps/backend/internal/architecture apps/backend/internal/task
```

## Files likely touched

- `apps/backend/internal/architecture/service.go`
- `apps/backend/internal/architecture/service_test.go`
- `apps/backend/internal/task/models/architecture_evidence.go`
- `apps/backend/internal/task/repository/sqlite/base_schema.go`
- `apps/backend/internal/task/repository/sqlite/base_migrations.go`
- `apps/backend/internal/task/repository/sqlite/architecture_evidence.go`
- `apps/backend/internal/task/repository/sqlite/architecture_evidence_test.go`
- `apps/backend/internal/task/repository/sqlite/architecture_evidence_postgres_test.go`
- `apps/backend/internal/task/repository/sqlite/workspace_deletion.go`

## Dependencies

None.

## Risks

- Receipt shape drift must fail closed without retaining unbounded raw output.
- Cache cleanup must not delete artifacts used by another in-flight comparison.

## Parallelism

`sequential`

## Inputs

- System design sections: Evidence state model, Capture service, Comparison, Persistence.
- Existing `internal/architecture.Service` and task repository migration patterns.
- Archify `compare architecture` and delivery receipt contracts.

## Results

Implemented SHA-explicit capture/compare, bounded evidence persistence, event history, runtime
contract hashing, and disposable derived cache.
