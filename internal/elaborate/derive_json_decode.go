package elaborate

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
)

func typesTypeRef(module, name string) types.TypeRef {
	return types.TypeRef{Module: module, Name: name}
}

func (g *deriveJsonGen) appendPath(base, suffix ast.Expr) ast.Expr {
	if bl, ok := base.(*ast.Literal); ok && bl.Kind == ast.StringLit {
		if sl, ok := suffix.(*ast.Literal); ok && sl.Kind == ast.StringLit {
			bs, _ := bl.Value.(string)
			ss, _ := sl.Value.(string)
			return g.strLit(bs + ss)
		}
	}
	return g.callBuiltin("concat_String", base, suffix)
}

func (g *deriveJsonGen) okExpr(v ast.Expr) ast.Expr  { return g.ctorCall(djOk, v) }
func (g *deriveJsonGen) errExpr(m ast.Expr) ast.Expr { return g.ctorCall(djErr, m) }
func (g *deriveJsonGen) errMsg(p ast.Expr, suffix string) ast.Expr {
	return g.errExpr(g.appendPath(p, g.strLit(suffix)))
}

// matchResult builds `match res { Err(e) => Err(e), Ok(okVar) => okBody }`.
func (g *deriveJsonGen) matchResult(res ast.Expr, okVar string, okBody ast.Expr) *ast.Match {
	errVar := g.fresh("err")
	return g.matchExpr(
		res,
		g.arm(g.ctorPat(djErr, errVar), g.errExpr(g.ident(errVar))),
		g.arm(g.ctorPat(djOk, okVar), okBody),
	)
}

func (g *deriveJsonGen) buildDecodeFunc(decl *ast.TypeDecl) (*ast.FuncDecl, error) {
	paramName := "j"
	rootPath := g.strLit(decl.Name)
	var body ast.Expr
	var err error

	switch def := decl.Definition.(type) {
	case *ast.RecordType:
		body, err = g.decodeRecord(decl.Name, def, g.ident(paramName), rootPath)
	case *ast.AlgebraicType:
		body, err = g.decodeADT(decl.Name, def, g.ident(paramName), rootPath)
	case *ast.TypeAlias:
		body, err = g.decodeValue(decl.Name, def.Target, g.ident(paramName), rootPath)
	default:
		return nil, fmt.Errorf("TC_DERIVE_JSON_UNSUPPORTED at %s: unsupported type definition %T", sitePos(g.file, decl.Pos), def)
	}
	if err != nil {
		return nil, err
	}

	retType := &ast.TypeApp{
		Constructor: "Result",
		Args:        []ast.Type{&ast.SimpleType{Name: decl.Name, Pos: g.pos}, &ast.SimpleType{Name: "string", Pos: g.pos}},
		Pos:         g.pos,
	}
	return &ast.FuncDecl{
		Name:       "decode" + decl.Name,
		Params:     []*ast.Param{{Name: paramName, Type: &ast.SimpleType{Name: g.jsonTypeName, Pos: g.pos}, Pos: g.pos}},
		ReturnType: retType,
		Body:       body,
		IsPure:     true,
		IsExport:   decl.Exported,
		Annotations: []*ast.Annotation{{
			Name: "allow_empty_ok",
			Args: []ast.Expr{g.strLit("derived Json decoder legitimately decodes empty records and collections")},
			Pos:  g.pos,
		}},
		Pos:    g.pos,
		Origin: "derived_json_decode",
	}, nil
}

type namedTypeField struct {
	name string
	typ  ast.Type
}

func (g *deriveJsonGen) findKeyInKVs(kvsVar, key string) ast.Expr {
	kvParam := g.fresh("kv")
	pred := &ast.Lambda{
		Params: []*ast.Param{{Name: kvParam, Pos: g.pos}},
		Body:   g.callBuiltin("_str_eq", &ast.RecordAccess{Record: g.ident(kvParam), Field: "key", Pos: g.pos}, g.strLit(key)),
		Pos:    g.pos,
	}
	return g.callBuiltin("_list_findIndex", pred, g.ident(kvsVar))
}

func (g *deriveJsonGen) kvValueAt(kvsVar, idxVar string) ast.Expr {
	return &ast.RecordAccess{
		Record: g.callBuiltin("_list_nth", g.ident(kvsVar), g.ident(idxVar)),
		Field:  "value",
		Pos:    g.pos,
	}
}

func (g *deriveJsonGen) decodeRecord(ownerType string, rec *ast.RecordType, jsonExpr, pathExpr ast.Expr) (ast.Expr, error) {
	if rec.Row != nil {
		return nil, fmt.Errorf("TC_DERIVE_JSON_POLY at %s: cannot derive Json for row-polymorphic record in '%s'", sitePos(g.file, g.pos), ownerType)
	}
	fields := make([]namedTypeField, len(rec.Fields))
	for i, rf := range rec.Fields {
		fields[i] = namedTypeField{name: rf.Name, typ: rf.Type}
	}
	kvsVar := g.fresh("kvs")
	inner, err := g.decodeNamedFieldsChain(ownerType, kvsVar, fields, pathExpr, func(vals []ast.Expr) ast.Expr {
		recFields := make([]*ast.Field, len(fields))
		for i, f := range fields {
			recFields[i] = &ast.Field{Name: f.name, Value: vals[i], Pos: g.pos}
		}
		return g.okExpr(&ast.Record{Fields: recFields, Pos: g.pos})
	})
	if err != nil {
		return nil, err
	}
	return g.matchExpr(
		jsonExpr,
		g.arm(g.ctorPat(djJObject, kvsVar), inner),
		g.arm(g.wildPat(), g.errMsg(pathExpr, ": expected object")),
	), nil
}

func (g *deriveJsonGen) decodeNamedFieldsChain(
	ownerType, kvsVar string,
	fields []namedTypeField,
	pathExpr ast.Expr,
	finish func([]ast.Expr) ast.Expr,
) (ast.Expr, error) {
	fieldVars := make([]string, len(fields))
	valExprs := make([]ast.Expr, len(fields))
	for i := range fields {
		v := g.fresh("fv")
		fieldVars[i], valExprs[i] = v, g.ident(v)
	}

	next := finish(valExprs)
	for i := len(fields) - 1; i >= 0; i-- {
		f := fields[i]
		fPath := g.appendPath(pathExpr, g.strLit("."+f.name))
		fieldRes, err := g.decodeSingleField(ownerType, kvsVar, f, fPath)
		if err != nil {
			return nil, err
		}
		next = g.matchResult(fieldRes, fieldVars[i], next)
	}
	return next, nil
}

func (g *deriveJsonGen) decodeSingleField(ownerType, kvsVar string, f namedTypeField, fPath ast.Expr) (ast.Expr, error) {
	idxVar, rawVar := g.fresh("fidx"), g.fresh("fraw")
	if optInner, isOpt := g.unwrapOptionType(f.typ); isOpt {
		innerVal := g.fresh("optv")
		decInner, err := g.decodeValue(ownerType, optInner, g.ident(rawVar), fPath)
		if err != nil {
			return nil, err
		}
		nonNull := g.matchResult(decInner, innerVal, g.okExpr(g.ctorCall(djSome, g.ident(innerVal))))
		checkNull := g.matchExpr(
			g.ident(rawVar),
			g.arm(g.ident(djJNull), g.okExpr(g.ctorCall(djNone))),
			g.arm(g.wildPat(), nonNull),
		)
		return g.matchExpr(
			g.findKeyInKVs(kvsVar, f.name),
			g.arm(g.ident(djNone), g.okExpr(g.ctorCall(djNone))),
			g.arm(g.ctorPat(djSome, idxVar), g.letExpr(rawVar, g.kvValueAt(kvsVar, idxVar), checkNull)),
		), nil
	}

	decField, err := g.decodeValue(ownerType, f.typ, g.ident(rawVar), fPath)
	if err != nil {
		return nil, err
	}
	return g.matchExpr(
		g.findKeyInKVs(kvsVar, f.name),
		g.arm(g.ident(djNone), g.errMsg(fPath, ": missing required field")),
		g.arm(g.ctorPat(djSome, idxVar), g.letExpr(rawVar, g.kvValueAt(kvsVar, idxVar), decField)),
	), nil
}

func (g *deriveJsonGen) decodeADT(ownerType string, adt *ast.AlgebraicType, jsonExpr, pathExpr ast.Expr) (ast.Expr, error) {
	kvsVar, tagIdxVar, tagVar := g.fresh("kvs"), g.fresh("tidx"), g.fresh("tag")
	var dispatch ast.Expr = g.errExpr(
		g.callBuiltin("concat_String", g.appendPath(pathExpr, g.strLit(": unknown variant tag: ")), g.ident(tagVar)),
	)

	for i := len(adt.Constructors) - 1; i >= 0; i-- {
		ctor := adt.Constructors[i]
		ctorBody, err := g.decodeADTCtor(ownerType, ctor, kvsVar, pathExpr)
		if err != nil {
			return nil, err
		}
		dispatch = g.ifExpr(g.callBuiltin("_str_eq", g.ident(tagVar), g.strLit(ctor.Name)), ctorBody, dispatch)
	}

	tagMatch := g.matchExpr(
		g.kvValueAt(kvsVar, tagIdxVar),
		g.arm(g.ctorPat(djJString, tagVar), dispatch),
		g.arm(g.wildPat(), g.errMsg(pathExpr, ".tag: expected string")),
	)
	findTag := g.matchExpr(
		g.findKeyInKVs(kvsVar, "tag"),
		g.arm(g.ident(djNone), g.errMsg(pathExpr, ": missing 'tag' field")),
		g.arm(g.ctorPat(djSome, tagIdxVar), tagMatch),
	)
	return g.matchExpr(
		jsonExpr,
		g.arm(g.ctorPat(djJObject, kvsVar), findTag),
		g.arm(g.wildPat(), g.errMsg(pathExpr, ": expected object")),
	), nil
}

func (g *deriveJsonGen) decodeADTCtor(ownerType string, ctor *ast.Constructor, kvsVar string, pathExpr ast.Expr) (ast.Expr, error) {
	ctorPath := g.appendPath(pathExpr, g.strLit("."+ctor.Name))
	if len(ctor.Fields) == 0 {
		fIdxVar, arrVar := g.fresh("fidx"), g.fresh("farr")
		ctorVal := g.okExpr(g.ctorCall(ctor.Name))
		emptyCheck := g.ifExpr(
			g.binOp(g.callBuiltin("_list_length", g.ident(arrVar)), "==", g.intLit(0)),
			ctorVal,
			g.errMsg(ctorPath, ": expected 0 fields"),
		)
		fieldsVal := g.matchExpr(
			g.kvValueAt(kvsVar, fIdxVar),
			g.arm(g.ident(djJNull), ctorVal),
			g.arm(g.ctorPat(djJArray, arrVar), emptyCheck),
			g.arm(g.wildPat(), g.errMsg(ctorPath, ".fields: expected array")),
		)
		return g.matchExpr(
			g.findKeyInKVs(kvsVar, "fields"),
			g.arm(g.ident(djNone), ctorVal),
			g.arm(g.ctorPat(djSome, fIdxVar), fieldsVal),
		), nil
	}

	if allNamedFields(ctor.Fields) {
		nf := make([]namedTypeField, len(ctor.Fields))
		for i, cf := range ctor.Fields {
			nf[i] = namedTypeField{name: cf.Name, typ: cf.Type}
		}
		return g.decodeNamedFieldsChain(ownerType, kvsVar, nf, ctorPath, func(vals []ast.Expr) ast.Expr {
			return g.okExpr(g.ctorCall(ctor.Name, vals...))
		})
	}

	fIdxVar, arrVar := g.fresh("fidx"), g.fresh("farr")
	typesList := make([]ast.Type, len(ctor.Fields))
	for i, cf := range ctor.Fields {
		typesList[i] = cf.Type
	}
	posChain, err := g.decodeFixedArrayElems(ownerType, arrVar, typesList, ctorPath, func(vals []ast.Expr) ast.Expr {
		return g.okExpr(g.ctorCall(ctor.Name, vals...))
	})
	if err != nil {
		return nil, err
	}
	arityCheck := g.ifExpr(
		g.binOp(g.callBuiltin("_list_length", g.ident(arrVar)), "==", g.intLit(int64(len(ctor.Fields)))),
		posChain,
		g.errMsg(ctorPath, fmt.Sprintf(": expected %d field(s)", len(ctor.Fields))),
	)
	fieldsMatch := g.matchExpr(
		g.kvValueAt(kvsVar, fIdxVar),
		g.arm(g.ctorPat(djJArray, arrVar), arityCheck),
		g.arm(g.wildPat(), g.errMsg(ctorPath, ".fields: expected array")),
	)
	return g.matchExpr(
		g.findKeyInKVs(kvsVar, "fields"),
		g.arm(g.ident(djNone), g.errMsg(ctorPath, ": missing 'fields' array")),
		g.arm(g.ctorPat(djSome, fIdxVar), fieldsMatch),
	), nil
}

func (g *deriveJsonGen) decodeFixedArrayElems(
	ownerType, arrVar string,
	elemTypes []ast.Type,
	basePath ast.Expr,
	finish func([]ast.Expr) ast.Expr,
) (ast.Expr, error) {
	vars := make([]string, len(elemTypes))
	exprs := make([]ast.Expr, len(elemTypes))
	for i := range elemTypes {
		v := g.fresh("pv")
		vars[i], exprs[i] = v, g.ident(v)
	}
	next := finish(exprs)
	for i := len(elemTypes) - 1; i >= 0; i-- {
		elExpr := g.callBuiltin("_list_nth", g.ident(arrVar), g.intLit(int64(i)))
		elPath := g.appendPath(basePath, g.strLit(fmt.Sprintf("[%d]", i)))
		decEl, err := g.decodeValue(ownerType, elemTypes[i], elExpr, elPath)
		if err != nil {
			return nil, err
		}
		next = g.matchResult(decEl, vars[i], next)
	}
	return next, nil
}

func (g *deriveJsonGen) decodeValue(ownerType string, t ast.Type, jsonExpr, pathExpr ast.Expr) (ast.Expr, error) {
	if t == nil {
		return nil, g.unsupportedTypeErr(ownerType, nil)
	}
	if lt, ok := t.(*ast.LabelledType); ok {
		return g.decodeValue(ownerType, lt.Base, jsonExpr, pathExpr)
	}

	switch typ := t.(type) {
	case *ast.SimpleType:
		switch typ.Name {
		case "int":
			return g.decodeInt(jsonExpr, pathExpr), nil
		case "float":
			return g.decodeFloat(jsonExpr, pathExpr), nil
		case "string":
			return g.decodeSimpleScalar(djJString, "string", jsonExpr, pathExpr), nil
		case "bool":
			return g.decodeSimpleScalar(djJBool, "bool", jsonExpr, pathExpr), nil
		case "()", "unit":
			return g.matchExpr(
				jsonExpr,
				g.arm(g.ident(djJNull), g.okExpr(&ast.Literal{Kind: ast.UnitLit, Value: "()", Pos: g.pos})),
				g.arm(g.wildPat(), g.errMsg(pathExpr, ": expected null")),
			), nil
		case "bytes", "char":
			return nil, g.unsupportedTypeErr(ownerType, t)
		}
		if g.isStdJsonType(typ.Name) {
			return g.okExpr(jsonExpr), nil
		}
		if g.elab.simpleTypeIsVar(typ.Name) {
			return nil, fmt.Errorf("TC_DERIVE_JSON_POLY at %s: cannot derive Json for type '%s' with type variable '%s'", sitePos(g.file, g.pos), ownerType, typ.Name)
		}
		uVar, eVar := g.fresh("u"), g.fresh("uerr")
		return g.matchExpr(
			g.callUserCodec("decode", typ.Name, jsonExpr),
			g.arm(g.ctorPat(djOk, uVar), g.okExpr(g.ident(uVar))),
			g.arm(g.ctorPat(djErr, eVar), g.errExpr(g.callBuiltin("concat_String", g.appendPath(pathExpr, g.strLit(": ")), g.ident(eVar)))),
		), nil

	case *ast.TypeVar:
		return nil, fmt.Errorf("TC_DERIVE_JSON_POLY at %s: cannot derive Json for type '%s' with type variable '%s'", sitePos(g.file, g.pos), ownerType, typ.Name)

	case *ast.ListType:
		return g.decodeList(ownerType, typ.Element, jsonExpr, pathExpr)

	case *ast.ArrayType:
		listRes, err := g.decodeList(ownerType, typ.Element, jsonExpr, pathExpr)
		if err != nil {
			return nil, err
		}
		xsVar := g.fresh("arrxs")
		return g.matchResult(listRes, xsVar, g.okExpr(g.callBuiltin("_array_from_list", g.ident(xsVar)))), nil

	case *ast.TupleType:
		arrVar := g.fresh("tarr")
		chain, err := g.decodeFixedArrayElems(ownerType, arrVar, typ.Elements, pathExpr, func(vals []ast.Expr) ast.Expr {
			return g.okExpr(&ast.Tuple{Elements: vals, Pos: g.pos})
		})
		if err != nil {
			return nil, err
		}
		arity := g.ifExpr(
			g.binOp(g.callBuiltin("_list_length", g.ident(arrVar)), "==", g.intLit(int64(len(typ.Elements)))),
			chain,
			g.errMsg(pathExpr, fmt.Sprintf(": expected %d-element array", len(typ.Elements))),
		)
		return g.matchExpr(
			jsonExpr,
			g.arm(g.ctorPat(djJArray, arrVar), arity),
			g.arm(g.wildPat(), g.errMsg(pathExpr, ": expected array")),
		), nil

	case *ast.RecordType:
		return g.decodeRecord(ownerType, typ, jsonExpr, pathExpr)

	case *ast.TypeApp:
		if typ.Constructor == "list" && len(typ.Args) == 1 {
			return g.decodeList(ownerType, typ.Args[0], jsonExpr, pathExpr)
		}
		if optInner, isOpt := g.unwrapOptionType(typ); isOpt {
			rawVar, innerVar := g.fresh("oj"), g.fresh("ov")
			decInner, err := g.decodeValue(ownerType, optInner, g.ident(rawVar), pathExpr)
			if err != nil {
				return nil, err
			}
			body := g.matchExpr(
				g.ident(rawVar),
				g.arm(g.ident(djJNull), g.okExpr(g.ctorCall(djNone))),
				g.arm(g.wildPat(), g.matchResult(decInner, innerVar, g.okExpr(g.ctorCall(djSome, g.ident(innerVar))))),
			)
			return g.letExpr(rawVar, jsonExpr, body), nil
		}
		return nil, g.unsupportedTypeErr(ownerType, t)

	default:
		return nil, g.unsupportedTypeErr(ownerType, t)
	}
}

func (g *deriveJsonGen) decodeSimpleScalar(ctorName, typeLabel string, jsonExpr, pathExpr ast.Expr) ast.Expr {
	v := g.fresh("sc")
	return g.matchExpr(
		jsonExpr,
		g.arm(g.ctorPat(ctorName, v), g.okExpr(g.ident(v))),
		g.arm(g.wildPat(), g.errMsg(pathExpr, ": expected "+typeLabel)),
	)
}

func (g *deriveJsonGen) decodeInt(jsonExpr, pathExpr ast.Expr) ast.Expr {
	nVar, fVar := g.fresh("i"), g.fresh("f")
	errBody := g.errMsg(pathExpr, ": expected int")
	minBound := g.binOp(g.ident(fVar), ">=", &ast.UnaryOp{Op: "-", Expr: g.floatLit(9223372036854775808.0), Pos: g.pos})
	maxBound := g.binOp(g.ident(fVar), "<", g.floatLit(9223372036854775808.0))
	asInt := g.callBuiltin("_float_to_int", g.ident(fVar))
	exact := g.binOp(g.callBuiltin("_int_to_float", asInt), "==", g.ident(fVar))
	cond := g.binOp(g.binOp(minBound, "&&", maxBound), "&&", exact)

	return g.matchExpr(
		jsonExpr,
		g.arm(g.ctorPat(djJInt, nVar), g.okExpr(g.ident(nVar))),
		g.arm(g.ctorPat(djJNumber, fVar), g.ifExpr(cond, g.okExpr(g.callBuiltin("_float_to_int", g.ident(fVar))), errBody)),
		g.arm(g.wildPat(), errBody),
	)
}

func (g *deriveJsonGen) decodeFloat(jsonExpr, pathExpr ast.Expr) ast.Expr {
	fVar, iVar := g.fresh("f"), g.fresh("i")
	return g.matchExpr(
		jsonExpr,
		g.arm(g.ctorPat(djJNumber, fVar), g.okExpr(g.ident(fVar))),
		g.arm(g.ctorPat(djJInt, iVar), g.okExpr(g.callBuiltin("_int_to_float", g.ident(iVar)))),
		g.arm(g.wildPat(), g.errMsg(pathExpr, ": expected float")),
	)
}

func (g *deriveJsonGen) decodeList(ownerType string, elemType ast.Type, jsonExpr, pathExpr ast.Expr) (ast.Expr, error) {
	xsVar, stParam, elParam := g.fresh("xs"), g.fresh("st"), g.fresh("el")
	resVar, idxVar, accVar := g.fresh("res"), g.fresh("idx"), g.fresh("acc")
	errVar, itemVar, itemErrVar, epathVar := g.fresh("lerr"), g.fresh("item"), g.fresh("ierr"), g.fresh("epath")

	idxSuffix := g.callBuiltin(
		"concat_String",
		g.strLit("["),
		g.callBuiltin("concat_String", g.callBuiltin("_string_intToStr", g.ident(idxVar)), g.strLit("]")),
	)
	decElem, err := g.decodeValue(ownerType, elemType, g.ident(elParam), g.ident(epathVar))
	if err != nil {
		return nil, err
	}

	nextIdx := g.binOp(g.ident(idxVar), "+", g.intLit(1))
	pair := func(a, b ast.Expr) *ast.Tuple { return &ast.Tuple{Elements: []ast.Expr{a, b}, Pos: g.pos} }

	okStep := g.letExpr(
		epathVar,
		g.appendPath(pathExpr, idxSuffix),
		g.matchExpr(
			decElem,
			g.arm(g.ctorPat(djOk, itemVar), pair(g.okExpr(g.consExpr(g.ident(itemVar), g.ident(accVar))), nextIdx)),
			g.arm(g.ctorPat(djErr, itemErrVar), pair(g.errExpr(g.ident(itemErrVar)), nextIdx)),
		),
	)
	stepResMatch := g.matchExpr(
		g.ident(resVar),
		g.arm(g.ctorPat(djErr, errVar), pair(g.errExpr(g.ident(errVar)), g.ident(idxVar))),
		g.arm(g.ctorPat(djOk, accVar), okStep),
	)
	stepBody := g.matchExpr(
		g.ident(stParam),
		g.arm(&ast.TuplePattern{Elements: []ast.Pattern{g.ident(resVar), g.ident(idxVar)}, Pos: g.pos}, stepResMatch),
	)

	stepLam := &ast.Lambda{
		Params: []*ast.Param{{Name: stParam, Pos: g.pos}, {Name: elParam, Pos: g.pos}},
		Body:   stepBody,
		Pos:    g.pos,
	}
	folded := g.callBuiltin("_list_foldl", stepLam, pair(g.okExpr(g.listLit()), g.intLit(0)), g.ident(xsVar))

	fResVar, revVar := g.fresh("fres"), g.fresh("rev")
	finalize := g.matchExpr(
		folded,
		g.arm(
			&ast.TuplePattern{Elements: []ast.Pattern{g.ident(fResVar), g.wildPat()}, Pos: g.pos},
			g.matchResult(g.ident(fResVar), revVar, g.okExpr(g.callBuiltin("_list_reverse", g.ident(revVar)))),
		),
	)
	return g.matchExpr(
		jsonExpr,
		g.arm(g.ctorPat(djJArray, xsVar), finalize),
		g.arm(g.wildPat(), g.errMsg(pathExpr, ": expected array")),
	), nil
}
