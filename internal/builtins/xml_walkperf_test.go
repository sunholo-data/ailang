package builtins

import (
	"fmt"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

// ============================================================================
// Helpers
// ============================================================================

// makeElementWP builds an Element XmlNode with the given tag, attrs, and children.
// Attrs are passed as flat pairs: "name1", "val1", "name2", "val2", ...
func makeElementWP(tag string, attrPairs []string, children ...eval.Value) eval.Value {
	if len(attrPairs)%2 != 0 {
		panic("makeElementWP: attrPairs must be even-length")
	}
	attrs := make([]eval.Value, 0, len(attrPairs)/2)
	for i := 0; i < len(attrPairs); i += 2 {
		attrs = append(attrs, makeXmlAttr(attrPairs[i], attrPairs[i+1]))
	}
	return makeXmlElement(tag, attrs, children)
}

// counter is a BuiltinFunction that increments an IntValue accumulator.
// Useful for testing fold control flow without caring about node identity.
func counterHandler() eval.Value {
	return &eval.BuiltinFunction{
		Name: "counter",
		Fn: func(args []eval.Value) (eval.Value, error) {
			acc := args[0].(*eval.IntValue)
			return &eval.IntValue{Value: acc.Value + 1}, nil
		},
	}
}

// continueAllHandler returns Continue(acc + 1) for every child — never stops.
func continueAllHandler() eval.Value {
	return &eval.BuiltinFunction{
		Name: "continue_all",
		Fn: func(args []eval.Value) (eval.Value, error) {
			acc := args[0].(*eval.IntValue)
			next := &eval.IntValue{Value: acc.Value + 1}
			return &eval.TaggedValue{TypeName: "FoldStep", CtorName: "Continue", Fields: []eval.Value{next}}, nil
		},
	}
}

// stopAtNthHandler returns Stop(acc + 1) when acc == n - 1; otherwise Continue(acc + 1).
// I.e. processes exactly n children before stopping.
func stopAtNthHandler(n int) eval.Value {
	return &eval.BuiltinFunction{
		Name: fmt.Sprintf("stop_at_%d", n),
		Fn: func(args []eval.Value) (eval.Value, error) {
			acc := args[0].(*eval.IntValue)
			next := &eval.IntValue{Value: acc.Value + 1}
			ctor := "Continue"
			if next.Value >= n {
				ctor = "Stop"
			}
			return &eval.TaggedValue{TypeName: "FoldStep", CtorName: ctor, Fields: []eval.Value{next}}, nil
		},
	}
}

// ============================================================================
// _xml_foldChildren
// ============================================================================

func TestXmlFoldChildren_CountsDirectChildren(t *testing.T) {
	root := makeElementWP("root", nil,
		makeElementWP("a", nil),
		makeElementWP("b", nil),
		makeElementWP("c", nil),
	)
	ctx := xmlFoldTestCtx()
	result, err := xmlFoldChildrenImpl(ctx, []eval.Value{
		root,
		&eval.IntValue{Value: 0},
		counterHandler(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	iv, ok := result.(*eval.IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", result)
	}
	if iv.Value != 3 {
		t.Errorf("expected 3 direct children, got %d", iv.Value)
	}
}

func TestXmlFoldChildren_VisitsInDocumentOrder(t *testing.T) {
	root := makeElementWP("root", nil,
		makeElementWP("a", nil),
		makeElementWP("b", nil),
		makeElementWP("c", nil),
	)
	ctx := xmlFoldTestCtx()
	// Concat tag names so we observe order.
	handler := &eval.BuiltinFunction{
		Name: "concat_tags",
		Fn: func(args []eval.Value) (eval.Value, error) {
			acc := args[0].(*eval.StringValue)
			child := args[1].(*eval.TaggedValue)
			tag := child.Fields[0].(*eval.StringValue).Value
			return &eval.StringValue{Value: acc.Value + tag}, nil
		},
	}
	result, err := xmlFoldChildrenImpl(ctx, []eval.Value{
		root,
		&eval.StringValue{Value: ""},
		handler,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := result.(*eval.StringValue).Value
	if got != "abc" {
		t.Errorf("expected 'abc' (document order), got %q", got)
	}
}

func TestXmlFoldChildren_TextNodeReturnsInitUnchanged(t *testing.T) {
	textNode := makeXmlText("hello")
	ctx := xmlFoldTestCtx()
	result, err := xmlFoldChildrenImpl(ctx, []eval.Value{
		textNode,
		&eval.IntValue{Value: 42},
		counterHandler(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	iv := result.(*eval.IntValue)
	if iv.Value != 42 {
		t.Errorf("expected init=42 unchanged on Text node, got %d", iv.Value)
	}
}

func TestXmlFoldChildren_CommentNodeReturnsInitUnchanged(t *testing.T) {
	commentNode := makeXmlComment("note")
	ctx := xmlFoldTestCtx()
	result, err := xmlFoldChildrenImpl(ctx, []eval.Value{
		commentNode,
		&eval.IntValue{Value: 7},
		counterHandler(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.(*eval.IntValue).Value != 7 {
		t.Errorf("expected init unchanged on Comment node, got %v", result)
	}
}

func TestXmlFoldChildren_EmptyChildrenReturnsInit(t *testing.T) {
	root := makeElementWP("root", nil) // no children
	ctx := xmlFoldTestCtx()
	result, err := xmlFoldChildrenImpl(ctx, []eval.Value{
		root,
		&eval.IntValue{Value: 99},
		counterHandler(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.(*eval.IntValue).Value != 99 {
		t.Errorf("expected 99 (init) for empty children, got %d", result.(*eval.IntValue).Value)
	}
}

func TestXmlFoldChildren_HandlerErrorPropagates(t *testing.T) {
	root := makeElementWP("root", nil, makeElementWP("a", nil))
	ctx := xmlFoldTestCtx()
	failing := &eval.BuiltinFunction{
		Name: "fail",
		Fn: func(args []eval.Value) (eval.Value, error) {
			return nil, fmt.Errorf("handler boom")
		},
	}
	_, err := xmlFoldChildrenImpl(ctx, []eval.Value{
		root,
		&eval.IntValue{Value: 0},
		failing,
	})
	if err == nil {
		t.Fatal("expected handler error to propagate")
	}
}

func TestXmlFoldChildren_NilFnCallerN(t *testing.T) {
	root := makeElementWP("root", nil, makeElementWP("a", nil))
	ctx := effects.NewEffContext(nil) // FnCallerN not wired
	_, err := xmlFoldChildrenImpl(ctx, []eval.Value{
		root,
		&eval.IntValue{Value: 0},
		counterHandler(),
	})
	if err == nil {
		t.Fatal("expected error when FnCallerN not set")
	}
}

func TestXmlFoldChildren_OnlyDirectChildrenNotDescendants(t *testing.T) {
	// <root><a><nested/></a></root> — 1 direct child of root, not 2.
	root := makeElementWP("root", nil,
		makeElementWP("a", nil, makeElementWP("nested", nil)),
	)
	ctx := xmlFoldTestCtx()
	result, err := xmlFoldChildrenImpl(ctx, []eval.Value{
		root,
		&eval.IntValue{Value: 0},
		counterHandler(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.(*eval.IntValue).Value != 1 {
		t.Errorf("expected 1 (only direct child), got %d", result.(*eval.IntValue).Value)
	}
}

// ============================================================================
// _xml_foldChildrenStep
// ============================================================================

func TestXmlFoldChildrenStep_ContinuesToEnd(t *testing.T) {
	root := makeElementWP("root", nil,
		makeElementWP("a", nil), makeElementWP("b", nil), makeElementWP("c", nil),
	)
	ctx := xmlFoldTestCtx()
	result, err := xmlFoldChildrenStepImpl(ctx, []eval.Value{
		root,
		&eval.IntValue{Value: 0},
		continueAllHandler(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.(*eval.IntValue).Value != 3 {
		t.Errorf("expected 3 (visited all), got %d", result.(*eval.IntValue).Value)
	}
}

func TestXmlFoldChildrenStep_StopAtFirst(t *testing.T) {
	root := makeElementWP("root", nil,
		makeElementWP("a", nil), makeElementWP("b", nil), makeElementWP("c", nil),
	)
	ctx := xmlFoldTestCtx()
	result, err := xmlFoldChildrenStepImpl(ctx, []eval.Value{
		root,
		&eval.IntValue{Value: 0},
		stopAtNthHandler(1),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.(*eval.IntValue).Value != 1 {
		t.Errorf("expected 1 (stopped after first child), got %d", result.(*eval.IntValue).Value)
	}
}

func TestXmlFoldChildrenStep_StopAfterTwo(t *testing.T) {
	root := makeElementWP("root", nil,
		makeElementWP("a", nil), makeElementWP("b", nil), makeElementWP("c", nil),
	)
	ctx := xmlFoldTestCtx()
	result, err := xmlFoldChildrenStepImpl(ctx, []eval.Value{
		root,
		&eval.IntValue{Value: 0},
		stopAtNthHandler(2),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.(*eval.IntValue).Value != 2 {
		t.Errorf("expected 2 (stopped after second child), got %d", result.(*eval.IntValue).Value)
	}
}

func TestXmlFoldChildrenStep_NonElementReturnsInit(t *testing.T) {
	textNode := makeXmlText("hello")
	ctx := xmlFoldTestCtx()
	result, err := xmlFoldChildrenStepImpl(ctx, []eval.Value{
		textNode,
		&eval.IntValue{Value: 5},
		continueAllHandler(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.(*eval.IntValue).Value != 5 {
		t.Errorf("expected init=5 on Text node, got %d", result.(*eval.IntValue).Value)
	}
}

func TestXmlFoldChildrenStep_RejectsBadConstructor(t *testing.T) {
	root := makeElementWP("root", nil, makeElementWP("a", nil))
	ctx := xmlFoldTestCtx()
	bogus := &eval.BuiltinFunction{
		Name: "bogus",
		Fn: func(args []eval.Value) (eval.Value, error) {
			return &eval.TaggedValue{TypeName: "FoldStep", CtorName: "Maybe", Fields: []eval.Value{&eval.IntValue{Value: 0}}}, nil
		},
	}
	_, err := xmlFoldChildrenStepImpl(ctx, []eval.Value{
		root,
		&eval.IntValue{Value: 0},
		bogus,
	})
	if err == nil {
		t.Fatal("expected error on unknown FoldStep constructor")
	}
}

// ============================================================================
// _xml_getAttrMap
// ============================================================================

func TestXmlGetAttrMap_AllAttrsPresent(t *testing.T) {
	node := makeElementWP("img", []string{"src", "/a.png", "alt", "hi", "width", "100"})
	result, err := xmlGetAttrMapImpl(nil, []eval.Value{node})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mv := result.(*eval.MapValue)
	if mv.Size() != 3 {
		t.Errorf("expected 3 entries, got %d", mv.Size())
	}
	want := map[string]string{"src": "/a.png", "alt": "hi", "width": "100"}
	for k, v := range want {
		got, found := mv.Lookup(&eval.StringValue{Value: k})
		if !found {
			t.Errorf("expected key %q present", k)
			continue
		}
		if got.(*eval.StringValue).Value != v {
			t.Errorf("attr %q: expected %q, got %q", k, v, got.(*eval.StringValue).Value)
		}
	}
}

func TestXmlGetAttrMap_DuplicateNamesLastWriteWins(t *testing.T) {
	node := makeElementWP("x", []string{"a", "first", "a", "second", "a", "third"})
	result, err := xmlGetAttrMapImpl(nil, []eval.Value{node})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mv := result.(*eval.MapValue)
	if mv.Size() != 1 {
		t.Errorf("expected 1 entry (collapsed), got %d", mv.Size())
	}
	got, _ := mv.Lookup(&eval.StringValue{Value: "a"})
	if got.(*eval.StringValue).Value != "third" {
		t.Errorf("expected last-write-wins ('third'), got %q", got.(*eval.StringValue).Value)
	}
}

func TestXmlGetAttrMap_TextNodeReturnsEmpty(t *testing.T) {
	result, err := xmlGetAttrMapImpl(nil, []eval.Value{makeXmlText("hi")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mv := result.(*eval.MapValue)
	if mv.Size() != 0 {
		t.Errorf("expected empty map for Text node, got size %d", mv.Size())
	}
}

func TestXmlGetAttrMap_CommentNodeReturnsEmpty(t *testing.T) {
	result, err := xmlGetAttrMapImpl(nil, []eval.Value{makeXmlComment("note")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.(*eval.MapValue).Size() != 0 {
		t.Errorf("expected empty map for Comment node")
	}
}

func TestXmlGetAttrMap_ElementWithNoAttrs(t *testing.T) {
	result, err := xmlGetAttrMapImpl(nil, []eval.Value{makeElementWP("div", nil)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.(*eval.MapValue).Size() != 0 {
		t.Errorf("expected empty map for element with no attrs")
	}
}

// ============================================================================
// _xml_nodeKind
// ============================================================================

func TestXmlNodeKind_Element(t *testing.T) {
	result, err := xmlNodeKindImpl(nil, []eval.Value{makeElementWP("div", nil)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tv := result.(*eval.TaggedValue)
	if tv.CtorName != "KindElement" {
		t.Errorf("expected KindElement, got %q", tv.CtorName)
	}
	if tv.TypeName != "NodeKind" {
		t.Errorf("expected TypeName NodeKind, got %q", tv.TypeName)
	}
	if len(tv.Fields) != 0 {
		t.Errorf("expected no payload, got %d fields", len(tv.Fields))
	}
}

func TestXmlNodeKind_Text(t *testing.T) {
	result, err := xmlNodeKindImpl(nil, []eval.Value{makeXmlText("hi")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.(*eval.TaggedValue).CtorName != "KindText" {
		t.Errorf("expected KindText, got %q", result.(*eval.TaggedValue).CtorName)
	}
}

func TestXmlNodeKind_Comment(t *testing.T) {
	result, err := xmlNodeKindImpl(nil, []eval.Value{makeXmlComment("c")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.(*eval.TaggedValue).CtorName != "KindComment" {
		t.Errorf("expected KindComment, got %q", result.(*eval.TaggedValue).CtorName)
	}
}

func TestXmlNodeKind_NonTaggedValueFallsBackToKindComment(t *testing.T) {
	// Defensive — should not happen in normal use, but xml_nodeKind must not panic.
	result, err := xmlNodeKindImpl(nil, []eval.Value{&eval.StringValue{Value: "not a node"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.(*eval.TaggedValue).CtorName != "KindComment" {
		t.Errorf("expected defensive KindComment, got %q", result.(*eval.TaggedValue).CtorName)
	}
}

// ============================================================================
// Benchmarks
// ============================================================================
//
// Honesty note: these benchmarks run at the Go level, NOT through the AILANG
// evaluator. In test mode `FnCallerN` is just `goFn.Fn(args)` — a direct Go
// call with no real closure-invocation, no argument boxing/unboxing, no effect
// context propagation, no builtin dispatch via the registry. So the headline
// FFI cost the design doc targets does not appear here.
//
// What these benchmarks DO catch:
//   1. Regression in the Go implementations themselves (alloc count, time).
//   2. Floor cost when FnCallerN overhead is absent.
//
// The user-visible win — collapsing ~1,900 AILANG↔Go crossings per page into
// roughly one — is measured at the AILANG level by the example file
// examples/runnable/xml_walk_perf.ail. Quote those numbers in CHANGELOG,
// not these.

// makeWideTree builds an Element with `fanout` direct children, each itself an Element
// with a couple of nested grandchildren and some attributes. Approximates the per-page
// shape the ailang-parse profile observed.
func makeWideTree(fanout int) eval.Value {
	children := make([]eval.Value, fanout)
	for i := 0; i < fanout; i++ {
		children[i] = makeElementWP("p",
			[]string{"class", "para", "id", fmt.Sprintf("p%d", i)},
			makeXmlText("some paragraph text content"),
			makeElementWP("span", []string{"style", "bold"}, makeXmlText("inline")),
		)
	}
	return makeXmlElement("article", nil, children)
}

// BenchmarkXmlWalk_Classic emulates `flatMap(\c. count + 1, getChildren(node))`:
// call _xml_getChildren to materialise a [XmlNode] list, then invoke the
// per-child handler via FnCallerN (the same callback machinery foldChildren
// uses internally). This is the apples-to-apples comparison — both variants
// incur the AILANG↔Go round-trip per child.
func BenchmarkXmlWalk_Classic(b *testing.B) {
	root := makeWideTree(1900)
	ctx := xmlFoldTestCtx()
	handler := counterHandler()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		got, err := xmlGetChildrenImpl(nil, []eval.Value{root})
		if err != nil {
			b.Fatal(err)
		}
		list := got.(*eval.ListValue)
		acc := eval.Value(&eval.IntValue{Value: 0})
		for _, c := range list.Elements {
			next, err := ctx.FnCallerN(handler, []eval.Value{acc, c})
			if err != nil {
				b.Fatal(err)
			}
			acc = next
		}
	}
}

// BenchmarkXmlWalk_FoldChildren uses _xml_foldChildren directly — no
// intermediate [XmlNode] list allocation.
func BenchmarkXmlWalk_FoldChildren(b *testing.B) {
	root := makeWideTree(1900)
	ctx := xmlFoldTestCtx()
	handler := counterHandler()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := xmlFoldChildrenImpl(ctx, []eval.Value{
			root,
			&eval.IntValue{Value: 0},
			handler,
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAttr_PerAttr_7x100 simulates extracting 7 attributes from each of 100 nodes
// using individual getAttr-style lookups (here, by walking the attr list manually
// since _xml_getAttr is the same shape).
func BenchmarkAttr_PerAttr_7x100(b *testing.B) {
	const numNodes = 100
	const numAttrs = 7
	attrNames := []string{"src", "alt", "width", "height", "srcset", "title", "loading"}
	nodes := make([]eval.Value, numNodes)
	for i := 0; i < numNodes; i++ {
		pairs := make([]string, 0, numAttrs*2)
		for _, name := range attrNames {
			pairs = append(pairs, name, "value")
		}
		nodes[i] = makeElementWP("img", pairs)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		for _, node := range nodes {
			for _, name := range attrNames {
				_, _ = xmlGetAttrImpl(nil, []eval.Value{node, &eval.StringValue{Value: name}})
			}
		}
	}
}

// BenchmarkAttr_AttrMap_7x100 extracts the same 7 attributes from 100 nodes
// using one _xml_getAttrMap call per node, then 7 lookups against the map.
func BenchmarkAttr_AttrMap_7x100(b *testing.B) {
	const numNodes = 100
	const numAttrs = 7
	attrNames := []string{"src", "alt", "width", "height", "srcset", "title", "loading"}
	nodes := make([]eval.Value, numNodes)
	for i := 0; i < numNodes; i++ {
		pairs := make([]string, 0, numAttrs*2)
		for _, name := range attrNames {
			pairs = append(pairs, name, "value")
		}
		nodes[i] = makeElementWP("img", pairs)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		for _, node := range nodes {
			got, _ := xmlGetAttrMapImpl(nil, []eval.Value{node})
			mv := got.(*eval.MapValue)
			for _, name := range attrNames {
				_, _ = mv.Lookup(&eval.StringValue{Value: name})
			}
		}
	}
}
