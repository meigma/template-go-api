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

## 2026-06-23 15:26 — Built and functionally verified
Worktree `feat/compose-dev-stack` (`.wt/feat-compose-dev-stack`) off `master`.

Discovery: a **Dockerfile, `.dockerignore`, and `.go-version` already existed**
(README "Container Image" section). My initial `ls Dockerfile* docker-compose*`
returned nothing only because **zsh's `nomatch` aborts the whole compound
command when any glob is empty** — the Dockerfile was there all along. Reused it
as-is: multi-stage, `CGO_ENABLED=0` static build, distroless/static:nonroot,
ENTRYPOINT `/usr/local/bin/template-go-api`, so `migrate up` and `serve` are just
args. No Dockerfile change needed.

Files added/changed (additive, non-Go):
- `compose.yaml` — `postgres → migrate → seed → api` DAG. Build anchor
  (`x-app-image`) so migrate+api share one locally-built image; prebaked conn
  string anchor (`x-database-url`). postgres healthcheck = `pg_isready`;
  `migrate` one-shot = `["migrate","up"]`; `seed` one-shot = `postgres:17-alpine`
  running an sh loop applying `/sql/*.sql` (`ON_ERROR_STOP=1`, no-ops empty);
  `api` = serve with `TEMPLATE_GO_API_STORE=postgres` + the conn string, ports
  8080/9090. **No persisted volume** (ephemeral). Compose interpolation escaped
  with `$$` in the seed script.
- `hack/sql/0001_seed_todos.sql` — 3 example todos (ON CONFLICT DO NOTHING).
- `hack/sql/README.md` — documents the drop-in seed convention.
- `README.md` — new "## Local stack (Docker Compose)" section (step table +
  ephemerality + hack/sql) and `compose.yaml`/`hack/sql/` in the layout tree.

Functional verification (real `docker compose up --build`, Docker 29.4 / Compose
v5.1.2): DAG ran in order — postgres healthy → migrate `Exited (0)` (goose v1
applied) → seed `Exited (0)` ("applied 1 file(s)") → api `Up`. `GET /readyz` →
`{"status":"ready","checks":{"postgres":"ok"}}`; `GET /todos` returned the 3
seeds; `POST /todos` persisted a 4th (write path OK). **Ephemerality proven:**
`down` + `up` reset to exactly the 3 seeds, dropping the POSTed row.

Next: confirm `moon run root:check` is green, commit, push, open PR.

## 2026-06-23 15:28 — PR opened
`moon run root:check` green (9 tasks). Committed `03364a5`, pushed
`feat/compose-dev-stack`, opened **PR #7**
(https://github.com/meigma/template-go-api/pull/7). CI running (ci / GitHub
Pages / Kusari pending; release + container dry-runs correctly skip on this
branch). Watching checks to completion before handing back. Stack torn down
locally (`docker compose down`); `template-go-api:dev` image left cached.

CI green on PR #7: `ci` pass (58s), `GitHub Pages` pass (36s), `Kusari
Inspector` pass (49s); release/container/Pages-deploy dry-runs skip as expected
on a PR branch. PR ready for review/merge. Session work complete pending
merge — end goal (remove the in-memory tier, default to PostgreSQL-only) is a
separate follow-up, not in this PR.

## 2026-06-23 15:35 — Close
User approved ("LGTM"). **PR #7 squash-merged to `master` `8b68bd4`**; remote
branch deleted. Local `master` fast-forwarded `18b56e7..8b68bd4`; session
worktree `feat/compose-dev-stack` removed via `wt remove` (tree matched master).
Other worktrees left untouched (`journal/jmgilman`; session 007's
`refactor/domain-coupled-internal`). On close-out the only dirty journal path was
`007/NOTES.md` (a concurrent session) — committed as `docs(journal): checkpoint
session 007` before the sync rebase; 007's status untouched.

Recorded: `SUMMARY.md` written; `INDEX.md` row 006 → complete; `TECH_NOTES.md`
updated (compose stack + planned memory-tier removal). Handoff: feature is live
on `master`; next up is the postgres-only refactor (drop `memory` adapter +
`--store`). Session 006 closed.
