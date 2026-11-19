package pi

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

// The lean lane (tool profile "lean"): core tools only, a fixed short system prompt,
// and no ambient discovery. A fine-tune is trained on exactly this prefix.
func TestBuildPiArgs_LeanLane(t *testing.T) {
	tools, err := executor.ProfileTools(executor.ToolProfileLean)
	if err != nil {
		t.Fatal(err)
	}
	task := &executor.Task{ID: "t", AllowedTools: tools, MinimalHarnessContext: true}
	args, err := buildPiArgs("ollama/m", task, "do it")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, "\x00")
	for _, want := range []string{"--system-prompt\x00" + LeanSystemPrompt, "--no-skills", "--no-prompt-templates",
		"--no-context-files", "--no-extensions", "--tools\x00read,bash,edit,write"} {
		if !strings.Contains(joined, want) {
			t.Errorf("lean args missing %q: %q", want, args)
		}
	}
	if strings.Count(joined, "--no-extensions") != 1 {
		t.Errorf("--no-extensions must appear once: %q", args)
	}
	// Control: an ordinary task keeps pi's defaults.
	plain, _ := buildPiArgs("ollama/m", &executor.Task{ID: "t"}, "do it")
	for _, banned := range []string{"--system-prompt", "--no-skills"} {
		if strings.Contains(strings.Join(plain, " "), banned) {
			t.Errorf("ordinary task got %s: %q", banned, plain)
		}
	}
}
