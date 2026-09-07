package dispatch

import (
	"strings"
	"testing"
)

func validRequest(t *testing.T) Request {
	t.Helper()
	return Request{Version: 1, MissionID: "test", WorkItemID: "item", StageID: "judge", AttemptID: "attempt-1", Role: "evaluator", Workspace: t.TempDir(), InputRevision: "source-sha", Instructions: "Review the artifact", Models: []string{"judge"}, AuthorModels: []string{"author"}, TimeoutSeconds: 30, MaxTokens: 1000, MaxCostUSD: 0.10}
}

func TestRequestValidationAndDigest(t *testing.T) {
	r := validRequest(t)
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	firstDigest := r.Digest()
	if firstDigest != r.Digest() {
		t.Fatal("unstable digest")
	}
	changed := r
	changed.Instructions += " differently"
	if changed.Digest() == r.Digest() {
		t.Fatal("instructions omitted from digest")
	}
	changed = r
	changed.InputRevision = "new-sha"
	if changed.Digest() == r.Digest() {
		t.Fatal("revision omitted from digest")
	}
	for _, mutate := range []func(*Request){
		func(r *Request) { r.Version = 2 }, func(r *Request) { r.Role = "judge" },
		func(r *Request) { r.Instructions = " " }, func(r *Request) { r.Workspace = "relative" },
		func(r *Request) { r.TimeoutSeconds = 0 }, func(r *Request) { r.TimeoutSeconds = 1801 },
		func(r *Request) { r.MaxTokens = 0 }, func(r *Request) { r.MaxCostUSD = 0 },
		func(r *Request) { r.AuthorModels = nil }, func(r *Request) { r.Models = nil },
		func(r *Request) { r.Models = []string{"judge", "judge"} }, func(r *Request) { r.AttemptID = "../escape" },
	} {
		bad := r
		mutate(&bad)
		if err := bad.Validate(); err == nil {
			t.Errorf("accepted invalid request: %+v", bad)
		}
	}
}

func TestDecodeRequestStrict(t *testing.T) {
	for _, body := range []string{`{"version":1,"typo":true}`, `{} {}`, `null`} {
		if _, err := DecodeRequest(strings.NewReader(body)); err == nil {
			t.Errorf("accepted %s", body)
		}
	}
}
