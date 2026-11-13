package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

// typeLeaves renders the names of a type in source order, each tagged var: or
// con: by the node the parser built for it.
func typeLeaves(t ast.Type, out *[]string) {
	switch x := t.(type) {
	case *ast.TypeVar:
		*out = append(*out, "var:"+x.Name)
	case *ast.SimpleType:
		*out = append(*out, "con:"+x.Name)
	case *ast.TypeApp:
		*out = append(*out, "app:"+x.Constructor)
		for _, a := range x.Args {
			typeLeaves(a, out)
		}
	case *ast.FuncType:
		for _, p := range x.Params {
			typeLeaves(p, out)
		}
		typeLeaves(x.Return, out)
	case *ast.TupleType:
		for _, e := range x.Elements {
			typeLeaves(e, out)
		}
	case *ast.RecordType:
		for _, f := range x.Fields {
			typeLeaves(f.Type, out)
		}
	case *ast.ListType:
		typeLeaves(x.Element, out)
	}
}

func parseTypeNamesFile(t *testing.T, src string) (*ast.File, *Parser) {
	t.Helper()
	p := New(lexer.New(src, "t.ail"))
	return p.ParseFile(), p
}

// funcSig renders a function's parameter and return types.
func funcSig(t *testing.T, file *ast.File, name string) string {
	t.Helper()
	for _, fn := range file.Funcs {
		if fn.Name != name {
			continue
		}
		var out []string
		for _, p := range fn.Params {
			typeLeaves(p.Type, &out)
		}
		out = append(out, "->")
		typeLeaves(fn.ReturnType, &out)
		return strings.Join(out, " ")
	}
	t.Fatalf("no func %s", name)
	return ""
}

// typeBody renders a type declaration's body.
func typeBody(t *testing.T, file *ast.File, name string) string {
	t.Helper()
	for _, d := range file.Decls {
		td, ok := d.(*ast.TypeDecl)
		if !ok || td.Name != name {
			continue
		}
		var out []string
		switch def := td.Definition.(type) {
		case *ast.AlgebraicType:
			for _, c := range def.Constructors {
				for _, f := range c.Fields {
					typeLeaves(f.Type, &out)
				}
			}
		case *ast.RecordType:
			typeLeaves(def, &out)
		case *ast.TypeAlias:
			typeLeaves(def.Target, &out)
		}
		return strings.Join(out, " ")
	}
	t.Fatalf("no type %s", name)
	return ""
}

func TestTypeNames_ScopeNotCase(t *testing.T) {
	tests := []struct {
		name, src, fn, want string
		typ                 bool // fn names a type declaration, not a function
	}{
		// #2240: a name the type-parameter list binds is a variable, any case.
		{"uppercase func param", "pure func id[T](x: T) -> T = x", "id", "var:T -> var:T", false},
		{"two uppercase params in nested types",
			"pure func m[A, B](f: (A) -> B, xs: [A]) -> Option[B] = None", "m",
			"var:A var:B app:list var:A -> app:Option var:B", false},
		{"uppercase type-decl param", "type Box[T] = Box(T)", "Box", "var:T", true},
		{"param shadows a declared type", "type T = Tee(int)\npure func id[T](x: T) -> T = x", "id", "var:T -> var:T", false},
		{"declared type outside the binder's scope",
			"type T = Tee(int)\npure func id[T](x: T) -> T = x\npure func un(t: T) -> int = 0", "un", "con:T -> con:int", false},
		// #2211: a lowercase name the file declares or imports by name is a type.
		{"lowercase declared type", "type point = {x: int, y: int}\npure func f(p: point) -> point = p", "f", "con:point -> con:point", false},
		{"use before the lowercase declaration", "pure func f(p: point) -> point = p\ntype point = {x: int}", "f", "con:point -> con:point", false},
		{"lowercase type inside another declaration", "type point = {x: int}\ntype seg = {a: point, b: point}", "seg", "con:point con:point", true},
		{"lowercase import by name", "import std/sem (sem_frame)\npure func f(s: sem_frame) -> int = 0", "f", "con:sem_frame -> con:int", false},
		{"lowercase import alias", "import std/sem (sem_frame as frame)\npure func f(s: frame) -> int = 0", "f", "con:frame -> con:int", false},
		{"binder shadows a lowercase declared type", "type point = {x: int}\npure func f[point](p: point) -> point = p", "f", "var:point -> var:point", false},
		// Unchanged: implicit lowercase variables, uppercase types, builtins.
		{"implicit lowercase variables", "pure func f(g: (acc, elem) -> acc, xs: [elem]) -> acc = 0", "f",
			"var:acc var:elem var:acc app:list var:elem -> var:acc", false},
		{"explicit lowercase list", "pure func f[a](x: a) -> a = x", "f", "var:a -> var:a", false},
		{"uppercase undeclared name stays a type", "pure func f(x: Foo) -> int = 0", "f", "con:Foo -> con:int", false},
		{"builtins", "pure func f(x: string, y: bytes) -> bool = true", "f", "con:string con:bytes -> con:bool", false},
		{"a record field name is not a type", "pure func f(r: {kind: string}) -> int = 0\npure func g(x: kind) -> kind = x", "g", "var:kind -> var:kind", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, p := parseTypeNamesFile(t, tt.src)
			AssertNoErrors(t, p)
			got := ""
			if tt.typ {
				got = typeBody(t, file, tt.fn)
			} else {
				got = funcSig(t, file, tt.fn)
			}
			if got != tt.want {
				t.Errorf("got  %s\nwant %s", got, tt.want)
			}
		})
	}
}

// The binder's scope ends with its declaration: a later function sees T as
// a type again, and a later lowercase use is an implicit variable.
func TestTypeNames_BinderScopeEnds(t *testing.T) {
	file, p := parseTypeNamesFile(t, "pure func id[T](x: T) -> T = x\npure func g(x: T) -> T = x\n"+
		"type Box[a] = Box(a)\npure func h(x: a) -> a = x")
	AssertNoErrors(t, p)
	if got := funcSig(t, file, "g"); got != "con:T -> con:T" {
		t.Errorf("g: %s", got)
	}
	if got := funcSig(t, file, "h"); got != "var:a -> var:a" {
		t.Errorf("h: %s", got)
	}
}

func TestTypeNames_BuiltinParamRejected(t *testing.T) {
	for _, src := range []string{
		"type Res[int] = Ok(int) | Err(string)",
		"pure func f[string](x: string) -> string = x",
	} {
		_, p := parseTypeNamesFile(t, src)
		var msgs []string
		for _, e := range p.Errors() {
			msgs = append(msgs, e.Error())
		}
		if !strings.Contains(strings.Join(msgs, "\n"), "PAR_TYPE_PARAM_BUILTIN") {
			t.Errorf("%q: want PAR_TYPE_PARAM_BUILTIN, got %v", src, msgs)
		}
	}
}

// A type parameter written with arguments is PAR_TYPE_PARAM_APPLIED; an
// unbound name with arguments is an ordinary type application.
func TestTypeNames_ParamApplied(t *testing.T) {
	for _, src := range []string{
		"pure func same[m](x: m[int]) -> int = 0",
		"pure func same[F](x: F[int]) -> int = 0",
	} {
		_, p := parseTypeNamesFile(t, src)
		var msgs []string
		for _, e := range p.Errors() {
			msgs = append(msgs, e.Error())
		}
		if !strings.Contains(strings.Join(msgs, "\n"), "PAR_TYPE_PARAM_APPLIED") {
			t.Errorf("%q: want PAR_TYPE_PARAM_APPLIED, got %v", src, msgs)
		}
	}
	file, p := parseTypeNamesFile(t, "pure func f(x: m[int]) -> int = 0")
	AssertNoErrors(t, p)
	if got := funcSig(t, file, "f"); got != "app:m con:int -> con:int" {
		t.Errorf("unbound applied name: %s", got)
	}
}

// File.LowerTypeNames records each bare lowercase name resolved without a
// binder, with how it was read and the declaration it is in; bound and
// built-in names are not recorded.
func TestTypeNames_LowerTypeNamesRecorded(t *testing.T) {
	file, p := parseTypeNamesFile(t, "import m/x (frame, run)\n"+
		"type acc = {n: int}\n"+
		"pure func f[a](x: a, y: frame, z: acc) -> elem = 0\n"+
		"pure func g(r: run) -> int = 0")
	AssertNoErrors(t, p)
	var got []string
	for _, r := range file.LowerTypeNames {
		got = append(got, fmt.Sprintf("%s:%s:%d@%d", r.Decl, r.Name, r.Kind, r.Pos.Line))
	}
	want := []string{
		fmt.Sprintf("f:frame:%d@3", ast.LowerImported),
		fmt.Sprintf("f:acc:%d@3", ast.LowerLocal),
		fmt.Sprintf("f:elem:%d@3", ast.LowerImplicitVar),
		fmt.Sprintf("g:run:%d@4", ast.LowerImported),
	}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("got  %v\nwant %v", got, want)
	}
}

// A declaration in the file wins over an import of the same lowercase name.
func TestTypeNames_LocalBeatsImport(t *testing.T) {
	file, p := parseTypeNamesFile(t, "import m/x (acc)\ntype acc = {n: int}\npure func f(z: acc) -> int = 0")
	AssertNoErrors(t, p)
	if len(file.LowerTypeNames) != 1 || file.LowerTypeNames[0].Kind != ast.LowerLocal {
		t.Errorf("got %+v", file.LowerTypeNames)
	}
}

// A type declaration named like a built-in is PAR_TYPE_DECL_BUILTIN (the
// declaration could never be used: the name always means the built-in).
func TestTypeNames_DeclBuiltinRejected(t *testing.T) {
	for _, name := range ast.BuiltinTypeNames() {
		_, p := parseTypeNamesFile(t, "type "+name+" = int")
		var msgs []string
		for _, e := range p.Errors() {
			msgs = append(msgs, e.Error())
		}
		if len(msgs) != 1 || !strings.Contains(msgs[0], "PAR_TYPE_DECL_BUILTIN") || !strings.Contains(msgs[0], "type '"+name+"'") {
			t.Errorf("type %s = int: want one PAR_TYPE_DECL_BUILTIN, got %v", name, msgs)
		}
	}
	_, p := parseTypeNamesFile(t, "type Float = float\ntype floaty = float")
	AssertNoErrors(t, p)
}
