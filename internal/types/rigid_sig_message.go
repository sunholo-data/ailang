package types

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
)

// Wording of TC_RIGID_001 (#2273). Every name shown is the one the author
// wrote, and a type the message tells the author to write is one they can
// paste: it never contains an inference variable. When the body fixes only the
// shape of the type, the message gives the shape (`(_) -> _`) and no type to
// paste.

var (
	rigidLocRE = regexp.MustCompile(`([^\s:()]+\.ail):(\d+):(\d+)`)
	// An ANF temporary is not a name the author wrote: "function application
	// of $tmp1" is just "function application".
	rigidTmpCalleeRE = regexp.MustCompile(` of \$[A-Za-z0-9_]+`)
)

// rigidWhere renders a constraint path as " (what)" and its position.
func rigidWhere(path []string) (string, ast.Pos, bool) {
	joined := strings.Join(path, " ")
	m := rigidLocRE.FindStringSubmatchIndex(joined)
	if m == nil {
		return "", ast.Pos{}, false
	}
	var pos ast.Pos
	pos.File = joined[m[2]:m[3]]
	fmt.Sscanf(joined[m[4]:m[5]], "%d", &pos.Line)
	fmt.Sscanf(joined[m[6]:m[7]], "%d", &pos.Column)
	what := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(joined[:m[0]]), "at"))
	what = rigidTmpCalleeRE.ReplaceAllString(what, "")
	if what == "" {
		return "", pos, true
	}
	return " (" + what + ")", pos, true
}

func (v *rigidViolation) toError(sig *RigidSig) error {
	name := v.v.Display
	where, pos, ok := rigidWhere(v.at)
	if !ok {
		pos = v.v.Pos
	}
	var msg, fix string
	switch v.kind {
	case "concrete":
		msg, fix = v.concreteWording(sig, where)
	case "equal":
		other := v.other.Display
		if v.v.Row {
			msg = fmt.Sprintf("the signature of `%s` says `| %s` and `| %s` stand for independent further record fields, but the body requires them to be the same%s",
				sig.Func, name, other, where)
			fix = fmt.Sprintf("either use one row variable for both in the signature of `%s`, or keep the records apart in the body", sig.Func)
			break
		}
		msg = fmt.Sprintf("the signature of `%s` says `%s` and `%s` are independent types, but the body requires them to be the same type%s",
			sig.Func, name, other, where)
		fix = fmt.Sprintf("either use one type variable for both in the signature of `%s`, or keep values of `%s` and `%s` apart in the body",
			sig.Func, name, other)
	case "literal":
		msg = fmt.Sprintf("the signature of `%s` says `%s` is any type, but the body uses an integer literal where a value of type `%s` is needed, and a literal has no value of an arbitrary type",
			sig.Func, name, name)
		fix = fmt.Sprintf("write the concrete numeric type (`int` or `float`) instead of `%s` in the signature of `%s`%s", name, sig.Func, dropBinder(v.v))
	default: // escape
		msg = fmt.Sprintf("the signature of `%s` says `%s` is any type, but in the body it is the type of a value from outside `%s`",
			sig.Func, name, sig.Func)
		fix = fmt.Sprintf("write the outer type instead of `%s` in the signature of `%s`", name, sig.Func)
	}
	if len(v.group) > 0 {
		msg = strings.Replace(msg, "but the body requires", "but a call within its recursive group requires", 1)
		msg += fmt.Sprintf(". `%s` is in a recursive group (%s), and calls within a recursive group use one type for `%s` (polymorphic recursion is not supported)",
			sig.Func, backtickList(v.group), name)
		fix = fmt.Sprintf("call `%s` at its own type `%s` within the group, move the call out of the group, or write the type the call needs instead of `%s`", sig.Func, name, name)
	}
	if v.v.ScopeNote != "" {
		msg += ". " + v.v.ScopeNote
	}
	if v.v.NestedAt.Line > 0 && v.kind != "equal" {
		msg += fmt.Sprintf(". `%s` is also written in an annotation inside `%s` at %s: there it is `%s`'s own `%s`, not a new type variable — give that annotation another name if it meant its own type",
			name, sig.Func, v.v.NestedAt, sig.Func, name)
	}
	if v.kind == "equal" && v.other.ScopeNote != "" {
		msg += ". " + v.other.ScopeNote
	}
	if !v.v.Explicit && v.v.ParamName {
		msg += fmt.Sprintf(". `%s` is the name of a parameter of `%s`, not a type: in a signature, a lowercase name that is not a declared or imported type is a type variable", name, sig.Func)
	} else if !v.v.Explicit && v.v.Suggest != "" && v.typoOK {
		msg += fmt.Sprintf(". Did you mean the type `%s`? A lowercase name in a signature that is not a declared or imported type is a type variable", v.v.Suggest)
		fix = fmt.Sprintf("write `%s` instead of `%s`", v.v.Suggest, name)
	}
	if v.clause {
		// Every wording names "the body"; here the clause broke the promise,
		// in the message and in the fix ("keep values … apart in the clause").
		msg = strings.ReplaceAll(msg, "the body", "the clause")
		fix = strings.ReplaceAll(fix, "the body", "the clause")
	}
	data := map[string]any{"function": sig.Func, "type_var": name, "kind": v.kind}
	if v.v.Suggest != "" && !v.v.Explicit && v.typoOK {
		data["did_you_mean"] = v.v.Suggest
	}
	err := newRigidSigError(pos, msg, fix, data)
	err.kind = v.kind
	return err
}

// concreteWording words a variable the body pins to one type. The type is
// offered for pasting only when the body fixes it completely.
func (v *rigidViolation) concreteWording(sig *RigidSig, where string) (msg, fix string) {
	name := v.v.Display
	got, exact := neededTypeText(v.got, sig)
	if v.v.Row {
		return rowWording(sig, name, v.got, where)
	}
	if !exact {
		msg = fmt.Sprintf("the signature of `%s` says `%s` is any type, but the body requires `%s` to have the form `%s`%s",
			sig.Func, name, name, got, where)
		fix = fmt.Sprintf("give `%s` a type of the form `%s` in the signature of `%s` (the body does not fix the parts written `_`), or make the body work for every type `%s`",
			name, got, sig.Func, name)
		return msg, fix
	}
	msg = fmt.Sprintf("the signature of `%s` says `%s` is any type, but the body requires `%s` to be `%s`%s",
		sig.Func, name, name, got, where)
	fix = fmt.Sprintf("either write `%s` instead of `%s` in the signature of `%s`%s, or make the body work for every type `%s`",
		got, name, sig.Func, dropBinder(v.v), name)
	if !v.v.Explicit && v.v.ParamName {
		fix = fmt.Sprintf("write the parameter's type (`%s`) instead of `%s`", got, name)
	}
	return msg, fix
}

// dropBinder completes a "write T instead of a" fix for a variable listed in
// the declaration's `[...]`: the binder goes too (`[int]` is not a binder).
func dropBinder(v RigidVar) string {
	if !v.Explicit {
		return ""
	}
	return fmt.Sprintf(" and drop `%s` from its `[...]` list", v.Display)
}

// rowWording words a record row variable (`| r`) the body pins: to extra
// fields it uses, or to no further fields at all (a closed record).
func rowWording(sig *RigidSig, name string, got Type, where string) (msg, fix string) {
	fields := rowLabels(got)
	if len(fields) == 0 {
		msg = fmt.Sprintf("the signature of `%s` says `| %s` stands for any further record fields, but the body requires the record to have no further fields%s",
			sig.Func, name, where)
		fix = fmt.Sprintf("drop `| %s` from the signature of `%s` (a closed record type), or keep the caller's other fields (update the record instead of building a new one)", name, sig.Func)
		return msg, fix
	}
	msg = fmt.Sprintf("the signature of `%s` says `| %s` stands for any further record fields, but the body uses the field(s) %s, which the record type does not list%s",
		sig.Func, name, backtickList(fields), where)
	fix = fmt.Sprintf("add %s to the record type in the signature of `%s`, before `| %s`", backtickList(fields), sig.Func, name)
	return msg, fix
}

// rowLabels returns the sorted field names a record row resolved to.
func rowLabels(t Type) []string {
	var m map[string]Type
	switch r := t.(type) {
	case *Row:
		m = r.Labels
	case *TRecordOpen:
		m = r.Fields
	case *TRecord:
		m = r.Fields
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func backtickList(names []string) string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = "`" + n + "`"
	}
	return strings.Join(out, ", ")
}

// neededTypeText renders the type the body needs a signature variable to be,
// in source syntax with written names. exact is false when it still contains
// an inference variable; those positions are written `_`. An open record's
// unknown rest is written `| r` (a fresh row variable of the signature, so the
// same text pasted at two positions still names one row).
func neededTypeText(t Type, sig *RigidSig) (string, bool) {
	written := map[string]string{}
	taken := map[string]bool{}
	for _, v := range sig.Vars {
		written[v.Name] = v.Display
		taken[v.Display] = true
	}
	row := "r"
	for i := 1; taken[row]; i++ {
		row = fmt.Sprintf("r%d", i)
	}
	p := &sourcePrinter{written: written, row: row, exact: true}
	text := p.print(t, 0)
	return text, p.exact
}

// sourcePrinter renders a type as an annotation would write it.
type sourcePrinter struct {
	written map[string]string // rigid inference name -> written name
	row     string            // the name for an unknown record rest
	exact   bool              // no inference variable was printed as `_`
}

func (p *sourcePrinter) print(t Type, depth int) string {
	if depth > 8 {
		p.exact = false
		return "_"
	}
	switch tt := t.(type) {
	case *TVar2:
		if w, ok := p.written[tt.Name]; ok {
			return w
		}
		p.exact = false
		return "_"
	case *TVar:
		p.exact = false
		return "_"
	case *TCon:
		return tt.Name
	case *TApp:
		args := make([]string, len(tt.Args))
		for i, a := range tt.Args {
			args[i] = p.print(a, depth+1)
		}
		if c, ok := tt.Constructor.(*TCon); ok && c.Name == "list" && len(args) == 1 {
			return "[" + args[0] + "]"
		}
		return fmt.Sprintf("%s[%s]", p.print(tt.Constructor, depth+1), strings.Join(args, ", "))
	case *TList:
		return "[" + p.print(tt.Element, depth+1) + "]"
	case *TArray:
		return "Array[" + p.print(tt.Element, depth+1) + "]"
	case *TTuple:
		elems := make([]string, len(tt.Elements))
		for i, e := range tt.Elements {
			elems[i] = p.print(e, depth+1)
		}
		return "(" + strings.Join(elems, ", ") + ")"
	case *TFunc2:
		params := make([]string, len(tt.Params))
		for i, x := range tt.Params {
			params[i] = p.print(x, depth+1)
		}
		out := "(" + strings.Join(params, ", ") + ") -> " + p.print(tt.Return, depth+1)
		// Only named effects: an inference tail (`...ρ3`) is not writable,
		// and an annotation without `! {...}` already leaves the row open.
		if tt.EffectRow != nil && len(tt.EffectRow.Labels) > 0 {
			labels := make([]string, 0, len(tt.EffectRow.Labels))
			for l := range tt.EffectRow.Labels {
				labels = append(labels, l)
			}
			sort.Strings(labels)
			out += " ! {" + strings.Join(labels, ", ") + "}"
		}
		return out
	case *TRecord:
		return p.record(tt.Fields, tt.Row, depth)
	case *TRecordOpen:
		return p.record(tt.Fields, tt.Row, depth)
	}
	if strings.ContainsAny(t.String(), "αβρε@") {
		p.exact = false
		return "_"
	}
	return t.String()
}

// record writes a record type with sorted fields and, when it is open, its
// rest: a signature row by its written name, any other as `| r`.
func (p *sourcePrinter) record(fields map[string]Type, rest Type, depth int) string {
	names := make([]string, 0, len(fields))
	for n := range fields {
		names = append(names, n)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, n := range names {
		parts = append(parts, n+": "+p.print(fields[n], depth+1))
	}
	out := "{" + strings.Join(parts, ", ")
	if rest != nil {
		tail := p.row
		if rv, ok := rest.(*RowVar); ok {
			if w, ok := p.written[rv.Name]; ok {
				tail = w
			}
		}
		out += " | " + tail
	}
	return out + "}"
}

// typoConsistent reports whether the type a misspelled variable probably
// meant (suggest) agrees with what the body needs (v): the same type, or the
// record alias whose fields the body uses. A hint that contradicts the body
// would send the author to a second error.
func (tc *CoreTypeChecker) typoConsistent(suggest string, v *rigidViolation) bool {
	if suggest == "" {
		return false
	}
	switch v.kind {
	case "literal":
		return suggest == "int" || suggest == "float"
	case "concrete":
	default:
		return false
	}
	switch g := v.got.(type) {
	case *TCon:
		return g.Name == suggest
	case *TApp:
		c, ok := g.Constructor.(*TCon)
		return ok && c.Name == suggest
	case *TRecord:
		return tc.recordAliasHas(suggest, g.Fields)
	case *TRecordOpen:
		return tc.recordAliasHas(suggest, g.Fields)
	}
	return false
}

// recordAliasHas reports whether a record type alias named name (from any
// module) has every field of fields.
func (tc *CoreTypeChecker) recordAliasHas(name string, fields map[string]Type) bool {
	for key, body := range tc.aliasEnv {
		if key != name && !strings.HasSuffix(key, "::"+name) {
			continue
		}
		var have map[string]Type
		switch b := body.(type) {
		case *TRecord:
			have = b.Fields
		case *TRecordOpen:
			have = b.Fields
		}
		if have == nil {
			continue
		}
		all := true
		for f := range fields {
			if _, ok := have[f]; !ok {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	return false
}

// SuggestTypeForVar returns the type in candidates an implicit signature
// variable named name was probably meant to be: the same name up to case and
// underscores, else one within edit distance 1 (names of 3-5 letters) or 2
// (longer). Single letters never match: `a` is a variable on purpose.
func SuggestTypeForVar(name string, candidates []string, distance func(a, b string) int) string {
	if len(name) < 2 {
		return ""
	}
	norm := func(s string) string { return strings.ReplaceAll(strings.ToLower(s), "_", "") }
	// Candidates are taken in the order given (callers list the primitives
	// first), so `strng` meets `string` before the module-less `String`.
	sorted := candidates
	for _, c := range sorted {
		if c != name && norm(c) == norm(name) {
			return c
		}
	}
	if len(name) < 3 || distance == nil {
		return ""
	}
	limit := 1
	if len(name) > 5 {
		limit = 2
	}
	best, bestD := "", limit+1
	for _, c := range sorted {
		if c == name {
			continue
		}
		if d := distance(strings.ToLower(name), strings.ToLower(c)); d < bestD {
			best, bestD = c, d
		}
	}
	return best
}
