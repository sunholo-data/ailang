package builtins

import (
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

func isInt(want int) eval.Value {
	return goFn(func(args []eval.Value) (eval.Value, error) {
		return &eval.BoolValue{Value: args[0].(*eval.IntValue).Value == want}, nil
	})
}

// countingPred returns a predicate x == want that records how many times it ran.
func countingPred(want int, calls *int) eval.Value {
	return goFn(func(args []eval.Value) (eval.Value, error) {
		*calls++
		return &eval.BoolValue{Value: args[0].(*eval.IntValue).Value == want}, nil
	})
}

func optionString(v eval.Value) string {
	tv, ok := v.(*eval.TaggedValue)
	if !ok {
		return "not-an-option"
	}
	if tv.CtorName == "None" {
		return "None"
	}
	return "Some(" + tv.Fields[0].String() + ")"
}

func TestListAnyFindIndex(t *testing.T) {
	ctx := newTestEffCtx()
	cases := []struct {
		name      string
		want      int
		xs        *eval.ListValue
		any       bool
		idx       string
		wantCalls int
	}{
		{"empty", 1, intList(), false, "None", 0},
		{"single hit", 1, intList(1), true, "Some(0)", 1},
		{"single miss", 2, intList(1), false, "None", 1},
		{"first of duplicates", 7, intList(1, 7, 7, 3), true, "Some(1)", 2},
		{"last", 3, intList(1, 7, 7, 3), true, "Some(3)", 4},
		{"none", 9, intList(1, 7, 7, 3), false, "None", 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			calls := 0
			got, err := listAnyImpl(ctx, []eval.Value{countingPred(c.want, &calls), c.xs})
			if err != nil {
				t.Fatal(err)
			}
			if got.(*eval.BoolValue).Value != c.any {
				t.Errorf("any = %v, want %v", got, c.any)
			}
			if calls != c.wantCalls {
				t.Errorf("any called p %d times, want %d (must stop at the first match)", calls, c.wantCalls)
			}
			calls = 0
			got, err = listFindIndexImpl(ctx, []eval.Value{countingPred(c.want, &calls), c.xs})
			if err != nil {
				t.Fatal(err)
			}
			if s := optionString(got); s != c.idx {
				t.Errorf("findIndex = %s, want %s", s, c.idx)
			}
			if calls != c.wantCalls {
				t.Errorf("findIndex called p %d times, want %d", calls, c.wantCalls)
			}
		})
	}
}

func TestListAnyErrors(t *testing.T) {
	ctx := newTestEffCtx()
	notBool := goFn(func([]eval.Value) (eval.Value, error) { return &eval.IntValue{Value: 1}, nil })
	if _, err := listAnyImpl(ctx, []eval.Value{notBool, intList(1)}); err == nil || !strings.Contains(err.Error(), "must return bool") {
		t.Errorf("non-bool predicate: err = %v", err)
	}
	boom := errors.New("boom")
	failing := goFn(func([]eval.Value) (eval.Value, error) { return nil, boom })
	if _, err := listFindIndexImpl(ctx, []eval.Value{failing, intList(1)}); !errors.Is(err, boom) {
		t.Errorf("callback error not propagated: %v", err)
	}
	if _, err := listAnyImpl(ctx, []eval.Value{isInt(1), &eval.IntValue{Value: 1}}); err == nil {
		t.Error("non-list argument accepted")
	}
	if _, err := listAnyImpl(nil, []eval.Value{isInt(1), intList(1)}); err == nil {
		t.Error("nil context accepted")
	}
}

func TestListFoldrOrder(t *testing.T) {
	ctx := newTestEffCtx()
	// f(x, acc) = acc*10 + x, so foldr over [1,2,3] from 0 visits 3, 2, 1: 321
	step := goFn(func(args []eval.Value) (eval.Value, error) {
		x, acc := args[0].(*eval.IntValue).Value, args[1].(*eval.IntValue).Value
		return &eval.IntValue{Value: acc*10 + x}, nil
	})
	for _, c := range []struct {
		xs   *eval.ListValue
		want int
	}{{intList(), 0}, {intList(1), 1}, {intList(1, 2, 3), 321}} {
		got, err := listFoldrImpl(ctx, []eval.Value{step, &eval.IntValue{Value: 0}, c.xs})
		if err != nil {
			t.Fatal(err)
		}
		if v := got.(*eval.IntValue).Value; v != c.want {
			t.Errorf("foldr = %d, want %d", v, c.want)
		}
	}
}

func TestListMapAccumL(t *testing.T) {
	ctx := newTestEffCtx()
	// running totals: (st + x, st + x)
	step := goFn(func(args []eval.Value) (eval.Value, error) {
		st, x := args[0].(*eval.IntValue).Value, args[1].(*eval.IntValue).Value
		return &eval.TupleValue{Elements: []eval.Value{&eval.IntValue{Value: st + x}, &eval.IntValue{Value: st + x}}}, nil
	})
	xs := intList(1, 2, 3)
	got, err := listMapAccumLImpl(ctx, []eval.Value{step, &eval.IntValue{Value: 0}, xs})
	if err != nil {
		t.Fatal(err)
	}
	pair := got.(*eval.TupleValue)
	outs := toInts(t, pair.Elements[0])
	if len(outs) != 3 || outs[0] != 1 || outs[1] != 3 || outs[2] != 6 {
		t.Errorf("outputs = %v, want [1 3 6]", outs)
	}
	if st := pair.Elements[1].(*eval.IntValue).Value; st != 6 {
		t.Errorf("final state = %d, want 6", st)
	}
	// the output must not alias the input's backing array
	if &pair.Elements[0].(*eval.ListValue).Elements[0] == &xs.Elements[0] {
		t.Error("output aliases input")
	}

	// empty input: ([], s0) and f never runs
	never := goFn(func([]eval.Value) (eval.Value, error) { t.Fatal("f called on empty list"); return nil, nil })
	got, err = listMapAccumLImpl(ctx, []eval.Value{never, &eval.StringValue{Value: "s0"}, intList()})
	if err != nil {
		t.Fatal(err)
	}
	pair = got.(*eval.TupleValue)
	if n := len(pair.Elements[0].(*eval.ListValue).Elements); n != 0 || pair.Elements[1].(*eval.StringValue).Value != "s0" {
		t.Errorf("empty = %v, want ([], \"s0\")", got)
	}

	notPair := goFn(func([]eval.Value) (eval.Value, error) { return &eval.IntValue{Value: 1}, nil })
	if _, err := listMapAccumLImpl(ctx, []eval.Value{notPair, &eval.IntValue{Value: 0}, xs}); err == nil || !strings.Contains(err.Error(), "pair") {
		t.Errorf("non-pair step: err = %v", err)
	}
}

// Pure builtins must give the same answer on every call.
func TestListSearchFoldDeterminism(t *testing.T) {
	ctx := newTestEffCtx()
	xs := intList(5, 3, 8, 3, 9, 1)
	sub := goFn(func(args []eval.Value) (eval.Value, error) {
		return &eval.IntValue{Value: args[0].(*eval.IntValue).Value - args[1].(*eval.IntValue).Value}, nil
	})
	acc := goFn(func(args []eval.Value) (eval.Value, error) {
		st, x := args[0].(*eval.IntValue).Value, args[1].(*eval.IntValue).Value
		return &eval.TupleValue{Elements: []eval.Value{&eval.IntValue{Value: st * x}, &eval.IntValue{Value: st + x}}}, nil
	})
	var first string
	for i := 0; i < 20; i++ {
		a, _ := listAnyImpl(ctx, []eval.Value{isInt(9), xs})
		f, _ := listFindIndexImpl(ctx, []eval.Value{isInt(3), xs})
		r, _ := listFoldrImpl(ctx, []eval.Value{sub, &eval.IntValue{Value: 0}, xs})
		m, _ := listMapAccumLImpl(ctx, []eval.Value{acc, &eval.IntValue{Value: 1}, xs})
		got := a.String() + " " + optionString(f) + " " + r.String() + " " + m.String()
		if i == 0 {
			first = got
			if !strings.HasPrefix(got, "true Some(1) ") {
				t.Fatalf("unexpected answer %q", got)
			}
		} else if got != first {
			t.Fatalf("run %d = %q, run 0 = %q", i, got, first)
		}
	}
}
