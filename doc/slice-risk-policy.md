# Slice Risk And Approval Policy

## Purpose

Every slice declares a risk tier and applicable decision domains in
`slice.yaml`. The tier determines review depth; the domains determine mandatory
human approvals. Risk classification never reduces legal, privacy, licensing,
or clinical-safety obligations.

## Risk Tiers

### Critical

Use `critical` when failure could directly cause patient harm, invalidate
consent, create an unsafe prescription or calculation, misidentify a patient,
or irreversibly corrupt clinically significant information.

Minimum controls:

- Full evidence and independent verification.
- Complete permission, state, concurrency, audit, migration, and parity models.
- Negative, boundary, concurrency, migration, and browser acceptance coverage.
- Independent technical review and every applicable human approval.
- Explicit rollback, recovery, and operational monitoring plan.

Typical examples include prescribing, consent, surgical workflow, patient
identity matching, and clinically authoritative calculations.

### High

Use `high` when failure could expose regulated data, bypass authorization,
damage auditability, corrupt migrated data, disrupt clinical operations, or
break an external clinical integration.

Minimum controls:

- Full evidence and independent verification for affected behaviour.
- Complete security, data, audit, failure, migration, and concurrency analysis
  where applicable.
- Independent technical review and every applicable human approval.
- Integration and end-to-end tests for the affected boundary.

Typical examples include authentication, RBAC, audit infrastructure, clinical
data migration, worklist state, and HL7/FHIR/DICOM boundaries.

### Standard

Use `standard` for user-facing or operational behaviour without a direct
clinical decision, privileged security boundary, or regulated-data migration.

Minimum controls:

- Evidence appropriate to the preserved behaviour.
- Acceptance criteria, contract coverage, authorization analysis, and tests.
- Product or domain review where behaviour changes.

Reporting is not automatically standard: reports containing clinical data,
patient-identifying data, or operational safety signals may be high risk.

### Foundation

Use `foundation` only for tooling, documentation, local developer experience,
or isolated scaffolding that cannot process production data or alter runtime
clinical behaviour.

Minimum controls:

- Clear scope and exclusions.
- Automated tests or deterministic validation appropriate to the tool.
- Maintainer review.

Deployment, secrets, identity, authorization, audit, database migration, and
production observability work must not be classified as foundation merely
because it is infrastructure.

## Decision Domains

Allowed decision domains are:

- `clinical`
- `prescribing`
- `consent`
- `migration`
- `security`
- `privacy`
- `legal`
- `product`
- `operations`
- `accessibility`

The following domains always require a named human approval before `ready`:

- `clinical`
- `prescribing`
- `consent`
- `migration`
- `security`

Privacy and legal approval is also mandatory when the slice changes regulated
data processing, retention, external data transfer, licensing, or distribution.
The validator enforces declared approval requirements, while reviewers remain
responsible for detecting missing domains.

## Slice Lifecycle

Allowed slice statuses are:

1. `discovery`
2. `specification`
3. `ready`
4. `implementation`
5. `complete`
6. `deferred`

Structural validation applies at every status. A slice at `ready`,
`implementation`, or `complete` additionally fails validation when:

- A blocking question is unresolved.
- An evidence claim lacks an accepted, rejected, or uncertain verification.
- An accepted legacy claim lacks its evidence and accepted verification record.
- A required source group is not verified or explicitly excluded.
- A required human approval is missing, pending, agent-authored, or lacks durable
  decision evidence.

## Approval Records

Approvals are recorded in `slice.yaml`. An approved entry must include:

- Decision area.
- `status: approved`.
- `approver_kind: human`.
- Named approver or accountable role.
- ISO 8601 decision date.
- Durable evidence reference.
- Scope or notes sufficient to understand what was approved.

AI verification records and AI code review do not satisfy a human approval.

## Classification Changes

Risk may increase as information is discovered. Lowering a tier or removing a
decision domain requires a recorded rationale and maintainer approval. A slice
must be revalidated after every classification or approval change.
