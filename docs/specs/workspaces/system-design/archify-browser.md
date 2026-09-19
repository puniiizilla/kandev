---
status: draft
system: workspaces
requirements:
  - REQ-WORKSPACES-ARCHIFY-BROWSER-001
---

# Archify Browser System Design

## Purpose and boundaries

The workspace system binds a Kandev repository to Git-native Archify sources and presents them at
`/architecture`. Git objects remain the source authority. Kandev persists only repository-scoped
reference fields and keeps validation and HTML output in replaceable cache. The UI system's
existing iframe and responsive primitives supply presentation patterns but do not own the data.

This design follows
[ADR-2026-09-17-repository-bound-archify-workspaces](../../../decisions/2026-09-17-repository-bound-archify-workspaces.md).

## Requirement mapping

| Requirement | Design sections |
| --- | --- |
| `REQ-WORKSPACES-ARCHIFY-BROWSER-001` | [Repository binding](#repository-binding), [Git and source resolution](#git-and-source-resolution), [Validation and render cache](#validation-and-render-cache), [HTTP contracts](#http-contracts), [Responsive UI](#responsive-ui), [Failure and security boundaries](#failure-and-security-boundaries), [Verification strategy](#verification-strategy) |

## Architecture

```text
/architecture
    |
    v
ArchitecturePageClient ---- GET/POST /api/v1/architecture/*
                                    |
                                    v
                       repository authorization + binding
                                    |
                         resolve ref to immutable SHA
                                    |
                  +-----------------+------------------+
                  |                                    |
             git object reads                  managed temp checkout
           inventory + source                    at resolved SHA
                                                       |
                                  existing Archify runtime validate/render
                                                       |
                                             derived HTML cache
                                                       |
                                             authenticated iframe
```

The Kandev backend remains the only public server. The session port proxy is not used because the
source lifecycle belongs to a repository, not a task execution.

## Repository binding

Extend `models.Repository` and the existing `repositories` table with nullable string fields:

```text
architecture_git_ref
architecture_path
archify_runtime
```

The repository row supplies `repository_id`; `default_branch` is not an implicit architecture
fallback. All three architecture fields must be present for the repository to be eligible. The
existing repository create/update projections may carry these fields, but V1 adds no dedicated
settings UI. The current installation can set the supplied binding through the authenticated
repository update API.

`/architecture` operates in the active workspace. Exactly one visible repository with a complete
binding is eligible. Zero bindings produces a configuration-empty state; multiple bindings fail as
ambiguous rather than selecting by list order. This rule leaves room for an explicit repository
selector later without changing source ownership.

The SQLite and PostgreSQL migration adds only these columns with empty defaults. No architecture
table or JSON payload is introduced.

## Git and source resolution

The backend uses the registered local repository path after normal repository authorization and
trust checks.

1. Resolve `architecture_git_ref` through Git to a full commit SHA and reject non-commit results.
2. Normalize `architecture_path` as a repository-relative path. Reject absolute paths, traversal,
   symlink escape, and any source outside the resolved tree.
3. Enumerate direct children matching `*.architecture.json` at the resolved commit. V1 supports
   Archify `architecture` inputs only and derives the stable diagram ID from the filename.
4. Read source bytes through Git object access at the resolved commit. Parse only metadata needed
   for title and type; do not normalize or persist the JSON.
5. Report cleanliness for the configured source directory from the registered checkout separately
   from committed source bytes. Dirty state is informational and never changes the resolved input.

Every inventory response includes a revision token derived from repository ID plus resolved SHA.
Diagram source/render routes require that token so a branch movement cannot mix revisions.

## Validation and render cache

Archify runs as a bounded subprocess with an argument array:

```text
node <archify_runtime> validate architecture <source> --quality showcase --repo-root <checkout> --json
node <archify_runtime> render architecture <source> <cache-output.html> --quality showcase --repo-root <checkout>
```

The runtime path must be absolute, regular, non-symlink-escaping, and executable through the
configured Node runtime. The backend does not discover or install another Archify copy.

For validation/render, a cache manager creates a detached checkout at the resolved SHA in a
Kandev-managed temporary directory. Cache identity includes repository ID, SHA, source digest, and
runtime file identity. A per-key lock coalesces concurrent requests. Subprocess context cancellation,
timeout, stdout/stderr bounds, and atomic output replacement prevent orphaned or partial results.
The detached checkout and outputs are registered with `internal/system/storage/tempartifacts` and
are safe to delete on restart or storage maintenance.

Successful validation retains a structured projection of Archify's JSON receipt in memory/cache:
overall status plus diagnostic `subject`, `code`, `message`, and `supportedFixes`. Source JSON is
never copied into Kandev persistence. Render HTML is served only after validation succeeds. Refresh
evicts the selected revision's derived entry and reruns validation/render; it never edits source.

## HTTP contracts

Routes are authenticated and workspace-scoped:

```text
GET  /api/v1/architecture
GET  /api/v1/architecture/diagrams/:diagramID/source?revision=<token>
GET  /api/v1/architecture/diagrams/:diagramID/render?revision=<token>
POST /api/v1/architecture/refresh
```

The inventory response contains binding metadata, resolved SHA, dirty state, revision token, and
diagram entries with title, type, path, validation state, diagnostics, and render availability.
`render` returns trusted generated HTML with `Cache-Control: no-store`, content-type hardening, and
the same iframe security policy used by native HTML preview. Source returns JSON text for a
read-only code viewer. Refresh returns the replacement inventory only after its revision is fully
resolved.

HTTP handlers call workspace authorization before repository or filesystem reads. A foreign or
missing binding returns the same not-found surface. Subprocess stderr is sanitized and bounded;
source content and absolute host paths are not logged.

## Responsive UI

### Desktop

`ArchitecturePageClient` uses `PageShell` with a three-region workbench: diagram navigation,
selected render, and a source/validation detail panel. Render owns iframe scrolling. Navigation and
detail regions own their internal vertical scroll; the document does not become a two-axis scroller.
Refresh is the primary page action. Source and validation use tabs inside the detail panel.

### Phone

The `/architecture` destination appears in shared navigation. The phone page shows one focused
diagram. A discoverable diagram button opens an inset `Drawer` using the same hierarchy pattern as
`MobilePickerSheet`; selecting a diagram closes the drawer. Render is the primary surface. Source,
validation, and Git metadata use a full-height detail view reached by visible tabs/actions, not a
compressed three-column layout. `100dvh`, safe-area padding, one internal scroll owner, and 44-pixel
touch actions apply.

Shared hooks own inventory loading, revision token, selected diagram, refresh, and errors. Desktop
and phone wrappers only change composition. The nearest shipped patterns are `PageShell`,
`HtmlPreviewContent`, `MobileFileViewerPanel`, and `MobilePickerSheet`.

## Failure and security boundaries

- Missing or ambiguous binding: configuration state; no fallback repository.
- Missing repository/ref/path or non-commit ref: structured repository error; no cache mutation.
- Invalid JSON or Archify diagnostics: source remains viewable; render is blocked and diagnostics
  remain visible.
- Missing runtime, timeout, oversized output, or non-zero exit: render error with retry; no partial
  HTML is served.
- Ref movement during refresh: the completed response uses the newly resolved SHA atomically.
- Cache eviction or backend restart: the next render request regenerates from Git.
- Generated HTML runs only in the existing trusted-workspace iframe boundary and is never injected
  into the parent DOM.
- Source and materialization paths remain repository-confined; route parameters never become raw
  filesystem paths or subprocess arguments without closed validation.

## Observability

Structured logs record repository ID, resolved SHA, diagram ID, cache hit/miss, validation outcome,
render duration, and closed error code. They omit source bytes, generated HTML, runtime stderr, and
absolute checkout paths. V1 adds no high-cardinality metrics.

## Verification strategy

- Repository model and migration tests cover round-trip persistence and empty defaults on SQLite
  and PostgreSQL.
- Backend tests cover authorization, binding cardinality, ref resolution, path confinement,
  committed-source reads, dirty reporting, runtime validation, deterministic render caching,
  concurrency, timeout, refresh, and failure mapping using a fake runtime.
- Frontend tests cover routing, navigation manifest presence, inventory states, diagram switching,
  source/validation tabs, Git metadata, refresh, and stale-selection fallback.
- Desktop Chromium E2E proves all seven sources, navigation, source view, visible validation, render,
  refresh, and resolved SHA.
- Mobile Chrome E2E proves destination discovery, drawer navigation, focused render, source and
  validation access, 44-pixel actions, containment, and no document horizontal overflow.

## Related decisions

- [Repository-bound Archify workspaces](../../../decisions/2026-09-17-repository-bound-archify-workspaces.md)
- [Explicit local repository trust](../../../decisions/2026-07-20-explicit-local-repository-trust.md)
- [Trusted browser HTML preview](../../../decisions/2026-09-05-trusted-browser-html-preview.md)
