---
status: draft
system: agents
requirements:
  - REQ-AGENTS-OMNIROUTE-POLICY-001
  - REQ-AGENTS-OMNIROUTE-POLICY-002
  - REQ-AGENTS-OMNIROUTE-POLICY-003
---

# OmniRoute Policy Routing System Design

## Purpose and boundaries

This design extends the existing `dynamic` conductor with a policy mode. Kandev
decides which configured route classes are eligible and when escalation is
authorized. Each candidate is still a complete concrete agent profile;
OmniRoute resolves aliases such as `auto/best-free` to providers and models.
No Kandev component probes or ranks an OmniRoute model catalogue.

The task system remains the owner of labels and task lifecycle. The task usage
ledger remains the owner of token and cost facts. Provider-error recovery remains
the owner of rate-limit, quota, timeout, effect-safety, retry, and circuit rules.

## Requirement mapping

| Requirement | Design section |
| --- | --- |
| `REQ-AGENTS-OMNIROUTE-POLICY-001` | [Policy configuration](#policy-configuration), [Selection flow](#selection-flow) |
| `REQ-AGENTS-OMNIROUTE-POLICY-002` | [Quality results](#quality-results), [Failure behavior](#failure-behavior) |
| `REQ-AGENTS-OMNIROUTE-POLICY-003` | [Evidence and usage linkage](#evidence-and-usage-linkage), [Persistence](#persistence) |

## Existing components reused

- `internal/agent/runtime/dynamic` remains the sole cross-candidate conductor.
- `internal/agent/runtime/routingerr` classifies provider failures.
- `internal/agent/runtime/routingpolicy` owns retry/wait/exhaustion evaluation.
- `internal/agent/settings` validates and stores dynamic profile candidates.
- `internal/task/repository` stores route state, route attempts, and task usage.
- `internal/task/usage` normalizes tokens and provider-reported cost.

No new server, provider adapter, routing daemon, or model registry is added.

## Policy configuration

`dynamic_agent_profiles` gains a nullable `policy_kind`. Empty preserves current
behavior; `omniroute_cost_first_v1` activates this contract. The existing
version guards policy edits.

Each `dynamic_agent_routes` record gains a nullable `route_class`. It is required
for every enabled candidate when the profile policy is active and must be one of
`local`, `free`, `low_cost`, or `strong`. The candidate continues to reference a
complete concrete profile. Model IDs and OmniRoute endpoint/credential bindings
remain in that profile, not in policy code.

Validation requires at least one `local` and one `free` candidate. It rejects
unknown classes, missing classes, nested dynamic profiles, `AutoFallback`
candidates, duplicate candidates, and a non-OmniRoute concrete profile. The
latter uses the existing resolved endpoint and secret-reference metadata; it
does not call a provider or inspect credential values.

The initial configured profile is expected to reference concrete profiles for:

1. `local`: `ollama/qwen2.5-coder:14b`
2. `free`: `auto/best-free`
3. `low_cost`: `auto/cheap`
4. `strong`: `codex/gpt-5.6-sol`

These are deployment data, not constants. Profile IDs and order are returned by
the existing profile API. Policy activation uses the existing runtime feature
flag and profile enablement; no second feature flag is needed.

## Task classification

The classifier parses the existing `Task.Labels` JSON array at the trust
boundary. Reserved labels map exactly:

- `routing:simple` to `simple`
- `routing:medium` to `medium`
- `routing:high-risk` to `high_risk`
- `routing:high-complexity` to `high_complexity`

No reserved label means `simple`. More than one fails closed. The resolved class
and source (`default` or `label`) are snapshotted into route state and evidence;
later label edits affect only the next safe route decision. This reuses task
labels and avoids another task classification column.

## Selection flow

At a new route decision the conductor:

1. Loads the selected dynamic profile and validates active policy configuration.
2. Loads the task, parses its classification, and snapshots it.
3. Filters configured candidates by the allowed route-class sequence.
4. Applies existing enabled/profile-scope/circuit eligibility in configured order.
5. Persists the selected candidate and attempt before launch.
6. Launches the complete concrete profile through the existing ACP path.

Allowed sequences are:

- `simple`: `local`, `free`; `strong` becomes eligible only with a durable
  `cheap_chain_exhausted` or quality authorization.
- `medium`: `local`, `free`, `low_cost`; `strong` becomes eligible only with a
  durable `cheap_chain_exhausted` or quality authorization.
- `high_risk` and `high_complexity`: `strong` is immediately eligible. If the
  configured order places cheaper candidates first, that order is respected;
  the class permits strong use but does not force it.

Kandev chooses only a candidate profile. For `auto/*`, OmniRoute chooses the
provider/model. For the explicit local and strong identifiers, OmniRoute still
receives the request through the profile's configured endpoint.

## Quality results

Add a backend domain command consumed by the orchestrator:

```text
RecordRouteQualityResult(task_id, session_id, route_attempt_id,
                         result, source, source_reference)
```

`source` is a closed enum for existing trusted producers (`workflow`, `review`,
`verification`, `operator`). `source_reference` is a bounded opaque identifier,
not free-form diagnostic text. The command checks task/session/attempt ownership,
current generation, and idempotency before persisting.

The conductor maps the result as follows:

- `pass`: terminal quality record; no route change.
- `retry`: same candidate through existing retry/effect-safety rules.
- `fail`: quality failure; advance to the next eligible candidate and authorize
  `strong` if no cheap candidate remains.
- `escalate`: explicit advance authorization, including `strong`.

The quality command does not judge output. Existing workflow/review/verification
owners decide when to emit it. No workflow hook is added by this package.

## Failure behavior

Provider failures retain the existing `routingerr.Code` and `routingerr.Class`.
Evidence additionally records a closed failure category:

- `rate_limit`
- `quota`
- `timeout`
- `provider_other`
- `quality_gate`
- `configuration`

The first four derive only from the existing classifier. Quality commands can
produce only `quality_gate`. Invalid profile policy or classification produces
`configuration`. Unknown or ambiguous errors stop; they never become quality
failures. The existing effect-safety gate always precedes retry or escalation.

## Evidence and usage linkage

Extend `dynamic_route_attempts` with stable attempt identity, `task_id`, selected
route/profile ID, route class, task class/source, attempt ordinal, escalation
reason, failure category, quality result/source, effective provider/model,
latency, and timestamps. Nullable provider/model/latency fields remain absent
until observed.

Add a nullable unique `usage_event_id` foreign reference to
`task_usage_events.usage_event_id`. Token counts and cost are projected by join;
they are not copied into the route table. This preserves `task_usage_events` as
the only cost and token ledger and retains `cost_source`, `estimated`, and
provider-reported-zero semantics.

The runtime correlates usage to the current attempt using task, session, turn,
and route generation already carried by the turn snapshot. Effective provider
and model come from the usage/turn attribution contract, never the requested
profile model alone. A finalized attempt without observable usage remains
auditable with null telemetry.

`ListTaskRoutingEvidence(taskID)` returns ordered evidence with joined usage and
a derived `paid_escalation_count`. The count is `COUNT(DISTINCT attempt_id)` for
attempts whose snapshotted route class is `strong`; no mutable counter column is
stored.

## Persistence

SQLite and PostgreSQL migrations are additive. Existing dynamic profiles have
empty `policy_kind`; existing candidates have null route class and retain old
behavior. Existing attempt rows receive generated stable IDs where absent and
nullable policy fields. No historical route class, provider, model, or cost is
guessed.

Route selection and attempt insertion remain in the generation-fenced
transaction. Quality-result insertion and the authorized transition share one
transaction and an idempotency key. Usage linkage is idempotent and may occur
after turn completion. Restart reconciliation reads the snapshotted task class,
authorization, and policy version rather than reinterpreting old evidence.

## API contract

Existing dynamic-profile create/update responses add optional `policy_kind` and
candidate `route_class`. Omitting both preserves compatibility.

Backend-only v1 adds:

- `POST /api/v1/tasks/:taskID/routing-quality-results`
- `GET /api/v1/tasks/:taskID/routing-evidence`

Writes require the existing task mutation authorization. Reads require task
read access. Validation failures are field-addressable. The evidence endpoint
returns policy/classification snapshots, attempts, joined usage, and the derived
paid count. No UI is part of this package.

## Observability

Structured logs and bounded metrics record route class, task class, result,
failure category, and escalation reason. They never include credentials,
prompts, task text, or unbounded task/profile identifiers as metric labels.
Counters cover attempts, cheap-chain exhaustion, quality-authorized escalation,
strong selections, and rejected quality commands. Latency uses bounded
histograms. Durable evidence, not logs, is the audit source.

## Security

Only complete stored profiles can become candidates. Endpoint validation uses
non-secret configuration metadata. Quality results are ownership-checked,
generation-fenced, idempotent, and source-typed. Evidence APIs apply workspace
task authorization. Secret references and environment values are never stored
in routing evidence.

## Related decisions

- [Keep OmniRoute as routing authority](../../../decisions/2026-09-17-omniroute-policy-authority.md)
- [Unify provider routing behind dynamic profiles](../../../decisions/2026-08-13-dynamic-agent-profile-routing.md)
- [Classify provider errors before policy](../../../decisions/2026-08-17-provider-error-classes-and-policies.md)
