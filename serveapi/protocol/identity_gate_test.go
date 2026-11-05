package protocol

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// M-MCP-OAUTH-CLIENT-MODELS M2 (D5): the identity gate admits only a non-empty
// subject, and returns it; the bool gate's contract is unchanged and names no
// identity.
func TestIdentityGate_Outcomes(t *testing.T) {
	verify := func(_ context.Context, tok string) (*Identity, error) {
		switch tok {
		case "u1":
			return &Identity{Subject: "u1", ClientID: "https://c.example/cimd", Scopes: []string{"read"}}, nil
		case "empty":
			return &Identity{Subject: "", ClientID: "c"}, nil
		case "boom":
			return nil, errors.New("db down")
		case "panic":
			panic("verifier bug")
		}
		return nil, nil
	}
	g := NewIdentityGate(verify, metadataFor, time.Second, 4)
	cases := []struct {
		token       string
		wantAdmit   bool
		wantStatus  int
		wantSubject string
	}{
		{"", false, 401, ""},
		{"none", false, 401, ""},
		{"empty", false, 401, ""}, // an empty subject is a rejection, never an anonymous admit
		{"boom", false, 503, ""},
		{"panic", false, 503, ""},
		{"u1", true, 200, "u1"},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		id, ok := g.AdmitCallIdentity(w, gateRequest(c.token), json.RawMessage(`1`))
		status := w.Code
		if ok {
			status = 200
		}
		if ok != c.wantAdmit || status != c.wantStatus {
			t.Fatalf("token %q: admit=%v status=%d, want %v %d (%s)", c.token, ok, status, c.wantAdmit, c.wantStatus, w.Body)
		}
		switch {
		case ok && (id == nil || id.Subject != c.wantSubject || id.ClientID == "" || len(id.Scopes) != 1):
			t.Errorf("token %q: identity %+v", c.token, id)
		case !ok && id != nil:
			t.Errorf("token %q: refused but returned identity %+v", c.token, id)
		}
		if c.wantStatus == 401 && !strings.Contains(w.Header().Get("WWW-Authenticate"), "resource_metadata=") {
			t.Errorf("token %q: 401 without challenge", c.token)
		}
	}
	// AdmitCall on an identity gate is the same decision.
	if g.AdmitCall(httptest.NewRecorder(), gateRequest("empty"), nil) {
		t.Error("AdmitCall admitted an empty subject")
	}
}

// A bool gate admits exactly as before and returns no identity.
func TestBoolGate_AdmitCallIdentityIsNil(t *testing.T) {
	g := NewBearerGate(func(_ context.Context, tok string) (bool, error) { return tok == "good", nil }, metadataFor, time.Second, 4)
	id, ok := g.AdmitCallIdentity(httptest.NewRecorder(), gateRequest("good"), nil)
	if !ok || id != nil {
		t.Fatalf("bool gate: ok=%v id=%+v, want admit with nil identity", ok, id)
	}
	if _, ok := g.AdmitCallIdentity(httptest.NewRecorder(), gateRequest("bad"), nil); ok {
		t.Fatal("bool gate admitted a rejected token")
	}
}

func TestWriteRateLimited(t *testing.T) {
	w := httptest.NewRecorder()
	WriteRateLimited(w, json.RawMessage(`"c1"`), 1500*time.Millisecond)
	if w.Code != 429 || w.Header().Get("Retry-After") != "2" {
		t.Fatalf("status %d Retry-After %q", w.Code, w.Header().Get("Retry-After"))
	}
	var msg struct {
		ID     json.RawMessage `json:"id"`
		Result struct {
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &msg); err != nil || string(msg.ID) != `"c1"` || !msg.Result.IsError {
		t.Fatalf("body %s (%v)", w.Body, err)
	}
	w = httptest.NewRecorder()
	WriteRateLimited(w, nil, 0)
	if w.Code != 429 || w.Header().Get("Retry-After") != "1" || !strings.Contains(w.Body.String(), `"rate_limited"`) {
		t.Fatalf("no call id: %d %q %s", w.Code, w.Header().Get("Retry-After"), w.Body)
	}
}
