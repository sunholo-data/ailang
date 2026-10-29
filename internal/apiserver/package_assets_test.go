package apiserver

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/testutil/pkgassets"
)

func TestAPIHandlerLockedAssetBytes(t *testing.T) {
	root, _ := pkgassets.New(t)
	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.NewCapability("FS"))
	srv := New(root, Config{Port: "0", EffCtx: ctx})
	defer srv.Close()
	if err := srv.LoadModules([]string{filepath.Join(root, "asset_fixture.ail")}); err != nil {
		t.Fatal(err)
	}
	handler := srv.buildRoutes()
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/api/asset_fixture/read", strings.NewReader(`{"args":[]}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		var resp FunctionCallResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if w.Code != 200 || resp.Error != "" || resp.Result != "locked" {
			t.Fatalf("%d %s", w.Code, w.Body.String())
		}
	}
}
