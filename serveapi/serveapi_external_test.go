package serveapi

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/apiserver"
)

type fixtureHost struct{}

func (fixtureHost) ResolveSession(context.Context, *http.Request) (Session, error) {
	return "session", nil
}
func (fixtureHost) Tools(context.Context, Session) ([]ToolDescriptor, error) { return nil, nil }
func (fixtureHost) Invoke(context.Context, Session, Invocation) (InvocationResult, error) {
	return InvocationResult{}, nil
}

func validConfig() Config {
	host := fixtureHost{}
	return Config{
		Resolver: host,
		Tools:    host,
		Invoker:  host,
		Agent:    AgentInfo{Name: "fixture", Version: "1.0.0"},
	}
}

func TestNewValidationAndEffectiveDefaultDeadline(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{"nil resolver", func(c *Config) { c.Resolver = nil }},
		{"nil tools", func(c *Config) { c.Tools = nil }},
		{"nil invoker", func(c *Config) { c.Invoker = nil }},
		{"empty agent name", func(c *Config) { c.Agent.Name = " " }},
		{"empty agent version", func(c *Config) { c.Agent.Version = "" }},
		{"negative timeout", func(c *Config) { c.CallbackTimeout = -time.Second }},
		{"negative concurrency", func(c *Config) { c.MaxConcurrentCallbacks = -1 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := validConfig()
			test.mutate(&cfg)
			if _, err := New(cfg); err == nil {
				t.Fatal("New() succeeded, want validation error")
			}
		})
	}

	server, err := New(validConfig())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	deadline, err := apiserver.RunCallback(context.Background(), server.runner, func(ctx context.Context) (time.Time, error) {
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("callback context has no deadline")
		}
		return deadline, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if deadline.Before(now.Add(4*time.Second)) || deadline.After(now.Add(6*time.Second)) {
		t.Fatalf("default deadline %v outside observed window from %v", deadline, now)
	}
	value, err := apiserver.RunCallback(context.Background(), server.runner, func(context.Context) (int, error) { return 137, nil })
	if err != nil || value != 137 {
		t.Fatalf("fast callback = (%d, %v)", value, err)
	}
}

func TestExternalModuleCanImportFacadeButNotInternal(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("instrument failure: runtime.Caller failed")
	}
	root := filepath.Dir(filepath.Dir(currentFile))
	dir := t.TempDir()
	t.Logf("external fixture directory: %s", dir)
	goMod := fmt.Sprintf("module externalfixture\n\ngo 1.26.5\n\nrequire github.com/sunholo-data/ailang v0.0.0\nreplace github.com/sunholo-data/ailang => %s\n", root)
	writeFixtureFile(t, filepath.Join(dir, "go.mod"), []byte(goMod))
	sum, err := os.ReadFile(filepath.Join(root, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, filepath.Join(dir, "go.sum"), sum)
	writeFixtureFile(t, filepath.Join(dir, "main.go"), []byte(externalFixtureSource))
	writeFixtureFile(t, filepath.Join(dir, "denied.go"), []byte(deniedFixtureSource))

	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOPROXY=off")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("external facade build failed: %v\n%s", err, output)
	}
	t.Log("external facade build: rc=0")

	cmd = exec.Command("go", "build", "-tags=denied", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOPROXY=off")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("internal import unexpectedly built")
	}
	want := "use of internal package github.com/sunholo-data/ailang/internal/apiserver not allowed"
	if !strings.Contains(string(output), want) {
		t.Fatalf("denied build error did not contain %q:\n%s", want, output)
	}
	t.Logf("denied internal import: nonzero rc, matched %q", want)
}

func writeFixtureFile(t *testing.T, path string, contents []byte) {
	t.Helper()
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
}

const externalFixtureSource = `package main
import (
 "context"
 "net/http"
 "github.com/sunholo-data/ailang/serveapi"
)
type host struct{}
func (host) ResolveSession(context.Context, *http.Request) (serveapi.Session, error) { return "s", nil }
func (host) Tools(context.Context, serveapi.Session) ([]serveapi.ToolDescriptor, error) { return nil, nil }
func (host) Invoke(context.Context, serveapi.Session, serveapi.Invocation) (serveapi.InvocationResult, error) { return serveapi.InvocationResult{}, nil }
func main() {
 h := host{}
 api, err := serveapi.New(serveapi.Config{Resolver:h, Tools:h, Invoker:h, Agent:serveapi.AgentInfo{Name:"fixture", Version:"1"}})
 if err != nil { panic(err) }
 _ = api.MCPHandler(); _ = api.A2AHandler()
 api.Mount(http.NewServeMux())
}
`

const deniedFixtureSource = `//go:build denied
package main
import _ "github.com/sunholo-data/ailang/internal/apiserver"
`
