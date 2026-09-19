---
status: draft
system: agents
created: 2026-09-17
owners:
  - kandev
---

# OmniRoute Policy Routing Requirements

## Overview

Kandev can apply a cost-first escalation policy to a dynamic agent profile while
OmniRoute remains the authority that resolves an allowed route to a provider and
model. The agent system owns the policy because it already owns dynamic profile
selection, provider-error recovery, route attempts, and immutable execution
attribution. Tasks supply explicit classification labels and quality-gate
results; they do not select providers.

## Terminology

- **Task class:** One of `simple`, `medium`, `high_risk`, or `high_complexity`.
- **Route class:** One of `local`, `free`, `low_cost`, or `strong`.
- **Cheap chain:** The ordered non-strong candidates allowed for a task class.
- **Quality result:** One of `pass`, `fail`, `retry`, or `escalate`, recorded by
  an existing workflow, review, or verification owner rather than inferred by
  the router.
- **Paid escalation:** Selection of a candidate classified as `strong`.

## Requirements

### REQ-AGENTS-OMNIROUTE-POLICY-001: Activatable cost-first policy

**Intent:** Express task-aware route eligibility without creating a Kandev
provider catalogue or replacing OmniRoute routing.

#### Acceptance criteria

- **AC-AGENTS-OMNIROUTE-POLICY-001.1:** When the policy is disabled, existing
  concrete and dynamic profiles shall retain their current selection and
  recovery behavior without requiring route-class metadata.
- **AC-AGENTS-OMNIROUTE-POLICY-001.2:** When the policy is enabled for a dynamic
  profile, every enabled candidate shall reference one complete concrete
  profile and declare exactly one route class. Kandev shall not copy provider
  credentials, resolve a provider catalogue, or issue a direct model-provider
  request.
- **AC-AGENTS-OMNIROUTE-POLICY-001.3:** When a task has no routing-class label,
  it shall be treated as `simple`. Exactly one of `routing:simple`,
  `routing:medium`, `routing:high-risk`, or `routing:high-complexity` may be
  present. Conflicting routing-class labels shall fail closed before launch.
- **AC-AGENTS-OMNIROUTE-POLICY-001.4:** A `simple` task shall try eligible
  `local` candidates and then eligible `free` candidates. A `medium` task shall
  try `local`, `free`, and then `low_cost`. Candidate order within a route class
  shall remain the configured dynamic-profile order.
- **AC-AGENTS-OMNIROUTE-POLICY-001.5:** A `high_risk` or `high_complexity` task
  may select a `strong` candidate immediately. A `simple` or `medium` task shall
  select `strong` only after its cheap chain is exhausted or after a documented
  quality result of `escalate` or `fail` authorizes escalation. A retry result
  shall remain on the current eligible route while its retry policy permits.
- **AC-AGENTS-OMNIROUTE-POLICY-001.6:** If no eligible candidate exists, policy
  configuration is invalid, classification is ambiguous, or escalation lacks
  one of the permitted reasons, the route shall enter an actionable stopped
  state and shall not silently select a strong or direct-provider profile.
- **AC-AGENTS-OMNIROUTE-POLICY-001.7:** The first released policy shall support
  concrete profiles configured for `ollama/qwen2.5-coder:14b`,
  `auto/best-free`, `auto/cheap`, and `codex/gpt-5.6-sol` without embedding
  those model identifiers in runtime selection code.

### REQ-AGENTS-OMNIROUTE-POLICY-002: Quality-controlled escalation

**Intent:** Make paid escalation deliberate, bounded, and attributable.

#### Acceptance criteria

- **AC-AGENTS-OMNIROUTE-POLICY-002.1:** A quality result shall identify the
  task, session or run, route attempt, result, source, and recorded time. Missing,
  stale, duplicated, or cross-task results shall not authorize escalation.
- **AC-AGENTS-OMNIROUTE-POLICY-002.2:** `pass` shall complete the quality
  decision without changing route; `retry` shall request the same candidate;
  `fail` shall record a quality failure and may authorize the next eligible
  route; `escalate` shall explicitly authorize the next eligible route,
  including `strong` when policy permits.
- **AC-AGENTS-OMNIROUTE-POLICY-002.3:** Provider rate-limit, quota, and timeout
  failures shall remain separate failure categories from quality failures and
  shall continue through the existing provider-error classifier and effect-
  safety gate.
- **AC-AGENTS-OMNIROUTE-POLICY-002.4:** Automatic retry or cross-candidate
  escalation shall stop when output or effects make replay ambiguous, regardless
  of task class or quality result.

### REQ-AGENTS-OMNIROUTE-POLICY-003: Durable routing evidence

**Intent:** Let operators audit route selection, escalation, usage, cost, and
latency without creating another usage or pricing authority.

#### Acceptance criteria

- **AC-AGENTS-OMNIROUTE-POLICY-003.1:** Each route attempt shall persist its
  task ID, selected route, route class, provider, model, attempt ordinal,
  escalation reason, failure category, token usage, reported cost, latency,
  and timestamp. Unknown telemetry shall remain explicitly absent rather than
  estimated as zero.
- **AC-AGENTS-OMNIROUTE-POLICY-003.2:** Provider and model attribution shall be
  the effective values reported through the existing agent/usage contracts.
  Kandev shall not infer them from a model catalogue or profile display name.
- **AC-AGENTS-OMNIROUTE-POLICY-003.3:** Reported cost and tokens shall reuse the
  task usage ledger and its source semantics. Routing evidence may reference a
  usage event but shall not copy or independently price the same usage.
- **AC-AGENTS-OMNIROUTE-POLICY-003.4:** The paid-escalation count for a task
  shall equal its distinct persisted attempts that selected a `strong`
  candidate. Retries and replayed events shall not double count one attempt.
- **AC-AGENTS-OMNIROUTE-POLICY-003.5:** After restart, route policy state,
  escalation authorization, attempt history, and paid-escalation count shall
  remain queryable and shall preserve their original policy and classification
  snapshots.

## Out of scope

- Provider selection inside an OmniRoute alias or an independent model catalogue.
- Direct provider API calls, new credentials, credential removal, or network rules.
- Model-based task classification or model-based quality judging.
- New agent platforms, memory systems, workflow hooks, or Office-specific routing.
- A settings or audit user interface in the first implementation package.
