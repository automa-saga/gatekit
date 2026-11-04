package gatekit

import (
	"fmt"
	"sort"
)

// VersionSource yields the activation version currently in force. gatekit does not
// decide who owns this value: a consumer wires it to whatever is authoritative —
// a tool-decided active version (see package gatekit/migration), or an
// externally-set value.
type VersionSource interface {
	Current() Activation
}

// StaticVersion is a trivial VersionSource for a fixed activation version, convenient
// for simple consumers and tests.
type StaticVersion Activation

// Current implements VersionSource.
func (s StaticVersion) Current() Activation { return Activation(s) }

// FeatureGate resolves feature state against a VersionSource. It is safe for
// concurrent use: the registry is immutable after New and the VersionSource is
// read on each call.
type FeatureGate struct {
	specs      Registry
	source     VersionSource
	window     int
	overrides  map[Feature]bool
	allowAlpha bool
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
	for f, def := range specs {
		if len(def.Specs) == 0 {
			return nil, fmt.Errorf("gatekit: feature %q has no specs", f)
		}
		cp := append(VersionedSpecs(nil), def.Specs...)
		sort.Slice(cp, func(i, j int) bool { return cp[i].ActiveFrom < cp[j].ActiveFrom })
		for i, s := range cp {
			// ActiveFrom is a Activation (unsigned), so a negative version is
			// unrepresentable; only duplicates need checking here.
			if i > 0 && cp[i-1].ActiveFrom == s.ActiveFrom {
				return nil, fmt.Errorf("gatekit: feature %q has a duplicate activation version %d", f, s.ActiveFrom)
			}
		}
		normalized[f] = FeatureDef{Description: def.Description, Specs: cp}
	}

	for f := range cfg.overrides {
		if _, ok := normalized[f]; !ok {
			return nil, fmt.Errorf("gatekit: override for unknown feature %q", f)
		}
	}

	return &FeatureGate{
		specs:      normalized,
		source:     source,
		window:     cfg.window,
		overrides:  cfg.overrides,
		allowAlpha: cfg.allowAlpha,
	}, nil
}

// Enabled reports whether feature f is on at the current activation version. See
// resolve for the precedence (override > version-resolved default, subject to the
// ALPHA opt-in > off).
func (g *FeatureGate) Enabled(f Feature) bool {
	return g.resolve(f, g.source.Current())
}

// EnabledAt is Enabled resolved against an explicit activation version rather than
// the VersionSource. Useful for planning and tests.
func (g *FeatureGate) EnabledAt(f Feature, version Activation) bool {
	return g.resolve(f, version)
}

// resolve computes the effective state of feature f at version v, applying the
// precedence from core-spec §5, highest first:
//
//  1. an explicit override (WithOverrides) — wins regardless of the version,
//  2. the version-resolved default of the effective spec, except an ALPHA spec
//     resolves off unless the gate is opted in (WithExperimental),
//  3. off (unknown feature, or first spec not yet reached).
func (g *FeatureGate) resolve(f Feature, v Activation) bool {
	if ov, ok := g.overrides[f]; ok {
		return ov
	}
	spec, ok := g.specAt(f, v)
	if !ok {
		return false
	}
	if spec.Maturity == Alpha && !g.allowAlpha {
		return false
	}
	return spec.Default
}

// specAt returns the effective spec for f at version v. Specs are sorted
// ascending, so the last one with Version <= v wins.
func (g *FeatureGate) specAt(f Feature, v Activation) (FeatureSpec, bool) {
	def, ok := g.specs[f]
	if !ok {
		return FeatureSpec{}, false
	}
	var out FeatureSpec
	found := false
	for _, s := range def.Specs {
		if s.ActiveFrom > v {
			break
		}
		out, found = s, true
	}
	return out, found
}

// MaxActivation returns the highest activation version referenced by any spec in the
// registry, or 0 if the registry is empty. Consumers derive their max supported
// activation from this so it is never hand-maintained.
func (g *FeatureGate) MaxActivation() Activation {
	var max Activation
	for _, def := range g.specs {
		for _, s := range def.Specs {
			if s.ActiveFrom > max {
				max = s.ActiveFrom
			}
		}
	}
	return max
}

// Description returns the human-readable description registered for f, or the empty
// string if f is unknown or has none. It is informational and never affects whether
// f is Enabled.
func (g *FeatureGate) Description(f Feature) string {
	return g.specs[f].Description
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
