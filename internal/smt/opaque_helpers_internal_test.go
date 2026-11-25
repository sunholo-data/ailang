package smt

import (
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// A record update whose type is unknown picks the one declared record that has
// every updated field, deterministically; with several it is a skip, not a
// guess (the first in map order built the wrong record: a Z3 ERROR).
func TestLookupRecordForUpdateByFields(t *testing.T) {
	previous := activeRecordTypes
	defer func() { activeRecordTypes = previous }()
	activeRecordTypes = map[string]*RecordTypeInfo{
		"InlineRun":              {SortName: "InlineRun", FieldSorts: map[string]string{"text": "String", "href": "String"}},
		"Record_level_runs_text": {SortName: "Record_level_runs_text", FieldSorts: map[string]string{"text": "String", "level": "Int", "runs": "(Seq InlineRun)"}},
	}
	ru := &core.RecordUpdate{Base: &core.Var{Name: "n"}, Updates: map[string]core.CoreExpr{"href": &core.Lit{Kind: core.StringLit, Value: "x"}}}
	info, err := lookupRecordForUpdate(ru)
	if err != nil || info.SortName != "InlineRun" {
		t.Fatalf("unique candidate: %v %v", info, err)
	}
	ru.Updates = map[string]core.CoreExpr{"text": &core.Lit{Kind: core.StringLit, Value: "x"}}
	for i := 0; i < 20; i++ {
		if _, err := lookupRecordForUpdate(ru); err == nil || !strings.Contains(err.Error(), "InlineRun, Record_level_runs_text") {
			t.Fatalf("ambiguous update: %v", err)
		}
	}
}

// A helper's uninterpreted function whose signature names an undeclared sort
// is a skip (ErrUnresolvableTypes), not a Z3 ERROR.
func TestValidateDeclarationsChecksDeclareFunSignatures(t *testing.T) {
	ctx := NewSMTContext()
	if err := validateDeclarations([]string{`(declare-fun $spec_collect (Int) Record_count_rest)`}, ctx); !errors.Is(err, ErrUnresolvableTypes) ||
		!strings.Contains(err.Error(), `"Record_count_rest" in the signature of $spec_collect`) {
		t.Fatalf("undeclared result sort: %v", err)
	}
	if err := validateDeclarations([]string{`(declare-fun $spec_f ((Seq Missing) Int) Int)`}, ctx); !errors.Is(err, ErrUnresolvableTypes) {
		t.Fatalf("undeclared parameter sort: %v", err)
	}
	ok := []string{`(declare-datatype Box ((mk_Box (v Int))))`, `(declare-fun $spec_g ((Seq Box) String) (Option Int))`, `(declare-datatype Option (par (a) ((Some (Some_0 a)) (None))))`}
	if err := validateDeclarations([]string{ok[2], ok[0], ok[1]}, ctx); err != nil {
		t.Fatalf("declared sorts: %v", err)
	}
}
