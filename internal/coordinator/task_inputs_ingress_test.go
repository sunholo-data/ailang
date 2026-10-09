package coordinator

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
)

func TestTaskInputsHTTPAndPolling(t *testing.T) {
	d := newTestDaemonWithMsgStore(t)
	inputs := []TaskInput{{Repo: "a/b", Ref: "main", Path: "poster.png", Dest: "images/"}}
	req := map[string]any{"inbox": "site", "from": "sender", "title": "poster", "content": "publish", "inputs": inputs}
	raw, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	d.handlePostMessage(w, httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader(raw)))
	if w.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", w.Code, w.Body)
	}
	msgs, err := NewInboxMessageAdapter(d.msgStore, "site").ListUnread()
	if err != nil || len(msgs) != 1 || !reflect.DeepEqual(msgs[0].Inputs, inputs) {
		t.Fatalf("adapter: %v %v", msgs, err)
	}
	task := NewMessageWatcher(NewMockMessageStore(), 0).messageToTask(msgs[0])
	if !reflect.DeepEqual(task.Inputs, inputs) {
		t.Fatal("watcher dropped inputs")
	}
	req["inputs"] = []TaskInput{{Repo: "a/*", Ref: "main"}}
	raw, _ = json.Marshal(req)
	w = httptest.NewRecorder()
	d.handlePostMessage(w, httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader(raw)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid inputs status %d: %s", w.Code, w.Body)
	}
	list, err := d.msgStore.ListInboxMessages(messaging.InboxListOptions{})
	if err != nil || len(list) != 1 {
		t.Fatalf("invalid message stored: %v %v", list, err)
	}
}
