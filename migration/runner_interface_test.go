package migration

import (
	"context"
	"testing"
)

// TestRunnerInterface documents that both constructors return a value usable as the
// Runner interface, so a consumer can hold the execution engine abstractly and swap
// in the automa-backed runner from gatekit-automa.
func TestRunnerInterface(t *testing.T) {
	var runners = []Runner{NewRunner(), NewStdRunner()}
	a := &fakeMigration{id: "a", version: 5, scope: HostLocal, applies: true}
	for i, r := range runners {
		plan, _ := NewPlanner(a).Plan(context.Background(), 4, 5)
		if err := r.RunNow(context.Background(), plan); err != nil {
			t.Errorf("runner[%d].RunNow() error: %v", i, err)
		}
	}
}
