package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestContractsOnVMRefusal pins which flag combinations are refused (#1956).
// Red mutation: return "" unconditionally — the bytecode rows fail.
func TestContractsOnVMRefusal(t *testing.T) {
	tests := []struct {
		name     string
		opts     Options
		refused  bool
		mentions string
	}{
		{"evaluator with contracts", Options{VerifyContracts: true}, false, ""},
		{"VM without contracts", Options{BytecodeMode: true}, false, ""},
		{"--bytecode --verify-contracts", Options{VerifyContracts: true, BytecodeMode: true}, true, "Drop --bytecode"},
		{"--strict-bytecode --verify-contracts", Options{VerifyContracts: true, BytecodeMode: true, StrictBytecode: true}, true, "Drop --strict-bytecode"},
		{"--batch --bytecode --verify-contracts", Options{VerifyContracts: true, BytecodeMode: true, Batch: true}, true, "Drop --bytecode"},
		{"--batch --verify-contracts (evaluator)", Options{VerifyContracts: true, Batch: true}, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := ContractsOnVMRefusal(tt.opts)
			if (msg != "") != tt.refused {
				t.Fatalf("refused = %v (%q), want %v", msg != "", msg, tt.refused)
			}
			if tt.refused && (!strings.HasPrefix(msg, "CTR004: ") || !strings.Contains(msg, tt.mentions) || !strings.Contains(msg, "#1956")) {
				t.Fatalf("refusal %q must start with CTR004, name %q and the tracking issue", msg, tt.mentions)
			}
		})
	}
}

// TestRun_RefusesContractsOnVMBeforeExecuting: Run exits 2 before reading or
// compiling the file, so the program never runs unchecked. The file does not
// even exist — reaching the read would print a different error and exit 1.
func TestRun_RefusesContractsOnVMBeforeExecuting(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "never_read.ail")
	if _, err := os.Stat(missing); err == nil {
		t.Fatal("fixture path unexpectedly exists")
	}
	code := Run(context.Background(), Options{Filename: missing, VerifyContracts: true, BytecodeMode: true, Quiet: true})
	if code != 2 {
		t.Fatalf("Run exit = %d, want 2 (CTR004 refusal before reading the file)", code)
	}
}
