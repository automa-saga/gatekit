// Package migration adds deferred activation and gated, one-time migrations on
// top of the gatekit feature-gate core.
//
// Each migration is tied to an activation version and a scope. When an activation
// version activates, its local migrations may run immediately; its shared
// migrations are run-at-finalize for a coordinated finalize, so a staggered rollout never
// leaves a mixed-activation fleet. finalize is the one-way line: because migrations
// may be irreversible, binary rollback is clean only before it.
//
// The design is generalized from the migration framework in hashgraph/solo-weaver
// (internal/migration): a Migration interface, a planner that selects which
// migrations run now versus are run-at-finalize, and a runner that executes them in order
// with rollback. This scaffold runs migrations with a dependency-light sequential
// runner; richer ordered execution and reporting via automa-saga/automa is a
// follow-up.
package migration

import "context"

// Scope is a consumer-defined classification of a migration. gatekit does not fix
// the set of scopes; a consumer defines as many as it needs and maps each to a
// RunPolicy via a ScopePolicy (see Planner.WithScopePolicy). Local and Shared
// are provided only as the keys of the built-in DefaultScopePolicy, not as a closed
// enum.
type Scope string

const (
	// Local migrations affect only the host they run on. In the built-in
	// default policy they run as soon as their activation version activates (e.g. at
	// install).
	Local Scope = "local"
	// Shared migrations affect cross-host or network agreement. In the built-in
	// default policy they are run-at-finalize until a coordinated finalize, to avoid a
	// mixed-activation fleet.
	Shared Scope = "shared"
)

// RunPolicy is the timing a migration step is eligible for, resolved from its
// Scope by a ScopePolicy.
type RunPolicy string

const (
	// PolicyRunNow marks a step eligible to run at install / immediate activation.
	PolicyRunNow RunPolicy = "run-now"
	// PolicyRunAtFinalize marks a step deferred until a deliberate finalize. It is also the
	// conservative fallback for a scope the policy does not map, since a run-at-finalize step
	// never fires automatically at install.
	PolicyRunAtFinalize RunPolicy = "run-at-finalize"
)

// ScopePolicy maps a Scope to the RunPolicy that decides its timing. A consumer
// supplies one to classify its own scopes; DefaultScopePolicy is used when none is
// given.
type ScopePolicy map[Scope]RunPolicy

// DefaultScopePolicy is gatekit's built-in classifier, preserving the original
// behavior: local runs now, shared is run-at-finalize. It is applied when a consumer
// supplies no policy, and underlies any consumer policy (which may override it).
func DefaultScopePolicy() ScopePolicy {
	return ScopePolicy{Local: PolicyRunNow, Shared: PolicyRunAtFinalize}
}

// Migration is a one-time, idempotent transformation tied to an activation version.
type Migration interface {
	// ID uniquely identifies the migration (stable across releases).
	ID() string
	// Description is a short human-readable summary.
	Description() string
	// Version is the activation version this migration activates with.
	Version() int
	// Scope controls whether it may run at install or is run at finalize.
	Scope() Scope
	// Applies reports whether the migration still needs to run by reading ground
	// truth (not a ledger), so it is safe to re-run.
	Applies(ctx context.Context) (bool, error)
	// Execute performs the forward transformation.
	Execute(ctx context.Context) error
	// Rollback reverses Execute on a best-effort basis when a later step fails
	// before finalize, and, for a migration that declares itself Reversible, when
	// lowering the active version. It MUST be idempotent.
	Rollback(ctx context.Context) error
}

// Reversible is an optional interface a Migration implements to declare that
// Rollback fully and safely undoes Execute (migration-spec §1). A migration that does
// not implement it, or returns false, is irreversible: gatekit never infers
// reversibility from the mere presence of Rollback.
type Reversible interface {
	Reversible() bool
}

// Readable is an optional interface a Migration implements to declare the write
// floor (activation-spec §7): ReadableFrom is the oldest activation version able to
// read what Execute writes. A migration that does not implement it is readable by
// every version (0).
type Readable interface {
	ReadableFrom() int
}

// IsReversible reports whether m declares itself reversible. The default is false.
func IsReversible(m Migration) bool {
	r, ok := m.(Reversible)
	return ok && r.Reversible()
}

// ReadableFromOf returns the oldest version able to read m's output, or 0 when m does
// not declare one.
func ReadableFromOf(m Migration) int {
	if r, ok := m.(Readable); ok {
		return r.ReadableFrom()
	}
	return 0
}

// Store persists and reports the currently-active activation version. A consumer
// wires its active value into a gatekit.VersionSource (converting the int to
// gatekit.Activation, e.g. gatekit.StaticVersion(store.Current())), so the same
// value drives both feature-gate resolution and migration planning.
type Store interface {
	// Current returns the currently-active activation version.
	Current() int
	// SetActive persists a new active activation version: Finalize raises it and
	// Lower lowers it. A store SHOULD refuse any other change that lowers it
	// (activation-spec §6).
	SetActive(version int) error
}
