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
	registerXmlParseElements()
	registerXmlParseWithLimit()
	registerXmlGetText()
	registerXmlGetAttr()
	registerXmlGetChildren()
	registerXmlGetTag()
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
	return xmlMakeOk(makeXmlElement("", nil, children)), nil
}

// ============================================================================
// _xml_parseElements: string -> string -> int -> Result[List[XmlNode], string]
// Streaming element extraction — never builds full tree
// ============================================================================

func registerXmlParseElements() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/xml",
		Name:    "_xml_parseElements",
		NumArgs: 3,
		IsPure:  true,
		Type:    makeXmlParseElementsType,
		Impl:    xmlParseElementsImpl,
		Metadata: &BuiltinMetadata{
			Description: "Stream-parse XML extracting up to N elements matching a tag",
			LongDesc:    "Scans the XML token stream without building a full tree. Only builds subtrees for elements matching the given tag name, stopping after maxResults matches. Memory usage is O(largest matched element) instead of O(entire document).",
			Params: []ParamDoc{
				{Name: "xml", Description: "XML string to parse"},
				{Name: "tagName", Description: "Tag name to extract"},
				{Name: "maxResults", Description: "Maximum number of elements to extract"},
			},
			Returns: "Result[List[XmlNode], string] - Ok(list of matched elements) or Err(message)",
			Examples: []Example{
				{Code: `_xml_parseElements(sheetXml, "row", 10000)`, Description: "Extract first 10000 <row> elements from large spreadsheet XML"},
			},
			SeeAlso:   []string{"_xml_parse", "_xml_findAll", "_xml_parseWithLimit"},
			Since:     "v0.9.3",
			Stability: StabilityStable,
			Tags:      []string{"xml", "parsing", "streaming", "performance"},
			Category:  "xml",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _xml_parseElements: %v", err))
	}
}

func makeXmlParseElementsType() types.Type {
	T := types.NewBuilder()
	xmlNodeType := T.Con("XmlNode")
	return T.Func(T.String(), T.String(), T.Int()).Returns(
		T.App("Result", T.List(xmlNodeType), T.String()),
	).Build()
}

func xmlParseElementsImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	strVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_xml_parseElements: expected String for xml, got %T", args[0])
	}
	tagVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_xml_parseElements: expected String for tagName, got %T", args[1])
	}
	limitVal, ok := args[2].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_xml_parseElements: expected Int for maxResults, got %T", args[2])
	}

	input := strVal.Value
	if len(input) > xmlMaxInputSize {
		return xmlMakeErr(fmt.Sprintf("XML input too large: %d bytes (max %d)", len(input), xmlMaxInputSize)), nil
	}

	tagName := tagVal.Value
	limit := int(limitVal.Value)
	if limit <= 0 {
		return xmlMakeOk(&eval.ListValue{Elements: nil}), nil
	}

	decoder := xml.NewDecoder(strings.NewReader(input))
	var results []eval.Value
	scanForElements(decoder, tagName, limit, &results)

	return xmlMakeOk(&eval.ListValue{Elements: results}), nil
}

// scanForElements walks the XML token stream, building subtrees only for matching
// elements and descending (without allocation) into non-matching elements.
// Returns true if the limit was reached.
func scanForElements(decoder *xml.Decoder, tagName string, limit int, results *[]eval.Value) bool {
	for {
		tok, err := decoder.Token()
		if err != nil {
			return false // EOF or error — stop scanning
		}

		switch t := tok.(type) {
		case xml.StartElement:
			resolvedName := resolveTagName(t.Name, nil)
			if resolvedName == tagName {
				// Build subtree for this matched element
				localPM := extractPrefixMap(t, nil)
				attrs := buildAttrs(t, localPM)
				childNodes, err := parseXmlChildren(decoder, 1, localPM)
				if err != nil {
					return false
				}
				finalTag := resolveTagName(t.Name, localPM)
				*results = append(*results, makeXmlElement(finalTag, attrs, childNodes))
				if len(*results) >= limit {
					return true
				}
			} else {
				// Descend into non-matching element (no allocation, just keep scanning)
				if scanForElements(decoder, tagName, limit, results) {
					return true
				}
			}
		case xml.EndElement:
			return false // End of current element scope
		}
		// CharData, Comment, etc. — skip without allocation
	}
}

// extractPrefixMap builds a prefixMap from a StartElement's xmlns attributes
func extractPrefixMap(start xml.StartElement, parent *prefixMap) *prefixMap {
	pm := &prefixMap{parent: parent, entries: make(map[string]string)}
	for _, a := range start.Attr {
		if a.Name.Space == "xmlns" {
			pm.entries[a.Name.Local] = a.Value
		} else if a.Name.Local == "xmlns" && a.Name.Space == "" {
			pm.entries[""] = a.Value
		}
	}
	return pm
}

// buildAttrs extracts non-xmlns attributes from a StartElement
func buildAttrs(start xml.StartElement, pm *prefixMap) []eval.Value {
	attrs := make([]eval.Value, 0, len(start.Attr))
	for _, a := range start.Attr {
		attrName := resolveTagName(a.Name, pm)
		if attrName == "xmlns" || strings.HasPrefix(attrName, "xmlns:") {
			continue
		}
		attrs = append(attrs, makeXmlAttr(attrName, a.Value))
	}
	return attrs
}

// ============================================================================
// _xml_parseWithLimit: string -> int -> Result[XmlNode, string]
// Full parse with node count limit — fail fast on huge documents
// ============================================================================

func registerXmlParseWithLimit() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/xml",
		Name:    "_xml_parseWithLimit",
		NumArgs: 2,
		IsPure:  true,
		Type:    makeXmlParseWithLimitType,
		Impl:    xmlParseWithLimitImpl,
		Metadata: &BuiltinMetadata{
			Description: "Parse XML with a node count limit to prevent OOM on large documents",
			LongDesc:    "Like parse() but tracks the total number of nodes created. Returns Err if the count exceeds maxNodes, preventing memory exhaustion on pathologically large XML documents.",
			Params: []ParamDoc{
				{Name: "xml", Description: "XML string to parse"},
				{Name: "maxNodes", Description: "Maximum number of nodes to allow before failing"},
			},
			Returns: "Result[XmlNode, string] - Ok(XmlNode tree) or Err(message)",
			Examples: []Example{
				{Code: `_xml_parseWithLimit(xml, 100000)`, Description: "Parse XML, fail if tree exceeds 100K nodes"},
			},
			SeeAlso:   []string{"_xml_parse", "_xml_parseElements"},
			Since:     "v0.9.3",
			Stability: StabilityStable,
			Tags:      []string{"xml", "parsing", "safety", "limit"},
			Category:  "xml",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _xml_parseWithLimit: %v", err))
	}
}

func makeXmlParseWithLimitType() types.Type {
	T := types.NewBuilder()
	xmlNodeType := T.Con("XmlNode")
	return T.Func(T.String(), T.Int()).Returns(
		T.App("Result", xmlNodeType, T.String()),
	).Build()
}

func xmlParseWithLimitImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	strVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_xml_parseWithLimit: expected String, got %T", args[0])
	}
	limitVal, ok := args[1].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_xml_parseWithLimit: expected Int for maxNodes, got %T", args[1])
	}

	input := strVal.Value
	if len(input) > xmlMaxInputSize {
		return xmlMakeErr(fmt.Sprintf("XML input too large: %d bytes (max %d)", len(input), xmlMaxInputSize)), nil
	}

	maxNodes := int(limitVal.Value)
	nodeCount := 0

	decoder := xml.NewDecoder(strings.NewReader(input))
	children, err := parseXmlChildrenLimited(decoder, 0, nil, &nodeCount, maxNodes)
	if err != nil {
		return xmlMakeErr(fmt.Sprintf("XML parse error: %v", err)), nil
	}

	if len(children) == 0 {
		return xmlMakeErr("XML parse error: empty document"), nil
	}
	if len(children) == 1 {
		return xmlMakeOk(children[0]), nil
	}
	return xmlMakeOk(makeXmlElement("", nil, children)), nil
}

func parseXmlChildrenLimited(decoder *xml.Decoder, depth int, pm *prefixMap, nodeCount *int, maxNodes int) ([]eval.Value, error) {
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
			*nodeCount++
			if *nodeCount > maxNodes {
				return nil, fmt.Errorf("node limit exceeded: %d nodes (max %d)", *nodeCount, maxNodes)
			}
			localPM := extractPrefixMap(t, pm)
			attrs := buildAttrs(t, localPM)
			childNodes, err := parseXmlChildrenLimited(decoder, depth+1, localPM, nodeCount, maxNodes)
			if err != nil {
				return nil, err
			}
			tagName := resolveTagName(t.Name, localPM)
			children = append(children, makeXmlElement(tagName, attrs, childNodes))

		case xml.EndElement:
			return children, nil

		case xml.CharData:
			if !isAllWhitespace(t) {
				*nodeCount++
				if *nodeCount > maxNodes {
					return nil, fmt.Errorf("node limit exceeded: %d nodes (max %d)", *nodeCount, maxNodes)
				}
				children = append(children, makeXmlText(string(t)))
			}

		case xml.Comment:
			*nodeCount++
			if *nodeCount > maxNodes {
				return nil, fmt.Errorf("node limit exceeded: %d nodes (max %d)", *nodeCount, maxNodes)
			}
			children = append(children, makeXmlComment(string(t)))

		case xml.Directive:
		case xml.ProcInst:
		}
	}
}

// ============================================================================
// Shared XML infrastructure
// ============================================================================

type prefixMap struct {
	parent  *prefixMap
	entries map[string]string // prefix → URI
}

func (pm *prefixMap) lookupPrefix(uri string) string {
	for p := pm; p != nil; p = p.parent {
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

func resolveTagName(name xml.Name, pm *prefixMap) string {
	if name.Space == "" {
		return name.Local
	}
	if pm != nil {
		if prefix := pm.lookupPrefix(name.Space); prefix != "" {
			return prefix + ":" + name.Local
		}
	}
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
			localPM := extractPrefixMap(t, pm)
			attrs := buildAttrs(t, localPM)
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
		case xml.ProcInst:
		}
	}
}

func isAllWhitespace(data []byte) bool {
	for _, b := range data {
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			return false
		}
	}
	return true
}

// ============================================================================
// Accessor builtins: getText, getAttr, getChildren, getTag
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
			LongDesc:    "Looks up an attribute by name in the Element's attribute list. Returns Some(value) if found, None otherwise.",
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
	if lv, ok := tv.Fields[2].(*eval.ListValue); ok {
		return lv, nil
	}
	return &eval.ListValue{Elements: []eval.Value{}}, nil
}

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
	if sv, ok := tv.Fields[0].(*eval.StringValue); ok {
		return sv, nil
	}
	return &eval.StringValue{Value: ""}, nil
}
