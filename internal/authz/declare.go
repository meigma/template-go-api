package authz

import (
	"strings"

	"github.com/cedar-policy/cedar-go/types"
	"github.com/danielgtaylor/huma/v2"
)

// metadataKey is the Operation.Metadata key under which a route's authorization
// declaration is recorded. The global middleware reads it to decide how to
// enforce the operation; an operation without it is denied (fail-closed).
const metadataKey = "authz"

// SecuritySchemeName is the OpenAPI security-scheme identifier the API-key
// authenticator registers and that Require references, so protected operations
// advertise their requirement in the generated docs.
const SecuritySchemeName = "apiKey"

// kind distinguishes the two authorization declarations a route may carry.
type kind int

const (
	// kindRequire marks an operation that requires an authorized action.
	kindRequire kind = iota
	// kindPublic marks an operation explicitly opted out of authorization.
	kindPublic
)

// declaration is the parsed authorization intent for one operation, stored in
// Operation.Metadata by Require/Public and read by the middleware.
type declaration struct {
	kind kind
	// action is the Cedar action the operation requires (kindRequire only).
	action types.EntityUID
	// idParam, when non-empty, names the path parameter the middleware reads to
	// build an instance-level Resource (Type::"<id>"); empty binds the
	// type-level resource for collection operations. Used in Phase B.
	idParam string
}

// Require declares that an operation requires authorization for action. With an
// optional idParam, the middleware builds an instance-level resource from that
// path parameter (Type::"<id>"); without it, the resource is type-level (for
// collection operations). The returned map is assigned to Operation.Metadata;
// it also carries the OpenAPI Security requirement under the "security" key so a
// registrar can spread it onto Operation.Security, making the requirement
// visible in the generated docs. Only the first idParam is used.
func Require(action types.EntityUID, idParam ...string) map[string]any {
	var id string
	if len(idParam) > 0 {
		id = idParam[0]
	}

	return map[string]any{
		metadataKey: &declaration{kind: kindRequire, action: action, idParam: id},
		"security":  SecurityRequirement(),
	}
}

// Public declares that an operation is reachable without authorization. It is
// the explicit opt-out that satisfies the deny-by-default posture: an operation
// with no declaration is denied, so a public route must say so. The returned map
// is assigned to Operation.Metadata.
func Public() map[string]any {
	return map[string]any{
		metadataKey: &declaration{kind: kindPublic},
	}
}

// SecurityRequirement returns the OpenAPI security requirement for a protected
// operation, referencing the API-key scheme registered by RegisterSecurityScheme.
// Require embeds it in the operation metadata; registrars may also assign it to
// Operation.Security directly.
func SecurityRequirement() []map[string][]string {
	return []map[string][]string{{SecuritySchemeName: {}}}
}

// RegisterSecurityScheme declares the API-key security scheme on api's OpenAPI
// document, so the Security requirement Require advertises resolves to a defined
// scheme. The composition root calls it once when authorization is enabled.
func RegisterSecurityScheme(api huma.API) {
	components := api.OpenAPI().Components
	if components.SecuritySchemes == nil {
		components.SecuritySchemes = map[string]*huma.SecurityScheme{}
	}
	components.SecuritySchemes[SecuritySchemeName] = &huma.SecurityScheme{
		Type:        "apiKey",
		In:          "header",
		Name:        APIKeyHeader,
		Description: "API key supplied via the " + APIKeyHeader + " header or an Authorization: Bearer credential.",
	}
}

// resourceTypeFromAction derives the type-level Cedar resource for an action.
// By the naming convention (§8A) an action is Action::"<resource>:<verb>"; the
// resource type is the PascalCased <resource> segment (todo -> Todo), with no
// instance id. Phase B refines this to an instance resource when an idParam is
// declared. An action without the "<resource>:" prefix yields a zero resource,
// which coarse principal-only policies (for example the admin override) ignore.
func resourceTypeFromAction(action types.EntityUID) types.EntityUID {
	resource, _, found := strings.Cut(string(action.ID), ":")
	if !found || resource == "" {
		return types.EntityUID{}
	}

	return types.EntityUID{Type: types.EntityType(pascalCase(resource))}
}

// pascalCase upper-cases the first rune of s, mapping a lowercase resource
// segment to its PascalCase entity type (todo -> Todo).
func pascalCase(s string) string {
	if s == "" {
		return s
	}

	return strings.ToUpper(s[:1]) + s[1:]
}

// declarationFrom extracts the authorization declaration recorded on op by
// Require/Public. The boolean is false for an undeclared operation, which the
// middleware denies.
func declarationFrom(op *huma.Operation) (*declaration, bool) {
	if op == nil || op.Metadata == nil {
		return nil, false
	}
	decl, ok := op.Metadata[metadataKey].(*declaration)

	return decl, ok
}
