package gatekit

import "fmt"

// EffectiveVersion is the protocol-version triple a binary advertises for rolling
// upgrades (protocol-spec §2):
//
//   - Max is the newest protocol version the registry supports (MaxVersion).
//   - Min is the oldest still supported, derived from the support window W as
//     max(firstVersion, Max-W), where firstVersion is the lowest version in the
//     registry (0 if empty).
//   - Active is the version currently in force, read from the VersionSource; it is
//     the value resolution (Enabled) uses.
type EffectiveVersion struct {
	Min    int
	Max    int
	Active int
}

// VersionRangeError is the typed error Check returns when the run invariant
// Min <= Active <= Max does not hold. It carries the offending triple.
type VersionRangeError struct {
	EffectiveVersion
}

// Error implements error.
func (e *VersionRangeError) Error() string {
	return fmt.Sprintf(
		"gatekit: active protocol version %d is outside the supported range [%d, %d]",
		e.Active, e.Min, e.Max,
	)
}

// EffectiveVersion returns the {Min, Max, Active} triple for the gate. Max and Min
// are derived from the registry and the support window W; Active is read from the
// VersionSource on each call.
func (g *FeatureGate) EffectiveVersion() EffectiveVersion {
	max := g.MaxVersion()
	min := max - g.window
	if first := g.firstVersion(); min < first {
		min = first
	}
	return EffectiveVersion{Min: min, Max: max, Active: g.source.Current()}
}

// Check enforces the run invariant Min <= Active <= Max (protocol-spec §3). It
// returns a *VersionRangeError when Active is below Min (the binary is too new for
// a far-behind state) or above Max (the binary is too old). A consumer SHOULD call
// Check at startup and refuse to operate on error.
func (g *FeatureGate) Check() error {
	ev := g.EffectiveVersion()
	if ev.Active < ev.Min || ev.Active > ev.Max {
		return &VersionRangeError{EffectiveVersion: ev}
	}
	return nil
}

// firstVersion returns the lowest Version across all specs in the registry, or 0
// when the registry is empty. Each feature's specs are sorted ascending by New, so
// the first element is that feature's lowest version.
func (g *FeatureGate) firstVersion() int {
	first := 0
	seen := false
	for _, vs := range g.specs {
		v := vs[0].Version
		if !seen || v < first {
			first, seen = v, true
		}
	}
	return first
}
