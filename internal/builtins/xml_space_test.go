package builtins

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// Regression coverage for ailang#646: a whitespace-only text node was dropped
// unconditionally, so OOXML separator runs (`<w:t xml:space="preserve"> </w:t>`)
// vanished and "plain bold italic" extracted as "plain bolditalic".
//
// The arms below are split by BRANCH rather than by scenario: each names the one
// line it exists to red, because `xml:space` handling is four decisions (match the
// attribute, honour "preserve", honour "default", inherit otherwise) plus one
// thread through each of the two recursive parsers, and a single end-to-end arm
// would pass with most of them broken.

// xmlTextsOf returns the Text-node contents directly under elem, in order.
func xmlTextsOf(t *testing.T, elem *eval.TaggedValue) []string {
	t.Helper()
	var out []string
	for _, c := range xmlGetChildren(t, elem) {
		tv, ok := c.(*eval.TaggedValue)
		if !ok || tv.CtorName != "Text" {
			continue
		}
		sv, ok := tv.Fields[0].(*eval.StringValue)
		if !ok {
			t.Fatalf("Text node field 0 is %T, want StringValue", tv.Fields[0])
		}
		out = append(out, sv.Value)
	}
	return out
}

// xmlParseOne parses doc with impl and returns the single root element.
func xmlParseOne(t *testing.T, doc string, tag string) *eval.TaggedValue {
	t.Helper()
	result, err := xmlParseImpl(xmlTestCtx(t), []eval.Value{&eval.StringValue{Value: doc}})
	if err != nil {
		t.Fatalf("xmlParseImpl(%q): %v", doc, err)
	}
	return xmlAssertElement(t, xmlAssertOk(t, result), tag)
}

// xmlFirstChildElem returns the first Element child of elem.
func xmlFirstChildElem(t *testing.T, elem *eval.TaggedValue, tag string) *eval.TaggedValue {
	t.Helper()
	for _, c := range xmlGetChildren(t, elem) {
		if tv, ok := c.(*eval.TaggedValue); ok && tv.CtorName == "Element" {
			return xmlAssertElement(t, tv, tag)
		}
	}
	t.Fatalf("no Element child <%s> under the given element", tag)
	return nil
}

// TestXmlSpace_PreserveKeepsWhitespaceOnlyText is the reported defect itself.
// It asserts the CONTENT (" "), not merely that a child exists, because a Text
// node carrying "" would satisfy a count-based assertion while still losing the
// separator that #646 is about.
func TestXmlSpace_PreserveKeepsWhitespaceOnlyText(t *testing.T) {
	root := xmlParseOne(t, `<r><t xml:space="preserve"> </t></r>`, "r")
	got := xmlTextsOf(t, xmlFirstChildElem(t, root, "t"))
	if len(got) != 1 || got[0] != " " {
		t.Fatalf("texts under <t xml:space=\"preserve\"> = %q, want [\" \"]", got)
	}
}

// TestXmlSpace_AbsentAttributeStillDropsWhitespace pins the DEFAULT, which the
// fix must not move: element-content XML is normally pretty-printed, and every
// existing caller of getChildren depends on formatting whitespace staying out of
// the tree.
func TestXmlSpace_AbsentAttributeStillDropsWhitespace(t *testing.T) {
	root := xmlParseOne(t, `<r><t> </t></r>`, "r")
	if got := xmlTextsOf(t, xmlFirstChildElem(t, root, "t")); len(got) != 0 {
		t.Fatalf("texts under <t> with no xml:space = %q, want none", got)
	}

	// The pretty-printed shape, which is the one that would regress loudly.
	pretty := xmlParseOne(t, "<r>\n  <a>x</a>\n  <b>y</b>\n</r>", "r")
	if got := xmlTextsOf(t, pretty); len(got) != 0 {
		t.Fatalf("texts directly under pretty-printed <r> = %q, want none", got)
	}
	if n := len(xmlGetChildren(t, pretty)); n != 2 {
		t.Fatalf("children of pretty-printed <r> = %d, want 2 (the elements only)", n)
	}
}

// TestXmlSpace_InheritedByDescendants pins the recursive thread: xml:space is
// declared once on an ancestor (XML 1.0 §2.10) and governs descendant content.
// Seeding preserve only from the element that carries the attribute would leave
// this arm red while the arm above stays green.
func TestXmlSpace_InheritedByDescendants(t *testing.T) {
	root := xmlParseOne(t, `<r xml:space="preserve"><t> </t></r>`, "r")
	got := xmlTextsOf(t, xmlFirstChildElem(t, root, "t"))
	if len(got) != 1 || got[0] != " " {
		t.Fatalf("texts under an inherited preserve scope = %q, want [\" \"]", got)
	}
}

// TestXmlSpace_DefaultOverridesInheritedPreserve pins the "default" branch,
// which is the only way back out of a preserve scope. Dropping it would leave
// every arm above green.
func TestXmlSpace_DefaultOverridesInheritedPreserve(t *testing.T) {
	root := xmlParseOne(t, `<r xml:space="preserve"><t xml:space="default"> </t></r>`, "r")
	if got := xmlTextsOf(t, xmlFirstChildElem(t, root, "t")); len(got) != 0 {
		t.Fatalf(`texts under xml:space="default" inside a preserve scope = %q, want none`, got)
	}

	// ...and re-entering preserve below that default works, so the value is
	// read per element rather than latched once.
	root2 := xmlParseOne(t, `<r xml:space="preserve"><t xml:space="default"><u xml:space="preserve"> </u></t></r>`, "r")
	u := xmlFirstChildElem(t, xmlFirstChildElem(t, root2, "t"), "u")
	if got := xmlTextsOf(t, u); len(got) != 1 || got[0] != " " {
		t.Fatalf("texts under a re-entered preserve scope = %q, want [\" \"]", got)
	}
}

// TestXmlSpace_UndefinedValueInherits pins the fallthrough. The spec defines
// exactly two values; a third is not an override, and the two directions are
// asserted together so that hardcoding either answer reds.
func TestXmlSpace_UndefinedValueInherits(t *testing.T) {
	outside := xmlParseOne(t, `<r><t xml:space="bogus"> </t></r>`, "r")
	if got := xmlTextsOf(t, xmlFirstChildElem(t, outside, "t")); len(got) != 0 {
		t.Fatalf("undefined xml:space value outside a preserve scope kept %q, want none", got)
	}

	inside := xmlParseOne(t, `<r xml:space="preserve"><t xml:space="bogus"> </t></r>`, "r")
	got := xmlTextsOf(t, xmlFirstChildElem(t, inside, "t"))
	if len(got) != 1 || got[0] != " " {
		t.Fatalf("undefined xml:space value inside a preserve scope = %q, want [\" \"]", got)
	}
}

// TestXmlSpace_UnprefixedSpaceAttributeIsNotXmlSpace pins the namespace half of
// the attribute match. A bare `space="preserve"` is an ordinary attribute in no
// namespace and must NOT arm preservation — matching on Local alone would pass
// every other arm in this file.
func TestXmlSpace_UnprefixedSpaceAttributeIsNotXmlSpace(t *testing.T) {
	root := xmlParseOne(t, `<r><t space="preserve"> </t></r>`, "r")
	if got := xmlTextsOf(t, xmlFirstChildElem(t, root, "t")); len(got) != 0 {
		t.Fatalf(`bare space="preserve" armed preservation (kept %q); it is not xml:space`, got)
	}
}

// TestXmlSpace_ParseWithLimitHonoursPreserve covers the second recursive parser.
// parseXmlChildrenLimited is a near-copy of parseXmlChildren, so a fix applied to
// one and not the other is the likeliest way this regresses; the arms above all
// run through the unlimited path only.
func TestXmlSpace_ParseWithLimitHonoursPreserve(t *testing.T) {
	result, err := xmlParseWithLimitImpl(xmlTestCtx(t), []eval.Value{
		&eval.StringValue{Value: `<r><t xml:space="preserve"> </t></r>`},
		&eval.IntValue{Value: 100},
	})
	if err != nil {
		t.Fatalf("xmlParseWithLimitImpl: %v", err)
	}
	root := xmlAssertElement(t, xmlAssertOk(t, result), "r")
	got := xmlTextsOf(t, xmlFirstChildElem(t, root, "t"))
	if len(got) != 1 || got[0] != " " {
		t.Fatalf("parseWithLimit texts under preserve = %q, want [\" \"]", got)
	}
}

// TestXmlSpace_PreservedTextCountsAgainstNodeLimit pins the interaction between
// the two: a preserved whitespace node is a node, so it must be counted and
// limit-checked like any other. Emitting it outside the counter would let a
// document defeat maxNodes with whitespace.
func TestXmlSpace_PreservedTextCountsAgainstNodeLimit(t *testing.T) {
	doc := `<r xml:space="preserve"><a> </a><b> </b><c> </c></r>`
	// r,a,text,b,text,c,text = 7 nodes; a limit of 6 must refuse.
	result, err := xmlParseWithLimitImpl(xmlTestCtx(t), []eval.Value{
		&eval.StringValue{Value: doc},
		&eval.IntValue{Value: 6},
	})
	if err != nil {
		t.Fatalf("xmlParseWithLimitImpl: %v", err)
	}
	xmlAssertErr(t, result, "node limit exceeded")

	// Control: the same document parses under a limit that accommodates them,
	// so the arm above fails for the count and not for the parse.
	ok, err := xmlParseWithLimitImpl(xmlTestCtx(t), []eval.Value{
		&eval.StringValue{Value: doc},
		&eval.IntValue{Value: 7},
	})
	if err != nil {
		t.Fatalf("xmlParseWithLimitImpl (control): %v", err)
	}
	xmlAssertOk(t, ok)
}

// TestXmlSpace_DocxSeparatorRunSurvives is the reporter's own scenario, end to
// end: Word splits runs at every formatting boundary and the separating space
// very commonly lands in a run of its own.
func TestXmlSpace_DocxSeparatorRunSurvives(t *testing.T) {
	doc := `<w:p xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:r><w:t xml:space="preserve">plain </w:t></w:r>` +
		`<w:r><w:t>bold</w:t></w:r>` +
		`<w:r><w:t xml:space="preserve"> </w:t></w:r>` +
		`<w:r><w:t>italic</w:t></w:r>` +
		`</w:p>`
	result, err := xmlGetTextImpl(xmlTestCtx(t), []eval.Value{
		func() eval.Value {
			r, err := xmlParseImpl(xmlTestCtx(t), []eval.Value{&eval.StringValue{Value: doc}})
			if err != nil {
				t.Fatalf("xmlParseImpl: %v", err)
			}
			return xmlAssertOk(t, r)
		}(),
	})
	if err != nil {
		t.Fatalf("xmlGetTextImpl: %v", err)
	}
	sv, ok := result.(*eval.StringValue)
	if !ok {
		t.Fatalf("getText returned %T, want StringValue", result)
	}
	if sv.Value != "plain bold italic" {
		t.Fatalf("DOCX run extraction = %q, want %q", sv.Value, "plain bold italic")
	}
}

// TestXmlSpace_StreamingPathsHonourPreserve covers the three scanners that build
// a matched subtree without a full tree — parseElements, parseFold, parseFoldStep.
// They seed preservation from the MATCHED element itself, because a streaming scan
// has no ancestor context by construction (it passes a nil prefixMap for the same
// reason), so their seed is a third code path that the two recursive parsers'
// arms do not reach: the drill that added this arm found the parseElements seed
// surviving every other assertion in this file.
func TestXmlSpace_StreamingPathsHonourPreserve(t *testing.T) {
	const doc = `<r><t xml:space="preserve"> </t></r>`

	t.Run("parseElements", func(t *testing.T) {
		result, err := xmlParseElementsImpl(xmlTestCtx(t), []eval.Value{
			&eval.StringValue{Value: doc},
			&eval.StringValue{Value: "t"},
			&eval.IntValue{Value: 10},
		})
		if err != nil {
			t.Fatalf("xmlParseElementsImpl: %v", err)
		}
		list, ok := xmlAssertOk(t, result).(*eval.ListValue)
		if !ok {
			t.Fatalf("parseElements payload is not a list")
		}
		if len(list.Elements) != 1 {
			t.Fatalf("parseElements matched %d elements, want 1", len(list.Elements))
		}
		el := xmlAssertElement(t, list.Elements[0], "t")
		if got := xmlTextsOf(t, el); len(got) != 1 || got[0] != " " {
			t.Fatalf("parseElements texts under preserve = %q, want [\" \"]", got)
		}
	})

	t.Run("parseFold", func(t *testing.T) {
		result, err := xmlParseFoldImpl(xmlFoldTestCtx(), []eval.Value{
			&eval.StringValue{Value: doc},
			&eval.StringValue{Value: "t"},
			&eval.ListValue{Elements: nil},
			xmlFoldHandler(),
		})
		if err != nil {
			t.Fatalf("xmlParseFoldImpl: %v", err)
		}
		list, ok := xmlAssertOk(t, result).(*eval.ListValue)
		if !ok || len(list.Elements) != 1 {
			t.Fatalf("parseFold accumulated %v, want one entry", xmlAssertOk(t, result))
		}
		sv, ok := list.Elements[0].(*eval.StringValue)
		if !ok || sv.Value != " " {
			t.Fatalf("parseFold text under preserve = %#v, want \" \"", list.Elements[0])
		}
	})

	t.Run("parseFoldStep", func(t *testing.T) {
		result, err := xmlParseFoldStepImpl(xmlFoldTestCtx(), []eval.Value{
			&eval.StringValue{Value: doc},
			&eval.StringValue{Value: "t"},
			&eval.ListValue{Elements: nil},
			xmlFoldStepContinueHandler(),
		})
		if err != nil {
			t.Fatalf("xmlParseFoldStepImpl: %v", err)
		}
		list, ok := xmlAssertOk(t, result).(*eval.ListValue)
		if !ok || len(list.Elements) != 1 {
			t.Fatalf("parseFoldStep accumulated %v, want one entry", xmlAssertOk(t, result))
		}
		sv, ok := list.Elements[0].(*eval.StringValue)
		if !ok || sv.Value != " " {
			t.Fatalf("parseFoldStep text under preserve = %#v, want \" \"", list.Elements[0])
		}
	})
}

// xmlFoldStepContinueHandler is xmlFoldHandler wrapped in Continue, so the
// parseFoldStep arm observes the same accumulated text as the parseFold arm.
func xmlFoldStepContinueHandler() eval.Value {
	inner := xmlFoldHandler().(*eval.BuiltinFunction)
	return &eval.BuiltinFunction{
		Name: "test_foldstep_continue_handler",
		Fn: func(args []eval.Value) (eval.Value, error) {
			acc, err := inner.Fn(args)
			if err != nil {
				return nil, err
			}
			return &eval.TaggedValue{
				TypeName: "FoldStep",
				CtorName: "Continue",
				Fields:   []eval.Value{acc},
			}, nil
		},
	}
}
