package builtins

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// foldStepHandler returns a handler that increments a counter and returns
// Stop(count) once count reaches stopAt, else Continue(count).
func foldStepCounterHandler(stopAt int) eval.Value {
	return &eval.BuiltinFunction{
		Name: "foldstep_counter",
		Fn: func(args []eval.Value) (eval.Value, error) {
			acc := args[0].(*eval.IntValue)
			next := acc.Value + 1
			ctor := "Continue"
			if next >= stopAt {
				ctor = "Stop"
			}
			return &eval.TaggedValue{
				TypeName: "FoldStep",
				CtorName: ctor,
				Fields:   []eval.Value{&eval.IntValue{Value: next}},
			}, nil
		},
	}
}

// TestXmlParseFoldStep_StopHalts verifies that returning Stop(acc) halts the
// scan immediately — subsequent matching elements are NOT passed to the handler.
func TestXmlParseFoldStep_StopHalts(t *testing.T) {
	xmlStr := `<root><r>1</r><r>2</r><r>3</r><r>4</r><r>5</r></root>`
	ctx := xmlFoldTestCtx()

	result, err := xmlParseFoldStepImpl(ctx, []eval.Value{
		&eval.StringValue{Value: xmlStr},
		&eval.StringValue{Value: "r"},
		&eval.IntValue{Value: 0},
		foldStepCounterHandler(2), // stop after visiting 2 elements
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := xmlAssertOk(t, result)
	iv, ok := inner.(*eval.IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", inner)
	}
	if iv.Value != 2 {
		t.Errorf("expected count=2 (stopped after 2), got %d — scan did not halt on Stop", iv.Value)
	}
}

// TestXmlParseFoldStep_ContinueOnlyEquivalentToFold verifies that a handler
// that always returns Continue is semantically equivalent to _xml_parseFold.
func TestXmlParseFoldStep_ContinueOnlyEquivalentToFold(t *testing.T) {
	xmlStr := `<root><r>1</r><r>2</r><r>3</r></root>`
	ctx := xmlFoldTestCtx()

	continueHandler := &eval.BuiltinFunction{
		Name: "continue_counter",
		Fn: func(args []eval.Value) (eval.Value, error) {
			acc := args[0].(*eval.IntValue)
			return &eval.TaggedValue{
				TypeName: "FoldStep",
				CtorName: "Continue",
				Fields:   []eval.Value{&eval.IntValue{Value: acc.Value + 1}},
			}, nil
		},
	}

	result, err := xmlParseFoldStepImpl(ctx, []eval.Value{
		&eval.StringValue{Value: xmlStr},
		&eval.StringValue{Value: "r"},
		&eval.IntValue{Value: 0},
		continueHandler,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := xmlAssertOk(t, result)
	iv := inner.(*eval.IntValue)
	if iv.Value != 3 {
		t.Errorf("Continue-only fold should visit all 3 elements, got %d", iv.Value)
	}
}

// TestXmlParseFoldStep_StopAtFirst verifies Stop on the very first element
// returns immediately without visiting any others.
func TestXmlParseFoldStep_StopAtFirst(t *testing.T) {
	xmlStr := `<root><r>1</r><r>2</r><r>3</r></root>`
	ctx := xmlFoldTestCtx()

	result, err := xmlParseFoldStepImpl(ctx, []eval.Value{
		&eval.StringValue{Value: xmlStr},
		&eval.StringValue{Value: "r"},
		&eval.IntValue{Value: 0},
		foldStepCounterHandler(1),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := xmlAssertOk(t, result)
	iv := inner.(*eval.IntValue)
	if iv.Value != 1 {
		t.Errorf("expected count=1 (stopped on first), got %d", iv.Value)
	}
}

// TestXmlParseFoldStep_BadReturnType verifies the builtin surfaces a clear
// error when the handler returns something that isn't a FoldStep tagged value.
func TestXmlParseFoldStep_BadReturnType(t *testing.T) {
	xmlStr := `<root><r>1</r></root>`
	ctx := xmlFoldTestCtx()

	badHandler := &eval.BuiltinFunction{
		Name: "bad_handler",
		Fn: func(args []eval.Value) (eval.Value, error) {
			return &eval.IntValue{Value: 42}, nil
		},
	}

	result, err := xmlParseFoldStepImpl(ctx, []eval.Value{
		&eval.StringValue{Value: xmlStr},
		&eval.StringValue{Value: "r"},
		&eval.IntValue{Value: 0},
		badHandler,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Expect Err(...) because handler did not return a FoldStep.
	tagged, ok := result.(*eval.TaggedValue)
	if !ok || tagged.CtorName != "Err" {
		t.Fatalf("expected Err(...) wrapped result for bad handler return, got %v", result)
	}
}
