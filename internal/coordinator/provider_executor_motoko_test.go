package coordinator

import (
	"testing"
)

// TestProviderResolution_Motoko verifies the M-MOTOKO-EXECUTOR-ADAPTER
// blank-import wiring: NewExecutorProvider("motoko") must succeed without
// any factory/dispatch/coordinator code changes — proving EXECUTOR_SHAPE.md
// §3 auto-discovery holds for the new executor.
func TestProviderResolution_Motoko(t *testing.T) {
	p, err := NewExecutorProvider("motoko")
	if err != nil {
		t.Fatalf("NewExecutorProvider(\"motoko\") failed: %v\n"+
			"This means motoko's blank import in provider_executor.go is missing\n"+
			"or motoko/init() isn't running.", err)
	}
	if p == nil {
		t.Fatal("NewExecutorProvider returned nil provider")
	}
}
