package smt

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
)

// Obligations inside definitions (M-VERIFY-SOUNDNESS-COVERAGE: helpers, HOF
// lambdas, unrolled recursion).
//
// Runtime code the encoder emits as a define-fun — a helper's body, a level of
// an unrolled recursion, a map/filter/foldl specialisation and so its lambda —
// cannot hold a free $req_ok_ symbol. Its obligations become one Bool
// define-fun per obligation name over the same parameters:
//
//	$ob_<name>@<def>(params) := (=> assumptions obligations-of-<name>)
//
// where both sides come from the facts captured while the body was encoded
// (path-scoped, like the contracted body's own). Every call to <def> adds
// ($ob_<name>@<def> args) as an obligation of <name> at that call, so the
// check composes through any nesting and stays scoped by the caller's path.
//
// An unrolled chain bottoms out in a stub whose obligations are unknown. Its
// obligations are then checked only to the unrolling depth, and the names are
// reported as bounded — except for the contracted function's own recursion
// when it has no requires: every level is the same body checked for every
// input, so the bottom level follows by induction.

const obPrefix = "$ob_"

var (
	// activeDefObligations maps a define-fun to the obligation names it carries.
	activeDefObligations map[string][]string
	// activeObDefs maps an obligation define-fun to its text (for the
	// uninterpreted-dependency check).
	activeObDefs map[string]string
	// activeBoundedDefs maps a define-fun to the obligation names checked in it
	// only to an unrolling depth.
	activeBoundedDefs map[string][]string
	// activeBounded collects, for the definition being encoded (or the
	// contracted body), the bounded names its calls reach.
	activeBounded map[string]bool
	// activeLevelRequires maps a level of the contracted function's own
	// unrolling to the Bool define-fun of the function's requires.
	activeLevelRequires map[string]string
	// activeLevelOwner is the function those levels unroll.
	activeLevelOwner string
)

// beginObligations turns obligations on for one EncodeFunction: runtime code —
// the body and every helper, unrolled level and HOF specialisation it calls —
// carries them; contracts are encoded with them off (withoutObligations). It
// returns the set of names the body reaches only to an unrolling depth, and a
// func restoring the previous state.
func beginObligations() (map[string]bool, func()) {
	previousOn, previousDefObs, previousObDefs, previousBoundedDefs, previousBounded := obligationsOn, activeDefObligations, activeObDefs, activeBoundedDefs, activeBounded
	obligationsOn = true
	activeDefObligations = make(map[string][]string)
	activeObDefs = make(map[string]string)
	activeBoundedDefs = make(map[string][]string)
	bounded := make(map[string]bool)
	activeBounded = bounded
	return bounded, func() {
		obligationsOn, activeDefObligations, activeObDefs, activeBoundedDefs, activeBounded = previousOn, previousDefObs, previousObDefs, previousBoundedDefs, previousBounded
	}
}

func obName(name, def string) string { return obPrefix + name + "@" + def }

var reqOkName = regexp.MustCompile(regexp.QuoteMeta(reqOkPrefix) + `([^\s()]+)`)

// reqOkNames returns, sorted, the obligation names whose symbol occurs in facts.
func reqOkNames(facts ...string) []string {
	seen := map[string]bool{}
	for _, f := range facts {
		for _, m := range reqOkName.FindAllStringSubmatch(f, -1) {
			seen[m[1]] = true
		}
	}
	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// captureDefinition encodes the body of define-fun def, collecting its facts
// and the bounded obligation names its calls reach.
func captureDefinition(encode func() (string, error)) (string, string, map[string]bool, error) {
	previous := activeBounded
	bounded := map[string]bool{}
	activeBounded = bounded
	defer func() { activeBounded = previous }()
	value, facts, err := captureCallFacts(encode)
	return value, facts, bounded, err
}

// definitionObligations turns the facts captured from def's body into its
// obligation define-funs (returned) and the assumption-only facts (returned,
// for a helper's $facts_ condition). bounded names are recorded for def.
func definitionObligations(def string, defParams []FunctionParam, facts string, bounded map[string]bool) ([]string, string) {
	names := reqOkNames(facts)
	if len(bounded) > 0 && activeBoundedDefs != nil {
		activeBoundedDefs[def] = sortedKeys(bounded)
	}
	if len(names) == 0 || activeDefObligations == nil {
		return nil, facts
	}
	assumptions := substituteRequiresOk(facts, "")
	decls := make([]string, 0, len(names))
	for _, n := range names {
		body := fmt.Sprintf("(=> %s %s)", assumptions, substituteRequiresOk(facts, n))
		decl := buildDefineFun(obName(n, def), defParams, "Bool", body)
		activeObDefs[obName(n, def)] = decl
		decls = append(decls, decl)
	}
	activeDefObligations[def] = names
	return decls, assumptions
}

// aliasDefinitionObligations gives define-fun alias, defined as target over the
// same parameters, target's obligations.
func aliasDefinitionObligations(alias, target string, defParams []FunctionParam) []string {
	if b := activeBoundedDefs[target]; len(b) > 0 {
		activeBoundedDefs[alias] = b
	}
	names := activeDefObligations[target]
	if len(names) == 0 {
		return nil
	}
	args := make([]string, len(defParams))
	for i, p := range defParams {
		args[i] = p.Name
	}
	decls := make([]string, 0, len(names))
	for _, n := range names {
		call := obName(n, target)
		if len(args) > 0 {
			call = fmt.Sprintf("(%s %s)", call, strings.Join(args, " "))
		}
		decl := buildDefineFun(obName(n, alias), defParams, "Bool", call)
		activeObDefs[obName(n, alias)] = decl
		decls = append(decls, decl)
	}
	activeDefObligations[alias] = names
	return decls
}

// addDefinitionObligations adds, at a call to def, def's obligations on the
// call's encoded arguments.
func addDefinitionObligations(def string, encodedArgs []string) {
	if !obligationsOn {
		return
	}
	if activeBounded != nil {
		for _, n := range activeBoundedDefs[def] {
			activeBounded[n] = true
		}
	}
	for _, n := range activeDefObligations[def] {
		call := obName(n, def)
		if len(encodedArgs) > 0 {
			call = fmt.Sprintf("(%s %s)", call, strings.Join(encodedArgs, " "))
		}
		addObligation(n, call)
	}
}

// substituteRequiresOk replaces every $req_ok_ symbol in facts by true for
// obligation `on` and false for the others. A define-fun must not mention the
// symbol at all, even bound (leakedObligation guards that).
func substituteRequiresOk(facts, on string) string {
	return reqOkToken.ReplaceAllStringFunc(facts, func(tok string) string {
		if strings.TrimPrefix(tok, reqOkPrefix) == on {
			return "true"
		}
		return "false"
	})
}

var reqOkToken = regexp.MustCompile(regexp.QuoteMeta(reqOkPrefix) + `[^\s()]+`)

var obRef = regexp.MustCompile(regexp.QuoteMeta(obPrefix) + `[^\s()]+`)

// expandObligationDefs returns text followed by the text of every obligation
// define-fun it reaches.
func expandObligationDefs(text string) string {
	var b strings.Builder
	b.WriteString(text)
	seen := map[string]bool{}
	queue := obRef.FindAllString(text, -1)
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if seen[name] {
			continue
		}
		seen[name] = true
		def := activeObDefs[name]
		b.WriteString("\n")
		b.WriteString(def)
		queue = append(queue, obRef.FindAllString(def, -1)...)
	}
	return b.String()
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// selfCallRequires declares the contracted function's requires over its
// parameters and makes each call to a level of its unrolling carry it as an
// obligation named after the function. It reports whether the unrolling's
// obligations follow by induction: no requires, or one that is checked at
// every recursive call. The returned func restores the previous state.
func selfCallRequires(funcName string, meta *core.DeclMeta, params []FunctionParam, depth int, result *EncodeResult) (func(), bool) {
	previous, previousOwner := activeLevelRequires, activeLevelOwner
	restore := func() { activeLevelRequires, activeLevelOwner = previous, previousOwner }
	if !hasRequires(meta) {
		return restore, true
	}
	defParams, _, _ := withArgRenames(params, func() (string, error) { return "", nil })
	var decl string
	reqName, uncheckable := calleeRequiresDef(funcName, meta, params, defParams, &decl)
	if reqName == "" || uncheckable != "" {
		return restore, false
	}
	result.Declarations = append(result.Declarations, strings.TrimPrefix(decl, "\n"))
	activeLevelRequires = make(map[string]string, depth)
	for k := 0; k < depth; k++ {
		activeLevelRequires[fmt.Sprintf("%s_%d", funcName, k)] = reqName
	}
	activeLevelOwner = funcName
	return restore, true
}

// addLevelRequires adds, at a call to a level of the contracted function's
// unrolling, the function's requires on the call's arguments.
func addLevelRequires(level string, encodedArgs []string) {
	req := activeLevelRequires[level]
	if req == "" {
		return
	}
	call := req
	if len(encodedArgs) > 0 {
		call = fmt.Sprintf("(%s %s)", req, strings.Join(encodedArgs, " "))
	}
	addObligation(activeLevelOwner, call)
}
