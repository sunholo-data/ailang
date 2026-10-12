package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func assertPolicyCacheClean(t *testing.T, base string) {
	t.Helper()
	dirs, err := filepath.Glob(filepath.Join(base, "ailang-policy-cache-*"))
	if err != nil || len(dirs) != 0 {
		t.Fatalf("leaked generated cache: %v (%v)", dirs, err)
	}
}

func TestRunPolicy_CacheDirectory(t *testing.T) {
	skipWithoutRestrictedMode(t)
	bin := buildAilang(t)
	for _, mode := range []string{"unset", "empty", "outside", "inside", "symlink-leaf", "trusted"} {
		t.Run(mode, func(t *testing.T) {
			dir, base := t.TempDir(), t.TempDir()
			sandbox := filepath.Join(dir, "sandbox")
			body := "allowed_caps = [\"IO\", \"FS\"]\nfs_sandbox = \"${SANDBOX}\"\nentry = \"main\"\ntimeout_ms = 30000\n"
			if mode == "trusted" {
				body += "security_mode = \"trusted_host\"\n"
			}
			pol := writePolicy(t, dir, body)
			f := writeAil(t, sandbox, "prog.ail", "module prog\nexport func main() -> () ! {IO} = println(\"cache-run\")\n")
			t.Setenv("TMPDIR", base)
			cache := ""
			switch mode {
			case "outside":
				cache = filepath.Join(t.TempDir(), "operator")
			case "inside":
				cache = filepath.Join(sandbox, "operator")
			case "symlink-leaf":
				alias := filepath.Join(t.TempDir(), "alias")
				if err := os.Symlink(sandbox, alias); err != nil {
					t.Skip(err)
				}
				cache = filepath.Join(alias, "missing", "cache")
			}
			t.Setenv("AILANG_CACHE_DIR", cache)
			if mode == "unset" {
				if err := os.Unsetenv("AILANG_CACHE_DIR"); err != nil {
					t.Fatal(err)
				}
			}
			start := time.Now()
			stdout, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, f)
			t.Logf("%s compile/run latency: %s", mode, time.Since(start))
			if code != 0 || !strings.Contains(stdout, "cache-run") {
				t.Fatalf("exit %d: %s%s", code, stdout, stderr)
			}
			assertPolicyCacheClean(t, base)
			inside := mode == "inside" || mode == "symlink-leaf"
			warnings := strings.Count(stderr, "warning: AILANG_CACHE_DIR")
			if inside {
				if warnings != 1 || !strings.Contains(stderr, "poison") || !strings.Contains(stderr, "fs_deny_write") {
					t.Fatalf("warning: %s", stderr)
				}
			} else if warnings != 0 {
				t.Fatal(stderr)
			}
			if cache != "" {
				if _, err := os.Stat(filepath.Join(cache, "compile")); err != nil {
					t.Fatal(err)
				}
				if mode == "outside" {
					start = time.Now()
					_, stderr, code = runAilangBin(t, bin, "run", "--policy", pol, f)
					t.Logf("persistent override second run latency: %s", time.Since(start))
					if code != 0 {
						t.Fatalf("second run: %s", stderr)
					}
				}
			}
			if !inside && mode != "trusted" {
				if _, err := os.Stat(filepath.Join(sandbox, ".ailang")); !os.IsNotExist(err) {
					t.Fatalf("sandbox .ailang: %v", err)
				}
			}
			if mode == "trusted" {
				if _, err := os.Stat(filepath.Join(sandbox, ".ailang", "cache", "compile")); err != nil {
					t.Fatalf("trusted default changed: %v", err)
				}
			}
		})
	}
}

func TestRunPolicy_CacheUnsafeTemporaryDirectory(t *testing.T) {
	skipWithoutRestrictedMode(t)
	bin := buildAilang(t)
	for _, symlink := range []bool{false, true} {
		t.Run(map[bool]string{false: "direct", true: "symlink"}[symlink], func(t *testing.T) {
			dir := t.TempDir()
			pol := writePolicy(t, dir, "allowed_caps = [\"IO\", \"FS\"]\nfs_sandbox = \"${SANDBOX}\"\nentry = \"main\"\ntimeout_ms = 30000\n")
			sandbox := filepath.Join(dir, "sandbox")
			f := writeAil(t, sandbox, "prog.ail", "module prog\nexport func main() -> () ! {IO} = println(\"must-not-run\")\n")
			base := sandbox
			if symlink {
				base = filepath.Join(t.TempDir(), "alias")
				if err := os.Symlink(sandbox, base); err != nil {
					t.Skip(err)
				}
			}
			t.Setenv("AILANG_CACHE_DIR", "")
			t.Setenv("TMPDIR", base)
			stdout, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, f)
			if code != 1 || !strings.Contains(stderr, "temporary compile cache is inside fs_sandbox") || strings.Contains(stdout, "must-not-run") {
				t.Fatalf("exit %d: %s%s", code, stdout, stderr)
			}
			assertPolicyCacheClean(t, sandbox)
		})
	}
}

func TestRunPolicy_CacheCleanupOnWorkerTermination(t *testing.T) {
	skipWithoutRestrictedMode(t)
	bin := buildAilang(t)
	for _, reason := range []string{"timeout", "output_limit", "worker_error", "admission_denial"} {
		t.Run(reason, func(t *testing.T) {
			dir, base := t.TempDir(), t.TempDir()
			timeout, limit, caps := "30000", "0", "\"IO\", \"FS\", \"Clock\""
			prog := "module prog\nimport std/io (exit)\nexport func main() -> () ! {IO} = exit(7)\n"
			switch reason {
			case "timeout":
				timeout, prog = "300", loopProgram
			case "output_limit":
				limit, prog = "100", "module prog\nfunc loop(i: int) -> () ! {IO} = if i > 300 then () else { println(\"long-output-line\"); loop(i + 1) }\nexport func main() -> () ! {IO} = loop(1)\n"
			case "admission_denial":
				caps, prog = "\"FS\"", "module prog\nexport func main() -> () ! {IO} = println(\"denied\")\n"
			}
			pol := writePolicy(t, dir, "allowed_caps = ["+caps+"]\nfs_sandbox = \"${SANDBOX}\"\nentry = \"main\"\ntimeout_ms = "+timeout+"\nmax_output_bytes = "+limit+"\n")
			f := writeAil(t, filepath.Join(dir, "sandbox"), "prog.ail", prog)
			t.Setenv("AILANG_CACHE_DIR", "")
			t.Setenv("TMPDIR", base)
			_, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, f)
			want := 3
			if reason == "worker_error" {
				want = 7
			}
			if reason == "admission_denial" {
				want = 2
			}
			if code != want {
				t.Fatalf("exit %d, want %d: %s", code, want, stderr)
			}
			assertPolicyCacheClean(t, base)
		})
	}
}
