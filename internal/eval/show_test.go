package eval

import (
	"strings"
	"testing"
)

// The REPL / simple-evaluator `show` and `toText` used their own renderer,
// which quoted strings and spelled floats with %g, so `show({a: "x"})` was
// {a: "x"} in the REPL but {a: x} under `ailang run`. They now call Show, the
// renderer the `show` builtin and the bytecode VM share (#1453 cleanup). The
// renderer's own cases live in internal/builtins/show_test.go.
func TestEnvShowIsTheCanonicalRenderer(t *testing.T) {
	env := NewEnvironment()
	registerBuiltins(env, nil)
	show, ok := env.Get("show")
	if !ok {
		t.Fatal("show not registered")
	}
	toText, _ := env.Get("toText")

	long := make([]Value, 40)
	for i := range long {
		long[i] = &IntValue{Value: i}
	}
	cases := []struct {
		v    Value
		want string
	}{
		{&RecordValue{Fields: map[string]Value{"a": &StringValue{Value: "x"}}}, "{a: x}"},
		{&FloatValue{Value: 1e20}, "100000000000000000000.0"},
		{&StringValue{Value: `say "hi"`}, `say "hi"`},
		{&BuiltinFunction{Name: "print"}, "<function>"},
		{&TaggedValue{CtorName: "Some", Fields: []Value{&FloatValue{Value: 2}}}, "Some(2.0)"},
		{&ListValue{Elements: []Value{&ListValue{Elements: []Value{&ListValue{Elements: []Value{
			&ListValue{Elements: []Value{&ListValue{Elements: []Value{&IntValue{Value: 1}}}}}}}}}}}, "[[[[...]]]]"},
		{&ListValue{Elements: long}, "[0, 1, 2, 3, 4, 5, 6... 35, 36, 37, 38, 39]"},
	}
	for _, c := range cases {
		if got := Show(c.v); got != c.want {
			t.Errorf("Show(%v) = %q, want %q", c.v, got, c.want)
		}
		for name, fn := range map[string]Value{"show": show, "toText": toText} {
			out, err := fn.(*BuiltinFunction).Fn([]Value{c.v})
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if got := out.(*StringValue).Value; got != c.want {
				t.Errorf("env %s(%v) = %q, want %q", name, c.v, got, c.want)
			}
		}
	}
	if strings.Contains(Show(&StringValue{Value: "a"}), `"`) {
		t.Error("show must not quote strings")
	}
}
