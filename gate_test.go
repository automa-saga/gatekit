package gatekit

import "testing"

const (
	featureA Feature = "featureA"
	featureB Feature = "featureB"
)

func testRegistry() Registry {
	return Registry{
		featureA: {
			{Version: 4, Default: false, PreRelease: Beta},
			{Version: 6, Default: true, PreRelease: GA},
		},
		featureB: {
			{Version: 5, Default: false, PreRelease: Alpha},
		},
	}
}

func TestNewValidation(t *testing.T) {
	tests := []struct {
		name    string
		specs   Registry
		source  VersionSource
		wantErr bool
	}{
		{name: "ok", specs: testRegistry(), source: StaticVersion(5)},
		{name: "nil source", specs: testRegistry(), source: nil, wantErr: true},
		{name: "empty specs", specs: Registry{featureA: {}}, source: StaticVersion(0), wantErr: true},
		{name: "negative version", specs: Registry{featureA: {{Version: -1}}}, source: StaticVersion(0), wantErr: true},
		{name: "duplicate version", specs: Registry{featureA: {{Version: 3}, {Version: 3}}}, source: StaticVersion(0), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.specs, tt.source)
			if (err != nil) != tt.wantErr {
				t.Fatalf("New() err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnabledAt(t *testing.T) {
	g, err := New(testRegistry(), StaticVersion(0))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	tests := []struct {
		feature Feature
		version int
		want    bool
	}{
		{featureA, 3, false},    // before first spec
		{featureA, 4, false},    // Beta, default off
		{featureA, 5, false},    // still the v4 spec
		{featureA, 6, true},     // GA, default on
		{featureA, 99, true},    // stays at the latest spec
		{featureB, 5, false},    // Alpha, default off
		{"unknown", 100, false}, // unknown feature
	}
	for _, tt := range tests {
		if got := g.EnabledAt(tt.feature, tt.version); got != tt.want {
			t.Errorf("EnabledAt(%q, %d) = %v, want %v", tt.feature, tt.version, got, tt.want)
		}
	}
}

func TestEnabledUsesSource(t *testing.T) {
	g, err := New(testRegistry(), StaticVersion(6))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if !g.Enabled(featureA) {
		t.Errorf("Enabled(featureA) at v6 = false, want true")
	}
	if g.Enabled(featureB) {
		t.Errorf("Enabled(featureB) at v6 = true, want false")
	}
}

func TestMaxVersion(t *testing.T) {
	g, err := New(testRegistry(), StaticVersion(0))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if got := g.MaxVersion(); got != 6 {
		t.Errorf("MaxVersion() = %d, want 6", got)
	}
	empty, err := New(Registry{}, StaticVersion(0))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if got := empty.MaxVersion(); got != 0 {
		t.Errorf("MaxVersion() empty = %d, want 0", got)
	}
}

func TestFeatures(t *testing.T) {
	g, err := New(testRegistry(), StaticVersion(0))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	got := g.Features()
	want := []Feature{featureA, featureB}
	if len(got) != len(want) {
		t.Fatalf("Features() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Features()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestStaticVersion(t *testing.T) {
	if StaticVersion(7).Current() != 7 {
		t.Errorf("StaticVersion(7).Current() != 7")
	}
}
