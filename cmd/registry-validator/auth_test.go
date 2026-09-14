package main

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/pkg"
)

// memKeyStore is the test double for the Firestore store.
type memKeyStore struct {
	mu   sync.Mutex
	recs map[string]*keyRecord
	err  error // if set, every call fails with it
}

func newMemKeyStore() *memKeyStore { return &memKeyStore{recs: map[string]*keyRecord{}} }

func (m *memKeyStore) Get(_ context.Context, id string) (*keyRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	rec, ok := m.recs[id]
	if !ok {
		return nil, errKeyNotFound
	}
	cp := *rec
	cp.ID = id
	return &cp, nil
}

func (m *memKeyStore) Put(_ context.Context, id string, rec *keyRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	cp := *rec
	m.recs[id] = &cp
	return nil
}

func (m *memKeyStore) List(_ context.Context) ([]*keyRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*keyRecord
	for id, rec := range m.recs {
		cp := *rec
		cp.ID = id
		out = append(out, &cp)
	}
	return out, nil
}

func (m *memKeyStore) Revoke(_ context.Context, id string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.recs[id]
	if !ok {
		return errKeyNotFound
	}
	rec.RevokedAt = &at
	return nil
}

const testSuperKey = "super-secret"

// scopedValidator returns a validator with a superuser key and one scoped key
// for owner "daneel" covering the given scopes. Returns the plaintext key.
func scopedValidator(t *testing.T, scopes ...string) (*validator, string) {
	t.Helper()
	store := newMemKeyStore()
	key, id, err := newScopedKey()
	if err != nil {
		t.Fatal(err)
	}
	store.Put(context.Background(), id, &keyRecord{Owner: "daneel", Scopes: scopes, CreatedAt: time.Now()})
	return &validator{apiKey: testSuperKey, keys: store}, key
}

func publishReq(t *testing.T, pkgName, key string) *http.Request {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "ailang.toml"), []byte(`
[package]
name = "`+pkgName+`"
version = "0.1.0"
edition = "1"

[exports]
modules = ["`+pkgName+`/core"]

[effects]
max = []

[stability]
level = "experimental"
`), 0644)
	os.WriteFile(filepath.Join(dir, "core.ail"), []byte("module "+pkgName+"/core\n\nexport pure func add(a: int, b: int) -> int = a + b\n"), 0644)
	tarball, err := pkg.CreateTarball(dir)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, _ := mw.CreateFormFile("package", "package.tar.gz")
	part.Write(tarball)
	mw.Close()
	req := httptest.NewRequest("POST", "/publish", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if key != "" {
		req.Header.Set("X-API-Key", key)
	}
	return req
}

func errBody(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp["error"]
}

func TestPrincipal_CanWrite(t *testing.T) {
	cases := []struct {
		scopes []string
		pkg    string
		want   bool
	}{
		{[]string{"*"}, "anything/at_all", true},
		{[]string{"daneel/*"}, "daneel/foo", true},
		{[]string{"daneel/*"}, "sunholo/foo", false},
		{[]string{"daneel/*"}, "daneel", false},
		{[]string{"sunholo/daneel_*"}, "sunholo/daneel_tools", true},
		{[]string{"sunholo/daneel_*"}, "sunholo/tools", false},
		{[]string{"sunholo/exact"}, "sunholo/exact", true},
		{[]string{"sunholo/exact"}, "sunholo/exact2", false},
		{[]string{"a/*", "b/*"}, "b/x", true},
		{nil, "daneel/foo", false},
	}
	for _, c := range cases {
		p := principal{Owner: "x", Scopes: c.scopes}
		if got := p.canWrite(c.pkg); got != c.want {
			t.Errorf("scopes=%v pkg=%s: got %v want %v", c.scopes, c.pkg, got, c.want)
		}
	}
}

func TestValidateScopes(t *testing.T) {
	if err := validateScopes(nil); err == nil {
		t.Error("empty scopes should be rejected")
	}
	if err := validateScopes([]string{"*"}); err == nil {
		t.Error(`"*" should be reserved for superuser`)
	}
	if err := validateScopes([]string{"daneel"}); err == nil {
		t.Error("scope without a slash should be rejected")
	}
	if err := validateScopes([]string{"daneel/["}); err == nil {
		t.Error("malformed glob should be rejected")
	}
	if err := validateScopes([]string{"daneel/*", "sunholo/daneel_*"}); err != nil {
		t.Errorf("valid scopes rejected: %v", err)
	}
}

func TestAuthenticate(t *testing.T) {
	v, key := scopedValidator(t, "daneel/*")
	ctx := context.Background()

	mk := func(k string) *http.Request {
		r := httptest.NewRequest("POST", "/publish", nil)
		if k != "" {
			r.Header.Set("X-API-Key", k)
		}
		return r
	}

	if p, aerr := v.authenticate(ctx, mk(testSuperKey)); aerr != nil || !p.isSuperuser() {
		t.Errorf("superuser key: got %+v, %v", p, aerr)
	}
	if p, aerr := v.authenticate(ctx, mk(key)); aerr != nil || p.Owner != "daneel" {
		t.Errorf("scoped key: got %+v, %v", p, aerr)
	}
	if _, aerr := v.authenticate(ctx, mk("")); aerr == nil || aerr.status != http.StatusForbidden {
		t.Errorf("missing key: want 403, got %v", aerr)
	}
	if _, aerr := v.authenticate(ctx, mk("ailr_nope")); aerr == nil || aerr.status != http.StatusForbidden {
		t.Errorf("unknown scoped key: want 403, got %v", aerr)
	}
	if _, aerr := v.authenticate(ctx, mk("not-a-key")); aerr == nil || aerr.status != http.StatusForbidden {
		t.Errorf("garbage key: want 403, got %v", aerr)
	}

	// query param form still works
	r := httptest.NewRequest("POST", "/publish?api_key="+key, nil)
	if p, aerr := v.authenticate(ctx, r); aerr != nil || p.Owner != "daneel" {
		t.Errorf("query-param key: got %+v, %v", p, aerr)
	}
}

func TestAuthenticate_Revoked(t *testing.T) {
	v, key := scopedValidator(t, "daneel/*")
	ctx := context.Background()
	if err := v.keys.Revoke(ctx, keyID(key), time.Now()); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "/publish", nil)
	r.Header.Set("X-API-Key", key)
	_, aerr := v.authenticate(ctx, r)
	if aerr == nil || aerr.status != http.StatusForbidden || !strings.Contains(aerr.msg, "revoked") {
		t.Errorf("revoked key: want 403 mentioning revoked, got %v", aerr)
	}
}

func TestAuthenticate_NoStoreIsLoud(t *testing.T) {
	v := &validator{apiKey: testSuperKey} // keys == nil
	r := httptest.NewRequest("POST", "/publish", nil)
	r.Header.Set("X-API-Key", "ailr_whatever")
	_, aerr := v.authenticate(context.Background(), r)
	if aerr == nil || !strings.Contains(aerr.msg, "not configured") {
		t.Errorf("scoped key without a store should say so, got %v", aerr)
	}
}

func TestAuthenticate_StoreErrorIs500(t *testing.T) {
	store := newMemKeyStore()
	store.err = context.DeadlineExceeded
	v := &validator{apiKey: testSuperKey, keys: store}
	r := httptest.NewRequest("POST", "/publish", nil)
	r.Header.Set("X-API-Key", "ailr_whatever")
	_, aerr := v.authenticate(context.Background(), r)
	if aerr == nil || aerr.status != http.StatusInternalServerError {
		t.Errorf("store failure must not masquerade as 403, got %v", aerr)
	}
}

func TestPublish_ScopedKey_OutOfScopeIs403(t *testing.T) {
	v, key := scopedValidator(t, "daneel/*")
	w := httptest.NewRecorder()
	v.handlePublish(w, publishReq(t, "sunholo/foo", key))
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d: %s", w.Code, w.Body.String())
	}
	msg := errBody(t, w)
	if !strings.Contains(msg, `"daneel"`) || !strings.Contains(msg, "sunholo/foo") {
		t.Errorf("403 should name the owner and package, got: %s", msg)
	}
}

func TestPublish_ScopedKey_InScopePassesAuth(t *testing.T) {
	v, key := scopedValidator(t, "daneel/*")
	w := httptest.NewRecorder()
	v.handlePublish(w, publishReq(t, "daneel/foo", key))
	// Past auth, the pipeline hits ailang check / GCS which may fail in the
	// test env; the assertion is only that auth did not reject it.
	if w.Code == http.StatusForbidden {
		t.Fatalf("in-scope publish rejected: %s", w.Body.String())
	}
}

func TestPublish_NoKeyIs403BeforeBody(t *testing.T) {
	v, _ := scopedValidator(t, "daneel/*")
	w := httptest.NewRecorder()
	v.handlePublish(w, publishReq(t, "daneel/foo", ""))
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPublish_SuperuserAnyNamespace(t *testing.T) {
	v, _ := scopedValidator(t, "daneel/*")
	w := httptest.NewRecorder()
	v.handlePublish(w, publishReq(t, "sunholo/foo", testSuperKey))
	if w.Code == http.StatusForbidden {
		t.Fatalf("superuser rejected: %s", w.Body.String())
	}
}

func TestUnpublish_ScopedKey_OutOfScopeIs403(t *testing.T) {
	v, key := scopedValidator(t, "daneel/*")
	r := httptest.NewRequest(http.MethodDelete, "/unpublish?name=sunholo/foo&version=0.1.0", nil)
	r.Header.Set("X-API-Key", key)
	w := httptest.NewRecorder()
	v.handleUnpublish(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRebuildIndex_ScopedKeyIs403(t *testing.T) {
	v, key := scopedValidator(t, "daneel/*")
	r := httptest.NewRequest(http.MethodPost, "/rebuild-index", nil)
	r.Header.Set("X-API-Key", key)
	w := httptest.NewRecorder()
	v.handleRebuildIndex(w, r)
	if w.Code != http.StatusForbidden || !strings.Contains(errBody(t, w), "superuser") {
		t.Fatalf("want 403 superuser-only, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminKeys_Lifecycle(t *testing.T) {
	v := &validator{apiKey: testSuperKey, keys: newMemKeyStore()}

	// Scoped key may not mint.
	_, scoped := scopedValidator(t, "daneel/*")
	r := httptest.NewRequest(http.MethodPost, "/admin/keys", strings.NewReader(`{"owner":"x","scopes":["x/*"]}`))
	r.Header.Set("X-API-Key", scoped)
	w := httptest.NewRecorder()
	v.handleAdminKeys(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("scoped key minting: want 403, got %d", w.Code)
	}

	// Superuser mints.
	r = httptest.NewRequest(http.MethodPost, "/admin/keys", strings.NewReader(`{"owner":"daneel","scopes":["daneel/*"],"note":"test"}`))
	r.Header.Set("X-API-Key", testSuperKey)
	w = httptest.NewRecorder()
	v.handleAdminKeys(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("mint: want 201, got %d: %s", w.Code, w.Body.String())
	}
	var minted struct {
		ID, Key, Owner string
		Scopes         []string
	}
	json.Unmarshal(w.Body.Bytes(), &minted)
	if !strings.HasPrefix(minted.Key, keyPrefix) || minted.ID != keyID(minted.Key) || minted.Owner != "daneel" {
		t.Fatalf("mint response malformed: %+v", minted)
	}

	// The minted key works, and stamps published_by.
	w = httptest.NewRecorder()
	v.handlePublish(w, publishReq(t, "daneel/bar", minted.Key))
	if w.Code == http.StatusForbidden {
		t.Fatalf("minted key rejected: %s", w.Body.String())
	}

	// List shows it without the plaintext.
	r = httptest.NewRequest(http.MethodGet, "/admin/keys", nil)
	r.Header.Set("X-API-Key", testSuperKey)
	w = httptest.NewRecorder()
	v.handleAdminKeys(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), minted.ID) || strings.Contains(w.Body.String(), minted.Key) {
		t.Fatalf("list: got %d: %s", w.Code, w.Body.String())
	}

	// Revoke, then the key is dead.
	r = httptest.NewRequest(http.MethodDelete, "/admin/keys/"+minted.ID, nil)
	r.Header.Set("X-API-Key", testSuperKey)
	w = httptest.NewRecorder()
	v.handleAdminKeys(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("revoke: got %d: %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	v.handlePublish(w, publishReq(t, "daneel/bar", minted.Key))
	if w.Code != http.StatusForbidden {
		t.Fatalf("revoked key should be 403, got %d", w.Code)
	}

	// Revoking twice / unknown id → 404.
	r = httptest.NewRequest(http.MethodDelete, "/admin/keys/deadbeef", nil)
	r.Header.Set("X-API-Key", testSuperKey)
	w = httptest.NewRecorder()
	v.handleAdminKeys(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("unknown revoke: want 404, got %d", w.Code)
	}
}

func TestAdminKeys_BadRequests(t *testing.T) {
	v := &validator{apiKey: testSuperKey, keys: newMemKeyStore()}
	for _, body := range []string{
		`{"owner":"","scopes":["x/*"]}`,
		`{"owner":"superuser","scopes":["x/*"]}`,
		`{"owner":"x","scopes":[]}`,
		`{"owner":"x","scopes":["*"]}`,
		`not json`,
	} {
		r := httptest.NewRequest(http.MethodPost, "/admin/keys", strings.NewReader(body))
		r.Header.Set("X-API-Key", testSuperKey)
		w := httptest.NewRecorder()
		v.handleAdminKeys(w, r)
		if w.Code != http.StatusBadRequest {
			t.Errorf("body %s: want 400, got %d", body, w.Code)
		}
	}
}
