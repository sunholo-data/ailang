package builtins

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// Coverage for the second half of ailang#646: the prefix map is built from the
// xmlns declarations in scope, and the "xml" prefix is bound BY DEFINITION and
// must not be declared (XML Namespaces §3). So no document can put it in the
// map, resolveTagName found nothing, and it fell through to the bare local name.
//
// The consequence is not cosmetic: xml:space arrived as "space", indistinguishable
// from an ordinary unprefixed attribute of that name, getAttr(node, "xml:space")
// returned None on a document that plainly has one, and serialize round-tripped
// <t xml:space="preserve"/> out as <t space="preserve"/> — silently changing what
// the document means.

func xmlAttrNames(t *testing.T, elem *eval.TaggedValue) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, a := range xmlGetAttrs(t, elem) {
		rv, ok := a.(*eval.RecordValue)
		if !ok {
			t.Fatalf("attr is %T, want RecordValue", a)
		}
		name, ok := rv.Fields["name"].(*eval.StringValue)
		if !ok {
			t.Fatalf("attr name is %T, want StringValue", rv.Fields["name"])
		}
		val, ok := rv.Fields["value"].(*eval.StringValue)
		if !ok {
			t.Fatalf("attr value is %T, want StringValue", rv.Fields["value"])
		}
		out[name.Value] = val.Value
	}
	return out
}

// TestXmlImplicitPrefix_XmlNamespaceKeepsItsPrefix pins the mapping itself, on
// three of the four reserved attributes, so that a fix special-casing only
// xml:space (the one #646 needed) still reds here.
func TestXmlImplicitPrefix_XmlNamespaceKeepsItsPrefix(t *testing.T) {
	root := xmlParseOne(t, `<r><t xml:space="preserve" xml:lang="en-GB" xml:id="a1" plain="p">x</t></r>`, "r")
	attrs := xmlAttrNames(t, xmlFirstChildElem(t, root, "t"))

	for name, want := range map[string]string{
		"xml:space": "preserve",
		"xml:lang":  "en-GB",
		"xml:id":    "a1",
		"plain":     "p", // control: an attribute in no namespace is untouched
	} {
		if got, ok := attrs[name]; !ok || got != want {
			t.Fatalf("attr %q = %q (present=%v), want %q; full attr set: %v", name, got, ok, want, attrs)
		}
	}
	// The stripped spelling must be gone, not merely accompanied — otherwise a
	// caller written against the old behaviour keeps working and the two names
	// diverge silently.
	if _, ok := attrs["space"]; ok {
		t.Fatalf("bare %q still present alongside %q: %v", "space", "xml:space", attrs)
	}
}

// TestXmlImplicitPrefix_DeclaredPrefixesUnaffected is the negative control for
// the mapping: ordinary declared prefixes must resolve exactly as before, and an
// attribute in a declared namespace must not be rewritten to xml:.
func TestXmlImplicitPrefix_DeclaredPrefixesUnaffected(t *testing.T) {
	root := xmlParseOne(t,
		`<w:p xmlns:w="urn:w"><w:t w:val="1" xml:space="preserve"> </w:t></w:p>`, "w:p")
	wt := xmlFirstChildElem(t, root, "w:t")
	attrs := xmlAttrNames(t, wt)
	if got, ok := attrs["w:val"]; !ok || got != "1" {
		t.Fatalf("declared-prefix attr w:val = %q (present=%v), want \"1\"; full attr set: %v", got, ok, attrs)
	}
	if got, ok := attrs["xml:space"]; !ok || got != "preserve" {
		t.Fatalf("xml:space alongside a declared prefix = %q (present=%v), want \"preserve\"", got, ok)
	}
}

// TestXmlImplicitPrefix_GetAttrFindsXmlSpace pins the accessor a caller actually
// reaches for. Without it a user cannot observe the condition the parser is now
// acting on, which is the seam that makes the #646 fix legible.
func TestXmlImplicitPrefix_GetAttrFindsXmlSpace(t *testing.T) {
	wt := xmlFirstChildElem(t, xmlParseOne(t, `<r><t xml:space="preserve"> </t></r>`, "r"), "t")

	got, err := xmlGetAttrImpl(xmlTestCtx(t), []eval.Value{wt, &eval.StringValue{Value: "xml:space"}})
	if err != nil {
		t.Fatalf("xmlGetAttrImpl: %v", err)
	}
	sv, ok := xmlAssertSome(t, got).(*eval.StringValue)
	if !ok {
		t.Fatalf("getAttr payload is %T, want StringValue", got)
	}
	if sv.Value != "preserve" {
		t.Fatalf(`getAttr(t, "xml:space") = %q, want "preserve"`, sv.Value)
	}

	// The old spelling must no longer answer.
	none, err := xmlGetAttrImpl(xmlTestCtx(t), []eval.Value{wt, &eval.StringValue{Value: "space"}})
	if err != nil {
		t.Fatalf("xmlGetAttrImpl (bare name): %v", err)
	}
	xmlAssertNone(t, none)
}

// TestXmlImplicitPrefix_SerializeRoundTrip is the artifact-level arm: the two
// halves of this fix are only worth anything together, since a preserved space
// serialized back out under a rewritten attribute name no longer says to the
// next reader what the source said.
func TestXmlImplicitPrefix_SerializeRoundTrip(t *testing.T) {
	const src = `<r><t xml:space="preserve"> </t></r>`
	root := xmlParseOne(t, src, "r")

	out, err := xmlSerializeImpl(xmlTestCtx(t), []eval.Value{root})
	if err != nil {
		t.Fatalf("xmlSerializeImpl: %v", err)
	}
	sv, ok := out.(*eval.StringValue)
	if !ok {
		t.Fatalf("serialize returned %T, want StringValue", out)
	}
	if sv.Value != src {
		t.Fatalf("serialize round-trip = %q, want %q", sv.Value, src)
	}
}
