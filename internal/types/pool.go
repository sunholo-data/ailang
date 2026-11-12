package types

import "sync"

// M-PERF6: Pools for visited maps used in type traversal functions.
// These maps are allocated on every call to occurs, substitute, equals, etc.
// Pooling reduces GC pressure during type checking of large modules.

var typeBoolPool = sync.Pool{
	New: func() interface{} {
		return make(map[Type]bool, 8)
	},
}

var typeTypePool = sync.Pool{
	New: func() interface{} {
		return make(map[Type]Type, 8)
	},
}

var typePairBoolPool = sync.Pool{
	New: func() interface{} {
		return make(map[typePair]bool, 8)
	},
}

// maxPooledMapLen: a map that held more entries than this is dropped, not
// pooled. A Go map keeps its capacity, and clearing (or ranging over) it costs
// that capacity, so one walk over a record with hundreds of fields used to make
// every later small walk pay for it (#2210 review, item 6: 15% of `check` on a
// 200-field record updated 200 times).
const maxPooledMapLen = 256

// getTypeBoolMap gets a map[Type]bool from the pool.
func getTypeBoolMap() map[Type]bool {
	return typeBoolPool.Get().(map[Type]bool)
}

// putTypeBoolMap returns a map[Type]bool to the pool after clearing it.
func putTypeBoolMap(m map[Type]bool) {
	if len(m) > maxPooledMapLen {
		return
	}
	clear(m)
	typeBoolPool.Put(m)
}

// getTypeTypeMap gets a map[Type]Type from the pool.
func getTypeTypeMap() map[Type]Type {
	return typeTypePool.Get().(map[Type]Type)
}

// putTypeTypeMap returns a map[Type]Type to the pool after clearing it.
func putTypeTypeMap(m map[Type]Type) {
	if len(m) > maxPooledMapLen {
		return
	}
	clear(m)
	typeTypePool.Put(m)
}

// getTypePairBoolMap gets a map[typePair]bool from the pool.
func getTypePairBoolMap() map[typePair]bool {
	return typePairBoolPool.Get().(map[typePair]bool)
}

// putTypePairBoolMap returns a map[typePair]bool to the pool after clearing it.
func putTypePairBoolMap(m map[typePair]bool) {
	if len(m) > maxPooledMapLen {
		return
	}
	clear(m)
	typePairBoolPool.Put(m)
}
