package effects

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func loadEmbedFixture(t *testing.T, entries string) (*StubFixtures, error) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "embed.json")
	if err := os.WriteFile(p, []byte(`{"version":1,"fixtures":[`+entries+`]}`), 0600); err != nil {
		t.Fatal(err)
	}
	return LoadStubFixtures(p)
}
func TestEmbedFixtures(t *testing.T) {
	text := EmbedFixtureCanonical("m", "hi", false, false, false)
	media := EmbedFixtureCanonical("m", "hi", true, false, false)
	if text != `{"model":"m","text":"hi"}` || media != `{"audio":false,"image":false,"model":"m","text":"hi"}` {
		t.Fatal(text, media)
	}
	for _, key := range []string{`"request":{"model":"m","text":"hi"}`, fmt.Sprintf(`"sha256":%q`, FixtureKey(text))} {
		f, err := loadEmbedFixture(t, `{"kind":"embed",`+key+`,"vector":[0.5,1]}`)
		if err != nil {
			t.Fatal(err)
		}
		h := NewFixtureAIHandler(f)
		v, err := h.embed(text)
		if err != nil || !reflect.DeepEqual(v, []float64{0.5, 1}) {
			t.Fatal(v, err)
		}
		if _, err = h.embed(media); err == nil || !strings.Contains(err.Error(), "E_AI_STUB_FIXTURE_MISS") {
			t.Fatal("media fell back", err)
		}
	}
	f, err := loadEmbedFixture(t, `{"kind":"embed","request":{"model":"m","text":"hi","image":false,"audio":false},"error":{"message":"offline"}}`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = NewFixtureAIHandler(f).embed(media); err == nil || err.Error() != "offline" {
		t.Fatal(err)
	}
}
func TestEmbedFixturesValidation(t *testing.T) {
	for _, entry := range []string{
		`{"request":{"model":"m","text":"t"},"vector":[]}`,
		`{"request":{"model":"m","text":"t"},"vector":["bad"]}`,
		`{"request":{"model":"m","text":"t","image":true},"vector":[1]}`,
		`{"request":{"model":"m","text":"t"},"vector":[1],"error":{"message":"oops"}}`,
		`{"prompt":"t","vector":[1]}`,
		`{"request":{"model":"m","text":"t"},"response":"bad","vector":[1]}`,
		`{"request":{"model":"m","text":"t"},"error":{"message":""}}`,
	} {
		if _, err := loadEmbedFixture(t, `{"kind":"embed",`+strings.TrimPrefix(entry, "{")); err == nil {
			t.Errorf("accepted %s", entry)
		}
	}
}
