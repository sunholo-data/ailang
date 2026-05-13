package builtins

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// Reuses xmlTestCtx / xmlAssertOk / xmlAssertErr / xmlAssertElement /
// xmlGetChildren / xmlGetAttrs from xml_test.go — the std/html parser
// emits the same XmlNode shape as std/xml, so the helpers apply unchanged.

// findDescendantElement walks the tree recursively, returning the first
// Element with the given tag in document order.
func findDescendantElement(t *testing.T, elem *eval.TaggedValue, tag string) *eval.TaggedValue {
	t.Helper()
	res := findDescendantElementOpt(elem, tag)
	if res == nil {
		t.Fatalf("no <%s> descendant found", tag)
	}
	return res
}

func findDescendantElementOpt(elem *eval.TaggedValue, tag string) *eval.TaggedValue {
	if elem.CtorName != "Element" {
		return nil
	}
	sv, _ := elem.Fields[0].(*eval.StringValue)
	if sv != nil && sv.Value == tag {
		return elem
	}
	lv, ok := elem.Fields[2].(*eval.ListValue)
	if !ok {
		return nil
	}
	for _, c := range lv.Elements {
		tv, ok := c.(*eval.TaggedValue)
		if !ok {
			continue
		}
		if got := findDescendantElementOpt(tv, tag); got != nil {
			return got
		}
	}
	return nil
}

// attrValue returns the value of the named attribute on an Element, or "" + false.
func attrValue(t *testing.T, elem *eval.TaggedValue, name string) (string, bool) {
	t.Helper()
	for _, av := range xmlGetAttrs(t, elem) {
		rv, ok := av.(*eval.RecordValue)
		if !ok {
			continue
		}
		nameVal, _ := rv.Fields["name"].(*eval.StringValue)
		valVal, _ := rv.Fields["value"].(*eval.StringValue)
		if nameVal == nil || valVal == nil {
			continue
		}
		if nameVal.Value == name {
			return valVal.Value, true
		}
	}
	return "", false
}

// ============================================================================
// _html_parse tests
// ============================================================================

func TestHtmlParse_SimpleParagraph(t *testing.T) {
	ctx := xmlTestCtx(t)
	args := []eval.Value{&eval.StringValue{Value: "<p>hello</p>"}}
	result, err := htmlParseImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inner := xmlAssertOk(t, result)
	root := xmlAssertElement(t, inner, "html")
	p := findDescendantElement(t, root, "p")
	children := xmlGetChildren(t, p)
	if len(children) != 1 {
		t.Fatalf("expected 1 child in <p>, got %d", len(children))
	}
	textTV, ok := children[0].(*eval.TaggedValue)
	if !ok || textTV.CtorName != "Text" {
		t.Fatalf("expected Text child, got %v", children[0])
	}
}

func TestHtmlParse_UnclosedParagraphs(t *testing.T) {
	// Real HTML5: <p>a<p>b is two sibling paragraphs, not nested.
	ctx := xmlTestCtx(t)
	args := []eval.Value{&eval.StringValue{Value: "<p>a<p>b"}}
	result, err := htmlParseImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inner := xmlAssertOk(t, result)
	root := xmlAssertElement(t, inner, "html")
	body := findDescendantElement(t, root, "body")

	var pCount int
	for _, c := range xmlGetChildren(t, body) {
		tv, ok := c.(*eval.TaggedValue)
		if !ok || tv.CtorName != "Element" {
			continue
		}
		sv, _ := tv.Fields[0].(*eval.StringValue)
		if sv != nil && sv.Value == "p" {
			pCount++
		}
	}
	if pCount != 2 {
		t.Fatalf("expected 2 sibling <p> elements, got %d", pCount)
	}
}

func TestHtmlParse_BooleanAttribute(t *testing.T) {
	ctx := xmlTestCtx(t)
	args := []eval.Value{&eval.StringValue{Value: "<input disabled>"}}
	result, err := htmlParseImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inner := xmlAssertOk(t, result)
	root := xmlAssertElement(t, inner, "html")
	input := findDescendantElement(t, root, "input")
	val, ok := attrValue(t, input, "disabled")
	if !ok {
		t.Fatal("expected 'disabled' attribute to be present")
	}
	if val != "" {
		t.Fatalf("expected boolean attr value '', got %q", val)
	}
}

func TestHtmlParse_LowerCasedTag(t *testing.T) {
	// HTML5 normalizes tag names to lowercase.
	ctx := xmlTestCtx(t)
	args := []eval.Value{&eval.StringValue{Value: "<DIV>x</DIV>"}}
	result, err := htmlParseImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inner := xmlAssertOk(t, result)
	root := xmlAssertElement(t, inner, "html")
	// Must find a "div", not "DIV".
	_ = findDescendantElement(t, root, "div")
}

func TestHtmlParse_CommentPreserved(t *testing.T) {
	ctx := xmlTestCtx(t)
	args := []eval.Value{&eval.StringValue{Value: "<p><!-- note --></p>"}}
	result, err := htmlParseImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inner := xmlAssertOk(t, result)
	root := xmlAssertElement(t, inner, "html")
	p := findDescendantElement(t, root, "p")

	var foundComment bool
	for _, c := range xmlGetChildren(t, p) {
		tv, ok := c.(*eval.TaggedValue)
		if !ok {
			continue
		}
		if tv.CtorName == "Comment" {
			sv, _ := tv.Fields[0].(*eval.StringValue)
			if sv != nil && strings.Contains(sv.Value, "note") {
				foundComment = true
			}
		}
	}
	if !foundComment {
		t.Fatal("expected Comment child preserved inside <p>")
	}
}

func TestHtmlParse_ScriptPassthrough(t *testing.T) {
	// <script> contents are passed through as Text — caller filters if needed.
	ctx := xmlTestCtx(t)
	args := []eval.Value{&eval.StringValue{Value: "<script>alert(1)</script>"}}
	result, err := htmlParseImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inner := xmlAssertOk(t, result)
	root := xmlAssertElement(t, inner, "html")
	script := findDescendantElement(t, root, "script")
	children := xmlGetChildren(t, script)
	if len(children) == 0 {
		t.Fatal("expected text inside <script>")
	}
	textTV, ok := children[0].(*eval.TaggedValue)
	if !ok || textTV.CtorName != "Text" {
		t.Fatalf("expected Text child, got %v", children[0])
	}
	sv, _ := textTV.Fields[0].(*eval.StringValue)
	if sv == nil || !strings.Contains(sv.Value, "alert(1)") {
		t.Fatalf("expected 'alert(1)' in script text, got %v", sv)
	}
}

func TestHtmlParse_NamespacedTag(t *testing.T) {
	// Word exports use namespace prefixes like <o:p>. net/html keeps the
	// prefix in the local name.
	ctx := xmlTestCtx(t)
	args := []eval.Value{&eval.StringValue{Value: "<o:p>x</o:p>"}}
	result, err := htmlParseImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inner := xmlAssertOk(t, result)
	root := xmlAssertElement(t, inner, "html")
	// HTML5 lowercases, but the colon-prefix is preserved in the data field.
	if findDescendantElementOpt(root, "o:p") == nil {
		// Some net/html versions strip the namespace prefix entirely.
		// Accept either "o:p" or "p" — what matters is the parser
		// did not panic and produced an Element.
		if findDescendantElementOpt(root, "p") == nil {
			t.Fatal("expected <o:p> or <p> element, found neither")
		}
	}
}

func TestHtmlParse_OversizedInput(t *testing.T) {
	ctx := xmlTestCtx(t)
	big := strings.Repeat("a", htmlMaxInputSize+1)
	args := []eval.Value{&eval.StringValue{Value: big}}
	result, err := htmlParseImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	xmlAssertErr(t, result, "too large")
}

func TestHtmlParse_EmptyInput(t *testing.T) {
	// HTML5 parser synthesizes <html><head></head><body></body></html> for
	// empty input. We accept any Ok result with <html> root.
	ctx := xmlTestCtx(t)
	args := []eval.Value{&eval.StringValue{Value: ""}}
	result, err := htmlParseImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inner := xmlAssertOk(t, result)
	_ = xmlAssertElement(t, inner, "html")
}

func TestHtmlParse_RejectsWrongArgType(t *testing.T) {
	ctx := xmlTestCtx(t)
	args := []eval.Value{&eval.IntValue{Value: 42}}
	_, err := htmlParseImpl(ctx, args)
	if err == nil {
		t.Fatal("expected error for non-string argument")
	}
}

// ============================================================================
// _html_parseFragment tests
// ============================================================================

func TestHtmlParseFragment_MultipleRoots(t *testing.T) {
	ctx := xmlTestCtx(t)
	args := []eval.Value{&eval.StringValue{Value: "<p>a</p><p>b</p>"}}
	result, err := htmlParseFragmentImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inner := xmlAssertOk(t, result)
	lv, ok := inner.(*eval.ListValue)
	if !ok {
		t.Fatalf("expected ListValue, got %T", inner)
	}
	// Count <p> elements — fragment may include other nodes (whitespace text).
	var pCount int
	for _, n := range lv.Elements {
		tv, ok := n.(*eval.TaggedValue)
		if !ok || tv.CtorName != "Element" {
			continue
		}
		sv, _ := tv.Fields[0].(*eval.StringValue)
		if sv != nil && sv.Value == "p" {
			pCount++
		}
	}
	if pCount != 2 {
		t.Fatalf("expected 2 <p> top-level fragments, got %d (total nodes: %d)", pCount, len(lv.Elements))
	}
}

func TestHtmlParseFragment_OversizedInput(t *testing.T) {
	ctx := xmlTestCtx(t)
	big := strings.Repeat("a", htmlMaxInputSize+1)
	args := []eval.Value{&eval.StringValue{Value: big}}
	result, err := htmlParseFragmentImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	xmlAssertErr(t, result, "too large")
}
