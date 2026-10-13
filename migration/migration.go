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

// Scope declares when a migration may run relative to activation.
type Scope string

const (
	// HostLocal migrations affect only the host they run on, so they may run as
	// soon as their protocol version activates (e.g. at install).
	HostLocal Scope = "host-local"
	// Interop migrations affect cross-host or network agreement, so they are held
	// until a coordinated finalize to avoid a mixed-protocol fleet.
	Interop Scope = "interop"
)

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
