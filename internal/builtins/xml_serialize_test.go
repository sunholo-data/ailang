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
