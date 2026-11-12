package gatekit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// conformanceDir is where the gatekit-spec fixtures live, as a git submodule.
const conformanceDir = "docs/spec/conformance"

type specJSON struct {
	ActiveFrom Activation `json:"activeFrom"`
	Default    bool       `json:"default"`
	Maturity   string     `json:"maturity"`
}

type featureDefJSON struct {
	Description string     `json:"description"`
	Specs       []specJSON `json:"specs"`
}

type resolutionFixture struct {
	Name     string                    `json:"name"`
	Registry map[string]featureDefJSON `json:"registry"`
	Cases    []struct {
		Active int `json:"active"`
		// AllowAlpha opts the gate in to ALPHA features gate-wide (optional,
		// default false).
		AllowAlpha bool `json:"allowAlpha"`
		// Overrides forces features on/off regardless of version (optional).
		Overrides map[string]bool `json:"overrides"`
		Enabled   map[string]bool `json:"enabled"`
	} `json:"cases"`
}

// TestConformanceResolution runs the language-neutral resolution fixtures from
// gatekit-spec against this implementation. It skips when the spec submodule is
// not checked out (e.g. a clone without `git submodule update --init`).
func TestConformanceResolution(t *testing.T) {
	if _, err := os.Stat(conformanceDir); os.IsNotExist(err) {
		t.Skip("gatekit-spec submodule not checked out; run: git submodule update --init")
	}

	files, err := filepath.Glob(filepath.Join(conformanceDir, "resolution", "*.json"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) == 0 {
		t.Skip("no resolution fixtures found")
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		var fx resolutionFixture
		if err := json.Unmarshal(data, &fx); err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}

		reg := Registry{}
		for f, def := range fx.Registry {
			vs := make(VersionedSpecs, 0, len(def.Specs))
			for _, s := range def.Specs {
				vs = append(vs, FeatureSpec{ActiveFrom: s.ActiveFrom, Default: s.Default, Maturity: Maturity(s.Maturity)})
			}
			reg[Feature(f)] = FeatureDef{Description: def.Description, Specs: vs}
		}

		for _, c := range fx.Cases {
			// Overrides and the alpha opt-in are construction-time, so the gate is
			// built per case.
			var opts []Option
			if c.AllowAlpha {
				opts = append(opts, WithExperimental(true))
			}
			if len(c.Overrides) > 0 {
				ov := make(map[Feature]bool, len(c.Overrides))
				for k, v := range c.Overrides {
					ov[Feature(k)] = v
				}
				opts = append(opts, WithOverrides(ov))
			}
			gate, err := New(reg, StaticVersion(0), opts...)
			if err != nil {
				t.Fatalf("%s: New() error: %v", fx.Name, err)
			}
			for feature, want := range c.Enabled {
				if got := gate.EnabledAt(Feature(feature), Activation(c.Active)); got != want {
					t.Errorf("%s: EnabledAt(%q, %d) = %v, want %v", fx.Name, feature, c.Active, got, want)
				}
			}
		}
	}
}

type invariantFixture struct {
	Name         string `json:"name"`
	Max          int    `json:"max"`
	Window       int    `json:"window"`
	FirstVersion int    `json:"firstVersion"`
	Cases        []struct {
		Active int  `json:"active"`
		Valid  bool `json:"valid"`
	} `json:"cases"`
}

// TestConformanceInvariant runs the language-neutral run-invariant fixtures from
// gatekit-spec (activation-spec §3) against this implementation. Each fixture gives
// max, the window W, and firstVersion; the expected min is
// max(firstVersion, max-W). The check is exercised end-to-end through New +
// Check by building a registry whose MaxActivation and firstActivation match the
// fixture. It skips when the spec submodule is not checked out.
func TestConformanceInvariant(t *testing.T) {
	if _, err := os.Stat(conformanceDir); os.IsNotExist(err) {
		t.Skip("gatekit-spec submodule not checked out; run: git submodule update --init")
	}

	files, err := filepath.Glob(filepath.Join(conformanceDir, "invariant", "*.json"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) == 0 {
		t.Skip("no invariant fixtures found")
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		var fx invariantFixture
		if err := json.Unmarshal(data, &fx); err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}

		reg := Registry{Feature("f"): {Specs: invariantSpecs(fx.FirstVersion, fx.Max)}}
		for _, c := range fx.Cases {
			gate, err := New(reg, StaticVersion(c.Active), WithSupportWindow(fx.Window))
			if err != nil {
				t.Fatalf("%s: New() error: %v", fx.Name, err)
			}
			valid := gate.Check() == nil
			if valid != c.Valid {
				t.Errorf("%s: Check(active=%d) valid = %v, want %v (max=%d window=%d first=%d)",
					fx.Name, c.Active, valid, c.Valid, fx.Max, fx.Window, fx.FirstVersion)
			}
		}
	}
}

// invariantSpecs builds a minimal VersionedSpecs whose lowest activation is first and
// whose highest is max, so the gate's MaxActivation and firstActivation match a fixture.
func invariantSpecs(first, max int) VersionedSpecs {
	if first == max {
		return VersionedSpecs{{ActiveFrom: Activation(first)}}
	}
	return VersionedSpecs{{ActiveFrom: Activation(first)}, {ActiveFrom: Activation(max)}}
}

// registryFromJSON builds a Registry from the fixture's JSON feature map.
func registryFromJSON(m map[string]featureDefJSON) Registry {
	reg := Registry{}
	for f, def := range m {
		vs := make(VersionedSpecs, 0, len(def.Specs))
		for _, s := range def.Specs {
			vs = append(vs, FeatureSpec{ActiveFrom: s.ActiveFrom, Default: s.Default, Maturity: Maturity(s.Maturity)})
		}
		reg[Feature(f)] = FeatureDef{Description: def.Description, Specs: vs}
	}
	return reg
}

type baselineFixture struct {
	Name     string `json:"name"`
	Baseline struct {
		Registry map[string]featureDefJSON `json:"registry"`
	} `json:"baseline"`
	Current struct {
		Registry map[string]featureDefJSON `json:"registry"`
	} `json:"current"`
	Violations []struct {
		Kind       string     `json:"kind"`
		Feature    string     `json:"feature"`
		ActiveFrom Activation `json:"activeFrom"`
	} `json:"violations"`
}

// TestConformanceBaseline runs the language-neutral baseline-lint fixtures from
// gatekit-spec (activation-spec §10) against this implementation. It skips when the spec
// submodule is not checked out.
func TestConformanceBaseline(t *testing.T) {
	if _, err := os.Stat(conformanceDir); os.IsNotExist(err) {
		t.Skip("gatekit-spec submodule not checked out; run: git submodule update --init")
	}

	files, err := filepath.Glob(filepath.Join(conformanceDir, "baseline", "*.json"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) == 0 {
		t.Skip("no baseline fixtures found")
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		var fx baselineFixture
		if err := json.Unmarshal(data, &fx); err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}

		got, err := Baseline(registryFromJSON(fx.Current.Registry), registryFromJSON(fx.Baseline.Registry))
		if err != nil {
			t.Fatalf("%s: Baseline() error: %v", fx.Name, err)
		}
		if len(got) != len(fx.Violations) {
			t.Fatalf("%s: got %d violations %+v, want %d", fx.Name, len(got), got, len(fx.Violations))
		}
		for i, w := range fx.Violations {
			if string(got[i].Kind) != w.Kind || string(got[i].Feature) != w.Feature || got[i].ActiveFrom != w.ActiveFrom {
				t.Errorf("%s: violation %d = {kind:%s feature:%q activeFrom:%d}, want {kind:%s feature:%q activeFrom:%d}",
					fx.Name, i, got[i].Kind, got[i].Feature, got[i].ActiveFrom, w.Kind, w.Feature, w.ActiveFrom)
			}
		}
	}
}
