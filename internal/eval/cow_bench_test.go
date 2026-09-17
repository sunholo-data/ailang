package eval

import (
	"fmt"
	"testing"
)

// Copy-on-write cost, as allocation per op (perf-sweep). MapValue.Insert
// copies the whole map each time (M-V1-MEMORY-FOOTPRINT F17); a persistent
// map would flatten it. Measured as B/op
// because the copies are garbage — peak RSS of the same loop swung +34% with
// no code change (2026-09-17), which is GC timing, not the runtime.

func BenchmarkMapInsert5K(b *testing.B) {
	keys := make([]Value, 5000)
	for i := range keys {
		keys[i] = &StringValue{Value: fmt.Sprintf("k%d", i)}
	}
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		m := &MapValue{Entries: map[string]*MapEntry{}}
		for i, k := range keys {
			var err error
			if m, err = m.Insert(k, &IntValue{Value: i}); err != nil {
				b.Fatal(err)
			}
		}
	}
}
