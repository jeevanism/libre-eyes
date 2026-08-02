# TanStack Router Patterns

Last verified against official TanStack Router documentation: 2026-08-02.

Official references:

- <https://tanstack.com/router/latest/docs/quick-start>
- <https://tanstack.com/router/latest/docs/guide/type-safety>
- <https://tanstack.com/router/latest/docs/framework/react/guide/authenticated-routes>
- <https://tanstack.com/router/latest/docs/framework/react/guide/external-data-loading>

## Route Structure

- Use the recommended file-based routing mode and Vite router plugin.
- Treat the generated route tree as generated code; never edit it manually.
- Use pathless layout routes for authenticated and patient-context shells.
- Keep route files thin. Put substantial UI and domain behaviour in the owning
  feature directory.
- Code-split feature routes and provide deliberate pending, error, and not-found
  components.

## Type Safety

- Use route-specific `Route.useParams()` and `Route.useSearch()` APIs.
- Provide `from` or `to` when using shared navigation APIs so TypeScript narrows
  to the relevant route branch.
- Avoid `strict: false` except in truly route-agnostic shared infrastructure.
- Validate all search parameters at runtime. URL input is untrusted even when
  TypeScript types compile.
- Use search params for shareable filters, sorting, pagination, tabs, and other
  navigation state. Use replace navigation for high-frequency transient edits
  where browser-history entries would be harmful.

## Query Integration

- Provide the TanStack Query client through router context.
- Use route loaders to call `queryClient.ensureQueryData` for critical render
  data, preventing request waterfalls and loading flashes.
- Let TanStack Query own caching and freshness. Do not duplicate fetched data
  in router context or component/global state.
- Prefetch only data justified by the navigation path and confidentiality
  boundary.
- Ensure loader and query errors reach route error boundaries without exposing
  patient information in logs or messages.

## Authentication And Context

- Use parent `beforeLoad` checks to redirect unauthenticated users and establish
  typed route context for descendants.
- Client route guards improve navigation only. Every Go endpoint must enforce
  authentication, authorization, institution, site, firm, and patient context.
- Preserve the intended destination safely through login, but validate redirect
  targets to prevent open redirects.
- Clear or invalidate patient-scoped queries when identity or clinical context
  changes.
- Never preload protected patient data before the required context and access
  checks have succeeded.

## Boundaries

- Use TanStack Router, not React Router, for the current frontend.
- Use TanStack Router alone, not TanStack Start. The Go service remains the only
  application backend.
- A future Vue port will replace the router implementation but retain the same
  URL design and Go/OpenAPI contracts where practical.
