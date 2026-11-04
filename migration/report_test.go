package migration

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestReportStructured(t *testing.T) {
	a := &fakeMigration{id: "a", version: 5, scope: HostLocal, applies: true}
	b := &fakeMigration{id: "b", version: 6, scope: Interop, applies: true}
	plan, _ := NewPlanner(a, b).Plan(context.Background(), 4, 6)

	rep := plan.Report()
	if rep.From != 4 || rep.To != 6 {
		t.Errorf("Report() from/to = %d/%d, want 4/6", rep.From, rep.To)
	}
	if len(rep.RunNow) != 1 || rep.RunNow[0].ID != "a" {
		t.Fatalf("RunNow = %+v, want [a]", rep.RunNow)
	}
	if rep.RunNow[0].Scope != HostLocal || !rep.RunNow[0].RunNow {
		t.Errorf("run-now step = %+v, want host-local RunNow=true", rep.RunNow[0])
	}
	if rep.RunNow[0].Reason == "" || rep.RunNow[0].Description != "a" {
		t.Errorf("run-now step missing reason/description: %+v", rep.RunNow[0])
	}
	if len(rep.Held) != 1 || rep.Held[0].ID != "b" {
		t.Fatalf("Held = %+v, want [b]", rep.Held)
	}
	if rep.Held[0].Scope != Interop || rep.Held[0].RunNow {
		t.Errorf("held step = %+v, want interop RunNow=false", rep.Held[0])
	}
}

func TestReportText(t *testing.T) {
	a := &fakeMigration{id: "a", version: 5, scope: HostLocal, applies: true}
	b := &fakeMigration{id: "b", version: 6, scope: Interop, applies: true}
	plan, _ := NewPlanner(a, b).Plan(context.Background(), 4, 6)

	text := plan.Report().String()
	for _, want := range []string{
		"Plan: protocol 4 -> 6",
		"Run now (host-local)",
		"- a [host-local] a",
		"Held for finalize (interop)",
		"- b [interop] b",
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
	if len(rep.RunNow) != 0 || len(rep.Held) != 0 {
		t.Fatalf("empty plan report has steps: %+v", rep)
	}
	text := rep.String()
	if !strings.Contains(text, "(nothing to do)") {
		t.Errorf("empty plan text = %q, want it to note nothing to do", text)
	}
}

func TestReportOnlyHeldGroup(t *testing.T) {
	b := &fakeMigration{id: "b", version: 6, scope: Interop, applies: true}
	plan, _ := NewPlanner(b).Plan(context.Background(), 5, 6)
	text := plan.Report().String()
	if !strings.Contains(text, "Run now (host-local)") || !strings.Contains(text, "(none)") {
		t.Errorf("want run-now group rendered as (none):\n%s", text)
	}
}

func TestReportJSONRoundTrip(t *testing.T) {
	a := &fakeMigration{id: "a", version: 5, scope: HostLocal, applies: true}
	b := &fakeMigration{id: "b", version: 6, scope: Interop, applies: true}
	plan, _ := NewPlanner(a, b).Plan(context.Background(), 4, 6)

	data, err := json.Marshal(plan.Report())
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var back PlanReport
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if back.From != 4 || back.To != 6 || len(back.RunNow) != 1 || len(back.Held) != 1 {
		t.Errorf("round-tripped report = %+v", back)
	}
	if back.RunNow[0].ID != "a" || back.Held[0].ID != "b" {
		t.Errorf("round-tripped ids wrong: %+v", back)
	}
}
