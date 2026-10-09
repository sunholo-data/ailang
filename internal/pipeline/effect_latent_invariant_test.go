package pipeline

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/testutil"
	"github.com/sunholo-data/ailang/internal/types"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLatentMaskPublicationInvariant(t *testing.T) {
	fn := &core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "identity"}
	app := &core.App{CoreNode: core.CoreNode{NodeID: 2}, Func: fn}
	lam := &core.Lambda{Body: app}
	prog := &core.Program{Decls: []core.CoreExpr{&core.Let{Name: "use", Value: lam}}}
	ti := types.CoreTypeInfo{1: &types.TFunc2{Return: types.TInt, EffectRow: types.EmptyEffectRow()}, 2: types.TInt}
	for _, present := range []bool{false, true} {
		err := ValidateEffectsWithCalls(&ast.File{}, prog, ti, func(uint64) (types.ApplicationEffects, bool) {
			return types.ApplicationEffects{CallRow: types.EmptyEffectRow()}, present
		}, nil)
		if present {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || !strings.Contains(err.Error(), "internal invariant: missing or malformed ApplicationEffects") {
			t.Fatalf("want invariant error: %v", err)
		}
	}
}

func TestLatentRunRejectsBeforeExecution(t *testing.T) {
	binary := testutil.FindAilangBinary(t)
	t.Setenv("AILANG_NO_CACHE", "1")
	for _, tt := range []struct{ name, src, blame string }{
		{"hof", `func loud(x:int) -> int ! {IO} {let _ = println("EFFECT PERFORMED"); x}
func applyTo(f:int -> int,x:int) -> int = f(x)
pure func sneaky() -> int = applyTo(loud,1)
export func main() -> () ! {IO} = println(show(sneaky()))`, "sneaky"},
		{"field", `type Hooks = {f:int -> int ! {IO}}
func loud(x:int) -> int ! {IO} {let _ = println("EFFECT PERFORMED"); x}
func rowless(h:Hooks) -> int = h.f(1)
export func main() -> () ! {IO} = println(show(rowless({f:loud})))`, "rowless"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), tt.name+".ail")
			if err := os.WriteFile(path, []byte("module "+tt.name+"\nimport std/io (println)\n"+tt.src), 0600); err != nil {
				t.Fatal(err)
			}
			for _, bytecode := range []bool{false, true} {
				args := []string{"run", "--caps", "IO", "--entry", "main"}
				if bytecode {
					args = append(args, "--bytecode")
				}
				output, err := exec.Command(binary, append(args, path)...).CombinedOutput()
				if err == nil {
					t.Fatalf("executed unsound program: %s", output)
				}
				text := string(output)
				if !strings.Contains(text, "Missing effects: IO") || !strings.Contains(text, "function '"+tt.blame+"'") || strings.Contains(text, "EFFECT PERFORMED") {
					t.Fatalf("wrong rejection: %s", output)
				}
			}
		})
	}
}

func TestApplicationPublicationMalformedRows(t *testing.T) {
	fn := &core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "identity"}
	app := &core.App{CoreNode: core.CoreNode{NodeID: 2}, Func: fn}
	prog := &core.Program{Decls: []core.CoreExpr{&core.Let{Name: "use", Value: &core.Lambda{Body: app}}}}
	ti := types.CoreTypeInfo{1: &types.TFunc2{Return: types.TInt}, 2: types.TInt}
	for _, row := range []*types.Row{nil, {}, {Kind: types.RecordRow}, {Kind: types.EffectRow, Tail: &types.RowVar{Name: "unowned", Kind: types.EffectRow}}} {
		err := ValidateEffectsWithCalls(&ast.File{}, prog, ti, func(uint64) (types.ApplicationEffects, bool) { return types.ApplicationEffects{CallRow: row}, true }, nil)
		if err == nil || !strings.Contains(err.Error(), "typed application 2") {
			t.Fatalf("malformed publication accepted: %v", err)
		}
	}
}
