package builtins

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// XML query builtins — extracted from xml.go for file size compliance
// findAll, findFirst, findAllTexts, findAllAttrs

func init() {
	registerXmlFindAll()
	registerXmlFindFirst()
	registerXmlFindAllTexts()
	registerXmlFindAllAttrs()
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
