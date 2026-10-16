package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/pipeline"
)

func TestAIEmbedStdlibEffectContract(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "std"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", root)
	for _, effects := range []string{"AI", ""} {
		source := fmt.Sprintf("module probe\nimport std/ai (embed, AIError)\nimport std/result (Result)\nexport func main() -> Result[[float], AIError] ! {%s} = embed(\"hello\")\n", effects)
		path := filepath.Join(t.TempDir(), "probe.ail")
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		_, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, NoCache: true, RelaxModules: true}, pipeline.Source{Filename: path})
		if effects == "AI" && err != nil {
			t.Fatalf("embedding import failed: %v", err)
		}
		if effects == "" && err == nil {
			t.Fatal("embedding accepted without declared AI effect")
		}
	}
}
