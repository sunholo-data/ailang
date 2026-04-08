package lower

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/gen/stmt"
	"github.com/sunholo/ailang/internal/types"
)

// LowerProgram converts a Core program + AST (for type declarations) into
// a Statement IR Program ready for emission.
//
// This is the main entry point for the lowering pipeline:
//
//	pipeline.Run(ModeCheck) → Core + CoreTypeInfo + AST
//	lower.LowerProgram(core, cti, ast) → stmt.Program
//	emitter.Emit(program) → Go/Rust/etc. source code
func LowerProgram(
	coreProg *core.Program,
	cti types.CoreTypeInfo,
	astFile *ast.File,
	pkgName string,
) (*stmt.Program, error) {
	prog := &stmt.Program{
		Package: pkgName,
	}

	// Phase 1: Lower type declarations from AST.
	if astFile != nil {
		prog.TypeDecls = lowerTypeDecls(astFile)
	}

	// Phase 2: Lower function declarations from Core.
	if coreProg != nil {
		funcs, err := lowerFuncDecls(coreProg, cti)
		if err != nil {
			return nil, fmt.Errorf("lowering functions: %w", err)
		}
		prog.FuncDecls = funcs
	}

	// Phase 3: Collect required imports.
	prog.Imports = collectImports(prog)

	// Phase 4: Validate the result.
	if err := stmt.Validate(prog); err != nil {
		return nil, fmt.Errorf("Statement IR validation failed: %w", err)
	}

	return prog, nil
}

// LowerMultiModule handles multi-module compilation by lowering each module
// and merging the results into a single Program.
func LowerMultiModule(
	modules map[string]*ModuleInput,
	pkgName string,
) (*stmt.Program, error) {
	prog := &stmt.Program{
		Package: pkgName,
	}

	for modName, mod := range modules {
		// Lower type declarations.
		if mod.AST != nil {
			typeDecls := lowerTypeDecls(mod.AST)
			prog.TypeDecls = append(prog.TypeDecls, typeDecls...)
		}

		// Lower function declarations.
		if mod.Core != nil {
			funcs, err := lowerFuncDecls(mod.Core, mod.CTI)
			if err != nil {
				return nil, fmt.Errorf("module %s: %w", modName, err)
			}
			// Tag functions with their source module.
			for i := range funcs {
				funcs[i].Module = modName
			}
			prog.FuncDecls = append(prog.FuncDecls, funcs...)
		}
	}

	prog.Imports = collectImports(prog)

	if err := stmt.Validate(prog); err != nil {
		return nil, fmt.Errorf("Statement IR validation failed: %w", err)
	}

	return prog, nil
}

// ModuleInput bundles the artifacts needed to lower one module.
type ModuleInput struct {
	Core *core.Program
	CTI  types.CoreTypeInfo
	AST  *ast.File
}

// lowerTypeDecls extracts type declarations from the AST.
func lowerTypeDecls(astFile *ast.File) []stmt.TypeDecl {
	var result []stmt.TypeDecl

	for _, node := range astFile.Decls {
		if td, ok := node.(*ast.TypeDecl); ok {
			result = append(result, LowerTypeDecl(td))
		}
	}
	// Also check Statements (some type decls may be there).
	for _, node := range astFile.Statements {
		if td, ok := node.(*ast.TypeDecl); ok {
			result = append(result, LowerTypeDecl(td))
		}
	}

	return result
}

// lowerFuncDecls converts Core top-level declarations into FuncDecls.
func lowerFuncDecls(coreProg *core.Program, cti types.CoreTypeInfo) ([]stmt.FuncDecl, error) {
	var result []stmt.FuncDecl

	for _, decl := range coreProg.Decls {
		fds, err := lowerTopLevelDeclSafe(decl, coreProg.Meta, cti)
		if err != nil {
			return nil, err
		}
		result = append(result, fds...)
	}

	return result, nil
}

// lowerTopLevelDeclSafe wraps lowerTopLevelDecl with a panic recovery so a
// single function that trips a lower-pass fast-fail (e.g. non-tail-position
// match lowering) is degraded to an EvalOnly stub instead of aborting the
// entire image compile. This is the M-BYTECODE-BATCH mechanism that lets
// --bytecode --batch run on real-world programs where a minority of
// functions use shapes the lower pass can't emit yet. At call time, the
// bridge dispatches these stubs to the evaluator.
//
// The recovered declaration is returned with at least a best-effort name
// (harvested from the Core binding), zero params, and LowerError set to the
// panic reason. The compiler phase inspects LowerError and marks the proto
// EvalOnly accordingly.
func lowerTopLevelDeclSafe(
	e core.CoreExpr,
	meta map[string]*core.DeclMeta,
	cti types.CoreTypeInfo,
) (result []stmt.FuncDecl, err error) {
	defer func() {
		if r := recover(); r != nil {
			// Extract any names we can identify for the offending
			// declaration so the EvalOnly stubs can still be resolved by
			// the bridge at call time. For LetRec we may emit multiple
			// stubs; for a plain Let we emit one.
			stubs := makeEvalOnlyStubs(e, fmt.Sprintf("lower panic: %v", r))
			if len(stubs) == 0 {
				err = fmt.Errorf("lower: %v", r)
				return
			}
			result = stubs
			err = nil
		}
	}()
	return lowerTopLevelDecl(e, meta, cti)
}

// makeEvalOnlyStubs emits placeholder FuncDecls for the names exposed by a
// top-level Core binding whose body failed to lower. The stubs carry no
// body — only Name, Params (for arity), Exported, and LowerError — which
// is enough for the compiler phase to tag them EvalOnly and for the bridge
// to dispatch calls through the evaluator. Parameter arity is preserved
// from the Core Lambda so that call-site arity checks in the VM runner
// accept the right number of arguments before delegating to the bridge.
func makeEvalOnlyStubs(e core.CoreExpr, reason string) []stmt.FuncDecl {
	switch e := e.(type) {
	case *core.Let:
		return []stmt.FuncDecl{makeStub(e.Name, e.Value, reason)}
	case *core.LetRec:
		stubs := make([]stmt.FuncDecl, 0, len(e.Bindings))
		for _, b := range e.Bindings {
			stubs = append(stubs, makeStub(b.Name, b.Value, reason))
		}
		return stubs
	}
	return nil
}

// makeStub builds an EvalOnly placeholder FuncDecl with the right arity.
// It sniffs the Core binding value for a Lambda to recover parameter
// names; everything else is ignored because the body never runs.
func makeStub(name string, value core.CoreExpr, reason string) stmt.FuncDecl {
	params := coreLambdaParams(value)
	stubParams := make([]stmt.Param, len(params))
	for i, p := range params {
		stubParams[i] = stmt.Param{Name: p, Type: placeholderType}
	}
	return stmt.FuncDecl{
		Name:       name,
		Params:     stubParams,
		ReturnType: placeholderType,
		Exported:   true,
		LowerError: reason,
	}
}

// coreLambdaParams peels outer Lambda / DictAbs wrappers to find the
// user-visible parameter list. Returns nil for nullary or non-lambda
// bindings — the Compile phase handles zero-arg stubs fine.
func coreLambdaParams(e core.CoreExpr) []string {
	for {
		switch v := e.(type) {
		case *core.Lambda:
			return v.Params
		case *core.DictAbs:
			// Dictionary abstraction wraps the user lambda. Dict params
			// are synthetic and invisible at the call boundary; unwrap.
			e = v.Body
			continue
		}
		return nil
	}
}

// placeholderType is used for stub param/return types so validate.go
// doesn't reject the stub. The stub never reaches the emitter, so the
// actual type content is irrelevant — any non-nil ResolvedType works.
var placeholderType stmt.ResolvedType = stmt.PrimitiveType{Kind: stmt.PrimUnit}

// lowerTopLevelDecl converts a single Core top-level expression into one or
// more FuncDecls. Top-level Core expressions are typically Let bindings
// wrapping Lambdas (non-recursive) or LetRec with one or more bindings
// (recursive / mutually-recursive).
func lowerTopLevelDecl(
	e core.CoreExpr,
	meta map[string]*core.DeclMeta,
	cti types.CoreTypeInfo,
) ([]stmt.FuncDecl, error) {
	switch e := e.(type) {
	case *core.Let:
		fd := bindingToFuncDecl(e.Name, e.Value, meta, cti)
		if fd == nil {
			return nil, nil
		}
		return []stmt.FuncDecl{*fd}, nil

	case *core.LetRec:
		// Recursive (and mutually-recursive) top-level declarations.
		// Each binding becomes its own FuncDecl. Recursive references
		// inside the bodies are bare core.Vars and are rewritten to
		// module-qualified names later by QualifyFuncRefs.
		var result []stmt.FuncDecl
		for _, b := range e.Bindings {
			fd := bindingToFuncDecl(b.Name, b.Value, meta, cti)
			if fd != nil {
				result = append(result, *fd)
			}
		}
		return result, nil
	}

	// Some other top-level shape (REPL expression, etc.) — skip.
	return nil, nil
}

// bindingToFuncDecl turns a single (name, value) top-level binding into a
// FuncDecl. The value is typically a Lambda (or DictAbs wrapping one); if it
// is anything else, the binding is lowered as a nullary function returning
// the value.
func bindingToFuncDecl(
	name string,
	value core.CoreExpr,
	meta map[string]*core.DeclMeta,
	cti types.CoreTypeInfo,
) *stmt.FuncDecl {
	dm := meta[name]
	exported := dm != nil && dm.IsExport

	// Unwrap DictAbs — erase dictionary abstraction.
	if da, ok := value.(*core.DictAbs); ok {
		value = da.Body
	}

	lam, ok := value.(*core.Lambda)
	if !ok {
		// Top-level value binding (not a function). Lower as a 0-arg function.
		retType := resolveExprType(value, cti)
		body, retExpr := FlattenBlock(value, cti)
		file, line := spanOf(value)
		return &stmt.FuncDecl{
			Name:       name,
			ReturnType: retType,
			Body:       body,
			Return:     retExpr,
			Exported:   exported,
			File:       file,
			Line:       line,
		}
	}

	params := lowerParams(lam, cti)
	retType := resolveReturnType(lam, cti)
	body, retExpr := FlattenBlock(lam.Body, cti)
	file, line := spanOf(lam)

	return &stmt.FuncDecl{
		Name:       name,
		Params:     params,
		ReturnType: retType,
		Body:       body,
		Return:     retExpr,
		Exported:   exported,
		File:       file,
		Line:       line,
	}
}

// spanOf returns the source file and line of a Core expression. Prefers the
// original surface span (what users see) over the desugared Core span. Returns
// ("", 0) when no position info is available.
func spanOf(e core.CoreExpr) (string, int) {
	if e == nil {
		return "", 0
	}
	if p := e.OriginalSpan(); p.Line > 0 {
		return p.File, p.Line
	}
	if p := e.Span(); p.Line > 0 {
		return p.File, p.Line
	}
	return "", 0
}

// lowerParams extracts parameter types from a Lambda's type info.
func lowerParams(lam *core.Lambda, cti types.CoreTypeInfo) []stmt.Param {
	params := make([]stmt.Param, len(lam.Params))

	// Try to get function type from CoreTypeInfo.
	var funcType *types.TFunc2
	if t, ok := cti[lam.ID()]; ok {
		funcType, _ = t.(*types.TFunc2)
	}

	for i, name := range lam.Params {
		paramType := stmt.ResolvedType(stmt.InterfaceType{})
		if funcType != nil && i < len(funcType.Params) {
			paramType = ProjectType(funcType.Params[i])
		}
		params[i] = stmt.Param{Name: name, Type: paramType}
	}

	return params
}

// resolveReturnType extracts the return type of a lambda.
func resolveReturnType(lam *core.Lambda, cti types.CoreTypeInfo) stmt.ResolvedType {
	if t, ok := cti[lam.ID()]; ok {
		if ft, ok := t.(*types.TFunc2); ok {
			return ProjectType(ft.Return)
		}
	}
	return stmt.InterfaceType{}
}

// resolveExprType gets the type of an expression from CoreTypeInfo.
func resolveExprType(e core.CoreExpr, cti types.CoreTypeInfo) stmt.ResolvedType {
	if e == nil {
		return stmt.InterfaceType{}
	}
	if t, ok := cti[e.ID()]; ok {
		return ProjectType(t)
	}
	return stmt.InterfaceType{}
}

// collectImports determines which imports are needed based on the program's
// usage of builtins, string operations, etc.
func collectImports(prog *stmt.Program) []stmt.ImportSpec {
	needs := map[string]bool{}

	// Walk all expressions to find what's used.
	for _, fd := range prog.FuncDecls {
		walkExprsInFunc(&fd, func(e stmt.Expr) {
			switch e := e.(type) {
			case stmt.BuiltinCall:
				if strings.HasPrefix(e.Name, "_str_") || strings.HasPrefix(e.Name, "_show") {
					needs["strings"] = true
					needs["fmt"] = true
				}
			case stmt.LitString:
				// String literals may need fmt for formatting.
			}
		})
	}

	var imports []stmt.ImportSpec
	for path := range needs {
		imports = append(imports, stmt.ImportSpec{Path: path})
	}
	return imports
}

// QualifyFuncRefs rewrites VarRef nodes that reference top-level functions
// to use the full module-prefixed name that matches the Go function definition.
// Core IR uses bare Var references for same-module calls; we rewrite those to
// include the module__name prefix (matching what funcName() produces in the emitter).
func QualifyFuncRefs(prog *stmt.Program) {
	// Build a map from bare function name → full Go function name.
	// This matches the naming logic in emitgo.funcName().
	funcFullName := make(map[string]string)
	for _, fd := range prog.FuncDecls {
		fullName := fd.Name
		if fd.Module != "" {
			fullName = sanitizeModName(fd.Module) + "__" + fd.Name
		}
		if fd.Exported {
			fullName = capitalizeFirst(fullName)
		}
		funcFullName[fd.Name] = fullName
	}

	// Rewrite VarRef names that match function names (but not local vars).
	for i := range prog.FuncDecls {
		fd := &prog.FuncDecls[i]
		locals := make(map[string]bool)
		for _, p := range fd.Params {
			locals[p.Name] = true
		}
		collectLocals(fd.Body, locals)

		fd.Body = rewriteStmts(fd.Body, funcFullName, locals)
		fd.Return = rewriteExpr(fd.Return, funcFullName, locals)
	}
}

func sanitizeModName(name string) string {
	return strings.ReplaceAll(strings.ReplaceAll(name, "/", "_"), "-", "_")
}

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func collectLocals(stmts []stmt.Stmt, locals map[string]bool) {
	for _, s := range stmts {
		switch s := s.(type) {
		case stmt.VarDecl:
			locals[s.Name] = true
		case stmt.AssignStmt:
			locals[s.Name] = true
		case stmt.IfStmt:
			collectLocals(s.Then, locals)
			collectLocals(s.Else, locals)
		case stmt.SwitchStmt:
			for _, c := range s.Cases {
				for _, b := range c.Bindings {
					locals[b.Name] = true
				}
				collectLocals(c.Body, locals)
			}
			collectLocals(s.Default, locals)
		}
	}
}

func rewriteStmts(stmts []stmt.Stmt, funcModule map[string]string, locals map[string]bool) []stmt.Stmt {
	result := make([]stmt.Stmt, len(stmts))
	for i, s := range stmts {
		result[i] = rewriteStmt(s, funcModule, locals)
	}
	return result
}

func rewriteStmt(s stmt.Stmt, funcModule map[string]string, locals map[string]bool) stmt.Stmt {
	switch s := s.(type) {
	case stmt.VarDecl:
		s.Value = rewriteExpr(s.Value, funcModule, locals)
		return s
	case stmt.AssignStmt:
		s.Value = rewriteExpr(s.Value, funcModule, locals)
		return s
	case stmt.ReturnStmt:
		s.Value = rewriteExpr(s.Value, funcModule, locals)
		return s
	case stmt.ExprStmt:
		s.Value = rewriteExpr(s.Value, funcModule, locals)
		return s
	case stmt.IfStmt:
		s.Cond = rewriteExpr(s.Cond, funcModule, locals)
		s.Then = rewriteStmts(s.Then, funcModule, locals)
		s.Else = rewriteStmts(s.Else, funcModule, locals)
		return s
	case stmt.SwitchStmt:
		s.Scrutinee = rewriteExpr(s.Scrutinee, funcModule, locals)
		for i := range s.Cases {
			s.Cases[i].Body = rewriteStmts(s.Cases[i].Body, funcModule, locals)
		}
		s.Default = rewriteStmts(s.Default, funcModule, locals)
		return s
	}
	return s
}

func rewriteExpr(e stmt.Expr, funcModule map[string]string, locals map[string]bool) stmt.Expr {
	if e == nil {
		return nil
	}
	switch ex := e.(type) {
	case stmt.VarRef:
		// Only rewrite if name matches a function AND is not a local variable.
		if fullName, ok := funcModule[ex.Name]; ok && !locals[ex.Name] {
			return stmt.VarRef{Name: fullName}
		}
		return ex
	case stmt.BinOp:
		ex.Left = rewriteExpr(ex.Left, funcModule, locals)
		ex.Right = rewriteExpr(ex.Right, funcModule, locals)
		return ex
	case stmt.UnOp:
		ex.Operand = rewriteExpr(ex.Operand, funcModule, locals)
		return ex
	case stmt.Call:
		ex.Func = rewriteExpr(ex.Func, funcModule, locals)
		for i := range ex.Args {
			ex.Args[i] = rewriteExpr(ex.Args[i], funcModule, locals)
		}
		return ex
	case stmt.FieldAccess:
		ex.Record = rewriteExpr(ex.Record, funcModule, locals)
		return ex
	case stmt.RecordLit:
		for i := range ex.Fields {
			ex.Fields[i].Value = rewriteExpr(ex.Fields[i].Value, funcModule, locals)
		}
		return ex
	case stmt.RecordUpdate:
		ex.Base = rewriteExpr(ex.Base, funcModule, locals)
		for i := range ex.Fields {
			ex.Fields[i].Value = rewriteExpr(ex.Fields[i].Value, funcModule, locals)
		}
		return ex
	case stmt.ListLit:
		for i := range ex.Elems {
			ex.Elems[i] = rewriteExpr(ex.Elems[i], funcModule, locals)
		}
		return ex
	case stmt.ArrayLit:
		for i := range ex.Elems {
			ex.Elems[i] = rewriteExpr(ex.Elems[i], funcModule, locals)
		}
		return ex
	case stmt.TupleLit:
		for i := range ex.Elems {
			ex.Elems[i] = rewriteExpr(ex.Elems[i], funcModule, locals)
		}
		return ex
	case stmt.Cons:
		ex.Head = rewriteExpr(ex.Head, funcModule, locals)
		ex.Tail = rewriteExpr(ex.Tail, funcModule, locals)
		return ex
	case stmt.ADTConstructor:
		for i := range ex.Args {
			ex.Args[i] = rewriteExpr(ex.Args[i], funcModule, locals)
		}
		return ex
	case stmt.Lambda:
		// Lambda introduces new locals — don't rewrite those.
		innerLocals := make(map[string]bool)
		for k, v := range locals {
			innerLocals[k] = v
		}
		for _, p := range ex.Params {
			innerLocals[p.Name] = true
		}
		collectLocals(ex.Body, innerLocals)
		ex.Body = rewriteStmts(ex.Body, funcModule, innerLocals)
		ex.Return = rewriteExpr(ex.Return, funcModule, innerLocals)
		return ex
	case stmt.TypeAssert:
		ex.Value = rewriteExpr(ex.Value, funcModule, locals)
		return ex
	case stmt.IfExpr:
		ex.Cond = rewriteExpr(ex.Cond, funcModule, locals)
		ex.Then = rewriteExpr(ex.Then, funcModule, locals)
		ex.Else = rewriteExpr(ex.Else, funcModule, locals)
		return ex
	case stmt.BuiltinCall:
		for i := range ex.Args {
			ex.Args[i] = rewriteExpr(ex.Args[i], funcModule, locals)
		}
		return ex
	}
	return e
}

// walkExprsInFunc visits all expressions in a function declaration.
func walkExprsInFunc(fd *stmt.FuncDecl, visit func(stmt.Expr)) {
	for _, s := range fd.Body {
		walkExprsInStmt(s, visit)
	}
	if fd.Return != nil {
		walkExpr(fd.Return, visit)
	}
}

func walkExprsInStmt(s stmt.Stmt, visit func(stmt.Expr)) {
	switch s := s.(type) {
	case stmt.VarDecl:
		if s.Value != nil {
			walkExpr(s.Value, visit)
		}
	case stmt.AssignStmt:
		walkExpr(s.Value, visit)
	case stmt.ReturnStmt:
		walkExpr(s.Value, visit)
	case stmt.ExprStmt:
		walkExpr(s.Value, visit)
	case stmt.IfStmt:
		walkExpr(s.Cond, visit)
		for _, ts := range s.Then {
			walkExprsInStmt(ts, visit)
		}
		for _, es := range s.Else {
			walkExprsInStmt(es, visit)
		}
	case stmt.SwitchStmt:
		walkExpr(s.Scrutinee, visit)
		for _, c := range s.Cases {
			for _, bs := range c.Body {
				walkExprsInStmt(bs, visit)
			}
		}
		for _, ds := range s.Default {
			walkExprsInStmt(ds, visit)
		}
	}
}

func walkExpr(e stmt.Expr, visit func(stmt.Expr)) {
	if e == nil {
		return
	}
	visit(e)

	switch e := e.(type) {
	case stmt.BinOp:
		walkExpr(e.Left, visit)
		walkExpr(e.Right, visit)
	case stmt.UnOp:
		walkExpr(e.Operand, visit)
	case stmt.Call:
		walkExpr(e.Func, visit)
		for _, a := range e.Args {
			walkExpr(a, visit)
		}
	case stmt.FieldAccess:
		walkExpr(e.Record, visit)
	case stmt.RecordLit:
		for _, f := range e.Fields {
			walkExpr(f.Value, visit)
		}
	case stmt.RecordUpdate:
		walkExpr(e.Base, visit)
		for _, f := range e.Fields {
			walkExpr(f.Value, visit)
		}
	case stmt.ListLit:
		for _, el := range e.Elems {
			walkExpr(el, visit)
		}
	case stmt.ArrayLit:
		for _, el := range e.Elems {
			walkExpr(el, visit)
		}
	case stmt.TupleLit:
		for _, el := range e.Elems {
			walkExpr(el, visit)
		}
	case stmt.Cons:
		walkExpr(e.Head, visit)
		walkExpr(e.Tail, visit)
	case stmt.ADTConstructor:
		for _, a := range e.Args {
			walkExpr(a, visit)
		}
	case stmt.Lambda:
		for _, s := range e.Body {
			walkExprsInStmt(s, visit)
		}
		if e.Return != nil {
			walkExpr(e.Return, visit)
		}
	case stmt.TypeAssert:
		walkExpr(e.Value, visit)
	case stmt.IfExpr:
		walkExpr(e.Cond, visit)
		walkExpr(e.Then, visit)
		walkExpr(e.Else, visit)
	case stmt.BuiltinCall:
		for _, a := range e.Args {
			walkExpr(a, visit)
		}
	}
}
