package runner

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/types"
)

func repoRootForM4Test(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// captureRunStreams runs fn while capturing os.Stdout and os.Stderr.
func captureRunStreams(t *testing.T, fn func() int) (int, string, string) {
	t.Helper()
	origOut, origErr := os.Stdout, os.Stderr
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = outW
	os.Stderr = errW
	defer func() {
		os.Stdout = origOut
		os.Stderr = origErr
	}()

	code := fn()
	outW.Close()
	errW.Close()
	var outBuf, errBuf bytes.Buffer
	_, _ = io.Copy(&outBuf, outR)
	_, _ = io.Copy(&errBuf, errR)
	outR.Close()
	errR.Close()
	return code, outBuf.String(), errBuf.String()
}

// TST-RC1-15
func TestStdlibResultAndMathCombinators(t *testing.T) {
	root := repoRootForM4Test(t)
	dir := t.TempDir()
	prog := filepath.Join(dir, "combinators.ail")
	codeSrc := `module combinators
import std/result (Result, Ok, Err, mapResult, collect)
import std/math (minInt, maxInt, clampInt)
import std/io (println)

pure func parsePos(n: int) -> Result[int, string] {
  if n > 0 then Ok(n * 10) else Err("non-positive:${show(n)}")
}

export func main() -> () ! {IO} {
  println(show(mapResult(parsePos, [1, 2, 3])));
  println(show(mapResult(parsePos, [1, -2, -3])));
  println(show(mapResult(parsePos, [])));
  println(show(collect([Ok(1), Ok(2), Ok(3)])));
  println(show(collect([Ok(1), Err("first"), Err("second")])));
  println(show(collect([])));
  println(show(minInt(3, 5)));
  println(show(minInt(-2, 4)));
  println(show(maxInt(3, 5)));
  println(show(maxInt(-2, -8)));
  println(show(clampInt(5, 0, 10)));
  println(show(clampInt(-3, 0, 10)));
  println(show(clampInt(15, 0, 10)))
}
`
	if err := os.WriteFile(prog, []byte(codeSrc), 0o600); err != nil {
		t.Fatal(err)
	}

	wantLines := []string{
		"Ok([10, 20, 30])",
		`Err("non-positive:-2")`,
		"Ok([])",
		"Ok([1, 2, 3])",
		`Err("first")`,
		"Ok([])",
		"3",
		"-2",
		"5",
		"-2",
		"5",
		"0",
		"10",
	}
	want := strings.Join(wantLines, "\n")

	for _, strictBC := range []bool{false, true} {
		mode := "evaluator"
		if strictBC {
			mode = "strict-bytecode"
		}
		t.Run(mode, func(t *testing.T) {
			rc, stdout, stderr := captureRunStreams(t, func() int {
				return Run(context.Background(), Options{
					Filename:       prog,
					Entry:          "main",
					ArgsJSON:       "null",
					Caps:           "IO",
					Quiet:          true,
					ExplicitQuiet:  true,
					Print:          true,
					RelaxModules:   true,
					StdlibPath:     filepath.Join(root, "std"),
					BytecodeMode:   strictBC,
					StrictBytecode: strictBC,
				})
			})
			if rc != 0 {
				t.Fatalf("%s failed (rc=%d): stdout=%q stderr=%q", mode, rc, stdout, stderr)
			}
			if strings.TrimSpace(stdout) != want {
				t.Fatalf("%s output mismatch:\ngot:\n%s\nwant:\n%s", mode, strings.TrimSpace(stdout), want)
			}
		})
	}
}

// TST-RC1-18
func TestCapsHygiene_ResolveAutoCapsExcludesProcess(t *testing.T) {
	moduleIface := iface.NewIface("test")
	moduleIface.AddExport("main", &types.Scheme{
		Type: &types.TFunc2{
			EffectRow: &types.Row{
				Kind: types.EffectRow,
				Labels: map[string]types.Type{
					"IO":      types.Unit(),
					"FS":      types.Unit(),
					"Process": types.Unit(),
				},
			},
			Return: types.Unit(),
		},
	}, false)

	got := ResolveAutoCaps(moduleIface, "main")
	want := []string{"FS", "IO"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveAutoCaps() = %v, want %v (must exclude Process)", got, want)
	}
	if !EntryDeclaresEffect(moduleIface, "main", "Process") {
		t.Fatalf("EntryDeclaresEffect must report Process on main")
	}
}

// TST-RC1-18
func TestCapsHygiene_CheckMissingEntryCaps(t *testing.T) {
	makeIface := func(effects ...string) *iface.Iface {
		m := iface.NewIface("test")
		labels := make(map[string]types.Type, len(effects))
		for _, e := range effects {
			labels[e] = types.Unit()
		}
		m.AddExport("main", &types.Scheme{
			Type: &types.TFunc2{
				EffectRow: &types.Row{
					Kind:   types.EffectRow,
					Labels: labels,
				},
				Return: types.Unit(),
			},
		}, false)
		return m
	}

	// Missing FS when only IO granted -> E_CAPS_MISSING naming FS and --caps FS,IO.
	err := CheckMissingEntryCaps(makeIface("IO", "FS"), "main", "IO")
	if err == nil {
		t.Fatal("expected E_CAPS_MISSING for missing FS, got nil")
	}
	if !strings.Contains(err.Error(), "E_CAPS_MISSING") || !strings.Contains(err.Error(), "FS") || !strings.Contains(err.Error(), "--caps") {
		t.Fatalf("unexpected error format: %v", err)
	}

	// Ghost Debug effect is auto-granted and must NOT trigger E_CAPS_MISSING.
	if err := CheckMissingEntryCaps(makeIface("IO", "Debug"), "main", "IO"); err != nil {
		t.Fatalf("Debug ghost effect must not trigger E_CAPS_MISSING: %v", err)
	}

	// Pure entrypoint never triggers E_CAPS_MISSING.
	if err := CheckMissingEntryCaps(makeIface(), "main", ""); err != nil {
		t.Fatalf("pure entrypoint must not trigger E_CAPS_MISSING: %v", err)
	}

	// Exact grant passes.
	if err := CheckMissingEntryCaps(makeIface("IO", "FS"), "main", "IO,FS"); err != nil {
		t.Fatalf("exact grant must pass: %v", err)
	}
}

// TST-RC1-18
func TestCapsHygiene_RunPreflightAndAutoDisclosure(t *testing.T) {
	root := repoRootForM4Test(t)
	dir := t.TempDir()

	// 1. Entry declares {IO, FS}, prints before FS; with --caps IO must fail pre-flight with ZERO stdout!
	iofs := filepath.Join(dir, "iofs.ail")
	if err := os.WriteFile(iofs, []byte(`module iofs
import std/io (println)
import std/fs (fileExists)
export func main() -> () ! {IO, FS} {
  println("SIDE_EFFECT_BEFORE_FS");
  let _ = fileExists("nope.txt");
  ()
}
`), 0o600); err != nil {
		t.Fatal(err)
	}

	rc, stdout, stderr := captureRunStreams(t, func() int {
		return Run(context.Background(), Options{
			Filename:      iofs,
			Entry:         "main",
			ArgsJSON:      "null",
			Caps:          "IO",
			Quiet:         true,
			ExplicitQuiet: true,
			Print:         true,
			RelaxModules:  true,
			StdlibPath:    filepath.Join(root, "std"),
		})
	})
	if rc == 0 {
		t.Fatalf("expected non-zero exit for missing FS, got 0 (stdout=%q stderr=%q)", stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("pre-flight E_CAPS_MISSING must prevent side effects (stdout must be empty), got %q", stdout)
	}
	if !strings.Contains(stderr, "E_CAPS_MISSING") || !strings.Contains(stderr, "FS") {
		t.Fatalf("expected E_CAPS_MISSING naming FS on stderr, got %q", stderr)
	}

	// 2. --caps auto discloses granted caps to stderr when ExplicitQuiet is false (even with default Quiet=true).
	rc, stdout, stderr = captureRunStreams(t, func() int {
		return Run(context.Background(), Options{
			Filename:      iofs,
			Entry:         "main",
			ArgsJSON:      "null",
			Caps:          "auto",
			Quiet:         true,
			ExplicitQuiet: false,
			Print:         true,
			RelaxModules:  true,
			StdlibPath:    filepath.Join(root, "std"),
		})
	})
	if rc != 0 {
		t.Fatalf("--caps auto on {IO, FS} failed (rc=%d): stdout=%q stderr=%q", rc, stdout, stderr)
	}
	if !strings.Contains(stderr, "auto-granted capabilities: FS,IO") {
		t.Fatalf("expected auto-granted capabilities disclosure on stderr, got %q", stderr)
	}

	// 3. Explicit --quiet suppresses the auto-granted capabilities line.
	rc, stdout, stderr = captureRunStreams(t, func() int {
		return Run(context.Background(), Options{
			Filename:      iofs,
			Entry:         "main",
			ArgsJSON:      "null",
			Caps:          "auto",
			Quiet:         true,
			ExplicitQuiet: true,
			Print:         true,
			RelaxModules:  true,
			StdlibPath:    filepath.Join(root, "std"),
		})
	})
	if rc != 0 {
		t.Fatalf("--quiet --caps auto failed (rc=%d): stderr=%q", rc, stderr)
	}
	if strings.Contains(stderr, "auto-granted capabilities:") {
		t.Fatalf("explicit --quiet must suppress auto-granted line, got %q", stderr)
	}

	// 4. --caps auto refuses Process unless explicitly passed (--caps auto,Process).
	proc := filepath.Join(dir, "proc.ail")
	if err := os.WriteFile(proc, []byte(`module proc
import std/process (exec)
import std/io (println)
export func main() -> () ! {IO, Process} {
  let _ = exec("echo", ["hi"]);
  println("ran")
}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	rc, stdout, stderr = captureRunStreams(t, func() int {
		return Run(context.Background(), Options{
			Filename:      proc,
			Entry:         "main",
			ArgsJSON:      "null",
			Caps:          "auto",
			Quiet:         true,
			ExplicitQuiet: false,
			Print:         true,
			RelaxModules:  true,
			StdlibPath:    filepath.Join(root, "std"),
		})
	})
	if rc == 0 || !strings.Contains(stderr, "Process") || !strings.Contains(stderr, "auto,Process") {
		t.Fatalf("--caps auto must refuse Process with auto,Process hint: rc=%d stdout=%q stderr=%q", rc, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("refused Process run must produce no stdout, got %q", stdout)
	}
}

// TST-RC1-19
func TestSemVerGoSurface_GoopusAndStabilityDoc(t *testing.T) {
	root := repoRootForM4Test(t)

	// 1. third_party/goopus must not exist at module root; it lives under internal/.
	oldPath := filepath.Join(root, "third_party", "goopus")
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be moved under internal/, but stat returned err=%v", filepath.ToSlash(oldPath), err)
	}
	newPath := filepath.Join(root, "internal", "third_party", "goopus", "opus", "encoder.go")
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("expected %s to exist after move: %v", filepath.ToSlash(newPath), err)
	}

	// 2. serveapi packages must carry Internal / not covered by 1.x notices in doc.go.
	for _, rel := range []string{
		"serveapi/doc.go",
		"serveapi/protocol/doc.go",
		"serveapi/protocol/hostcall/doc.go",
		"serveapi/protocol/mcphttp/doc.go",
	} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
		text := string(data)
		if !strings.Contains(text, "Internal") || !strings.Contains(text, "not covered by the 1.x") {
			t.Errorf("%s must contain Internal and 'not covered by the 1.x' notice:\n%s", rel, text)
		}
	}

	// 3. docs/docs/reference/stability.md must document Go packages (Internal) and binary split.
	stab, err := os.ReadFile(filepath.Join(root, "docs", "docs", "reference", "stability.md"))
	if err != nil {
		t.Fatal(err)
	}
	stabText := string(stab)
	for _, want := range []string{
		"Go packages",
		"not covered by the 1.x promise",
		"What ships in the binary",
		"separate Go module or binary",
	} {
		if !strings.Contains(stabText, want) {
			t.Errorf("stability.md missing %q", want)
		}
	}
}
