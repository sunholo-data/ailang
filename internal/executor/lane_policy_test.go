package executor

import (
	"strings"
	"testing"
)

// M-EXECUTOR-POLICY-HARDENING D3: ailang_only + trusted_host is refused;
// restricted passes; the full profile is not judged.
func TestCheckLanePolicy(t *testing.T) {
	restricted := []byte("allowed_caps = [\"IO\"]\nentry = \"main\"\n")
	trusted := []byte("allowed_caps = [\"IO\", \"Process\"]\nsecurity_mode = \"trusted_host\"\nprocess_allow = [\"git:status\"]\nentry = \"main\"\n")
	if err := CheckLanePolicy(ToolProfileAILANGOnly, restricted); err != nil {
		t.Fatalf("restricted under ailang_only must pass: %v", err)
	}
	err := CheckLanePolicy(ToolProfileAILANGOnly, trusted)
	if err == nil || !strings.Contains(err.Error(), "trusted_host") || !strings.Contains(err.Error(), "tool_policy: full") {
		t.Fatalf("trusted_host under ailang_only must be refused naming the migration, got %v", err)
	}
	if err := CheckLanePolicy(ToolProfileFull, trusted); err != nil {
		t.Fatalf("the full profile is not judged: %v", err)
	}
	if err := CheckLanePolicy("Read,AilangRun", trusted); err == nil {
		t.Fatal("a custom restricted list is a lane too")
	}
	// A policy that does not resolve at all is refused by its own reason.
	if err := CheckLanePolicy(ToolProfileAILANGOnly, []byte("allowed_caps = [\"IO\", \"Process\"]\n")); err == nil || !strings.Contains(err.Error(), "Process") {
		t.Fatalf("unresolvable policy: %v", err)
	}
}
