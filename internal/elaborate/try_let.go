package elaborate

import (
	"fmt"
	"strings"
	"sync"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/types"
)

// M-AI-GO-TO-LANGUAGE M2 (D2=a): `let`-bound `?` propagation for Result & Option.
//
// Desugars `let x = expr?` (and `let (a, b) = expr?` / `let {a, b} = expr?`)
// along the return spine of functions / function literals whose declared return
// type is `Result[_, E]` (`Ok`/`Err`) or `Option[_]` (`Some`/`None`):
//
//   - Result[_, E]: match expr { Ok(x) => <cont>, Err($try_err) => Err($try_err) }
//   - Option[_]:    match expr { Some(x) => <cont>, None => None }
//
// Rejects:
//   - TC_TRY_RETURN_TYPE when used outside a function or in a function whose
//     declared return type is neither Result[_, E] nor Option[_].
//   - TC_TRY_NON_SPINE when used inside a non-tail expression (e.g. a let RHS
//     block `let x = { let y = r?; y + 1 } in Ok(x)`), guaranteeing `Ok(Err(e))`
//     can never occur without ANF hoisting across bindings.

const (
	CodeTryReturnType = "TC_TRY_RETURN_TYPE"
	CodeTryNonSpine   = "TC_TRY_NON_SPINE"
)

type tryKind int

const (
	tryKindInvalid tryKind = iota
	tryKindResult
	tryKindOption
)

type tryOwnerKind int

const (
	tryOwnerFuncDecl tryOwnerKind = iota
	tryOwnerFuncLit
	tryOwnerLambda
)

type tryFrame struct {
	ownerKind    tryOwnerKind
	funcName     string
	funcPos      ast.Pos
	retType      ast.Type
	rootBody     ast.Expr
	enteredBody  bool // distinguishes elaborateFuncDecl's wrapper Lambda from nested \x. lambdas
	computed     bool
	kind         tryKind
	typeName     string
	typeModule   string
	spineLets    map[*ast.Let]bool
	spineLetPats map[*ast.LetPattern]bool
}

var (
	tryFramesMu sync.Mutex
	tryFrames   = map[*Elaborator][]*tryFrame{}
)

func (e *Elaborator) pushTryFrame(f *tryFrame) {
	tryFramesMu.Lock()
	tryFrames[e] = append(tryFrames[e], f)
	tryFramesMu.Unlock()
}

func (e *Elaborator) closeTryScope() {
	tryFramesMu.Lock()
	stack := tryFrames[e]
	if len(stack) <= 1 {
		delete(tryFrames, e)
	} else {
		tryFrames[e] = stack[:len(stack)-1]
	}
	tryFramesMu.Unlock()
}

func (e *Elaborator) currentTryFrame() *tryFrame {
	tryFramesMu.Lock()
	stack := tryFrames[e]
	var top *tryFrame
	if len(stack) > 0 {
		top = stack[len(stack)-1]
	}
	tryFramesMu.Unlock()
	return top
}

// openTryFuncDecl pushes the try-boundary frame for a top-level/module FuncDecl
// (hooked via openSignature in rigid_sig.go, popped via closeSignature).
func (e *Elaborator) openTryFuncDecl(fn *ast.FuncDecl) {
	if fn == nil {
		e.pushTryFrame(&tryFrame{ownerKind: tryOwnerFuncDecl})
		return
	}
	e.pushTryFrame(&tryFrame{
		ownerKind: tryOwnerFuncDecl,
		funcName:  fn.Name,
		funcPos:   fn.Pos,
		retType:   fn.ReturnType,
		rootBody:  fn.Body,
	})
}

// enterTryFuncLit pushes the try-boundary frame for an inline `func(...) -> R { ... }`.
func (e *Elaborator) enterTryFuncLit(fl *ast.FuncLit) func() {
	e.pushTryFrame(&tryFrame{
		ownerKind: tryOwnerFuncLit,
		funcPos:   fl.Pos,
		retType:   fl.ReturnType,
		rootBody:  fl.Body,
	})
	return e.closeTryScope
}

// enterTryLambda handles `*ast.Lambda`: if it is `elaborateFuncDecl`'s wrapper
// lambda (`lam.Pos == fn.Pos` and `!top.enteredBody`), it reuses the open
// FuncDecl frame; otherwise it pushes an unannotated `\x. ...` lambda boundary
// so `?` inside a nested lambda cannot borrow an outer function's return type.
func (e *Elaborator) enterTryLambda(lam *ast.Lambda) func() {
	if top := e.currentTryFrame(); top != nil && top.ownerKind == tryOwnerFuncDecl && !top.enteredBody && top.funcPos == lam.Pos {
		top.enteredBody = true
		return func() {}
	}
	e.pushTryFrame(&tryFrame{
		ownerKind: tryOwnerLambda,
		funcPos:   lam.Pos,
		retType:   nil,
		rootBody:  lam.Body,
	})
	return e.closeTryScope
}

func (e *Elaborator) ensureTryFrameComputed(f *tryFrame) {
	if f.computed {
		return
	}
	f.computed = true
	f.kind, f.typeName, f.typeModule = e.classifyTryReturnType(f.retType)
	f.spineLets = make(map[*ast.Let]bool)
	f.spineLetPats = make(map[*ast.LetPattern]bool)
	markTrySpine(f.rootBody, f.spineLets, f.spineLetPats)
}

// classifyTryReturnType checks whether ret is Result[T, E] or Option[T] (either
// directly or via a local/imported type alias).
func (e *Elaborator) classifyTryReturnType(ret ast.Type) (tryKind, string, string) {
	for t := ret; t != nil; {
		lbl, ok := t.(*ast.LabelledType)
		if !ok {
			ret = t
			break
		}
		t = lbl.Base
	}
	if ret == nil {
		return tryKindInvalid, "", ""
	}

	if tapp, ok := ret.(*ast.TypeApp); ok {
		switch {
		case tapp.Constructor == "Result" && len(tapp.Args) == 2:
			if info, exists := e.constructors["Ok"]; exists && info.Arity >= 0 && info.Arity != 1 {
				return tryKindInvalid, "", ""
			}
			return tryKindResult, "Result", e.resolveTryTypeModule("Result", tapp.Pos, "std/result")
		case tapp.Constructor == "Option" && len(tapp.Args) == 1:
			if info, exists := e.constructors["Some"]; exists && info.Arity >= 0 && info.Arity != 1 {
				return tryKindInvalid, "", ""
			}
			return tryKindOption, "Option", e.resolveTryTypeModule("Option", tapp.Pos, "std/option")
		}
	}

	// Support type aliases expanding to Result[T, E] or Option[T].
	internal := e.astTypeToInternalType(ret)
	for depth := 0; depth < 8 && internal != nil; depth++ {
		switch it := internal.(type) {
		case *types.TApp:
			if con, ok := it.Constructor.(*types.TCon); ok {
				base := bareNominalName(con.Name)
				if base == "Result" && len(it.Args) == 2 {
					mod := con.Module
					if mod == "" {
						mod = "std/result"
					}
					return tryKindResult, "Result", mod
				}
				if base == "Option" && len(it.Args) == 1 {
					mod := con.Module
					if mod == "" {
						mod = "std/option"
					}
					return tryKindOption, "Option", mod
				}
				if target, isAlias := e.typeAliases[con.Name]; isAlias {
					internal = target
					continue
				}
			}
		case *types.TCon:
			if target, isAlias := e.typeAliases[it.Name]; isAlias {
				internal = target
				continue
			}
			if target, isAlias := e.typeAliases[bareNominalName(it.Name)]; isAlias {
				internal = target
				continue
			}
		}
		break
	}

	return tryKindInvalid, "", ""
}

func (e *Elaborator) resolveTryTypeModule(typeName string, pos ast.Pos, stdDefault string) string {
	if e.typeScope != nil {
		if r, err := e.typeScope.Lookup(typeName); err == nil && r.Module != "" && r.Module != replModuleID {
			return r.Module
		}
	}
	return stdDefault
}

func bareNominalName(name string) string {
	if idx := strings.LastIndex(name, "::"); idx >= 0 {
		return name[idx+2:]
	}
	return name
}

// markTrySpine records every *ast.Let and *ast.LetPattern that sits on the
// tail return spine of root. Any `IsTry` node NOT in these sets sits inside a
// non-tail position (e.g. a let RHS, call argument, match scrutinee, if
// condition, or discarded block statement) and is rejected with TC_TRY_NON_SPINE.
func markTrySpine(expr ast.Expr, lets map[*ast.Let]bool, letPats map[*ast.LetPattern]bool) {
	if expr == nil {
		return
	}
	switch n := expr.(type) {
	case *ast.Let:
		lets[n] = true
		if n.Body != nil {
			markTrySpine(n.Body, lets, letPats)
		}
	case *ast.LetPattern:
		letPats[n] = true
		if n.Body != nil {
			markTrySpine(n.Body, lets, letPats)
		}
	case *ast.LetRec:
		if n.Body != nil {
			markTrySpine(n.Body, lets, letPats)
		}
	case *ast.Block:
		if len(n.Exprs) == 0 {
			return
		}
		for i := 0; i < len(n.Exprs)-1; i++ {
			switch st := n.Exprs[i].(type) {
			case *ast.Let:
				if st.Body == nil {
					lets[st] = true
				}
			case *ast.LetPattern:
				if st.Body == nil {
					letPats[st] = true
				}
			}
		}
		markTrySpine(n.Exprs[len(n.Exprs)-1], lets, letPats)
	case *ast.If:
		markTrySpine(n.Then, lets, letPats)
		markTrySpine(n.Else, lets, letPats)
	case *ast.Match:
		for _, c := range n.Cases {
			if c != nil {
				markTrySpine(c.Body, lets, letPats)
			}
		}
	}
}

func (e *Elaborator) validateTrySite(tryPos, fallbackPos ast.Pos, binderDesc string, isSpine func(*tryFrame) bool) (*tryFrame, error) {
	pos := tryPos
	if pos.Line == 0 {
		pos = fallbackPos
	}
	top := e.currentTryFrame()
	if top == nil {
		return nil, tryReport(
			CodeTryReturnType,
			pos,
			"cannot use '?' outside a function returning Result[T, E] or Option[T]",
			"Move 'let "+binderDesc+" = ...?' inside a function with return type '-> Result[T, E]' or '-> Option[T]'",
		)
	}
	e.ensureTryFrameComputed(top)
	if top.kind == tryKindInvalid {
		if top.ownerKind == tryOwnerLambda {
			return nil, tryReport(
				CodeTryReturnType,
				pos,
				"cannot use '?' inside an unannotated '\\...' lambda; enclosing function must declare return type Result[T, E] or Option[T]",
				"Rewrite the lambda as 'func(...) -> Result[T, E] { ... }' or 'func(...) -> Option[T] { ... }'",
			)
		}
		fnLabel := "enclosing function"
		if top.funcName != "" {
			fnLabel = fmt.Sprintf("function '%s'", top.funcName)
		}
		if top.retType == nil {
			return nil, tryReport(
				CodeTryReturnType,
				pos,
				fmt.Sprintf("cannot use '?' in %s without a declared return type; %s must declare return type Result[T, E] or Option[T]", fnLabel, fnLabel),
				"Add '-> Result[T, E]' or '-> Option[T]' to the function signature",
			)
		}
		return nil, tryReport(
			CodeTryReturnType,
			pos,
			fmt.Sprintf("cannot use '?' in %s returning %s; %s must declare return type Result[T, E] or Option[T]", fnLabel, top.retType.String(), fnLabel),
			"Change the function's return type to Result[T, E] or Option[T], or unwrap explicitly with 'match'",
		)
	}
	if !isSpine(top) {
		fnLabel := "the enclosing function"
		if top.funcName != "" {
			fnLabel = fmt.Sprintf("'%s'", top.funcName)
		}
		return nil, tryReport(
			CodeTryNonSpine,
			pos,
			fmt.Sprintf("cannot use '?' on 'let %s' inside a non-tail expression; '?' must appear on the return spine of %s", binderDesc, fnLabel),
			fmt.Sprintf("Hoist 'let %s = ...?' out to the outer block or let-chain of %s", binderDesc, fnLabel),
		)
	}
	return top, nil
}

func tryReport(code string, pos ast.Pos, msg, suggestion string) error {
	return errors.WrapReport(&errors.Report{
		Schema:  "ailang.error/v1",
		Code:    code,
		Phase:   "typecheck",
		Message: msg,
		Span:    &ast.Span{Start: pos, End: pos},
		Fix:     &errors.Fix{Suggestion: suggestion},
	})
}

func (e *Elaborator) freshTryVar(role string) string {
	e.freshVarNum++
	return fmt.Sprintf("$try_%s_%d", role, e.freshVarNum)
}

func (e *Elaborator) normalizeTryLet(let *ast.Let) (core.CoreExpr, error) {
	frame, err := e.validateTrySite(let.TryPos, let.Pos, let.Name, func(f *tryFrame) bool {
		return f.spineLets[let]
	})
	if err != nil {
		return nil, err
	}
	tryPos := let.TryPos
	if tryPos.Line == 0 {
		tryPos = let.Pos
	}
	scrutinee, binds, err := e.normalizeToAtomic(let.Value)
	if err != nil {
		return nil, err
	}
	var cont core.CoreExpr
	if let.Body == nil {
		cont = &core.Lit{
			CoreNode: e.makeNode(let.Pos),
			Kind:     core.UnitLit,
			Value:    "()",
		}
	} else {
		if let.Name != "_" {
			e.pushScope(let.Name)
		}
		cont, err = e.normalize(let.Body)
		if let.Name != "_" {
			e.popScope()
		}
		if err != nil {
			return nil, err
		}
	}
	return e.finishTryLetMatch(frame, scrutinee, binds, let.Name, let.Type, let.Pos, tryPos, cont)
}

func (e *Elaborator) normalizeTryBlockLet(let *ast.Let, cont core.CoreExpr) (core.CoreExpr, error) {
	frame, err := e.validateTrySite(let.TryPos, let.Pos, let.Name, func(f *tryFrame) bool {
		return f.spineLets[let]
	})
	if err != nil {
		return nil, err
	}
	tryPos := let.TryPos
	if tryPos.Line == 0 {
		tryPos = let.Pos
	}
	scrutinee, binds, err := e.normalizeToAtomic(let.Value)
	if err != nil {
		return nil, err
	}
	return e.finishTryLetMatch(frame, scrutinee, binds, let.Name, let.Type, let.Pos, tryPos, cont)
}

func (e *Elaborator) finishTryLetMatch(
	frame *tryFrame,
	scrutinee core.CoreExpr,
	binds []binding,
	binderName string,
	annotType ast.Type,
	letPos, tryPos ast.Pos,
	cont core.CoreExpr,
) (core.CoreExpr, error) {
	patVar := binderName
	armBody := cont
	if annotType != nil {
		patVar = e.freshTryVar("val")
		letNode := &core.Let{
			CoreNode: e.makeNode(letPos),
			Name:     binderName,
			Value:    &core.Var{CoreNode: e.makeNode(tryPos), Name: patVar},
			Body:     cont,
		}
		if annot := e.astTypeToInternalType(annotType); annot != nil {
			e.letTypeAnnots[letNode.ID()] = annot
		}
		armBody = letNode
	}

	var innerPat core.CorePattern
	if patVar == "_" {
		innerPat = &core.WildcardPattern{}
	} else {
		innerPat = &core.VarPattern{Name: patVar}
	}
	return e.buildTryMatch(frame, scrutinee, binds, innerPat, armBody, tryPos)
}

func (e *Elaborator) normalizeTryLetPattern(lp *ast.LetPattern) (core.CoreExpr, error) {
	frame, err := e.validateTrySite(lp.TryPos, lp.Pos, lp.Pattern.String(), func(f *tryFrame) bool {
		return f.spineLetPats[lp]
	})
	if err != nil {
		return nil, err
	}
	tryPos := lp.TryPos
	if tryPos.Line == 0 {
		tryPos = lp.Pos
	}
	scrutinee, binds, err := e.normalizeToAtomic(lp.Value)
	if err != nil {
		return nil, err
	}
	corePat, err := e.elaboratePattern(lp.Pattern)
	if err != nil {
		return nil, err
	}
	var cont core.CoreExpr
	if lp.Body == nil {
		cont = &core.Lit{
			CoreNode: e.makeNode(lp.Pos),
			Kind:     core.UnitLit,
			Value:    "()",
		}
	} else {
		e.pushScope(corePatternBinders(corePat)...)
		cont, err = e.normalize(lp.Body)
		e.popScope()
		if err != nil {
			return nil, err
		}
	}
	return e.finishTryPatternMatch(frame, scrutinee, binds, corePat, lp.Type, lp.Pos, tryPos, cont)
}

func (e *Elaborator) normalizeTryBlockLetPattern(lp *ast.LetPattern, cont core.CoreExpr) (core.CoreExpr, error) {
	frame, err := e.validateTrySite(lp.TryPos, lp.Pos, lp.Pattern.String(), func(f *tryFrame) bool {
		return f.spineLetPats[lp]
	})
	if err != nil {
		return nil, err
	}
	tryPos := lp.TryPos
	if tryPos.Line == 0 {
		tryPos = lp.Pos
	}
	scrutinee, binds, err := e.normalizeToAtomic(lp.Value)
	if err != nil {
		return nil, err
	}
	corePat, err := e.elaboratePattern(lp.Pattern)
	if err != nil {
		return nil, err
	}
	return e.finishTryPatternMatch(frame, scrutinee, binds, corePat, lp.Type, lp.Pos, tryPos, cont)
}

func (e *Elaborator) finishTryPatternMatch(
	frame *tryFrame,
	scrutinee core.CoreExpr,
	binds []binding,
	corePat core.CorePattern,
	annotType ast.Type,
	letPos, tryPos ast.Pos,
	cont core.CoreExpr,
) (core.CoreExpr, error) {
	rawVar := e.freshTryVar("pat")
	destructMatch := &core.Match{
		CoreNode:   e.makeNode(letPos),
		Scrutinee:  &core.Var{CoreNode: e.makeNode(tryPos), Name: rawVar},
		Arms:       []core.MatchArm{{Pattern: corePat, Body: cont}},
		Exhaustive: true,
	}
	var armBody core.CoreExpr = destructMatch
	if annotType != nil {
		outerVar := e.freshTryVar("val")
		letNode := &core.Let{
			CoreNode: e.makeNode(letPos),
			Name:     rawVar,
			Value:    &core.Var{CoreNode: e.makeNode(tryPos), Name: outerVar},
			Body:     destructMatch,
		}
		if annot := e.astTypeToInternalType(annotType); annot != nil {
			e.letTypeAnnots[letNode.ID()] = annot
		}
		armBody = letNode
		rawVar = outerVar
	}
	return e.buildTryMatch(frame, scrutinee, binds, &core.VarPattern{Name: rawVar}, armBody, tryPos)
}

func (e *Elaborator) buildTryMatch(
	frame *tryFrame,
	scrutinee core.CoreExpr,
	binds []binding,
	okSubPat core.CorePattern,
	okBody core.CoreExpr,
	tryPos ast.Pos,
) (core.CoreExpr, error) {
	switch frame.kind {
	case tryKindResult:
		okName, okType, okMod := e.resolveTryCtor("Ok", frame.typeName, frame.typeModule, 1)
		errName, errType, errMod := e.resolveTryCtor("Err", frame.typeName, frame.typeModule, 1)
		errVar := e.freshTryVar("err")
		errRet := e.makeTryCtorExpr(errName, errType, errMod, []core.CoreExpr{
			&core.Var{CoreNode: e.makeNode(tryPos), Name: errVar},
		}, tryPos)
		matchExpr := &core.Match{
			CoreNode:  e.makeNode(tryPos),
			Scrutinee: scrutinee,
			Arms: []core.MatchArm{
				{
					Pattern: &core.ConstructorPattern{
						Name:       okName,
						TypeName:   okType,
						TypeModule: okMod,
						Args:       []core.CorePattern{okSubPat},
						Pos:        tryPos,
					},
					Body: okBody,
				},
				{
					Pattern: &core.ConstructorPattern{
						Name:       errName,
						TypeName:   errType,
						TypeModule: errMod,
						Args:       []core.CorePattern{&core.VarPattern{Name: errVar}},
						Pos:        tryPos,
					},
					Body: errRet,
				},
			},
		}
		return e.wrapWithBindings(matchExpr, binds), nil

	case tryKindOption:
		someName, someType, someMod := e.resolveTryCtor("Some", frame.typeName, frame.typeModule, 1)
		noneName, noneType, noneMod := e.resolveTryCtor("None", frame.typeName, frame.typeModule, 0)
		noneRet := e.makeTryCtorExpr(noneName, noneType, noneMod, nil, tryPos)
		matchExpr := &core.Match{
			CoreNode:  e.makeNode(tryPos),
			Scrutinee: scrutinee,
			Arms: []core.MatchArm{
				{
					Pattern: &core.ConstructorPattern{
						Name:       someName,
						TypeName:   someType,
						TypeModule: someMod,
						Args:       []core.CorePattern{okSubPat},
						Pos:        tryPos,
					},
					Body: okBody,
				},
				{
					Pattern: &core.ConstructorPattern{
						Name:       noneName,
						TypeName:   noneType,
						TypeModule: noneMod,
						Args:       nil,
						Pos:        tryPos,
					},
					Body: noneRet,
				},
			},
		}
		return e.wrapWithBindings(matchExpr, binds), nil

	default:
		return nil, fmt.Errorf("internal: invalid tryKind %d at %v", frame.kind, tryPos)
	}
}

func (e *Elaborator) resolveTryCtor(ctorName, fallbackType, fallbackMod string, arity int) (string, string, string) {
	if _, amb := e.ambiguousCtors[ctorName]; !amb {
		if info, ok := e.constructors[ctorName]; ok && info.CtorName != "" && info.Arity == arity && info.TypeName == fallbackType {
			return info.CtorName, info.TypeName, info.Module
		}
	}
	return ctorName, fallbackType, fallbackMod
}

func (e *Elaborator) makeTryCtorExpr(ctorName, typeName, typeMod string, args []core.CoreExpr, pos ast.Pos) core.CoreExpr {
	factoryRef := &core.VarGlobal{
		CoreNode: e.makeNode(pos),
		Ref: core.GlobalRef{
			Module: core.ADTModule,
			Name:   core.ADTFactoryName(typeMod, typeName, ctorName),
		},
	}
	if len(args) == 0 {
		return factoryRef
	}
	return &core.App{
		CoreNode: e.makeNode(pos),
		Func:     factoryRef,
		Args:     args,
	}
}
