package mapval_test

import (
	"encoding/json"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/mapval"
)

// Every numeric representation a map can hold must read the same. The
// float64-only and int64-only readers this package replaced each read the
// other's representation as 0.
func TestNumericReadersAcceptEveryDecoderRepresentation(t *testing.T) {
	reps := map[string]interface{}{
		"int":     int(42),
		"int32":   int32(42),
		"int64":   int64(42),
		"float32": float32(42),
		"float64": float64(42),
		"number":  json.Number("42"),
	}
	for name := range reps {
		if got := mapval.Int(reps, name); got != 42 {
			t.Errorf("Int(%s) = %d, want 42", name, got)
		}
		if got := mapval.Int64(reps, name); got != 42 {
			t.Errorf("Int64(%s) = %d, want 42", name, got)
		}
		if got := mapval.Float(reps, name); got != 42 {
			t.Errorf("Float(%s) = %f, want 42", name, got)
		}
	}

	// A real Firestore int64 that does not fit float64 exactly survives Int64.
	big := map[string]interface{}{"fp": int64(-6615550055289275125)}
	if got := mapval.Int64(big, "fp"); got != -6615550055289275125 {
		t.Errorf("Int64 lost precision on a Firestore int64: %d", got)
	}
	// encoding/json output with UseNumber: a fractional number still reads.
	frac := map[string]interface{}{"n": json.Number("3.9")}
	if got := mapval.Int(frac, "n"); got != 3 {
		t.Errorf("Int(json.Number 3.9) = %d, want 3", got)
	}
	if got := mapval.Float(frac, "n"); got != 3.9 {
		t.Errorf("Float(json.Number 3.9) = %f", got)
	}
}

func TestZeroValuesForMissingNilAndWrongKind(t *testing.T) {
	m := map[string]interface{}{"nil": nil, "str": "x", "num": 1.5, "b": true}
	for _, key := range []string{"missing", "nil", "str"} {
		if mapval.Int(m, key) != 0 || mapval.Int64(m, key) != 0 || mapval.Float(m, key) != 0 {
			t.Errorf("numeric reader of %q must be 0", key)
		}
		if mapval.Bool(m, key) {
			t.Errorf("Bool(%q) must be false", key)
		}
		if mapval.Strings(m, key) != nil {
			t.Errorf("Strings(%q) must be nil", key)
		}
	}
	if mapval.String(m, "num") != "" || mapval.String(m, "missing") != "" {
		t.Error("String of a non-string must be empty")
	}
	if mapval.String(m, "str") != "x" || !mapval.Bool(m, "b") {
		t.Error("typed reads failed")
	}
	if mapval.String(nil, "k") != "" || mapval.Int(nil, "k") != 0 {
		t.Error("a nil map reads as zero values")
	}
}

func TestStringsAcceptsBothArrayShapes(t *testing.T) {
	m := map[string]interface{}{
		"typed":   []string{"a", "b"},
		"decoded": []interface{}{"a", 1, "b", nil},
	}
	want := []string{"a", "b"}
	if got := mapval.Strings(m, "typed"); !reflect.DeepEqual(got, want) {
		t.Errorf("typed: %v", got)
	}
	if got := mapval.Strings(m, "decoded"); !reflect.DeepEqual(got, want) {
		t.Errorf("decoded (non-strings dropped): %v", got)
	}
}

// mapval is a LEAF (stdlib only): the Firestore store, the server bridge
// and cmd all read decoded maps through it.
func TestMapvalIsALeaf(t *testing.T) {
	const module = "github.com/sunholo-data/ailang/"
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}
	sawControl := false
	for _, d := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if d == "encoding/json" {
			sawControl = true
		}
		first, _, _ := strings.Cut(d, "/")
		if strings.Contains(first, ".") && d != module+"internal/mapval" {
			t.Errorf("mapval must be stdlib-only but depends on %s", d)
		}
	}
	if !sawControl {
		t.Fatal("instrument check failed: encoding/json absent from deps")
	}
}
