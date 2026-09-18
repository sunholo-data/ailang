package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/loader"
)

// TestDeclaredModuleMatchesPackageLayout_AbsoluteEntryFromAnyCwd pins the
// defect email-parse measured on 2026-09-18: an installed [bin] shim runs
// `ailang run --package-dir <pkg> /abs/path/cli.ail`, CanonicalModuleID strips
// the leading "/", and the layout check then rooted the id at the caller's cwd
// — so the shim passed MOD010 from `/` and failed from everywhere else.
//
// The package deliberately sits under t.TempDir(): this function is called
// BEFORE the temp-path relaxation, so it must answer correctly on its own.
func TestDeclaredModuleMatchesPackageLayout_AbsoluteEntryFromAnyCwd(t *testing.T) {
	pkgDir := filepath.Join(t.TempDir(), "greeter")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := "[package]\nname = \"test/greeter\"\nversion = \"0.1.0\"\nedition = \"1\"\n[exports]\nmodules = [\"test/greeter/cli\"]\n"
	if err := os.WriteFile(filepath.Join(pkgDir, "ailang.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(pkgDir, "cli.ail")
	if err := os.WriteFile(file, []byte("module test/greeter/cli\nexport func main() -> () ! {IO} = ()\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// A cwd that is neither "/" nor the package: the only cwd the old code
	// got right was "/".
	elsewhere := t.TempDir()
	cwd, _ := os.Getwd()
	if err := os.Chdir(elsewhere); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	modID := loader.CanonicalModuleID(file)
	if strings.HasPrefix(modID, "/") {
		t.Fatalf("premise: CanonicalModuleID keeps the leading slash (%q); the fallback below is then untested", modID)
	}

	// With the real file path (what the pipeline passes).
	if !declaredModuleMatchesPackageLayout(pkgDir, "test/greeter/cli", modID, file) {
		t.Errorf("absolute entry with file path must match the package layout from cwd %s", elsewhere)
	}
	// Without it: the stripped id alone must still resolve by reattaching "/".
	if !declaredModuleMatchesPackageLayout(pkgDir, "test/greeter/cli", modID, "") {
		t.Errorf("stripped absolute id must match by reattaching the leading slash")
	}
	// A cwd-relative id is still honoured (check --package from inside the package).
	if err := os.Chdir(pkgDir); err != nil {
		t.Fatal(err)
	}
	if !declaredModuleMatchesPackageLayout(pkgDir, "test/greeter/cli", "cli", "") {
		t.Errorf("relative id from inside the package must match")
	}
	// And a wrong declaration is still a mismatch.
	if declaredModuleMatchesPackageLayout(pkgDir, "test/greeter/other", modID, file) {
		t.Errorf("a module declared elsewhere must not match")
	}
}
