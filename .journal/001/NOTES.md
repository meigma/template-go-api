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
