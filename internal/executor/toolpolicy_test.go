package executor

import (
	"reflect"
	"strings"
	"testing"
)

func TestProfileTools(t *testing.T) {
	if got, err := ProfileTools("full"); err != nil || got != nil {
		t.Fatalf("full → %v, %v; want nil (CLI defaults)", got, err)
	}
	got, err := ProfileTools(ToolProfileAILANGOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range got {
		if tool == "Bash" {
			t.Fatal("ailang_only must not include Bash")
		}
	}
	// M-EXECUTOR-POLICY-HARDENING M4: the file tools are the policy's
	// sandboxed ones, never pi's native Read/Edit/Write.
	if !reflect.DeepEqual(got, []string{"AilangRead", "AilangEdit", "AilangWrite", "AilangCheck", "AilangRun", "BuiltinsSearch", "ExamplesSearch", "AilangCLI"}) {
		t.Fatalf("ailang_only = %v", got)
	}
	for _, native := range []string{"Read", "Edit", "Write"} {
		for _, tool := range got {
			if tool == native {
				t.Fatalf("ailang_only must not include native %s", native)
			}
		}
	}
	if _, err := ProfileTools("Read,Nope"); err == nil || !strings.Contains(err.Error(), `"Nope"`) {
		t.Fatalf("unknown name must error by name, got %v", err)
	}
}

func TestIntersectTools(t *testing.T) {
	got := IntersectTools([]string{"Read", "AilangCheck"}, []string{"Read", "Edit", "Write", "AilangCheck", "AilangRun"})
	if !reflect.DeepEqual(got, []string{"Read", "AilangCheck"}) {
		t.Fatalf("got %v", got)
	}
	if got := IntersectTools([]string{"Read", "Grep"}, []string{"Read"}); !reflect.DeepEqual(got, []string{"Read"}) {
		t.Fatalf("got %v", got)
	}
}
