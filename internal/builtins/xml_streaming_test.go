package builtins

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/eval"
)

// ============================================================================
// Tests for _xml_parseElements (streaming element extraction)
// ============================================================================

func TestXmlParseElements_Basic(t *testing.T) {
	xml := `<root><item>one</item><item>two</item><item>three</item><item>four</item><item>five</item></root>`
	ctx := xmlTestCtx(t)

	result, err := xmlParseElementsImpl(ctx, []eval.Value{
		&eval.StringValue{Value: xml},
		&eval.StringValue{Value: "item"},
		&eval.IntValue{Value: 3},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := xmlAssertOk(t, result)
	list, ok := inner.(*eval.ListValue)
	if !ok {
		t.Fatalf("expected ListValue, got %T", inner)
	}
	if len(list.Elements) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(list.Elements))
	}

	// Verify text content of extracted elements
	for i, expected := range []string{"one", "two", "three"} {
		tv := list.Elements[i].(*eval.TaggedValue)
		var buf strings.Builder
		collectText(tv, &buf)
		if buf.String() != expected {
			t.Errorf("element %d: expected %q, got %q", i, expected, buf.String())
		}
	}
}

func TestXmlParseElements_FewerThanLimit(t *testing.T) {
	xml := `<root><item>a</item><item>b</item></root>`
	ctx := xmlTestCtx(t)

	result, err := xmlParseElementsImpl(ctx, []eval.Value{
		&eval.StringValue{Value: xml},
		&eval.StringValue{Value: "item"},
		&eval.IntValue{Value: 100},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := xmlAssertOk(t, result)
	list := inner.(*eval.ListValue)
	if len(list.Elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(list.Elements))
	}
}

func TestXmlParseElements_NoMatch(t *testing.T) {
	xml := `<root><item>hello</item></root>`
	ctx := xmlTestCtx(t)

	result, err := xmlParseElementsImpl(ctx, []eval.Value{
		&eval.StringValue{Value: xml},
		&eval.StringValue{Value: "nonexistent"},
		&eval.IntValue{Value: 10},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := xmlAssertOk(t, result)
	list := inner.(*eval.ListValue)
	if len(list.Elements) != 0 {
		t.Fatalf("expected 0 elements, got %d", len(list.Elements))
	}
}

func TestXmlParseElements_ZeroLimit(t *testing.T) {
	xml := `<root><item>hello</item></root>`
	ctx := xmlTestCtx(t)

	result, err := xmlParseElementsImpl(ctx, []eval.Value{
		&eval.StringValue{Value: xml},
		&eval.StringValue{Value: "item"},
		&eval.IntValue{Value: 0},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := xmlAssertOk(t, result)
	list := inner.(*eval.ListValue)
	if len(list.Elements) != 0 {
		t.Fatalf("expected 0 elements for zero limit, got %d", len(list.Elements))
	}
}

func TestXmlParseElements_PreservesAttrsAndChildren(t *testing.T) {
	xml := `<root><row id="1"><c t="s">val1</c><c t="n">42</c></row><row id="2"><c>val2</c></row></root>`
	ctx := xmlTestCtx(t)

	result, err := xmlParseElementsImpl(ctx, []eval.Value{
		&eval.StringValue{Value: xml},
		&eval.StringValue{Value: "row"},
		&eval.IntValue{Value: 10},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := xmlAssertOk(t, result)
	list := inner.(*eval.ListValue)
	if len(list.Elements) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(list.Elements))
	}

	// Check first row has id="1" and 2 children
	row1 := list.Elements[0].(*eval.TaggedValue)
	if row1.CtorName != "Element" {
		t.Fatalf("expected Element, got %s", row1.CtorName)
	}
	// Check tag
	tag := row1.Fields[0].(*eval.StringValue).Value
	if tag != "row" {
		t.Errorf("expected tag 'row', got %q", tag)
	}
	// Check attrs
	attrs := row1.Fields[1].(*eval.ListValue)
	if len(attrs.Elements) != 1 {
		t.Fatalf("expected 1 attr, got %d", len(attrs.Elements))
	}
	// Check children
	children := row1.Fields[2].(*eval.ListValue)
	if len(children.Elements) != 2 {
		t.Errorf("expected 2 children in row 1, got %d", len(children.Elements))
	}
}

func TestXmlParseElements_LargeDoc(t *testing.T) {
	// Generate XML with 10000 items, extract only 100
	var sb strings.Builder
	sb.WriteString("<root>")
	for i := 0; i < 10000; i++ {
		fmt.Fprintf(&sb, `<item id="%d">value%d</item>`, i, i)
	}
	sb.WriteString("</root>")

	ctx := xmlTestCtx(t)
	result, err := xmlParseElementsImpl(ctx, []eval.Value{
		&eval.StringValue{Value: sb.String()},
		&eval.StringValue{Value: "item"},
		&eval.IntValue{Value: 100},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := xmlAssertOk(t, result)
	list := inner.(*eval.ListValue)
	if len(list.Elements) != 100 {
		t.Fatalf("expected 100 elements, got %d", len(list.Elements))
	}

	// Verify first element has id="0"
	first := list.Elements[0].(*eval.TaggedValue)
	attrList := first.Fields[1].(*eval.ListValue)
	rec := attrList.Elements[0].(*eval.RecordValue)
	idVal := rec.Fields["value"].(*eval.StringValue).Value
	if idVal != "0" {
		t.Errorf("first element id: expected '0', got %q", idVal)
	}

	// Verify last element has id="99"
	last := list.Elements[99].(*eval.TaggedValue)
	lastAttrList := last.Fields[1].(*eval.ListValue)
	lastRec := lastAttrList.Elements[0].(*eval.RecordValue)
	lastIdVal := lastRec.Fields["value"].(*eval.StringValue).Value
	if lastIdVal != "99" {
		t.Errorf("last element id: expected '99', got %q", lastIdVal)
	}
}

func TestXmlParseElements_SkipsNonMatching(t *testing.T) {
	// Verify that non-matching elements with deep subtrees are properly skipped
	xml := `<root><header><title>Hello</title><meta>stuff</meta></header><row>data1</row><footer>end</footer><row>data2</row></root>`
	ctx := xmlTestCtx(t)

	result, err := xmlParseElementsImpl(ctx, []eval.Value{
		&eval.StringValue{Value: xml},
		&eval.StringValue{Value: "row"},
		&eval.IntValue{Value: 10},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := xmlAssertOk(t, result)
	list := inner.(*eval.ListValue)
	if len(list.Elements) != 2 {
		t.Fatalf("expected 2 row elements, got %d", len(list.Elements))
	}

	var buf strings.Builder
	collectText(list.Elements[0], &buf)
	if buf.String() != "data1" {
		t.Errorf("first row: expected 'data1', got %q", buf.String())
	}
}

// ============================================================================
// Tests for _xml_parseWithLimit (fail-fast node count limit)
// ============================================================================

func TestXmlParseWithLimit_WithinLimit(t *testing.T) {
	xml := `<root><a>1</a><b>2</b></root>`
	ctx := xmlTestCtx(t)

	result, err := xmlParseWithLimitImpl(ctx, []eval.Value{
		&eval.StringValue{Value: xml},
		&eval.IntValue{Value: 100},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := xmlAssertOk(t, result)
	tv := inner.(*eval.TaggedValue)
	if tv.CtorName != "Element" {
		t.Fatalf("expected Element, got %s", tv.CtorName)
	}
	tag := tv.Fields[0].(*eval.StringValue).Value
	if tag != "root" {
		t.Errorf("expected tag 'root', got %q", tag)
	}
}

func TestXmlParseWithLimit_ExceedsLimit(t *testing.T) {
	// <root> has 3 elements + 3 text nodes = 7 nodes total (including root)
	xml := `<root><a>1</a><b>2</b><c>3</c></root>`
	ctx := xmlTestCtx(t)

	result, err := xmlParseWithLimitImpl(ctx, []eval.Value{
		&eval.StringValue{Value: xml},
		&eval.IntValue{Value: 3}, // Only allow 3 nodes
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be Err
	tv, ok := result.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", result)
	}
	if tv.CtorName != "Err" {
		t.Fatalf("expected Err (node limit), got %s", tv.CtorName)
	}
	errMsg := tv.Fields[0].(*eval.StringValue).Value
	if !strings.Contains(errMsg, "node limit exceeded") {
		t.Errorf("expected 'node limit exceeded' in error, got %q", errMsg)
	}
}

func TestXmlParseWithLimit_ExactLimit(t *testing.T) {
	// <root><a>1</a></root> = root(1) + a(2) + text(3) = 3 nodes
	xml := `<root><a>1</a></root>`
	ctx := xmlTestCtx(t)

	result, err := xmlParseWithLimitImpl(ctx, []eval.Value{
		&eval.StringValue{Value: xml},
		&eval.IntValue{Value: 3},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should succeed — exactly at the limit
	inner := xmlAssertOk(t, result)
	tv := inner.(*eval.TaggedValue)
	if tv.CtorName != "Element" {
		t.Fatalf("expected Element, got %s", tv.CtorName)
	}
}

func TestXmlParseWithLimit_LargeDocFails(t *testing.T) {
	// Generate 1000 elements, limit to 100
	var sb strings.Builder
	sb.WriteString("<root>")
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&sb, "<item>%d</item>", i)
	}
	sb.WriteString("</root>")

	ctx := xmlTestCtx(t)
	result, err := xmlParseWithLimitImpl(ctx, []eval.Value{
		&eval.StringValue{Value: sb.String()},
		&eval.IntValue{Value: 100},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tv := result.(*eval.TaggedValue)
	if tv.CtorName != "Err" {
		t.Fatalf("expected Err for large doc with small limit, got %s", tv.CtorName)
	}
}
