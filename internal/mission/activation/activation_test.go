package activation

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func fixture(t *testing.T) (*Manager, Request) {
	t.Helper()
	root := t.TempDir()
	return &Manager{Dir: filepath.Join(root, "records")}, Request{OperationID: "test-one", MissionID: "docs", WorkItemID: "work-one", MarkerPath: filepath.Join(root, "disabled"), BindingPath: filepath.Join(root, "binding.toml"), Binding: []byte("version = 1\nstate_db = \"" + testAbsPath("canary.db") + "\"\nworkspace_root = \"" + testAbsPath("canary") + "\"\n")}
}
func stopped(context.Context, Record) error { return nil }
func TestRestoreBaselines(t *testing.T) {
	if !HostSupported() {
		t.Skip("local mission activation requires macOS or Linux host locking (lock_other.go)")
	}
	for _, present := range []bool{false, true} {
		t.Run(map[bool]string{false: "absent", true: "present"}[present], func(t *testing.T) {
			m, q := fixture(t)
			old := []byte("version=1\nstate_db=\"" + testAbsPath("old.db") + "\"\nworkspace_root=\"" + testAbsPath("old") + "\"\n")
			if present {
				if err := os.WriteFile(q.BindingPath, old, 0640); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := m.Activate(context.Background(), q, stopped); err != nil {
				t.Fatal(err)
			}
			if _, err := m.Recover(context.Background(), q.OperationID, stopped); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(q.MarkerPath); !os.IsNotExist(err) {
				t.Fatalf("marker retained: %v", err)
			}
			got, err := os.ReadFile(q.BindingPath)
			if present {
				if err != nil || string(got) != string(old) {
					t.Fatalf("baseline not restored: %q %v", got, err)
				}
				info, _ := os.Stat(q.BindingPath)
				if info.Mode().Perm() != 0640 {
					t.Fatal(info.Mode())
				}
			} else if !os.IsNotExist(err) {
				t.Fatal("absent baseline created")
			}
			if _, err := m.Recover(context.Background(), q.OperationID, nil); err != nil {
				t.Fatal(err)
			}
			r, err := m.Inspect(q.OperationID)
			if err != nil || string(r.Binding.Installed) != string(q.Binding) || r.Phase != "restored" {
				t.Fatalf("retained evidence: %+v %v", r, err)
			}
		})
	}
}
func TestUnverifiedOrChangedFilesStayHeld(t *testing.T) {
	if !HostSupported() {
		t.Skip("local mission activation requires macOS or Linux host locking (lock_other.go)")
	}
	m, q := fixture(t)
	if _, err := m.Activate(context.Background(), q, stopped); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Recover(context.Background(), q.OperationID, func(context.Context, Record) error { return errors.New("child identity unknown") }); err == nil {
		t.Fatal("unverified cleanup")
	}
	changed := []byte("external edit")
	if err := os.WriteFile(q.BindingPath, changed, 0600); err != nil {
		t.Fatal(err)
	}
	r, err := m.Recover(context.Background(), q.OperationID, stopped)
	if err == nil || r.CleanupPending == "" {
		t.Fatal("changed file accepted")
	}
	b, _ := os.ReadFile(q.BindingPath)
	if string(b) != string(changed) {
		t.Fatal("external edit lost")
	}
	if _, err := os.Stat(q.MarkerPath); err != nil {
		t.Fatal("pause lost")
	}
	q.OperationID = "other"
	if _, err := m.Activate(context.Background(), q, stopped); err == nil {
		t.Fatal("foreign owner replaced")
	}
}
func TestForeignMarkerAndSecretsRejected(t *testing.T) {
	m, q := fixture(t)
	if err := os.WriteFile(q.MarkerPath, []byte("foreign"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Activate(context.Background(), q, stopped); err == nil {
		t.Fatal("foreign marker adopted")
	}
	if err := os.Remove(q.MarkerPath); err != nil {
		t.Fatal(err)
	}
	for _, data := range []string{string(q.Binding) + "secret=\"sensitive\"\n", string(q.Binding) + "# token=sensitive\n"} {
		if err := os.WriteFile(q.BindingPath, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := m.Activate(context.Background(), q, stopped); err == nil {
			t.Fatal("secret-bearing baseline persisted")
		}
	}
}
func TestProcessDeathRecovery(t *testing.T) {
	if !HostSupported() {
		t.Skip("local mission activation requires macOS or Linux host locking (lock_other.go)")
	}
	if os.Getenv("AILANG_ACTIVATION_CRASH") != "" {
		root := os.Getenv("AILANG_ACTIVATION_ROOT")
		m := &Manager{Dir: filepath.Join(root, "records"), checkpoint: func(point string) {
			if point == os.Getenv("AILANG_ACTIVATION_CRASH") {
				os.Exit(37)
			}
		}}
		q := Request{OperationID: "crash", MissionID: "docs", WorkItemID: "work", MarkerPath: filepath.Join(root, "disabled"), BindingPath: filepath.Join(root, "binding.toml"), Binding: []byte("version=1\nstate_db=\"" + testAbsPath("crash.db") + "\"\nworkspace_root=\"" + testAbsPath("crash") + "\"\n")}
		if os.Getenv("AILANG_ACTIVATION_RECOVER") == "yes" {
			_, _ = m.Recover(context.Background(), q.OperationID, stopped)
		} else {
			_, _ = m.Activate(context.Background(), q, stopped)
		}
		os.Exit(2)
	}
	for _, point := range []string{"record_prepared", "owner_persisted", "marker_before", "marker_after", "binding_before", "binding_after", "active", "restore_binding_before", "restore_binding_after", "restore_marker_before", "restore_marker_after", "restored", "owner_released"} {
		t.Run(point, func(t *testing.T) {
			root := t.TempDir()
			m := &Manager{Dir: filepath.Join(root, "records")}
			q := Request{OperationID: "crash", MissionID: "docs", WorkItemID: "work", MarkerPath: filepath.Join(root, "disabled"), BindingPath: filepath.Join(root, "binding.toml"), Binding: []byte("version=1\nstate_db=\"" + testAbsPath("crash.db") + "\"\nworkspace_root=\"" + testAbsPath("crash") + "\"\n")}
			env := append(os.Environ(), "AILANG_ACTIVATION_CRASH="+point, "AILANG_ACTIVATION_ROOT="+root)
			if len(point) >= 7 && point[:7] == "restore" || point == "owner_released" {
				if _, err := m.Activate(context.Background(), q, stopped); err != nil {
					t.Fatal(err)
				}
				env = append(env, "AILANG_ACTIVATION_RECOVER=yes")
			}
			cmd := exec.Command(os.Args[0], "-test.run=^TestProcessDeathRecovery$")
			cmd.Env = env
			err := cmd.Run()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 37 {
				t.Fatalf("checkpoint not reached: %v", err)
			}
			if _, err := m.Recover(context.Background(), "crash", stopped); err != nil {
				t.Fatal(err)
			}
			for _, p := range []string{q.MarkerPath, q.BindingPath} {
				if _, err := os.Stat(p); !os.IsNotExist(err) {
					t.Fatalf("crash left %s: %v", p, err)
				}
			}
		})
	}
}

func TestConcurrentHostOperations(t *testing.T) {
	m, q := fixture(t)
	entered, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	var once sync.Once
	go func() {
		_, err := m.Activate(context.Background(), q, func(context.Context, Record) error { once.Do(func() { close(entered); <-release }); return nil })
		done <- err
	}()
	<-entered
	other := q
	other.OperationID = "other"
	if _, err := m.Activate(context.Background(), other, stopped); err == nil {
		t.Error("concurrent operation entered locked transaction")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if _, err := m.Activate(context.Background(), other, stopped); err == nil {
		t.Fatal("durable owner did not survive transaction exit")
	}
	if _, err := m.Recover(context.Background(), q.OperationID, stopped); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Activate(context.Background(), other, stopped); err != nil {
		t.Fatal(err)
	}
	// Replaying an old restore must not touch a newer owner's identical binding.
	if _, err := m.Recover(context.Background(), q.OperationID, stopped); err == nil {
		t.Fatal("old record bypassed current owner")
	}
}

func TestSymlinkAndMissingOwnershipRefused(t *testing.T) {
	m, q := fixture(t)
	target := filepath.Join(filepath.Dir(q.BindingPath), "foreign")
	if err := os.WriteFile(target, []byte("foreign"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, q.BindingPath); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Activate(context.Background(), q, stopped); err == nil {
		t.Fatal("symlink overwritten")
	}
	if err := os.Remove(q.BindingPath); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Activate(context.Background(), q, nil); err == nil {
		t.Fatal("nil idle verification accepted")
	}
	if _, err := m.Activate(context.Background(), q, stopped); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(m.Dir, "active")); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Recover(context.Background(), q.OperationID, stopped); err == nil {
		t.Fatal("lost ownership treated as proof of stop")
	}
	if _, err := os.Stat(q.MarkerPath); err != nil {
		t.Fatal("unowned marker removed")
	}
}

func TestCrashRecoveryPresentBinding(t *testing.T) {
	for _, point := range []string{"marker_after", "binding_after", "restore_binding_after", "restore_marker_after"} {
		t.Run(point, func(t *testing.T) {
			m, q := fixture(t)
			old := []byte("version=1\nstate_db=\"" + testAbsPath("baseline.db") + "\"\nworkspace_root=\"" + testAbsPath("baseline") + "\"\n")
			if err := os.WriteFile(q.BindingPath, old, 0644); err != nil {
				t.Fatal(err)
			}
			crash := func() {
				defer func() {
					if recover() == nil {
						t.Error("did not reach checkpoint")
					}
				}()
				if point[:7] == "restore" {
					if _, err := m.Activate(context.Background(), q, stopped); err != nil {
						t.Fatal(err)
					}
				}
				m.checkpoint = func(p string) {
					if p == point {
						panic("simulated process exit")
					}
				}
				if point[:7] == "restore" {
					_, _ = m.Recover(context.Background(), q.OperationID, stopped)
				} else {
					_, _ = m.Activate(context.Background(), q, stopped)
				}
			}
			crash()
			m.checkpoint = nil
			if _, err := m.Recover(context.Background(), q.OperationID, stopped); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(q.BindingPath)
			if err != nil || string(got) != string(old) {
				t.Fatalf("baseline lost: %q %v", got, err)
			}
		})
	}
}

func TestLegacyStartDuringPauseKeepsOriginalBinding(t *testing.T) {
	m, q := fixture(t)
	calls := 0
	r, err := m.Activate(context.Background(), q, func(context.Context, Record) error {
		calls++
		if calls == 2 {
			return errors.New("legacy task started before pause")
		}
		return nil
	})
	if err == nil || r.CleanupPending == "" {
		t.Fatal("legacy start ignored")
	}
	if _, err := os.Stat(q.BindingPath); !os.IsNotExist(err) {
		t.Fatal("binding changed under legacy task")
	}
	if _, err := os.Stat(q.MarkerPath); err != nil {
		t.Fatal("pause not held")
	}
	if _, err := m.Recover(context.Background(), q.OperationID, stopped); err != nil {
		t.Fatal(err)
	}
}

// testAbsPath returns a host-absolute path for a binding fixture.
//
// The fixtures hardcoded "/tmp/...", which filepath.IsAbs REJECTS on Windows, so the binding
// validator refused every fixture there and four tests failed for a reason that had nothing
// to do with what they assert. Forward slashes after a drive letter are absolute on Windows
// and need no TOML escaping, unlike a backslash inside a basic string.
func testAbsPath(rel string) string {
	if runtime.GOOS == "windows" {
		return "C:/tmp/" + rel
	}
	return "/tmp/" + rel
}
