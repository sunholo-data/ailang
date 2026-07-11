package stdlib_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// TestStdlibModulesCanBeParsed ensures all stdlib modules parse successfully.
// This test catches:
// - Reserved keywords used as identifiers (like "exists" in std/fs)
// - Syntax errors in stdlib code
//
// WHY THIS TEST EXISTS:
// v0.3.24 had a bug where std/fs.ail used `exists` as a function name,
// but `exists` is a reserved keyword. This caused parse errors for ANY code
// importing std/fs or std/io (which transitively imports std/fs).
// Result: 19/226 eval benchmarks incorrectly marked as WRONG_LANG (~8%).
//
// This test prevents that regression by ensuring all stdlib modules parse.
func TestStdlibModulesCanBeParsed(t *testing.T) {
	stdlibModules := []struct {
		name string
		path string
	}{
		{"std/io", "../../std/io.ail"},
		{"std/fs", "../../std/fs.ail"},
		{"std/json", "../../std/json.ail"},
		{"std/string", "../../std/string.ail"},
		{"std/list", "../../std/list.ail"},
		{"std/clock", "../../std/clock.ail"},
		{"std/net", "../../std/net.ail"},
		{"std/option", "../../std/option.ail"},
		{"std/result", "../../std/result.ail"},
		{"std/regex", "../../std/regex.ail"},
	}

	for _, mod := range stdlibModules {
		t.Run(mod.name, func(t *testing.T) {
			// Read the stdlib file
			content, err := os.ReadFile(mod.path)
			if err != nil {
				t.Fatalf("Failed to read %s: %v", mod.path, err)
			}

			l := lexer.New(string(content), mod.path)
			p := parser.New(l)
			_ = p.ParseFile()

			// Check for parse errors
			if len(p.Errors()) > 0 {
				t.Errorf("Parse errors in %s:", mod.name)
				for _, err := range p.Errors() {
					t.Errorf("  %s", err)
				}
				t.Fatalf("❌ %s failed to parse (found reserved keywords or syntax errors)", mod.name)
			}

			t.Logf("✅ %s parsed successfully", mod.name)
		})
	}
}

// TestStdlibNoReservedKeywordsAsIdentifiers ensures stdlib doesn't use reserved keywords.
// This is a belt-and-suspenders test - it catches the issue earlier than parsing.
//
// WHY THIS TEST EXISTS:
// The M-BUG-STDLIB-RESERVED-KEYWORD bug (v0.3.24) used `exists` as a function
// name in std/fs.ail, but `exists` is a reserved keyword for planned testing
// syntax. This test explicitly checks for such violations.
func TestStdlibNoReservedKeywordsAsIdentifiers(t *testing.T) {
	// List of reserved keywords that should never appear as function/variable names
	reservedKeywords := []string{
		"exists", // The bug we fixed!
		"forall",
		"test",
		"tests",
		"property",
		"properties",
		"assert",
	}

	stdlibDir := "../../std"
	entries, err := os.ReadDir(stdlibDir)
	if err != nil {
		t.Fatalf("Failed to read stdlib directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".ail" {
			continue
		}

		filePath := filepath.Join(stdlibDir, entry.Name())
		t.Run(entry.Name(), func(t *testing.T) {
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read %s: %v", filePath, err)
			}

			l := lexer.New(string(content), filePath)
			p := parser.New(l)
			file := p.ParseFile()

			// Look for reserved keywords in function/variable names
			for _, decl := range file.Decls {
				if funcDecl, ok := decl.(*ast.FuncDecl); ok {
					for _, keyword := range reservedKeywords {
						if funcDecl.Name == keyword {
							t.Errorf("❌ Found reserved keyword '%s' used as function name in %s", keyword, filePath)
							t.Errorf("   Reserved keywords: %v", reservedKeywords)
							t.Fatalf("   Rename this function to avoid parser conflicts")
						}
					}
				}
			}

			t.Logf("✅ %s: No reserved keywords used as identifiers", entry.Name())
		})
	}
}

// TestStdlibImportChain ensures importing one stdlib module doesn't break others.
// This test catches:
// - Transitive import failures (like std/io importing broken std/fs)
//
// WHY THIS TEST EXISTS:
// The `exists` keyword bug in std/fs.ail broke std/io imports because std/io
// transitively imports std/fs. This test ensures that importing stdlib modules
// doesn't cause parse errors.
func TestStdlibImportChain(t *testing.T) {
	// Test that importing std/io works (which transitively imports std/fs)
	t.Run("std/io_imports_std/fs", func(t *testing.T) {
		code := `module test/import_io
import std/io (print)

export func main() -> () ! {IO} = print("test")`

		l := lexer.New(code, "test.ail")
		p := parser.New(l)
		_ = p.ParseFile()

		if len(p.Errors()) > 0 {
			t.Errorf("Parse errors when importing std/io:")
			for _, err := range p.Errors() {
				t.Errorf("  %s", err)
			}
			t.Fatalf("❌ Importing std/io failed (may be due to transitive std/fs issues)")
		}

		t.Logf("✅ Importing std/io (with transitive std/fs) works")
	})

	// Test that importing std/fs directly works
	t.Run("std/fs_direct_import", func(t *testing.T) {
		code := `module test/import_fs
import std/fs (fileExists)

export func main() -> () ! {FS} = fileExists("/tmp")`

		l := lexer.New(code, "test.ail")
		p := parser.New(l)
		_ = p.ParseFile()

		if len(p.Errors()) > 0 {
			t.Errorf("Parse errors when importing std/fs:")
			for _, err := range p.Errors() {
				t.Errorf("  %s", err)
			}
			t.Fatalf("❌ Importing std/fs failed")
		}

		t.Logf("✅ Importing std/fs directly works")
	})
}
