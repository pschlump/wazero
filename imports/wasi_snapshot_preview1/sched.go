package wasi_snapshot_preview1

import (
	"context"

	"github.com/pschlump/wazero/api"
	"github.com/pschlump/wazero/experimental/sys"
	"github.com/pschlump/wazero/internal/wasip1"
	"github.com/pschlump/wazero/internal/wasm"
)

// schedYield is the WASI function named SchedYieldName which temporarily
// yields execution of the calling thread.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-sched_yield---errno
var schedYield = newHostFunc(wasip1.SchedYieldName, schedYieldFn, nil)

func schedYieldFn(_ context.Context, mod api.Module, _ []uint64) sys.Errno {
	sysCtx := mod.(*wasm.ModuleInstance).Sys
	sysCtx.Osyield()
	return 0
}
