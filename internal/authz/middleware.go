package authz

import (
	"log/slog"
	"net/http"

	"github.com/cedar-policy/cedar-go"
	"github.com/danielgtaylor/huma/v2"
)

// APIKeyHeader is the request header the API-key authenticator reads a key from,
// and the header named by the OpenAPI security scheme. It is defined here, in the
// base package, so the security-scheme declaration (declare.go) needs no
// dependency on the apikey adapter.
//
//nolint:gosec // G101: this is a header name, not a credential value.
const APIKeyHeader = "X-API-Key"

// genericForbidden is the client-facing detail for any authorization denial. The
// specific reason (diagnostic, missing declaration) is logged, not returned, so
// the API does not leak its policy structure.
const genericForbidden = "you are not authorized to perform this action"

// genericUnauthorized is the client-facing detail when an anonymous caller is
// denied — it signals that credentials are required.
const genericUnauthorized = "authentication is required to perform this action"

// Middleware bundles the authentication and authorization Huma middleware behind
// one switch. When disabled it is inert (pass-through), so the template stays
// green before any route carries an authorization declaration.
type Middleware struct {
	authenticator Authenticator
	authorizer    *Authorizer
	api           huma.API
	logger        *slog.Logger
	enabled       bool
}

// NewMiddleware builds the authz middleware over authenticator and authorizer.
// api is the Huma API used to write RFC 9457 problem responses; logger records
// denials and fail-closed errors. When enabled is false the middleware is a
// pass-through (Install is a no-op), the escape hatch for incremental adoption.
func NewMiddleware(
	api huma.API,
	authenticator Authenticator,
	authorizer *Authorizer,
	logger *slog.Logger,
	enabled bool,
) *Middleware {
	if logger == nil {
		logger = slog.Default()
	}

	return &Middleware{
		authenticator: authenticator,
		authorizer:    authorizer,
		api:           api,
		logger:        logger,
		enabled:       enabled,
	}
}

// Install registers the authn and authz middleware on the API, declares the
// API-key security scheme, and stamps the OpenAPI Security requirement onto every
// operation that Require declared. It must run after the operations are
// registered so ApplySecurity sees them. It is a no-op when the middleware is
// disabled, so all operations run unauthenticated and unauthorized — the default
// that keeps untagged routes (and their tests) working until they are tagged.
func (m *Middleware) Install() {
	if !m.enabled {
		return
	}

	RegisterSecurityScheme(m.api)
	ApplySecurity(m.api)
	m.api.UseMiddleware(m.authenticate, m.authorize)
}

// authenticate runs the configured Authenticator and stores the resulting
// Principal in the request context for the downstream authz middleware. A
// missing credential yields an anonymous principal (authorization decides
// whether that is acceptable); a malformed credential is rejected with 401.
func (m *Middleware) authenticate(ctx huma.Context, next func(huma.Context)) {
	principal, err := m.authenticator.Authenticate(ctx)
	if err != nil {
		// A credential was present but invalid. Do not log the credential
		// itself; the access-log middleware redacts the carrying headers.
		m.logger.WarnContext(ctx.Context(), "authentication failed", slog.Any("error", err))
		m.writeErr(ctx, http.StatusUnauthorized, genericUnauthorized)

		return
	}

	next(huma.WithContext(ctx, WithPrincipal(ctx.Context(), principal)))
}

// authorize enforces the operation's authorization declaration. Deny-by-default:
// an operation with no declaration is denied and logged; Public proceeds; Require
// evaluates the Cedar request and proceeds only on Allow. A captured entity-load
// error fails closed with 500.
func (m *Middleware) authorize(ctx huma.Context, next func(huma.Context)) {
	principal := m.principal(ctx)

	decl, ok := declarationFrom(ctx.Operation())
	if !ok {
		// Fail-closed: a route that declared neither Require nor Public is a
		// programming omission, not a public endpoint.
		m.logger.WarnContext(ctx.Context(), "denying undeclared operation",
			slog.String("operation", ctx.Operation().OperationID))
		m.deny(ctx, principal)

		return
	}

	if decl.kind == kindPublic {
		next(ctx)

		return
	}

	allowed := m.evaluate(ctx, principal, decl)
	switch allowed {
	case decisionAllow:
		next(ctx)
	case decisionDeny:
		m.deny(ctx, principal)
	case decisionError:
		m.writeErr(ctx, http.StatusInternalServerError, "authorization is temporarily unavailable")
	default:
		// Fail closed: an unrecognized outcome (including the zero value) is
		// treated as an error rather than allowed.
		m.writeErr(ctx, http.StatusInternalServerError, "authorization is temporarily unavailable")
	}
}

// outcome is the resolved authorization result the middleware acts on. The zero
// value is decisionError so any unset or unhandled outcome fails closed, keeping
// the decision pipeline deny-by-default by construction.
type outcome int

const (
	decisionError outcome = iota
	decisionAllow
	decisionDeny
)

// evaluate builds the Cedar request for decl, runs the authorizer over a
// request-scoped composite getter, and resolves the outcome. A getter load
// failure is reported as decisionError so the caller fails closed; a Cedar Deny
// (or an evaluation diagnostic) is decisionDeny.
func (m *Middleware) evaluate(ctx huma.Context, principal Principal, decl *declaration) outcome {
	getter := newGetter(ctx.Context(), principal, m.authorizer.Contributions())

	// The resource is bound at the type level (the action's resource segment by
	// convention). Instance binding from decl.idParam is not yet implemented;
	// idParam is recorded on the declaration for that step.
	req := cedar.Request{
		Principal: principal.UID,
		Action:    decl.action,
		Resource:  resourceFor(decl, ctx),
		Context:   principal.Claims,
	}

	decision, diag := m.authorizer.Authorize(getter, req)

	if err := getter.Err(); err != nil {
		m.logger.ErrorContext(ctx.Context(), "authorization entity load failed",
			slog.String("operation", ctx.Operation().OperationID),
			slog.Any("error", err))

		return decisionError
	}

	if decision == cedar.Allow {
		return decisionAllow
	}

	m.logger.InfoContext(ctx.Context(), "authorization denied",
		slog.String("operation", ctx.Operation().OperationID),
		slog.String("principal", principal.UID.String()),
		slog.String("action", decl.action.String()),
		slog.Any("reasons", diag.Reasons))

	return decisionDeny
}

// resourceFor builds the Cedar resource entity for decl. It returns a type-level
// resource derived from the action's resource segment; instance binding that
// reads decl.idParam from ctx to build an instance-level Type::"<id>" is not yet
// implemented.
func resourceFor(decl *declaration, _ huma.Context) cedar.EntityUID {
	return resourceTypeFromAction(decl.action)
}

// principal returns the request principal, defaulting to anonymous when the
// authn middleware did not store one (for example, when authz runs without authn
// in a test).
func (m *Middleware) principal(ctx huma.Context) Principal {
	if p, ok := PrincipalFrom(ctx.Context()); ok {
		return p
	}

	return Anonymous()
}

// deny writes a 403 for an authenticated caller and a 401 for an anonymous one,
// signalling that credentials are required rather than insufficient.
func (m *Middleware) deny(ctx huma.Context, principal Principal) {
	if principal.IsAnonymous() {
		m.writeErr(ctx, http.StatusUnauthorized, genericUnauthorized)

		return
	}

	m.writeErr(ctx, http.StatusForbidden, genericForbidden)
}

// writeErr emits an RFC 9457 problem response through Huma's content negotiation,
// matching the error shape every other surface returns.
func (m *Middleware) writeErr(ctx huma.Context, status int, detail string) {
	_ = huma.WriteErr(m.api, ctx, status, detail)
}
