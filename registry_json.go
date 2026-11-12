package gatekit

import (
	"encoding/json"
	"fmt"
	"sort"
)

// jsonSpec and jsonFeatureDef are the canonical on-disk JSON shape of a Registry — the
// same shape the gatekit-spec conformance fixtures use for a registry. They are an
// implementation detail of MarshalRegistry/UnmarshalRegistry; callers work with Registry.
type jsonSpec struct {
	ActiveFrom Activation `json:"activeFrom"`
	Default    bool       `json:"default"`
	Maturity   Maturity   `json:"maturity"`
}

type jsonFeatureDef struct {
	Description string     `json:"description,omitempty"`
	RemoveBy    Activation `json:"removeBy,omitempty"`
	Specs       []jsonSpec `json:"specs"`
}

// MarshalRegistry serializes reg to canonical, indented JSON: an object keyed by feature
// name, each with its optional description/removeBy and its specs. Output is
// deterministic — feature keys are sorted by encoding/json and specs are sorted by
// ActiveFrom — so a committed snapshot (such as a baseline for Baseline) diffs cleanly and
// regenerates reproducibly.
//
// The shape matches a registry object in the conformance fixtures, so the same bytes round
// trip through UnmarshalRegistry. Any metadata a consumer wants around it (the release
// version a baseline was taken at, a timestamp) is theirs to wrap; MarshalRegistry emits
// only the registry.
func MarshalRegistry(reg Registry) ([]byte, error) {
	out := make(map[string]jsonFeatureDef, len(reg))
	for f, def := range reg {
		specs := make([]jsonSpec, len(def.Specs))
		for i, s := range def.Specs {
			specs[i] = jsonSpec(s)
		}
		sort.Slice(specs, func(i, j int) bool { return specs[i].ActiveFrom < specs[j].ActiveFrom })
		out[string(f)] = jsonFeatureDef{Description: def.Description, RemoveBy: def.RemoveBy, Specs: specs}
	}
	return json.MarshalIndent(out, "", "  ")
}

// UnmarshalRegistry parses JSON produced by MarshalRegistry (or a registry object from a
// conformance fixture) back into a Registry. It does not validate the registry's rules —
// pass the result to New, Lint, or Baseline, which report an invalid one.
func UnmarshalRegistry(data []byte) (Registry, error) {
	var in map[string]jsonFeatureDef
	if err := json.Unmarshal(data, &in); err != nil {
		return nil, fmt.Errorf("gatekit: invalid registry JSON: %w", err)
	}
	reg := make(Registry, len(in))
	for f, def := range in {
		specs := make(VersionedSpecs, len(def.Specs))
		for i, s := range def.Specs {
			specs[i] = FeatureSpec(s)
		}
		reg[Feature(f)] = FeatureDef{Description: def.Description, RemoveBy: def.RemoveBy, Specs: specs}
	}
	return reg, nil
}
