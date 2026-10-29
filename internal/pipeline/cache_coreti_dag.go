package pipeline

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
)

// CoreTypeInfo is encoded as a hash-consed DAG (M-CORETI-SHARED-ENCODING, #1778).
// gob does not preserve pointer sharing, so a plain encode writes every expanded
// alias body again under each node that mentions it: motoko's session.ail went
// from 29 MB to 188 MB when alias bodies were closed (#1706), while the same
// blob holds only ~2.4k distinct subtrees. Here every distinct subtree is
// written once and referenced by index.
//
// The decode of an encoded value is reflect.DeepEqual to a plain gob round trip
// of it (the tests hold both codecs to that), so a cache hit behaves as before.
// Decoded types share pointers between entries; nothing in the pipeline mutates
// a CoreTypeInfo type in place.

const coreTIDAGFormat = 1

// Node tags. Zero is never a valid tag, so a zero node fails decode.
const (
	ctiTagCon uint8 = iota + 1
	ctiTagVar
	ctiTagVar2
	ctiTagRowVar
	ctiTagList
	ctiTagArray
	ctiTagMap
	ctiTagTuple
	ctiTagRecord
	ctiTagRecordOpen
	ctiTagRecord2
	ctiTagApp
	ctiTagFunc2
	ctiTagLabelled
	ctiTagRow
	ctiTagRaw
)

// Node refs are index+1; 0 is nil. Every child ref is below its parent's own ref.
type coreTINode struct {
	Tag        uint8
	Name       string
	Kind       types.Kind
	A, B       int32
	Kids       []int32
	Keys       []string
	Flags      uint8
	Label      []byte // types.MarshalLabel, as TLabelled's own GobEncode does
	Budgets    map[string]*int
	MinBudgets map[string]*int
	Params     map[string]map[string]string
	Provenance map[string]ast.Span
	Raw        types.Type
}

type coreTIBlob struct {
	Format uint8
	Nodes  []coreTINode
	IDs    []uint64
	Roots  []int32
}

const (
	ctiFlagImplicitEffect uint8 = 1 << iota
	ctiFlagConcreteEffect
	// gob keeps an empty non-nil map distinct from nil (it collapses only empty
	// slices), so an empty Fields/Labels map is flagged rather than dropped.
	ctiFlagEmptyMap
)

type coreTIEncoder struct {
	nodes []coreTINode
	byKey map[string]int32
	byPtr map[types.Type]int32
}

func encodeCoreTIDAG(info types.CoreTypeInfo) ([]byte, error) {
	e := &coreTIEncoder{byKey: make(map[string]int32), byPtr: make(map[types.Type]int32)}
	blob := coreTIBlob{Format: coreTIDAGFormat, IDs: make([]uint64, 0, len(info))}
	for id := range info {
		blob.IDs = append(blob.IDs, id)
	}
	sort.Slice(blob.IDs, func(i, j int) bool { return blob.IDs[i] < blob.IDs[j] })
	blob.Roots = make([]int32, len(blob.IDs))
	for i, id := range blob.IDs {
		ref, err := e.ref(info[id])
		if err != nil {
			return nil, fmt.Errorf("coretypeinfo node %d: %w", id, err)
		}
		blob.Roots[i] = ref
	}
	blob.Nodes = e.nodes
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(&blob)
	return buf.Bytes(), err
}

// ref encodes t (children first) and returns its node ref.
func (e *coreTIEncoder) ref(t types.Type) (int32, error) {
	if t == nil {
		return 0, nil
	}
	if v := reflect.ValueOf(t); v.Kind() == reflect.Pointer && v.IsNil() {
		// gob cannot encode a nil pointer held in an interface either.
		return 0, fmt.Errorf("cannot encode nil %T held in a types.Type", t)
	}
	if r, ok := e.byPtr[t]; ok {
		return r, nil
	}
	n, err := e.node(t)
	if err != nil {
		return 0, err
	}
	key := n.key()
	r, ok := e.byKey[key]
	if !ok {
		e.nodes = append(e.nodes, n)
		r = int32(len(e.nodes))
		e.byKey[key] = r
	}
	if isPointerType(t) {
		e.byPtr[t] = r
	}
	return r, nil
}

func isPointerType(t types.Type) bool {
	return reflect.TypeOf(t).Kind() == reflect.Pointer
}

func (e *coreTIEncoder) refs(ts []types.Type) ([]int32, error) {
	if len(ts) == 0 {
		return nil, nil
	}
	out := make([]int32, len(ts))
	for i, t := range ts {
		r, err := e.ref(t)
		if err != nil {
			return nil, err
		}
		out[i] = r
	}
	return out, nil
}

func (e *coreTIEncoder) fields(n *coreTINode, m map[string]types.Type) ([]string, []int32, error) {
	if m != nil && len(m) == 0 {
		n.Flags |= ctiFlagEmptyMap
	}
	if len(m) == 0 {
		return nil, nil, nil
	}
	keys := sortedTypeKeys(m)
	kids := make([]int32, len(keys))
	for i, k := range keys {
		r, err := e.ref(m[k])
		if err != nil {
			return nil, nil, err
		}
		kids[i] = r
	}
	return keys, kids, nil
}

func (e *coreTIEncoder) row(r *types.Row) (int32, error) {
	if r == nil {
		return 0, nil
	}
	return e.ref(r)
}

func (e *coreTIEncoder) node(t types.Type) (coreTINode, error) {
	var err error
	switch t := t.(type) {
	case *types.TCon:
		return coreTINode{Tag: ctiTagCon, Name: t.Name}, nil
	case *types.TVar:
		return coreTINode{Tag: ctiTagVar, Name: t.Name}, nil
	case *types.TVar2:
		return coreTINode{Tag: ctiTagVar2, Name: t.Name, Kind: t.Kind}, nil
	case *types.RowVar:
		return coreTINode{Tag: ctiTagRowVar, Name: t.Name, Kind: t.Kind}, nil
	case *types.TList:
		n := coreTINode{Tag: ctiTagList}
		n.A, err = e.ref(t.Element)
		return n, err
	case *types.TArray:
		n := coreTINode{Tag: ctiTagArray}
		n.A, err = e.ref(t.Element)
		return n, err
	case *types.TMap:
		n := coreTINode{Tag: ctiTagMap}
		if n.A, err = e.ref(t.Key); err != nil {
			return n, err
		}
		n.B, err = e.ref(t.Value)
		return n, err
	case *types.TTuple:
		n := coreTINode{Tag: ctiTagTuple}
		n.Kids, err = e.refs(t.Elements)
		return n, err
	case *types.TRecord:
		n := coreTINode{Tag: ctiTagRecord, Name: t.TypeName}
		if n.Keys, n.Kids, err = e.fields(&n, t.Fields); err != nil {
			return n, err
		}
		n.A, err = e.ref(t.Row)
		return n, err
	case *types.TRecordOpen:
		n := coreTINode{Tag: ctiTagRecordOpen}
		if n.Keys, n.Kids, err = e.fields(&n, t.Fields); err != nil {
			return n, err
		}
		n.A, err = e.ref(t.Row)
		return n, err
	case *types.TRecord2:
		n := coreTINode{Tag: ctiTagRecord2}
		n.A, err = e.row(t.Row)
		return n, err
	case *types.TApp:
		n := coreTINode{Tag: ctiTagApp}
		if n.A, err = e.ref(t.Constructor); err != nil {
			return n, err
		}
		n.Kids, err = e.refs(t.Args)
		return n, err
	case *types.TFunc2:
		n := coreTINode{Tag: ctiTagFunc2}
		if t.ImplicitEffectContract {
			n.Flags |= ctiFlagImplicitEffect
		}
		if t.ConcreteEffectContract {
			n.Flags |= ctiFlagConcreteEffect
		}
		if n.Kids, err = e.refs(t.Params); err != nil {
			return n, err
		}
		if n.A, err = e.ref(t.Return); err != nil {
			return n, err
		}
		n.B, err = e.row(t.EffectRow)
		return n, err
	case *types.TLabelled:
		n := coreTINode{Tag: ctiTagLabelled}
		if n.Label, err = types.MarshalLabel(t.L); err != nil {
			return n, fmt.Errorf("TLabelled: encoding label: %w", err)
		}
		n.A, err = e.ref(t.Inner)
		return n, err
	case *types.Row:
		n := coreTINode{Tag: ctiTagRow, Kind: t.Kind, Budgets: t.Budgets, MinBudgets: t.MinBudgets, Params: t.Params, Provenance: t.Provenance}
		if n.Keys, n.Kids, err = e.fields(&n, t.Labels); err != nil {
			return n, err
		}
		if t.Tail != nil {
			n.A, err = e.ref(t.Tail)
		}
		return n, err
	default:
		// An implementor this codec does not know keeps today's plain gob encoding.
		return coreTINode{Tag: ctiTagRaw, Raw: t}, nil
	}
}

// key is exact: two nodes share a key only if they decode to equal values.
func (n *coreTINode) key() string {
	var b strings.Builder
	b.WriteString(strconv.Itoa(int(n.Tag)))
	b.WriteByte('|')
	b.WriteString(strconv.Quote(n.Name))
	fmt.Fprintf(&b, "|%#v|%d|%d|%d|%v|%q", n.Kind, n.A, n.B, n.Flags, n.Kids, n.Keys)
	if n.Label != nil {
		fmt.Fprintf(&b, "|L%q", n.Label)
	}
	writeBudgetKey(&b, "B", n.Budgets)
	writeBudgetKey(&b, "M", n.MinBudgets)
	if n.Params != nil {
		fmt.Fprintf(&b, "|P%d%v", len(n.Params), n.Params)
	}
	if n.Provenance != nil {
		fmt.Fprintf(&b, "|V%d%v", len(n.Provenance), n.Provenance)
	}
	if n.Raw != nil {
		// Raw nodes are never shared by structure, only by pointer.
		fmt.Fprintf(&b, "|R%p", n.Raw)
	}
	return b.String()
}

func writeBudgetKey(b *strings.Builder, tag string, m map[string]*int) {
	if m == nil {
		return
	}
	fmt.Fprintf(b, "|%s%d", tag, len(m))
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if v := m[k]; v == nil {
			fmt.Fprintf(b, ",%q=nil", k)
		} else {
			fmt.Fprintf(b, ",%q=%d", k, *v)
		}
	}
}

func decodeCoreTIDAG(data []byte) (types.CoreTypeInfo, error) {
	var blob coreTIBlob
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&blob); err != nil {
		return nil, err
	}
	if blob.Format != coreTIDAGFormat {
		return nil, fmt.Errorf("coretypeinfo: unsupported format %d", blob.Format)
	}
	if len(blob.IDs) != len(blob.Roots) {
		return nil, fmt.Errorf("coretypeinfo: %d ids but %d roots", len(blob.IDs), len(blob.Roots))
	}
	d := coreTIDecoder{built: make([]types.Type, len(blob.Nodes))}
	for i := range blob.Nodes {
		t, err := d.build(&blob.Nodes[i], int32(i+1))
		if err != nil {
			return nil, fmt.Errorf("coretypeinfo node %d: %w", i, err)
		}
		d.built[i] = t
	}
	info := make(types.CoreTypeInfo, len(blob.IDs))
	for i, id := range blob.IDs {
		t, err := d.get(blob.Roots[i], int32(len(blob.Nodes)+1))
		if err != nil {
			return nil, fmt.Errorf("coretypeinfo root %d: %w", id, err)
		}
		info[id] = t
	}
	return info, nil
}

type coreTIDecoder struct {
	built []types.Type
}

// get resolves a child ref, which must point below the node being built (self).
func (d *coreTIDecoder) get(ref, self int32) (types.Type, error) {
	if ref == 0 {
		return nil, nil
	}
	if ref < 0 || ref >= self || int(ref) > len(d.built) {
		return nil, fmt.Errorf("ref %d out of range (node %d)", ref, self)
	}
	return d.built[ref-1], nil
}

func (d *coreTIDecoder) list(refs []int32, self int32) ([]types.Type, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	out := make([]types.Type, len(refs))
	for i, r := range refs {
		t, err := d.get(r, self)
		if err != nil {
			return nil, err
		}
		out[i] = t
	}
	return out, nil
}

func (d *coreTIDecoder) fields(n *coreTINode, self int32) (map[string]types.Type, error) {
	keys, refs := n.Keys, n.Kids
	if len(keys) != len(refs) {
		return nil, fmt.Errorf("%d keys but %d children", len(keys), len(refs))
	}
	if len(keys) == 0 {
		if n.Flags&ctiFlagEmptyMap != 0 {
			return map[string]types.Type{}, nil
		}
		return nil, nil
	}
	out := make(map[string]types.Type, len(keys))
	for i, k := range keys {
		t, err := d.get(refs[i], self)
		if err != nil {
			return nil, err
		}
		out[k] = t
	}
	return out, nil
}

func (d *coreTIDecoder) row(ref, self int32) (*types.Row, error) {
	t, err := d.get(ref, self)
	if err != nil || t == nil {
		return nil, err
	}
	r, ok := t.(*types.Row)
	if !ok {
		return nil, fmt.Errorf("ref %d is %T, want *types.Row", ref, t)
	}
	return r, nil
}

func (d *coreTIDecoder) build(n *coreTINode, self int32) (types.Type, error) {
	var err error
	switch n.Tag {
	case ctiTagCon:
		return &types.TCon{Name: n.Name}, nil
	case ctiTagVar:
		return &types.TVar{Name: n.Name}, nil
	case ctiTagVar2:
		return &types.TVar2{Name: n.Name, Kind: n.Kind}, nil
	case ctiTagRowVar:
		return &types.RowVar{Name: n.Name, Kind: n.Kind}, nil
	case ctiTagList:
		t := &types.TList{}
		t.Element, err = d.get(n.A, self)
		return t, err
	case ctiTagArray:
		t := &types.TArray{}
		t.Element, err = d.get(n.A, self)
		return t, err
	case ctiTagMap:
		t := &types.TMap{}
		if t.Key, err = d.get(n.A, self); err != nil {
			return nil, err
		}
		t.Value, err = d.get(n.B, self)
		return t, err
	case ctiTagTuple:
		t := &types.TTuple{}
		t.Elements, err = d.list(n.Kids, self)
		return t, err
	case ctiTagRecord:
		t := &types.TRecord{TypeName: n.Name}
		if t.Fields, err = d.fields(n, self); err != nil {
			return nil, err
		}
		t.Row, err = d.get(n.A, self)
		return t, err
	case ctiTagRecordOpen:
		t := &types.TRecordOpen{}
		if t.Fields, err = d.fields(n, self); err != nil {
			return nil, err
		}
		t.Row, err = d.get(n.A, self)
		return t, err
	case ctiTagRecord2:
		t := &types.TRecord2{}
		t.Row, err = d.row(n.A, self)
		return t, err
	case ctiTagApp:
		t := &types.TApp{}
		if t.Constructor, err = d.get(n.A, self); err != nil {
			return nil, err
		}
		t.Args, err = d.list(n.Kids, self)
		return t, err
	case ctiTagFunc2:
		t := &types.TFunc2{
			ImplicitEffectContract: n.Flags&ctiFlagImplicitEffect != 0,
			ConcreteEffectContract: n.Flags&ctiFlagConcreteEffect != 0,
		}
		if t.Params, err = d.list(n.Kids, self); err != nil {
			return nil, err
		}
		if t.Return, err = d.get(n.A, self); err != nil {
			return nil, err
		}
		t.EffectRow, err = d.row(n.B, self)
		return t, err
	case ctiTagLabelled:
		label, err := types.UnmarshalLabel(n.Label)
		if err != nil {
			return nil, fmt.Errorf("TLabelled: decoding label: %w", err)
		}
		t := &types.TLabelled{L: label}
		t.Inner, err = d.get(n.A, self)
		return t, err
	case ctiTagRow:
		t := &types.Row{Kind: n.Kind, Budgets: n.Budgets, MinBudgets: n.MinBudgets, Params: n.Params, Provenance: n.Provenance}
		if t.Labels, err = d.fields(n, self); err != nil {
			return nil, err
		}
		tail, err := d.get(n.A, self)
		if err != nil || tail == nil {
			return t, err
		}
		rv, ok := tail.(*types.RowVar)
		if !ok {
			return nil, fmt.Errorf("row tail is %T, want *types.RowVar", tail)
		}
		t.Tail = rv
		return t, nil
	case ctiTagRaw:
		if n.Raw == nil {
			return nil, fmt.Errorf("raw node without a value")
		}
		return n.Raw, nil
	default:
		return nil, fmt.Errorf("unknown node tag %d", n.Tag)
	}
}
