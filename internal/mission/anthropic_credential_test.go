package mission

import "testing"

func TestFreshestClaudeCredential(t *testing.T) {
	stale := []byte(`{"claudeAiOauth":{"accessToken":"old","expiresAt":1000}}`)
	fresh := []byte(`{"claudeAiOauth":{"accessToken":"new","expiresAt":2000}}`)
	blank := []byte(`{"claudeAiOauth":{"accessToken":"","expiresAt":0}}`)
	cases := []struct {
		name    string
		blobs   [][]byte
		want    string
		wantExp int64
	}{
		{"stale keychain, fresh file", [][]byte{stale, fresh}, "new", 2000},
		{"fresh keychain, stale file", [][]byte{fresh, stale}, "new", 2000},
		{"blanked keychain never wins", [][]byte{blank, stale}, "old", 1000},
		{"unreadable keychain, file only", [][]byte{nil, fresh}, "new", 2000},
		{"garbage skipped", [][]byte{[]byte("not json"), stale}, "old", 1000},
		{"legacy top-level token", [][]byte{[]byte(`{"accessToken":"legacy"}`)}, "legacy", 0},
		{"nothing usable", [][]byte{nil, blank}, "", 0},
	}
	for _, c := range cases {
		got, exp := freshestClaudeCredential(c.blobs...)
		if got != c.want || exp != c.wantExp {
			t.Errorf("%s: got (%q,%d) want (%q,%d)", c.name, got, exp, c.want, c.wantExp)
		}
	}
}
