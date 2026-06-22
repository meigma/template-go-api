---
id: 002
title: Implement API-server template per TARGET_SHAPE.md
started: 2026-06-22
---

## 2026-06-22 13:24 — Kickoff
Goal for the session: implement the Go web API server template per the agreed
design in `.journal/TARGET_SHAPE.md` (written and approved in session 001).

Current state of the world:
- Repo is a verbatim copy of the generic `template-go` Cobra/Viper CLI scaffold,
  re-referenced to `template-go-api` (PR #3, merged, commit `3d1edae` on `master`).
  No API code exists yet.
- `.journal/TARGET_SHAPE.md` is the approved v1 plan. Headline decisions: chi v5
  on net/http; Huma v2 code-first OpenAPI scoped to the transport layer; pragmatic
  ports & adapters; slog + OpenMetrics `/metrics`; full in-memory reference slice
  with `/healthz` + `/readyz`; Cobra root exposing `serve`/`version`/`openapi`;
  OpenAPI spec exported server-less → MkDocs neoteroi OAD.
- Guiding constraints (TECH_NOTES.md): hexagonal, functional-test-first, agile.

Open threads from session 001 (the implementation work):
- Entrypoint transformation (Cobra `serve`/`version`/`openapi`).
- domain/adapter/app/observability packages.
- Huma + chi wiring, in-memory reference slice.
- OpenAPI → MkDocs pipeline.
- Dependency additions (chi, huma, prometheus client, neoteroi-mkdocs).

Plan: awaiting the user's specific direction on where to start this session.
Will scope mechanical work into a plan and collaborate on design separately
(per memory `separate-mechanical-from-design-work`).
