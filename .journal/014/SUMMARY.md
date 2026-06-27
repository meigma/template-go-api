---
id: 014
title: Research container-image build strategy — moon docker layer vs ko vs buildpacks vs the hand-rolled Dockerfile
date: 2026-06-27
status: complete
repos_touched: []
related_sessions: ["006", "011"]
---

## Goal
Research-only session (no code changes). Evaluate replacing the template's
hand-rolled multi-stage `Dockerfile` with a higher-level tool, assessing three
candidates in turn against the customized-Dockerfile status quo: (1) moon's Docker
compatibility layer, (2) `ko`, (3) Cloud Native Buildpacks. The cross-cutting
driver that emerged: the developer maintains OTHER non-Go templates and would
prefer one image strategy across the fleet.

## Outcome
Met (as research). All three alternatives were assessed and grounded in current
docs + the local CLI; findings are captured in `NOTES.md` and synthesized below.
**No implementation decision was made and no template files were changed** — the
hand-rolled `Dockerfile` remains the status quo. The strategic fork (fleet
uniformity vs. per-language best-fit vs. controlled cross-language) was surfaced to
the developer, who chose to close the session before deciding. Only journal entries
were written/committed (on `journal/jmgilman`); `master` is untouched and clean.

## Key Decisions
- **No change committed — research deferred to a future session.** The decision is
  strategic (fleet-wide), not a quick mechanical swap; the developer closed before
  choosing. Recorded so the next session starts from the synthesis, not from zero.
- **moon Docker layer judged the weakest fit.** It targets multi-project monorepos
  with dependency installation; for a single static-binary Go service its
  default generation (`moon docker file root`) *regresses* the current image — no
  minimal runtime stage, base `golang:1.26.4` + `curl|bash` moon install, no
  distroless, no multi-arch cross-compile, no ldflags stamping. `installDependencies:
  false` and a single `root` project gut its scaffold/setup value here.
- **ko judged the best fit for THIS service, but Go-only.** Every runtime asset is
  `go:embed`'d (migrations + both Cedar policies), so the image is literally just the
  binary — ko's exact sweet spot. `.goreleaser.yaml` already carries the build
  settings ko reuses, and GoReleaser's native `kos:` block makes adoption ≈ one config
  block. Preserves/improves every Dockerfile property (native multi-arch, auto SPDX
  SBOM, reproducible) and shrinks `release.yml`. Disqualifier for fleet use: Go only.
- **Buildpacks judged the only cross-language option, but with real costs.** One
  Dockerfile-free strategy for Go + non-Go, native SBOM (CycloneDX), and `pack rebase`
  for fleet CVE patching. Costs: larger/less-minimal run image (Paketo tiny is
  jammy-based, not distroless-static), multi-arch friction (CNB can't cross-build app
  images — needs native-arch runners + `pack manifest`, or QEMU), heaviest CI (>1GB
  builder), and the least build transparency — the sharpest clash with this template's
  "every line pinned & auditable" posture.

## Changes
- **No repository changes.** No edits to `Dockerfile`, `compose.yaml`, `.goreleaser.yaml`,
  `release.yml`, or any source — this was assessment only.
- Journal only (on `journal/jmgilman`): `.journal/014/NOTES.md` (kickoff + four research
  checkpoints + one correction), `.journal/INDEX.md` row, and the TECH_NOTES pointer
  added at close.

## Open Threads
- **The strategic decision is unmade.** Three live directions: (A) per-language
  best-fit — `ko` for Go templates (smallest image, effortless multi-arch, native
  `kos:` integration) + a separate choice for non-Go; (B) buildpacks everywhere for
  one uniform Dockerfile-free system (accepting larger images / multi-arch friction /
  less transparency); (C) a shared/parameterized Dockerfile reused across templates
  (full control + minimalism, hand-maintained).
- **If ko (option A) is chosen**, the two concrete sub-decisions are: integrate via
  GoReleaser `kos:` (cleanest — reuses `builds`, likely no separate ko binary to pin)
  vs. standalone `ko` in `release.yml`; and how to keep the day-one `docker compose up
  --build` UX (ko isn't a `docker build` backend — keep a thin Dockerfile for local,
  use `ko build --local`, or a moon task). The compose stack was a deliberate session-006
  feature, so this must be designed around.
- **Blast radius for any Dockerfile replacement** (for whoever implements): `release.yml`
  + `release-dry-run.yml` (buildx matrix linux/amd64+arm64, build-args→ldflags,
  push-by-digest, `imagetools create` manifest, per-platform smoke tests `--version` /
  `openapi | grep openapi: 3.0.3`), `security-scan.yml`, `compose.yaml`, and
  README/CONTRIBUTING/DELETE_ME/docs prose.

## References
- moon: https://moonrepo.dev/docs/guides/docker, https://moonrepo.dev/docs/commands/docker/file
- ko: https://ko.build/configuration/, https://goreleaser.com/customization/ko/,
  https://www.chainguard.dev/unchained/automatic-sboms-with-ko
- buildpacks: https://paketo.io/docs/howto/go/,
  https://buildpacks.io/docs/for-app-developers/how-to/special-cases/build-for-arm/
- GoReleaser buildpacks history: shipped in 0.179.0 (2021), removed in 2022 —
  https://github.com/goreleaser/goreleaser/issues/2976
- Session log: `.journal/014/NOTES.md`
- Builds on: `.journal/006/SUMMARY.md` (the Docker Compose day-one stack that any
  image-build change must preserve), `.journal/011/SUMMARY.md` (the last finalization).

## Lessons
- **Don't assert a tool's capabilities from memory — verify.** I flatly claimed
  "GoReleaser has no buildpacks integration"; the developer corrected me with the
  0.179.0 (2021) release note. The accurate story: GoReleaser *added* a native
  `buildpacks` block in 2021 and *removed* it in 2022 (issue #2976) — for the exact
  reasons (rebuilds the binary + no ARM) that are themselves cons of buildpacks. Ground
  capability/support claims (especially negatives) before stating them. Saved as the
  `verify-tooling-support-claims` auto-memory.
- **ko vs buildpacks is decided by scope, not by this repo.** For a single Go service
  ko wins on nearly every axis; buildpacks only earns its place at the fleet level
  (cross-language uniformity + rebase). The maintainer ecosystem reflects this:
  GoReleaser kept ko, dropped buildpacks.
