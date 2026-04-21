package builtins

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

// xmlFoldTestCtx creates an EffContext with FnCallerN wired for fold callbacks.
func xmlFoldTestCtx() *effects.EffContext {
	ctx := effects.NewEffContext(nil)
	ctx.FnCaller = func(fn eval.Value, arg eval.Value) (eval.Value, error) {
		goFn, ok := fn.(*eval.BuiltinFunction)
		if !ok {
			return nil, fmt.Errorf("test FnCaller: expected BuiltinFunction, got %T", fn)
		}
		return goFn.Fn([]eval.Value{arg})
	}
	ctx.FnCallerN = func(fn eval.Value, args []eval.Value) (eval.Value, error) {
		goFn, ok := fn.(*eval.BuiltinFunction)
		if !ok {
			return nil, fmt.Errorf("test FnCallerN: expected BuiltinFunction, got %T", fn)
		}
		return goFn.Fn(args)
	}
	return ctx
}

// xmlFoldHandler creates a fold handler that appends getText(node) to a list accumulator.
func xmlFoldHandler() eval.Value {
	return &eval.BuiltinFunction{
		Name: "test_fold_handler",
		Fn: func(args []eval.Value) (eval.Value, error) {
			acc := args[0].(*eval.ListValue)
			node := args[1]
			// Extract text from XmlNode
			var buf strings.Builder
			collectText(node, &buf)
			newElems := make([]eval.Value, len(acc.Elements)+1)
			copy(newElems, acc.Elements)
			newElems[len(acc.Elements)] = &eval.StringValue{Value: buf.String()}
			return &eval.ListValue{Elements: newElems}, nil
		},
	}
}

func TestXmlParseFold_Basic(t *testing.T) {
	xmlStr := `<root><item>one</item><item>two</item><item>three</item></root>`
	ctx := xmlFoldTestCtx()

	result, err := xmlParseFoldImpl(ctx, []eval.Value{
		&eval.StringValue{Value: xmlStr},
		&eval.StringValue{Value: "item"},
		&eval.ListValue{Elements: nil},
		xmlFoldHandler(),
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
	for i, expected := range []string{"one", "two", "three"} {
		sv := list.Elements[i].(*eval.StringValue)
		if sv.Value != expected {
			t.Errorf("element %d: expected %q, got %q", i, expected, sv.Value)
		}
	}
}

func TestXmlParseFold_EmptyXml(t *testing.T) {
	xmlStr := `<root></root>`
	ctx := xmlFoldTestCtx()

	result, err := xmlParseFoldImpl(ctx, []eval.Value{
		&eval.StringValue{Value: xmlStr},
		&eval.StringValue{Value: "item"},
		&eval.IntValue{Value: 0}, // init = 0
		&eval.BuiltinFunction{
			Name: "counter",
			Fn: func(args []eval.Value) (eval.Value, error) {
				acc := args[0].(*eval.IntValue)
				return &eval.IntValue{Value: acc.Value + 1}, nil
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := xmlAssertOk(t, result)
	iv, ok := inner.(*eval.IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", inner)
	}
	if iv.Value != 0 {
		t.Errorf("expected 0 (no matches), got %d", iv.Value)
	}
}

func TestXmlParseFold_CountElements(t *testing.T) {
	xmlStr := `<root><si>a</si><si>b</si><si>c</si><si>d</si><si>e</si></root>`
	ctx := xmlFoldTestCtx()

	counter := &eval.BuiltinFunction{
		Name: "counter",
		Fn: func(args []eval.Value) (eval.Value, error) {
			acc := args[0].(*eval.IntValue)
			return &eval.IntValue{Value: acc.Value + 1}, nil
		},
	}

	result, err := xmlParseFoldImpl(ctx, []eval.Value{
		&eval.StringValue{Value: xmlStr},
		&eval.StringValue{Value: "si"},
		&eval.IntValue{Value: 0},
		counter,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := xmlAssertOk(t, result)
	iv := inner.(*eval.IntValue)
	if iv.Value != 5 {
		t.Errorf("expected count 5, got %d", iv.Value)
	}
}

func TestXmlParseFold_NestedElements(t *testing.T) {
	// Items are nested inside sections — fold should find them at any depth
	xmlStr := `<root><section><item>a</item><item>b</item></section><section><item>c</item></section></root>`
	ctx := xmlFoldTestCtx()

	result, err := xmlParseFoldImpl(ctx, []eval.Value{
		&eval.StringValue{Value: xmlStr},
		&eval.StringValue{Value: "item"},
		&eval.ListValue{Elements: nil},
		xmlFoldHandler(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := xmlAssertOk(t, result)
	list := inner.(*eval.ListValue)
	if len(list.Elements) != 3 {
		t.Fatalf("expected 3 nested items, got %d", len(list.Elements))
	}
	for i, expected := range []string{"a", "b", "c"} {
		sv := list.Elements[i].(*eval.StringValue)
		if sv.Value != expected {
			t.Errorf("element %d: expected %q, got %q", i, expected, sv.Value)
		}
	}
}

func TestXmlParseFold_HandlerError(t *testing.T) {
	xmlStr := `<root><item>a</item><item>b</item></root>`
	ctx := xmlFoldTestCtx()

	errorHandler := &eval.BuiltinFunction{
		Name: "error_handler",
		Fn: func(args []eval.Value) (eval.Value, error) {
			return nil, fmt.Errorf("handler exploded")
		},
	}

	result, err := xmlParseFoldImpl(ctx, []eval.Value{
		&eval.StringValue{Value: xmlStr},
		&eval.StringValue{Value: "item"},
		&eval.IntValue{Value: 0},
		errorHandler,
	})
	if err != nil {
		t.Fatalf("unexpected Go-level error: %v", err)
	}

	// Should return Err result (not Go error)
	tv, ok := result.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", result)
	}
	if tv.CtorName != "Err" {
		t.Fatalf("expected Err result for handler error, got %s", tv.CtorName)
	}
	errMsg := tv.Fields[0].(*eval.StringValue).Value
	if !strings.Contains(errMsg, "handler exploded") {
		t.Errorf("expected 'handler exploded' in error, got %q", errMsg)
	}
}

func TestXmlParseFold_NilFnCallerN(t *testing.T) {
	xmlStr := `<root><item>a</item></root>`
	ctx := effects.NewEffContext(nil) // FnCallerN not set

	_, err := xmlParseFoldImpl(ctx, []eval.Value{
		&eval.StringValue{Value: xmlStr},
		&eval.StringValue{Value: "item"},
		&eval.IntValue{Value: 0},
		&eval.BuiltinFunction{Name: "noop", Fn: func(args []eval.Value) (eval.Value, error) { return args[0], nil }},
	})
	if err == nil {
		t.Fatal("expected error for nil FnCallerN")
	}
	if !strings.Contains(err.Error(), "FnCallerN not set") {
		t.Errorf("error should mention FnCallerN: %v", err)
	}
}

func TestXmlParseFold_MatchesParseElements(t *testing.T) {
	// Verify fold produces same data as parseElements for determinism (run 20x per A1)
	xmlStr := `<root><row id="1"><c>val1</c></row><row id="2"><c>val2</c></row><row id="3"><c>val3</c></row></root>`

	for run := 0; run < 20; run++ {
		ctx := xmlFoldTestCtx()

		// Get parseElements result
		peResult, err := xmlParseElementsImpl(ctx, []eval.Value{
			&eval.StringValue{Value: xmlStr},
			&eval.StringValue{Value: "row"},
			&eval.IntValue{Value: 100},
		})
		if err != nil {
			t.Fatalf("run %d: parseElements error: %v", run, err)
		}
		peList := xmlAssertOk(t, peResult).(*eval.ListValue)

		// Get parseFold result (collect into list to compare)
		pfResult, err := xmlParseFoldImpl(ctx, []eval.Value{
			&eval.StringValue{Value: xmlStr},
			&eval.StringValue{Value: "row"},
			&eval.ListValue{Elements: nil},
			&eval.BuiltinFunction{
				Name: "collect",
				Fn: func(args []eval.Value) (eval.Value, error) {
					acc := args[0].(*eval.ListValue)
					node := args[1]
					newElems := make([]eval.Value, len(acc.Elements)+1)
					copy(newElems, acc.Elements)
					newElems[len(acc.Elements)] = node
					return &eval.ListValue{Elements: newElems}, nil
				},
			},
		})
		if err != nil {
			t.Fatalf("run %d: parseFold error: %v", run, err)
		}
		pfList := xmlAssertOk(t, pfResult).(*eval.ListValue)

		if len(peList.Elements) != len(pfList.Elements) {
			t.Fatalf("run %d: length mismatch: parseElements=%d, parseFold=%d",
				run, len(peList.Elements), len(pfList.Elements))
		}
		for i := range peList.Elements {
			peStr := peList.Elements[i].String()
			pfStr := pfList.Elements[i].String()
			if peStr != pfStr {
				t.Errorf("run %d element %d: parseElements=%q, parseFold=%q", run, i, peStr, pfStr)
			}
		}
	}
}

func TestXmlParseFold_LargeDoc(t *testing.T) {
	// Generate XML with 5000 items, fold all of them (no limit)
	var sb strings.Builder
	sb.WriteString("<root>")
	for i := 0; i < 5000; i++ {
		fmt.Fprintf(&sb, `<si><t>string%d</t></si>`, i)
	}
	sb.WriteString("</root>")

	ctx := xmlFoldTestCtx()
	counter := &eval.BuiltinFunction{
		Name: "counter",
		Fn: func(args []eval.Value) (eval.Value, error) {
			acc := args[0].(*eval.IntValue)
			return &eval.IntValue{Value: acc.Value + 1}, nil
		},
	}

	result, err := xmlParseFoldImpl(ctx, []eval.Value{
		&eval.StringValue{Value: sb.String()},
		&eval.StringValue{Value: "si"},
		&eval.IntValue{Value: 0},
		counter,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := xmlAssertOk(t, result)
	iv := inner.(*eval.IntValue)
	if iv.Value != 5000 {
		t.Errorf("expected count 5000, got %d", iv.Value)
	}
}
