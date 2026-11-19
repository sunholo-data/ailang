package prefixpkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pkg"
)

// In-package test so the coverage filter keeps this package; the fixture's
// real consumers are the pipeline and runner M1b tests.
func TestNewWritesLockedFixture(t *testing.T) {
	app, shapes := New(t, map[string]string{"app/extra.ail": "module extra\n"})
	for _, rel := range []string{filepath.Join(app, "main.ail"), filepath.Join(app, "extra.ail"), filepath.Join(shapes, ShapeFile)} {
		if _, err := os.Stat(rel); err != nil {
			t.Fatalf("fixture file missing: %v", err)
		}
	}
	lf, err := pkg.LoadLockFile(app)
	if err != nil {
		t.Fatal(err)
	}
	locked, ok := lf.FindPackage("acme/shapes")
	if !ok || locked.Source != "path" || locked.Path != "../shapes" {
		t.Fatalf("lock entry = %+v", locked)
	}
	if err := lf.ValidateContentHashesFrom(app); err != nil {
		t.Fatalf("lock content hash should match the written package: %v", err)
	}
	if !strings.Contains(Files()["shapes/"+ShapeFile], InitLog) {
		t.Fatal("shape module must log InitLog from its module-level binding")
	}
}
