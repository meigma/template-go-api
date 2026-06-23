# Authorization Tier — Design Doc (temporary, journal-only)

**Status:** DRAFT for review · **Session:** 005 · **Date:** 2026-06-23
**Role:** Source of truth for the authorization-tier implementation, mirroring
`.journal/004/POSTGRES_TIER.md`'s role for the Postgres tier. Journal-only; the
product repo is untouched until the gated build runs.

**Provenance:** Two `deep-research` passes (`.journal/005/RESEARCH-authz-middleware.md`)
+ a cedar-go capability check. Engine decision and architecture settled
collaboratively with the user across session 005.

---

## 1. Decision summary (locked)

1. **Engine: AWS Cedar via `github.com/cedar-policy/cedar-go`**, shipped as **the**
   authorization engine. **No engine-portability layer** — we expose Cedar's real
   types behind one thin app-owned package rather than a lossy vendor-neutral
   `Decision` interface. (Rationale: the neutral port was the most expensive and
   least useful boundary; committing lets us surface `Diagnostic` reasons, the typed
   `Request`, and real `.cedar` policies, and drops a DTO-mapping layer.)
2. **Authentication is deferred to the integrator** (JWT/OIDC/session/passkey). The
   template ships the *seam* + a clearly-marked **dev authenticator** + documented
   JWT/OIDC reference verifiers — never production authn.
3. **Modular, per-slice authorization** (vertical-slice pattern, mirrors the HTTP
   `Registrar` seam). Each domain ships an *authz slice* contributing its policies,
   action identifiers, and entity ("fact") resolvers. The composition root merges
   all contributions into one runtime `PolicySet` + one composite `EntityGetter`.
4. **Expression UX:** per-operation declaration (`authz.Require(action[, idParam])`
   / `authz.Public()`) recorded as Huma operation metadata, enforced by **one global
   Huma middleware**; the declaration also populates the operation's OpenAPI
   `Security` so protection is visible in the generated docs.
5. **Deny-by-default** for Huma operations: an operation with no authz declaration
   is denied (fail-closed) and logged.
6. **URL-fed resource identity:** the middleware builds `Request.Resource =
   Todo::"<id>"` straight from the path param — no load — enabling identity- and
   principal-based instance authorization at middleware time.
7. **Lazy, request-scoped fact resolvers:** the composite `EntityGetter` is assembled
   per request, bound to the request context (claims + repositories), and loads
   entities **on demand** — Cedar only pulls the entities the applicable policies
   actually dereference.
8. **Hexagonal boundaries kept** (these are *not* engine-portability boundaries):
   the domain (`internal/todo`) stays Cedar-free; the authn→authz handoff is an
   opaque principal carried in `context`; `EntityGetter` decouples entity sourcing
   from the decision.

---

## 2. Package layout

```
internal/
  todo/                         # pure domain — NO cedar import (unchanged)
  authz/                        # base/engine package (resource-agnostic)
    authz.go                    #   Authorizer (wraps PolicySet); Authorize(ctx, Request)
    contribution.go             #   type Contribution { Policies; Actions; Resolver } + collection
    middleware.go               #   Huma global middleware: authn-principal -> request -> decision
    declare.go                  #   Require(action[, idParam]) / Public() -> operation metadata
    principal.go                #   Principal (opaque): EntityUID + claims Record; context get/put
    authn.go                    #   Authenticator seam (interface) + dev authenticator
    getter.go                   #   request-scoped composite EntityGetter (lazy, error-capturing)
    base.cedar        (embed)   #   cross-cutting policies (e.g. admin override)
  todo/todoauthz/               # the todo authz slice (imports todo + cedar; domain core does not)
    policy.cedar      (embed)   #   todo-specific policies
    actions.go                  #   ActionCreate = Action::"todo:create", ActionRead, ...
    facts.go                    #   todo.Todo -> cedar.Entity (attributes + parents); the resolver
    contribution.go             #   Contribution() consumed by the composition root
  adapter/http/todoapi/         # consumes todoauthz.Action* when registering routes (unchanged shape)
  app/                          # composition root: collects []authz.Contribution, wires Authorizer
  config/                       # new flags (see §9)
```

`cedar-go` is a plain Go module dependency (`go get`), not a Proto-managed tool —
no `.moon/proto` or `.prototools` changes (contrast the sqlc/goose tier).

---

## 3. The contribution model (modular authoring, unified evaluation)

Each authz slice exposes a `Contribution`:

```go
// internal/authz
type Contribution struct {
    Policies []byte            // embedded .cedar source for this slice
    Actions  []types.EntityUID // declared actions (for validation/discovery)
    Resolver ResolverFactory   // builds this slice's entity resolver, bound to a request
}

type ResolverFactory func(ctx context.Context, p Principal) EntityResolver
type EntityResolver interface { // narrower than cedar's EntityGetter; composed into one
    Resolve(uid types.EntityUID) (types.Entity, bool)
    Types() []string           // entity type names this resolver owns (for routing)
}
```

The composition root:

```go
contribs := []authz.Contribution{ todoauthz.Contribution(todoRepo) /*, ... */ }
authorizer, err := authz.New(contribs)   // merges policies into one PolicySet (slice-prefixed IDs)
```

**This is the same move as collecting HTTP registrars.** Adding a resource = adding
its slice (domain + transport + persistence + authz); deleting one is surgical
(DELETE_ME stays clean).

**Modular authoring, unified evaluation:** at runtime there is *one* merged
`PolicySet` over *one* shared entity space. Consequences, handled by convention:
- **Namespacing** (see §8A): action IDs `Action::"<resource>:<verb>"`; entity types
  PascalCase (`Todo`, `User`, `Group`); policy IDs slice-prefixed for uniqueness.
- **Cross-cutting rules + shared principal types** (`User`/`Group`/`Org`) live in the
  base `authz` package (`base.cedar`), not in any single slice.
- **Reuse is real and intentional:** because evaluation is unified, a policy authored
  in one slice may reference shared principal groups or another slice's entities
  (e.g. `resource in Project::"x"`). This is the upside of the shared namespace.

---

## 4. Per-request flow

Middleware order (all Huma-level via `api.UseMiddleware`, after the existing chi
stack of request-id → recover → access-log → timeout → CORS → client-IP):

1. **authn middleware** — runs the configured `Authenticator` (dev or real). On
   success, builds an opaque `authz.Principal` (`EntityUID` + claims `Record`) and
   stores it via `huma.WithValue`. On no/invalid credentials, stores "anonymous"
   (does **not** reject here — let authz decide; public ops still work).
2. **authz middleware** — for the matched operation:
   a. read the declaration from `ctx.Operation().Metadata` (`Require`/`Public`/none).
   b. `Public()` → `next`. None → **deny 403** (fail-closed) + warn-log.
   c. `Require(action, idParam)` → build `cedar.Request{ Principal, Action: action,
      Resource: <type or Type::"<id from idParam>">, Context: claims }`.
   d. construct the **request-scoped composite `EntityGetter`** (each slice's
      `ResolverFactory` bound to `ctx`+`Principal`).
   e. `dec, diag := cedar.Authorize(policySet, getter, req)`.
   f. check the getter's captured error → if set, **500 fail-closed** (RFC 9457).
   g. `dec == Allow` → `next`; else **403** with `diag.Reasons` in the log (problem
      detail kept generic for the client).
   h. no principal + deny → **401** instead of 403.

Rejections reuse the existing `internal/adapter/http/problem` RFC 9457 writer.

---

## 5. Expression UX (what a developer writes)

At route registration in the slice's `todoapi` registrar:

```go
huma.Register(api, huma.Operation{
    OperationID: "get-todo",
    Method:      http.MethodGet,
    Path:        "/api/todo/{todoID}",
    Metadata:    authz.Require(todoauthz.ActionRead, "todoID"), // item: binds id -> Resource
    // Security is populated by Require(...) for OpenAPI visibility
}, handler)

huma.Register(api, huma.Operation{
    OperationID: "list-todos",
    Metadata:    authz.Require(todoauthz.ActionList),           // collection: type-level resource
}, handler)

huma.Register(api, huma.Operation{
    OperationID: "healthcheck-ish-public-op",
    Metadata:    authz.Public(),                                // explicit opt-out
}, handler)
```

`authz.Require`/`authz.Public` return the `map[string]any` for `Operation.Metadata`
(and `Require` also sets the OpenAPI `Security` requirement). The single global
middleware enforces — so **forgetting a declaration fails closed**, not open.

---

## 6. URL-fed resource identity & the lazy getter

- **Identity, free, no load:** `Require(action, "todoID")` tells the middleware to set
  `Request.Resource = Todo::"<todoID>"` from the path. Policies can then decide on the
  specific instance (`resource == …`, `principal in resource`, ownership carried on the
  principal's claims) with **zero** database access.
- **Attributes, lazy, on demand:** if a policy dereferences `resource.owner`, Cedar
  calls the getter's `Get(Todo::"123")`; the todo slice's resolver loads the todo from
  its repository **at that moment** and maps it to a `cedar.Entity`
  (`{UID, Attributes, Parents, Tags}`). Selectivity is automatic — coarse policies load
  nothing; only attribute/relationship policies trigger a load.

**Two engineering rules (from the pull-interface's shape — `Get(uid) (Entity, bool)`,
no `context`, no `error`):**
- **Bind context at construction:** the getter is a per-request struct closing over
  `ctx` + repos (Cedar won't pass `ctx` to `Get`). This is *why* facts receive the
  request context.
- **Fail-closed error capture:** `Get` cannot return an error, so the getter records
  the first load failure; the middleware checks it after `Authorize` and returns
  **500** rather than trusting a decision made on missing data. The getter also
  **caches** per request (fixes N+1 on entity chains).

---

## 7. Principal & the deferred-authn seam

```go
type Authenticator interface {
    // Authenticate inspects the request and returns a verified principal, or
    // (anonymous, nil) when no credentials are present. Returns an error only on
    // a malformed/invalid credential (-> 401).
    Authenticate(ctx huma.Context) (Principal, error)
}

type Principal struct {
    UID    types.EntityUID // e.g. User::"alice"; or Anonymous
    Claims types.Record    // roles/groups/scopes/arbitrary — opaque to the template
}
```

- The principal's group memberships (for `principal in Group::"…"`) are built into the
  principal entity's `Parents` from claims day-one (no load); a base-package resolver
  can resolve them lazily from an IdP/DB later.
- **Dev authenticator** (template default — see §8C): derives the principal from
  explicit `X-Dev-*` headers (subject, roles), logs a prominent startup warning, and is
  called out as the #1 thing to replace in DELETE_ME/README.
- **Reference production authn** (documented, not wired): JWT via `lestrrat-go/jwx`,
  OIDC via `coreos/go-oidc` — each implements `Authenticator` and hands verified claims
  into `Principal`.

---

## 8. Settled smaller forks (proposed — confirm at review)

**A. Naming convention.** Entity types PascalCase (`Todo`, `User`, `Group`, `Org`);
actions `Action::"<resource>:<verb>"` (`"todo:create|read|update|delete|list"`);
policy IDs slice-prefixed (`todo#0`…) for merge uniqueness. Go-side, each slice owns
typed action constants (`todoauthz.ActionRead`). (Cedar formal namespaces —
`Todo::Action::"read"` — noted as the scale-up option; the string convention is
lighter for a template.)

**B. Untagged-route default = DENY (fail-closed).** Every Huma operation must declare
`Require` or `Public`; the global middleware denies the undeclared and logs a warning.
Safest posture and instructive. (Infra routes — `/healthz` `/readyz` `/metrics`
`/openapi` — are raw chi routes outside the Huma authz middleware, so they're
unaffected.)

**C. Day-one authn default — THE choice that needs your call (security stakes).**
Recommendation: **ship the dev authenticator enabled by default**, so `go run` + curl
demonstrates authz end-to-end (send `X-Dev-*` headers to be a user and exercise
allow-paths; omit them to see 401/403 enforcement). Guard rails: loud startup warning,
prominent DELETE_ME/README "replace me" guidance, and `--auth-dev-mode=false` to
disable.
- *Trade-off:* copy-to-prod footgun — a deployed copy that forgets to replace dev auth
  would trust `X-Dev-Subject` from anyone (auth bypass).
- *Safer alternative:* dev authenticator **off by default** → out-of-the-box protected
  endpoints return 401 (still demonstrates enforcement), and you enable dev-mode to see
  allow-paths. No implicit trust if copied to prod.
- A middle option: dev-mode auto-enables only outside a "production" config and refuses
  otherwise.
**→ Need your pick: dev-on-by-default (best demo) vs dev-off-by-default (safest).**

**D. Double-load default.** Day-one shipped policies are coarse (principal + URL
identity) → no resource load → no double-load. Ship the request-scoped getter cache;
document that attribute-policy consumers incur one extra PK read unless they read
through the cache (or keep attribute-cases handler-side). No special handling needed
for the default.

---

## 9. Config (Viper, `TEMPLATE_GO_API_*` prefix)

- `--authz-enabled` (default `true`) — master switch; `false` bypasses the authz
  middleware entirely (escape hatch / incremental adoption).
- `--authz-policy-dir` (optional) — load `.cedar` files from a directory instead of the
  embedded set (loaded at startup; embedded is the default).
- `--auth-dev-mode` (default per §8C) — enable the dev authenticator.
- (optional) `--auth-dev-subject` / `--auth-dev-roles` — convenience defaults for
  dev-mode when headers are absent.

---

## 10. What ships day-one vs. extension points

**Ships (the demonstration):**
- Base `authz` package; one global middleware; deny-by-default; dev authenticator;
  RFC 9457 rejections.
- A `todoauthz` slice with embedded `policy.cedar`, action constants, and a fact
  resolver; the todo routes tagged with their actions.
- A coarse reference policy (e.g. authenticated users may CRUD todos; an `admin` role
  may do anything via `base.cedar`) — exercises allow + deny without resource loads.
- Functional/integration tests covering allow, deny (401/403), public, undeclared
  (deny), and dev-mode.

**Documented extension points (not built):**
- Attribute/relationship policies (`resource.owner == principal`) via the lazy
  resolver, with the double-load note.
- Real authn (JWT/OIDC) replacing the dev authenticator.
- Lazy group/role resolution from an IdP/DB.
- Cedar schema validation (experimental in `x/exp/schema` — adopt when it graduates).

---

## 11. Out of scope / explicitly not now

Policy templates and full residual partial evaluation (absent in cedar-go today);
schema-based compile-time validation as a load-bearing feature; per-tenant template
linking; a policy admin API; external authz services. Future-slice seams unchanged
elsewhere (rate limiting, pagination, API versioning, mockery, OTel).

---

## 12. Implementation phasing (gated build — one workflow run per phase)

Branch `feat/authz-tier` in its own worktree; integrate via squash-merged PR; human
gate after each phase (per `separate-mechanical-from-design-work`).

- **Phase A — base `authz` package:** `go get cedar-go`; Authorizer + PolicySet merge;
  Principal + context; Authenticator seam + dev authenticator; global middleware
  (deny-default, 401/403/500, problem+json); `Require`/`Public` declarations + Security
  population; request-scoped lazy getter (cache + error capture); config flags;
  `base.cedar`. Composition-root wiring with an empty contribution set.
- **Phase B — `todoauthz` slice + wiring:** policies, actions, fact resolver
  (todo repo-backed); register `Contribution`; tag todo routes; URL-id → Resource.
- **Phase C — tests:** functional/integration coverage (allow/deny/public/undeclared/
  dev-mode; URL-identity; fail-closed error path). Add `//go:build` only if needed
  (these are hermetic — no container).
- **Phase D — docs:** README (authz section + the modular pattern), DELETE_ME
  (replace-the-dev-authenticator as #1; slice-removal guidance), `docs/index.md`
  quickstart; refresh OpenAPI (`moon run openapi`) for the new `Security`.

---

## 13. Open questions to resolve at/after review

1. **§8C dev-auth default** — the one decision with security stakes; needs your pick.
2. Confirm Huma exposes the path param to middleware (`ctx.Param("todoID")` or via the
   chi route context) — feasibility certain (route is matched pre-middleware), exact
   accessor to verify in Phase A.
3. Confirm cedar-go calls `EntityGetter.Get` on-demand during evaluation (interface
   shape implies it; verify the principal/resource aren't eagerly required) — Phase A.
4. Policy reload: startup-only (proposed) vs hot-reload on `--authz-policy-dir`.
