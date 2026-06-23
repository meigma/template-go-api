# Research: Authorization middleware seam for the Go API template

**Date:** 2026-06-23 · **Session:** 005 · **Method:** `deep-research` workflow
(run `wf_9380176c-9e0`) — 5 angles · 20 sources fetched · 97 claims → top 25
3-vote adversarially verified → 24 confirmed / 1 refuted · 102 agents.

> Question (refined): survey + rank the modern (2024–2026) Go ecosystem for an
> **authorization** middleware seam in a hexagonal chi v5 + Huma v2 template.
> Authentication is deliberately **deferred** to the integrator; the template
> provides a loosely-coupled, principal-agnostic authorization decision wired as
> middleware. Constraint: **in-process / embeddable only** (external engines only
> behind a port). Survey all paradigms then rank; include authn-verifier
> reference examples.

---

## Headline recommendation

Define an **authorization-decision port** (an allow/deny interface) and wire it as
**Huma v2 middleware** that consumes an **opaque principal carried via context**.
Default the port to an **embeddable in-process engine**; keep the engine swappable.

Verified ranking by deployment coupling (the template's critical constraint):

1. **cedar-go** (AWS Cedar's official Go library, Apache-2.0) — *recommended default.*
   Pure in-process, zero infra, synchronous local API.
2. **Embedded OPA** (`open-policy-agent/opa/v1/rego`) — *runner-up*, policy-as-code
   in-process, but Go-only and engine updates need re-vendor/redeploy.
3. **OpenFGA** (Zanzibar / ReBAC) — *behind a port only*; embeddable as a Go
   library but that means running the full relationship engine in-process (heavy).

> ⚠️ **Coverage gap (see Open Questions):** Casbin, Cerbos, SpiceDB, Permify,
> Topaz/Aserto, and Oso were in scope but **no claims about them survived
> verification** — the "survey all" inventory is therefore incomplete. Casbin
> (a major embeddable Go authz lib) and Cerbos (has an embeddable Go SDK) are
> notable omissions to fill before locking a design decision.

---

## Inventory & ranking (verified)

### 1. cedar-go — recommended default (confidence: high, 3-0)
- Apache-2.0; official `cedar-policy` org (authored by StrongDM, accepted upstream);
  v1.0.0 GA, actively maintained 2024–2026.
- **Embeddable / in-process**: used by importing the package — no server, no client
  config, no network handshake. Matches the template's zero-infra default.
- API is local + synchronous:
  `Authorize(policies PolicyIterator, entities types.EntityGetter, req Request) (Decision, Diagnostic)`.
  `Request` carries Principal/Action/Resource (`EntityUID`) + `Context` (`Record`)
  = the **PARC** model; `Decision` is allow/deny; `Diagnostic.Reasons` names the
  deciding policy (good for audit/debug).
- Trivially wrappable behind a `Decision` port.
- AWS Verified Permissions is a *separate hosted product* — not required by cedar-go.
- Sources: github.com/cedar-policy/cedar-go · pkg.go.dev · LICENSE (Apache-2.0).

### 2. Embedded OPA — runner-up (confidence: high, 3-0 / one 2-1)
- `github.com/open-policy-agent/opa/v1/rego` (low-level eval) + `v1/sdk` (high-level);
  current as of v1.17.1 (June 2026). The older `opa/sdk` path is **deprecated** in
  favor of `v1/sdk`.
- **In-process** (same OS process, less overhead than REST), policy-as-code in Rego.
- Tradeoff (the only non-unanimous vote, 2-1): Go-library mode is **Go-only**, and
  updating the *engine* requires re-vendor/redeploy of the host service; the REST/
  sidecar deployment decouples OPA upgrades and works for any language. Note: policy
  **data/bundles** can still be reloaded at runtime — the re-vendor cost is the
  engine itself.
- `rego.Eval` satisfies the same `Decision` port shape as cedar-go.
- Sources: openpolicyagent.org/docs/integration · pkg.go.dev/.../opa/v1/rego.

### 3. OpenFGA — ReBAC, behind a port only (confidence: high, 3-0)
- CNCF Incubation (donated Sept 2022); Zanzibar-inspired relationship-based (ReBAC)
  fine-grained authorization.
- **Can** be embedded as a Go library: `pkg/server.NewServerWithOpts` /
  `MustNewServerWithOpts` backed by an in-process `memory.New()` datastore, then
  `Check`/`Write`/`ListObjects` with no separate process.
- **Caveat:** "embeddable" here = the full Zanzibar relationship engine in-process
  (schema + datastore plumbing), much heavier than a policy library; its zero-infra
  story is weaker than cedar-go's. Fits the template only behind a clean port/adapter,
  not as the default. `server.Check` satisfies the same `Decision` port shape.
- Sources: github.com/openfga/openfga · pkg.go.dev/.../openfga/pkg/server · openfga.dev/docs.

---

## Middleware integration (chi v5 + Huma v2) — verified

- **Express authz as Huma-native, router-agnostic middleware:**
  `func(ctx huma.Context, next func(huma.Context))`, registered globally via
  `api.UseMiddleware`, or **per-operation** via `huma.Operation.Middlewares`.
  Execution order: router middleware → `api.Middlewares()` → `op.Middlewares` →
  handler. (Per-operation `Middlewares` is documented as *one option among several*,
  not labeled THE idiomatic one.)
- **Declarative per-operation requirements:** read `ctx.Operation().Security`
  (`[]map[string][]string`, e.g. `{{"myAuth":{"scope1"}}}`) at request time, then
  short-circuit: missing token → `huma.WriteErr(api, ctx, http.StatusUnauthorized, …)`;
  missing scope → `…StatusForbidden`; call `next(ctx)` only on authorized paths.
  **Caveat:** this is a developer-wired convention — the map keys are OpenAPI
  security-scheme names the developer chooses, *not* an auto-enforced authz primitive.
- **Minimal principal model (authn-pluggable):** carry an opaque verified identity/
  claims through context — inject via `huma.WithValue(ctx, key, value)` (or wrap
  `huma.Context` and override `Context()` to return `context.WithValue(...)`), read
  via `ctx.Context().Value(...)`. The Huma maintainer (danielgtaylor, issue #224)
  endorses a shared `huma.Resolver` input struct as the idiomatic read pattern. Keeps
  the principal opaque so any authn scheme can populate it — no rigid user schema.
- Sources: huma.rocks/features/middleware · huma.rocks/how-to/oauth2-jwt ·
  huma discussions #389 · huma issue #224 · openapi.go · pkg.go.dev/huma/v2.

---

## Deferred-authn verifier reference examples (secondary) — verified

Drop-in building blocks the integrator wires to satisfy the authn seam; each hands
verified claims to the authz layer via context:

- **lestrrat-go/jwx** — `jwt.Parse(signed, jwt.WithKey(...))` (verify-on-parse);
  `jwk` package: `Set`, `Fetch`, `NewCache` (auto-refresh), `NewCachedSet`,
  `jwt.WithKeySet` → the JWKS + verify building blocks OIDC needs. Active (v2/v3).
- **coreos/go-oidc** — after `Verify()`, `idToken.Claims(&customStruct)` unmarshals
  the verified ID-token payload into an arbitrary struct = the hand-off point to authz.
  *(Refuted nuance: it makes no authz statement — it supplies verified claims an authz
  layer consumes.)*
- **go-chi/jwtauth** — `Verifier` sets verified token/claims on request context;
  `jwtauth.FromContext(ctx) → (jwt.Token, map[string]interface{}, error)` (v5 current).
- **auth0/go-jwt-middleware** v3 — authentication-only (no scope/role/policy);
  v3.2.0 (May 2026) migrated crypto core from square/go-jose to lestrrat-go/jwx v3.
- Sources: github.com/lestrrat-go/jwx · github.com/coreos/go-oidc ·
  github.com/go-chi/jwtauth · auth0.com/blog/rebuilding-go-jwt-middleware-v3.

---

## Design conclusion (confidence: medium — synthesis, not a single quoted claim)

- Put the authorization **decision behind a Go port** (allow/deny). cedar-go's
  synchronous `Authorize`, OPA's `rego.Eval`, and OpenFGA's `server.Check` all
  satisfy the same port shape → the engine is swappable.
- Default the port to **cedar-go** (zero-infra, in-process); offer embedded OPA and
  OpenFGA (behind the port) as documented alternatives.
- Invoke the port from **one Huma middleware seam** with a **context-carried opaque
  principal** (`huma.WithValue` / `Resolver`) → no principal-schema lock-in, authn
  stays pluggable, testable, morphable.
- Coupling ranking grounded: cedar-go (pure in-process) > embedded OPA (in-process,
  Go-only, redeploy to upgrade) > OpenFGA (embeddable but heavy / usually a service,
  behind a port only).

---

## Caveats (matter for implementation)

1. Huma per-operation `Middlewares` is *one option among several*, not labeled THE
   idiomatic approach — docs are deliberately neutral.
2. Per-operation `Security`-driven scope checking is a developer-wired convention,
   not an auto-enforced primitive; map keys are developer-chosen scheme names.
3. OpenFGA embeddability = the full Zanzibar engine in-process (schema + datastore);
   heavier than a policy lib → behind a port, not the default.
4. OPA re-vendor/redeploy cost applies to the *engine* binary; policy data/bundles
   can still reload at runtime (this sub-claim was the lone 2-1 vote).
5. Sources are overwhelmingly primary (official repos, godoc, vendor docs, maintainer
   comments); the one vendor blog (auth0) was corroborated against repo deps.
6. Versions are mid-2026 current: OPA v1.17.1, go-jwt-middleware v3.2.0,
   cedar-go v1.0.0 GA; `opa/sdk` deprecated → `v1/sdk`.
7. Refuted (1-2): the framing that go-oidc "explicitly does not handle authorization."
   Surviving accurate framing: it supplies verified claims for an authz layer.

---

## Open questions / gaps to close before finalizing the design

1. **Incomplete paradigm survey.** Casbin, Cerbos, SpiceDB (authzed), Permify,
   Topaz/Aserto, and Oso were in scope but produced **no verified claims**. Casbin
   (a well-known in-process Go authz lib) and Cerbos (embeddable Go SDK) especially
   should be inventoried for embeddability / licensing / Go ergonomics before locking
   the ranking. The current top-3 is sound but not exhaustive.
2. **Per-request performance:** measured eval latency / allocations of cedar-go vs
   embedded OPA (`rego.PreparedEvalQuery`) at realistic policy sizes, to confirm the
   per-request bar.
3. **Cedar entity sourcing in hexagonal layout:** how to source/cache Cedar entities
   (the principal/resource graph) per-request without coupling the authz port to a
   specific persistence adapter (memory vs PostgreSQL).
4. **Huma authz hook:** does Huma v2 offer/plan a first-class authorization hook
   beyond manually reading `ctx.Operation().Security`, and a recommended way to map
   declarative operation `Security` entries to a generic `Decision` port instead of
   hard-coding scheme names like `myAuth`?

---

## Primary sources

- cedar-go: https://github.com/cedar-policy/cedar-go · https://pkg.go.dev/github.com/cedar-policy/cedar-go
- OPA: https://www.openpolicyagent.org/docs/integration · https://pkg.go.dev/github.com/open-policy-agent/opa/v1/rego
- OpenFGA: https://github.com/openfga/openfga · https://pkg.go.dev/github.com/openfga/openfga/pkg/server · https://openfga.dev/docs/authorization-concepts
- Huma v2: https://huma.rocks/features/middleware/ · https://huma.rocks/how-to/oauth2-jwt/ · https://github.com/danielgtaylor/huma/discussions/389 · https://github.com/danielgtaylor/huma/issues/224
- Authn verifiers: https://github.com/lestrrat-go/jwx · https://github.com/coreos/go-oidc · https://github.com/go-chi/jwtauth · https://auth0.com/blog/rebuilding-go-jwt-middleware-v3/
- Patterns: https://pkg.go.dev/k8s.io/apiserver/pkg/authorization/authorizer · https://www.cerbos.dev/blog/how-to-implement-authorization-in-go · https://www.calhoun.io/pitfalls-of-context-values-and-how-to-avoid-or-mitigate-them/

---

# UPDATE — Gap-fill pass (2026-06-23): the six uncovered engines

**Method:** second `deep-research` run `wf_0cf09b78-afe` — 5 angles · 23 sources ·
106 claims → top 25 verified → 24 confirmed / 1 refuted · 105 agents. Scoped to the
six engines the first pass missed, ranked relative to cedar-go / OPA / OpenFGA.

## Casbin — VERIFIED (high confidence)
- **True in-process embeddable Go library** — `github.com/casbin/casbin/v2`
  (canonical; v3 is the newer active major under `apache/casbin`). `NewEnforcer(...)`
  + synchronous `Enforce(rvals ...interface{}) (bool, error)`. No server. (An optional
  separate `casbin-server` exists but is distinct.)
- **License: Apache-2.0**, no copyleft / source-available / open-core / paywall — NO
  template red flags. Now an ASF **incubating** (podling) project (SGA/IP transfer in
  progress; doesn't affect the code license).
- **Authoring model:** the PERM metamodel in a `model.conf`
  (`[request_definition]`/`[policy_definition]`/`[policy_effect]`/`[matchers]`,
  + `[role_definition]` for RBAC) separated from policy data (CSV file or DB adapter).
  Supports ACL, RBAC, RBAC-with-domains/tenants, ABAC, RESTful (keyMatch). "Change how
  authz works by editing a config file."
- **Maturity:** v2.135.0 (Dec 9 2025), ~1,806 importers, ~20.2k stars; active v3.x line
  (v3.10.0 Jan 2026). Go is the flagship implementation.
- **Verdict:** the **strongest in-process peer to cedar-go** — above embedded OPA and
  OpenFGA on the embeddability axis. A credible *alternative default*, but does NOT
  displace cedar-go. Choose Casbin over cedar-go only when a **config-file-driven
  PERM/RBAC(-with-domains)/ABAC** model + CSV/DB policy adapters is explicitly
  preferred over Cedar's purpose-built policy language + allow/deny+reasons output.
  Minor cost: variadic `Enforce` and reflection/struct-based ABAC leak the
  principal/resource model into matcher expressions — a small ergonomic tax behind a
  minimal-principal Decision port.

## Cerbos — VERIFIED (high confidence): NOT an in-process Go option
- **No supported in-process Go mode.** Cerbos Embedded (ePDP, WASM) is
  **JavaScript/TypeScript-only** (`@cerbos/embedded-client`/`-server`); official docs
  explicitly route Go to **service PDPs**. (A Go integrator could DIY-load the WASM
  bundle via wazero, but that is unsupported.)
- **Cerbos Embedded additionally requires the commercial Cerbos Hub** to compile/serve
  policy bundles (CDN-delivered, polled). No standalone OSS local-compilation path.
- The official `github.com/cerbos/cerbos-sdk-go` is a **client** to an external/sidecar
  PDP over gRPC/REST (TCP/Unix socket) — not a local evaluator. Repo is Apache-2.0, active.
- Policies authored as YAML (resource policies + derived roles).
- **Verdict:** for Go, Cerbos is **service-only → behind-a-port only** (same bucket as
  OpenFGA-as-a-service). Does **NOT** change cedar-go-as-default; disqualified for the
  zero-infra default by JS-only embedding + the Hub paywall.

## SpiceDB · Permify · Topaz/Aserto · Oso — DIRECTIONAL ONLY (not 3-vote verified this pass)
The verification budget again dropped these four (`budgetDropped: 7`); primary sources
were fetched but no claims survived to the verified top-25. Directional read from those
fetched primary sources (treat as unverified, lower confidence):
- **SpiceDB** (`authzed/spicedb`) — Zanzibar/ReBAC gRPC **server**; embed via Go client,
  not a first-party in-process lib (a community `borkfork/spicedb-embedded` exists).
  → behind-a-port, same class as OpenFGA.
- **Permify** (`Permify/permify`) — Zanzibar/ReBAC **server**. → behind-a-port.
- **Topaz/Aserto** (`aserto-dev/topaz`) — OPA + a Zanzibar-style directory, deployed as a
  container/**sidecar**; Aserto is the commercial hosted product. → behind-a-port.
- **Oso** — the OSS `go-oso` library (Polar language) is **DEPRECATED** in favor of the
  hosted **Oso Cloud** (deprecation notice on osohq.com). → effectively a dead end for a
  copyable template; Oso Cloud is hosted/paywalled.
- One Cerbos sub-claim (WASM bundle excludes policies) was refuted (1-2); not relied upon.

## Consolidated final ranking (in-process embeddability = the template's priority)

**Tier 1 — true in-process embeddable Go policy libraries (zero infra; viable defaults):**
1. **cedar-go** — *recommended default.* Apache-2.0; Cedar policy language (PARC);
   `Authorize → Decision + Diagnostic` (allow/deny + reasons).
2. **Casbin** — *strongest peer / alternative default.* Apache-2.0; PERM metamodel
   config + CSV/DB adapters; `Enforce → (bool, error)`. Pick when config-driven
   RBAC/ABAC is preferred over a policy language.
3. **embedded OPA** — Rego policy-as-code; in-process but Go-only, engine upgrade =
   redeploy (`opa/v1/rego`).

**Tier 2 — embeddable but heavy → behind a port:**
4. **OpenFGA** — ReBAC; embeddable Go lib but runs the full Zanzibar engine in-process.

**Tier 3 — service/sidecar-only for Go → behind a port only:**
- **Cerbos** (Go SDK = client to external PDP; embedded mode JS-only + commercial Hub),
  **SpiceDB**, **Permify**, **Topaz/Aserto** *(last three directional)*.

**Disqualified for a copyable template:**
- **Oso OSS** (`go-oso`) — deprecated in favor of hosted Oso Cloud *(directional)*.

### Bottom line
The real fork for the template default is **cedar-go vs Casbin** — both fully verified,
both Apache-2.0, both pure in-process. Difference is the authoring model: Cedar's
purpose-built policy language (allow/deny + reasons, naturally principal-agnostic) vs
Casbin's PERM config metamodel (flexible models, but ABAC leaks struct/reflection into
matchers). Everything else is behind-a-port (ReBAC services) or disqualified (Oso OSS).

### Still-open (only if formal verification of the four ReBAC/deprecated engines is wanted)
SpiceDB / Permify / Topaz / Oso exact license + embeddability remain *directional, not
3-vote verified*. Also open: Casbin v2-vs-v3 import recommendation; cedar-go entity
sourcing in the hexagonal layout (resource/principal graph per-request without coupling
the authz port to a persistence adapter); per-request eval latency cedar-go vs OPA.

### Gap-fill pass sources
- Casbin: https://github.com/apache/casbin · https://pkg.go.dev/github.com/casbin/casbin/v2 · https://casbin.org/docs/how-it-works/ · https://casbin.org/docs/abac/ · https://casbin.apache.org/ · https://incubator.apache.org/projects/casbin.html
- Cerbos: https://docs.cerbos.dev/cerbos-hub/decision-points-embedded · https://www.cerbos.dev/features-benefits-and-use-cases/wasm-embedded-pdp · https://github.com/cerbos/cerbos-sdk-go
- Directional: https://github.com/authzed/spicedb · https://github.com/Permify/permify · https://github.com/aserto-dev/topaz · https://www.osohq.com/docs/oss/any/getting-started/deprecation.html · https://github.com/osohq/go-oso
