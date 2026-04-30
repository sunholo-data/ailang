package main

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/smt"
	"github.com/sunholo-data/ailang/internal/types"
)

// TestFilterRecordAliasesForFunction_OnlyDirectlyUsed verifies that a function
// using only TableCell does not get unrelated aliases (DocMetadata, ParsedDocument)
// in its preamble.
//
// This is M1 of M-SMT-CROSS-MODULE-TYPES: extends the existing per-function
// ADT filter to also filter named record aliases. Without this filter, every
// function in document.ail receives ALL aliases from all imported modules,
// causing cascade Z3 errors when any single alias references an undeclared sort
// (e.g., ParsedDocument referencing Block which is filtered out for primitive-only
// functions).
func TestFilterRecordAliasesForFunction_OnlyDirectlyUsed(t *testing.T) {
	// All known aliases in the file: TableCell, DocMetadata, ParsedDocument, ExtractionResult
	allAliases := map[string]*types.TRecord{
		"TableCell": {
			Fields: map[string]types.Type{
				"text":    &types.TCon{Name: "string"},
				"colSpan": &types.TCon{Name: "int"},
				"rowSpan": &types.TCon{Name: "int"},
				"merged":  &types.TCon{Name: "bool"},
			},
			TypeName: "TableCell",
		},
		"DocMetadata": {
			Fields: map[string]types.Type{
				"title":     &types.TCon{Name: "string"},
				"pageCount": &types.TCon{Name: "int"},
			},
			TypeName: "DocMetadata",
		},
		"ParsedDocument": {
			Fields: map[string]types.Type{
				"format":   &types.TCon{Name: "string"},
				"metadata": &types.TCon{Name: "DocMetadata"},
				"blocks":   &types.TList{Element: &types.TCon{Name: "Block"}},
			},
			TypeName: "ParsedDocument",
		},
	}

	// Function: simpleCell(text: string) -> TableCell
	// Uses only TableCell. Should NOT pull in DocMetadata or ParsedDocument.
	params := []smt.FunctionParam{{Name: "text", Type: &types.TCon{Name: "string"}}}
	returnSort := "TableCell"

	got := filterRecordAliasesForFunction(params, returnSort, nil, allAliases, nil)

	if _, ok := got["TableCell"]; !ok {
		t.Errorf("expected TableCell to be retained, got: %v", keysOf(got))
	}
	if _, ok := got["DocMetadata"]; ok {
		t.Errorf("DocMetadata should NOT be in result for simpleCell — function does not reference it. Got: %v", keysOf(got))
	}
	if _, ok := got["ParsedDocument"]; ok {
		t.Errorf("ParsedDocument should NOT be in result for simpleCell — function does not reference it. Got: %v", keysOf(got))
	}
}

// TestFilterRecordAliasesForFunction_TransitiveClosure verifies that a function
// using ParsedDocument transitively pulls in DocMetadata (which ParsedDocument
// references via its metadata field). Block (an ADT) is OUT of scope for this
// test — it's filtered separately via filterADTTypesForFunction.
func TestFilterRecordAliasesForFunction_TransitiveClosure(t *testing.T) {
	allAliases := map[string]*types.TRecord{
		"DocMetadata": {
			Fields: map[string]types.Type{
				"title": &types.TCon{Name: "string"},
			},
			TypeName: "DocMetadata",
		},
		"ParsedDocument": {
			Fields: map[string]types.Type{
				"metadata": &types.TCon{Name: "DocMetadata"},
				"blocks":   &types.TList{Element: &types.TCon{Name: "Block"}},
			},
			TypeName: "ParsedDocument",
		},
		"UnrelatedAlias": {
			Fields: map[string]types.Type{
				"foo": &types.TCon{Name: "int"},
			},
			TypeName: "UnrelatedAlias",
		},
	}

	// Function returns ParsedDocument
	params := []smt.FunctionParam{}
	returnSort := "ParsedDocument"

	got := filterRecordAliasesForFunction(params, returnSort, nil, allAliases, nil)

	if _, ok := got["ParsedDocument"]; !ok {
		t.Errorf("ParsedDocument should be retained (used directly). Got: %v", keysOf(got))
	}
	if _, ok := got["DocMetadata"]; !ok {
		t.Errorf("DocMetadata should be retained (transitively via ParsedDocument.metadata). Got: %v", keysOf(got))
	}
	if _, ok := got["UnrelatedAlias"]; ok {
		t.Errorf("UnrelatedAlias should NOT be retained — not reachable from ParsedDocument. Got: %v", keysOf(got))
	}
}

// TestFilterRecordAliasesForFunction_EmptyInput is a sanity check: empty alias
// map returns empty result.
func TestFilterRecordAliasesForFunction_EmptyInput(t *testing.T) {
	got := filterRecordAliasesForFunction(nil, "Int", nil, nil, nil)
	if len(got) != 0 {
		t.Errorf("expected empty result for nil aliases, got: %v", keysOf(got))
	}
}

// TestFilterExtraDeclarationsForFunction verifies that inline-record declarations
// (like Record_blocks_kind from Block's SectionBlock variant) are filtered out
// when a function does not reference Block (or any record that contains Block-related
// fields). This is what currently causes simpleCell to fail: even though the ADT
// filter correctly drops Block, Record_blocks_kind sneaks in via ExtraDeclarations
// and references the (now-undeclared) Block sort.
func TestFilterExtraDeclarationsForFunction(t *testing.T) {
	// Synthetic SMT-LIB declarations for testing
	allDecls := []string{
		`(declare-datatype Record_blocks_kind ((mk_Record_blocks_kind (blocks (Seq Block)) (kind String))))`,
		`(declare-datatype Record_unrelated ((mk_Record_unrelated (foo Int))))`,
	}

	// seeds = {TableCell} — does not include Block or any inline-record sort
	seeds := map[string]bool{"TableCell": true}

	got := filterExtraDeclarationsForFunction(allDecls, seeds)

	for _, decl := range got {
		if containsSubstring(decl, "Record_blocks_kind") {
			t.Errorf("Record_blocks_kind should be filtered out (refs undeclared Block). Got decl: %s", decl)
		}
	}
}

// helpers
func keysOf(m map[string]*types.TRecord) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func containsSubstring(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
