// Package prefixpkg builds an offline package fixture whose modules can be
// reached under more than one spelling: a dependency package `acme/shapes`
// with `module_prefix = "shp"`, whose own files import each other through the
// prefix (`shp/types/shape`), consumed by a root package `acme/app` that
// imports the same modules by their published path
// (`pkg/acme/shapes/types/shape`). Before M1b (canonical module identity) the
// loader compiled and evaluated `types/shape.ail` twice, once per spelling.
package prefixpkg

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// Canonical is the one identity every spelling of the shape module maps to in
// a CONSUMER's build (acme/app): the published path. Checked inside acme/shapes
// itself, the identity is the package's own spelling, PrefixSpelling.
const Canonical = "pkg/acme/shapes/types/shape"

// PrefixSpelling is how the shapes package's own files spell the module.
const PrefixSpelling = "shp/types/shape"

// ShapeFile is the shape module's path relative to the shapes package root.
const ShapeFile = "shp/types/shape.ail"

// InitLog is the message the shape module's module-level binding logs when it
// is evaluated. Counting it counts module initialisations.
const InitLog = "init shape module"

// Files is the fixture, keyed by path relative to the fixture root. Callers
// override entries (for instance a broken import) through New.
func Files() map[string]string {
	return map[string]string{
		"shapes/" + pkg.ManifestFile: `[package]
name = "acme/shapes"
module_prefix = "shp"
version = "0.1.0"
edition = "1"

[exports]
modules = ["shp/types/shape", "shp/ops/area", "shp/ops/self"]
`,
		"shapes/" + ShapeFile: `module shp/types/shape

import std/debug (log)

export type Shape = Circle(int) | Square(int)

let initMarker = {
  log("` + InitLog + `");
  1
}

export func mkCircle(r: int) -> Shape = Circle(r)

export func marked() -> int = initMarker
`,
		"shapes/shp/ops/area.ail": `module shp/ops/area

import shp/types/shape (Shape, Circle, Square)

export func area(s: Shape) -> int =
  match s { Circle(r) => 3 * r * r, Square(w) => w * w }

export func unitSquare() -> Shape = Square(1)
`,
		// The bare self-import spelling, legal only inside the package itself.
		"shapes/shp/ops/self.ail": `module shp/ops/self

import acme/shapes/types/shape (Shape, Square)

export func sq(n: int) -> Shape = Square(n)
`,
		// Entry inside the shapes package: reaches the shape module by the
		// prefix (via area), the bare self-import (via self) and the published
		// path (directly) — three spellings, one file.
		"shapes/shp/both.ail": `module shp/both

import std/io (println)
import shp/ops/area (area)
import shp/ops/self (sq)
import pkg/acme/shapes/types/shape (Circle)

export func main() -> () ! {IO} = println(show(area(sq(3)) + area(Circle(1))))
`,
		"app/" + pkg.ManifestFile: `[package]
name = "acme/app"
version = "0.1.0"
edition = "1"

[dependencies]
"acme/shapes" = { path = "../shapes" }
`,
		// A value of type Shape built by area (prefix spelling inside the
		// dependency) flows into main's annotation and back into area; a
		// Circle built here (published spelling) flows into area.
		"app/main.ail": `module main

import std/io (println)
import pkg/acme/shapes/types/shape (Shape, Circle, mkCircle, marked)
import pkg/acme/shapes/ops/area (area, unitSquare)

export func main() -> () ! {IO} {
  let s: Shape = unitSquare();
  println(show(area(Circle(2)) + area(s) + area(mkCircle(1)) + marked()))
}
`,
	}
}

// Want is main's output: 12 + 1 + 3 + 1.
const Want = "17\n"

// New writes the fixture (with overrides applied) under a fresh temp dir, with
// HOME isolated, and locks app against shapes. It returns the app and shapes
// package directories.
func New(t *testing.T, overrides map[string]string) (app, shapes string) {
	t.Helper()
	files := Files()
	for k, v := range overrides {
		files[k] = v
	}
	root := Tree(t, files)
	app, shapes = filepath.Join(root, "app"), filepath.Join(root, "shapes")
	Lock(t, app, Dep{Name: "acme/shapes", Dir: "../shapes"})
	return app, shapes
}

// Tree writes files (keyed by slash path relative to the root) under a fresh
// temp dir, with HOME isolated and no package-root or stdlib override, and
// returns the root.
func Tree(t *testing.T, files map[string]string) string {
	t.Helper()
	testutil.SetHomeDir(t, t.TempDir())
	t.Setenv("AILANG_PACKAGE_ROOT", "")
	t.Setenv("AILANG_STDLIB_PATH", "")
	root := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// Dep is one path dependency in a lock written by Lock: Dir is relative to
// the locking package's directory.
type Dep struct {
	Name string
	Dir  string
}

// Lock writes an ailang.lock in dir listing deps as path dependencies with the
// content hashes of the directories they name. List transitive dependencies
// too: `ailang lock` writes a flat lock.
func Lock(t *testing.T, dir string, deps ...Dep) {
	t.Helper()
	locked := make([]pkg.LockedPackage, 0, len(deps))
	for _, d := range deps {
		hash, err := pkg.ContentHash(filepath.Join(dir, filepath.FromSlash(d.Dir)))
		if err != nil {
			t.Fatal(err)
		}
		locked = append(locked, pkg.LockedPackage{Name: d.Name, Version: "0.1.0", Source: "path", Path: d.Dir, ContentHash: hash})
	}
	if err := pkg.NewLockFile(locked, "prefixpkg-fixture").Save(dir); err != nil {
		t.Fatal(err)
	}
}

// Manifest renders a minimal ailang.toml. prefix, exports and maxEffects may
// be empty; deps maps a dependency name to its path relative to the package.
func Manifest(name, prefix string, exports []string, deps map[string]string, maxEffects []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[package]\nname = %q\n", name)
	if prefix != "" {
		fmt.Fprintf(&b, "module_prefix = %q\n", prefix)
	}
	b.WriteString("version = \"0.1.0\"\nedition = \"1\"\n")
	if len(exports) > 0 {
		b.WriteString("\n[exports]\nmodules = [")
		for i, e := range exports {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "%q", e)
		}
		b.WriteString("]\n")
	}
	if len(deps) > 0 {
		names := make([]string, 0, len(deps))
		for n := range deps {
			names = append(names, n)
		}
		sort.Strings(names)
		b.WriteString("\n[dependencies]\n")
		for _, n := range names {
			fmt.Fprintf(&b, "%q = { path = %q }\n", n, deps[n])
		}
	}
	if maxEffects != nil {
		b.WriteString("\n[effects]\nmax = [")
		for i, e := range maxEffects {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "%q", e)
		}
		b.WriteString("]\n")
	}
	return b.String()
}
