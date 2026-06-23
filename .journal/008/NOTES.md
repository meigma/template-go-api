---
id: 008
title: Remove the memory adapter and ship PostgreSQL-only
started: 2026-06-23
---

## 2026-06-23 16:00 — Kickoff
Goal for the session: Remove the in-memory (`memory`) adapter layer and promote
the PostgreSQL adapter to be the **only** persistence layer shipped with the
template. This realizes the long-planned "PostgreSQL-only" direction recorded in
TECH_NOTES and in sessions 006/007's open threads — dropping the `memory`
adapter and the `--store` toggle to cut boilerplate that template consumers
would otherwise delete.

Current state of the world:
- The template is built and merged through PR #8 (`1f1e5a7`, master).
- Persistence is selected at runtime via `--store=memory|postgres`; `memory` is
  the zero-infra default (PR #6 `18b56e7`).
- Per-domain layout (PR #8): `internal/todo` core + `internal/todo/{httpapi,memory,postgres}`;
  shared infra under `internal/adapter/{http,postgres}`. The memory adapter to
  remove is `internal/todo/memory`.
- The Docker Compose day-one stack (PR #7 `8b68bd4`) already runs
  `--store=postgres` explicitly, so it assumes postgres today.
- Removing the memory tier touches: `internal/todo/memory` (delete), the
  `--store`/store-selection plumbing in `internal/config` + `internal/app`,
  the shared adapter test that asserts identical memory/postgres behavior,
  docs (README/DELETE_ME persistence sections), and any compose flag that is now
  redundant.
- Session 005 remains `in-progress` in INDEX.md (pre-existing dangling entry,
  out of scope for this session).

Plan: (rough) survey the current memory/store wiring, agree on the desired
shape (e.g. does postgres stay behind a port with no toggle; what happens to the
shared adapter contract test; default flags; README/compose cleanup), then
execute on an implementation worktree branched from `origin/master` and ship via
a squash-merged PR. Awaiting the user's go-ahead to dig in.
