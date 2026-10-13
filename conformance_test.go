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
	Version    int    `json:"version"`
	Default    bool   `json:"default"`
	PreRelease string `json:"preRelease"`
}

type resolutionFixture struct {
	Name     string                `json:"name"`
	Registry map[string][]specJSON `json:"registry"`
	Cases    []struct {
		Active  int             `json:"active"`
		Enabled map[string]bool `json:"enabled"`
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
		for f, specs := range fx.Registry {
			vs := make(VersionedSpecs, 0, len(specs))
			for _, s := range specs {
				vs = append(vs, FeatureSpec{Version: s.Version, Default: s.Default, PreRelease: PreRelease(s.PreRelease)})
			}
			reg[Feature(f)] = vs
		}
		gate, err := New(reg, StaticVersion(0))
		if err != nil {
			t.Fatalf("%s: New() error: %v", fx.Name, err)
		}

		for _, c := range fx.Cases {
			for feature, want := range c.Enabled {
				if got := gate.EnabledAt(Feature(feature), c.Active); got != want {
					t.Errorf("%s: EnabledAt(%q, %d) = %v, want %v", fx.Name, feature, c.Active, got, want)
				}
			}
		}
	}
}

// TestConformanceInvariant is reserved until the EffectiveVersion run invariant
// lands (automa-saga/gatekit#9); the fixtures already exist in the spec.
func TestConformanceInvariant(t *testing.T) {
	t.Skip("invariant conformance pending EffectiveVersion.Check (#9)")
}
