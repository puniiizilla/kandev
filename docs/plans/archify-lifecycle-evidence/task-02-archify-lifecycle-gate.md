---
id: "02-archify-lifecycle-gate"
title: "Enforce relevance and lifecycle admission"
status: complete
wave: 2
depends_on:
  - "01-archify-evidence-core"
plan: "plan.md"
requirements:
  - REQ-TASKS-ARCHIFY-EVIDENCE-001
  - REQ-TASKS-ARCHIFY-EVIDENCE-002
  - REQ-TASKS-ARCHIFY-EVIDENCE-003
acceptance_criteria:
  - AC-TASKS-ARCHIFY-EVIDENCE-001.1
  - AC-TASKS-ARCHIFY-EVIDENCE-001.2
  - AC-TASKS-ARCHIFY-EVIDENCE-001.3
  - AC-TASKS-ARCHIFY-EVIDENCE-001.4
  - AC-TASKS-ARCHIFY-EVIDENCE-002.1
  - AC-TASKS-ARCHIFY-EVIDENCE-002.2
  - AC-TASKS-ARCHIFY-EVIDENCE-003.2
  - AC-TASKS-ARCHIFY-EVIDENCE-003.3
  - AC-TASKS-ARCHIFY-EVIDENCE-003.4
system_design:
  - ../../specs/tasks/system-design/archify-lifecycle-evidence.md
---

# Task 02: Enforce relevance and lifecycle admission

## Summary

Add the deterministic relevance policy and connect SHA-bound evidence to task start and every
review/approval admission path. Expose authorized task APIs and live evidence updates, while keeping
the workflow engine as transition authority.

## In scope

- `archify_relevance_v1`, explicit override with reason/actor, and changed-path evaluation.
- Baseline capture at implementation admission and late baseline capture from recorded base SHA.
- After capture, comparison, current-head staleness, retry, and single-flight coordination.
- Fail-closed gates for manual, engine, and compatibility task-state review/approval paths.
- Task-authorized get, refresh, override, delta, and WebSocket projection.
- Focused service, handler, orchestrator, and migration tests.

## Out of scope

- React UI, Playwright flows, workflow-template changes, automatic approval, or merge.
- Architecture edits or automatic correction.

## Acceptance

- Policy results are deterministic, versioned, explainable, and cannot suppress a changed Archify
  source through a not-required override.
- Required tasks capture baseline/after evidence from pinned SHAs and cannot enter review or
  approval with missing, failed, stale, or mismatched evidence.
- All task APIs enforce workspace authorization and return safe structured diagnostics.

## Verification

```bash
(cd apps/backend && go test ./internal/task/service ./internal/task/handlers ./internal/orchestrator -run 'Architecture|Archify|Review|Approval')
(cd apps/backend && go test ./internal/architecture ./internal/task/repository/sqlite)
(cd apps/backend && golangci-lint run ./internal/architecture/... ./internal/task/service/... ./internal/task/handlers/... ./internal/orchestrator/...)
git diff --check -- apps/backend/internal/architecture apps/backend/internal/task apps/backend/internal/orchestrator
```

## Files likely touched

- `apps/backend/internal/task/service/architecture_evidence.go`
- `apps/backend/internal/task/service/architecture_evidence_test.go`
- `apps/backend/internal/task/handlers/architecture_evidence.go`
- `apps/backend/internal/task/handlers/architecture_evidence_test.go`
- `apps/backend/internal/orchestrator/architecture_evidence_gate.go`
- `apps/backend/internal/orchestrator/architecture_evidence_gate_test.go`
- `apps/backend/internal/orchestrator/workflow_store.go`
- `apps/backend/internal/orchestrator/event_handlers_workflow.go`
- `apps/backend/internal/backendapp/services.go`
- `apps/backend/internal/backendapp/helpers.go`

## Dependencies

- Task 01 evidence persistence and Archify adapter.

## Risks

- A missed transition path would violate fail-closed admission; inventory and table-drive all paths.
- Archify work must run outside transition transactions, followed by a short compare-and-swap.

## Parallelism

`sequential`

## Inputs

- System design sections: Relevance policy, Lifecycle admission, API and review projection.
- Existing workflow transition, task authorization, and task event patterns.

## Results

Implemented versioned relevance, start baseline capture, dirty/stale guards, review/approval
admission, authorized task endpoints, single-flight refresh, task-scoped live updates, override, and
receipt-verified delta delivery.
