# Session Journal

| ID  | Date       | Title | Status | Summary |
|-----|------------|-------|--------|---------|
| 001 | 2026-06-22 | Rename template-go to template-go-api and define target shape | complete | Renamed all references to `template-go-api` (PR #3, merged) and wrote `.journal/TARGET_SHAPE.md` capturing the agreed Go web API server template design. |
| 002 | 2026-06-22 | Implement API-server template per TARGET_SHAPE.md | complete | Shipped slice 1 of the template (todo vertical slice: chi + Huma, ports & adapters, in-memory store, observability, RFC 9457) — PR #4 merged. |
| 003 | 2026-06-22 | Finish API-server template — slice 2 / deferred follow-ups | complete | Completed the template: docs render pipeline + drift-guard, CORS, safe client-IP, request-scoped logging, named readiness, dedicated metrics listener, docs refresh — PR #5 merged. |
| 004 | 2026-06-22 | PostgreSQL persistence tier (research → build) | complete | Researched modern Go+PostgreSQL data access, then designed and shipped the sqlc+pgx+goose+testcontainers persistence tier behind the repository port via a gated multi-agent workflow — PR #6 merged (`18b56e7`). |
| 005 | 2026-06-23 | Authorization tier (Cedar middleware + deferred API-key authn) | complete | Researched the Go authz ecosystem, committed to Cedar (`cedar-go`) behind a modular per-resource deny-by-default middleware seam with deferred API-key authn, built via a 4-phase gated workflow — PR #10 merged (`13a1fe5`). |
| 006 | 2026-06-23 | Docker Compose day-one stack (API + PostgreSQL) | complete | Shipped `compose.yaml` (postgres → migrate → seed → api DAG) with a drop-in `hack/sql/` seed hook over an ephemeral Postgres — PR #7 merged (`8b68bd4`). |
| 007 | 2026-06-23 | Restructure internal/ to couple each domain's code | complete | Moved the todo adapters under `internal/todo/{httpapi,memory,postgres}` (shared transport/DB infra stays under `internal/adapter/`) as a behavior-preserving reorg — PR #8 merged (`1f1e5a7`). |
| 008 | 2026-06-23 | Remove the memory adapter, ship PostgreSQL-only | complete | Dropped the in-memory tier and `--store` toggle (PostgreSQL-only, `--database-url` required) and adopted mockery for repository test doubles — PR #9 merged (`8a46286`). |
| 009 | 2026-06-23 | UX/completeness review before declaring the template inheritable | complete | A 3-agent first-time-integrator review found 8 sharp edges; all fixed and merged (PRs #11 + #12 `598d130`), and the container-backed integration suite now runs in CI, proven on `ubuntu-latest`. |
