# AGENTS.md

Guidance for AI coding agents working in this repository.

## Project overview

This repository is a **fork of [tetratelabs/wazero](https://github.com/tetratelabs/wazero)**
with the Go module path renamed to `github.com/pschlump/wazero` so the fork can
be compiled and tested independently. All internal imports have been rewritten
to the `github.com/pschlump/wazero` path. Some docs, Makefile variables, badges
and links still point at the upstream `tetratelabs/wazero` — those are upstream
leftovers, not the current module path.

wazero itself is a **zero-dependency WebAssembly runtime written in Go**. It
executes WebAssembly (Wasm) modules (`%.wasm` binaries) and is compliant with
the WebAssembly Core Specification 1.0 and 2.0. It does **not** use CGO, which
keeps cross-compilation working for any Go target. The only module dependency
is `golang.org/x/sys`.

Key facts:

- Module: `github.com/pschlump/wazero` (see `go.mod`)
- Floor Go version: the one declared in `go.mod` (wazero supports the two most
  recent Go releases, matching Go's release policy). CI builds with the Go
  version pinned in `.github/workflows/commit.yaml` (`GO_VERSION` env).
- License: Apache 2.0 (`LICENSE`)
- The design decisions behind the runtime are documented at length in
  `RATIONALE.md` and `internal/engine/RATIONALE.md`.

## Runtime architecture

There are two execution engines, selected via `RuntimeConfig`:

- **Interpreter** (`internal/engine/interpreter`): naive Wasm VM with no
  platform-specific code; runs on every GOOS/GOARCH Go supports. Selected with
  `wazero.NewRuntimeConfigInterpreter()`.
- **Compiler (wazevo)** (`internal/engine/wazevo`): compiles Wasm to native
  machine code ahead of time during `Runtime.CompileModule`. Much faster, but
  only supports amd64 and arm64. This is the default where supported. It has
  its own SSA pipeline (`internal/engine/wazevo/ssa`), frontend, and per-ISA
  backends, plus a few hand-written assembly entrypoints (`*.s` files).

Platform-specific code (mmap, syscalls, time) lives in `internal/platform`,
`internal/sysfs` and `internal/sys` and must always fall back gracefully on
unsupported platforms so the interpreter keeps working everywhere. `make check`
verifies this by cross-compiling to plan9, js/wasm, wasip1, aix, s390x,
ppc64le, arm, 386 and freebsd.

## Repository layout

- `*.go` (root package `wazero`): the main public API — `runtime.go`
  (`Runtime`), `config.go` (`RuntimeConfig`, `ModuleConfig`), `builder.go`
  (`HostModuleBuilder`), `cache.go` (compilation cache), `fsconfig.go`
  (`FSConfig`).
- `api/`: public API interfaces and types shared by the runtime
  (`api.Module`, `api.Function`, features in `features.go`).
- `sys/`: public system error types (`sys.ExitError` etc.).
- `experimental/`: public APIs that are **not** covered by the stability
  promise and may change: `experimental/sys` (filesystem/syscall layer),
  `experimental/sysfs`, `experimental/sock`, `experimental/table`,
  `experimental/logging`, `experimental/wazerotest` (fake runtime for testing
  code that embeds wazero), listeners, checkpoint, etc.
- `imports/`: host-module implementations for language toolchains:
  `wasi_snapshot_preview1` (WASI), `assemblyscript`, `emscripten`.
- `internal/`: all implementation details (not importable by users). Notable
  packages:
  - `internal/wasm`: Wasm binary decoding/validation, module/store/memory/table
    model (`internal/wasm/binary` is the binary format codec).
  - `internal/engine/interpreter`, `internal/engine/wazevo`: the two engines.
  - `internal/wasip1`: WASI snapshot_preview1 internals.
  - `internal/platform`, `internal/sys`, `internal/sysfs`, `internal/sock`:
    OS abstraction layers.
  - `internal/testing/require`: the in-house assertion library used by tests.
  - `internal/testing/{binaryencoding,dwarftestdata,fs,hammer,maintester,nodiff,proxy}`:
    test support packages.
  - `internal/integration_test/`: heavyweight tests — `spectest` (WebAssembly
    spec conformance suites: v1, v2, threads, tail-call, extended-const,
    exception-handling, typed-function-references), `bench`, `fuzz` (Rust
    cargo-fuzz harnesses), `filecache`, `libsodium`, `stdlibs`.
  - `internal/version`: CLI version stamping (via ldflags).
- `cmd/wazero/`: the `wazero` CLI that runs standalone Wasm binaries
  (`wazero run app.wasm`).
- `examples/`: embeddable examples (basic, allocation, cli, import-go, …).
- `testdata/`, `*/testdata/`: pre-built `.wasm`/`.wat` fixtures. These are
  checked in; you do **not** need tinygo/zig/cargo/emscripten/npm to build or
  test the project — those toolchains are only needed to *regenerate* fixtures
  via the corresponding `build.examples.*` make targets.
- `site/`: Hugo-based documentation site (deployed by Netlify, `netlify.toml`).
- `packaging/`: MSI installer assets for the CLI.

## Build and test commands

Everything is driven by the root `Makefile`:

- `go build ./...` — build all packages (fast sanity check).
- `make test` — run the full unit test suite (`go test ./...` plus the version
  testdata module and the fuzz wazerolib). A quick `go test ./...` also works.
- `make test.examples` — run tests under `examples/` and `imports/` examples.
- `make check` — **pre-flight check for pull requests**: cross-compile checks
  for many GOOS/GOARCH pairs, `make lint` (golangci-lint, run for both arm64
  and amd64), `make format`, `go mod tidy`, then fails if the working tree is
  dirty. Run this before considering a change done.
- `make format` — formats code with `gofumpt`, `gosimports` and `asmfmt`.
  Note: the Makefile passes `-local github.com/tetratelabs/` to gosimports and
  uses a tetratelabs ldflags path for CLI version stamping; both are upstream
  leftovers from the fork rename (actual module is `github.com/pschlump/wazero`).
- `make lint` — runs golangci-lint (installed on demand via `go install`).
- `make coverage` — coverage report for the main packages.
- `make build.spectest` — downloads the WebAssembly spec test suites and
  regenerates the spectest JSON fixtures (requires `wast2json` from wabt,
  `wasm-tools`, `jq`, `curl`, `perl`; normally only run in CI).
- `make fuzz` — differential fuzzing harness (requires Rust + cargo-fuzz).
- `make dist VERSION=x.y.z` — cross-builds the `wazero` CLI archives
  (darwin/linux/windows, amd64/arm64) under `dist/`.
- `make site` — serves the documentation site locally with Hugo.
- `make clean` — removes build artifacts and the test cache.

## Code style guidelines

- Standard Go with `gofumpt` formatting (stricter than gofmt); run
  `make format` before committing. `.editorconfig`: UTF-8, LF, final newline,
  no trailing whitespace.
- Exported identifiers in public packages (`wazero`, `api`, `sys`,
  `experimental`, `imports`) must have doc comments — the API is consumed via
  pkg.go.dev, and comments follow Go doc conventions with `# Notes` sections.
- The public API has a semantic-versioning stability promise: never break an
  exported signature outside a major version. Anything unstable belongs in
  `experimental/`. Implementation details belong in `internal/`.
- Errors in the syscall/filesystem layers are modelled as `sys.Errno`-style
  values; public-facing file APIs mirror Go's `io/fs` idioms.
- Assembly files (`*.s`) are formatted with `asmfmt`.
- Commits require a DCO sign-off (`git commit -s`), see `CONTRIBUTING.md`.
- Pull request conventions (from `CONTRIBUTING.md`): title describes the
  change without issue numbers; address review comments with additional
  commits (no squashing during review); maintainers squash on merge.

## Testing instructions

- Tests are standard Go table-driven tests, colocated with the code
  (`foo_test.go` next to `foo.go`), using the internal assertion library
  `internal/testing/require` instead of testify.
- Example functions (`Example*` in `*_example_test.go`) double as
  documentation and are enforced by the `testableexamples` linter.
- Spec conformance: `internal/integration_test/spectest` runs the official
  WebAssembly spec test suites against both engines — this is the primary
  correctness gate for engine changes.
- Engine-specific tests: `internal/engine/interpreter` and
  `internal/engine/wazevo` (including `e2e_test.go` and per-ISA backend
  tests). `internal/testing/nodiff` compares interpreter vs compiler results.
- Fuzzing lives in `internal/integration_test/fuzz` (Rust cargo-fuzz targets
  that drive the Go library) and `internal/integration_test/fuzzcases`
  (checked-in crash reproducers).
- When changing platform-specific code, make sure the relevant cross-compile
  targets in `make check` still build.
- Tests default to `-timeout 300s` (set via `go_test_options` in the
  Makefile). Full `make test` covers `./...` plus two extra modules.

## CI and release

- GitHub Actions in `.github/workflows/`: `commit.yaml` (main gate: installs
  wabt/wasm-tools, runs `make build.spectest` then `make check`, plus the test
  matrix across OS/arch/Go versions), `examples.yaml`, `integration.yaml`
  (spectest, fuzz, stdlibs on multiple platforms including BSD VMs),
  `release.yaml` (CLI release built with `make dist`), `clear_cache.yaml`.
- The documentation site (`site/`) is built with Hugo and deployed by Netlify
  (`netlify.toml`); the Hugo version there must stay in sync with the Makefile.
- Go module releases follow semver; beta tags listed in `go.mod` are retracted.

## Security considerations

- wazero executes untrusted Wasm inside a sandboxed VM with no CGO and no
  host dependencies; guest code can only touch host resources explicitly
  granted through `ModuleConfig`/`FSConfig` (stdin/stdout, env, args, mounted
  filesystems, sockets via `experimental/sock`).
- When adding host functionality, keep the sandbox default-deny: nothing
  should be accessible unless the embedding application opts in.
- Compiler-engine changes deal with generated machine code and memory bounds
  checks — treat bounds-check elimination or mmap changes as security
  sensitive and cover them with spectest + fuzz runs.
- Do not introduce new module dependencies: zero-dependency (besides
  `golang.org/x/sys`) is a core project guarantee, verified in CI by testing
  in Docker's scratch image.
