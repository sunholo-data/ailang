package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/types"
)

// M-EXECUTOR-POLICY-HARDENING M3 — the typed, immutable policy and the
// admission ordering fix.

// An OPEN row with no concrete labels (`{ | e }`) is still parametric: it can
// instantiate to anything. Check used to return "admitted" for
// len(Labels)==0 before it looked at the tail (V12).
func TestCheck_OpenEmptyRowDenied(t *testing.T) {
	p := &Policy{AllowedCaps: []string{"IO"}}
	tail := &types.RowVar{Name: "e", Kind: types.EffectRow}
	row := mkRow(nil, tail)
	if row == nil || row.Tail == nil {
		t.Fatal("fixture: expected an open row with an empty label set")
	}
	d := Check(p, "main", row)
	if d.OK {
		t.Fatalf("open empty row must be rejected, got %+v", d)
	}
	if d.ErrorKind != KindParametricEntry {
		t.Errorf("error_kind = %q, want %q", d.ErrorKind, KindParametricEntry)
	}
}

func restricted(caps ...string) *Policy {
	p := DefaultPolicy()
	p.AllowedCaps = caps
	p.FSSandbox = "/tmp/sb"
	for _, c := range caps {
		if c == "Net" || c == "Stream" {
			p.NetAllow = []string{"api.example"}
		}
	}
	return p
}

func TestResolve_DefaultsAreRestricted(t *testing.T) {
	r, err := Resolve(restricted("IO", "FS"), "digest")
	if err != nil {
		t.Fatal(err)
	}
	if r.Mode != ModeRestricted {
		t.Errorf("absent security_mode must resolve to restricted, got %q", r.Mode)
	}
	if r.Timeout != 5*time.Second {
		t.Errorf("default timeout = %v, want 5s", r.Timeout)
	}
	if r.MaxSourceBytes != DefaultRestrictedMaxSourceBytes || r.MaxModuleGraphBytes != DefaultRestrictedMaxModuleGraphBytes ||
		r.MaxOutputBytes != DefaultRestrictedMaxOutputBytes || r.MaxFSTransferBytes != DefaultRestrictedMaxFSTransferBytes {
		t.Errorf("restricted defaults not applied: %+v", r)
	}
	if r.Digest != "digest" || r.Entry != "main" {
		t.Errorf("digest/entry not carried: %+v", r)
	}
	if !r.Restricted() {
		t.Error("Restricted() must be true")
	}
}

func TestResolve_RestrictedRefusesUnadaptedEffects(t *testing.T) {
	// Every registry label outside RestrictedEffects — the registry, not a
	// hand list, is what a future addition would extend.
	// AI (M7) and confined Process (M6) are admitted with their conditions;
	// the labels below have no adapter at all.
	for _, cap := range []string{"Process", "Env", "Secret", "Cog", "DOM", "Msg", "Debug", "Trace", "DB", "Async", "SharedMem", "SharedIndex"} {
		p := restricted("IO", cap)
		if cap == "Process" {
			// Process IS admitted for confined entries (M6); an entry with no
			// confined schema is what restricted mode refuses.
			p.ProcessAllow = []string{"sh"}
		}
		_, err := Resolve(p, "d")
		if err == nil {
			t.Errorf("restricted mode must refuse %s", cap)
			continue
		}
		if !strings.Contains(err.Error(), cap) || !strings.Contains(err.Error(), ModeTrustedHost) {
			t.Errorf("%s: refusal must name the effect and the trusted_host migration, got %v", cap, err)
		}
	}
}

// #1557: the policy keyed "known capability" on the effect op registry, so
// labels with no runtime ops there (Declassify, Rand) were refused as unknown
// in every mode, and a program declaring them could never run under --policy.
// Every canonical effect must at least be KNOWN; restricted mode then decides
// whether it has an adapter.
func TestResolve_EveryCanonicalEffectIsKnown(t *testing.T) {
	for _, cap := range types.KnownEffectNames() {
		p := restricted("IO", cap)
		p.SecurityMode = ModeTrustedHost
		_, err := Resolve(p, "d")
		if err != nil && strings.Contains(err.Error(), "unknown capability") {
			t.Errorf("%s is a canonical effect but the policy calls it unknown: %v", cap, err)
		}
	}
}

func TestResolve_RestrictedAdmitsDeclassifyAndRand(t *testing.T) {
	for _, cap := range []string{"Declassify", "Rand"} {
		r, err := Resolve(restricted("IO", cap), "d")
		if err != nil {
			t.Errorf("restricted mode must admit %s: %v", cap, err)
			continue
		}
		if !r.Admits(cap) {
			t.Errorf("resolved policy does not admit %s: %v", cap, r.Effects)
		}
	}
}

func TestResolve_TrustedHostKeepsProcess(t *testing.T) {
	p := restricted("IO", "Process")
	p.SecurityMode = ModeTrustedHost
	p.ProcessAllow = []string{"git:status"}
	r, err := Resolve(p, "d")
	if err != nil {
		t.Fatalf("trusted_host must keep Process: %v", err)
	}
	if r.Mode != ModeTrustedHost || r.Restricted() {
		t.Errorf("mode = %q", r.Mode)
	}
	// trusted_host has no restricted defaults: unset caps stay unlimited.
	if r.MaxOutputBytes != 0 || r.MaxModuleGraphBytes != 0 {
		t.Errorf("trusted_host must not inherit restricted caps: %+v", r)
	}
}

func TestResolve_UnknownModeRefused(t *testing.T) {
	p := restricted("IO")
	p.SecurityMode = "sandboxed"
	if _, err := Resolve(p, "d"); err == nil || !strings.Contains(err.Error(), "security_mode") {
		t.Fatalf("unknown mode must be refused by name, got %v", err)
	}
}

func TestResolve_TimeoutValidation(t *testing.T) {
	for _, ms := range []int{0, -1, 25 * 60 * 60 * 1000} {
		p := restricted("IO")
		p.TimeoutMs = ms
		if _, err := Resolve(p, "d"); err == nil || !strings.Contains(err.Error(), "timeout_ms") {
			t.Errorf("timeout_ms=%d must be refused by name, got %v", ms, err)
		}
	}
	p := restricted("IO")
	p.TimeoutMs = 250
	r, err := Resolve(p, "d")
	if err != nil || r.Timeout != 250*time.Millisecond {
		t.Errorf("timeout_ms=250 → %v, %v", r, err)
	}
}

func TestResolve_ConsistencyRules(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*Policy)
		want string
	}{
		{"FS without sandbox", func(p *Policy) { p.FSSandbox = "" }, "fs_sandbox"},
		{"Net without net_allow", func(p *Policy) { p.NetAllow = nil }, "net_allow"},
		{"net_allow without Net", func(p *Policy) { p.AllowedCaps = []string{"IO"} }, "net_allow"},
		{"unknown cap", func(p *Policy) { p.AllowedCaps = []string{"Wifi"} }, "Wifi"},
		{"negative max_source_bytes", func(p *Policy) { p.MaxSourceBytes = -1 }, "max_source_bytes"},
		{"negative max_output_bytes", func(p *Policy) { p.MaxOutputBytes = -1 }, "max_output_bytes"},
		{"budget negative", func(p *Policy) { p.Budgets = map[string]int{"FS": -1} }, "budgets"},
		{"budget for unadmitted effect", func(p *Policy) { p.Budgets = map[string]int{"Clock": 3} }, "budgets"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := restricted("IO", "FS", "Net")
			c.mut(p)
			_, err := Resolve(p, "d")
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want an error naming %q, got %v", c.want, err)
			}
		})
	}
}

func TestResolve_ExplicitZeroBudgetIsZero(t *testing.T) {
	p := restricted("IO", "FS")
	p.Budgets = map[string]int{"FS": 0}
	r, err := Resolve(p, "d")
	if err != nil {
		t.Fatal(err)
	}
	n, ok := r.Budgets["FS"]
	if !ok || n != 0 {
		t.Errorf("explicit zero must survive as zero: %v", r.Budgets)
	}
	if _, ok := r.Budgets["IO"]; ok {
		t.Error("an omitted budget is unlimited: it must not appear")
	}
}

func TestResolve_ProcessAllowSyntax(t *testing.T) {
	p := restricted("IO", "Process")
	p.SecurityMode = ModeTrustedHost
	p.ProcessAllow = []string{"git:", ""}
	if _, err := Resolve(p, "d"); err == nil || !strings.Contains(err.Error(), "process_allow") {
		t.Fatalf("malformed process_allow must be refused by name, got %v", err)
	}
}

// LoadResolved reads the bytes ONCE and digests exactly what it decoded.
func TestLoadResolved_DigestsAcceptedBytes(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "p.toml")
	body := "allowed_caps = [\"IO\"]\nentry = \"main\"\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	r, raw, err := LoadResolved(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != body {
		t.Errorf("raw bytes must be the accepted bytes")
	}
	if r.Digest != DigestBytes([]byte(body)) {
		t.Errorf("digest must be over the accepted bytes")
	}
	if r.Mode != ModeRestricted {
		t.Errorf("mode = %q", r.Mode)
	}
}

func TestLoad_SecurityModeField(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "p.toml")
	if err := os.WriteFile(p, []byte("allowed_caps = [\"IO\"]\nsecurity_mode = \"trusted_host\"\nmax_output_bytes = 1024\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pol, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if pol.SecurityMode != ModeTrustedHost || pol.MaxOutputBytes != 1024 {
		t.Errorf("fields not decoded: %+v", pol)
	}
}

func TestResolved_IsImmutableCopy(t *testing.T) {
	p := restricted("IO", "Net")
	r, err := Resolve(p, "d")
	if err != nil {
		t.Fatal(err)
	}
	p.NetAllow[0] = "evil.example"
	p.AllowedCaps[0] = "Process"
	if r.NetAllow[0] != "api.example" || r.Effects[0] == "Process" {
		t.Errorf("Resolved must not alias the Policy's slices: %+v", r)
	}
}

// M6: restricted mode admits Process for entries with a confined schema —
// read-only git — and refuses any other entry by name.
func TestResolve_RestrictedConfinedProcess(t *testing.T) {
	p := restricted("IO", "FS", "Process")
	p.ProcessAllow = []string{"git:status", "git:diff", "git:log"}
	r, err := Resolve(p, "d")
	if err != nil {
		t.Fatalf("confined git entries must resolve in restricted mode: %v", err)
	}
	if !r.Admits("Process") || len(r.ProcessAllow) != 3 {
		t.Fatalf("%+v", r)
	}
	for _, bad := range []string{"git:push", "git:*", "git", "gh:pr:list", "sh"} {
		p := restricted("IO", "Process")
		p.ProcessAllow = []string{"git:status", bad}
		_, err := Resolve(p, "d")
		if err == nil || !strings.Contains(err.Error(), bad) || !strings.Contains(err.Error(), "trusted_host") {
			t.Errorf("%s: must be refused by name with the migration, got %v", bad, err)
		}
	}
}

// M7: AI in restricted mode — pinned provider AND an explicit AI budget.
func TestResolve_RestrictedAIRequiresProviderAndBudget(t *testing.T) {
	p := restricted("IO", "AI")
	p.AIProvider = "stub"
	if _, err := Resolve(p, "d"); err == nil || !strings.Contains(err.Error(), "budgets") || !strings.Contains(err.Error(), "AI") {
		t.Fatalf("restricted AI without [budgets] AI must be refused naming it, got %v", err)
	}
	p.Budgets = map[string]int{"AI": 25}
	r, err := Resolve(p, "d")
	if err != nil {
		t.Fatalf("restricted AI with a pinned provider and a budget must resolve: %v", err)
	}
	if !r.Admits("AI") || r.AIProvider != "stub" || r.Budgets["AI"] != 25 {
		t.Fatalf("%+v", r)
	}
	p.Budgets = map[string]int{"AI": 0}
	if _, err := Resolve(p, "d"); err != nil {
		t.Fatalf("an explicit AI = 0 is a valid (zero-call) ceiling: %v", err)
	}
	p.Budgets = map[string]int{"AI": 5}
	p.AIProvider = ""
	if _, err := Resolve(p, "d"); err == nil || !strings.Contains(err.Error(), "ai_provider") {
		t.Fatalf("AI without ai_provider: %v", err)
	}
}

// M7: fs_deny_write patterns are validated and carried.
func TestResolve_FSDenyWrite(t *testing.T) {
	p := restricted("IO", "FS")
	p.FSDenyWrite = []string{".github/**", "Makefile", "*.yml", ".pi/**"}
	r, err := Resolve(p, "d")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.DenyWrite) != 4 {
		t.Fatalf("%+v", r.DenyWrite)
	}
	for _, bad := range []string{"/etc/x", "../x", "a/../b", ""} {
		p.FSDenyWrite = []string{bad}
		if _, err := Resolve(p, "d"); err == nil || !strings.Contains(err.Error(), "fs_deny_write") {
			t.Errorf("%q: must be refused by name, got %v", bad, err)
		}
	}
	p.FSDenyWrite = []string{"x"}
	p.AllowedCaps = []string{"IO"}
	p.FSSandbox = ""
	if _, err := Resolve(p, "d"); err == nil || !strings.Contains(err.Error(), "fs_deny_write") {
		t.Fatalf("fs_deny_write without FS is a contradiction: %v", err)
	}
}

// Security audit 2026-10-01 F-A1 (Q3): confined git runs at the sandbox root
// with discovery bounded there, so restricted Process without a sandbox —
// no fs_sandbox, or fs_sandbox without FS (Resolved.Root is then empty) — is
// a named policy-resolution error. trusted_host keeps its operator choice.
func TestResolve_RestrictedProcessRequiresSandbox(t *testing.T) {
	noSandbox := restricted("IO", "Process") // (FS without fs_sandbox is refused earlier, by FS)
	noSandbox.FSSandbox = ""
	noFS := restricted("IO", "Process") // fs_sandbox set, FS not admitted
	for name, p := range map[string]*Policy{"no fs_sandbox": noSandbox, "no FS": noFS} {
		p.ProcessAllow = []string{"git:log"}
		_, err := Resolve(p, "d")
		if err == nil || !strings.Contains(err.Error(), "Process") || !strings.Contains(err.Error(), "fs_sandbox") {
			t.Errorf("%s: restricted Process without a sandbox must be refused naming Process and fs_sandbox, got %v", name, err)
		}
	}

	ok := restricted("IO", "FS", "Process")
	ok.ProcessAllow = []string{"git:status", "git:diff", "git:log"}
	r, err := Resolve(ok, "d")
	if err != nil || r.Root != "/tmp/sb" {
		t.Fatalf("restricted Process with FS + fs_sandbox must resolve with the root: %v %+v", err, r)
	}

	trusted := restricted("IO", "Process")
	trusted.FSSandbox = ""
	trusted.SecurityMode = ModeTrustedHost
	trusted.ProcessAllow = []string{"git:status"}
	if _, err := Resolve(trusted, "d"); err != nil {
		t.Fatalf("trusted_host Process without fs_sandbox stays the operator's choice: %v", err)
	}
}

// #1558 — net_allow entries are validated at load time; restricted mode
// admits loopback only as a port-qualified literal (that port, nothing else)
// and refuses bare loopback, loopback names and private/link-local literals.
func TestResolve_NetAllowEntries(t *testing.T) {
	withAllow := func(mode string, entries ...string) *Policy {
		p := restricted("IO", "Net")
		p.SecurityMode = mode
		p.NetAllow = entries
		p.NetAllowHTTP = true
		return p
	}
	admitted := []struct {
		mode    string
		entries []string
	}{
		{ModeRestricted, []string{"127.0.0.1:7655"}},
		{ModeRestricted, []string{"[::1]:7655"}},
		{ModeRestricted, []string{"api.example:8443", "*.svc.example:443", "other.example"}},
		{ModeTrustedHost, []string{"127.0.0.1:7655", "localhost:7655", "[::1]:7655"}},
		{ModeTrustedHost, []string{"127.0.0.1"}}, // trusted_host's explicit all-port grant stays
	}
	for _, c := range admitted {
		if _, err := Resolve(withAllow(c.mode, c.entries...), "d"); err != nil {
			t.Errorf("%s %v must be admitted, got %v", c.mode, c.entries, err)
		}
	}
	refused := []struct {
		mode    string
		entry   string
		wantSub string
	}{
		{ModeRestricted, "127.0.0.1", "127.0.0.1:PORT"},
		{ModeRestricted, "::1", "PORT"},
		{ModeRestricted, "localhost", "PORT"},
		{ModeRestricted, "localhost:7655", "127.0.0.1:PORT"},
		{ModeRestricted, "10.0.0.5:80", "private"},
		{ModeRestricted, "169.254.169.254:80", "link-local"},
		{ModeRestricted, "192.168.1.1", "private"},
		{ModeRestricted, "127.0.0.1:0", "port"},
		{ModeTrustedHost, "127.0.0.1:99999", "port"},
		{ModeTrustedHost, "http://api.example", "net_allow"},
		{ModeTrustedHost, "", "net_allow"},
	}
	for _, c := range refused {
		_, err := Resolve(withAllow(c.mode, c.entry), "d")
		if err == nil || !strings.Contains(err.Error(), c.wantSub) {
			t.Errorf("%s net_allow %q: want an error containing %q, got %v", c.mode, c.entry, c.wantSub, err)
		}
	}
}
