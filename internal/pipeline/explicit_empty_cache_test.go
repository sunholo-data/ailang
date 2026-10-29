package pipeline

import (
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExplicitEmptyAliasDigest(t *testing.T) {
	// Same printed row before this fix, but different callback ownership.
	a := &types.TFunc2{Params: []types.Type{types.TInt}, Return: types.TInt, EffectRow: types.EmptyEffectRow(), ImplicitEffectContract: true}
	b := *a
	b.ImplicitEffectContract = false
	b.ConcreteEffectContract = true
	ia := newAliasDigestFixture(a)
	ib := newAliasDigestFixture(&b)
	if aliasDigest(ia) == aliasDigest(ib) {
		t.Fatal("dependency alias contract edit cannot invalidate cache")
	}
}

func TestExplicitEmptyCacheDependencyEdits(t *testing.T) {
	for _, transitive := range []bool{false, true} {
		name := "direct"
		if transitive {
			name = "transitive"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)
			t.Setenv("AILANG_NO_CACHE", "")
			t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))
			write := func(file, src string) {
				t.Helper()
				if err := os.WriteFile(file, []byte(src), 0600); err != nil {
					t.Fatal(err)
				}
			}
			declaration := func(empty bool) string {
				ann := ""
				if empty {
					ann = " ! {}"
				}
				return "module leaf\nexport type Callback = (int) -> int" + ann + "\n"
			}
			module, alias := "leaf", "Callback"
			if transitive {
				write("middle.ail", "module middle\nimport leaf (Callback)\nexport type Outer = {run: Callback}\n")
				module, alias = "middle", "Outer"
			}
			body := "func apply(f: Callback) -> int = f(1)\nfunc caller() -> int ! {IO} = apply(noisy)\n"
			if transitive {
				body = "func apply(h: Outer) -> int = h.run(1)\nfunc caller() -> int ! {IO} = apply({run: noisy})\n"
			}
			write("main.ail", "module main\nimport "+module+" ("+alias+")\nimport std/io (println)\nfunc noisy(x:int)->int ! {IO} = { println(\"not run\"); x }\n"+body)
			decoded := 0
			check := func() error {
				_, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: "main.ail"}, newDecodeCountDeps(&decoded))
				return err
			}
			write("leaf.ail", declaration(false))
			if err := check(); err != nil {
				t.Fatalf("omitted cold: %v", err)
			}
			decoded = 0
			if err := check(); err != nil {
				t.Fatalf("omitted warm: %v", err)
			}
			if decoded == 0 {
				t.Fatal("instrument failure: no warm artifact decoded")
			}
			write("leaf.ail", declaration(true))
			if err := check(); err == nil || !strings.Contains(err.Error(), "IO") {
				t.Fatalf("dependency edit reused accepted verdict: %v", err)
			}
			write("leaf.ail", declaration(false))
			if err := check(); err != nil {
				t.Fatalf("reverse dependency edit: %v", err)
			}
		})
	}
}

func newAliasDigestFixture(t types.Type) *iface.Iface {
	i := iface.NewIface("dep")
	i.TypeAliases["Callback"] = t
	return i
}

func TestExplicitEmptyCacheExportAndConstructorEdits(t *testing.T) {
	for _, ctor := range []bool{false, true} {
		name := "export"
		if ctor {
			name = "constructor"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)
			t.Setenv("AILANG_NO_CACHE", "")
			t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))
			write := func(file, src string) {
				t.Helper()
				if err := os.WriteFile(file, []byte(src), 0600); err != nil {
					t.Fatal(err)
				}
			}
			dep := func(empty bool) string {
				ann := ""
				if empty {
					ann = " ! {}"
				}
				if ctor {
					return "module dep\nexport type Box = Box((int) -> int" + ann + ")\n"
				}
				return "module dep\nexport func apply(f:(int)->int" + ann + ") -> int = f(1)\n"
			}
			use := "import dep (apply)\nfunc caller() -> int ! {IO} = apply(noisy)\n"
			if ctor {
				use = "import dep (Box)\npure func caller() -> Box = Box(inc)\n"
			}
			write("main.ail", "module main\nimport std/io (println)\n"+strings.Split(use, "\n")[0]+"\npure func inc(x:int)->int = x+1\nfunc noisy(x:int)->int ! {IO} = { println(\"not run\"); x }\n"+strings.Join(strings.Split(use, "\n")[1:], "\n"))
			decoded := 0
			check := func() error {
				_, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: "main.ail"}, newDecodeCountDeps(&decoded))
				return err
			}
			write("dep.ail", dep(false))
			if err := check(); err != nil {
				t.Fatal(err)
			}
			decoded = 0
			if err := check(); err != nil {
				t.Fatal(err)
			}
			if decoded == 0 {
				t.Fatal("instrument failure: no warm artifact decoded")
			}
			store, err := NewCacheStore(root)
			if err != nil {
				t.Fatal(err)
			}
			before := store.manifest.Entries["main"]
			depBefore := store.manifest.Entries["dep"]
			if before == nil || depBefore == nil {
				t.Fatal("instrument failure: expected cached main and dep")
			}
			write("dep.ail", dep(true))
			if err := check(); ctor {
				if err != nil {
					t.Fatalf("pure constructor argument must remain accepted: %v", err)
				}
				store, err = NewCacheStore(root)
				if err != nil {
					t.Fatal(err)
				}
				after, depAfter := store.manifest.Entries["main"], store.manifest.Entries["dep"]
				if after == nil || depAfter == nil || before.CacheKey == after.CacheKey || depBefore.IfaceDigest == depAfter.IfaceDigest {
					t.Fatal("constructor contract edit did not invalidate dependency and importer identities")
				}
				write("bad.ail", "module bad\nimport dep (Box)\nimport std/io (println)\nfunc noisy(x:int)->int ! {IO} = { println(\"not run\"); x }\nfunc caller()->Box ! {IO} = Box(noisy)\n")
				if _, err := Run(Config{Mode: ModeCheck}, Source{Filename: "bad.ail"}); err == nil || !strings.Contains(err.Error(), "IO") {
					t.Fatalf("cached pure constructor accepted IO: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "IO") {
				t.Fatalf("warm contract edit reused acceptance: %v", err)
			}
			write("dep.ail", dep(false))
			if err := check(); err != nil {
				t.Fatalf("reverse edit: %v", err)
			}
		})
	}
}

func TestExplicitEmptyCoreTIDAGRoundTrip(t *testing.T) {
	fn := &types.TFunc2{Params: []types.Type{types.TInt}, Return: types.TInt, EffectRow: types.EmptyEffectRow(), ConcreteEffectContract: true}
	got, _, err := dagRoundTripCoreTI(t, types.CoreTypeInfo{1: fn})
	if err != nil {
		t.Fatal(err)
	}
	cb := got[1].(*types.TFunc2)
	if cb.EffectRow == nil || cb.EffectRow.Tail != nil || len(cb.EffectRow.Labels) != 0 || !cb.ConcreteEffectContract || cb.ImplicitEffectContract {
		t.Fatalf("valid pure contract changed through DAG: %+v", cb)
	}
}
