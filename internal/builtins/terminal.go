package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

func init() {
	T := types.NewBuilder()
	session := T.Con("TerminalSession")
	terminalErr := T.Con("TerminalError")
	options := T.Record(types.Field("alternate_screen", T.Bool()), types.Field("hide_cursor", T.Bool()))
	specs := []struct {
		name, op, description string
		args                  int
		typ                   func() types.Type
		params                []ParamDoc
	}{
		{"_terminal_info", "terminalInfo", "Query the configured terminal endpoints and physical size", 1,
			func() types.Type {
				return T.Func(T.Unit()).Returns(T.App("Result", T.Con("TerminalInfo"), terminalErr)).Effects("IO")
			}, nil},
		{"_terminal_withTerminal", "withTerminal", "Run a callback inside a native terminal scope with mandatory host restoration", 2,
			func() types.Type {
				a := T.Var("a")
				callback := T.Func(session).Returns(a).RowTail("e").Build()
				return T.Func(options, callback).Returns(T.App("Result", a, terminalErr)).RowTail("e").Effects("IO")
			}, []ParamDoc{{Name: "options", Description: "Alternate screen and cursor settings"}, {Name: "body", Description: "Callback; its effects propagate into the scope"}}},
		{"_terminal_readEvent", "terminalReadEvent", "Read a key, resize, EOF, interruption or idle event from a live terminal session", 2,
			func() types.Type {
				return T.Func(session, T.Int()).Returns(T.App("Result", T.Con("TerminalEvent"), terminalErr)).Effects("IO")
			},
			[]ParamDoc{{Name: "session", Description: "Host-validated scoped terminal handle"}, {Name: "timeout_ms", Description: "0 polls, 1..60000 bounds wait, -1 blocks"}}},
	}
	for _, entry := range specs {
		op := entry.op
		err := RegisterEffectBuiltin(BuiltinSpec{
			Module: "std/terminal", Name: entry.name, NumArgs: entry.args, Effect: "IO", Type: entry.typ,
			Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
				if op == "terminalInfo" {
					if len(args) != 1 {
						return nil, fmt.Errorf("_terminal_info: expected unit")
					}
					if _, ok := args[0].(*eval.UnitValue); !ok {
						return nil, fmt.Errorf("_terminal_info: expected unit")
					}
					args = nil
				}
				return effects.Call(ctx, "IO", op, args)
			},
			Metadata: &BuiltinMetadata{Description: entry.description, Params: entry.params, Returns: "Result containing the value or typed TerminalError", Examples: []Example{{Code: entry.name, Description: entry.description}}, Since: "v0.54.0", Stability: StabilityExperimental, Tags: []string{"io", "terminal", "native"}, Category: "io"},
		})
		if err != nil {
			panic(fmt.Sprintf("register %s: %v", entry.name, err))
		}
	}
}
