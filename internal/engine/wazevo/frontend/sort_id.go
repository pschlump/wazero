package frontend

import (
	"slices"

	"github.com/pschlump/wazero/internal/engine/wazevo/ssa"
)

func sortSSAValueIDs(IDs []ssa.ValueID) {
	slices.SortFunc(IDs, func(i, j ssa.ValueID) int {
		return int(i) - int(j)
	})
}
