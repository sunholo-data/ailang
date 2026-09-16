package pi

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

func executorTaskWithTools(tools []string) *executor.Task {
	return &executor.Task{AllowedTools: tools}
}

func TestToolArgs_ProfileAILANGOnlyHasNoBash(t *testing.T) {
	args, err := ProfileArgs(ToolProfileAILANGOnly)
	if err != nil {
		t.Fatal(err)
	}
	want := "--no-builtin-tools --tools read,edit,write,ailang_check,ailang_run"
	if args != want {
		t.Fatalf("ailang_only = %q, want %q", args, want)
	}
	if strings.Contains(args, "bash") {
		t.Fatal("ailang_only must never enable bash")
	}
}

func TestToolArgs_FullIsCLIDefault(t *testing.T) {
	for _, p := range []string{"", "full"} {
		if a, err := ProfileArgs(p); err != nil || a != "" {
			t.Fatalf("%q → %q, %v; want no flags (pi defaults)", p, a, err)
		}
	}
}

func TestToolArgs_UnmappedNameErrorsByName(t *testing.T) {
	// The coordinator's question-kind list: pi has none of Grep/Glob/WebFetch/
	// WebSearch, and used to run with ZERO tools, silently (V4).
	_, err := ToolArgs([]string{"Read", "Grep", "Glob", "WebFetch", "WebSearch"})
	if err == nil || !strings.Contains(err.Error(), `"Grep"`) {
		t.Fatalf("want an error naming Grep, got %v", err)
	}
}

func TestToolArgs_EmptyIsNoTools(t *testing.T) {
	a, _ := ToolArgs([]string{})
	if !reflect.DeepEqual(a, []string{"--no-tools"}) {
		t.Fatalf("got %v", a)
	}
}

func TestBuildPiArgs_UsesMappedNames(t *testing.T) {
	args, err := buildPiArgs("m", executorTaskWithTools([]string{"Read", "Write"}), "d")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--no-builtin-tools --tools read,write") || strings.Contains(joined, "Read") {
		t.Fatalf("args = %v", args)
	}
	if _, err := buildPiArgs("m", executorTaskWithTools([]string{"Glob"}), "d"); err == nil {
		t.Fatal("an unmapped tool must fail at buildPiArgs, not reach pi")
	}
}
