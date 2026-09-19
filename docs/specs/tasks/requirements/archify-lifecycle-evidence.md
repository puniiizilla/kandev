---
status: draft
system: tasks
created: 2026-09-18
owners:
  - kandev
---

# Archify Lifecycle Evidence Requirements

## Overview

Kandev shall attach reproducible architecture evidence to architecture-relevant tasks. The task
system owns this contract because relevance, lifecycle admission, evidence history, review, and the
human gate are task-scoped outcomes. Repository and Archify binding resolution remain owned by the
workspace system.

## Terminology

- **Architecture evidence:** SHA-bound Archify validation, render, receipt, and comparison results
  associated with a task. It is not an architecture source.
- **Baseline:** Evidence derived from the task repository at its recorded base commit.
- **After capture:** Evidence derived from the same repository and source path at the task head
  commit.
- **Required task:** A task whose deterministic policy result is `ARCHIFY_REQUIRED=YES`.
- **Semantic change:** A component, connection, boundary, authority, reader/writer, or domain-flow
  change reported by Archify comparison. Layout-only changes are visual changes.

## Requirements

### REQ-TASKS-ARCHIFY-EVIDENCE-001: Determine architecture-evidence relevance

**Intent:** Run architecture gates only for tasks with an explicit or observable architecture
impact while keeping the decision understandable and overridable by a human.

#### Acceptance criteria

- **AC-TASKS-ARCHIFY-EVIDENCE-001.1:** When a task has an explicit architecture-required or
  architecture-not-required override, Kandev shall expose the selected value and the human or
  system reason that produced it.
- **AC-TASKS-ARCHIFY-EVIDENCE-001.2:** When no override exists, Kandev shall evaluate a deterministic
  policy from task labels and changed repository paths, return `ARCHIFY_REQUIRED=YES|NO`, and expose
  the matched rule. It shall not invoke an LLM or infer from uncommitted prose.
- **AC-TASKS-ARCHIFY-EVIDENCE-001.3:** When an Archify source under the configured architecture path
  changes, the policy shall return `ARCHIFY_REQUIRED=YES`; a not-required override shall not suppress
  this source-change rule.
- **AC-TASKS-ARCHIFY-EVIDENCE-001.4:** When repository identity, the base commit, or the architecture
  binding is missing or ambiguous, a task otherwise classified as required shall fail closed and
  shall not enter review or approval as if architecture evidence passed.

### REQ-TASKS-ARCHIFY-EVIDENCE-002: Capture SHA-bound baseline and after evidence

**Intent:** Make architecture evidence reproducible without copying repository architecture sources
into Kandev.

#### Acceptance criteria

- **AC-TASKS-ARCHIFY-EVIDENCE-002.1:** When a required task becomes ready for implementation,
  Kandev shall record its repository ID, base commit SHA, configured Archify source path, validation
  status, receipt identity, and source hash as the baseline before implementation proceeds.
- **AC-TASKS-ARCHIFY-EVIDENCE-002.2:** When a task becomes relevant only after changed paths are
  known, Kandev shall reproduce the baseline from the immutable base SHA before review rather than
  treating the current checkout as the baseline.
- **AC-TASKS-ARCHIFY-EVIDENCE-002.3:** Before a required task enters review or approval, Kandev shall
  resolve its current head commit, validate and render the configured Archify sources at that exact
  commit, and record the after receipt, source hashes, and validation status.
- **AC-TASKS-ARCHIFY-EVIDENCE-002.4:** Capture shall not modify Archify sources, the task worktree,
  productive repository files, or productive data. Derived render files shall remain replaceable
  cache and shall never be committed automatically.
- **AC-TASKS-ARCHIFY-EVIDENCE-002.5:** Retrying the same task, repository, source path, SHA, and
  Archify runtime contract shall reuse or replace the same logical capture and shall not create an
  unbounded series of duplicate evidence records.

### REQ-TASKS-ARCHIFY-EVIDENCE-003: Compare architecture and gate lifecycle failures

**Intent:** Put validated semantic and visual architecture deltas beside code and test evidence
before human review.

#### Acceptance criteria

- **AC-TASKS-ARCHIFY-EVIDENCE-003.1:** After both captures pass, Kandev shall use the configured
  Archify comparison capability to report affected diagrams, added, removed, and changed components
  and connections, boundary or authority changes, visual-only changes, semantic-change status, and
  aggregate validation status.
- **AC-TASKS-ARCHIFY-EVIDENCE-003.2:** A required task shall not enter review or approval while its
  current evidence state is `ARCHIFY_BASELINE_FAILED`, `ARCHIFY_AFTER_FAILED`,
  `ARCHIFY_DIFF_FAILED`, or `ARCHIFY_VALIDATION_FAILED`; the state and actionable diagnostics shall
  remain visible and retryable.
- **AC-TASKS-ARCHIFY-EVIDENCE-003.3:** A successful architecture comparison shall not merge, approve,
  or complete a task. Existing reviewer decisions and the human gate remain authoritative.
- **AC-TASKS-ARCHIFY-EVIDENCE-003.4:** If the task head changes after a successful after capture,
  Kandev shall mark that evidence stale and require a new after capture and comparison before review
  or approval admission.

### REQ-TASKS-ARCHIFY-EVIDENCE-004: Review architecture evidence with code and tests

**Intent:** Let reviewers understand architecture impact without leaving the task lifecycle.

#### Acceptance criteria

- **AC-TASKS-ARCHIFY-EVIDENCE-004.1:** For a required task, the task review surface shall show base
  SHA, head SHA, architecture-changed status, semantic-change status, validation status, affected
  diagrams, and a concise diff summary beside the existing code and test review context.
- **AC-TASKS-ARCHIFY-EVIDENCE-004.2:** The review surface shall link to the repository-bound
  `/architecture` browser and shall expose the SHA-bound comparison artifact or structured delta.
  It shall not use a session-bound preview URL.
- **AC-TASKS-ARCHIFY-EVIDENCE-004.3:** Pending, skipped, stale, and each failure state shall remain
  distinguishable on desktop and phone. Phone controls shall have at least a 44-pixel active touch
  dimension and shall not introduce horizontal page overflow.
- **AC-TASKS-ARCHIFY-EVIDENCE-004.4:** A human gate shall present Git changes, test status, and the
  current Archify evidence together and shall require a human decision even when all automated
  checks pass.
- **AC-TASKS-ARCHIFY-EVIDENCE-004.5:** Only users authorized for the task workspace shall read,
  regenerate, or override its architecture evidence. Diagnostics shall not expose credentials,
  unrestricted local paths, or raw subprocess output.

## Out of scope

- Editing, saving, or automatically repairing Archify sources.
- A Kandev architecture model, architecture database, model catalog, or independent diff engine.
- Automatic code or architecture refactoring, automatic commits, or automatic merge.
- OmniRoute, agent-routing, workflow-template, or provider changes.
- Replacing Git as implementation authority or Archify as architecture-evidence authority.
