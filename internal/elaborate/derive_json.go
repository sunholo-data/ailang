package elaborate

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
)

// Hygienic synthetic constructor aliases registered in Elaborator.constructors
// for derived Json codecs. Dollar-suffixed names can never be written in user
// source, while leading uppercase runes keep pattern binder classification
// (isUpperIdent in scc.go / patterns.go) identical to ordinary constructors.
const (
	djJNull   = "JNull$json"
	djJBool   = "JBool$json"
	djJNumber = "JNumber$json"
	djJString = "JString$json"
	djJArray  = "JArray$json"
	djJObject = "JObject$json"
	djJInt    = "JInt$json"

	djOk   = "Ok$result"
	djErr  = "Err$result"
	djSome = "Some$option"
	djNone = "None$option"
)

func hasDeriveKind(decl *ast.TypeDecl, kind ast.DeriveKind) bool {
	if decl == nil {
		return false
	}
	for _, d := range decl.Deriving {
		if d == kind {
			return true
		}
	}
	return false
}

func fileHasDerivingJson(file *ast.File) bool {
	if file == nil {
		return false
	}
	for _, list := range [][]ast.Node{file.Decls, file.Statements} {
		for _, node := range list {
			if td, ok := node.(*ast.TypeDecl); ok && hasDeriveKind(td, ast.DeriveJson) {
				return true
			}
		}
	}
	return false
}

func checkDerivingJsonPoly(file *ast.File, decl *ast.TypeDecl) error {
	if !hasDeriveKind(decl, ast.DeriveJson) {
		return nil
	}
	loc := decl.Pos.String()
	if file != nil {
		loc = sitePos(file, decl.Pos)
	}
	if len(decl.TypeParams) > 0 {
		return fmt.Errorf(
			"TC_DERIVE_JSON_POLY at %s: cannot derive Json for polymorphic type %s[%s]; "+
				"instantiate a concrete type or write encode%s/decode%s functions explicitly",
			loc, decl.Name, strings.Join(decl.TypeParams, ", "), decl.Name, decl.Name)
	}
	if rec, ok := decl.Definition.(*ast.RecordType); ok && rec.Row != nil {
		return fmt.Errorf(
			"TC_DERIVE_JSON_POLY at %s: cannot derive Json for row-polymorphic record type %s; "+
				"use a closed record type or write encode%s/decode%s functions explicitly",
			loc, decl.Name, decl.Name, decl.Name)
	}
	return nil
}

// synthesizeDerivedJsonDecls synthesizes pure encode<Type> and decode<Type>
// AST function declarations for every monomorphic TypeDecl with deriving (Json),
// appending them to file.Funcs and file.Decls prior to SCC call-graph construction.
func (e *Elaborator) synthesizeDerivedJsonDecls(file *ast.File) error {
	if file == nil {
		return nil
	}
	var targets []*ast.TypeDecl
	seen := make(map[string]bool)
	for _, list := range [][]ast.Node{file.Decls, file.Statements} {
		for _, node := range list {
			td, ok := node.(*ast.TypeDecl)
			if !ok || !hasDeriveKind(td, ast.DeriveJson) || seen[td.Name] {
				continue
			}
			seen[td.Name] = true
			if err := checkDerivingJsonPoly(file, td); err != nil {
				return err
			}
			targets = append(targets, td)
		}
	}
	if len(targets) == 0 {
		return nil
	}

	jsonTypeName, err := e.prepareDerivedJsonEnv(file, targets[0])
	if err != nil {
		return err
	}

	for _, decl := range targets {
		gen := &deriveJsonGen{elab: e, file: file, pos: decl.Pos, jsonTypeName: jsonTypeName}
		encodeFn, err := gen.buildEncodeFunc(decl)
		if err != nil {
			return err
		}
		decodeFn, err := gen.buildDecodeFunc(decl)
		if err != nil {
			return err
		}
		file.Funcs = append(file.Funcs, encodeFn, decodeFn)
		file.Decls = append(file.Decls, encodeFn, decodeFn)

		if e.typeScopeImports && e.moduleLoader != nil && decl.Exported {
			if loaded, loadErr := e.moduleLoader.Load(e.ModuleID()); loadErr == nil && loaded != nil {
				if loaded.Exports == nil {
					loaded.Exports = make(map[string]*ast.FuncDecl)
				}
				loaded.Exports[encodeFn.Name] = encodeFn
				loaded.Exports[decodeFn.Name] = decodeFn
			}
		}
	}
	return nil
}

func (e *Elaborator) prepareDerivedJsonEnv(file *ast.File, firstDecl *ast.TypeDecl) (string, error) {
	isSelfJson := e.ModuleID() == "std/json" || (file.Module != nil && file.Module.Path == "std/json")
	var stdJsonImp *ast.ImportDecl
	for _, imp := range file.Imports {
		if imp.Path == "std/json" {
			stdJsonImp = imp
			break
		}
	}
	if !isSelfJson && stdJsonImp == nil {
		return "", fmt.Errorf(
			"IMP_DERIVE_JSON at %s: deriving (Json) on type '%s' requires importing std/json.\n"+
				"  Fix: add 'import std/json (Json)'",
			sitePos(file, firstDecl.Pos), firstDecl.Name)
	}

	jsonTypeName := "Json"
	if !isSelfJson {
		resolved := false
		if stdJsonImp.SymbolAliases != nil {
			if alias, ok := stdJsonImp.SymbolAliases["Json"]; ok && alias != "" {
				jsonTypeName, resolved = alias, true
			}
		}
		if !resolved && e.typeScope != nil {
			if ref, err := e.typeScope.Lookup("Json"); err == nil && ref.Module == "std/json" && ref.Name == "Json" {
				jsonTypeName, resolved = "Json", true
			}
		}
		if !resolved && stdJsonImp.ModuleAlias != "" {
			jsonTypeName, resolved = stdJsonImp.ModuleAlias+".Json", true
		}
		if !resolved && !e.typeScopeImports {
			if e.typeScope == nil {
				e.ensureTypeScope(file)
			}
			if ref, err := e.typeScope.Lookup("Json"); err == nil && ref.Module == e.ModuleID() && !containsTypeDecl(file, "Json") {
				e.typeScope.BindExplicit("Json", typesTypeRef("std/json", "Json"))
				jsonTypeName, resolved = "Json", true
			}
		}
		if !resolved {
			return "", fmt.Errorf(
				"IMP_DERIVE_JSON at %s: type 'Json' from std/json is shadowed in this module; "+
					"alias it via 'import std/json (Json as StdJson)' or 'import std/json as J'",
				sitePos(file, firstDecl.Pos))
		}
	}

	jsonMod := "std/json"
	if !e.typeScopeImports && !isSelfJson {
		if c, ok := e.constructors["JNull"]; ok {
			jsonMod = c.Module
		}
	}
	for _, spec := range []struct {
		alias, canon string
		arity        int
	}{
		{djJNull, "JNull", 0}, {djJBool, "JBool", 1}, {djJNumber, "JNumber", 1},
		{djJString, "JString", 1}, {djJArray, "JArray", 1}, {djJObject, "JObject", 1}, {djJInt, "JInt", 1},
	} {
		e.RegisterImportedConstructorAs(spec.alias, spec.canon, jsonMod, "Json", spec.arity, 0)
	}

	resMod := "std/result"
	if !e.typeScopeImports {
		if c, ok := e.constructors["Ok"]; ok {
			resMod = c.Module
		}
	}
	e.RegisterImportedConstructorAs(djOk, "Ok", resMod, "Result", 1, 2)
	e.RegisterImportedConstructorAs(djErr, "Err", resMod, "Result", 1, 2)

	optMod := "std/option"
	if !e.typeScopeImports {
		if c, ok := e.constructors["Some"]; ok {
			optMod = c.Module
		}
	}
	e.RegisterImportedConstructorAs(djSome, "Some", optMod, "Option", 1, 1)
	e.RegisterImportedConstructorAs(djNone, "None", optMod, "Option", 0, 1)

	return jsonTypeName, nil
}

func containsTypeDecl(file *ast.File, name string) bool {
	for _, n := range LocalTypeNames(file) {
		if n == name {
			return true
		}
	}
	return false
}

type deriveJsonGen struct {
	elab         *Elaborator
	file         *ast.File
	pos          ast.Pos
	jsonTypeName string
	varSeq       int
}

func (g *deriveJsonGen) fresh(prefix string) string {
	g.varSeq++
	return fmt.Sprintf("_dj_%s_%d", prefix, g.varSeq)
}

func (g *deriveJsonGen) ident(name string) *ast.Identifier {
	return &ast.Identifier{Name: name, Pos: g.pos}
}

func (g *deriveJsonGen) builtinIdent(name string) *ast.Identifier {
	return &ast.Identifier{Name: name, ResolveAsBuiltin: true, Pos: g.pos}
}

func (g *deriveJsonGen) strLit(s string) *ast.Literal {
	return &ast.Literal{Kind: ast.StringLit, Value: s, Pos: g.pos}
}

func (g *deriveJsonGen) intLit(n int64) *ast.Literal {
	return &ast.Literal{Kind: ast.IntLit, Value: n, Pos: g.pos}
}

func (g *deriveJsonGen) floatLit(f float64) *ast.Literal {
	return &ast.Literal{Kind: ast.FloatLit, Value: f, Pos: g.pos}
}

func (g *deriveJsonGen) call(fn ast.Expr, args ...ast.Expr) *ast.FuncCall {
	return &ast.FuncCall{Func: fn, Args: args, Pos: g.pos}
}

func (g *deriveJsonGen) callBuiltin(name string, args ...ast.Expr) *ast.FuncCall {
	return g.call(g.builtinIdent(name), args...)
}

func (g *deriveJsonGen) ctorCall(ctor string, args ...ast.Expr) ast.Expr {
	if len(args) == 0 {
		return g.ident(ctor)
	}
	return g.call(g.ident(ctor), args...)
}

func (g *deriveJsonGen) ctorPat(ctor string, bindVars ...string) *ast.ConstructorPattern {
	pats := make([]ast.Pattern, len(bindVars))
	for i, v := range bindVars {
		pats[i] = g.ident(v)
	}
	return &ast.ConstructorPattern{Name: ctor, Patterns: pats, Pos: g.pos}
}

func (g *deriveJsonGen) wildPat() *ast.WildcardPattern {
	return &ast.WildcardPattern{Pos: g.pos}
}

func (g *deriveJsonGen) arm(pat ast.Pattern, body ast.Expr) *ast.Case {
	return &ast.Case{Pattern: pat, Body: body, Pos: g.pos}
}

func (g *deriveJsonGen) matchExpr(scrut ast.Expr, arms ...*ast.Case) *ast.Match {
	return &ast.Match{Expr: scrut, Cases: arms, Pos: g.pos}
}

func (g *deriveJsonGen) letExpr(name string, val, body ast.Expr) *ast.Let {
	return &ast.Let{Name: name, Value: val, Body: body, Pos: g.pos}
}

func (g *deriveJsonGen) ifExpr(cond, thenB, elseB ast.Expr) *ast.If {
	return &ast.If{Condition: cond, Then: thenB, Else: elseB, Pos: g.pos}
}

func (g *deriveJsonGen) binOp(l ast.Expr, op string, r ast.Expr) *ast.BinaryOp {
	return &ast.BinaryOp{Left: l, Op: op, Right: r, Pos: g.pos}
}

func (g *deriveJsonGen) listLit(elems ...ast.Expr) *ast.List {
	return &ast.List{Elements: elems, Pos: g.pos}
}

func (g *deriveJsonGen) kvExpr(key string, val ast.Expr) *ast.Record {
	return &ast.Record{
		Fields: []*ast.Field{
			{Name: "key", Value: g.strLit(key), Pos: g.pos},
			{Name: "value", Value: val, Pos: g.pos},
		},
		Pos: g.pos,
	}
}

func (g *deriveJsonGen) consExpr(head, tail ast.Expr) ast.Expr {
	return g.callBuiltin("::", head, tail)
}

func (g *deriveJsonGen) isStdJsonType(name string) bool {
	if name == g.jsonTypeName {
		return true
	}
	if g.elab.typeScope != nil && !strings.Contains(name, ".") {
		if ref, err := g.elab.typeScope.Lookup(name); err == nil {
			return ref.Module == "std/json" && ref.Name == "Json"
		}
	}
	return false
}

func (g *deriveJsonGen) unwrapOptionType(t ast.Type) (ast.Type, bool) {
	if lt, ok := t.(*ast.LabelledType); ok {
		t = lt.Base
	}
	app, ok := t.(*ast.TypeApp)
	if !ok || len(app.Args) != 1 || app.Constructor != "Option" {
		return nil, false
	}
	if g.elab.typeScope == nil {
		return app.Args[0], true
	}
	if ref, err := g.elab.typeScope.Lookup("Option"); err == nil && ref.Module == "std/option" && ref.Name == "Option" {
		return app.Args[0], true
	}
	return nil, false
}

func allNamedFields(fields []*ast.ConstructorField) bool {
	if len(fields) == 0 {
		return false
	}
	for _, f := range fields {
		if f.Name == "" || f.Name == "tag" {
			return false
		}
	}
	return true
}

func (g *deriveJsonGen) unsupportedTypeErr(ownerType string, t ast.Type) error {
	desc := "<nil>"
	if t != nil {
		desc = t.String()
	}
	return fmt.Errorf(
		"TC_DERIVE_JSON_UNSUPPORTED at %s: cannot derive Json for type '%s': unsupported field/payload type '%s'",
		sitePos(g.file, g.pos), ownerType, desc)
}

func (g *deriveJsonGen) buildEncodeFunc(decl *ast.TypeDecl) (*ast.FuncDecl, error) {
	paramName := "v"
	var body ast.Expr
	var err error

	switch def := decl.Definition.(type) {
	case *ast.RecordType:
		body, err = g.encodeRecord(decl.Name, def, g.ident(paramName))
	case *ast.AlgebraicType:
		body, err = g.encodeADT(decl.Name, def, g.ident(paramName))
	case *ast.TypeAlias:
		body, err = g.encodeValue(decl.Name, def.Target, g.ident(paramName))
	default:
		return nil, fmt.Errorf("TC_DERIVE_JSON_UNSUPPORTED at %s: unsupported type definition %T", sitePos(g.file, decl.Pos), def)
	}
	if err != nil {
		return nil, err
	}

	return &ast.FuncDecl{
		Name:       "encode" + decl.Name,
		Params:     []*ast.Param{{Name: paramName, Type: &ast.SimpleType{Name: decl.Name, Pos: g.pos}, Pos: g.pos}},
		ReturnType: &ast.SimpleType{Name: g.jsonTypeName, Pos: g.pos},
		Body:       body,
		IsPure:     true,
		IsExport:   decl.Exported,
		Pos:        g.pos,
		Origin:     "derived_json_encode",
	}, nil
}

type namedValueField struct {
	name string
	typ  ast.Type
	expr ast.Expr
}

func (g *deriveJsonGen) encodeRecord(ownerType string, rec *ast.RecordType, recExpr ast.Expr) (ast.Expr, error) {
	if rec.Row != nil {
		return nil, fmt.Errorf("TC_DERIVE_JSON_POLY at %s: cannot derive Json for row-polymorphic record in '%s'", sitePos(g.file, g.pos), ownerType)
	}
	fields := make([]namedValueField, len(rec.Fields))
	for i, rf := range rec.Fields {
		fields[i] = namedValueField{name: rf.Name, typ: rf.Type, expr: &ast.RecordAccess{Record: recExpr, Field: rf.Name, Pos: g.pos}}
	}
	kvs, err := g.encodeFieldKVs(ownerType, fields, g.listLit())
	if err != nil {
		return nil, err
	}
	return g.ctorCall(djJObject, kvs), nil
}

func (g *deriveJsonGen) encodeFieldKVs(ownerType string, fields []namedValueField, tail ast.Expr) (ast.Expr, error) {
	hasOption := false
	for _, f := range fields {
		if _, ok := g.unwrapOptionType(f.typ); ok {
			hasOption = true
			break
		}
	}
	if !hasOption {
		if tailList, ok := tail.(*ast.List); ok && len(tailList.Elements) == 0 {
			elems := make([]ast.Expr, len(fields))
			for i, f := range fields {
				ev, err := g.encodeValue(ownerType, f.typ, f.expr)
				if err != nil {
					return nil, err
				}
				elems[i] = g.kvExpr(f.name, ev)
			}
			return g.listLit(elems...), nil
		}
	}

	acc := tail
	for i := len(fields) - 1; i >= 0; i-- {
		f := fields[i]
		if optInner, isOpt := g.unwrapOptionType(f.typ); isOpt {
			restVar, someVar := g.fresh("kvs"), g.fresh("opt")
			encInner, err := g.encodeValue(ownerType, optInner, g.ident(someVar))
			if err != nil {
				return nil, err
			}
			matchOpt := g.matchExpr(
				f.expr,
				g.arm(g.ctorPat(djSome, someVar), g.consExpr(g.kvExpr(f.name, encInner), g.ident(restVar))),
				g.arm(g.ident(djNone), g.ident(restVar)),
			)
			acc = g.letExpr(restVar, acc, matchOpt)
			continue
		}
		ev, err := g.encodeValue(ownerType, f.typ, f.expr)
		if err != nil {
			return nil, err
		}
		acc = g.consExpr(g.kvExpr(f.name, ev), acc)
	}
	return acc, nil
}

func (g *deriveJsonGen) encodeADT(ownerType string, adt *ast.AlgebraicType, valExpr ast.Expr) (ast.Expr, error) {
	cases := make([]*ast.Case, len(adt.Constructors))
	for i, ctor := range adt.Constructors {
		tagKV := g.kvExpr("tag", g.ctorCall(djJString, g.strLit(ctor.Name)))
		if len(ctor.Fields) == 0 {
			cases[i] = g.arm(g.ident(ctor.Name), g.ctorCall(djJObject, g.listLit(tagKV)))
			continue
		}

		varNames := make([]string, len(ctor.Fields))
		for j := range ctor.Fields {
			varNames[j] = g.fresh("cf")
		}
		ctorPat := g.ctorPat(ctor.Name, varNames...)

		if allNamedFields(ctor.Fields) {
			nv := make([]namedValueField, len(ctor.Fields))
			for j, cf := range ctor.Fields {
				nv[j] = namedValueField{name: cf.Name, typ: cf.Type, expr: g.ident(varNames[j])}
			}
			restKVs, err := g.encodeFieldKVs(ownerType, nv, g.listLit())
			if err != nil {
				return nil, err
			}
			cases[i] = g.arm(ctorPat, g.ctorCall(djJObject, g.consExpr(tagKV, restKVs)))
			continue
		}

		encodedFields := make([]ast.Expr, len(ctor.Fields))
		for j, cf := range ctor.Fields {
			ev, err := g.encodeValue(ownerType, cf.Type, g.ident(varNames[j]))
			if err != nil {
				return nil, err
			}
			encodedFields[j] = ev
		}
		fieldsKV := g.kvExpr("fields", g.ctorCall(djJArray, g.listLit(encodedFields...)))
		cases[i] = g.arm(ctorPat, g.ctorCall(djJObject, g.listLit(tagKV, fieldsKV)))
	}
	return g.matchExpr(valExpr, cases...), nil
}

func (g *deriveJsonGen) encodeValue(ownerType string, t ast.Type, valExpr ast.Expr) (ast.Expr, error) {
	if t == nil {
		return nil, g.unsupportedTypeErr(ownerType, nil)
	}
	if lt, ok := t.(*ast.LabelledType); ok {
		return g.encodeValue(ownerType, lt.Base, valExpr)
	}

	switch typ := t.(type) {
	case *ast.SimpleType:
		switch typ.Name {
		case "int":
			return g.ctorCall(djJInt, valExpr), nil
		case "float":
			return g.ctorCall(djJNumber, valExpr), nil
		case "string":
			return g.ctorCall(djJString, valExpr), nil
		case "bool":
			return g.ctorCall(djJBool, valExpr), nil
		case "()", "unit":
			return g.ctorCall(djJNull), nil
		case "bytes", "char":
			return nil, g.unsupportedTypeErr(ownerType, t)
		}
		if g.isStdJsonType(typ.Name) {
			return valExpr, nil
		}
		if g.elab.simpleTypeIsVar(typ.Name) {
			return nil, fmt.Errorf("TC_DERIVE_JSON_POLY at %s: cannot derive Json for type '%s' with type variable '%s'", sitePos(g.file, g.pos), ownerType, typ.Name)
		}
		return g.callUserCodec("encode", typ.Name, valExpr), nil

	case *ast.TypeVar:
		return nil, fmt.Errorf("TC_DERIVE_JSON_POLY at %s: cannot derive Json for type '%s' with type variable '%s'", sitePos(g.file, g.pos), ownerType, typ.Name)

	case *ast.ListType:
		return g.encodeList(ownerType, typ.Element, valExpr)

	case *ast.ArrayType:
		return g.encodeList(ownerType, typ.Element, g.callBuiltin("_array_to_list", valExpr))

	case *ast.TupleType:
		pats := make([]ast.Pattern, len(typ.Elements))
		elems := make([]ast.Expr, len(typ.Elements))
		for i, et := range typ.Elements {
			v := g.fresh("tup")
			pats[i] = g.ident(v)
			ev, err := g.encodeValue(ownerType, et, g.ident(v))
			if err != nil {
				return nil, err
			}
			elems[i] = ev
		}
		return g.matchExpr(valExpr, g.arm(&ast.TuplePattern{Elements: pats, Pos: g.pos}, g.ctorCall(djJArray, g.listLit(elems...)))), nil

	case *ast.RecordType:
		varName := g.fresh("rec")
		recBody, err := g.encodeRecord(ownerType, typ, g.ident(varName))
		if err != nil {
			return nil, err
		}
		return g.letExpr(varName, valExpr, recBody), nil

	case *ast.TypeApp:
		if typ.Constructor == "list" && len(typ.Args) == 1 {
			return g.encodeList(ownerType, typ.Args[0], valExpr)
		}
		if optInner, isOpt := g.unwrapOptionType(typ); isOpt {
			someVar := g.fresh("opt")
			encInner, err := g.encodeValue(ownerType, optInner, g.ident(someVar))
			if err != nil {
				return nil, err
			}
			return g.matchExpr(valExpr, g.arm(g.ident(djNone), g.ctorCall(djJNull)), g.arm(g.ctorPat(djSome, someVar), encInner)), nil
		}
		return nil, g.unsupportedTypeErr(ownerType, t)

	default:
		return nil, g.unsupportedTypeErr(ownerType, t)
	}
}

func (g *deriveJsonGen) encodeList(ownerType string, elemType ast.Type, listExpr ast.Expr) (ast.Expr, error) {
	elVar := g.fresh("el")
	encElem, err := g.encodeValue(ownerType, elemType, g.ident(elVar))
	if err != nil {
		return nil, err
	}
	lam := &ast.Lambda{Params: []*ast.Param{{Name: elVar, Pos: g.pos}}, Body: encElem, Pos: g.pos}
	return g.ctorCall(djJArray, g.callBuiltin("_list_map", lam, listExpr)), nil
}

func (g *deriveJsonGen) callUserCodec(prefix, typeName string, arg ast.Expr) ast.Expr {
	if dot := strings.Index(typeName, "."); dot >= 0 {
		return g.call(&ast.RecordAccess{Record: g.ident(typeName[:dot]), Field: prefix + typeName[dot+1:], Pos: g.pos}, arg)
	}
	return g.call(g.ident(prefix+typeName), arg)
}
