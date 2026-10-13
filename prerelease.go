package gatekit

// PreRelease is the maturity of a feature at a given protocol version. It mirrors
// the Kubernetes feature-gate lifecycle. Alpha features are intended to require an
// explicit opt-in by the consumer (e.g. an --experimental flag); Beta/GA follow
// their declared Default; Deprecated marks a feature on its way out.
type PreRelease string

const (
	Alpha      PreRelease = "ALPHA"
	Beta       PreRelease = "BETA"
	GA         PreRelease = "GA"
	Deprecated PreRelease = "DEPRECATED"
)
