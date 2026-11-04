package gatekit

import "fmt"

// Option configures a FeatureGate at construction. Options are applied by New in
// the order given; the last one to set a value wins.
type Option func(*config) error

// config holds the resolved construction settings before a FeatureGate is built.
// It carries defaults so New can apply options over a known baseline.
type config struct {
	// registry is the (unnormalized) feature registry New was called with. New
	// populates it before applying options so an Option that needs to validate
	// against the known features — notably WithEnvOverrides — can read it.
	registry Registry
	// window is the support window W: the number of prior activation versions the
	// gate also supports. Defaults to 1 (strict N/N+1).
	window int
	// overrides forces specific features on or off regardless of the
	// version-resolved default (core-spec §5). Nil until an override is set.
	overrides map[Feature]bool
	// allowAlpha opts the gate in to ALPHA features gate-wide (core-spec §4).
	allowAlpha bool
}

// defaultConfig returns the baseline settings New applies options over.
func defaultConfig() config {
	return config{window: 1}
}

// WithSupportWindow sets the support window W — the number of prior activation
// versions the gate supports in addition to Max, so Min = max(firstVersion,
// Max - W). W defaults to 1 (strict N/N+1); a consumer MAY widen it (N/N+3,
// N/N+5, ...). W is the support breadth and is distinct from the per-release
// increment rule (activation version grows by at most 1 per release). W must be
// non-negative.
func WithSupportWindow(w int) Option {
	return func(c *config) error {
		if w < 0 {
			return fmt.Errorf("gatekit: support window must be non-negative, got %d", w)
		}
		c.window = w
		return nil
	}
}

// WithOverrides forces specific features on or off regardless of the
// version-resolved default — the highest-precedence input to resolution
// (core-spec §5), mirroring Kubernetes --feature-gates=Foo=true. An override
// takes effect regardless of the current activation version, including before a
// feature's first spec is reached. An override MAY only name a registered
// feature; New rejects an override for an unknown feature (a mistyped identifier
// would otherwise silently force nothing). Repeated calls merge, last value wins.
func WithOverrides(overrides map[Feature]bool) Option {
	return func(c *config) error {
		if c.overrides == nil {
			c.overrides = make(map[Feature]bool, len(overrides))
		}
		for f, v := range overrides {
			c.overrides[f] = v
		}
		return nil
	}
}

// WithExperimental opts the gate in to ALPHA features gate-wide: an ALPHA feature
// then follows its declared default instead of being forced off (core-spec §4),
// the equivalent of an --experimental switch. A per-feature override (WithOverrides)
// still takes precedence over this. BETA/GA features are unaffected.
func WithExperimental(allow bool) Option {
	return func(c *config) error {
		c.allowAlpha = allow
		return nil
	}
}
