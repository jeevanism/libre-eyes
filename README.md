# LibreEyes

LibreEyes is an experimental, open-source proof of concept exploring a modern technology stack for ophthalmology electronic patient records (EPR). Inspired by the proven clinical workflows of the NHS classic [OpenEyes](https://github.com/appertafoundation/openeyes), it decouples the legacy monolith into a high-performance Go backend, an extensible React frontend (featuring a native wrapper for the original EyeDraw library), and a PostgreSQL data store.

* **Live Demonstration**: [https://libre-eyes.onrender.com](https://libre-eyes.onrender.com/) *(Demo login: `clinician` / `123456`)*
* **GitHub Repository**: [https://github.com/jeevanism/libre-eyes](https://github.com/jeevanism/libre-eyes)
* **Clinical Workflow Guide**: [Workflow Demonstration & Screenshots](WORKFLOW_DEMO.md)

---

## Background and Motivation

[OpenEyes](https://github.com/appertafoundation/openeyes) is a widely regarded open-source clinical application with extensive deployments across the UK National Health Service (NHS). While it is an elegant and clinically mature system, it is built on a traditional monolithic PHP framework, alongside the operational and maintenance challenges that legacy architectures bring over time.

LibreEyes represents a developer's trial to modernise this clinical platform. Rather than attempting a feature-complete clone, LibreEyes re-engineers the core architecture from the ground up using **Go**, **React 19**, and **PostgreSQL** to address the pain points of legacy monoliths—delivering instant cold-start deployments, minimal memory footprints, clean API boundaries, and simplified long-term maintenance, all while preserving essential clinical tools like EyeDraw.

None of the legacy OpenEyes backend code has been reused; however, the overall clinical workflows are heavily inspired by it. The upstream EyeDraw drawing engine has been preserved and wrapped into a dedicated React domain component inside LibreEyes, keeping all original licences intact and giving full credit to the original authors.

### Core Architectural Goals

LibreEyes was architected and designed from scratch around five key principles:

* **High-performance backend core**: Minimal CPU and memory overhead with predictable Go concurrency and direct SQL persistence.
* **Decoupled frontend and backend**: Clean REST/OpenAPI contracts empowering independent, modern React/TypeScript interface development.
* **Streamlined deployment**: Near-zero external runtime dependencies, enabling instant containerised startups via Docker, Podman, or cloud hosts like Render.
* **Configuration over customisation**: Multi-tenant institutional controls, roles, and catalogues managed via structured configuration rather than bespoke code branching.
* **Extensible clinical tooling**: Modular examination registries making it straightforward to introduce new clinical tools while keeping the transactional persistence core rock-solid.

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

## Licence and Attribution

LibreEyes is distributed under the **GNU Affero General Public License version 3 only (AGPLv3-only)**; see [LICENSE](LICENSE).

### EyeDraw Upstream Attribution and Intellectual Property

* **Full Credit to Original Authors**: EyeDraw was originally conceived, designed, and developed for the [OpenEyes project](https://github.com/appertafoundation/openeyes) under the stewardship of the **Apperta Foundation**. Full credit for the clinical drawing concepts, symbols, and graphics engine belongs entirely to the original OpenEyes and EyeDraw contributors.
* **Unmodified Licences & Integrity**: The upstream EyeDraw library and runtime files located under [`web/public/vendor/eyedraw/`](web/public/vendor/eyedraw/) retain their original copyright notices, upstream authorship, and licence files in full. None of the original copyright headers or licence texts have been altered, removed, or tampered with.
* **Preservation on Redistribution**: Anyone redistributing or modifying LibreEyes must preserve all copyright headers, attribution files, and licensing notices intact; see [`NOTICE`](NOTICE) and [`web/public/vendor/eyedraw/ATTRIBUTION.md`](web/public/vendor/eyedraw/ATTRIBUTION.md).
