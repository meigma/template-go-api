---
id: 014
title: Evaluate adopting moon's Docker integration vs. the hand-rolled Dockerfile
started: 2026-06-27
---

## 2026-06-27 11:20 — Kickoff
Goal for the session: not yet stated. Session primed via `/session-new`; awaiting
the developer's first request before scoping a title and plan.

Current state of the world:
- The `template-go-api` reference template is **built and finalized**. Sessions
  001–011 are all complete and merged to `master` (tip `5d120e2` at session 011
  close; local `master` shows `3a10e80` after later release/CI PRs #21–#23).
- All previously-deferred feature seams are built: chi v5 + Huma v2 (code-first
  OpenAPI), per-domain ports & adapters under `internal/`, PostgreSQL-only
  persistence (sqlc + pgx + goose), Cedar authz with deferred API-key authn,
  Docker Compose day-one stack, API versioning (`/v1`), per-IP rate limiting, and
  opt-in OTel tracing. See `.journal/TECH_NOTES.md` for the authoritative map.
- Two earlier sessions are dangling `in-progress` in `INDEX.md` with no
  `SUMMARY.md`: **012** ("Basic finalization") and **013** ("New session —
  awaiting goal"). Both were primed but never given a stated goal or closed.
  Worth flagging to the developer; may want to close/abandon them.

Plan: wait for the developer's actual request, then refine the title and scope.

## 2026-06-27 11:30 — Goal set + moon Docker research (no changes made)
Goal (developer's words): research moon's Docker compatibility layer, with the
intent to update the template to "take advantage of that rather than rolling our
own" Dockerfile. Research only this turn — no changes.

What moon's Docker layer is (grounded in moonrepo.dev docs + local `moon 2.3.5`):
- Four CLI primitives: `moon docker scaffold <ids>` (copy a minimal repo skeleton
  — `.moon/docker/configs` + `.moon/docker/sources` — for layer caching),
  `moon docker setup` (install toolchain + deps from the skeleton), `moon docker
  prune` (drop extraneous deps / production-only), and `moon docker file <project>`
  (generate a multi-stage Dockerfile wiring those three). `.moon/docker/` is
  gitignored already.
- Designed primarily for **monorepos with dependency installation** (node_modules-
  style). The scaffold/prune value is avoiding per-project manifest copying at scale.

Ran `moon docker file root --defaults --build-task build` into a THROWAWAY file
(`.moon-sample.Dockerfile`), inspected it, deleted it, confirmed tree clean. The
generated Dockerfile for THIS repo:
- base = `golang:1.26.4` (full Debian image) + moon installed via
  `curl -fsSL https://moonrepo.dev/install/moon.sh | bash` (unpinned curl-to-bash).
- skeleton: `COPY . .` then `moon docker scaffold root`.
- build: copy configs → `moon docker setup` → copy sources → `moon run root:build`
  → `moon docker prune`. **No start/runtime stage at all** (repo has no serve/start
  task), so no minimal final image is produced.

Key finding — adopting the DEFAULT generation REGRESSES the current image. The
hand-rolled `Dockerfile` currently provides, and moon's default does not:
- distroless `static-debian12:nonroot` final image (digest-pinned) vs. leaving the
  artifact in the ~800MB golang image;
- multi-arch cross-compile (`--platform=$BUILDPLATFORM`, TARGETOS/TARGETARCH,
  CGO_ENABLED=0, -trimpath, -buildvcs=false, static binary);
- version stamping via build-args VERSION/COMMIT/DATE → ldflags (release.yml passes
  these; moon's `go build` task does not consume them);
- digest-pinned base images + no network curl-to-bash (supply-chain posture);
- a `test` stage.
Also: workspace sets `pipeline.installDependencies: false`, so `moon docker setup`'s
dependency-install value is explicitly off here; and `root` is a single project, so
scaffold's selective-copy value is ~nil (the project source IS the whole repo).

Blast radius if the Dockerfile changes: `release.yml` + `release-dry-run.yml`
(buildx matrix linux/amd64+arm64, build-args→ldflags, push-by-digest, manifest
assembly, per-platform smoke tests `--version` / `openapi | grep openapi: 3.0.3`),
`security-scan.yml`, `compose.yaml` (build context + migrate/api share the image),
README/CONTRIBUTING/DELETE_ME/docs prose.

Realistic adoption options to discuss with the developer (NOT yet decided):
1. Don't adopt — the layer targets multi-project dep-install monorepos; little fit
   for a single static-binary Go service, and default gen regresses hardening.
2. Adopt the `--template` path (moon v2 `moon docker file --template`): moon-native
   generation but with a custom template that keeps distroless + ldflags + multi-arch.
3. Use scaffold/setup/prune as primitives inside a still-hand-written multi-stage
   Dockerfile (keeps the minimal runtime + stamping; gains moon toolchain consistency).
Next: present findings, then collaborate on which direction (design decision).

