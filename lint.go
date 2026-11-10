package gatekit

import "sort"

// LintKind classifies a lifecycle lint violation (core-spec §8).
type LintKind string

const (
	// StaleSpec: a later spec has ActiveFrom <= Min, so this spec can never be the
	// effective spec for any permitted active version. Delete it and its dead path.
	StaleSpec LintKind = "stale-spec"
	// PastRemoveBy: the feature declares RemoveBy and Min >= RemoveBy. Delete it.
	PastRemoveBy LintKind = "past-remove-by"
	// DeprecatedDue: the feature's latest spec is Deprecated and due for removal.
	DeprecatedDue LintKind = "deprecated"
)

// Violation is one lifecycle lint finding.
type Violation struct {
	Feature Feature
	Kind    LintKind
	// ActiveFrom is the stale spec's activation version. It is meaningful only
	// for StaleSpec and is zero for the other kinds, so do not print it for them.
	ActiveFrom Activation
}

// kindOrder defines the ordering of kinds within a feature (core-spec §8).
var kindOrder = map[LintKind]int{StaleSpec: 0, PastRemoveBy: 1, DeprecatedDue: 2}

// Lint enforces the contract step (activation-spec §4, core-spec §8). It is a pure
// function of the registry and the support window: it needs no version source, so a
// consumer can run it from a test or CI. Violations are ordered by feature, then
// kind, then ActiveFrom; a clean registry returns none.
//
// Only WithSupportWindow affects the findings. Other options are accepted so the
// same option list can be shared with New, and are validated (an invalid one, such
// as an override naming an unknown feature, returns an error) but otherwise ignored.
//
// Lint validates the registry as New does and returns that error for an invalid
// one. It does not scan source code: deleting the dead branches a stale spec
// guarded is the consumer's job.
func Lint(reg Registry, opts ...Option) ([]Violation, error) {
	g, err := New(reg, StaticVersion(0), opts...)
	if err != nil {
		return nil, err
	}
	min := g.EffectiveVersion().Min

	var out []Violation
	for _, f := range g.Features() {
		def := g.specs[f]
		for i := 0; i+1 < len(def.Specs); i++ {
			if def.Specs[i+1].ActiveFrom <= min {
				out = append(out, Violation{Feature: f, Kind: StaleSpec, ActiveFrom: def.Specs[i].ActiveFrom})
			}
		}
		if def.RemoveBy != 0 && min >= def.RemoveBy {
			out = append(out, Violation{Feature: f, Kind: PastRemoveBy})
		}
		if def.Specs[len(def.Specs)-1].Maturity == Deprecated {
			out = append(out, Violation{Feature: f, Kind: DeprecatedDue})
		}
	}
	// Features are already sorted and stale specs ascend, so this only orders kinds.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Feature != out[j].Feature {
			return out[i].Feature < out[j].Feature
		}
		if out[i].Kind != out[j].Kind {
			return kindOrder[out[i].Kind] < kindOrder[out[j].Kind]
		}
		return out[i].ActiveFrom < out[j].ActiveFrom
	})
	return out, nil
}
