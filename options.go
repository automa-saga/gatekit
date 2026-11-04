package gatekit

import "fmt"

// Option configures a FeatureGate at construction. Options are applied by New in
// the order given; the last one to set a value wins.
type Option func(*config) error

// config holds the resolved construction settings before a FeatureGate is built.
// It carries defaults so New can apply options over a known baseline.
type config struct {
	// window is the support window W: the number of prior protocol versions the
	// gate also supports. Defaults to 1 (strict N/N+1).
	window int
}

// defaultConfig returns the baseline settings New applies options over.
func defaultConfig() config {
	return config{window: 1}
}

// WithSupportWindow sets the support window W — the number of prior protocol
// versions the gate supports in addition to Max, so Min = max(firstVersion,
// Max - W). W defaults to 1 (strict N/N+1); a consumer MAY widen it (N/N+3,
// N/N+5, ...). W is the support breadth and is distinct from the per-release
// increment rule (protocol version grows by at most 1 per release). W must be
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
