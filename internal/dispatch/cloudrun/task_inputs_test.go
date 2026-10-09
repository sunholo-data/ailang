package cloudrun

import (
	"context"
	"reflect"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/messaging"
)

func TestTaskInputsDispatchEnv(t *testing.T) {
	mock := &mockJobRunner{}
	d := newDispatcherWithClient(mock, "project", "region", "ailang")
	inputs := []coordinator.TaskInput{{Repo: "a/b", Ref: "main", Path: "poster", Dest: "images/"}}
	if err := d.Dispatch(context.Background(), coordinator.DispatchParams{TaskID: "task", Inputs: inputs}); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, env := range mock.lastReq.Overrides.ContainerOverrides[0].Env {
		if env.Name == config.EnvTaskInputs {
			found = true
			got, err := messaging.DecodeTaskInputs([]byte(env.GetValue()))
			if err != nil || !reflect.DeepEqual(got, inputs) {
				t.Fatalf("env: %v %v", got, err)
			}
		}
	}
	if !found {
		t.Fatal("inputs env missing")
	}
	if err := d.Dispatch(context.Background(), coordinator.DispatchParams{TaskID: "task"}); err != nil {
		t.Fatal(err)
	}
	for _, env := range mock.lastReq.Overrides.ContainerOverrides[0].Env {
		if env.Name == config.EnvTaskInputs {
			t.Fatal("zero-input dispatch changed")
		}
	}
}
