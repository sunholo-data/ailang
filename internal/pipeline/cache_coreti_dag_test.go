package pipeline

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
	"testing"

	srcast "github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
)

// gobRoundTripCoreTI is the oracle: what a cache hit decoded before the DAG codec.
func gobRoundTripCoreTI(t *testing.T, info types.CoreTypeInfo) (types.CoreTypeInfo, error) {
	t.Helper()
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(info); err != nil {
		return nil, err
	}
	var out types.CoreTypeInfo
	err := gob.NewDecoder(&buf).Decode(&out)
	return out, err
}

func dagRoundTripCoreTI(t *testing.T, info types.CoreTypeInfo) (types.CoreTypeInfo, []byte, error) {
	t.Helper()
	data, err := encodeCoreTIDAG(info)
	if err != nil {
		return nil, nil, err
	}
	out, err := decodeCoreTIDAG(data)
	return out, data, err
}

func assertCoreTIEquivalent(t *testing.T, info types.CoreTypeInfo) []byte {
	t.Helper()
	want, gobErr := gobRoundTripCoreTI(t, info)
	got, data, dagErr := dagRoundTripCoreTI(t, info)
	if (gobErr != nil) != (dagErr != nil) {
		t.Fatalf("codecs disagree on failure: gob=%v dag=%v", gobErr, dagErr)
	}
	if gobErr != nil {
		return nil
	}
	if !reflect.DeepEqual(got, want) {
		for id := range want {
			if !reflect.DeepEqual(got[id], want[id]) {
				t.Fatalf("node %d: dag %#v (%v), gob %#v (%v)", id, got[id], got[id], want[id], want[id])
			}
		}
		t.Fatalf("dag %#v, gob %#v", got, want)
	}
	return data
}

func ctiIntPtr(n int) *int { return &n }

// One value per concrete types.Type, with every field populated.
func coreTIVariantSamples() map[string]types.Type {
	tail := &types.RowVar{Name: "r", Kind: types.RecordRow}
	effRow := &types.Row{
		Kind:       types.EffectRow,
		Labels:     map[string]types.Type{"IO": types.TUnit, "Net": types.TUnit},
		Tail:       &types.RowVar{Name: "e", Kind: types.EffectRow},
		Budgets:    map[string]*int{"Net": ctiIntPtr(3)},
		MinBudgets: map[string]*int{"IO": ctiIntPtr(1)},
		Params:     map[string]map[string]string{"Net": {"host": "example.com"}},
		Provenance: map[string]srcast.Span{"IO": {Start: srcast.Pos{File: "a.ail", Line: 1, Column: 2}, End: srcast.Pos{File: "a.ail", Line: 1, Column: 9}}},
	}
	recRow := &types.Row{Kind: types.RecordRow, Labels: map[string]types.Type{"x": types.TInt}, Tail: tail}
	return map[string]types.Type{
		"TCon":        &types.TCon{Name: "Item"},
		"TVar":        &types.TVar{Name: "a"},
		"TVar2":       &types.TVar2{Name: "b", Kind: types.Star},
		"RowVar":      tail,
		"TList":       &types.TList{Element: types.TInt},
		"TArray":      &types.TArray{Element: types.TString},
		"TMap":        &types.TMap{Key: types.TString, Value: types.TInt},
		"TTuple":      &types.TTuple{Elements: []types.Type{types.TInt, types.TBool}},
		"TRecord":     &types.TRecord{Fields: map[string]types.Type{"x": types.TInt, "y": &types.TList{Element: types.TInt}}, Row: &types.TVar{Name: "r"}, TypeName: "Point"},
		"TRecordOpen": &types.TRecordOpen{Fields: map[string]types.Type{"x": types.TInt}, Row: &types.TVar{Name: "r"}},
		"TRecord2":    &types.TRecord2{Row: recRow},
		"TApp":        &types.TApp{Constructor: &types.TCon{Name: "Option"}, Args: []types.Type{types.TInt}},
		"TFunc2":      &types.TFunc2{ImplicitEffectContract: true, ConcreteEffectContract: true, Params: []types.Type{types.TInt, types.TString}, Return: types.TBool, EffectRow: effRow},
		"TLabelled":   &types.TLabelled{Inner: types.TInt, L: types.LabelBottom()},
		"Row":         effRow,
	}
}

func TestCoreTIDAG_EveryVariantMatchesGob(t *testing.T) {
	samples := coreTIVariantSamples()
	covered := map[string]bool{}
	for name, ty := range samples {
		covered[reflect.TypeOf(ty).Elem().Name()] = true
		t.Run(name, func(t *testing.T) {
			assertCoreTIEquivalent(t, types.CoreTypeInfo{1: ty, 2: &types.TList{Element: ty}})
		})
	}
	// Zero-valued variants: gob normalizes empty maps and slices; the DAG codec must too.
	for _, ty := range []types.Type{
		&types.TTuple{}, &types.TTuple{Elements: []types.Type{}}, &types.TRecord{}, &types.TRecord{Fields: map[string]types.Type{}},
		&types.TRecordOpen{}, &types.TRecord2{}, &types.TRecord2{Row: &types.Row{}}, &types.TApp{}, &types.TFunc2{},
		&types.TFunc2{EffectRow: &types.Row{Labels: map[string]types.Type{}}}, &types.Row{}, &types.TCon{}, &types.TList{},
	} {
		t.Run(fmt.Sprintf("zero/%T", ty), func(t *testing.T) {
			assertCoreTIEquivalent(t, types.CoreTypeInfo{7: ty})
		})
	}
	assertCoreTIEquivalent(t, types.CoreTypeInfo{})
	assertCoreTIEquivalent(t, nil)

	packages, err := parser.ParseDir(token.NewFileSet(), "../types", func(info os.FileInfo) bool { return !strings.HasSuffix(info.Name(), "_test.go") }, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range packages["types"].Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "Substitute" || fn.Recv == nil {
				continue
			}
			if star, ok := fn.Recv.List[0].Type.(*ast.StarExpr); ok && !covered[star.X.(*ast.Ident).Name] {
				t.Errorf("concrete types.Type %s has no DAG codec sample (add a case to coreTIEncoder.node and coreTIVariantSamples)", star.X.(*ast.Ident).Name)
			}
		}
	}
}

// Adding a field to a type struct must fail here until the codec carries it.
func TestCoreTIDAG_FieldCountsPinned(t *testing.T) {
	want := map[string]int{
		"TCon": 1, "TVar": 1, "TVar2": 2, "RowVar": 2, "TList": 1, "TArray": 1, "TMap": 2, "TTuple": 1,
		// TRecord's 4th field is groundMemo, a per-node cache (#1787) that is
		// deliberately not encoded: a decoded record recomputes it on demand.
		"TRecord": 4, "TRecordOpen": 2, "TRecord2": 1, "TApp": 2, "TFunc2": 5, "TLabelled": 2, "Row": 7,
	}
	for name, ty := range coreTIVariantSamples() {
		if got := reflect.TypeOf(ty).Elem().NumField(); got != want[name] {
			t.Errorf("%s has %d fields, the DAG codec encodes %d: update coreTIEncoder.node, coreTIDecoder.build and this pin", name, got, want[name])
		}
	}
}

func TestCoreTIDAG_TypedNilFailsLikeGob(t *testing.T) {
	var nilRow *types.Row
	for _, ty := range []types.Type{nilRow, &types.TList{Element: (*types.TCon)(nil)}} {
		info := types.CoreTypeInfo{1: ty}
		_, gobErr := gobRoundTripCoreTI(t, info)
		_, dagErr := encodeCoreTIDAG(info)
		if gobErr == nil || dagErr == nil {
			t.Fatalf("%#v: want both codecs to refuse a typed nil, gob=%v dag=%v", ty, gobErr, dagErr)
		}
	}
	// A nil *Row/*RowVar in a concrete-typed field is an ordinary nil.
	assertCoreTIEquivalent(t, types.CoreTypeInfo{1: &types.TFunc2{Return: types.TInt}, 2: &types.Row{Kind: types.EffectRow}})
}

// The #1778 shape: one record alias expanded under every node that mentions it.
// Depth 5, fan-out 3: 243 leaf records and ~480 type nodes per entry.
func expandedRecordCoreTI(entries int) types.CoreTypeInfo {
	leaf := &types.TRecord{Fields: map[string]types.Type{"id": types.TString, "n": types.TInt}, TypeName: "Leaf"}
	var rec types.Type = leaf
	for depth := 0; depth < 5; depth++ {
		fields := map[string]types.Type{}
		for i := 0; i < 3; i++ {
			fields[fmt.Sprintf("f%d", i)] = &types.TList{Element: rec}
		}
		rec = &types.TRecord{Fields: fields, TypeName: fmt.Sprintf("Level%d", depth)}
	}
	info := make(types.CoreTypeInfo, entries)
	for i := 0; i < entries; i++ {
		// Fresh pointers each time, as substitution produces: sharing must come from structure.
		info[uint64(i)] = &types.TFunc2{Params: []types.Type{cloneCoreTIType(rec)}, Return: types.TInt, EffectRow: &types.Row{Kind: types.EffectRow}}
	}
	return info
}

func cloneCoreTIType(t types.Type) types.Type {
	switch t := t.(type) {
	case *types.TRecord:
		fields := make(map[string]types.Type, len(t.Fields))
		for k, v := range t.Fields {
			fields[k] = cloneCoreTIType(v)
		}
		return &types.TRecord{Fields: fields, Row: t.Row, TypeName: t.TypeName}
	case *types.TList:
		return &types.TList{Element: cloneCoreTIType(t.Element)}
	default:
		return t
	}
}

func TestCoreTIDAG_SharedSubtreesCollapse(t *testing.T) {
	info := expandedRecordCoreTI(200)
	data := assertCoreTIEquivalent(t, info)
	var plain bytes.Buffer
	if err := gob.NewEncoder(&plain).Encode(info); err != nil {
		t.Fatal(err)
	}
	t.Logf("200 entries: plain gob %d bytes, dag %d bytes", plain.Len(), len(data))
	if len(data)*100 > plain.Len() {
		t.Fatalf("dag %d bytes is not 100x smaller than plain gob %d bytes", len(data), plain.Len())
	}
	big, err := encodeCoreTIDAG(expandedRecordCoreTI(10000))
	if err != nil {
		t.Fatal(err)
	}
	if len(big) > 64<<10 {
		t.Fatalf("10k entries over one nested record encode to %d bytes, want < 64 KiB", len(big))
	}
}

func TestCoreTIDAG_Deterministic(t *testing.T) {
	info := expandedRecordCoreTI(50)
	for name, ty := range coreTIVariantSamples() {
		info[uint64(1000+len(name))] = ty
	}
	first, err := encodeCoreTIDAG(info)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		again, err := encodeCoreTIDAG(info)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(first, again) {
			t.Fatal("two encodes of one CoreTypeInfo differ")
		}
	}
}

func encodeCoreTIBlobForTest(t *testing.T, blob coreTIBlob) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(&blob); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestCoreTIDAG_MalformedFailsWithoutPanic(t *testing.T) {
	valid, err := encodeCoreTIDAG(types.CoreTypeInfo{1: &types.TList{Element: types.TInt}})
	if err != nil {
		t.Fatal(err)
	}
	con := coreTINode{Tag: ctiTagCon, Name: "int"}
	cases := map[string][]byte{
		"truncated":     valid[:len(valid)/2],
		"garbage":       []byte("not a gob"),
		"format":        encodeCoreTIBlobForTest(t, coreTIBlob{Format: 9}),
		"unknown tag":   encodeCoreTIBlobForTest(t, coreTIBlob{Format: coreTIDAGFormat, Nodes: []coreTINode{{Tag: 200}}}),
		"zero tag":      encodeCoreTIBlobForTest(t, coreTIBlob{Format: coreTIDAGFormat, Nodes: []coreTINode{con, {}}}),
		"self ref":      encodeCoreTIBlobForTest(t, coreTIBlob{Format: coreTIDAGFormat, Nodes: []coreTINode{{Tag: ctiTagList, A: 1}}}),
		"forward ref":   encodeCoreTIBlobForTest(t, coreTIBlob{Format: coreTIDAGFormat, Nodes: []coreTINode{{Tag: ctiTagList, A: 2}, con}}),
		"negative ref":  encodeCoreTIBlobForTest(t, coreTIBlob{Format: coreTIDAGFormat, Nodes: []coreTINode{{Tag: ctiTagTuple, Kids: []int32{-4}}}}),
		"root range":    encodeCoreTIBlobForTest(t, coreTIBlob{Format: coreTIDAGFormat, Nodes: []coreTINode{con}, IDs: []uint64{1}, Roots: []int32{5}}),
		"roots count":   encodeCoreTIBlobForTest(t, coreTIBlob{Format: coreTIDAGFormat, Nodes: []coreTINode{con}, IDs: []uint64{1, 2}, Roots: []int32{1}}),
		"keys mismatch": encodeCoreTIBlobForTest(t, coreTIBlob{Format: coreTIDAGFormat, Nodes: []coreTINode{con, {Tag: ctiTagRecord, Keys: []string{"a", "b"}, Kids: []int32{1}}}}),
		"row kind":      encodeCoreTIBlobForTest(t, coreTIBlob{Format: coreTIDAGFormat, Nodes: []coreTINode{con, {Tag: ctiTagRecord2, A: 1}}}),
		"row tail kind": encodeCoreTIBlobForTest(t, coreTIBlob{Format: coreTIDAGFormat, Nodes: []coreTINode{con, {Tag: ctiTagRow, A: 1}}}),
		"empty raw":     encodeCoreTIBlobForTest(t, coreTIBlob{Format: coreTIDAGFormat, Nodes: []coreTINode{{Tag: ctiTagRaw}}}),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeCoreTIDAG(data); err == nil {
				t.Fatal("malformed blob decoded without error")
			}
		})
	}
}
