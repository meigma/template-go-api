---
id: 005
title: Authorization tier — Cedar middleware seam with deferred API-key authn
date: 2026-06-23
status: complete
repos_touched: [template-go-api]
related_sessions: ["004", "006", "007", "008"]
---

## Goal
Fill the long-open authn/authz future-slice seam with a sensible, loosely-coupled
authorization starting point for the template: an authorization decision expressed
as **middleware between endpoints**, with **authentication deferred** to the
integrator. Research the Go ecosystem first (do not build an engine from scratch),
settle the design collaboratively, then build it.

## Outcome
Met. Two `deep-research` passes → committed to **AWS Cedar (`cedar-go`)** → a
collaborative design (`.journal/005/AUTHZ_TIER.md`) → a **four-phase gated build**
(implement → 3-lens adversarial review → fix → validate, human gate between phases)
→ shipped as **PR #10, squash-merged to `master` `13a1fe5`** (+3997/−35, 49 files).
CI green (ci / Pages / Kusari — no supply-chain blocker from the new deps). Verified
across phases: `moon run root:check`, unit tests, a container-backed integration suite
(`postgres:17-alpine`), and a **live `docker compose` day-one demo** (no key → 401,
seeded key → 200, admin override, Bearer, by-id). The domain core (`internal/todo`)
was not touched.

## Key Decisions
- **Cedar via `cedar-go`, committed as *the* engine, no portability layer.** Research
  found it the best embeddable in-process fit (Apache-2.0, synchronous PARC `Authorize`,
  `Diagnostic` reasons). Committing lets us expose Cedar's real types behind a thin
  `internal/authz` rather than a lossy vendor-neutral `Decision` port + DTO layer.
  (Casbin = strongest in-process alternative; Cerbos/SpiceDB/Permify/Topaz are
  service/behind-a-port; Oso OSS deprecated → Oso Cloud.)
- **Authentication deferred** via an `Authenticator` seam; shipped a replaceable
  **API-key** adapter (postgres `api_keys`, `X-API-Key`/`Bearer`) as the day-one
  starting point — a real-but-minimal mechanism (not header impersonation) that
  integrators swap for JWT/OIDC/session.
- **Modular per-resource authz slices** (`internal/<domain>/authz` contribute
  policies + actions + a fact resolver, merged at the composition root) — mirrors the
  HTTP registrar; "modular authoring, unified evaluation" (one merged `PolicySet` +
  one entity space; namespaced actions/types; the principal type is reserved).
- **Deny-by-default** global Huma middleware; `Require`/`Public` declarations also
  stamp the OpenAPI security requirement so protection shows in the docs.
- **URL-fed resource identity** (path param → `Resource = Todo::"<id>"`, no load) +
  a **lazy, request-scoped, fail-closed** composite `EntityGetter` for attribute
  policies (binds ctx at construction, captures the first load error → 500).
- **Mock keys ship via the dev-only `hack/sql` seed, NOT a migration** → safe by
  construction (migrations run everywhere; seeds touch only the ephemeral compose DB,
  so mock creds can never reach a real deploy). The `api_keys` *table* is a goose
  migration; `omit_unused_structs` in `sqlc.yaml` keeps the shared migration from
  bleeding an `ApiKey` model into the todo sqlc package (and keeps removal regen-free).
- **Built via a gated multi-agent workflow**, one run per phase, with me holding the
  human gate between phases (per `separate-mechanical-from-design-work`).

## Changes
All in PR #10 (`13a1fe5`); `internal/todo` core unchanged.
- `internal/authz` — base engine: `Authorizer`/PolicySet merge, `Contribution` model,
  `Principal` + context, `Authenticator` seam, lazy composite `EntityGetter`,
  `Require`/`Public` + OpenAPI security (Install/Finalize split), deny-default
  middleware, principal resolver, `base.cedar`.
- `internal/authz/apikey` — API-key `Authenticator` + `APIKeyStore` port + postgres
  adapter (hand-written pgx, no second sqlc package).
- `internal/todo/authz` — todo slice: `actions.go`, `policy.cedar`, fact resolver,
  contribution (imported as `todoauthz`).
- `internal/adapter/postgres/migrations/00002_create_api_keys.sql`;
  `hack/sql/0002_seed_api_keys.sql` (dev-only mock keys).
- `internal/config` (`--authz-enabled` default true, `--authz-policy-dir`);
  `internal/app` (authz wiring + `WithAuthenticator` test seam);
  `internal/adapter/http/router.go` (Install/Finalize); `internal/todo/httpapi` (route
  tagging + URL-id binding).
- `internal/integration/{apikey_store_test,authz_e2e_test}.go` (container-backed).
- `sqlc.yaml` `omit_unused_structs`; `.mockery.yaml` + `moon.yml` mockery for the new
  ports; README / DELETE_ME / `docs/docs/index.md` authz docs; `docs/docs/openapi.yaml`
  security.

## Open Threads
- **Wire `test-integration` into CI** — pre-existing; GitHub workflows are `.disabled`,
  need a Docker-capable runner. Unchanged by this session.
- **Attribute/relationship policies** (e.g. `resource.owner == principal`) are a
  documented extension: the lazy getter supports them, but the todo domain has no owner
  field, so the shipped policy is coarse/role-based.
- **Replace the API-key authenticator with real authn (JWT/OIDC)** is the integrator's
  #1 task (DELETE_ME).
- Minor, intentionally deferred: the e2e test leaks the app's pgx pool (test-only,
  no prod `App.Close`); deny subtests assert status, not the RFC 9457 body; no
  full-stack public/undeclared e2e route (covered at the middleware layer);
  `ActionDelete` declared without a delete route (full CRUD vocab).

## References
- PR #10: https://github.com/meigma/template-go-api/pull/10 (merged, `13a1fe5`)
- Design doc: `.journal/005/AUTHZ_TIER.md`; research: `.journal/005/RESEARCH-authz-middleware.md`
- Builds on: `.journal/004/SUMMARY.md` (postgres tier), `.journal/006/SUMMARY.md`
  (compose + `hack/sql` seed hook), `.journal/007/SUMMARY.md` (per-domain layout),
  `.journal/008/SUMMARY.md` (postgres-only + mockery)
- Memory: `separate-mechanical-from-design-work`, `subagents-may-read-divergent-worktree`,
  `gh-merge-delete-branch-worktree-fails`

## Lessons
- **Huma freezes an operation's middleware at `huma.Register` time.** So the authz
  middleware must be installed BEFORE `Register` and the OpenAPI security stamped AFTER
  (the Install/Finalize split). "Register routes, then install authz" silently bypasses
  enforcement — invisible to a passing build, caught only by a deny test returning 201.
  The most dangerous bug of the build.
- **Adversarial review earns its keep.** It caught a precedence-fix regression that
  would have silently broken role membership for any custom (non-`User`) principal type —
  i.e. the documented JWT/OIDC extension seam — failing closed. Routing the principal
  resolver by static reserved types was wrong; register it under the actual UID type too.
- **For a copyable template, dev/mock secrets belong in the dev-only seed hook
  (`hack/sql`), never a migration** — migrations run in every environment, seeds don't.
  Safe-by-construction beats "remember to remove it."
- **sqlc emits a model for every table in its schema dir;** `omit_unused_structs` stops
  a shared-migrations layout from bleeding cross-domain models and makes feature removal
  regeneration-free.
- **`gh pr merge --delete-branch` fails its LOCAL cleanup when the default branch is
  checked out in another worktree** (the squash merge still succeeds server-side) —
  finish remote-branch + worktree cleanup by hand. See memory
  `gh-merge-delete-branch-worktree-fails`.
