# ADR-2026-09-17: Keep OmniRoute as the Provider and Model Routing Authority

**Status:** proposed
**Date:** 2026-09-17
**Area:** agents, tasks, persistence, observability

## Context

Kandev already owns dynamic profile fallback and effect-safe provider-error
recovery. OmniRoute already exposes verified local, free, low-cost, and strong
routes. Adding task-aware cost escalation can accidentally create a second
model router if Kandev discovers, ranks, prices, or directly invokes providers.

The policy also needs durable quality authorization and cost evidence. Existing
task usage events already own token and cost facts, so copying those values into
a second ledger would create conflicting accounting.

## Decision

Kandev selects only an ordered route class and one configured concrete profile.
That profile targets OmniRoute. OmniRoute remains responsible for resolving its
route alias to the effective provider and model.

Kandev may persist:

- explicit task classification,
- candidate route class and order,
- escalation authorization and reason,
- route-attempt identity and outcome,
- effective provider/model attribution reported by the runtime, and
- a reference to the authoritative task usage event.

Kandev does not maintain a model catalogue, probe provider availability, rank
models, calculate an independent price for routing evidence, or call a provider
directly. Paid-attempt counts are derived from immutable strong-route attempts.

Existing generic dynamic profiles remain valid. Cost-first behavior is opt-in
through a versioned policy kind, and its feature-disabled path preserves stored
configuration without executing it.

## Consequences

- Existing dynamic conductor, provider-error classifier, circuits, and usage
  ledger are reused.
- Deployment configuration supplies the four concrete OmniRoute profiles; model
  identifiers are not compiled into routing logic.
- `auto/*` provider choice can change without a Kandev release while historical
  evidence retains the effective reported provider/model.
- A missing OmniRoute binding or ambiguous task class fails closed.
- Quality producers remain outside routing; the router consumes explicit,
  attributable results instead of becoming a model judge.

## Alternatives considered

### Build a Kandev model router

Rejected. It duplicates OmniRoute authority, provider health, pricing, and
catalogue lifecycle.

### Store tokens and cost on every route attempt

Rejected. The task usage ledger already owns these facts and their provenance.
Routing evidence links to that ledger.

### Infer risk and quality with a model

Rejected for v1. It adds another paid inference path and makes escalation
authorization non-deterministic. Explicit task labels and trusted quality
results are auditable.

## Related specifications

- [OmniRoute Policy Routing Requirements](../specs/agents/requirements/omniroute-policy-routing.md)
- [OmniRoute Policy Routing System Design](../specs/agents/system-design/omniroute-policy-routing.md)
