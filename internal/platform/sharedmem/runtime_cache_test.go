package sharedmem

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCheckedSharedMemCASConcurrentConnections(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.db")
	a, err := NewSQLiteSharedCache(path)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := NewSQLiteSharedCache(path)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	for _, old := range [][]byte{nil, []byte("initial")} {
		if old != nil {
			if err := a.PutChecked("key", old); err != nil {
				t.Fatal(err)
			}
		}
		var successes atomic.Int32
		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				c := a
				if i%2 == 1 {
					c = b
				}
				ok, err := c.CASChecked("key", old, []byte("updated"))
				if err != nil {
					t.Error(err)
				}
				if ok {
					successes.Add(1)
				}
			}(i)
		}
		wg.Wait()
		if successes.Load() != 1 {
			t.Fatalf("successful swaps = %d", successes.Load())
		}
	}
}

func TestCheckedSharedMemFailureAndEmptyBytes(t *testing.T) {
	c, err := NewSQLiteSharedCache(filepath.Join(t.TempDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.PutChecked("empty", nil); err != nil {
		t.Fatal(err)
	}
	v, ok, err := c.GetChecked("empty")
	if err != nil || !ok || len(v) != 0 {
		t.Fatalf("get empty = %v %v %v", v, ok, err)
	}
	ok, err = c.CASChecked("empty", []byte{}, nil)
	if err != nil || !ok {
		t.Fatalf("CAS empty = %v %v", ok, err)
	}
	c.Close()
	if _, _, err := c.GetChecked("x"); err == nil {
		t.Error("closed get succeeded")
	}
	if err := c.PutChecked("x", nil); err == nil {
		t.Error("closed put succeeded")
	}
	if _, err := c.CASChecked("x", nil, nil); err == nil {
		t.Error("closed CAS succeeded")
	}
	if err := c.DeleteChecked("x"); err == nil {
		t.Error("closed delete succeeded")
	}
	if _, err := c.KeysChecked(); err == nil {
		t.Error("closed keys succeeded")
	}
}

// TestCheckedSharedMemCASProcess is also the child entry point. Children wait
// until every process has opened the database before competing for one swap.
func TestCheckedSharedMemCASProcess(t *testing.T) {
	if path := os.Getenv("AILANG_TEST_CAS_DB"); path != "" {
		c, err := NewSQLiteSharedCache(path)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		if err := os.WriteFile(os.Getenv("AILANG_TEST_CAS_READY"), nil, 0600); err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(20 * time.Second)
		for {
			if _, err := os.Stat(os.Getenv("AILANG_TEST_CAS_GATE")); err == nil {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("gate timed out")
			}
			time.Sleep(5 * time.Millisecond)
		}
		var old []byte
		if os.Getenv("AILANG_TEST_CAS_EXISTING") == "1" {
			old = []byte("initial")
		}
		success, err := c.CASChecked("key", old, []byte("updated"))
		if err != nil {
			t.Fatal(err)
		}
		if success {
			fmt.Println("CAS_WINNER")
		}
		return
	}
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprint(existing), func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "cache.db")
			gate := filepath.Join(dir, "gate")
			c, err := NewSQLiteSharedCache(path)
			if err != nil {
				t.Fatal(err)
			}
			if existing {
				if err := c.PutChecked("key", []byte("initial")); err != nil {
					t.Fatal(err)
				}
			}
			c.Close()
			const count = 4
			commands := make([]*exec.Cmd, count)
			outputs := make([]bytes.Buffer, count)
			for i := range commands {
				cmd := exec.Command(os.Args[0], "-test.run=^TestCheckedSharedMemCASProcess$")
				cmd.Env = append(os.Environ(), "AILANG_TEST_CAS_DB="+path, "AILANG_TEST_CAS_GATE="+gate, "AILANG_TEST_CAS_READY="+filepath.Join(dir, fmt.Sprintf("ready%d", i)))
				if existing {
					cmd.Env = append(cmd.Env, "AILANG_TEST_CAS_EXISTING=1")
				}
				cmd.Stdout = &outputs[i]
				cmd.Stderr = &outputs[i]
				if err := cmd.Start(); err != nil {
					t.Fatal(err)
				}
				commands[i] = cmd
				t.Cleanup(func() {
					if cmd.Process != nil {
						_ = cmd.Process.Kill()
					}
				})
			}
			deadline := time.Now().Add(20 * time.Second)
			for {
				ready := 0
				for i := range commands {
					if _, err := os.Stat(filepath.Join(dir, fmt.Sprintf("ready%d", i))); err == nil {
						ready++
					}
				}
				if ready == count {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("children failed to open databases")
				}
				time.Sleep(5 * time.Millisecond)
			}
			if err := os.WriteFile(gate, nil, 0600); err != nil {
				t.Fatal(err)
			}
			winners := 0
			for i, cmd := range commands {
				if err := cmd.Wait(); err != nil {
					t.Fatalf("child: %v: %s", err, outputs[i].String())
				}
				winners += strings.Count(outputs[i].String(), "CAS_WINNER")
			}
			if winners != 1 {
				t.Fatalf("successful process swaps = %d", winners)
			}
		})
	}
}
