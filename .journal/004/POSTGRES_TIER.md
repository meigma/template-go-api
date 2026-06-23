---
title: PostgreSQL persistence tier — implementation design (source of truth)
session: 004
date: 2026-06-22
status: approved (LGTM) — temporary design doc, journal-only, mirrors TARGET_SHAPE.md's role
audience: workflow implementation agents (this file is authoritative for implementation)
---

# PostgreSQL persistence tier for template-go-api

This document is the **single source of truth** for implementing the PostgreSQL
persistence tier. Workflow agents MUST read it in full and implement exactly the
phase they are assigned — no earlier or later phase scope. Derived from session
004's research (`.journal/004/RESEARCH-go-postgres-data-access.md`) and the
collaboratively-approved proposal.

## Goal & non-goals

**Goal.** Add a production-grade PostgreSQL adapter implementing the existing
`todo.Repository` port, selectable at runtime alongside the in-memory adapter,
with type-safe queries (sqlc + pgx/v5), real migrations (goose), and
container-backed integration tests (testcontainers). The domain and transport
layers do not change. The in-memory adapter remains the zero-infra default so
the template still runs with no Docker/DB.

**Non-goals (leave untouched).** Auth, OTel tracing, rate limiting, pagination,
API versioning, mockery. No new resource — `todo` stays the only example. No
builder dependency shipped (see Dynamic Queries). No auto-migrate on serve.

## Locked decisions

- Driver: **jackc/pgx v5** (`pgxpool` for the pool).
- Static typed queries: **sqlc** (`sql_package: pgx/v5`), generated code committed.
- Migrations: **goose**, embedded, run via an explicit `migrate` subcommand
  (NOT auto-run on serve).
- Dynamic queries: **sqlc-only default** — no Squirrel/Bob dependency shipped.
  Demonstrate optional filtering via `sqlc.narg()`/array idioms in a commented
  example; document Squirrel & Bob as escape hatches that live *inside the
  adapter*, behind the port.
- Tests: **testcontainers-go** Postgres module with snapshot/restore; gated by a
  `//go:build integration` tag and a dedicated moon task.
- Tooling: pin `sqlc` and `goose` via **`go tool` directives** (Go 1.26 native).
- Generated code is **committed + drift-guarded**, mirroring the existing
  `openapi` / `openapi-check` pattern.
- Store selection: explicit **`--store=memory|postgres`** flag (default `memory`).
- ID column: **`uuid`** with a sqlc type-override to `github.com/google/uuid.UUID`,
  converted to/from the domain's `string` at the mapping boundary.
- Status column: **`text` + CHECK constraint** (not a native PG enum).

## Existing architecture this must fit (ground truth — verify in code)

- Domain `internal/todo`: entity `Todo{ID string, Title string, Status Status,
  CreatedAt time.Time, CompletedAt *time.Time}`; `Status` is `"open"|"completed"`;
  `ErrNotFound`, `ErrInvalidTitle`. Outbound port `todo.Repository` with exactly:
  `Save(ctx, Todo) error` (insert-or-replace / upsert semantics),
  `FindByID(ctx, id) (Todo, error)` (absent → `todo.ErrNotFound`),
  `List(ctx) ([]Todo, error)`.
- Reference adapter `internal/adapter/memory.TodoRepository` implements the port
  over a mutex-guarded map. The new adapter is its peer.
- Composition root `internal/app/app.go`: `New(cfg config.Config, logger
  *slog.Logger, version string) *App` builds `todo.NewService(memory.NewTodoRepository(),
  logger)`, wires the router via `adapterhttp.RouterDeps`, and currently passes
  `Readiness: nil` with a comment showing the intended
  `[]adapterhttp.ReadinessCheck{{Name, Check}}` shape. `OpenAPIYAML(version)`
  builds the spec server-lessly using a memory repo. `App.Run`/`shutdown`
  (`internal/app/serve.go`) manage the API + optional metrics listeners and
  graceful shutdown.
- Config `internal/config`: Viper-backed, env prefix `TEMPLATE_GO_API_*`, flags
  registered in `RegisterFlags`, loaded in `Load`, checked in `Validate`.
- CLI `internal/cli`: cobra `serve` (`runServe` calls `app.New(...).Run(ctx)`),
  `version`, `openapi`. `Options{Viper, Err, Build, ...}`.
- Build: **moon** (`moon.yml` at root) + **proto** for tool pinning. Tasks:
  `format`, `lint`, `build`, `test` (`go test ./...`), `openapi`,
  `openapi-check` (temp-file drift guard), `check` (deps: format, lint, build,
  test, openapi-check, docs:build; `runInCI: true`). Validate with
  `moon run root:check`.
- Go `1.26.4`. Module `github.com/meigma/template-go-api`.

## Lint / style constraints (enforced — do not fight them)

Enabled linters include: `sloglint`, `gochecknoglobals`, `funlen`, `cyclop`,
`godoclint`, `mnd`, `promlinter`. `exhaustruct` and `wrapcheck` are **disabled**.
Practical implications: godoc every exported symbol; no naked package-level
mutable globals (use constructors/locals); keep functions within length/complexity
limits (split helpers); name magic numbers as consts; wrap errors with `%w` and
context. Generated sqlc code carries `// Code generated … DO NOT EDIT.` and is
skipped by golangci-lint automatically.

## Target file layout

```
internal/adapter/postgres/
├── postgres.go          # Connect(ctx, cfg) (*pgxpool.Pool, error); pool tuning; Close
├── repository.go        # TodoRepository implements todo.Repository over sqlc.Queries; Ping(ctx)
├── mapping.go           # sqlc row structs <-> todo.Todo (uuid<->string, status<->Status, null times)
├── migrations.go        # //go:embed migrations/*.sql -> embed.FS (exported for goose + tests)
├── migrations/
│   └── 00001_create_todos.sql        # goose Up/Down — the single schema source sqlc reads
├── queries/
│   └── todos.sql                     # hand-written sqlc queries (named -- name: ... :one/:many/:exec)
├── sqlc/                             # GENERATED, COMMITTED (DO NOT EDIT): db.go, models.go, querier.go, todos.sql.go
└── repository_test.go   # //go:build integration — testcontainers Postgres + snapshot/restore
sqlc.yaml                            # repo root
```

`internal/cli/migrate.go` (+ `app`/wiring as needed) for the `migrate` subcommand.

## Schema & type mapping

`migrations/00001_create_todos.sql` (goose format; sqlc reads the Up section as
its schema — sqlc natively understands goose annotations, so there is no separate
`schema.sql` to drift):

```sql
-- +goose Up
CREATE TABLE todos (
    id           uuid        PRIMARY KEY,
    title        text        NOT NULL,
    status       text        NOT NULL DEFAULT 'open'
                             CHECK (status IN ('open', 'completed')),
    created_at   timestamptz NOT NULL,
    completed_at timestamptz
);

-- +goose Down
DROP TABLE todos;
```

| Domain field           | Column                    | Mapping |
|------------------------|---------------------------|---------|
| `ID string`            | `id uuid`                 | sqlc override → `uuid.UUID`; `.String()` ↔ `uuid.Parse` at boundary |
| `Title string`         | `title text NOT NULL`     | direct |
| `Status Status`        | `status text` + CHECK     | `string(status)` ↔ `todo.Status`; validate on read |
| `CreatedAt time.Time`  | `created_at timestamptz`  | direct |
| `CompletedAt *time.Time` | `completed_at timestamptz NULL` | sqlc `emit_pointers_for_null_types: true` → `*time.Time` |

## sqlc configuration (`sqlc.yaml`, sketch — implementer finalizes)

```yaml
version: "2"
sql:
  - engine: postgresql
    schema: internal/adapter/postgres/migrations
    queries: internal/adapter/postgres/queries
    gen:
      go:
        package: sqlc
        out: internal/adapter/postgres/sqlc
        sql_package: pgx/v5
        emit_pointers_for_null_types: true
        emit_interface: true        # Querier interface aids testability
        overrides:
          - db_type: uuid
            go_type: github.com/google/uuid.UUID
```

Queries (`queries/todos.sql`) cover the three port methods; `Save` is an
**upsert** to honor insert-or-replace:

```sql
-- name: UpsertTodo :exec
INSERT INTO todos (id, title, status, created_at, completed_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE
  SET title = EXCLUDED.title,
      status = EXCLUDED.status,
      completed_at = EXCLUDED.completed_at;

-- name: GetTodo :one
SELECT id, title, status, created_at, completed_at FROM todos WHERE id = $1;

-- name: ListTodos :many
SELECT id, title, status, created_at, completed_at FROM todos ORDER BY created_at;
```

Also include, **commented out** (illustrates the dynamic-query pattern without
shipping a builder), an optional-filter example using `sqlc.narg`:

```sql
-- Example (commented): optional status filter via narg.
-- WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
```

## Adapter behavior

`postgres.TodoRepository` holds a `*pgxpool.Pool` and a `*sqlc.Queries`:

- `NewTodoRepository(pool *pgxpool.Pool) *TodoRepository`.
- `Save` → `UpsertTodo` (maps domain → params).
- `FindByID` → `GetTodo`; translate `errors.Is(err, pgx.ErrNoRows)` →
  `todo.ErrNotFound` so transport's 404 mapping keeps working.
- `List` → `ListTodos`.
- `Ping(ctx) error` → `pool.Ping(ctx)`, for readiness.
- `Connect(ctx, cfg) (*pgxpool.Pool, error)` in `postgres.go` parses the URL,
  applies pool tuning (max conns), and verifies connectivity with a ping.

## Migrations

- `//go:embed migrations/*.sql` exposes an `embed.FS` consumed by goose
  (as a library) and by the integration tests.
- New `migrate` subcommand: `migrate up | down | status`, using goose against the
  embedded FS and the configured `--database-url`. NOT run automatically by
  `serve` (avoids multi-replica races; migrations are explicit).
- moon task `migrate` wraps the subcommand for local dev.

## Config & composition-root changes (the intended ripple)

- New config fields/flags: `--store` (`memory`|`postgres`, default `memory`),
  `--database-url` (env `TEMPLATE_GO_API_DATABASE_URL`), optional
  `--db-max-conns`. `Validate()` requires `database-url` when `store=postgres`
  and rejects unknown `store` values.
- `app.New` signature changes to **`New(ctx context.Context, cfg config.Config,
  logger *slog.Logger, version string) (*App, error)`** — connecting a pool needs
  a context and can fail. Update `runServe` (cli/serve.go) and `app_test.go`
  accordingly.
- `App` gains the pool (when postgres) and closes it during graceful shutdown
  (extend `shutdown` in serve.go).
- When `store=postgres`, wire `Readiness:
  []adapterhttp.ReadinessCheck{{Name: "postgres", Check: repo.Ping}}` so `/readyz`
  becomes a real check (this is exactly the hook the current app.go comment
  anticipates).
- `OpenAPIYAML` stays memory-only — spec generation needs no DB, so the `openapi`
  task and docs pipeline are unaffected.

## Dynamic queries — what ships vs. documented

Ship **no** builder dependency. `List` stays parameter-free for now. Add:
- the commented `sqlc.narg` example in `queries/todos.sql`;
- a README/docs section: "Need genuinely dynamic queries? Define a domain
  `Filter`/criteria struct on the port, translate it to SQL **inside the
  adapter**, and reach for Squirrel or Bob *there* — never above the adapter
  boundary," with a short `TodoFilter` struct snippet and the rule that builder
  types must never appear in a port signature.

## Testing (testcontainers)

- `repository_test.go` (`//go:build integration`): start the testcontainers
  Postgres module (`postgres.Run(ctx, image, opts...)`), apply embedded goose
  migrations, `Snapshot()` once, then `Restore()` between tests for fast
  isolation. Do not name the database "postgres"; mind non-default usernames.
- Verify the adapter against the same behavioral contract the memory adapter
  satisfies (save/get/list/upsert-idempotency/not-found).
- A small shared helper sets up container + migrate + snapshot.
- Default `go test ./...` and `moon run root:check` stay hermetic (tag excludes
  the suite). New moon task **`test-integration`** runs the tagged suite (needs
  Docker). Wiring it into CI is a flagged follow-up (GitHub workflows are
  currently `.disabled`). If Docker is unavailable in the implementation
  environment, report that at the phase gate rather than faking a pass.

## Phase plan (each phase = one workflow run, then a human gate)

Work happens on branch `feat/postgres-tier` in its own worktree. Commit per phase
with Conventional Commits (messy intermediate commits are fine; the PR
squash-merges). After each phase: validate, then STOP for human review.

### Phase A — Tooling + schema + generated layer
Scope: `sqlc.yaml`; `migrations/00001_create_todos.sql`; `queries/todos.sql`
(incl. commented narg example); `go tool` directives for sqlc + goose; moon tasks
`sqlc` (generate) and `sqlc-check` (temp-dir regenerate + `git diff --exit-code`
drift guard) wired into `check` deps; run generation and **commit the generated
`internal/adapter/postgres/sqlc/` package**. No adapter wiring yet.
Acceptance: `moon run sqlc` is reproducible; `moon run root:check` passes
(including the new `sqlc-check`); generated code compiles.

### Phase B — Adapter + wiring + config + migrate subcommand
Scope: `postgres.go`, `repository.go`, `mapping.go`, `migrations.go`;
config `--store`/`--database-url`/`--db-max-conns` + `Validate`; `app.New` ripple
(ctx + error + pool close in shutdown) + store selection + postgres readiness;
`migrate` subcommand. Update `runServe`, `app_test.go`, and any callers.
Acceptance: builds; `serve --store=memory` unchanged; `--store=postgres` connects
and serves with a real `/readyz`; `migrate up/down/status` work; `moon run
root:check` passes.

### Phase C — Integration tests
Scope: `repository_test.go` (`integration` tag) + shared container helper;
`test-integration` moon task; CI-wiring note. Acceptance: `moon run
test-integration` passes locally (Docker present); default `check` stays
hermetic. If Docker absent, report at the gate.

### Phase D — Docs
Scope: README / DELETE_ME / `docs/` updates — running with Postgres, the
`migrate` subcommand, regenerating sqlc, running integration tests, and the
dynamic-query guidance. Acceptance: docs build; links/commands accurate; `moon
run root:check` passes.

## Definition of done (whole tier)
`moon run root:check` green; `moon run test-integration` green with Docker; the
template runs with `--store=memory` (zero infra) and `--store=postgres` (real DB
with migrations + readiness); generated code committed and drift-guarded; docs
updated; domain & transport unchanged. Integrate via a GitHub PR (squash merge).
