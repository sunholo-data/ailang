package compiler

import (
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
	"github.com/sunholo-data/ailang/internal/vm"
)

// A MatchFail statement compiles to MATCH_FAIL, which validates,
// disassembles with its position, and raises RT002 at run time instead of
// letting the function fall through (M-V1-MATCH-EXHAUSTIVENESS M0).
func TestCompile_MatchFail(t *testing.T) {
	cases := []struct {
		name  string
		trap  stmt.MatchFail
		arg   bytecode.Value
		want  string // RT002 text
		inDis string
	}{
		{
			name:  "list scrutinee, known position",
			trap:  stmt.MatchFail{Scrutinee: stmt.VarRef{Name: "x"}, File: "m.ail", Line: 4, Col: 9},
			arg:   bytecode.NewList([]bytecode.Value{bytecode.NewInt(1), bytecode.NewInt(2)}),
			want:  "RT002: no arm matched in match at m.ail:4:9 (scrutinee: list of 2 elements)",
			inDis: `MATCH_FAIL`,
		},
		{
			name:  "int scrutinee, unknown position",
			trap:  stmt.MatchFail{Scrutinee: stmt.VarRef{Name: "x"}},
			arg:   bytecode.NewInt(7),
			want:  "RT002: no arm matched in match (scrutinee: int)",
			inDis: `""`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog := &stmt.Program{FuncDecls: []stmt.FuncDecl{{
				Name:     "test",
				Params:   []stmt.Param{{Name: "x"}},
				Body:     []stmt.Stmt{tc.trap},
				Return:   stmt.LitInt{Value: 0}, // never reached
				Exported: true,
			}}}
			img, err := Compile(prog)
			if err != nil {
				t.Fatalf("Compile: %v", err)
			}
			if err := img.Validate(); err != nil {
				t.Fatalf("Validate: %v", err)
			}
			if dis := bytecode.Disassemble(img); !strings.Contains(dis, tc.inDis) {
				t.Errorf("disassembly lacks %q:\n%s", tc.inDis, dis)
			}
			got, err := vm.NewVM(img).Run(img.Prototypes[img.EntryPoint], []bytecode.Value{tc.arg})
			var mf *ailerrors.MatchFailureError
			if !errors.As(err, &mf) {
				t.Fatalf("Run = %v, %v; want RT002 *MatchFailureError (never a value)", got, err)
			}
			if mf.Error() != tc.want {
				t.Errorf("RT002 text %q, want %q", mf.Error(), tc.want)
			}
		})
	}
}
