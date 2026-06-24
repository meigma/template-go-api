---
id: 010
title: Address independent security review findings
started: 2026-06-23
---

## 2026-06-23 21:47 — Kickoff
Goal for the session: address security review findings raised by an independent
review of the template. The specific findings have not been shared yet — the user
will provide them next.

Current state of the world: the API-server template is feature-complete and
declared inheritable (session 009). Security-relevant surfaces already built:
- **Authorization tier** (PR #10 `13a1fe5`): AWS Cedar via `cedar-go` behind a
  global deny-by-default Huma middleware; modular per-resource authz slices
  (`internal/<domain>/authz`) merged at the composition root; shared principal
  types + cross-cutting policies in `internal/authz/base.cedar`.
- **Deferred API-key authentication**: `Authenticator` seam; shipped impl
  `internal/authz/apikey` (X-API-Key/Bearer → `APIKeyStore` port → postgres
  `api_keys` table). Explicitly a replaceable placeholder, NOT production authn.
  Dev mock keys seed via `hack/sql/0002_seed_api_keys.sql` (dev-only, never a
  migration).
- **Persistence**: pgx v5 + sqlc (parameterized queries) + goose migrations
  (never auto-run on serve); `--database-url` required.
- **Transport**: chi v5 + Huma v2; RFC 9457 problem responses; CORS; safe
  client-IP handling; dedicated metrics listener (`:9090`).
- CI runs the container-backed integration suite on `ubuntu-latest` (PR #11/#12).

Plan: wait for the user to share the review findings, then triage them
(severity, validity, scope), agree on which to fix and how, and address them —
likely via one or more PRs off `master`, following the worktree/PR flow. Will
favor verifying each finding against `master` before acting (findings can be
stale or false positives).
