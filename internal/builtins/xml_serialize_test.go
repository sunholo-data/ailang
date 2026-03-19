package builtins

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
)

// M-DOCPARSE-DX M4: XML serialization tests

func makeElement(tag string, attrs []*eval.RecordValue, children []eval.Value) *eval.TaggedValue {
	attrVals := make([]eval.Value, len(attrs))
	for i, a := range attrs {
		attrVals[i] = a
	}
	return &eval.TaggedValue{
		ModulePath: "std/xml",
		TypeName:   "XmlNode",
		CtorName:   "Element",
		Fields: []eval.Value{
			&eval.StringValue{Value: tag},
			&eval.ListValue{Elements: attrVals},
			&eval.ListValue{Elements: children},
		},
	}
}

func makeTextNode(text string) *eval.TaggedValue {
	return &eval.TaggedValue{
		ModulePath: "std/xml",
		TypeName:   "XmlNode",
		CtorName:   "Text",
		Fields:     []eval.Value{&eval.StringValue{Value: text}},
	}
}

func makeCDataNode(text string) *eval.TaggedValue {
	return &eval.TaggedValue{
		ModulePath: "std/xml",
		TypeName:   "XmlNode",
		CtorName:   "CData",
		Fields:     []eval.Value{&eval.StringValue{Value: text}},
	}
}

func makeCommentNode(text string) *eval.TaggedValue {
	return &eval.TaggedValue{
		ModulePath: "std/xml",
		TypeName:   "XmlNode",
		CtorName:   "Comment",
		Fields:     []eval.Value{&eval.StringValue{Value: text}},
	}
}

func makeAttr(name, value string) *eval.RecordValue {
	return &eval.RecordValue{
		Fields: map[string]eval.Value{
			"name":  &eval.StringValue{Value: name},
			"value": &eval.StringValue{Value: value},
		},
	}
}

func TestXmlSerialize_SimpleElement(t *testing.T) {
	ctx := effects.NewEffContext([]string{})
	node := makeElement("root", nil, []eval.Value{makeTextNode("hello")})

	result, err := xmlSerializeImpl(ctx, []eval.Value{node})
	require.NoError(t, err)
	assert.Equal(t, "<root>hello</root>", result.(*eval.StringValue).Value)
}

func TestXmlSerialize_SelfClosing(t *testing.T) {
	ctx := effects.NewEffContext([]string{})
	node := makeElement("br", nil, nil)

	result, err := xmlSerializeImpl(ctx, []eval.Value{node})
	require.NoError(t, err)
	assert.Equal(t, "<br/>", result.(*eval.StringValue).Value)
}

func TestXmlSerialize_Attributes(t *testing.T) {
	ctx := effects.NewEffContext([]string{})
	node := makeElement("div", []*eval.RecordValue{
		makeAttr("class", "main"),
		makeAttr("id", "content"),
	}, []eval.Value{makeTextNode("test")})

	result, err := xmlSerializeImpl(ctx, []eval.Value{node})
	require.NoError(t, err)
	assert.Equal(t, `<div class="main" id="content">test</div>`, result.(*eval.StringValue).Value)
}

func TestXmlSerialize_Nested(t *testing.T) {
	ctx := effects.NewEffContext([]string{})
	inner := makeElement("span", nil, []eval.Value{makeTextNode("world")})
	node := makeElement("div", nil, []eval.Value{makeTextNode("hello "), inner})

	result, err := xmlSerializeImpl(ctx, []eval.Value{node})
	require.NoError(t, err)
	assert.Equal(t, "<div>hello <span>world</span></div>", result.(*eval.StringValue).Value)
}

func TestXmlSerialize_CData(t *testing.T) {
	ctx := effects.NewEffContext([]string{})
	node := makeElement("script", nil, []eval.Value{makeCDataNode("x < y && y > z")})

	result, err := xmlSerializeImpl(ctx, []eval.Value{node})
	require.NoError(t, err)
	assert.Equal(t, "<script><![CDATA[x < y && y > z]]></script>", result.(*eval.StringValue).Value)
}

func TestXmlSerialize_Comment(t *testing.T) {
	ctx := effects.NewEffContext([]string{})
	node := makeElement("root", nil, []eval.Value{makeCommentNode(" a comment ")})

	result, err := xmlSerializeImpl(ctx, []eval.Value{node})
	require.NoError(t, err)
	assert.Equal(t, "<root><!-- a comment --></root>", result.(*eval.StringValue).Value)
}

func TestXmlSerialize_Escaping(t *testing.T) {
	ctx := effects.NewEffContext([]string{})
	node := makeElement("p", nil, []eval.Value{makeTextNode("a < b & c > d")})

	result, err := xmlSerializeImpl(ctx, []eval.Value{node})
	require.NoError(t, err)
	assert.Equal(t, "<p>a &lt; b &amp; c &gt; d</p>", result.(*eval.StringValue).Value)
}

func TestXmlSerialize_AttrEscaping(t *testing.T) {
	ctx := effects.NewEffContext([]string{})
	node := makeElement("a", []*eval.RecordValue{
		makeAttr("href", "page?x=1&y=2"),
	}, []eval.Value{makeTextNode("link")})

	result, err := xmlSerializeImpl(ctx, []eval.Value{node})
	require.NoError(t, err)
	assert.Contains(t, result.(*eval.StringValue).Value, "&amp;")
}

func TestXmlSerializeWithDecl(t *testing.T) {
	ctx := effects.NewEffContext([]string{})
	node := makeElement("root", nil, []eval.Value{makeTextNode("hello")})

	result, err := xmlSerializeWithDeclImpl(ctx, []eval.Value{node})
	require.NoError(t, err)
	xml := result.(*eval.StringValue).Value
	assert.True(t, strings.HasPrefix(xml, `<?xml version="1.0" encoding="UTF-8"?>`))
	assert.True(t, strings.HasSuffix(xml, "<root>hello</root>"))
}

// M-STDLIB-XML-V2: escapeXml tests

func TestEscapeXml_BasicEntities(t *testing.T) {
	ctx := effects.NewEffContext([]string{})
	tests := []struct {
		input, expected string
	}{
		{"<", "&lt;"},
		{">", "&gt;"},
		{"&", "&amp;"},
		{`"`, "&#34;"},
		{"'", "&#39;"},
		{"hello", "hello"},
		{"", ""},
		{"a < b & c > d", "a &lt; b &amp; c &gt; d"},
		{`<div class="main">`, `&lt;div class=&#34;main&#34;&gt;`},
	}
	for _, tt := range tests {
		result, err := escapeXmlImpl(ctx, []eval.Value{&eval.StringValue{Value: tt.input}})
		require.NoError(t, err)
		assert.Equal(t, tt.expected, result.(*eval.StringValue).Value, "input: %q", tt.input)
	}
}

func TestEscapeXml_DoubleEscape(t *testing.T) {
	ctx := effects.NewEffContext([]string{})
	// Already-escaped text should be double-escaped
	result, err := escapeXmlImpl(ctx, []eval.Value{&eval.StringValue{Value: "&amp;"}})
	require.NoError(t, err)
	assert.Equal(t, "&amp;amp;", result.(*eval.StringValue).Value)
}

func TestEscapeXml_WrongType(t *testing.T) {
	ctx := effects.NewEffContext([]string{})
	_, err := escapeXmlImpl(ctx, []eval.Value{&eval.IntValue{Value: 42}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "_escapeXml")
}

// M-STDLIB-XML-V2: XmlNode constructor tests

func TestXmlElementCtor_Basic(t *testing.T) {
	ctx := effects.NewEffContext([]string{})
	// xmlElement("p", [], [xmlText("hello")])
	textNode, err := xmlTextCtorImpl(ctx, []eval.Value{&eval.StringValue{Value: "hello"}})
	require.NoError(t, err)

	result, err := xmlElementCtorImpl(ctx, []eval.Value{
		&eval.StringValue{Value: "p"},
		&eval.ListValue{Elements: nil},
		&eval.ListValue{Elements: []eval.Value{textNode}},
	})
	require.NoError(t, err)

	// Serialize and verify
	serialized, err := xmlSerializeImpl(ctx, []eval.Value{result})
	require.NoError(t, err)
	assert.Equal(t, "<p>hello</p>", serialized.(*eval.StringValue).Value)
}

func TestXmlElementCtor_WithAttrs(t *testing.T) {
	ctx := effects.NewEffContext([]string{})
	attrs := &eval.ListValue{Elements: []eval.Value{
		&eval.RecordValue{Fields: map[string]eval.Value{
			"name":  &eval.StringValue{Value: "class"},
			"value": &eval.StringValue{Value: "main"},
		}},
	}}
	result, err := xmlElementCtorImpl(ctx, []eval.Value{
		&eval.StringValue{Value: "div"},
		attrs,
		&eval.ListValue{Elements: nil},
	})
	require.NoError(t, err)

	serialized, err := xmlSerializeImpl(ctx, []eval.Value{result})
	require.NoError(t, err)
	assert.Equal(t, `<div class="main"/>`, serialized.(*eval.StringValue).Value)
}

func TestXmlTextCtor(t *testing.T) {
	ctx := effects.NewEffContext([]string{})
	result, err := xmlTextCtorImpl(ctx, []eval.Value{&eval.StringValue{Value: "hello world"}})
	require.NoError(t, err)

	tv := result.(*eval.TaggedValue)
	assert.Equal(t, "Text", tv.CtorName)
	assert.Equal(t, "XmlNode", tv.TypeName)
	assert.Equal(t, "hello world", tv.Fields[0].(*eval.StringValue).Value)
}

func TestXmlCommentCtor(t *testing.T) {
	ctx := effects.NewEffContext([]string{})
	result, err := xmlCommentCtorImpl(ctx, []eval.Value{&eval.StringValue{Value: " TODO "}})
	require.NoError(t, err)

	serialized, err := xmlSerializeImpl(ctx, []eval.Value{result})
	require.NoError(t, err)
	assert.Equal(t, "<!-- TODO -->", serialized.(*eval.StringValue).Value)
}

func TestXmlCtors_RoundTrip(t *testing.T) {
	// Build: <root><p class="intro">Hello</p><p>World</p></root>
	ctx := effects.NewEffContext([]string{})

	text1, _ := xmlTextCtorImpl(ctx, []eval.Value{&eval.StringValue{Value: "Hello"}})
	text2, _ := xmlTextCtorImpl(ctx, []eval.Value{&eval.StringValue{Value: "World"}})

	p1, _ := xmlElementCtorImpl(ctx, []eval.Value{
		&eval.StringValue{Value: "p"},
		&eval.ListValue{Elements: []eval.Value{
			&eval.RecordValue{Fields: map[string]eval.Value{
				"name":  &eval.StringValue{Value: "class"},
				"value": &eval.StringValue{Value: "intro"},
			}},
		}},
		&eval.ListValue{Elements: []eval.Value{text1}},
	})
	p2, _ := xmlElementCtorImpl(ctx, []eval.Value{
		&eval.StringValue{Value: "p"},
		&eval.ListValue{Elements: nil},
		&eval.ListValue{Elements: []eval.Value{text2}},
	})

	root, _ := xmlElementCtorImpl(ctx, []eval.Value{
		&eval.StringValue{Value: "root"},
		&eval.ListValue{Elements: nil},
		&eval.ListValue{Elements: []eval.Value{p1, p2}},
	})

	serialized, err := xmlSerializeImpl(ctx, []eval.Value{root})
	require.NoError(t, err)
	assert.Equal(t, `<root><p class="intro">Hello</p><p>World</p></root>`, serialized.(*eval.StringValue).Value)
}

func TestXmlElementCtor_WrongType(t *testing.T) {
	ctx := effects.NewEffContext([]string{})
	_, err := xmlElementCtorImpl(ctx, []eval.Value{
		&eval.IntValue{Value: 42},
		&eval.ListValue{Elements: nil},
		&eval.ListValue{Elements: nil},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "_xmlElement")
}

func TestXmlSerialize_Roundtrip(t *testing.T) {
	// Parse XML, then serialize, then parse again — should produce equivalent trees
	ctx := effects.NewEffContext([]string{})
	input := `<root><child id="1">hello</child><child id="2">world</child></root>`

	// Parse
	parseResult, err := xmlParseImpl(ctx, []eval.Value{&eval.StringValue{Value: input}})
	require.NoError(t, err)
	tagged := parseResult.(*eval.TaggedValue)
	require.Equal(t, "Ok", tagged.CtorName)
	tree := tagged.Fields[0]

	// Serialize
	serialized, err := xmlSerializeImpl(ctx, []eval.Value{tree})
	require.NoError(t, err)
	xmlStr := serialized.(*eval.StringValue).Value

	// Parse again
	parseResult2, err := xmlParseImpl(ctx, []eval.Value{&eval.StringValue{Value: xmlStr}})
	require.NoError(t, err)
	tagged2 := parseResult2.(*eval.TaggedValue)
	require.Equal(t, "Ok", tagged2.CtorName)
	tree2 := tagged2.Fields[0]

	// Serialize again — should produce same string
	serialized2, err := xmlSerializeImpl(ctx, []eval.Value{tree2})
	require.NoError(t, err)
	assert.Equal(t, xmlStr, serialized2.(*eval.StringValue).Value)
}
