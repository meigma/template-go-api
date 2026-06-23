---
id: 007
title: Explore restructuring internal/ to couple domain code
started: 2026-06-23
---

## 2026-06-23 15:12 — Kickoff
Goal for the session: explore how plausible it is to refactor the current
`internal/` package structure toward coupling domain code into one logical
package hierarchy, rather than the present split across multiple top-level
packages. This is an exploration/feasibility session — no implementation yet.

Current state of the world: the template is fully built and merged through PR #6
(`18b56e7`). `internal/` currently splits a single domain (`todo`) across several
top-level packages following pragmatic ports & adapters:
- domain: `internal/todo`
- adapters: `internal/adapter/{memory,http,postgres}` (+ `http/middleware`,
  `http/problem`, `http/todoapi`)
- cross-cutting: `internal/{config,observability,logctx,app,cli,integration}`

The user's framing: today a domain's code (e.g. `todo`) is spread across
`internal/todo`, `internal/adapter/memory`, `internal/adapter/http/todoapi`,
`internal/adapter/postgres`, etc. The question is whether to instead group all of
a domain's code under one logical hierarchy (e.g. everything `todo`-related
nested together) and how plausible/desirable that is for the template.

Plan: paused after session setup per the user's instruction. Awaiting the user's
detailed framing and constraints before exploring options.

## 2026-06-23 15:25 — Feasibility assessment of domain-nested layout
User's concrete proposal: nest each domain's adapters under the domain package —
`internal/todo` (core) + `internal/todo/adapters/{http,postgres,memory}`; shared
adapter code stays at `internal/adapter/...` and is imported as needed; HTTP
registration stays manual/composed in `app`. Asked me to double-check the
circular-dependency reasoning + give honest "is it cleaner" feedback.

Verified the real production import graph (grep, test files excluded) and the
registration seam (`adapter/http/api.go`, `router.go`, `todoapi/handler.go`,
`app/app.go`):
- `todo` → `logctx` only; imports NOTHING from any adapter (strictly inward).
- `adapter/memory` → `todo`; `adapter/postgres` → `todo` + `postgres/sqlc`.
- `adapter/http/todoapi` → `todo` (+ huma). It does NOT import `adapter/http`.
- `adapter/http` (shared, resource-agnostic) → `middleware`, `problem`,
  `observability`; does NOT import `todoapi`.
- Registration seam: `adapter/http` defines `type Registrar func(huma.API)`;
  `todoapi.Register(api huma.API, svc *todo.Service)`; `app.registerResources`
  adapts one to the other. todoapi and adapter/http meet ONLY at `app`.

Verdict given:
1. **Cycle fear is unfounded.** Go package nesting confers no special import
   relationship — `internal/todo/adapters/http` is just a sibling package with a
   shared path prefix. No new edges. Since the domain never imports its adapters
   (hexagonal), arrows stay one-directional (adapter→domain); no cycle possible.
   Caveat flagged: keep a single root `internal/` — do NOT nest a 2nd `internal/`
   (would hide packages from `app` and break registration). Visibility unchanged.
2. **Already textbook-hexagonal** — the "spread" is physical, not coupling. The
   refactor buys organization, not decoupling.
3. **Cleaner? Real but mostly at 2+ domains.** Upside: co-location + fixes a
   scaling wart (today a 2nd domain's repo would pile into the single shared
   `adapter/postgres` pkg — go-style anti-pattern). Strongest arg = pedagogical
   (template should model a self-contained domain). Costs: package-name
   collisions across domains (`http`/`postgres` pkgs → mandatory aliases, vs
   today's unique `todoapi`); a standing "shared vs per-domain" judgment call.
4. **Refinements pushed:** (a) drop the `adapters/` layer → flatter
   `internal/todo/{httpapi,postgres,memory}` (dodges name collisions too);
   (b) DB migrations/schema/pool are DB-level, NOT domain-level — keep shared
   (`internal/postgres`), move only todo's queries+sqlc+repository under the
   domain; (c) mind non-code path refs: `sqlc.yaml` output + `root:sqlc-check`,
   migrations embed path, `moon.yml` tasks, `internal/integration` imports.

Bottom line: feasible, low-risk, almost entirely mechanical (per memory
`separate-mechanical-from-design-work`, settle the exact target shape first, then
the move is mechanical). Offered to sketch the concrete target tree or prototype
the move on a worktree to prove it compiles. Awaiting user direction.

## 2026-06-23 15:55 — Decisions + plan + implementation
User chose the **flat** model (`internal/todo/{httpapi,memory,postgres}`, no
`adapters/` layer) and to keep **DB schema/migrations/pool infra shared**.
Clarified via AskUserQuestion: integration tests **stay** in `internal/integration`.
Plan written + approved (`~/.claude/plans/concurrent-chasing-wirth.md`).

Key refinement found while reading files: migrations are already a sibling of the
shared pool/migrate code, so the lowest-churn split is to keep ALL shared infra
exactly where it is (`internal/adapter/http`, slimmed `internal/adapter/postgres`)
and move only the todo-specific files out. Symmetric result: `internal/adapter/` =
shared cross-domain infra; `internal/todo/` = todo's own code.

Implemented on worktree `refactor/domain-coupled-internal` (off master):
- `git mv` 3 groups → `internal/todo/{httpapi,memory,postgres}` (history preserved).
  Package `todoapi`→`httpapi`; `memory`/`postgres` names unchanged.
- `internal/adapter/postgres` keeps `postgres.go`/`migrate.go`/`migrations.go`/
  `migrations/`; todo repo+mapping+queries+sqlc moved out.
- `app.go` + integration fixture alias the per-domain postgres import as
  `todopostgres` (clash with shared `internal/adapter/postgres`, both pkg `postgres`).
- `sqlc.yaml`/`moon.yml`: queries+out → `internal/todo/postgres/{queries,sqlc}`;
  schema/migrations paths unchanged. Regenerated sqlc → byte-identical (drift clean).
- Updated `api.go` doc comment, README (layout + persistence + add-a-resource),
  DELETE_ME (add-a-resource + delete-SQL-tier).

Verification ALL green: `moon run root:check` (incl openapi-check byte-identical +
sqlc-check no drift), `moon run root:test-integration` (postgres:17-alpine, 5.5s),
`go build`/`go vet`. Gotcha: a stale golangci-lint cache falsely flagged `modernize`
on the moved generated `sqlc/db.go`; `golangci-lint cache clean` cleared it (the
generated-file exclusion works fine) — matches the worktree-tooling-flakiness theme.

**Shipped as PR #8** (https://github.com/meigma/template-go-api/pull/8), not yet
merged. No go.mod/go.sum changes (pure move). CI: only Kusari runs (ci/Pages are
`.disabled`); **Kusari passed (21s)** — PR fully green, ready to squash-merge.
Awaiting user go-ahead to merge.
