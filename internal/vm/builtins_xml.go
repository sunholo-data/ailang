package vm

// VM-native XML builtins: constructors, parsers, serializers, queries.
//
// Strategy: call existing evaluator-side Go implementations (internal/builtins/xml*.go)
// then convert between eval.Value (TaggedValue) and bytecode.Value (ADTObj).
//
// XmlNode ADT tag ordering (must match std/xml.ail declaration order):
//
//   export type XmlNode =
//     | Element(string, [{name: string, value: string}], [XmlNode]) → tag 0
//     | Text(string)                                                 → tag 1
//     | Comment(string)                                              → tag 2
//
// Option ADT (std/option.ail): Some=0, None=1.
// Result ADT (std/result.ail): Ok=0, Err=1 (reused from builtins_json.go).

import (
	"fmt"

	"github.com/sunholo/ailang/internal/builtins"
	"github.com/sunholo/ailang/internal/bytecode"
	"github.com/sunholo/ailang/internal/eval"
)

// XmlNode ADT variant tags.
const (
	xmlNodeTagElement = 0
	xmlNodeTagText    = 1
	xmlNodeTagComment = 2
)

// Option ADT variant tags reused from builtins_string.go:
// optionTagSome = 0, optionTagNone = 1

// callEvalBuiltin calls an evaluator-side builtin by name via the spec registry.
// All XML builtins are pure (IsPure=true), so nil EffContext is safe.
func callEvalBuiltin(name string, args []eval.Value) (eval.Value, error) {
	spec, ok := builtins.GetSpec(name)
	if !ok {
		return nil, fmt.Errorf("callEvalBuiltin: unknown builtin %q", name)
	}
	return spec.Impl(nil, args)
}

// ---------------------------------------------------------------------------
// XmlNode ↔ bytecode.Value converters
// ---------------------------------------------------------------------------

// xmlNodeToBytecode converts an evaluator XmlNode (TaggedValue) to a
// bytecode ADT value. Recursively converts Element children.
func xmlNodeToBytecode(v eval.Value) (bytecode.Value, error) {
	tv, ok := v.(*eval.TaggedValue)
	if !ok {
		return bytecode.Value{}, fmt.Errorf("xmlNodeToBytecode: expected TaggedValue, got %T", v)
	}
	switch tv.CtorName {
	case "Element":
		if len(tv.Fields) < 3 {
			return bytecode.Value{}, fmt.Errorf("xmlNodeToBytecode: Element needs 3 fields, got %d", len(tv.Fields))
		}
		tag, ok := tv.Fields[0].(*eval.StringValue)
		if !ok {
			return bytecode.Value{}, fmt.Errorf("xmlNodeToBytecode: Element tag must be string")
		}
		attrList, ok := tv.Fields[1].(*eval.ListValue)
		if !ok {
			return bytecode.Value{}, fmt.Errorf("xmlNodeToBytecode: Element attrs must be list")
		}
		bcAttrs := make([]bytecode.Value, len(attrList.Elements))
		for i, attr := range attrList.Elements {
			rec, ok := attr.(*eval.RecordValue)
			if !ok {
				return bytecode.Value{}, fmt.Errorf("xmlNodeToBytecode: attr must be record, got %T", attr)
			}
			nameVal, _ := rec.Fields["name"].(*eval.StringValue)
			valVal, _ := rec.Fields["value"].(*eval.StringValue)
			n, v := "", ""
			if nameVal != nil {
				n = nameVal.Value
			}
			if valVal != nil {
				v = valVal.Value
			}
			bcAttrs[i] = bytecode.NewRecord([]bytecode.RecordField{
				{Name: "name", Value: bytecode.NewString(n)},
				{Name: "value", Value: bytecode.NewString(v)},
			})
		}
		childList, ok := tv.Fields[2].(*eval.ListValue)
		if !ok {
			return bytecode.Value{}, fmt.Errorf("xmlNodeToBytecode: Element children must be list")
		}
		bcChildren := make([]bytecode.Value, len(childList.Elements))
		for i, child := range childList.Elements {
			bc, err := xmlNodeToBytecode(child)
			if err != nil {
				return bytecode.Value{}, err
			}
			bcChildren[i] = bc
		}
		return bytecode.NewADT(xmlNodeTagElement, []bytecode.Value{
			bytecode.NewString(tag.Value),
			bytecode.NewList(bcAttrs),
			bytecode.NewList(bcChildren),
		}), nil

	case "Text":
		if len(tv.Fields) < 1 {
			return bytecode.Value{}, fmt.Errorf("xmlNodeToBytecode: Text needs 1 field")
		}
		text, ok := tv.Fields[0].(*eval.StringValue)
		if !ok {
			return bytecode.Value{}, fmt.Errorf("xmlNodeToBytecode: Text content must be string")
		}
		return bytecode.NewADT(xmlNodeTagText, []bytecode.Value{
			bytecode.NewString(text.Value),
		}), nil

	case "Comment":
		if len(tv.Fields) < 1 {
			return bytecode.Value{}, fmt.Errorf("xmlNodeToBytecode: Comment needs 1 field")
		}
		text, ok := tv.Fields[0].(*eval.StringValue)
		if !ok {
			return bytecode.Value{}, fmt.Errorf("xmlNodeToBytecode: Comment content must be string")
		}
		return bytecode.NewADT(xmlNodeTagComment, []bytecode.Value{
			bytecode.NewString(text.Value),
		}), nil

	default:
		return bytecode.Value{}, fmt.Errorf("xmlNodeToBytecode: unknown XmlNode ctor %q", tv.CtorName)
	}
}

// bytecodeToXmlNode converts a bytecode ADT (XmlNode) back to an evaluator
// TaggedValue so existing Go XML implementations can operate on it.
func bytecodeToXmlNode(v bytecode.Value) (eval.Value, error) {
	if v.Tag != bytecode.TagADT {
		return nil, fmt.Errorf("bytecodeToXmlNode: expected ADT, got %s", v.Tag)
	}
	adt := v.AsADT()
	switch adt.Tag {
	case xmlNodeTagElement:
		if len(adt.Fields) < 3 {
			return nil, fmt.Errorf("bytecodeToXmlNode: Element needs 3 fields")
		}
		tag := adt.Fields[0].AsString()
		attrElems := adt.Fields[1].AsList()
		evalAttrs := make([]eval.Value, len(attrElems))
		for i, a := range attrElems {
			fields := a.AsRecord()
			var name, val string
			for _, f := range fields {
				switch f.Name {
				case "name":
					name = f.Value.AsString()
				case "value":
					val = f.Value.AsString()
				}
			}
			evalAttrs[i] = &eval.RecordValue{
				Fields: map[string]eval.Value{
					"name":  &eval.StringValue{Value: name},
					"value": &eval.StringValue{Value: val},
				},
			}
		}
		childElems := adt.Fields[2].AsList()
		evalChildren := make([]eval.Value, len(childElems))
		for i, c := range childElems {
			ec, err := bytecodeToXmlNode(c)
			if err != nil {
				return nil, err
			}
			evalChildren[i] = ec
		}
		return &eval.TaggedValue{
			ModulePath: "std/xml",
			TypeName:   "XmlNode",
			CtorName:   "Element",
			Fields: []eval.Value{
				&eval.StringValue{Value: tag},
				&eval.ListValue{Elements: evalAttrs},
				&eval.ListValue{Elements: evalChildren},
			},
		}, nil

	case xmlNodeTagText:
		if len(adt.Fields) < 1 {
			return nil, fmt.Errorf("bytecodeToXmlNode: Text needs 1 field")
		}
		return &eval.TaggedValue{
			ModulePath: "std/xml",
			TypeName:   "XmlNode",
			CtorName:   "Text",
			Fields:     []eval.Value{&eval.StringValue{Value: adt.Fields[0].AsString()}},
		}, nil

	case xmlNodeTagComment:
		if len(adt.Fields) < 1 {
			return nil, fmt.Errorf("bytecodeToXmlNode: Comment needs 1 field")
		}
		return &eval.TaggedValue{
			ModulePath: "std/xml",
			TypeName:   "XmlNode",
			CtorName:   "Comment",
			Fields:     []eval.Value{&eval.StringValue{Value: adt.Fields[0].AsString()}},
		}, nil

	default:
		return nil, fmt.Errorf("bytecodeToXmlNode: unknown XmlNode tag %d", adt.Tag)
	}
}

// evalResultToBytecode converts a Result[_, string] from the evaluator
// to a bytecode Result ADT, recursively converting the Ok payload.
func evalResultToBytecode(v eval.Value) (bytecode.Value, error) {
	tv, ok := v.(*eval.TaggedValue)
	if !ok {
		return bytecode.Value{}, fmt.Errorf("evalResultToBytecode: expected TaggedValue, got %T", v)
	}
	switch tv.CtorName {
	case "Ok":
		payload, err := evalPayloadToBytecode(tv.Fields[0])
		if err != nil {
			return bytecode.Value{}, err
		}
		return vmResultOk(payload), nil
	case "Err":
		msg := tv.Fields[0].(*eval.StringValue).Value
		return vmResultErr(msg), nil
	default:
		return bytecode.Value{}, fmt.Errorf("evalResultToBytecode: unknown ctor %q", tv.CtorName)
	}
}

// evalPayloadToBytecode converts an eval.Value that may be an XmlNode or a
// list of XmlNodes to bytecode.
func evalPayloadToBytecode(v eval.Value) (bytecode.Value, error) {
	switch val := v.(type) {
	case *eval.TaggedValue:
		return xmlNodeToBytecode(val)
	case *eval.ListValue:
		elems := make([]bytecode.Value, len(val.Elements))
		for i, e := range val.Elements {
			bc, err := evalPayloadToBytecode(e)
			if err != nil {
				return bytecode.Value{}, err
			}
			elems[i] = bc
		}
		return bytecode.NewList(elems), nil
	case *eval.StringValue:
		return bytecode.NewString(val.Value), nil
	default:
		return bytecode.Value{}, fmt.Errorf("evalPayloadToBytecode: unexpected type %T", v)
	}
}

// ---------------------------------------------------------------------------
// M1: Constructor builtins (3)
// ---------------------------------------------------------------------------

func builtinXmlElement(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 3 {
		return bytecode.Value{}, fmt.Errorf("__xmlElement: expected 3 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__xmlElement: arg 0 must be string")
	}
	if args[1].Tag != bytecode.TagList {
		return bytecode.Value{}, fmt.Errorf("__xmlElement: arg 1 must be list")
	}
	if args[2].Tag != bytecode.TagList {
		return bytecode.Value{}, fmt.Errorf("__xmlElement: arg 2 must be list")
	}
	return bytecode.NewADT(xmlNodeTagElement, []bytecode.Value{
		args[0], args[1], args[2],
	}), nil
}

func builtinXmlText(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__xmlText: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__xmlText: arg must be string")
	}
	return bytecode.NewADT(xmlNodeTagText, []bytecode.Value{args[0]}), nil
}

func builtinXmlComment(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__xmlComment: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__xmlComment: arg must be string")
	}
	return bytecode.NewADT(xmlNodeTagComment, []bytecode.Value{args[0]}), nil
}

// ---------------------------------------------------------------------------
// M2: String-returning builtins (4)
// ---------------------------------------------------------------------------

func builtinXmlGetText(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__xml_getText: expected 1 arg, got %d", len(args))
	}
	node, err := bytecodeToXmlNode(args[0])
	if err != nil {
		return bytecode.Value{}, err
	}
	result, err := callEvalBuiltin("_xml_getText", []eval.Value{node})
	if err != nil {
		return bytecode.Value{}, err
	}
	return bytecode.NewString(result.(*eval.StringValue).Value), nil
}

func builtinXmlGetTag(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__xml_getTag: expected 1 arg, got %d", len(args))
	}
	node, err := bytecodeToXmlNode(args[0])
	if err != nil {
		return bytecode.Value{}, err
	}
	result, err := callEvalBuiltin("_xml_getTag", []eval.Value{node})
	if err != nil {
		return bytecode.Value{}, err
	}
	return bytecode.NewString(result.(*eval.StringValue).Value), nil
}

func builtinXmlSerialize(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__xml_serialize: expected 1 arg, got %d", len(args))
	}
	node, err := bytecodeToXmlNode(args[0])
	if err != nil {
		return bytecode.Value{}, err
	}
	result, err := callEvalBuiltin("_xml_serialize", []eval.Value{node})
	if err != nil {
		return bytecode.Value{}, err
	}
	return bytecode.NewString(result.(*eval.StringValue).Value), nil
}

func builtinXmlSerializeWithDecl(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__xml_serializeWithDecl: expected 1 arg, got %d", len(args))
	}
	node, err := bytecodeToXmlNode(args[0])
	if err != nil {
		return bytecode.Value{}, err
	}
	result, err := callEvalBuiltin("_xml_serializeWithDecl", []eval.Value{node})
	if err != nil {
		return bytecode.Value{}, err
	}
	return bytecode.NewString(result.(*eval.StringValue).Value), nil
}

// ---------------------------------------------------------------------------
// M3: Parse builtins (3)
// ---------------------------------------------------------------------------

func builtinXmlParse(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__xml_parse: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__xml_parse: arg must be string")
	}
	result, err := callEvalBuiltin("_xml_parse", []eval.Value{
		&eval.StringValue{Value: args[0].AsString()},
	})
	if err != nil {
		return bytecode.Value{}, err
	}
	return evalResultToBytecode(result)
}

func builtinXmlParseElements(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 3 {
		return bytecode.Value{}, fmt.Errorf("__xml_parseElements: expected 3 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__xml_parseElements: arg 0 must be string")
	}
	if args[1].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__xml_parseElements: arg 1 must be string")
	}
	if args[2].Tag != bytecode.TagInt {
		return bytecode.Value{}, fmt.Errorf("__xml_parseElements: arg 2 must be int")
	}
	result, err := callEvalBuiltin("_xml_parseElements", []eval.Value{
		&eval.StringValue{Value: args[0].AsString()},
		&eval.StringValue{Value: args[1].AsString()},
		&eval.IntValue{Value: int(args[2].Int)},
	})
	if err != nil {
		return bytecode.Value{}, err
	}
	return evalResultToBytecode(result)
}

func builtinXmlParseWithLimit(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("__xml_parseWithLimit: expected 2 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__xml_parseWithLimit: arg 0 must be string")
	}
	if args[1].Tag != bytecode.TagInt {
		return bytecode.Value{}, fmt.Errorf("__xml_parseWithLimit: arg 1 must be int")
	}
	result, err := callEvalBuiltin("_xml_parseWithLimit", []eval.Value{
		&eval.StringValue{Value: args[0].AsString()},
		&eval.IntValue{Value: int(args[1].Int)},
	})
	if err != nil {
		return bytecode.Value{}, err
	}
	return evalResultToBytecode(result)
}

// ---------------------------------------------------------------------------
// M4: Query builtins (6)
// ---------------------------------------------------------------------------

func builtinXmlFindAll(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("__xml_findAll: expected 2 args, got %d", len(args))
	}
	node, err := bytecodeToXmlNode(args[0])
	if err != nil {
		return bytecode.Value{}, err
	}
	if args[1].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__xml_findAll: arg 1 must be string")
	}
	result, err := callEvalBuiltin("_xml_findAll", []eval.Value{
		node,
		&eval.StringValue{Value: args[1].AsString()},
	})
	if err != nil {
		return bytecode.Value{}, err
	}
	return evalListXmlNodesToBytecode(result)
}

func builtinXmlFindFirst(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("__xml_findFirst: expected 2 args, got %d", len(args))
	}
	node, err := bytecodeToXmlNode(args[0])
	if err != nil {
		return bytecode.Value{}, err
	}
	if args[1].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__xml_findFirst: arg 1 must be string")
	}
	result, err := callEvalBuiltin("_xml_findFirst", []eval.Value{
		node,
		&eval.StringValue{Value: args[1].AsString()},
	})
	if err != nil {
		return bytecode.Value{}, err
	}
	return evalOptionXmlNodeToBytecode(result)
}

func builtinXmlGetAttr(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("__xml_getAttr: expected 2 args, got %d", len(args))
	}
	node, err := bytecodeToXmlNode(args[0])
	if err != nil {
		return bytecode.Value{}, err
	}
	if args[1].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__xml_getAttr: arg 1 must be string")
	}
	result, err := callEvalBuiltin("_xml_getAttr", []eval.Value{
		node,
		&eval.StringValue{Value: args[1].AsString()},
	})
	if err != nil {
		return bytecode.Value{}, err
	}
	return evalOptionStringToBytecode(result)
}

func builtinXmlGetChildren(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__xml_getChildren: expected 1 arg, got %d", len(args))
	}
	node, err := bytecodeToXmlNode(args[0])
	if err != nil {
		return bytecode.Value{}, err
	}
	result, err := callEvalBuiltin("_xml_getChildren", []eval.Value{node})
	if err != nil {
		return bytecode.Value{}, err
	}
	return evalListXmlNodesToBytecode(result)
}

func builtinXmlFindAllTexts(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("__xml_findAllTexts: expected 2 args, got %d", len(args))
	}
	node, err := bytecodeToXmlNode(args[0])
	if err != nil {
		return bytecode.Value{}, err
	}
	if args[1].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__xml_findAllTexts: arg 1 must be string")
	}
	result, err := callEvalBuiltin("_xml_findAllTexts", []eval.Value{
		node,
		&eval.StringValue{Value: args[1].AsString()},
	})
	if err != nil {
		return bytecode.Value{}, err
	}
	return evalListStringsToBytecode(result)
}

func builtinXmlFindAllAttrs(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 3 {
		return bytecode.Value{}, fmt.Errorf("__xml_findAllAttrs: expected 3 args, got %d", len(args))
	}
	node, err := bytecodeToXmlNode(args[0])
	if err != nil {
		return bytecode.Value{}, err
	}
	if args[1].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__xml_findAllAttrs: arg 1 must be string")
	}
	if args[2].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__xml_findAllAttrs: arg 2 must be string")
	}
	result, err := callEvalBuiltin("_xml_findAllAttrs", []eval.Value{
		node,
		&eval.StringValue{Value: args[1].AsString()},
		&eval.StringValue{Value: args[2].AsString()},
	})
	if err != nil {
		return bytecode.Value{}, err
	}
	return evalListStringsToBytecode(result)
}

// ---------------------------------------------------------------------------
// M5: parseFold HOF builtin
// ---------------------------------------------------------------------------

func hofBuiltinXmlParseFold(caller ClosureCaller, args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 4 {
		return bytecode.Value{}, fmt.Errorf("__xml_parseFold: expected 4 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__xml_parseFold: arg 0 must be string")
	}
	if args[1].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__xml_parseFold: arg 1 must be string")
	}
	xmlStr := args[0].AsString()
	tagName := args[1].AsString()
	acc := args[2]
	fn := args[3]

	// Use the evaluator's parseElements to get matching nodes, then fold
	// with the VM closure.
	result, err := callEvalBuiltin("_xml_parseElements", []eval.Value{
		&eval.StringValue{Value: xmlStr},
		&eval.StringValue{Value: tagName},
		&eval.IntValue{Value: 10000000},
	})
	if err != nil {
		return bytecode.Value{}, err
	}

	tv := result.(*eval.TaggedValue)
	if tv.CtorName == "Err" {
		msg := tv.Fields[0].(*eval.StringValue).Value
		return vmResultErr(msg), nil
	}

	nodeList := tv.Fields[0].(*eval.ListValue)
	for _, node := range nodeList.Elements {
		bcNode, err := xmlNodeToBytecode(node)
		if err != nil {
			return bytecode.Value{}, err
		}
		acc, err = caller.CallClosure(fn, []bytecode.Value{acc, bcNode})
		if err != nil {
			return bytecode.Value{}, err
		}
	}
	return vmResultOk(acc), nil
}

// ---------------------------------------------------------------------------
// Eval → bytecode result converters
// ---------------------------------------------------------------------------

func evalListXmlNodesToBytecode(v eval.Value) (bytecode.Value, error) {
	lv, ok := v.(*eval.ListValue)
	if !ok {
		return bytecode.Value{}, fmt.Errorf("evalListXmlNodesToBytecode: expected list, got %T", v)
	}
	elems := make([]bytecode.Value, len(lv.Elements))
	for i, e := range lv.Elements {
		bc, err := xmlNodeToBytecode(e)
		if err != nil {
			return bytecode.Value{}, err
		}
		elems[i] = bc
	}
	return bytecode.NewList(elems), nil
}

func evalOptionXmlNodeToBytecode(v eval.Value) (bytecode.Value, error) {
	tv, ok := v.(*eval.TaggedValue)
	if !ok {
		return bytecode.Value{}, fmt.Errorf("evalOptionXmlNodeToBytecode: expected TaggedValue, got %T", v)
	}
	switch tv.CtorName {
	case "Some":
		inner, err := xmlNodeToBytecode(tv.Fields[0])
		if err != nil {
			return bytecode.Value{}, err
		}
		return bytecode.NewADT(optionTagSome, []bytecode.Value{inner}), nil
	case "None":
		return bytecode.NewADT(optionTagNone, nil), nil
	default:
		return bytecode.Value{}, fmt.Errorf("evalOptionXmlNodeToBytecode: unknown ctor %q", tv.CtorName)
	}
}

func evalOptionStringToBytecode(v eval.Value) (bytecode.Value, error) {
	tv, ok := v.(*eval.TaggedValue)
	if !ok {
		return bytecode.Value{}, fmt.Errorf("evalOptionStringToBytecode: expected TaggedValue, got %T", v)
	}
	switch tv.CtorName {
	case "Some":
		s := tv.Fields[0].(*eval.StringValue).Value
		return bytecode.NewADT(optionTagSome, []bytecode.Value{bytecode.NewString(s)}), nil
	case "None":
		return bytecode.NewADT(optionTagNone, nil), nil
	default:
		return bytecode.Value{}, fmt.Errorf("evalOptionStringToBytecode: unknown ctor %q", tv.CtorName)
	}
}

func evalListStringsToBytecode(v eval.Value) (bytecode.Value, error) {
	lv, ok := v.(*eval.ListValue)
	if !ok {
		return bytecode.Value{}, fmt.Errorf("evalListStringsToBytecode: expected list, got %T", v)
	}
	elems := make([]bytecode.Value, len(lv.Elements))
	for i, e := range lv.Elements {
		sv, ok := e.(*eval.StringValue)
		if !ok {
			return bytecode.Value{}, fmt.Errorf("evalListStringsToBytecode: expected string, got %T", e)
		}
		elems[i] = bytecode.NewString(sv.Value)
	}
	return bytecode.NewList(elems), nil
}
