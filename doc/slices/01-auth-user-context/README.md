# Slice 01: Authentication And User Context

## Status

`specification`

This workspace prepares the verified specification for the authentication,
session, RBAC, and user-context portion of the first walking skeleton.

No application implementation is authorized by this workspace yet.

## Scope

- Local username and password authentication.
- Password status, expiry, failure, and lockout behaviour.
- Session creation, regeneration, timeout, and revocation.
- User, institution, site, firm, and related context selection.
- RBAC roles, tasks, operations, inheritance, and administrative restrictions.
- Authentication and authorization audit behaviour.
- Electronic-signature PIN dependencies that constrain the auth design.
- Legacy account and authorization data migration requirements.

## Explicitly Deferred

- Customer-specific LDAP schemas and infrastructure.
- Production SSO provider integration.
- Complete user-administration UI.
- Clinical feature permissions beyond the representative rules needed to verify
  RBAC inheritance.
- The modern API design, except for a placeholder contract awaiting verified
  behaviour.

## Workspace Files

- `sources.yaml`: Initial legacy source inventory.
- `work-queue.yaml`: Bounded extraction and verification tasks.
- `verified-behaviour.md`: Only independently accepted legacy facts.
- `permission-matrix.md`: Actor and operation authorization evidence.
- `state-transitions.md`: Authentication and session state machines.
- `data-mapping.md`: Legacy tables and eventual PostgreSQL mapping decisions.
- `acceptance-scenarios.md`: Candidate executable requirements.
- `api-contract.yaml`: Placeholder OpenAPI document for later design.
- `parity-tests.md`: Legacy-versus-new comparison plan.
- `open-questions.md`: Unresolved decisions with owners or next actions.
- `evidence/README.md`: Required evidence record format.
- `verification/README.md`: Independent verification record format.

## Extraction Order

1. Verify the core login and local password flow.
2. Verify session lifecycle and context switching.
3. Verify RBAC evaluation and inheritance.
4. Verify institution, site, and firm context.
5. Verify PIN and electronic-signature dependencies.
6. Verify relevant schema history, seed data, tests, and fixtures.
7. Convert accepted facts into state transitions and acceptance scenarios.
8. Resolve or explicitly defer blocking open questions.

## Definition Of Ready

This slice is ready for implementation only when:

- All required source groups have a completed work-queue item.
- Every behavioural statement in `verified-behaviour.md` has an accepted
  verification record.
- Login, lockout, password, session, and context state transitions are explicit.
- The permission matrix covers the walking-skeleton operations.
- Required audit events are explicit.
- Migration mapping and reconciliation rules are defined.
- Acceptance and parity scenarios are testable.
- Deferred LDAP and SSO behaviour has an approved boundary.
- Blocking product, security, governance, and licensing questions have owners
  and recorded decisions.
