package types

import (
	"fmt"
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
	"strings"
	"testing"
)

func TestIFCCallableJoinAlgebra(t *testing.T) {
	a := single(&ifcFn{kind: fnDeclared, name: "a", params: []ifcDemand{{forbid: []string{"email"}}}})
	b := single(&ifcFn{kind: fnDeclared, name: "b", params: []ifcDemand{{forbid: []string{"secret"}}}})
	c := single(&ifcFn{kind: fnDeclared, name: "c", params: []ifcDemand{{covers: []Label{LabelConst("sqlsafe")}}}})
	key := callableKey
	if key(joinCallable(a, b)) != key(joinCallable(b, a)) {
		t.Fatal("join is not commutative")
	}
	if key(joinCallable(joinCallable(a, b), c)) != key(joinCallable(a, joinCallable(b, c))) {
		t.Fatal("join is not associative")
	}
	if key(joinCallable(a, a)) != key(a) {
		t.Fatal("join is not idempotent")
	}
	// Joining incompatible positive covers is their meet: only bottom remains valid.
	d := single(&ifcFn{kind: fnDeclared, name: "d", params: []ifcDemand{{covers: []Label{LabelConst("email")}}}})
	both := joinCallable(c, d)
	for _, l := range []Label{LabelBottom(), LabelConst("email"), LabelConst("sqlsafe"), LabelConst("secret")} {
		if got := demandViolations(both, l) > 0; got != !LabelEqual(l, LabelBottom()) {
			t.Fatalf("cover meet on %s: rejected=%v", l, got)
		}
	}
}

func TestIFCCallableWrittenAliasBeyondSixteen(t *testing.T) {
	var s strings.Builder
	s.WriteString("module r/aliases\n" + fvPrelude + "type Alias0 = string{not email} -> string\n")
	for i := 1; i <= 40; i++ {
		fmt.Fprintf(&s, "type Alias%d = Alias%d\n", i, i-1)
	}
	p := parser.New(lexer.New(s.String(), "aliases.ail"))
	file := p.ParseFile()
	if len(p.Errors()) > 0 {
		t.Fatal(p.Errors())
	}
	ts := newIFCTypes(file)
	fn := ts.fromFuncType(&ast.SimpleType{Name: "Alias40"}, "g")
	if fn == nil || demandViolations(fn, LabelConst("email")) == 0 {
		t.Fatal("written function demands disappeared through a finite alias chain")
	}
}

func TestIFCDemandCanonicalForbidsAndBottomCover(t *testing.T) {
	d := canonicalDemand(ifcDemand{forbid: []string{"secret", "email", "secret"}})
	if strings.Join(d.forbid, ",") != "email,secret" {
		t.Fatalf("forbids are not a sorted set: %v", d.forbid)
	}
	for _, label := range []string{"email", "secret"} {
		if d.violation(LabelConst(label)) == "" {
			t.Fatalf("lost forbid %s", label)
		}
	}
	absent := ifcDemand{}
	bottom := ifcDemand{covers: []Label{LabelBottom()}}
	if absent.violation(LabelConst("email")) != "" || bottom.violation(LabelConst("email")) == "" || bottom.violation(LabelBottom()) != "" {
		t.Fatal("absent cover and bottom cover were conflated")
	}
}
