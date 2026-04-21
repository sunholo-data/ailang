package compiler

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// --- Lists ------------------------------------------------------------------

func TestCompile_ListLit_Empty(t *testing.T) {
	got := runCompiled(t, nil, nil, stmt.ListLit{}, nil)
	if got.Tag != bytecode.TagList {
		t.Fatalf("got tag %v, want List", got.Tag)
	}
	if len(got.AsList()) != 0 {
		t.Errorf("got %d elems, want 0", len(got.AsList()))
	}
}

func TestCompile_ListLit_ThreeInts(t *testing.T) {
	ret := stmt.ListLit{Elems: []stmt.Expr{
		stmt.LitInt{Value: 1},
		stmt.LitInt{Value: 2},
		stmt.LitInt{Value: 3},
	}}
	got := runCompiled(t, nil, nil, ret, nil)
	elems := got.AsList()
	if len(elems) != 3 {
		t.Fatalf("got %d, want 3", len(elems))
	}
	for i, want := range []int64{1, 2, 3} {
		if elems[i].Int != want {
			t.Errorf("elem %d: got %d, want %d", i, elems[i].Int, want)
		}
	}
}

func TestCompile_Cons(t *testing.T) {
	// 0 :: [1,2,3]
	ret := stmt.Cons{
		Head: stmt.LitInt{Value: 0},
		Tail: stmt.ListLit{Elems: []stmt.Expr{
			stmt.LitInt{Value: 1}, stmt.LitInt{Value: 2}, stmt.LitInt{Value: 3},
		}},
	}
	got := runCompiled(t, nil, nil, ret, nil)
	elems := got.AsList()
	if len(elems) != 4 {
		t.Fatalf("got %d, want 4", len(elems))
	}
	for i, want := range []int64{0, 1, 2, 3} {
		if elems[i].Int != want {
			t.Errorf("elem %d: got %d, want %d", i, elems[i].Int, want)
		}
	}
}

// --- Tuples -----------------------------------------------------------------

func TestCompile_TupleLit(t *testing.T) {
	ret := stmt.TupleLit{Elems: []stmt.Expr{
		stmt.LitInt{Value: 1},
		stmt.LitBool{Value: true},
		stmt.LitString{Value: "x"},
	}}
	got := runCompiled(t, nil, nil, ret, nil)
	if got.Tag != bytecode.TagTuple {
		t.Fatalf("got tag %v, want Tuple", got.Tag)
	}
	tup := got.AsTuple()
	if len(tup) != 3 {
		t.Fatalf("got %d, want 3", len(tup))
	}
	if tup[0].Int != 1 || !tup[1].Bool || tup[2].AsString() != "x" {
		t.Errorf("tuple contents wrong: %v", tup)
	}
}

// --- Records ----------------------------------------------------------------

func TestCompile_RecordLit_AndFieldAccess(t *testing.T) {
	// type Position = {x: int, y: int}
	// origin() = {x: 10, y: 20}; getX(p) = p.x
	// caller() = getX(origin())
	prog := &stmt.Program{
		TypeDecls: []stmt.TypeDecl{
			{
				Name: "Position",
				Kind: stmt.RecordDecl{Fields: []stmt.RecordField{
					{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}},
					{Name: "y", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}},
				}},
			},
		},
		FuncDecls: []stmt.FuncDecl{
			{
				Name: "origin",
				Return: stmt.RecordLit{
					TypeName: "Position",
					Fields: []stmt.FieldInit{
						{Name: "x", Value: stmt.LitInt{Value: 10}},
						{Name: "y", Value: stmt.LitInt{Value: 20}},
					},
				},
			},
			{
				Name:   "getX",
				Params: []stmt.Param{{Name: "p", Type: stmt.NamedType{Name: "Position"}}},
				Return: stmt.FieldAccess{Record: stmt.VarRef{Name: "p"}, Field: "x"},
			},
			{
				Name: "caller",
				Return: stmt.Call{
					Func: stmt.VarRef{Name: "getX"},
					Args: []stmt.Expr{stmt.Call{Func: stmt.VarRef{Name: "origin"}}},
				},
				Exported: true,
			},
		},
	}
	got := runProgram(t, prog, "caller", nil)
	if got.Int != 10 {
		t.Errorf("got %d, want 10", got.Int)
	}
}

func TestCompile_RecordUpdate(t *testing.T) {
	// type Position = {x: int, y: int}
	// moveRight(p, dx) = {p | x: p.x + dx}
	// caller() = moveRight({x: 1, y: 2}, 5).x   → 6
	prog := &stmt.Program{
		TypeDecls: []stmt.TypeDecl{
			{
				Name: "Position",
				Kind: stmt.RecordDecl{Fields: []stmt.RecordField{
					{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}},
					{Name: "y", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}},
				}},
			},
		},
		FuncDecls: []stmt.FuncDecl{
			{
				Name: "moveRight",
				Params: []stmt.Param{
					{Name: "p", Type: stmt.NamedType{Name: "Position"}},
					{Name: "dx", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}},
				},
				Return: stmt.RecordUpdate{
					Base: stmt.VarRef{Name: "p"},
					Fields: []stmt.FieldInit{
						{Name: "x", Value: stmt.BinOp{
							Op:    stmt.OpAdd,
							Left:  stmt.FieldAccess{Record: stmt.VarRef{Name: "p"}, Field: "x"},
							Right: stmt.VarRef{Name: "dx"},
						}},
					},
				},
			},
			{
				Name: "caller",
				Return: stmt.FieldAccess{
					Record: stmt.Call{
						Func: stmt.VarRef{Name: "moveRight"},
						Args: []stmt.Expr{
							stmt.RecordLit{
								TypeName: "Position",
								Fields: []stmt.FieldInit{
									{Name: "x", Value: stmt.LitInt{Value: 1}},
									{Name: "y", Value: stmt.LitInt{Value: 2}},
								},
							},
							stmt.LitInt{Value: 5},
						},
					},
					Field: "x",
				},
				Exported: true,
			},
		},
	}
	got := runProgram(t, prog, "caller", nil)
	if got.Int != 6 {
		t.Errorf("got %d, want 6", got.Int)
	}
}

// --- ADTs + pattern match ---------------------------------------------------

func TestCompile_ADT_SimpleConstructor(t *testing.T) {
	// type Color = Red | Green | Blue
	// caller() = Red  → ADT tag 0
	prog := &stmt.Program{
		TypeDecls: []stmt.TypeDecl{
			{
				Name: "Color",
				Kind: stmt.ADTDecl{Variants: []stmt.ADTVariant{
					{Tag: "Red"}, {Tag: "Green"}, {Tag: "Blue"},
				}},
			},
		},
		FuncDecls: []stmt.FuncDecl{
			{
				Name:     "caller",
				Return:   stmt.ADTConstructor{TypeName: "Color", Tag: "Red"},
				Exported: true,
			},
		},
	}
	got := runProgram(t, prog, "caller", nil)
	if got.Tag != bytecode.TagADT {
		t.Fatalf("got %v, want ADT", got.Tag)
	}
	if got.AsADT().Tag != 0 {
		t.Errorf("Red ordinal: got %d, want 0", got.AsADT().Tag)
	}
}

func TestCompile_ADT_Match_UnwrapOr(t *testing.T) {
	// type Option = Some(int) | None
	// unwrapOr(opt, default) = match opt { Some(x) => x, None => default }
	prog := &stmt.Program{
		TypeDecls: []stmt.TypeDecl{
			{
				Name: "Option",
				Kind: stmt.ADTDecl{Variants: []stmt.ADTVariant{
					{Tag: "Some", Fields: []stmt.ResolvedType{stmt.PrimitiveType{Kind: stmt.PrimInt}}},
					{Tag: "None"},
				}},
			},
		},
		FuncDecls: []stmt.FuncDecl{
			{
				Name: "unwrapOr",
				Params: []stmt.Param{
					{Name: "opt", Type: stmt.NamedType{Name: "Option"}},
					{Name: "default", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}},
				},
				Body: []stmt.Stmt{
					stmt.SwitchStmt{
						Scrutinee: stmt.VarRef{Name: "opt"},
						ADTName:   "Option",
						Cases: []stmt.SwitchCase{
							{
								Tag: "Some",
								Bindings: []stmt.Binding{
									{Name: "x", FieldIndex: 0, Type: stmt.PrimitiveType{Kind: stmt.PrimInt}},
								},
								Body: []stmt.Stmt{stmt.ReturnStmt{Value: stmt.VarRef{Name: "x"}}},
							},
							{
								Tag:  "None",
								Body: []stmt.Stmt{stmt.ReturnStmt{Value: stmt.VarRef{Name: "default"}}},
							},
						},
					},
				},
				Return:   stmt.LitInt{Value: 0}, // unreachable
				Exported: true,
			},
		},
	}
	// Build Some(42) and None to test both branches.
	// We can't easily build ADT values from outside, so add helper functions.
	prog.FuncDecls = append(prog.FuncDecls,
		stmt.FuncDecl{
			Name: "test_some",
			Return: stmt.Call{
				Func: stmt.VarRef{Name: "unwrapOr"},
				Args: []stmt.Expr{
					stmt.ADTConstructor{TypeName: "Option", Tag: "Some", Args: []stmt.Expr{stmt.LitInt{Value: 42}}},
					stmt.LitInt{Value: 99},
				},
			},
			Exported: true,
		},
		stmt.FuncDecl{
			Name: "test_none",
			Return: stmt.Call{
				Func: stmt.VarRef{Name: "unwrapOr"},
				Args: []stmt.Expr{
					stmt.ADTConstructor{TypeName: "Option", Tag: "None"},
					stmt.LitInt{Value: 99},
				},
			},
			Exported: true,
		},
	)

	gotSome := runProgram(t, prog, "test_some", nil)
	if gotSome.Int != 42 {
		t.Errorf("Some(42) → unwrapOr: got %d, want 42", gotSome.Int)
	}
	gotNone := runProgram(t, prog, "test_none", nil)
	if gotNone.Int != 99 {
		t.Errorf("None → unwrapOr: got %d, want 99", gotNone.Int)
	}
}

func TestCompile_ADT_Match_MultiArg(t *testing.T) {
	// type Shape = Circle(float) | Rectangle(float, float) | Point
	// area(s) = match s {
	//    Circle(r)      => 3.14159 * r * r,
	//    Rectangle(w,h) => w * h,
	//    Point          => 0.0
	// }
	prog := &stmt.Program{
		TypeDecls: []stmt.TypeDecl{
			{
				Name: "Shape",
				Kind: stmt.ADTDecl{Variants: []stmt.ADTVariant{
					{Tag: "Circle", Fields: []stmt.ResolvedType{stmt.PrimitiveType{Kind: stmt.PrimFloat}}},
					{Tag: "Rectangle", Fields: []stmt.ResolvedType{stmt.PrimitiveType{Kind: stmt.PrimFloat}, stmt.PrimitiveType{Kind: stmt.PrimFloat}}},
					{Tag: "Point"},
				}},
			},
		},
		FuncDecls: []stmt.FuncDecl{
			{
				Name:   "area",
				Params: []stmt.Param{{Name: "s", Type: stmt.NamedType{Name: "Shape"}}},
				Body: []stmt.Stmt{
					stmt.SwitchStmt{
						Scrutinee: stmt.VarRef{Name: "s"},
						ADTName:   "Shape",
						Cases: []stmt.SwitchCase{
							{
								Tag: "Circle",
								Bindings: []stmt.Binding{
									{Name: "r", FieldIndex: 0, Type: stmt.PrimitiveType{Kind: stmt.PrimFloat}},
								},
								Body: []stmt.Stmt{stmt.ReturnStmt{Value: stmt.BinOp{
									Op:   stmt.OpMul,
									Left: stmt.LitFloat{Value: 3.14159},
									Right: stmt.BinOp{Op: stmt.OpMul,
										Left:  stmt.VarRef{Name: "r"},
										Right: stmt.VarRef{Name: "r"},
									},
								}}},
							},
							{
								Tag: "Rectangle",
								Bindings: []stmt.Binding{
									{Name: "w", FieldIndex: 0, Type: stmt.PrimitiveType{Kind: stmt.PrimFloat}},
									{Name: "h", FieldIndex: 1, Type: stmt.PrimitiveType{Kind: stmt.PrimFloat}},
								},
								Body: []stmt.Stmt{stmt.ReturnStmt{Value: stmt.BinOp{
									Op: stmt.OpMul, Left: stmt.VarRef{Name: "w"}, Right: stmt.VarRef{Name: "h"},
								}}},
							},
							{
								Tag:  "Point",
								Body: []stmt.Stmt{stmt.ReturnStmt{Value: stmt.LitFloat{Value: 0.0}}},
							},
						},
					},
				},
				Return: stmt.LitFloat{Value: 0.0},
			},
			{
				Name: "test_circle",
				Return: stmt.Call{
					Func: stmt.VarRef{Name: "area"},
					Args: []stmt.Expr{stmt.ADTConstructor{TypeName: "Shape", Tag: "Circle", Args: []stmt.Expr{stmt.LitFloat{Value: 2.0}}}},
				},
				Exported: true,
			},
			{
				Name: "test_rect",
				Return: stmt.Call{
					Func: stmt.VarRef{Name: "area"},
					Args: []stmt.Expr{stmt.ADTConstructor{TypeName: "Shape", Tag: "Rectangle", Args: []stmt.Expr{stmt.LitFloat{Value: 3.0}, stmt.LitFloat{Value: 4.0}}}},
				},
				Exported: true,
			},
			{
				Name: "test_point",
				Return: stmt.Call{
					Func: stmt.VarRef{Name: "area"},
					Args: []stmt.Expr{stmt.ADTConstructor{TypeName: "Shape", Tag: "Point"}},
				},
				Exported: true,
			},
		},
	}

	gotCircle := runProgram(t, prog, "test_circle", nil)
	wantCircle := 3.14159 * 4.0
	if gotCircle.Flt != wantCircle {
		t.Errorf("Circle(2.0): got %v, want %v", gotCircle.Flt, wantCircle)
	}
	gotRect := runProgram(t, prog, "test_rect", nil)
	if gotRect.Flt != 12.0 {
		t.Errorf("Rectangle(3,4): got %v, want 12", gotRect.Flt)
	}
	gotPoint := runProgram(t, prog, "test_point", nil)
	if gotPoint.Flt != 0.0 {
		t.Errorf("Point: got %v, want 0", gotPoint.Flt)
	}
}
