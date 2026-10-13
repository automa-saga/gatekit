package migration

import (
	"context"
	"fmt"
)

// Runner executes migration steps in order, rolling back the steps it completed
// if a later one fails.
//
// This is the dependency-light scaffold. Integrating automa-saga/automa for
// richer ordered execution, reporting, and rollback semantics (as in
// solo-weaver's internal/migration) is tracked as a follow-up.
type Runner struct{}

// NewRunner returns a Runner.
func NewRunner() *Runner { return &Runner{} }

// RunNow executes the plan's run-now steps in order. On the first failure it rolls
// back the already-executed steps in reverse (best-effort) and returns the error.
// Held (interop) steps are ignored here; they run at finalize.
func (r *Runner) RunNow(ctx context.Context, plan Plan) error {
	var done []Migration
	for _, s := range plan.Steps {
		if !s.RunNow {
			continue
		}
		if err := s.Migration.Execute(ctx); err != nil {
			r.rollback(ctx, done)
			return fmt.Errorf("gatekit/migration: %q failed: %w", s.Migration.ID(), err)
		}
		done = append(done, s.Migration)
	}
	return nil
}

// Finalize runs the held steps of a plan (the interop migrations), then persists
// the new active version via the Store. It runs the held steps in order, rolling
// back on failure without advancing the version. On success the version is
// advanced — the one-way line.
func (r *Runner) Finalize(ctx context.Context, plan Plan, store Store) error {
	var done []Migration
	for _, s := range plan.Steps {
		if s.RunNow {
			continue
		}
		if err := s.Migration.Execute(ctx); err != nil {
			r.rollback(ctx, done)
			return fmt.Errorf("gatekit/migration: finalize %q failed: %w", s.Migration.ID(), err)
		}
		done = append(done, s.Migration)
	}
	if err := store.SetActive(plan.To); err != nil {
		r.rollback(ctx, done)
		return fmt.Errorf("gatekit/migration: finalize could not advance to %d: %w", plan.To, err)
	}
	return nil
}

func (r *Runner) rollback(ctx context.Context, done []Migration) {
	for i := len(done) - 1; i >= 0; i-- {
		_ = done[i].Rollback(ctx)
	}
}
