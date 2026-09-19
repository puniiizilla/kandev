---
id: "04-archify-lifecycle-e2e"
title: "Verify the complete lifecycle flow"
status: complete
wave: 4
depends_on:
  - "03-archify-review-surface"
plan: "plan.md"
requirements:
  - REQ-TASKS-ARCHIFY-EVIDENCE-001
  - REQ-TASKS-ARCHIFY-EVIDENCE-002
  - REQ-TASKS-ARCHIFY-EVIDENCE-003
  - REQ-TASKS-ARCHIFY-EVIDENCE-004
acceptance_criteria:
  - AC-TASKS-ARCHIFY-EVIDENCE-001.4
  - AC-TASKS-ARCHIFY-EVIDENCE-002.1
  - AC-TASKS-ARCHIFY-EVIDENCE-002.2
  - AC-TASKS-ARCHIFY-EVIDENCE-002.3
  - AC-TASKS-ARCHIFY-EVIDENCE-002.4
  - AC-TASKS-ARCHIFY-EVIDENCE-003.1
  - AC-TASKS-ARCHIFY-EVIDENCE-003.2
  - AC-TASKS-ARCHIFY-EVIDENCE-003.3
  - AC-TASKS-ARCHIFY-EVIDENCE-003.4
  - AC-TASKS-ARCHIFY-EVIDENCE-004.1
  - AC-TASKS-ARCHIFY-EVIDENCE-004.2
  - AC-TASKS-ARCHIFY-EVIDENCE-004.3
  - AC-TASKS-ARCHIFY-EVIDENCE-004.4
system_design:
  - ../../specs/tasks/system-design/archify-lifecycle-evidence.md
---

# Task 04: Verify the complete lifecycle flow

## Summary

Prove the repository-bound task flow from baseline through human review using real Git commits and
the configured Archify runtime fixture. Produce desktop and phone evidence and close traceability;
do not add adjacent product behavior.

## In scope

- E2E fixture support for disposable Git repositories with Archify bindings and committed sources.
- Required and not-required lifecycle scenarios.
- Baseline, after, validation, compare, refresh, stale-head, and fail-closed transition scenarios.
- Desktop and Pixel 5 review evidence, delta navigation, touch sizing, and overflow checks.
- Assertions that source files, productive data, workflow decisions, and merge state are unchanged.
- Full package traceability, focused lint, spec validation, and scope diff.

## Out of scope

- Editing, auto-fix, automatic approval/merge, OmniRoute, or new workflow hooks beyond this design.
- Repairing unrelated lint, Office, roadmap, or existing test failures.

## Acceptance

- A real required task follows base SHA capture, implementation commit, after capture, Archify
  compare, review display, and unchanged human gate without session URLs or copied source data.
- Invalid configuration, validation failure, compare failure, and post-capture head change each
  block review with the correct visible state and recover after an authorized retry.
- Desktop and phone evidence proves equal user value, SHA display, diagram navigation, 44-pixel
  touch actions, and zero document horizontal overflow.

## ASCII UI preview

Verify `UI-01` and `UI-02` in [plan.md](plan.md#ascii-ui-preview). No new composition is introduced
by this work order.

## Verification

```bash
(cd apps/backend && go test ./internal/architecture ./internal/task/... ./internal/orchestrator/...)
(cd apps/backend && golangci-lint run ./internal/architecture/... ./internal/task/... ./internal/orchestrator/...)
(cd apps/web && pnpm e2e:run tests/review/archify-lifecycle-evidence.spec.ts)
(cd apps/web && pnpm e2e:run --project mobile-chrome tests/review/mobile-archify-lifecycle-evidence.spec.ts)
(cd apps/web && pnpm run typecheck && pnpm run i18n:check && pnpm run i18n:ratchet)
python3 scripts/list-docs.py validate
python3 scripts/lint-spec-files.test.py
python3 scripts/lint-spec-files.py --all
node --test scripts/validate-public-docs.test.mjs
node scripts/validate-public-docs.mjs
git diff --check
```

## Files likely touched

- `apps/web/e2e/helpers/api-client.ts`
- `apps/web/e2e/helpers/architecture-evidence-fixture.ts`
- `apps/web/e2e/tests/review/archify-lifecycle-evidence.spec.ts`
- `apps/web/e2e/tests/review/mobile-archify-lifecycle-evidence.spec.ts`
- `docs/plans/archify-lifecycle-evidence/plan.md`
- `docs/plans/archify-lifecycle-evidence/task-04-archify-lifecycle-e2e.md`
- `docs/specs/tasks/requirements/archify-lifecycle-evidence.md`
- `docs/specs/tasks/system-design/archify-lifecycle-evidence.md`

## Dependencies

- Tasks 01 through 03 complete with targeted checks passing.

## Risks

- E2E must use a disposable repository and cache root; never point mutation helpers at a developer or
  productive TheBrain checkout.
- Browser evidence can expose stale assets unless the managed runner rebuilds backend and frontend.

## Parallelism

`sequential`

## Inputs

- All requirement acceptance criteria and the complete system design.
- Existing architecture browser fixture, task workflow fixture, and Changes panel page objects.

## Results

Added disposable Git/Archify lifecycle E2E and mobile review coverage; desktop lifecycle E2E passes
with a real committed base/head, generated comparison receipt, and dirty-worktree rejection.
