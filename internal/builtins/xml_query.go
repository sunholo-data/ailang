package builtins

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// XML query builtins — extracted from xml.go for file size compliance
// findAll, findFirst, findAllTexts, findAllAttrs

func init() {
	registerXmlFindAll()
	registerXmlFindFirst()
	registerXmlFindAllTexts()
	registerXmlFindAllAttrs()
	registerXmlFoldChildren()
	registerXmlFoldChildrenStep()
	registerXmlGetAttrMap()
	registerXmlNodeKind()
}

// ============================================================================
// _xml_findAll: XmlNode -> string -> [XmlNode]
// ============================================================================

func registerXmlFindAll() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/xml",
		Name:    "_xml_findAll",
		NumArgs: 2,
		IsPure:  true,
		Type:    makeXmlNodeStringToListType,
		Impl:    xmlFindAllImpl,
		Metadata: &BuiltinMetadata{
			Description: "Find all descendant elements matching a tag name",
			LongDesc:    "Performs a depth-first search of the XmlNode tree and returns all Element nodes whose tag name matches the given string.",
			Params: []ParamDoc{
				{Name: "node", Description: "Root XmlNode to search"},
				{Name: "tagName", Description: "Tag name to match"},
			},
			Returns:   "[XmlNode] - list of matching elements",
			Examples:  []Example{{Code: `_xml_findAll(root, "item")`, Description: "Returns all <item> elements"}},
			SeeAlso:   []string{"_xml_findFirst", "_xml_parse"},
			Since:     "v0.7.3",
			Stability: StabilityStable,
			Tags:      []string{"xml", "query", "search"},
			Category:  "xml",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _xml_findAll: %v", err))
	}
}

func makeXmlNodeStringToListType() types.Type {
	T := types.NewBuilder()
	xmlNodeType := T.Con("XmlNode")
	return T.Func(xmlNodeType, T.String()).Returns(T.List(xmlNodeType)).Build()
}

func xmlFindAllImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	node := args[0]
	tagVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_xml_findAll: expected String for tagName, got %T", args[1])
	}

	var results []eval.Value
	findAllRecursive(node, tagVal.Value, &results)
	return &eval.ListValue{Elements: results}, nil
}

func findAllRecursive(node eval.Value, tagName string, results *[]eval.Value) {
	tv, ok := node.(*eval.TaggedValue)
	if !ok || tv.CtorName != "Element" {
		return
	}
	if sv, ok := tv.Fields[0].(*eval.StringValue); ok && sv.Value == tagName {
		*results = append(*results, node)
	}
	if lv, ok := tv.Fields[2].(*eval.ListValue); ok {
		for _, child := range lv.Elements {
			findAllRecursive(child, tagName, results)
		}
	}
}

// ============================================================================
// _xml_findFirst: XmlNode -> string -> Option[XmlNode]
// ============================================================================

func registerXmlFindFirst() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/xml",
		Name:    "_xml_findFirst",
		NumArgs: 2,
		IsPure:  true,
		Type:    makeXmlNodeStringToOptionType,
		Impl:    xmlFindFirstImpl,
		Metadata: &BuiltinMetadata{
			Description: "Find first descendant element matching a tag name",
			LongDesc:    "Performs a depth-first search and returns the first matching Element as Some(node), or None if not found.",
			Params: []ParamDoc{
				{Name: "node", Description: "Root XmlNode to search"},
				{Name: "tagName", Description: "Tag name to match"},
			},
			Returns:   "Option[XmlNode] - Some(element) or None",
			Examples:  []Example{{Code: `_xml_findFirst(root, "title")`, Description: "Returns Some(Element(\"title\", ...)) or None"}},
			SeeAlso:   []string{"_xml_findAll", "_xml_parse"},
			Since:     "v0.7.3",
			Stability: StabilityStable,
			Tags:      []string{"xml", "query", "search"},
			Category:  "xml",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _xml_findFirst: %v", err))
	}
}

func makeXmlNodeStringToOptionType() types.Type {
	T := types.NewBuilder()
	xmlNodeType := T.Con("XmlNode")
	return T.Func(xmlNodeType, T.String()).Returns(T.App("Option", xmlNodeType)).Build()
}

func xmlFindFirstImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	node := args[0]
	tagVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_xml_findFirst: expected String for tagName, got %T", args[1])
	}

	if found := findFirstRecursive(node, tagVal.Value); found != nil {
		return xmlMakeSome(found), nil
	}
	return xmlMakeNone(), nil
}

func findFirstRecursive(node eval.Value, tagName string) eval.Value {
	tv, ok := node.(*eval.TaggedValue)
	if !ok || tv.CtorName != "Element" {
		return nil
	}
	if sv, ok := tv.Fields[0].(*eval.StringValue); ok && sv.Value == tagName {
		return node
	}
	if lv, ok := tv.Fields[2].(*eval.ListValue); ok {
		for _, child := range lv.Elements {
			if found := findFirstRecursive(child, tagName); found != nil {
				return found
			}
		}
	}
	return nil
}

// ============================================================================
// _xml_findAllTexts: XmlNode -> string -> [string]
// ============================================================================

func registerXmlFindAllTexts() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/xml",
		Name:    "_xml_findAllTexts",
		NumArgs: 2,
		IsPure:  true,
		Type:    makeXmlNodeStringToStringListType,
		Impl:    xmlFindAllTextsImpl,
		Metadata: &BuiltinMetadata{
			Description: "Find all matching elements and extract their text content",
			LongDesc:    "Combines findAll + getText in a single Go call, avoiding per-element interpreter overhead. Returns a list of text strings for all elements whose tag matches.",
			Params: []ParamDoc{
				{Name: "node", Description: "Root XmlNode to search"},
				{Name: "tagName", Description: "Tag name to match"},
			},
			Returns:   "[string] - text content of each matching element",
			Examples:  []Example{{Code: `_xml_findAllTexts(root, "p")`, Description: "Returns text of all <p> elements"}},
			SeeAlso:   []string{"_xml_findAll", "_xml_getText", "_xml_findAllAttrs"},
			Since:     "v0.9.2",
			Stability: StabilityStable,
			Tags:      []string{"xml", "query", "search", "bulk"},
			Category:  "xml",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _xml_findAllTexts: %v", err))
	}
}

func makeXmlNodeStringToStringListType() types.Type {
	T := types.NewBuilder()
	xmlNodeType := T.Con("XmlNode")
	return T.Func(xmlNodeType, T.String()).Returns(T.List(T.String())).Build()
}

func xmlFindAllTextsImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	node := args[0]
	tagVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_xml_findAllTexts: expected String for tagName, got %T", args[1])
	}

	var results []eval.Value
	findAllTextsRecursive(node, tagVal.Value, &results)
	return &eval.ListValue{Elements: results}, nil
}

func findAllTextsRecursive(node eval.Value, tagName string, results *[]eval.Value) {
	tv, ok := node.(*eval.TaggedValue)
	if !ok || tv.CtorName != "Element" {
		return
	}
	if sv, ok := tv.Fields[0].(*eval.StringValue); ok && sv.Value == tagName {
		var buf strings.Builder
		collectText(node, &buf)
		*results = append(*results, &eval.StringValue{Value: buf.String()})
	}
	if lv, ok := tv.Fields[2].(*eval.ListValue); ok {
		for _, child := range lv.Elements {
			findAllTextsRecursive(child, tagName, results)
		}
	}
}

// ============================================================================
// _xml_findAllAttrs: XmlNode -> string -> string -> [string]
// ============================================================================

func registerXmlFindAllAttrs() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/xml",
		Name:    "_xml_findAllAttrs",
		NumArgs: 3,
		IsPure:  true,
		Type:    makeXmlFindAllAttrsType,
		Impl:    xmlFindAllAttrsImpl,
		Metadata: &BuiltinMetadata{
			Description: "Find all matching elements and extract a named attribute",
			LongDesc:    "Combines findAll + getAttr in a single Go call. Returns a list of attribute values (as strings) for all elements whose tag matches. Elements missing the attribute are skipped.",
			Params: []ParamDoc{
				{Name: "node", Description: "Root XmlNode to search"},
				{Name: "tagName", Description: "Tag name to match"},
				{Name: "attrName", Description: "Attribute name to extract"},
			},
			Returns:   "[string] - attribute values from matching elements",
			Examples:  []Example{{Code: `_xml_findAllAttrs(root, "item", "href")`, Description: "Returns href attrs of all <item> elements"}},
			SeeAlso:   []string{"_xml_findAll", "_xml_getAttr", "_xml_findAllTexts"},
			Since:     "v0.9.2",
			Stability: StabilityStable,
			Tags:      []string{"xml", "query", "search", "bulk"},
			Category:  "xml",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _xml_findAllAttrs: %v", err))
	}
}

func makeXmlFindAllAttrsType() types.Type {
	T := types.NewBuilder()
	xmlNodeType := T.Con("XmlNode")
	return T.Func(xmlNodeType, T.String(), T.String()).Returns(T.List(T.String())).Build()
}

func xmlFindAllAttrsImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	node := args[0]
	tagVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_xml_findAllAttrs: expected String for tagName, got %T", args[1])
	}
	attrVal, ok := args[2].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_xml_findAllAttrs: expected String for attrName, got %T", args[2])
	}

	var results []eval.Value
	findAllAttrsRecursive(node, tagVal.Value, attrVal.Value, &results)
	return &eval.ListValue{Elements: results}, nil
}

func findAllAttrsRecursive(node eval.Value, tagName, attrName string, results *[]eval.Value) {
	tv, ok := node.(*eval.TaggedValue)
	if !ok || tv.CtorName != "Element" {
		return
	}
	if sv, ok := tv.Fields[0].(*eval.StringValue); ok && sv.Value == tagName {
		if attrList, ok := tv.Fields[1].(*eval.ListValue); ok {
			for _, attr := range attrList.Elements {
				rec, ok := attr.(*eval.RecordValue)
				if !ok {
					continue
				}
				nameVal, ok := rec.Fields["name"].(*eval.StringValue)
				if !ok || nameVal.Value != attrName {
					continue
				}
				if valVal, ok := rec.Fields["value"].(*eval.StringValue); ok {
					*results = append(*results, &eval.StringValue{Value: valVal.Value})
				}
			}
		}
	}
	if lv, ok := tv.Fields[2].(*eval.ListValue); ok {
		for _, child := range lv.Elements {
			findAllAttrsRecursive(child, tagName, attrName, results)
		}
	}
}

// ============================================================================
// _xml_foldChildren: XmlNode -> a -> ((a, XmlNode) -> a) -> a
// Fold over direct children without materializing the [XmlNode] list.
// ============================================================================

func registerXmlFoldChildren() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/xml",
		Name:    "_xml_foldChildren",
		NumArgs: 3,
		IsPure:  true,
		Type:    makeXmlFoldChildrenType,
		Impl:    xmlFoldChildrenImpl,
		Metadata: &BuiltinMetadata{
			Description: "Fold over direct children of an Element without allocating a [XmlNode] list",
			LongDesc:    "Walks the direct children of an Element in document order, threading an accumulator through a handler. Replaces foldl(f, init, getChildren(node)). Non-Element nodes return init unchanged. Memory: O(1) — no intermediate list is built.",
			Params: []ParamDoc{
				{Name: "node", Description: "Element to walk"},
				{Name: "init", Description: "Initial accumulator"},
				{Name: "handler", Description: "Fold function: (acc, XmlNode) -> acc"},
			},
			Returns:   "a - final accumulator after visiting every direct child",
			Examples:  []Example{{Code: `_xml_foldChildren(node, 0, \acc, _. acc + 1)`, Description: "Count direct children without allocating a list"}},
			SeeAlso:   []string{"_xml_foldChildrenStep", "_xml_getChildren", "_xml_parseFold"},
			Since:     "v0.21.0",
			Stability: StabilityStable,
			Tags:      []string{"xml", "fold", "walk", "performance"},
			Category:  "xml",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _xml_foldChildren: %v", err))
	}
}

// Type: forall a. (XmlNode, a, (a, XmlNode) -> a) -> a
func makeXmlFoldChildrenType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	xmlNodeType := T.Con("XmlNode")
	fn := T.Func(a, xmlNodeType).Returns(a).Build()
	return T.Func(xmlNodeType, a, fn).Returns(a).Build()
}

func xmlFoldChildrenImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if ctx == nil || ctx.FnCallerN == nil {
		return nil, fmt.Errorf("_xml_foldChildren: FnCallerN not set (evaluator not wired)")
	}
	tv, ok := args[0].(*eval.TaggedValue)
	if !ok || tv.CtorName != "Element" {
		return args[1], nil
	}
	children, ok := tv.Fields[2].(*eval.ListValue)
	if !ok {
		return args[1], nil
	}
	acc := args[1]
	handler := args[2]
	for _, child := range children.Elements {
		next, err := ctx.FnCallerN(handler, []eval.Value{acc, child})
		if err != nil {
			return nil, err
		}
		acc = next
	}
	return acc, nil
}

// ============================================================================
// _xml_foldChildrenStep: XmlNode -> a -> ((a, XmlNode) -> FoldStep[a]) -> a
// Bounded fold over direct children — handler returns Continue(a) | Stop(a).
// ============================================================================

func registerXmlFoldChildrenStep() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/xml",
		Name:    "_xml_foldChildrenStep",
		NumArgs: 3,
		IsPure:  true,
		Type:    makeXmlFoldChildrenStepType,
		Impl:    xmlFoldChildrenStepImpl,
		Metadata: &BuiltinMetadata{
			Description: "Bounded fold over direct children with FoldStep[a] early termination",
			LongDesc:    "Like _xml_foldChildren, but the handler returns FoldStep[a] = Continue(a) | Stop(a). Stop(acc') halts the walk immediately and returns acc'. Useful for short-circuit searches over direct children (e.g. first child matching a predicate).",
			Params: []ParamDoc{
				{Name: "node", Description: "Element to walk"},
				{Name: "init", Description: "Initial accumulator"},
				{Name: "handler", Description: "Fold function: (acc, XmlNode) -> FoldStep[acc]"},
			},
			Returns:   "a - final accumulator (after Stop or after visiting every child)",
			SeeAlso:   []string{"_xml_foldChildren", "_xml_parseFoldStep"},
			Since:     "v0.21.0",
			Stability: StabilityStable,
			Tags:      []string{"xml", "fold", "walk", "bounded"},
			Category:  "xml",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _xml_foldChildrenStep: %v", err))
	}
}

// Type: forall a. (XmlNode, a, (a, XmlNode) -> FoldStep[a]) -> a
func makeXmlFoldChildrenStepType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	xmlNodeType := T.Con("XmlNode")
	stepType := T.App("FoldStep", a)
	fn := T.Func(a, xmlNodeType).Returns(stepType).Build()
	return T.Func(xmlNodeType, a, fn).Returns(a).Build()
}

func xmlFoldChildrenStepImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if ctx == nil || ctx.FnCallerN == nil {
		return nil, fmt.Errorf("_xml_foldChildrenStep: FnCallerN not set (evaluator not wired)")
	}
	tv, ok := args[0].(*eval.TaggedValue)
	if !ok || tv.CtorName != "Element" {
		return args[1], nil
	}
	children, ok := tv.Fields[2].(*eval.ListValue)
	if !ok {
		return args[1], nil
	}
	acc := args[1]
	handler := args[2]
	for _, child := range children.Elements {
		stepVal, err := ctx.FnCallerN(handler, []eval.Value{acc, child})
		if err != nil {
			return nil, err
		}
		tagged, ok := stepVal.(*eval.TaggedValue)
		if !ok || len(tagged.Fields) != 1 {
			return nil, fmt.Errorf("_xml_foldChildrenStep: handler must return FoldStep[a] (Continue(acc)|Stop(acc)), got %T", stepVal)
		}
		acc = tagged.Fields[0]
		switch tagged.CtorName {
		case "Stop":
			return acc, nil
		case "Continue":
			// fall through to next child
		default:
			return nil, fmt.Errorf("_xml_foldChildrenStep: handler returned unknown FoldStep constructor %q (expected Continue or Stop)", tagged.CtorName)
		}
	}
	return acc, nil
}

// ============================================================================
// _xml_getAttrMap: XmlNode -> Map[string, string]
// Returns all attributes of an Element as a Map. Non-Element nodes return empty map.
// Duplicate attribute names: last write wins.
// ============================================================================

func registerXmlGetAttrMap() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/xml",
		Name:    "_xml_getAttrMap",
		NumArgs: 1,
		IsPure:  true,
		Type:    makeXmlGetAttrMapType,
		Impl:    xmlGetAttrMapImpl,
		Metadata: &BuiltinMetadata{
			Description: "Get all attributes of an Element as a Map[string, string]",
			LongDesc:    "Builds a Map keyed by attribute name in one FFI call. Replaces N separate _xml_getAttr calls when extracting multiple attributes from one node. Non-Element nodes return an empty map. Duplicate attribute names: last write wins (source order processed left-to-right).",
			Params: []ParamDoc{
				{Name: "node", Description: "XmlNode to read attributes from"},
			},
			Returns:   "Map[string, string] - keyed by attribute name",
			Examples:  []Example{{Code: `_xml_getAttrMap(img)`, Description: "All <img> attrs in one call instead of 7 separate getAttr calls"}},
			SeeAlso:   []string{"_xml_getAttr", "_xml_findAllAttrs"},
			Since:     "v0.21.0",
			Stability: StabilityStable,
			Tags:      []string{"xml", "attrs", "map", "performance"},
			Category:  "xml",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _xml_getAttrMap: %v", err))
	}
}

func makeXmlGetAttrMapType() types.Type {
	T := types.NewBuilder()
	xmlNodeType := T.Con("XmlNode")
	return T.Func(xmlNodeType).Returns(T.Map(T.String(), T.String())).Build()
}

func xmlGetAttrMapImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	empty := &eval.MapValue{Entries: make(map[string]*eval.MapEntry)}
	tv, ok := args[0].(*eval.TaggedValue)
	if !ok || tv.CtorName != "Element" {
		return empty, nil
	}
	attrList, ok := tv.Fields[1].(*eval.ListValue)
	if !ok {
		return empty, nil
	}
	entries := make(map[string]*eval.MapEntry, len(attrList.Elements))
	for _, attr := range attrList.Elements {
		rec, ok := attr.(*eval.RecordValue)
		if !ok {
			continue
		}
		nameVal, ok := rec.Fields["name"].(*eval.StringValue)
		if !ok {
			continue
		}
		valVal, ok := rec.Fields["value"].(*eval.StringValue)
		if !ok {
			continue
		}
		keyVal := &eval.StringValue{Value: nameVal.Value}
		canonKey, err := eval.MapKey(keyVal)
		if err != nil {
			continue
		}
		// Last write wins on duplicate names (overwrites prior entry).
		entries[canonKey] = &eval.MapEntry{Key: keyVal, Value: &eval.StringValue{Value: valVal.Value}}
	}
	return &eval.MapValue{Entries: entries}, nil
}

// ============================================================================
// _xml_nodeKind: XmlNode -> NodeKind
// Classify the node without string-equality on the tag.
// ============================================================================

func registerXmlNodeKind() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/xml",
		Name:    "_xml_nodeKind",
		NumArgs: 1,
		IsPure:  true,
		Type:    makeXmlNodeKindType,
		Impl:    xmlNodeKindImpl,
		Metadata: &BuiltinMetadata{
			Description: "Classify an XmlNode as Element | Text | Comment",
			LongDesc:    "Returns a NodeKind variant so callers can pattern-match instead of doing string-equality on the tag name (e.g. `getTag(n) == \"\"` to detect text). Faster (one FFI call returning a tagged value) and exhaustiveness-checkable.",
			Params: []ParamDoc{
				{Name: "node", Description: "XmlNode to classify"},
			},
			Returns:   "NodeKind - Element | Text | Comment",
			SeeAlso:   []string{"_xml_getTag", "_xml_getText"},
			Since:     "v0.21.0",
			Stability: StabilityStable,
			Tags:      []string{"xml", "classify", "variant"},
			Category:  "xml",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _xml_nodeKind: %v", err))
	}
}

func makeXmlNodeKindType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.Con("XmlNode")).Returns(T.Con("NodeKind")).Build()
}

func xmlNodeKindImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	tv, ok := args[0].(*eval.TaggedValue)
	if !ok {
		// Defensive: unknown value shape — classify as KindComment (a no-op variant for most walkers).
		return &eval.TaggedValue{TypeName: "NodeKind", CtorName: "KindComment", Fields: nil}, nil
	}
	switch tv.CtorName {
	case "Element":
		return &eval.TaggedValue{TypeName: "NodeKind", CtorName: "KindElement", Fields: nil}, nil
	case "Text":
		return &eval.TaggedValue{TypeName: "NodeKind", CtorName: "KindText", Fields: nil}, nil
	default:
		// KindComment (and anything else — defensive default since XmlNode is closed).
		return &eval.TaggedValue{TypeName: "NodeKind", CtorName: "KindComment", Fields: nil}, nil
	}
}
