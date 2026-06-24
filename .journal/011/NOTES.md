---
id: 011
title: Finalize the repo
started: 2026-06-24
---

## 2026-06-24 09:51 — Kickoff
Goal for the session: finalize this repository today. The specific scope of
"finalize" is not yet stated — awaiting the user's concrete request before
starting substantive work.

Current state of the world:
- The API-server template is feature-complete and on `master` at `f2c5210`
  (PR #15). Sessions 001–010 are all closed/complete in `INDEX.md`.
- Architecture (per `TECH_NOTES.md`): chi v5 + Huma v2 (code-first OpenAPI),
  per-domain ports & adapters under `internal/todo/{httpapi,postgres,authz}`
  with shared infra under `internal/adapter/{http,postgres}`; PostgreSQL-only
  persistence (pgx + sqlc + goose); Cedar authz with deferred API-key authn
  (now stored as SHA-256 hashes); Docker Compose day-one stack; keyset
  pagination on `GET /todos`; CI runs the container-backed integration suite on
  `ubuntu-latest`; sqlc binary integrity-pinned.
- Recent work (session 010) remediated all three findings from an independent
  Codex security review as separate squash PRs (#13/#14/#15).

Known open threads carried from prior sessions:
- Future-slice seams left as documented extension points (not built): OTel
  tracing, rate limiting, API versioning.
- Session **005** still shows `in-progress`/empty in `INDEX.md` (pre-existing).
- The main checkout has untracked local tooling dirs (`.agents/skills/
  codex-security-scan/`, `.codex-security/`, `.claude/`) — local artifacts vs.
  template content; disposition still open.
- The other three Proto plugins verify against unsigned upstream checksum files
  (defense-in-depth pinning was scoped out of #1).

Plan: wait for the user's concrete definition of "finalize," then scope and
sequence the work (worktree → PR → squash-merge per the session protocol).

## 2026-06-24 10:14 — API versioning: decision + plan

Scope for the session is "tie up the loose threads." Starting with **API
versioning** (a documented-but-unbuilt seam).

**Design decisions (user-confirmed via AskUserQuestion):**
- **Strategy: URI path versioning under `/v1`** (over header / media-type /
  date-based). The pragmatic, discoverable, cache-friendly default a reference
  template should teach.
- **Depth: establish the `/v1` seam only** — mount all current operations under
  `/v1`, keep infra routes unversioned, document how a future `/v2` group is
  added. Do NOT build a second version that doesn't exist yet.

**Grounding (verified against Huma v2.38.0 source in the module cache, not
memory):**
- Use Huma's first-class `huma.NewGroup(api, "/v1")` (group.go). A `Group`
  implements `huma.API`, so it drops into the existing `Registrar
  func(huma.API)` seam with no signature changes.
- `Group.Middlewares()` = parent API middlewares + group's own (group.go:180).
  `huma.Register` bakes `api.Middlewares()` into each op (huma.go:777). Since
  `InstallAuthz(api)` runs `api.UseMiddleware` BEFORE `registerResources`
  creates the group, the authz middleware is inherited by every `/v1` route —
  **no auth bypass**. This was the key risk and it's cleared.
- The group writes into the parent's shared `OpenAPI()` doc (group.go:109), so
  `FinalizeAuthz`/`DocumentSecurity` and the server-less `SpecYAML` export both
  pick up the `/v1` paths. Single prefix ⇒ OperationIDs/tags untouched
  (group.go:24 guard); `{id}` param preserved so `ctx.Param("id")` still works.

**Implementation (single composition point):** wrap resource registration in
`app.go`'s `registerResources` with `huma.NewGroup(api, apiVersionV1)`. Both the
running server (`NewRouter`) and the spec export (`OpenAPIYAML`→`SpecYAML`) flow
through `registerResources`, so the prefix propagates consistently. The
`httpapi` adapter stays version-agnostic (its own paths remain `/todos`).

**Blast radius:**
- UNCHANGED (isolation unit tests on a bare API): `httpapi/handler.go`,
  `httpapi/api_test.go`, `internal/authz/middleware_test.go`,
  `internal/todo/authz/{policy,facts}_test.go`, `adapter/http/router_test.go`
  (Register nil; its `/todos` is an arbitrary unrouted path).
- UPDATE to `/v1/todos` (composed/full-app): `internal/app/app_test.go`,
  `internal/integration/authz_e2e_test.go`; tighten `internal/cli/openapi_test.go`
  to assert `/v1/todos`.
- REGENERATE: `docs/docs/openapi.yaml` (via `root:openapi`).
- DOCS: `README.md`, `docs/docs/index.md`, `compose.yaml` (comment),
  `DELETE_ME.md` ("add a resource" → register on `/v1`).

Next: implementation worktree off `master`, implement, `root:check` + container
integration suite, PR → squash-merge.
