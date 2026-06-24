---
id: 009
title: First-time-integrator UX/completeness review before declaring the template inheritable
date: 2026-06-23
status: complete
repos_touched: [template-go-api]
related_sessions: ["005", "006", "007", "008"]
---

## Goal
The template is feature-complete; before declaring it ready to be inherited from
(used as the base for new Go API services), review it through a first-time
integrator's eyes — find the sharp edges that would confuse, mislead, or trip up
an adopter — and fix them. Scope was friction/polish, not design expansion.

## Outcome
Met. A 3-agent first-time-integrator review (run as a Workflow) surfaced 8
sharp/rough edges, all verified against `master`; every one was fixed and merged.
Shipped as two squash-merged PRs:
- **PR #11 `test(ci): run the container-backed integration suite in CI`** (`c9a6bbf`).
- **PR #12 `docs: fix first-run commands and document resource/agent wiring`** (`598d130`).

The headline change — enabling the container-backed integration suite in CI — was
**proven end-to-end on GitHub's `ubuntu-latest` runner** (PR #12's CI executed
`root:test-integration`: `internal/integration ok 13.966s`, 15 tasks). All review
findings are resolved on `master` (`598d130`); both implementation worktrees were
removed and local `master` fast-forwarded.

## Key Decisions
- **The "needs a Docker-capable runner" rationale was false, not just stale (user
  pressed on it).** `ci.yml` runs `moon ci` on `ubuntu-latest`, which ships Docker;
  the suite was excluded only by `runInCI: false` on the `test-integration` moon
  task. So the fix went beyond rewording: flip the flag to actually run it in CI.
- **Keep `test-integration` OUT of the `root:check` aggregate.** It runs in CI as
  its own `runInCI` task, but the local `check` stays hermetic (no Docker) — the
  property the README promises. Flipping the flag, not adding a `check` dep.
- **Drop the orphaned `todo:delete` Cedar action (user-confirmed).** It was
  declared in `actions.go`/`policy.cedar`/README with no DELETE route. Adding a
  route would expand the API (out of scope), so removal restores a clean 1:1
  action↔route mapping in the reference slice.
- **Two PRs, sequenced (CI first), not one.** Cleaner Conventional-Commit history
  (one behavioral `test(ci):`, one `docs:`), and merging #11 first let #12's rebase
  exercise `test-integration` on the runner — turning the normal merge order into
  the validation. READMEs touched disjoint regions, so the rebase was conflict-free.
- **Document the second-resource sqlc/drift-guard wiring rather than generalize it.**
  `sqlc.yaml`'s single `sql:` block and the literal-path `sqlc-check`/`mockery`
  guards are a deliberate simple default; a callout in the "add a resource" guides
  is the right fix, not auto-discovery (scope guardrail held).
- **Placeholder creds in the new docker-run example.** Kusari flagged literal
  `app:app`; the self-contained "Running with PostgreSQL" block keeps real creds
  (it creates that DB), but the bring-your-own-DB example uses `<user>/<password>/<db>`.

## Changes
PR #11 (`c9a6bbf`):
- `moon.yml` — `test-integration` `runInCI: false→true`; reworded its comment;
  fixed the dead `releaseConfig` glob (`.github/workflows.disabled/**` →
  `.github/workflows/release*.yml`).
- `README.md` — replaced the false `.disabled`/follow-up note with the real CI story.

PR #12 (`598d130`):
- `CONTRIBUTING.md` — the bare `serve` smoke test (fails without `--database-url`)
  → Compose stack + a note on the binary path.
- `README.md` — docker-run example given a DB env var + placeholder creds + note;
  "add a resource" step 2 sqlc/drift-guard callout; dropped `ActionDelete` from the
  action list; `moon run openapi` → `root:openapi`.
- `DELETE_ME.md` — second-resource sqlc/drift-guard callout; new step 11 documenting
  the bundled agent-session tooling (keep/remove together); `root:` task prefixes.
- `.gitignore` — stopped ignoring `.agents/` (its `skills/` is committed content,
  so new files were silently untracked).
- `internal/todo/authz/actions.go`, `internal/todo/authz/policy.cedar` — dropped the
  `todo:delete` action.

## Open Threads
- Removing `.agents/` from `.gitignore` surfaces a previously-hidden local skill dir
  (`.agents/skills/codex-security-scan/`) as untracked in the main checkout. It is a
  local tooling artifact, not template content; a future decision is whether to commit
  it, re-scope the ignore, or leave it. Not session-009 work.
- `moon ci` affected-gating: a config/docs-only PR runs no tasks, so enabling a CI
  task is not self-proving (see Lessons). Recorded in `TECH_NOTES.md`.
- Pre-existing future-slice seams unchanged: OTel tracing, rate limiting, pagination,
  API versioning (documented extension points, intentionally not built).

## References
- PR #11: https://github.com/meigma/template-go-api/pull/11 (merged, `c9a6bbf`)
- PR #12: https://github.com/meigma/template-go-api/pull/12 (merged, `598d130`)
- Plan: `~/.claude/plans/please-propose-a-plan-joyful-nygaard.md`
- Review workflow run: `wf_54e1a5ea-2a1` (3 integrator passes → dedupe/rank)
- Session log: `.journal/009/NOTES.md`
- Builds on: `.journal/005/SUMMARY.md` (authz), `.journal/008/SUMMARY.md` (PG-only)

## Lessons
- **`moon ci` only runs tasks whose `inputs` globs are touched by the diff.** PR #11
  changed only `moon.yml`/`README.md` (inputs of no task), so its CI run reported
  "No tasks affected by changed files" and ran nothing — including the very suite it
  enabled. Enabling a CI task is therefore NOT self-proving: you need a PR that
  touches the task's `inputs` (`@group(goSources)` or the migrations) to exercise it
  on the runner. The clean proof was rebasing the docs PR (which edits `.go` files)
  onto the merged flip.
- **Pressing on a stale rationale paid off.** The review framed the `.disabled` note
  as a wording fix; questioning *why* CI supposedly couldn't run Docker exposed that
  the premise was simply false and turned a doc edit into a real CI improvement.
- **A `.gitignore` rule that contradicts committed content is a silent footgun.**
  `.agents/` was ignored while `.agents/skills/**` was committed, so new skill files
  vanished from `git status`. Fixing it also surfaces whatever was hiding there.
