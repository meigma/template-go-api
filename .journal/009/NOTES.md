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

## 2026-06-23 20:18 — Integrator UX review workflow complete
Ran the review as a workflow (user-directed): 3 parallel first-time-integrator
passes (identical prompt, Opus 4.8, high effort) → 1 dedupe/rank synthesis agent.
15 raw findings → 8 ranked. Workflow run `wf_54e1a5ea-2a1`. Themes: stale
upstream-template docs/wiring, and copy-pasteable commands that fail on first run.
All fixes are small text-level edits; no design changes. Returned the ranked list
to the user for triage; awaiting which to act on. Ranked findings:
1. (high, 3x) README:290 + moon.yml:191-192 claim CI workflows are `.disabled` —
   they're all active (6 live workflows). Reword to "CI active; only container
   integration suite excluded via runInCI:false".
2. (med, 2x) Documented `serve`/`docker run` smoke-tests fail instantly —
   `--database-url` is required (config.go:162). CONTRIBUTING.md:45-46 + README
   `docker run` example. Supply the URL or annotate the DB prereq.
3. (med, 2x) "Adding a resource" omits sqlc.yaml second `sql:` block + sqlc-check
   sed (moon.yml:134) + mockery-check dir list (moon.yml:174) edits a 2nd PG
   resource needs; new gen layers silently skipped + escape drift guards.
4. (med, 3x) moon.yml:48 releaseConfig fileGroup globs nonexistent
   `.github/workflows.disabled/**` → release-workflow edits don't invalidate
   root:check. Point at real `.github/workflows/`.
5. (med, 2x) DELETE_ME.md cleanup checklist never mentions the Meigma agent-session
   tooling (.session.md, AGENTS.md/CLAUDE.md, .agents/skills, scaffold/.journal);
   CLAUDE.md hard-fails if .session.md removed. Add a keep/remove checklist item.
6. (low, 1x) .gitignore:6 ignores `.agents/` but `.agents/skills/**` is committed →
   new files there silently untracked. Scope the ignore.
7. (low, 2x) todo authz defines ActionDelete (`todo:delete`) in actions.go:40,48 +
   policy.cedar + README, but no DELETE route in handler.go. Drop it or comment it.
8. (low, 1x) Docs mix `moon run root:<task>` and bare `moon run <task>`
   (README:499, DELETE_ME:92-93). Pick one form.

## 2026-06-23 21:05 — Plan approved; two PRs opened
User had me probe the "needs a Docker-capable runner" rationale (finding 1) — it's
false: ci.yml runs `moon ci` on ubuntu-latest (Docker preinstalled). Real cause the
integration suite is excluded is `runInCI: false` on the test-integration moon task.
User asked for a plan covering all 8 findings + removing that runInCI limitation.
Plan approved (`~/.claude/plans/please-propose-a-plan-joyful-nygaard.md`). Decision:
drop the orphaned ActionDelete (user-confirmed). Split into two PRs off master:
- **PR #11** `test(ci): run the container-backed integration suite in CI` (branch
  `ci/run-integration-tests`): moon.yml runInCI false→true on test-integration,
  reword the false `.disabled` claims (README + moon.yml comment), fix the dead
  `releaseConfig` glob (`.github/workflows.disabled/**` → `.github/workflows/release*.yml`).
  Local: root:check green w/o Docker; `moon task root:test-integration` = Runs in CI: Yes;
  test-integration green vs real postgres (11s).
- **PR #12** `docs: fix first-run commands and document resource/agent wiring` (branch
  `docs/template-onboarding-polish`): findings 2,3,5,6,7,8 — CONTRIBUTING/README first-run
  commands, adding-a-resource sqlc/drift-guard callout, DELETE_ME session-tooling item,
  .gitignore `.agents/` un-ignore, drop ActionDelete (actions.go+policy.cedar+README),
  `root:` prefix standardization. root:check + test-integration green; .agents probe ok.

**CI gotcha found (not yet resolved):** PR #11's `ci` check "passed" in 53s but was a
NO-OP — `moon ci` logs "No tasks affected by changed files." moon ci only runs tasks
whose `inputs` globs are touched; moon.yml/README aren't inputs of any task, so it ran
nothing — including test-integration. PR #12 changes .go files (so moon ci runs
build/lint/test) but is based off master WITHOUT the flip, so test-integration stays
gated there too. Net: the flip is correct + locally proven, but **test-integration has
NOT yet run on a GitHub runner.** Clean proof path: merge #11 → rebase #12 onto new
master → #12's CI (Go files affected + flip now present) exercises test-integration on
ubuntu-latest. READMEs touch disjoint regions so the rebase is conflict-free. Awaiting
user decision on merge order / whether to force-prove now.

## 2026-06-23 21:35 — Both PRs merged; CI enablement proven on the runner
User chose "merge #11, rebase #12". Executed:
- Merged **PR #11** (squash `c9a6bbf`). master fast-forwarded; #11 branch/worktree removed.
- Rebased **PR #12** onto new master (README regions disjoint → conflict-free), force-pushed.
- **Proof obtained:** #12's CI run (commit af268a2 — has the flip in base + .go changes)
  executed `root:test-integration` on ubuntu-latest: `internal/integration ok 13.966s`,
  "Tasks: 15 completed" (vs the 0-task no-op of #11's config-only run). testcontainers
  works on GitHub's hosted runner; the integration suite genuinely runs in CI now.
- **Kusari finding on #12:** flagged literal `app:app` creds in the new docker-run example
  as normalizing hardcoded credentials. Fixed → `<user>:<password>@host.docker.internal:5432/<db>`
  placeholders (the self-contained "Running with PostgreSQL" block keeps real creds it
  creates). Re-scan: Kusari pass; all checks CLEAN.
- Merged **PR #12** (squash `598d130`). master fast-forwarded to `598d130`; branch/worktree
  removed. Only `master` + `journal/jmgilman` worktrees remain.

**Outcome:** all 8 review findings resolved on master; the container-backed integration
suite runs in CI and is proven on ubuntu-latest. Template is in good shape for inheritance.
**Lesson:** `moon ci` runs only tasks whose `inputs` globs are touched — a config/docs-only
change (like the runInCI flip itself) triggers nothing, so enabling a CI task isn't
self-proving; you need a PR that touches the task's inputs to exercise it on the runner.
Session work complete; ready for session-close when the user is.
