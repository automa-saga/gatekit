package gatekit

import (
	"fmt"
	"sort"
)

const (
	// VersionJump: MaxActivation(current) > MaxActivation(baseline) + 1 — the activation
	// version rose by more than one release's worth (activation-spec §10). It carries no
	// Feature or ActiveFrom.
	VersionJump LintKind = "version-jump"
	// RewroteHistory: a spec in current at ActiveFrom < MaxActivation(baseline) is new, or
	// changed how it resolves (its Default differs, or it crosses the ALPHA boundary) versus
	// baseline — a frozen, already-released spec was rewritten (activation-spec §10). It
	// carries the Feature and ActiveFrom.
	RewroteHistory LintKind = "rewrote-history"
)

// baselineRank orders baseline violations: version-jump first, then rewrote-history.
var baselineRank = map[LintKind]int{VersionJump: 0, RewroteHistory: 1}

// Baseline is the baseline lint (activation-spec §10). It enforces the increment rule —
// the activation version rises by at most one per release, and an already-released spec is
// never rewritten — which a single registry cannot reveal, because it is relative to the
// previous release. A spec at ActiveFrom 11 is valid alone; what makes it wrong is that the
// last release was at 5. The run invariant catches that at startup, i.e. when the fleet
// refuses to boot — too late; Baseline moves it to review time.
//
// It is a pure function of two registries: current (at HEAD) and baseline (the last
// released registry — typically a committed snapshot regenerated when a release is tagged).
// It needs no version source. It derives bmax = MaxActivation(baseline) and
// cmax = MaxActivation(current) and returns:
//
//   - VersionJump when cmax > bmax+1;
//   - RewroteHistory for each current spec at ActiveFrom < bmax (strictly below the released
//     ceiling) that is new, or whose Default differs or crosses the ALPHA boundary versus
//     baseline.
//
// A spec at exactly bmax is the activation frontier, where in-place promotion is the normal
// way to activate, so it is never a rewrite. A pure BETA↔GA relabel (same Default) does not
// change resolution and is not flagged. Deletion is out of scope — that is the removal
// lint's contract step ([Lint], core-spec §8).
//
// Violations are ordered VersionJump first, then RewroteHistory by feature then ActiveFrom;
// a clean pair returns none. Unlike the lifecycle [Lint] (a warning), a baseline violation
// should fail the build: a jump or a rewritten shipped spec will break or split a running
// fleet. Baseline validates both registries as New does and returns that error for an
// invalid one; an empty baseline (a first release) is valid and yields bmax 0.
func Baseline(current, baseline Registry) ([]Violation, error) {
	cur, err := New(current, StaticVersion(0))
	if err != nil {
		return nil, fmt.Errorf("gatekit: invalid current registry: %w", err)
	}
	base, err := New(baseline, StaticVersion(0))
	if err != nil {
		return nil, fmt.Errorf("gatekit: invalid baseline registry: %w", err)
	}

	bmax := MaxActivation(baseline)
	cmax := MaxActivation(current)

	var out []Violation
	// cmax > bmax+1, written without the addition so it cannot overflow when bmax is at
	// the Activation (unsigned) maximum. cmax > bmax keeps the subtraction from underflowing.
	if cmax > bmax && cmax-bmax > 1 {
		out = append(out, Violation{Kind: VersionJump})
	}

	for _, f := range cur.Features() {
		baseByVersion := make(map[Activation]FeatureSpec, len(base.specs[f].Specs))
		for _, s := range base.specs[f].Specs {
			baseByVersion[s.ActiveFrom] = s
		}
		for _, s := range cur.specs[f].Specs {
			if s.ActiveFrom >= bmax {
				continue // the frontier (== bmax) and new versions above it are not frozen
			}
			b, ok := baseByVersion[s.ActiveFrom]
			if !ok || resolutionDiffers(s, b) {
				out = append(out, Violation{Feature: f, Kind: RewroteHistory, ActiveFrom: s.ActiveFrom})
			}
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if baselineRank[out[i].Kind] != baselineRank[out[j].Kind] {
			return baselineRank[out[i].Kind] < baselineRank[out[j].Kind]
		}
		if out[i].Feature != out[j].Feature {
			return out[i].Feature < out[j].Feature
		}
		return out[i].ActiveFrom < out[j].ActiveFrom
	})
	return out, nil
}

// resolutionDiffers reports whether current spec c resolves differently from baseline spec
// b at the same version: its Default changed, or it crosses the ALPHA boundary (an ALPHA
// spec resolves off unless opted in, so gaining or losing ALPHA changes resolution). A pure
// BETA↔GA relabel with the same Default resolves identically, so it is not a difference.
func resolutionDiffers(c, b FeatureSpec) bool {
	return c.Default != b.Default || (c.Maturity == Alpha) != (b.Maturity == Alpha)
}
