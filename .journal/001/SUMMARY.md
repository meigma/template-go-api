---
id: 001
title: Rename template-go to template-go-api and define target shape
date: 2026-06-22
status: complete
repos_touched: [template-go-api]
related_sessions: []
---

## Goal
Two things: (1) re-reference the repo (a verbatim copy of `template-go`) to its
new name `template-go-api`, and (2) define a temporary document describing the
final shape we want the template to take — a Go web/HTTP API server template.

## Outcome
Both met.
1. Referencing pass landed on `master` via squash-merged PR #3.
2. A temporary target-shape design doc was written at `.journal/TARGET_SHAPE.md`
   (journal-only, by the user's choice — no PR, product repo untouched), capturing
   all of the architecture decisions made collaboratively this session. It is a v1
   the user reviewed and approved ("LGTM") before close. No API code was written;
   that is future work guided by the doc.

## Key Decisions
- Scope split: the rename was done autonomously; the design doc was treated as a
  separate, direction-requiring step done collaboratively (not bundled into the
  autonomous plan). User preference — see memory `separate-mechanical-from-design-work`.
- Reset release history (CHANGELOG.md cleared, manifest -> 0.0.0) rather than
  rewriting old-repo URLs, since the v0.1.1 history belongs to upstream `template-go`.
- Reframed identity prose toward "Go web API server template" but left honest
  current-state "starter CLI" descriptions intact (no fabricated API features).
- HTTP layer: chi v5 on net/http -> 100% `http.Handler` types keep transport
  adapters thin/portable; lowest inheritance risk for a template. (Diverged from
  the user's initial Gin lean; research showed gin.Context couples handlers to the
  framework, working against the hexagonal goal.)
- OpenAPI: Huma v2 code-first, scoped to the transport layer only (no humacli);
  spec exported server-less at build time -> MkDocs via neoteroi OAD.
- Layering: pragmatic ports & adapters (consumer-defined ports, inward arrows,
  composition-root wiring).
- Cross-cutting: slog + OpenMetrics `/metrics`; tracing as an opt-in OTel seam.
- Reference slice: full vertical slice with an in-memory store + `/healthz` + `/readyz`.

## Changes
- PR #3 (merged, squash, commit `3d1edae`) — rename across 29 files: module path,
  `cmd/template-go` -> `cmd/template-go-api`, binary, `TEMPLATE_GO_API_*` env prefix,
  `ghcr.io/meigma/template-go-api` image, GoReleaser/ghd/Dockerfile/Moon/golangci
  config, three CI workflows, Python release-script fixtures, MkDocs config; docs
  package -> `template-go-api-docs`; CHANGELOG/manifest reset; identity prose reframed.
- `.journal/TARGET_SHAPE.md` — new temporary design doc (journal branch only).
- `.journal/TECH_NOTES.md` — added a discovery pointer to TARGET_SHAPE.md + the
  session's headline decisions.

## Open Threads
- Implement the API-server template per TARGET_SHAPE.md (future session): entrypoint
  transformation (Cobra `serve`/`version`/`openapi`), domain/adapter/app/observability
  packages, Huma+chi wiring, in-memory reference slice, OpenAPI->MkDocs pipeline,
  dependency additions (chi, huma, prometheus client, neoteroi-mkdocs).
- Out-of-scope seams left open: authn/authz, real persistence (Postgres adapter +
  testcontainers), OTel tracing exporter, rate limiting, pagination, API versioning.
- Once code lands, update README/DELETE_ME with concrete API-server usage.

## References
- PR #3: https://github.com/meigma/template-go-api/pull/3 (merged)
- Design doc: `.journal/TARGET_SHAPE.md`
- Memory: `separate-mechanical-from-design-work` (scope mechanical vs design work)
