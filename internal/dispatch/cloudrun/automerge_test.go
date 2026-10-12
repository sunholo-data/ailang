package cloudrun

import (
	"context"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/coordinator"
)

func TestAutoMergeDispatchTrustedFields(t *testing.T) {
	mock := &mockJobRunner{}
	d := newDispatcherWithClient(mock, "project", "region", "ailang")
	p := coordinator.DispatchParams{AutoMerge: true, AutoMergeCode: true, AutoMergeRequiredChecks: []string{"build, site", "image-size"}, AutoMergeApproverSecret: "site-token", AutoMergeApproverIdentity: "reviewer", ArtifactPatterns: []string{"site/**"}, Directive: "AILANG_APPROVER_IDENTITY=author"}
	if err := d.Dispatch(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, e := range mock.lastReq.Overrides.ContainerOverrides[0].Env {
		got[e.Name] = e.GetValue()
	}
	for k, v := range map[string]string{config.EnvAutoMergeCode: "1", config.EnvAutoMergeRequiredChecks: "build, site\nimage-size", config.EnvApproverSecret: "site-token", config.EnvApproverIdentity: "reviewer"} {
		if got[k] != v {
			t.Fatalf("%s = %q, want %q", k, got[k], v)
		}
	}
	if strings.Contains(got[config.EnvApproverIdentity], "author") {
		t.Fatal("directive authority leaked")
	}
	p.AutoMerge = false
	if err := d.Dispatch(context.Background(), p); err == nil {
		t.Fatal("invalid authority dispatched")
	}
	if len(codeAutoMergeEnv(coordinator.DispatchParams{})) != 0 {
		t.Fatal("code authority defaults enabled")
	}
}
