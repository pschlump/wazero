# wazero VM Initialization Time — Notes

Context: investigating a 45–50ms VM setup cost seen in a gopher-lua project,
and comparing against what wazero (this fork, `github.com/pschlump/wazero`)
can achieve for VM initialization.

All numbers below were measured on this machine (Apple Silicon, Go 1.25)
against this repo. Benchmark source: `note/benchmark-init/` (`main.go` + `go.mod`;
run `go run .` in that directory — change `wasmPath` in `main.go` to test
other modules; the file cache is written to `/tmp/wazerobench/filecache`).

## Measured results

Test module: `imports/wasi_snapshot_preview1/testdata/zig-cc/wasi.wasm`
(786 KB real-world WASI binary).

| Scenario                                               | Time        |
|--------------------------------------------------------|-------------|
| wazevo (compiler): cold compile, no cache              | **10.1 ms** |
| wazevo: compile with file cache (hit)                  | **0.8 ms**  |
| interpreter: cold "compile" (decode/validate only)     | **1.6 ms**  |
| instantiate a pre-compiled module                      | ~4 ms*      |
| full VM: runtime + WASI + cached compile + instantiate | **~5 ms**   |

\* The instantiate number is inflated because this module's `_start` runs
guest code at instantiation; pure instantiation is sub-millisecond.

## 1) Can compiled native code be saved and reused?

Yes — two mechanisms, both in `cache.go`:

- **`wazero.NewCompilationCache()`** — in-memory cache, shareable across
  multiple `Runtime` instances in one process (`cache.go:40`).
- **`wazero.NewCompilationCacheWithDir(dir)`** — persists compiled native
  code to disk, surviving process restarts (`cache.go:56`). Entries are keyed
  by module hash and namespaced by wazero version + GOOS/GOARCH
  (`cache.go:102`); storage is `internal/filecache`.

Measured effect: **10.1ms → 0.8ms** (~13x) for the 786KB module.

Caveats:

- The embedder must protect the cache directory from external modification.
- Concurrent first-compiles of the same module don't dedupe — compile once
  centrally and share the resulting `CompiledModule`.
- A single `CompiledModule` can be instantiated many times, concurrently,
  across goroutines — "compile once, instantiate per VM" is the standard
  pattern.

## 2) Is the performance problem the compile/optimize step?

For wazero's compiler engine, yes: cold AOT compilation dominates setup cost
(~10ms vs ~1ms for everything else on a 0.75MB module). Decode/validate and
instantiation are minor.

For the gopher-lua 45–50ms case: gopher-lua is a pure-Go interpreter with no
native compilation, so that cost comes from somewhere else (source
lexing/parsing, stdlib registration, allocation churn, ...). A full wazero VM
with a warm cache is ~5ms for a large module, so 45–50ms of gopher-lua setup
is abnormal — take a `pprof` CPU profile of the setup path before assuming it
is inherent.

## 3) Is the interpreter faster to load? By how much?

Yes: **1.6ms vs 10.1ms cold — about 6x faster to load** for this module. The
gap grows with module size: wazevo's SSA/optimization passes scale with code
size, while the interpreter only decodes and validates.

Trade-off: interpreter execution is roughly an order of magnitude slower than
compiled code. Rule of thumb:

- Many short-lived VMs running little code → interpreter wins on total wall
  time.
- VMs running any substantial workload → compiler + cache wins; the 10ms
  compile amortizes to nothing.

## 4) Other approaches to cut VM setup time

- **Compile once at process start, share the `CompiledModule`** — per-VM cost
  drops to instantiation only (sub-millisecond).
- **VM pooling** — pre-instantiate a pool of modules at startup, hand them
  out, reset state between uses instead of tearing down.
- **Snapshots** — `experimental/checkpoint.go` lets a host function capture
  execution state and restore it later. Warm a VM to a "ready" state,
  snapshot, then restore instead of redoing guest-side initialization.
- **Shrink the module** — large name/debug custom sections (the 786KB zig
  binary carries plenty). `wasm-opt -O3 --strip-debug` cuts decode+validate
  time for both engines and shrinks the file cache.
- **Compile in the background** — kick off `CompileModule` in a goroutine at
  startup; the cache is warm by the time the first request arrives.
- **gopher-lua specifics** — no JIT there, so 45–50ms is almost certainly
  repeated parse/init work: cache precompiled chunks (`lua.Compile` + dump),
  or pool `LState`s instead of rebuilding them.

## Bottom line

With `NewCompilationCacheWithDir` + a shared `CompiledModule`, wazero VM
setup lands in the ~1–5ms range even for large modules — an order of
magnitude below the 45–50ms seen in the gopher-lua project.
