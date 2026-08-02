# VisionOpus Engineering Constitution

Version: 1.0.0

Ratified: 2026-08-02

Last amended: 2026-08-02

## Purpose And Authority

This constitution defines the non-negotiable engineering and governance rules
for VisionOpus. It is authoritative for specifications, implementation plans,
code, migrations, tests, reviews, and AI-assisted work.

If another repository document conflicts with this constitution, the
constitution takes precedence. Legal, regulatory, information-governance, and
approved clinical-safety obligations take precedence over every repository
document.

## 1. Evidence Before Implementation

- Legacy OpenEyes behaviour is a claim until verified against source code,
  migrations, configuration, tests, fixtures, or a controlled reference
  environment.
- Every accepted legacy claim must record a source path, symbol or database
  object, and a practical line range or equivalent locator.
- Unsupported or ambiguous claims remain `uncertain`; an AI-generated statement
  is never evidence by itself.
- Verified legacy facts must remain separate from proposed VisionOpus design
  decisions.
- A slice cannot enter implementation until its Definition of Ready passes.

## 2. Human Accountability For High-Risk Decisions

- AI tools may extract, propose, implement, and review, but they do not approve
  clinical, prescribing, consent, migration, or security decisions.
- A named human must approve every applicable area before a slice is marked
  `ready` or enters implementation.
- Clinical-safety decisions require an appropriately qualified clinical owner.
- Security, privacy, retention, licensing, and deployment decisions require the
  appropriate accountable owner.
- Approval records must identify the approver, decision date, scope, and durable
  evidence such as an ADR, reviewed issue, or signed governance record.

## 3. Go Is Authoritative For Server Behaviour

- The Go backend and PostgreSQL constraints are authoritative for
  authorization, clinical validation, calculations, state transitions, and
  audit creation.
- React may validate for usability but must not become the only enforcement
  point for a clinical or security rule.
- The initial system is a modular monolith with explicit domain boundaries.
- New services, protocols, infrastructure, or architectural frameworks require
  an approved architecture decision.

## 4. Data Integrity And History

- Prefer relational PostgreSQL tables for structured clinical and operational
  data.
- Reserve JSONB for EyeDraw and genuinely variable, versioned payloads.
- Preserve clinically significant history; represent amendment, cancellation,
  correction, and invalidation explicitly.
- Keep clinically and security-significant audit events append-only.
- Use transactions for multi-record state changes and database constraints for
  invariants where practical.
- Concurrency behaviour must be explicit and tested for every mutable workflow.

## 5. Security And Privacy By Construction

- Never place live patient data, secrets, credentials, or production exports in
  prompts, repositories, logs, fixtures, screenshots, or consumer AI tools.
- Use synthetic or properly de-identified development data.
- Enforce authorization server-side and use parameterized SQL exclusively.
- Do not reproduce an insecure legacy behaviour merely to achieve parity.
- Record security-relevant decisions and include abuse, failure, and audit
  scenarios in acceptance criteria.
- Preserve applicable OpenEyes licensing and attribution; legal confirmation is
  required before changing the provisional AGPL-3.0-only baseline.

## 6. Contract-Driven Full-Stack Delivery

- REST JSON contracts are described by OpenAPI and reviewed before dependent UI
  implementation.
- Generated TypeScript clients must remain consistent with the committed API
  contract.
- TanStack Router owns URL and validated search state; TanStack Query owns remote
  server state.
- React components render and coordinate workflows but do not own authoritative
  clinical calculations.
- Accessibility, error states, loading states, and authorization failures are
  part of the feature contract, not optional polish.

## 7. Explicit Migration And Parity

- Every legacy behaviour is classified as preserved exactly, preserved with an
  approved change, intentionally removed, deferred, or unknown and blocking.
- Each migrated domain must define field mappings, transformations, invalid and
  orphan handling, identifier reconciliation, repeatability, recovery, and
  aggregate reconciliation.
- Legacy evidence remains immutable historical truth. VisionOpus behaviour is
  maintained through OpenAPI, migrations, ADRs, implementation, and executable
  tests rather than by rewriting legacy evidence.
- Parity exceptions require a recorded reason and the applicable human approval.

## 8. Deterministic Quality Gates

- Repository structure, YAML syntax, evidence linkage, blocking questions, and
  approvals must be checked by automation.
- CI must run on pull requests and protected branches; required checks cannot be
  replaced by an AI assertion that work is complete.
- Tests must be proportional to clinical, security, concurrency, migration, and
  operational risk.
- Coverage percentage alone is not a completion criterion.
- A failing or bypassed required gate prevents merge.

## 9. Controlled AI Collaboration

- One writing agent owns a file set at a time; independent reviewers inspect a
  stable diff.
- The author and final reviewer must differ for high-risk and critical work.
- Agents work from bounded task packets and report changed files, assumptions,
  tests, commands, and remaining risks.
- Conversation history is not a durable handoff. Specifications, decisions,
  test results, and approvals must be committed artifacts or linked records.
- AI agents must not merge directly to a protected branch.

## Governance

### Amendments

Constitution changes require:

1. A pull request describing the reason and affected rules.
2. Review by the repository maintainer.
3. Human review from each affected clinical, security, migration, legal, or
   governance area.
4. A version update using semantic versioning:
   - Major for removed or materially weakened principles.
   - Minor for new principles or materially expanded obligations.
   - Patch for clarifications that do not change obligations.

### Exceptions

An exception must be narrow, time-bound, and recorded in an ADR or tracked
decision. It must state the rule, justification, risk owner, compensating
controls, expiry condition, and approval. Exceptions cannot waive legal or
clinical-safety obligations.

### Compliance Review

Every slice review must confirm its risk classification, constitution
compliance, required approvals, and deterministic validation result. The
constitution is reviewed at least at each production milestone and whenever a
material architecture, deployment, clinical, security, or migration decision
changes.
