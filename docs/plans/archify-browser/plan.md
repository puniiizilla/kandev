---
created: 2026-09-17
status: draft
requirements:
  - REQ-WORKSPACES-ARCHIFY-BROWSER-001
system_design:
  - ../../specs/workspaces/system-design/archify-browser.md
legacy_specs: []
---

# Implementation Plan: Stable Archify Browser

## Overview

Add repository-bound Archify configuration and an authenticated backend pipeline before exposing
the `/architecture` page. Backend-first order makes revision, validation, cache, and authorization
contracts executable before the UI depends on them. Desktop and phone E2E then prove the complete
stable browsing outcome against seven fixtures.

## Scope

### In scope

- Persist an Archify Git ref, relative source path, and existing runtime path on a repository.
- Resolve one active-workspace binding and one immutable commit per response.
- Inventory, validate, render, cache, and serve Git-native Archify architecture sources.
- Add `/architecture` to desktop, phone, and command navigation.
- Show diagram navigation, render, source, validation, and Git metadata.
- Refresh and regenerate without repository writes.

### Out of scope

- Editing, saving, semantic classification, commits, architecture diffs, task hooks, reviews,
  OmniRoute, TheBrain logic, or production-data changes.
- Additional Archify installation, architecture source copies, or a separate web server.
- Multi-binding repository selection UI; V1 fails visibly when more than one binding is configured.

## Technical approach

### Repository reference and migration

- Extend `apps/backend/internal/task/models.Repository`, request DTOs, repository SQL projections,
  and SQLite/PostgreSQL migrations with `architecture_git_ref`, `architecture_path`, and
  `archify_runtime`.
- Reuse `Service.GetRepository`/workspace authorization and the current repository PATCH endpoint.
- Configure the current installation with repository
  `a4fd4043-d36e-4de2-a8f5-03cbddc08efc`, ref
  `cb7f70a35611abcbc48cbae3463f7e777e3d5444`, path `docs/archify/ist`, and the supplied runtime
  after the new binary is running. Do not bake these values into product defaults.

### Backend inventory and derived cache

- Add a focused package under `apps/backend/internal/architecture/` for binding resolution, Git
  object reads, diagram inventory, subprocess validation/rendering, keyed concurrency, and cache
  lifecycle.
- Use a detached checkout at the resolved SHA registered through
  `internal/system/storage/tempartifacts`; never switch or write the registered source checkout.
- Add authenticated repository/workspace-scoped handlers under `/api/v1/architecture`.
- Integrate, but do not replace, the user's pending repository planning work. Its architecture card
  may link to `/architecture`; the new package must not adopt `.planning/architecture` as source.

### Frontend route and navigation

- Add a lazy `ArchitecturePageClient`, typed API client, shared state hook, and read-only source
  viewer using existing code/iframe primitives.
- Add an `architecture` destination to `APP_DESTINATIONS`, `/architecture` SPA resolution, shared
  navigation surfaces, command palette, and localized catalogs in all required locales.
- Preserve the pending `/roadmap` route and sidebar edits; merge route and navigation changes rather
  than overwriting them.

### Responsive composition

- Desktop uses a navigator, render canvas, and source/validation detail panel inside `PageShell`.
- Phone uses one focused render, a `MobilePickerSheet`-style diagram drawer, and direct full-height
  source/validation details. Shared state and actions remain viewport-independent.
- Render iframe owns its scroll. Surrounding surfaces use `100dvh`, internal vertical scrolling,
  safe-area padding, and no document-level horizontal overflow.

## ASCII UI preview

### UI-01: Architecture browser, desktop ready state

Entry point: `/architecture`. AC: `.1`, `.2`, `.3`, `.4`, `.6`.

```text
+----------------------------------------------------------------------------------+
| Architecture   thebrain · cb7f70a · clean                    [Refresh/Regenerate] |
+----------------------+--------------------------------------+--------------------+
| DIAGRAMS             | RENDER                               | DETAILS            |
| > System Overview    |                                      | [Source][Validation]|
|   Module Dependencies|       interactive Archify HTML       |                    |
|   Data Flow          |       zoom/pan owned by viewer       | JSON read-only     |
|   Reader/Writer      |                                      | or diagnostics     |
|   Runtime Topology   |                                      |                    |
|   Database           |                                      | path · type · SHA  |
|   End-to-End Flows   |                                      |                    |
+----------------------+--------------------------------------+--------------------+
```

Navigator, render, and details have separate internal vertical scroll owners. Panel widths are
illustrative; existing Kandev tokens and resizable patterns remain authoritative.

### UI-02: Architecture browser, phone ready state

Entry point: shared mobile navigation to `/architecture`. AC: `.2`, `.3`, `.7`.

```text
+----------------------------------+
| Architecture             [Refresh]|
| [System Overview            v]   |  opens inset diagram drawer
+----------------------------------+
|                                  |
|      focused Archify render      |  render owns scrolling
|                                  |
+----------------------------------+
| [Render] [Source] [Validation]   |  44px touch actions
| thebrain · cb7f70a · clean       |
+----------------------------------+
```

Source and Validation replace the focused body as full-height views; they are not stacked below the
render. The diagram drawer owns its scroll and clears the bottom safe area.

### UI-03: Missing or failed state

Entry point: `/architecture` without a usable binding or with validation/render failure. AC: `.5`.

```text
+----------------------------------------------------+
| Architecture                                       |
| Architecture source unavailable                    |
| <specific closed error and affected repository>    |
|                                      [Retry]        |
+----------------------------------------------------+
```

Validation failure additionally keeps Source visible and lists structured diagnostic code,
element, message, and supported correction. It never offers an automatic repair.

## Tests

| Acceptance criteria | Evidence |
| --- | --- |
| `.1`, `.6`, `.8` | repository persistence, authorization, ref-resolution, revision-token, and refresh tests |
| `.2`, `.3` | inventory/source handler tests plus frontend route/component tests |
| `.4` | fake-runtime validation/render tests, deterministic cache-key tests, and repository no-write assertions |
| `.5` | backend error mapping and frontend loading/empty/error/retry tests |
| `.7` | responsive component tests and mobile Playwright geometry/outcome checks |

## E2E tests

- `apps/web/e2e/tests/architecture/architecture-browser.spec.ts`: configure a disposable repository
  with seven source fixtures, browse every diagram, inspect source and validation, render, refresh,
  and confirm the SHA and repository remain unchanged. Covers `.1` through `.6` and `.8`.
- `apps/web/e2e/tests/architecture/mobile-architecture-browser.spec.ts`: enter through mobile
  navigation, switch diagrams through the drawer, open source and validation, refresh, verify 44px
  controls, containment, and zero document horizontal overflow. Covers `.2`, `.3`, `.5`, `.7`.

## Work orders

- [ ] [Task 01: Repository binding and Archify backend](task-01-archify-backend.md)
- [x] [Task 02: Responsive architecture browser](task-02-architecture-ui.md)
- [ ] [Task 03: Stable browser end-to-end evidence](task-03-architecture-e2e.md)

## Verification results

Pending.

## Risks

- The absolute runtime path is intentionally operator-owned and local-install specific. Missing
  runtime must fail closed; product defaults cannot contain the supplied home path.
- Archify HTML is trusted workspace output. It must retain iframe isolation and never enter the
  Kandev parent DOM.
- Detached Git materialization must integrate with temporary-artifact cleanup without task
  ownership or source-checkout mutation.
- Pending Roadmap work overlaps route, sidebar, locale, and repository-handler files; implementation
  must preserve and accommodate those edits.
