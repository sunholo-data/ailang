package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// #2390: extern functions were implemented only by the removed Go emitter
// (#2389), so no engine can run one. Every `extern func` declaration is
// EXT004 at the declaration, called or not, on every path that compiles a
// program, and an importer of a module that declares one fails naming it.
func TestExternFuncRejectedOnEveryPath(t *testing.T) {
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	bin := buildAilang(t)
	dir := t.TempDir()
	// `ailang test` compiles a module only when it has a test to run (a file
	// with none reports "No tests found" without compiling, for any error),
	// so every case carries one.
	const aTest = "test \"t\" { assert 1 == 1 }\n"
	files := map[string]string{
		"decl.ail": "module decl\nextern func hostAdd(x: int, y: int) -> int\nexport func main() -> () ! {IO} = println(\"ok\")\n" + aTest,
		"call.ail": "module call\nextern func hostAdd(x: int, y: int) -> int\nexport func main() -> () ! {IO} = println(show(hostAdd(1, 2)))\n" + aTest,
		// Shapes the retired EXT001-EXT003 rejected, and `extern pure func`
		// (PAR_EXTERN_REQUIRES_FUNC before): all are EXT004 now.
		"shapes.ail": "module shapes\nextern pure func _bad[a](x: a)\nexport func main() -> () ! {IO} = println(\"ok\")\n" + aTest,
		// Referenced only from a contract and a test body.
		"bodies.ail": "module bodies\nextern func host(x: int) -> int\n" +
			"export func f(x: int) -> int\nrequires { host(x) > 0 }\n{ x }\n" +
			"test \"t\" { assert host(1) == 1 }\n" +
			"export func main() -> () ! {IO} = println(show(f(1)))\n",
		"lib/host.ail": "module lib/host\nextern func hostAdd(x: int, y: int) -> int\nexport func add(x: int, y: int) -> int = x + y\n",
		"use.ail":      "module use\nimport lib/host (add)\nexport func main() -> () ! {IO} = println(show(add(1, 2)))\n" + aTest,
	}
	for p, src := range files {
		full := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run := func(args ...string) (string, int) {
		stdout, stderr, code := testutil.RunBounded(t, dir, 90*time.Second, bin, args...)
		return filepath.ToSlash(stdout + stderr), code
	}
	cases := []struct{ entry, at, name string }{
		{"decl.ail", "decl.ail:2:1", "hostAdd"},
		{"call.ail", "call.ail:2:1", "hostAdd"},
		{"shapes.ail", "shapes.ail:2:1", "_bad"},
		{"bodies.ail", "bodies.ail:2:1", "host"},
		{"use.ail", "lib/host.ail:2:1", "hostAdd"}, // the dependency is named
	}
	for _, c := range cases {
		t.Run(c.entry, func(t *testing.T) {
			for _, args := range [][]string{
				{"check", c.entry},
				{"run", "--quiet", "--caps", "IO", c.entry},
				{"run", "--quiet", "--caps", "IO", "--bytecode", c.entry},
				{"run", "--quiet", "--caps", "IO", "--strict-bytecode", c.entry},
				{"test", c.entry},
			} {
				out, code := run(args...)
				if code == 0 || !strings.Contains(out, "EXT004") || !strings.Contains(out, c.at) ||
					!strings.Contains(out, "'"+c.name+"'") || !strings.Contains(out, "Go emitter") {
					t.Errorf("%v: exit %d, want EXT004 for %q at %s:\n%s", args, code, c.name, c.at, out)
				}
				if strings.Contains(out, "#366") || strings.Contains(out, "panic:") {
					t.Errorf("%v: misleading output:\n%s", args, out)
				}
			}
			stdout, _, code := testutil.RunBounded(t, dir, 90*time.Second, bin, "check", "--json", c.entry)
			var res checkJSONOutput
			if err := json.Unmarshal([]byte(stdout), &res); err != nil || code == 0 || len(res.Errors) == 0 {
				t.Fatalf("check --json: exit %d, %v\n%s", code, err, stdout)
			}
			if e := res.Errors[0]; e.Code != "EXT004" || e.Line != 2 || e.Column != 1 ||
				!strings.HasSuffix(filepath.ToSlash(e.File), strings.SplitN(c.at, ":", 2)[0]) {
				t.Errorf("check --json: first error %+v, want EXT004 at %s", e, c.at)
			}
		})
	}
}

// Negative controls: an ordinary func and a builtin-backed std function still
// check and run, and `extern` stays a reserved word (it was already one:
// PAR_RESERVED_KEYWORD), not an identifier.
func TestExternRejectionNegativeControls(t *testing.T) {
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	bin := buildAilang(t)
	dir := t.TempDir()
	write := func(name, src string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("ok.ail", "module ok\nimport std/string (trim)\nexport func hostAdd(x: int, y: int) -> int = x + y\n"+
		"export func main() -> () ! {IO} = println(\"${show(hostAdd(1, 2))}${trim(\" a \")}\")\n")
	write("ident.ail", "module ident\nexport func main() -> () ! {IO} {\n  let extern = 1;\n  println(show(extern))\n}\n")
	for _, flags := range [][]string{nil, {"--bytecode"}, {"--strict-bytecode"}} {
		args := append(append([]string{"run", "--quiet", "--caps", "IO"}, flags...), "ok.ail")
		stdout, stderr, code := testutil.RunBounded(t, dir, 90*time.Second, bin, args...)
		if code != 0 || strings.TrimSpace(stdout) != "3a" {
			t.Errorf("%v: exit %d, stdout %q\n%s", args, code, stdout, stderr)
		}
	}
	stdout, _, code := testutil.RunBounded(t, dir, 90*time.Second, bin, "check", "--json", "ident.ail")
	var res checkJSONOutput
	if err := json.Unmarshal([]byte(stdout), &res); err != nil || code == 0 || len(res.Errors) == 0 ||
		res.Errors[0].Code != "PAR_RESERVED_KEYWORD" {
		t.Errorf("`let extern`: exit %d, %v, want first error PAR_RESERVED_KEYWORD:\n%s", code, err, stdout)
	}
}
