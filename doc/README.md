# Documentation

## Governance

- `constitution.md`: Canonical engineering and governance principles.
- `slice-risk-policy.md`: Risk tiers, lifecycle gates, and human approvals.
- `templates/slice/`: Required starting structure for new vertical slices.
- `modernization-strategy.md`: Delivery, extraction, architecture, and migration
  strategy.

Validate all slice documentation with:

```bash
./scripts/validate-slices
```

## Strategy

- [Modernization strategy](modernization-strategy.md)

## Architecture Decisions

- [ADR-0001: React and TanStack Router frontend](decisions/0001-react-tanstack-router.md)

## Active Slice

- [01 - Authentication and user context](slices/01-auth-user-context/README.md)

## Status Vocabulary

- `discovery`: Legacy sources and scope are being identified.
- `specification`: Evidence is being verified and contracts are being prepared.
- `ready`: The Definition of Ready and required approvals have passed.
- `implementation`: Approved implementation is underway.
- `complete`: Implementation and the Definition of Done have passed.
- `deferred`: The slice is intentionally outside the current milestone.

Documentation must not use percentage-based completeness claims. A slice becomes
implementation-ready only by satisfying its explicit Definition of Ready.
