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
	if !reflect.DeepEqual(got, []string{"Read", "Edit", "Write", "AilangCheck", "AilangRun", "BuiltinsSearch"}) {
		t.Fatalf("ailang_only = %v", got)
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
