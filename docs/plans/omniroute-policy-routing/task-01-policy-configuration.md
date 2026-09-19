---
status: done
requirements:
  - REQ-AGENTS-OMNIROUTE-POLICY-001
acceptance_criteria:
  - AC-AGENTS-OMNIROUTE-POLICY-001.1
  - AC-AGENTS-OMNIROUTE-POLICY-001.2
  - AC-AGENTS-OMNIROUTE-POLICY-001.3
  - AC-AGENTS-OMNIROUTE-POLICY-001.4
  - AC-AGENTS-OMNIROUTE-POLICY-001.7
system_design:
  - docs/specs/agents/system-design/omniroute-policy-routing.md
---

# Task 01: Policy Configuration and Classification

## Outcome

Dynamic profiles can opt into the versioned policy, candidates carry validated
route classes, and task labels resolve deterministically to one task class.

## In scope

- Additive SQLite/PostgreSQL profile and candidate schema migration.
- Dynamic-profile DTO, CRUD, versioning, and validation changes.
- Reserved-label parser with default-simple and conflict failure.
- Feature-disabled and legacy-profile compatibility tests.

## Exclusions

- Route execution, quality commands, evidence API, UI, profile seeding, or edits
  to live operator configuration.

## Implementation acceptance

1. Invalid or incomplete active policy profiles fail before persistence or launch.
2. Existing profiles without `policy_kind` behave exactly as before.
3. Label parsing covers missing, each valid class, malformed JSON, and conflicts.

## Likely files

- `apps/backend/internal/agent/settings/models/dynamic.go`
- `apps/backend/internal/agent/settings/controller/`
- `apps/backend/internal/agent/settings/store/`
- `apps/backend/internal/task/models/models.go`
- persistence store-conformance fixtures

## Verification

```bash
make -C apps/backend test TEST_PKGS='./internal/agent/settings/... ./internal/task/...'
python3 scripts/list-docs.py validate
python3 scripts/lint-spec-files.py --all
git diff --check
```

## Results

Implemented on 2026-09-18. Dynamic profiles now persist optional
`omniroute_cost_first_v1` policy identity and per-candidate route classes.
Validation requires enabled local and free candidates, rejects unknown policy or
route classes, and leaves profiles without a policy unchanged. Task-label
classification defaults to simple, recognizes all four reserved labels, and
fails closed on malformed or conflicting values. Focused settings, routing,
provider-policy, classifier, and store-conformance tests pass.
