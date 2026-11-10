package gatekit_test

import (
	"fmt"

	"github.com/automa-saga/gatekit"
)

// A consumer runs Lint from its own test so CI fails when a spec falls out of
// the support window.
func ExampleLint() {
	registry := gatekit.Registry{
		"SplitCatalog": {Specs: gatekit.VersionedSpecs{
			{ActiveFrom: 5, Default: false, Maturity: gatekit.Beta},
			{ActiveFrom: 6, Default: true, Maturity: gatekit.GA},
		}},
		"NewIndexer": {Specs: gatekit.VersionedSpecs{
			{ActiveFrom: 7, Default: false, Maturity: gatekit.Alpha},
		}},
	}

	// Max is 7, so with W=1 Min is 6: SplitCatalog's spec 5 is superseded at 6.
	violations, err := gatekit.Lint(registry, 1)
	if err != nil {
		panic(err)
	}
	for _, v := range violations {
		fmt.Println(v.Feature, v.Kind, v.ActiveFrom)
	}
	// Output: SplitCatalog stale-spec 5
}
