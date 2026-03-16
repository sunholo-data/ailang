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
