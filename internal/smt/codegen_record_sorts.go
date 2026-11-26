package smt

import (
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// beginRecordSorts makes every record sort this function's script will
// declare findable by its field set, and returns the alias declarations still
// to be emitted. It runs BEFORE callee resolution: helper bodies are encoded
// there, and a record literal in a helper (std/json's kv builds
// {key, value}) needs the sort as much as one in the verified body does.
// Before, the tables were created after the helpers were encoded, so every
// helper that built a record was dropped.
//
// Two sources, in priority order: named record aliases (type TableCell =
// {...}), then the inline records of ADT constructor fields
// (ExtraDeclarations, e.g. Json's Record_key_value). Those are emitted by the
// ADT pass with their ADT; here they are only registered.
func beginRecordSorts(opts []EncodeFunctionOpts) ([]pendingDecl, func(), error) {
	activeRecordTypes = make(map[string]*RecordTypeInfo)
	activeFieldSetToSort = make(map[string]string)
	activeShadowedRecords = make(map[string][]*RecordTypeInfo)
	done := func() { activeRecordTypes = nil; activeFieldSetToSort = nil; activeShadowedRecords = nil }
	if len(opts) == 0 {
		return nil, done, nil
	}
	var aliasDecls []pendingDecl
	aliasNames := make([]string, 0, len(opts[0].RecordTypeAliases))
	for name := range opts[0].RecordTypeAliases {
		aliasNames = append(aliasNames, name)
	}
	sort.Strings(aliasNames)
	for _, aliasName := range aliasNames {
		rec := opts[0].RecordTypeAliases[aliasName]
		fieldSorts, err := MapRecordFields(&types.TRecord{Fields: rec.Fields, TypeName: aliasName})
		if err != nil {
			done()
			return nil, func() {}, fmt.Errorf("%w: record alias %q: %v", ErrUnresolvableTypes, aliasName, err)
		}
		registerRecordAlias(aliasName, fieldSorts)
		aliasDecls = append(aliasDecls, pendingDecl{sortName: aliasName, decl: DeclareRecordDatatype(aliasName, fieldSorts)})
	}
	for _, decl := range opts[0].ExtraDeclarations {
		info := recordInfoFromDecl(decl)
		if info == nil {
			continue
		}
		key := strings.Join(info.FieldNames, ",")
		if owner, taken := activeFieldSetToSort[key]; taken {
			if owner == info.SortName {
				continue // the same sort declared twice
			}
			// A named alias, or another inline record with the same field
			// names (record_names.go), is found first; a literal tells them
			// apart by its type (recordForLiteral). Each sort is a candidate
			// once, however often it is declared.
			if _, seen := activeRecordTypes[info.SortName]; !seen {
				activeShadowedRecords[key] = append(activeShadowedRecords[key], info)
				activeRecordTypes[info.SortName] = info
			}
			continue
		}
		activeRecordTypes[info.SortName] = info
		activeFieldSetToSort[key] = info.SortName
	}
	return aliasDecls, done, nil
}

// activeShadowedRecords holds, by field set, the inline records of ADT fields
// whose field names a named alias already claims: std/json's {key, value:
// Json} next to a package's `type A2UIProp = {key: string, value: string}`.
var activeShadowedRecords map[string][]*RecordTypeInfo

// recordForLiteral is the record sort a literal builds. By field names alone
// it is ambiguous when an alias and an inline record share them, and taking
// the alias built an A2UIProp holding a Json, which Z3 rejected (an ERROR,
// not a skip). The literal's checked type then picks: its name, else its
// field sorts. A literal it cannot pick for is a skip.
func recordForLiteral(rec *core.Record) (*RecordTypeInfo, error) {
	primary := lookupRecordByFields(rec.Fields)
	names := fieldNamesFromExprMap(rec.Fields)
	shadowed := activeShadowedRecords[strings.Join(names, ",")]
	if primary == nil || len(shadowed) == 0 {
		return primary, nil
	}
	candidates := append([]*RecordTypeInfo{primary}, shadowed...)
	if t, ok := hofExprType(rec).(*types.TRecord); ok {
		for _, c := range candidates {
			if t.TypeName != "" && c.SortName == t.TypeName {
				return c, nil
			}
		}
		if sorts, err := MapRecordFields(t); err == nil {
			var match []*RecordTypeInfo
			for _, c := range candidates {
				if maps.Equal(c.FieldSorts, sorts) {
					match = append(match, c)
				}
			}
			if len(match) == 1 {
				return match[0], nil
			}
		}
	}
	sortNames := make([]string, len(candidates))
	for i, c := range candidates {
		sortNames[i] = c.SortName
	}
	return nil, fmt.Errorf("%w: record construction: fields %v match more than one declared record (%s), and the literal's type does not pick one", ErrUnsupportedConstruct, names, strings.Join(sortNames, ", "))
}

// recordInfoFromDecl reads a record sort back from its declaration,
// `(declare-datatype R ((mk_R (f1 S1) (f2 S2))))` as DeclareRecordDatatype
// writes it. Anything else (an ADT, a parametric datatype) is nil.
func recordInfoFromDecl(decl string) *RecordTypeInfo {
	es, err := ParseSExprs(decl)
	if err != nil || len(es) != 1 || es[0].Head() != "declare-datatype" || len(es[0].List) != 3 {
		return nil
	}
	name, ctors := es[0].List[1].Atom, es[0].List[2]
	if name == "" || len(ctors.List) != 1 {
		return nil
	}
	ctor := ctors.List[0]
	if ctor.Head() != RecordConstructorName(name) {
		return nil
	}
	info := &RecordTypeInfo{SortName: name, CtorName: ctor.Head(), FieldSorts: map[string]string{}}
	for _, f := range ctor.List[1:] {
		if len(f.List) != 2 || f.List[0].Atom == "" {
			return nil
		}
		info.FieldSorts[f.List[0].Atom] = f.List[1].String()
	}
	info.FieldNames = SortedFieldNamesStr(info.FieldSorts)
	return info
}

// recordLiteralSeed stands for a record literal's field set in the demand
// closure's seeds. It cannot collide with a sort name ("{" is not a symbol
// character); resolveRecordLiteralSeeds replaces it with the sort it names.
func recordLiteralSeed(fields map[string]core.CoreExpr) string {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return "{" + strings.Join(names, ",") + "}"
}

// resolveRecordLiteralSeeds replaces each record-literal seed with the sort
// declared for that field set: a named alias first (as beginRecordSorts
// registers them), else an inline record of an ADT field. A literal reached
// only through a helper body or a foldl accumulator otherwise seeded nothing,
// the alias was filtered out, and the literal had no sort. A field set no
// sort declares is dropped; encodeRecord then reports it.
func resolveRecordLiteralSeeds(seeds map[string]bool, aliases map[string]*types.TRecord, extraDecls []string) {
	var literals []string
	for s := range seeds {
		if strings.HasPrefix(s, "{") {
			literals = append(literals, s)
		}
	}
	if len(literals) == 0 {
		return
	}
	// Two inline records may share a field set once their names carry their
	// field sorts (record_names.go): the literal may build either, so both
	// are seeded.
	bySet := map[string][]string{}
	for _, decl := range extraDecls {
		if info := recordInfoFromDecl(decl); info != nil {
			key := strings.Join(info.FieldNames, ",")
			bySet[key] = append(bySet[key], info.SortName)
		}
	}
	for name, rec := range aliases {
		names := make([]string, 0, len(rec.Fields))
		for f := range rec.Fields {
			names = append(names, f)
		}
		sort.Strings(names)
		bySet[strings.Join(names, ",")] = []string{name} // an alias wins
	}
	for _, lit := range literals {
		delete(seeds, lit)
		for _, name := range bySet[strings.TrimSuffix(strings.TrimPrefix(lit, "{"), "}")] {
			seeds[name] = true
		}
	}
}
