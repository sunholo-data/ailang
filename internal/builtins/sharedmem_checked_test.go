package builtins

import (
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

type failedSharedCache struct{ effects.SharedCache }

func (failedSharedCache) GetChecked(string) ([]byte, bool, error) {
	return nil, false, errors.New("storage unavailable")
}
func (failedSharedCache) PutChecked(string, []byte) error { return errors.New("storage unavailable") }
func (failedSharedCache) DeleteChecked(string) error      { return errors.New("storage unavailable") }
func (failedSharedCache) CASChecked(string, []byte, []byte) (bool, error) {
	return false, errors.New("storage unavailable")
}
func (failedSharedCache) KeysChecked() ([]string, error) {
	return nil, errors.New("storage unavailable")
}

func TestSharedMemCheckedStorageErrors(t *testing.T) {
	ctx := &effects.EffContext{SharedMem: effects.NewSharedMemContext(failedSharedCache{effects.NewInMemorySharedCache()})}
	key := &eval.StringValue{Value: "key"}
	value := &eval.BytesValue{Value: []byte("value")}
	for _, tc := range []struct {
		name string
		call func(*effects.EffContext, []eval.Value) (eval.Value, error)
		args []eval.Value
	}{
		{"get", sharedMemGetImpl, []eval.Value{key}},
		{"put", sharedMemPutImpl, []eval.Value{key, value}},
		{"delete", sharedMemDeleteImpl, []eval.Value{key}},
		{"cas", sharedMemCASImpl, []eval.Value{key, &eval.TaggedValue{CtorName: "None"}, value}},
		{"keys", sharedMemKeysImpl, []eval.Value{&eval.UnitValue{}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.call(ctx, tc.args)
			if err == nil || !strings.Contains(err.Error(), "storage unavailable") || result != nil {
				t.Fatalf("result=%v err=%v", result, err)
			}
		})
	}
}
