//go:build integration
// +build integration

// Integration tests for submit_feedback routing — validates that the
// {package, auto_dispatch} matrix actually routes to the right Firestore
// inbox + carries the right Pub/Sub category attribute.
//
// Excluded from `go test ./...` by build tag. Run via:
//
//	make test-feedback-integration
//
// which sets AILANG_STORAGE=gcp + AILANG_CLOUD_PROJECT=ailang-multivac-test
// and runs `go test -tags integration -count=1 ./internal/feedback/`.
//
// Test docs are tagged with from_agent="mcp-public-test" so TestMain can
// clean them up after the run — keeps the test env tidy across runs.
package feedback

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

const (
	// testTitlePrefix marks integration-test submissions for cleanup. We use
	// a title prefix (not from_agent) because the publisher hardcodes
	// from_agent="mcp-public" and we don't want to add a runtime override
	// just for tests — title-prefix scanning is just as reliable.
	testTitlePrefix     = "integration-test:"
	testFromAgent       = "mcp-public"
	testInboxCollection = "inbox_messages"
	expectedTestProject = "ailang-multivac-test"
)

func TestMain(m *testing.M) {
	if os.Getenv("AILANG_STORAGE") != "gcp" {
		fmt.Fprintln(os.Stderr, "integration tests require AILANG_STORAGE=gcp; skipping")
		os.Exit(0)
	}
	if proj := os.Getenv("AILANG_CLOUD_PROJECT"); proj != expectedTestProject {
		fmt.Fprintf(os.Stderr, "integration tests require AILANG_CLOUD_PROJECT=%s (got %q) — refuse to run against prod\n", expectedTestProject, proj)
		os.Exit(1)
	}
	code := m.Run()
	if err := cleanupTestDocs(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "cleanup warning: %v\n", err)
	}
	os.Exit(code)
}

func mustPublisher(t *testing.T) *Publisher {
	t.Helper()
	pub, err := Get(context.Background())
	if err != nil {
		t.Fatalf("publisher init failed: %v", err)
	}
	return pub
}

func TestIntegration_DefaultRouting(t *testing.T) {
	pub := mustPublisher(t)
	res, err := pub.Submit(context.Background(), testRequest("default", "docs", "", false))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	doc := mustReadBack(t, res.TicketID)
	assertString(t, doc, "to_inbox", "public-feedback")
	assertString(t, doc, "category", "docs")
	assertString(t, doc, "from_agent", testFromAgent)
	assertString(t, doc, "message_type", "feedback")
	// Title carries the marker prefix so cleanup can find it.
	title, _ := doc["title"].(string)
	if !strings.HasPrefix(title, testTitlePrefix) {
		t.Errorf("title %q missing test marker prefix %q (would orphan in cleanup)", title, testTitlePrefix)
	}

	// Regression: the Firestore doc key MUST equal the returned ticket_id.
	// The coordinator's pubsub adapter and `ailang messages read <fb_*>`
	// both call MessageStore.GetInboxMessage(ticketID), which is a direct
	// Doc(ticketID).Get(). If the publisher leaves InboxMessage.ID empty,
	// the store auto-generates an inbox_<ts>_<rand> key and the lookup
	// fails ("inbox message not found") — exactly the bug fixed alongside
	// this test.
	mustReadBackByDocID(t, res.TicketID)
}

func TestIntegration_PackageRouting_NoDispatch(t *testing.T) {
	pub := mustPublisher(t)
	res, err := pub.Submit(context.Background(), testRequest("pkg-no-dispatch", "docs", "sunholo/auth", false))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	doc := mustReadBack(t, res.TicketID)
	assertString(t, doc, "to_inbox", "pkg:sunholo/auth")
	// auto_dispatch=false → category stored as-is. The auto: prefix lives
	// on the Pub/Sub notification attribute (separate from the doc).
	assertString(t, doc, "category", "docs")
}

func TestIntegration_PackageRouting_WithDispatch(t *testing.T) {
	pub := mustPublisher(t)
	res, err := pub.Submit(context.Background(), testRequest("pkg-dispatch", "docs", "sunholo/auth", true))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	doc := mustReadBack(t, res.TicketID)
	assertString(t, doc, "to_inbox", "pkg:sunholo/auth")
	// Firestore doc keeps the canonical category; Pub/Sub attribute carries
	// the "auto:" prefix the coordinator filters on. Doc-side observers
	// shouldn't see prefix noise.
	assertString(t, doc, "category", "docs")
}

func TestIntegration_InvalidPackage(t *testing.T) {
	pub := mustPublisher(t)
	_, err := pub.Submit(context.Background(), testRequest("bad-pkg", "docs", "noslash", false))
	if err == nil {
		t.Fatal("expected validation error for invalid package, got nil")
	}
	var fe *FieldError
	if !errors.As(err, &fe) {
		t.Fatalf("expected *FieldError, got %T: %v", err, err)
	}
	if fe.Code != "invalid_package" || fe.Field != "package" {
		t.Errorf("got FieldError{Code: %q, Field: %q}, want {Code: invalid_package, Field: package}", fe.Code, fe.Field)
	}
}

func TestIntegration_MissingBody(t *testing.T) {
	pub := mustPublisher(t)
	req := testRequest("empty-body", "docs", "", false)
	req.Body = ""
	_, err := pub.Submit(context.Background(), req)
	if err == nil {
		t.Fatal("expected validation error for empty body, got nil")
	}
	var fe *FieldError
	if !errors.As(err, &fe) || fe.Field != "body" {
		t.Fatalf("expected FieldError on field=body, got %v", err)
	}
}

func TestIntegration_ValidateOnly(t *testing.T) {
	for _, c := range []struct {
		name      string
		req       Request
		wantField string
	}{
		{"missing title", testRequestModify("ok", "docs", "", false, func(r *Request) { r.Title = "" }), "title"},
		{"oversized body", testRequestModify("ok", "docs", "", false, func(r *Request) { r.Body = strings.Repeat("x", maxBodyBytes+1) }), "body"},
		{"invalid category", testRequestModify("ok", "weird-cat", "", false, nil), "category"},
		{"invalid package format", testRequestModify("ok", "docs", "no-slash-here", false, nil), "package"},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := Validate(c.req)
			var fe *FieldError
			if !errors.As(err, &fe) {
				t.Fatalf("Validate returned %v, want *FieldError", err)
			}
			if fe.Field != c.wantField {
				t.Errorf("got Field=%q, want %q", fe.Field, c.wantField)
			}
		})
	}
}

// ─── helpers ────────────────────────────────────────────────────────────

func testRequest(label, category, pkg string, dispatch bool) Request {
	return Request{
		Title:         fmt.Sprintf("integration-test:%s:%d", label, time.Now().UnixNano()),
		Body:          "Integration test submission. Safe to delete.",
		Category:      category,
		AILangVersion: "v0.0.0-test",
		Snippet:       "",
		Contact:       "",
		Package:       pkg,
		AutoDispatch:  dispatch,
	}
}

func testRequestModify(label, category, pkg string, dispatch bool, modify func(*Request)) Request {
	r := testRequest(label, category, pkg, dispatch)
	if modify != nil {
		modify(&r)
	}
	return r
}

func mustReadBack(t *testing.T, messageID string) map[string]interface{} {
	t.Helper()
	ctx := context.Background()
	client := mustFirestoreClient(t)
	defer client.Close()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		iter := client.Collection(testInboxCollection).Where("message_id", "==", messageID).Limit(1).Documents(ctx)
		doc, err := iter.Next()
		if err == iterator.Done {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if err != nil {
			t.Fatalf("read-back query for %s: %v", messageID, err)
		}
		return doc.Data()
	}
	t.Fatalf("message %s did not appear in Firestore within 10s", messageID)
	return nil
}

// mustReadBackByDocID asserts the doc is retrievable by ticket_id as the
// Firestore doc key — the same code path the coordinator's pubsub adapter
// (GetInboxMessage) and `ailang messages read` use. Distinct from
// mustReadBack, which queries the message_id field.
func mustReadBackByDocID(t *testing.T, ticketID string) {
	t.Helper()
	ctx := context.Background()
	client := mustFirestoreClient(t)
	defer client.Close()

	doc, err := client.Collection(testInboxCollection).Doc(ticketID).Get(ctx)
	if err != nil {
		t.Fatalf("doc-key lookup for ticket %s failed: %v (this is the GetInboxMessage path the coordinator and CLI use)", ticketID, err)
	}
	// Sanity: the doc we found by key should also carry the same message_id field.
	if got, _ := doc.Data()["message_id"].(string); got != ticketID {
		t.Errorf("doc[%s].message_id = %q, want %q (doc key and message_id field must agree)", ticketID, got, ticketID)
	}
}

func mustFirestoreClient(t *testing.T) *firestore.Client {
	t.Helper()
	client, err := firestore.NewClient(context.Background(), os.Getenv("AILANG_CLOUD_PROJECT"))
	if err != nil {
		t.Fatalf("firestore.NewClient: %v", err)
	}
	return client
}

func assertString(t *testing.T, doc map[string]interface{}, key, want string) {
	t.Helper()
	got, ok := doc[key].(string)
	if !ok {
		t.Fatalf("doc[%q] is not a string: %T (%v)", key, doc[key], doc[key])
	}
	if got != want {
		t.Errorf("doc[%q] = %q, want %q", key, got, want)
	}
}

// cleanupTestDocs deletes any inbox_messages doc whose title starts with
// the test marker prefix. Best-effort — failures logged, don't fail suite.
// Firestore doesn't support startswith, so we fetch the recent mcp-public
// submissions and filter client-side.
func cleanupTestDocs(ctx context.Context) error {
	if os.Getenv("AILANG_CLOUD_PROJECT") != expectedTestProject {
		return errors.New("refusing to clean up against non-test project")
	}
	client, err := firestore.NewClient(ctx, expectedTestProject)
	if err != nil {
		return fmt.Errorf("firestore client: %w", err)
	}
	defer client.Close()

	// Scope the scan: only mcp-public submissions in the last 1h. Avoids
	// touching real public-feedback or pkg:* messages from non-test traffic.
	cutoff := time.Now().Add(-1 * time.Hour)
	iter := client.Collection(testInboxCollection).
		Where("from_agent", "==", testFromAgent).
		Where("created_at", ">=", cutoff).
		Documents(ctx)
	deleted, scanned := 0, 0
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		scanned++
		title, _ := doc.Data()["title"].(string)
		if !strings.HasPrefix(title, testTitlePrefix) {
			continue
		}
		if _, err := doc.Ref.Delete(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "  cleanup delete %s failed: %v\n", doc.Ref.ID, err)
			continue
		}
		deleted++
	}
	fmt.Fprintf(os.Stderr, "cleanup: scanned %d recent mcp-public docs, deleted %d test-tagged\n", scanned, deleted)
	return nil
}
