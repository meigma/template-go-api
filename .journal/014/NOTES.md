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

## 2026-06-27 11:45 — ko assessment (research only, no changes)
Developer asked for the same-style assessment of `ko` (ko.build) vs a customized
Dockerfile. ko = purpose-built Go→OCI image builder, NO Dockerfile; native cross-
compile (no QEMU), distroless base, multi-arch, auto SBOM, reproducible.

Decisive FIT signals for this repo:
- Pure Go, `CGO_ENABLED=0`, single static binary, and ALL runtime assets are
  `go:embed`'d — migrations (`internal/adapter/postgres/migrations.go`), Cedar
  policies (`internal/authz/authz.go` base.cedar, `internal/todo/authz/
  contribution.go` policy.cedar). `--authz-policy-dir` is an OPTIONAL override
  (default empty = embedded). So the image needs ONLY the binary — ko's exact
  sweet spot; no `kodata` needed.
- `.goreleaser.yaml` already declares the precise build settings ko reuses:
  `CGO_ENABLED=0`, `-trimpath`, ldflags (`-s -w -buildid=` + version/commit/date),
  `mod_timestamp`, goos darwin+linux, goarch amd64+arm64, plus binary SBOMs.
- GoReleaser has native `kos:` integration that REUSES the `builds` block (ldflags/
  flags/env) and uses ko as a LIBRARY (no separate ko binary to pin — verify). So
  adoption ≈ add a `kos:` block; supports multi-arch + SPDX SBOM + labels/annotations.

Grounded facts (ko.build docs + web): default base `cgr.dev/chainguard/static`
(distroless, no shell/glibc, SLSA-L3, SBOM'd), pin by digest via `defaultBaseImage`/
`baseImageOverrides`; multi-arch via `defaultPlatforms`/`KO_DEFAULTPLATFORMS`; SBOM
SPDX default pushed to registry; cosign-friendly. Current Dockerfile base is
`gcr.io/distroless/static-debian12:nonroot` (parallel).

ko PRESERVES or IMPROVES every current-Dockerfile property: distroless nonroot,
multi-arch (native cross-compile, faster than buildx+QEMU), ldflags stamping (via
goreleaser builds), digest-pinned base, OCI labels, + adds auto per-image SBOM and
reproducible images. It SIMPLIFIES release.yml (replaces the buildx matrix +
push-by-digest + `imagetools create` manifest assembly + per-platform smoke loop
with one ko/goreleaser build).

Honest tradeoffs / frictions:
1. Day-one compose: `compose.yaml` uses `build: context:.` (Dockerfile) for
   migrate+api. ko is registry/`--local`-oriented, not a `docker build` backend, so
   `docker compose up --build` needs rework — keep a thin Dockerfile for local, OR
   `ko build --local` → reference `ko.local/...` in compose, OR a moon/make task.
   (Session 006's day-one stack was a deliberate feature — design around this.)
2. Static-only: no shell/apt/CGO. The goal here, but a boundary for forkers needing
   system libs — document it.
3. ko becomes a pinned tool IF used standalone; it publishes checksums + cosign sigs
   (verifies cleanly, better than sqlc). Via goreleaser `kos` it's a library → likely
   nothing extra to pin (verify).
4. Minor: Dockerfile `test` stage drops (moon `root:test` + goreleaser before-hook
   already run tests — redundant). OCI-label parity via `kos.labels`.
5. Values tradeoff: a Dockerfile is explicit/auditable/pedagogical + language-agnostic;
   ko is less code + Go-native but more "magic" / Go-only.

Assessment: ko is a SUBSTANTIALLY better fit than moon's Docker layer, and arguably
better than a customized Dockerfile on most axes for THIS service (the canonical ko
use case). Two real design decisions: (A) compose/local UX, (B) integrate via
goreleaser `kos:` (cleanest) vs standalone `ko` in release.yml. Still research-only;
no changes made.

## 2026-06-27 12:05 — Cloud Native Buildpacks assessment (research only)
Developer's driver: ko is Go-only; they maintain OTHER non-Go templates and want
ONE image strategy across the fleet. So buildpacks (CNB / `pack` / Paketo) evaluated
mainly as a LANGUAGE-AGNOSTIC strategy, not just for this repo.

Grounded facts (paketo.io Go howto + buildpacks.io multi-arch search):
- Paketo Go: ldflags via `BP_GO_BUILD_LDFLAGS` (`-X main.version=...`), flags via
  `BP_GO_BUILD_FLAGS` (e.g. `-trimpath`), entrypoints via `BP_GO_TARGETS`; config
  through `project.toml` `[[io.buildpacks.build.env]]`. Builds with Paketo Tiny
  Builder → tiny run image (distroless-like: no shell, nonroot cnb user). SBOM
  native = CycloneDX (`sbom.cdx.json`, also SPDX/Syft), `pack build --sbom-output-dir`.
- Multi-arch is the WEAK spot vs ko: CNB cannot cross-build app images (ARM-on-AMD)
  natively — needs native-arch runners + `pack manifest` to assemble the index, OR
  QEMU emulation (perf penalty). heroku/builder:24 ships multi-arch Go/Node/Java/etc.
  This re-introduces the buildx-matrix complexity ko removed — and is arguably worse
  than the CURRENT Dockerfile, which Go-natively cross-compiles via
  `--platform=$BUILDPLATFORM` + GOOS/GOARCH (no QEMU). The Go buildpack compiles
  INSIDE the target-arch image, so it can't cross-compile that way.
- `pack rebase`: swap the run/OS base layer for CVE patches WITHOUT rebuilding the
  app — a genuine FLEET supply-chain win. Reproducible builds supported.
- No GoReleaser integration (it has native `ko`, not `pack`), so buildpacks would NOT
  reuse the goreleaser `builds` block — invoke `pack` separately in release.yml.

Fit/tradeoffs for THIS repo + fleet:
- FOR (fleet): one Dockerfile-free, language-agnostic build for Go AND non-Go
  templates; native SBOM; rebase for fleet patching; per-language config via BP_*/
  project.toml. This is the ONE thing ko can't do and a Dockerfile only does by
  hand-rolling N files.
- AGAINST (this service): tiny run image is Ubuntu(jammy)-based, larger & less
  minimal than ko/distroless-static "just the binary"; multi-arch friction (native
  runners or QEMU + manifest); heaviest CI (pull a >1GB builder, slower detect+build);
  LEAST transparent/auditable (build logic lives in the opaque third-party builder)
  — the sharpest clash with this template's "every line pinned & auditable" posture;
  no goreleaser reuse.

Synthesis across all four options (this concludes the research arc):
- moon docker layer: weakest fit (monorepo dep-install tool; default gen regresses).
- ko: best IMAGE + least effort for THIS Go service; Go-only (no fleet reach).
- buildpacks: best FLEET uniformity (Go + non-Go, SBOM, rebase); least minimal/
  transparent, multi-arch friction, heaviest CI.
- customized Dockerfile (status quo): full control + minimalism + native Go
  cross-compile; language-agnostic but hand-maintained per template.
Strategic fork = fleet uniformity (buildpacks) vs per-language best-fit (ko for Go +
something for others) vs controlled cross-language (shared/parameterized Dockerfile).
Next: surface the strategic decision to the developer. Still no changes made.



