# Go OpenEyes Rebuild

This repository is the working home for a Go, React, and PostgreSQL rebuild of
the legacy OpenEyes ophthalmology EMR.

The permanent product name has not been selected. `go-openeyes` is a temporary
working name.

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
- React and TypeScript for the web application.
- PostgreSQL for persistence.
- REST JSON with an OpenAPI contract.
- A separate FastAPI service may be considered later for AI-specific
  capabilities; it is not part of the current milestone.

## Data Safety

Do not use live patient data in development or provide it to external AI tools.
Use synthetic or properly de-identified fixtures approved for development.
