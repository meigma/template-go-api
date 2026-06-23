---
id: 006
title: Docker Compose day-one stack (API + PostgreSQL)
date: 2026-06-23
status: complete
repos_touched: [template-go-api]
related_sessions: ["004"]
---

## Goal
Stand up a Docker Compose stack so a day-one run of the template exercises the
**full** API + PostgreSQL backend with one command, including a drop-in mechanism
to prepopulate the local database with arbitrary SQL — without touching
migrations or adding setup code to the server.

## Outcome
Met. Shipped as **PR #7, squash-merged to `master` `8b68bd4`**. `docker compose
up --build` brings up `postgres → migrate → seed → api` as an ordered DAG and
serves the template against PostgreSQL. Functionally verified against a real
`docker compose up` (Docker 29.4 / Compose v5.1.2) and `moon run root:check`
(green, 9 tasks); PR CI green (ci / Pages / Kusari). **No Go code changed** — the
work is additive (`compose.yaml`, `hack/sql/`, README docs) and reuses the
existing Dockerfile unchanged.

## Key Decisions
- **Sequenced DAG, NOT the Postgres init-dir.** Seeds depend on the `todos`
  table, which only exists after `migrate up` (explicit, binary-run, never on
  serve, never by Postgres). Mounting `hack/sql` into
  `/docker-entrypoint-initdb.d/` would run it at DB first-boot *before*
  migrations and fail. So: postgres (healthcheck) → migrate (one-shot `migrate
  up`) → seed (one-shot) → api (`serve --store=postgres`), wired with
  `depends_on` health/completed conditions.
- **Seed runs in a `postgres:17-alpine` one-shot via psql** — not in the server
  binary and not as a migration (the boundary the user asked for). An `sh` loop
  applies every `/sql/*.sql` (sorted, `ON_ERROR_STOP=1`), no-ops on an empty dir.
  Compose interpolation escaped with `$$` in the inline script.
- **Ephemeral/reproducible DB (user choice).** No persisted volume: every `up`
  rebuilds a clean, migrated, seeded database; `down` discards it. Seeds need not
  be idempotent (the shipped example still uses `ON CONFLICT` so it is safe to
  re-run by hand).
- **Reused the existing Dockerfile** (it already exists; see Lessons). Multi-stage
  static + distroless/nonroot with ENTRYPOINT = the binary, so `migrate up` and
  `serve` are just args; `migrate` and `api` share one locally-built image via a
  compose build anchor (`x-app-image`). Prebaked conn string anchor.
- **`compose.yaml`** (modern filename) at repo root; documented in README. No
  moon task — a long-running interactive process doesn't fit moon's cache model.

## Changes
- `compose.yaml` (new) — `postgres`/`migrate`/`seed`/`api` DAG; build +
  database-url YAML anchors; `pg_isready` healthcheck; ports 8080/9090; no volume.
- `hack/sql/0001_seed_todos.sql` (new) — 3 example todos (`ON CONFLICT DO NOTHING`).
- `hack/sql/README.md` (new) — documents the drop-in seed convention.
- `README.md` — new "Local stack (Docker Compose)" section (step table +
  ephemerality + `hack/sql`) and `compose.yaml`/`hack/sql/` in the layout tree.

## Open Threads
- **Remove the in-memory tier / default the template to PostgreSQL-only** — the
  user's stated *end goal*, intentionally **not** in this PR. The compose stack
  sets `--store=postgres` explicitly today; dropping the memory adapter and the
  `--store` toggle (to cut boilerplate consumers would delete) is the planned
  next change.
- The compose stack is local-dev only and not wired into CI (the integration /
  docker GitHub workflows remain `.disabled`, needing a Docker-capable runner —
  carried from session 004).

## References
- PR #7: https://github.com/meigma/template-go-api/pull/7 (merged, `8b68bd4`)
- Persistence tier this builds on: `.journal/004/SUMMARY.md` (PR #6, `18b56e7`)
- Session log: `.journal/006/NOTES.md`
- Memory: `separate-mechanical-from-design-work`

## Lessons
- **zsh's `nomatch` aborts a whole compound command when any glob is empty.**
  `ls Dockerfile* docker-compose* compose* hack` printed *nothing* — not because
  the Dockerfile was missing, but because `docker-compose*`/`compose*` matched no
  files and zsh killed the entire `ls`. The Dockerfile (and `.dockerignore` /
  `.go-version`) existed all along. When probing for optional files, glob each
  independently or guard with `2>/dev/null`, and confirm a "missing" file with an
  explicit `ls <path>` before building a new one.
- **Verify the ephemeral-reset claim, don't just assert it.** Proving
  "reproducible" meant actually running `down` + `up` and re-counting rows (the
  POSTed 4th gone, the 3 seeds back) — not reasoning that "no volume implies a
  clean DB."
