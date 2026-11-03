package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestCheckEnv(t *testing.T) {
	env, err := checkEnv([]string{"KEEP=yes", "GOMEMLIMIT=1MiB", "GOMEMLIMIT=2MiB"}, 896<<20)
	if err != nil || strings.Join(env, ",") != "KEEP=yes,GOMEMLIMIT=832MiB" {
		t.Fatalf("env=%v err=%v", env, err)
	}
	for _, cap := range []uint64{0, 1, 64 << 20} {
		if _, err := checkEnv(nil, cap); err == nil {
			t.Fatalf("cap %d must fail", cap)
		}
	}
}

func TestBoundedHelper(t *testing.T) {
	if os.Getenv("VALIDATOR_ENV_HELPER") != "1" {
		return
	}
	os.Stdout.WriteString(os.Getenv("GOMEMLIMIT"))
	time.Sleep(250 * time.Millisecond)
	os.Exit(0)
}

func TestBoundedEnvAndStats(t *testing.T) {
	env, err := checkEnv(append(os.Environ(), "VALIDATOR_ENV_HELPER=1"), 896<<20)
	if err != nil {
		t.Fatal(err)
	}
	out, stats, err := runBoundedEnv(t.TempDir(), 896<<20, 10*time.Second, fixedRSS(12<<20), env, os.Args[0], "-test.run=^TestBoundedHelper$")
	if err != nil || string(out) != "832MiB" || stats.PeakRSS != 12<<20 || stats.Wall < 250*time.Millisecond {
		t.Fatalf("out=%q stats=%+v err=%v", out, stats, err)
	}
	_, stats, err = runBoundedEnv(t.TempDir(), 128<<20, 10*time.Second, fixedRSS(600<<20), env, os.Args[0], "-test.run=^TestBoundedHelper$")
	if _, ok := asBudgetError(err); !ok || stats.PeakRSS != 600<<20 || stats.Wall <= 0 {
		t.Fatalf("stats=%+v err=%v", stats, err)
	}
}
