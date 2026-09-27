package apiserver

import (
	"os"
	"path/filepath"
	"testing"
)

// M-SERVEAPI-OPERATOR-SURFACE D5: the directory given on the command line is
// the floor of the base path; module headers may only move it outward.

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, src := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func realDir(t *testing.T) string {
	t.Helper()
	d, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestResolveBasePath(t *testing.T) {
	t.Run("daneel layout: subdirectory module does not narrow the base", func(t *testing.T) {
		d := realDir(t)
		writeTree(t, d, map[string]string{
			"client/wsclient.ail": "module wsclient\n",
			"talk.ail":            "module talk\n",
		})
		// cwd elsewhere, so the cwd fallback cannot rescue a wrong answer.
		if got := ResolveBasePath([]string{d}, realDir(t)); got != d {
			t.Fatalf("got %q, want the given directory %q", got, d)
		}
	})
	t.Run("an inner implied root is never taken, even when it is the only one", func(t *testing.T) {
		d := realDir(t)
		writeTree(t, d, map[string]string{
			"client/wsclient.ail": "module wsclient\n",
			"talk.ail":            "-- no module header\nexport func f() -> int = 1\n",
		})
		if got := ResolveBasePath([]string{d}, d); got != d {
			t.Fatalf("got %q, want the given directory %q", got, d)
		}
	})
	t.Run("./api with module api/handlers widens to the project root", func(t *testing.T) {
		root := realDir(t)
		writeTree(t, root, map[string]string{
			"api/handlers.ail": "-- handlers\nmodule api/handlers\n",
			"api/a_util.ail":   "module a_util\n", // implies root/api, sorts first; loses to the outermost
		})
		if got := ResolveBasePath([]string{filepath.Join(root, "api")}, realDir(t)); got != root {
			t.Fatalf("got %q, want project root %q", got, root)
		}
	})
	t.Run("single file argument with a matching header", func(t *testing.T) {
		root := realDir(t)
		writeTree(t, root, map[string]string{"tools/serve/daneel_serve.ail": "module tools/serve/daneel_serve\n"})
		if got := ResolveBasePath([]string{filepath.Join(root, "tools/serve/daneel_serve.ail")}, realDir(t)); got != root {
			t.Fatalf("got %q, want %q", got, root)
		}
	})
	t.Run("no usable header: cwd when it contains the arguments", func(t *testing.T) {
		root := realDir(t)
		writeTree(t, root, map[string]string{"srv/a.ail": "module elsewhere/a\n"})
		if got := ResolveBasePath([]string{filepath.Join(root, "srv")}, root); got != root {
			t.Fatalf("got %q, want cwd %q", got, root)
		}
	})
	t.Run("no usable header, cwd unrelated: the argument's directory", func(t *testing.T) {
		root := realDir(t)
		writeTree(t, root, map[string]string{"srv/a.ail": "module elsewhere/a\n"})
		want := filepath.Join(root, "srv")
		if got := ResolveBasePath([]string{want}, realDir(t)); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
	t.Run("pkg directories are not walked", func(t *testing.T) {
		d := realDir(t)
		writeTree(t, d, map[string]string{
			"main.ail":           "module main\n",
			"pkg/vendor/x/y.ail": "module y\n",
		})
		if got := ResolveBasePath([]string{d}, realDir(t)); got != d {
			t.Fatalf("got %q, want %q", got, d)
		}
	})
}
