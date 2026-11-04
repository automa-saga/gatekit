package gatekit

import (
	"fmt"
	"strings"
)

// FeatureInfo is the operator-facing view of one feature at a protocol version: its
// description, maturity, and resolved state. It is what a listing / dry-run shows.
type FeatureInfo struct {
	Feature     Feature
	Description string
	// Maturity is the lifecycle stage of the effective spec at the reported version,
	// or empty if the feature's first spec is not yet reached.
	Maturity Maturity
	Enabled  bool
}

// FeatureReport is a listing of features with their descriptions and resolved state
// — the gate's --plan / dry-run view for operators and generated docs. Migration
// steps are reported separately (package gatekit/migration).
type FeatureReport struct {
	// Version is the protocol version the report was resolved against.
	Version  int
	Features []FeatureInfo
}

// Report lists every feature with its description and state resolved against the
// current VersionSource. See ReportAt for an explicit version.
func (g *FeatureGate) Report() FeatureReport {
	return g.ReportAt(g.source.Current())
}

// ReportAt lists every feature with its description and state resolved against an
// explicit version, sorted by feature identifier.
func (g *FeatureGate) ReportAt(version int) FeatureReport {
	rep := FeatureReport{Version: version}
	for _, f := range g.Features() {
		info := FeatureInfo{
			Feature:     f,
			Description: g.specs[f].Description,
			Enabled:     g.resolve(f, version),
		}
		if spec, ok := g.specAt(f, version); ok {
			info.Maturity = spec.Maturity
		}
		rep.Features = append(rep.Features, info)
	}
	return rep
}

// String renders the report as a human-readable dry-run summary, one line per
// feature with its state, maturity, and description.
func (r FeatureReport) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Features at protocol %d:\n", r.Version)
	if len(r.Features) == 0 {
		b.WriteString("  (none)\n")
		return b.String()
	}
	for _, f := range r.Features {
		state := "off"
		if f.Enabled {
			state = "on"
		}
		maturity := string(f.Maturity)
		if maturity == "" {
			maturity = "-"
		}
		fmt.Fprintf(&b, "  - %s [%s] %s", f.Feature, maturity, state)
		if f.Description != "" {
			fmt.Fprintf(&b, "  %s", f.Description)
		}
		b.WriteByte('\n')
	}
	return b.String()
}
