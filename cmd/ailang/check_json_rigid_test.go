package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/types"
)

// #2273: a signature variable the body specialises reaches `check --json` as
// TC_RIGID_001 with its line, in file mode and in package/dir mode.
func TestCheckJSON_RigidSignatureCode(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	src := "module q/main\n\npure func toInt[a](x: a) -> int = x\n\nexport pure func run() -> int = toInt(1)\n"
	if err := os.MkdirAll("q", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("q", "main.ail"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, NoCache: true}, pipeline.Source{Filename: "q/main.ail"})
	if err == nil {
		t.Fatal("toInt[a] returning its argument as int was accepted")
	}
	entry := errorToCheckJSONError(err, "q/main.ail")
	if entry.Code != types.TC_RIGID_001 {
		t.Fatalf("--json code = %q, want %s (%v)", entry.Code, types.TC_RIGID_001, err)
	}
	if entry.Line != 3 || entry.Suggestion == "" {
		t.Fatalf("--json entry = %+v, want line 3 with a suggestion", entry)
	}
	if got := jsonErrorCode(err.Error()); got != types.TC_RIGID_001 {
		t.Fatalf("package/dir --json code = %q, want %s", got, types.TC_RIGID_001)
	}
}
