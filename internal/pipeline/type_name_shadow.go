package pipeline

import (
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
)

// TC_TYPE_SHADOW_001 codes the use of an imported type alias whose body names a
// type that the using module declares itself (M-TYPE-NAME-SHADOW).
const TC_TYPE_SHADOW_001 = "TC_TYPE_SHADOW_001"

// shadowLocalTypeNames makes a module's OWN type declarations win over any
// same-named type reaching it from another module (M-TYPE-NAME-SHADOW), and
// withholds imported aliases that the local names would capture.
//
// 1. Shadowing. compileFreshModule registered local aliases on the type checker
// and then re-registered every imported alias over them, and the imported set
// holds every alias of every direct import plus (for any module with at least
// one import) every alias of every module compiled so far. So a module exporting
// `type Row` silently replaced the body of `type Row` in every module compiled
// after it (Daneel, 2026-09-26). Local names are now removed from the imported
// alias, alias-param and ADT-param tables — for every local type kind, so a
// local sum type Row is not expanded by an imported record alias Row either.
//
// 2. Capture. Interface alias bodies still name other types by BARE name (links'
// `Seen = {rows: [Row]}`), so in a module with its own Row, expanding Seen would
// resolve to the LOCAL Row. Such aliases — and, to a fixpoint, aliases whose
// bodies name them — are withheld from the alias env and registered as captured:
// USING one is a loud TC_TYPE_SHADOW_001 naming both definitions; merely having
// it in reach is fine. Closing alias bodies over their defining module removes
// the need for this (M2 of design_docs/planned/v0_44_0/m-type-name-shadow-and-cache.md).
func shadowLocalTypeNames(imports *moduleImports, file *ast.File, modID string) {
	if file == nil {
		return
	}
	local := make(map[string]bool)
	for _, name := range localTypeNames(file) {
		local[name] = true
		delete(imports.ImportedTypeAliases, name)
		delete(imports.ImportedAliasParams, name)
		delete(imports.ImportedADTTypeParams, name)
	}
	if len(local) == 0 {
		return
	}

	// captured[alias] = the name in its body that is shadowed here (a local type,
	// or another captured alias). Fixpoint so chains are caught; sorted for
	// deterministic messages.
	captured := make(map[string]string)
	for changed := true; changed; {
		changed = false
		for _, name := range sortedTypeKeys(imports.ImportedTypeAliases) {
			if _, done := captured[name]; done {
				continue
			}
			refs := make(map[string]bool)
			collectTConNames(imports.ImportedTypeAliases[name], refs)
			for _, ref := range sortedBoolKeys(refs) {
				_, viaCaptured := captured[ref]
				if local[ref] || viaCaptured {
					captured[name] = ref
					changed = true
					break
				}
			}
		}
	}

	for name, ref := range captured {
		imports.CapturedAliases[name] = capturedAliasMsg(name, imports.ImportedAliasOrigin[name], ref, local[ref], modID)
		delete(imports.ImportedTypeAliases, name)
		delete(imports.ImportedAliasParams, name)
	}
}

// capturedAliasMsg names both definitions: the imported alias with its module,
// and the local declaration that shadows the type its body refers to.
func capturedAliasMsg(alias, origin, ref string, refIsLocal bool, modID string) string {
	if origin == "" {
		origin = "an imported module"
	}
	if refIsLocal {
		return fmt.Sprintf("%s: type %s (from module %s) refers to a type named %s, but module %s declares its own type %s, "+
			"which shadows the %s that %s means — expanding %s here would silently use the wrong %s.\n"+
			"  Suggestion: type names are not yet module-qualified; rename one of the two %s types "+
			"(e.g. the one in %s to a more specific name such as %s%s).",
			TC_TYPE_SHADOW_001, alias, origin, ref, modID, ref, ref, origin, alias, ref, ref, modID, lastSegment(modID), ref)
	}
	return fmt.Sprintf("%s: type %s (from module %s) refers to type %s, which cannot be expanded in module %s "+
		"because its own body names a type that %s declares itself.\n"+
		"  Suggestion: type names are not yet module-qualified; rename the conflicting local type in %s.",
		TC_TYPE_SHADOW_001, alias, origin, ref, modID, modID, modID)
}

// localTypeNames lists the names of every type declared in a file, exported or
// not, of any kind (record, alias, sum type, newtype).
func localTypeNames(file *ast.File) []string {
	var names []string
	for _, decls := range [][]ast.Node{file.Decls, file.Statements} {
		for _, decl := range decls {
			if td, ok := decl.(*ast.TypeDecl); ok && td != nil {
				names = append(names, td.Name)
			}
		}
	}
	return names
}

func sortedTypeKeys(m map[string]types.Type) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedBoolKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// lastSegment returns the final path segment of a module ID, capitalised, for
// a rename suggestion (daneel_links → Daneel_links; pkg/a/types → Types).
func lastSegment(modID string) string {
	seg := modID
	for i := len(modID) - 1; i >= 0; i-- {
		if modID[i] == '/' {
			seg = modID[i+1:]
			break
		}
	}
	if seg == "" {
		return ""
	}
	if c := seg[0]; c >= 'a' && c <= 'z' {
		return string(c-'a'+'A') + seg[1:]
	}
	return seg
}
