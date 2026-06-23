package main

import "testing"

// TestCompactInterface: the dense typed-interface renderer (AILANG-native agent-context view).
func TestCompactInterface(t *testing.T) {
	j := `{"module":"docparse/x","types":[{"name":"Block","ctors":["A","B"]},{"name":"Rec","ctors":[]}],"funcs":[{"name":"f","type":"(int)->int!{IO}"},{"name":"g","type":"(string)->[string]!{FS}"}]}`
	out, err := compactInterface([]byte(j))
	if err != nil {
		t.Fatalf("compactInterface: %v", err)
	}
	want := "module docparse/x\ntype Block = A | B\ntype Rec\nf : (int)->int!{IO}\ng : (string)->[string]!{FS}\n"
	if out != want {
		t.Errorf("compact mismatch:\n got: %q\nwant: %q", out, want)
	}
}
