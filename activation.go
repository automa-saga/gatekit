package gatekit

// Activation is the activation axis gatekit keys on: a non-negative, monotonically
// increasing activation version that a VersionSource supplies and that feature specs
// threshold on (FeatureSpec.ActiveFrom). It is deliberately a distinct type from a
// release version (e.g. semver), which lives in a separate number space — keeping
// them apart is what lets a release ship with a behavior dormant and flip it later.
type Activation uint
