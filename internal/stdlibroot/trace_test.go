package stdlibroot

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureStderr runs fn with os.Stderr redirected and returns what it wrote.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	fn()
	os.Stderr = orig
	_ = w.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}

// --trace-loader prints the chosen root and every candidate, for each outcome.
func TestResolve_TraceNamesRootAndCandidates(t *testing.T) {
	dir := isolate(t)

	Configure(Options{Trace: true})
	out := captureStderr(t, func() { _, _ = Resolve("") })
	if !strings.Contains(out, "[trace-loader] stdlib root: embedded") || !strings.Contains(out, "tried 1. "+filepath.Join(dir, "std")) {
		t.Errorf("embedded trace = %q", out)
	}

	disk := makeStd(t, filepath.Join(dir, "flagstd"), "io.ail")
	out = captureStderr(t, func() { _, _ = Resolve(disk) })
	if !strings.Contains(out, "stdlib root: "+disk+" (flag)") {
		t.Errorf("on-disk trace = %q", out)
	}

	out = captureStderr(t, func() { _, _ = Resolve(filepath.Join(dir, "nope")) })
	if !strings.Contains(out, "stdlib root: unresolved (flag)") {
		t.Errorf("unresolved trace = %q", out)
	}

	// Configure without Trace is silent.
	Configure(Options{})
	if out := captureStderr(t, func() { _, _ = Resolve("") }); out != "" {
		t.Errorf("untraced Resolve wrote %q", out)
	}
}

func TestNotStdlibError_EnvWording(t *testing.T) {
	err := &NotStdlibError{What: "AILANG_STDLIB_PATH", Tried: []string{"/a", "/b"}}
	msg := err.Error()
	for _, want := range []string{"AILANG_STDLIB_PATH does not name a stdlib directory", "  - /a", "  - /b", "unset it"} {
		if !strings.Contains(msg, want) {
			t.Errorf("missing %q in %q", want, msg)
		}
	}
	flag := (&NotStdlibError{What: "--stdlib-path", Tried: []string{"/a"}}).Error()
	if !strings.Contains(flag, "drop the flag") {
		t.Errorf("flag wording = %q", flag)
	}
}

func TestIsStdlibDirAndDisplayPath(t *testing.T) {
	dir := isolate(t)
	std := makeStd(t, filepath.Join(dir, "std"), "io.ail")
	if !IsStdlibDir(std) || IsStdlibDir(dir) {
		t.Fatal("IsStdlibDir must key on io.ail")
	}
	r := Root{Dir: std}
	if got := r.DisplayPath("ai/core.ail"); got != filepath.Join(std, "ai", "core.ail") {
		t.Errorf("DisplayPath = %q", got)
	}
	if got := Current(); got != (Options{}) {
		t.Errorf("Current = %+v, want zero options", got)
	}
}
