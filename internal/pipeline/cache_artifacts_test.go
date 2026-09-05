package pipeline

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/types"
)

func TestCacheArtifacts_Authorization(t *testing.T) {
	const moduleID = "auth/module"
	const cacheKey = "caller-computed-key"

	t.Run("valid", func(t *testing.T) {
		cs := newArtifactTestStore(t)
		mustStoreArtifacts(t, cs, moduleID, cacheKey, testCachedModule(moduleID, 42))
		got, err := cs.LoadArtifacts(moduleID, cacheKey)
		if err != nil {
			t.Fatalf("valid artifacts rejected: %v", err)
		}
		if gotValue := cachedLiteral(t, got); gotValue != 42 {
			t.Fatalf("cached value = %d, want 42", gotValue)
		}
	})

	cases := []struct {
		name   string
		mutate func(t *testing.T, cs *CacheStore, stampPath string)
		key    string
	}{
		{"missing_stamp", func(t *testing.T, _ *CacheStore, path string) { mustRemove(t, path) }, cacheKey},
		{"corrupt_stamp", func(t *testing.T, _ *CacheStore, path string) { mustWriteArtifactTest(t, path, []byte("{")) }, cacheKey},
		{"wrong_version", mutateStamp(func(stamp *artifactStamp) { stamp.Version = "v3" }), cacheKey},
		{"empty_stored_key", mutateStamp(func(stamp *artifactStamp) { stamp.CacheKey = "" }), cacheKey},
		{"wrong_stored_key", mutateStamp(func(stamp *artifactStamp) { stamp.CacheKey = "other" }), cacheKey},
		{"wrong_expected_key", func(*testing.T, *CacheStore, string) {}, "other"},
		{"wrong_module", mutateStamp(func(stamp *artifactStamp) { stamp.ModuleID = "other/module" }), cacheKey},
		{"extra_digest", mutateStamp(func(stamp *artifactStamp) { stamp.SHA256["extra"] = strings.Repeat("0", 64) }), cacheKey},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cs := newArtifactTestStore(t)
			mustStoreArtifacts(t, cs, moduleID, cacheKey, testCachedModule(moduleID, 42))
			stampPath := filepath.Join(cs.moduleArtifactDir(moduleID), artifactStampName)
			tc.mutate(t, cs, stampPath)
			if got, err := cs.LoadArtifacts(moduleID, tc.key); err == nil || got != nil {
				t.Fatalf("unauthorized artifacts loaded: got=%v err=%v", got, err)
			}
		})
	}

	t.Run("sanitized_collision_uses_exact_module_id", func(t *testing.T) {
		cs := newArtifactTestStore(t)
		mustStoreArtifacts(t, cs, "a/b", cacheKey, testCachedModule("a/b", 42))
		if sanitizeModuleID("a/b") != sanitizeModuleID("a__b") {
			t.Fatal("fixture does not exercise the known directory collision")
		}
		if got, err := cs.LoadArtifacts("a__b", cacheKey); err == nil || got != nil {
			t.Fatalf("colliding module ID loaded another module: got=%v err=%v", got, err)
		}
	})
}

func TestCacheArtifacts_PartialWrite(t *testing.T) {
	const moduleID = "partial/module"
	const keyA = "key-a"
	const keyB = "key-b"

	t.Run("well_formed_blob_substitution_and_missing_digests", func(t *testing.T) {
		for _, name := range artifactPayloadNames() {
			t.Run(name, func(t *testing.T) {
				csA := newArtifactTestStore(t)
				csB := newArtifactTestStore(t)
				mustStoreArtifacts(t, csA, moduleID, keyA, testCachedModule(moduleID, 42))
				mustStoreArtifacts(t, csB, moduleID, keyB, testCachedModule(moduleID, 99))
				data := mustRead(t, filepath.Join(csB.moduleArtifactDir(moduleID), name))
				mustWriteArtifactTest(t, filepath.Join(csA.moduleArtifactDir(moduleID), name), data)
				if got, err := csA.LoadArtifacts(moduleID, keyA); err == nil || got != nil {
					t.Fatalf("substituted %s was accepted", name)
				}
			})

			t.Run("missing_"+name, func(t *testing.T) {
				cs := newArtifactTestStore(t)
				mustStoreArtifacts(t, cs, moduleID, keyA, testCachedModule(moduleID, 42))
				path := filepath.Join(cs.moduleArtifactDir(moduleID), artifactStampName)
				mutateStamp(func(stamp *artifactStamp) { delete(stamp.SHA256, name) })(t, cs, path)
				if got, err := cs.LoadArtifacts(moduleID, keyA); err == nil || got != nil {
					t.Fatalf("stamp missing %s digest was accepted", name)
				}
			})
		}
	})

	t.Run("encoding_failure_preserves_A", func(t *testing.T) {
		for _, stage := range artifactPayloadNames() {
			t.Run(stage, func(t *testing.T) {
				cs := newArtifactTestStore(t)
				mustStoreArtifacts(t, cs, moduleID, keyA, testCachedModule(moduleID, 42))
				injectEncodingFailure(&cs.artifactCodec, stage)
				if err := cs.StoreArtifacts(moduleID, keyB, testCachedModule(moduleID, 99)); err == nil {
					t.Fatal("injected encoding failure returned nil")
				}
				got, err := cs.LoadArtifacts(moduleID, keyA)
				if err != nil || cachedLiteral(t, got) != 42 {
					t.Fatalf("encoding failure disturbed A: got=%v err=%v", got, err)
				}
			})
		}
	})

	t.Run("publication_interruption_never_authorizes_B", func(t *testing.T) {
		names := artifactPayloadNames()
		stages := append(names[:], "stamp_close", "stamp_rename")
		for _, stage := range stages {
			t.Run(stage, func(t *testing.T) {
				cs := newArtifactTestStore(t)
				mustStoreArtifacts(t, cs, moduleID, keyA, testCachedModule(moduleID, 42))
				injectPublicationFailure(cs, stage)
				if err := cs.StoreArtifacts(moduleID, keyB, testCachedModule(moduleID, 99)); err == nil {
					t.Fatal("injected publication failure returned nil")
				}
				if got, err := cs.LoadArtifacts(moduleID, keyB); err == nil || got != nil {
					t.Fatalf("partial B publication was authorized: got=%v err=%v", got, err)
				}
				if got, err := cs.LoadArtifacts(moduleID, keyA); err == nil && cachedLiteral(t, got) != 42 {
					t.Fatalf("old key decoded mixed/new value %d", cachedLiteral(t, got))
				}
			})
		}
	})

	cs := newArtifactTestStore(t)
	mustStoreArtifacts(t, cs, moduleID, keyA, testCachedModule(moduleID, 42))
	if got, err := cs.LoadArtifacts(moduleID, keyA); err != nil || cachedLiteral(t, got) != 42 {
		t.Fatalf("untouched A control failed: got=%v err=%v", got, err)
	}
}

func TestCacheArtifacts_ReadSnapshot(t *testing.T) {
	const moduleID = "snapshot/module"
	const key = "snapshot-key"
	cs := newArtifactTestStore(t)
	mustStoreArtifacts(t, cs, moduleID, key, testCachedModule(moduleID, 42))

	other := newArtifactTestStore(t)
	mustStoreArtifacts(t, other, moduleID, "other-key", testCachedModule(moduleID, 99))
	replacement := mustRead(t, filepath.Join(other.moduleArtifactDir(moduleID), artifactCoreName))
	corePath := filepath.Join(cs.moduleArtifactDir(moduleID), artifactCoreName)

	originalOpen := cs.artifactIO.open
	mutated := false
	cs.artifactIO.open = func(path string) (artifactReadFile, error) {
		file, err := originalOpen(path)
		if err != nil || path != corePath {
			return file, err
		}
		return &afterCloseArtifactFile{artifactReadFile: file, after: func() {
			if !mutated {
				mutated = true
				mustWriteArtifactTest(t, corePath, replacement)
			}
		}}, nil
	}

	got, err := cs.LoadArtifacts(moduleID, key)
	if err != nil {
		t.Fatalf("verified snapshot rejected after disk mutation: %v", err)
	}
	if value := cachedLiteral(t, got); value != 42 {
		t.Fatalf("decoded reopened/unverified bytes: got %d, want verified snapshot 42", value)
	}
	if !mutated {
		t.Fatal("test seam did not mutate disk after the verified read")
	}
}

func TestCacheArtifacts_ByteLimits(t *testing.T) {
	if maxArtifactBlobBytes != 16<<20 || maxArtifactStampBytes != 64<<10 || maxModuleArtifactBytes != 32<<20 {
		t.Fatalf("production limits changed: blob=%d stamp=%d module=%d", maxArtifactBlobBytes, maxArtifactStampBytes, maxModuleArtifactBytes)
	}

	t.Run("oversized_blob_rejected_by_stat_without_read", func(t *testing.T) {
		cs := newArtifactTestStore(t)
		mustStoreArtifacts(t, cs, "limits/module", "limits-key", testCachedModule("limits/module", 42))
		path := filepath.Join(cs.moduleArtifactDir("limits/module"), artifactCoreName)
		if err := os.Truncate(path, maxArtifactBlobBytes+1); err != nil {
			t.Fatalf("create sparse oversized blob: %v", err)
		}
		reads := 0
		originalOpen := cs.artifactIO.open
		cs.artifactIO.open = func(openPath string) (artifactReadFile, error) {
			file, err := originalOpen(openPath)
			if err != nil || openPath != path {
				return file, err
			}
			return &countingArtifactFile{artifactReadFile: file, reads: &reads}, nil
		}
		_, err := cs.LoadArtifacts("limits/module", "limits-key")
		assertArtifactTooLarge(t, err, "blob", maxArtifactBlobBytes)
		if reads != 0 {
			t.Fatalf("oversized stat still read payload %d times", reads)
		}
	})

	t.Run("stamp_and_aggregate_scopes", func(t *testing.T) {
		cs := newArtifactTestStore(t)
		mustStoreArtifacts(t, cs, "limits/module", "limits-key", testCachedModule("limits/module", 42))
		stampPath := filepath.Join(cs.moduleArtifactDir("limits/module"), artifactStampName)
		mustWriteArtifactTest(t, stampPath, bytes.Repeat([]byte(" "), int(maxArtifactStampBytes+1)))
		_, err := cs.LoadArtifacts("limits/module", "limits-key")
		assertArtifactTooLarge(t, err, "stamp", maxArtifactStampBytes)

		mustStoreArtifacts(t, cs, "limits/module", "limits-key", testCachedModule("limits/module", 42))
		stampPath = filepath.Join(cs.moduleArtifactDir("limits/module"), artifactStampName)
		stamp := readArtifactStamp(t, stampPath)
		for _, name := range []string{artifactIfaceName, artifactConstructorsName} {
			path := filepath.Join(cs.moduleArtifactDir("limits/module"), name)
			data := mustRead(t, path)
			data = append(data, bytes.Repeat([]byte(" "), int(maxArtifactBlobBytes)-len(data))...)
			mustWriteArtifactTest(t, path, data)
			sum := sha256.Sum256(data)
			stamp.SHA256[name] = fmt.Sprintf("%x", sum)
		}
		stampData, marshalErr := json.Marshal(stamp)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		mustWriteArtifactTest(t, stampPath, stampData)
		_, err = cs.LoadArtifacts("limits/module", "limits-key")
		assertArtifactTooLarge(t, err, "module", 0)
	})

	t.Run("limit_reader_uses_extra_byte_sentinel", func(t *testing.T) {
		cs := newArtifactTestStore(t)
		cs.artifactLimits = artifactLimits{blob: 4, stamp: 4, module: 8}
		cs.artifactIO.open = func(string) (artifactReadFile, error) {
			return newMemoryArtifactFile([]byte("1234"), 4, 0), nil
		}
		accepted := int64(0)
		data, err := cs.readBoundedArtifact("exact", 4, "blob", &accepted)
		if err != nil || string(data) != "1234" {
			t.Fatalf("exact limit with clean EOF rejected: data=%q err=%v", data, err)
		}

		cs.artifactIO.open = func(string) (artifactReadFile, error) {
			return newMemoryArtifactFile([]byte("12345"), 4, 0), nil
		}
		accepted = 0
		_, err = cs.readBoundedArtifact("grew-after-stat", 4, "blob", &accepted)
		assertArtifactTooLarge(t, err, "blob", 4)
	})

	t.Run("regular_files_and_hashes_precede_decode", func(t *testing.T) {
		cs := newArtifactTestStore(t)
		mustStoreArtifacts(t, cs, "limits/module", "limits-key", testCachedModule("limits/module", 42))
		decoded := 0
		originalDecode := cs.artifactCodec.decodeCore
		cs.artifactCodec.decodeCore = func(data []byte) (*core.Program, error) {
			decoded++
			return originalDecode(data)
		}
		ctorPath := filepath.Join(cs.moduleArtifactDir("limits/module"), artifactConstructorsName)
		mustWriteArtifactTest(t, ctorPath, []byte(`{"corrupt":true}`))
		if got, err := cs.LoadArtifacts("limits/module", "limits-key"); err == nil || got != nil {
			t.Fatal("hash-mismatching payload was accepted")
		}
		if decoded != 0 {
			t.Fatalf("decoded %d payloads before every hash passed", decoded)
		}

		originalOpen := cs.artifactIO.open
		cs.artifactIO.open = func(path string) (artifactReadFile, error) {
			file, err := originalOpen(path)
			if err != nil || filepath.Base(path) != artifactCoreName {
				return file, err
			}
			return &modeArtifactFile{artifactReadFile: file, mode: fs.ModeDir}, nil
		}
		if got, err := cs.LoadArtifacts("limits/module", "limits-key"); err == nil || got != nil {
			t.Fatal("non-regular payload was accepted")
		}
	})

	t.Run("oversized_publication_is_non_authorizing", func(t *testing.T) {
		cs := newArtifactTestStore(t)
		cs.artifactLimits.blob = 1
		var warnings bytes.Buffer
		runtime := &cacheRuntime{store: cs, stderr: &warnings, invalidWarned: make(map[string]bool), writeWarned: make(map[string]bool)}
		published := runtime.publish("large/module", "large-key", &CacheEntry{CacheKey: "large-key"}, testCachedModule("large/module", 42))
		if published {
			t.Fatal("oversized artifact publication reported success")
		}
		if _, ok := cs.Lookup("large/module", "large-key"); ok {
			t.Fatal("oversized artifacts published a manifest entry")
		}
		if !strings.Contains(warnings.String(), "CACHE_WRITE_FAILED") || !strings.Contains(warnings.String(), "stage=encoding") {
			t.Fatalf("missing encoding warning: %q", warnings.String())
		}
	})

	t.Run("pipeline_miss_repairs_oversized_blob_then_hits", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		t.Setenv("AILANG_CACHE_DIR", "")
		mustWriteArtifactTest(t, "answer.ail", []byte("module answer\nexport pure func main() -> int = 42\n"))
		cfg := Config{Mode: ModeCheck}
		if _, err := runModuleWithCacheDependencies(t.Context(), cfg, Source{Filename: "answer.ail"}, productionCacheDependencies()); err != nil {
			t.Fatalf("warm compile: %v", err)
		}
		cacheDir := filepath.Join(root, ".ailang", "cache", "compile")
		corePath := filepath.Join(cacheDir, "modules", "answer", artifactCoreName)
		if err := os.Truncate(corePath, maxArtifactBlobBytes+1); err != nil {
			t.Fatalf("poison core: %v", err)
		}
		var warnings bytes.Buffer
		deps := cacheDependencies{newStore: NewCacheStore, stderr: &warnings}
		result, err := runModuleWithCacheDependencies(t.Context(), cfg, Source{Filename: "answer.ail"}, deps)
		if err != nil || result.Interface == nil {
			t.Fatalf("oversized cache miss did not compile fresh: iface=%v err=%v", result.Interface, err)
		}
		if warning := warnings.String(); !strings.Contains(warning, "CACHE_INVALID module=answer") || !strings.Contains(warning, "reason=ARTIFACT_TOO_LARGE scope=blob") {
			t.Fatalf("missing overflow diagnostic: %q", warning)
		}

		decodeCount := 0
		hitDeps := cacheDependencies{stderr: io.Discard, newStore: func(projectDir string) (*CacheStore, error) {
			store, openErr := NewCacheStore(projectDir)
			if openErr == nil {
				decode := store.artifactCodec.decodeCore
				store.artifactCodec.decodeCore = func(data []byte) (*core.Program, error) {
					decodeCount++
					return decode(data)
				}
			}
			return store, openErr
		}}
		if _, err := runModuleWithCacheDependencies(t.Context(), cfg, Source{Filename: "answer.ail"}, hitDeps); err != nil {
			t.Fatalf("post-repair warm hit: %v", err)
		}
		if decodeCount == 0 {
			t.Fatal("post-repair run did not decode any verified cached core")
		}
	})
}

func newArtifactTestStore(t *testing.T) *CacheStore {
	t.Helper()
	t.Setenv("AILANG_CACHE_DIR", "")
	cs, err := NewCacheStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewCacheStore: %v", err)
	}
	return cs
}

func testCachedModule(moduleID string, value int) *CachedModule {
	return &CachedModule{
		Core: &core.Program{
			Decls: []core.CoreExpr{&core.Lit{CoreNode: core.CoreNode{NodeID: 1}, Kind: core.IntLit, Value: value}},
			Meta:  map[string]*core.DeclMeta{},
			Flags: core.ProgramFlags{Lowered: true},
		},
		CoreTI: types.CoreTypeInfo{
			1: &types.TCon{Name: "int"},
			2: &types.TCon{Name: fmt.Sprintf("fixture-%d", value)},
		},
		Iface: &iface.Iface{
			Module:       moduleID,
			Schema:       "ailang.iface/v1",
			Digest:       fmt.Sprintf("digest-%d", value),
			Exports:      map[string]*iface.IfaceItem{},
			Constructors: map[string]*iface.ConstructorScheme{},
			Types:        map[string]*iface.TypeExport{},
			TypeAliases:  map[string]types.Type{},
			AliasParams:  map[string][]string{},
		},
		Constructors: map[string]*ConstructorInfo{
			fmt.Sprintf("Fixture%d", value): {
				TypeName: fmt.Sprintf("Fixture%d", value),
				CtorName: fmt.Sprintf("Fixture%d", value),
			},
		},
	}
}

func cachedLiteral(t *testing.T, cm *CachedModule) int {
	t.Helper()
	if cm == nil || cm.Core == nil || len(cm.Core.Decls) != 1 {
		t.Fatalf("invalid cached fixture: %#v", cm)
	}
	lit, ok := cm.Core.Decls[0].(*core.Lit)
	if !ok {
		t.Fatalf("cached decl type = %T, want *core.Lit", cm.Core.Decls[0])
	}
	value, ok := lit.Value.(int)
	if !ok {
		t.Fatalf("cached literal value type = %T, want int", lit.Value)
	}
	return value
}

func mustStoreArtifacts(t *testing.T, cs *CacheStore, moduleID, key string, cm *CachedModule) {
	t.Helper()
	if err := cs.StoreArtifacts(moduleID, key, cm); err != nil {
		t.Fatalf("StoreArtifacts: %v", err)
	}
}

func mutateStamp(mutate func(*artifactStamp)) func(*testing.T, *CacheStore, string) {
	return func(t *testing.T, _ *CacheStore, path string) {
		t.Helper()
		var stamp artifactStamp
		if err := json.Unmarshal(mustRead(t, path), &stamp); err != nil {
			t.Fatalf("read stamp: %v", err)
		}
		mutate(&stamp)
		data, err := json.Marshal(stamp)
		if err != nil {
			t.Fatalf("marshal stamp: %v", err)
		}
		mustWriteArtifactTest(t, path, data)
	}
}

func injectEncodingFailure(codec *cacheArtifactCodec, name string) {
	fail := func() ([]byte, error) { return nil, os.ErrPermission }
	switch name {
	case artifactCoreName:
		codec.encodeCore = func(*core.Program) ([]byte, error) { return fail() }
	case artifactCoreTIName:
		codec.encodeCoreTI = func(types.CoreTypeInfo) ([]byte, error) { return fail() }
	case artifactIfaceName:
		codec.encodeIface = func(*iface.Iface) ([]byte, error) { return fail() }
	case artifactConstructorsName:
		codec.encodeConstructors = func(map[string]*ConstructorInfo) ([]byte, error) { return fail() }
	}
}

func injectPublicationFailure(cs *CacheStore, stage string) {
	if stage == "stamp_close" {
		create := cs.artifactIO.createTemp
		cs.artifactIO.createTemp = func(dir, pattern string) (artifactTempFile, error) {
			file, err := create(dir, pattern)
			return &closeFailTempFile{artifactTempFile: file}, err
		}
		return
	}
	if stage == "stamp_rename" {
		cs.artifactIO.rename = func(string, string) error { return os.ErrPermission }
		return
	}
	write := cs.artifactIO.writeFile
	cs.artifactIO.writeFile = func(path string, data []byte, mode fs.FileMode) error {
		if filepath.Base(path) == stage {
			return os.ErrPermission
		}
		return write(path, data, mode)
	}
}

func assertArtifactTooLarge(t *testing.T, err error, scope string, limit int64) {
	t.Helper()
	var artifactErr *cacheArtifactError
	if !errors.As(err, &artifactErr) {
		t.Fatalf("error type = %T (%v), want cacheArtifactError", err, err)
	}
	if artifactErr.Reason != artifactTooLargeReason || artifactErr.Scope != scope {
		t.Fatalf("overflow = reason %q scope %q, want %q/%q", artifactErr.Reason, artifactErr.Scope, artifactTooLargeReason, scope)
	}
	if limit != 0 && artifactErr.LimitBytes != limit {
		t.Fatalf("overflow limit = %d, want %d", artifactErr.LimitBytes, limit)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

func mustWriteArtifactTest(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustRemove(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove %s: %v", path, err)
	}
}

type afterCloseArtifactFile struct {
	artifactReadFile
	after func()
}

func (file *afterCloseArtifactFile) Close() error {
	err := file.artifactReadFile.Close()
	file.after()
	return err
}

type countingArtifactFile struct {
	artifactReadFile
	reads *int
}

func (file *countingArtifactFile) Read(data []byte) (int, error) {
	(*file.reads)++
	return file.artifactReadFile.Read(data)
}

type modeArtifactFile struct {
	artifactReadFile
	mode fs.FileMode
}

func (file *modeArtifactFile) Stat() (fs.FileInfo, error) {
	info, err := file.artifactReadFile.Stat()
	if err != nil {
		return nil, err
	}
	return fakeFileInfo{name: info.Name(), size: info.Size(), mode: file.mode}, nil
}

type closeFailTempFile struct{ artifactTempFile }

func (file *closeFailTempFile) Close() error {
	_ = file.artifactTempFile.Close()
	return os.ErrPermission
}

type memoryArtifactFile struct {
	*bytes.Reader
	info fakeFileInfo
}

func newMemoryArtifactFile(data []byte, statSize int64, mode fs.FileMode) *memoryArtifactFile {
	if mode == 0 {
		mode = 0o644
	}
	return &memoryArtifactFile{Reader: bytes.NewReader(data), info: fakeFileInfo{name: "memory", size: statSize, mode: mode}}
}

func (file *memoryArtifactFile) Stat() (fs.FileInfo, error) { return file.info, nil }
func (file *memoryArtifactFile) Close() error               { return nil }

type fakeFileInfo struct {
	name string
	size int64
	mode fs.FileMode
}

func (info fakeFileInfo) Name() string       { return info.name }
func (info fakeFileInfo) Size() int64        { return info.size }
func (info fakeFileInfo) Mode() fs.FileMode  { return info.mode }
func (info fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (info fakeFileInfo) IsDir() bool        { return info.mode.IsDir() }
func (info fakeFileInfo) Sys() any           { return nil }
