package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func freezeFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "prompts"), 0o755)
	os.MkdirAll(filepath.Join(root, "cmd", "ailang", "prompts"), 0o755)
	r := &orderedRegistry{SchemaVersion: "1.0", Versions: map[string]*registryEntry{}, RawEntries: map[string]json.RawMessage{}, Active: "active", Notes: []string{"fixture"}}
	for _, id := range []string{"b1", "b2", "b3", "l1", "l2", "l3", "l4", "active"} {
		content := []byte("prompt " + id)
		h := sha256.Sum256(content)
		e := &registryEntry{File: "prompts/" + id + ".md", Hash: hex.EncodeToString(h[:]), Description: id, Created: "2026-01-01", Tags: []string{"test"}, Notes: "fixture"}
		r.VersionKeys = append(r.VersionKeys, id)
		r.Versions[id] = e
		os.WriteFile(filepath.Join(root, e.File), content, 0o644)
		os.WriteFile(filepath.Join(root, "cmd", "ailang", e.File), content, 0o644)
	}
	writeOrderedRegistry(filepath.Join(root, "prompts", "versions.json"), r)
	writeOrderedRegistry(filepath.Join(root, "cmd", "ailang", "prompts", "versions.json"), r)
	runFreezeCheckGit(t, root, "init", "-q")
	runFreezeCheckGit(t, root, "config", "user.email", "freeze-check@example.invalid")
	runFreezeCheckGit(t, root, "config", "user.name", "Freeze Check")
	runFreezeCheckGit(t, root, "add", ".")
	runFreezeCheckGit(t, root, "commit", "-q", "-m", "base")
	runFreezeCheckGit(t, root, "update-ref", "refs/remotes/origin/dev", "HEAD")
	old := corpusScanner
	corpusScanner = func(string) (map[string]corpusEvidence, error) {
		return map[string]corpusEvidence{"b1": {210, "eval_results/baselines/a.json"}, "b2": {38, "eval_results/baselines/b.json"}, "b3": {1, "eval_results/baselines/c.json"}}, nil
	}
	t.Cleanup(func() { corpusScanner = old })
	return root
}
func split(r *orderedRegistry) (int, int, int, string) {
	b, l, m, mID := 0, 0, 0, ""
	for id, e := range r.Versions {
		if e.Frozen == nil {
			m++
			mID = id
		} else if e.Frozen.Reason == "banked" {
			b++
		} else if e.Frozen.Reason == "legacy" {
			l++
		}
	}
	return b, l, m, mID
}

func TestPromptFreezeMigrate_SplitCounts(t *testing.T) {
	root := freezeFixture(t)
	path := filepath.Join(root, "prompts", "versions.json")
	before, _ := os.ReadFile(path)
	r, _ := loadOrderedRegistry(path)
	writeOrderedRegistry(path, r)
	round, _ := os.ReadFile(path)
	if !bytes.Equal(before, round) {
		t.Fatal("ordered writer changed unmigrated input")
	}
	b, l, m, err := migrateRegistries(root, "2026-08-27")
	if err != nil || b != 3 || l != 4 || m != 1 {
		t.Fatalf("%d/%d/%d %v", b, l, m, err)
	}
	r, _ = loadOrderedRegistry(path)
	b, l, m, id := split(r)
	if b != 3 || l != 4 || m != 1 || id != "active" || r.Versions["b1"].Frozen.EvidenceCount != 210 {
		t.Fatalf("bad split %d/%d/%d %s", b, l, m, id)
	}
}
func TestPromptFreezeMigrate_WritesBothRegistries(t *testing.T) {
	root := freezeFixture(t)
	migrateRegistries(root, "2026-08-27")
	a, _ := os.ReadFile(filepath.Join(root, "prompts", "versions.json"))
	b, _ := os.ReadFile(filepath.Join(root, "cmd", "ailang", "prompts", "versions.json"))
	if len(b) == 0 || !bytes.Equal(a, b) {
		t.Fatal("registries differ")
	}
	r, _ := loadOrderedRegistry(filepath.Join(root, "cmd", "ailang", "prompts", "versions.json"))
	x, y, z, _ := split(r)
	if x != 3 || y != 4 || z != 1 {
		t.Fatalf("mirror split %d/%d/%d", x, y, z)
	}
}
func TestPromptFreezeMigrate_RefusesStaleHash(t *testing.T) {
	root := freezeFixture(t)
	p := filepath.Join(root, "prompts", "b1.md")
	os.WriteFile(p, []byte("tampered"), 0o644)
	aPath := filepath.Join(root, "prompts", "versions.json")
	bPath := filepath.Join(root, "cmd", "ailang", "prompts", "versions.json")
	a, _ := os.ReadFile(aPath)
	b, _ := os.ReadFile(bPath)
	_, _, _, err := migrateRegistries(root, "2026-08-27")
	if err == nil || !strings.Contains(err.Error(), "b1") {
		t.Fatalf("unexpected %v", err)
	}
	aa, _ := os.ReadFile(aPath)
	bb, _ := os.ReadFile(bPath)
	if !bytes.Equal(a, aa) || !bytes.Equal(b, bb) {
		t.Fatal("registry changed on refusal")
	}
}
func TestPromptFreezeMigrate_Idempotent(t *testing.T) {
	root := freezeFixture(t)
	migrateRegistries(root, "2026-08-27")
	aPath := filepath.Join(root, "prompts", "versions.json")
	a, _ := os.ReadFile(aPath)
	migrateRegistries(root, "2026-08-28")
	b, _ := os.ReadFile(aPath)
	if !bytes.Equal(a, b) {
		t.Fatal("second migration changed bytes")
	}
}
func TestPromptFreezeCheck_GreenOnMigratedFixture(t *testing.T) {
	root := freezeFixture(t)
	migrateRegistries(root, "2026-08-27")
	v, err := checkRegistries(root)
	if err != nil || len(v) != 0 {
		t.Fatalf("%v %v", v, err)
	}
}
func TestPromptFreezeCheck_RedOnMissingMarker(t *testing.T) {
	root := freezeFixture(t)
	migrateRegistries(root, "2026-08-27")
	pair := promptRegistryPair(root)
	r, _ := loadOrderedRegistry(pair.Source)
	r.Versions["b1"].Frozen = nil
	writeOrderedRegistry(pair.Source, r)
	writeOrderedRegistry(pair.Mirror, r)
	v, err := checkRegistries(root)
	if err != nil || len(v) != 1 || !strings.Contains(v[0], "b1") || !strings.Contains(v[0], "corpus-evidenced but not frozen") {
		t.Fatalf("%v %v", v, err)
	}
}
func TestRealRegistry_PostMigrationSplitCounts(t *testing.T) {
	source, err := loadOrderedRegistry(filepath.Join("..", "..", "prompts", "versions.json"))
	if err != nil {
		t.Fatal(err)
	}
	mirror, err := loadOrderedRegistry(filepath.Join("prompts", "versions.json"))
	if err != nil {
		t.Fatal(err)
	}
	b, l, m, id := split(source)
	if b != 19 || l != 39 || m != 1 || id != "v0.16.6" {
		t.Fatalf("split %d/%d/%d %s", b, l, m, id)
	}
	a, _ := json.Marshal(source.Versions)
	z, _ := json.Marshal(mirror.Versions)
	if !bytes.Equal(a, z) {
		t.Fatal("registries differ")
	}
	h, err := fileHash(filepath.Join("..", "..", source.Versions["aver"].File))
	if err != nil || h != source.Versions["aver"].Hash {
		t.Fatalf("aver hash %s %v", h, err)
	}
}
