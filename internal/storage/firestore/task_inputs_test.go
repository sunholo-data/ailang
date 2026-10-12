package firestore

import (
	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/messaging"
	"reflect"
	"testing"
)

func TestInboxTaskInputsMapRoundTrip(t *testing.T) {
	m := &messaging.InboxMessage{ID: "inputs", Inputs: []messaging.TaskInput{{Repo: "a/b", Ref: "main", Path: "photo.png", Dest: "images/"}, {Repo: "a/b", Ref: "v1", Path: "notes"}}}
	got, err := mapToInbox(inboxToMap(m))
	if err != nil || !reflect.DeepEqual(m.Inputs, got.Inputs) {
		t.Fatalf("round trip: %#v %v", got, err)
	}
	legacy, err := mapToInbox(map[string]any{"id": "old"})
	if err != nil || len(legacy.Inputs) != 0 {
		t.Fatalf("legacy: %v %v", legacy, err)
	}
	if _, err := mapToInbox(map[string]any{"inputs": "invalid"}); err == nil {
		t.Fatal("malformed inputs silently dropped")
	}
	// Also decode Firestore's native array/map shape from external writers.
	native := map[string]any{"inputs": []any{map[string]any{"repo": "a/b", "ref": "main", "path": "photo.png"}}}
	if got, err := mapToInbox(native); err != nil || len(got.Inputs) != 1 {
		t.Fatalf("native: %v %v", got, err)
	}
}

func TestTaskInputsTaskMapRoundTrip(t *testing.T) {
	task := &coordinator.TaskRecord{ID: "inputs", Inputs: []coordinator.TaskInput{{Repo: "a/b", Ref: "main", Path: "file"}}}
	got, err := mapToTask(taskToMap(task))
	if err != nil || !reflect.DeepEqual(task.Inputs, got.Inputs) {
		t.Fatalf("task: %v %v", got, err)
	}
	old, err := mapToTask(map[string]any{"id": "old"})
	if err != nil || len(old.Inputs) != 0 {
		t.Fatalf("legacy: %v %v", old, err)
	}
	if _, err := mapToTask(map[string]any{"inputs": "broken"}); err == nil {
		t.Fatal("malformed task inputs dropped")
	}
}
