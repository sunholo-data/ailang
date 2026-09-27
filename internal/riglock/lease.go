// lease.go — the rig lock as a LEASE the GPU gateway can check
// (M-RIG-GPU-ADMISSION-GATEWAY).
//
// Taking the lock mints a random token into the lock directory and exports it to
// the holder's children as AILANG_RIG_LEASE; releasing (or stealing) the lock
// removes the directory, so the token dies with it. The rig gateway admits long
// GPU work only from requests carrying the live token while the lock is held.
// That is what ends orphans structurally: an agent whose job's lock is gone
// carries a dead token and is refused on its next request, with no process
// tracking needed. tools/launchd/rig-lock.sh mints the same file the same way.
package riglock

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

// leaseFile holds the token inside the lock directory. Mode 0640 so the rig
// group (the shared directory is setgid "rig") can read it and nobody else.
const leaseFile = "token"

// mintLease writes a fresh 128-bit token into dir and returns it.
func mintLease(dir string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	tok := hex.EncodeToString(b)
	if err := os.WriteFile(filepath.Join(dir, leaseFile), []byte(tok+"\n"), 0o640); err != nil {
		return "", err
	}
	return tok, nil
}

// Lease is the gateway's view of the rig lock at one instant.
type Lease struct {
	Held   bool   // the lock directory exists
	Holder string // the holder line as written ("<pid> <ts> ..."), for refusal messages
	Token  string // "" when the holder minted none (an older holder binary)
}

// CurrentLease reads the lock directory without taking it.
func CurrentLease() Lease {
	return leaseAt(lockDir())
}

func leaseAt(dir string) Lease {
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return Lease{}
	}
	l := Lease{Held: true}
	if b, err := os.ReadFile(filepath.Join(dir, "holder")); err == nil {
		l.Holder = strings.TrimSpace(string(b))
	}
	if b, err := os.ReadFile(filepath.Join(dir, leaseFile)); err == nil {
		l.Token = strings.TrimSpace(string(b))
	}
	return l
}
