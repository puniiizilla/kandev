---
id: "03-archify-review-surface"
title: "Surface evidence in task review"
status: complete
wave: 3
depends_on:
  - "02-archify-lifecycle-gate"
plan: "plan.md"
requirements:
  - REQ-TASKS-ARCHIFY-EVIDENCE-004
acceptance_criteria:
  - AC-TASKS-ARCHIFY-EVIDENCE-004.1
  - AC-TASKS-ARCHIFY-EVIDENCE-004.2
  - AC-TASKS-ARCHIFY-EVIDENCE-004.3
  - AC-TASKS-ARCHIFY-EVIDENCE-004.4
  - AC-TASKS-ARCHIFY-EVIDENCE-004.5
system_design:
  - ../../specs/tasks/system-design/archify-lifecycle-evidence.md
---

# Task 03: Surface evidence in task review

## Summary

Render current architecture evidence beside Git changes and test context with shared desktop and
phone behavior. Add all localized states and actions, plus public guidance for reviewers.

## In scope

- Typed architecture-evidence API client, hook, and WebSocket reconciliation.
- Shared evidence card in the existing Changes/review surface.
- Ready, not-required, pending, stale, validation, baseline, after, and diff failure states.
- Refresh, `/architecture`, and SHA-bound delta actions.
- Desktop and phone component tests, locale catalogs, and public task/review documentation.

## Out of scope

- New task tabs, routes, architecture editor controls, merge controls, or workflow decisions.
- Backend lifecycle behavior beyond contract fixes discovered by focused tests.

## Acceptance

- Reviewers see current base/head SHAs, required status, validation, architecture and semantic
  change status, affected diagrams, summary, and safe diagnostics beside Git changes.
- Desktop and phone expose the same actions and states; phone uses one Changes scroll owner, stacked
  touch-sized actions, and no document horizontal overflow.
- PASS never changes or bypasses the existing review/approval/human decision controls.

## ASCII UI preview

### UI-01: Desktop task Changes panel, review stage

Full preview: [plan.md](plan.md#ui-01-desktop-task-changes-panel-review-stage).

```text
| Architecture evidence                                      [Refresh] |
| Required: Yes  Validation: PASS  Changed: YES  Semantic: YES         |
| Base cb7f70a3   Head 91b3f842   Affected: SYSTEM_OVERVIEW, DATA_FLOW |
| [Open architecture] [Open architecture delta]                         |
```

### UI-02: Phone task Changes panel, review stage

Full preview: [plan.md](plan.md#ui-02-phone-task-changes-panel-review-stage).

```text
| Architecture evidence        [Retry] |
| Validation PASS  Semantic YES        |
| Base cb7f70a3  Head 91b3f842         |
| [Open architecture]                  |
| [Open architecture delta]            |
```

Actions are at least 44 pixels high on phone/coarse pointers. Both views map to
`AC-TASKS-ARCHIFY-EVIDENCE-004.1` through `.4`.

## Verification

```bash
(cd apps && pnpm --filter @kandev/web test -- architecture-evidence)
(cd apps/web && pnpm run typecheck)
(cd apps && pnpm --filter @kandev/web lint)
(cd apps/web && pnpm run i18n:check && pnpm run i18n:ratchet)
node --test scripts/validate-public-docs.test.mjs
node scripts/validate-public-docs.mjs
git diff --check -- apps/web docs/public
```

## Files likely touched

- `apps/web/lib/types/architecture-evidence.ts`
- `apps/web/lib/api/domains/architecture-evidence-api.ts`
- `apps/web/hooks/domains/task/use-architecture-evidence.ts`
- `apps/web/components/task/architecture-evidence-card.tsx`
- `apps/web/components/task/architecture-evidence-card.test.tsx`
- `apps/web/components/task/changes-panel.tsx`
- `apps/web/src/locales/*/task.json`
- `docs/public/sessions-and-review.md`
- `docs/public/tasks-and-workflows.md`

## Dependencies

- Task 02 task API, lifecycle state, and live projection.

## Risks

- Evidence updates must not reorder or remount the existing diff view.
- Long diagram names and translated diagnostics can overflow narrow cards.

## Parallelism

`sequential`

## Inputs

- System design sections: API and review projection, Review surface, Failure and recovery.
- Existing Changes panel, task authorization, responsive breakpoint, and mobile task layout.

## Results

Implemented localized desktop/phone evidence card in Changes with SHA, validation, semantic status,
affected diagrams, refresh/retry, architecture, and delta actions.
