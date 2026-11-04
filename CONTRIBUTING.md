# Contributing to gatekit

Thanks for your interest in contributing! Bug reports, documentation fixes, and
code contributions are all welcome.

This document explains how to set up your environment, the standards a change is
expected to meet, and how the pull-request process works.

## Ways to contribute

- **Report a bug** — open an issue with a minimal reproduction (a small `main.go`
  or a failing test) and the observed vs. expected behavior.
- **Suggest a feature** — open an issue describing the use case first. Because
  gatekit follows a [normative spec](https://github.com/automa-saga/gatekit-spec),
  behavioral changes are discussed against the spec before implementation (see
  [Specs & conformance](#specs--conformance)).
- **Improve docs** — the README and GoDoc comments are fair game.
- **Submit code** — see the workflow below.

## Prerequisites

- **Go 1.27+** (see [`go.mod`](go.mod)).
- **[Task](https://taskfile.dev/installation/)** — the task runner used for all
  common commands ([`Taskfile.yaml`](Taskfile.yaml)).

## Getting started

```sh
# Fork and clone your fork, then:
git clone https://github.com/<your-username>/gatekit.git
cd gatekit

# Fetch the gatekit-spec submodule (docs/spec) so the conformance tests run.
git submodule update --init

# Build, lint, and test to confirm a clean baseline.
task build
task lint:check
task test:unit
```

Common tasks:

| Command | What it does |
| --- | --- |
| `task build` | `go build -v ./...` |
| `task lint` | Format the code (`go fmt ./...`) |
| `task lint:check` | Fail if any file is not gofmt-clean (CI runs this) |
| `task test:unit` | Run the unit tests (`-race -cover`) and write `unit-test-report.md` |
| `task test:coverage` | Run tests with coverage gating ([`.testcoverage.yml`](.testcoverage.yml)) |
| `task clean` | Remove test artifacts and clear the Go build/test cache |

The conformance tests (`conformance_test.go`, `migration/conformance_test.go`)
skip automatically when the `docs/spec` submodule is not checked out.

## Making a change

1. Create a branch off `main`.
2. Make your change, keeping commits focused.
3. Add or update tests. Bug fixes should come with a test that fails before the
   fix and passes after; new behavior needs coverage.
4. Run `task lint:check` and `task test:unit` locally — CI runs the same checks.
5. Update documentation (README, GoDoc) when behavior or the public API changes.

### Definition of done

- [ ] `task build` succeeds.
- [ ] `task lint:check` is clean.
- [ ] `task test:unit` passes, including new/updated tests.
- [ ] `task test:coverage` meets the thresholds.
- [ ] Public behavior changes are reflected in the spec and conformance fixtures
      (see below).

## Specs & conformance

gatekit is **spec-first**. The behavior is defined by the language-neutral
specification in [automa-saga/gatekit-spec](https://github.com/automa-saga/gatekit-spec),
vendored here as a git submodule at [`docs/spec`](docs/spec), and pinned by
cross-implementation conformance fixtures. This Go package is the first conformant
implementation; **where the code and the spec disagree, the spec wins.**

If your change alters observable behavior (feature resolution, the run invariant,
plan computation), it must also update the relevant spec section and conformance
fixtures. Spec changes land in the gatekit-spec repository first; then the
`docs/spec` submodule pointer is bumped here in the same change that implements
them.

## Commits & sign-off

- **DCO sign-off is required.** Every commit must be signed off, certifying the
  [Developer Certificate of Origin](https://developercertificate.org/). Add the
  `Signed-off-by` trailer with `git commit -s`.
- **Commit messages** follow the
  [Conventional Commits](https://www.conventionalcommits.org/) format.

## Pull requests

- Open your PR against `main`.
- PRs are **squash-merged**, and the **PR title becomes the squashed commit
  message** — so the PR title **must** follow Conventional Commits (e.g.
  `feat: add support window option`, `fix: correct min clamp`). This drives
  automated release versioning.
- Keep PRs focused; smaller, self-contained changes are reviewed faster.
- Fill in what changed and why, and link any related issue.
- Ensure CI is green before requesting review.

## License

By contributing, you agree that your contributions will be licensed under the
project's [Apache License 2.0](LICENSE).
