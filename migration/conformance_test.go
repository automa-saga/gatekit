package migration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// planningDir is the gatekit-spec planning fixtures, relative to this package
// (the spec is a submodule at the repo root's docs/spec).
const planningDir = "../docs/spec/conformance/planning"

type planningFixture struct {
	Name       string `json:"name"`
	Migrations []struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
		Scope   string `json:"scope"`
		Applies bool   `json:"applies"`
	} `json:"migrations"`
	From   int      `json:"from"`
	To     int      `json:"to"`
	RunNow []string `json:"runNow"`
	Held   []string `json:"held"`
}

// TestConformancePlanning runs the language-neutral planning fixtures from
// gatekit-spec. It skips when the spec submodule is not checked out.
func TestConformancePlanning(t *testing.T) {
	if _, err := os.Stat(planningDir); os.IsNotExist(err) {
		t.Skip("gatekit-spec submodule not checked out; run: git submodule update --init")
	}
	files, err := filepath.Glob(filepath.Join(planningDir, "*.json"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) == 0 {
		t.Skip("no planning fixtures found")
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		var fx planningFixture
		if err := json.Unmarshal(data, &fx); err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}

		migs := make([]Migration, 0, len(fx.Migrations))
		for _, m := range fx.Migrations {
			migs = append(migs, &fakeMigration{id: m.ID, version: m.Version, scope: Scope(m.Scope), applies: m.Applies})
		}
		plan, err := NewPlanner(migs...).Plan(context.Background(), fx.From, fx.To)
		if err != nil {
			t.Fatalf("%s: Plan() error: %v", fx.Name, err)
		}

		if got := stepIDs(plan.RunNow()); !equalStrings(got, fx.RunNow) {
			t.Errorf("%s: runNow = %v, want %v", fx.Name, got, fx.RunNow)
		}
		if got := stepIDs(plan.Held()); !equalStrings(got, fx.Held) {
			t.Errorf("%s: held = %v, want %v", fx.Name, got, fx.Held)
		}
	}
}

func stepIDs(steps []Step) []string {
	out := make([]string, 0, len(steps))
	for _, s := range steps {
		out = append(out, s.Migration.ID())
	}
	return out
}

// equalStrings compares two id lists for exact order-sensitive equality, treating
// nil and empty as equal.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
