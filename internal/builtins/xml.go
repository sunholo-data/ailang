package builtins

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// XML parsing builtins for AILANG
// These provide read-only XML parsing and querying as pure functions
// Part of M-STDLIB-XML (v0.7.3)

const (
	xmlMaxDepth     = 256
	xmlMaxInputSize = 50 * 1024 * 1024 // 50MB
)

func init() {
	registerXmlParse()
	registerXmlFindAll()
	registerXmlFindFirst()
	registerXmlGetText()
	registerXmlGetAttr()
	registerXmlGetChildren()
	registerXmlGetTag()
	registerXmlFindAllTexts()
	registerXmlFindAllAttrs()
}

// ============================================================================
// XmlNode ADT constructors
// ============================================================================

// makeXmlElement creates Element(tag, attrs, children)
func makeXmlElement(tag string, attrs []eval.Value, children []eval.Value) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/xml",
		TypeName:   "XmlNode",
		CtorName:   "Element",
		Fields: []eval.Value{
			&eval.StringValue{Value: tag},
			&eval.ListValue{Elements: attrs},
			&eval.ListValue{Elements: children},
		},
	}
}

// makeXmlText creates Text(content)
func makeXmlText(content string) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/xml",
		TypeName:   "XmlNode",
		CtorName:   "Text",
		Fields:     []eval.Value{&eval.StringValue{Value: content}},
	}
}

// makeXmlComment creates Comment(content)
func makeXmlComment(content string) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/xml",
		TypeName:   "XmlNode",
		CtorName:   "Comment",
		Fields:     []eval.Value{&eval.StringValue{Value: content}},
	}
}

// makeXmlAttr creates {name: string, value: string} record
func makeXmlAttr(name, value string) eval.Value {
	return &eval.RecordValue{
		Fields: map[string]eval.Value{
			"name":  &eval.StringValue{Value: name},
			"value": &eval.StringValue{Value: value},
		},
	}
}

// Result helpers reusing the same pattern as zip.go
func xmlMakeOk(val eval.Value) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Ok",
		Fields:     []eval.Value{val},
	}
}

func xmlMakeErr(msg string) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Err",
		Fields:     []eval.Value{&eval.StringValue{Value: msg}},
	}
}

// Option helpers
func xmlMakeSome(val eval.Value) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/option",
		TypeName:   "Option",
		CtorName:   "Some",
		Fields:     []eval.Value{val},
	}
}

func xmlMakeNone() eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/option",
		TypeName:   "Option",
		CtorName:   "None",
		Fields:     []eval.Value{},
	}
}

// ============================================================================
// _xml_parse: string -> Result[XmlNode, string]
// ============================================================================

func registerXmlParse() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/xml",
		Name:    "_xml_parse",
		NumArgs: 1,
		IsPure:  true,
		Type:    makeXmlParseType,
		Impl:    xmlParseImpl,
		Metadata: &BuiltinMetadata{
			Description: "Parse an XML string into an XmlNode tree",
			LongDesc:    "Parses well-formed XML into an XmlNode algebraic data type tree. Supports elements, text, CDATA, and comments. Namespace prefixes are preserved in tag names (e.g., w:p). Rejects input >50MB and depth >256.",
			Params: []ParamDoc{
				{Name: "xml", Description: "XML string to parse"},
			},
			Returns: "Result[XmlNode, string] - Ok(XmlNode tree) or Err(error message)",
			Examples: []Example{
				{Code: `_xml_parse("<root><item>hello</item></root>")`, Description: `Returns Ok(Element("root", [], [Element("item", [], [Text("hello")])]))`},
				{Code: `_xml_parse("not xml")`, Description: `Returns Err("XML parse error: ...")`},
			},
			SeeAlso:   []string{"_xml_findAll", "_xml_findFirst", "_xml_getText", "_xml_getAttr"},
			Since:     "v0.7.3",
			Stability: StabilityStable,
			Tags:      []string{"xml", "parsing", "tree", "adt"},
			Category:  "xml",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _xml_parse: %v", err))
	}
}

func makeXmlParseType() types.Type {
	T := types.NewBuilder()
	xmlNodeType := T.Con("XmlNode")
	return T.Func(T.String()).Returns(
		T.App("Result", xmlNodeType, T.String()),
	).Build()
}

func xmlParseImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	strVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_xml_parse: expected String, got %T", args[0])
	}

	input := strVal.Value
	if len(input) > xmlMaxInputSize {
		return xmlMakeErr(fmt.Sprintf("XML input too large: %d bytes (max %d)", len(input), xmlMaxInputSize)), nil
	}

	decoder := xml.NewDecoder(strings.NewReader(input))
	// Build prefix map from xmlns attributes at each level
	children, err := parseXmlChildren(decoder, 0, nil)
	if err != nil {
		return xmlMakeErr(fmt.Sprintf("XML parse error: %v", err)), nil
	}

	if len(children) == 0 {
		return xmlMakeErr("XML parse error: empty document"), nil
	}
	if len(children) == 1 {
		return xmlMakeOk(children[0]), nil
	}
	// Multiple top-level nodes: wrap in synthetic root
	return xmlMakeOk(makeXmlElement("", nil, children)), nil
}

// prefixMap tracks xmlns prefix → URI mappings for reverse-mapping
type prefixMap struct {
	parent  *prefixMap
	entries map[string]string // prefix → URI
}

func (pm *prefixMap) lookupPrefix(uri string) string {
	for p := pm; p != nil; p = p.parent {
		// Check default namespace first (empty prefix) for determinism.
		// Go map iteration is random, so iterating directly could return
		// different prefixes across runs when multiple prefixes map to
		// the same URI (e.g. xmlns="..." and xmlns:opf="..." both point
		// to the same namespace). Always prefer the default namespace.
		if u, ok := p.entries[""]; ok && u == uri {
			return ""
		}
		for prefix, u := range p.entries {
			if prefix != "" && u == uri {
				return prefix
			}
		}
	}
	return ""
}

// resolveTagName converts Go xml.Name (which has URI) back to prefix:local form
func resolveTagName(name xml.Name, pm *prefixMap) string {
	if name.Space == "" {
		return name.Local
	}
	if pm != nil {
		if prefix := pm.lookupPrefix(name.Space); prefix != "" {
			return prefix + ":" + name.Local
		}
	}
	// Fallback: use local name only (drops namespace)
	return name.Local
}

func parseXmlChildren(decoder *xml.Decoder, depth int, pm *prefixMap) ([]eval.Value, error) {
	if depth > xmlMaxDepth {
		return nil, fmt.Errorf("maximum depth exceeded (%d)", xmlMaxDepth)
	}

	var children []eval.Value
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			return children, nil
		}
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			// Extract xmlns prefix mappings from attributes
			localPM := &prefixMap{parent: pm, entries: make(map[string]string)}
			attrs := make([]eval.Value, 0, len(t.Attr))
			for _, a := range t.Attr {
				if a.Name.Space == "xmlns" {
					// xmlns:prefix="URI" — record the mapping
					localPM.entries[a.Name.Local] = a.Value
				} else if a.Name.Local == "xmlns" && a.Name.Space == "" {
					// default namespace xmlns="URI"
					localPM.entries[""] = a.Value
				}
				// Build attribute with resolved name
				attrName := resolveTagName(a.Name, localPM)
				// Skip xmlns declarations from attribute list
				if attrName == "xmlns" || strings.HasPrefix(attrName, "xmlns:") {
					continue
				}
				attrs = append(attrs, makeXmlAttr(attrName, a.Value))
			}

			// Recursively parse children
			childNodes, err := parseXmlChildren(decoder, depth+1, localPM)
			if err != nil {
				return nil, err
			}

			tagName := resolveTagName(t.Name, localPM)
			children = append(children, makeXmlElement(tagName, attrs, childNodes))

		case xml.EndElement:
			return children, nil

		case xml.CharData:
			if !isAllWhitespace(t) {
				children = append(children, makeXmlText(string(t)))
			}

		case xml.Comment:
			children = append(children, makeXmlComment(string(t)))

		case xml.Directive:
			// Skip DTD directives

		case xml.ProcInst:
			// Skip processing instructions (<?xml ...?>)
		}
	}
}

// isAllWhitespace checks if a byte slice contains only whitespace,
// without allocating a new string (unlike strings.TrimSpace).
func isAllWhitespace(data []byte) bool {
	for _, b := range data {
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			return false
		}
	}
	return true
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
	// Fields[0] = tag name
	if sv, ok := tv.Fields[0].(*eval.StringValue); ok && sv.Value == tagName {
		*results = append(*results, node)
	}
	// Fields[2] = children
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
// _xml_getText: XmlNode -> string
// ============================================================================

func registerXmlGetText() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/xml",
		Name:    "_xml_getText",
		NumArgs: 1,
		IsPure:  true,
		Type:    makeXmlNodeToStringType,
		Impl:    xmlGetTextImpl,
		Metadata: &BuiltinMetadata{
			Description: "Get concatenated text content of a node and all descendants",
			LongDesc:    "Recursively collects all Text and CData content from the node and its descendants, concatenating them in document order.",
			Params: []ParamDoc{
				{Name: "node", Description: "XmlNode to extract text from"},
			},
			Returns:   "string - concatenated text content",
			Examples:  []Example{{Code: `_xml_getText(Element("p", [], [Text("hello "), Element("b", [], [Text("world")])]))`, Description: `Returns "hello world"`}},
			SeeAlso:   []string{"_xml_getTag", "_xml_getAttr"},
			Since:     "v0.7.3",
			Stability: StabilityStable,
			Tags:      []string{"xml", "query", "text"},
			Category:  "xml",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _xml_getText: %v", err))
	}
}

func makeXmlNodeToStringType() types.Type {
	T := types.NewBuilder()
	xmlNodeType := T.Con("XmlNode")
	return T.Func(xmlNodeType).Returns(T.String()).Build()
}

func xmlGetTextImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	var buf strings.Builder
	collectText(args[0], &buf)
	return &eval.StringValue{Value: buf.String()}, nil
}

func collectText(node eval.Value, buf *strings.Builder) {
	tv, ok := node.(*eval.TaggedValue)
	if !ok {
		return
	}
	switch tv.CtorName {
	case "Text", "CData":
		if sv, ok := tv.Fields[0].(*eval.StringValue); ok {
			buf.WriteString(sv.Value)
		}
	case "Element":
		if lv, ok := tv.Fields[2].(*eval.ListValue); ok {
			for _, child := range lv.Elements {
				collectText(child, buf)
			}
		}
	}
}

// ============================================================================
// _xml_getAttr: XmlNode -> string -> Option[string]
// ============================================================================

func registerXmlGetAttr() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/xml",
		Name:    "_xml_getAttr",
		NumArgs: 2,
		IsPure:  true,
		Type:    makeXmlGetAttrType,
		Impl:    xmlGetAttrImpl,
		Metadata: &BuiltinMetadata{
			Description: "Get an attribute value by name from an Element node",
			LongDesc:    "Looks up an attribute by name in the Element's attribute list. Returns Some(value) if found, None otherwise. Non-Element nodes always return None.",
			Params: []ParamDoc{
				{Name: "node", Description: "XmlNode to look up attribute on"},
				{Name: "attrName", Description: "Attribute name to find"},
			},
			Returns:   "Option[string] - Some(value) or None",
			Examples:  []Example{{Code: `_xml_getAttr(element, "class")`, Description: `Returns Some("main") if attribute exists`}},
			SeeAlso:   []string{"_xml_getText", "_xml_getTag"},
			Since:     "v0.7.3",
			Stability: StabilityStable,
			Tags:      []string{"xml", "query", "attribute"},
			Category:  "xml",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _xml_getAttr: %v", err))
	}
}

func makeXmlGetAttrType() types.Type {
	T := types.NewBuilder()
	xmlNodeType := T.Con("XmlNode")
	return T.Func(xmlNodeType, T.String()).Returns(T.App("Option", T.String())).Build()
}

func xmlGetAttrImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	tv, ok := args[0].(*eval.TaggedValue)
	if !ok || tv.CtorName != "Element" {
		return xmlMakeNone(), nil
	}

	attrName, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_xml_getAttr: expected String for attrName, got %T", args[1])
	}

	// Fields[1] = attributes list
	attrList, ok := tv.Fields[1].(*eval.ListValue)
	if !ok {
		return xmlMakeNone(), nil
	}

	for _, attr := range attrList.Elements {
		rec, ok := attr.(*eval.RecordValue)
		if !ok {
			continue
		}
		nameVal, ok := rec.Fields["name"].(*eval.StringValue)
		if !ok {
			continue
		}
		if nameVal.Value == attrName.Value {
			valVal, ok := rec.Fields["value"].(*eval.StringValue)
			if !ok {
				continue
			}
			return xmlMakeSome(&eval.StringValue{Value: valVal.Value}), nil
		}
	}

	return xmlMakeNone(), nil
}

// ============================================================================
// _xml_getChildren: XmlNode -> [XmlNode]
// ============================================================================

func registerXmlGetChildren() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/xml",
		Name:    "_xml_getChildren",
		NumArgs: 1,
		IsPure:  true,
		Type:    makeXmlNodeToListType,
		Impl:    xmlGetChildrenImpl,
		Metadata: &BuiltinMetadata{
			Description: "Get direct child nodes of an Element",
			LongDesc:    "Returns the list of direct child nodes of an Element. Non-Element nodes return an empty list.",
			Params: []ParamDoc{
				{Name: "node", Description: "XmlNode to get children from"},
			},
			Returns:   "[XmlNode] - list of direct child nodes",
			Examples:  []Example{{Code: `_xml_getChildren(root)`, Description: "Returns direct children of root element"}},
			SeeAlso:   []string{"_xml_findAll", "_xml_getTag"},
			Since:     "v0.7.3",
			Stability: StabilityStable,
			Tags:      []string{"xml", "query", "children"},
			Category:  "xml",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _xml_getChildren: %v", err))
	}
}

func makeXmlNodeToListType() types.Type {
	T := types.NewBuilder()
	xmlNodeType := T.Con("XmlNode")
	return T.Func(xmlNodeType).Returns(T.List(xmlNodeType)).Build()
}

func xmlGetChildrenImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	tv, ok := args[0].(*eval.TaggedValue)
	if !ok || tv.CtorName != "Element" {
		return &eval.ListValue{Elements: []eval.Value{}}, nil
	}
	// Fields[2] = children
	if lv, ok := tv.Fields[2].(*eval.ListValue); ok {
		return lv, nil
	}
	return &eval.ListValue{Elements: []eval.Value{}}, nil
}

// ============================================================================
// _xml_getTag: XmlNode -> string
// ============================================================================

func registerXmlGetTag() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/xml",
		Name:    "_xml_getTag",
		NumArgs: 1,
		IsPure:  true,
		Type:    makeXmlNodeToStringType,
		Impl:    xmlGetTagImpl,
		Metadata: &BuiltinMetadata{
			Description: "Get the tag name of an Element node",
			LongDesc:    "Returns the tag name of an Element node. For non-Element nodes (Text, CData, Comment), returns an empty string.",
			Params: []ParamDoc{
				{Name: "node", Description: "XmlNode to get tag name from"},
			},
			Returns:   "string - tag name or empty string",
			Examples:  []Example{{Code: `_xml_getTag(Element("div", [], []))`, Description: `Returns "div"`}},
			SeeAlso:   []string{"_xml_getText", "_xml_getAttr"},
			Since:     "v0.7.3",
			Stability: StabilityStable,
			Tags:      []string{"xml", "query", "tag"},
			Category:  "xml",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _xml_getTag: %v", err))
	}
}

func xmlGetTagImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	tv, ok := args[0].(*eval.TaggedValue)
	if !ok || tv.CtorName != "Element" {
		return &eval.StringValue{Value: ""}, nil
	}
	// Fields[0] = tag name
	if sv, ok := tv.Fields[0].(*eval.StringValue); ok {
		return sv, nil
	}
	return &eval.StringValue{Value: ""}, nil
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
