package loader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathMatchesPackagePrefix(t *testing.T) {
	tests := []struct {
		canonPath string
		pkgName   string
		want      bool
	}{
		{"sunholo/linkedin", "sunholo/linkedin", true},
		{"sunholo/linkedin/types", "sunholo/linkedin", true},
		{"sunholo/linkedin/services/auth", "sunholo/linkedin", true},
		// boundary: prefix must end on a / so foo doesn't match foo_other
		{"sunholo/linkedin_other", "sunholo/linkedin", false},
		{"sunholo/linkedin_other/types", "sunholo/linkedin", false},
		// non-matching vendor
		{"sunholo/firestore/client", "sunholo/linkedin", false},
		// std/* and pkg/* never match
		{"std/io", "sunholo/linkedin", false},
		{"pkg/sunholo/linkedin/types", "sunholo/linkedin", false},
	}
	for _, tt := range tests {
		got := pathMatchesPackagePrefix(tt.canonPath, tt.pkgName)
		if got != tt.want {
			t.Errorf("pathMatchesPackagePrefix(%q, %q) = %v, want %v",
				tt.canonPath, tt.pkgName, got, tt.want)
		}
	}
}

func TestLoad_BareCanonicalSelfImport(t *testing.T) {
	// The LinkedIn case: `import sunholo/linkedin/types` (no pkg/ prefix)
	// from within the `sunholo/linkedin` package routes through the package
	// resolver's self-reference path.
	tmpDir := t.TempDir()
	typesFile := filepath.Join(tmpDir, "types.ail")
	if err := os.WriteFile(typesFile, []byte("module sunholo/linkedin/types\nexport func name() = \"Foo\"\n"), 0644); err != nil {
		t.Fatalf("write types.ail: %v", err)
	}

	ml := NewModuleLoader(tmpDir)
	ml.SetPackageResolver(&mockPkgResolver{
		files: map[string]string{
			// Self-resolution: bare `sunholo/linkedin/types` is routed via
			// pkgLoader.ResolveImport(canonPath). The pkg loader doesn't
			// receive a pkg/ prefix; the canonical pkg-relative form IS the
			// canonPath here.
			"sunholo/linkedin/types": typesFile,
		},
	})
	ml.SetCurrentPackageName("sunholo/linkedin")

	loaded, err := ml.Load("sunholo/linkedin/types")
	if err != nil {
		t.Fatalf("bare canonical self-import must resolve, got: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil LoadedModule")
	}
	if _, ok := loaded.Exports["name"]; !ok {
		t.Error("expected 'name' in exports")
	}
}

func TestLoad_BareCanonicalDoesNotMatchDifferentPackage(t *testing.T) {
	// `import sunholo/firestore/client` from within `sunholo/linkedin`
	// must NOT be treated as self-reference. The prefix-match boundary check
	// should reject it cleanly; the load then falls through to project-
	// relative resolution (and fails normally there).
	tmpDir := t.TempDir()

	ml := NewModuleLoader(tmpDir)
	ml.SetPackageResolver(&mockPkgResolver{files: map[string]string{}})
	ml.SetCurrentPackageName("sunholo/linkedin")

	_, err := ml.Load("sunholo/firestore/client")
	if err == nil {
		t.Fatal("non-matching prefix must NOT silently resolve via self-reference path")
	}
	// Error must come from the normal project-relative miss, not from our
	// self-reference branch's surfaced resolver error.
	if strings.Contains(err.Error(), "self(") {
		t.Errorf("error %q should not mention self-reference for a non-matching package", err.Error())
	}
}

func TestLoad_BareCanonicalDoesNotMatchSimilarPrefix(t *testing.T) {
	// Regression guard for the boundary check in pathMatchesPackagePrefix:
	// `sunholo/linkedin_other` must NOT match current package `sunholo/linkedin`.
	tmpDir := t.TempDir()

	ml := NewModuleLoader(tmpDir)
	ml.SetPackageResolver(&mockPkgResolver{
		files: map[string]string{
			// Even if this exists in the resolver, the path-match check
			// should reject it before we get here.
			"sunholo/linkedin_other/types": "/should/not/reach",
		},
	})
	ml.SetCurrentPackageName("sunholo/linkedin")

	_, err := ml.Load("sunholo/linkedin_other/types")
	if err == nil {
		t.Fatal("similar-but-different prefix must NOT route through self-reference")
	}
}

func TestLoad_NoCurrentPackageNameSkipsSelfBranch(t *testing.T) {
	// When SetCurrentPackageName hasn't been called (e.g. bare project, no
	// ailang.toml), the bare-canonical branch must NOT fire — falls through
	// to existing behaviour (modulePrefixMap or project-relative).
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "sunholo", "linkedin", "types.ail")
	if err := os.MkdirAll(filepath.Dir(localFile), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(localFile, []byte("module sunholo/linkedin/types\nexport func n() = \"x\"\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	ml := NewModuleLoader(tmpDir)
	// pkgLoader set but no currentPackageName — bare path must fall to
	// project-relative resolution.
	ml.SetPackageResolver(&mockPkgResolver{files: map[string]string{}})

	loaded, err := ml.Load("sunholo/linkedin/types")
	if err != nil {
		t.Fatalf("expected project-relative fallback to succeed, got: %v", err)
	}
	if _, ok := loaded.Exports["n"]; !ok {
		t.Error("expected 'n' in exports (project-relative path)")
	}
}
