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
