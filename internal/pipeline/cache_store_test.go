package pipeline

import (
	"os"
	"testing"
	"time"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/iface"
	"github.com/sunholo/ailang/internal/types"
)

func TestCacheStore_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	cs, err := NewCacheStore(dir)
	if err != nil {
		t.Fatalf("NewCacheStore: %v", err)
	}

	// Store an entry
	cs.Store("std/list", &CacheEntry{
		CacheKey:      "abc123",
		IfaceDigest:   "digest456",
		IfaceJSON:     []byte(`{"module":"std/list"}`),
		CompileTimeMs: 5,
		Timestamp:     time.Now(),
	})

	// Save to disk
	if err := cs.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Reload from disk
	cs2, err := NewCacheStore(dir)
	if err != nil {
		t.Fatalf("NewCacheStore reload: %v", err)
	}

	// Lookup with correct key
	entry, ok := cs2.Lookup("std/list", "abc123")
	if !ok {
		t.Fatal("expected cache hit for std/list")
	}
	if entry.IfaceDigest != "digest456" {
		t.Errorf("digest mismatch: got %s", entry.IfaceDigest)
	}
	if string(entry.IfaceJSON) != `{"module":"std/list"}` {
		t.Errorf("iface JSON mismatch: got %s", string(entry.IfaceJSON))
	}

	// Lookup with wrong key = miss
	_, ok = cs2.Lookup("std/list", "wrong_key")
	if ok {
		t.Error("expected cache miss for wrong key")
	}

	// Lookup missing module = miss
	_, ok = cs2.Lookup("std/nonexistent", "abc123")
	if ok {
		t.Error("expected cache miss for missing module")
	}
}

func TestCacheStore_Clear(t *testing.T) {
	dir := t.TempDir()
	cs, err := NewCacheStore(dir)
	if err != nil {
		t.Fatalf("NewCacheStore: %v", err)
	}

	cs.Store("mod1", &CacheEntry{CacheKey: "k1", Timestamp: time.Now()})
	cs.Store("mod2", &CacheEntry{CacheKey: "k2", Timestamp: time.Now()})
	if err := cs.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	entries, _ := cs.Stats()
	if entries != 2 {
		t.Errorf("expected 2 entries, got %d", entries)
	}

	if err := cs.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	entries, _ = cs.Stats()
	if entries != 0 {
		t.Errorf("expected 0 entries after clear, got %d", entries)
	}
}

func TestCacheStore_CorruptedManifest(t *testing.T) {
	dir := t.TempDir()

	// Create corrupt manifest
	cacheDir := dir + "/.ailang/cache/compile"
	os.MkdirAll(cacheDir, 0755)
	os.WriteFile(cacheDir+"/manifest.json", []byte("not json"), 0644)

	// Should handle gracefully
	cs, err := NewCacheStore(dir)
	if err != nil {
		t.Fatalf("NewCacheStore should not fail on corrupt: %v", err)
	}

	entries, _ := cs.Stats()
	if entries != 0 {
		t.Errorf("expected 0 entries for fresh cache, got %d", entries)
	}
}

func TestCacheStore_Stats(t *testing.T) {
	dir := t.TempDir()
	cs, err := NewCacheStore(dir)
	if err != nil {
		t.Fatalf("NewCacheStore: %v", err)
	}

	cs.Store("mod1", &CacheEntry{CacheKey: "k1", CompileTimeMs: 10, Timestamp: time.Now()})
	cs.Store("mod2", &CacheEntry{CacheKey: "k2", CompileTimeMs: 20, Timestamp: time.Now()})

	entries, totalMs := cs.Stats()
	if entries != 2 {
		t.Errorf("expected 2 entries, got %d", entries)
	}
	if totalMs != 30 {
		t.Errorf("expected 30ms total, got %d", totalMs)
	}
}

// M-INCREMENTAL-TYPECHECK: Round-trip test for CachedModule serialization.

func TestCacheStore_ArtifactRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	cs, err := NewCacheStore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	cm := &CachedModule{
		Core: &core.Program{
			Decls: []core.CoreExpr{
				&core.Let{
					CoreNode: core.CoreNode{NodeID: 1, CoreSpan: ast.Pos{Line: 1, Column: 1}},
					Name:     "x",
					Value:    &core.Lit{CoreNode: core.CoreNode{NodeID: 2}, Kind: core.IntLit, Value: 42},
					Body: &core.App{
						CoreNode: core.CoreNode{NodeID: 3},
						Func:     &core.Var{CoreNode: core.CoreNode{NodeID: 4}, Name: "print"},
						Args:     []core.CoreExpr{&core.Var{CoreNode: core.CoreNode{NodeID: 5}, Name: "x"}},
					},
				},
				&core.Lambda{
					CoreNode: core.CoreNode{NodeID: 10},
					Params:   []string{"a", "b"},
					Body: &core.BinOp{
						CoreNode: core.CoreNode{NodeID: 11},
						Op:       "+",
						Left:     &core.Var{CoreNode: core.CoreNode{NodeID: 12}, Name: "a"},
						Right:    &core.Var{CoreNode: core.CoreNode{NodeID: 13}, Name: "b"},
					},
				},
			},
			Meta: map[string]*core.DeclMeta{
				"x":   {Name: "x", IsExport: true, IsPure: true},
				"add": {Name: "add", IsExport: true, IsPure: true},
			},
			Flags: core.ProgramFlags{Lowered: true},
		},
		CoreTI: types.CoreTypeInfo{
			1:  &types.TCon{Name: "int"},
			2:  &types.TCon{Name: "int"},
			3:  &types.TCon{Name: "()"},
			10: &types.TFunc2{Params: []types.Type{&types.TCon{Name: "int"}, &types.TCon{Name: "int"}}, Return: &types.TCon{Name: "int"}},
		},
		Iface: &iface.Iface{
			Module: "test/module",
			Schema: "ailang.iface/v1",
			Digest: "abc123",
			Exports: map[string]*iface.IfaceItem{
				"x": {
					Name:   "x",
					Type:   &types.Scheme{Type: &types.TCon{Name: "int"}},
					Purity: true,
					Ref:    core.GlobalRef{Module: "test/module", Name: "x"},
				},
			},
			Constructors: map[string]*iface.ConstructorScheme{
				"Some": {
					TypeName:   "Option",
					CtorName:   "Some",
					FieldTypes: []types.Type{&types.TVar2{Name: "a", Kind: types.Star}},
					ResultType: &types.TApp{Constructor: &types.TCon{Name: "Option"}, Args: []types.Type{&types.TVar2{Name: "a", Kind: types.Star}}},
					Arity:      1,
				},
			},
			Types: map[string]*iface.TypeExport{
				"Option": {Name: "Option", Arity: 1},
			},
			TypeAliases: map[string]types.Type{
				"Point": &types.TRecord{
					Fields:   map[string]types.Type{"x": &types.TCon{Name: "float"}, "y": &types.TCon{Name: "float"}},
					TypeName: "Point",
				},
			},
		},
		Constructors: map[string]*ConstructorInfo{
			"Some": {
				TypeName:           "Option",
				CtorName:           "Some",
				Arity:              1,
				TypeParamCount:     1,
				TypeParamNames:     []string{"a"},
				InternalFieldTypes: []types.Type{&types.TVar2{Name: "a", Kind: types.Star}},
			},
		},
	}

	// Store
	if err := cs.StoreArtifacts("test/module", cm); err != nil {
		t.Fatalf("StoreArtifacts failed: %v", err)
	}

	// Load
	got, err := cs.LoadArtifacts("test/module")
	if err != nil {
		t.Fatalf("LoadArtifacts failed: %v", err)
	}

	// Verify core.Program
	if len(got.Core.Decls) != 2 {
		t.Errorf("Core.Decls count: got %d, want 2", len(got.Core.Decls))
	}
	if !got.Core.Flags.Lowered {
		t.Error("Core.Flags.Lowered should be true")
	}
	if let, ok := got.Core.Decls[0].(*core.Let); !ok {
		t.Errorf("Decl[0] type: got %T, want *core.Let", got.Core.Decls[0])
	} else if let.Name != "x" {
		t.Errorf("Decl[0].Name: got %q, want %q", let.Name, "x")
	} else if let.ID() != 1 {
		t.Errorf("Decl[0].ID: got %d, want 1", let.ID())
	}

	// Verify Meta
	if got.Core.Meta == nil || got.Core.Meta["x"] == nil {
		t.Fatal("Core.Meta missing 'x'")
	}
	if !got.Core.Meta["x"].IsExport {
		t.Error("Meta['x'].IsExport should be true")
	}

	// Verify CoreTypeInfo
	if len(got.CoreTI) != 4 {
		t.Errorf("CoreTI count: got %d, want 4", len(got.CoreTI))
	}
	if ti, ok := got.CoreTI[1]; !ok || !ti.Equals(&types.TCon{Name: "int"}) {
		t.Errorf("CoreTI[1] mismatch")
	}

	// Verify Iface
	if got.Iface.Module != "test/module" {
		t.Errorf("Iface.Module: got %q, want %q", got.Iface.Module, "test/module")
	}
	if got.Iface.Digest != "abc123" {
		t.Errorf("Iface.Digest: got %q", got.Iface.Digest)
	}
	if some := got.Iface.Constructors["Some"]; some == nil {
		t.Error("Iface.Constructors missing 'Some'")
	} else if some.Arity != 1 {
		t.Errorf("Constructor Some.Arity: got %d, want 1", some.Arity)
	}
	if pt, ok := got.Iface.TypeAliases["Point"]; !ok {
		t.Error("Iface.TypeAliases missing 'Point'")
	} else if rec, ok := pt.(*types.TRecord); !ok {
		t.Errorf("TypeAlias Point: got %T, want *types.TRecord", pt)
	} else if rec.TypeName != "Point" {
		t.Errorf("TypeAlias Point.TypeName: got %q", rec.TypeName)
	}

	// Verify Constructors
	if some := got.Constructors["Some"]; some == nil {
		t.Error("Constructors missing 'Some'")
	} else if some.TypeParamCount != 1 {
		t.Errorf("Some.TypeParamCount: got %d, want 1", some.TypeParamCount)
	}
}

func TestCacheStore_ArtifactRoundTrip_DiverseExprTypes(t *testing.T) {
	tmpDir := t.TempDir()
	cs, err := NewCacheStore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	cm := &CachedModule{
		Core: &core.Program{
			Decls: []core.CoreExpr{
				&core.Match{
					CoreNode:  core.CoreNode{NodeID: 100},
					Scrutinee: &core.Var{CoreNode: core.CoreNode{NodeID: 101}, Name: "x"},
					Arms: []core.MatchArm{
						{
							Pattern: &core.ConstructorPattern{Name: "Some", Args: []core.CorePattern{&core.VarPattern{Name: "v"}}},
							Body:    &core.Var{CoreNode: core.CoreNode{NodeID: 102}, Name: "v"},
						},
						{
							Pattern: &core.WildcardPattern{},
							Body:    &core.Lit{CoreNode: core.CoreNode{NodeID: 103}, Kind: core.IntLit, Value: 0},
						},
					},
					Exhaustive: true,
				},
				&core.Record{
					CoreNode: core.CoreNode{NodeID: 200},
					Fields: map[string]core.CoreExpr{
						"x": &core.Lit{CoreNode: core.CoreNode{NodeID: 201}, Kind: core.FloatLit, Value: 1.5},
					},
				},
				&core.If{
					CoreNode: core.CoreNode{NodeID: 300},
					Cond:     &core.Lit{CoreNode: core.CoreNode{NodeID: 301}, Kind: core.BoolLit, Value: true},
					Then:     &core.Lit{CoreNode: core.CoreNode{NodeID: 302}, Kind: core.StringLit, Value: "yes"},
					Else:     &core.Lit{CoreNode: core.CoreNode{NodeID: 303}, Kind: core.StringLit, Value: "no"},
				},
				&core.VarGlobal{
					CoreNode: core.CoreNode{NodeID: 500},
					Ref:      core.GlobalRef{Module: "std/list", Name: "map"},
				},
				&core.Intrinsic{
					CoreNode: core.CoreNode{NodeID: 600},
					Op:       core.OpAdd,
					Args: []core.CoreExpr{
						&core.Var{CoreNode: core.CoreNode{NodeID: 601}, Name: "a"},
						&core.Var{CoreNode: core.CoreNode{NodeID: 602}, Name: "b"},
					},
				},
				&core.DictRef{
					CoreNode:  core.CoreNode{NodeID: 700},
					ClassName: "Num",
					TypeName:  "Int",
				},
			},
			Meta:  map[string]*core.DeclMeta{},
			Flags: core.ProgramFlags{},
		},
		CoreTI:       types.CoreTypeInfo{100: &types.TCon{Name: "int"}},
		Iface:        &iface.Iface{Module: "test", Schema: "ailang.iface/v1", Exports: map[string]*iface.IfaceItem{}, Constructors: map[string]*iface.ConstructorScheme{}, Types: map[string]*iface.TypeExport{}, TypeAliases: map[string]types.Type{}},
		Constructors: map[string]*ConstructorInfo{},
	}

	if err := cs.StoreArtifacts("test/diverse", cm); err != nil {
		t.Fatalf("StoreArtifacts failed: %v", err)
	}

	got, err := cs.LoadArtifacts("test/diverse")
	if err != nil {
		t.Fatalf("LoadArtifacts failed: %v", err)
	}

	if len(got.Core.Decls) != 6 {
		t.Fatalf("Core.Decls count: got %d, want 6", len(got.Core.Decls))
	}

	// Verify Match
	if m, ok := got.Core.Decls[0].(*core.Match); !ok {
		t.Errorf("Decl[0]: got %T, want *core.Match", got.Core.Decls[0])
	} else if !m.Exhaustive || len(m.Arms) != 2 {
		t.Errorf("Match: exhaustive=%v, arms=%d", m.Exhaustive, len(m.Arms))
	}

	// Verify VarGlobal
	if vg, ok := got.Core.Decls[3].(*core.VarGlobal); !ok {
		t.Errorf("Decl[3]: got %T, want *core.VarGlobal", got.Core.Decls[3])
	} else if vg.Ref.Module != "std/list" {
		t.Errorf("VarGlobal.Ref.Module: got %q", vg.Ref.Module)
	}

	// Verify DictRef
	if dr, ok := got.Core.Decls[5].(*core.DictRef); !ok {
		t.Errorf("Decl[5]: got %T, want *core.DictRef", got.Core.Decls[5])
	} else if dr.ClassName != "Num" {
		t.Errorf("DictRef.ClassName: got %q", dr.ClassName)
	}
}
