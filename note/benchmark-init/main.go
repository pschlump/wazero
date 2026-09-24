package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pschlump/wazero"
	"github.com/pschlump/wazero/imports/wasi_snapshot_preview1"
	"github.com/pschlump/wazero/sys"
)

const wasmPath = "/Users/philip/go/src/github.com/pschlump/wazero/imports/wasi_snapshot_preview1/testdata/zig-cc/wasi.wasm"

func timeit(label string, n int, f func()) {
	// warmup once
	f()
	best := time.Duration(1<<63 - 1)
	var tot time.Duration
	for i := 0; i < n; i++ {
		t0 := time.Now()
		f()
		d := time.Since(t0)
		tot += d
		if d < best {
			best = d
		}
	}
	fmt.Printf("%-58s best %8.3fms  avg %8.3fms\n", label, float64(best)/1e6, float64(tot)/1e6/float64(n))
}

func main() {
	ctx := context.Background()
	bin, err := os.ReadFile(wasmPath)
	if err != nil {
		panic(err)
	}
	fmt.Printf("module: %s (%d bytes)\n\n", wasmPath, len(bin))

	cacheDir := "/tmp/wazerobench/filecache"
	os.MkdirAll(cacheDir, 0o755)

	// --- 1. Compiler engine (wazevo), cold, no cache ---
	timeit("wazevo: new runtime + CompileModule (cold, no cache)", 5, func() {
		r := wazero.NewRuntime(ctx)
		defer r.Close(ctx)
		if _, err := r.CompileModule(ctx, bin); err != nil {
			panic(err)
		}
	})

	// --- 2. Compiler engine with file cache (2nd+ run of process = warm) ---
	fc, err := wazero.NewCompilationCacheWithDir(cacheDir)
	if err != nil {
		panic(err)
	}
	defer fc.Close(ctx)
	cfg := wazero.NewRuntimeConfigCompiler().WithCompilationCache(fc)
	timeit("wazevo: new runtime + CompileModule (file cache)", 5, func() {
		r := wazero.NewRuntimeWithConfig(ctx, cfg)
		defer r.Close(ctx)
		if _, err := r.CompileModule(ctx, bin); err != nil {
			panic(err)
		}
	})

	// --- 3. Interpreter, cold ---
	timeit("interp: new runtime + CompileModule (cold)", 20, func() {
		r := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())
		defer r.Close(ctx)
		if _, err := r.CompileModule(ctx, bin); err != nil {
			panic(err)
		}
	})

	// --- 4. Instantiate-only cost on an already-compiled module ---
	{
		r := wazero.NewRuntime(ctx)
		defer r.Close(ctx)
		wasi_snapshot_preview1.MustInstantiate(ctx, r)
		compiled, err := r.CompileModule(ctx, bin)
		if err != nil {
			panic(err)
		}
		defer compiled.Close(ctx)
		timeit("wazevo: InstantiateModule only (pre-compiled)", 20, func() {
			mod, err := r.InstantiateModule(ctx, compiled, wazero.NewModuleConfig())
			if err != nil {
				if _, ok := err.(*sys.ExitError); !ok {
					panic(err)
				}
			} else {
				mod.Close(ctx)
			}
		})
	}
	{
		r := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())
		defer r.Close(ctx)
		wasi_snapshot_preview1.MustInstantiate(ctx, r)
		compiled, err := r.CompileModule(ctx, bin)
		if err != nil {
			panic(err)
		}
		defer compiled.Close(ctx)
		timeit("interp: InstantiateModule only (pre-compiled)", 20, func() {
			mod, err := r.InstantiateModule(ctx, compiled, wazero.NewModuleConfig())
			if err != nil {
				if _, ok := err.(*sys.ExitError); !ok {
					panic(err)
				}
			} else {
				mod.Close(ctx)
			}
		})
	}

	// --- 5. WASI host module instantiation (part of typical VM setup) ---
	timeit("wazevo: runtime + WASI instantiate + compile + instantiate", 5, func() {
		r := wazero.NewRuntimeWithConfig(ctx, cfg)
		defer r.Close(ctx)
		wasi_snapshot_preview1.MustInstantiate(ctx, r)
		mod, err := r.InstantiateWithConfig(ctx, bin, wazero.NewModuleConfig())
		if err != nil {
			if _, ok := err.(*sys.ExitError); !ok {
				panic(err)
			}
		} else {
			mod.Close(ctx)
		}
	})
}
