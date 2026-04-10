package types

import "encoding/gob"

// M-PERF6B M1: Register all Type, Kind, and Row implementations for gob encoding.
// This enables gob serialization of CoreTypeInfo (map[uint64]Type), replacing
// the slower JSON serialization used in M-INCREMENTAL-TYPECHECK.

func init() {
	// Type implementations (14 types)
	gob.Register(&TCon{})
	gob.Register(&TVar{})
	gob.Register(&TVar2{})
	gob.Register(&TList{})
	gob.Register(&TArray{})
	gob.Register(&TMap{})
	gob.Register(&TTuple{})
	gob.Register(&TFunc2{})
	gob.Register(&TRecord{})
	gob.Register(&TRecordOpen{})
	gob.Register(&TRecord2{})
	gob.Register(&TApp{})
	gob.Register(&Row{})
	gob.Register(&RowVar{})

	// Kind implementations (used by TVar2, TApp, etc.)
	gob.Register(KStar{})
	gob.Register(KEffect{})
	gob.Register(KRecord{})
	gob.Register(KRow{})

	// Scheme (used in Iface exports)
	gob.Register(&Scheme{})

	// TypeClass (used in Scheme constraints)
	gob.Register(&TypeClass{})
}
