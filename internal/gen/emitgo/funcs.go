package emitgo

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/gen/stmt"
)

func (e *emitter) emitFuncDecl(fd stmt.FuncDecl) {
	name := funcName(fd)

	// Build parameter list.
	var params []string
	for _, p := range fd.Params {
		params = append(params, fmt.Sprintf("%s %s", sanitizeGoIdent(p.Name), p.Type.GoString()))
	}

	retType := fd.ReturnType.GoString()

	e.writeLine("func %s(%s) %s {", name, strings.Join(params, ", "), retType)
	e.indent++

	// Body statements.
	for _, s := range fd.Body {
		e.emitStmt(s)
	}

	// Return expression.
	if fd.Return != nil {
		e.writeIndent()
		e.writef("return ")
		e.emitExpr(fd.Return)
		e.writef("\n")
	}

	e.indent--
	e.writeLine("}")
}

func funcName(fd stmt.FuncDecl) string {
	name := fd.Name
	if fd.Module != "" {
		name = sanitizeModuleName(fd.Module) + "__" + name
	}
	if fd.Exported {
		return capitalize(name)
	}
	return name
}

func sanitizeModuleName(name string) string {
	return strings.ReplaceAll(strings.ReplaceAll(name, "/", "_"), "-", "_")
}

func sanitizeGoIdent(name string) string {
	// Replace $ prefix (ANF temp names like $tmp1) with underscore.
	if strings.HasPrefix(name, "$") {
		name = "_" + name[1:]
	}
	// Replace any remaining illegal characters.
	name = strings.ReplaceAll(name, "$", "_")

	// Go reserved words.
	switch name {
	case "break", "case", "chan", "const", "continue", "default", "defer",
		"else", "fallthrough", "for", "func", "go", "goto", "if", "import",
		"interface", "map", "package", "range", "return", "select", "struct",
		"switch", "type", "var":
		return name + "_"
	}
	return name
}

// --- Statements ---

func (e *emitter) emitStmt(s stmt.Stmt) {
	switch s := s.(type) {
	case stmt.VarDecl:
		e.emitVarDecl(s)
	case stmt.AssignStmt:
		e.writeIndent()
		e.writef("%s = ", sanitizeGoIdent(s.Name))
		e.emitExpr(s.Value)
		e.writef("\n")
	case stmt.ReturnStmt:
		e.writeIndent()
		e.writef("return ")
		e.emitExpr(s.Value)
		e.writef("\n")
	case stmt.ExprStmt:
		e.writeIndent()
		e.emitExpr(s.Value)
		e.writef("\n")
	case stmt.IfStmt:
		e.emitIfStmt(s)
	case stmt.SwitchStmt:
		e.emitSwitchStmt(s)
	}
}

func (e *emitter) emitVarDecl(vd stmt.VarDecl) {
	e.writeIndent()
	if vd.Type != nil {
		e.writef("var %s %s = ", sanitizeGoIdent(vd.Name), vd.Type.GoString())
	} else {
		e.writef("%s := ", sanitizeGoIdent(vd.Name))
	}
	e.emitExpr(vd.Value)
	e.writef("\n")
}

func (e *emitter) emitIfStmt(s stmt.IfStmt) {
	e.writeIndent()
	e.writef("if ")
	e.emitExpr(s.Cond)
	e.writef(" {\n")
	e.indent++
	for _, ts := range s.Then {
		e.emitStmt(ts)
	}
	e.indent--
	if len(s.Else) > 0 {
		e.writeLine("} else {")
		e.indent++
		for _, es := range s.Else {
			e.emitStmt(es)
		}
		e.indent--
	}
	e.writeLine("}")
}

func (e *emitter) emitSwitchStmt(s stmt.SwitchStmt) {
	// Emit scrutinee into a temp variable.
	e.writeIndent()
	e.writef("_scrutinee := ")
	e.emitExpr(s.Scrutinee)
	e.writef("\n")

	// Type-assert to the ADT type (inferred from case tags).
	// For now, use a simple switch on the Kind field.
	e.writeLine("switch _scrutinee.Kind {")
	for _, c := range s.Cases {
		e.emitSwitchCase(c, s)
	}
	if len(s.Default) > 0 {
		e.writeLine("default:")
		e.indent++
		for _, ds := range s.Default {
			e.emitStmt(ds)
		}
		e.indent--
	}
	e.writeLine("}")
}

func (e *emitter) emitSwitchCase(c stmt.SwitchCase, sw stmt.SwitchStmt) {
	e.writeIndent()

	adtType := sw.ADTName
	if adtType != "" {
		e.writef("case %sKind%s:\n", capitalize(adtType), capitalize(c.Tag))
	} else {
		e.writef("case %q: // tag match\n", c.Tag)
	}

	e.indent++

	// Emit bindings — extract fields from the variant struct.
	for _, b := range c.Bindings {
		e.writeIndent()
		if adtType != "" {
			e.writef("%s := _scrutinee.%s.Value%d\n", sanitizeGoIdent(b.Name), capitalize(c.Tag), b.FieldIndex)
		} else {
			e.writef("%s := _scrutinee // binding %d\n", sanitizeGoIdent(b.Name), b.FieldIndex)
		}
	}

	for _, bs := range c.Body {
		e.emitStmt(bs)
	}
	e.indent--
}

// --- Expressions ---

func (e *emitter) emitExpr(expr stmt.Expr) {
	switch ex := expr.(type) {
	case stmt.LitInt:
		e.writef("int64(%d)", ex.Value)
	case stmt.LitFloat:
		e.writef("float64(%f)", ex.Value)
	case stmt.LitBool:
		if ex.Value {
			e.writef("true")
		} else {
			e.writef("false")
		}
	case stmt.LitString:
		e.writef("%q", ex.Value)
	case stmt.LitUnit:
		e.writef("struct{}{}")
	case stmt.VarRef:
		e.writef("%s", sanitizeGoIdent(ex.Name))
	case stmt.GlobalRef:
		e.writef("%s__%s", sanitizeGoIdent(sanitizeModuleName(ex.Module)), sanitizeGoIdent(capitalize(ex.Name)))
	case stmt.BinOp:
		e.emitBinOp(ex)
	case stmt.UnOp:
		e.emitUnOp(ex)
	case stmt.Call:
		e.emitCall(ex)
	case stmt.FieldAccess:
		e.emitExpr(ex.Record)
		e.writef(".%s", capitalize(ex.Field))
	case stmt.RecordLit:
		e.emitRecordLit(ex)
	case stmt.RecordUpdate:
		e.emitRecordUpdate(ex)
	case stmt.ListLit:
		e.emitListLit(ex)
	case stmt.ArrayLit:
		e.emitArrayLit(ex)
	case stmt.TupleLit:
		e.emitTupleLit(ex)
	case stmt.Cons:
		e.writef("append([]interface{}{")
		e.emitExpr(ex.Head)
		e.writef("}, ")
		e.emitExpr(ex.Tail)
		e.writef("...)")
	case stmt.ADTConstructor:
		e.emitADTConstructor(ex)
	case stmt.Lambda:
		e.emitLambda(ex)
	case stmt.TypeAssert:
		e.emitExpr(ex.Value)
		e.writef(".(%s)", ex.Type.GoString())
	case stmt.IfExpr:
		e.emitIfExpr(ex)
	case stmt.BuiltinCall:
		e.emitBuiltinCall(ex)
	default:
		e.writef("nil /* unknown expr %T */", expr)
	}
}

func (e *emitter) emitBinOp(b stmt.BinOp) {
	if b.Op == stmt.OpConcat {
		// String concatenation.
		e.emitExpr(b.Left)
		e.writef(" + ")
		e.emitExpr(b.Right)
		return
	}

	e.writef("(")
	e.emitExpr(b.Left)
	e.writef(" %s ", binOpSymbol(b.Op))
	e.emitExpr(b.Right)
	e.writef(")")
}

func binOpSymbol(op stmt.BinOpKind) string {
	switch op {
	case stmt.OpAdd:
		return "+"
	case stmt.OpSub:
		return "-"
	case stmt.OpMul:
		return "*"
	case stmt.OpDiv:
		return "/"
	case stmt.OpMod:
		return "%"
	case stmt.OpEq:
		return "=="
	case stmt.OpNeq:
		return "!="
	case stmt.OpLt:
		return "<"
	case stmt.OpLte:
		return "<="
	case stmt.OpGt:
		return ">"
	case stmt.OpGte:
		return ">="
	case stmt.OpAnd:
		return "&&"
	case stmt.OpOr:
		return "||"
	case stmt.OpConcat:
		return "+"
	default:
		return "+"
	}
}

func (e *emitter) emitUnOp(u stmt.UnOp) {
	switch u.Op {
	case stmt.OpNeg:
		e.writef("-(")
		e.emitExpr(u.Operand)
		e.writef(")")
	case stmt.OpNot:
		e.writef("!(")
		e.emitExpr(u.Operand)
		e.writef(")")
	}
}

func (e *emitter) emitCall(c stmt.Call) {
	e.emitExpr(c.Func)
	e.writef("(")
	for i, arg := range c.Args {
		if i > 0 {
			e.writef(", ")
		}
		e.emitExpr(arg)
	}
	e.writef(")")
}

func (e *emitter) emitRecordLit(r stmt.RecordLit) {
	if r.TypeName != "" {
		e.writef("%s{", capitalize(r.TypeName))
	} else {
		e.writef("struct{}{") // anonymous struct — shouldn't happen often
	}
	for i, f := range r.Fields {
		if i > 0 {
			e.writef(", ")
		}
		e.writef("%s: ", capitalize(f.Name))
		e.emitExpr(f.Value)
	}
	e.writef("}")
}

func (e *emitter) emitRecordUpdate(r stmt.RecordUpdate) {
	// Record update: copy base, then override fields.
	// Go doesn't have native record update syntax, so we generate:
	// func() TypeName { tmp := base; tmp.Field = val; return tmp }()
	e.writef("func() interface{} {\n")
	e.indent++
	e.writeIndent()
	e.writef("_tmp := ")
	e.emitExpr(r.Base)
	e.writef("\n")
	for _, f := range r.Fields {
		e.writeIndent()
		e.writef("_tmp.%s = ", capitalize(f.Name))
		e.emitExpr(f.Value)
		e.writef("\n")
	}
	e.writeLine("return _tmp")
	e.indent--
	e.writeIndent()
	e.writef("}()")
}

func (e *emitter) emitListLit(l stmt.ListLit) {
	e.writef("[]%s{", l.ElemType.GoString())
	for i, el := range l.Elems {
		if i > 0 {
			e.writef(", ")
		}
		e.emitExpr(el)
	}
	e.writef("}")
}

func (e *emitter) emitArrayLit(a stmt.ArrayLit) {
	e.writef("[]%s{", a.ElemType.GoString())
	for i, el := range a.Elems {
		if i > 0 {
			e.writef(", ")
		}
		e.emitExpr(el)
	}
	e.writef("}")
}

func (e *emitter) emitTupleLit(t stmt.TupleLit) {
	// Tuples compiled as anonymous structs or named Tuple types.
	n := len(t.Elems)
	e.writef("Tuple%d{", n)
	for i, el := range t.Elems {
		if i > 0 {
			e.writef(", ")
		}
		e.writef("V%d: ", i)
		e.emitExpr(el)
	}
	e.writef("}")
}

func (e *emitter) emitADTConstructor(a stmt.ADTConstructor) {
	ctorFunc := "New" + capitalize(a.TypeName) + capitalize(a.Tag)
	e.writef("%s(", ctorFunc)
	for i, arg := range a.Args {
		if i > 0 {
			e.writef(", ")
		}
		e.emitExpr(arg)
	}
	e.writef(")")
}

func (e *emitter) emitLambda(l stmt.Lambda) {
	e.writef("func(")
	for i, p := range l.Params {
		if i > 0 {
			e.writef(", ")
		}
		e.writef("%s %s", sanitizeGoIdent(p.Name), p.Type.GoString())
	}
	e.writef(") ")

	// Infer return type from the return expression or use interface{}.
	e.writef("interface{} {\n")
	e.indent++
	for _, s := range l.Body {
		e.emitStmt(s)
	}
	if l.Return != nil {
		e.writeIndent()
		e.writef("return ")
		e.emitExpr(l.Return)
		e.writef("\n")
	}
	e.indent--
	e.writeIndent()
	e.writef("}")
}

func (e *emitter) emitIfExpr(ie stmt.IfExpr) {
	// Go doesn't have ternary, so use IIFE.
	e.writef("func() interface{} {\n")
	e.indent++
	e.writeIndent()
	e.writef("if ")
	e.emitExpr(ie.Cond)
	e.writef(" {\n")
	e.indent++
	e.writeIndent()
	e.writef("return ")
	e.emitExpr(ie.Then)
	e.writef("\n")
	e.indent--
	e.writeLine("}")
	e.writeIndent()
	e.writef("return ")
	e.emitExpr(ie.Else)
	e.writef("\n")
	e.indent--
	e.writeIndent()
	e.writef("}()")
}

func (e *emitter) emitBuiltinCall(bc stmt.BuiltinCall) {
	// Map builtin names to Go function calls.
	e.writef("%s(", goBuiltinName(bc.Name))
	for i, arg := range bc.Args {
		if i > 0 {
			e.writef(", ")
		}
		e.emitExpr(arg)
	}
	e.writef(")")
}

func goBuiltinName(name string) string {
	// Strip leading underscore for Go function name.
	if strings.HasPrefix(name, "_") {
		return "Builtin" + capitalize(strings.TrimPrefix(name, "_"))
	}
	return name
}
