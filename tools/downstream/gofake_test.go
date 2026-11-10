package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if os.Getenv(goFakeEnv) == "1" {
		os.Exit(goFakeMain())
	}
	os.Exit(m.Run())
}

// goFakeEnv switches the test binary into goFakeMain: the test-binary re-exec
// pattern gives the timeout tests a stand-in compiler that is a real Go binary.
// A Go binary ignores SIGALRM unless it registers for it (the runtime marks it
// _SigNotify), which is exactly how the real `ailang` defeated the old
// `perl -e 'alarm …; exec …'` timeout; a bash fake dies on SIGALRM and hid that.
const goFakeEnv = "DOWNSTREAM_TEST_GO_FAKE"

// goFakeMain implements enough of the `check --json` contract for the canary
// and for hangs: "not an int" fails, HANG sleeps, HANG_CHILD first starts a
// grandchild that holds stdout open and records its pid in the arm's state dir.
func goFakeMain() int {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "--version" {
		fmt.Println("go-fake v0.58.1")
		return 0
	}
	if len(args) == 0 || args[0] != "check" {
		fmt.Fprintln(os.Stderr, "usage")
		return 64
	}
	file := args[len(args)-1]
	if len(args) > 1 && args[1] == "--package" {
		fmt.Println(`{"file":".","passed":true,"errors":[]}`)
		return 0
	}
	data, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	content := string(data)
	switch {
	case strings.Contains(content, "not an int"):
		fmt.Printf(`{"file":%q,"passed":false,"errors":[{"code":"ERROR","message":"type error in m (decl 0): cannot unify string vs int"}]}`+"\n", file)
		return 1
	case strings.Contains(content, "HANG_CHILD"):
		child := exec.Command("sleep", "300")
		child.Stdout, child.Stderr = os.Stdout, os.Stderr
		if err := child.Start(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		_ = os.WriteFile(filepath.Join(os.Getenv("AILANG_STATE_DIR"), "grandchild.pid"), []byte(fmt.Sprint(child.Process.Pid)), 0o644)
		time.Sleep(30 * time.Second)
	case strings.Contains(content, "HANG"):
		time.Sleep(30 * time.Second)
	}
	fmt.Printf(`{"file":%q,"passed":true,"errors":[]}`+"\n", file)
	return 0
}
