package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/mission"
)

// M-MISSION-LOOP-WORKBENCH: `ailang mission <list|doctor|install>`.
//
// One place to answer "which missions exist, where do they run, and does what is on
// disk match what was reviewed?" — the question that used to need reading six
// unsynchronised surfaces plus a truth-table maintained by hand in a code comment.
//
// This registry does NOT own which MODEL runs a role. That is
// M-MODEL-REGISTRY-SINGLE-SOURCE (`ailang models role`), ratified 2026-08-27.

// missionRegistryDir is where the per-mission TOML entries live (HD-1(a)).
const missionRegistryDir = "missions"

func missionCommand(args []string) error {
	if len(args) == 0 {
		printMissionHelp()
		return nil
	}
	switch args[0] {
	case "list":
		return missionList()
	case "doctor":
		return missionDoctor(args[1:])
	case "install":
		return missionInstall(args[1:])
	case "help", "--help", "-h":
		printMissionHelp()
		return nil
	default:
		return fmt.Errorf("unknown mission subcommand %q (want: list, doctor, install)", args[0])
	}
}

func printMissionHelp() {
	fmt.Print(`ailang mission — the mission-loop registry

  ailang mission list              which missions exist, and how each is wired
  ailang mission doctor [<name>]   does what is installed match what was reviewed?
                                   exit 0 clean, 1 drift, 2 registry error
  ailang mission install <name>    render this mission's artifacts to *.staged

Source of truth: missions/<name>.toml

install RENDERS ONLY. It writes nothing the fleet reads — the driver re-sources its
env file on every fire, so writing that path in place would apply new config with
nobody reloading anything. Promotion is a separate step (apply, not yet implemented).

Model and role assignment is NOT here: see ` + "`ailang models role`" + `.
`)
}

func loadMissionRegistry() (*mission.Registry, error) {
	dir := missionRegistryDir
	if _, err := os.Stat(dir); err != nil {
		// Allow running from anywhere inside the repo.
		if wd, e := os.Getwd(); e == nil {
			for d := wd; d != "/" && d != "."; d = filepath.Dir(d) {
				cand := filepath.Join(d, missionRegistryDir)
				if _, e := os.Stat(cand); e == nil {
					dir = cand
					break
				}
			}
		}
	}
	return mission.Load(dir)
}

func missionList() error {
	reg, err := loadMissionRegistry()
	if err != nil {
		return err
	}
	rep := mission.Doctor(reg, mission.DefaultPaths())
	fmt.Printf("%-8s %-28s %-18s %-7s %s\n", "NAME", "REPO", "SCHEDULE", "PIN", "DRIVER")
	for _, r := range rep.Rows {
		pin := "yes"
		if !r.Pinned {
			pin = "NO"
		}
		driver := r.Driver
		if r.Fork {
			driver += "  (FORK)"
		}
		fmt.Printf("%-8s %-28s %-18s %-7s %s\n", r.Name, r.Repo, r.Schedule, pin, driver)
	}
	return nil
}

func missionDoctor(args []string) error {
	reg, err := loadMissionRegistry()
	if err != nil {
		// Exit 2 distinguishes "the registry itself is broken" from "the fleet has
		// drifted" — different problems with different fixes.
		fmt.Fprintf(os.Stderr, "registry error: %v\n", err)
		os.Exit(2)
	}
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		m, ok := reg.Get(args[0])
		if !ok {
			return fmt.Errorf("no mission %q in %s (have: %s)", args[0], missionRegistryDir, strings.Join(reg.Names(), ", "))
		}
		reg = &mission.Registry{Missions: []*mission.Mission{m}}
	}
	rep := mission.Doctor(reg, mission.DefaultPaths())
	for _, f := range rep.Findings {
		fmt.Println(f)
	}
	if len(rep.Findings) == 0 {
		fmt.Printf("%d mission(s): no drift\n", len(rep.Rows))
	}
	os.Exit(rep.ExitCode())
	return nil
}

func missionInstall(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ailang mission install <name>")
	}
	reg, err := loadMissionRegistry()
	if err != nil {
		return err
	}
	m, ok := reg.Get(args[0])
	if !ok {
		return fmt.Errorf("no mission %q in %s (have: %s)", args[0], missionRegistryDir, strings.Join(reg.Names(), ", "))
	}
	p := mission.DefaultPaths()
	s, err := mission.RenderStaged(m, p.EnvPath(m.Name), p.PlistPath(m))
	if err != nil {
		return err
	}
	fmt.Printf("rendered (nothing that runs was touched):\n  %s\n  %s\n\n", s.EnvStaged, s.PlistStaged)
	fmt.Printf("review with:\n  diff %s %s\n  diff %s %s\n", s.EnvTarget, s.EnvStaged, s.PlistTarget, s.PlistStaged)
	return nil
}
