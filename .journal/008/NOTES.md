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

## 2026-06-23 16:10 — Investigation + design decision
Clarified with the user: the `todo.Repository` **port stays** — we are removing
one adapter option, not collapsing the abstraction. Integrators must still be
able to add their own adapters by implementing the port.

How the memory adapter (`internal/todo/memory`) is used today — TWO roles:
- **Production store option**: `app.selectStore()` returns it whenever
  `cfg.Store != postgres`, and `memory` is the *default* (`config.go`
  `defaultStore = StoreMemory`). `app.OpenAPIYAML()` also news up a memory repo
  purely to construct the service for server-less spec generation — the repo is
  never actually called.
- **Test double (critical)**: `internal/todo/httpapi/api_test.go` `newTestServer`
  backs the HTTP functional tests (`TestTodoAPIFunctional`,
  `TestServiceLogCarriesRequestID`) with a real `memory.NewTodoRepository()`.
  `internal/app/app_test.go` `TestAppWiring` implicitly exercises it as the
  default store. NOT used by `service_test.go` (own hand-rolled `fakeRepo`),
  `memory/repository_test.go` (dies with the adapter), or the postgres
  integration suite.

go-testing skill deviation confirmed: the skill mandates **mockery** for all
mocks ("No exceptions"); the repo ships zero mockery setup and hand-rolls fakes.
Worth fixing while we're here.

**Decision (user picked "Mockery + small fake"):**
- Adopt **mockery** (Proto-pinned CLI + `.mockery.yaml` + moon task + drift-check,
  matching the sqlc/goose convention). Generate `mocks.Repository` for
  `todo.Repository`.
- Use the generated mock for the interaction/error-injection unit tests
  (`service_test.go` — esp. the `listErr` case).
- Use a tiny **test-only in-memory fake** for the stateful HTTP round-trip
  (`httpapi/api_test.go`) — a scripted mock fits that flow poorly (runtime-gen ID
  flows POST→GET→List→complete).
- Not promoting these to testcontainers (user explicit).

Mechanical wrinkle noted: after removing memory there is no harmless repo for
`OpenAPIYAML` server-less spec generation — will introduce a minimal no-op stub
repo (prod-safe, never persists) since memory previously filled that role.

## 2026-06-23 16:20 — Finalized scope (both design forks settled)
`TestAppWiring` fork → user picked **"Add a repo-injection seam"**: give
`app.New` a functional option (`app.WithRepository`) that skips the postgres
connect when a store is injected; the wiring test injects the `todotest` fake and
stays a fast no-DB unit test. Bonus alignment: the seam also lets integrators
wire a custom adapter without editing `selectStore`.

Full agreed scope (blast radius verified by grep):
1. **mockery tooling** — `.moon/proto/mockery.toml` + `.prototools` pin;
   `.mockery.yaml` for `todo.Repository`; `moon.yml` `mockery` + `mockery-check`
   (mirror `sqlc-check`'s mktemp + `git diff --no-index`); add to `root:check`.
   Generated mock committed at `internal/todo/mocks` (per-domain). golangci
   auto-excludes the generated header; existing `testify/mock.Mock` exclusion
   (`.golangci.yml:279`) covers the embedded type.
2. **Test doubles** — new stateful fake `internal/todo/todotest`
   (`todotest.NewRepository()`) for `httpapi/api_test.go`; rewrite
   `service_test.go` onto generated `mocks.Repository` (listErr →
   `.On("List").Return(nil,err)`; fixed clock/ID make expected todos
   deterministic).
3. **Remove memory + `--store`** — delete `internal/todo/memory/`; `config.go`
   drop `Store` type/consts/`defaultStore`/`--store` flag/store-switch,
   `--database-url` now ALWAYS required (clear error if missing); update
   `config_test.go`; `app.go` collapse `selectStore`→always-postgres + drop
   memory import/branch; reword in-memory doc comments in
   `app.go`/`serve.go`/`ports.go`; drop redundant `--store=postgres` from
   `compose.yaml`.
4. **No-op stub repo** — minimal unexported prod repo in `app` for `OpenAPIYAML`
   server-less spec emit (never persists).
5. **Docs** — README (persistence section, flags table, layout tree, serve
   example) + `DELETE_ME.md` + `docs/docs/index.md`: postgres-only,
   `--database-url` required, drop `--store`, add a short mockery/testing note;
   reword integration "in-memory peer" comments.
6. **Verify** — `moon run mockery`, `moon run openapi` (spec must stay
   byte-identical — same handlers), `moon run root:check` (+ new
   `mockery-check`), `moon run test`, `moon run test-integration` (Docker).

Next: create impl worktree off `origin/master` (`refactor/postgres-only-store`)
and execute. PR title TBD at PR time (likely `refactor(store): remove in-memory
adapter and ship PostgreSQL-only`).

## 2026-06-23 16:34 — Implemented + PR #9 opened
Built on worktree `.wt/refactor-postgres-only-store` (branched off
`origin/master` `1f1e5a7`). Shipped as **PR #9**:
https://github.com/meigma/template-go-api/pull/9 (open, CI pending at handoff).
Commit `16e39e1` `refactor(store): drop the in-memory adapter, ship
PostgreSQL-only`.

What landed (matches the agreed scope):
- **mockery adopted**: Proto pin `mockery=3.7.1` (`.moon/proto/mockery.toml`,
  assets `mockery_{version}_{Darwin|Linux}_{x86_64|arm64}.tar.gz`, arch map
  `aarch64→arm64`, `x86_64→x86_64`); `.mockery.yaml` testify template →
  `internal/todo/mocks/mock_Repository.go` (type `mocks.Repository`, ctor
  `NewRepository(t)` auto-asserts). `moon.yml` `mockery` + `mockery-check`
  (sed-swaps `dir` in a temp config, `git diff --no-index`, mirrors `sqlc-check`)
  added to `root:check`. `go mod tidy` pulled in `stretchr/objx` (go.sum already
  had it). golangci auto-excludes the `// Code generated by mockery` header.
- **service_test.go** → external `package todo_test`, rewritten on the generated
  mock. `TestServiceComplete` restructured to inject the existing todo via
  FindByID (no create-then-read), so every case is interaction/error-based — the
  mock fits cleanly, no stateful awkwardness.
- **todotest fake** (`internal/todo/todotest`, copy of the old memory adapter)
  backs the stateful HTTP functional tests (`httpapi/api_test.go`) and
  `app_test.go` (via the new `app.WithRepository` option).
- **Memory tier removed**: deleted `internal/todo/memory/`; `config` dropped the
  `Store` type/consts/`--store` flag, `--database-url` now always required;
  `app` collapsed `selectStore`→`resolveStore` + `WithRepository` seam +
  `noopRepository{}` for `OpenAPIYAML` spec gen; reworded in-memory comments in
  `app.go`/`serve.go`/`ports.go`/integration; dropped `TEMPLATE_GO_API_STORE`
  from `compose.yaml`.
- **Docs**: README (intro/prereqs/quickstart→compose-first/flags/Persistence/new
  Testing section/layout/add-a-resource), `DELETE_ME.md`, `docs/docs/index.md`.

Verification: `moon run root:check` green (10 tasks, incl. new `mockery-check`);
OpenAPI spec byte-identical; `moon run root:test-integration` green vs real
`postgres:17-alpine` (~4.4s). Functional: `serve` with no `--database-url` exits
1 with `invalid configuration: database-url is required`; `--store` gone from
`serve --help`.

Decision worth keeping: `app.WithRepository` was added as the seam so the
no-DB app-wiring unit test survives postgres-only — it doubles as the
integrator extension point (wire a custom adapter without editing
`resolveStore`). The port did NOT collapse (user was explicit).

Open: awaiting PR #9 review/merge + CI. After merge, `wt remove` the impl
worktree and close the session.

## 2026-06-23 16:47 — Merged + cleaned up
User approved (LGTM). CI green (ci, GitHub Pages, Kusari all pass; dry-run/deploy
jobs skipped as expected). **PR #9 squash-merged to `master` `8a46286`**
(`refactor(store): drop the in-memory adapter, ship PostgreSQL-only (#9)`).
Cleanup: deleted remote branch `refactor/postgres-only-store`; `wt remove`'d the
impl worktree (gone — only `master` + `journal/jmgilman` remain); local `master`
fast-forwarded to `8a46286`.

Mid-session aside the user raised (worth keeping): `mocks` has exactly ONE real
consumer (`service_test.go`); `todotest` has two (`app_test.go`,
`httpapi/api_test.go`). Kept mockery anyway — it's irreplaceable for the
error-injection test (`TestServiceListPropagatesError`; the fake's `List` never
errors) and "Save never called" assertions, and as a template it's the
skill-mandated reference pattern downstream inherits. User agreed: fine as-is.

Session goal met. Ready for `session-close`.

## 2026-06-23 16:59 — Close
Session closed. Goal met: in-memory tier + `--store` removed, PostgreSQL-only,
`--database-url` required; mockery adopted for repository test doubles.
- Merged: **PR #9** `8a46286` (`refactor(store): drop the in-memory adapter,
  ship PostgreSQL-only (#9)`), squash-merged to `master`.
- Repo `template-go-api`: local `master` fast-forwarded to `8a46286`; impl
  worktree removed; remote branch deleted. No journal contamination on master
  (`git ls-files .journal` empty).
- Journal: wrote `008/SUMMARY.md`; flipped INDEX 008 → complete; revised
  `TECH_NOTES.md` (memory tier removed, `--store` gone, `app.WithRepository`
  seam, mockery convention; "planned PostgreSQL-only" → done).
- Note: session **005** is still a dangling empty `in-progress` row (left
  untouched per the close-target decision); future cleanup could mark it
  `abandoned`.
