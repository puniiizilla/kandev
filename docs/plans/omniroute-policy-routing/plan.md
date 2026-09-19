---
spec: docs/specs/agents/requirements/omniroute-policy-routing.md
created: 2026-09-17
status: complete
---

# Implementation Plan: OmniRoute Policy Routing

## Overview

Extend the existing dynamic conductor with an opt-in, cost-first OmniRoute
policy. Reuse task labels, complete concrete profiles, provider-error recovery,
route-generation fencing, and task usage events. Add no provider logic, model
catalogue, direct API client, UI, or workflow hook.

## Dependency order

1. [Task 01: Policy configuration and classification](task-01-policy-configuration.md) (done)
2. [Task 02: Escalation execution and quality results](task-02-escalation-execution.md) (done)
3. [Task 03: Evidence, telemetry linkage, and verification](task-03-routing-evidence.md) (done)

Tasks are sequential because Task 02 consumes Task 01's persisted policy and
Task 03 records Task 02's attempt lifecycle.

## Key risks

- A policy bypasses the existing effect-safety gate. Mitigation: conductor tests
  require safety evaluation before every retry or escalation.
- Usage arrives after an attempt settles. Mitigation: idempotent correlation and
  nullable telemetry, never a fabricated zero.
- Existing dynamic profiles regress. Mitigation: empty `policy_kind` preserves
  byte-for-byte current selection semantics in compatibility tests.
- `auto/*` effective identity differs from requested model. Mitigation: persist
  only runtime-reported provider/model attribution.

## Verification strategy

- Focused Go unit tests for label parsing, policy validation, eligibility, and
  escalation authorization.
- SQLite and PostgreSQL store-conformance tests for migrations, idempotency,
  joins, restart recovery, and paid-count derivation.
- Orchestrator integration tests using mock concrete profiles for all four route
  classes, provider failures, quality results, and effect-unsafe stops.
- API tests for authorization, field errors, evidence projection, and disabled
  behavior.
- Existing dynamic routing, task usage, and provider recovery suites remain green.
- Spec catalog/lint and `git diff --check` pass.

## Scope guard

No frontend files, provider adapters, OmniRoute code, credentials, network
configuration, workflow hooks, Office routing, or model catalogue may change.
