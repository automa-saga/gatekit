package gatekit

import (
	"errors"
	"testing"
)

func TestEffectiveVersion(t *testing.T) {
	tests := []struct {
		name    string
		specs   Registry
		window  int // -1 means "use the default (omit the option)"
		active  int
		wantMin int
		wantMax int
	}{
		{
			name:    "default window W=1 floors above firstVersion",
			specs:   testRegistry(), // firstVersion 4, max 6
			window:  -1,
			active:  6,
			wantMin: 5, // max(4, 6-1)
			wantMax: 6,
		},
		{
			name:    "wide window clamps to firstVersion",
			specs:   Registry{featureA: {Specs: VersionedSpecs{{Version: 0}, {Version: 6}}}},
			window:  5,
			active:  3,
			wantMin: 1, // max(0, 6-5)
			wantMax: 6,
		},
		{
			name:    "W=0 pins min to max",
			specs:   testRegistry(),
			window:  0,
			active:  6,
			wantMin: 6,
			wantMax: 6,
		},
		{
			name:    "empty registry is all zeros",
			specs:   Registry{},
			window:  -1,
			active:  0,
			wantMin: 0,
			wantMax: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts []Option
			if tt.window >= 0 {
				opts = append(opts, WithSupportWindow(tt.window))
			}
			g, err := New(tt.specs, StaticVersion(tt.active), opts...)
			if err != nil {
				t.Fatalf("New() error: %v", err)
			}
			ev := g.EffectiveVersion()
			if ev.Min != tt.wantMin || ev.Max != tt.wantMax || ev.Active != tt.active {
				t.Errorf("EffectiveVersion() = %+v, want {Min:%d Max:%d Active:%d}",
					ev, tt.wantMin, tt.wantMax, tt.active)
			}
		})
	}
}

func TestCheck(t *testing.T) {
	// testRegistry has firstVersion 4, max 6. Try both W=1 (min 5) and W=3
	// (min max(4, 6-3)=4), to cover W=1 and W>1.
	tests := []struct {
		name    string
		window  int
		active  int
		wantErr bool
	}{
		{name: "W=1 active=max ok", window: 1, active: 6, wantErr: false},
		{name: "W=1 active=min ok", window: 1, active: 5, wantErr: false},
		{name: "W=1 active below min", window: 1, active: 4, wantErr: true},
		{name: "W=1 active above max", window: 1, active: 7, wantErr: true},
		{name: "W=3 active=min ok", window: 3, active: 4, wantErr: false},
		{name: "W=3 active below min", window: 3, active: 3, wantErr: true},
		{name: "W=3 active above max", window: 3, active: 7, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := New(testRegistry(), StaticVersion(tt.active), WithSupportWindow(tt.window))
			if err != nil {
				t.Fatalf("New() error: %v", err)
			}
			err = g.Check()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Check() err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				var vre *VersionRangeError
				if !errors.As(err, &vre) {
					t.Fatalf("Check() err = %v, want *VersionRangeError", err)
				}
				if vre.Active != tt.active {
					t.Errorf("VersionRangeError.Active = %d, want %d", vre.Active, tt.active)
				}
			}
		})
	}
}

func TestCheckDefaultWindow(t *testing.T) {
	// No WithSupportWindow option => default W=1 => min 5 for testRegistry.
	g, err := New(testRegistry(), StaticVersion(4))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if err := g.Check(); err == nil {
		t.Errorf("Check() at active=4 with default W=1 = nil, want error")
	}
}

func TestWithSupportWindowRejectsNegative(t *testing.T) {
	if _, err := New(testRegistry(), StaticVersion(6), WithSupportWindow(-1)); err == nil {
		t.Errorf("New() with negative window = nil error, want error")
	}
}
