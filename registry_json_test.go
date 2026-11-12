package gatekit

import (
	"bytes"
	"reflect"
	"testing"
)

func TestRegistryJSONRoundTrip(t *testing.T) {
	// Specs are already ascending so the round trip (which sorts by ActiveFrom) is DeepEqual.
	reg := Registry{
		"A": {RemoveBy: 9, Specs: VersionedSpecs{
			{ActiveFrom: 3, Default: true, Maturity: Beta},
			{ActiveFrom: 5, Default: true, Maturity: GA},
		}},
		"B": {Description: "b", Specs: VersionedSpecs{
			{ActiveFrom: 6, Default: false, Maturity: Alpha},
		}},
	}
	data, err := MarshalRegistry(reg)
	if err != nil {
		t.Fatal(err)
	}
	got, err := UnmarshalRegistry(data)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, reg) {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", got, reg)
	}
}

func TestMarshalRegistryIsDeterministic(t *testing.T) {
	// Specs given out of order must serialize sorted, and repeated marshals are identical.
	reg := Registry{
		"Z": {Specs: VersionedSpecs{{ActiveFrom: 5, Maturity: GA}, {ActiveFrom: 2, Maturity: Beta}}},
		"A": {Specs: VersionedSpecs{{ActiveFrom: 1, Maturity: GA}}},
	}
	first, err := MarshalRegistry(reg)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		again, err := MarshalRegistry(reg)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(first, again) {
			t.Fatal("MarshalRegistry output is not deterministic")
		}
	}
	// Keys are sorted (A before Z) and Z's specs are sorted (2 before 5).
	if a, z := bytes.Index(first, []byte(`"A"`)), bytes.Index(first, []byte(`"Z"`)); a < 0 || z < 0 || a > z {
		t.Errorf("feature keys not sorted:\n%s", first)
	}
	if two, five := bytes.Index(first, []byte(`"activeFrom": 2`)), bytes.Index(first, []byte(`"activeFrom": 5`)); two < 0 || five < 0 || two > five {
		t.Errorf("specs not sorted by activeFrom:\n%s", first)
	}
}

func TestUnmarshalRegistryRejectsBadJSON(t *testing.T) {
	if _, err := UnmarshalRegistry([]byte("{ not json")); err == nil {
		t.Error("UnmarshalRegistry(bad) = nil error, want error")
	}
}
