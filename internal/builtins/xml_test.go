package builtins

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
)

// ============================================================================
// Test helpers
// ============================================================================

func xmlTestCtx(t *testing.T) *effects.EffContext {
	t.Helper()
	return effects.NewEffContext(nil)
}

func xmlAssertOk(t *testing.T, result eval.Value) eval.Value {
	t.Helper()
	tv, ok := result.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", result)
	}
	if tv.CtorName != "Ok" {
		errMsg := ""
		if len(tv.Fields) > 0 {
			if sv, ok := tv.Fields[0].(*eval.StringValue); ok {
				errMsg = sv.Value
			}
		}
		t.Fatalf("expected Ok, got Err(%q)", errMsg)
	}
	if len(tv.Fields) != 1 {
		t.Fatalf("expected 1 field in Ok, got %d", len(tv.Fields))
	}
	return tv.Fields[0]
}

func xmlAssertErr(t *testing.T, result eval.Value, contains string) {
	t.Helper()
	tv, ok := result.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", result)
	}
	if tv.CtorName != "Err" {
		t.Fatalf("expected Err, got Ok")
	}
	if len(tv.Fields) != 1 {
		t.Fatalf("expected 1 field in Err, got %d", len(tv.Fields))
	}
	sv, ok := tv.Fields[0].(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue in Err, got %T", tv.Fields[0])
	}
	if !strings.Contains(sv.Value, contains) {
		t.Fatalf("error message %q does not contain %q", sv.Value, contains)
	}
}

func xmlAssertElement(t *testing.T, node eval.Value, tag string) *eval.TaggedValue {
	t.Helper()
	tv, ok := node.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", node)
	}
	if tv.CtorName != "Element" {
		t.Fatalf("expected Element, got %s", tv.CtorName)
	}
	sv, ok := tv.Fields[0].(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue for tag, got %T", tv.Fields[0])
	}
	if sv.Value != tag {
		t.Fatalf("expected tag %q, got %q", tag, sv.Value)
	}
	return tv
}

func xmlGetChildren(t *testing.T, elem *eval.TaggedValue) []eval.Value {
	t.Helper()
	lv, ok := elem.Fields[2].(*eval.ListValue)
	if !ok {
		t.Fatalf("expected ListValue for children, got %T", elem.Fields[2])
	}
	return lv.Elements
}

func xmlGetAttrs(t *testing.T, elem *eval.TaggedValue) []eval.Value {
	t.Helper()
	lv, ok := elem.Fields[1].(*eval.ListValue)
	if !ok {
		t.Fatalf("expected ListValue for attrs, got %T", elem.Fields[1])
	}
	return lv.Elements
}

func xmlAssertSome(t *testing.T, result eval.Value) eval.Value {
	t.Helper()
	tv, ok := result.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", result)
	}
	if tv.CtorName != "Some" {
		t.Fatalf("expected Some, got %s", tv.CtorName)
	}
	return tv.Fields[0]
}

func xmlAssertNone(t *testing.T, result eval.Value) {
	t.Helper()
	tv, ok := result.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", result)
	}
	if tv.CtorName != "None" {
		t.Fatalf("expected None, got %s", tv.CtorName)
	}
}

// ============================================================================
// _xml_parse tests
// ============================================================================

func TestXmlParse_SimpleElement(t *testing.T) {
	ctx := xmlTestCtx(t)
	args := []eval.Value{&eval.StringValue{Value: "<root/>"}}
	result, err := xmlParseImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inner := xmlAssertOk(t, result)
	elem := xmlAssertElement(t, inner, "root")
	children := xmlGetChildren(t, elem)
	if len(children) != 0 {
		t.Fatalf("expected 0 children, got %d", len(children))
	}
	attrs := xmlGetAttrs(t, elem)
	if len(attrs) != 0 {
		t.Fatalf("expected 0 attrs, got %d", len(attrs))
	}
}

func TestXmlParse_NestedElements(t *testing.T) {
	ctx := xmlTestCtx(t)
	args := []eval.Value{&eval.StringValue{Value: "<root><item>hello</item></root>"}}
	result, err := xmlParseImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inner := xmlAssertOk(t, result)
	root := xmlAssertElement(t, inner, "root")
	children := xmlGetChildren(t, root)
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	item := xmlAssertElement(t, children[0], "item")
	itemChildren := xmlGetChildren(t, item)
	if len(itemChildren) != 1 {
		t.Fatalf("expected 1 child of item, got %d", len(itemChildren))
	}
	textNode, ok := itemChildren[0].(*eval.TaggedValue)
	if !ok || textNode.CtorName != "Text" {
		t.Fatalf("expected Text node, got %v", itemChildren[0])
	}
	text := textNode.Fields[0].(*eval.StringValue).Value
	if text != "hello" {
		t.Fatalf("expected 'hello', got %q", text)
	}
}

func TestXmlParse_Attributes(t *testing.T) {
	ctx := xmlTestCtx(t)
	args := []eval.Value{&eval.StringValue{Value: `<div class="main" id="1"/>`}}
	result, err := xmlParseImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inner := xmlAssertOk(t, result)
	elem := xmlAssertElement(t, inner, "div")
	attrs := xmlGetAttrs(t, elem)
	if len(attrs) != 2 {
		t.Fatalf("expected 2 attrs, got %d", len(attrs))
	}

	// Check first attribute
	rec0, ok := attrs[0].(*eval.RecordValue)
	if !ok {
		t.Fatalf("expected RecordValue, got %T", attrs[0])
	}
	if rec0.Fields["name"].(*eval.StringValue).Value != "class" {
		t.Fatalf("expected attr name 'class', got %q", rec0.Fields["name"].(*eval.StringValue).Value)
	}
	if rec0.Fields["value"].(*eval.StringValue).Value != "main" {
		t.Fatalf("expected attr value 'main', got %q", rec0.Fields["value"].(*eval.StringValue).Value)
	}
}

func TestXmlParse_MixedContent(t *testing.T) {
	ctx := xmlTestCtx(t)
	args := []eval.Value{&eval.StringValue{Value: "<p>hello <b>world</b> end</p>"}}
	result, err := xmlParseImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inner := xmlAssertOk(t, result)
	p := xmlAssertElement(t, inner, "p")
	children := xmlGetChildren(t, p)
	// Should have: Text("hello "), Element("b", ...), Text(" end")
	if len(children) != 3 {
		t.Fatalf("expected 3 children in mixed content, got %d", len(children))
	}

	// First child: Text
	text0, ok := children[0].(*eval.TaggedValue)
	if !ok || text0.CtorName != "Text" {
		t.Fatalf("expected Text node at [0], got %v", children[0])
	}
	if text0.Fields[0].(*eval.StringValue).Value != "hello " {
		t.Fatalf("expected 'hello ', got %q", text0.Fields[0].(*eval.StringValue).Value)
	}

	// Second child: Element("b", ...)
	xmlAssertElement(t, children[1], "b")

	// Third child: Text
	text2, ok := children[2].(*eval.TaggedValue)
	if !ok || text2.CtorName != "Text" {
		t.Fatalf("expected Text node at [2], got %v", children[2])
	}
	if text2.Fields[0].(*eval.StringValue).Value != " end" {
		t.Fatalf("expected ' end', got %q", text2.Fields[0].(*eval.StringValue).Value)
	}
}

func TestXmlParse_Comment(t *testing.T) {
	ctx := xmlTestCtx(t)
	args := []eval.Value{&eval.StringValue{Value: "<root><!-- a comment --><item/></root>"}}
	result, err := xmlParseImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inner := xmlAssertOk(t, result)
	root := xmlAssertElement(t, inner, "root")
	children := xmlGetChildren(t, root)
	if len(children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(children))
	}
	comment, ok := children[0].(*eval.TaggedValue)
	if !ok || comment.CtorName != "Comment" {
		t.Fatalf("expected Comment node, got %v", children[0])
	}
	if comment.Fields[0].(*eval.StringValue).Value != " a comment " {
		t.Fatalf("expected ' a comment ', got %q", comment.Fields[0].(*eval.StringValue).Value)
	}
}

func TestXmlParse_CDATA(t *testing.T) {
	ctx := xmlTestCtx(t)
	args := []eval.Value{&eval.StringValue{Value: "<root><![CDATA[raw <data>]]></root>"}}
	result, err := xmlParseImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inner := xmlAssertOk(t, result)
	root := xmlAssertElement(t, inner, "root")
	children := xmlGetChildren(t, root)
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	// Go encoding/xml delivers CDATA as CharData (not separately distinguishable)
	// So it shows up as Text, not CData — this is a known Go limitation
	tv, ok := children[0].(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", children[0])
	}
	// Accept either Text or CData
	if tv.CtorName != "Text" && tv.CtorName != "CData" {
		t.Fatalf("expected Text or CData, got %s", tv.CtorName)
	}
	if tv.Fields[0].(*eval.StringValue).Value != "raw <data>" {
		t.Fatalf("expected 'raw <data>', got %q", tv.Fields[0].(*eval.StringValue).Value)
	}
}

func TestXmlParse_Namespaces(t *testing.T) {
	ctx := xmlTestCtx(t)
	xml := `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:t>Hello</w:t></w:p></w:body></w:document>`
	args := []eval.Value{&eval.StringValue{Value: xml}}
	result, err := xmlParseImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inner := xmlAssertOk(t, result)
	doc := xmlAssertElement(t, inner, "w:document")
	docChildren := xmlGetChildren(t, doc)
	if len(docChildren) != 1 {
		t.Fatalf("expected 1 child of w:document, got %d", len(docChildren))
	}
	body := xmlAssertElement(t, docChildren[0], "w:body")
	bodyChildren := xmlGetChildren(t, body)
	if len(bodyChildren) != 1 {
		t.Fatalf("expected 1 child of w:body, got %d", len(bodyChildren))
	}
	p := xmlAssertElement(t, bodyChildren[0], "w:p")
	pChildren := xmlGetChildren(t, p)
	if len(pChildren) != 1 {
		t.Fatalf("expected 1 child of w:p, got %d", len(pChildren))
	}
	wt := xmlAssertElement(t, pChildren[0], "w:t")
	wtChildren := xmlGetChildren(t, wt)
	if len(wtChildren) != 1 {
		t.Fatalf("expected 1 child of w:t, got %d", len(wtChildren))
	}
}

func TestXmlParse_MalformedXml(t *testing.T) {
	ctx := xmlTestCtx(t)
	// Plain text is treated as CharData by Go's xml decoder, so it produces
	// a Text node. Use actually malformed XML with mismatched tags instead.
	args := []eval.Value{&eval.StringValue{Value: "<root><mismatch></root>"}}
	result, err := xmlParseImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	xmlAssertErr(t, result, "XML parse error")
}

func TestXmlParse_MalformedTags(t *testing.T) {
	ctx := xmlTestCtx(t)
	args := []eval.Value{&eval.StringValue{Value: "<root><unclosed>"}}
	result, err := xmlParseImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	xmlAssertErr(t, result, "XML parse error")
}

func TestXmlParse_EmptyInput(t *testing.T) {
	ctx := xmlTestCtx(t)
	args := []eval.Value{&eval.StringValue{Value: ""}}
	result, err := xmlParseImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	xmlAssertErr(t, result, "empty document")
}

func TestXmlParse_DepthLimit(t *testing.T) {
	ctx := xmlTestCtx(t)
	// Create XML with depth > 256
	var sb strings.Builder
	for i := 0; i < 260; i++ {
		sb.WriteString("<a>")
	}
	sb.WriteString("deep")
	for i := 0; i < 260; i++ {
		sb.WriteString("</a>")
	}
	args := []eval.Value{&eval.StringValue{Value: sb.String()}}
	result, err := xmlParseImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	xmlAssertErr(t, result, "maximum depth exceeded")
}

func TestXmlParse_SizeLimit(t *testing.T) {
	ctx := xmlTestCtx(t)
	// Create input > 50MB
	huge := strings.Repeat("x", xmlMaxInputSize+1)
	args := []eval.Value{&eval.StringValue{Value: huge}}
	result, err := xmlParseImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	xmlAssertErr(t, result, "XML input too large")
}

// ============================================================================
// _xml_findAll tests
// ============================================================================

func parseTestXml(t *testing.T, xmlStr string) eval.Value {
	t.Helper()
	ctx := xmlTestCtx(t)
	result, err := xmlParseImpl(ctx, []eval.Value{&eval.StringValue{Value: xmlStr}})
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	return xmlAssertOk(t, result)
}

func TestXmlFindAll_Basic(t *testing.T) {
	root := parseTestXml(t, "<root><item>a</item><other/><item>b</item></root>")
	ctx := xmlTestCtx(t)
	result, err := xmlFindAllImpl(ctx, []eval.Value{root, &eval.StringValue{Value: "item"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, ok := result.(*eval.ListValue)
	if !ok {
		t.Fatalf("expected ListValue, got %T", result)
	}
	if len(list.Elements) != 2 {
		t.Fatalf("expected 2 items, got %d", len(list.Elements))
	}
	xmlAssertElement(t, list.Elements[0], "item")
	xmlAssertElement(t, list.Elements[1], "item")
}

func TestXmlFindAll_Nested(t *testing.T) {
	root := parseTestXml(t, "<root><a><b><a>deep</a></b></a></root>")
	ctx := xmlTestCtx(t)
	result, err := xmlFindAllImpl(ctx, []eval.Value{root, &eval.StringValue{Value: "a"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list := result.(*eval.ListValue)
	if len(list.Elements) != 2 {
		t.Fatalf("expected 2 'a' elements (nested), got %d", len(list.Elements))
	}
}

func TestXmlFindAll_NoMatch(t *testing.T) {
	root := parseTestXml(t, "<root><item/></root>")
	ctx := xmlTestCtx(t)
	result, err := xmlFindAllImpl(ctx, []eval.Value{root, &eval.StringValue{Value: "nonexistent"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list := result.(*eval.ListValue)
	if len(list.Elements) != 0 {
		t.Fatalf("expected 0 results, got %d", len(list.Elements))
	}
}

// ============================================================================
// _xml_findFirst tests
// ============================================================================

func TestXmlFindFirst_Found(t *testing.T) {
	root := parseTestXml(t, "<root><a/><b><c/></b></root>")
	ctx := xmlTestCtx(t)
	result, err := xmlFindFirstImpl(ctx, []eval.Value{root, &eval.StringValue{Value: "c"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inner := xmlAssertSome(t, result)
	xmlAssertElement(t, inner, "c")
}

func TestXmlFindFirst_NotFound(t *testing.T) {
	root := parseTestXml(t, "<root><a/></root>")
	ctx := xmlTestCtx(t)
	result, err := xmlFindFirstImpl(ctx, []eval.Value{root, &eval.StringValue{Value: "missing"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	xmlAssertNone(t, result)
}

// ============================================================================
// _xml_getText tests
// ============================================================================

func TestXmlGetText_SimpleElement(t *testing.T) {
	root := parseTestXml(t, "<p>hello</p>")
	ctx := xmlTestCtx(t)
	result, err := xmlGetTextImpl(ctx, []eval.Value{root})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sv := result.(*eval.StringValue)
	if sv.Value != "hello" {
		t.Fatalf("expected 'hello', got %q", sv.Value)
	}
}

func TestXmlGetText_MixedContent(t *testing.T) {
	root := parseTestXml(t, "<p>hello <b>world</b> end</p>")
	ctx := xmlTestCtx(t)
	result, err := xmlGetTextImpl(ctx, []eval.Value{root})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sv := result.(*eval.StringValue)
	if sv.Value != "hello world end" {
		t.Fatalf("expected 'hello world end', got %q", sv.Value)
	}
}

func TestXmlGetText_TextNode(t *testing.T) {
	// getText on a Text node directly
	textNode := makeXmlText("direct text")
	ctx := xmlTestCtx(t)
	result, err := xmlGetTextImpl(ctx, []eval.Value{textNode})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sv := result.(*eval.StringValue)
	if sv.Value != "direct text" {
		t.Fatalf("expected 'direct text', got %q", sv.Value)
	}
}

func TestXmlGetText_EmptyElement(t *testing.T) {
	root := parseTestXml(t, "<empty/>")
	ctx := xmlTestCtx(t)
	result, err := xmlGetTextImpl(ctx, []eval.Value{root})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sv := result.(*eval.StringValue)
	if sv.Value != "" {
		t.Fatalf("expected empty string, got %q", sv.Value)
	}
}

// ============================================================================
// _xml_getAttr tests
// ============================================================================

func TestXmlGetAttr_Found(t *testing.T) {
	root := parseTestXml(t, `<div class="main" id="1"/>`)
	ctx := xmlTestCtx(t)
	result, err := xmlGetAttrImpl(ctx, []eval.Value{root, &eval.StringValue{Value: "class"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inner := xmlAssertSome(t, result)
	sv := inner.(*eval.StringValue)
	if sv.Value != "main" {
		t.Fatalf("expected 'main', got %q", sv.Value)
	}
}

func TestXmlGetAttr_NotFound(t *testing.T) {
	root := parseTestXml(t, `<div class="main"/>`)
	ctx := xmlTestCtx(t)
	result, err := xmlGetAttrImpl(ctx, []eval.Value{root, &eval.StringValue{Value: "missing"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	xmlAssertNone(t, result)
}

func TestXmlGetAttr_NonElement(t *testing.T) {
	textNode := makeXmlText("hello")
	ctx := xmlTestCtx(t)
	result, err := xmlGetAttrImpl(ctx, []eval.Value{textNode, &eval.StringValue{Value: "attr"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	xmlAssertNone(t, result)
}

// ============================================================================
// _xml_getChildren tests
// ============================================================================

func TestXmlGetChildren_Element(t *testing.T) {
	root := parseTestXml(t, "<root><a/><b/><c/></root>")
	ctx := xmlTestCtx(t)
	result, err := xmlGetChildrenImpl(ctx, []eval.Value{root})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list := result.(*eval.ListValue)
	if len(list.Elements) != 3 {
		t.Fatalf("expected 3 children, got %d", len(list.Elements))
	}
	xmlAssertElement(t, list.Elements[0], "a")
	xmlAssertElement(t, list.Elements[1], "b")
	xmlAssertElement(t, list.Elements[2], "c")
}

func TestXmlGetChildren_NonElement(t *testing.T) {
	textNode := makeXmlText("hello")
	ctx := xmlTestCtx(t)
	result, err := xmlGetChildrenImpl(ctx, []eval.Value{textNode})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list := result.(*eval.ListValue)
	if len(list.Elements) != 0 {
		t.Fatalf("expected 0 children for Text node, got %d", len(list.Elements))
	}
}

// ============================================================================
// _xml_getTag tests
// ============================================================================

func TestXmlGetTag_Element(t *testing.T) {
	root := parseTestXml(t, "<myTag/>")
	ctx := xmlTestCtx(t)
	result, err := xmlGetTagImpl(ctx, []eval.Value{root})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sv := result.(*eval.StringValue)
	if sv.Value != "myTag" {
		t.Fatalf("expected 'myTag', got %q", sv.Value)
	}
}

func TestXmlGetTag_NamespacedElement(t *testing.T) {
	root := parseTestXml(t, `<w:p xmlns:w="http://example.com"/>`)
	ctx := xmlTestCtx(t)
	result, err := xmlGetTagImpl(ctx, []eval.Value{root})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sv := result.(*eval.StringValue)
	if sv.Value != "w:p" {
		t.Fatalf("expected 'w:p', got %q", sv.Value)
	}
}

func TestXmlGetTag_NonElement(t *testing.T) {
	textNode := makeXmlText("hello")
	ctx := xmlTestCtx(t)
	result, err := xmlGetTagImpl(ctx, []eval.Value{textNode})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sv := result.(*eval.StringValue)
	if sv.Value != "" {
		t.Fatalf("expected empty string, got %q", sv.Value)
	}
}

// ============================================================================
// OOXML integration test
// ============================================================================

func TestXmlParse_OOXMLFragment(t *testing.T) {
	// Simulate a real DOCX word/document.xml excerpt
	ooxml := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r>
        <w:t>First paragraph</w:t>
      </w:r>
    </w:p>
    <w:p>
      <w:r>
        <w:t>Second paragraph</w:t>
      </w:r>
    </w:p>
  </w:body>
</w:document>`

	root := parseTestXml(t, ooxml)
	ctx := xmlTestCtx(t)

	// Find all w:t elements
	result, err := xmlFindAllImpl(ctx, []eval.Value{root, &eval.StringValue{Value: "w:t"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list := result.(*eval.ListValue)
	if len(list.Elements) != 2 {
		t.Fatalf("expected 2 w:t elements, got %d", len(list.Elements))
	}

	// Extract text from first w:t
	text1, err := xmlGetTextImpl(ctx, []eval.Value{list.Elements[0]})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text1.(*eval.StringValue).Value != "First paragraph" {
		t.Fatalf("expected 'First paragraph', got %q", text1.(*eval.StringValue).Value)
	}

	// Extract text from second w:t
	text2, err := xmlGetTextImpl(ctx, []eval.Value{list.Elements[1]})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text2.(*eval.StringValue).Value != "Second paragraph" {
		t.Fatalf("expected 'Second paragraph', got %q", text2.(*eval.StringValue).Value)
	}

	// findFirst for w:body
	bodyResult, err := xmlFindFirstImpl(ctx, []eval.Value{root, &eval.StringValue{Value: "w:body"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body := xmlAssertSome(t, bodyResult)
	xmlAssertElement(t, body, "w:body")

	// getChildren of body should have 2 w:p elements
	bodyChildren, err := xmlGetChildrenImpl(ctx, []eval.Value{body})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	childList := bodyChildren.(*eval.ListValue)
	if len(childList.Elements) != 2 {
		t.Fatalf("expected 2 w:p children, got %d", len(childList.Elements))
	}
	xmlAssertElement(t, childList.Elements[0], "w:p")
	xmlAssertElement(t, childList.Elements[1], "w:p")
}

func TestXmlParse_OOXMLTable(t *testing.T) {
	// Simulate a real DOCX table with deeply nested cell text
	ooxml := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:tbl>
      <w:tblPr>
        <w:tblW w:w="5000" w:type="pct"/>
      </w:tblPr>
      <w:tr>
        <w:tc>
          <w:tcPr><w:tcW w:w="2500" w:type="pct"/></w:tcPr>
          <w:p>
            <w:r>
              <w:t>Cell A1</w:t>
            </w:r>
          </w:p>
        </w:tc>
        <w:tc>
          <w:tcPr><w:tcW w:w="2500" w:type="pct"/></w:tcPr>
          <w:p>
            <w:r>
              <w:t>Cell B1</w:t>
            </w:r>
          </w:p>
        </w:tc>
      </w:tr>
      <w:tr>
        <w:tc>
          <w:p>
            <w:r>
              <w:t xml:space="preserve">Cell A2</w:t>
            </w:r>
          </w:p>
        </w:tc>
        <w:tc>
          <w:p>
            <w:r>
              <w:t xml:space="preserve">Cell B2</w:t>
            </w:r>
          </w:p>
        </w:tc>
      </w:tr>
    </w:tbl>
  </w:body>
</w:document>`

	root := parseTestXml(t, ooxml)
	ctx := xmlTestCtx(t)

	// Find all table cells
	cellResult, err := xmlFindAllImpl(ctx, []eval.Value{root, &eval.StringValue{Value: "w:tc"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cells := cellResult.(*eval.ListValue)
	if len(cells.Elements) != 4 {
		t.Fatalf("expected 4 w:tc cells, got %d", len(cells.Elements))
	}

	// Extract text from each cell - this is what docparse's getCellText does
	expectedTexts := []string{"Cell A1", "Cell B1", "Cell A2", "Cell B2"}
	for i, cell := range cells.Elements {
		text, err := xmlGetTextImpl(ctx, []eval.Value{cell})
		if err != nil {
			t.Fatalf("cell %d: unexpected error: %v", i, err)
		}
		got := text.(*eval.StringValue).Value
		if got != expectedTexts[i] {
			t.Errorf("cell %d: expected %q, got %q", i, expectedTexts[i], got)
		}
	}

	// Also test findAll for w:t elements (deeply nested in table)
	textResult, err := xmlFindAllImpl(ctx, []eval.Value{root, &eval.StringValue{Value: "w:t"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	textElems := textResult.(*eval.ListValue)
	if len(textElems.Elements) != 4 {
		t.Fatalf("expected 4 w:t elements, got %d", len(textElems.Elements))
	}

	// Verify getText on individual w:t elements
	for i, te := range textElems.Elements {
		text, err := xmlGetTextImpl(ctx, []eval.Value{te})
		if err != nil {
			t.Fatalf("w:t %d: unexpected error: %v", i, err)
		}
		got := text.(*eval.StringValue).Value
		if got != expectedTexts[i] {
			t.Errorf("w:t %d: expected %q, got %q", i, expectedTexts[i], got)
		}
	}

	// Test finding rows
	rowResult, err := xmlFindAllImpl(ctx, []eval.Value{root, &eval.StringValue{Value: "w:tr"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rows := rowResult.(*eval.ListValue)
	if len(rows.Elements) != 2 {
		t.Fatalf("expected 2 w:tr rows, got %d", len(rows.Elements))
	}
}
