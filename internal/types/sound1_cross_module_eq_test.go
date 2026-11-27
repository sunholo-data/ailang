package types

import (
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// TST-RC1-01
func TestSound1_CrossModuleRecordAliasDoesNotHijackEq(t *testing.T) {
	env := LoadBuiltinInstances()
	// std/regex exports `type Regex = Regex(string) deriving (Eq)`
	if err := env.AddDerivedEqForADT("std/regex", "Regex"); err != nil {
		t.Fatal(err)
	}
	// s1/a exports `type Pt = { x: int, y: int } deriving (Eq)`
	if err := env.AddDerivedEqForADT("s1/a", "Pt"); err != nil {
		t.Fatal(err)
	}

	tc := NewCoreTypeCheckerWithInstances(env)
	strToBool := &TFunc2{Params: []Type{TString}, Return: TBool, EffectRow: EmptyEffectRow()}
	userRegexRec := &TRecord{
		Fields: map[string]Type{
			"pattern": TString,
			"matcher": strToBool,
		},
		TypeName:   "Regex",
		TypeModule: "s1/a",
	}
	ptRec := &TRecord{
		Fields: map[string]Type{
			"x": TInt,
			"y": TInt,
		},
		TypeName:   "Pt",
		TypeModule: "s1/a",
	}
	tc.RegisterTypeAlias(NominalKey("s1/a", "Regex"), userRegexRec)
	tc.RegisterTypeAlias(NominalKey("s1/a", "Pt"), ptRec)

	// Build `\r1 r2. r1 == r2` annotated with closed interface record types `s1/a.Regex`
	lamID := uint64(10)
	eqID := uint64(11)
	r1ID := uint64(12)
	r2ID := uint64(13)
	lam := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: lamID},
		Params:   []string{"r1", "r2"},
		Body: &core.Intrinsic{
			CoreNode: core.CoreNode{NodeID: eqID},
			Op:       core.OpEq,
			Args: []core.CoreExpr{
				&core.Var{CoreNode: core.CoreNode{NodeID: r1ID}, Name: "r1"},
				&core.Var{CoreNode: core.CoreNode{NodeID: r2ID}, Name: "r2"},
			},
		},
	}
	tc.SetParamTypeAnnotations(map[uint64][]Type{
		lamID: {userRegexRec, userRegexRec},
	})

	_, _, err := tc.CheckCoreExpr(lam, NewTypeEnv())
	if err == nil {
		t.Fatalf("expected type error comparing cross-module s1/a.Regex without deriving (Eq), but CheckCoreExpr succeeded")
	}
	if !strings.Contains(err.Error(), "No instance for Eq") {
		t.Fatalf("expected 'No instance for Eq' error, got: %v", err)
	}

	// Valid same-module derived record `s1/a.Pt` MUST still resolve Eq
	s1PtCon := &TCon{Name: "Pt", Module: "s1/a"}
	ptLamID := uint64(20)
	ptLam := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: ptLamID},
		Params:   []string{"p1", "p2"},
		Body: &core.Intrinsic{
			CoreNode: core.CoreNode{NodeID: 21},
			Op:       core.OpEq,
			Args: []core.CoreExpr{
				&core.Var{CoreNode: core.CoreNode{NodeID: 22}, Name: "p1"},
				&core.Var{CoreNode: core.CoreNode{NodeID: 23}, Name: "p2"},
			},
		},
	}
	tc2 := NewCoreTypeCheckerWithInstances(env)
	tc2.RegisterTypeAlias(NominalKey("s1/a", "Pt"), ptRec)
	tc2.SetParamTypeAnnotations(map[uint64][]Type{
		ptLamID: {s1PtCon, s1PtCon},
	})
	if _, _, err := tc2.CheckCoreExpr(ptLam, NewTypeEnv()); err != nil {
		t.Fatalf("expected s1/a.Pt deriving (Eq) to type-check, got error: %v", err)
	}

	// Ensure std/regex.Regex ADT itself still resolves Eq
	if _, err := env.Lookup("Eq", &TCon{Name: "Regex", Module: "std/regex"}); err != nil {
		t.Fatalf("std/regex.Regex must keep its Eq instance: %v", err)
	}
	var missing *MissingInstanceError
	if _, err := env.Lookup("Eq", &TCon{Name: "Regex", Module: "s1/a"}); !errors.As(err, &missing) {
		t.Fatalf("s1/a.Regex TCon must not have Eq: got %v", err)
	}
}

// TST-RC1-01
func TestRecordFieldMismatchError_OpenPatternSecondaryHint(t *testing.T) {
	// Contravariant mismatch when a closed record pattern `{a, b}` meets a wider
	// caller record `{a, b, c}`: expected has `{a, b, c}`, actual has `{a, b}`,
	// so `c` is reported in missingFields.
	expected := map[string]Type{"a": TInt, "b": TInt, "c": TInt}
	actual := map[string]Type{"a": TInt, "b": TInt}

	msg := recordFieldMismatchError(expected, actual).Error()
	if !strings.Contains(msg, "missing required field(s): c") {
		t.Errorf("expected missing required field c in message, got:\n%s", msg)
	}
	if !strings.Contains(msg, "{a, b, ...}") {
		t.Errorf("expected open-record pattern hint '{a, b, ...}' in message, got:\n%s", msg)
	}
	if strings.Contains(msg, "open record syntax") {
		t.Errorf("must preserve TestRecordFieldMismatchError_MissingFieldHint invariant (no 'open record syntax' phrasing), got:\n%s", msg)
	}
}
