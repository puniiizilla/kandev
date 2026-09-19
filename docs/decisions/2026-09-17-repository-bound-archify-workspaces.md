# ADR-2026-09-17-repository-bound-archify-workspaces: Bind Archify workspaces to repository Git references

**Status:** accepted
**Date:** 2026-09-17
**Area:** backend

## Context

Kandev needs a stable architecture browser for Archify sources that remain owned by a Git
repository. Task sessions and agentctl port-proxy URLs are ephemeral, while copying Archify JSON
into Kandev would create a second source of truth. Rendering an arbitrary configured revision must
also avoid changing the operator's checkout or committing generated HTML.

## Decision

Store the Archify binding on the existing Kandev repository record: Git ref, repository-relative
source directory, and Archify runtime path. The repository row itself supplies `repository_id`.
Resolve the configured ref to an immutable commit before every inventory or regeneration flow.

Read source JSON from Git at that resolved commit. Materialize a detached, read-only checkout only
inside a Kandev-managed temporary cache when Archify needs repository evidence for validation or
rendering. Generated HTML and receipts are derived cache entries keyed by repository identity,
resolved commit, source bytes, and runtime identity. They never enter the source repository or
Kandev persistence.

Serve source, metadata, validation results, and cached renders through authenticated Kandev HTTP
routes. Do not route this repository-lifetime feature through a task session or agentctl port
proxy. The browser entry remains `/architecture`; ephemeral render URLs are implementation detail.

## Consequences

- Archify JSON remains the only architecture source of truth.
- A branch may advance without making one response internally inconsistent because each response
  reports and uses one resolved commit.
- Kandev gains three nullable repository fields but no new database or architecture model.
- Rendering depends on a trusted local Archify runtime and Git checkout. Missing or invalid
  configuration fails visibly and does not fall back to another repository, ref, or runtime.
- Cached detached checkouts and render artifacts require bounded cleanup and can always be deleted
  and regenerated.
- The existing task-session port proxy remains unchanged and continues to own task-bound previews.

## Alternatives Considered

1. **Copy Archify JSON into Kandev storage.** Rejected because it creates a second source of truth.
2. **Use the live repository working tree.** Rejected because dirty files and branch movement would
   make a configured Git ref dishonest and rendering non-reproducible.
3. **Use a task session and port proxy.** Rejected because task and agentctl lifetimes cannot back a
   stable repository-level route.
4. **Run a separate Archify web server.** Rejected because Kandev already owns authenticated HTTP
   delivery and only needs bounded subprocess execution plus static cached output.
5. **Store one install-global architecture binding.** Rejected because repository ownership,
   authorization, and multi-workspace behavior would become ambiguous.
