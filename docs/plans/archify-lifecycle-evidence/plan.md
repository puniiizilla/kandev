---
created: 2026-09-18
status: complete
requirements:
  - REQ-TASKS-ARCHIFY-EVIDENCE-001
  - REQ-TASKS-ARCHIFY-EVIDENCE-002
  - REQ-TASKS-ARCHIFY-EVIDENCE-003
  - REQ-TASKS-ARCHIFY-EVIDENCE-004
system_design:
  - ../../specs/tasks/system-design/archify-lifecycle-evidence.md
legacy_specs: []
---

# Implementation Plan: Archify Lifecycle Evidence

## Overview

Deliver the feature in four sequential slices: establish SHA-bound evidence persistence and an
Archify compare adapter, add deterministic relevance and lifecycle admission, expose the evidence in
the existing task Changes/review surface, then prove the complete workflow with desktop and phone
E2E evidence. This order keeps Git and Archify authoritative at every intermediate boundary.

## Scope

### In scope

- Task-owned relevance, evidence state, persistence, API, and audit history.
- Baseline and after capture from immutable Git commits through the configured Archify runtime.
- Archify-owned comparison and semantic/visual classification.
- Fail-closed work/review/approval lifecycle integration.
- Desktop and phone review evidence beside Git changes and tests.
- Focused public documentation for architecture evidence in task review.

### Out of scope

- Archify editing, source repair, source copies, or an architecture database.
- New workflow stage types or workflow-template changes.
- Automatic commit, approval, completion, merge, or architecture refactoring.
- OmniRoute or agent/provider routing changes.
- A Kandev model catalog or independent architecture diff engine.

## Technical approach

### Evidence and Archify adapter

Extend `internal/architecture.Service` with revision-explicit capture and comparison methods that
reuse its validated repository binding, immutable checkout, bounded subprocess, and temp cache.
Add task models and SQLite/PostgreSQL migrations for current evidence plus state-transition audit
events. Store hashes, bounded receipt projections, summary fields, and cache identity; never store
source JSON.

### Policy and lifecycle gate

Add `ArchitectureEvidenceService` under the task domain. It owns the versioned rule evaluator,
capture idempotency, retries, stale-head checks, and safe projection. Wire baseline capture into task
implementation admission. Wire one pre-review gate into every manual and engine path entering a
`review` or `approval` stage or compatibility task state `REVIEW`. The final transition transaction
checks the evidence token and head SHA before committing the workflow move.

### API and task projection

Expose task-authorized get, refresh, override, and delta routes. Publish one task-scoped WebSocket
update after evidence state changes. Extend E2E seeding only through production APIs; do not create a
test-only architecture authority.

### Review UI

Add a shared architecture-evidence view model and card to the existing Changes panel. Desktop and
phone reuse data and actions. Review/approval expands evidence by default; other stages collapse it.
Use the task locale namespace for all copy and update all locale catalogs. Update
`docs/public/sessions-and-review.md` and `docs/public/tasks-and-workflows.md` with the operator flow
and failure recovery.

## ASCII UI preview

### UI-01: Desktop task Changes panel, review stage

Entry point: task detail, **Changes** panel. State: current evidence ready for human review.

```text
+ Changes ---------------------------------------------------------------+
| branch: task/archify-lifecycle                    Review changes        |
+------------------------------------------------------------------------+
| Architecture evidence                                      [Refresh]   |
| Required: Yes   Validation: PASS   Architecture changed: YES           |
| Semantic change: YES                                                    |
| Base cb7f70a3                         Head 91b3f842                      |
| Affected diagrams: SYSTEM_OVERVIEW, DATA_FLOW                           |
| 2 components changed, 1 connection added, layout adjusted              |
| [Open architecture]  [Open architecture delta]                          |
+------------------------------------------------------------------------+
| Tests: PASS                                                             |
| Git changes                                                             |
|  M apps/backend/...                                                     |
|  M docs/specs/...                                                       |
+------------------------------------------------------------------------+
```

Failure uses the same section with the exact state, bounded diagnostic, and Retry. `PASS` is never
an approval control. Existing review/human actions remain outside this card.

### UI-02: Phone task Changes panel, review stage

Entry point: task mobile workspace, **Changes**. State: same evidence as UI-01.

```text
+ Changes -----------------------------+
| Architecture evidence        [Retry] |
| Required                       Yes   |
| Validation                     PASS  |
| Architecture changed           YES   |
| Semantic change                YES   |
| Base cb7f70a3   Head 91b3f842        |
|                                      |
| Affected diagrams                    |
| SYSTEM_OVERVIEW                      |
| DATA_FLOW                            |
|                                      |
| [Open architecture]                  |
| [Open architecture delta]            |
+--------------------------------------+
| Tests                                |
| Git changes                          |
| ... single vertical scroll ...       |
+--------------------------------------+
```

The existing Changes panel owns vertical scrolling. Actions are stacked, at least 44 pixels high,
and do not create document horizontal overflow. The phone view is focused content, not a compressed
desktop side rail.

Structural requirements map to `AC-TASKS-ARCHIFY-EVIDENCE-004.1` through `.4`; exact spacing and
word wrapping are illustrative.

## Tests

- `apps/backend/internal/architecture/service_test.go`: revision-explicit capture, compare receipt,
  source addition/removal, cache regeneration, time/output bounds, and fail-closed binding.
- `apps/backend/internal/task/repository/sqlite/architecture_evidence_test.go`: SQLite persistence,
  idempotency, event history, cascade, stale token, and bounded JSON.
- `apps/backend/internal/task/repository/sqlite/architecture_evidence_postgres_test.go`: migration and
  repository parity.
- `apps/backend/internal/task/service/architecture_evidence_test.go`: policy precedence, capture
  state machine, authorization, override audit, retries, and stale head.
- `apps/backend/internal/orchestrator/architecture_evidence_gate_test.go`: baseline admission and all
  review/approval transition paths, including manual moves and engine retries.
- `apps/web/components/task/architecture-evidence-card.test.tsx`: state, diagnostics, actions,
  accessibility, and stage expansion.
- `apps/web/lib/api/domains/architecture-evidence-api.test.ts`: wire contract and errors.

## E2E tests

- `apps/web/e2e/tests/review/archify-lifecycle-evidence.spec.ts`: required task from base capture to
  review, source/render/validation/delta links, refresh, stale head, and failure block.
- `apps/web/e2e/tests/review/mobile-archify-lifecycle-evidence.spec.ts`: same review value on Pixel 5,
  touch actions, single scroll owner, and no horizontal overflow.
- Backend integration scenario uses a temporary Git repository and the configured Archify runtime
  fixture to prove no session URL, source copy, productive repository write, or automatic approval.

## Work orders

- [x] [Task 01: Persist SHA-bound capture and comparison evidence](task-01-archify-evidence-core.md)
- [x] [Task 02: Enforce relevance and lifecycle admission](task-02-archify-lifecycle-gate.md)
- [x] [Task 03: Surface evidence in task review](task-03-archify-review-surface.md)
- [x] [Task 04: Verify the complete lifecycle flow](task-04-archify-lifecycle-e2e.md)

## Verification results

Backend evidence, lifecycle gates, review UI, desktop E2E, focused tests, i18n, typecheck, and
specification validation pass. Generated artifacts remain temporary and the E2E repository tree is
restored after verification.

## Risks

- Long-running Archify subprocesses must not hold task/workflow database locks; admission therefore
  needs a short final SHA/state compare-and-swap.
- Existing transition paths are numerous. The lifecycle work order must inventory manual, engine,
  task-state, and compatibility review entry paths before claiming fail-closed coverage.
- Archify runtime upgrades can change receipt bytes. The runtime contract hash prevents stale cache
  reuse but retained evidence must remain readable as a bounded historical projection.
- Multi-repository tasks need one evidence result per configured repository. V1 aggregates required
  repositories and fails review if any required repository fails; repositories without an Archify
  binding are not silently substituted.
