# gatekit

`gatekit` is a version-keyed feature-gate library for Go. It lets you merge
partial or not-yet-finished work behind a gate that is **off**, ship releases with
the new code path dormant (running the old path), and flip it on later by
advancing a single monotonic version — so trunk-based development and rolling
upgrades never force a disrupted release cycle or a long-lived branch.

`gatekit` is deliberately **in-binary and deterministic**: feature defaults are
declared at compile time and resolved against a version, not fetched from a remote
flag service. It is the standalone analogue of Kubernetes'
`component-base/featuregate`.

The version gatekit keys on is the **activation version** — a standalone counter
that advances only on a gated behavior change, independent of the release version.
This is the same decoupled-counter idea as CockroachDB's cluster version,
Kubernetes' emulation version (KEP-4330), and Kafka's inter-broker protocol
version; a consumer maps the activation version onto whichever counter is
authoritative for them.

## Two layers

Take only what you need:

- **Gate core** (`github.com/automa-saga/gatekit`) — a `Feature` registry of
  `FeatureDef`s (an optional `Description` plus `VersionedSpecs`) and a
  `FeatureGate` that resolves each feature against a pluggable `VersionSource`.
  Dependency-light; usable by any Go project.
- **Migration / activation** (`github.com/automa-saga/gatekit/migration`) —
  deferred activation, gated one-time migrations, and a `finalize` step for
  rolling upgrades. Opt-in.

`gatekit` never decides *who owns* the current version — that is the
`VersionSource`'s job — so a consumer whose version is set externally can reuse
the gate core without inheriting the activation model.

## Why not `k8s.io/component-base/featuregate`?

If you are building a Kubernetes component that already imports
apimachinery/component-base, use it — it is battle-tested and integrates with
Kubernetes' version skew, storage versions, and `--feature-gates`. gatekit exists
for everything else:

- **Dependency-light.** The gate core has no external dependencies; component-base
  pulls in a large part of the Kubernetes module tree and pins you to its release
  cadence.
- **Release-independent version.** gatekit keys on a standalone monotonic
  activation version that moves only on a behavior change — not on a Kubernetes
  release minor, which component-base's emulation version is tied to.
- **Configurable skew window.** N/N+1 by default, widenable per consumer;
  component-base fixes roughly three minors.
- **Pluggable version source.** The active version can be owned externally (e.g. a
  CRD field another component sets), not only a flag the component owns.
- **Migrations included.** An opt-in layer for deferred activation, gated one-time
  migrations, finalize, and reversibility. component-base is gates only; Kubernetes
  does migration/conversion with separate machinery.
- **Multi-language, spec-first.** A language-neutral
  [specification](https://github.com/automa-saga/gatekit-spec) + conformance
  fixtures so non-Go implementations interoperate; component-base is Go-only and
  Kubernetes-internal.

gatekit is newer and less battle-tested than component-base — the trade for
portability, the migration layer, and cross-language conformance.

## Gate core

```go
// Declare each feature once, as a typed constant — a single source of truth
// referenced by both the registry and the call sites (no string drift).
const SplitCatalog gatekit.Feature = "SplitCatalog"

specs := gatekit.Registry{
    SplitCatalog: {
        Description: "Load the per-minor split catalog instead of the single file",
        Specs: gatekit.VersionedSpecs{
            {ActiveFrom: 5, Default: false, Maturity: gatekit.Beta},
            {ActiveFrom: 6, Default: true,  Maturity: gatekit.GA},
        },
    },
}

gate, err := gatekit.New(specs, gatekit.StaticVersion(6))
if err != nil {
    // handle configuration error
}

if gate.Enabled(SplitCatalog) {
    loadSplitCatalog() // new path
} else {
    loadLegacyCatalog() // old path
}
```

A feature's state is the `Default` of the latest spec whose `ActiveFrom` is `<=`
the current activation version. Unknown or not-yet-reached features are off.
`gate.MaxActivation()` derives the newest activation version in the registry, so
consumers never hand-maintain it.

Each feature may carry an optional, free-form `Description`. It is informational —
it never affects resolution or the run invariant — and surfaces in operator-facing
output: retrieve one with `gate.Description(feature)`, or list them all with
`gate.Report()` (a dry-run of every feature's state, maturity, and description).

Wire `VersionSource` to whatever is authoritative for you — a static value, a
tool-decided active version (see the `migration` package), or a value set
externally (for example a CRD field).

### Environment-variable overrides

`WithEnvOverrides` lets operators force features on or off from the environment
without each consumer reimplementing the scan and the name conversion. The
**prefix stays yours** — gatekit is general-purpose and bakes in no
consumer-specific prefix — while gatekit owns the **suffix convention**, so every
consumer, and every language port, derives the same variable name for a feature.

The suffix is the `Feature` identifier converted from CamelCase to
`UPPER_SNAKE_CASE`: `EnvVarName(SplitCatalog, "MYAPP_ENABLE_")` yields
`MYAPP_ENABLE_SPLIT_CATALOG`.

```go
gate, err := gatekit.New(specs, source,
    gatekit.WithEnvOverrides("MYAPP_ENABLE_"),
)
```

With that option, `MYAPP_ENABLE_SPLIT_CATALOG=true` (or `false`, or any value
`strconv.ParseBool` accepts) applies as a per-feature **override** — it beats
version/maturity resolution exactly like `WithOverrides`, and forcing a feature on
before its `ActiveFrom` is allowed (the override intentionally bypasses the version
gate).

`WithEnvOverrides` **fails loudly** rather than silently ignore a mistyped opt-in:
`New` returns an error, naming the offending variable, if a `<prefix>*` variable
does not map to any registered feature or its value is not a parseable bool. Option
order is last-writer, so an explicit `WithOverrides` passed after `WithEnvOverrides`
still wins per feature.

### Removal lint

A gate that is never deleted is a bug. `gatekit.Lint` enforces the contract step: when
a version leaves the support window, the specs and code paths it guarded must go.

```go
violations, err := gatekit.Lint(registry, 1)
```

It is a pure function of the registry and the window, so run it from a `go test`.
Each `Violation` has a `Kind`:

- `StaleSpec` — a later spec has `ActiveFrom <= Min`, so this spec can never be selected.
- `PastRemoveBy` — the feature sets `FeatureDef.RemoveBy` and `Min >= RemoveBy`.
- `DeprecatedDue` — the latest spec is `Deprecated`.

It does not scan source code; deleting the dead branches is up to you.

#### Run it in your CI

gatekit never calls `Lint` itself, so nothing runs unless you add it. Put it in a unit
test; your normal `go test` CI job then picks it up with no extra tooling:

```go
func TestRegistryLifecycle(t *testing.T) {
	// Use the same window your binary passes to gatekit.New, because Min depends on it.
	violations, err := gatekit.Lint(registry, 1)
	if err != nil {
		t.Fatal(err) // the registry itself is invalid
	}
	for _, v := range violations {
		if v.Kind == gatekit.StaleSpec {
			t.Errorf("%s: %s (spec %d)", v.Feature, v.Kind, v.ActiveFrom)
		} else {
			t.Errorf("%s: %s", v.Feature, v.Kind)
		}
	}
}
```

Notes:

- **When it fails.** Adding a spec at a higher activation version raises `Max` and
  therefore `Min`, so specs that fall out of the window start failing. Delete those
  specs and the code paths they guarded in the same change.
- **Not at startup.** Lint is about code hygiene, not about the running fleet. Use
  `gate.Check()` at startup and keep `Lint` in tests.
- **Deprecated features.** `DeprecatedDue` fires on every run until the feature is
  deleted. If you only want to enforce the window, skip that kind in the loop.
- **Dead code.** Lint reports the spec, not the branch. Search for the feature
  identifier to find what to delete.

## Migration / activation

The `migration` package ties one-time, idempotent migrations to activation
versions. **Local** migrations may run as soon as their version activates;
**shared** migrations are deferred to a coordinated `finalize`, so a staggered
rollout never leaves a mixed-activation fleet. `finalize` advances the active
version — the one-way line, since migrations may be irreversible.

```go
plan, err := migration.NewPlanner(migs...).Plan(ctx, store.Current(), target)
// plan.RunNow()        — local, run at install
// plan.RunAtFinalize() — shared, run at finalize

runner := migration.NewRunner()
_ = runner.RunNow(ctx, plan)          // at install
_ = runner.Finalize(ctx, plan, store) // during a controlled window
```

### Rollback and the write floor

Two optional interfaces describe a migration's limits. Both default to the safe
answer, so existing migrations keep working.

- `Reversible() bool` — can this migration be undone with `Rollback`? Defaults to
  `false`. It is never guessed from whether `Rollback` exists.
- `ReadableFrom() int` — the oldest version that can still read the data this
  migration writes. Defaults to `0`.

**Write floor.** `Planner.WithMin(min)` sets the lowest version you still allow
rolling back to (the min-compat version). A step is `Blocked` when its
`ReadableFrom` is above `min`, because running it would leave data that version
`min` cannot read. `RunNow` and `Finalize` check this first and run nothing if
any step is blocked. If a migration in range declares a `ReadableFrom` above `0`
but you never call `WithMin`, the floor would be silently ignored, so `Plan`
refuses with an `*UnenforcedFloorError` — set the min, or drop `ReadableFrom` if
the data really is readable by every version. Migrations that declare no floor
need no `WithMin`.

This is why a format change is split into expand and contract: a change that old
versions cannot read cannot finalize in the release that introduces it. You ship
the readable form first, raise the floor later, and only then finalize.

**Lowering.** `Planner.PlanLower(from, to, min)` builds the reverse plan. It
rejects a target below `min`, ignores `Applies`, and orders steps from newest to
oldest. Any irreversible step is refused, and all of them are named.
`StdRunner.Lower` rolls each step back in order, then sets the active version
last. If a step fails, it stops and leaves the active version unchanged.

```go
planner := migration.NewPlanner(migs...).WithMin(4)

lp, err := planner.PlanLower(6, 4, 4) // from 6 down to 4
if err != nil {
    var irr *migration.IrreversibleStepError
    if errors.As(err, &irr) { /* these steps cannot be undone */ }
    return err
}
_ = runner.Lower(ctx, lp, store) // data first, active version last
```

Typed errors: `BelowMinError`, `IrreversibleStepError`, `WriteFloorError`.
Callers with their own runner can call `plan.CheckWriteFloor(runNow)` directly.

## Specification & conformance

gatekit's behavior is defined by a language-neutral specification in
[automa-saga/gatekit-spec](https://github.com/automa-saga/gatekit-spec), vendored
here as a git submodule at [`docs/spec`](docs/spec) pinned to a **tagged
gatekit-spec release** (this version conforms to gatekit-spec `v0.3.0`). This Go
package is the **first conformant implementation**; where the code and the spec
disagree, the spec wins.

### Version compatibility

| gatekit | gatekit-spec | Works with gatekit-automa |
| --- | --- | --- |
| v0.1.0 | v0.1.0 | — |
| v0.2.0 | v0.1.0 | v0.1.0 |
| v0.3.0 | v0.1.0 | — |
| v0.4.0 | v0.2.0 | v0.2.0 |
| v0.5.0 | untagged commit `c4c7235` (see note) | PR #4 (not released yet) |
| unreleased (this version) | v0.3.0 | needs a follow-up (see note) |

Notes:

- v0.5.0 pinned a spec commit between v0.2.0 and v0.3.0. It has the removal lint
  (spec §8, also in v0.3.0) but not rollback or the write floor.
- gatekit-automa does not call `plan.CheckWriteFloor` yet, so it does not fully
  honor the write floor until it is updated.

The `docs/spec` submodule is always pinned to a tagged spec release; update this
table in the same PR that changes the pin.

The conformance runner (`conformance_test.go`) executes the spec's shared fixtures
against this implementation. After cloning, fetch the submodule:

```bash
git submodule update --init
```

The resolution tests skip automatically if the submodule is not checked out.

## Development

```bash
task build        # compile
task test:unit    # race + coverage, writes unit-test-report.md
task lint:check   # gofmt check
```

## License

Apache License 2.0 — see [LICENSE](LICENSE).
