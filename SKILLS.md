# Go Skill Policy

This file defines how AI coding agents select and apply Go skills in this
repository. It is a routing policy, not a substitute for reading the relevant
skill files.

## Authority And Provenance

The reviewed skill bundle is:

- Project: `samber/cc-skills-golang`
- License: MIT
- Reviewed source: `/home/jeevanism/Documents/Projects/AI-Engineering/agent-skills/golang-skills/cc-skills-golang`
- Reviewed revision: `8b2d019212d6a5390d472a7660a8489109d7db49`
- Revision date: 2026-07-02

Canonical skill identifiers use
`samber/cc-skills-golang@<skill-name>`. Agent tools may expose equivalent names,
such as `cc-skills-golang:<skill-name>`; use the available equivalent.

Resolve skill instructions in this order:

1. `AGENTS.md` and approved architecture decisions.
2. This project routing policy.
3. A tool-native installation of the named skill.
4. The reviewed source bundle above when no tool-native copy is available.
5. Official Go and dependency documentation for version-sensitive facts.

Generic skill advice must not override project architecture, clinical safety,
security, licensing, or extraction gates.

## Mandatory Baseline

Load all of these for every Go-related task:

| Skill | Project reason |
| --- | --- |
| `golang-how-to` | Selects the complete task-specific skill set. |
| `golang-error-handling` | Preserves explicit failure semantics across service boundaries. |
| `golang-safety` | Reduces panics, corruption, aliasing, and lifecycle defects. |
| `golang-security` | Required for authentication, patient data, input, I/O, and secrets. |
| `golang-testing` | Makes executable evidence part of implementation. |
| `golang-context` | Required through request, database, and background I/O boundaries. |

Do not load all remaining skills indiscriminately. Select every skill relevant
to the current task using the routes below.

## Task Routing

| Task | Additional skills to load |
| --- | --- |
| Repository or module scaffold | `golang-project-layout`, `golang-design-patterns`, `golang-dependency-injection`, `golang-lint`, `golang-dependency-management` |
| Domain model or public API design | `golang-design-patterns`, `golang-structs-interfaces`, `golang-naming`, `golang-code-style`, `golang-documentation` |
| REST handler or middleware | `golang-design-patterns`, `golang-structs-interfaces`, `golang-code-style`, `golang-observability` |
| PostgreSQL, migrations, transactions, or `sqlc` | `golang-database`, `golang-structs-interfaces`, `golang-lint` |
| Authentication, authorization, sessions, PIN, or cryptography | `golang-database`, `golang-design-patterns`, `golang-structs-interfaces`, `golang-lint`, `golang-troubleshooting` when investigating legacy discrepancies |
| Goroutines, queues, locks, or shutdown | `golang-concurrency`, `golang-context`, `golang-safety`, `golang-testing` |
| Configuration or command-line program | `golang-cli`, `golang-project-layout`; add Cobra or Viper skills only after those dependencies are approved |
| Logging, metrics, tracing, or audit telemetry | `golang-observability`, `golang-error-handling`, `golang-security` |
| Performance investigation | `golang-benchmark`, `golang-troubleshooting`; load `golang-performance` only after measurement identifies a bottleneck |
| Bug, panic, race, deadlock, or flaky test | `golang-troubleshooting`, `golang-safety`; add `golang-concurrency` for race or deadlock work |
| Dependency selection or upgrade | `golang-popular-libraries`, `golang-pkg-go-dev`, `golang-dependency-management`, `golang-security` |
| CI and release checks | `golang-continuous-integration`, `golang-lint`, `golang-security`, `golang-dependency-management` |
| Go version upgrade or old idiom review | `golang-modernize`, `golang-code-style`, `golang-lint`, `golang-testing` |
| Documentation | `golang-documentation`, `golang-naming` |

When a task spans multiple rows, load the union of their skills. Read referenced
skill material only as needed; do not import unrelated framework guidance into
the decision.

## Project Constraints On Skill Advice

- The future root `go.mod` defines the target Go language version. Until it is
  approved and created, do not write production Go or assume that the installed
  toolchain version is the project target.
- The supplementary `golanfSKILL.md` in the source directory is advisory only.
  Its version-specific recommendations apply only when supported by `go.mod`
  and verified against official Go documentation.
- Use manual constructor injection. The Wire, Dig, Fx, and `samber/do` skills
  are inactive unless an architecture decision changes this rule.
- Build REST JSON contracts described by OpenAPI. GraphQL and gRPC skills are
  inactive unless an architecture decision changes the protocol.
- Prefer the standard library and already approved dependencies. Popularity or
  presence in this catalog is not sufficient justification for a dependency.
- Prefer standard `testing` first. Load the Testify skill only if Testify is
  approved and present in `go.mod`.
- Do not activate Cobra, Viper, `samber/*`, Swagger annotation, cache, or
  reactive-stream skills merely because they are available.
- Benchmark before optimizing. Never trade clinical correctness, auditability,
  or clarity for an unmeasured performance claim.
- Security guidance may intentionally depart from unsafe legacy parity. Record
  the departure in the relevant slice specification or architecture decision.

## Complete Catalog

The following 44 skills are available in the reviewed bundle.

### Core Quality

- `golang-code-style`
- `golang-documentation`
- `golang-error-handling`
- `golang-lint`
- `golang-modernize`
- `golang-naming`
- `golang-safety`
- `golang-security`
- `golang-structs-interfaces`

### Architecture And Runtime

- `golang-concurrency`
- `golang-context`
- `golang-data-structures`
- `golang-database`
- `golang-dependency-injection`
- `golang-design-patterns`
- `golang-observability`

### Testing, Performance, And Diagnostics

- `golang-benchmark`
- `golang-performance`
- `golang-testing`
- `golang-troubleshooting`

### Project And Dependencies

- `golang-cli`
- `golang-continuous-integration`
- `golang-dependency-management`
- `golang-how-to`
- `golang-pkg-go-dev`
- `golang-popular-libraries`
- `golang-project-layout`
- `golang-stay-updated`

### APIs And Tools

- `golang-google-wire`
- `golang-graphql`
- `golang-grpc`
- `golang-spf13-cobra`
- `golang-spf13-viper`
- `golang-swagger`
- `golang-stretchr-testify`
- `golang-uber-dig`
- `golang-uber-fx`

### Samber Libraries

- `golang-samber-do`
- `golang-samber-hot`
- `golang-samber-lo`
- `golang-samber-mo`
- `golang-samber-oops`
- `golang-samber-ro`
- `golang-samber-slog`

## Task Report

For each Go-related implementation or review, include a `Skills applied` line
in the required agent report. List the skill names actually loaded and note any
skill recommendation rejected because it conflicted with project rules.
