package gatekit

// Feature is the stable identifier of a gated behavior.
type Feature string

// FeatureSpec declares a feature's default and maturity from a given activation
// version onward. A feature carries one spec per version at which its default or
// maturity changes; the spec in force is the one with the highest ActiveFrom that
// is less than or equal to the current activation version.
//
// Example lifecycle for a feature introduced at activation 5 and promoted at 6:
//
//	VersionedSpecs{
//	    {ActiveFrom: 5, Default: false, Maturity: Beta},
//	    {ActiveFrom: 6, Default: true,  Maturity: GA},
//	}
type FeatureSpec struct {
	// ActiveFrom is the activation version at which this spec takes effect.
	ActiveFrom Activation
	// Default is whether the feature is on at or after ActiveFrom, unless a later
	// spec overrides it.
	Default bool
	// Maturity is the feature's lifecycle stage at this version.
	Maturity Maturity
}

// VersionedSpecs is a feature's lifecycle across activation versions. Order does
// not matter at declaration time; gatekit sorts by ActiveFrom when the gate is built.
type VersionedSpecs []FeatureSpec

// FeatureDef is a feature's registry entry: its versioned specs plus an optional
// human-readable description.
//
//	Registry{
//	    SplitCatalog: {
//	        Description: "Load the per-minor split catalog instead of the single file",
//	        Specs: VersionedSpecs{
//	            {ActiveFrom: 5, Default: false, Maturity: Beta},
//	            {ActiveFrom: 6, Default: true,  Maturity: GA},
//	        },
//	    },
//	}
type FeatureDef struct {
	// Description is a human-readable summary of the feature, for operator-facing
	// listings, generated docs, and the plan / dry-run output. It is informational
	// only: it never affects resolution or the run invariant, and MAY be empty.
	Description string
	// Specs is the feature's lifecycle across activation versions.
	Specs VersionedSpecs
}

// Registry maps each feature to its definition (description + versioned specs).
type Registry map[Feature]FeatureDef
