package smt

import (
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
)

// Disambiguated record names (M-VERIFY-SOUNDNESS-COVERAGE). An anonymous
// record is named by its field names alone (Record_desc_key), so
// decisions' Choice(string, [{key, desc: string}]) and ChoiceR(Json, [{key,
// desc: Json}]) named one sort with two bodies, and every function needing it
// was skipped. Within one Verify, a field-name sort that two records share
// with different field sorts is named by its field sorts as well:
// Record_desc_key__String__String and Record_desc_key__Json__String. Every
// other record keeps its name. A record literal picks between the two by its
// checked type (recordForLiteral), as for an alias that shares an inline
// record's field names.

// activeRecordNameConflicts holds the field-name sorts that name more than one
// record in the current Verify; nil outside one.
var activeRecordNameConflicts map[string]bool

// disambiguatedRecordName is base, the field-name sort of rec, extended by its
// field sorts in field-name order when base is a conflicting name. A field
// with no sort leaves base: such a record is not declared at all.
func disambiguatedRecordName(base string, rec *types.TRecord) string {
	if !activeRecordNameConflicts[base] {
		return base
	}
	mangle := strings.NewReplacer("(", "", ")", "", " ", "_")
	parts := []string{base}
	for _, name := range sortedFieldNames(rec.Fields) {
		sort, err := MapType(rec.Fields[name])
		if err != nil {
			return base
		}
		parts = append(parts, mangle.Replace(sort))
	}
	return strings.Join(parts, "__")
}

// verifyTypes are the type tables Verify encodes with, from the verified
// module and its imports.
type verifyTypes struct {
	adts        map[string][]ADTVariant
	params      map[string][]string
	ambiguous   map[string]bool // ADTs and constructors two modules declare
	recordDecls []string
	aliases     map[string]*types.TRecord
	conflicts   map[string]bool // record sorts still naming two bodies
}

// extractVerifyTypes builds the type tables. When two records share a
// field-name sort, it extracts again with those names disambiguated by their
// field sorts; a name that still has two bodies is a conflict, and a function
// needing it is skipped. The caller resets activeRecordNameConflicts.
func extractVerifyTypes(surfaceAST *ast.File, modules map[string]VerifyModule) verifyTypes {
	vt, conflicts := extractVerifyTypesOnce(surfaceAST, modules)
	if len(conflicts) == 0 {
		return vt
	}
	activeRecordNameConflicts = conflicts
	vt, residual := extractVerifyTypesOnce(surfaceAST, modules)
	vt.conflicts = residual
	return vt
}

func extractVerifyTypesOnce(surfaceAST *ast.File, modules map[string]VerifyModule) (verifyTypes, map[string]bool) {
	root := ExtractADTTypesWithRecords(surfaceAST)
	vt := verifyTypes{
		adts:        root.ADTTypes,
		params:      root.TypeParams,
		ambiguous:   make(map[string]bool),
		recordDecls: root.RecordDecls,
		aliases:     root.RecordAliases,
	}
	recordConflicts := append([]string(nil), root.RecordConflicts...)
	typeOwner := make(map[string]string)
	owner := surfaceAST.Path
	if surfaceAST.Module != nil {
		owner = surfaceAST.Module.Path
	}
	for name := range vt.adts {
		typeOwner[name] = owner
	}
	// Imported modules too, so cross-module types (Block, XmlNode,
	// TableCell) are declared. In path order: the tables are first-wins.
	paths := make([]string, 0, len(modules))
	for path := range modules {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		mod := modules[path]
		if mod.File == nil {
			continue
		}
		modResult := ExtractADTTypesWithRecords(mod.File)
		modOwner := mod.File.Path
		if mod.File.Module != nil {
			modOwner = mod.File.Module.Path
		}
		for name, variants := range modResult.ADTTypes {
			if previous, exists := typeOwner[name]; exists && previous != modOwner {
				vt.ambiguous[name] = true
				for _, v := range variants {
					vt.ambiguous[v.Name] = true
				}
				for _, v := range vt.adts[name] {
					vt.ambiguous[v.Name] = true
				}
			}
			if _, exists := vt.adts[name]; !exists {
				typeOwner[name] = modOwner
				vt.adts[name] = variants
				vt.params[name] = modResult.TypeParams[name]
			}
		}
		vt.recordDecls = append(vt.recordDecls, modResult.RecordDecls...)
		recordConflicts = append(recordConflicts, modResult.RecordConflicts...)
		for name, rec := range modResult.RecordAliases {
			if _, exists := vt.aliases[name]; !exists {
				vt.aliases[name] = rec
			}
		}
	}
	return vt, conflictingRecordSorts(vt.recordDecls, recordConflicts)
}
