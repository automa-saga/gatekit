package gatekit

// Protocol is the activation axis gatekit keys on: a non-negative, monotonically
// increasing protocol version that a VersionSource supplies and that feature specs
// threshold on (FeatureSpec.Since). It is deliberately a distinct type from a
// release version (e.g. semver), which lives in a separate number space — keeping
// them apart is what lets a release ship with a behavior dormant and flip it later.
type Protocol uint
