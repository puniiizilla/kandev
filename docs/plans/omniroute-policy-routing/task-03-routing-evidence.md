---
status: complete
requirements:
  - REQ-AGENTS-OMNIROUTE-POLICY-003
acceptance_criteria:
  - AC-AGENTS-OMNIROUTE-POLICY-003.1
  - AC-AGENTS-OMNIROUTE-POLICY-003.2
  - AC-AGENTS-OMNIROUTE-POLICY-003.3
  - AC-AGENTS-OMNIROUTE-POLICY-003.4
  - AC-AGENTS-OMNIROUTE-POLICY-003.5
system_design:
  - docs/specs/agents/system-design/omniroute-policy-routing.md
---

# Task 03: Routing Evidence and Telemetry Linkage

## Outcome

Every routed attempt is auditable through durable evidence joined to the
authoritative task usage ledger, with a replay-safe paid escalation count.

## In scope

- Additive route-attempt/evidence migrations for SQLite and PostgreSQL.
- Attempt lifecycle correlation with task, session, turn, and usage event.
- Evidence read repository and authorized task API.
- Derived paid-escalation count and bounded metrics.
- End-to-end backend verification across restart and event replay.

## Exclusions

- Duplicate token/cost storage, independent pricing, dashboards, frontend UI,
  external telemetry exporters, or OmniRoute changes.

## Implementation acceptance

1. Evidence returns every required field and represents unobserved telemetry as absent.
2. Usage replay links once and cannot double count a strong attempt.
3. Backend integration tests prove local/free/low-cost/strong attribution,
   fail-closed errors, restart persistence, and no direct provider call path.

## Likely files

- `apps/backend/internal/task/repository/sqlite/dynamic_route.go`
- `apps/backend/internal/task/repository/sqlite/usage_events*.go`
- PostgreSQL/store-conformance migrations and fixtures
- `apps/backend/internal/task/handlers/`
- `apps/backend/internal/agent/runtime/dynamic/`

## Verification

```bash
make -C apps/backend test TEST_PKGS='./internal/task/... ./internal/agent/runtime/dynamic/... ./internal/orchestrator/...'
make -C apps/backend lint
python3 scripts/list-docs.py validate
python3 scripts/lint-spec-files.py --all
git diff --check
```

## Results

- Extended route attempts with task, route, classification, escalation,
  failure, quality, usage-link, and latency evidence using additive migrations.
- Linked the existing task usage ledger transactionally by `usage_event_id`;
  token, cost, provider, and model values remain joined ledger facts and are
  absent when unobserved.
- Added authorized `GET /api/v1/tasks/:taskID/routing-evidence`, derived distinct
  strong-attempt count, and bounded policy attempt/escalation metrics.
- Added SQLite integration coverage for local/free/low-cost/strong routing,
  quality escalation, restart recovery, usage replay, paid counting, and
  fail-closed ownership. PostgreSQL parity coverage is environment-gated.
