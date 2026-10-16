package builtins

import (
	"errors"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

var indexStorageFailure = errors.New("index storage failed")

// Embed the two distinct contracts so tests can independently override checked
// operations while leaving legacy behavior available.
type failingCheckedIndex struct {
	effects.SharedIndex
	effects.CheckedSharedIndex
}

func (f failingCheckedIndex) UpsertChecked(string, string, int64, int64, int64) error {
	return indexStorageFailure
}
func (f failingCheckedIndex) DeleteChecked(string, string) error { return indexStorageFailure }
func (f failingCheckedIndex) UpsertWithEmbeddingChecked(string, string, int64, []float64, int64, int64) error {
	return indexStorageFailure
}
func (f failingCheckedIndex) EntryCountChecked(string) (int, error) { return 0, indexStorageFailure }
func (f failingCheckedIndex) NamespacesChecked() ([]string, error)  { return nil, indexStorageFailure }
func (f failingCheckedIndex) FindSimilarSimHashChecked(string, int64, int, int, effects.DeterminismMode) ([]effects.SearchResult, error) {
	return nil, indexStorageFailure
}
func (f failingCheckedIndex) FindSimilarByEmbeddingChecked(string, []float64, int, int, effects.DeterminismMode) ([]effects.SearchResult, error) {
	return nil, indexStorageFailure
}

func TestSharedIndexStorageErrorsAtEffectBoundary(t *testing.T) {
	ns := &eval.StringValue{Value: "n"}
	key := &eval.StringValue{Value: "key"}
	n := &eval.IntValue{Value: 1}
	emb := &eval.ListValue{Elements: []eval.Value{&eval.FloatValue{Value: 1}}}
	strict := &eval.BoolValue{Value: true}
	tests := []struct {
		name string
		call func(*effects.EffContext, []eval.Value) (eval.Value, error)
		args []eval.Value
	}{
		{"upsert", sharedIndexUpsertImpl, []eval.Value{ns, key, n, n, n}},
		{"delete", sharedIndexDeleteImpl, []eval.Value{ns, key}},
		{"embedding write", sharedIndexUpsertEmbImpl, []eval.Value{ns, key, n, emb, n, n}},
		{"count", sharedIndexEntryCountImpl, []eval.Value{ns}},
		{"namespaces", sharedIndexNamespacesImpl, []eval.Value{&eval.UnitValue{}}},
		{"simhash search", sharedIndexFindSimHashImpl, []eval.Value{ns, n, n, n, strict}},
		{"embedding search", sharedIndexFindByEmbeddingImpl, []eval.Value{ns, emb, n, n, strict}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := createSharedIndexContext()
			ctx.SharedIndex.Index = failingCheckedIndex{SharedIndex: effects.NewInMemorySharedIndex()}
			value, err := tt.call(ctx, tt.args)
			if !errors.Is(err, indexStorageFailure) || value != nil {
				t.Fatalf("got %v, %v", value, err)
			}
			if ctx.SharedIndex.UpsertCount != 0 || ctx.SharedIndex.DeleteCount != 0 || ctx.SharedIndex.SearchCount != 0 {
				t.Fatal("failed operation counted as success")
			}
		})
	}
}
