---
id: 016
title: Author mise, melange, and apko tooling skills
date: 2026-06-27
status: complete
repos_touched: [template-go-api]
related_sessions: ["015", "014"]
---

## Goal
Add three focused agent skills under `.agents/skills` — `mise`, `melange`, `apko` —
that spend minimal time introducing the tools and maximal time on how each is used
**in this repo**, reinforce disciplined usage (the headline rule: mise owns the
tooling lifecycle and nothing else), and embed accurate command/flag reference
material so agents don't hallucinate. Mirror the existing `git`/`worktrunk` skill
shape (`SKILL.md` + `references/<tool>-commands.md`).

## Outcome
**Met.** Three skills (6 files, ~1069 lines) shipped via PR #35 (`e61e926`),
squash-merged to `master`; CI green (the `.agents/**`-only diff is a moon
affected-gating near no-op; CodeQL/Pages/Kusari all passed). The implementation
worktree `docs/tooling-skills` and its branch are removed; local `master` is
fast-forwarded. Each skill leads with repo wiring + non-obvious operations, states
the tool's "lane" as numbered rules, and ships a version-stamped command reference
grounded in the pinned CLIs (mise 2026.6.14, melange v0.54.0, apko v1.2.19).

## Key Decisions
- **Ground everything in primary sources, then adversarially verify.** Captured
  verbatim `--help` for the *pinned* tool versions (not memory/web) plus the repo
  config as the authoring substrate; a workflow authored the three skills in
  parallel from a shared brief, then a per-skill verifier re-checked every command
  and flag against live `--help` and the actual config. The verify pass corrected
  real errors in the authoring brief — proof the guard earned its place (see Lessons).
- **Skills are task-specific, not always-required → NOT added to `SKILLS.md`.**
  `.journal/SKILLS.md` lists only the always-load skills (`git`, `worktrunk`);
  `mise`/`melange`/`apko` are surveyed-and-loaded per task, so adding them there
  would have been wrong.
- **One PR, docs-only, via the normal worktree→PR→squash flow.** No code or config
  touched; the skills are pure documentation under `.agents/skills/**`.

## Changes
- `.agents/skills/mise/SKILL.md` + `references/mise-commands.md` — mise as the single
  source of truth for pinned tool versions + integrity; the verifying-backend rule,
  `locked`/`mise.lock` semantics, bump/add workflows, `.wt/` trust gotcha, and the
  lane boundary (mise = tooling lifecycle; moon = the task gate).
- `.agents/skills/melange/SKILL.md` + `references/melange-commands.md` — source →
  signed Wolfi apk via `go/build`; `--vars-file` metadata, ephemeral per-arch keys,
  the apko keyring handoff.
- `.agents/skills/apko/SKILL.md` + `references/apko-commands.md` — apk + Wolfi base →
  minimal multi-arch nonroot OCI image; `@local` apk wiring, the `--sbom-path`
  pre-exist and single-arch `<tag>-<arch>` retag gotchas, why `apko lock` is unused.

## Open Threads
- None for this work. Pre-existing session-015 threads remain (the `v1.0.4` GitHub
  release is still a draft awaiting manual publish; residual `v1.0.1–v1.0.3` tags +
  CHANGELOG entries + superseded GHCR image tags; the `attest-sbom` deprecation
  warning). Journal hygiene: sessions 012 and 013 are still `in-progress` rows in
  `INDEX.md` (never formally closed) — untouched here.

## References
- PR: https://github.com/meigma/template-go-api/pull/35 (`e61e926`).
- Builds on: `.journal/015/SUMMARY.md` (the mise + melange/apko + SLSA-L3 migration
  these skills document) and `.journal/014/SUMMARY.md` (the container-strategy research).
- Session log: `.journal/016/NOTES.md`. Authoring brief + captured `--help` lived in
  the session scratchpad (ephemeral, not committed).

## Lessons
- **Adversarial verification against primary sources catches author *and* briefer
  errors.** The authoring brief I wrote asserted three things the verify pass
  disproved against the real `mise.lock` / `mise install --help`: (1) `locked = true`
  gates on pre-resolved **URL** presence, not checksum; (2) `aqua:sqlc-dev/sqlc`
  carries a `url` but **no checksum** on any platform (sqlc publishes none — the very
  reason the old `.moon/proto/sqlc.sha256` existed); (3) the `mise.lock` `provenance`
  field is recorded on only a **subset** of tools (`github-attestations` on
  uv/golangci-lint/python, `cosign` on cosign; the rest carry none) — not the broad
  set the brief claimed. The agents correctly followed the primary source over the
  brief. Takeaway: when authoring reference material, the verifier must check against
  the tool/config itself, never against the brief that seeded the work.
- **Capture `--help` from the pinned binary, not from memory or the web.** Flag sets,
  defaults, and value/toggle typing differ across versions; verbatim `--help` for the
  exact pinned version (via `mise exec -- <tool> … --help`) is the only reliable
  anti-hallucination substrate for a committed command reference.
