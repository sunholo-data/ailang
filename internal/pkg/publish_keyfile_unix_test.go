//go:build darwin || linux

package pkg

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
)

func dummyPublishFile(t *testing.T) string {
	t.Helper()
	t.Setenv(config.EnvAgentPolicy, "")
	path := filepath.Join(t.TempDir(), "key")
	if err := os.WriteFile(path, []byte("dummy-publish-key\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPublishKeyFile_ProtectedModes(t *testing.T) {
	for _, mode := range []os.FileMode{0400, 0600, 0644, 0640, 0700, 0600 | os.ModeSetuid} {
		t.Run(mode.String(), func(t *testing.T) {
			path := dummyPublishFile(t)
			if err := os.Chmod(path, mode); err != nil {
				t.Fatal(err)
			}
			data, err := readPublishKeyBytes(path)
			allowed := mode == 0400 || mode == 0600
			if (err == nil) != allowed {
				t.Fatalf("mode%v accepted=%v err=%v", mode, err == nil, err)
			}
			if allowed && string(data) != "dummy-publish-key\n" {
				t.Fatal("wrong protected bytes")
			}
			if err != nil && strings.Contains(err.Error(), "dummy-publish-key") {
				t.Fatal("secret in error")
			}
		})
	}
}

func TestPublishKeyFile_RefusesSymlinkDirectoryAndOwner(t *testing.T) {
	path := dummyPublishFile(t)
	link := path + ".link"
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{link, filepath.Dir(path), path + ".missing"} {
		if _, err := readPublishKeyBytes(name); err == nil {
			t.Fatalf("unsafe path accepted: %s", name)
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	uid := uint32(os.Geteuid())
	if err := validatePublishKeyInfo(info, uid+1, uid); err == nil {
		t.Fatal("foreign owner accepted")
	}
}

func TestPublishKeyFile_SameValidatedHandle(t *testing.T) {
	path := dummyPublishFile(t)
	f, err := openPublishKeyFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := os.Rename(path, path+".old"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("replacement-dummy"), 0600); err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(f)
	if err != nil || string(data) != "dummy-publish-key\n" {
		t.Fatalf("validated handle changed: %q %v", data, err)
	}
}

type endlessPublishReader struct{ bytes int }

func (r *endlessPublishReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}
	r.bytes += len(p)
	return len(p), nil
}
func TestPublishKeyFile_BoundedStream(t *testing.T) {
	r := &endlessPublishReader{}
	if _, err := readPublishKeyStream(r); err == nil {
		t.Fatal("growing input accepted")
	}
	if r.bytes != 16*1024+1 {
		t.Fatalf("read %d bytes from unbounded source", r.bytes)
	}
	path := dummyPublishFile(t)
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 16*1024+1)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readPublishKeyBytes(path); err == nil {
		t.Fatal("oversized file accepted")
	}
}

func TestPublishKeyFile_FIFOIsBounded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "key.fifo")
	if err := syscall.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(config.EnvAgentPolicy, "")
	done := make(chan error, 1)
	go func() { _, err := readPublishKeyBytes(path); done <- err }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("FIFO accepted")
		}
	case <-time.After(time.Second):
		// Release a reader regressed to blocking open, without a surviving process.
		fd, err := syscall.Open(path, syscall.O_RDWR|syscall.O_NONBLOCK, 0)
		if err == nil {
			_ = syscall.Close(fd)
		}
		select {
		case <-done:
		case <-time.After(time.Second):
		}
		t.Fatal("FIFO reader blocked before validating the file type")
	}
}
