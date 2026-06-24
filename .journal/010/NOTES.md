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

## 2026-06-23 22:11 — Finding 2/3: plaintext API-key storage → SHA-256 at rest (PR #13 open)
Codex Security finding 2 of 3 (Medium, CWE-256/CWE-522): the default API-key
store kept the credential itself in `api_keys.key` and matched by direct equality
(`WHERE key = $1`), so a table/backup dump leaked replayable credentials. Valid —
confirmed against `master` via two Explore agents.

Remediation (plan: `~/.claude/plans/here-is-the-first-logical-hartmanis.md`,
approved): store only a lowercase-hex SHA-256 digest (`key` → `key_hash`), look up
by the hash of the presented key. SHA-256 (no salt) is correct for high-entropy
tokens; it's the exact path the package's own `SECURITY:` comment prescribed.

Key design decisions (some user-chosen via AskUserQuestion):
- **Hash below the port** — hashing lives in `apikey.Store`; the Authenticator
  still passes the raw key, so the `APIKeyStore` interface, its mockery mock, and
  the mock-based unit tests are untouched. (mockery-check confirmed: no drift.)
- **Seed + integration tests hash in SQL** via `encode(sha256($1::bytea),'hex')`
  (Postgres built-in, no pgcrypto). Plaintext dev keys stay readable; the
  integration suite passing PROVES Go-hash == SQL-hash agreement.
- **Edited migration `00002` in place** (user pick) — unreleased table, all DBs
  build fresh; keeps clean inherited history. No `00003` ALTER.
- **No new CLI** (user pick) — documented the `printf '%s' "$KEY" | sha256sum`
  mint one-liner in DELETE_ME/README/docs instead.
- **No `ConstantTimeCompare`** — indexed equality on a preimage-resistant 256-bit
  digest is not a practical timing oracle; rewrote the SECURITY comment to say so
  rather than promise a compare we (correctly) don't do.

Files (8): `00002_create_api_keys.sql`, `apikey/store.go`, `apikey/apikey.go`,
`hack/sql/0002_seed_api_keys.sql`, `integration/apikey_store_test.go`, README,
DELETE_ME, docs/index.

Verified: `root:check` green (no sqlc/mockery/openapi drift — column rename and
below-port hashing disturb neither generated code nor OpenAPI); `test-integration`
green vs postgres:17 (11.5s); compose smoke — dev-user-key→200, dev-admin Bearer→
200, bogus→401, no key→401; `api_keys` holds only 64-char hex digests, plaintext
appears 0× as a stored value.

GOTCHA hit again: `root:lint` first failed on a stale golangci cache pointing at
removed sibling worktrees (`.wt/ci-run-integration-tests`, etc.) — the session-007
lesson. `golangci-lint cache clean` fixed it; re-run green.

Branch `fix/api-key-hashing` (`f866af5`) → PR #13
(https://github.com/meigma/template-go-api/pull/13), `fix(authz): store API keys
as SHA-256 hashes at rest`. CI watching (`gh pr checks 13 --watch`); this PR
touches `.go`+migration so it should actually exercise `test-integration` on the
runner. Next: merge on green, clean up worktree. Findings 1 and 3 still pending
from the user.

## 2026-06-23 22:13 — Finding 2/3 merged (PR #13 `ff55a2e`)
CI green on the runner — confirmed `root:test-integration | ok internal/integration
15.169s` actually executed on `ubuntu-latest` (PR touched `.go`+migration, so
affected-gating ran it; not a self-proving config-only PR). Also pass: `ci`,
GitHub Pages, Kusari Inspector. Squash-merged PR #13 → `master ff55a2e`
(`fix(authz): store API keys as SHA-256 hashes at rest`).

Cleanup done: remote branch deleted, `wt remove`'d the worktree (it reported the
branch "unmerged" — expected for squash-merge — so force-deleted the local branch
`git branch -D`), local `master` fast-forwarded to `ff55a2e`. Invariants OK:
only `master` + `journal/jmgilman` remain; `git ls-files .journal` empty on master.

Finding 2/3 COMPLETE. Findings 1 and 3 not yet shared by the user — session stays
open for them.
