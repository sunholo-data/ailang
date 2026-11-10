package prefixpkg

import (
	"os"
	"path/filepath"
	"testing"
)

// Spellings are the two ways a root package can name a dependency module that
// declares module_prefix "ev": its published path and its prefix.
var Spellings = map[string]string{"pkg": "pkg/evil/dep/", "pfx": "ev/"}

// CeilingCases name the root modules Ceiling writes, per spelling, that a root
// with [effects].max = ["IO"] must reject. "call" performs the dependency's FS
// effect (its main writes the file CeilingMarker names); the others only
// REFERENCE an effectful dependency function, so only the ceiling's charge on
// a reference to a dependency global (effect_ceiling.go, rule (b)) can reject
// them: "ref" returns it, "net" returns a Net function, "trans" returns a
// dependency function that calls another one inside the dependency, "ho"
// stores it in a record.
var CeilingCases = []string{"call", "ref", "net", "trans", "ho"}

// CeilingMarker is the file the "call" case's main would write.
func CeilingMarker(app, spelling string) string {
	return filepath.Join(app, "PWNED_call_"+spelling)
}

// Ceiling writes a dependency evil/dep (module_prefix "ev") with FS and Net
// functions and a root acme/app capped at [effects].max = ["IO"], one root
// module per case and spelling (`<case>_<spelling>.ail`, plus a passing
// `pure_<spelling>.ail` that uses only a pure function), and returns the
// root's directory.
func Ceiling(t *testing.T) string {
	t.Helper()
	files := map[string]string{
		"dep/ailang.toml": Manifest("evil/dep", "ev", []string{"ev/files", "ev/wrap"}, nil, nil),
		"dep/ev/files.ail": `module ev/files

import std/fs (writeFile)
import std/net (httpGet)

export func touch(p: string) -> () ! {FS} = writeFile(p, "pwned")

export func fetch(u: string) -> string ! {Net} = httpGet(u)

export func harmless(n: int) -> int = n + 1
`,
		"dep/ev/wrap.ail": `module ev/wrap

import ev/files (touch)

export func touchVia(p: string) -> () ! {FS} = touch(p)
`,
		"app/ailang.toml": Manifest("acme/app", "", nil, map[string]string{"evil/dep": "../dep"}, []string{"IO"}),
	}
	for sp, pre := range Spellings {
		files["app/ref_"+sp+".ail"] = "module ref_" + sp + "\n\nimport " + pre + "files (touch)\n\n" +
			"export func toucher() -> (string -> () ! {FS}) = touch\n"
		files["app/net_"+sp+".ail"] = "module net_" + sp + "\n\nimport " + pre + "files (fetch)\n\n" +
			"export func fetcher() -> (string -> string ! {Net}) = fetch\n"
		files["app/trans_"+sp+".ail"] = "module trans_" + sp + "\n\nimport " + pre + "wrap (touchVia)\n\n" +
			"export func viaWrap() -> (string -> () ! {FS}) = touchVia\n"
		files["app/ho_"+sp+".ail"] = "module ho_" + sp + "\n\nimport " + pre + "files (touch)\n\n" +
			"export func box() -> {f: string -> () ! {FS}} = {f: touch}\n"
		files["app/pure_"+sp+".ail"] = "module pure_" + sp + "\n\nimport std/io (println)\nimport " + pre + "files (harmless)\n\n" +
			"export func main() -> () ! {IO} = println(show(harmless(1)))\n"
	}
	root := Tree(t, files)
	app := filepath.Join(root, "app")
	for sp, pre := range Spellings {
		src := "module call_" + sp + "\n\nimport std/io (println)\nimport " + pre + "files (touch)\n\n" +
			"export func main() -> () ! {IO, FS} {\n  touch(\"" + CeilingMarker(app, sp) + "\");\n  println(\"done\")\n}\n"
		if err := os.WriteFile(filepath.Join(app, "call_"+sp+".ail"), []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	Lock(t, app, Dep{Name: "evil/dep", Dir: "../dep"})
	return app
}
