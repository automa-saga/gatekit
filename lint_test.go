package gatekit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type lintFixture struct {
	Name     string `json:"name"`
	Window   int    `json:"window"`
	Registry map[string]struct {
		RemoveBy Activation `json:"removeBy"`
		Specs    []specJSON `json:"specs"`
	} `json:"registry"`
	Violations []struct {
		Feature    string     `json:"feature"`
		Kind       string     `json:"kind"`
		ActiveFrom Activation `json:"activeFrom"`
	} `json:"violations"`
}

// TestConformanceLint runs the gatekit-spec lint fixtures (core-spec §8).
func TestConformanceLint(t *testing.T) {
	if _, err := os.Stat(conformanceDir); os.IsNotExist(err) {
		t.Skip("gatekit-spec submodule not checked out; run: git submodule update --init")
	}
	files, err := filepath.Glob(filepath.Join(conformanceDir, "lint", "*.json"))
	if err != nil || len(files) == 0 {
		t.Skip("no lint fixtures found")
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		var fx lintFixture
		if err := json.Unmarshal(data, &fx); err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		reg := Registry{}
		for f, def := range fx.Registry {
			vs := make(VersionedSpecs, 0, len(def.Specs))
			for _, s := range def.Specs {
				vs = append(vs, FeatureSpec{ActiveFrom: s.ActiveFrom, Default: s.Default, Maturity: Maturity(s.Maturity)})
			}
			reg[Feature(f)] = FeatureDef{RemoveBy: def.RemoveBy, Specs: vs}
		}
		want := []Violation{}
		for _, v := range fx.Violations {
			want = append(want, Violation{Feature: Feature(v.Feature), Kind: LintKind(v.Kind), ActiveFrom: v.ActiveFrom})
		}
		got, err := Lint(reg, WithSupportWindow(fx.Window))
		if err != nil {
			t.Fatalf("%s: Lint: %v", fx.Name, err)
		}
		if got == nil {
			got = []Violation{}
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %+v, want %+v", fx.Name, got, want)
		}
	}
}

func TestLintInvalidRegistry(t *testing.T) {
	if _, err := Lint(Registry{"A": {}}); err == nil {
		t.Fatal("expected an error for a feature with no specs")
	}
	if _, err := Lint(Registry{"A": {Specs: VersionedSpecs{{ActiveFrom: 1}}}}, WithSupportWindow(-1)); err == nil {
		t.Fatal("expected an error for a negative window")
	}
}

func TestLintEmptyRegistry(t *testing.T) {
	got, err := Lint(Registry{})
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v; want no violations", got, err)
	}
}

func TestRemoveByDoesNotAffectResolution(t *testing.T) {
	g, err := New(Registry{"A": {RemoveBy: 1, Specs: VersionedSpecs{{ActiveFrom: 2, Default: true, Maturity: GA}}}}, StaticVersion(2))
	if err != nil {
		t.Fatal(err)
	}
	if !g.Enabled("A") {
		t.Fatal("RemoveBy must not change resolution")
	}
}
