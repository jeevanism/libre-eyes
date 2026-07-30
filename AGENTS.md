# Go OpenEyes Agent Rules

These instructions apply to the entire repository.

## Current Phase

The project is in targeted reverse-engineering and repository preparation.
Do not implement application features unless the relevant slice satisfies its
Definition of Ready or the maintainer explicitly authorizes a bounded
foundation task.

## Target Stack

- Go for the primary backend.
- React and TypeScript for the frontend.
- PostgreSQL for persistence.
- REST JSON described by OpenAPI.
- No FastAPI service during the current milestone.

Use a modular monolith with explicit domain boundaries. Prefer manual
constructor injection and explicit wiring. Do not add a dependency injection
framework, microservices, gRPC, Redis, WebSockets, or Kubernetes without an
approved architecture decision.

## Source Authority

The legacy OpenEyes source is at `../../OpenEyes/openeyes`.

- Existing extraction documents are leads, not proof.
- Verify behaviour against source, migrations, tests, fixtures, configuration,
  or a runnable legacy environment.
- Cite the source path, symbol or table, and line range where practical.
- Mark unsupported or ambiguous claims `uncertain`; do not guess.
- Keep verified legacy facts separate from proposed rewrite decisions.
- Do not reproduce a legacy security weakness solely for parity.

## Extraction Workflow

Work one vertical slice at a time under `doc/slices/`.

Before implementation, a slice must document:

- Source manifest and completed work queue.
- Verified behaviour.
- Permission matrix.
- State transitions and concurrency rules.
- Legacy-to-PostgreSQL data mapping.
- Testable acceptance scenarios.
- Parity strategy.
- Open questions with owners or next actions.

## Engineering Rules

- Keep changes narrowly scoped to one task.
- Use one branch and worktree per implementation task.
- Do not let multiple agents edit the same files concurrently.
- Use manual, versioned SQL migrations and review them carefully.
- Prefer relational tables for structured clinical data.
- Reserve JSONB for EyeDraw and genuinely variable, versioned payloads.
- Keep clinically significant audit records append-only.
- Pass context through all I/O boundaries.
- Use parameterized SQL only.
- Use explicit constructors; avoid hidden mutable globals and `init()` side
  effects.
- Add tests proportional to clinical, security, concurrency, and migration
  risk.

## Required Agent Report

Every implementation report must include:

- Files changed.
- Behaviour implemented.
- Assumptions.
- Tests added.
- Commands executed and results.
- Remaining risks and open questions.

Reviews must present findings first, ordered by severity, with file and line
references.

## Privacy, Security, And Licensing

- Never place live patient data, secrets, credentials, or production exports in
  prompts, fixtures, logs, screenshots, or the repository.
- Use synthetic or properly de-identified development data.
- Clinical, prescribing, consent, retention, and security decisions require
  appropriate human review.
- Preserve applicable OpenEyes copyright and attribution when translating or
  closely adapting legacy material.
- Treat AGPL-3.0-only as the provisional licensing baseline pending maintainer
  and legal confirmation.
