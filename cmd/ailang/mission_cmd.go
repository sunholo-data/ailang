package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	case "report":
		if code := runMissionReport(args[1:]); code != 0 {
			return iterationExit(code, fmt.Errorf("mission report failed (exit %d)", code))
		}
		return nil
	case "activation":
		return missionActivationCommand(args[1:])
	case "retry-review":
		return missionRetryReviewCommand(args[1:])
	case "confirm-stopped":
		return missionConfirmStoppedCommand(args[1:])
	case "iterate", "status", "resume", "cancel":
		return missionIterationCommand(args[0], args[1:])
	case "list":
		return missionList()
	case "doctor":
		return missionDoctor(args[1:])
	case "install":
		return missionInstall(args[1:])
	case "apply":
		return missionApply(args[1:])
	case "rotate-log":
		return missionRotateLog(args[1:])
	case "normalize":
		return missionNormalize(args[1:])
	case "attempt":
		return missionAttemptCommand(args[1:])
	case "role-run":
		return missionRoleRun(args[1:])
	case "quota":
		return missionQuota(args[1:])
	case "help", "--help", "-h":
		printMissionHelp()
		return nil
	default:
		return fmt.Errorf("unknown mission subcommand %q (want: list, doctor, install, apply, rotate-log, normalize, quota, role-run, attempt, iterate, status, resume, cancel, retry-review, confirm-stopped, activation, report)", args[0])
	}
}

func printMissionHelp() {
	fmt.Print(`ailang mission — registry, execution and communications

  ailang mission report --mission NAME --body-file FILE [--dry-run]
                                   post a report to the resolved bookkeeping issue
                                   see report --help for state/issue overrides
  ailang mission iterate --work-item FILE [--dry-run]
                                   one frozen work item through validated completion
  ailang mission status NAME --work-item ID [--json] [--activation OP]
                                   read existing state; never creates or migrates a DB
  ailang mission resume NAME --work-item ID
                                   continue saved input/routes after ownership expires
  ailang mission cancel NAME --work-item ID --version N
                                   fence parent/child; unknown processes retain admission
  ailang mission retry-review NAME --work-item ID --new-id ID --output FILE
      --max-tokens N --timeout-seconds N --max-cost-usd N [--activation OP]
      [--evaluator MODEL] [--authority-file JSON --base-revision COMMIT]
                                   prepare evaluator-only successor; never dispatches
                                   creates FILE, FILE.manifest.json, FILE.models.yml
                                   without authority: non-executable approval draft
  ailang mission confirm-stopped NAME --work-item ID --version N --attestation TEXT
                                   record verified process stop after cancellation/deadline
                                   does not clear generic outcome_unknown or retry work
  ailang mission activation run docs --operation OP --work-item FILE --binding FILE
                                   supervise one Docs canary and verified cleanup
  ailang mission activation inspect docs --operation OP
                                   inspect owned installation and cleanup status
  ailang mission activation recover docs --operation OP
                                   restore unchanged owned files after verified stop
  Runtime placement: ~/.config/ailang/mission-runtime.toml
    version=1; absolute state_db and workspace_root (outside source checkout).
    macOS/Linux execution only; no per-command DB override.
    AILANG_MODELS_PATH=/absolute/FILE.models.yml selects the retry bundle model
    snapshot for successor dry-run and first execution.
    --activation OP reads the exact retained binding for status/retry-review only;
    requested mission/work item must match the owned activation record.
    AILANG_MISSION_REGISTRY: optional absolute existing missions/ directory,
    for first admission from a foreign project. Resume uses the saved snapshot.
    Exit: 0 complete/read-only, 2 invalid, 3 waiting, 4 reconciliation,
          5 execution/verification failure, 130 cancelled.

  ailang mission role-run --request FILE --receipt NEW_FILE [--state-db FILE] [--dry-run]
                                   execute one explicit role; output is not acceptance
  ailang mission attempt <status|reconcile|cancel> --state-db FILE
                                   inspect or fence durable role attempts (see action --help)
  ailang mission list              which missions exist, and how each is wired
  ailang mission doctor [<name>]   does what is installed match what was reviewed?
                                   exit 0 clean, 1 drift, 2 registry error
  ailang mission install <name>    render this mission's artifacts to *.staged
  ailang mission apply <name>      promote the staged artifacts, then reload launchd
  ailang mission quota [--bucket B] [--json] [--consolidate] [--over]
    Codex: local CODEX_HOME (default ~/.codex) provider usage; stale/missing blocks routing.
    Ollama Cloud: OLLAMA_API_KEY usage gauge warns at 80%, blocks at 95% in either window.
    Optional ~/.ailang/state/ollama-quota-limits.json adds verified reset-aware pacing.
    Unknown quota blocks cloud routing; local Ollama models are unaffected.
                                   fleet-wide subscription spend per (bucket, window),
                                   against the 10%/day ration
  ailang mission rotate-log <name> [--keep N]
                                   trim the live log, archive the rest, and regenerate
                                   the COMPLETE one-line index (default keep 20)
                                   --status rotates the STATUS-stamp archive instead
  ailang mission normalize [<name>] [--apply]
                                   rewrite mission-doc headings to the ONE canonical shape
                                   (dry run by default; reports what it will not convert)
                                   --adopt  acknowledge replacing a hand-written plist
                                   --force  proceed while an iteration is running
                                   --no-reload  promote without touching launchd

Source of truth: missions/<name>.toml

install RENDERS ONLY. It writes nothing the fleet reads — the driver re-sources its
env file on every fire, so writing that path in place would apply new config with
nobody reloading anything. apply is the only verb that changes what runs, and every
wait inside it is bounded (bootout 10s, bootstrap 10s, verify 15s).

Model and role assignment is NOT here: see ` + "`ailang models role`" + `.
`)
}

func loadMissionRegistry() (*mission.Registry, error) {
	if dir := os.Getenv("AILANG_MISSION_REGISTRY"); dir != "" {
		if !filepath.IsAbs(dir) {
			return nil, fmt.Errorf("AILANG_MISSION_REGISTRY must be an absolute existing directory")
		}
		info, err := os.Stat(dir)
		if err != nil {
			return nil, fmt.Errorf("AILANG_MISSION_REGISTRY: %w", err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("AILANG_MISSION_REGISTRY must name a directory")
		}
		reg, err := mission.Load(dir)
		if err != nil {
			return nil, fmt.Errorf("AILANG_MISSION_REGISTRY: %w", err)
		}
		return reg, nil
	}
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
	rep := mission.DoctorWith(reg, mission.DefaultPaths(), mission.NewLaunchCtl())
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
	s, err := mission.RenderStagedFrom(m, p.EnvPath(m.Name), p.PlistPath(m), p.ReviewedEnvPath(m.Name))
	if err != nil {
		return err
	}
	fmt.Printf("passthrough source: %s\n", s.Source)
	fmt.Printf("rendered (nothing that runs was touched):\n  %s\n  %s\n\n", s.EnvStaged, s.PlistStaged)
	fmt.Printf("review with:\n  diff %s %s\n  diff %s %s\n", s.EnvTarget, s.EnvStaged, s.PlistTarget, s.PlistStaged)
	return nil
}

func missionApply(args []string) error {
	var name string
	opts := mission.ApplyOpts{BackupDir: filepath.Join(os.Getenv("HOME"), ".ailang", "state", "mission-backups")}
	for _, a := range args {
		switch a {
		case "--adopt":
			opts.Adopt = true
		case "--force":
			opts.Force = true
		case "--no-reload":
			opts.SkipReload = true
		default:
			if strings.HasPrefix(a, "-") {
				return fmt.Errorf("unknown flag %q (want: --adopt, --force, --no-reload)", a)
			}
			name = a
		}
	}
	if name == "" {
		return fmt.Errorf("usage: ailang mission apply <name> [--adopt] [--force] [--no-reload]")
	}
	reg, err := loadMissionRegistry()
	if err != nil {
		return err
	}
	m, ok := reg.Get(name)
	if !ok {
		return fmt.Errorf("no mission %q in %s (have: %s)", name, missionRegistryDir, strings.Join(reg.Names(), ", "))
	}
	res, err := mission.Apply(m, mission.DefaultPaths(), mission.NewLaunchCtl(), opts)
	if res != nil {
		for _, b := range res.Backups {
			fmt.Printf("backed up: %s\n", b)
		}
		for _, p := range res.Promoted {
			fmt.Printf("promoted:  %s\n", p)
		}
		for _, n := range res.Notes {
			fmt.Printf("note:      %s\n", n)
		}
		if res.Reloaded {
			fmt.Printf("reloaded:  %s\n", m.Label())
		}
	}
	return err
}

func missionRotateLog(args []string) error {
	keep := 20
	var name, stream string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--status":
			stream = "status"
		case "--keep":
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
				return fmt.Errorf("unknown flag %q (want: --keep N, --status)", args[i])
			}
			name = args[i]
		}
	}
	if name == "" {
		return fmt.Errorf("usage: ailang mission rotate-log <name> [--keep N] [--status]")
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
		if root, rerr := repoRootFor(missionRegistryDir); rerr == nil {
			logDir = root
		}
	}
	logPath := filepath.Join(logDir, "design_docs", m.Name+"-mission-log.md")
	if stream == "status" {
		// The STATUS-stamp archive is append-only and unbounded exactly like the log —
		// v1's reached 1.69 MB / ~423k tokens across 295 entries — so it rotates the same
		// way, into its own archive and its own index.
		logPath = filepath.Join(logDir, "design_docs", m.Name+"-mission-status-archive.md")
	}
	res, err := mission.RotateLog(logPath, keep)
	if err != nil {
		return err
	}
	fmt.Printf("live log   %s\n  %d -> %d bytes (~%dk -> ~%dk tokens), %d of %d entries kept\n",
		res.LogPath, res.LogBefore, res.LogAfter, res.LogBefore/4000, res.LogAfter/4000, res.Kept, res.Total)
	fmt.Printf("archive    %s\n  %d entries rotated out (full bodies retained)\n", res.ArchivePath, res.Archived)
	fmt.Printf("INDEX      %s\n  %d iterations, complete history — grep this before picking work\n", res.IndexPath, res.IndexEntries)
	return nil
}

// sharedRepoSlug is the repo whose checkout holds the registry and the canonical
// design_docs for every mission that lives in it.
const sharedRepoSlug = "sunholo-data/ailang"

// repoRootFor returns the directory containing the registry dir, i.e. the checkout root.
func repoRootFor(regDir string) (string, error) {
	if _, err := os.Stat(regDir); err == nil {
		return filepath.Abs(filepath.Join(regDir, ".."))
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for d := wd; d != "/" && d != "."; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, missionRegistryDir)); err == nil {
			return d, nil
		}
	}
	return "", fmt.Errorf("no %s directory found from %s upward", missionRegistryDir, wd)
}

// canonicalDocs are the mission documents whose headings are records and therefore
// normalisable. Charters are excluded: they are curated prose, not a record stream.
func canonicalDocs(dir, name string) []string {
	return []string{
		filepath.Join(dir, "design_docs", name+"-mission-log.md"),
		filepath.Join(dir, "design_docs", name+"-mission-log-archive.md"),
		filepath.Join(dir, "design_docs", name+"-mission-status-archive.md"),
		filepath.Join(dir, "design_docs", name+"-mission-status-archive-old.md"),
	}
}

func missionNormalize(args []string) error {
	apply := false
	var only string
	for _, a := range args {
		switch a {
		case "--apply":
			apply = true
		default:
			if strings.HasPrefix(a, "-") {
				return fmt.Errorf("unknown flag %q (want: --apply)", a)
			}
			only = a
		}
	}
	reg, err := loadMissionRegistry()
	if err != nil {
		return err
	}
	root, rerr := repoRootFor(missionRegistryDir)
	if rerr != nil {
		return rerr
	}
	totalRe, totalUn := 0, 0
	for _, m := range reg.Missions {
		if only != "" && m.Name != only {
			continue
		}
		dir := root
		if m.Repo != sharedRepoSlug {
			dir = m.Workdir
		}
		for _, doc := range canonicalDocs(dir, m.Name) {
			if _, serr := os.Stat(doc); serr != nil {
				continue
			}
			res, nerr := mission.Normalize(doc, apply)
			if nerr != nil {
				return nerr
			}
			if len(res.Rewrites) == 0 && len(res.Unhandled) == 0 {
				continue
			}
			fmt.Printf("%s\n", doc)
			if len(res.Rewrites) > 0 {
				fmt.Printf("  %d heading(s) %s\n", len(res.Rewrites), map[bool]string{true: "REWRITTEN", false: "would be rewritten"}[apply])
				for i, r := range res.Rewrites {
					if i == 3 {
						fmt.Printf("    ... and %d more\n", len(res.Rewrites)-3)
						break
					}
					fmt.Printf("    - %s\n    + %s\n", trunc(r.From), trunc(r.To))
				}
			}
			for _, u := range res.Unhandled {
				fmt.Printf("  UNCONVERTIBLE line %d (reported, never guessed at):\n    %s\n", u.Line, trunc(u.From))
			}
			totalRe += len(res.Rewrites)
			totalUn += len(res.Unhandled)
		}
	}
	verb := "would be rewritten"
	if apply {
		verb = "rewritten"
	}
	fmt.Printf("\n%d heading(s) %s, %d unconvertible\n", totalRe, verb, totalUn)
	if !apply && totalRe > 0 {
		fmt.Println("re-run with --apply to write them")
	}
	return nil
}

func trunc(s string) string {
	if r := []rune(s); len(r) > 96 {
		return string(r[:93]) + "..."
	}
	return s
}
