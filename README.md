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

## Two layers

Take only what you need:

- **Gate core** (`github.com/automa-saga/gatekit`) — a `Feature` registry of
  `VersionedSpecs` and a `FeatureGate` that resolves each feature against a
  pluggable `VersionSource`. Dependency-light; usable by any Go project.
- **Migration / activation** (`github.com/automa-saga/gatekit/migration`) —
  deferred activation, gated one-time migrations, and a `finalize` step for
  rolling upgrades. Opt-in.

`gatekit` never decides *who owns* the current version — that is the
`VersionSource`'s job — so a consumer whose version is set externally can reuse
the gate core without inheriting the activation model.

## Gate core

```go
specs := gatekit.Registry{
    "SplitCatalog": {
        {Version: 5, Default: false, PreRelease: gatekit.Beta},
        {Version: 6, Default: true,  PreRelease: gatekit.GA},
    },
}

gate, err := gatekit.New(specs, gatekit.StaticVersion(6))
if err != nil {
    // handle configuration error
}

if gate.Enabled("SplitCatalog") {
    loadSplitCatalog() // new path
} else {
    loadLegacyCatalog() // old path
}
```

A feature's state is the `Default` of the latest spec whose `Version` is `<=` the
current version. Unknown or not-yet-reached features are off. `gate.MaxVersion()`
derives the newest protocol version in the registry, so consumers never
hand-maintain it.

Wire `VersionSource` to whatever is authoritative for you — a static value, a
tool-decided active version (see the `migration` package), or a value set
externally (for example a CRD field).

## Migration / activation

The `migration` package ties one-time, idempotent migrations to protocol
versions. **Host-local** migrations may run as soon as their version activates;
**interop** migrations are held for a coordinated `finalize`, so a staggered
rollout never leaves a mixed-protocol fleet. `finalize` advances the active
version — the one-way line, since migrations may be irreversible.

```go
plan, err := migration.NewPlanner(migs...).Plan(ctx, store.Current(), target)
// plan.RunNow() — host-local, run at install
// plan.Held()   — interop, run at finalize

runner := migration.NewRunner()
_ = runner.RunNow(ctx, plan)          // at install
_ = runner.Finalize(ctx, plan, store) // during a controlled window
```

## Development

```bash
task build        # compile
task test:unit    # race + coverage, writes unit-test-report.md
task lint:check   # gofmt check
```

## License

Apache License 2.0 — see [LICENSE](LICENSE).
