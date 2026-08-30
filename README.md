# LibreEyes

LibreEyes is a Go, React, and PostgreSQL ophthalmology application inspired by
the clinical concepts and user experience of OpenEyes. This repository contains
the frozen synthetic demonstration release.

## Demonstration scope

The application demonstrates patient search and summaries, care episodes,
examination drafts, EyeDraw, visual acuity, IOP, diagnosis, prescribing,
consent, correspondence, referrals, clinic flow, theatre scheduling, and
tenant-scoped access controls.

It is synthetic demonstration software only. Do not use real patient data and
do not use this repository for clinical care or production NHS deployment.

## Technology

- Go modular monolith and REST/OpenAPI API
- React 19, TypeScript, Vite, TanStack Router, and TanStack Query
- PostgreSQL with versioned migrations
- Bun for the frontend build
- Docker and Render deployment configuration

## Backend persistence

The Go backend uses `pgx/v5` directly. Repositories execute explicit,
parameterised SQL and scan results into Go types; no ORM or query-generation
framework is used. This keeps transactions, tenant-isolation predicates,
audit-sensitive writes, and PostgreSQL behaviour visible in the application
code.

## Local verification

```bash
go test ./...
bun install --cwd web --frozen-lockfile
bun run --cwd web typecheck
bun run --cwd web test -- --run
bun run --cwd web build
```

Build the demonstration container with:

```bash
docker build --tag libre-eyes-demo .
```

## Quick start for demonstrators

Non-technical users only need Docker Desktop (which includes Docker Compose).
No Go, Bun, Node.js, or source checkout is required. Run this single command:

```bash
mkdir libreeyes-demo && cd libreeyes-demo && curl -fsSL https://raw.githubusercontent.com/jeevanism/libre-eyes/main/compose.yaml -o compose.yaml && docker compose up
```

Compose downloads the configuration and published LibreEyes image, starts
PostgreSQL, applies migrations, seeds the synthetic account, and launches the
application. Open `http://localhost:10000/login` and use `clinician` / `123456`.
This fixed password is for demonstration only. Press Ctrl-C to stop the
containers; remove them with `docker compose down`.

## Run from source

Developers who want to inspect or modify the source can clone the repository.
Install Docker, Go, and Bun, then run:

```bash
git clone https://github.com/jeevanism/libre-eyes.git
cd libre-eyes
./demo.sh
```

The source launcher starts PostgreSQL, applies migrations, seeds the synthetic
clinician, and runs the Go API and Bun-powered React development server. Open
`http://localhost:5173/login`. Set `LIBREEYES_DEV_PASSWORD` before running it
to choose a different local password.

The public Render deployment configuration is in `Dockerfile` and
`render.yaml`. Never commit secrets; use `.env.example` only as a template.

## Licence

LibreEyes is distributed under the GNU Affero General Public License version 3
only (AGPLv3-only). The EyeDraw-derived files under
`web/public/vendor/eyedraw/` retain their upstream attribution and licence
notices; see `NOTICE` and that directory's `ATTRIBUTION.md`.

## Upstream notices

The EyeDraw bundle under `web/public/vendor/eyedraw/` contains files derived
from the EyeDraw/OpenEyes projects. Preserve the included licence and
attribution notices when redistributing this repository.
