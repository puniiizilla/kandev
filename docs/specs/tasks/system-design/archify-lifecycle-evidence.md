---
status: draft
system: tasks
requirements:
  - REQ-TASKS-ARCHIFY-EVIDENCE-001
  - REQ-TASKS-ARCHIFY-EVIDENCE-002
  - REQ-TASKS-ARCHIFY-EVIDENCE-003
  - REQ-TASKS-ARCHIFY-EVIDENCE-004
---

# Archify Lifecycle Evidence System Design

## Purpose and boundaries

The task system owns relevance, capture timing, review admission, evidence persistence, and the
review projection. It consumes the repository-bound Archify binding and fail-closed Git resolution
defined by the [Archify browser design](../../workspaces/system-design/archify-browser.md). Git
objects remain implementation authority. Archify source JSON remains architecture authority.
Kandev stores only task evidence and disposable derived artifacts.

## Requirement mapping

| Requirement | Design sections |
| --- | --- |
| `REQ-TASKS-ARCHIFY-EVIDENCE-001` | [Relevance policy](#relevance-policy), [Failure and recovery](#failure-and-recovery) |
| `REQ-TASKS-ARCHIFY-EVIDENCE-002` | [Capture service](#capture-service), [Persistence](#persistence) |
| `REQ-TASKS-ARCHIFY-EVIDENCE-003` | [Comparison](#comparison), [Lifecycle admission](#lifecycle-admission) |
| `REQ-TASKS-ARCHIFY-EVIDENCE-004` | [API and review projection](#api-and-review-projection), [Review surface](#review-surface) |

## Components and responsibilities

- `internal/architecture.Service` gains revision-explicit inventory, validation, render, and compare
  operations. These operations reuse the configured repository binding and the existing Archify
  runtime; they never resolve a default branch, current worktree HEAD, or session URL implicitly.
- A task-owned `ArchitectureEvidenceService` classifies relevance, coordinates baseline/after
  capture, persists evidence, and returns a review-admission result. It does not interpret provider
  or model catalogs and does not parse architecture semantics independently of Archify.
- The task repository stores one current evidence record per task/repository/binding generation and
  immutable attempt outcomes needed for audit. Repository deletion cascades these rows.
- Orchestrator task-start and workflow-transition paths call the service at the two lifecycle
  boundaries. The workflow engine remains transition authority; the evidence service supplies a
  precondition result only.
- Task HTTP/WS projection exposes the current evidence. A shared React evidence card renders it in
  the task Changes/review context on desktop and phone.

## Evidence state model

The current evidence state is a closed set:

```text
NOT_REQUIRED
BASELINE_PENDING
BASELINE_READY
AFTER_PENDING
DIFF_PENDING
READY_FOR_REVIEW
ARCHIFY_BASELINE_FAILED
ARCHIFY_AFTER_FAILED
ARCHIFY_DIFF_FAILED
ARCHIFY_VALIDATION_FAILED
STALE
```

`READY_FOR_REVIEW` means only that architecture evidence is current and valid. It is never an
approval or merge verdict. Failure codes remain the exact public names required by the product
contract; diagnostics carry a bounded Archify code, element, message, supported correction, and
stage.

## Relevance policy

The policy is deterministic and versioned as `archify_relevance_v1`:

1. A changed tracked file under the configured `architecture_path` always yields required.
2. Otherwise an explicit `archify:required` task label yields required.
3. Otherwise an explicit stored human override yields its selected value and reason.
4. Otherwise labels `architecture`, `runtime`, `dataflow`, `persistence`, or `authority` yield
   required.
5. Otherwise the result is not required.

Conflicting labels fail closed as required. A not-required override cannot defeat rule 1. The
record stores the policy version, matched rule, evaluator, time, and override reason. Path matching
uses Git's changed-file list between the recorded base SHA and candidate head SHA, not filesystem
mtime or untracked session state.

This intentionally avoids module heuristics and configuration of arbitrary code path patterns.
Those can be added only if observed false negatives justify a new policy version.

## Capture service

### Baseline

At implementation admission, a task with an initially required policy result records the task's
repository ID and immutable base SHA from its task repository binding. The capture service calls the
revision-explicit architecture operation for every `*.architecture.json` directly under the
configured source path, validates with `--quality showcase`, and records source SHA-256 values and
receipt projections.

If a task is promoted to required by changed paths before review, the same operation materializes
the baseline from its already recorded base SHA. It never reads baseline bytes from the mutable
task checkout.

### After

Before review or approval admission, the service resolves a full commit SHA from the task's
canonical repository checkout. A dirty worktree cannot be represented by a SHA and therefore
cannot produce after evidence; the gate returns `ARCHIFY_AFTER_FAILED` until the relevant work is
committed. The service validates and renders sources from the immutable head commit. A missing
diagram on one side is represented to comparison as an addition or removal, not synthesized JSON.

The capture key is:

```text
(task_id, repository_id, architecture_path, side, commit_sha, runtime_contract)
```

`runtime_contract` is a stable hash of the configured runtime file and command contract. It prevents
cache reuse across an Archify runtime change without storing another runtime installation.

## Comparison

For diagram IDs present on both sides, execute:

```text
node <archify_runtime> compare architecture <base.json> <head.json> <delta.html>
  --receipt <delta.receipt.json> --json --quality showcase --repo-root <immutable-checkout>
```

The service projects Archify's receipt fields for completeness, semantic hashes, changes,
proof level, and validation. It never reimplements component, connection, boundary, authority, or
visual classification. Diagram additions and removals are computed from the two Git inventories;
the source hash and presence change are recorded and the diagram is classified semantic.

HTML, staged JSON, and complete raw receipts live only below the existing process temp/cache root,
keyed by repository, base SHA, head SHA, and diagram ID. The database stores content hashes, bounded
receipt JSON needed for audit, the structured summary, and cache locators that may expire. Missing
cache files are regenerated from the pinned SHAs.

## Lifecycle admission

```text
task start / work admission
  -> record base SHA
  -> evaluate initial relevance
  -> capture baseline when required
  -> admit implementation only after BASELINE_READY

transition to review or approval
  -> resolve candidate head SHA
  -> evaluate final relevance from labels, override, and Git changed paths
  -> if not required: persist NOT_REQUIRED and continue
  -> ensure baseline at recorded base SHA
  -> capture after at candidate head SHA
  -> compare
  -> atomically verify task source step and candidate head SHA
  -> admit only with READY_FOR_REVIEW
```

The transition preflight runs before the workflow step write. The final transition transaction
checks the evidence record's task ID, source step, base SHA, head SHA, and `READY_FOR_REVIEW` state.
If the step or head changed while Archify ran, the preflight result becomes `STALE` and the caller
retries; stale evidence never authorizes a transition. Manual moves, engine transitions, and task
state changes that enter a `review` or `approval` stage use the same precondition.

For workflows without stage types, entering task state `REVIEW` is the compatibility boundary.
No step-name matching is allowed. Entering `approval` is the human-gate boundary and always keeps
the existing decision requirement.

## Persistence

Add `task_architecture_evidence` with these stable fields:

```text
id, task_id, repository_id, architecture_path
policy_version, required, relevance_rule, override_actor_id, override_reason
base_sha, head_sha, runtime_contract
state, validation_status, architecture_changed, semantic_change
affected_diagrams_json, diff_summary_json, diagnostics_json
baseline_receipt_json, baseline_hash, after_receipt_json, after_hash
delta_receipt_json, delta_hash, cache_key
source_workflow_step_id, attempt, created_at, updated_at, completed_at
```

JSON fields are bounded evidence projections, not source JSON. They reject unknown top-level
receipt contracts and values beyond repository limits. `UNIQUE(task_id, repository_id,
architecture_path)` provides the current row; `attempt` increments on a real rerun. A separate
append-only `task_architecture_evidence_events` row records state transitions, reason codes, SHAs,
actor identity, and timestamp without artifact bodies. Both SQLite and PostgreSQL schemas use the
same constraints and cascade with task deletion.

The task's existing base revision contract supplies `base_sha`; this feature must not create a
second branch/base resolver. The workspace repository row supplies the Archify binding. Neither
value falls back to `cwd`, a current session, `origin/main`, or a default branch.

## API and review projection

Add authenticated task-scoped endpoints:

```text
GET  /api/v1/tasks/:taskID/architecture-evidence
POST /api/v1/tasks/:taskID/architecture-evidence/refresh
POST /api/v1/tasks/:taskID/architecture-evidence/override
GET  /api/v1/tasks/:taskID/architecture-evidence/delta/:diagramID
```

`GET` returns current state, relevance reason, SHAs, validation, summary, affected diagrams, and
stable `/architecture` and delta links. `refresh` reruns the required current phase without changing
Git. `override` requires `required`, a nonempty reason, and actor attribution. The delta endpoint
serves a cache artifact only after verifying its receipt hash; absent cache regenerates from the
pinned SHAs. Existing task authorization runs before repository or cache access.

State changes publish one task-scoped WebSocket event so open review surfaces update without
polling. The payload uses the same projection and does not contain source JSON or HTML.

## Review surface

The task Changes panel receives an **Architecture evidence** section above the code file list. It
shows required/state, validation, architecture changed, semantic change, base/head short SHAs,
affected diagrams, summary, Retry when allowed, an `/architecture` link, and a delta action.
Review and approval stages expand the section by default; implementation stages keep the same
information collapsed.

Phone uses one stacked card in the existing Changes panel scroll owner. Summary precedes diagram
actions; no desktop side rail is compressed into the phone view. All copy uses the task locale
namespace. The card's PASS state uses text and icon in addition to color.

## Failure and recovery

- Missing or ambiguous binding, repository, base SHA, or head SHA: fail closed with the phase's
  public failure state and a safe diagnostic.
- Validation failure: persist `ARCHIFY_VALIDATION_FAILED`; retain source identifiers and diagnostics,
  but do not compare or admit review.
- Runtime, render, or receipt mismatch: persist the corresponding baseline, after, or diff failure.
- Timeout: terminate the bounded subprocess, record a retryable diagnostic, and preserve last
  successful evidence as historical but not current.
- Backend restart: resume no subprocess. A pending record is retryable; pinned SHAs make rerun
  deterministic.
- Human not-required override: audit actor and reason. It never changes Git, Archify source, review
  verdicts, or human-gate requirements.

No failure is converted to not required. Retry never edits source or applies an Archify correction.

## Security and resource bounds

Reuse architecture binding validation, immutable Git archive extraction, argument-array subprocess
execution, and task/workspace authorization. Per-task capture is single-flight. Validation, render,
and compare use bounded time, output bytes, receipt bytes, and diagnostic counts. Cache paths derive
only from validated IDs and hashes. Raw local paths and subprocess output are logged only through
existing sanitization and are not returned to the browser.

## Observability

Structured logs include task ID, repository ID, phase, base/head SHA prefixes, policy rule, state,
attempt, duration, and Archify error code. Counters cover capture outcomes and review-admission
blocks by phase/reason; durations cover baseline, after, and compare. No task or repository IDs are
metric labels.

## Related decisions

- [SHA-pinned Archify task evidence](../../../decisions/2026-09-18-sha-pinned-archify-task-evidence.md)
- [Repository-bound Archify workspaces](../../../decisions/2026-09-17-repository-bound-archify-workspaces.md)
