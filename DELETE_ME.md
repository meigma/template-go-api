# Welcome to the Meigma Go API Template

This repository was generated from `template-go-api`, the standard starter for Meigma Go web API services.
It is meant to give new repositories a working baseline on day one: a small Go entrypoint scaffold by default, Moon task orchestration, pinned CI, dependency automation, repository security defaults, and an enabled release pipeline that has already been exercised by the template application.

Delete this file after you finish the first-repository setup checklist below.
It is only here to orient the initial project owner.

## What This Template Provides

- A runnable hexagonal HTTP API server (chi + Huma) at `github.com/meigma/template-go-api`, with a `todo` example resource, RFC 9457 errors, `/healthz`, `/readyz`, and `/metrics`, runtime API docs at `/docs`, and an `openapi` spec-export command.
- Two persistence adapters behind one port: an in-memory store (the zero-infrastructure default) and a PostgreSQL adapter (pgx + sqlc typed queries + goose migrations), selected at runtime with `--store=memory|postgres`. The PostgreSQL path adds a `migrate` subcommand, a committed-and-drift-guarded sqlc layer, a real `/readyz` check, and container-backed integration tests.
- A Cobra/Viper entrypoint under `cmd/template-go-api` and `internal/cli` exposing `serve` (default), `version`, `openapi`, and `migrate`.
- Moon tasks for `format`, `lint`, `build`, `test`, and `check`, plus persistence tasks `sqlc` / `sqlc-check` (regenerate and drift-guard the typed query layer), `migrate` (run database migrations), and `test-integration` (container-backed adapter tests).
- `golangci-lint`, `sqlc`, and `goose` wired through Proto and Moon.
- CI that delegates to `moon ci --summary minimal` with pinned actions, dependency caches, and minimal token permissions.
- A scheduled container vulnerability scan that uploads SARIF results to GitHub code scanning.
- Dependabot coverage for GitHub Actions, Docker base images, Go modules, and the docs uv project.
- MkDocs Material docs under `docs/` that render the exported OpenAPI spec as an API reference (neoteroi OAD), published to GitHub Pages, with a CI drift-guard that fails if the committed spec falls out of sync with the code.
- Repository settings for signed commits, squash-only merges, immutable releases, private vulnerability reporting, and protected tags.
- Release workflows for Release Please, GoReleaser binary assets, GHCR container images, checksums, SBOMs, and GitHub artifact attestations.
- A root `ghd.toml` package manifest so released binaries can be installed with `ghd`.

## How It Works

Moon is the main entrypoint for local development and CI:

```sh
moon run root:check
```

That aggregate check runs the Go formatter/linter/build/tests plus the docs build.
The GitHub Actions CI workflow runs the same path through:

```sh
moon ci --summary minimal
```

The workflow caches Go modules, Go build artifacts, golangci-lint state, and uv's download cache through GitHub Actions. If that is not enough for a larger generated repository, add Moon remote caching later with Depot or another Bazel Remote Execution-compatible backend and repository credentials.

The `GitHub Pages` workflow builds the MkDocs site on pull requests and deploys the default-branch `docs/build` output to Pages. The repository settings manifest defaults Pages to workflow-based publishing with HTTPS enforcement.

The release machinery is intentionally enabled in the template repository so the starter app proves Release Please, GoReleaser binary releases, native-runner container image builds, artifact validation, and attestations before generated projects inherit the setup.
The nominal generated-project path is an HTTP service with both a downloadable binary and a container image. If the new project is binary-only, container-only, or a pure Go library, trim the release files as described below before the first release.

## First Setup Checklist

1. Rename the Go module:

   ```sh
   go mod edit -module github.com/meigma/YOUR_REPO
   ```

2. Choose the project shape.

   Most applications should keep both the binary and container paths. For other shapes:

   - Binary plus container: keep the default layout and update names.
   - Binary only: keep GoReleaser and `ghd.toml`; remove the container release jobs and Dockerfile if the project will not ship images.
   - Container only: keep the Dockerfile and container jobs; remove GoReleaser release assets and `ghd.toml` if users should not install a standalone binary.
   - Library only: remove the CLI, Dockerfile, GoReleaser, `ghd.toml`, and publish workflow pieces. Keep Release Please only if the library should still get changelogs, tags, and draft GitHub releases.

3. For a binary-producing project, rename the binary directory:

   ```sh
   mv cmd/template-go-api cmd/YOUR_BINARY
   ```

   For a library-only project, delete `cmd/template-go-api`, remove or rewrite `internal/cli`, and remove Cobra/Viper dependencies that are no longer used.

4. Replace template placeholders:

   ```sh
   rg "template-go-api|TEMPLATE_GO_API|github.com/meigma/template-go-api"
   ```

   Update Go imports, Moon metadata, README text, docs text, and CLI environment variable prefixes. For release-bearing projects, also update `.goreleaser.yaml`, `release-please-config.json`, `ghd.toml`, `Dockerfile`, and `.github/workflows/release*.yml` as applicable.
   Update `docs/mkdocs.yml` with the generated repository's GitHub Pages URL, usually `https://OWNER.github.io/REPO/`.
   Remember the `TEMPLATE_GO_API_` environment-variable prefix is set in `internal/cli/root.go` (`SetEnvPrefix`); rename it to match the new module.

5. Replace the example resource.

   The `todo` resource is a reference slice that demonstrates the hexagonal seams, not a product feature. To make it your own:

   - Add a domain package under `internal/<resource>` (entity, `Repository` port, and `Service`), mirroring `internal/todo`.
   - Implement the port: start from `internal/adapter/memory` for zero-infra, or mirror `internal/adapter/postgres` (pgx + sqlc + goose) when you need persistence. The README's [Persistence](README.md#persistence) section covers the migration and sqlc-regeneration workflow.
   - Add a transport adapter under `internal/adapter/http/<resource>api` (DTOs, domain mapping, error translation, and a `Register` function), mirroring `internal/adapter/http/todoapi`.
   - Add one `Register` call in `registerResources` in `internal/app/app.go`.
   - When you wire a real datastore, add a readiness check to the `Readiness` slice in `internal/app/app.go` so `/readyz` reflects it (the PostgreSQL adapter shows the pattern with its `Ping` check).
   - Run `moon run openapi` to refresh `docs/docs/openapi.yaml` after changing the API, and `moon run sqlc` (then commit) after changing PostgreSQL migrations or queries; both CI drift-guards fail if the committed output is stale.

   If your project never needs SQL persistence, you can delete `internal/adapter/postgres`, `sqlc.yaml`, the `sqlc`/`sqlc-check`/`migrate`/`test-integration` Moon tasks, the `migrate` subcommand, the `--store`/`--database-url`/`--db-max-conns` config flags, the `.moon/proto/{sqlc,goose}.toml` plugins and their `.prototools` pins, and the pgx/sqlc/goose/testcontainers Go dependencies, then run `go mod tidy`. If your project never uses the in-memory store, drop `internal/adapter/memory` and default `--store` to `postgres` instead.

   Keep the generic transport in `internal/adapter/http` (router, middleware, `/healthz`/`/readyz`/`/metrics`, RFC 9457 fallbacks, the `Registrar` seam), `internal/config`, and `internal/observability` as-is unless you have a reason to change them.

6. Refresh module metadata:

   ```sh
   go mod tidy
   ```

7. Configure releases for the chosen shape.

   For the nominal binary plus container case:

   - Update `.goreleaser.yaml`: `project_name`, build `id`, `main`, binary name, archive name template, and any linked package paths.
   - Update `ghd.toml`: `provenance.signer_workflow`, package name, description, asset patterns, and installed binary path.
   - Update `Dockerfile`: binary path, labels, default `SOURCE`, base-image tags/digests, and runtime command if this is a service instead of a CLI.
   - Update `.github/workflows/release.yml`: `IMAGE_NAME`, binary validation names, container labels, summary commands, and verification examples.
   - Update `.github/workflows/release-dry-run.yml`: binary validation names, local container image name, and smoke-test commands.
   - Update `.github/workflows/security-scan.yml`: local container image name and scan category.
   - Update `.github/repository-settings.toml` only if required status-check names change.

   For binary-only projects:

   - Keep `.goreleaser.yaml`, `ghd.toml`, `Release Please`, `Binary Release Dry Run`, and the binary asset portions of `release.yml`.
   - Remove the `container-image-release` job, container verification summary text, and `Container Image Dry Run`.
   - Remove `Dockerfile`, `.dockerignore`, and `.github/workflows/security-scan.yml` if no container build remains.
   - Remove `Container Image Dry Run` from required branch checks.

   For container-only projects:

   - Keep `Release Please`, `Container Image Dry Run`, `container-image-release`, `Dockerfile`, and `.dockerignore`.
   - Remove `.goreleaser.yaml`, `ghd.toml`, `binary-release-assets`, binary verification summary text, and `Binary Release Dry Run`.
   - Change `container-image-release` so it depends only on `resolve-release`.
   - Remove `Binary Release Dry Run` from required branch checks.

   For library-only projects:

   - Keep Release Please if version tags and changelogs are useful.
   - Delete `.github/workflows/release.yml`, `.github/workflows/release-dry-run.yml`, `.github/workflows/security-scan.yml`, `.goreleaser.yaml`, `ghd.toml`, `Dockerfile`, and `.dockerignore` unless the library publishes some other artifact.
   - Remove release dry-run checks from `.github/repository-settings.toml`.
   - If the library should not create releases at all, delete `.github/workflows/release-please.yml`, `release-please-config.json`, `.release-please-manifest.json`, and `CHANGELOG.md`.

   In every release-bearing project, configure the release app credentials, protected-tag bypass, and repository package permissions before the first release. Run the release dry-run workflow after these edits and before merging the first release PR.

8. Run the full local check:

   ```sh
   moon run root:check
   ```

9. Update project-facing docs:

   - Rewrite `README.md` for the actual project.
   - Review `CONTRIBUTING.md` and `SECURITY.md`.
   - Add a real license before publishing the repository.

10. Delete this file:

   ```sh
   rm DELETE_ME.md
   ```
