package migration

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestReportStructured(t *testing.T) {
	a := &fakeMigration{id: "a", version: 5, scope: Local, applies: true}
	b := &fakeMigration{id: "b", version: 6, scope: Shared, applies: true}
	plan, _ := NewPlanner(a, b).Plan(context.Background(), 4, 6)

	rep := plan.Report()
	if rep.From != 4 || rep.To != 6 {
		t.Errorf("Report() from/to = %d/%d, want 4/6", rep.From, rep.To)
	}
	if len(rep.RunNow) != 1 || rep.RunNow[0].ID != "a" {
		t.Fatalf("RunNow = %+v, want [a]", rep.RunNow)
	}
	if rep.RunNow[0].Scope != Local || !rep.RunNow[0].RunNow {
		t.Errorf("run-now step = %+v, want local RunNow=true", rep.RunNow[0])
	}
	if rep.RunNow[0].Reason == "" || rep.RunNow[0].Description != "a" {
		t.Errorf("run-now step missing reason/description: %+v", rep.RunNow[0])
	}
	if len(rep.RunAtFinalize) != 1 || rep.RunAtFinalize[0].ID != "b" {
		t.Fatalf("RunAtFinalize = %+v, want [b]", rep.RunAtFinalize)
	}
	if rep.RunAtFinalize[0].Scope != Shared || rep.RunAtFinalize[0].RunNow {
		t.Errorf("run-at-finalize step = %+v, want shared RunNow=false", rep.RunAtFinalize[0])
	}
}

func TestReportText(t *testing.T) {
	a := &fakeMigration{id: "a", version: 5, scope: Local, applies: true}
	b := &fakeMigration{id: "b", version: 6, scope: Shared, applies: true}
	plan, _ := NewPlanner(a, b).Plan(context.Background(), 4, 6)

	text := plan.Report().String()
	for _, want := range []string{
		"Plan: activation 4 -> 6",
		"Run now (local)",
		"- a [local] a",
		"Run at finalize (shared)",
		"- b [shared] b",
		"reason:",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("report text missing %q:\n%s", want, text)
		}
	}
}

func TestReportEmptyPlan(t *testing.T) {
	plan, _ := NewPlanner().Plan(context.Background(), 5, 5)
	rep := plan.Report()
	if len(rep.RunNow) != 0 || len(rep.RunAtFinalize) != 0 {
		t.Fatalf("empty plan report has steps: %+v", rep)
	}
	text := rep.String()
	if !strings.Contains(text, "(nothing to do)") {
		t.Errorf("empty plan text = %q, want it to note nothing to do", text)
	}
}

func TestReportOnlyHeldGroup(t *testing.T) {
	b := &fakeMigration{id: "b", version: 6, scope: Shared, applies: true}
	plan, _ := NewPlanner(b).Plan(context.Background(), 5, 6)
	text := plan.Report().String()
	if !strings.Contains(text, "Run now (local)") || !strings.Contains(text, "(none)") {
		t.Errorf("want run-now group rendered as (none):\n%s", text)
	}
}

func TestReportJSONRoundTrip(t *testing.T) {
	a := &fakeMigration{id: "a", version: 5, scope: Local, applies: true}
	b := &fakeMigration{id: "b", version: 6, scope: Shared, applies: true}
	plan, _ := NewPlanner(a, b).Plan(context.Background(), 4, 6)

	data, err := json.Marshal(plan.Report())
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var back PlanReport
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if back.From != 4 || back.To != 6 || len(back.RunNow) != 1 || len(back.RunAtFinalize) != 1 {
		t.Errorf("round-tripped report = %+v", back)
	}
	if back.RunNow[0].ID != "a" || back.RunAtFinalize[0].ID != "b" {
		t.Errorf("round-tripped ids wrong: %+v", back)
	}
}
