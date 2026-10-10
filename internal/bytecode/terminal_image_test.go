package bytecode

import (
	"strings"
	"testing"
)

// A malformed effect instruction must fail image admission before the VM can
// slice registers or dispatch a different host operation.
func TestImageAdmitsBoundedEffectCalls(t *testing.T) {
	for _, tc := range []struct {
		name string
		inst Instruction
		want string
	}{
		{"valid", EncodeABC(OpEffectCall, 0, 0, 1), ""},
		{"missing destination", EncodeABC(OpEffectCall, 2, 0, 0), "dest"},
		{"unknown operation", EncodeABC(OpEffectCall, 0, uint8(len(EffectBuiltinNames)), 1), "invalid effect index"},
		{"argument overflow", EncodeABC(OpEffectCall, 0, 0, 2), "effect args overflow"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			img := NewImage()
			img.AddPrototype(&FuncPrototype{Name: "effect", NumRegs: 2, Instructions: []Instruction{tc.inst, EncodeABC(OpReturn, 0, 0, 0)}})
			err := img.Validate()
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %s", err, tc.want)
			}
		})
	}
}

func TestDisassembleNativeEffectOperands(t *testing.T) {
	img := NewImage()
	img.AddPrototype(&FuncPrototype{Name: "native", NumRegs: 4, Instructions: []Instruction{EncodeABC(OpEffectCall, 1, 7, 2), EncodeABC(OpReturn, 1, 0, 0)}})
	out := Disassemble(img)
	if !strings.Contains(out, "EFFECT_CALL  r1, builtin#7, argc=2") {
		t.Fatalf("effect operands missing: %s", out)
	}
}

// Constructor identity is a pair of module and ADT name. Coincidentally named
// application constructors must never gain a standard host ordinal.
func TestTerminalADTOrdinalsAndUnknownIdentity(t *testing.T) {
	cases := []struct {
		typ, ctor string
		tag       int
	}{
		{"TerminalSession", "TerminalSession", 0},
		{"TerminalKey", "Text", 0}, {"TerminalKey", "Delete", 13},
		{"TerminalEvent", "Key", 0}, {"TerminalEvent", "Resize", 1}, {"TerminalEvent", "EndOfInput", 2}, {"TerminalEvent", "Interrupted", 3}, {"TerminalEvent", "Idle", 4},
		{"TerminalError", "Unsupported", 0}, {"TerminalError", "CleanupFailure", 7},
	}
	for _, tc := range cases {
		tag, ok := StdADTTag("std/terminal", tc.typ, tc.ctor)
		if !ok || tag != tc.tag {
			t.Errorf("%s.%s: got (%d,%v)", tc.typ, tc.ctor, tag, ok)
		}
	}
	for _, tc := range []struct{ module, typ, ctor string }{
		{"application/model", "TerminalEvent", "Key"},
		{"std/terminal", "UnknownType", "Key"},
		{"std/terminal", "TerminalKey", "UnknownKey"},
		{"std/terminal", "TerminalSession", "Text"},
	} {
		if _, ok := StdADTTag(tc.module, tc.typ, tc.ctor); ok {
			t.Errorf("foreign or unknown constructor accepted: %+v", tc)
		}
	}
}
