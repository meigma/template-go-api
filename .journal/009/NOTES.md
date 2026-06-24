---
id: 009
title: UX/completeness review before declaring the template inheritable
started: 2026-06-23
---

## 2026-06-23 20:09 — Kickoff
Goal for the session: the template is approaching "complete." Before declaring it
ready to be inherited from (used as the base for new Go API services), run a
UX/completeness review — does the template read well, onboard cleanly, and hold
together for a developer adopting it? Focus is review/polish, not a new feature
tier. Substantive work scope to be defined with the user.

Current state of the world:
- `master` at `13a1fe5` (PR #10, Cedar authz tier + deferred API-key authn).
- Template is feature-built across slices 1–2 plus persistence (PG-only), Docker
  Compose day-one stack, per-domain `internal/` layout, mockery test doubles,
  and a Cedar deny-by-default authz tier with deferred API-key authn.
- Last sessions (006/007/008) were structural/cleanup: compose stack, per-domain
  restructure, drop the memory adapter (PostgreSQL-only). Authz landed as 005.
- Open carried threads: wire `test-integration` into CI (workflows `.disabled`,
  need a Docker-capable runner); future-slice seams (OTel tracing, rate limiting,
  pagination, API versioning) left as documented extension points.
- Working tree on `master` clean except untracked `.claude/` and `.codex-security/`.

Plan: rough — orient on the template as an adopter would (README, DELETE_ME,
quickstart, docs, layout, naming, flags, errors), inventory rough edges and gaps
against "ready to inherit," then collaborate with the user on what's in scope
before doing substantive work. Await the user's actual review direction.
