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
