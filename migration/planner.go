package migration

import (
	"context"
	"fmt"
	"sort"
)

// Step is a single planned migration action.
type Step struct {
	Migration Migration
	// RunNow is true when the migration runs at this activation (local);
	// false when it is run-at-finalize for a coordinated finalize (shared).
	RunNow bool
	// Reason explains the RunNow decision, for the --plan / --dry-run preview.
	Reason string
	// Reversible is whether the migration declares itself reversible (see Reversible).
	Reversible bool
	// ReadableFrom is the oldest version able to read the migration's output.
	ReadableFrom int
	// Blocked is true when the write floor forbids running the step: ReadableFrom is
	// above the planner's min (see Planner.WithMin). It is always false when no min is set.
	Blocked bool
}

// Plan is the ordered set of migration steps to move from one activation version to
// another. It is a pure value: computed on demand, never stored.
type Plan struct {
	From  int
	To    int
	Steps []Step
	// Min is the planner's min (see Planner.WithMin), carried so a Runner can report
	// it in a write-floor error. It is meaningful only when a min was set.
	Min int
}

// Planner selects and orders the migrations that apply between two versions, and
// resolves each migration's Scope to a RunPolicy through its ScopePolicy.
type Planner struct {
	migrations []Migration
	policy     ScopePolicy
	min        int
	minSet     bool
}

// NewPlanner returns a Planner over the given migrations, ordered by Version then
// ID for deterministic plans. It uses DefaultScopePolicy until WithScopePolicy
// supplies a consumer classifier.
func NewPlanner(migrations ...Migration) *Planner {
	cp := append([]Migration(nil), migrations...)
	sort.Slice(cp, func(i, j int) bool {
		if cp[i].Version() != cp[j].Version() {
			return cp[i].Version() < cp[j].Version()
		}
		return cp[i].ID() < cp[j].ID()
	})
	return &Planner{migrations: cp, policy: DefaultScopePolicy()}
}

// WithScopePolicy registers a consumer scope→policy classifier and returns the
// Planner for chaining. The given entries are merged over the built-in default, so
// consumer scopes are added and local/shared keep their default policy unless
// overridden. A scope mapped by neither resolves to PolicyRunAtFinalize.
func (p *Planner) WithScopePolicy(policy ScopePolicy) *Planner {
	merged := DefaultScopePolicy()
	for s, pol := range policy {
		merged[s] = pol
	}
	p.policy = merged
	return p
}

// WithMin sets min, the oldest activation version this binary supports (for example
// gate.EffectiveVersion().Min), and returns the Planner for chaining. It turns on the
// write floor: a planned step is Blocked when its migration's ReadableFrom is above min
// (activation-spec §7). Without WithMin no step is ever blocked, so a consumer that
// wants the floor enforced MUST set it.
func (p *Planner) WithMin(min int) *Planner {
	p.min, p.minSet = min, true
	return p
}

// policyFor resolves a scope to its RunPolicy, defaulting to PolicyRunAtFinalize for a scope
// the policy does not map (the conservative choice: never auto-fire at install).
func (p *Planner) policyFor(s Scope) RunPolicy {
	if pol, ok := p.policy[s]; ok {
		return pol
	}
	return PolicyRunAtFinalize
}

// Plan returns the migrations whose Version is in the half-open range (from, to]
// and that still Applies, each marked RunNow (local) or run-at-finalize (shared).
// Migrations already satisfied (Applies == false) are omitted, so the plan is
// idempotent across re-runs.
//
// It validates its inputs first and returns, in order: *InvalidMinError when a negative
// min was set, *InvalidRangeError when (from, to) are not non-negative with from <= to,
// *InvalidReadableFromError when a planned migration declares a negative ReadableFrom, and
// *UnenforcedFloorError when a planned migration declares a floor but no min was set.
func (p *Planner) Plan(ctx context.Context, from, to int) (Plan, error) {
	if p.minSet && p.min < 0 {
		return Plan{}, &InvalidMinError{Min: p.min}
	}
	if from < 0 || to < 0 || to < from {
		return Plan{}, &InvalidRangeError{From: from, To: to}
	}
	plan := Plan{From: from, To: to, Min: p.min}
	var unenforced []string
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
		step := Step{Migration: m, Reversible: IsReversible(m), ReadableFrom: ReadableFromOf(m)}
		if step.ReadableFrom < 0 {
			return Plan{}, &InvalidReadableFromError{Step: m.ID(), ReadableFrom: step.ReadableFrom}
		}
		step.Blocked = p.minSet && step.ReadableFrom > p.min
		if !p.minSet && step.ReadableFrom > 0 {
			unenforced = append(unenforced, m.ID())
		}
		switch p.policyFor(m.Scope()) {
		case PolicyRunNow:
			step.RunNow = true
			step.Reason = fmt.Sprintf("%s: runs now on activation", m.Scope())
		default:
			step.RunNow = false
			step.Reason = fmt.Sprintf("%s: run at finalize", m.Scope())
		}
		plan.Steps = append(plan.Steps, step)
	}
	// A declared write floor with no min set would be silently ignored: refuse the plan so
	// the misconfiguration surfaces at planning time rather than as an unsafe run.
	if len(unenforced) > 0 {
		return Plan{}, &UnenforcedFloorError{Steps: unenforced}
	}
	return plan, nil
}

// RunAtFinalize returns the steps that will not run now (run at finalize).
func (pl Plan) RunAtFinalize() []Step {
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
