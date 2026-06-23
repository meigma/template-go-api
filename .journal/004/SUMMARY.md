---
id: 004
title: Research and build the PostgreSQL persistence tier
date: 2026-06-22
status: complete
repos_touched: [template-go-api]
related_sessions: ["002", "003"]
---

## Goal
Decide and build the persistence tier for the API template — the open
"Postgres adapter + testcontainers" seam. Start with deep research into modern
(2025–2026) Go+PostgreSQL data-access approaches, settle the design
collaboratively, then implement it behind the existing `todo.Repository` port
without touching the domain or transport.

## Outcome
Met. Research → design (`.journal/004/POSTGRES_TIER.md`) → four gated
implementation phases → shipped as **PR #6, squash-merged to `master` `18b56e7`**.
The template now selects `--store=memory|postgres` at runtime; in-memory stays the
zero-infra default. `moon run root:check` green; the testcontainers integration
suite passes against a real `postgres:17-alpine` (verified uncached, ~4s); domain
and transport unchanged. Built via a multi-agent workflow with a human review gate
after every phase.

## Key Decisions
- **sqlc + pgx/v5 over an ORM** — the deep-research report found the community has
  shifted from full-magic ORMs (GORM) toward sqlc (type-safe codegen, compile-time
  guarantees, raw-SQL perf) as the production default; ORMs reserved for
  CRUD/association-heavy apps. Cleanest fit for the repository port.
- **goose for migrations, not Atlas** — Atlas is open-core and moved
  `migrate lint` behind a paid tier (v0.38, Oct 2025); goose is fully OSS,
  single-tier — the right default for a copyable template.
- **Proto-managed sqlc/goose CLIs, not `go tool`** — matches the repo's
  golangci-lint convention and keeps `go.mod`/`go.sum` free of the tools'
  transitive deps (dropped goose's bundled dialect drivers entirely).
- **sqlc-only dynamic-query default** — no query-builder dependency shipped; a
  commented `sqlc.narg` example + README guidance show the pattern. Squirrel and
  Bob documented as escape hatches that live *inside the adapter* behind the port.
  Bob (v0.47, pre-1.0) evaluated but not adopted as default (adoption risk).
- **`Save` = full insert-or-replace (created_at replaced)** so the postgres and
  in-memory adapters satisfy an identical contract; the shared test can assert
  identical behavior.
- **Integration tests in a dedicated `internal/integration` package** (package
  `integration`, `//go:build integration`, a `doc.go` anchor) — discoverable in
  one place and forced through public APIs rather than sitting next to the code.
- **Migrations explicit** — embedded `migrate up|down|status` subcommand
  (goose-as-library); never auto-run on serve (avoids multi-replica races).
- **`app.New` → `New(ctx) (*App, error)`** — connecting a pool needs ctx and can
  fail; the pool is closed on every `Run` exit path (deferred); postgres
  contributes a real `/readyz` check.
- **Process: gated multi-agent workflow** — each phase ran implement → 3-lens
  adversarial review (correctness · doc-adherence · conventions) → fix → human
  gate. The design doc was the agents' single source of truth.

## Changes
All in PR #6 (`18b56e7`). Domain (`internal/todo`) and HTTP transport unchanged.
- `internal/adapter/postgres/` — adapter (`postgres.go` pool/Connect,
  `repository.go` Save/FindByID/List/Ping over sqlc, `mapping.go`, `migrate.go`
  goose-as-library, `migrations.go` embed), goose migration, hand-written
  `queries/todos.sql`, committed generated `sqlc/`.
- `sqlc.yaml`; `.moon/proto/{sqlc,goose}.toml` + `.prototools` pins.
- `internal/config` — `--store`/`--database-url`/`--db-max-conns` + validation.
- `internal/app` — store selection, pool lifecycle, postgres readiness, `New`
  signature change; `internal/cli/migrate.go` — `migrate` subcommand.
- `internal/integration/` — testcontainers suite (`doc.go`,
  `postgres_fixture_test.go`, `postgres_test.go`) with snapshot/restore.
- `moon.yml` — `sqlc`/`sqlc-check` (drift guard), `migrate`, `test-integration`
  tasks; `.golangci.yml` — `run.build-tags: [integration]`.
- `go.mod`/`go.sum` — pgx/v5, goose (library), testcontainers; **`golang.org/x/crypto`
  pinned to v0.53.0** to clear a CRITICAL CVE finding the new deps pulled in.
- README / DELETE_ME / `docs/docs/index.md` — persistence docs, dynamic-query
  guidance, integration-test convention.

## Open Threads
- **Wire `test-integration` into CI** — the GitHub workflows are currently
  `.disabled` and need a Docker-capable runner. No date; future slice.
- **`opencontainers/go-digest` CC-BY-SA-4.0 license note** (via goose) — advisory,
  non-blocking (Kusari passed); a future org-policy pass could allowlist/replace it.
- Still-open future-slice seams: authn/authz; OTel tracing; rate limiting;
  pagination; API versioning; mockery.

## References
- PR #6: https://github.com/meigma/template-go-api/pull/6 (merged, `18b56e7`)
- Design doc (source of truth): `.journal/004/POSTGRES_TIER.md`
- Research report: `.journal/004/RESEARCH-go-postgres-data-access.md`
- Prior slices: `.journal/002/SUMMARY.md`, `.journal/003/SUMMARY.md`
- Memory: `separate-mechanical-from-design-work`, `subagents-may-read-divergent-worktree`

## Lessons
- **Deep-research synthesis is brittle at scale.** The workflow's final
  schema'd synthesis agent bled tool-call parameter markup into its JSON and blew
  the StructuredOutput retry cap. Fix: drop the schema (free-text markdown out) and
  resume with `resumeFromRunId` — the cached search/fetch/verify prefix replayed
  instantly; only synthesis re-ran. Pass the same `args` on resume or the script
  short-circuits.
- **gopls diagnostics lie in worktree flows.** "BrokenImport / undefined"
  diagnostics fired throughout because the `.wt/` implementation worktree wasn't in
  the editor's `go.work`. They were all false; `go build`/`go vet`/`moon run check`
  were the truth. Don't react to IDE diagnostics for files in a separate worktree.
- **Check CI even when local checks are green.** Kusari caught a CRITICAL the
  local `root:check` could not: new deps dragged `x/crypto v0.51.0` (13 CVEs) into
  the module graph though `go mod why` showed our build never imports it. An
  indirect-version pin cleared it.
- **A tag-only test directory breaks `go test ./...`** ("build constraints
  exclude all Go files"); a non-tagged `doc.go` anchors the package.
- **moon v2's arg tokenizer splits on `=`** — use `-tags integration` (space) in
  tasks, never `-tags=integration`, or the flag silently becomes a bogus package
  arg and the suite is skipped with a false "[no test files]" pass.
