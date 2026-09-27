package riglock

import (
	"os"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
)

// Acquire mints a token, exports it, and CurrentLease reads the same one;
// release revokes it (directory gone, env back to "none").
func TestLease_MintedOnAcquireRevokedOnRelease(t *testing.T) {
	isolate(t)
	t.Setenv(config.EnvRigLease, config.RigLeaseNone)

	if l := CurrentLease(); l.Held {
		t.Fatalf("lease held before acquire: %+v", l)
	}
	ok, release, err := Acquire(NoWait)
	if err != nil || !ok {
		t.Fatalf("Acquire: ok=%v err=%v", ok, err)
	}
	l := CurrentLease()
	if !l.Held || len(l.Token) != 32 {
		t.Fatalf("CurrentLease after acquire = %+v, want held with a 32-hex token", l)
	}
	if got := os.Getenv(config.EnvRigLease); got != l.Token {
		t.Fatalf("%s = %q, want the minted token %q", config.EnvRigLease, got, l.Token)
	}

	release()
	if l2 := CurrentLease(); l2.Held || l2.Token != "" {
		t.Fatalf("lease survived release: %+v", l2)
	}
	if got := os.Getenv(config.EnvRigLease); got != config.RigLeaseNone {
		t.Fatalf("%s = %q after release, want %q", config.EnvRigLease, got, config.RigLeaseNone)
	}
}

// Two acquisitions never share a token, so a job's orphan cannot ride the next
// job's lease.
func TestLease_FreshTokenPerAcquire(t *testing.T) {
	isolate(t)
	ok, release, _ := Acquire(NoWait)
	if !ok {
		t.Fatal("first acquire failed")
	}
	first := CurrentLease().Token
	release()
	t.Setenv(EnvHeld, "")
	ok, release, _ = Acquire(NoWait)
	if !ok {
		t.Fatal("second acquire failed")
	}
	defer release()
	if second := CurrentLease().Token; second == "" || second == first {
		t.Fatalf("tokens %q then %q: want two distinct non-empty tokens", first, second)
	}
}
