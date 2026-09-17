package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/smt"
)

// M-PKG-QUALITY-LADDER M2 — the seam between the validator and `ailang`.
//
// The validator shells out to the `ailang` on PATH (exactly as deployed), so
// these tests build the worktree's binary and put it first on PATH: the
// contract-count fix lives in that binary's `verify --package`, and a stale
// system binary would make the seam test pass or fail for the wrong reason.

func buildWorktreeAilangOnPath(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "ailang")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-o", bin, "./cmd/ailang")
	build.Dir = filepath.Join("..", "..")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build ailang: %v\n%s", err, out)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func postFixtureTarball(t *testing.T, v *validator, fixture string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	dir := filepath.Join("..", "..", "internal", "pkg", "testdata", "quality", fixture)
	tarball, err := pkg.CreateTarball(dir)
	if err != nil {
		t.Fatalf("CreateTarball: %v", err)
	}
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("package", "package.tar.gz")
	_, _ = part.Write(tarball)
	_ = writer.Close()
	req := httptest.NewRequest("POST", "/publish", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	for k, val := range headers {
		req.Header.Set(k, val)
	}
	w := httptest.NewRecorder()
	v.handlePublish(w, req)
	return w
}

// Kills: (a) decoding a bare array in runAilangVerify (counts stay 0);
// (b) dropping the InterfaceHashV2 call (hash empty); (c) banking schema v1.
func TestPublish_BanksContractsAndV2Identity(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("z3 not available (Windows CI has no z3)")
	}
	buildWorktreeAilangOnPath(t)
	v := &validator{} // validation-only mode (no bucket)

	w := postFixtureTarball(t, v, "flat_self_import", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("publish exited %d: %s", w.Code, w.Body.String())
	}
	var meta pkg.PackageMetadata
	if err := json.Unmarshal(w.Body.Bytes(), &meta); err != nil {
		t.Fatalf("metadata decode: %v\n%s", err, w.Body.String())
	}
	if meta.Schema != pkg.PackageMetadataSchemaV2 {
		t.Errorf("schema = %q, want %q", meta.Schema, pkg.PackageMetadataSchemaV2)
	}
	if meta.Validation.ContractsError != "" {
		t.Fatalf("contracts_error = %q", meta.Validation.ContractsError)
	}
	if meta.Validation.ContractsVerified != 2 || meta.Validation.ContractsTotal != 2 {
		t.Errorf("contracts verified/total = %d/%d, want 2/2", meta.Validation.ContractsVerified, meta.Validation.ContractsTotal)
	}
	if !strings.HasPrefix(meta.InterfaceHashV2, "sha256:ifacev2:") {
		t.Errorf("interface_hash_v2 = %q (error %q)", meta.InterfaceHashV2, meta.InterfaceV2Error)
	}
	if len(meta.InterfaceSignatures) == 0 || !strings.Contains(strings.Join(meta.InterfaceSignatures, "\n"), "capAt") {
		t.Errorf("interface_signatures missing capAt: %v", meta.InterfaceSignatures)
	}
	if streak, fails := v.v2Outcomes(); streak != 1 || fails != 0 {
		t.Errorf("v2 streak/fails = %d/%d, want 1/0", streak, fails)
	}
}

// Version skew: a publisher whose binary computes a different v2 identity is
// refused with both hashes, never silently overwritten (parent doc M7).
func TestPublish_RefusesInterfaceIdentitySkew(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("z3 not available (Windows CI has no z3)")
	}
	buildWorktreeAilangOnPath(t)
	v := &validator{}

	w := postFixtureTarball(t, v, "flat_self_import", map[string]string{"X-Interface-Hash-V2": "sha256:ifacev2:deadbeef"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "PUB005") || !strings.Contains(body, "sha256:ifacev2:deadbeef") || !strings.Contains(body, "validator computed sha256:ifacev2:") {
		t.Errorf("skew refusal must name PUB005 and both hashes:\n%s", body)
	}
}

// A v1 metadata.json (no v2 fields) must keep decoding — every published
// version today is v1 and `pkg info`/the explorer read them.
func TestPackageMetadata_V1RecordStillDecodes(t *testing.T) {
	v1 := `{"schema":"ailang.package-metadata/v1","name":"sunholo/gcp_auth","version":"0.8.1",
	  "published_at":"2026-04-20T15:23:16Z","published_by":"","content_hash":"sha256:a","interface_hash":"sha256:b",
	  "tarball_hash":"sha256:c","tarball_size_bytes":3364,
	  "validation":{"compiles":true,"effects_valid":true,"contracts_verified":0,"contracts_total":0,"contracts_skipped":0,"ailang_version":"AILANG dev"},
	  "manifest":{"edition":"1","effects_max":["FS","Net","Env"],"exports":["sunholo/gcp_auth/token"],"stability":"experimental","ai_summary":"x","has_agent_doc":true}}`
	var meta pkg.PackageMetadata
	if err := json.Unmarshal([]byte(v1), &meta); err != nil {
		t.Fatalf("v1 decode: %v", err)
	}
	if meta.InterfaceHashV2 != "" || meta.Validation.ContractsError != "" || meta.Schema != pkg.PackageMetadataSchemaV1 {
		t.Errorf("v1 record gained phantom v2 fields: %+v", meta)
	}
}
