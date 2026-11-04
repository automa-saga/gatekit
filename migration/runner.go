package migration

import (
	"context"
	"fmt"
)

// Runner executes a migration Plan. gatekit ships StdRunner, a dependency-free
// sequential implementation; the separate module github.com/automa-saga/gatekit-automa
// provides a durable, crash-resumable implementation backed by automa. Both satisfy
// this interface, so a consumer swaps the execution engine without changing call
// sites — and the automa dependency never enters a plain gatekit consumer's module
// graph.
type Runner interface {
	// RunNow executes the plan's run-now (host-local) steps in plan order. On the
	// first failure it rolls back the already-executed steps in reverse order
	// (best-effort) and returns the error. Held steps are ignored here; they run at
	// Finalize. It MUST NOT advance the active version.
	RunNow(ctx context.Context, plan Plan) error
	// Finalize runs the plan's held (interop) steps in plan order, then advances the
	// active version to plan.To via the Store. On failure it rolls back the executed
	// held steps and leaves the active version unchanged; the version advances only
	// when every held step succeeds (the one-way line).
	Finalize(ctx context.Context, plan Plan, store Store) error
}

// StdRunner is the dependency-free, sequential Runner: it executes steps in plan
// order and compensates completed steps in reverse on failure. It is the default
// for consumers that do not need durability; for crash-safe, resumable execution
// use the automa-backed Runner in github.com/automa-saga/gatekit-automa.
type StdRunner struct{}

// NewStdRunner returns a StdRunner.
func NewStdRunner() *StdRunner { return &StdRunner{} }

// NewRunner returns the default (dependency-free) Runner. It is retained as a
// convenience alias for NewStdRunner.
func NewRunner() *StdRunner { return NewStdRunner() }

// StdRunner satisfies Runner.
var _ Runner = (*StdRunner)(nil)

// RunNow executes the plan's run-now steps in order. On the first failure it rolls
// back the already-executed steps in reverse (best-effort) and returns the error.
// Held (interop) steps are ignored here; they run at finalize.
func (r *StdRunner) RunNow(ctx context.Context, plan Plan) error {
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
func (r *StdRunner) Finalize(ctx context.Context, plan Plan, store Store) error {
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

func (r *StdRunner) rollback(ctx context.Context, done []Migration) {
	for i := len(done) - 1; i >= 0; i-- {
		_ = done[i].Rollback(ctx)
	}
}
