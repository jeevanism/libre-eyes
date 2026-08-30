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

The public Render deployment configuration is in `Dockerfile` and
`render.yaml`. Never commit secrets; use `.env.example` only as a template.

## Upstream notices

The EyeDraw bundle under `web/public/vendor/eyedraw/` contains files derived
from the EyeDraw/OpenEyes projects. Preserve the included licence and
attribution notices when redistributing this repository.
