package v1

import (
	"context"
	"testing"

	"github.com/pschlump/wazero"
	"github.com/pschlump/wazero/api"
	"github.com/pschlump/wazero/internal/integration_test/spectest"
	"github.com/pschlump/wazero/internal/platform"
)

func TestCompiler(t *testing.T) {
	if !platform.CompilerSupported() {
		t.Skip()
	}
	spectest.Run(t, Testcases, context.Background(), wazero.NewRuntimeConfigCompiler().WithCoreFeatures(api.CoreFeaturesV1))
}

func TestInterpreter(t *testing.T) {
	spectest.Run(t, Testcases, context.Background(), wazero.NewRuntimeConfigInterpreter().WithCoreFeatures(api.CoreFeaturesV1))
}
