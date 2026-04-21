package builtins

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// ============================================================================
// _xml_findAllTexts tests
// ============================================================================

func TestXmlFindAllTexts_Basic(t *testing.T) {
	root := parseTestXml(t, "<root><p>Hello</p><div/><p>World</p></root>")
	ctx := xmlTestCtx(t)
	result, err := xmlFindAllTextsImpl(ctx, []eval.Value{root, &eval.StringValue{Value: "p"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list := result.(*eval.ListValue)
	if len(list.Elements) != 2 {
		t.Fatalf("expected 2 texts, got %d", len(list.Elements))
	}
	if list.Elements[0].(*eval.StringValue).Value != "Hello" {
		t.Errorf("expected 'Hello', got %q", list.Elements[0].(*eval.StringValue).Value)
	}
	if list.Elements[1].(*eval.StringValue).Value != "World" {
		t.Errorf("expected 'World', got %q", list.Elements[1].(*eval.StringValue).Value)
	}
}

func TestXmlFindAllTexts_Nested(t *testing.T) {
	root := parseTestXml(t, "<root><p>One <b>bold</b> text</p><p>Two</p></root>")
	ctx := xmlTestCtx(t)
	result, err := xmlFindAllTextsImpl(ctx, []eval.Value{root, &eval.StringValue{Value: "p"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list := result.(*eval.ListValue)
	if len(list.Elements) != 2 {
		t.Fatalf("expected 2 texts, got %d", len(list.Elements))
	}
	// getText concatenates all descendant text
	if list.Elements[0].(*eval.StringValue).Value != "One bold text" {
		t.Errorf("expected 'One bold text', got %q", list.Elements[0].(*eval.StringValue).Value)
	}
}

func TestXmlFindAllTexts_NoMatch(t *testing.T) {
	root := parseTestXml(t, "<root><item/></root>")
	ctx := xmlTestCtx(t)
	result, err := xmlFindAllTextsImpl(ctx, []eval.Value{root, &eval.StringValue{Value: "p"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list := result.(*eval.ListValue)
	if len(list.Elements) != 0 {
		t.Fatalf("expected 0 results, got %d", len(list.Elements))
	}
}

func TestXmlFindAllTexts_DuplicateNamespace(t *testing.T) {
	// Determinism test: 20 iterations with duplicate namespace prefixes
	xmlStr := `<package xmlns="http://www.idpf.org/2007/opf" xmlns:opf="http://www.idpf.org/2007/opf">
		<spine><itemref>ch1</itemref><itemref>ch2</itemref><itemref>ch3</itemref>
		<itemref>ch4</itemref><itemref>ch5</itemref><itemref>ch6</itemref>
		<itemref>ch7</itemref><itemref>ch8</itemref><itemref>ch9</itemref>
		<itemref>ch10</itemref><itemref>ch11</itemref><itemref>ch12</itemref></spine>
	</package>`
	ctx := xmlTestCtx(t)
	for i := 0; i < 20; i++ {
		root := parseTestXml(t, xmlStr)
		result, err := xmlFindAllTextsImpl(ctx, []eval.Value{root, &eval.StringValue{Value: "itemref"}})
		if err != nil {
			t.Fatalf("run %d: unexpected error: %v", i, err)
		}
		list := result.(*eval.ListValue)
		if len(list.Elements) != 12 {
			t.Fatalf("run %d: expected 12 itemref texts, got %d (nondeterministic)", i, len(list.Elements))
		}
	}
}

// ============================================================================
// _xml_findAllAttrs tests
// ============================================================================

func TestXmlFindAllAttrs_Basic(t *testing.T) {
	root := parseTestXml(t, `<root><item href="a.html"/><div/><item href="b.html"/><item href="c.html"/></root>`)
	ctx := xmlTestCtx(t)
	result, err := xmlFindAllAttrsImpl(ctx, []eval.Value{
		root,
		&eval.StringValue{Value: "item"},
		&eval.StringValue{Value: "href"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list := result.(*eval.ListValue)
	if len(list.Elements) != 3 {
		t.Fatalf("expected 3 attrs, got %d", len(list.Elements))
	}
	expected := []string{"a.html", "b.html", "c.html"}
	for i, exp := range expected {
		got := list.Elements[i].(*eval.StringValue).Value
		if got != exp {
			t.Errorf("element %d: expected %q, got %q", i, exp, got)
		}
	}
}

func TestXmlFindAllAttrs_MissingAttr(t *testing.T) {
	// Elements without the requested attribute are skipped
	root := parseTestXml(t, `<root><item href="a.html"/><item/><item href="c.html"/></root>`)
	ctx := xmlTestCtx(t)
	result, err := xmlFindAllAttrsImpl(ctx, []eval.Value{
		root,
		&eval.StringValue{Value: "item"},
		&eval.StringValue{Value: "href"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list := result.(*eval.ListValue)
	if len(list.Elements) != 2 {
		t.Fatalf("expected 2 attrs (skipping element without href), got %d", len(list.Elements))
	}
}

func TestXmlFindAllAttrs_DuplicateNamespace(t *testing.T) {
	// Determinism test with duplicate namespace prefixes
	xmlStr := `<manifest xmlns="http://www.idpf.org/2007/opf" xmlns:opf="http://www.idpf.org/2007/opf">
		<item id="ch1" href="ch1.xhtml"/><item id="ch2" href="ch2.xhtml"/>
		<item id="ch3" href="ch3.xhtml"/><item id="ch4" href="ch4.xhtml"/>
		<item id="ch5" href="ch5.xhtml"/><item id="ch6" href="ch6.xhtml"/>
		<item id="ch7" href="ch7.xhtml"/><item id="ch8" href="ch8.xhtml"/>
		<item id="ch9" href="ch9.xhtml"/><item id="ch10" href="ch10.xhtml"/>
		<item id="ch11" href="ch11.xhtml"/><item id="ch12" href="ch12.xhtml"/>
		<item id="ch13" href="ch13.xhtml"/><item id="ch14" href="ch14.xhtml"/>
		<item id="ch15" href="ch15.xhtml"/><item id="ch16" href="ch16.xhtml"/>
		<item id="ch17" href="ch17.xhtml"/><item id="ch18" href="ch18.xhtml"/>
		<item id="ch19" href="ch19.xhtml"/><item id="ch20" href="ch20.xhtml"/>
		<item id="ch21" href="ch21.xhtml"/>
	</manifest>`
	ctx := xmlTestCtx(t)
	for i := 0; i < 20; i++ {
		root := parseTestXml(t, xmlStr)
		result, err := xmlFindAllAttrsImpl(ctx, []eval.Value{
			root,
			&eval.StringValue{Value: "item"},
			&eval.StringValue{Value: "href"},
		})
		if err != nil {
			t.Fatalf("run %d: unexpected error: %v", i, err)
		}
		list := result.(*eval.ListValue)
		if len(list.Elements) != 21 {
			t.Fatalf("run %d: expected 21 href attrs, got %d (nondeterministic)", i, len(list.Elements))
		}
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
