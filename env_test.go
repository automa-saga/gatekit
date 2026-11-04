package gatekit

import (
	"strings"
	"testing"
)

func TestEnvVarName(t *testing.T) {
	const prefix = "SOLO_PROVISIONER_ENABLE_"
	tests := []struct {
		feature Feature
		want    string
	}{
		{"ConsensusNodeOperatorManaged", prefix + "CONSENSUS_NODE_OPERATOR_MANAGED"},
		{"BlockNodeOperatorManaged", prefix + "BLOCK_NODE_OPERATOR_MANAGED"},
		{"SplitCatalog", prefix + "SPLIT_CATALOG"},
		{"Simple", prefix + "SIMPLE"},
		{"lowerFirst", prefix + "LOWER_FIRST"},
		{"FeatureV2Rollout", prefix + "FEATURE_V2_ROLLOUT"},
	}
	for _, tt := range tests {
		if got := EnvVarName(tt.feature, prefix); got != tt.want {
			t.Errorf("EnvVarName(%q) = %q, want %q", tt.feature, got, tt.want)
		}
	}
	// The prefix is the consumer's; an empty prefix yields the bare suffix.
	if got := EnvVarName("SplitCatalog", ""); got != "SPLIT_CATALOG" {
		t.Errorf("EnvVarName with empty prefix = %q, want %q", got, "SPLIT_CATALOG")
	}
}

func TestWithEnvOverridesApplies(t *testing.T) {
	const prefix = "GK_TEST_ENABLE_"
	t.Setenv(prefix+"BETA_FEATURE", "false") // force a default-on beta off
	t.Setenv(prefix+"ALPHA_FEATURE", "true") // force an alpha on without WithExperimental

	g, err := New(overrideRegistry(), StaticVersion(5), WithEnvOverrides(prefix))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if g.Enabled("betaFeature") {
		t.Errorf("Enabled(betaFeature) = true, want false (env override off)")
	}
	if !g.Enabled("alphaFeature") {
		t.Errorf("Enabled(alphaFeature) = false, want true (env override on, bypasses alpha gate)")
	}
}

func TestWithEnvOverridesBeforeActiveFromNotError(t *testing.T) {
	const prefix = "GK_TEST_ENABLE_"
	t.Setenv(prefix+"BETA_FEATURE", "true")

	// version 2 is before betaFeature's ActiveFrom (4); an override still applies
	// and is not an error.
	g, err := New(overrideRegistry(), StaticVersion(2), WithEnvOverrides(prefix))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if !g.Enabled("betaFeature") {
		t.Errorf("Enabled(betaFeature) at v2 = false, want true (override bypasses version gate)")
	}
}

func TestExplicitOverrideWinsOverEnv(t *testing.T) {
	const prefix = "GK_TEST_ENABLE_"
	t.Setenv(prefix+"BETA_FEATURE", "true")

	// WithOverrides applied after WithEnvOverrides wins per feature.
	g, err := New(overrideRegistry(), StaticVersion(5),
		WithEnvOverrides(prefix),
		WithOverrides(map[Feature]bool{"betaFeature": false}),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if g.Enabled("betaFeature") {
		t.Errorf("Enabled(betaFeature) = true, want false (explicit override wins over env)")
	}
}

func TestWithEnvOverridesUnknownVariableErrors(t *testing.T) {
	const prefix = "GK_TEST_ENABLE_"
	const bad = prefix + "NOPE_FEATURE"
	t.Setenv(bad, "true")

	_, err := New(overrideRegistry(), StaticVersion(5), WithEnvOverrides(prefix))
	if err == nil {
		t.Fatalf("New() with unknown env override = nil error, want error")
	}
	if !strings.Contains(err.Error(), bad) {
		t.Errorf("error %q does not name the offending variable %q", err, bad)
	}
}

func TestWithEnvOverridesUnparseableValueErrors(t *testing.T) {
	const prefix = "GK_TEST_ENABLE_"
	const bad = prefix + "BETA_FEATURE"
	t.Setenv(bad, "notabool")

	_, err := New(overrideRegistry(), StaticVersion(5), WithEnvOverrides(prefix))
	if err == nil {
		t.Fatalf("New() with unparseable env override = nil error, want error")
	}
	if !strings.Contains(err.Error(), bad) {
		t.Errorf("error %q does not name the offending variable %q", err, bad)
	}
}
