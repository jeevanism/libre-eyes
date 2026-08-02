# ADR-0001: React And TanStack Router Frontend

## Status

Accepted

## Date

2026-08-02

## Context

VisionOpus needs a long-lived clinical frontend for patient search, worklists,
dynamic examination forms, medication and consent workflows, theatre
scheduling, and EyeDraw. The application has a separate Go REST API and does
not currently need a JavaScript server runtime, public-page SEO, or server-side
rendering.

The workspace already contains React-oriented modernization specifications and
a React, TypeScript, and Vite EyeDraw integration reference. Vue can implement
the same workflows, but using two frontend frameworks now would duplicate
implementation and verification effort.

VisionOpus also needs typed routing for patient and event identifiers, validated
search parameters for dense worklists, nested authenticated layouts, route data
loading, and explicit route error boundaries.

## Decision

Build the current VisionOpus frontend with:

- The stable React 19 line.
- Strict TypeScript.
- Vite as the client build tool.
- TanStack Router using its recommended generated file-based routing.
- TanStack Query for remote server state and mutation invalidation.
- A generated TypeScript client from the Go OpenAPI contract.

TanStack Router owns URL and navigation state. TanStack Query owns API-backed
server state. Forms and transient UI state remain locally owned. The Go backend
remains authoritative for security, clinical validation, calculations, audit,
and persistence.

Use TanStack Router alone, not TanStack Start. Do not add React Router as a
second routing system.

## Alternatives Considered

### Vue 3

Vue is technically capable and offers a cohesive official ecosystem. It is
deferred because the current React EyeDraw reference and React-oriented design
work reduce near-term integration risk. A Vue port may be reconsidered only
after the Go, React, and PostgreSQL application is complete, stable, and fully
verified.

### React Router

React Router is mature, but TanStack Router was selected for end-to-end route
type inference, runtime-search validation, typed navigation, and loader/query
coordination suited to VisionOpus's context-heavy clinical routes.

### Next.js Or TanStack Start

Rejected for the current architecture. Their JavaScript server and full-stack
features would overlap with the Go backend and introduce deployment and
security boundaries VisionOpus does not need.

### Parallel React And Vue Frontends

Rejected. It would double UI implementation, accessibility review, browser
testing, dependency maintenance, and clinical verification before delivering a
complete replacement for OpenEyes.

## Consequences

- All frontend agents must follow `FRONTEND-SKILLS.md` and the project-local
  `visionopus-react` skill.
- Routes use TanStack Router conventions from the beginning; later migration
  from another router is avoided.
- The frontend remains replaceable because it consumes versioned Go/OpenAPI
  contracts and contains no authoritative clinical rules.
- A future Vue implementation would rewrite components, routing, and framework
  bindings while retaining the Go backend, PostgreSQL schema, OpenAPI contract,
  and verified workflows.
- React and TanStack dependency changes require official-documentation review,
  automated regression tests, and browser verification.
