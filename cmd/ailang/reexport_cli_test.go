package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// reexportPackages is three path packages: core defines Color, facade
// re-exports it (marked moved), and app depends ONLY on facade.
func reexportPackages(facadeLine string) map[string]string {
	pkg := func(name, prefix, exports, deps, maxFx string) string {
		return "[package]\nname = \"" + name + "\"\nversion = \"0.1.0\"\nedition = \"1\"\nailang = \">= 0.40.0\"\nmodule_prefix = \"" + prefix + "\"\n" +
			exports + deps + "[effects]\nmax = [" + maxFx + "]\n"
	}
	return map[string]string{
		"core/ailang.toml": pkg("demo/core", "core", "[exports]\nmodules = [\"core/lib/color\"]\n", "", ""),
		"core/core/lib/color.ail": "module core/lib/color\nexport type Color = Red | Blue(int)\n" +
			"export pure func name(c: Color) -> string = match c { Red => \"red\", Blue(n) => \"blue ${show(n)}\" }\n",
		"facade/ailang.toml": pkg("demo/facade", "facade", "[exports]\nmodules = [\"facade/colors\"]\n",
			"[dependencies]\n\"demo/core\" = { path = \"../core\" }\n", ""),
		"facade/facade/colors.ail": "module facade/colors\n" + facadeLine + "\n",
		"app/ailang.toml":          pkg("demo/app", "app", "", "[dependencies]\n\"demo/facade\" = { path = \"../facade\" }\n", "\"IO\""),
		"app/app/main.ail": "module app/main\nimport std/io (println)\nimport pkg/demo/facade/colors (Color, Red, Blue, name)\n" +
			"pure func code(c: Color) -> int = match c { Red => 1, Blue(_) => 2 }\n" +
			"export func main() -> () ! {IO} { println(name(Blue(3))); println(show(code(Red))) }\n",
	}
}

// A dependent package imports a type, its constructors and a function through
// a facade in another package, without depending on the package that defines
// them; check and both engines agree (M-PKG-PARALLEL-AGENTS C1).
func TestCLI_ReexportAcrossPackages(t *testing.T) {
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	bin := buildAilang(t)
	dir := writeTree(t, reexportPackages("export import pkg/demo/core/lib/color (Color, Red, Blue, name)"))
	ailangIn(t, bin, dir+"/facade", "lock").want(t, "lock facade", 0)
	ailangIn(t, bin, dir+"/app", "lock").want(t, "lock app", 0)
	ailangIn(t, bin, dir+"/app", "check", "app/main.ail").want(t, "check", 0, "No errors found")
	runBoth(t, bin, dir+"/app", "app/main.ail", "blue 3")
	runBoth(t, bin, dir+"/app", "app/main.ail", "1")
}

// TST-PKGPAR-16
// A facade package's [effects].max must cover the functions it re-exports:
// re-exporting a ! {Net} function under max = [] fails the facade's check,
// naming the function and where it is defined. Widening max to Net passes.
func TestCLI_ReexportEffectCeiling(t *testing.T) {
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	bin := buildAilang(t)
	files := map[string]string{
		"core/ailang.toml": "[package]\nname = \"demo/core\"\nversion = \"0.1.0\"\nedition = \"1\"\nailang = \">= 0.40.0\"\nmodule_prefix = \"core\"\n" +
			"[exports]\nmodules = [\"core/net\"]\n[effects]\nmax = [\"Net\"]\n",
		"core/core/net.ail": "module core/net\nexport func fetch(url: string) -> string ! {Net} = url\nexport pure func plain(s: string) -> string = s\n",
		"facade/ailang.toml": "[package]\nname = \"demo/facade\"\nversion = \"0.1.0\"\nedition = \"1\"\nailang = \">= 0.40.0\"\nmodule_prefix = \"facade\"\n" +
			"[exports]\nmodules = [\"facade/api\"]\n[dependencies]\n\"demo/core\" = { path = \"../core\" }\n[effects]\nmax = []\n",
		"facade/facade/api.ail": "module facade/api\nexport import pkg/demo/core/net (fetch, plain)\n",
	}
	dir := writeTree(t, files)
	ailangIn(t, bin, dir+"/facade", "lock").want(t, "lock", 0)
	r := ailangIn(t, bin, dir+"/facade", "check", "facade/api.ail")
	r.want(t, "check under max = []", 1, "effect ceiling violation", "re-exported function fetch", "pkg/demo/core/net")

	writeTree2(t, dir, map[string]string{"facade/ailang.toml": strings.Replace(files["facade/ailang.toml"], "max = []", "max = [\"Net\"]", 1)})
	ailangIn(t, bin, dir+"/facade", "check", "facade/api.ail").want(t, "check under max = [Net]", 0, "No errors found")
}

// writeTree2 writes files under an existing dir.
func writeTree2(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, src := range files {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
