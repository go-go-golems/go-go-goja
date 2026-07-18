package gojahttp

import (
	"fmt"
	"sort"
	"strings"
)

// Grant is a typed, Go-owned permission envelope used by programmatic
// credentials and automation agents. It is intentionally more structured than
// OAuth-style scope strings so application code does not concatenate security
// policy by hand.
type Grant struct {
	Action       string
	TenantID     string
	ResourceType string
	ResourceID   string
}

// GrantSet is a normalized collection of grants. The zero value denies every
// action.
type GrantSet struct {
	Grants []Grant
}

// NewGrantSet normalizes and validates grants.
func NewGrantSet(grants ...Grant) (GrantSet, error) {
	return (GrantSet{Grants: grants}).Normalize()
}

// Normalize trims, validates, deduplicates, and sorts grants so serialized
// output and tests are deterministic.
func (s GrantSet) Normalize() (GrantSet, error) {
	seen := map[string]Grant{}
	for _, grant := range s.Grants {
		grant = normalizeGrant(grant)
		if grant.Action == "" {
			return GrantSet{}, fmt.Errorf("grant action is required")
		}
		if grant.ResourceID != "" && grant.ResourceType == "" {
			return GrantSet{}, fmt.Errorf("grant action %q resource id requires resource type", grant.Action)
		}
		seen[grant.ScopeString()] = grant
	}
	out := make([]Grant, 0, len(seen))
	for _, grant := range seen {
		out = append(out, grant)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ScopeString() < out[j].ScopeString() })
	return GrantSet{Grants: out}, nil
}

// Clone returns a deep copy of the set.
func (s GrantSet) Clone() GrantSet {
	return GrantSet{Grants: append([]Grant(nil), s.Grants...)}
}

// Intersect returns the grants allowed by both sets. Empty tenant/resource
// fields act as wildcards, and action "*" acts as an action wildcard. The
// returned grant is the more specific overlap for each compatible pair.
func (s GrantSet) Intersect(other GrantSet) (GrantSet, error) {
	left, err := s.Normalize()
	if err != nil {
		return GrantSet{}, err
	}
	right, err := other.Normalize()
	if err != nil {
		return GrantSet{}, err
	}
	out := make([]Grant, 0)
	for _, a := range left.Grants {
		for _, b := range right.Grants {
			grant, ok := intersectGrant(a, b)
			if ok {
				out = append(out, grant)
			}
		}
	}
	return NewGrantSet(out...)
}

// ScopeStrings returns a deterministic debug/wire view of the grant set.
func (s GrantSet) ScopeStrings() []string {
	normalized, err := s.Normalize()
	if err != nil {
		return nil
	}
	out := make([]string, len(normalized.Grants))
	for i, grant := range normalized.Grants {
		out[i] = grant.ScopeString()
	}
	return out
}

// Allows reports whether the set permits action for resource. A grant field left
// empty acts as a wildcard for that dimension; a populated field must match.
func (s GrantSet) Allows(action string, resource *ResourceRef) bool {
	tenantID, resourceType, resourceID := "", "", ""
	if resource != nil {
		tenantID = resource.TenantID
		resourceType = resource.Type
		resourceID = resource.ID
	}
	return s.AllowsResource(action, tenantID, resourceType, resourceID)
}

// AllowsResource reports whether the set permits action for the supplied tenant
// and resource dimensions.
func (s GrantSet) AllowsResource(action, tenantID, resourceType, resourceID string) bool {
	action = strings.TrimSpace(action)
	if action == "" {
		return false
	}
	for _, grant := range s.Grants {
		grant = normalizeGrant(grant)
		if grant.Action != "*" && grant.Action != action {
			continue
		}
		if grant.TenantID != "" && grant.TenantID != strings.TrimSpace(tenantID) {
			continue
		}
		if grant.ResourceType != "" && grant.ResourceType != strings.TrimSpace(resourceType) {
			continue
		}
		if grant.ResourceID != "" && grant.ResourceID != strings.TrimSpace(resourceID) {
			continue
		}
		return true
	}
	return false
}

// ScopeString returns a deterministic string representation suitable for wire
// protocols, storage, logs, and diagnostics. It is not the internal authority.
func (g Grant) ScopeString() string {
	g = normalizeGrant(g)
	parts := make([]string, 0, 6)
	if g.TenantID != "" {
		parts = append(parts, "tenant", g.TenantID)
	}
	if g.ResourceType != "" {
		parts = append(parts, "resource", g.ResourceType)
		if g.ResourceID != "" {
			parts = append(parts, g.ResourceID)
		}
	}
	parts = append(parts, g.Action)
	return strings.Join(parts, ":")
}

func normalizeGrant(grant Grant) Grant {
	return Grant{
		Action:       strings.TrimSpace(grant.Action),
		TenantID:     strings.TrimSpace(grant.TenantID),
		ResourceType: strings.TrimSpace(grant.ResourceType),
		ResourceID:   strings.TrimSpace(grant.ResourceID),
	}
}

func intersectGrant(a, b Grant) (Grant, bool) {
	a = normalizeGrant(a)
	b = normalizeGrant(b)
	action, ok := intersectGrantDimension(a.Action, b.Action, "*")
	if !ok {
		return Grant{}, false
	}
	tenantID, ok := intersectGrantDimension(a.TenantID, b.TenantID, "")
	if !ok {
		return Grant{}, false
	}
	resourceType, ok := intersectGrantDimension(a.ResourceType, b.ResourceType, "")
	if !ok {
		return Grant{}, false
	}
	resourceID, ok := intersectGrantDimension(a.ResourceID, b.ResourceID, "")
	if !ok {
		return Grant{}, false
	}
	return Grant{Action: action, TenantID: tenantID, ResourceType: resourceType, ResourceID: resourceID}, true
}

func intersectGrantDimension(a, b, wildcard string) (string, bool) {
	if a == b {
		return a, true
	}
	if a == wildcard {
		return b, true
	}
	if b == wildcard {
		return a, true
	}
	return "", false
}
