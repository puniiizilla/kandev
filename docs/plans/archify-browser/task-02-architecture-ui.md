---
id: "02-architecture-ui"
title: "Responsive architecture browser"
status: done
wave: 2
depends_on:
  - "01-archify-backend"
plan: "plan.md"
requirements:
  - REQ-WORKSPACES-ARCHIFY-BROWSER-001
acceptance_criteria:
  - AC-WORKSPACES-ARCHIFY-BROWSER-001.1
  - AC-WORKSPACES-ARCHIFY-BROWSER-001.2
  - AC-WORKSPACES-ARCHIFY-BROWSER-001.3
  - AC-WORKSPACES-ARCHIFY-BROWSER-001.5
  - AC-WORKSPACES-ARCHIFY-BROWSER-001.6
  - AC-WORKSPACES-ARCHIFY-BROWSER-001.7
system_design:
  - ../../specs/workspaces/system-design/archify-browser.md
---

# Task 02: Responsive architecture browser

## Summary

Add the stable `/architecture` route, shared navigation destination, localized states, and
responsive read-only workbench. Desktop uses three focused regions; phone uses a diagram drawer
and one full-height content surface.

## In scope

- Typed API client, shared page state, revision-safe refresh, and diagram selection.
- Desktop render/source/validation composition and Git metadata.
- Phone navigation drawer, focused content modes, safe-area/scroll/touch behavior.
- SPA route, navigation catalog, command destination, required locales, and component tests.
- Optional link from the pending Roadmap architecture card without changing its source contract.

## Out of scope

- Backend rendering, editing, semantic classification, saving, diffs, or task hooks.

## Acceptance

- `/architecture` shows all returned diagrams, exact source, render, validation, repository/ref/SHA,
  dirty state, and refresh/error outcomes without mixing revisions.
- Architecture is discoverable on desktop and phone navigation; pending `/roadmap` behavior remains.
- Phone provides capability parity with 44px actions, one scroll owner per surface, safe-area
  clearance, and no document horizontal overflow.

## ASCII UI preview

### UI-01: Architecture browser, desktop ready state

Full preview: [plan.md](plan.md#ui-01-architecture-browser-desktop-ready-state). AC: `.1`, `.2`,
`.3`, `.6`.

```text
| DIAGRAMS           | interactive RENDER         | [Source][Validation] |
| > System Overview  |                             | JSON or diagnostics  |
|   ...              |                             | path · type · SHA    |
```

### UI-02: Architecture browser, phone ready state

Full preview: [plan.md](plan.md#ui-02-architecture-browser-phone-ready-state). AC: `.2`, `.3`, `.7`.

```text
| Architecture [Refresh] |
| [System Overview   v]  |
| focused render/source  |
| [Render][Source][Valid]|
```

### UI-03: Missing or failed state

Full preview: [plan.md](plan.md#ui-03-missing-or-failed-state). AC: `.5`.

```text
| Architecture source unavailable |
| specific closed error    [Retry] |
```

## Verification

```bash
(cd apps && pnpm --filter @kandev/web typecheck)
(cd apps && pnpm --filter @kandev/web lint)
(cd apps && pnpm --filter @kandev/web test -- --run architecture)
(cd apps/web && pnpm run i18n:check)
```

## Files likely touched

- `apps/web/app/architecture/`
- `apps/web/lib/api/domains/architecture-api.ts`
- `apps/web/lib/navigation/core-destinations.ts`
- `apps/web/src/spa-routes.tsx`
- `apps/web/src/spa-routing.test.ts`
- `apps/web/components/navigation/`
- `apps/web/src/locales/*/architecture.json`
- `apps/web/src/i18n.ts`
- `apps/web/app/roadmap/roadmap-page-client.tsx`

## Dependencies

Task 01 API contracts and error shapes.

## Risks

- Existing uncommitted Roadmap route/sidebar/locale edits overlap likely integration files.
- Generated HTML retains trusted-workspace execution authority and must stay iframe-isolated.
- Dense source/render content can accidentally introduce nested or horizontal page scrolling.

## Parallelism

`sequential`

## Inputs

- Responsive contract and HTTP response shapes from the system design.
- Existing `PageShell`, `HtmlPreviewContent`, `MobileFileViewerPanel`, and `MobilePickerSheet` patterns.

## Results

- Added the repository-scoped `/architecture` SPA route, shared navigation destination, and typed API client.
- Added revision-safe shared loading, selection, render, source, validation, refresh, empty, and fail-closed error states.
- Added a three-region desktop workbench and focused phone composition using the existing mobile picker pattern, touch-sized actions, internal scrolling, and safe-area padding.
- Added localized catalogs for all shipped locales plus API, route, navigation, desktop, mobile, refresh, and error-state tests.
