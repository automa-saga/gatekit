// Package gatekit is a version-keyed feature-gate library for Go.
//
// It lets a project merge partial or not-yet-complete work behind a gate that is
// off, ship releases with the new code path dormant (running the old path), and
// flip it on later by advancing a single monotonic version — enabling
// trunk-based development and rolling upgrades without long-lived branches or a
// disrupted release cycle.
//
// gatekit is deliberately in-binary and deterministic: feature defaults are
// declared at compile time and resolved against a version, not fetched from a
// remote flag service. It is the standalone analogue of Kubernetes'
// component-base/featuregate.
//
// Two layers, so a consumer takes only what it needs:
//
//   - The gate core (this package): a Feature registry of VersionedSpecs and a
//     FeatureGate that resolves each feature against a pluggable VersionSource.
//     Dependency-light; usable by any Go project.
//   - The migration/activation layer (package gatekit/migration): deferred
//     activation, gated one-time migrations, and a finalize step for rolling
//     upgrades. Opt-in.
//
// gatekit never decides who owns the current version — that is the
// VersionSource's job — so a consumer whose version is set externally can reuse
// the gate core without inheriting the activation model.
package gatekit
