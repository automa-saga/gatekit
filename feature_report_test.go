package gatekit

import (
	"strings"
	"testing"
)

func TestDescriptionGetter(t *testing.T) {
	g, err := New(testRegistry(), StaticVersion(6))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if got := g.Description(featureA); got != "feature A" {
		t.Errorf("Description(featureA) = %q, want %q", got, "feature A")
	}
	if got := g.Description(featureB); got != "" {
		t.Errorf("Description(featureB) = %q, want empty", got)
	}
	if got := g.Description("unknown"); got != "" {
		t.Errorf("Description(unknown) = %q, want empty", got)
	}
}

func TestDescriptionDoesNotAffectResolution(t *testing.T) {
	withDesc := Registry{
		featureA: {Description: "has a description", Specs: VersionedSpecs{{Version: 4, Default: true, PreRelease: GA}}},
	}
	noDesc := Registry{
		featureA: {Specs: VersionedSpecs{{Version: 4, Default: true, PreRelease: GA}}},
	}
	g1, _ := New(withDesc, StaticVersion(5))
	g2, _ := New(noDesc, StaticVersion(5))
	if g1.Enabled(featureA) != g2.Enabled(featureA) {
		t.Errorf("description changed resolution: with=%v without=%v",
			g1.Enabled(featureA), g2.Enabled(featureA))
	}
	if g1.EffectiveVersion() != g2.EffectiveVersion() {
		t.Errorf("description changed EffectiveVersion: with=%+v without=%+v",
			g1.EffectiveVersion(), g2.EffectiveVersion())
	}
}

func TestReport(t *testing.T) {
	g, err := New(testRegistry(), StaticVersion(6))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	rep := g.Report()
	if rep.Version != 6 {
		t.Errorf("Report().Version = %d, want 6", rep.Version)
	}
	if len(rep.Features) != 2 {
		t.Fatalf("Report() has %d features, want 2", len(rep.Features))
	}
	// Sorted by feature identifier: featureA, featureB.
	a := rep.Features[0]
	if a.Feature != featureA || a.Description != "feature A" || !a.Enabled || a.PreRelease != GA {
		t.Errorf("featureA info = %+v, want {featureA, desc, enabled, GA}", a)
	}
	b := rep.Features[1]
	if b.Feature != featureB || b.Enabled || b.PreRelease != Alpha {
		t.Errorf("featureB info = %+v, want {featureB, off, ALPHA}", b)
	}

	text := rep.String()
	for _, want := range []string{"Features at protocol 6:", "featureA", "[GA] on", "feature A", "featureB", "[ALPHA] off"} {
		if !strings.Contains(text, want) {
			t.Errorf("report text missing %q:\n%s", want, text)
		}
	}
}

func TestReportAtUnreachedFeature(t *testing.T) {
	g, err := New(testRegistry(), StaticVersion(0))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	rep := g.ReportAt(3) // before featureA's first spec (v4)
	if rep.Features[0].PreRelease != "" {
		t.Errorf("unreached feature PreRelease = %q, want empty", rep.Features[0].PreRelease)
	}
	if rep.Features[0].Enabled {
		t.Errorf("unreached feature should be off")
	}
}
