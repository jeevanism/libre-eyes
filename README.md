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

## Workflow demo screenshots

For detailed clinical workflows, default login credentials (`clinician` / `123456`), custom user provisioning instructions, and step-by-step visual walk-throughs, please see the [Workflow Demonstration Guide](WORKFLOW_DEMO.md).

## Technology

- Go modular monolith and REST/OpenAPI API
- React 19, TypeScript, Vite, TanStack Router, and TanStack Query
- PostgreSQL with versioned migrations
- Bun for the frontend build
- Docker / Podman and Render deployment configuration

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

Build the demonstration container with Docker or Podman:

```bash
docker build --tag libre-eyes-demo .
# or
podman build --tag libre-eyes-demo .
```

## Quick start for demonstrators

Non-technical users only need Docker Desktop or Podman (with Compose support).
No Go, Bun, Node.js, or source checkout is required. Run this single command:

```bash
mkdir libreeyes-demo && cd libreeyes-demo && curl -fsSL https://raw.githubusercontent.com/jeevanism/libre-eyes/main/compose.yaml -o compose.yaml && docker compose up
```

*(When using Podman, replace `docker compose` with `podman compose` or `podman-compose`)*

Compose downloads the configuration and published LibreEyes image, starts
PostgreSQL, applies migrations, seeds the synthetic account, and launches the
application. Open `http://localhost:10000/login` and use `clinician` / `123456`.
This fixed password is for demonstration only. Press Ctrl-C to stop the
containers; remove them with `docker compose down` (or `podman compose down`).

## Run from source

Developers who want to inspect or modify the source can clone the repository.
Install Docker or Podman, Go, and Bun, then run:

```bash
git clone https://github.com/jeevanism/libre-eyes.git
cd libre-eyes
./demo.sh
```

`demo.sh` automatically detects either `docker` or `podman` on your PATH (you can also explicitly specify your runtime with `CONTAINER_CLI=podman ./demo.sh` or `CONTAINER_CLI=docker ./demo.sh`).

The source launcher starts PostgreSQL, applies migrations, seeds the synthetic
clinician, and runs the Go API and Bun-powered React development server. Open
`http://localhost:5173/login`. You can set `LIBREEYES_DEV_USERNAME` and
`LIBREEYES_DEV_PASSWORD` before running it to choose your own local credentials,
or provision custom accounts directly via the in-app Admin console.

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
