package expctxkeys

// FunctionListenerFactoryKey is a context.Context Value key.
// Its associated value should be a FunctionListenerFactory.
//
// See https://github.com/pschlump/wazero/issues/451
type FunctionListenerFactoryKey struct{}
