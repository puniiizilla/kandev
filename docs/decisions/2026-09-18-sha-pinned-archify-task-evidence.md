# ADR-2026-09-18-sha-pinned-archify-task-evidence: Store SHA-pinned Archify evidence, not architecture sources

**Status:** accepted
**Date:** 2026-09-18
**Area:** workflow

## Context

Task review needs to show whether implementation changed architecture and whether the relevant
Archify diagrams remain valid. A mutable worktree or latest-branch view cannot reproduce that
claim. Copying Archify JSON into task storage would create a second architecture authority, while a
Kandev-specific semantic diff would compete with Archify.

## Decision

Kandev records task architecture evidence against an explicit repository ID, base commit SHA, head
commit SHA, repository-relative Archify path, and Archify runtime contract. It persists bounded
validation and comparison receipts, hashes, status, diagnostics, and summaries. It does not persist
Archify source JSON as task data. Derived HTML and staged inputs are disposable cache and can be
regenerated from Git objects.

Archify's `compare architecture` remains the semantic and visual classification authority. Kandev
may classify a diagram added or removed from the Git inventory as semantic, but does not implement
a competing field-level diff.

For tasks classified `ARCHIFY_REQUIRED=YES`, current passing evidence is a precondition for entering
review or approval. It is not an approval, completion, or merge decision. A human gate remains
required.

## Consequences

- Evidence is reproducible while the pinned commits and configured Archify runtime contract remain
  available.
- A dirty worktree cannot serve as after evidence; relevant changes must be committed first.
- Cache eviction does not destroy evidence identity, but regenerating a visual artifact requires the
  pinned Git objects and compatible runtime.
- Lifecycle integration must fail closed for missing bindings, SHAs, validation, receipts, or stale
  head commits.
- Kandev persistence needs task evidence and audit rows, but no architecture database or source-copy
  retention.

## Alternatives Considered

- **Copy baseline and after JSON into Kandev:** Rejected because it creates a second editable or
  retained architecture representation and complicates source authority.
- **Diff JSON inside Kandev:** Rejected because Archify already owns architecture semantics and has a
  validated comparison command.
- **Compare the mutable task worktree:** Rejected because the result cannot be bound to or
  reproduced from a commit.
- **Run evidence after entering review:** Rejected because reviewers could observe or act on a task
  before required validation completed.
