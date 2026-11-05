package migration

import (
	"fmt"
	"strings"
)

// StepReport is the previewable form of one planned step: the fields an operator
// needs before running it (migration-spec §2).
type StepReport struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Scope       Scope  `json:"scope"`
	RunNow      bool   `json:"runNow"`
	Reason      string `json:"reason"`
}

// PlanReport is the structured, machine-renderable preview of a Plan: a from→to
// activation step with its steps split into those that run now (local) and
// those run-at-finalize for a coordinated finalize (shared). It marshals to JSON directly
// and renders to text via String.
type PlanReport struct {
	From          int          `json:"from"`
	To            int          `json:"to"`
	RunNow        []StepReport `json:"runNow"`
	RunAtFinalize []StepReport `json:"runAtFinalize"`
}

// Report renders a Plan as a structured PlanReport for machine output (e.g. JSON)
// or further formatting. Steps keep their plan order within each group; an empty
// plan yields empty groups.
func (pl Plan) Report() PlanReport {
	rep := PlanReport{From: pl.From, To: pl.To}
	for _, s := range pl.Steps {
		sr := StepReport{
			ID:          s.Migration.ID(),
			Description: s.Migration.Description(),
			Scope:       s.Migration.Scope(),
			RunNow:      s.RunNow,
			Reason:      s.Reason,
		}
		if s.RunNow {
			rep.RunNow = append(rep.RunNow, sr)
		} else {
			rep.RunAtFinalize = append(rep.RunAtFinalize, sr)
		}
	}
	return rep
}

// String renders the report as a human-readable --plan / --dry-run summary,
// grouping run-now and run-at-finalize steps. An empty plan renders as "(nothing to do)".
func (r PlanReport) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Plan: activation %d -> %d\n", r.From, r.To)
	if len(r.RunNow) == 0 && len(r.RunAtFinalize) == 0 {
		b.WriteString("  (nothing to do)\n")
		return b.String()
	}
	writeGroup(&b, "Run now (local)", r.RunNow)
	writeGroup(&b, "Run at finalize (shared)", r.RunAtFinalize)
	return b.String()
}

// writeGroup renders a titled group of steps, or "(none)" when empty.
func writeGroup(b *strings.Builder, title string, steps []StepReport) {
	fmt.Fprintf(b, "  %s:\n", title)
	if len(steps) == 0 {
		b.WriteString("    (none)\n")
		return
	}
	for _, s := range steps {
		fmt.Fprintf(b, "    - %s [%s] %s\n", s.ID, s.Scope, s.Description)
		if s.Reason != "" {
			fmt.Fprintf(b, "        reason: %s\n", s.Reason)
		}
	}
}
