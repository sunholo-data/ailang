package core

import "encoding/gob"

// M-INCREMENTAL-TYPECHECK: Register all CoreExpr and CorePattern implementations
// for gob encoding, enabling disk-caching of compiled core.Program.

func init() {
	// CoreExpr implementations
	gob.Register(&Var{})
	gob.Register(&VarGlobal{})
	gob.Register(&Lit{})
	gob.Register(&Lambda{})
	gob.Register(&Lam{})
	gob.Register(&Let{})
	gob.Register(&LetRec{})
	gob.Register(&App{})
	gob.Register(&If{})
	gob.Register(&Match{})
	gob.Register(&BinOp{})
	gob.Register(&UnOp{})
	gob.Register(&Record{})
	gob.Register(&RecordAccess{})
	gob.Register(&RecordUpdate{})
	gob.Register(&List{})
	gob.Register(&Array{})
	gob.Register(&Tuple{})
	gob.Register(&Intrinsic{})
	gob.Register(&DictAbs{})
	gob.Register(&DictApp{})
	gob.Register(&DictRef{})
	gob.Register(&Forall{})

	// CorePattern implementations
	gob.Register(&VarPattern{})
	gob.Register(&LitPattern{})
	gob.Register(&ConstructorPattern{})
	gob.Register(&ListPattern{})
	gob.Register(&RecordPattern{})
	gob.Register(&WildcardPattern{})
	gob.Register(&TuplePattern{})

	// Lit.Value / LitPattern.Value concrete types
	gob.Register(int(0))
	gob.Register(int64(0))
	gob.Register(float64(0))
	gob.Register("")
	gob.Register(true)
	gob.Register([]byte{})
}
