package coordinator

import (
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/pubsub"
)

func TestTaskInputsDaemonPollingBothCreationSites(t *testing.T) {
	for _, cloud := range []bool{false, true} {
		t.Run(map[bool]string{false: "local transport", true: "cloud transport"}[cloud], func(t *testing.T) {
			d := newTestDaemonWithMsgStore(t)
			d.ctx = context.Background()
			d.analyzer = NewTaskAnalyzer(0.8)
			d.agentRegistry = NewAgentRegistry()
			if err := d.agentRegistry.Register(&AgentConfig{ID: "site", Inbox: "site", Workspace: "org/workspace", InputsAllow: []string{"org/data"}}); err != nil {
				t.Fatal(err)
			}
			store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "tasks.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			d.taskStore = store
			inputs := []TaskInput{{Repo: "org/data", Ref: "main", Path: "poster.png"}, {Repo: "org/data", Ref: "v1", Path: "notes"}}
			msg := &messaging.InboxMessage{ID: "msg-12345678", ToInbox: "site", FromAgent: "sender", MessageType: "request", Title: "poster", Payload: "build a landing page", Inputs: inputs}
			if err := d.msgStore.InsertInboxMessage(msg); err != nil {
				t.Fatal(err)
			}
			if cloud {
				d.cloudInboxAdapter = NewPubSubInboxAdapter(nil, "", "site", d.msgStore, d.logger)
				data, _ := json.Marshal(pubsub.MessageNotification{MessageID: msg.ID})
				if err := d.cloudInboxAdapter.HandleNotification(data, map[string]string{"inbox": "site"}); err != nil {
					t.Fatal(err)
				}
			} else {
				d.inboxAdapters = map[string]*InboxMessageAdapter{"site": NewInboxMessageAdapter(d.msgStore, "site")}
			}
			if err := d.pollAndProcessTasks(); err != nil {
				t.Fatal(err)
			}
			tasks, err := store.ListTasks(d.ctx, &TaskFilter{})
			if err != nil || len(tasks) != 1 {
				t.Fatalf("tasks: %v %v", tasks, err)
			}
			if !reflect.DeepEqual(tasks[0].Inputs, inputs) {
				t.Fatalf("polling dropped input order/fields: %+v", tasks[0])
			}
			// Permanently fail a repo outside the grant and bank its thread reason.
			tasks[0].Inputs[0].Repo = "private/denied"
			if err := store.UpdateTask(d.ctx, tasks[0]); err != nil {
				t.Fatal(err)
			}
			if err := d.dispatchTasksCloud(); err != nil {
				t.Fatal(err)
			}
			got, err := store.GetTask(d.ctx, tasks[0].ID)
			if err != nil || got.Status != TaskStatusFailed {
				t.Fatalf("denied task: %v %v", got, err)
			}
			messages, err := d.msgStore.GetMessagesFromSeq(got.ThreadID, 0, 10)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, m := range messages {
				if strings.Contains(m.Content, "private/denied") && strings.Contains(m.Content, "site") {
					found = true
				}
			}
			if !found {
				t.Fatal("permanent denial reason was not posted to task thread")
			}
		})
	}
}
