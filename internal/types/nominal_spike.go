package types

// M-V1-NOMINAL-TYPE-IDENTITY M0 — MEASUREMENT INSTRUMENT, NOT MERGEABLE.
//
// TCon carries a Module. Two TCons are the same nominal type when their names
// match and either module is "" (the wildcard) or both modules are equal. The
// wildcard keeps every existing program's verdict; what this file adds is a
// counter of every place the wildcard is *used*, keyed by type name, the site
// that minted the unqualified TCon, and the site that compared it.
//
// Enable with AILANG_NOMINAL_SPIKE_LOG=<file>. Each process appends TSV lines
//   pid  count  kind  name  mint-site  compare-site  outer-site
// at counts 1, 2, 4, 8, ... per distinct key, so the max count per (pid, key)
// is within 2x of the true count. tools aggregate with max-per-pid, sum-over-pids.
//
// kinds:
//   q-vs-empty      one side qualified, the other "" — the wildcard did real
//                   work; in M2 (no wildcard) this is a false rejection unless
//                   the empty side's minting site is qualified.
//   empty-vs-empty  both "" on a nominal (capitalised) name — an identity that
//                   never got a module on either side.
//   q-mismatch      both qualified, modules differ — a nominal rejection the
//                   spike introduces (the a-rows of the design doc).

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"unicode"
)

var (
	nominalSpikeLog  = os.Getenv("AILANG_NOMINAL_SPIKE_LOG") //nolint:forbidigo // spike instrument
	nominalSpikeMu   sync.Mutex
	nominalSpikeCnt  = map[string]int{}
	nominalSpikeMint sync.Map // *TCon -> string (minting site)
)

// NominalSpikeEnabled reports whether the M0 counter is on.
func NominalSpikeEnabled() bool { return nominalSpikeLog != "" }

// isNominalName: capitalised names are nominal (ADTs, records-by-name,
// builtin opaque types). Lower-case names are primitives and structural heads.
func isNominalName(name string) bool {
	for _, r := range name {
		return unicode.IsUpper(r)
	}
	return false
}

// SpikeCon mints a TCon and, when the spike is on, records where it was minted.
// Every non-primitive `&TCon{Name: x}` literal in non-test code was rewritten
// to this call for M0 so that wildcard events can be attributed to a site.
func SpikeCon(name string) *TCon {
	t := &TCon{Name: name}
	if nominalSpikeLog != "" && isNominalName(name) {
		nominalSpikeMint.Store(t, callerSite(2, false)+" <- "+callerSite(3, false))
	}
	return t
}

// SpikeConIn mints a qualified TCon (module may be "", then it is SpikeCon).
func SpikeConIn(name, module string) *TCon {
	if module == "" {
		t := &TCon{Name: name}
		if nominalSpikeLog != "" && isNominalName(name) {
			nominalSpikeMint.Store(t, callerSite(2, false)+" <- "+callerSite(3, false))
		}
		return t
	}
	return &TCon{Name: name, Module: module}
}

// NominalSame is the single (Module, Name) comparison used by Equals, the
// unifier's TCon case, equalHead and SafeEquals.
func NominalSame(a, b *TCon) bool {
	if a.Name != b.Name {
		return false
	}
	if a.Module == b.Module {
		if a.Module == "" && nominalSpikeLog != "" && isNominalName(a.Name) {
			nominalSpikeRecord("empty-vs-empty", a, b)
		}
		return true
	}
	if a.Module == "" || b.Module == "" {
		if nominalSpikeLog != "" && isNominalName(a.Name) {
			empty := a
			if b.Module == "" {
				empty = b
			}
			nominalSpikeRecord("q-vs-empty", empty, nil)
		}
		return true
	}
	if nominalSpikeLog != "" {
		nominalSpikeRecord("q-mismatch:"+a.Module+"|"+b.Module, a, b)
	}
	return false
}

func mintSiteOf(t *TCon) string {
	if t == nil {
		return "-"
	}
	if v, ok := nominalSpikeMint.Load(t); ok {
		return v.(string)
	}
	return "<unattributed:literal-or-decoded>"
}

func nominalSpikeRecord(kind string, a, b *TCon) {
	mint := mintSiteOf(a)
	if b != nil && kind == "empty-vs-empty" {
		mb := mintSiteOf(b)
		if mb < mint { // order-independent key
			mint, mb = mb, mint
		}
		mint = mint + " ~ " + mb
	}
	cmp := callerSite(3, false)
	outer := callerSite(3, true)
	key := kind + "\t" + a.Name + "\t" + mint + "\t" + cmp + "\t" + outer
	nominalSpikeMu.Lock()
	nominalSpikeCnt[key]++
	n := nominalSpikeCnt[key]
	nominalSpikeMu.Unlock()
	if n&(n-1) != 0 { // log at powers of two only
		return
	}
	f, err := os.OpenFile(nominalSpikeLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	fmt.Fprintf(f, "%d\t%d\t%s\n", os.Getpid(), n, key)
	_ = f.Close()
}

// callerSite returns "pkg/file.go:line func" of the frame `skip` levels above
// the caller. With outer=true it walks further up to the first frame that is
// not in the unifier/equality machinery, so recursion inside Unify collapses
// to the checker site that started it.
func callerSite(skip int, outer bool) string {
	pcs := make([]uintptr, 48)
	n := runtime.Callers(skip+1, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	for {
		fr, more := frames.Next()
		file := fr.File
		if i := strings.Index(file, "/internal/"); i >= 0 {
			file = file[i+len("/internal/"):]
		}
		fn := fr.Function
		if i := strings.LastIndex(fn, "/"); i >= 0 {
			fn = fn[i+1:]
		}
		site := fmt.Sprintf("%s:%d %s", file, fr.Line, fn)
		if !outer {
			return site
		}
		if !isUnifierPlumbing(file) {
			return site
		}
		if !more {
			return site
		}
	}
}

func isUnifierPlumbing(file string) bool {
	if !strings.HasPrefix(file, "types/") {
		return false
	}
	base := file[len("types/"):]
	switch {
	case strings.HasPrefix(base, "unification"),
		strings.HasPrefix(base, "row_unification"),
		base == "types.go", base == "nominal_spike.go", base == "safe_string.go",
		strings.HasPrefix(base, "types_v2"):
		return true
	}
	return false
}

// QualifyNames returns t with every unqualified TCon whose name is in `names`
// stamped with `module`. Copy-on-write: shared TCons are never mutated.
// Cycle-safety: types reaching here are trees built by the elaborator, the
// iface builder or the checker's env; recursion is depth-capped at 256.
func QualifyNames(t Type, names map[string]bool, module string) Type {
	if module == "" || len(names) == 0 {
		return t
	}
	return qualifyNames(t, names, module, 0)
}

func qualifyNames(t Type, names map[string]bool, module string, depth int) Type {
	if t == nil || depth > 256 {
		return t
	}
	q := func(x Type) Type { return qualifyNames(x, names, module, depth+1) }
	switch ty := t.(type) {
	case *TCon:
		if ty.Module == "" && names[ty.Name] {
			return &TCon{Name: ty.Name, Module: module}
		}
		return ty
	case *TApp:
		args := make([]Type, len(ty.Args))
		for i, a := range ty.Args {
			args[i] = q(a)
		}
		return &TApp{Constructor: q(ty.Constructor), Args: args}
	case *TFunc2:
		params := make([]Type, len(ty.Params))
		for i, p := range ty.Params {
			params[i] = q(p)
		}
		cp := *ty
		cp.Params = params
		cp.Return = q(ty.Return)
		return &cp
	case *TList:
		return &TList{Element: q(ty.Element)}
	case *TArray:
		return &TArray{Element: q(ty.Element)}
	case *TMap:
		return &TMap{Key: q(ty.Key), Value: q(ty.Value)}
	case *TTuple:
		els := make([]Type, len(ty.Elements))
		for i, e := range ty.Elements {
			els[i] = q(e)
		}
		return &TTuple{Elements: els}
	case *TRecord:
		fields := make(map[string]Type, len(ty.Fields))
		for k, v := range ty.Fields {
			fields[k] = q(v)
		}
		return &TRecord{Fields: fields, Row: ty.Row, TypeName: ty.TypeName}
	case *TRecordOpen:
		fields := make(map[string]Type, len(ty.Fields))
		for k, v := range ty.Fields {
			fields[k] = q(v)
		}
		return &TRecordOpen{Fields: fields, Row: ty.Row}
	case *TRecord2:
		if ty.Row == nil || ty.Row.Labels == nil {
			return ty
		}
		cp := *ty.Row
		cp.Labels = make(map[string]Type, len(ty.Row.Labels))
		for k, v := range ty.Row.Labels {
			cp.Labels[k] = q(v)
		}
		return &TRecord2{Row: &cp}
	case *TLabelled:
		cp := *ty
		cp.Inner = q(ty.Inner)
		return &cp
	default:
		return t
	}
}
