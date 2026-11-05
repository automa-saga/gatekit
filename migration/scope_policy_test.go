package migration

import (
	"context"
	"testing"
)

func planIDs(steps []Step) []string {
	out := make([]string, 0, len(steps))
	for _, s := range steps {
		out = append(out, s.Migration.ID())
	}
	return out
}

func sameIDs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestDefaultPolicyBackCompat: with no WithScopePolicy, local runs now and
// shared is run-at-finalize (unchanged behavior).
func TestDefaultPolicyBackCompat(t *testing.T) {
	a := &fakeMigration{id: "a", version: 5, scope: Local, applies: true}
	b := &fakeMigration{id: "b", version: 6, scope: Shared, applies: true}
	plan, err := NewPlanner(a, b).Plan(context.Background(), 4, 6)
	if err != nil {
		t.Fatalf("Plan() error: %v", err)
	}
	if got := planIDs(plan.RunNow()); !sameIDs(got, []string{"a"}) {
		t.Errorf("RunNow = %v, want [a]", got)
	}
	if got := planIDs(plan.RunAtFinalize()); !sameIDs(got, []string{"b"}) {
		t.Errorf("RunAtFinalize = %v, want [b]", got)
	}
}

// TestCustomThreeScopePolicy: a consumer defines three scopes mapped to policies,
// and an unmapped scope falls back to run-at-finalize.
func TestCustomThreeScopePolicy(t *testing.T) {
	const (
		filesystem Scope = "filesystem"
		hostSystem Scope = "host-system"
		cluster    Scope = "cluster"
		mystery    Scope = "mystery"
	)
	fs := &fakeMigration{id: "fs", version: 5, scope: filesystem, applies: true}
	host := &fakeMigration{id: "host", version: 5, scope: hostSystem, applies: true}
	clu := &fakeMigration{id: "clu", version: 6, scope: cluster, applies: true}
	unk := &fakeMigration{id: "unk", version: 6, scope: mystery, applies: true}

	planner := NewPlanner(fs, host, clu, unk).WithScopePolicy(ScopePolicy{
		filesystem: PolicyRunNow,
		hostSystem: PolicyRunNow,
		cluster:    PolicyRunAtFinalize,
	})
	plan, err := planner.Plan(context.Background(), 4, 6)
	if err != nil {
		t.Fatalf("Plan() error: %v", err)
	}
	if got := planIDs(plan.RunNow()); !sameIDs(got, []string{"fs", "host"}) {
		t.Errorf("RunNow = %v, want [fs host]", got)
	}
	if got := planIDs(plan.RunAtFinalize()); !sameIDs(got, []string{"clu", "unk"}) {
		t.Errorf("RunAtFinalize = %v, want [clu unk] (unmapped scope must fall back to run-at-finalize)", got)
	}
}

// TestScopePolicyOverridesDefault: a consumer may override the built-in mapping for
// a standard scope.
func TestScopePolicyOverridesDefault(t *testing.T) {
	a := &fakeMigration{id: "a", version: 5, scope: Shared, applies: true}
	planner := NewPlanner(a).WithScopePolicy(ScopePolicy{Shared: PolicyRunNow})
	plan, err := planner.Plan(context.Background(), 4, 5)
	if err != nil {
		t.Fatalf("Plan() error: %v", err)
	}
	if got := planIDs(plan.RunNow()); !sameIDs(got, []string{"a"}) {
		t.Errorf("RunNow = %v, want [a] (shared overridden to run-now)", got)
	}
	if len(plan.RunAtFinalize()) != 0 {
		t.Errorf("RunAtFinalize = %v, want empty", planIDs(plan.RunAtFinalize()))
	}
}

// TestScopePolicyKeepsDefaultsWhenMerged: a consumer policy that only names custom
// scopes still resolves local/shared via the merged-in default.
func TestScopePolicyKeepsDefaultsWhenMerged(t *testing.T) {
	a := &fakeMigration{id: "a", version: 5, scope: Local, applies: true}
	b := &fakeMigration{id: "b", version: 6, scope: "custom", applies: true}
	planner := NewPlanner(a, b).WithScopePolicy(ScopePolicy{"custom": PolicyRunNow})
	plan, err := planner.Plan(context.Background(), 4, 6)
	if err != nil {
		t.Fatalf("Plan() error: %v", err)
	}
	if got := planIDs(plan.RunNow()); !sameIDs(got, []string{"a", "b"}) {
		t.Errorf("RunNow = %v, want [a b] (local default retained)", got)
	}
}
