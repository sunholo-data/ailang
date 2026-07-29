package eval_harness

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestRunAICheckParsesJSONFromNonzeroChild(t *testing.T) {
	dir := t.TempDir()
	source := `package main
import ("fmt"; "os")
func main() {
	fmt.Print("{\"file\":\"fixture.ail\",\"check\":{\"passed\":true,\"error_count\":0},\"verify\":{\"available\":true,\"verified\":0,\"counterexample\":0,\"skipped\":0,\"errors\":1}}")
	os.Exit(1)
}`
	src := filepath.Join(dir, "main.go")
	if err := os.WriteFile(src, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "fake-ailang")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-o", bin, src)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build helper: %v\n%s", err, output)
	}
	result, _, err := RunAICheck(bin, "fixture.ail", time.Second)
	if err != nil {
		t.Fatalf("RunAICheck: %v", err)
	}
	if result.Verify.Errors != 1 {
		t.Fatalf("verify.errors = %d, want 1", result.Verify.Errors)
	}
}
