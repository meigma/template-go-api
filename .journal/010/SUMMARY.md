---
id: 010
title: Address independent security review findings (Codex)
date: 2026-06-24
status: complete
repos_touched: [template-go-api]
related_sessions: ["005", "008", "009"]
---

## Goal
Remediate the three findings from an independent Codex Security review of the
template, triaging each for validity/scope before fixing, and ship via the
standard worktree → PR → squash-merge flow.

## Outcome
Met. All three findings were verified against `master`, remediated, and merged as
three separate squash PRs (each CI-green on `ubuntu-latest`, worktrees removed,
`master` fast-forwarded to `f2c5210`):

- **#2/3 — Plaintext API-key storage** (Medium, CWE-256/522) → **PR #13 `ff55a2e`**
  `fix(authz): store API keys as SHA-256 hashes at rest`.
- **#3/3 — Unbounded `GET /todos`** (Medium, CWE-400/770) → **PR #14 `879e2be`**
  `feat(todo): paginate the list endpoint with keyset cursors`.
- **#1/3 — sqlc binary downloaded without integrity check** (Medium, CWE-494) →
  **PR #15 `f2c5210`** `build(sqlc): verify the pinned sqlc binary against a
  committed checksum`.

The findings were addressed in the order received (2, 3, then 1 — the panel
numbered them oddly). Each fix was verified beyond the unit gate: `root:check`
green with no sqlc/mockery/openapi drift on all three; the container suite ran on
the runner for #2/#3; #1 was proven by a tamper negative-test and confirmed
running on the linux runner.

## Key Decisions
- **#2 hash below the `APIKeyStore` port** (not in the Authenticator): the port,
  its mockery mock, and all mock-based unit tests stay unchanged; only the Store
  hashes. Seed + integration fixtures hash in SQL via
  `encode(sha256($1::bytea),'hex')` so the plaintext dev keys stay readable and
  the suite passing *proves* Go-side and SQL-side hashing agree.
- **#2 no `ConstantTimeCompare`** (the package comment had promised it): a
  `WHERE key_hash = $1` indexed equality on a preimage-resistant 256-bit digest
  is not a practical timing oracle, and there is no in-process secret compare —
  the comment was rewritten to state this rather than cargo-cult a compare.
- **#3 keyset (cursor) pagination over offset/cap** (user choice): stable under
  concurrent inserts, O(log n) at depth — the pattern a reference template should
  teach. Default 20 / max 100 as **Go constants** (user choice); the max doubles
  as a static Huma `maximum` tag (edge 422) *and* the service clamps (so a direct
  non-HTTP caller is bounded too). The opaque cursor is transport-only; the port
  speaks `PageQuery`/`PageResult`; the +1 over-fetch lives in the adapters.
- **#3 edited the todos migration in place** to add the `(created_at, id)` index
  (unreleased template; same precedent as #2's `00002` edit).
- **#1 repo-pinned binary hash + `sqlc-verify` guard, sqlc-only** (user choice):
  sqlc publishes no upstream checksum/signature, so `checksum-url` is impossible;
  Proto can't reference a local checksum file and its lockfile is unstable +
  unconfirmed for a no-checksum tool. The guard verifies the resolved binary
  against `.moon/proto/sqlc.sha256` *before* execution. The other 3 Proto plugins
  already verify via `checksum-url`, so they were left alone. Rejected `go install`
  (breaks Proto uniformity, needs a C toolchain for sqlc's cgo) and `unstable-lockfile`
  (unfit for an inherited template).
- **#1 trust anchor**: each pinned digest was produced by verifying the release
  archive against GitHub's published per-asset `digest` before extracting/hashing
  the binary — not blind trust-on-first-use.

## Changes
- **PR #13** (`ff55a2e`): migration `00002_create_api_keys.sql` (`key`→`key_hash`);
  `internal/authz/apikey/{store.go,apikey.go}` (hashKey + lookup-by-digest + doc);
  `hack/sql/0002_seed_api_keys.sql` + `internal/integration/apikey_store_test.go`
  (SQL-side sha256); README/DELETE_ME/docs prose.
- **PR #14** (`879e2be`): new `internal/todo/page.go` (`Cursor`/`PageQuery`/
  `PageResult`/consts/`ErrInvalidCursor`); `ports.go`/`service.go` (PageQuery +
  clamp); `internal/todo/postgres/{queries/todos.sql,repository.go}` (keyset +
  over-fetch) + `(created_at,id)` index in `00001_create_todos.sql`; new
  `internal/todo/httpapi/cursor.go` + `dto.go`/`handler.go`/`errors.go`;
  `todotest/repository.go` fake; regenerated sqlc/mockery/openapi; tests at all
  layers; README/docs/DELETE_ME prose.
- **PR #15** (`f2c5210`): new `.moon/proto/sqlc.sha256`; `moon.yml` `sqlc-verify`
  task + deps wiring + `check` aggregate; README/DELETE_ME prose.

## Open Threads
- Pre-existing future-slice seams unchanged: OTel tracing, **rate limiting**
  (mentioned by finding #3 but a separate cross-cutting concern — left as a
  documented seam), API versioning.
- Session **005** remains `in-progress`/empty in INDEX (pre-existing, untouched).
- The other three Proto plugins verify against *unsigned* upstream checksum files;
  pinning them against repo-committed hashes (defense-in-depth) was scoped out of
  #1 per the user.
- The main checkout still shows untracked local tooling dirs (`.agents/skills/
  codex-security-scan/`, `.codex-security/`, `.claude`) — local artifacts, not
  template content; disposition still open (carried from session 009).

## References
- PR #13: https://github.com/meigma/template-go-api/pull/13 (merged, `ff55a2e`)
- PR #14: https://github.com/meigma/template-go-api/pull/14 (merged, `879e2be`)
- PR #15: https://github.com/meigma/template-go-api/pull/15 (merged, `f2c5210`)
- Plans: `~/.claude/plans/here-is-the-first-logical-hartmanis.md` (overwritten per finding)
- Session log: `.journal/010/NOTES.md`
- Builds on: `.journal/005/SUMMARY.md` (authz/apikey), `.journal/008/SUMMARY.md`
  (postgres-only/mockery), `.journal/009/SUMMARY.md` (CI affected-gating)

## Lessons
- **sqlc ships no checksum/signature** (verified via `gh`): a Proto plugin can't
  verify it via `checksum-url` like its siblings. GitHub's release API exposes a
  per-asset `digest` (sha256 of the *archive*) — the usable trust anchor when a
  project publishes no checksums file. `proto bin <tool>` installs-on-demand and
  prints the path *without executing* the tool, which is what makes a
  verify-before-execute guard possible.
- **`root:format` is `golangci-lint fmt`, not gofmt** — it aligns struct tags
  *within* the backticks, so `gofmt -w` doesn't satisfy it; run
  `proto run golangci-lint -- fmt --config .golangci.yml` (no `--diff`) to fix.
- **The stale-golangci-cache gotcha (session 007) still bites across worktrees**:
  `root:lint` fails referencing removed sibling `.wt/` paths; `golangci-lint cache
  clean` before `root:check` clears it. Hit on two of the three PRs.
- **The Compose stack seeds 3 demo todos** (ids `1111…/2222…/3333…`), not just the
  api_keys — a smoke test that counts rows must account for them (a "24 ≠ 21"
  scare during #3 was the seeds, not a pagination bug).
- **`moon ci` affected-gating ran `sqlc-verify` on a no-Go PR** because the task's
  declared input (`.moon/proto/sqlc.sha256`) changed — so a config/tooling task
  *is* self-proving when its own inputs are in the diff (contrast session 009's
  finding that a docs-only PR triggered nothing).
