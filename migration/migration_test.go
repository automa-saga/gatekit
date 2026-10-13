package migration

import (
	"context"
	"errors"
	"testing"
)

// fakeMigration is a test double.
type fakeMigration struct {
	id       string
	version  int
	scope    Scope
	applies  bool
	execErr  error
	executed bool
	rolled   bool
}

func (f *fakeMigration) ID() string                            { return f.id }
func (f *fakeMigration) Description() string                   { return f.id }
func (f *fakeMigration) Version() int                          { return f.version }
func (f *fakeMigration) Scope() Scope                          { return f.scope }
func (f *fakeMigration) Applies(context.Context) (bool, error) { return f.applies, nil }
func (f *fakeMigration) Execute(context.Context) error {
	f.executed = true
	return f.execErr
}
func (f *fakeMigration) Rollback(context.Context) error {
	f.rolled = true
	return nil
}

// memStore is an in-memory Store.
type memStore struct{ v int }

func (m *memStore) Current() int          { return m.v }
func (m *memStore) SetActive(v int) error { m.v = v; return nil }

func TestPlanSelectsRangeAndScope(t *testing.T) {
	migs := []Migration{
		&fakeMigration{id: "a", version: 5, scope: HostLocal, applies: true},
		&fakeMigration{id: "b", version: 6, scope: Interop, applies: true},
		&fakeMigration{id: "c", version: 6, scope: HostLocal, applies: false}, // already satisfied
		&fakeMigration{id: "d", version: 7, scope: HostLocal, applies: true},  // out of range
	}
	p := NewPlanner(migs...)
	plan, err := p.Plan(context.Background(), 4, 6)
	if err != nil {
		t.Fatalf("Plan() error: %v", err)
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("Plan() steps = %d, want 2 (%+v)", len(plan.Steps), plan.Steps)
	}
	if len(plan.RunNow()) != 1 || plan.RunNow()[0].Migration.ID() != "a" {
		t.Errorf("RunNow() = %+v, want [a]", plan.RunNow())
	}
	if len(plan.Held()) != 1 || plan.Held()[0].Migration.ID() != "b" {
		t.Errorf("Held() = %+v, want [b]", plan.Held())
	}
}

func TestRunNowExecutesOnlyHostLocal(t *testing.T) {
	a := &fakeMigration{id: "a", version: 5, scope: HostLocal, applies: true}
	b := &fakeMigration{id: "b", version: 6, scope: Interop, applies: true}
	plan, _ := NewPlanner(a, b).Plan(context.Background(), 4, 6)
	if err := NewRunner().RunNow(context.Background(), plan); err != nil {
		t.Fatalf("RunNow() error: %v", err)
	}
	if !a.executed {
		t.Errorf("host-local migration a was not executed")
	}
	if b.executed {
		t.Errorf("interop migration b should be held, not executed")
	}
}

func TestRunNowRollsBackOnFailure(t *testing.T) {
	a := &fakeMigration{id: "a", version: 5, scope: HostLocal, applies: true}
	b := &fakeMigration{id: "b", version: 5, scope: HostLocal, applies: true, execErr: errors.New("boom")}
	plan, _ := NewPlanner(a, b).Plan(context.Background(), 4, 5)
	err := NewRunner().RunNow(context.Background(), plan)
	if err == nil {
		t.Fatalf("RunNow() error = nil, want failure")
	}
	if !a.rolled {
		t.Errorf("migration a should have been rolled back after b failed")
	}
}

func TestFinalizeRunsHeldAndAdvances(t *testing.T) {
	b := &fakeMigration{id: "b", version: 6, scope: Interop, applies: true}
	plan, _ := NewPlanner(b).Plan(context.Background(), 5, 6)
	store := &memStore{v: 5}
	if err := NewRunner().Finalize(context.Background(), plan, store); err != nil {
		t.Fatalf("Finalize() error: %v", err)
	}
	if !b.executed {
		t.Errorf("interop migration b should run at finalize")
	}
	if store.Current() != 6 {
		t.Errorf("store version = %d, want 6 after finalize", store.Current())
	}
}

func TestFinalizeDoesNotAdvanceOnFailure(t *testing.T) {
	b := &fakeMigration{id: "b", version: 6, scope: Interop, applies: true, execErr: errors.New("boom")}
	plan, _ := NewPlanner(b).Plan(context.Background(), 5, 6)
	store := &memStore{v: 5}
	if err := NewRunner().Finalize(context.Background(), plan, store); err == nil {
		t.Fatalf("Finalize() error = nil, want failure")
	}
	if store.Current() != 5 {
		t.Errorf("store version = %d, want 5 (unchanged) after failed finalize", store.Current())
	}
}
