package runner

import (
	"os"
	"path/filepath"
	"testing"
)

// show meets #2228 and #2230: two modules export a type T with constructors
// of the same names (different fields), and values reach show through a
// deprecated alias (std/list.head_or). show renders a constructor by its name
// only, so a.A(1) and b.A("hi") differ by their fields; every engine must
// print the same text, and the deprecated alias changes nothing.
func TestShowSameNamedADTsAndDeprecatedAliasAllEngines(t *testing.T) {
	stdlib, err := filepath.Abs(filepath.Join("..", "..", "std"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", stdlib)
	root := t.TempDir()
	t.Chdir(root)
	files := map[string]string{
		"sn/a.ail": "module sn/a\nexport type T = A(int) | B(string)\nexport pure func mk() -> [T] = [A(1), B(\"x\")]\n",
		"sn/b.ail": "module sn/b\nexport type T = B | C(string) | A(string)\nexport pure func mk() -> [T] = [A(\"hi\"), B, C(\"c\\\"d\")]\n",
		"sn/main.ail": "module sn/main\nimport std/io (println)\nimport std/list (head_or)\nimport sn/a as Ma (mk)\nimport sn/b as Mb (mk)\n\n" +
			"export func main() -> () ! {IO} {\n" +
			"  println(show(Ma.mk()));\n" +
			"  println(show(Mb.mk()));\n" +
			"  println(show((Ma.mk(), Mb.mk())));\n" +
			"  println(show(head_or([\"a\"], \"z\")));\n" +
			"  println(show([head_or([Some(\"q\")], None)]))\n" +
			"}\n",
	}
	for rel, src := range files {
		if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(rel, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	const want = "[A(1), B(\"x\")]\n" +
		"[A(\"hi\"), B, C(\"c\\\"d\")]\n" +
		"([A(1), B(\"x\")], [A(\"hi\"), B, C(\"c\\\"d\")])\n" +
		"a\n" +
		"[Some(\"q\")]\n"
	for _, engine := range allEngines {
		t.Run(engine, func(t *testing.T) {
			if got := runOnEngine(t, "sn/main.ail", "", engine); got != want {
				t.Fatalf("%s:\n got %q\nwant %q", engine, got, want)
			}
		})
	}
}
