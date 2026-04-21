package builtins

import (
	"fmt"
	"html"
	"strings"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// M-DOCPARSE-DX M4: XML serialization builtins

func init() {
	registerXmlSerialize()
	registerXmlSerializeWithDecl()
	registerEscapeXml()
	registerXmlElementCtor()
	registerXmlTextCtor()
	registerXmlCommentCtor()
}

// serializeNode recursively converts an XmlNode tree to XML string
func serializeNode(node eval.Value, buf *strings.Builder, depth int) error {
	if depth > xmlMaxDepth {
		return fmt.Errorf("serialize: maximum nesting depth exceeded (%d)", xmlMaxDepth)
	}

	tv, ok := node.(*eval.TaggedValue)
	if !ok {
		return fmt.Errorf("serialize: expected XmlNode, got %T", node)
	}

	switch tv.CtorName {
	case "Element":
		if len(tv.Fields) < 3 {
			return fmt.Errorf("serialize: Element requires 3 fields, got %d", len(tv.Fields))
		}
		tag, ok := tv.Fields[0].(*eval.StringValue)
		if !ok {
			return fmt.Errorf("serialize: Element tag must be string, got %T", tv.Fields[0])
		}
		attrs, ok := tv.Fields[1].(*eval.ListValue)
		if !ok {
			return fmt.Errorf("serialize: Element attrs must be list, got %T", tv.Fields[1])
		}
		children, ok := tv.Fields[2].(*eval.ListValue)
		if !ok {
			return fmt.Errorf("serialize: Element children must be list, got %T", tv.Fields[2])
		}

		buf.WriteString("<")
		buf.WriteString(html.EscapeString(tag.Value))

		// Write attributes
		for _, attr := range attrs.Elements {
			rec, ok := attr.(*eval.RecordValue)
			if !ok {
				return fmt.Errorf("serialize: attribute must be record, got %T", attr)
			}
			nameVal, ok := rec.Fields["name"].(*eval.StringValue)
			if !ok {
				continue
			}
			valVal, ok := rec.Fields["value"].(*eval.StringValue)
			if !ok {
				continue
			}
			buf.WriteString(" ")
			buf.WriteString(html.EscapeString(nameVal.Value))
			buf.WriteString("=\"")
			buf.WriteString(html.EscapeString(valVal.Value))
			buf.WriteString("\"")
		}

		if len(children.Elements) == 0 {
			buf.WriteString("/>")
		} else {
			buf.WriteString(">")
			for _, child := range children.Elements {
				if err := serializeNode(child, buf, depth+1); err != nil {
					return err
				}
			}
			buf.WriteString("</")
			buf.WriteString(html.EscapeString(tag.Value))
			buf.WriteString(">")
		}

	case "Text":
		if len(tv.Fields) < 1 {
			return fmt.Errorf("serialize: Text requires 1 field")
		}
		text, ok := tv.Fields[0].(*eval.StringValue)
		if !ok {
			return fmt.Errorf("serialize: Text content must be string, got %T", tv.Fields[0])
		}
		buf.WriteString(html.EscapeString(text.Value))

	case "CData":
		if len(tv.Fields) < 1 {
			return fmt.Errorf("serialize: CData requires 1 field")
		}
		text, ok := tv.Fields[0].(*eval.StringValue)
		if !ok {
			return fmt.Errorf("serialize: CData content must be string, got %T", tv.Fields[0])
		}
		buf.WriteString("<![CDATA[")
		buf.WriteString(text.Value)
		buf.WriteString("]]>")

	case "Comment":
		if len(tv.Fields) < 1 {
			return fmt.Errorf("serialize: Comment requires 1 field")
		}
		text, ok := tv.Fields[0].(*eval.StringValue)
		if !ok {
			return fmt.Errorf("serialize: Comment content must be string, got %T", tv.Fields[0])
		}
		buf.WriteString("<!--")
		buf.WriteString(text.Value)
		buf.WriteString("-->")

	default:
		return fmt.Errorf("serialize: unknown XmlNode constructor %q", tv.CtorName)
	}

	return nil
}

func registerXmlSerialize() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/xml",
		Name:    "_xml_serialize",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeXmlSerializeType,
		Impl:    xmlSerializeImpl,
		Metadata: &BuiltinMetadata{
			Description: "Serialize XmlNode tree to XML string",
			Params: []ParamDoc{
				{Name: "node", Description: "XmlNode tree to serialize"},
			},
			Returns:   "XML string representation",
			Examples:  []Example{{Code: `serialize(element("root", [], [textNode("hello")]))`, Description: `Returns "<root>hello</root>"`}},
			Since:     "v0.9.3",
			Stability: StabilityStable,
			Tags:      []string{"xml", "serialize", "write", "tree"},
			Category:  "xml",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _xml_serialize: %v", err))
	}
}

func makeXmlSerializeType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.Con("XmlNode")).Returns(T.String()).Build()
}

func xmlSerializeImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	var buf strings.Builder
	if err := serializeNode(args[0], &buf, 0); err != nil {
		return nil, err
	}
	return &eval.StringValue{Value: buf.String()}, nil
}

func registerXmlSerializeWithDecl() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/xml",
		Name:    "_xml_serializeWithDecl",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeXmlSerializeType,
		Impl:    xmlSerializeWithDeclImpl,
		Metadata: &BuiltinMetadata{
			Description: "Serialize XmlNode tree to XML string with XML declaration",
			Params: []ParamDoc{
				{Name: "node", Description: "XmlNode tree to serialize"},
			},
			Returns:   "XML string with <?xml ...?> declaration",
			Examples:  []Example{{Code: `serializeWithDecl(element("root", [], [textNode("hello")]))`, Description: `Returns "<?xml version=\"1.0\" encoding=\"UTF-8\"?><root>hello</root>"`}},
			Since:     "v0.9.3",
			Stability: StabilityStable,
			Tags:      []string{"xml", "serialize", "write", "declaration"},
			Category:  "xml",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _xml_serializeWithDecl: %v", err))
	}
}

func xmlSerializeWithDeclImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	var buf strings.Builder
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	if err := serializeNode(args[0], &buf, 0); err != nil {
		return nil, err
	}
	return &eval.StringValue{Value: buf.String()}, nil
}

// _escapeXml: Escape string for XML text content
func registerEscapeXml() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/xml",
		Name:    "_escapeXml",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeEscapeXmlType,
		Impl:    escapeXmlImpl,
		Metadata: &BuiltinMetadata{
			Description: "Escape string for XML text content",
			Returns:     "string - XML-escaped text",
			Since:       "v0.9.4",
			Stability:   StabilityStable,
			Tags:        []string{"xml", "escape", "text"},
			Category:    "xml",
		},
	})
	if err != nil {
		panic("failed to register _escapeXml: " + err.Error())
	}
}

func makeEscapeXmlType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(T.String()).Build()
}

func escapeXmlImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	strVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_escapeXml: expected string, got %T", args[0])
	}
	escaped := html.EscapeString(strVal.Value)
	return &eval.StringValue{Value: escaped}, nil
}

// M-STDLIB-XML-V2: XmlNode constructor builtins
// Expose makeXmlElement/makeXmlText/makeXmlComment as AILANG builtins

func registerXmlElementCtor() {
	attrType := types.NewBuilder().Record(
		types.Field("name", types.NewBuilder().String()),
		types.Field("value", types.NewBuilder().String()),
	)
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/xml",
		Name:    "_xmlElement",
		NumArgs: 3,
		IsPure:  true,
		Effect:  "",
		Type: func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.String(), T.List(attrType), T.List(T.Con("XmlNode"))).Returns(T.Con("XmlNode")).Build()
		},
		Impl: xmlElementCtorImpl,
		Metadata: &BuiltinMetadata{
			Description: "Create an XML Element node",
			Params: []ParamDoc{
				{Name: "tag", Description: "Element tag name"},
				{Name: "attrs", Description: "List of {name, value} attribute records"},
				{Name: "children", Description: "List of child XmlNode values"},
			},
			Returns:   "XmlNode Element",
			Examples:  []Example{{Code: `xmlElement("div", [{name: "class", value: "main"}], [xmlText("hello")])`, Description: `Creates <div class="main">hello</div>`}},
			Since:     "v0.9.4",
			Stability: StabilityStable,
			Tags:      []string{"xml", "constructor", "element"},
			Category:  "xml",
		},
	})
	if err != nil {
		panic("failed to register _xmlElement: " + err.Error())
	}
}

func xmlElementCtorImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	tag, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_xmlElement: expected string for tag, got %T", args[0])
	}
	attrs, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_xmlElement: expected list for attrs, got %T", args[1])
	}
	children, ok := args[2].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_xmlElement: expected list for children, got %T", args[2])
	}
	return makeXmlElement(tag.Value, attrs.Elements, children.Elements), nil
}

func registerXmlTextCtor() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/xml",
		Name:    "_xmlText",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type: func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.String()).Returns(T.Con("XmlNode")).Build()
		},
		Impl: xmlTextCtorImpl,
		Metadata: &BuiltinMetadata{
			Description: "Create an XML Text node",
			Params:      []ParamDoc{{Name: "content", Description: "Text content"}},
			Returns:     "XmlNode Text",
			Examples:    []Example{{Code: `xmlText("hello")`, Description: `Creates Text("hello")`}},
			Since:       "v0.9.4",
			Stability:   StabilityStable,
			Tags:        []string{"xml", "constructor", "text"},
			Category:    "xml",
		},
	})
	if err != nil {
		panic("failed to register _xmlText: " + err.Error())
	}
}

func xmlTextCtorImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	text, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_xmlText: expected string, got %T", args[0])
	}
	return makeXmlText(text.Value), nil
}

func registerXmlCommentCtor() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/xml",
		Name:    "_xmlComment",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type: func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.String()).Returns(T.Con("XmlNode")).Build()
		},
		Impl: xmlCommentCtorImpl,
		Metadata: &BuiltinMetadata{
			Description: "Create an XML Comment node",
			Params:      []ParamDoc{{Name: "content", Description: "Comment content"}},
			Returns:     "XmlNode Comment",
			Examples:    []Example{{Code: `xmlComment(" TODO: fix this ")`, Description: `Creates Comment(" TODO: fix this ")`}},
			Since:       "v0.9.4",
			Stability:   StabilityStable,
			Tags:        []string{"xml", "constructor", "comment"},
			Category:    "xml",
		},
	})
	if err != nil {
		panic("failed to register _xmlComment: " + err.Error())
	}
}

func xmlCommentCtorImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	text, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_xmlComment: expected string, got %T", args[0])
	}
	return makeXmlComment(text.Value), nil
}
