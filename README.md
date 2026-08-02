# VisionOpus

VisionOpus is the Go, React, and PostgreSQL replacement for the legacy OpenEyes
ophthalmology EMR. The repository directory may retain the temporary
`go-openeyes` name until a coordinated repository rename is approved.

## Current Status

The project is in repository preparation and targeted reverse-engineering.
Application implementation has not started.

The first planned walking skeleton is:

```text
local authentication
  -> user, institution, site, and firm context
  -> RBAC authorization
  -> audit event
  -> patient search
  -> patient summary header
```

## Documentation

- [Modernization strategy](doc/modernization-strategy.md)
- [Documentation index](doc/README.md)
- [React and TanStack Router decision](doc/decisions/0001-react-tanstack-router.md)
- [Authentication extraction workspace](doc/slices/01-auth-user-context/README.md)

## Source Authority

The legacy source is outside this repository at:

```text
../../OpenEyes/openeyes
```

Existing extraction documents are research leads. Legacy source files,
migrations, tests, fixtures, configuration, and observed runtime behaviour are
the authority.

## Technology Direction

- Go modular monolith for the primary backend.
- React 19, strict TypeScript, Vite, TanStack Router, and TanStack Query for the
  web application.
- PostgreSQL for persistence.
- REST JSON with an OpenAPI contract.
- A separate FastAPI service may be considered later for AI-specific
  capabilities; it is not part of the current milestone.

## Data Safety

Do not use live patient data in development or provide it to external AI tools.
Use synthetic or properly de-identified fixtures approved for development.
