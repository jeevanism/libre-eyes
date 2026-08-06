# VisionOpus

VisionOpus is the Go, React, and PostgreSQL replacement for the legacy OpenEyes
ophthalmology EMR. The repository directory may retain the temporary
`go-openeyes` name until a coordinated repository rename is approved.

## Current Status

The repository and governance foundation is established. The authentication and
user-context specification is complete, independently reviewed, and
human-approved. The first bounded Go, PostgreSQL, and React authentication
walking skeleton is implemented, independently reviewed, and fully verified.
The patient identity schema, exact search repository, and audited patient-search
REST boundary are also implemented with synthetic PostgreSQL integration tests.
The authenticated React patient-search and exact-only duplicate-check workflow
now runs end to end against that boundary on desktop and mobile, with semantic
light, dark, and system themes.

See the [current project status](doc/project-status.md) for completed work,
validation results, current gates, and next steps.

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

- [Engineering constitution](doc/constitution.md)
- [Current project status](doc/project-status.md)
- [Slice risk and approval policy](doc/slice-risk-policy.md)
- [Modernization strategy](doc/modernization-strategy.md)
- [Documentation index](doc/README.md)
- [React and TanStack Router decision](doc/decisions/0001-react-tanstack-router.md)
- [Authentication foundation decision](doc/decisions/0003-authentication-foundation-policy.md)
- [Authentication security and migration approval](doc/approvals/0001-authentication-security-migration.md)
- [Authentication extraction workspace](doc/slices/01-auth-user-context/README.md)

Validate slice structure and readiness gates with:

```bash
./scripts/validate-slices
```

## Local Development

Prerequisites are Go 1.26.5, Node.js 24, npm, Docker with Compose, and a
Chromium-compatible browser for Playwright. The module retains Go 1.25 language
compatibility while CI and development use Go 1.26.5.

For normal development, one command prepares PostgreSQL, applies migrations,
prompts for a synthetic clinician password, seeds the development identity, and
starts the Go API and React application:

```bash
make dev
```

Open `http://localhost:5173/login` and sign in as `clinician` with the password
entered at startup. Press `Ctrl+C` to stop the API and frontend. PostgreSQL is
left running for the next session; stop it with `make db-down`.

If port `5432` belongs to another Docker container, the script names the
container and asks you to stop it rather than stopping unrelated work
automatically.

The equivalent manual workflow remains available. Create and load development
configuration, then start PostgreSQL and apply the versioned schema:

```bash
cp .env.example .env
set -a
source .env
set +a
make db-up
make migrate-up
```

Create a synthetic local identity by supplying a password at execution time;
the password is never stored in the repository:

```bash
VISIONOPUS_DEV_PASSWORD='choose-a-local-development-password' make dev-seed
```

Run the API and web application in separate terminals:

```bash
make api
make web-dev
```

The application is then available at `http://localhost:5173`. Run the bounded
verification suite with `make check`; PostgreSQL integration and live browser
tests are also enforced by Application CI.

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
