package types

import (
	"testing"
)

func TestExtractTVarName(t *testing.T) {
	tests := []struct {
		name     string
		input    Type
		wantName string
		wantOk   bool
	}{
		{
			name:     "TVar returns name",
			input:    &TVar{Name: "alpha"},
			wantName: "alpha",
			wantOk:   true,
		},
		{
			name:     "TVar2 returns name",
			input:    &TVar2{Name: "beta", Kind: KStar{}},
			wantName: "beta",
			wantOk:   true,
		},
		{
			name:     "TCon returns false",
			input:    &TCon{Name: "Int"},
			wantName: "",
			wantOk:   false,
		},
		{
			name:     "TFunc2 returns false",
			input:    &TFunc2{Params: []Type{&TCon{Name: "Int"}}, Return: &TCon{Name: "Int"}},
			wantName: "",
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotOk := ExtractTVarName(tt.input)
			if gotName != tt.wantName {
				t.Errorf("ExtractTVarName() gotName = %v, want %v", gotName, tt.wantName)
			}
			if gotOk != tt.wantOk {
				t.Errorf("ExtractTVarName() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func TestIsTVar(t *testing.T) {
	tests := []struct {
		name  string
		input Type
		want  bool
	}{
		{
			name:  "TVar is type variable",
			input: &TVar{Name: "alpha"},
			want:  true,
		},
		{
			name:  "TVar2 is type variable",
			input: &TVar2{Name: "beta", Kind: KStar{}},
			want:  true,
		},
		{
			name:  "TCon is not type variable",
			input: &TCon{Name: "Int"},
			want:  false,
		},
		{
			name:  "TFunc2 is not type variable",
			input: &TFunc2{Params: []Type{&TCon{Name: "Int"}}, Return: &TCon{Name: "Int"}},
			want:  false,
		},
		{
			name:  "TRecord is not type variable",
			input: &TRecord{Fields: map[string]Type{}},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTVar(tt.input); got != tt.want {
				t.Errorf("IsTVar() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractTVarKind(t *testing.T) {
	kstar := KStar{}
	tests := []struct {
		name     string
		input    Type
		wantKind Kind
		wantOk   bool
	}{
		{
			name:     "TVar2 returns Kind",
			input:    &TVar2{Name: "alpha", Kind: kstar},
			wantKind: kstar,
			wantOk:   true,
		},
		{
			name:     "TVar has no Kind",
			input:    &TVar{Name: "alpha"},
			wantKind: nil,
			wantOk:   false,
		},
		{
			name:     "TCon has no Kind",
			input:    &TCon{Name: "Int"},
			wantKind: nil,
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKind, gotOk := ExtractTVarKind(tt.input)
			if gotOk != tt.wantOk {
				t.Errorf("ExtractTVarKind() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
			if gotOk && gotKind != tt.wantKind {
				t.Errorf("ExtractTVarKind() gotKind = %v, want %v", gotKind, tt.wantKind)
			}
		})
	}
}

// TestAsList tests the AsList helper for DX-17: TList/TApp unification
func TestAsList(t *testing.T) {
	T := NewBuilder()

	tests := []struct {
		name     string
		input    Type
		wantElem Type
		wantOk   bool
	}{
		{
			name:     "TList returns element type",
			input:    &TList{Element: &TCon{Name: "int"}},
			wantElem: &TCon{Name: "int"},
			wantOk:   true,
		},
		{
			name:     "TApp(list, int) returns element type",
			input:    T.List(T.Int()),
			wantElem: &TCon{Name: "int"},
			wantOk:   true,
		},
		{
			name:     "TApp(List, int) uppercase returns false (case sensitive)",
			input:    T.App("List", T.Int()),
			wantElem: nil,
			wantOk:   false,
		},
		{
			name:     "TCon(string) returns false",
			input:    T.String(),
			wantElem: nil,
			wantOk:   false,
		},
		{
			name:     "TApp(Option, int) returns false",
			input:    T.App("Option", T.Int()),
			wantElem: nil,
			wantOk:   false,
		},
		{
			name:     "TVar2 returns false",
			input:    &TVar2{Name: "a", Kind: KStar{}},
			wantElem: nil,
			wantOk:   false,
		},
		{
			name:     "TList with polymorphic element",
			input:    &TList{Element: &TVar2{Name: "a", Kind: KStar{}}},
			wantElem: &TVar2{Name: "a", Kind: KStar{}},
			wantOk:   true,
		},
		{
			name:     "TApp(list, TVar2) with polymorphic element",
			input:    T.List(&TVar2{Name: "a", Kind: KStar{}}),
			wantElem: &TVar2{Name: "a", Kind: KStar{}},
			wantOk:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotElem, gotOk := AsList(tt.input)
			if gotOk != tt.wantOk {
				t.Errorf("AsList() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
			// Use string comparison for type equality in tests
			if gotOk && gotElem.String() != tt.wantElem.String() {
				t.Errorf("AsList() gotElem = %v, want %v", gotElem, tt.wantElem)
			}
		})
	}
}
