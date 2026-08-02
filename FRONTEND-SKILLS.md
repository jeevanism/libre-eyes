# Frontend Skill Policy

This file defines the skill routing and non-negotiable engineering standards
for the VisionOpus React frontend.

## Project Skill

Always load the project-local skill:

- `.agents/skills/visionopus-react/SKILL.md`

Its references contain the detailed React and TanStack Router rules. Project
rules override generic frontend guidance.

## Workflow Skill Provenance

The reviewed general workflow bundle is:

- Project: `addyosmani/agent-skills`
- License: MIT
- Reviewed source: `/home/jeevanism/Documents/Projects/AI-Engineering/agent-skills/agent-skills`
- Reviewed revision: `d187883b7d761265309cdcc0f202cc76b4b3fb06`
- Revision date: 2026-06-10

Use an equivalent tool-native skill when available. Otherwise read the named
`SKILL.md` directly from the reviewed source.

## Mandatory Frontend Skills

Load these for every frontend implementation or review:

| Skill | Purpose |
| --- | --- |
| `visionopus-react` | Enforces the approved React and TanStack architecture. |
| `frontend-ui-engineering` | Applies accessible, production-quality UI patterns. |
| `source-driven-development` | Rechecks version-sensitive framework behaviour against official documentation. |
| `test-driven-development` | Requires behavioural tests around each change. |
| `browser-testing-with-devtools` | Verifies rendering, interaction, console, and network behaviour in a real browser. |
| `code-review-and-quality` | Reviews correctness, simplicity, architecture, security, and performance before completion. |

## Task Routing

| Task | Additional skills |
| --- | --- |
| New feature or workflow | `spec-driven-development`, `incremental-implementation`, `context-engineering` |
| OpenAPI client or frontend/backend boundary | `api-and-interface-design`, `source-driven-development` |
| Authentication or sensitive patient UI | `security-and-hardening`, `doubt-driven-development` |
| UI defect or browser failure | `debugging-and-error-recovery`, `browser-testing-with-devtools` |
| Performance concern | `performance-optimization` only after browser measurements identify a bottleneck |
| Framework or dependency upgrade | `deprecation-and-migration`, `source-driven-development` |
| Architecture decision | `documentation-and-adrs`, `doubt-driven-development` |

## Approved Frontend Baseline

- React 19 stable line; pin an exact supported version in `package.json`.
- TypeScript strict mode with no unexplained `any`, unchecked casts, or ignored
  compiler errors.
- Vite for the client build.
- TanStack Router for routing, using its recommended file-based route
  generation.
- TanStack Query for API-backed server state and mutation invalidation.
- OpenAPI-generated request and response types.
- Vitest and Testing Library for unit and component tests.
- Playwright for browser workflows.
- Automated accessibility checks plus keyboard and screen-reader verification
  for critical clinical workflows.

Dependency versions are deliberately not recorded here because they change.
Before scaffolding or upgrading, verify the current stable versions and peer
requirements in official documentation.

## Dependency Gates

- Do not introduce React Router alongside TanStack Router.
- Do not introduce Next.js or TanStack Start; VisionOpus has a separate Go
  backend and currently requires a client-rendered application.
- Do not add Redux, Zustand, or another global store until a documented state
  ownership problem cannot be handled by the router, TanStack Query, form
  state, or component state.
- Do not fetch API data in `useEffect`; coordinate route-critical queries in
  route loaders and consume cached data through TanStack Query.
- Do not add a component framework, form library, schema validator, grid, or
  styling system solely because an agent commonly uses it. Evaluate and record
  the dependency first.
- Do not enable experimental or canary React features in production code.
- Do not rewrite EyeDraw while building the initial wrapper integration. First
  preserve and verify the legacy serialization and interaction contract.

## Required Verification

Every frontend change must report:

- Skills applied.
- Type checking, lint, unit/component test, and production build results.
- Browser routes and workflows exercised.
- Console errors and failed network requests observed.
- Accessibility checks performed.
- Screenshots for material visual changes.
- Remaining clinical, browser, performance, or dependency risks.
