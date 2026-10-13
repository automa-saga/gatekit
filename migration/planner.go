package migration

import (
	"context"
	"sort"
)

// Step is a single planned migration action.
type Step struct {
	Migration Migration
	// RunNow is true when the migration runs at this activation (host-local);
	// false when it is held for a coordinated finalize (interop).
	RunNow bool
	// Reason explains the RunNow decision, for the --plan / --dry-run preview.
	Reason string
}

// Plan is the ordered set of migration steps to move from one protocol version to
// another. It is a pure value: computed on demand, never stored.
type Plan struct {
	From  int
	To    int
	Steps []Step
}

// Planner selects and orders the migrations that apply between two versions.
type Planner struct {
	migrations []Migration
}

// NewPlanner returns a Planner over the given migrations, ordered by Version then
// ID for deterministic plans.
func NewPlanner(migrations ...Migration) *Planner {
	cp := append([]Migration(nil), migrations...)
	sort.Slice(cp, func(i, j int) bool {
		if cp[i].Version() != cp[j].Version() {
			return cp[i].Version() < cp[j].Version()
		}
		return cp[i].ID() < cp[j].ID()
	})
	return &Planner{migrations: cp}
}

// Plan returns the migrations whose Version is in the half-open range (from, to]
// and that still Applies, each marked RunNow (host-local) or held (interop).
// Migrations already satisfied (Applies == false) are omitted, so the plan is
// idempotent across re-runs.
func (p *Planner) Plan(ctx context.Context, from, to int) (Plan, error) {
	plan := Plan{From: from, To: to}
	for _, m := range p.migrations {
		v := m.Version()
		if v <= from || v > to {
			continue
		}
		applies, err := m.Applies(ctx)
		if err != nil {
			return Plan{}, err
		}
		if !applies {
			continue
		}
		step := Step{Migration: m}
		switch m.Scope() {
		case HostLocal:
			step.RunNow, step.Reason = true, "host-local: runs on activation"
		default:
			step.RunNow, step.Reason = false, "interop: held for finalize"
		}
		plan.Steps = append(plan.Steps, step)
	}
	return plan, nil
}

// Held returns the steps that will not run now (held for finalize).
func (pl Plan) Held() []Step {
	var out []Step
	for _, s := range pl.Steps {
		if !s.RunNow {
			out = append(out, s)
		}
	}
	return out
}

// RunNow returns the steps that run at this activation.
func (pl Plan) RunNow() []Step {
	var out []Step
	for _, s := range pl.Steps {
		if s.RunNow {
			out = append(out, s)
		}
	}
	return out
}
