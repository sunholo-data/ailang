package emitgo

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// A MatchFail trap emits a panic carrying the same RT002 text both
// interpreters raise, positioned at the match (M-V1-MATCH-EXHAUSTIVENESS M0).
func TestEmit_MatchFail(t *testing.T) {
	cases := []struct {
		trap stmt.MatchFail
		want string
	}{
		{stmt.MatchFail{Scrutinee: stmt.VarRef{Name: "c"}, File: "p1.ail", Line: 3, Col: 38},
			`panic("RT002: no arm matched in match at p1.ail:3:38")`},
		{stmt.MatchFail{Scrutinee: stmt.VarRef{Name: "c"}, File: "p1.ail", Line: 3},
			`panic("RT002: no arm matched in match at p1.ail:3")`},
		{stmt.MatchFail{Scrutinee: stmt.VarRef{Name: "c"}},
			`panic("RT002: no arm matched in match")`},
	}
	for _, tc := range cases {
		prog := &stmt.Program{
			Package: "main",
			FuncDecls: []stmt.FuncDecl{{
				Name:       "code",
				Params:     []stmt.Param{{Name: "c", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}},
				ReturnType: stmt.PrimitiveType{Kind: stmt.PrimInt},
				Body:       []stmt.Stmt{tc.trap},
			}},
		}
		if err := stmt.Validate(prog); err != nil {
			t.Fatalf("Validate: %v", err)
		}
		out, err := Emit(prog)
		if err != nil {
			t.Fatalf("Emit: %v", err)
		}
		if !strings.Contains(string(out), tc.want) {
			t.Errorf("want %s in:\n%s", tc.want, out)
		}
	}
	if err := stmt.Validate(&stmt.Program{FuncDecls: []stmt.FuncDecl{{Name: "f", Body: []stmt.Stmt{stmt.MatchFail{}}}}}); err == nil {
		t.Error("Validate accepted a MatchFail with no scrutinee")
	}
}
