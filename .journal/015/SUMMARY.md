---
id: 015
title: Swap dev tooling to mise + moon-system + melange/apko, add SLSA L3, prove via a real release
date: 2026-06-27
status: complete
repos_touched: [template-go-api]
related_sessions: ["014", "011", "010", "009", "005"]
---

## Goal
Resume session 014's deferred container-image decision and swap the template's dev
tooling onto three tools the maintainer wants fleet-wide: **mise** (manage all
tools + env + a local task runner, replacing Proto), **moonrepo** (stay the CI gate
but run **system** binaries from mise), and **Railpack** (replace the bespoke
Dockerfile). Keep GoReleaser or find a cross-language alternative. **Preserve or
improve** the supply-chain posture (SBOM, attestation, signing, reproducibility,
SLSA). Started as grounded research; became the full migration + SLSA L3 + a real
release rehearsal.

## Outcome
**Met and then some.** Research overturned two of the three premises, and the
implementation shipped across five merged PRs plus a forced-release shakeout:
- **Railpack was NOT adopted** — research (workflow + agents, all grounded) showed it
  regresses this template's hardened posture: `debian:bookworm-slim` (not
  distroless), QEMU multi-arch (not native cross-compile), no built-in
  SBOM/provenance/cosign, breaks `docker compose up --build`. Chose **melange + apko**
  (signed Wolfi apk → minimal multi-arch nonroot OCI image) instead.
- **GoReleaser kept** — it is already cross-language (Rust/Zig/Bun/Deno/Node/Python
  builders since v2.5) and owns the full supply chain (cosign v3, SBOM, attestations,
  SLSA); no credible cross-language replacement matched it.
- **PR #24** (`7aac1e1`): Proto → mise; moon runs `system` binaries; `mise.lock`
  (`locked=true`) fail-closed integrity replaces `checksum-url` + `sqlc-verify` +
  `.moon/proto/sqlc.sha256`.
- **PR #25** (`4098277`): Dockerfile → melange/apko; keyless cosign signature; syft
  SBOM + provenance attestations; native-runner multi-arch (no QEMU); local stack via
  `mise run stack-up`.
- **PR #26** (`8d5007d`): provenance generated in an **isolated reusable workflow**
  (`.github/workflows/attest.yml`) → SLSA Build L3, while keeping GitHub's attestation
  API (`gh attestation verify`).
- **Forced release to validate** (user asked): the first real tag shook out **3
  distinct pipeline bugs** (see Lessons), each fixed via a `fix(release):` PR (#29,
  #31, #33) and re-run, ending at **v1.0.4 — a fully successful, cryptographically
  verified release** (image published, cosign-verified, SLSA-L3-provenance verified to
  signer `attest.yml`, multi-arch amd64+arm64, 9 draft-release binary assets).

## Key Decisions
- **mise over Proto, moon on `system` toolchain** — omit moon's `go:`/`unstable_*`
  toolchain blocks + `toolchains.default: system`; tasks drop the `proto run … --`
  wrapper. `mise.lock` is a strict upgrade over Proto (`checksum-url` + the hand-rolled
  `sqlc.sha256`): aqua/github backends record per-platform checksum **plus**
  cosign/SLSA/GitHub-attestation verification, enforced fail-closed by `locked=true`.
- **melange/apko over Railpack/ko/buildpacks** — only path that keeps distroless-grade
  minimalism + native multi-arch + integrated supply chain *and* is cross-language for
  the fleet (ko is Go-only; Railpack/buildpacks regress minimalism + multi-arch).
- **Wolfi base floats; reproducibility via SBOM + attestation, NOT `apko.lock.json`** —
  `apko lock` can't pin an image whose app is a per-build `@local` apk (would pin a
  stale checksum), and pinning ca-certs/tzdata fights Wolfi's fresh-by-design model;
  the per-build SBOM + provenance record the exact versions instead. (User chose this.)
- **SLSA L3 via Option B (reusable-workflow `attest*`), not slsa-github-generator** —
  the generator yields `slsa-verifier`-recognized L3 but moves provenance **off**
  GitHub's attestation API; the user explicitly wanted to keep `gh attestation verify`,
  so we isolate `actions/attest*` in `attest.yml` (GitHub-asserted L3) instead.
- **Forced a real `1.0.x` release to validate** — `release.yml`'s tag-only attest/
  publish path is unreachable by the dry-run, so the only real test was a tag; accepted
  burning patch versions to shake it out.

## Changes
- `mise.toml`, `mise.lock` (new); `moon.yml` (system tooling, drop `sqlc-verify`,
  fileGroups → mise pins); `.moon/toolchains.yml`, `docs/moon.yml` → system;
  `ci.yml`/`docs-pages.yml` → `jdx/mise-action`; deleted `.prototools`, `.moon/proto/*`,
  `.nvmrc`. (PR #24)
- `melange.yaml`, `apko.yaml` (new); `release.yml`/`release-dry-run.yml`/
  `security-scan.yml` → melange+apko; `compose.yaml` → prebuilt image + `mise run
  stack-up`/`image-local` tasks; `release-please-config.json` `extra-files`; deleted
  `Dockerfile`, `.dockerignore`, `.go-version` (setup-go → `go.mod`). (PR #25)
- `.github/workflows/attest.yml` (new reusable); `release.yml` provenance moved to
  `attest-binaries`/`attest-image` `uses:` jobs; `ghd.toml` signer_workflow +
  `stage_ghd_release_assets.py` (+ test) + dry-run `expected_signer` → `attest.yml`.
  (PR #26)
- Release fixes: attest-binaries `packages:write` (#29); apko `mkdir -p sbom` (#31);
  GHCR `docker/login-action` in attest.yml (#33).
- Prose: README (Prerequisites, Container Image, Release Layer, CI/Security),
  CONTRIBUTING, DELETE_ME, docs/index across the PRs.

## Open Threads
- **`v1.0.4` GitHub release is a DRAFT** — by design (human publishes after
  inspection). The image is already live on GHCR; the draft awaits manual publish.
- **Residue from the shakeout** (left for the maintainer): tags `v1.0.1–v1.0.3` still
  exist (broken draft releases deleted; tags kept consistent with CHANGELOG, likely
  protected); `CHANGELOG.md` carries `1.0.1–1.0.4` entries; GHCR holds superseded
  `v1.0.2/v1.0.3` image tags (referrer-aware cleanup via `/orgs/meigma/packages/…`).
- **SLSA L3 is GitHub-self-asserted** (reusable-workflow isolation), not a
  `slsa-verifier`-recognized builder ID — the deliberate trade for keeping
  `gh attestation verify`. Watch the slsa-github-generator redesign (rebuilding atop
  GitHub artifact attestations) for future convergence.
- The `attest-sbom` action emits a deprecation warning (use `actions/attest`) — benign,
  worth a future cleanup.

## References
- PRs: #24 `7aac1e1`, #25 `4098277`, #26 `8d5007d` (migration + L3); #29/#31/#33
  (release fixes); #27/#28/#30/#32/#34 (release-please force + 1.0.1→1.0.4).
- Released (draft): `v1.0.4` — `ghcr.io/meigma/template-go-api:v1.0.4`
  (`sha256:6d9162a328cba6c2f5fb0bf01d303cfe46c07814921e78bf26f5cd83a44c294f`).
- Approved plan: `~/.claude/plans/understood-ok-with-all-curious-tide.md`.
- Builds on: `.journal/014/SUMMARY.md` (the deferred image decision this resolved).
- Session log: `.journal/015/NOTES.md`.

## Lessons
- **The release pipeline's tag-only path hid 3 bugs the dry-run can't reach** (dry-run
  never calls `attest.yml` and never pushes): (1) a reusable workflow can't request
  more permissions than its caller grants — the shared `attest.yml` job declares
  `packages: write`, so EVERY caller (incl. the binary one) must grant it or the run
  hits `startup_failure`; (2) `apko publish --sbom-path <dir>` requires the dir to
  pre-exist (`mkdir -p` it); (3) `attest-build-provenance --push-to-registry` runs in
  the reusable workflow's own runner and needs its OWN `docker/login-action` — the
  build job's GHCR login does not cross the reusable-workflow boundary. **A real tag
  (or a throwaway prerelease tag) is the only way to exercise this.**
- **mise**: `lockfile=true` alone does NOT create `mise.lock` (touch it / `mise lock`
  first, then `mise lock --platform <all>`); enforcement key is `settings.locked`; aqua
  backend verifies cosign/SLSA/attestations by default. `mise lock` resolves but does
  not persist the moon `macos-x64` entry (a mise write quirk) — hand-added from moon's
  published checksum; re-running `mise lock` may drop it.
- **apko**: Wolfi has no `nonroot` *package* — create the user via apko `accounts`
  (uid/gid 65532). apko single-arch `build` loads as `<tag>-<arch>`.
- **`.wt/` worktrees nest under the repo**, so mise's config search also loads the main
  checkout's `mise.toml` — `mise trust` both the worktree and the parent.
- Railpack/ko/buildpacks landscape and the GoReleaser-is-cross-language finding are in
  the session-015 NOTES if the fleet revisits image/binary tooling.
