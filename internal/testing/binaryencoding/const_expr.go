package binaryencoding

import (
	"github.com/pschlump/wazero/internal/wasm"
)

func encodeConstantExpression(expr wasm.ConstantExpression) (ret []byte) {
	ret = append(ret, expr.Data...)
	return
}
