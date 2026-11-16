package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pkg"
)

// bombTarball builds a tar.gz whose entries are runs of zeros: tiny on the
// wire, large once extracted.
func bombTarball(t *testing.T, sizes map[string]int64) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	zeros := make([]byte, 64<<10)
	for name, size := range sizes {
		if err := tw.WriteHeader(&tar.Header{Name: name, Size: size, Mode: 0o644, Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		for left := size; left > 0; {
			n := int64(len(zeros))
			if left < n {
				n = left
			}
			if _, err := tw.Write(zeros[:n]); err != nil {
				t.Fatal(err)
			}
			left -= n
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func multipartPublish(t *testing.T, tarball []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, _ := mw.CreateFormFile("package", "package.tar.gz")
	part.Write(tarball)
	mw.Close()
	req := httptest.NewRequest("POST", "/publish", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestPublish_LimitsRefusedWith413(t *testing.T) {
	entries := map[string]int64{}
	for i := 0; i <= pkg.MaxTarballEntries; i++ {
		entries[fmt.Sprintf("m%04d.ail", i)] = 1
	}
	total := map[string]int64{}
	for i := 0; i <= pkg.MaxTarballTotalBytes/pkg.MaxTarballEntryBytes; i++ {
		total[fmt.Sprintf("assets/p%02d.bin", i)] = pkg.MaxTarballEntryBytes
	}
	cases := []struct {
		name, limit string
		tarball     []byte
	}{
		{"oversized entry", "pkg.MaxTarballEntryBytes",
			bombTarball(t, map[string]int64{"assets/bomb.bin": pkg.MaxTarballEntryBytes + 1})},
		{"oversized total", "pkg.MaxTarballTotalBytes", bombTarball(t, total)},
		{"too many entries", "pkg.MaxTarballEntries", bombTarball(t, entries)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			(&validator{}).handlePublish(w, multipartPublish(t, tc.tarball))
			if w.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("want 413, got %d %q", w.Code, errBody(t, w))
			}
			if msg := errBody(t, w); !strings.Contains(msg, tc.limit) {
				t.Fatalf("error must name %s: %q", tc.limit, msg)
			}
		})
	}
}

func TestPublish_OversizedBodyRefusedWith413(t *testing.T) {
	w := httptest.NewRecorder()
	(&validator{}).handlePublish(w, multipartPublish(t, make([]byte, maxPublishBodyBytes)))
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("want 413, got %d %q", w.Code, errBody(t, w))
	}
	if msg := errBody(t, w); !strings.Contains(msg, "pkg.MaxTarballBytes") {
		t.Fatalf("error must name pkg.MaxTarballBytes: %q", msg)
	}
}
