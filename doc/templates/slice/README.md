# {{SLICE_TITLE}}

## Status

`discovery`

## Scope

- {{IN_SCOPE_BEHAVIOUR}}

## Explicitly Excluded

- {{OUT_OF_SCOPE_BEHAVIOUR}}

## Definition Of Ready

- The source manifest and work queue are complete.
- Accepted claims have source-cited evidence and independent verification.
- Permission, state, concurrency, audit, data, migration, and parity behaviour
  is explicit or marked not applicable with a reason.
- Acceptance scenarios and API contracts are testable.
- Blocking questions are resolved.
- Required human approvals in `slice.yaml` are approved.
- `./scripts/validate-slices` passes without readiness warnings.
