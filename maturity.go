package gatekit

// Maturity is the lifecycle stage of a feature at a given activation version. It
// mirrors the Kubernetes feature-gate lifecycle, but is named Maturity because the
// value set includes GA and Deprecated (not pre-releases). Alpha features require an
// explicit opt-in by the consumer (e.g. an --experimental flag); Beta/GA follow
// their declared Default; Deprecated marks a feature on its way out.
type Maturity string

const (
	Alpha      Maturity = "ALPHA"
	Beta       Maturity = "BETA"
	GA         Maturity = "GA"
	Deprecated Maturity = "DEPRECATED"
)
