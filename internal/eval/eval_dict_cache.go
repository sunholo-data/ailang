package eval

import "github.com/sunholo-data/ailang/internal/types"

// dictRefCache holds the class dictionaries evalDictRef has built, keyed by
// (class, type).
//
// Before it existed, every overloaded operator evaluated through a dictionary
// (x + y, a == b, i < n on a DictApp) rebuilt the whole dictionary: one
// fmt.Sprintf key and one registry lookup per method (6-10 of them), a fresh
// map and fresh BuiltinFunction wrappers, on every call. Measured 2026-11-03
// on a Haiku cp_60164 solution (eval_results/cp_climb/step2_*): evalDictRef was
// 47% of evaluator CPU, MakeDictionaryKey alone 33%.
//
// The dictionary is an immutable RecordValue, and evaluation never writes a
// record's Fields in place, so one value can be shared by every call site. The
// cache is valid for one registry at one generation: SetDictionaryRegistry
// replaces it, and any Register (derived instances can land after evaluation
// starts, see the M-DX19 notes in buildDictRef) bumps the generation, which
// drops every entry. Not safe for concurrent use; an evaluator runs one
// goroutine at a time (stack hops block), and Fork starts with an empty cache.
type dictRefCache struct {
	registry   *types.DictionaryRegistry
	generation uint64
	entries    map[dictRefKey]*RecordValue
}

type dictRefKey struct{ className, typeName string }

func (c *dictRefCache) get(reg *types.DictionaryRegistry, className, typeName string) (*RecordValue, bool) {
	if c.entries == nil || c.registry != reg || c.generation != reg.Generation() {
		return nil, false
	}
	d, ok := c.entries[dictRefKey{className, typeName}]
	return d, ok
}

func (c *dictRefCache) put(reg *types.DictionaryRegistry, className, typeName string, d *RecordValue) {
	if c.entries == nil || c.registry != reg || c.generation != reg.Generation() {
		c.registry, c.generation = reg, reg.Generation()
		c.entries = make(map[dictRefKey]*RecordValue)
	}
	c.entries[dictRefKey{className, typeName}] = d
}
