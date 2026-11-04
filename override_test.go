package gatekit

import "testing"

// overrideRegistry has a BETA feature (default on from v4) and an ALPHA feature
// (default on from v4) so the ALPHA opt-in rule is observable independently of
// the declared default.
func overrideRegistry() Registry {
	return Registry{
		"betaFeature":  {Specs: VersionedSpecs{{Version: 4, Default: true, Maturity: Beta}}},
		"alphaFeature": {Specs: VersionedSpecs{{Version: 4, Default: true, Maturity: Alpha}}},
	}
}

func TestResolutionPrecedence(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		feature Feature
		version int
		want    bool
	}{
		{name: "beta follows default", feature: "betaFeature", version: 5, want: true},
		{name: "alpha off by default", feature: "alphaFeature", version: 5, want: false},
		{
			name:    "alpha on with experimental opt-in",
			opts:    []Option{WithExperimental(true)},
			feature: "alphaFeature", version: 5, want: true,
		},
		{
			name:    "override forces alpha on",
			opts:    []Option{WithOverrides(map[Feature]bool{"alphaFeature": true})},
			feature: "alphaFeature", version: 5, want: true,
		},
		{
			name:    "override forces beta off",
			opts:    []Option{WithOverrides(map[Feature]bool{"betaFeature": false})},
			feature: "betaFeature", version: 5, want: false,
		},
		{
			name:    "override wins before first spec is reached",
			opts:    []Option{WithOverrides(map[Feature]bool{"betaFeature": true})},
			feature: "betaFeature", version: 3, want: true,
		},
		{
			name:    "override beats experimental opt-in",
			opts:    []Option{WithExperimental(true), WithOverrides(map[Feature]bool{"alphaFeature": false})},
			feature: "alphaFeature", version: 5, want: false,
		},
		{name: "no spec reached is off", feature: "betaFeature", version: 3, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := New(overrideRegistry(), StaticVersion(0), tt.opts...)
			if err != nil {
				t.Fatalf("New() error: %v", err)
			}
			if got := g.EnabledAt(tt.feature, tt.version); got != tt.want {
				t.Errorf("EnabledAt(%q, %d) = %v, want %v", tt.feature, tt.version, got, tt.want)
			}
		})
	}
}

func TestEnabledAppliesOverridesAndAlpha(t *testing.T) {
	g, err := New(overrideRegistry(), StaticVersion(5),
		WithExperimental(true),
		WithOverrides(map[Feature]bool{"betaFeature": false}),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if g.Enabled("betaFeature") {
		t.Errorf("Enabled(betaFeature) = true, want false (overridden off)")
	}
	if !g.Enabled("alphaFeature") {
		t.Errorf("Enabled(alphaFeature) = false, want true (experimental opt-in)")
	}
}

func TestOverrideUnknownFeatureRejected(t *testing.T) {
	_, err := New(overrideRegistry(), StaticVersion(5),
		WithOverrides(map[Feature]bool{"nope": true}),
	)
	if err == nil {
		t.Errorf("New() with override for unknown feature = nil error, want error")
	}
}

func TestWithOverridesMerges(t *testing.T) {
	g, err := New(overrideRegistry(), StaticVersion(5),
		WithOverrides(map[Feature]bool{"betaFeature": false}),
		WithOverrides(map[Feature]bool{"alphaFeature": true}),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if g.Enabled("betaFeature") || !g.Enabled("alphaFeature") {
		t.Errorf("merged overrides not both applied: beta=%v alpha=%v",
			g.Enabled("betaFeature"), g.Enabled("alphaFeature"))
	}
}
