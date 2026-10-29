//go:build linux

package proctree

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// Adversarial probe for audit H-6 P0.2 (M-EXECUTOR-UID-SPLIT AC2).
//
// The test binary re-executes itself twice: as a PARENT holding a sentinel
// secret in its initial environment, and, through Configure (the spawn path
// every executor uses), as a same-UID CHILD that tries to read the parent's
// /proc environ and mem. The control run skips DisableDump and must LEAK, so
// a probe that can no longer see the secret fails instead of passing.

const (
	h6RoleEnv   = "AILANG_H6_PROBE_ROLE"
	h6DumpEnv   = "AILANG_H6_PROBE_DISABLE_DUMP"
	h6SecretEnv = "AILANG_H6_PROBE_SECRET"
	h6Secret    = "h6-sentinel-7f3a"
)

func TestMain(m *testing.M) {
	switch os.Getenv(h6RoleEnv) {
	case "parent":
		os.Exit(h6Parent())
	case "child":
		os.Exit(h6Child())
	}
	os.Exit(m.Run())
}

func h6Parent() int {
	if os.Getenv(h6DumpEnv) == "1" {
		if err := DisableDump(); err != nil {
			fmt.Printf("disable-dump=error %v\n", err)
			return 2
		}
	}
	// CommandContext, as every executor does: Configure installs a Cancel.
	child := exec.CommandContext(context.Background(), os.Args[0])
	child.Env = []string{h6RoleEnv + "=child"} // the child is NOT given the secret
	Configure(child)
	out, err := child.CombinedOutput()
	os.Stdout.Write(out)
	if err != nil {
		fmt.Printf("child-error %v\n", err)
		return 3
	}
	return 0
}

func h6Child() int {
	ppid := strconv.Itoa(os.Getppid())
	env, err := os.ReadFile("/proc/" + ppid + "/environ")
	switch {
	case err != nil:
		fmt.Printf("environ=denied %v\n", err)
	case bytes.Contains(env, []byte(h6Secret)):
		fmt.Println("environ=LEAKED")
	default:
		fmt.Println("environ=readable-no-secret")
	}
	if f, err := os.Open("/proc/" + ppid + "/mem"); err != nil {
		fmt.Printf("mem=denied %v\n", err)
	} else {
		_ = f.Close()
		fmt.Println("mem=OPENED")
	}
	return 0
}

func runH6Probe(t *testing.T, disableDump bool) string {
	t.Helper()
	parent := exec.Command(os.Args[0], "-test.run=^$")
	dump := "0"
	if disableDump {
		dump = "1"
	}
	parent.Env = append(os.Environ(), h6RoleEnv+"=parent", h6DumpEnv+"="+dump, h6SecretEnv+"="+h6Secret)
	out, err := parent.CombinedOutput()
	if err != nil {
		t.Fatalf("probe parent failed: %v\n%s", err, out)
	}
	return string(out)
}

func TestDisableDumpDeniesSameUIDChild(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses dump protection (CAP_SYS_PTRACE); the executor runs non-root")
	}
	control := runH6Probe(t, false)
	if !strings.Contains(control, "environ=LEAKED") {
		t.Fatalf("control run did not leak the parent's secret, so the probe cannot see a leak:\n%s", control)
	}
	got := runH6Probe(t, true)
	if !strings.Contains(got, "environ=denied") {
		t.Errorf("with DisableDump the child still reads /proc/$PPID/environ:\n%s", got)
	}
	if !strings.Contains(got, "mem=denied") {
		t.Errorf("with DisableDump the child still opens /proc/$PPID/mem:\n%s", got)
	}
}

func TestDisableDumpReadsBackZero(t *testing.T) {
	// In a subprocess: the test binary's own dumpable flag must not change.
	out := runH6Probe(t, true)
	if strings.Contains(out, "disable-dump=error") {
		t.Fatalf("DisableDump failed: %s", out)
	}
}
