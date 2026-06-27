---
id: 015
title: New session — awaiting goal
started: 2026-06-27
---

## 2026-06-27 12:16 — Kickoff
Goal for the session: not yet stated. Session primed via `/session-new`; the
developer has not yet given a request. Title and scope will be refined on their
first message.

Current state of the world:
- Template is feature-complete on `master` (`3a10e80`); working tree clean.
- All previously-deferred feature seams are built (session 011): API versioning
  (`/v1`), per-IP rate limiting, OpenTelemetry tracing.
- Persistence is PostgreSQL-only (sqlc + pgx + goose); authz is Cedar
  deny-by-default with deferred API-key authn; day-one Docker Compose stack.
- Security review remediated (session 010): API keys hashed at rest, `GET /todos`
  keyset-paginated, sqlc binary integrity-pinned.
- Last session (014) was research-only: assessed moon docker / ko / buildpacks vs
  the hand-rolled `Dockerfile` — **no change made; container-image build strategy
  decision still deferred** (live fork: ko per-language vs buildpacks fleet-wide
  vs shared Dockerfile). See `.journal/014/SUMMARY.md` if that thread resumes.

Plan: await the developer's request, then refine the title/scope and journal
accordingly.

## 2026-06-27 12:20 — Goal stated: research dev-tooling swap
Developer wants to swap the template's dev tooling for a new stack centered on
three tools, building on session 014's deferred container-image decision:
- **mise** — manages ALL tooling (incl. the `moon` and `railpack` CLIs), replaces
  `.envrc`/direnv for env, and becomes the primary local task runner. (Replaces
  Proto pinning.)
- **moonrepo** — stays the primary CI runner / pre-PR gate / GHA entry point, but
  overridden to use **system** binaries so it uses mise-managed tooling.
- **Railpack** — replaces the bespoke `Dockerfile` as the image builder; ideally
  triggered by moon (needs investigation).
- **GoReleaser** — still needed for binary build/publish, but open to a
  cross-language alternative.

Open questions to answer in this research pass:
1. How do moon and Railpack integrate?
2. Do cross-platform (multi-arch) builds still work under Railpack?
3. How does Railpack integrate with the SBOM/supply-chain stack (Cosign, SLSA,
   GitHub attestations)?
4. Cross-language (non-Go) alternative to GoReleaser for producing/publishing
   binaries?

This pass is RESEARCH ONLY — understand the three tools and give preliminary
answers. Launched 4 parallel research agents (mise; moon+integration; Railpack;
GoReleaser alternatives), all instructed to ground claims in current docs (per
the `verify-tooling-support-claims` lesson from session 014). Synthesis to follow.

## 2026-06-27 12:35 — Research findings (4 agents)
Ground truth on current stack (read from source): tooling is **Proto**-pinned
(`.prototools` + `.moon/proto/*.toml`: golangci-lint, goose, mockery, sqlc; moon
tasks call `proto run <tool> -- …`). moon v2 `go` toolchain 1.26.4 + unstable
python/uv for docs. **No `.envrc`/direnv today** (mise's env role is additive).
CI = `moonrepo/setup-toolchain` → `moon ci`. Release bar is HIGH: `release.yml`
does multi-arch via **native arm64 runners** (not QEMU), BuildKit `sbom: true` +
`provenance: mode=max`, GitHub artifact attestations on image + checksums, image
is **distroless static-debian12:nonroot**, reproducible (`-trimpath -buildid=`,
`mod_timestamp`). The bespoke `sqlc-verify` task exists ONLY because sqlc ships no
upstream checksum.

**mise (strong fit).** Manages every tool we use: golangci-lint/mockery/sqlc/
goreleaser via `aqua:` (full checksum + Cosign + SLSA + GH-attestation in
`mise.lock`), goose via `aqua:pressly/goose`, moon via `aqua:moonrepo/moon`
(verify old `@moonrepo/cli` version-naming), railpack via `github:railwayapp/
railpack`. `mise.lock` is **opt-in** (`touch mise.lock` + install, or `mise lock`;
`locked = true` enforces) and records per-platform hashes + provenance for aqua/
github backends — EQUALS or BEATS Proto's `checksum-url` and **makes the bespoke
`sqlc-verify` task obsolete** (sqlc has checksum metadata in aqua-registry). Risk:
asdf/npm/cargo/pipx backends are version-only (no hash) — pin everything to
aqua/github. Env via `[env]`/`mise.local.toml`/`_.file` fully replaces .envrc for
our needs. Task runner (TOML/file tasks, depends, sources/outputs **mtime** not
content-hash, parallel-4) is simpler than moon — intended split: mise = local
runner, moon = CI gate. CI: `jdx/mise-action@v3` (install+cache) adds shims to
`$GITHUB_PATH` so moon finds tools on PATH. GOTCHA: shims inject PATH only, not
`[env]` vars — use `mise exec --` if env needed in CI.

**moon as system runner (clean, mechanical).** Use system binaries by **omitting
the `go:` toolchain block** (cleanest) and/or `MOON_TOOLCHAIN_FORCE_GLOBALS=true`,
and/or per-task `toolchains: system`. moon resolves task `command` from PATH
(v1.18+), so `proto run golangci-lint -- run …` → `golangci-lint run …`, etc.
No native moon↔mise / moon↔railpack integration — purely compositional (Railpack's
own README recommends installing it via mise). moon→Railpack = a task running
`railpack build .` with `cache: false` (image output, no file artifact). moon
docker layer = distraction (confirms session 014). CI swap: drop
`moonrepo/setup-toolchain`, use `jdx/mise-action` + install moon + `moon ci`;
affected-gating unchanged. GOTCHA: moon's task hash does NOT include system binary
versions → stale cache on a tool bump with unchanged source. Idiomatic fix
(already used here for `.prototools`): put `mise.toml`/`mise.lock` in task `inputs`
so a version bump invalidates the hash.

**Railpack (fights this template's posture).** Clears easy bars (Go auto-detect,
CLI-invocable, multi-arch exists) but MISSES three hard requirements: (1) runtime
is **debian:bookworm-slim, not distroless**, no nonroot user by default; (2)
multi-arch is **QEMU sequential per-platform + must push** (3–10× slower than the
current native Go cross-compile; no local multi-arch image); (3) **no built-in
SBOM/provenance/Cosign** — SLSA L3 not OOB; SBOM only via external Syft or
untested BuildKit-frontend passthrough; provenance via the same
`actions/attest-build-provenance` layer we already have. Also: no ldflags
passthrough (VERSION/COMMIT/DATE needs a `--build-cmd` override), **breaks
`docker compose up --build`** (not a docker-build drop-in → keep a thin
Dockerfile), buildkitd is a hard prereq, v0.30 sub-v1.0, only a community GHA
action. It's essentially the "buildpacks family" tradeoff from session 014
(zero-config cross-language, fatter/less-minimal, QEMU multi-arch).

**GoReleaser (keep — and it's already cross-language).** Not Go-locked since
v2.5–v2.9 (2025): OSS builders for Rust/Zig/Bun/Deno/Node/Python(uv,poetry) +
`pre-built` import. Already owns the full supply chain we use (Cosign v3 bundles,
SBOM, GH attestations, SLSA). `dist` (cargo-dist) is Rust-first w/ experimental
generic builds, better INSTALLER UX, but **lacks Cosign + SLSA** and SBOM is
Rust-only; axodotdev is a ~2-person company (continuity risk). No single
cross-language tool matches GoReleaser's integrated supply-chain stack. Verdict:
keep GoReleaser; it IS the cross-language binary tool (validate non-Go builders
under our Cosign/SBOM/attestation config before fleet-wide reuse).

Net read: mise + moon-as-system are clean, high-value swaps (mise's lockfile is a
genuine supply-chain upgrade and deletes `sqlc-verify`). Railpack is the weak link
— it regresses minimalism, multi-arch speed, and integrated SBOM/provenance for
this hardened template; its only real upside is fleet cross-language uniformity
(the same axis as buildpacks in session 014). GoReleaser stays. Decision on the
image builder still owed to the developer; mise+moon could proceed independently.

## 2026-06-27 13:05 — Fleet tangent: is there a `ko` for JS? + apko
Developer asked (cross-fleet context — they maintain non-Go templates too)
whether a `ko`-equivalent exists for JS (Vue/React/Node). Ran one grounded
research agent. Findings:
- **No JS-native ko-equivalent at maturity** — structural, not immaturity: ko
  works because Go → one static binary (`base + binary = image`). JS has no
  default single-artifact output. ko's real sibling is **Jib** (JVM); the "ko
  family" is Go + Java, NO Node member. Google's `nodejs-container-image-builder`
  was archived Sep 2025.
- **Frontend SPA = easy case**: artifact is static files → image is `tiny static
  server + dist/`. No one-shot tool, but only a ~10-line two-stage Dockerfile on
  a shared base: **Chainguard nginx** (apko-built, SBOM) or **static-web-server**
  (4 MB Rust binary).
- **Backend Node**: only "minimal" via compile-to-binary — **Bun** `build
  --compile` (stable, best; ~60–100 MB; glibc→`distroless/base`, not scratch;
  cross-compiles incl. musl), **Deno** `compile` (mature; ~58 MB), **Node SEA**
  (Stability 1.1, pre-stable — don't use yet), **vercel/pkg** (archived Jan 2024),
  **nexe** (unmaintained). All still need a one-line COPY; none is daemonless
  one-shot like ko.
- **apko (Chainguard)** is the cross-fleet thread: explicitly ko-inspired,
  daemonless, bitwise-reproducible, native SBOM, multi-arch — but at the
  BASE-IMAGE layer (declarative YAML, no Dockerfile, no RUN/COPY-from-host). Can't
  inject your app like ko injects a Go binary (would need melange to package the
  app as an apk). Pairs with melange (source→apk) over Wolfi.
- **Fleet takeaway** (reinforces the prior turn): don't chase "ko for every
  language." Anchor consistency at a shared **apko/Chainguard hardened base-image
  factory** + a thin per-language "get the artifact in" step (ko injects for Go;
  COPY onto a shared static-server base for SPAs; Bun-compile+COPY for Node). apko
  is the closest cross-language carrier of ko's *philosophy*. (Cute escape hatch
  for ko literally everywhere: a tiny Go static-server that `go:embed`s the SPA
  `dist/`, ko-built — couples the frontend image to a Go binary.)
