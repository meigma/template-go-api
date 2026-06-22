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
