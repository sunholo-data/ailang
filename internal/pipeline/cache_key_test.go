package pipeline

import (
	"testing"
)

func TestModuleCacheKey_Deterministic(t *testing.T) {
	key1 := ModuleCacheKey("v0.9.3", "let x = 1", map[string]string{"std/list": "abc123"})
	key2 := ModuleCacheKey("v0.9.3", "let x = 1", map[string]string{"std/list": "abc123"})
	if key1 != key2 {
		t.Errorf("same inputs produced different keys: %s != %s", key1, key2)
	}
}

func TestModuleCacheKey_DifferentSource(t *testing.T) {
	key1 := ModuleCacheKey("v0.9.3", "let x = 1", map[string]string{})
	key2 := ModuleCacheKey("v0.9.3", "let x = 2", map[string]string{})
	if key1 == key2 {
		t.Errorf("different sources produced same key")
	}
}

func TestModuleCacheKey_DifferentVersion(t *testing.T) {
	key1 := ModuleCacheKey("v0.9.3", "let x = 1", map[string]string{})
	key2 := ModuleCacheKey("v0.9.4", "let x = 1", map[string]string{})
	if key1 == key2 {
		t.Errorf("different versions produced same key")
	}
}

func TestModuleCacheKey_DifferentDep(t *testing.T) {
	key1 := ModuleCacheKey("v0.9.3", "let x = 1", map[string]string{"std/list": "abc123"})
	key2 := ModuleCacheKey("v0.9.3", "let x = 1", map[string]string{"std/list": "def456"})
	if key1 == key2 {
		t.Errorf("different dep digests produced same key")
	}
}

func TestModuleCacheKey_DepOrderIndependent(t *testing.T) {
	key1 := ModuleCacheKey("v0.9.3", "let x = 1", map[string]string{
		"std/list":   "aaa",
		"std/option": "bbb",
	})
	key2 := ModuleCacheKey("v0.9.3", "let x = 1", map[string]string{
		"std/option": "bbb",
		"std/list":   "aaa",
	})
	if key1 != key2 {
		t.Errorf("different dep ordering produced different keys: %s != %s", key1, key2)
	}
}

// TestModuleCacheKey_CommitChange verifies that changing the compiler-identity
// component (e.g. the build commit) invalidates the cache key.
//
// Regression test for M-POLY-ORD M1: callers used to pass the format constant
// "v1" as compilerVersion, so rebuilding ailang did not invalidate cache.
// Now pipeline_module.go threads internal/version.Commit here.
func TestModuleCacheKey_CommitChange(t *testing.T) {
	src := "let max = \\x.\\y. if x > y then x else y"
	deps := map[string]string{"std/list": "abc123"}

	keyOldCommit := ModuleCacheKey("commit-aaaaaa", src, deps)
	keyNewCommit := ModuleCacheKey("commit-bbbbbb", src, deps)

	if keyOldCommit == keyNewCommit {
		t.Errorf("changing compiler commit must invalidate cache key: got %s == %s",
			keyOldCommit, keyNewCommit)
	}

	// And the same commit should be stable.
	keyOldCommit2 := ModuleCacheKey("commit-aaaaaa", src, deps)
	if keyOldCommit != keyOldCommit2 {
		t.Errorf("same commit should produce same key: got %s != %s",
			keyOldCommit, keyOldCommit2)
	}
}

func TestModuleCacheKey_NoDeps(t *testing.T) {
	key := ModuleCacheKey("v0.9.3", "let x = 1", nil)
	if key == "" {
		t.Errorf("empty key for nil deps")
	}
	key2 := ModuleCacheKey("v0.9.3", "let x = 1", map[string]string{})
	if key != key2 {
		t.Errorf("nil deps and empty deps produced different keys")
	}
}
