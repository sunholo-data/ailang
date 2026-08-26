package auth

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// cookieJar is a realistic Playwright storage-state blob: it carries a session
// cookie that can impersonate the account.
const cookieJar = `{"cookies":[{"name":"sid","value":"s%3AJ7qk-SUPER-SECRET-SESSION","domain":"crm.example.com"}],"origins":[{"origin":"https://crm.example.com","localStorage":[{"name":"access_token","value":"eyJhbGciOiJIUzI1NiJ9.LEAKME"}]}]}`

const hostedContextID = "ctx_01HZX9QK4M8N2P7R3T5V6W8Y0Z"

// leakMarkers are the substrings that must never survive any presentation of
// SensitiveProfileMaterial.
var leakMarkers = []string{
	"SUPER-SECRET-SESSION",
	"LEAKME",
	"access_token",
	hostedContextID,
	"ctx_01HZX",
}

func assertNoLeak(t *testing.T, label, rendered string) {
	t.Helper()
	for _, marker := range leakMarkers {
		if strings.Contains(rendered, marker) {
			t.Fatalf("%s leaked %q: %s", label, marker, rendered)
		}
	}
	if !strings.Contains(rendered, Redacted) {
		t.Fatalf("%s did not announce redaction: %s", label, rendered)
	}
}

// TestStorageStateMaterialRedactsEveryPresentation covers the six presentations
// named in the M1 acceptance criteria.
func TestStorageStateMaterialRedactsEveryPresentation(t *testing.T) {
	material := NewStorageStateMaterial([]byte(cookieJar))

	assertNoLeak(t, "String()", material.String())
	assertNoLeak(t, "Error()", material.Error())
	assertNoLeak(t, "%v", fmt.Sprintf("%v", material))
	assertNoLeak(t, "%+v", fmt.Sprintf("%+v", material))
	assertNoLeak(t, "%#v", fmt.Sprintf("%#v", material))

	encoded, err := json.Marshal(material)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	assertNoLeak(t, "MarshalJSON", string(encoded))
}

func TestProviderContextMaterialRedactsEveryPresentation(t *testing.T) {
	material := NewProviderContextMaterial(hostedContextID)

	assertNoLeak(t, "String()", material.String())
	assertNoLeak(t, "Error()", material.Error())
	assertNoLeak(t, "%v", fmt.Sprintf("%v", material))
	assertNoLeak(t, "%+v", fmt.Sprintf("%+v", material))
	assertNoLeak(t, "%#v", fmt.Sprintf("%#v", material))

	encoded, err := json.Marshal(material)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	assertNoLeak(t, "MarshalJSON", string(encoded))
}

// TestMaterialRedactsInsideContainingStruct is the case a value-only test
// misses: fmt walks into a struct and prints each field.
func TestMaterialRedactsInsideContainingStruct(t *testing.T) {
	type carrier struct {
		Alias    string
		Material SensitiveProfileMaterial
		Nested   *SensitiveProfileMaterial
	}
	pointerMaterial := NewProviderContextMaterial(hostedContextID)
	value := carrier{
		Alias:    "crm-readonly-eu",
		Material: NewStorageStateMaterial([]byte(cookieJar)),
		Nested:   &pointerMaterial,
	}

	for _, verb := range []string{"%v", "%+v", "%#v"} {
		rendered := fmt.Sprintf(verb, value)
		for _, marker := range leakMarkers {
			if strings.Contains(rendered, marker) {
				t.Fatalf("carrier %s leaked %q: %s", verb, marker, rendered)
			}
		}
		if !strings.Contains(rendered, "crm-readonly-eu") {
			t.Fatalf("carrier %s dropped safe identity: %s", verb, rendered)
		}
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal carrier: %v", err)
	}
	for _, marker := range leakMarkers {
		if strings.Contains(string(encoded), marker) {
			t.Fatalf("carrier JSON leaked %q: %s", marker, encoded)
		}
	}
}

// TestMaterializeIsTheOnlyExtractionPath proves the bytes are still reachable
// deliberately — redaction that also destroys the value would be useless.
func TestMaterializeIsTheOnlyExtractionPath(t *testing.T) {
	material := NewStorageStateMaterial([]byte(cookieJar))

	kind, state, contextID := material.Materialize()
	if kind != MaterialStorageState {
		t.Fatalf("kind = %q, want %q", kind, MaterialStorageState)
	}
	if string(state) != cookieJar {
		t.Fatalf("Materialize did not round-trip the storage state")
	}
	if contextID != "" {
		t.Fatalf("storage-state material carried a context ID: %q", contextID)
	}

	hosted := NewProviderContextMaterial(hostedContextID)
	kind, state, contextID = hosted.Materialize()
	if kind != MaterialProviderContext {
		t.Fatalf("kind = %q, want %q", kind, MaterialProviderContext)
	}
	if len(state) != 0 {
		t.Fatalf("hosted material carried storage-state bytes")
	}
	if contextID != hostedContextID {
		t.Fatalf("Materialize did not round-trip the context ID")
	}
}

// TestMaterializeReturnsACopy stops a caller from mutating registry-held bytes.
func TestMaterializeReturnsACopy(t *testing.T) {
	original := []byte(cookieJar)
	material := NewStorageStateMaterial(original)

	original[0] = 'X'
	_, first, _ := material.Materialize()
	if string(first) != cookieJar {
		t.Fatalf("material aliased the caller's slice")
	}

	first[0] = 'Y'
	_, second, _ := material.Materialize()
	if string(second) != cookieJar {
		t.Fatalf("Materialize aliased its own backing array")
	}
}

func TestEmptyMaterial(t *testing.T) {
	var zero SensitiveProfileMaterial
	if !zero.Empty() {
		t.Fatalf("zero material reported non-empty")
	}
	if got := zero.String(); !strings.Contains(got, Redacted) {
		t.Fatalf("zero material String() = %q, want a redaction marker", got)
	}
	if !NewStorageStateMaterial(nil).Empty() {
		t.Fatalf("nil storage state reported non-empty")
	}
	if NewProviderContextMaterial(hostedContextID).Empty() {
		t.Fatalf("populated hosted material reported empty")
	}
}
