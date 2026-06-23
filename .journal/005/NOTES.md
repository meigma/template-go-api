---
id: 005
title: Session 005
started: 2026-06-23
---

## 2026-06-23 12:21 — Kickoff
Goal for the session: not yet stated. Session opened via `session-new`;
awaiting the user's actual request.

Current state of the world:
- `master` is at `18b56e7` — the hexagonal Go API-server template with the
  PostgreSQL persistence tier merged (PR #6). Working tree clean except untracked
  `.claude/` and `.codex-security/`.
- Architecture as built: chi v5 + Huma v2 (transport-scoped, code-first
  OpenAPI 3.0.3); pragmatic ports & adapters; domain `internal/todo`; adapters
  under `internal/adapter/{memory,http,postgres}` (+ `http/middleware`,
  `http/problem`, `http/todoapi`); `internal/{config,observability,logctx,app,
  cli,integration}`; slog + Prometheus `/metrics` on a dedicated listener
  (`--metrics-addr`, default `:9090`); RFC 9457 on every non-Huma surface;
  OpenAPI exported server-less → neoteroi OAD render with a `root:check`
  drift-guard.
- Persistence: `--store=memory|postgres` (memory default); pgx/v5 + sqlc
  (committed + drift-guarded) + goose (embedded, `migrate` subcommand) + a
  testcontainers integration suite under `internal/integration` (build-tagged
  `integration`, run via `moon run root:test-integration`).
- Future-slice seams left open (not built): authn/authz; OTel tracing; rate
  limiting; pagination; API versioning; mockery. Known follow-up: wire
  `test-integration` into CI once the `.disabled` GitHub workflows get a
  Docker-capable runner.
- NOTE: session 004 did all the PostgreSQL work above but was never formally
  closed (no `SUMMARY.md`; still shows `in-progress` in `INDEX.md`). Flagged to
  the user at kickoff of 005.

Plan: wait for the user's request, then scope the work and proceed per
`.session.md`.

## 2026-06-23 12:41 — Goal set: deep research on authorization middleware for the template
User's request: run a deep-research report on the modern (2024–2026) Go
ecosystem for an **authorization** starting point in the template. This feeds
the open "authn/authz" future-slice seam — but reframed: authentication is
deliberately **deferred** to the template user (JWT/Passkey/OIDC/session), and
the template instead provides a loosely-coupled, morphable **authorization
middleware seam** between API endpoints. Explicitly do NOT build from scratch —
discover, inventory, and rank existing Go options. Minimal assumptions about the
principal/identity model.

Refined scope via 3 clarifying questions (user answers):
- Authz model: **survey ALL paradigms then rank** — RBAC/scope, policy-as-code
  (cedar-go, OPA/Rego, Casbin, Oso), relationship-based/Zanzibar (OpenFGA,
  SpiceDB, Permify, Topaz/Aserto).
- Coupling: **in-process / embeddable only** — engine runs in the binary (zero
  infra, mirrors the in-memory store default); external services acceptable only
  behind a port/adapter.
- Authn scope: **include authn-verifier reference examples** (JWT — jwx/
  golang-jwt; OIDC — coreos/go-oidc) as integration-seam references, with
  authorization staying the primary focus.
- Also asked the report to cover chi v5 + Huma v2 middleware integration
  (per-route/per-operation authz) and a minimal principal-via-context model.

Action: launched the `deep-research` workflow — run `wf_9380176c-9e0`, task
`wx0d47prv`. Awaiting the synthesized cited report. This is a DESIGN-input
research step, not yet a build (keep design separate per
`separate-mechanical-from-design-work`).

Pending loose end (not blocking): an uncommitted working-tree edit to
`.journal/INDEX.md` reverts session 004's row back to `in-progress`, contradicting
its committed close (`749ddb1`) + `SUMMARY.md`. Flagged to user; recommended
`git restore .journal/INDEX.md`. Awaiting confirmation before touching it (the
harness flagged the change as intentional).

## 2026-06-23 12:52 — Research complete; report saved
`deep-research` run `wf_9380176c-9e0` finished (5 angles · 20 sources · 97 claims
→ top 25 verified → 24 confirmed / 1 refuted · 102 agents). Full report saved →
`.journal/005/RESEARCH-authz-middleware.md`.

Headline: define an **authz-decision port** (allow/deny) + wire it as **Huma v2
middleware** consuming an **opaque principal carried via context**. Verified
ranking by coupling: (1) **cedar-go** (AWS Cedar official Go lib, Apache-2.0,
pure in-process, synchronous PARC `Authorize` → Decision/Diagnostic) =
recommended default; (2) **embedded OPA** (`opa/v1/rego`, in-process but Go-only,
engine upgrade = redeploy) = runner-up; (3) **OpenFGA** (Zanzibar/ReBAC,
embeddable but heavy) = behind a port only. cedar-go/`rego.Eval`/OpenFGA
`server.Check` all satisfy one port shape → engine swappable. Huma seam:
`func(ctx huma.Context, next …)` via `api.UseMiddleware` or per-op
`Operation.Middlewares`; read `ctx.Operation().Security`; short-circuit 401/403
via `huma.WriteErr`; principal via `huma.WithValue` / `Resolver` (maintainer-
endorsed, issue #224). Authn deferred to drop-in verifiers (jwx, coreos/go-oidc,
go-chi/jwtauth, auth0/go-jwt-middleware v3).

**Coverage gap flagged:** Casbin, Cerbos, SpiceDB, Permify, Topaz/Aserto, Oso were
in scope but produced NO verified claims — the "survey all" inventory is
incomplete. Casbin (popular embeddable Go authz lib) + Cerbos (embeddable Go SDK)
are the notable omissions to fill before locking a design decision.

Next: this is DESIGN-input research, not a build. Awaiting the user's direction on
whether to (a) close the Casbin/Cerbos/etc. gap with a follow-up research pass,
and/or (b) move to collaborative design of the authz port + Huma middleware seam
(keep design separate from mechanical build per
`separate-mechanical-from-design-work`).

## 2026-06-23 13:05 — Gap-fill research complete (the six uncovered engines)
User chose "close the Casbin/Cerbos gap first." Ran 2nd `deep-research` run
`wf_0cf09b78-afe` (5 angles · 23 sources · 106 claims → 24 confirmed / 1 refuted ·
105 agents). Findings appended to `.journal/005/RESEARCH-authz-middleware.md`
("UPDATE — Gap-fill pass").

Verified (high confidence):
- **Casbin** = true in-process embeddable Go lib (`casbin/casbin/v2`,
  `Enforce → (bool,error)`), Apache-2.0 (no red flags, ASF incubating), PERM
  metamodel config (ACL/RBAC/RBAC-domains/ABAC/RESTful), mature (v2.135.0 Dec
  2025, ~1806 importers). = **strongest in-process peer to cedar-go**, above
  OPA/OpenFGA on embeddability; credible alternative default but does NOT displace
  cedar-go. Minor cost: ABAC leaks struct/reflection into matcher config.
- **Cerbos** = NOT a supported in-process Go option. Embedded (WASM) ePDP is
  JS/TS-only AND needs the commercial Cerbos Hub; the Go SDK is a client to an
  external/sidecar PDP. → behind-a-port only; disqualified for the zero-infra
  default. Does NOT change cedar-go-as-default.

Directional only (budget-dropped again; primary sources fetched but not 3-vote
verified): **SpiceDB**/**Permify** = Zanzibar gRPC servers → behind-a-port;
**Topaz/Aserto** = OPA+directory sidecar → behind-a-port; **Oso OSS (`go-oso`)** =
DEPRECATED for hosted Oso Cloud → disqualified.

Consolidated ranking: Tier 1 in-process libs = cedar-go (default) · Casbin (peer)
· embedded OPA; Tier 2 = OpenFGA (behind port); Tier 3 service-only = Cerbos/
SpiceDB/Permify/Topaz; disqualified = Oso OSS. **Real default fork = cedar-go vs
Casbin** (both verified, Apache-2.0, in-process; differ on authoring model).

Next: gap is closed for the decision that matters. Teed up to user: optional 3rd
pass to formally verify the four ReBAC/deprecated engines, vs move to
collaborative design (cedar-go vs Casbin default + the authz port + Huma seam).
Design stays separate from build per `separate-mechanical-from-design-work`.

## 2026-06-23 13:47 — DECISION: commit to Cedar (cedar-go), no portability layer
Collaborated through the cedar-go vs Casbin fork. User chose to be **opinionated**:
ship **Cedar via `cedar-go` as THE authorization engine**, drop the engine-
portability boundary (no neutral Decision interface, no compatibility layer with
other engines) — while keeping hexagonal hygiene. Rationale: the neutral port was
the most expensive / least useful boundary; committing lets us expose Cedar's real
API (typed Request, `Diagnostic` reasons, `.cedar` policy files) instead of a lossy
LCD `bool` port, drops a DTO mapping layer, and improves the test story. Boundaries
KEPT (not engine-portability): authn→authz handoff via opaque principal/claims in
context; `EntityGetter` for entity sourcing (Cedar's own interface — trivial/empty
for the coarse default, repo-backed for fine-grained later); domain (`internal/todo`)
stays Cedar-free; one thin app-owned `internal/authz` package speaks Cedar.

Design notes settled in discussion:
- Resource-level authz (needs the loaded resource) lives in the DOWNSTREAM
  handler/service — NOT day-one; coarse middleware default (principal + claims-as-
  context + route action, no entity graph) ships first. User confirmed this framing.
- The "entity graph" problem (Cedar needs entity attributes + parent/hierarchy
  edges to evaluate `resource.owner == principal` / `principal in Group`) only
  bites for fine-grained rules → documented `EntityGetter` extension point, not the
  default.

De-risking check (subagent, grounded in pkg.go.dev + repo README/releases):
**cedar-go v1.8.0 (2026-06-01)**, single v1 module, official AWS org, ~monthly
cadence. CORE LOOP IS FULLY STABLE (non-`x/`): `NewPolicySetFromBytes`/
`NewPolicyListFromBytes` (+ runtime-mutable `PolicySet`); `cedar.Authorize(policies,
entities, req) (Decision, Diagnostic)` (old `IsAuthorized` deprecated); `types.
EntityGetter`/`types.EntityMap`, `Entity{UID,Parents,Attributes,Tags}`;
`Diagnostic{Reasons[],Errors[]}` (reasons carry deciding `PolicyID`); JSON + all
core value types. GAPS (all advanced, NOT day-one): schema validation 🧪
experimental (`x/exp/schema`); policy templates ❌; full residual partial-eval ❌
(only experimental batch var-substitution); policy formatter ❌. Verdict: safe
commitment for the middleware starting point.

Next: move to COLLABORATIVE DESIGN of the authz seam (capture in a design doc à la
session 004's POSTGRES_TIER.md, then a gated build). First/biggest design fork to
settle = HOW the developer EXPRESSES per-endpoint authorization (the UX) — Huma
`Security` scheme-name convention vs a custom per-operation action/resource
declaration the middleware maps into a Cedar request.

## 2026-06-23 14:43 — Design converged; AUTHZ_TIER.md drafted; PAUSED for review
Collaborated through the full design across several forks; all captured in
`.journal/005/AUTHZ_TIER.md` (journal-only, source of truth, mirrors POSTGRES_TIER.md).
Converged decisions:
- **Modular from day one** (user confirmed): per-slice authz contributions (policies +
  action constants + lazy fact resolver), merged by the composition root into one
  `PolicySet` + one composite `EntityGetter` — mirrors the HTTP registrar seam. Base
  `internal/authz` holds engine plumbing + cross-cutting policies + shared principal
  types; `internal/todo/todoauthz` is the todo slice; `todoapi` consumes its actions.
- **Expression UX:** `authz.Require(action[, idParam])` / `authz.Public()` set Huma
  operation Metadata; ONE global Huma middleware enforces (reconciled from per-op
  middleware → global, to get deny-default). Require also populates OpenAPI `Security`.
- **Deny-by-default** for Huma operations (undeclared → 403 + warn). Infra routes are
  raw chi, outside the Huma authz mw.
- **URL-fed resource identity** (user's idea): middleware sets `Request.Resource =
  Todo::"<id>"` from the path param, no load → identity/principal-based instance authz
  in middleware.
- **Lazy request-scoped fact resolvers** (user's idea): composite `EntityGetter` bound
  to request ctx (claims+repos), loads on demand; Cedar pulls only what policies
  dereference. Two rules from the pull interface (`Get(uid)(Entity,bool)`, no ctx/err):
  bind ctx at construction; capture first load error → 500 fail-closed; cache per
  request (N+1). This pulls the attribute-based case back into middleware as an option
  (double-load caveat; coarse default never triggers it).
- Smaller forks settled (proposed): naming `Action::"todo:create"` + PascalCase types
  + slice-prefixed policy IDs; double-load = none day-one (coarse), cache shipped.

**ONE decision flagged for the user (security stakes): §8C day-one authn default** —
dev authenticator ON by default (best demo, copy-to-prod footgun) vs OFF by default
(safest; out-of-box protected routes 401). Recommended ON-with-guardrails but
explicitly deferred to the user.

cedar-go is a plain `go get` dep (no Proto tooling, unlike sqlc/goose). Build phasing
in doc §12: A base package → B todo slice+wiring → C tests → D docs; branch
`feat/authz-tier`, gated PR. PAUSED for user review of AUTHZ_TIER.md before any build.

## 2026-06-23 14:52 — Review nit: slice package naming
User: `todo/todoauthz` is redundant → use `todo/authz` (cleaner; will align the HTTP
layer's `todoapi`→`todo/http` separately, later). Updated AUTHZ_TIER.md §2: slice dir
is now `internal/todo/authz` (`package authz`). Consequence handled in doc: base engine
is also `package authz`, so the composition root + `todoapi` alias the slice import
(`todoauthz "…/internal/todo/authz"`) — dir stays clean, `todoauthz.X` in examples is
just that alias. Cedar-free-domain rule still holds (dep runs slice→core only). Still
paused for review; §8C (dev-auth default) remains the open user decision.
