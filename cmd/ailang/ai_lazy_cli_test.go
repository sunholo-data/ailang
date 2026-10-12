package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pkg"
)

// The program, including its AI branch, is type-checked in every case. Offline
// and application help must reach main without provider credentials/readiness.
func TestAILazyCLI_KeylessOfflineAndHelp(t *testing.T) {
	for _, k := range []string{"OPENROUTER_API_KEY", "GOOGLE_API_KEY", "GEMINI_API_KEY", "AILANG_AI_KEY_FILE"} {
		t.Setenv(k, "")
	}
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", filepath.Join(t.TempDir(), "absent-adc"))
	t.Setenv("OLLAMA_HOST", "http://127.0.0.1:1")
	root, _ := filepath.Abs("../..")
	t.Setenv("AILANG_STDLIB_PATH", filepath.Join(root, "std"))
	dir := t.TempDir()
	source := `module test/offline/cli
import std/io (println)
import std/env (getArgs)
import std/ai (call)
export func main() -> () ! {IO, Env, AI} {
 match getArgs() {
 "--offline" :: _ => println("offline"),
 "--help" :: _ => println("help"),
 _ => { println("before AI"); println(call("hello")) }
 }
}
`
	manifest := `[package]
name = "test/offline"
version = "0.1.0"
edition = "1"
[exports]
modules = ["test/offline/cli"]
[bin]
offline-cli = { module = "cli", caps = "IO,Env,AI", run_flags = ["--ai", "z-ai/glm-5.3-flash"] }
`
	for name, body := range map[string]string{"cli.ail": source, "ailang.toml": manifest} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	bin := cliTestBin(t)
	for _, model := range []string{"z-ai/glm-5.3-flash", "gemini-2-5-flash", "ollama:unavailable"} {
		for _, keyPath := range []string{"", filepath.Join(dir, "missing.key"), dir} {
			for _, strict := range []bool{false, true} {
				for _, mode := range []string{"--offline", "--help"} {
					args := []string{"run", "--quiet", "--package-dir", dir, "--caps", "IO,Env,AI", "--ai", model, "--ai-no-adc"}
					if keyPath != "" {
						args = append(args, "--ai-key-file", keyPath)
					}
					if strict {
						args = append(args, "--bytecode", "--strict-bytecode")
					}
					args = append(args, filepath.Join(dir, "cli.ail"), "--", mode)
					res := runCLIIn(t, bin, t.TempDir(), dir, args...)
					if res.exitCode != 0 || strings.TrimSpace(res.stdout) != strings.TrimPrefix(mode, "--") {
						t.Errorf("model=%s key=%s strict=%v mode=%s: exit=%d stdout=%q stderr=%q", model, keyPath, strict, mode, res.exitCode, res.stdout, res.stderr)
					}
				}
			}
		}
	}
	res := runCLIIn(t, bin, t.TempDir(), dir, "run", "--quiet", "--package-dir", dir, "--caps", "IO,Env,AI", "--ai", "z-ai/glm-5.3-flash", filepath.Join(dir, "cli.ail"))
	if res.exitCode != 1 || !strings.Contains(res.stdout, "before AI") || !strings.Contains(res.stderr, "OPENROUTER_API_KEY") {
		t.Fatalf("first actual AI did not fail at effect: %+v", res)
	}
	// Installing a genuine local [bin] reuses the existing shim writer/lock.
	if runtime.GOOS != "windows" {
		m, err := pkg.LoadManifest(dir)
		if err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		binDir := t.TempDir()
		if err := installBinsTo(&out, dir, binDir, bin, m); err != nil {
			t.Fatal(err)
		}
		t.Setenv("AILANG_AI_KEY_FILE", filepath.Join(dir, "missing.key"))
		for _, mode := range []string{"--offline", "--help"} {
			res := runCLIIn(t, pkg.ShimPath(binDir, "offline-cli"), t.TempDir(), t.TempDir(), mode)
			if res.exitCode != 0 || strings.TrimSpace(res.stdout) != strings.TrimPrefix(mode, "--") {
				t.Errorf("bin %s: %+v", mode, res)
			}
		}
	}
}
