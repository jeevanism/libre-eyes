# ADR-0002: Defer Spec Kit Adoption

- Status: Accepted
- Date: 2026-08-02

## Context

VisionOpus already has an evidence-driven specification process under
`doc/slices/`, a canonical constitution, risk and approval metadata,
repository-owned templates, and deterministic CI validation. GitHub Spec Kit
was proposed as optional scaffolding beneath this process.

A bounded evaluation used official Spec Kit 0.15.1 on the isolated
`experiment/spec-kit-foundation` branch. It generated a specification for a
foundation-only slice-scaffolding workflow and installed Claude Code, Codex,
and Gemini integrations.

## Decision

Do not adopt Spec Kit as the project specification authority or default
workflow now.

- `doc/constitution.md` remains the only engineering constitution.
- `doc/slices/` remains the only authoritative slice specification tree.
- `doc/templates/slice/` remains the canonical slice scaffold.
- `./scripts/validate-slices` and CI remain the deterministic gates.
- `.specify/`, a parallel `specs/` tree, and generated Spec Kit agent adapters
  must not be added to `visionopus` without a superseding ADR.

The experiment is retained only on the remote evaluation branch for audit:

`https://github.com/jeevanism/libre-eyes/tree/experiment/spec-kit-foundation`

## Rationale

The evaluation produced useful user stories, edge cases, acceptance scenarios,
and checklists. However, out-of-box adoption also created:

- A second placeholder constitution.
- A competing `specs/` feature tree.
- Generic templates without VisionOpus evidence, parity, migration, permission,
  transition, risk, or approval structures.
- A tasks template in which tests are optional.
- Agent adapters without enforcement of reviewer independence or human
  approval.
- Gemini commands under the repository's intentionally ignored `.gemini/`
  directory.
- Additional mutable active-feature state unsuitable as a substitute for branch
  and worktree ownership.

Customizing these elements currently duplicates repository-native capabilities
without replacing the validator, CI, or human governance.

## Consequences

- No Spec Kit runtime or generated files are maintained on `visionopus`.
- Contributors use the repository-native extraction and specification workflow.
- Useful ideas from the experiment may be implemented directly in the existing
  templates or validator without adopting Spec Kit.
- The decision may be revisited after multiple completed slices demonstrate a
  measurable process gap, provided a custom integration can consume the
  canonical constitution and slice schema without copied policy or parallel
  specifications.
