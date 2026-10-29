# M-HARNESS-PHASE3A-SPIKE: skill-resolution measurement

**Design doc**: [M-HARNESS-MISSION-LOOP](m-harness-mission-loop.md), Phase 3a and Verification Log.
**Planner-Lane**: codex:gpt-6.1-sol
**Authorization**: Mark's 2026-09-26 directive, `design_docs/fleet-mission.md:263`.
**Scope**: measurement only; Phase 3b's mechanism choice remains Mark's decision.
**Duration**: 0.5 day (~4 hours including a bounded controller run); risk: instrumentation/auth.
**Build target**: ONE Bash 3.2 script, `design_docs/fleet-mission-evidence/iteration34/skill_resolution_spike.sh`.
The executor builds evidence tooling; the controller runs it outside the executor sandbox and files results.
No driver, installed skill, Go, credentials, configuration, or release-channel changes ship.

## Starting evidence and limits

- Controller-verified versions (2026-10-13): claude 2.1.296, codex-cli 0.162.0, pi 0.85.1.
- Controller-verified: AILANG user Claude skills symlink into the main checkout; driver prompt's
  absolute SKILL.md read is already pin-rooted. Measure discovery and relative resources separately.
- Controller observation, n=1: same-name Claude collision chose `~/.claude/skills/mission-control`.
  This is a hypothesis, not the spike result. V12's historical relative-driver-path claim is stale.
- Planner read-only checks: `~/.codex/skills` exists (tailscale-preview); `~/.agents/skills` exists;
  `~/.pi/agent/skills` is absent. The driver comments contain both old pi failure and later trust-based success.
- Planner read all four requested CLI helps locally; no inference performed. Sandbox permission warnings
  from help are not skill-resolution observations. Retain numbered help again at execution time.
- Prior sprint S1 estimated 910 LOC/3.5 days; recent commits are harness fixes, not comparable experiments.
  Use a bounded 340-line evidence-script budget + 20 lines of findings, not product-code velocity.
- Registry reuse: M1–M4 `none`: CLI/filesystem measurement and documentation, no package-like capability;
  registry search is inapplicable. No `.ail`, examples, changelog, or Go test changes are needed.

## Instrument contract (shared by M1–M3)

1. Script takes `--pin <absolute-this-worktree> --v1 <absolute-main-checkout>`; no git writes or messages.
   Record canonical cwd paths, CLI executable paths/versions, env variable NAMES, and read-only HEAD SHAs.
   Require v1 != pin; bank read-only `git diff --stat`, `git status --short`, and distinct P/V marker hashes.
2. Generate `runid` from UTC timestamp + `openssl rand -hex 12`; tokens from separate `openssl rand -hex 24`
   calls for EACH location and resource. Refuse collisions. Names `zz-fleet-spike-<runid>-<premise>`
   are never mission skill names. Use the same name at conflicting locations, with different tokens.
3. Each marker directory contains `SKILL.md` (valid name/description frontmatter, no token in description;
   description permits invocation only on an explicit request for its unique NAME)
   and `resources/probe.md`. SKILL.md body contains its secret skill token and instructs reading
   `resources/probe.md` relative to THAT SKILL.md directory; resource contains an independent secret token.
   Instruction: output ONLY JSON `{"skill_token":"...","resource_token":"..."}`. No operational actions.
4. Fresh CLI process/session per call. Discovery prompt D: “For this measurement as mission controller,
   use the available skill NAME. Follow its relative resource instruction and return only its JSON.
   If unavailable return ABSENT. Use only the skill catalog/tool and its declared files; do not search
   directories, enumerate other skills, or infer paths.” Substitute NAME, never a token or directory.
   Explicit prompt E: “Read ABSOLUTE_SKILL_MD, follow its relative resource instruction, return only its JSON.”
   In tables, `E(location)` expands to that marker directory's absolute `SKILL.md` path.
   Negative prompt N = D with a fresh name planted NOWHERE. Inventory all planted tokens privately in evidence.
5. Positive control per CLI: explicit path must return BOTH exact tokens. Negative control: final answer
   must be ABSENT and contain NONE of this run's planted tokens. Run controls before subjects and after
   the last subject with that CLI/environment; cwd/root changes require their own controls if inference works.
   Before discovery, explicitly probe each planted location in that cwd (pi prompt E for readability checks),
   so an unreadable losing location cannot masquerade as a resolution preference.
   A failed/missing control makes affected subjects INSTRUMENT_FAILURE; never “no discovery”.
6. Extract ONLY final assistant content (Claude terminal result event, Codex last-message file, pi terminal assistant
   event in JSON); never score token matches in echoed prompts, tool output, startup catalogs, or stderr.
   Require both tokens match ONE location. Mixed origins = RESOURCE_MISMATCH; zero/ambiguous/wrong tokens
   = INCONCLUSIVE unless controlled ABSENT. Bank raw tool traces to distinguish native catalog use from
   forbidden manual search. Search/path leakage invalidates discovery even when the token is correct.
7. Per call: `VERDICT runid=<id> arm=<id> status=<...> origin=<...> controls=<...> rc=<n> evidence=<relative-path>`.
   Print a table of every planned arm, including controls and NOT_RUN/INSTRUMENT_FAILURE subjects.
   Store stdout, stderr, exact argv/cwd, final answer, exit status, start/deadline/end, and verdict separately
   in `design_docs/fleet-mission-evidence/iteration34/runs/<runid>/`; preserve raw streams unchanged.
8. Deadline runner uses Bash indexed arrays (no associative arrays/mapfile/timeout dependency),
   `date +%s`, and a 120-second deadline (hard maximum 180). Every call has `< /dev/null`.
   Start each call in its own verified process group (`set -m`, check PGID); watchdog polls each second,
   TERM group at deadline-5, KILL at deadline, waits/reaps group and watchdog before cleanup.
   Refuse unsafe PGID; signal only the recorded group's processes. Timeout never yields a discovery verdict.
9. Repeat each Claude collision, World, and v1 subject in three fresh sessions; rotate tokens between
   repetitions and re-run explicit controls after rotation. Report counts, not a universal precedence rule.
   Other subjects run once; no retries with different models, billing modes, or credentials.

## Mutation safety and roots

- Register `trap cleanup EXIT` and INT/TERM handlers before planting; keep an ownership ledger of only
  successful NEW directory creations. `mkdir` must fail if any target exists (including dangling symlinks).
  Never modify/move/replace existing files or symlinks; never create a missing non-unique user parent.
- Snapshot each pre-existing parent that will receive a child (including main-checkout and scratch parents)
  with `(cd "$parent" && LC_ALL=C ls -1A)`, bank listing, `shasum` and `wc -l` before planting and after
  removal. `-A` includes hidden entries. Also bank `ls -ld`/readlink observations of existing skill symlinks.
  Snapshot real user skill parents even if an arm skips planting. Missing parents remain missing.
- Every write outside pin must live inside a NEW uniquely named owned directory removed by EXIT trap.
  Real-user markers only go into existing `~/.claude/skills`, `~/.agents/skills`, `~/.codex/skills` parents;
  v1 marker only into its existing `.claude/skills`. Do not create absent `~/.pi/agent/skills`.
  Cleanup removes only ledger-owned children after processes stop; a mismatch is SAFETY_FAILURE, nonzero exit.
  Concurrent parent changes are inconclusive safety evidence; never “repair” other people's entries.
- Put synthetic config/runtime roots and caches under the run's worktree scratch area; create World W as
  a unique `/private/tmp/zz-fleet-spike-<runid>-world` directory outside ALL checkout ancestry, with NO
  `.claude/skills` in W or its ancestors. Audit its pre-existing parent; trap removes W and markers.
  Extra project XP is a synthetic checkout-shaped directory under run scratch with `.claude/skills`.
- Prefer synthetic `CLAUDE_CONFIG_DIR=CS`, `CODEX_HOME=KS`, `PI_CODING_AGENT_DIR=PS`; keep actual HOME
  unchanged for the keychain identity and real-user-path tests. Do NOT repurpose shell HOME/CODEX_HOME
  variables; use task-specific variables and pass overrides only as child `env NAME=...` arguments.
- Real Claude config CU=`~/.claude` is needed to measure actual user/project collision and baseline subscription auth;
  real Codex config KU (effective controller CODEX_HOME, or `~/.codex`) is needed for authenticated baseline.
  Runtime config changes are prohibited, including auth refresh writes. Use `/usr/bin/sandbox-exec -f PROFILE`
  around each child: allow reads/network/processes, deny filesystem writes except run scratch and owned
  unique marker/scratch dirs, plus `/dev/null`; keep git metadata and existing skill files read-only.
  Set child TMPDIR to run scratch; disallow writes elsewhere even from hooks/child tools. Verify the fence
  with harmless allowed-write/denied-write fixtures inside run scratch before inference; never test against
  an existing user file. If the CLI requires blocked state/auth writes, record INSTRUMENT_FAILURE and the
  denied operation, never relax the fence or interpret sandbox failure as discovery evidence.
- Claude `--setting-sources ''` removes settings sources in controlled arms; measure it against default
  sources instead of assuming it removes skills. Use read/skill tools only. Codex uses read-only sandbox,
  no approvals, no daemon, ephemeral sessions. Pi uses synthetic PS, read-only tools, no extensions/context.
  Record these controls as experiment conditions; do not assert equivalence to installed extensions.

## Exact invocation templates and flag provenance

Define Bash function `claude-sub() { env -u ANTHROPIC_API_KEY -u ANTHROPIC_AUTH_TOKEN claude "$@"; }`.
Export/invoke it in the fenced Bash child; EVERY Claude inference AND execution-time help uses this wrapper.
Never bare Claude inference, `--bare`, login, setup-token, token-printing, credential copies, or auth.json writes.
No model substitutions: unavailable prescribed model = instrument failure. Prompts are quoted argv arguments.
Below commands are the inner fenced/deadline commands; each table row specifies cwd, root and extra args.

```bash
# C(root, cwd, prompt, extras): child env CLAUDE_CONFIG_DIR="$root"; cd "$cwd"
claude-sub -p --model claude-haiku-5-5 --no-session-persistence --output-format stream-json --verbose \
  --tools 'Read,Skill' --allowedTools 'Read,Skill' --permission-mode dontAsk \
  --permission-prompts none --debug-file "$armdir/debug.log" "${extras[@]}" "$prompt" < /dev/null
# C-auth uses the same C argv but REPLACES --tools/--allowedTools with --tools '' (no tools).
# K(root, cwd, prompt): child env CODEX_HOME="$root"
codex --no-daemon --ask-for-approval never exec --model gpt-6.1-sol --sandbox read-only \
  --ephemeral --skip-git-repo-check --json --cd "$cwd" \
  --output-last-message "$armdir/final.txt" "$prompt" < /dev/null
# I(cwd, prompt, extras): child env PI_CODING_AGENT_DIR="$PS"; cd "$cwd"
pi --print --model ollama/kimi-k3:cloud --mode json --no-session --tools read,grep,find,ls \
  --no-extensions --no-context-files --approve "${extras[@]}" "$prompt" < /dev/null
```

Help citations are numbered stdout lines from planner reads on 2026-10-13 (warnings excluded):

| Source | Flags relied upon, quoted help line(s) |
|---|---|
| `claude --help` | L10–11 `--add-dir`: “Additional directories to allow tool access to”; L23–25 `--allowedTools`; L75–76 `--debug-file`: “Write debug logs to a specific file path”; L134–137 `--model`; L142–149 `--no-session-persistence`, `--output-format`; L150–161 `--permission-mode` includes dontAsk, `--permission-prompts none` denies prompts; L170–174 `-p`: print and exit, trust dialog skipped; L225–226 `--setting-sources`: user, project, local; L254–260 `--tools`: built-in tool list, `--verbose`: override verbose mode |
| `codex exec --help` | L40–41 `--model`; L50–51 profile uses `$CODEX_HOME`; L53–56 `--sandbox read-only`; L69–70 `--cd`: working root; L87–94 `--skip-git-repo-check`, `--ephemeral`, `--ignore-user-config` says auth still uses CODEX_HOME; L108–112 `--json`, `--output-last-message` |
| `codex --help` | L113–119 `--ask-for-approval` includes never; L130–131 `--no-daemon`: run without shared server |
| `pi --help` | L18 `--model` accepts provider/id; L22–23 `--mode`, `--print`; L30 `--no-session`; L36–37 `--tools`; L42–44 `--no-extensions`, `--skill` file/directory, `--no-skills` disables discovery AND loading; L50 `--no-context-files`; L55 `--approve`; L169 `PI_CODING_AGENT_DIR` config root |

`CLAUDE_CONFIG_DIR` is a premise under test, not a help-established guarantee. No Codex skills-config
flag appears in either help: do not invent one. `--plugin-dir` is available (Claude L162–165) but outside
this minimal spike. `--add-dir` promises access, not skill loading. Fence `-f` is from sandbox-exec usage.

### M1: P1 — Claude resolution and auth (~160 LOC, 70 minutes)

Dependencies: none. Own shared runner/cleanup and C arms in the single evidence script.
Locations: P=pin `.claude/skills/NAME`, U=real `~/.claude/skills/NAME`,
V=v1 `.claude/skills/NAME`, X=XP extra project's `.claude/skills/NAME`, S=CS `/skills/NAME`.
Plant/remove only OUR named directory to select each arm's locations; tokens P/U/V/X/S differ.

| Arm | Planted locations; exact template invocation | Discriminator / question answered |
|---|---|---|
| C+ / C- | P,U; `C(CU,pin,E(P),[])` / `C(CU,pin,N,[])` | Positive pin pair / ABSENT, no planted tokens |
| C-project | P only; `C(CU,pin,D,[])` | P pair or controlled ABSENT: project discovery |
| C-user | U only; `C(CU,W,D,[])` | U pair or ABSENT: user-only discovery; World control |
| C-collision | P,U; `C(CU,pin,D,[])` | P vs U; three sessions, tests controller hypothesis |
| C-world | U,X; `C(CU,W,D,[--add-dir "$XP"])` | U vs X vs ABSENT: World with no project skills |
| C-add-only | X only; `C(CU,W,D,[--add-dir "$XP"])` | X or ABSENT separates add-dir reach from collision |
| C-v1 | P,U,V; `C(CU,v1,D,[])` | V vs U vs P; three sessions, live cwd differs from pin |
| C-settings | P,U; `C(CU,pin,D,[--setting-sources ''])` | Compare default collision; never infer isolation from flag |
| C-auth-base | none; `C-auth(CU,W,'Return only 1.',[])` | One-token response + subscription/non-API status in result |
| C-auth-isolated | none; `C-auth(CS,W,'Return only 1.',[])` | Same keychain/account, fresh config; success vs NOT_LOGGED_IN |
| C-synthetic+/-/subject | P,S; `C(CS,pin,E(P),[])`, `C(CS,pin,N,[])`, `C(CS,pin,D,[])` | P vs S; run only if isolated auth works; otherwise NOT_RUN |

Auth probes request exactly one output token (`1`), tools disabled, no marker task. Help exposes no output-token
cap: bank usage; extra output is INVALID_PROBE, not a silently accepted longer probe. Never provision auth.
Baseline failure prevents attributing isolated failure to config; explicit not-logged-in with a working baseline
is a legitimate auth finding. Auth arms are not discovery subjects and do not need marker responses.

- [ ] C controls pass; all C subjects have arm/runid verdicts, resource origins and raw evidence.
- [ ] World/v1 and three-session collision counts reported; isolated auth result distinguished from CLI/fence errors.
- [ ] Every external marker removed; parent listings/hash/counts identical, no credential writes.

### M2: P2 — Codex discovery versus by-path reads (~100 LOC, 45 minutes)

Dependencies: M1's runner, not a Claude discovery success. L=pin `.claude/skills/NAME`,
A=pin `.agents/skills/NAME`, B=real `~/.agents/skills/NAME`, H=real `~/.codex/skills/NAME`;
synthetic home KS is empty of credentials; T=KS `/skills/NAME`. All location tokens differ.

| Arm | Planted locations; exact invocation | Discriminator |
|---|---|---|
| K+ / K- | A; `K(KU,pin,E(A))` / `K(KU,pin,N)` | Explicit pair / ABSENT, no planted tokens |
| K-claude | L only; `K(KU,pin,D)` | L or controlled ABSENT |
| K-project | A only; `K(KU,pin,D)` | A or controlled ABSENT |
| K-agents-user | B only; `K(KU,W,D)` | B or controlled ABSENT |
| K-legacy-user | H only; `K(KU,W,D)` | H or controlled ABSENT |
| K-collision | A,B,H; `K(KU,pin,D)` | A/B/H token pair identifies catalog winner |
| K-by-path | A,B,H; `K(KU,pin,E(A))` | A despite collision, pin-rooted relative resource |
| K-home+/-/subject | A,T; `K(KS,pin,E(A))`, `K(KS,pin,N)`, `K(KS,pin,D)` | Auth unavailable = finding, subjects INSTRUMENT_FAILURE; otherwise A/T |

Bank catalog provenance/tool calls. Explicit success with controlled discovery ABSENT supports “by-path only
under these conditions”; auth or tool failure cannot answer P2. Do not copy/create auth.json or run login.

- [ ] Every K arm/control emits verdict; native catalog discovery separated from prompt-directed file reads.
- [ ] Synthetic CODEX_HOME auth failure, if present, recorded explicitly; B/H parents restored identically.

### M3: P3 — pi explicit pin path for controller (~80 LOC, 35 minutes)

Dependencies: M1's runner. A=pin `.agents/skills/NAME`, B=real `~/.agents/skills/NAME`,
T=PS `/skills/NAME`; synthetic PS avoids writing absent real pi parents. Distinct A/B/T tokens.

| Arm | Planted locations; exact invocation | Discriminator |
|---|---|---|
| I+ / I- | A,B,T; `I(pin,D,[--skill "$A/SKILL.md"])` / `I(pin,N,[--skill "$A/SKILL.md"])` | Explicit A pair / ABSENT, no tokens |
| I-project | A only; `I(pin,D,[])` | A or controlled ABSENT: remeasure driver's .agents claim |
| I-collision | A,B,T; `I(pin,D,[])` | A/B/T or ABSENT, actual automatic discovery |
| I-explicit | A,B,T; `I(pin,D,[--skill "$A/SKILL.md"])` | A pair proves absolute pin file route for controller |
| I-world | A,B,T; `I(W,D,[--skill "$A/SKILL.md"])` | A pair + relative resource despite foreign cwd |
| I-no-skills | A,B,T; `I(pin,D,[--no-skills --skill "$A/SKILL.md"])` | ABSENT vs A; measure interaction, help promises all loading disabled |

Repeat I+ and I- after subjects. Do not assume `--no-skills` preserves explicit paths; I-explicit uses it
WITHOUT that flag. Findings apply to stock controlled CLI, not the old installed trust-extension condition.

- [ ] Explicit absolute path returns pin skill AND pin resource token in pin/World cwd; all I controls fire.
- [ ] Automatic discovery and no-skills interaction reported separately; real pi missing parent stays missing.

### M4: Verification wording and parked Phase 3b decision (~20 LOC, 30 minutes)

Dependencies: M1–M3. Write `runs/<runid>/verification-log-rows.md` and `phase3b-decision.md` wording only.
Controller replaces the design doc's `P1–P3 | PENDING` row with P1, P2, P3 rows citing runid, versions,
commands/conditions, controls, winner counts, relative-resource origin, auth findings, and raw evidence links.
Add World/v1 consequences. Name providers whose evidence cannot establish a per-fire pin source as
“unpinned for skills under tested conditions”; do not claim goal 4 or mark Phase 3b complete.

Parked wording: “Phase 3b mechanism choice — FOR MARK. Phase 3a run <runid> measured <per-provider
findings>; World <finding>; v1 <finding>. Auth/isolation limitations: <findings>. Mark must choose the
per-provider skill source and whether v1's cwd moves before release-channel implementation. No mechanism
has been selected or activated by this spike.” Fill with observations; controller files the decision.

- [ ] Wording replaces PENDING with measured answers or explicit unresolved instrumentation, never invented results.
- [ ] Phase 3b remains parked for Mark; only evidence script/artifacts and controller-filed design wording change.

## Validation and completion

Executor: `/bin/bash -n` and available `shellcheck` on the ONE script; static review of trap/ledger, argv,
model guards, negative/positive controls and token extraction. No model calls inside executor sandbox.
Controller: run with `/bin/bash .../skill_resolution_spike.sh --pin "$PIN" --v1 "$V1"` outside sandbox.
Require every arm's VERDICT, every control attempted, every applicable positive/negative control passing,
raw per-call evidence, and SHA-identical post-run listings of EVERY touched pre-existing parent.
Auth-unavailable/fence/timeout discovery subjects remain INSTRUMENT_FAILURE; unresolved P1–P3 leave milestones
incomplete. Expected isolated-auth failure is a valid auth answer; optional synthetic subjects may be NOT_RUN
without invalidating controlled baseline resolution. Nonzero exit for failed controls/safety/unresolved premises.
Controller audits evidence and files M4 wording; all four JSON `passes` start null. No full Go suite required.
