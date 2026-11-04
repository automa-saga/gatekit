// Package migration adds deferred activation and gated, one-time migrations on
// top of the gatekit feature-gate core.
//
// Each migration is tied to a protocol version and a scope. When a protocol
// version activates, its host-local migrations may run immediately; its interop
// migrations are held for a coordinated finalize, so a staggered rollout never
// leaves a mixed-protocol fleet. finalize is the one-way line: because migrations
// may be irreversible, binary rollback is clean only before it.
//
// The design is generalized from the migration framework in hashgraph/solo-weaver
// (internal/migration): a Migration interface, a planner that selects which
// migrations run now versus are held, and a runner that executes them in order
// with rollback. This scaffold runs migrations with a dependency-light sequential
// runner; richer ordered execution and reporting via automa-saga/automa is a
// follow-up.
package migration

import "context"

// Scope is a consumer-defined classification of a migration. gatekit does not fix
// the set of scopes; a consumer defines as many as it needs and maps each to a
// RunPolicy via a ScopePolicy (see Planner.WithScopePolicy). HostLocal and Interop
// are provided only as the keys of the built-in DefaultScopePolicy, not as a closed
// enum.
type Scope string

const (
	// HostLocal migrations affect only the host they run on. In the built-in
	// default policy they run as soon as their protocol version activates (e.g. at
	// install).
	HostLocal Scope = "host-local"
	// Interop migrations affect cross-host or network agreement. In the built-in
	// default policy they are held until a coordinated finalize, to avoid a
	// mixed-protocol fleet.
	Interop Scope = "interop"
)

// RunPolicy is the timing a migration step is eligible for, resolved from its
// Scope by a ScopePolicy.
type RunPolicy string

const (
	// PolicyRunNow marks a step eligible to run at install / immediate activation.
	PolicyRunNow RunPolicy = "run-now"
	// PolicyHeld marks a step deferred until a deliberate finalize. It is also the
	// conservative fallback for a scope the policy does not map, since a held step
	// never fires automatically at install.
	PolicyHeld RunPolicy = "held"
)

// ScopePolicy maps a Scope to the RunPolicy that decides its timing. A consumer
// supplies one to classify its own scopes; DefaultScopePolicy is used when none is
// given.
type ScopePolicy map[Scope]RunPolicy

// DefaultScopePolicy is gatekit's built-in classifier, preserving the original
// behavior: host-local runs now, interop is held. It is applied when a consumer
// supplies no policy, and underlies any consumer policy (which may override it).
func DefaultScopePolicy() ScopePolicy {
	return ScopePolicy{HostLocal: PolicyRunNow, Interop: PolicyHeld}
}

// Migration is a one-time, idempotent transformation tied to a protocol version.
type Migration interface {
	// ID uniquely identifies the migration (stable across releases).
	ID() string
	// Description is a short human-readable summary.
	Description() string
	// Version is the protocol version this migration activates with.
	Version() int
	// Scope controls whether it may run at install or is held for finalize.
	Scope() Scope
	// Applies reports whether the migration still needs to run by reading ground
	// truth (not a ledger), so it is safe to re-run.
	Applies(ctx context.Context) (bool, error)
	// Execute performs the forward transformation.
	Execute(ctx context.Context) error
	// Rollback reverses Execute on a best-effort basis when a later step fails
	// before finalize.
	Rollback(ctx context.Context) error
}

// Store persists and reports the active protocol version. Its Current method
// satisfies gatekit.VersionSource, so the same value drives both feature-gate
// resolution and migration planning.
type Store interface {
	// Current returns the active protocol version.
	Current() int
	// SetActive persists a new active protocol version (the finalize step).
	SetActive(version int) error
}
