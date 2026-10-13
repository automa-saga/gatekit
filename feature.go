package gatekit

// Feature is the stable identifier of a gated behavior.
type Feature string

// FeatureSpec declares a feature's default and maturity from a given protocol
// version onward. A feature carries one spec per version at which its default or
// maturity changes; the spec in force is the one with the highest Version that is
// less than or equal to the current version.
//
// Example lifecycle for a feature introduced at protocol 5 and promoted at 6:
//
//	VersionedSpecs{
//	    {Version: 5, Default: false, PreRelease: Beta},
//	    {Version: 6, Default: true,  PreRelease: GA},
//	}
type FeatureSpec struct {
	// Version is the protocol version at which this spec takes effect (>= 0).
	Version int
	// Default is whether the feature is on at or after Version, unless a later
	// spec overrides it.
	Default bool
	// PreRelease is the feature's maturity at this version.
	PreRelease PreRelease
}

// VersionedSpecs is a feature's lifecycle across protocol versions. Order does
// not matter at declaration time; gatekit sorts by Version when the gate is built.
type VersionedSpecs []FeatureSpec

// Registry maps each feature to its versioned specs.
type Registry map[Feature]VersionedSpecs
