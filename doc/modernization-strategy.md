# VisionOpus Modernization Strategy

## 1. Purpose

This document defines how to rebuild the legacy OpenEyes EMR using:

- Go for the main backend.
- React 19, strict TypeScript, Vite, TanStack Router, and TanStack Query for the
  web application.
- PostgreSQL for persistence.
- A possible FastAPI service for future AI-related capabilities only.

The immediate goal is not to reproduce every legacy file. The goal is to
preserve verified clinical behaviour while delivering a maintainable modern
system through controlled vertical slices.

## 2. Executive Decision

The existing extraction is sufficient to begin development, but it is not a
complete production specification.

The project will therefore use the following approach:

1. Stop broad, system-wide extraction.
2. Continue targeted extraction for one vertical slice at a time.
3. Convert verified legacy behaviour into testable acceptance criteria.
4. Implement that slice in Go, PostgreSQL, and React.
5. Compare the new behaviour with the legacy application.
6. Complete review and human sign-off before beginning another high-risk slice.

This is a just-in-time extraction and implementation strategy.

## 3. What The Existing Extraction Provides

The current documents provide a useful architectural and functional map of:

- Authentication and RBAC.
- Patient search and duplicate detection.
- Patient summary.
- Episodes and clinical events.
- Worklists and patient flow.
- Clinical examination.
- Prescriptions and medication administration.
- Consent and surgical workflows.
- EyeDraw integration.
- Major MySQL tables and a preliminary PostgreSQL mapping.
- Important controllers, models, views, widgets, and security risks.

These artifacts should be treated as research leads. The legacy source,
migrations, tests, fixtures, configuration, and runnable application remain the
authority.

## 4. Known Extraction Gaps

The current extraction does not justify a claim of complete OpenEyes parity.
Important incomplete or under-specified areas include:

- Some clinical element validation and calculation rules.
- Legacy test behaviour and fixtures.
- Background jobs and scheduled maintenance.
- Email and notification triggers.
- Investigations, laboratory results, and device imports.
- Correspondence and document management.
- Reporting and analytics.
- System administration.
- External APIs, PAS, Mirth, HL7, FHIR, and DICOM integrations.
- Data retention, archival, and operational recovery.
- Accessibility and multilingual requirements.
- Local deployment-specific LDAP and SSO behaviour.
- Stakeholder and clinician validation.

These gaps do not prevent the first slice from starting. They must be resolved
before implementing the slice that depends on them.

## 5. Targeted Extraction Process

### 5.1 Extraction Scope

For each vertical slice, inspect all relevant:

- Controllers and routes.
- Models, behaviours, helpers, and validators.
- Database migrations, constraints, indexes, and seed data.
- Unit, integration, feature, API, and browser tests.
- Fixtures and factories.
- Views, widgets, JavaScript, and visible workflow behaviour.
- RBAC permissions and user, institution, site, and firm context.
- State transitions and concurrency controls.
- Audit records.
- Background jobs and external integrations.
- Failure handling and clinical edge cases.

### 5.2 Required Slice Artifacts

Each slice should have the following documentation:

```text
doc/slices/<slice-name>/
├── slice.yaml
├── README.md
├── sources.yaml
├── work-queue.yaml
├── verified-behaviour.md
├── permission-matrix.md
├── state-transitions.md
├── data-mapping.md
├── acceptance-scenarios.md
├── api-contract.yaml
├── parity-tests.md
├── open-questions.md
├── evidence/
└── verification/
```

Create new slices from `doc/templates/slice/`. `slice.yaml` records lifecycle
status, risk tier, decision domains, owners, and durable approval metadata.

### 5.3 Evidence Rules

Every accepted legacy claim must include:

- Source path.
- Class, function, symbol, table, migration, or test name.
- Line range where practical.
- Confidence or verification status.
- Any relevant open question.

Permitted verification states are:

- `accepted`
- `rejected`
- `uncertain`

An AI-generated statement without source evidence is not a requirement.
Uncertain behaviour must remain uncertain until verified by source, a runnable
legacy environment, or an appropriate human owner.

## 6. Definition Of Ready

A slice is ready for implementation only when the team can answer:

1. Which users can perform every operation?
2. Which institution, site, firm, or team restrictions apply?
3. Which inputs are required, optional, valid, and invalid?
4. Which tables are read and written?
5. Which database invariants must always hold?
6. What are the valid state transitions?
7. What happens when two users modify the same record?
8. Which actions must be audited?
9. Which legacy data must be migrated?
10. How will migration completeness be reconciled?
11. Which legacy tests or observations demonstrate expected behaviour?
12. What are the performance, accessibility, and security expectations?
13. Which questions require clinical, product, governance, or legal decisions?

Do not use a percentage such as "95% complete" as the readiness gate.

The deterministic readiness gate is `./scripts/validate-slices`. Risk tiers and
human approval requirements are defined in `doc/slice-risk-policy.md`. Pending
clinical, prescribing, consent, migration, or security approval prevents a
slice from being marked `ready`.

## 7. Target Architecture

### 7.1 Initial Architecture

Use a modular monolith with explicit domain boundaries:

```text
modern-go-openeyes/
├── cmd/
│   ├── api/
│   ├── worker/
│   └── migrate/
├── internal/
│   ├── platform/
│   │   ├── audit/
│   │   ├── authn/
│   │   ├── config/
│   │   ├── database/
│   │   ├── http/
│   │   ├── observability/
│   │   └── security/
│   ├── auth/
│   ├── patient/
│   ├── episode/
│   ├── worklist/
│   ├── examination/
│   ├── prescription/
│   ├── surgery/
│   └── eyedraw/
├── api/
│   └── openapi/
├── db/
│   ├── migrations/
│   └── queries/
├── web/
├── doc/
└── test/
    ├── fixtures/
    ├── integration/
    └── parity/
```

Domain packages should not access another domain's database tables directly.
Cross-domain actions should go through explicit application interfaces.

### 7.2 Initial Technology Choices

- Go standard HTTP server with a small router where useful.
- REST JSON APIs.
- OpenAPI as the HTTP contract.
- `pgx` for PostgreSQL access.
- `sqlc` for type-safe query generation.
- Versioned SQL migrations.
- React 19, strict TypeScript, and Vite.
- TanStack Router with generated file-based routes.
- TanStack Query for remote server state.
- A generated TypeScript API client.
- PostgreSQL-backed integration tests.
- Playwright for critical browser workflows.
- Structured logging, metrics, tracing, and request correlation.

### 7.3 Decisions To Defer

Do not introduce these until a measured requirement exists:

- Microservices.
- gRPC.
- Kubernetes.
- Redis.
- Casbin.
- WebSockets.
- Event streaming infrastructure.
- Generic plugin frameworks.
- A FastAPI AI service.

For real-time worklists, first validate whether REST updates with polling or
server-sent events satisfy the operational requirement.

### 7.4 Data Modelling Principles

- Prefer relational tables for structured clinical information.
- Use JSONB for EyeDraw and genuinely variable, versioned payloads.
- Preserve clinically significant history rather than overwriting it.
- Represent amendment, cancellation, and invalidation explicitly.
- Enforce important invariants with PostgreSQL constraints where practical.
- Keep audit events append-only.
- Use transactions for multi-record clinical state changes.
- Avoid hidden side effects unless they are documented and tested.

## 8. First Walking Skeleton

The first end-to-end milestone is:

> Local authentication and user context -> patient search -> patient summary
> header

### 8.1 Included Scope

- Local test-user authentication.
- Secure server-side session handling.
- User, institution, site, and firm context.
- Basic RBAC enforcement.
- Append-only audit events.
- Search by primary patient identifier.
- Search by patient name and date of birth.
- Pagination.
- Allowlisted sorting.
- Duplicate-search behaviour.
- Patient summary header.
- Synthetic or properly de-identified test data.
- Legacy-to-Go parity tests.

### 8.2 Deferred Scope

- Production LDAP and SSO integration.
- Complete user administration.
- Redis-backed sessions.
- Advanced fuzzy-search tuning.
- Clinical record editing.
- Worklist real-time messaging.
- Live production data migration.
- AI functionality.

### 8.3 Walking Skeleton Success Gate

The milestone is complete only when:

- A user can log in and obtain the correct context.
- Unauthorized access is rejected and audited.
- Patient searches return expected legacy-equivalent results.
- The selected patient header renders correctly in React.
- Unit, integration, API, and browser tests pass.
- Migration and reconciliation tests pass on representative data.
- An independent review finds no unresolved critical issue.

## 9. Delivery Sequence

Use the following sequence unless verified dependencies require adjustment:

1. Repository and engineering foundation.
2. Authentication, RBAC, and user context.
3. Patient search and duplicate detection.
4. Patient summary.
5. Episodes and events.
6. Audit and clinical-record history hardening.
7. Worklists and patient flow.
8. Clinical examination.
9. Prescriptions and medication administration.
10. Consent, booking, and surgery.
11. EyeDraw integration.
12. Correspondence and document management.
13. Investigations, results, and device imports.
14. Reporting and analytics.
15. External integrations and background jobs.
16. Full migration rehearsals and staged rollout.

Do not set a production launch date until at least one representative slice has
passed extraction, implementation, migration, parity, security, and human
validation.

## 10. Per-Slice Development Workflow

Every slice follows the same controlled loop:

1. Define scope and exclusions.
2. Build the legacy source manifest.
3. Extract and verify legacy behaviour.
4. Resolve blocking product and clinical questions.
5. Write acceptance scenarios.
6. Define the API contract.
7. Design the PostgreSQL data model and migration mapping.
8. Write parity fixtures and expected results.
9. Implement the Go domain and application logic.
10. Implement PostgreSQL repositories and migrations.
11. Implement HTTP handlers.
12. Generate and use the TypeScript client.
13. Implement the React workflow.
14. Run unit, integration, parity, security, and browser tests.
15. Perform independent AI review.
16. Obtain human sign-off where required.
17. Merge through a reviewed pull request.

## 11. AI-Assisted Engineering Model

### 11.1 Core Principle

The AI subscriptions are engineering accelerators, not replacements for
ownership, verification, or clinical judgment.

Plan as one accountable engineering lead supported by multiple assistants, not
as 100 autonomous developers.

### 11.2 Default Tool Responsibilities

| Tool | Default responsibility |
| --- | --- |
| Claude Code | Legacy archaeology and source-cited specification |
| Codex | Go/React implementation, test execution, and debugging |
| Gemini | Independent verification, alternative review, and migration or UI analysis |

Roles may rotate to reduce dependence on one model. The author and final
reviewer should not be the same model for high-risk changes.

### 11.3 Branch And Worktree Rules

- One issue per branch.
- One Git worktree per active implementation task.
- Only one writing agent owns a worktree.
- Agents must not edit the same files concurrently.
- Review agents should inspect diffs without modifying the author's worktree.
- AI agents must not merge directly to `main`.
- Every merge must pass deterministic CI.
- Parallelize only independent tasks with explicit ownership boundaries.

### 11.4 Required Agent Output

An implementation agent must report:

- Files changed.
- Behaviour implemented.
- Assumptions made.
- Tests added.
- Commands executed.
- Test results.
- Remaining risks or open questions.

A reviewer must report findings ordered by severity with file and line
references.

### 11.5 Standard Task Packet

Every AI coding task should provide:

```text
Objective:
Legacy evidence:
Required behaviour:
Acceptance scenarios:
Files or modules owned:
API contract:
Database changes:
Security and audit requirements:
Tests required:
Explicit exclusions:
Completion command:
```

Avoid prompts such as "rebuild patient search." Give the agent a bounded,
verifiable task.

## 12. Quality Gates

### 12.1 Definition Of Done

A change is complete only when:

- Acceptance criteria are satisfied.
- Go unit tests cover domain rules and error paths.
- PostgreSQL integration tests cover queries and transactions.
- HTTP contract tests cover success and failure responses.
- React tests cover important component states.
- Playwright covers critical user workflows.
- Legacy parity tests pass where parity is required.
- Authorization and audit behaviour are tested.
- Concurrency behaviour is tested where relevant.
- Migration reconciliation is demonstrated.
- Formatting, linting, vulnerability checks, and race tests pass.
- Documentation and API contracts are updated.
- Independent review findings are resolved.
- Required human sign-off is recorded.

Coverage percentage alone is not a completion criterion.

### 12.2 Continuous Integration Baseline

CI should eventually enforce:

- Slice structure, evidence linkage, blocking-question, and approval validation.
- Go formatting and static analysis.
- Go unit tests.
- Go race detection.
- Dependency vulnerability scanning.
- PostgreSQL integration tests.
- Migration validation.
- OpenAPI validation and generated-client consistency.
- TypeScript type checking.
- Frontend unit tests.
- Playwright smoke tests.
- Secret scanning and dependency review.

## 13. Migration And Parity Strategy

### 13.1 Reference Environment

Maintain a runnable legacy OpenEyes environment as a behavioural reference.
Use synthetic or properly de-identified fixtures.

### 13.2 Migration Requirements

Each migrated domain needs:

- A legacy-to-PostgreSQL field mapping.
- Transformation rules.
- Handling of invalid and orphaned legacy records.
- Stable legacy identifiers or an explicit identifier crosswalk.
- Row-count reconciliation.
- Domain-specific aggregate reconciliation.
- Sample record comparison.
- Repeatable and idempotent import execution.
- Failure recovery and rollback procedures.
- Audit evidence for migration runs.

### 13.3 Parity Categories

Every legacy behaviour must be classified as:

- Preserved exactly.
- Preserved with an approved implementation change.
- Intentionally removed.
- Deferred.
- Unknown and blocking.

Security weaknesses must not be reproduced merely for parity.

## 14. Security, Privacy, And Clinical Governance

`doc/constitution.md` is the canonical authority for engineering and governance
principles. This section summarizes operational obligations and does not replace
the constitution.

- Never provide live patient data to consumer AI tools.
- Use synthetic or properly de-identified fixtures for AI-assisted work.
- Obtain data-protection, security, legal, and information-governance approval
  before any external AI service processes regulated information.
- Do not let AI make unreviewed clinical, consent, prescribing, or retention
  decisions.
- Keep secrets out of prompts, repositories, logs, fixtures, and screenshots.
- Record security-relevant architecture decisions.
- Preserve upstream licensing and attribution obligations.
- Obtain legal review of the project's AGPL compliance and distribution model.

## 15. Immediate Action Checklist

### Repository Foundation

- [ ] Initialize Git in `modern-go-openeyes`.
- [ ] Add `README.md`, `LICENSE`, `NOTICE`, and `.gitignore`.
- [ ] Confirm the project name and licensing policy.
- [ ] Add shared `AGENTS.md`, `CLAUDE.md`, and Gemini project instructions.
- [ ] Record the Go-first modular-monolith architecture decision.
- [ ] Create the initial Go, React, database, API, and test directories.
- [ ] Add local PostgreSQL development infrastructure.
- [ ] Add deterministic build and test commands.
- [ ] Create the initial CI workflow.
- [ ] Make a reviewed baseline commit.

### First Slice Preparation

- [ ] Create the authentication and patient-search source manifests.
- [ ] Review relevant legacy tests and fixtures.
- [ ] Complete the permission matrix.
- [ ] Complete the user, institution, site, and firm context model.
- [ ] Write acceptance scenarios.
- [ ] Define the initial OpenAPI contract.
- [ ] Define the PostgreSQL mapping.
- [ ] Create parity fixtures and expected outputs.
- [ ] Record unresolved SSO and deployment-specific questions.

### First Slice Implementation

- [ ] Implement the minimum authentication and session foundation.
- [ ] Implement RBAC enforcement and audit events.
- [ ] Implement read-only patient search.
- [ ] Implement the patient summary header.
- [ ] Run all completion gates.
- [ ] Perform independent AI review.
- [ ] Record the lessons before starting the next slice.

## 16. Current Non-Goals

The following are not part of the first milestone:

- Rebuilding every legacy module.
- Creating ten microservices.
- Introducing a FastAPI AI service.
- Production deployment.
- Migrating live patient data.
- Implementing full examination, prescribing, surgery, or EyeDraw workflows.
- Matching insecure legacy behaviour.
- Treating AI output as verified clinical requirements.

## 17. Next Milestone

The next milestone is complete when the repository has a reviewed Go-first
foundation and an implementation-ready specification for the authentication to
patient-search walking skeleton.

Only then should application implementation begin.
