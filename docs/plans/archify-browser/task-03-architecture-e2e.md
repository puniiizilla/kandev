---
id: "03-architecture-e2e"
title: "Stable browser end-to-end evidence"
status: completed
wave: 3
depends_on:
  - "01-archify-backend"
  - "02-architecture-ui"
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
  - AC-WORKSPACES-ARCHIFY-BROWSER-001.7
  - AC-WORKSPACES-ARCHIFY-BROWSER-001.8
system_design:
  - ../../specs/workspaces/system-design/archify-browser.md
---

# Task 03: Stable browser end-to-end evidence

## Summary

Prove the complete desktop and phone user flows against a disposable Git repository containing
seven Archify sources and a deterministic fake runtime. Confirm derived output stays outside the
repository and refresh never changes source or Git state.

## In scope

- E2E fixture/API support for repository architecture bindings and fake Archify runtime.
- Desktop navigation, seven-source inventory, render, source, validation, metadata, and refresh.
- Failure/retry scenario for invalid validation or unavailable runtime.
- Phone drawer navigation, content modes, touch geometry, safe-area containment, and overflow.
- Repository no-write and cache-only derived-output assertions.

## Out of scope

- Real TheBrain production data, real user home paths, editing, save, diffs, or commits.

## Acceptance

- Desktop E2E reaches `/architecture`, visits all seven diagrams, observes source/render/validation
  and SHA, refreshes, and proves repository bytes and status are unchanged.
- Mobile E2E completes equivalent browsing through the drawer and focused detail views with valid
  touch geometry and no page overflow.
- Failed validation keeps source accessible, blocks render, shows structured diagnostics, and
  succeeds after fixture correction plus retry without automatic repair.

## ASCII UI preview

### UI-01/UI-02/UI-03: E2E structure

Full previews: [plan.md](plan.md#ascii-ui-preview). AC: `.1` through `.8`.

```text
Desktop: [diagram list] [render] [source/validation]
Phone:   [diagram drawer] -> [one focused render/source/validation surface]
Failure: [source remains] + [diagnostics] + [Retry]
```

## Verification

```bash
(cd apps/web && pnpm e2e:run tests/architecture/architecture-browser.spec.ts)
(cd apps/web && pnpm e2e:run --project mobile-chrome tests/architecture/mobile-architecture-browser.spec.ts)
```

## Files likely touched

- `apps/web/e2e/helpers/api-client.ts`
- `apps/web/e2e/pages/architecture-page.ts`
- `apps/web/e2e/tests/architecture/architecture-browser.spec.ts`
- `apps/web/e2e/tests/architecture/mobile-architecture-browser.spec.ts`
- `apps/web/e2e/fixtures/`

## Dependencies

Tasks 01 and 02.

## Risks

- E2E must use a disposable runtime and repository, never the operator's TheBrain checkout.
- The mobile project discovers only `mobile-*.spec.ts`; command and filename must remain aligned.
- Worker-scoped repository mutations need explicit restoration or disposable records.

## Parallelism

`sequential`

## Inputs

- All acceptance criteria and both prior work-order results.
- E2E fixture-state, cleanup, managed-runner, and mobile parity contracts.

## Results

Implemented repository-bound browser evidence with a disposable seven-source Git fixture and a
deterministic fake Archify runtime. Desktop coverage verifies navigation, render, source,
validation, metadata, refresh, invalid-source recovery, fail-closed configuration, and unchanged
Git state. Mobile coverage verifies equivalent browsing, 44-pixel controls, containment, and no
horizontal overflow. Playwright screenshots are emitted under `apps/web/e2e/test-results/`.

Verification:

- `pnpm e2e:run tests/architecture/architecture-browser.spec.ts`: 2 passed.
- `pnpm e2e:run --project mobile-chrome tests/architecture/mobile-architecture-browser.spec.ts`:
  1 passed.
