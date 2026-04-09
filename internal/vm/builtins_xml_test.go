package vm

import (
	"testing"

	"github.com/sunholo/ailang/internal/bytecode"
	"github.com/sunholo/ailang/internal/eval"
)

// ---------------------------------------------------------------------------
// Converter tests
// ---------------------------------------------------------------------------

func TestXmlNodeToBytecode_Text(t *testing.T) {
	for range 20 {
		ev := &eval.TaggedValue{
			ModulePath: "std/xml", TypeName: "XmlNode", CtorName: "Text",
			Fields: []eval.Value{&eval.StringValue{Value: "hello"}},
		}
		bc, err := xmlNodeToBytecode(ev)
		if err != nil {
			t.Fatalf("xmlNodeToBytecode(Text): %v", err)
		}
		if bc.Tag != bytecode.TagADT {
			t.Fatalf("expected ADT, got %s", bc.Tag)
		}
		adt := bc.AsADT()
		if adt.Tag != xmlNodeTagText {
			t.Fatalf("expected tag %d, got %d", xmlNodeTagText, adt.Tag)
		}
		if adt.Fields[0].AsString() != "hello" {
			t.Fatalf("expected hello, got %s", adt.Fields[0].AsString())
		}
	}
}

func TestXmlNodeToBytecode_Element(t *testing.T) {
	for range 20 {
		ev := &eval.TaggedValue{
			ModulePath: "std/xml", TypeName: "XmlNode", CtorName: "Element",
			Fields: []eval.Value{
				&eval.StringValue{Value: "div"},
				&eval.ListValue{Elements: []eval.Value{
					&eval.RecordValue{Fields: map[string]eval.Value{
						"name":  &eval.StringValue{Value: "class"},
						"value": &eval.StringValue{Value: "main"},
					}},
				}},
				&eval.ListValue{Elements: []eval.Value{
					&eval.TaggedValue{
						ModulePath: "std/xml", TypeName: "XmlNode", CtorName: "Text",
						Fields: []eval.Value{&eval.StringValue{Value: "hi"}},
					},
				}},
			},
		}
		bc, err := xmlNodeToBytecode(ev)
		if err != nil {
			t.Fatalf("xmlNodeToBytecode(Element): %v", err)
		}
		adt := bc.AsADT()
		if adt.Tag != xmlNodeTagElement {
			t.Fatalf("expected Element tag 0, got %d", adt.Tag)
		}
		if adt.Fields[0].AsString() != "div" {
			t.Fatalf("tag name: expected div, got %s", adt.Fields[0].AsString())
		}
		// Check attrs
		attrs := adt.Fields[1].AsList()
		if len(attrs) != 1 {
			t.Fatalf("expected 1 attr, got %d", len(attrs))
		}
		rec := attrs[0].AsRecord()
		// Sorted: name < value
		if rec[0].Name != "name" || rec[0].Value.AsString() != "class" {
			t.Fatalf("attr name: got %s=%s", rec[0].Name, rec[0].Value.AsString())
		}
		if rec[1].Name != "value" || rec[1].Value.AsString() != "main" {
			t.Fatalf("attr value: got %s=%s", rec[1].Name, rec[1].Value.AsString())
		}
		// Check children
		children := adt.Fields[2].AsList()
		if len(children) != 1 {
			t.Fatalf("expected 1 child, got %d", len(children))
		}
		childADT := children[0].AsADT()
		if childADT.Tag != xmlNodeTagText {
			t.Fatalf("child: expected Text, got tag %d", childADT.Tag)
		}
	}
}

func TestBytecodeToXmlNode_Roundtrip(t *testing.T) {
	for range 20 {
		// Build a bytecode Element, convert to eval, convert back.
		original := bytecode.NewADT(xmlNodeTagElement, []bytecode.Value{
			bytecode.NewString("p"),
			bytecode.NewList([]bytecode.Value{
				bytecode.NewRecord([]bytecode.RecordField{
					{Name: "name", Value: bytecode.NewString("id")},
					{Name: "value", Value: bytecode.NewString("x1")},
				}),
			}),
			bytecode.NewList([]bytecode.Value{
				bytecode.NewADT(xmlNodeTagText, []bytecode.Value{bytecode.NewString("text")}),
			}),
		})
		ev, err := bytecodeToXmlNode(original)
		if err != nil {
			t.Fatalf("bytecodeToXmlNode: %v", err)
		}
		roundtrip, err := xmlNodeToBytecode(ev)
		if err != nil {
			t.Fatalf("xmlNodeToBytecode roundtrip: %v", err)
		}
		// Verify structure matches
		rt := roundtrip.AsADT()
		if rt.Tag != xmlNodeTagElement {
			t.Fatalf("roundtrip tag: expected 0, got %d", rt.Tag)
		}
		if rt.Fields[0].AsString() != "p" {
			t.Fatalf("roundtrip tag name: expected p, got %s", rt.Fields[0].AsString())
		}
	}
}

// ---------------------------------------------------------------------------
// M1: Constructor tests
// ---------------------------------------------------------------------------

func TestBuiltinXmlText(t *testing.T) {
	for range 20 {
		result, err := builtinXmlText([]bytecode.Value{bytecode.NewString("hello")})
		if err != nil {
			t.Fatalf("builtinXmlText: %v", err)
		}
		adt := result.AsADT()
		if adt.Tag != xmlNodeTagText {
			t.Fatalf("expected Text tag, got %d", adt.Tag)
		}
		if adt.Fields[0].AsString() != "hello" {
			t.Fatalf("expected hello, got %s", adt.Fields[0].AsString())
		}
	}
}

func TestBuiltinXmlComment(t *testing.T) {
	for range 20 {
		result, err := builtinXmlComment([]bytecode.Value{bytecode.NewString("TODO")})
		if err != nil {
			t.Fatalf("builtinXmlComment: %v", err)
		}
		adt := result.AsADT()
		if adt.Tag != xmlNodeTagComment {
			t.Fatalf("expected Comment tag, got %d", adt.Tag)
		}
		if adt.Fields[0].AsString() != "TODO" {
			t.Fatalf("expected TODO, got %s", adt.Fields[0].AsString())
		}
	}
}

func TestBuiltinXmlElement(t *testing.T) {
	for range 20 {
		attrs := bytecode.NewList([]bytecode.Value{
			bytecode.NewRecord([]bytecode.RecordField{
				{Name: "name", Value: bytecode.NewString("class")},
				{Name: "value", Value: bytecode.NewString("main")},
			}),
		})
		children := bytecode.NewList([]bytecode.Value{
			bytecode.NewADT(xmlNodeTagText, []bytecode.Value{bytecode.NewString("hi")}),
		})
		result, err := builtinXmlElement([]bytecode.Value{
			bytecode.NewString("div"), attrs, children,
		})
		if err != nil {
			t.Fatalf("builtinXmlElement: %v", err)
		}
		adt := result.AsADT()
		if adt.Tag != xmlNodeTagElement {
			t.Fatalf("expected Element tag, got %d", adt.Tag)
		}
		if adt.Fields[0].AsString() != "div" {
			t.Fatalf("expected div, got %s", adt.Fields[0].AsString())
		}
	}
}

// ---------------------------------------------------------------------------
// M2: String-returning tests
// ---------------------------------------------------------------------------

func TestBuiltinXmlGetText(t *testing.T) {
	for range 20 {
		// Element("p", [], [Text("hello "), Element("b", [], [Text("world")])])
		node := bytecode.NewADT(xmlNodeTagElement, []bytecode.Value{
			bytecode.NewString("p"),
			bytecode.NewList(nil),
			bytecode.NewList([]bytecode.Value{
				bytecode.NewADT(xmlNodeTagText, []bytecode.Value{bytecode.NewString("hello ")}),
				bytecode.NewADT(xmlNodeTagElement, []bytecode.Value{
					bytecode.NewString("b"),
					bytecode.NewList(nil),
					bytecode.NewList([]bytecode.Value{
						bytecode.NewADT(xmlNodeTagText, []bytecode.Value{bytecode.NewString("world")}),
					}),
				}),
			}),
		})
		result, err := builtinXmlGetText([]bytecode.Value{node})
		if err != nil {
			t.Fatalf("builtinXmlGetText: %v", err)
		}
		if result.AsString() != "hello world" {
			t.Fatalf("expected 'hello world', got %q", result.AsString())
		}
	}
}

func TestBuiltinXmlGetTag(t *testing.T) {
	for range 20 {
		node := bytecode.NewADT(xmlNodeTagElement, []bytecode.Value{
			bytecode.NewString("div"),
			bytecode.NewList(nil),
			bytecode.NewList(nil),
		})
		result, err := builtinXmlGetTag([]bytecode.Value{node})
		if err != nil {
			t.Fatalf("builtinXmlGetTag: %v", err)
		}
		if result.AsString() != "div" {
			t.Fatalf("expected 'div', got %q", result.AsString())
		}
	}
}

func TestBuiltinXmlSerialize(t *testing.T) {
	for range 20 {
		node := bytecode.NewADT(xmlNodeTagElement, []bytecode.Value{
			bytecode.NewString("p"),
			bytecode.NewList(nil),
			bytecode.NewList([]bytecode.Value{
				bytecode.NewADT(xmlNodeTagText, []bytecode.Value{bytecode.NewString("hi")}),
			}),
		})
		result, err := builtinXmlSerialize([]bytecode.Value{node})
		if err != nil {
			t.Fatalf("builtinXmlSerialize: %v", err)
		}
		if result.AsString() != "<p>hi</p>" {
			t.Fatalf("expected '<p>hi</p>', got %q", result.AsString())
		}
	}
}

func TestBuiltinXmlSerializeWithDecl(t *testing.T) {
	for range 20 {
		node := bytecode.NewADT(xmlNodeTagElement, []bytecode.Value{
			bytecode.NewString("root"),
			bytecode.NewList(nil),
			bytecode.NewList(nil),
		})
		result, err := builtinXmlSerializeWithDecl([]bytecode.Value{node})
		if err != nil {
			t.Fatalf("builtinXmlSerializeWithDecl: %v", err)
		}
		expected := `<?xml version="1.0" encoding="UTF-8"?><root/>`
		if result.AsString() != expected {
			t.Fatalf("expected %q, got %q", expected, result.AsString())
		}
	}
}

// ---------------------------------------------------------------------------
// M3: Parse tests
// ---------------------------------------------------------------------------

func TestBuiltinXmlParse_Ok(t *testing.T) {
	for range 20 {
		result, err := builtinXmlParse([]bytecode.Value{
			bytecode.NewString("<root>hi</root>"),
		})
		if err != nil {
			t.Fatalf("builtinXmlParse: %v", err)
		}
		adt := result.AsADT()
		if adt.Tag != resultTagOk {
			t.Fatalf("expected Ok (tag 0), got tag %d", adt.Tag)
		}
		// Payload should be Element("root", [], [Text("hi")])
		payload := adt.Fields[0].AsADT()
		if payload.Tag != xmlNodeTagElement {
			t.Fatalf("payload: expected Element, got tag %d", payload.Tag)
		}
		if payload.Fields[0].AsString() != "root" {
			t.Fatalf("payload tag: expected root, got %s", payload.Fields[0].AsString())
		}
	}
}

func TestBuiltinXmlParse_Err(t *testing.T) {
	for range 20 {
		result, err := builtinXmlParse([]bytecode.Value{
			bytecode.NewString("not xml <<>>"),
		})
		if err != nil {
			t.Fatalf("builtinXmlParse: %v", err)
		}
		adt := result.AsADT()
		if adt.Tag != resultTagErr {
			t.Fatalf("expected Err (tag 1), got tag %d", adt.Tag)
		}
	}
}

func TestBuiltinXmlParseElements(t *testing.T) {
	for range 20 {
		xml := "<items><item>a</item><item>b</item><other>c</other></items>"
		result, err := builtinXmlParseElements([]bytecode.Value{
			bytecode.NewString(xml),
			bytecode.NewString("item"),
			bytecode.NewInt(10),
		})
		if err != nil {
			t.Fatalf("builtinXmlParseElements: %v", err)
		}
		adt := result.AsADT()
		if adt.Tag != resultTagOk {
			t.Fatalf("expected Ok, got tag %d", adt.Tag)
		}
		list := adt.Fields[0].AsList()
		if len(list) != 2 {
			t.Fatalf("expected 2 items, got %d", len(list))
		}
	}
}

func TestBuiltinXmlParseWithLimit(t *testing.T) {
	for range 20 {
		result, err := builtinXmlParseWithLimit([]bytecode.Value{
			bytecode.NewString("<root><a/><b/><c/></root>"),
			bytecode.NewInt(2),
		})
		if err != nil {
			t.Fatalf("builtinXmlParseWithLimit: %v", err)
		}
		adt := result.AsADT()
		// Should fail: 4 nodes > limit 2
		if adt.Tag != resultTagErr {
			t.Fatalf("expected Err for exceeded limit, got tag %d", adt.Tag)
		}
	}
}

// ---------------------------------------------------------------------------
// M4: Query tests
// ---------------------------------------------------------------------------

func makeTestTree() bytecode.Value {
	// <root><item id="1">a</item><item id="2">b</item><other>c</other></root>
	return bytecode.NewADT(xmlNodeTagElement, []bytecode.Value{
		bytecode.NewString("root"),
		bytecode.NewList(nil),
		bytecode.NewList([]bytecode.Value{
			bytecode.NewADT(xmlNodeTagElement, []bytecode.Value{
				bytecode.NewString("item"),
				bytecode.NewList([]bytecode.Value{
					bytecode.NewRecord([]bytecode.RecordField{
						{Name: "name", Value: bytecode.NewString("id")},
						{Name: "value", Value: bytecode.NewString("1")},
					}),
				}),
				bytecode.NewList([]bytecode.Value{
					bytecode.NewADT(xmlNodeTagText, []bytecode.Value{bytecode.NewString("a")}),
				}),
			}),
			bytecode.NewADT(xmlNodeTagElement, []bytecode.Value{
				bytecode.NewString("item"),
				bytecode.NewList([]bytecode.Value{
					bytecode.NewRecord([]bytecode.RecordField{
						{Name: "name", Value: bytecode.NewString("id")},
						{Name: "value", Value: bytecode.NewString("2")},
					}),
				}),
				bytecode.NewList([]bytecode.Value{
					bytecode.NewADT(xmlNodeTagText, []bytecode.Value{bytecode.NewString("b")}),
				}),
			}),
			bytecode.NewADT(xmlNodeTagElement, []bytecode.Value{
				bytecode.NewString("other"),
				bytecode.NewList(nil),
				bytecode.NewList([]bytecode.Value{
					bytecode.NewADT(xmlNodeTagText, []bytecode.Value{bytecode.NewString("c")}),
				}),
			}),
		}),
	})
}

func TestBuiltinXmlFindAll(t *testing.T) {
	tree := makeTestTree()
	for range 20 {
		result, err := builtinXmlFindAll([]bytecode.Value{tree, bytecode.NewString("item")})
		if err != nil {
			t.Fatalf("builtinXmlFindAll: %v", err)
		}
		list := result.AsList()
		if len(list) != 2 {
			t.Fatalf("expected 2 items, got %d", len(list))
		}
	}
}

func TestBuiltinXmlFindFirst_Found(t *testing.T) {
	tree := makeTestTree()
	for range 20 {
		result, err := builtinXmlFindFirst([]bytecode.Value{tree, bytecode.NewString("item")})
		if err != nil {
			t.Fatalf("builtinXmlFindFirst: %v", err)
		}
		adt := result.AsADT()
		if adt.Tag != optionTagSome {
			t.Fatalf("expected Some, got tag %d", adt.Tag)
		}
	}
}

func TestBuiltinXmlFindFirst_NotFound(t *testing.T) {
	tree := makeTestTree()
	for range 20 {
		result, err := builtinXmlFindFirst([]bytecode.Value{tree, bytecode.NewString("missing")})
		if err != nil {
			t.Fatalf("builtinXmlFindFirst: %v", err)
		}
		adt := result.AsADT()
		if adt.Tag != optionTagNone {
			t.Fatalf("expected None, got tag %d", adt.Tag)
		}
	}
}

func TestBuiltinXmlGetAttr_Found(t *testing.T) {
	tree := makeTestTree()
	for range 20 {
		// Get first item child
		children := tree.AsADT().Fields[2].AsList()
		item := children[0]
		result, err := builtinXmlGetAttr([]bytecode.Value{item, bytecode.NewString("id")})
		if err != nil {
			t.Fatalf("builtinXmlGetAttr: %v", err)
		}
		adt := result.AsADT()
		if adt.Tag != optionTagSome {
			t.Fatalf("expected Some, got tag %d", adt.Tag)
		}
		if adt.Fields[0].AsString() != "1" {
			t.Fatalf("expected '1', got %q", adt.Fields[0].AsString())
		}
	}
}

func TestBuiltinXmlGetAttr_NotFound(t *testing.T) {
	tree := makeTestTree()
	for range 20 {
		children := tree.AsADT().Fields[2].AsList()
		item := children[0]
		result, err := builtinXmlGetAttr([]bytecode.Value{item, bytecode.NewString("nope")})
		if err != nil {
			t.Fatalf("builtinXmlGetAttr: %v", err)
		}
		adt := result.AsADT()
		if adt.Tag != optionTagNone {
			t.Fatalf("expected None, got tag %d", adt.Tag)
		}
	}
}

func TestBuiltinXmlGetChildren(t *testing.T) {
	tree := makeTestTree()
	for range 20 {
		result, err := builtinXmlGetChildren([]bytecode.Value{tree})
		if err != nil {
			t.Fatalf("builtinXmlGetChildren: %v", err)
		}
		list := result.AsList()
		if len(list) != 3 {
			t.Fatalf("expected 3 children, got %d", len(list))
		}
	}
}

func TestBuiltinXmlFindAllTexts(t *testing.T) {
	tree := makeTestTree()
	for range 20 {
		result, err := builtinXmlFindAllTexts([]bytecode.Value{tree, bytecode.NewString("item")})
		if err != nil {
			t.Fatalf("builtinXmlFindAllTexts: %v", err)
		}
		list := result.AsList()
		if len(list) != 2 {
			t.Fatalf("expected 2 texts, got %d", len(list))
		}
		if list[0].AsString() != "a" || list[1].AsString() != "b" {
			t.Fatalf("expected [a, b], got [%s, %s]", list[0].AsString(), list[1].AsString())
		}
	}
}

func TestBuiltinXmlFindAllAttrs(t *testing.T) {
	tree := makeTestTree()
	for range 20 {
		result, err := builtinXmlFindAllAttrs([]bytecode.Value{
			tree, bytecode.NewString("item"), bytecode.NewString("id"),
		})
		if err != nil {
			t.Fatalf("builtinXmlFindAllAttrs: %v", err)
		}
		list := result.AsList()
		if len(list) != 2 {
			t.Fatalf("expected 2 attrs, got %d", len(list))
		}
		if list[0].AsString() != "1" || list[1].AsString() != "2" {
			t.Fatalf("expected [1, 2], got [%s, %s]", list[0].AsString(), list[1].AsString())
		}
	}
}
