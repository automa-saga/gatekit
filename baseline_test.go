package gatekit

import "testing"

func TestMaxActivationHelper(t *testing.T) {
	if got := MaxActivation(Registry{}); got != 0 {
		t.Errorf("MaxActivation(empty) = %d, want 0", got)
	}
	reg := Registry{
		"A": {Specs: VersionedSpecs{{ActiveFrom: 3}, {ActiveFrom: 5}}},
		"B": {Specs: VersionedSpecs{{ActiveFrom: 7}}},
	}
	if got := MaxActivation(reg); got != 7 {
		t.Errorf("MaxActivation = %d, want 7", got)
	}
}

func TestBaselineEmptyBaselineFirstRelease(t *testing.T) {
	// No prior baseline (first release): bmax = 0, nothing below the ceiling to rewrite.
	base := Registry{}

	// cmax 1 is a legal single bump from 0.
	okCur := Registry{"A": {Specs: VersionedSpecs{{ActiveFrom: 1, Default: true, Maturity: GA}}}}
	v, err := Baseline(okCur, base)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 0 {
		t.Errorf("first release to v1: got %+v, want no violations", v)
	}

	// cmax 2 jumps more than one release from 0.
	jumpCur := Registry{"A": {Specs: VersionedSpecs{{ActiveFrom: 2, Default: true, Maturity: GA}}}}
	v, err = Baseline(jumpCur, base)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 1 || v[0].Kind != VersionJump {
		t.Errorf("first release jumping to v2: got %+v, want one VersionJump", v)
	}
}

func TestBaselineVersionJumpNoOverflow(t *testing.T) {
	// bmax at the Activation (unsigned) maximum: a naive cmax > bmax+1 would overflow
	// bmax+1 to 0 and falsely flag a jump when nothing changed.
	const maxAct = ^Activation(0)
	reg := Registry{"A": {Specs: VersionedSpecs{{ActiveFrom: maxAct, Default: true, Maturity: GA}}}}
	v, err := Baseline(reg, reg)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 0 {
		t.Errorf("cmax == bmax at the type max: got %+v, want no violation", v)
	}
}

func TestBaselineRejectsInvalidRegistry(t *testing.T) {
	bad := Registry{"A": {Specs: VersionedSpecs{}}} // a feature with no specs is invalid
	good := Registry{"A": {Specs: VersionedSpecs{{ActiveFrom: 1, Maturity: GA}}}}
	if _, err := Baseline(bad, good); err == nil {
		t.Error("Baseline with invalid current registry = nil error, want error")
	}
	if _, err := Baseline(good, bad); err == nil {
		t.Error("Baseline with invalid baseline registry = nil error, want error")
	}
}
