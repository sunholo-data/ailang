package pipeline

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/loader"
)

// makeLoadedModule builds a minimal LoadedModule with a declared module header
// suitable for feeding detectModulePathCollisions.
func makeLoadedModule(canonicalID, filePath, declared string) *loader.LoadedModule {
	file := &ast.File{
		Path: filePath,
	}
	if declared != "" {
		file.Module = &ast.ModuleDecl{Path: declared}
	}
	return &loader.LoadedModule{
		Path: canonicalID,
		File: file,
	}
}

func TestDetectModulePathCollisions_NoCollision(t *testing.T) {
	mods := map[string]*loader.LoadedModule{
		"docparse/services/mcp_tools": makeLoadedModule(
			"docparse/services/mcp_tools",
			"/project/docparse/services/mcp_tools.ail",
			"docparse/services/mcp_tools",
		),
		"docparse/services/auth": makeLoadedModule(
			"docparse/services/auth",
			"/project/docparse/services/auth.ail",
			"docparse/services/auth",
		),
	}
	if err := detectModulePathCollisions(mods); err != nil {
		t.Fatalf("unexpected collision: %v", err)
	}
}

func TestDetectModulePathCollisions_SameFileTwice(t *testing.T) {
	// The same source file may legitimately appear under two canonical IDs
	// (e.g., absolute path + module_prefix-resolved path). Not a collision.
	mods := map[string]*loader.LoadedModule{
		"Users/dev/docparse/services/mcp_tools": makeLoadedModule(
			"Users/dev/docparse/services/mcp_tools",
			"/Users/dev/docparse/services/mcp_tools.ail",
			"docparse/services/mcp_tools",
		),
		"docparse/services/mcp_tools": makeLoadedModule(
			"docparse/services/mcp_tools",
			"/Users/dev/docparse/services/mcp_tools.ail",
			"docparse/services/mcp_tools",
		),
	}
	if err := detectModulePathCollisions(mods); err != nil {
		t.Fatalf("same-file re-loading should not be a collision: %v", err)
	}
}

func TestDetectModulePathCollisions_LocalVsPackage(t *testing.T) {
	// Two different files both declaring the same module header — this is
	// the docparse footgun. Must error.
	mods := map[string]*loader.LoadedModule{
		"Users/dev/docparse/services/mcp_tools": makeLoadedModule(
			"Users/dev/docparse/services/mcp_tools",
			"/Users/dev/docparse/services/mcp_tools.ail",
			"docparse/services/mcp_tools",
		),
		"pkg/sunholo/ailang_parse/services/mcp_tools": makeLoadedModule(
			"pkg/sunholo/ailang_parse/services/mcp_tools",
			"/home/.ailang/pkg/sunholo/ailang_parse/docparse/services/mcp_tools.ail",
			"docparse/services/mcp_tools",
		),
	}
	err := detectModulePathCollisions(mods)
	if err == nil {
		t.Fatal("expected MOD011 collision error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "MOD011") {
		t.Errorf("error should mention MOD011: %s", msg)
	}
	if !strings.Contains(msg, "docparse/services/mcp_tools") {
		t.Errorf("error should include the colliding module path: %s", msg)
	}
	if !strings.Contains(msg, "/Users/dev/docparse/services/mcp_tools.ail") {
		t.Errorf("error should include the first file path: %s", msg)
	}
	if !strings.Contains(msg, "/home/.ailang/pkg/sunholo/ailang_parse/docparse/services/mcp_tools.ail") {
		t.Errorf("error should include the second file path: %s", msg)
	}
}

func TestDetectModulePathCollisions_NoModuleHeader(t *testing.T) {
	// A module without a `module` header (unusual but possible) should not
	// trip the check.
	mods := map[string]*loader.LoadedModule{
		"a": makeLoadedModule("a", "/a.ail", ""),
		"b": makeLoadedModule("b", "/b.ail", ""),
	}
	if err := detectModulePathCollisions(mods); err != nil {
		t.Fatalf("header-less modules should not collide: %v", err)
	}
}

func TestDetectModulePathCollisions_DeterministicOrder(t *testing.T) {
	// The error message must be stable across runs regardless of Go map order.
	// Run the collision detection many times and assert the same error text.
	mods := map[string]*loader.LoadedModule{
		"zzz/later/claimant": makeLoadedModule(
			"zzz/later/claimant",
			"/zzz.ail",
			"shared/path",
		),
		"aaa/earlier/claimant": makeLoadedModule(
			"aaa/earlier/claimant",
			"/aaa.ail",
			"shared/path",
		),
	}
	var first string
	for i := 0; i < 50; i++ {
		err := detectModulePathCollisions(mods)
		if err == nil {
			t.Fatal("expected collision error")
		}
		if i == 0 {
			first = err.Error()
			continue
		}
		if err.Error() != first {
			t.Fatalf("non-deterministic error message:\n  run 0: %s\n  run %d: %s", first, i, err.Error())
		}
	}
	// aaa should be the first claimant (lexical order), zzz the second.
	if !strings.Contains(first, "/aaa.ail") || !strings.Contains(first, "/zzz.ail") {
		t.Errorf("error missing expected file paths: %s", first)
	}
	// aaa should come before zzz in the message.
	if strings.Index(first, "/aaa.ail") > strings.Index(first, "/zzz.ail") {
		t.Errorf("expected aaa to be listed before zzz: %s", first)
	}
}
