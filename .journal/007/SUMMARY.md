---
id: 007
title: Restructure internal/ to couple each domain's code under one hierarchy
date: 2026-06-23
status: complete
repos_touched: [template-go-api]
related_sessions: ["002", "003", "004"]
---

## Goal
Evaluate, then (after approval) execute, a restructure of `internal/` so each
domain's code lives in one logical package hierarchy instead of being split
across top-level `internal/adapter/*` packages. Start as a feasibility check —
was the user's circular-dependency worry founded? — then plan and ship if sound.

## Outcome
Met. Feasibility confirmed, then shipped as **PR #8, squash-merged to `master`
`1f1e5a7`**. The `todo` resource's adapters now live under
`internal/todo/{httpapi,memory,postgres}`; shared cross-domain infrastructure
stays under `internal/adapter/{http,postgres}`. Pure reorganization — OpenAPI
spec and generated sqlc output byte-identical; full `root:check` and the
container-backed integration suite green. `master` fast-forwarded to `1f1e5a7`;
the implementation worktree was removed.

## Key Decisions
- **Flat per-domain layout, no `adapters/` layer** (`internal/todo/httpapi`, not
  `internal/todo/adapters/http`) — shallower, and distinct package names
  (`httpapi`) shrink the cross-domain import-collision surface.
- **DB schema/migrations/pool stay shared** under `internal/adapter/postgres` —
  schema is a database-level concern, not a domain one. Only the todo-specific
  repository + mapping + queries + generated sqlc moved to `internal/todo/postgres`.
  Bonus: the shared infra files never moved, so churn stayed minimal.
- **Generic HTTP transport stays at `internal/adapter/http`** — resource-agnostic;
  domains plug in through the existing `Registrar func(huma.API)` seam composed in
  `app.go`. Only import paths changed, not the seam.
- **Integration tests stay in `internal/integration`** (user choice) — preserves
  the one-home convention over per-domain colocation.
- **The `sqlc/` generated package moves as a unit** — sqlc emits its `db.go`
  boilerplate per package, so the whole generated dir + `queries/` move with the
  domain; `sqlc.yaml` `out`/`queries` repointed, `schema` (migrations) unchanged.
- **`todopostgres` import alias** in `app.go` + the integration fixture — the
  per-domain `postgres` package collides by name with the shared
  `internal/adapter/postgres`; aliasing is the idiomatic resolution.

## Changes
All in PR #8 (`1f1e5a7`). Pure move + path updates; no behavior change.
- Moved (history preserved via `git mv`): `internal/adapter/http/todoapi` →
  `internal/todo/httpapi` (pkg `todoapi`→`httpapi`); `internal/adapter/memory` →
  `internal/todo/memory`; `internal/adapter/postgres/{repository,mapping}.go` +
  `queries/` + `sqlc/` → `internal/todo/postgres/`.
- Slimmed `internal/adapter/postgres` keeps `postgres.go` (Connect/Config),
  `migrate.go`, `migrations.go`, `migrations/`.
- `internal/app/app.go` — import paths + `todopostgres` alias + `httpapi.Register`.
- `internal/integration/postgres_fixture_test.go` — `todopostgres` alias.
- `internal/adapter/http/api.go` — doc-comment example path.
- `sqlc.yaml`, `moon.yml` — queries + sqlc paths (migrations path unchanged).
- `README.md`, `DELETE_ME.md` — directory layout + persistence paths + the
  "add a resource" guidance now teaches the flat per-domain shape.

## Open Threads
- The convention is documented but only modeled with one domain (`todo`). A
  second domain will need import aliases for the like-named `httpapi`/`postgres`
  packages in `app.go` (README notes this).
- Interaction with the planned PostgreSQL-only direction (TECH_NOTES): if the
  `memory` tier is dropped, `internal/todo/memory` goes with it.
- Pre-existing future-slice seams unchanged: authn/authz, OTel tracing, rate
  limiting, pagination, API versioning, mockery; wiring `test-integration` into
  CI (workflows still `.disabled`).

## References
- PR #8: https://github.com/meigma/template-go-api/pull/8 (merged, `1f1e5a7`)
- Plan: `~/.claude/plans/concurrent-chasing-wirth.md`
- Prior slices: `.journal/002/SUMMARY.md`, `.journal/003/SUMMARY.md`, `.journal/004/SUMMARY.md`
- Memory: `separate-mechanical-from-design-work`, `subagents-may-read-divergent-worktree`

## Lessons
- **Go package nesting is purely cosmetic to the compiler.** A nested path
  (`internal/todo/postgres`) confers no special import relationship with its
  parent; cycle-freedom depends only on the actual edge directions, which
  hexagonal design already keeps one-way (adapter → domain). The user's
  circular-dependency worry was unfounded for exactly this reason — worth stating
  plainly when someone proposes nesting domain code.
- **A stale golangci-lint cache can falsely flag moved generated files.** After
  the move, `moon run root:check` reported a `modernize` hit on the relocated
  generated `sqlc/db.go`, while direct invocation showed 0 issues. `golangci-lint
  cache clean` cleared it — the generated-file exclusion works fine. Same
  worktree-tooling-flakiness theme as the gopls/go.work note.
- **Branch off the current default when other sessions are active.** This branch
  was cut from `18b56e7` while session 006's PR #7 (`8b68bd4`) landed mid-session,
  so the merge hit a README conflict (both edited the directory-layout block). A
  rebase onto `origin/master` + re-running the gate resolved it cleanly; rebasing
  before opening the PR would have avoided the surprise at merge time.
- **`gh pr merge --delete-branch` fails its local cleanup when the default branch
  is checked out in another worktree** ("master is already used by worktree …").
  The server-side squash-merge still succeeds; finish by deleting the remote
  branch manually (`git push origin --delete`) and `wt remove` the worktree.
