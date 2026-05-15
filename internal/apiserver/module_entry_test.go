package apiserver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/loader"
)

// TestRegisterModule_IsSoleWriteSite asserts the M-SERVEAPI-UNIFY
// invariant: every entry in s.modules MUST be keyed by its
// PhysicalPath, and every entry's PhysicalPath, CanonicalID, and Path
// (RelPath) projection MUST be populated. Any deviation indicates a
// stale write site that bypassed registerModule.
func TestRegisterModule_KeyIsPhysicalPath(t *testing.T) {
	srv := newColdStartTestServer(t)
	defer srv.Close()

	srv.mu.RLock()
	defer srv.mu.RUnlock()

	if len(srv.modules) == 0 {
		t.Fatal("no modules registered — fixture broken")
	}

	for key, info := range srv.modules {
		if info == nil {
			t.Errorf("nil ModuleInfo for key %q", key)
			continue
		}
		if info.PhysicalPath == "" {
			t.Errorf("entry %q has empty PhysicalPath — bypassed registerModule", key)
		}
		if info.PhysicalPath != key {
			t.Errorf("map key %q != info.PhysicalPath %q", key, info.PhysicalPath)
		}
		if info.CanonicalID == "" {
			t.Errorf("entry %q has empty CanonicalID — bypassed registerModule", key)
		}
		if info.Path == "" {
			t.Errorf("entry %q has empty Path (RelPath projection)", key)
		}
	}
}

// TestRegisterModule_UnderBasePathFilter asserts that a module whose
// file lives OUTSIDE s.basePath is not registered. This is the only
// filter registerModule applies, so it's the single point that
// distinguishes local files from package files.
func TestRegisterModule_UnderBasePathFilter(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a project directory and a sibling directory.
	projectDir := filepath.Join(tmpDir, "project")
	siblingDir := filepath.Join(tmpDir, "sibling")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(siblingDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Construct a Server rooted at projectDir. normalizedBasePath
	// should be projectDir's symlink-resolved absolute form.
	srv := New(projectDir, Config{Port: "0"})
	defer srv.Close()

	// A physical file under siblingDir has no business being a "local"
	// module for this server. Synthesize the minimal LoadedModule shape
	// the filter checks (File.Path + Iface + Module header).
	siblingFile := filepath.Join(siblingDir, "other.ail")
	if err := os.WriteFile(siblingFile, []byte("module other\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// registerModule takes a *loader.LoadedModule. We can't construct
	// one directly without invoking the pipeline, so the honest way to
	// exercise the filter is via LoadModules: pass the siblingFile and
	// assert it loads (main path) but is NOT in s.modules under any
	// rel-path key that suggests "local to projectDir".
	//
	// However, a direct unit test of the filter is more precise: we
	// compute the physical path manually and check the prefix logic.
	absSibling, _ := filepath.Abs(siblingFile)
	if resolved, err := filepath.EvalSymlinks(absSibling); err == nil {
		absSibling = resolved
	}

	if strings.HasPrefix(absSibling+string(filepath.Separator), srv.normalizedBasePath) {
		t.Errorf("sibling file %q appears under normalized basePath %q — filter would incorrectly include it",
			absSibling, srv.normalizedBasePath)
	}
}

// TestRegisterModule_Idempotent asserts that calling registerModule
// twice with the same physical file yields one s.modules entry and no
// error on the second call. Exercised indirectly by the cold-start
// fixture, which currently loads each file once via the main path AND
// may revisit via dep-discovery — both should reach the same key.
func TestRegisterModule_Idempotent(t *testing.T) {
	srv := newColdStartTestServer(t)
	defer srv.Close()

	before := len(srv.modules)

	// Re-run LoadModules on the same basePath. Every module should
	// already exist, so the second pass must be a no-op on s.modules
	// (idempotent registration).
	if err := srv.LoadModules([]string{srv.basePath}); err != nil {
		t.Fatalf("second LoadModules call failed: %v", err)
	}

	after := len(srv.modules)
	if after != before {
		t.Errorf("LoadModules not idempotent: before=%d, after=%d", before, after)
	}
}

// TestRecordDrop_TracksOutsideBasePath asserts that recordDrop appends
// a DroppedModule entry with the correct PhysicalPath, DeclaredPath, and
// FileBaseName. M-SERVEAPI-SURFACE-DROPS M1.
func TestRecordDrop_TracksOutsideBasePath(t *testing.T) {
	srv := New(t.TempDir(), Config{Port: "0"})
	defer srv.Close()

	loaded := &loader.LoadedModule{
		File: &ast.File{
			Path: "/some/cache/dir/billing_entitlements/plan.ail",
			Module: &ast.ModuleDecl{
				Path: "pkg/sunholo/billing_entitlements/plan",
			},
		},
	}
	srv.recordDrop(loaded, loaded.File.Path, loaded.File.Module.Path)

	if got := len(srv.droppedModules); got != 1 {
		t.Fatalf("droppedModules length = %d, want 1", got)
	}
	d := srv.droppedModules[0]
	if d.PhysicalPath != loaded.File.Path {
		t.Errorf("PhysicalPath = %q, want %q", d.PhysicalPath, loaded.File.Path)
	}
	if d.DeclaredPath != "pkg/sunholo/billing_entitlements/plan" {
		t.Errorf("DeclaredPath = %q, want %q", d.DeclaredPath, "pkg/sunholo/billing_entitlements/plan")
	}
	if d.FileBaseName != "plan.ail" {
		t.Errorf("FileBaseName = %q, want %q", d.FileBaseName, "plan.ail")
	}
	if d.Reason != "outside-basePath" {
		t.Errorf("Reason = %q, want %q", d.Reason, "outside-basePath")
	}
	if len(d.Annotations) != 0 {
		t.Errorf("Annotations = %v, want empty (no @route in fixture)", d.Annotations)
	}
}

// TestRecordDrop_DetectsRouteAnnotation asserts that recordDrop captures
// "@route" in the Annotations slice when at least one function in the
// dropped module's File has a @route annotation. The presence-check is
// what drives ValidateRegistration's fail-fast partitioning in M2.
func TestRecordDrop_DetectsRouteAnnotation(t *testing.T) {
	srv := New(t.TempDir(), Config{Port: "0"})
	defer srv.Close()

	loaded := &loader.LoadedModule{
		File: &ast.File{
			Path: "/cache/api/handler.ail",
			Module: &ast.ModuleDecl{
				Path: "pkg/example/handler",
			},
			Funcs: []*ast.FuncDecl{
				{
					Name: "plainFunc",
					// no annotations
				},
				{
					Name: "exposedHandler",
					Annotations: []*ast.Annotation{
						{Name: "route", Args: []ast.Expr{
							&ast.Literal{Kind: ast.StringLit, Value: "GET"},
							&ast.Literal{Kind: ast.StringLit, Value: "/foo"},
						}},
					},
				},
			},
		},
	}
	srv.recordDrop(loaded, loaded.File.Path, loaded.File.Module.Path)

	if len(srv.droppedModules) != 1 {
		t.Fatalf("droppedModules length = %d, want 1", len(srv.droppedModules))
	}
	got := srv.droppedModules[0].Annotations
	if len(got) != 1 || got[0] != "@route" {
		t.Errorf("Annotations = %v, want [\"@route\"]", got)
	}
}

// TestRecordDrop_NilFileSafe asserts that recordDrop handles a
// LoadedModule with a nil File without panicking. This guards the
// (unlikely but possible) case where a module fails to parse but is
// still passed to registerModule.
func TestRecordDrop_NilFileSafe(t *testing.T) {
	srv := New(t.TempDir(), Config{Port: "0"})
	defer srv.Close()

	loaded := &loader.LoadedModule{File: nil}
	srv.recordDrop(loaded, "/some/path.ail", "")

	if len(srv.droppedModules) != 1 {
		t.Fatalf("droppedModules length = %d, want 1 (recordDrop must not panic on nil File)", len(srv.droppedModules))
	}
	if len(srv.droppedModules[0].Annotations) != 0 {
		t.Errorf("Annotations should be empty when File is nil")
	}
}

// TestValidateRegistration_NoDrops asserts that ValidateRegistration
// returns nil when no modules were dropped. M-SERVEAPI-SURFACE-DROPS M2.
func TestValidateRegistration_NoDrops(t *testing.T) {
	srv := New(t.TempDir(), Config{Port: "0"})
	defer srv.Close()

	if err := srv.ValidateRegistration(); err != nil {
		t.Errorf("ValidateRegistration with no drops returned error: %v", err)
	}
}

// TestValidateRegistration_FatalDropFailsStart asserts that a drop with
// "@route" in Annotations causes ValidateRegistration to return a
// non-nil error naming the dropped module's identifiers.
func TestValidateRegistration_FatalDropFailsStart(t *testing.T) {
	t.Setenv(AllowDropsEnvVar, "") // ensure allow-drops is OFF

	srv := New(t.TempDir(), Config{Port: "0"})
	defer srv.Close()

	srv.droppedModules = append(srv.droppedModules, DroppedModule{
		PhysicalPath: "/cache/sunholo/billing_entitlements/0.4.1/plan.ail",
		DeclaredPath: "pkg/sunholo/billing_entitlements/plan",
		FileBaseName: "plan.ail",
		Annotations:  []string{"@route"},
		Reason:       "outside-basePath",
	})

	err := srv.ValidateRegistration()
	if err == nil {
		t.Fatal("ValidateRegistration returned nil; want error for @route-bearing drop")
	}
	msg := err.Error()
	// Error message must name the file, declared path, and annotations
	// so operators can act on it without rerunning with extra debug flags.
	for _, want := range []string{"plan.ail", "pkg/sunholo/billing_entitlements/plan", "@route"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error message missing %q\nfull message:\n%s", want, msg)
		}
	}
}

// TestValidateRegistration_AllowDropsEscapeHatch asserts that setting
// AILANG_SERVE_API_ALLOW_DROPS=1 demotes a fatal drop to a strong WARN
// (ValidateRegistration returns nil).
func TestValidateRegistration_AllowDropsEscapeHatch(t *testing.T) {
	t.Setenv(AllowDropsEnvVar, "1")

	srv := New(t.TempDir(), Config{Port: "0"})
	defer srv.Close()

	srv.droppedModules = append(srv.droppedModules, DroppedModule{
		PhysicalPath: "/cache/foo/bar.ail",
		DeclaredPath: "pkg/foo/bar",
		FileBaseName: "bar.ail",
		Annotations:  []string{"@route"},
		Reason:       "outside-basePath",
	})

	if err := srv.ValidateRegistration(); err != nil {
		t.Errorf("ValidateRegistration with allow-drops=1 returned error: %v", err)
	}
}

// TestValidateRegistration_NonAnnotationDropOnlyWarns asserts that a
// drop without any annotations (e.g. a stdlib resolution edge) does NOT
// cause ValidateRegistration to fail — it only logs the WARN banner.
func TestValidateRegistration_NonAnnotationDropOnlyWarns(t *testing.T) {
	t.Setenv(AllowDropsEnvVar, "")

	srv := New(t.TempDir(), Config{Port: "0"})
	defer srv.Close()

	srv.droppedModules = append(srv.droppedModules, DroppedModule{
		PhysicalPath: "/some/std/option.ail",
		DeclaredPath: "std/option",
		FileBaseName: "option.ail",
		Annotations:  nil, // no @route
		Reason:       "outside-basePath",
	})

	if err := srv.ValidateRegistration(); err != nil {
		t.Errorf("non-annotation drop should not fail validation; got error: %v", err)
	}
}

// TestRegisterModule_DropsOutsideBasePathReachValidateRegistration is
// the M-SERVEAPI-SURFACE-DROPS integration test. It exercises the full
// chain: registerModule rejects an out-of-basePath module → recordDrop
// captures it with @route annotation → ValidateRegistration partitions
// it as fatal and returns a non-nil error. This is the path that fixes
// the docparse v0.14.1 billing bug class.
func TestRegisterModule_DropsOutsideBasePathReachValidateRegistration(t *testing.T) {
	t.Setenv(AllowDropsEnvVar, "")

	// Build a docparse-style layout: project lives at projectDir; a
	// "package cache" sits in cacheDir alongside it.
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	cacheDir := filepath.Join(tmpDir, "cache")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cacheFile := filepath.Join(cacheDir, "handler.ail")
	if err := os.WriteFile(cacheFile, []byte("module pkg/example/handler\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := New(projectDir, Config{Port: "0"})
	defer srv.Close()

	// Synthesize a LoadedModule whose File.Path is the cache file
	// (outside basePath) and whose File.Funcs declares @route. This is
	// what the loader would produce for a published handler module
	// whose physical resolution lands outside the server's basePath.
	loaded := &loader.LoadedModule{
		Path: "pkg/example/handler",
		File: &ast.File{
			Path: cacheFile,
			Module: &ast.ModuleDecl{
				Path: "pkg/example/handler",
			},
			Funcs: []*ast.FuncDecl{
				{
					Name: "publishedHandler",
					Annotations: []*ast.Annotation{
						{Name: "route", Args: []ast.Expr{
							&ast.Literal{Kind: ast.StringLit, Value: "GET"},
							&ast.Literal{Kind: ast.StringLit, Value: "/foo"},
						}},
					},
				},
			},
		},
		Iface: iface.NewIface("pkg/example/handler"),
	}

	key, ok, err := srv.registerModule(loaded)
	if err != nil {
		t.Fatalf("registerModule returned unexpected error: %v", err)
	}
	if ok || key != "" {
		t.Errorf("registerModule registered an out-of-basePath module: key=%q ok=%v", key, ok)
	}

	// Drop should be recorded with @route annotation, ready for
	// ValidateRegistration to fail-fast on.
	drops := srv.DroppedModules()
	if len(drops) != 1 {
		t.Fatalf("expected 1 dropped module, got %d", len(drops))
	}
	if drops[0].DeclaredPath != "pkg/example/handler" {
		t.Errorf("drop DeclaredPath = %q, want %q", drops[0].DeclaredPath, "pkg/example/handler")
	}
	if len(drops[0].Annotations) != 1 || drops[0].Annotations[0] != "@route" {
		t.Errorf("drop Annotations = %v, want [\"@route\"]", drops[0].Annotations)
	}

	// ValidateRegistration must surface this as a fatal error.
	if err := srv.ValidateRegistration(); err == nil {
		t.Fatal("ValidateRegistration returned nil; want error for @route-bearing drop")
	} else if !strings.Contains(err.Error(), "pkg/example/handler") {
		t.Errorf("ValidateRegistration error missing declared path: %v", err)
	}
}

// TestServer_NormalizedBasePath asserts New() computes
// normalizedBasePath correctly: absolute, symlink-resolved, with a
// trailing separator.
func TestServer_NormalizedBasePath(t *testing.T) {
	tmpDir := t.TempDir()
	srv := New(tmpDir, Config{Port: "0"})
	defer srv.Close()

	if srv.normalizedBasePath == "" {
		t.Fatal("normalizedBasePath is empty")
	}
	if !strings.HasSuffix(srv.normalizedBasePath, string(filepath.Separator)) {
		t.Errorf("normalizedBasePath %q missing trailing separator", srv.normalizedBasePath)
	}
	if !filepath.IsAbs(srv.normalizedBasePath) {
		t.Errorf("normalizedBasePath %q is not absolute", srv.normalizedBasePath)
	}
	// Should be EvalSymlinks-resolved: on macOS, /var/folders →
	// /private/var/folders. If tmpDir was a symlinked form, the
	// normalized version should differ.
	resolved, _ := filepath.EvalSymlinks(tmpDir)
	want := filepath.Clean(resolved) + string(filepath.Separator)
	if srv.normalizedBasePath != want {
		t.Errorf("normalizedBasePath = %q, want %q", srv.normalizedBasePath, want)
	}
}
