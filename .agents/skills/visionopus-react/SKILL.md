---
name: visionopus-react
description: Apply VisionOpus frontend architecture and modern React 19, strict TypeScript, Vite, TanStack Router, and TanStack Query patterns. Use for any React component, route, loader, search parameter, query, mutation, form, EyeDraw adapter, frontend test, accessibility review, or browser-facing implementation in this repository.
---

# VisionOpus React

Build the current React frontend as a typed client of the Go/OpenAPI API. Keep
route, server, form, and local UI state separate, and verify clinical workflows
in a real browser.

## Start Every Task

1. Read `AGENTS.md` and `FRONTEND-SKILLS.md`.
2. Identify the owning vertical slice and its accepted evidence.
3. Recheck version-sensitive APIs in official React and TanStack documentation.
4. Read only the relevant reference:
   - React components, hooks, forms, or state: `references/react-patterns.md`.
   - Routes, navigation, loaders, search params, or auth guards:
     `references/tanstack-router.md`.
5. Define behavioural and accessibility acceptance tests before implementation.

## Architecture Rules

- Use feature slices. Keep route files thin and domain UI under its feature.
- Consume only the generated OpenAPI client at the network boundary.
- Treat backend authorization and validation as authoritative. Route guards are
  user experience controls, not security controls.
- Put URL-reproducible filters, pagination, sorting, tabs, and selected context
  in typed router search parameters.
- Put remote data in TanStack Query, form drafts in the form owner, and
  transient interaction state in the closest component.
- Keep clinical calculations in tested domain modules or the Go backend, not in
  JSX or effects.
- Isolate imperative systems such as EyeDraw in adapters with mount, update,
  serialization, error, and unmount tests.

## Implementation Loop

1. Add or update a failing behavioural test.
2. Implement the smallest complete vertical behaviour.
3. Run strict type checking, lint, focused tests, and the production build.
4. Exercise the affected route in a real browser at desktop and supported
   compact viewport sizes.
5. Inspect console messages and network failures.
6. Verify keyboard navigation, focus, accessible names, error announcements,
   loading, empty, permission-denied, and failure states.
7. Review for unnecessary effects, duplicated state, unsafe assertions,
   request waterfalls, and accidental clinical logic in components.

## Completion Gate

Do not call frontend work complete until the build and relevant tests pass and
the rendered workflow has been inspected. Report the skills used, exact
commands, browser coverage, accessibility checks, screenshots, assumptions,
and remaining risks.
