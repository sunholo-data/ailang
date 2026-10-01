package bytecode

import (
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/builtins"
	"github.com/sunholo-data/ailang/internal/types"
)

// M-VM-PURE-BUILTIN-COVERAGE (#1447): every pure registry builtin reaches the
// VM in exactly one of three ways.
//
//  1. Native: a hand-written VM function listed in BuiltinNames/HOFBuiltinNames.
//  2. Adapted: listed in AdaptedBuiltinNames. The VM converts the arguments to
//     evaluator values, calls the builtin's registered Go Impl (not the
//     tree-walking evaluator), and converts the result back. These names are
//     appended to BuiltinNames' index space, so they dispatch through the
//     ordinary OpBuiltinCall.
//  3. Unported: AdaptReason names why the signature cannot be adapted. Strict
//     bytecode refuses these with that reason.
//
// The coverage test (internal/vm/builtin_coverage_test.go) fails if a pure builtin falls in none of
// the three, so the gap can no longer grow silently.

// Reasons a pure builtin's signature cannot cross the generic adapter.
const (
	ReasonPolymorphic = "polymorphic: a type variable can carry ADT or closure values the converter cannot name; needs a native port"
	ReasonMap         = "map-value: the VM has no Map value"
	ReasonClosure     = "closure-arg: function-typed parameter; needs a native HOF port"
	ReasonADTParam    = "adt-param: VM ADT values carry no type name, so they cannot be converted to evaluator values"
	ReasonOpaque      = "opaque-type: a named type the converter does not know"
	ReasonOpenRecord  = "open-record: row-polymorphic record"
)

var scalarTypes = map[string]bool{
	"int": true, "float": true, "bool": true, "string": true, "bytes": true, "()": true,
}

// AdaptReason returns "" when a builtin of type t can run through the generic
// adapter, or the reason it cannot. Parameters must convert VM→evaluator, so
// they may not contain ADTs. Results convert evaluator→VM, which also handles
// std Option and Result (bytecode.StdADTTag).
func AdaptReason(t types.Type) string {
	fn, ok := t.(*types.TFunc2)
	if !ok {
		return ReasonOpaque
	}
	for _, p := range fn.Params {
		if r := convertReason(p, false); r != "" {
			return r
		}
	}
	return convertReason(fn.Return, true)
}

func convertReason(t types.Type, isResult bool) string {
	switch t := t.(type) {
	case *types.TCon:
		if scalarTypes[t.Name] {
			return ""
		}
		return ReasonOpaque
	case *types.TVar, *types.TVar2:
		return ReasonPolymorphic
	case *types.TFunc2:
		return ReasonClosure
	case *types.TMap:
		return ReasonMap
	case *types.TArray:
		return convertReason(t.Element, isResult)
	case *types.TTuple:
		return firstReason(t.Elements, isResult)
	case *types.TRecord:
		fields := make([]types.Type, 0, len(t.Fields))
		for _, f := range t.Fields {
			fields = append(fields, f)
		}
		return firstReason(fields, isResult)
	case *types.TRecord2:
		if t.Row == nil {
			return ""
		}
		if t.Row.Tail != nil {
			return ReasonOpenRecord
		}
		fields := make([]types.Type, 0, len(t.Row.Labels))
		for _, f := range t.Row.Labels {
			fields = append(fields, f)
		}
		return firstReason(fields, isResult)
	case *types.TApp:
		con, ok := t.Constructor.(*types.TCon)
		if !ok {
			return ReasonOpaque
		}
		switch con.Name {
		case "list", "List", "Array", "array":
			return firstReason(t.Args, isResult)
		case "Option", "Result":
			if !isResult {
				return ReasonADTParam
			}
			return firstReason(t.Args, isResult)
		}
		return ReasonOpaque
	}
	return ReasonOpaque
}

// firstReason returns the most informative reason among ts, ranked by a fixed
// order so the result does not depend on map iteration (record fields).
func firstReason(ts []types.Type, isResult bool) string {
	best := ""
	for _, t := range ts {
		if r := convertReason(t, isResult); r != "" && reasonRank(r) < reasonRank(best) {
			best = r
		}
	}
	return best
}

func reasonRank(r string) int {
	for i, x := range []string{ReasonClosure, ReasonMap, ReasonADTParam, ReasonPolymorphic, ReasonOpenRecord, ReasonOpaque} {
		if r == x {
			return i
		}
	}
	return 1 << 30 // "" ranks last
}

// IsCallableBuiltin reports whether a registry builtin is reached through a
// lowered builtin call. Registry entries without the "_" prefix are operators
// (e.g. "::") that lower to dedicated opcodes instead.
func IsCallableBuiltin(registryName string) bool { return strings.HasPrefix(registryName, "_") }

// irName is the name a registry builtin has in a lowered stmt.BuiltinCall
// (internal/gen/lower/expr.go: "_" + registry name).
func irName(registryName string) string { return "_" + registryName }

// registryName inverts irName.
func registryName(ir string) string { return strings.TrimPrefix(ir, "_") }

// PureBuiltinSpec returns the registry spec of a pure builtin by its IR name.
func PureBuiltinSpec(ir string) (*builtins.BuiltinSpec, bool) {
	spec, ok := builtins.GetSpec(registryName(ir))
	if !ok || !spec.IsPure {
		return nil, false
	}
	return spec, true
}

// AdaptedBuiltinNames lists, by IR name and sorted, every pure registry builtin
// with no native VM implementation whose signature the adapter accepts. Its
// entries occupy OpBuiltinCall indices len(BuiltinNames)… in this order; the VM
// builds the matching dispatch entries from this same slice.
var AdaptedBuiltinNames = func() []string {
	native := make(map[string]bool, len(BuiltinNames)+len(HOFBuiltinNames))
	for _, n := range BuiltinNames {
		native[n] = true
	}
	for _, n := range HOFBuiltinNames {
		native[n] = true
	}
	var out []string
	for name, spec := range builtins.AllSpecs() {
		ir := irName(name)
		if !IsCallableBuiltin(name) || !spec.IsPure || native[ir] {
			continue
		}
		if AdaptReason(spec.Type()) == "" {
			out = append(out, ir)
		}
	}
	sort.Strings(out)
	return out
}()

// AdaptedRegistryName maps an AdaptedBuiltinNames entry to its registry name,
// for the VM side to look up the Impl.
func AdaptedRegistryName(ir string) string { return registryName(ir) }
