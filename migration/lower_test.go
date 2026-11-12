package migration

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const conformanceRoot = "../docs/spec/conformance"

// declMigration is a fakeMigration that also declares reversibility and the write floor.
type declMigration struct {
	fakeMigration
	reversible   bool
	readableFrom int
	rollbackErr  error
}

func (d *declMigration) Reversible() bool  { return d.reversible }
func (d *declMigration) ReadableFrom() int { return d.readableFrom }
func (d *declMigration) Rollback(context.Context) error {
	d.rolled = true
	return d.rollbackErr
}

type rollbackFixture struct {
	Name       string `json:"name"`
	Min        int    `json:"min"`
	From       int    `json:"from"`
	To         int    `json:"to"`
	Migrations []struct {
		ID         string `json:"id"`
		Version    int    `json:"version"`
		Reversible bool   `json:"reversible"`
	} `json:"migrations"`
	Outcome struct {
		Order []string `json:"order"`
		Error string   `json:"error"`
		Steps []string `json:"steps"`
	} `json:"outcome"`
}

type floorFixture struct {
	Name       string `json:"name"`
	Min        int    `json:"min"`
	From       int    `json:"from"`
	To         int    `json:"to"`
	Migrations []struct {
		ID           string `json:"id"`
		Version      int    `json:"version"`
		ReadableFrom int    `json:"readableFrom"`
		Applies      *bool  `json:"applies"`
	} `json:"migrations"`
	Blocked []string `json:"blocked"`
}

func fixtureFiles(t *testing.T, group string) []string {
	t.Helper()
	dir := filepath.Join(conformanceRoot, group)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skip("gatekit-spec submodule not checked out or has no " + group + " fixtures; run: git submodule update --init")
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil || len(files) == 0 {
		t.Skip("no " + group + " fixtures found")
	}
	return files
}

// TestConformanceRollback runs the lowering fixtures (migration-spec §3.5).
func TestConformanceRollback(t *testing.T) {
	for _, file := range fixtureFiles(t, "rollback") {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		var fx rollbackFixture
		if err := json.Unmarshal(data, &fx); err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		var migs []Migration
		for _, m := range fx.Migrations {
			migs = append(migs, &declMigration{fakeMigration: fakeMigration{id: m.ID, version: m.Version}, reversible: m.Reversible})
		}
		plan, err := NewPlanner(migs...).PlanLower(fx.From, fx.To, fx.Min)

		switch fx.Outcome.Error {
		case "":
			if err != nil {
				t.Errorf("%s: unexpected error: %v", fx.Name, err)
				continue
			}
			got := []string{}
			for _, m := range plan.Steps {
				got = append(got, m.ID())
			}
			want := fx.Outcome.Order
			if want == nil {
				want = []string{}
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s: order = %v, want %v", fx.Name, got, want)
			}
		case "below-min":
			var e *BelowMinError
			if !errors.As(err, &e) {
				t.Errorf("%s: err = %v, want *BelowMinError", fx.Name, err)
			}
		case "irreversible-step":
			var e *IrreversibleStepError
			if !errors.As(err, &e) {
				t.Errorf("%s: err = %v, want *IrreversibleStepError", fx.Name, err)
			} else if !reflect.DeepEqual(e.Steps, fx.Outcome.Steps) {
				t.Errorf("%s: steps = %v, want %v", fx.Name, e.Steps, fx.Outcome.Steps)
			}
		default:
			t.Fatalf("%s: unknown error kind %q", fx.Name, fx.Outcome.Error)
		}
	}
}

// TestConformanceFloor runs the write-floor fixtures (activation-spec §7).
func TestConformanceFloor(t *testing.T) {
	for _, file := range fixtureFiles(t, "floor") {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		var fx floorFixture
		if err := json.Unmarshal(data, &fx); err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		var migs []Migration
		for _, m := range fx.Migrations {
			applies := m.Applies == nil || *m.Applies
			migs = append(migs, &declMigration{
				fakeMigration: fakeMigration{id: m.ID, version: m.Version, scope: Shared, applies: applies},
				readableFrom:  m.ReadableFrom,
			})
		}
		plan, err := NewPlanner(migs...).WithMin(fx.Min).Plan(context.Background(), fx.From, fx.To)
		if err != nil {
			t.Fatalf("%s: Plan: %v", fx.Name, err)
		}
		got := []string{}
		for _, s := range plan.Steps {
			if s.Blocked {
				got = append(got, s.Migration.ID())
			}
		}
		want := fx.Blocked
		if want == nil {
			want = []string{}
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: blocked = %v, want %v", fx.Name, got, want)
		}
	}
}

func TestIsReversibleDefaultsToFalse(t *testing.T) {
	if IsReversible(&fakeMigration{}) {
		t.Fatal("a migration that does not declare Reversible must be irreversible")
	}
	if !IsReversible(&declMigration{reversible: true}) {
		t.Fatal("a migration declaring Reversible() == true must be reversible")
	}
	if got := ReadableFromOf(&fakeMigration{}); got != 0 {
		t.Fatalf("ReadableFromOf default = %d, want 0", got)
	}
}

func TestUnenforcedFloorWithoutMinErrors(t *testing.T) {
	// A declared write floor with no min set would be silently ignored, so Plan refuses it.
	m := &declMigration{fakeMigration: fakeMigration{id: "a", version: 6, scope: Shared, applies: true}, readableFrom: 6}
	_, err := NewPlanner(m).Plan(context.Background(), 5, 6)
	var uf *UnenforcedFloorError
	if !errors.As(err, &uf) || !reflect.DeepEqual(uf.Steps, []string{"a"}) {
		t.Fatalf("Plan with ReadableFrom but no min = %v, want *UnenforcedFloorError{Steps:[a]}", err)
	}
}

func TestNoFloorDeclaredWithoutMinIsFine(t *testing.T) {
	// The opt-out path: no migration declares a floor, so no min is needed.
	m := &declMigration{fakeMigration: fakeMigration{id: "a", version: 6, scope: Shared, applies: true}}
	plan, err := NewPlanner(m).Plan(context.Background(), 5, 6)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Steps[0].Blocked {
		t.Fatal("no step may be blocked when no floor is declared")
	}
}

func TestExplicitZeroMinEnforcesFloor(t *testing.T) {
	// WithMin(0) is a real floor (min genuinely 0), not the unset case: a ReadableFrom
	// above 0 is blocked, never reported as a misconfiguration.
	m := &declMigration{fakeMigration: fakeMigration{id: "a", version: 6, scope: Shared, applies: true}, readableFrom: 6}
	plan, err := NewPlanner(m).WithMin(0).Plan(context.Background(), 5, 6)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Steps[0].Blocked {
		t.Fatal("ReadableFrom above an explicit min(0) must be blocked")
	}
}

func TestNegativeMinIsRejected(t *testing.T) {
	// Activation versions are non-negative, so a negative min is a misconfiguration;
	// both Plan and PlanLower refuse it rather than silently block every step.
	m := &declMigration{fakeMigration: fakeMigration{id: "a", version: 6, scope: Shared, applies: true}}

	_, err := NewPlanner(m).WithMin(-1).Plan(context.Background(), 5, 6)
	var im *InvalidMinError
	if !errors.As(err, &im) || im.Min != -1 {
		t.Fatalf("Plan with min=-1 = %v, want *InvalidMinError{Min:-1}", err)
	}

	_, err = NewPlanner(m).PlanLower(6, 4, -99)
	if !errors.As(err, &im) || im.Min != -99 {
		t.Fatalf("PlanLower with min=-99 = %v, want *InvalidMinError{Min:-99}", err)
	}
}

func TestPlanLowerRejectsNonLoweringRange(t *testing.T) {
	// Lowering must strictly decrease a non-negative version; otherwise Lower would move
	// the active version the wrong way (or nowhere) on an empty plan.
	m := &declMigration{fakeMigration: fakeMigration{id: "a", version: 6, scope: Shared, applies: true}}
	p := NewPlanner(m)
	var re *InvalidRangeError
	for _, c := range []struct {
		name          string
		from, to, min int
	}{
		{"to>from raises active", 4, 6, 0},
		{"to==from is a no-op", 6, 6, 0},
		{"negative to", 6, -1, 0},
	} {
		if _, err := p.PlanLower(c.from, c.to, c.min); !errors.As(err, &re) {
			t.Fatalf("%s: PlanLower(%d,%d,%d) = %v, want *InvalidRangeError", c.name, c.from, c.to, c.min, err)
		}
	}
}

func TestPlanRejectsReversedRangeAndNegativeReadableFrom(t *testing.T) {
	// to < from would let Finalize lower the active version without a rollback.
	m := &declMigration{fakeMigration: fakeMigration{id: "a", version: 6, scope: Shared, applies: true}}
	var re *InvalidRangeError
	if _, err := NewPlanner(m).Plan(context.Background(), 6, 4); !errors.As(err, &re) {
		t.Fatalf("Plan(6,4) = %v, want *InvalidRangeError", err)
	}

	// A negative ReadableFrom would slip past both > 0 and > min, disabling the floor.
	bad := &declMigration{fakeMigration: fakeMigration{id: "bad", version: 6, scope: Shared, applies: true}, readableFrom: -1}
	var rf *InvalidReadableFromError
	if _, err := NewPlanner(bad).Plan(context.Background(), 5, 6); !errors.As(err, &rf) || rf.Step != "bad" {
		t.Fatalf("Plan with negative readableFrom = %v, want *InvalidReadableFromError{Step:bad}", err)
	}
}

func TestRunnersRefuseBlockedStepsBeforeExecuting(t *testing.T) {
	ok := &declMigration{fakeMigration: fakeMigration{id: "ok", version: 6, scope: Local, applies: true}}
	local := &declMigration{fakeMigration: fakeMigration{id: "l", version: 6, scope: Local, applies: true}, readableFrom: 6}
	shared := &declMigration{fakeMigration: fakeMigration{id: "s", version: 6, scope: Shared, applies: true}, readableFrom: 6}
	plan, err := NewPlanner(ok, local, shared).WithMin(5).Plan(context.Background(), 5, 6)
	if err != nil {
		t.Fatal(err)
	}
	r := NewStdRunner()

	var wf *WriteFloorError
	if err := r.RunNow(context.Background(), plan); !errors.As(err, &wf) || !reflect.DeepEqual(wf.Steps, []string{"l"}) {
		t.Fatalf("RunNow err = %v, want WriteFloorError naming l", err)
	}
	if ok.executed || local.executed {
		t.Fatal("RunNow executed a step despite a blocked one")
	}

	store := &memStore{v: 5}
	if err := r.Finalize(context.Background(), plan, store); !errors.As(err, &wf) || !reflect.DeepEqual(wf.Steps, []string{"s"}) {
		t.Fatalf("Finalize err = %v, want WriteFloorError naming s", err)
	}
	if shared.executed || store.v != 5 {
		t.Fatal("Finalize executed a step or advanced active despite a blocked one")
	}
}

func lowerFixtureMigs() (a, b *declMigration, p *Planner) {
	a = &declMigration{fakeMigration: fakeMigration{id: "a", version: 5}, reversible: true}
	b = &declMigration{fakeMigration: fakeMigration{id: "b", version: 6}, reversible: true}
	return a, b, NewPlanner(a, b)
}

func TestLowerRollsBackThenLowersActive(t *testing.T) {
	a, b, p := lowerFixtureMigs()
	plan, err := p.PlanLower(6, 4, 4)
	if err != nil {
		t.Fatal(err)
	}
	store := &memStore{v: 6}
	if err := NewStdRunner().Lower(context.Background(), plan, store); err != nil {
		t.Fatal(err)
	}
	if !a.rolled || !b.rolled || store.v != 4 {
		t.Fatalf("rolled a=%v b=%v active=%d, want both rolled and active 4", a.rolled, b.rolled, store.v)
	}
}

func TestLowerFailureLeavesActiveUnchanged(t *testing.T) {
	a, b, p := lowerFixtureMigs()
	b.rollbackErr = errors.New("boom")
	plan, _ := p.PlanLower(6, 4, 4)
	store := &memStore{v: 6}
	if err := NewStdRunner().Lower(context.Background(), plan, store); err == nil {
		t.Fatal("expected an error")
	}
	if store.v != 6 {
		t.Fatalf("active = %d, want 6 (unchanged)", store.v)
	}
	if a.rolled {
		t.Fatal("steps after the failing one must not run (b is first in descending order)")
	}
}

func TestLowerStoreErrorIsReported(t *testing.T) {
	_, _, p := lowerFixtureMigs()
	plan, _ := p.PlanLower(6, 4, 4)
	if err := NewStdRunner().Lower(context.Background(), plan, failingStore{}); err == nil {
		t.Fatal("expected the store error")
	}
}

type failingStore struct{}

func (failingStore) Current() int        { return 6 }
func (failingStore) SetActive(int) error { return errors.New("disk full") }

func TestErrorMessages(t *testing.T) {
	for _, e := range []error{
		&BelowMinError{To: 3, Min: 4},
		&IrreversibleStepError{Steps: []string{"x", "y"}},
		&WriteFloorError{Steps: []string{"a"}, Min: 5},
	} {
		if e.Error() == "" {
			t.Errorf("%T has an empty message", e)
		}
	}
}

func TestReportShowsReversibilityAndBlocked(t *testing.T) {
	m := &declMigration{fakeMigration: fakeMigration{id: "a", version: 6, scope: Shared, applies: true}, readableFrom: 6}
	plan, _ := NewPlanner(m).WithMin(5).Plan(context.Background(), 5, 6)
	rep := plan.Report()
	if s := rep.RunAtFinalize[0]; s.Reversible || !s.Blocked || s.ReadableFrom != 6 {
		t.Fatalf("report step = %+v, want irreversible, blocked, readableFrom 6", s)
	}
	text := rep.String()
	for _, want := range []string{"irreversible", "BLOCKED by the write floor"} {
		if !strings.Contains(text, want) {
			t.Errorf("report text missing %q:\n%s", want, text)
		}
	}
}
