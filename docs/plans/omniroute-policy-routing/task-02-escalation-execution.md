---
status: complete
requirements:
  - REQ-AGENTS-OMNIROUTE-POLICY-001
  - REQ-AGENTS-OMNIROUTE-POLICY-002
acceptance_criteria:
  - AC-AGENTS-OMNIROUTE-POLICY-001.4
  - AC-AGENTS-OMNIROUTE-POLICY-001.5
  - AC-AGENTS-OMNIROUTE-POLICY-001.6
  - AC-AGENTS-OMNIROUTE-POLICY-002.1
  - AC-AGENTS-OMNIROUTE-POLICY-002.2
  - AC-AGENTS-OMNIROUTE-POLICY-002.3
  - AC-AGENTS-OMNIROUTE-POLICY-002.4
system_design:
  - docs/specs/agents/system-design/omniroute-policy-routing.md
---

# Task 02: Escalation Execution and Quality Results

## Outcome

The conductor executes class-specific chains and accepts generation-fenced,
idempotent quality results without weakening provider recovery safety.

## In scope

- Candidate eligibility by snapshotted task class and route class.
- Durable cheap-chain exhaustion and escalation reasons.
- Quality-result domain command and authorized HTTP endpoint.
- Existing retry, circuit, continuation, and effect-safety integration.
- Restart and concurrency tests.

## Exclusions

- Quality judging, workflow producers/hooks, provider APIs, model lookup, UI, or
  automatic edits to stored profiles.

## Implementation acceptance

1. Every class follows its ordered eligibility contract, including strong-route guards.
2. Duplicate, stale, cross-task, and unauthorized quality results fail closed.
3. Rate limit, quota, timeout, quality failure, and configuration failure remain distinct.

## Likely files

- `apps/backend/internal/agent/runtime/dynamic/`
- `apps/backend/internal/orchestrator/`
- `apps/backend/internal/task/handlers/`
- `apps/backend/internal/task/repository/`

## Verification

```bash
make -C apps/backend test TEST_PKGS='./internal/agent/runtime/dynamic/... ./internal/orchestrator/... ./internal/task/...'
make -C apps/backend lint
git diff --check
```

## Results

- Added task-class-filtered candidate execution with durable classification and
  escalation snapshots. Strong routes stay guarded until high-risk/high-complexity,
  cheap-chain exhaustion, or a quality authorization.
- Added the authorized quality-result command and HTTP endpoint with closed enums,
  ownership/current-generation checks, duplicate rejection, and existing settled-turn
  route-action safety.
- Added additive quality-result persistence and focused policy, restart-state,
  repository, orchestrator, and route-action coverage.
- Focused Work Order 02 tests pass. The repository-wide suite still reaches the
  pre-existing Office migration failure in `TestMigrate_PriorityIdempotent`; lint
  remains blocked by pre-existing Work Order 01 migration errcheck findings and the
  documented `repository_planning.go` close errcheck finding.
