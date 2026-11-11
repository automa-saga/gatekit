package migration

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// BelowMinError is returned when the lowering target is below min, the oldest
// supported version (migration-spec §3.5). No supported binary can run there.
type BelowMinError struct {
	To  int
	Min int
}

// Error implements error.
func (e *BelowMinError) Error() string {
	return fmt.Sprintf("gatekit/migration: cannot lower to %d: below the oldest supported version %d", e.To, e.Min)
}

// IrreversibleStepError is returned when lowering would cross migrations that are not
// declared reversible. Steps lists every such migration id in reverse-plan order.
type IrreversibleStepError struct {
	Steps []string
}

// Error implements error.
func (e *IrreversibleStepError) Error() string {
	return fmt.Sprintf("gatekit/migration: cannot lower across irreversible migrations: %s", strings.Join(e.Steps, ", "))
}

// WriteFloorError is returned when a step's migration writes data the oldest supported
// version cannot read (ReadableFrom above min). Steps lists every blocked migration id
// in plan order.
type WriteFloorError struct {
	Steps []string
	Min   int
}

// Error implements error.
func (e *WriteFloorError) Error() string {
	return fmt.Sprintf("gatekit/migration: write floor blocks migrations (min %d): %s", e.Min, strings.Join(e.Steps, ", "))
}

// CheckWriteFloor is the write-floor preflight (migration-spec §3.6). It returns a
// *WriteFloorError naming every blocked step among those about to run, or nil. Pass
// runNow=true for RunNow, and false for Finalize. A Runner MUST call it before executing
// anything.
func (pl Plan) CheckWriteFloor(runNow bool) error {
	var ids []string
	for _, s := range pl.Steps {
		if s.RunNow == runNow && s.Blocked {
			ids = append(ids, s.Migration.ID())
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return &WriteFloorError{Steps: ids, Min: pl.Min}
}

// LowerPlan is the reverse plan for lowering the active version from From to To
// (migration-spec §3.5): the migrations to undo, in the order to undo them. Like Plan,
// it is a pure value, never stored.
type LowerPlan struct {
	From  int
	To    int
	Steps []Migration
}

// PlanLower builds the reverse plan for lowering from `from` to `to`, given min, the
// oldest supported version. It is pure: it never calls Applies, because reversal cannot
// tell a migration that ran from one whose effect was already present, so every
// migration with to < Version <= from is included (Rollback MUST be idempotent).
//
// It returns *BelowMinError when to < min (checked first), and *IrreversibleStepError
// when any migration in range is not declared reversible. Steps are ordered by version
// descending, then id descending.
func (p *Planner) PlanLower(from, to, min int) (LowerPlan, error) {
	if to < min {
		return LowerPlan{}, &BelowMinError{To: to, Min: min}
	}
	var steps []Migration
	for _, m := range p.migrations {
		if v := m.Version(); v > to && v <= from {
			steps = append(steps, m)
		}
	}
	sort.Slice(steps, func(i, j int) bool {
		if steps[i].Version() != steps[j].Version() {
			return steps[i].Version() > steps[j].Version()
		}
		return steps[i].ID() > steps[j].ID()
	})
	var irreversible []string
	for _, m := range steps {
		if !IsReversible(m) {
			irreversible = append(irreversible, m.ID())
		}
	}
	if len(irreversible) > 0 {
		return LowerPlan{}, &IrreversibleStepError{Steps: irreversible}
	}
	return LowerPlan{From: from, To: to, Steps: steps}, nil
}

// Lower runs the reverse plan's Rollbacks in order and then lowers the active version,
// the mirror of Finalize: data first, active last. It is part of the Runner contract
// (see runner.go); StdRunner is the dependency-free implementation.
func (r *StdRunner) Lower(ctx context.Context, plan LowerPlan, store Store) error {
	for _, m := range plan.Steps {
		if err := m.Rollback(ctx); err != nil {
			return fmt.Errorf("gatekit/migration: lower %q failed: %w", m.ID(), err)
		}
	}
	if err := store.SetActive(plan.To); err != nil {
		return fmt.Errorf("gatekit/migration: lower could not set %d: %w", plan.To, err)
	}
	return nil
}
