package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

// checkSource writes src to a temp dir as <name>.ail, type-checks it via the
// module pipeline, and returns any error. Mirrors the temp-dir + chdir pattern
// used by multi_module_test.go.
func checkSource(t *testing.T, name, src string) error {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "ailang-prelude-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	fname := name + ".ail"
	if err := os.WriteFile(filepath.Join(tempDir, fname), []byte(src), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", fname, err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	_, runErr := Run(Config{Mode: ModeCheck}, Source{Filename: fname})
	return runErr
}

// M-PRELUDE-OPTION-RESULT compile-time conflict-surface fixtures.
func TestPreludeOptionResult_ConflictSurface(t *testing.T) {
	cases := []struct {
		name      string
		src       string
		wantError bool // true = compile MUST fail
	}{
		{
			name: "option_no_import",
			src: `module m
export func main() -> () ! {IO} {
  let o = Some(42);
  match o { Some(x) => println(show(x)), None => println("none") }
}`,
		},
		{
			name: "result_no_import",
			src: `module m
export func main() -> () ! {IO} {
  let r = if true then Ok(1) else Err("e");
  match r { Ok(x) => println(show(x)), Err(e) => println(e) }
}`,
		},
		{
			name: "explicit_import_unchanged",
			src: `module m
import std/option (Option, Some, None)
export func main() -> () ! {IO} {
  match Some(1) { Some(x) => println(show(x)), None => println("n") }
}`,
		},
		{
			name: "local_option_shadows",
			src: `module m
type Option[a] = Some(a) | None
export func main() -> () ! {IO} {
  match Some(7) { Some(x) => println(show(x)), None => println("n") }
}`,
		},
		{
			name: "local_result_different_ctors_shadows",
			src: `module m
type Result = Pending | Done
export func main() -> () ! {IO} {
  match Done { Pending => println("p"), Done => println("d") }
}`,
		},
		{
			name: "match_no_import",
			src: `module m
export func main() -> () ! {IO} {
  let v = match Some(5) { Some(x) => x, None => 0 };
  println(show(v))
}`,
		},
		{
			// A library module (no exported main) must STILL require an explicit
			// import — the injection is entry-only.
			name: "library_module_requires_import",
			src: `module m
export func wrap(x: int) -> Option[int] { Some(x) }`,
			wantError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkSource(t, "m", tc.src)
			if tc.wantError && err == nil {
				t.Fatalf("expected compile error for %q, got none", tc.name)
			}
			if !tc.wantError && err != nil {
				t.Fatalf("expected clean compile for %q, got: %v", tc.name, err)
			}
		})
	}
}
