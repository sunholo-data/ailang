package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/mission"
)

func missionRotateLog(args []string) error {
	keep := 20
	var indexOnly, keepProvided bool
	var name, stream string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--status":
			return fmt.Errorf("--status was retired (D-FLEET-9, ruled 2026-10-01): it rotated the STATUS archive and was never a report. Use: ailang mission rotate-log <name> --stream status")
		case "--stream":
			if i+1 >= len(args) {
				return fmt.Errorf("--stream needs a value (log|status)")
			}
			stream = args[i+1]
			if stream != "log" && stream != "status" {
				return fmt.Errorf("--stream %q unknown (want: log|status)", stream)
			}
			i++
		case "--index-only":
			indexOnly = true
		case "--keep":
			keepProvided = true
			if i+1 >= len(args) {
				return fmt.Errorf("--keep needs a number")
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil {
				return fmt.Errorf("--keep %q is not a number", args[i+1])
			}
			keep = n
			i++
		default:
			if strings.HasPrefix(args[i], "-") {
				return fmt.Errorf("unknown flag %q (want: [--keep N | --index-only], --stream log|status)", args[i])
			}
			name = args[i]
		}
	}
	if indexOnly && keepProvided {
		return fmt.Errorf("usage: ailang mission rotate-log <name> [--keep N | --index-only] [--stream log|status]: --keep cannot be combined with --index-only")
	}
	if name == "" {
		return fmt.Errorf("usage: ailang mission rotate-log <name> [--keep N | --index-only] [--stream log|status] (default keep 20; --index-only writes only the index)")
	}
	reg, err := loadMissionRegistry()
	if err != nil {
		return err
	}
	m, ok := reg.Get(name)
	if !ok {
		return fmt.Errorf("no mission %q (have: %s)", name, strings.Join(reg.Names(), ", "))
	}
	// ROTATE WHERE THE FILE IS CANONICAL, NOT WHERE THE MISSION WORKS.
	//
	// docs and motoko work in CLONES of sunholo-data/ailang — ailang-docs and
	// ailang-motoko — which are hundreds of commits behind and are never pushed from
	// (they re-exec from the driver pin, so their working trees are irrelevant). Writing
	// a rotated log there would be thrown away by the next fetch, silently.
	//
	// Caught by doing exactly that to motoko's clone. The canonical copy for any mission
	// on the shared repo is the checkout this registry lives in.
	logDir := m.Workdir
	if m.Repo == sharedRepoSlug {
		root := m.Root()
		if root == "" {
			return fmt.Errorf("mission %q: shared-repo mission loaded without registry origin (Path %q) — refusing to guess a target root", m.Name, m.Path)
		}
		logDir = root
	}
	logPath := filepath.Join(logDir, "design_docs", m.Name+"-mission-log.md")
	if stream == "status" {
		// The STATUS-stamp archive is append-only and unbounded exactly like the log —
		// v1's reached 1.69 MB / ~423k tokens across 295 entries — so it rotates the same
		// way, into its own archive and its own index.
		logPath = filepath.Join(logDir, "design_docs", m.Name+"-mission-status-archive.md")
	}
	var res *mission.RotateResult
	if indexOnly {
		res, err = mission.RegenerateIndex(logPath)
	} else {
		res, err = mission.RotateLog(logPath, keep)
	}
	if err != nil {
		return err
	}
	if !indexOnly {
		fmt.Printf("live log   %s\n  %d -> %d bytes (~%dk -> ~%dk tokens), %d of %d entries kept\n",
			res.LogPath, res.LogBefore, res.LogAfter, res.LogBefore/4000, res.LogAfter/4000, res.Kept, res.Total)
		fmt.Printf("archive    %s\n  %d entries rotated out (full bodies retained)\n", res.ArchivePath, res.Archived)
		if res.Archived == 0 {
			fmt.Println("live log was not modified (no entries archived)")
		}
	}
	fmt.Printf("INDEX      %s\n  %d iterations, complete history — grep this before picking work\n", res.IndexPath, res.IndexEntries)
	if len(res.OrphanNumbers) > 0 {
		nums := make([]string, len(res.OrphanNumbers))
		for i, n := range res.OrphanNumbers {
			nums[i] = strconv.Itoa(n)
		}
		fmt.Printf("orphan rows preserved: %s\n", strings.Join(nums, ", "))
	}
	if res.IgnoredHeadings > 0 {
		fmt.Printf("non-entry headings ignored: %d\n", res.IgnoredHeadings)
	}
	return nil
}
