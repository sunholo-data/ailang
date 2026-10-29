package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// pkgVerbAuthority is the reviewed declaration of every pkg verb's authority
// (#1720 D5). nil = the verb needs none and is not refused at dispatch (its
// I/O is still gated in internal/pkg). A verb added to pkgSubcommands()
// without a row here fails TestPkgVerbAuthorityComplete, so nobody adds a
// verb without deciding.
var pkgVerbAuthority = map[string][]pkg.Authority{
	"docs":           nil, // provisioned read; registry fallback gated by I/O
	"add":            {pkg.AuthNetwork, pkg.AuthHomeWrite},
	"lock":           nil, // agent-usable via policy-tool; D2 carry-forward
	"tree":           nil,
	"install":        {pkg.AuthNetwork, pkg.AuthHomeWrite},
	"bin":            {pkg.AuthHomeWrite}, // uninstall only; list is read-only
	"search":         nil,                 // gated by the registry transport
	"publish":        {pkg.AuthNetwork, pkg.AuthHomeWrite},
	"unpublish":      {pkg.AuthNetwork},
	"quality":        nil,
	"info":           nil,
	"versions":       nil,
	"stats":          nil,
	"provenance":     nil,
	"history":        nil,
	"notify-upgrade": {pkg.AuthHomeWrite},
	"affected-by":    nil,
	"cascade":        {pkg.AuthNetwork},
	"key":            {pkg.AuthNetwork},
}

func TestPkgVerbAuthorityComplete(t *testing.T) {
	seen := map[string]bool{}
	for _, sub := range pkgSubcommands() {
		seen[sub.Name] = true
		want, ok := pkgVerbAuthority[sub.Name]
		if !ok {
			t.Errorf("pkg verb %q has no authority declaration in pkgVerbAuthority — decide what it needs (#1720 D5)", sub.Name)
			continue
		}
		if !reflect.DeepEqual(sub.Authority, want) && !(len(sub.Authority) == 0 && len(want) == 0) {
			t.Errorf("pkg %s: Command.Authority = %v, reviewed declaration %v", sub.Name, sub.Authority, want)
		}
	}
	for name := range pkgVerbAuthority {
		if !seen[name] {
			t.Errorf("pkgVerbAuthority names %q, which is not a pkg verb", name)
		}
	}
	// The legacy bare rows must carry the same declaration and the gate.
	for verb, top := range pkgLegacyTopLevel {
		c, ok := lookupCommand(top)
		if !ok {
			continue
		}
		if !reflect.DeepEqual(c.Authority, pkgVerbAuthority[verb]) && !(len(c.Authority) == 0 && len(pkgVerbAuthority[verb]) == 0) {
			t.Errorf("legacy `ailang %s` Authority %v differs from `pkg %s`", top, c.Authority, verb)
		}
	}
}

// D5: a confined operator verb is refused at dispatch — before it parses a
// manifest or writes anything — on both spellings. Help is never refused.
func TestPkgOperatorVerbsRefusedAtDispatch(t *testing.T) {
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	t.Setenv(config.EnvAgentPolicy, "policy.toml")
	t.Chdir(t.TempDir()) // no ailang.toml: a verb that ran would fail differently
	var add Command
	for _, sub := range pkgSubcommands() {
		if sub.Name == "add" {
			add = sub
		}
	}
	err := gatedRun(add)([]string{"--git", "https://example.invalid/r", "--tag", "v1"})
	if err == nil || err.Error() != "confined by AILANG_AGENT_POLICY: 'ailang pkg add' is operator authority — run it outside the sandbox" {
		t.Fatalf("confined `pkg add` not refused at dispatch: %v", err)
	}
	legacy, ok := lookupCommand("add")
	if !ok {
		t.Fatal("bare `ailang add` route missing")
	}
	if err := legacy.Run([]string{"--path", "x"}); !pkg.IsConfinedError(err) {
		t.Fatalf("confined bare `ailang add` not refused: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".ailang")); !os.IsNotExist(err) {
		t.Fatalf("HOME changed: %v", err)
	}

	var bin Command
	for _, sub := range pkgSubcommands() {
		if sub.Name == "bin" {
			bin = sub
		}
	}
	if err := gatedRun(bin)([]string{"uninstall", "x"}); !pkg.IsConfinedError(err) {
		t.Fatalf("confined `bin uninstall` not refused: %v", err)
	}
	if err := gatedRun(bin)([]string{"list", "--bin-dir", t.TempDir()}); pkg.IsConfinedError(err) {
		t.Fatalf("read-only `bin list` refused: %v", err)
	}
	if err := gatedRun(add)([]string{"--help"}); err != nil && strings.Contains(err.Error(), "confined") {
		t.Fatalf("help refused: %v", err)
	}
}
