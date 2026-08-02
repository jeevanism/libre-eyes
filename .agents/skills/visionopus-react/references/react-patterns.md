# Modern React Patterns

Last verified against official React documentation: 2026-08-02.

## Component Rules

- Use function components and hooks. Class components are permitted only at a
  boundary that still requires them, such as an existing error boundary.
- Components and hooks must be pure and idempotent. Never mutate props, state,
  hook arguments, or values already passed to JSX.
- Keep side effects out of render. Handle user-caused work in event handlers.
- Use `useEffect` only to synchronize with an external system such as EyeDraw,
  browser APIs, subscriptions, or timers. Do not use effects to derive render
  data, mirror props into state, or orchestrate ordinary user events.
- Calculate derived values during render. Memoize only expensive, measured work
  or when stable identity is required by an external API.
- Follow the Rules of Hooks and enable React Strict Mode plus the official React
  hooks lint rules.

Official references:

- <https://react.dev/reference/rules>
- <https://react.dev/learn/you-might-not-need-an-effect>
- <https://react.dev/learn/choosing-the-state-structure>

## State Ownership

Classify each value before choosing a state mechanism:

| State | Owner |
| --- | --- |
| Path, selected patient context, filters, sort, pagination, active tab | TanStack Router path or validated search params |
| API data, cache freshness, mutations | TanStack Query |
| Field values, validation display, dirty state | Form owner |
| Open dialog, hover, expanded row, temporary selection | Closest component |
| Theme, locale, current authenticated identity | Narrow application context when justified |

Avoid contradictory, redundant, duplicated, and deeply nested state. Store
stable identifiers rather than duplicated domain objects when possible.

## TypeScript

- Enable all strict compiler checks.
- Model valid states with discriminated unions instead of multiple booleans.
- Parse untrusted data at system boundaries. TypeScript types do not validate
  runtime JSON.
- Do not use non-null assertions or casts to silence a design problem.
- Use generated OpenAPI types for transport data and explicit domain/view
  models when UI needs differ.

## Component Boundaries

- Keep route components focused on composition and route dependencies.
- Separate domain behaviour from reusable presentation primitives.
- Prefer composition over large configuration objects and boolean-prop APIs.
- Do not create a hook merely to hide a few lines. Hooks represent reusable
  stateful behaviour or a boundary to an external system.
- Keep accessible labels, errors, descriptions, and focus behaviour inside the
  component contract.

## Data And Mutations

- Never call `fetch` directly from presentation components.
- Use query-key factories owned by the feature.
- Make mutation success update or invalidate the precise affected queries.
- Render intentional pending, empty, stale, error, permission-denied, and
  conflict states.
- Do not use optimistic updates for clinically significant writes unless the
  workflow specifies rollback, reconciliation, and audit behaviour.

## EyeDraw

EyeDraw is an imperative external system, so an effect-based adapter is valid.
The adapter must:

- Own the DOM container and runtime instance.
- Initialize once per intended instance lifecycle.
- Normalize and validate serialized input.
- Expose typed change and error callbacks.
- Clean up events, globals, observers, and instances on unmount.
- Avoid allowing legacy jQuery code to control DOM outside the adapter.
- Prove save/load parity and remount behaviour with automated and browser tests.
