package pipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/types"
)

func TestCacheIFCRoundTrip(t *testing.T) {
	i := iface.NewIface("test/lib")
	i.AddExport("f", &types.Scheme{Type: &types.TFunc2{Params: []types.Type{types.TString}, Return: types.TString}}, true)
	s := i.Exports["f"].IFC
	s.Params = []types.IFCParam{{Applied: types.EncodeIFCLabel(types.LabelJoin(types.LabelConst("email"), types.LabelVar("A"))), Name: "input", Label: types.EncodeIFCLabel(types.LabelConst("email")), Forbidden: []string{"email", "secret"}}}
	s.Return = types.EncodeIFCLabel(types.LabelConst("clean"))
	s.HasReturnLabel = true
	s.Intrinsic = types.EncodeIFCLabel(types.LabelJoin(types.LabelConst("secret"), types.LabelVar("L")))
	s.Declassify = []string{"email"}
	raw, err := marshalIfaceFull(i)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := unmarshalIfaceFull(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded.Exports["f"].IFC, s) {
		t.Fatalf("full cache lost IFC: %+v", decoded.Exports["f"].IFC)
	}
}

// A valid hash authorizes bytes, not their semantic schema. Replace the payload
// AND update its stamp so rejection must come from IFC validation.
func rewriteIFCArtifact(t *testing.T, dir string, mutate func(map[string]any)) {
	t.Helper()
	path := filepath.Join(dir, artifactIfaceName)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	mutate(payload)
	raw, err = json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteArtifactTest(t, path, raw)
	stampPath := filepath.Join(dir, artifactStampName)
	stamp := readArtifactStamp(t, stampPath)
	sum := sha256.Sum256(raw)
	stamp.SHA256[artifactIfaceName] = hex.EncodeToString(sum[:])
	raw, err = json.Marshal(stamp)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteArtifactTest(t, stampPath, raw)
}

func TestCacheIFCRejectsAuthenticatedBadSchema(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"v1", func(p map[string]any) { p["schema"] = "ailang.iface/v1" }},
		{"missing", func(p map[string]any) { delete(p["exports"].(map[string]any)["f"].(map[string]any), "ifc") }},
		{"malformed authority", func(p map[string]any) {
			p["exports"].(map[string]any)["f"].(map[string]any)["ifc"].(map[string]any)["declassify"] = true
		}},
		{"missing forbidden", func(p map[string]any) {
			s := p["exports"].(map[string]any)["f"].(map[string]any)["ifc"].(map[string]any)
			delete(s["params"].([]any)[0].(map[string]any), "forbidden")
		}},
		{"missing return flag", func(p map[string]any) {
			delete(p["exports"].(map[string]any)["f"].(map[string]any)["ifc"].(map[string]any), "has_return_label")
		}},
		{"missing exports", func(p map[string]any) { delete(p, "exports") }},
		{"wrong param count", func(p map[string]any) {
			p["exports"].(map[string]any)["f"].(map[string]any)["ifc"].(map[string]any)["params"] = []any{}
		}},
		{"null export", func(p map[string]any) { p["exports"].(map[string]any)["f"] = nil }},
		{"duplicate labels", func(p map[string]any) {
			p["exports"].(map[string]any)["f"].(map[string]any)["ifc"].(map[string]any)["intrinsic"].(map[string]any)["constants"] = []string{"email", "email"}
		}},
		{"null label", func(p map[string]any) {
			p["exports"].(map[string]any)["f"].(map[string]any)["ifc"].(map[string]any)["intrinsic"] = nil
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cs := newArtifactTestStore(t)
			cm := testCachedModule("ifc/lib", 42)
			cm.Iface = iface.NewIface("ifc/lib")
			cm.Iface.AddExport("f", &types.Scheme{Type: &types.TFunc2{Params: []types.Type{types.TString}, Return: types.TString}}, true)
			mustStoreArtifacts(t, cs, "ifc/lib", "key", cm)
			rewriteIFCArtifact(t, cs.moduleArtifactDir("ifc/lib"), tc.mutate)
			if got, err := cs.LoadArtifacts("ifc/lib", "key"); err == nil || got != nil {
				t.Fatalf("bad IFC served: %v %v", got, err)
			}
		})
	}
}

func TestCacheIFCValidatesInjectedDecoder(t *testing.T) {
	cs := newArtifactTestStore(t)
	cm := testCachedModule("ifc/lib", 42)
	cm.Iface.Schema = iface.SchemaVersion
	mustStoreArtifacts(t, cs, "ifc/lib", "key", cm)
	cs.artifactCodec.decodeIface = func([]byte) (*iface.Iface, error) {
		bad := iface.NewIface("ifc/lib")
		bad.AddExport("f", &types.Scheme{Type: types.TString}, true)
		bad.Exports["f"].IFC = nil
		return bad, nil
	}
	if got, err := cs.LoadArtifacts("ifc/lib", "key"); err == nil || got != nil {
		t.Fatalf("custom decoder bypassed validation: %v %v", got, err)
	}
}

func TestCacheIFCPreviousVersionKeyMisses(t *testing.T) {
	const source = "module lib\nexport func f() -> int { 42 }\n"
	old := sha256.New()
	fmt.Fprintln(old, "ailang-cache:v6:test-compiler")
	fmt.Fprintf(old, "source:%x\n", sha256.Sum256([]byte(source)))
	oldKey := hex.EncodeToString(old.Sum(nil))
	current := ModuleCacheKey("test-compiler", source, nil)
	cs := newArtifactTestStore(t)
	cs.Store("lib", &CacheEntry{CacheKey: oldKey})
	if _, hit := cs.Lookup("lib", oldKey); !hit {
		t.Fatal("invalid old-key fixture")
	}
	if _, hit := cs.Lookup("lib", current); hit {
		t.Fatal("pre-summary v6 key served under v7")
	}
}
