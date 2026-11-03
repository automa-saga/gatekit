package gatekit

import (
	"fmt"
	"sort"
)

// VersionSource yields the protocol version currently in force. gatekit does not
// decide who owns this value: a consumer wires it to whatever is authoritative —
// a tool-decided active version (see package gatekit/migration), or an
// externally-set value.
type VersionSource interface {
	Current() int
}

// StaticVersion is a trivial VersionSource for a fixed version, convenient for
// simple consumers and tests.
type StaticVersion int

// Current implements VersionSource.
func (s StaticVersion) Current() int { return int(s) }

// FeatureGate resolves feature state against a VersionSource. It is safe for
// concurrent use: the registry is immutable after New and the VersionSource is
// read on each call.
type FeatureGate struct {
	specs  Registry
	source VersionSource
	window int
}

// New validates the registry and returns a FeatureGate. The registry is copied
// and each feature's specs are sorted by Version ascending. Options (see Option)
// are applied over the defaults, notably WithSupportWindow. It returns an error
// if source is nil, an option is invalid, a feature has no specs, or a feature
// has a negative or duplicate version.
func New(specs Registry, source VersionSource, opts ...Option) (*FeatureGate, error) {
	if source == nil {
		return nil, fmt.Errorf("gatekit: VersionSource must not be nil")
	}

	cfg := defaultConfig()
	for _, opt := range opts {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	normalized := make(Registry, len(specs))
	for f, vs := range specs {
		if len(vs) == 0 {
			return nil, fmt.Errorf("gatekit: feature %q has no specs", f)
		}
		cp := append(VersionedSpecs(nil), vs...)
		sort.Slice(cp, func(i, j int) bool { return cp[i].Version < cp[j].Version })
		for i, s := range cp {
			if s.Version < 0 {
				return nil, fmt.Errorf("gatekit: feature %q has a negative version %d", f, s.Version)
			}
			if i > 0 && cp[i-1].Version == s.Version {
				return nil, fmt.Errorf("gatekit: feature %q has a duplicate version %d", f, s.Version)
			}
		}
		normalized[f] = cp
	}

	return &FeatureGate{specs: normalized, source: source, window: cfg.window}, nil
}

// Enabled reports whether feature f is on at the current protocol version: the
// Default of the latest spec whose Version is <= the current version. An unknown
// feature, or one whose first spec is not yet reached, is off.
func (g *FeatureGate) Enabled(f Feature) bool {
	spec, ok := g.specAt(f, g.source.Current())
	if !ok {
		return false
	}
	return spec.Default
}

// EnabledAt is Enabled resolved against an explicit version rather than the
// VersionSource. Useful for planning and tests.
func (g *FeatureGate) EnabledAt(f Feature, version int) bool {
	spec, ok := g.specAt(f, version)
	if !ok {
		return false
	}
	return spec.Default
}

// specAt returns the effective spec for f at version v. Specs are sorted
// ascending, so the last one with Version <= v wins.
func (g *FeatureGate) specAt(f Feature, v int) (FeatureSpec, bool) {
	vs, ok := g.specs[f]
	if !ok {
		return FeatureSpec{}, false
	}
	var out FeatureSpec
	found := false
	for _, s := range vs {
		if s.Version > v {
			break
		}
		out, found = s, true
	}
	return out, found
}

// MaxVersion returns the highest protocol version referenced by any spec in the
// registry, or 0 if the registry is empty. Consumers derive their
// maxProtocolVersion from this so it is never hand-maintained.
func (g *FeatureGate) MaxVersion() int {
	max := 0
	for _, vs := range g.specs {
		for _, s := range vs {
			if s.Version > max {
				max = s.Version
			}
		}
	}
	return max
}

// Features returns the registered feature identifiers, sorted.
func (g *FeatureGate) Features() []Feature {
	out := make([]Feature, 0, len(g.specs))
	for f := range g.specs {
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
