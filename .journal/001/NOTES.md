---
id: 001
title: Rename template-go to template-go-api and define target shape
started: 2026-06-22
---

## 2026-06-22 08:25 — Kickoff
Goal for the session:
1. Clean up naming/references so the repo fits its new name `template-go-api`
   (it is currently a verbatim copy of `../template-go`).
2. Write a temporary design document describing the final shape we want this
   template to take: a Go web-based API server template.

Current state of the world:
- Repo is a direct copy of `../template-go` with no changes yet (initial commit
  `a4ca9f0` on `master`).
- Remote: `git@github.com:meigma/template-go-api.git`, default branch `master`.
- Hexagonal architecture is the standing project constraint (see TECH_NOTES.md);
  business logic must stay isolated from external adapters.
- Prefer functional testing before declaring features complete; take an agile,
  prototype-early approach.

Plan (rough):
- Survey the current template to inventory `template-go` naming/references
  (module path, docs, CI, scripts, etc.).
- Rename/replace references to fit `template-go-api`.
- Draft a temporary target-shape document for the Go web API server template.
- Implementation work happens on a worktree branched from `master`, not on the
  journal branch; integrate via GitHub PR (squash merge).

Awaiting the user's go-ahead before starting substantive work.

## 2026-06-22 08:58 — Referencing pass done, PR open
Scope decision: the user split the session — this pass is **referencing only**.
Goal 2 (the temporary target-shape design doc) is deferred to a separate,
collaborative step and was explicitly kept out of the autonomous plan. (Saved as
memory `separate-mechanical-from-design-work`.)

Two confirmed decisions before execution:
- Reset release history (the v0.1.1 history belongs to upstream `template-go`).
- Reframe identity/purpose prose now toward "Go web API server template".

Inventory: three Explore agents found ~80+ refs across ~37 files (module path
`github.com/meigma/template-go`, `cmd/template-go`, binary, `TEMPLATE_GO_*` env
prefix, `ghcr.io/meigma/template-go`, GoReleaser/ghd/Dockerfile/Moon, CI
workflows, Python release-script fixtures, MkDocs config).

Work (on worktree `chore/rename-template-go-api` off `origin/master`):
- Mechanical rename `template-go`→`template-go-api` and `TEMPLATE_GO`→
  `TEMPLATE_GO_API` (single perl pass each over `git grep -l` matches); `git mv`
  the cmd dir; `go mod tidy`.
- Reset `CHANGELOG.md` to a fresh baseline; `.release-please-manifest.json` → 0.0.0.
- Reframed identity prose (README, DELETE_ME, CONTRIBUTING, docs index, Moon
  descriptions, image/package descriptions, root.go `Short`). Left honest
  current-state "starter CLI" descriptions of the existing CLI scaffold intact.

Verified: `go build/vet/test`, `go mod tidy` clean, runtime `--version` + env
prefix, `moon run root:check` (incl. strict docs build), 11/11 `.github/scripts`
Python tests, `goreleaser check`. No stray bare references remain. Note:
`docs/pyproject.toml` + `docs/uv.lock` carried the docs package name
`template-go-docs` → now `template-go-api-docs`; `uv lock` confirmed only the
name line changed.

PR: https://github.com/meigma/template-go-api/pull/3 (squash-merge).

Next: goal 2 — draft the temporary target-shape design doc collaboratively
(framework choice, hexagonal layering, endpoints/config/observability, testing).

## 2026-06-22 09:00 — PR #3 merged
PR #3 squash-merged into `master` (merge commit `3d1edae`). Goal 1 (referencing
pass) is fully landed. Local `master` fast-forwarded and now tracks
`origin/master`; implementation worktree removed; local + remote
`chore/rename-template-go-api` branches deleted. Proceeding to goal 2 next.

## 2026-06-22 10:08 — Framework decision: chi on net/http
Goal 2 step 1 (HTTP framework). Ran a research workflow (6 framework profilers +
synthesis, live 2026 data) over net/http, chi, gin, echo, fiber, gorilla/mux
across maturity, ecosystem, ease of use, DX, performance, net/http compat.

Decision (user): **chi v5 on net/http**, with plain stdlib ServeMux as the
zero-dependency fallback. Rationale: for a hexagonal, thin-adapter template the
dominant axis is net/http compatibility — chi keeps plain `http.Handler` /
`func(http.Handler) http.Handler` so the framework never leaks into core ports
and migration to/from raw ServeMux is near-zero churn; tiny stable core (since
2016) = low inheritance risk. Evidence pointed away from the user's initial Gin
lean (gin.Context couples every handler/middleware to the framework; slow/uneven
cadence + maintainer-bandwidth backlog). Performance is a non-factor (all radix
routers + 1.22 mux are one tier; Fiber's fasthttp edge vanishes behind a DB and
forfeits the net/http ecosystem + HTTP/2). gorilla/mux excluded (dormant).

Open follow-on decisions for the design doc (step-by-step):
- OpenAPI strategy: code-first (Huma, runs on chi/stdlib) vs spec-first
  (oapi-codegen chi target) vs defer. Keep router & OpenAPI as separate decisions.
- Hexagonal layering depth (explicit ports/adapters vs lighter handler→service→store).
- Cross-cutting: config (existing Viper), logging (slog), observability, middleware
  set (request id, recovery, CORS), graceful shutdown.
- Reference endpoint scope (healthz + one example resource vs minimal).

## 2026-06-22 12:05 — Decisions: OpenAPI strategy, docs integration, layering
Goal 2 steps 2–3 settled (verified Huma via context7 + web).

**OpenAPI strategy: Huma (code-first), scoped to the transport layer only.**
- Take: humachi adapter, typed `huma.Register` operations, input/output structs
  with tags, schema validation, RFC 9457 errors, OpenAPI 3.1 generation.
- Leave: humacli (keep existing Cobra/Viper for binary + config), and other
  out-of-scope extras.
- Middleware stays at the chi level (`func(http.Handler) http.Handler`); only
  register at Huma level when it must appear in the spec (e.g. security schemes).
- Spec-completeness convention: API endpoints go through Huma (in the spec);
  only infra routes (`/healthz`, `/metrics`) may be raw chi.
- Tagged structs are transport DTOs in the HTTP adapter, mapped to/from domain
  types; tags do structural validation, business rules stay in the service.
- Accepted risk: Huma bus factor (~1 maintainer, ~4.2k★), contained to the
  transport adapter; migration = rewrite handlers to plain chi, domain untouched.

**Docs integration (Huma ↔ MkDocs):** spec is a build artifact.
- Server-less export: small Go generator (own command or `go run ./tools/...`,
  NOT humacli) builds the api and writes `docs/docs/openapi.yaml` via
  `api.OpenAPI().DowngradeYAML()` (3.0.3 for the renderer; `.YAML()` = 3.1).
- New Moon `docs:openapi` (Go) task feeds the existing `docs:build`
  (`mkdocs build --strict`) → spec can't drift from code.
- Render with **neoteroi OAD** (static, themed, searchable; needs 3.0.3).
  Alt was mkdocs-swagger-ui-tag (interactive, 3.1). Huma's runtime /docs
  (Stoplight Elements) is a separate free surface. Optional drift-guard CI check.

**Hexagonal layering: pragmatic ports & adapters.**
- `internal/<domain>/` owns entities + service (use-case logic) + the outbound
  port interfaces it consumes (`ports.go`, e.g. Repository) — consumer-defined,
  inward dependency arrows.
- `internal/adapter/http/` inbound Huma handlers + DTOs + mapping;
  `internal/adapter/<store>/` outbound, implements the domain's port.
- `internal/config/` (existing Viper); `internal/app/` composition root wires +
  injects deps; `cmd/template-go-api` main parses flags → app.Run().
- Interface only where substitution is genuinely needed; no separate port pkg.

Next: cross-cutting concerns (config, slog logging, observability depth,
middleware set, graceful shutdown), then reference-endpoint scope.

## 2026-06-22 12:51 — Cross-cutting + reference slice decided; TARGET_SHAPE.md v1 written
Goal 2 steps 4–5 settled, completing the decision pass.

- **Cross-cutting (#4):** slog structured logging (injected, JSON, request-scoped
  child w/ request id) + **OpenMetrics `/metrics`** (Prometheus client: HTTP +
  Go runtime metrics). Tracing = opt-in OTel seam only. Viper config (server
  addr, timeouts, shutdown grace, log level/format, CORS). Middleware order:
  request id → recovery → access log → timeout → ClientIP → CORS. Graceful
  shutdown via signal.NotifyContext + server.Shutdown.
- **Reference slice (#5):** full vertical slice with an **in-memory** store
  (Huma op → service → consumer-defined Repository port → in-memory adapter),
  plus `/healthz` + `/readyz`. Zero infra; functional-testable out of the box.
- **Entrypoint:** Cobra root → `serve` (default) / `version` / `openapi`
  (server-less spec dump). Keeps Cobra/Viper.
- **Testing:** functional-first via httptest through the in-memory adapter;
  testcontainers-go reserved for a future real-DB adapter.

Wrote **`.journal/TARGET_SHAPE.md`** (v1, journal-only per user — no PR, product
repo untouched) capturing all of goal 2's decisions + proposed package layout,
request flow, OpenAPI/docs pipeline, deps to add, and out-of-scope seams. Added a
discovery pointer in TECH_NOTES.md. Awaiting user review of the v1 doc.

## 2026-06-22 13:19 — Close
User approved ("LGTM") and closed the session. Both goals met.

Hand-off state:
- Goal 1: PR #3 squash-merged to `master` (commit `3d1edae`); rename worktree
  removed; local `master` fast-forwarded and clean.
- Goal 2: `.journal/TARGET_SHAPE.md` v1 written (journal-only) and approved;
  TECH_NOTES.md points to it.
- No open implementation branches; no journal contamination on `master`.

Next session: implement the API-server template per TARGET_SHAPE.md (see
SUMMARY.md → Open Threads). PR #3: https://github.com/meigma/template-go-api/pull/3
