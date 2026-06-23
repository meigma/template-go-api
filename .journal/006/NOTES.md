---
id: 006
title: Docker Compose day-one stack (API + PostgreSQL)
started: 2026-06-23
---

## 2026-06-23 15:10 — Kickoff
Goal for the session: set up a Docker Compose file so the "day one" running of
the template exercises the **full** template API + PostgreSQL backend (not just
the zero-infra in-memory default). The compose stack should let a new consumer
bring up the API wired to Postgres with one command.

Current state of the world:
- The template is a finished hexagonal Go API server (chi v5 + Huma v2),
  `master` at `18b56e7`.
- Persistence tier is built (PR #6): `--store=memory|postgres` selects the
  adapter; memory is the zero-infra default. Postgres adapter uses jackc/pgx v5
  + sqlc (generated code committed) + goose migrations.
- Migrations are **explicit**: an embedded `migrate up|down|status` subcommand,
  NEVER auto-run on serve (avoids multi-replica races). So a compose stack must
  run migrations as a discrete step before/at API startup.
- Config is Viper-backed (`TEMPLATE_GO_API_*` env + flags): `--store`,
  `--database-url`, `--db-max-conns`, plus serve flags (addr, metrics-addr,
  CORS, trusted-proxy-header). Metrics on a dedicated listener (`:9090`).
- There is **no Dockerfile or docker-compose** yet — running against Postgres
  today means standing up Postgres yourself and exporting env/flags.
- Integration tests already spin up `postgres:17-alpine` via testcontainers, so
  a known-good Postgres image + schema bootstrap path exists to mirror.

Plan: awaiting further instructions from the user before any substantive work
(per the session-new request). Likely scope to discuss: a Dockerfile for the API
binary, a `docker-compose.yml` with a Postgres service + healthcheck + a
migration step (one-shot vs. init), env wiring (`--store=postgres`,
`--database-url`), and how this slots into moon tasks / README "day one" docs.

## 2026-06-23 15:21 — Design locked
User clarified the goal and answered the open forks. Recorded decisions:

- **End goal (NOT this session):** remove the in-memory tier and default the
  template to PostgreSQL-only (reduces boilerplate likely to be deleted by
  consumers). This session **ignores the memory removal** and only stands up the
  Docker Compose dev stack. Compose just sets `--store=postgres` explicitly.
- **Compose shape (a sequenced DAG, NOT Postgres init-dir):** the seeds depend on
  the `todos` table, which only exists after `migrate up` (explicit, binary-run,
  never on serve, never by Postgres). Mounting `hack/sql` into
  `/docker-entrypoint-initdb.d/` would run it at DB first-boot *before*
  migrations and fail. So:
  `postgres (healthcheck) → migrate (one-shot: binary `migrate up`) → seed
  (one-shot: psql applies hack/sql/*.sql) → api (serve, prebaked DATABASE_URL)`.
- **Seed mechanism:** a tiny `postgres:17-alpine` one-shot loops `psql -f` over
  every `/sql/*.sql` (`./hack/sql:/sql:ro`, sorted, `ON_ERROR_STOP=1`, no-ops on
  empty dir). Keeps seeding entirely OUT of the server binary and OUT of
  migrations — the boundary the user asked for. `hack/sql` is new.
- **DB lifecycle = ephemeral/reproducible** (user choice): no persisted volume;
  every `up` rebuilds a clean DB, migrates, applies all of hack/sql → identical
  state each time; seeds need NOT be idempotent; `down` wipes it.
- **Ship an example seed** (user choice): `hack/sql/0001_seed_todos.sql` inserts a
  couple of todos so `docker compose up` immediately shows real data via the API.
- **Image:** one multi-stage Dockerfile (CGO_ENABLED=0 static, distroless/static
  runtime) reused by both `migrate` and `api` via a compose build anchor; Go
  1.26.4; entrypoint `./cmd/template-go-api`. Postgres pinned to
  `postgres:17-alpine` (matches the integration suite). Prebaked conn string
  `postgres://app:app@postgres:5432/app?sslmode=disable`.
- **Filename:** modern `compose.yaml`. Doc the day-one flow in README; no moon
  task (long-running interactive process doesn't fit moon's cache model).

Next: build in an implementation worktree off `master`, functionally test
(`docker compose up` → curl the API → confirm seeded todos), then PR.
