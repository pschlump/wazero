package frontend

import (
	"github.com/pschlump/wazero/internal/engine/wazevo/ssa"
	"github.com/pschlump/wazero/internal/wasm"
)

func FunctionIndexToFuncRef(idx wasm.Index) ssa.FuncRef {
	return ssa.FuncRef(idx)
}
