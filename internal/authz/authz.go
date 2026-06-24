package authz

import (
	_ "embed"
	"fmt"

	"github.com/cedar-policy/cedar-go"
)

// basePolicies holds the cross-cutting policies merged ahead of every slice's
// contribution. Embedded so the template authorizes out of the box with no
// external policy files.
//
//go:embed base.cedar
var basePolicies []byte

// baseSlice is the synthetic slice name used to prefix policy IDs from
// base.cedar during the merge, keeping them distinct from any domain slice.
const baseSlice = "base"

// Authorizer evaluates a Cedar request against the merged PolicySet. It is the
// single decision point the middleware calls; the resolver/getter supplies the
// entities Cedar dereferences on demand.
type Authorizer struct {
	policies *cedar.PolicySet
	// contributions is retained so the middleware can build the request-scoped
	// composite getter from each slice's ResolverFactory.
	contributions []Contribution
}

// New merges base.cedar and every contribution's policies into one runtime
// PolicySet and returns an Authorizer. Policy IDs are re-assigned with a
// slice-prefixed, per-slice index ("<slice>#<n>") so policies stay unique across
// slices after the merge. Passing no contributions yields an authorizer with
// only the base policies, which is the Phase A composition-root default.
func New(contributions []Contribution) (*Authorizer, error) {
	merged := cedar.NewPolicySet()

	if err := mergePolicies(merged, baseSlice, basePolicies); err != nil {
		return nil, fmt.Errorf("merge base policies: %w", err)
	}

	for i, c := range contributions {
		if len(c.Policies) == 0 {
			continue
		}
		// Index the slice by position so two slices that omit a name still get
		// distinct policy-ID prefixes.
		slice := fmt.Sprintf("slice%d", i)
		if err := mergePolicies(merged, slice, c.Policies); err != nil {
			return nil, fmt.Errorf("merge contribution %d policies: %w", i, err)
		}
	}

	// Prepend the always-present principal resolver so cross-cutting and slice
	// policies can test principal group membership (principal in Role::"…")
	// without any slice contributing a principal resolver.
	all := append([]Contribution{principalContribution()}, contributions...)

	return &Authorizer{policies: merged, contributions: all}, nil
}

// mergePolicies parses document and adds each policy to dst under a
// slice-prefixed ID ("<slice>#<n>"), so merged policy IDs are unique and trace
// back to their source slice.
func mergePolicies(dst *cedar.PolicySet, slice string, document []byte) error {
	list, err := cedar.NewPolicyListFromBytes(slice+".cedar", document)
	if err != nil {
		return fmt.Errorf("parse policies: %w", err)
	}

	for n, policy := range list {
		id := cedar.PolicyID(fmt.Sprintf("%s#%d", slice, n))
		dst.Add(id, policy)
	}

	return nil
}

// Authorize evaluates req against the merged PolicySet using entities, returning
// Cedar's decision and diagnostic. entities is the request-scoped composite
// getter; Cedar pulls only the entities the applicable policies dereference. The
// ctx parameter is accepted for symmetry with the call site and future tracing;
// Cedar's evaluation itself takes no context.
func (a *Authorizer) Authorize(entities cedar.EntityGetter, req cedar.Request) (cedar.Decision, cedar.Diagnostic) {
	return cedar.Authorize(a.policies, entities, req)
}

// Contributions returns the slices the authorizer was built from, so the
// middleware can assemble the per-request composite getter.
func (a *Authorizer) Contributions() []Contribution {
	return a.contributions
}
