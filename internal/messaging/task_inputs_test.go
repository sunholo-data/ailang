package messaging

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func taskInputFixture() []TaskInput {
	return []TaskInput{{Repo: "sunholo-data/daneel-memory", Ref: "incoming/DNL-abc123", Path: "poster.png", Dest: "images/", SHA256: strings.Repeat("a", 64)}, {Repo: "sunholo-data/daneel-memory", Ref: "v1.0", Path: "notes"}}
}

func TestTaskInputsValidation(t *testing.T) {
	good := taskInputFixture()
	if err := ValidateTaskInputs(good); err != nil {
		t.Fatal(err)
	}
	if err := ValidateTaskInputs(nil); err != nil {
		t.Fatal(err)
	}
	bad := []TaskInput{
		{Repo: "https://github.com/a/b", Ref: "main"}, {Repo: "a/*", Ref: "main"}, {Repo: "a/..", Ref: "main"},
		{Repo: "a/b", Ref: "-main"}, {Repo: "a/b", Ref: "main..next"}, {Repo: "a/b", Ref: strings.Repeat("a", 40)},
		{Repo: "a/b", Ref: "refs/heads/a.lock"}, {Repo: "a/b", Ref: "main", Path: "../secret"},
		{Repo: "a/b", Ref: "main", Path: "a/../secret"}, {Repo: "a/b", Ref: "main", Dest: "/tmp"},
		{Repo: "a/b", Ref: "main", Dest: "C:\\tmp"}, {Repo: "a/b", Ref: "main", SHA256: strings.Repeat("a", 64)},
		{Repo: "a/b", Ref: "main", Path: "file", SHA256: "garbage"}, {Repo: "a/b", Ref: ""},
	}
	for _, input := range bad {
		t.Run(input.Repo+input.Ref+input.Path+input.Dest+input.SHA256, func(t *testing.T) {
			if err := ValidateTaskInputs([]TaskInput{input}); err == nil {
				t.Fatalf("accepted %#v", input)
			}
		})
	}
	if err := ValidateTaskInputs(make([]TaskInput, 17)); err == nil {
		t.Fatal("accepted >16 inputs")
	}
	for _, raw := range []string{`{}`, `[null]`, `[{"repo":"a/b","ref":"main","extra":true}]`, `[] []`, `[`} {
		if _, err := DecodeTaskInputs([]byte(raw)); err == nil {
			t.Errorf("accepted %q", raw)
		}
	}
	for _, raw := range []string{"", "null", "[]"} {
		if _, err := DecodeTaskInputs([]byte(raw)); err != nil {
			t.Fatal(err)
		}
	}
	b, _ := json.Marshal(good)
	decoded, err := DecodeTaskInputs(b)
	if err != nil || !reflect.DeepEqual(good, decoded) {
		t.Fatalf("round trip %v %v", decoded, err)
	}
}

func TestInboxTaskInputsPersistence(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "messages.db")
	s, err := OpenStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	msg := &InboxMessage{ID: "inputs", FromAgent: "sender", ToInbox: "site", Title: "poster", Inputs: taskInputFixture()}
	if err := s.InsertInboxMessage(msg); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = OpenStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.GetInboxMessage(msg.ID)
	if err != nil || !reflect.DeepEqual(msg.Inputs, got.Inputs) {
		t.Fatalf("get: %#v %v", got, err)
	}
	list, err := s.ListInboxMessages(InboxListOptions{})
	if err != nil || len(list) != 1 || !reflect.DeepEqual(msg.Inputs, list[0].Inputs) {
		t.Fatalf("list: %#v %v", list, err)
	}
	if _, err := s.db.Exec(`UPDATE inbox_messages SET inputs='broken' WHERE id=?`, msg.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetInboxMessage(msg.ID); err == nil {
		t.Fatal("get dropped malformed inputs")
	}
	if _, err := s.ListInboxMessages(InboxListOptions{}); err == nil {
		t.Fatal("list dropped malformed inputs")
	}
}

func TestTaskInputsMigrationPreservesMessages(t *testing.T) {
	db, path := oldSchemaDB(t)
	if err := migrateV180ToV190(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO inbox_messages(id,message_id,from_agent,to_inbox,title,created_at) VALUES('legacy','legacy','sender','site','brief','2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	s, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.GetInboxMessage("legacy")
	if err != nil || got == nil || len(got.Inputs) != 0 {
		t.Fatalf("legacy: %v %v", got, err)
	}
	var version string
	if err := s.db.QueryRow(`SELECT version FROM schema_version WHERE version='1.10.0'`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if err := MigrateDB(s.db); err != nil {
		t.Fatal(err)
	}
}
