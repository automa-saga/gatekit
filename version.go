package gatekit

import "fmt"

// EffectiveVersion is the activation-version triple a binary advertises for rolling
// upgrades (activation-spec §2):
//
//   - Max is the newest activation version the registry supports (MaxActivation).
//   - Min is the oldest still supported, derived from the support window W as
//     max(firstActivation, Max-W), where firstActivation is the lowest activation version
//     in the registry (0 if empty).
//   - Active is the activation version currently in force, read from the
//     VersionSource; it is the value resolution (Enabled) uses.
type EffectiveVersion struct {
	Min    Activation
	Max    Activation
	Active Activation
}

// VersionRangeError is the typed error Check returns when the run invariant
// Min <= Active <= Max does not hold. It carries the offending triple.
type VersionRangeError struct {
	EffectiveVersion
}

// Error implements error.
func (e *VersionRangeError) Error() string {
	return fmt.Sprintf(
		"gatekit: activation version %d is outside the supported range [%d, %d]",
		e.Active, e.Min, e.Max,
	)
}

// EffectiveVersion returns the {Min, Max, Active} triple for the gate. Max and Min
// are derived from the registry and the support window W; Active is read from the
// VersionSource on each call.
func (g *FeatureGate) EffectiveVersion() EffectiveVersion {
	max := g.MaxActivation()
	// Min = max(firstActivation, Max-W). Activation is unsigned, so floor Max-W at 0
	// rather than underflowing when the window is wider than Max.
	var floor Activation
	if w := Activation(g.window); max > w {
		floor = max - w
	}
	min := floor
	if first := g.firstActivation(); first > min {
		min = first
	}
	return EffectiveVersion{Min: min, Max: max, Active: g.source.Current()}
}

// Check enforces the run invariant Min <= Active <= Max (activation-spec §3). It
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

// firstActivation returns the lowest ActiveFrom across all specs in the registry, or 0
// when the registry is empty. Each feature's specs are sorted ascending by New, so
// the first element is that feature's lowest activation version.
func (g *FeatureGate) firstActivation() Activation {
	var first Activation
	seen := false
	for _, def := range g.specs {
		v := def.Specs[0].ActiveFrom
		if !seen || v < first {
			first, seen = v, true
		}
	}
	return first
}
