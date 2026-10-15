package vm

import (
	"fmt"
	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/eval"
)

func mapToEval(m *bytecode.MapObj, display bool) (eval.Value, error) {
	entries := make(map[string]*eval.MapEntry, len(m.Entries))
	for _, k := range m.SortedKeys() {
		e := m.Entries[k]
		key, err := bytecodeToEval(e.Key, display)
		if err != nil {
			return nil, err
		}
		val, err := bytecodeToEval(e.Value, display)
		if err != nil {
			return nil, fmt.Errorf("map[%s]: %w", k, err)
		}
		entries[k] = &eval.MapEntry{Key: key, Value: val}
	}
	return &eval.MapValue{Entries: entries}, nil
}

func mapFromEval(m *eval.MapValue) (bytecode.Value, error) {
	entries := make(map[string]bytecode.MapEntry, len(m.Entries))
	for k, e := range m.Entries {
		key, err := EvalToBytecode(e.Key)
		if err != nil {
			return bytecode.Value{}, err
		}
		val, err := EvalToBytecode(e.Value)
		if err != nil {
			return bytecode.Value{}, fmt.Errorf("map[%s]: %w", k, err)
		}
		entries[k] = bytecode.MapEntry{Key: key, Value: val}
	}
	return bytecode.NewMap(entries), nil
}
