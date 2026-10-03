package vm

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/builtins"
	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// unportedPure is the ratchet for #1447: every pure, callable registry builtin
// the VM cannot run, with the reason bytecode.AdaptReason computes for it. The
// coverage test fails when a pure builtin is in no bucket (add a native port,
// or list it here with its reason) and when an entry here has become native or
// adapted (delete it). The list may only shrink.
var unportedPure = map[string]string{
	"__get_header":           bytecode.ReasonPolymorphic,
	"__has_header":           bytecode.ReasonPolymorphic,
	"__html_parse":           bytecode.ReasonOpaque,
	"__html_parseFragment":   bytecode.ReasonOpaque,
	"__list_takeFlatMap":     bytecode.ReasonClosure,
	"__list_takeMap":         bytecode.ReasonClosure,
	"__map_empty":            bytecode.ReasonMap,
	"__map_from_list":        bytecode.ReasonPolymorphic,
	"__map_insert":           bytecode.ReasonMap,
	"__map_keys":             bytecode.ReasonMap,
	"__map_lookup":           bytecode.ReasonMap,
	"__map_member":           bytecode.ReasonMap,
	"__map_remove":           bytecode.ReasonMap,
	"__map_size":             bytecode.ReasonMap,
	"__map_to_list":          bytecode.ReasonMap,
	"__map_values":           bytecode.ReasonMap,
	"__xml_flatMapChildren":  bytecode.ReasonOpaque,
	"__xml_foldChildren":     bytecode.ReasonOpaque,
	"__xml_foldChildrenStep": bytecode.ReasonOpaque,
	"__xml_getAttrMap":       bytecode.ReasonOpaque,
	"__xml_mapChildren":      bytecode.ReasonOpaque,
	"__xml_nodeKind":         bytecode.ReasonOpaque,
	"__xml_parseFoldStep":    bytecode.ReasonPolymorphic,
}

func TestPureBuiltinCoverage(t *testing.T) {
	native := map[string]bool{}
	for _, n := range bytecode.BuiltinNames {
		native[n] = true
	}
	for _, n := range bytecode.HOFBuiltinNames {
		native[n] = true
	}
	adapted := map[string]bool{}
	for _, n := range bytecode.AdaptedBuiltinNames {
		adapted[n] = true
	}

	var counts = map[string]int{}
	seen := map[string]bool{}
	for name, spec := range builtins.AllSpecs() {
		if !spec.IsPure || !bytecode.IsCallableBuiltin(name) {
			continue
		}
		ir := "_" + name
		seen[ir] = true
		reason, listed := unportedPure[ir]
		switch {
		case native[ir] || adapted[ir]:
			if listed {
				t.Errorf("%s is now covered (native=%v adapted=%v): delete it from unportedPure", ir, native[ir], adapted[ir])
			}
			if native[ir] {
				counts["native"]++
			} else {
				counts["adapted"]++
			}
		case !listed:
			t.Errorf("pure builtin %s has no VM implementation (%s): port it natively or add it to unportedPure",
				ir, bytecode.AdaptReason(spec.Type()))
		default:
			if got := bytecode.AdaptReason(spec.Type()); got != reason {
				t.Errorf("unportedPure[%s] reason = %q, but AdaptReason now says %q", ir, reason, got)
			}
			counts["unported"]++
		}
	}
	for ir := range unportedPure {
		if !seen[ir] {
			t.Errorf("unportedPure lists %s, which is not a pure registry builtin", ir)
		}
	}
	t.Logf("pure builtins on the VM: native=%d adapted=%d unported=%d",
		counts["native"], counts["adapted"], counts["unported"])
}

// TestAdaptedBuiltinsNeverPanic calls every adapted builtin once with
// type-directed sample arguments. The adapter passes a nil EffContext, so a
// pure builtin that dereferences it would panic; the adapter must turn that
// into an error, and no adapted builtin should need one at all.
func TestAdaptedBuiltinsNeverPanic(t *testing.T) {
	for i, ir := range bytecode.AdaptedBuiltinNames {
		spec, ok := builtins.GetSpec(bytecode.AdaptedRegistryName(ir))
		if !ok {
			t.Fatalf("%s: not in registry", ir)
		}
		fn := spec.Type().(*types.TFunc2)
		args := make([]bytecode.Value, len(fn.Params))
		for j, p := range fn.Params {
			args[j] = sampleValue(t, p)
		}
		_, err := BuiltinTable[nativeBuiltinCount+i](args)
		if err != nil && strings.Contains(err.Error(), "panicked") {
			t.Errorf("%s panicked on sample args: %v", ir, err)
		}
	}
}

func sampleValue(t *testing.T, ty types.Type) bytecode.Value {
	t.Helper()
	switch ty := ty.(type) {
	case *types.TCon:
		switch ty.Name {
		case "int":
			return bytecode.NewInt(1)
		case "float":
			return bytecode.NewFloat(1.5)
		case "bool":
			return bytecode.NewBool(true)
		case "string":
			return bytecode.NewString("a")
		case "bytes":
			return bytecode.NewBytes(&bytecode.BytesObj{B: []byte("a")})
		case "()":
			return bytecode.Unit()
		}
	case *types.TApp:
		elem := sampleValue(t, ty.Args[0])
		if c, ok := ty.Constructor.(*types.TCon); ok && (c.Name == "Array" || c.Name == "array") {
			return bytecode.NewArray([]bytecode.Value{elem})
		}
		return bytecode.NewList([]bytecode.Value{elem})
	case *types.TArray:
		return bytecode.NewArray([]bytecode.Value{sampleValue(t, ty.Element)})
	case *types.TTuple:
		elems := make([]bytecode.Value, len(ty.Elements))
		for i, e := range ty.Elements {
			elems[i] = sampleValue(t, e)
		}
		return bytecode.NewTuple(elems)
	case *types.TRecord2:
		names := make([]string, 0, len(ty.Row.Labels))
		for n := range ty.Row.Labels {
			names = append(names, n)
		}
		sort.Strings(names)
		fields := make([]bytecode.RecordField, len(names))
		for i, n := range names {
			fields[i] = bytecode.RecordField{Name: n, Value: sampleValue(t, ty.Row.Labels[n])}
		}
		return bytecode.NewRecord(fields)
	}
	t.Fatalf("sampleValue: no sample for %s (%T)", ty, ty)
	return bytecode.Value{}
}

// TestAdaptedBuiltinMatchesImpl pins the adapter's contract on concrete
// values: same answer as calling the registered Impl directly.
func TestAdaptedBuiltinMatchesImpl(t *testing.T) {
	cases := []struct {
		ir   string
		args []bytecode.Value
		want string
	}{
		{"__str_repeat", []bytecode.Value{bytecode.NewString("ab"), bytecode.NewInt(3)}, `"ababab"`},
		{"__list_range", []bytecode.Value{bytecode.NewInt(2), bytecode.NewInt(5)}, "[2, 3, 4]"},
	}
	for _, c := range cases {
		idx := -1
		for i, n := range bytecode.AdaptedBuiltinNames {
			if n == c.ir {
				idx = i
			}
		}
		if idx < 0 {
			t.Errorf("%s is not adapted", c.ir)
			continue
		}
		got, err := BuiltinTable[nativeBuiltinCount+idx](c.args)
		if err != nil {
			t.Fatalf("%s: %v", c.ir, err)
		}
		spec, _ := builtins.GetSpec(bytecode.AdaptedRegistryName(c.ir))
		evArgs := make([]eval.Value, len(c.args))
		for i, a := range c.args {
			evArgs[i], _ = BytecodeToEval(a)
		}
		direct, err := spec.Impl(nil, evArgs)
		if err != nil {
			t.Fatalf("%s direct: %v", c.ir, err)
		}
		back, _ := BytecodeToEval(got)
		if back.String() != direct.String() {
			t.Errorf("%s: adapter %s, Impl %s", c.ir, back, direct)
		}
		if fmt.Sprint(got) != c.want {
			t.Errorf("%s = %s, want %s", c.ir, got, c.want)
		}
	}
}

// TestOperatorBuiltinsAreAdapted is #1450: operator builtins are registered
// without the "_" prefix (bitwiseXor_Int) but called as _bitwiseXor_Int. They
// were excluded from the adapter, so `n ^ 3` failed --strict-bytecode.
func TestOperatorBuiltinsAreAdapted(t *testing.T) {
	adapted := map[string]bool{}
	for _, n := range bytecode.AdaptedBuiltinNames {
		adapted[n] = true
	}
	for _, ir := range []string{"_bitwiseXor_Int", "_bitwiseAnd_Int", "_shiftLeft_Int", "_shiftRight_Int", "_shiftRightLogical_Int"} {
		if !adapted[ir] {
			t.Errorf("%s is not adapted", ir)
		}
	}
	if bytecode.IsCallableBuiltin("::") {
		t.Errorf(`"::" lowers to an opcode and must not count as a callable builtin`)
	}
}
