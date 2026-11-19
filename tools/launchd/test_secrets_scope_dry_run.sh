#!/bin/bash
# shellcheck shell=bash
# Real dry-run block, isolated synthetic HOME, stub probes; never run the driver.
set -eu
ROOT=$(cd "$(dirname "$0")/../.." && pwd -P)
SCOPE_TMP=$(mktemp -d "${TMPDIR:-/tmp}/scope-dry-run.XXXXXX")
trap 'rm -rf "$SCOPE_TMP"' EXIT
export TMPDIR="$SCOPE_TMP"
python3 - "$ROOT" "${1-}" "${2-}" <<'PY'
import difflib
import hashlib
import json
import os
import pathlib
import shutil
import subprocess
import sys
import tempfile

ROOT = pathlib.Path(sys.argv[1])
MODE, SELECT = sys.argv[2:]
DRIVER = 'tools/launchd/mission-control.sh'
FILES = [DRIVER, 'tools/launchd/lib/secrets-scope.sh', 'tools/launchd/lib/codex-env-args.sh']
ARMS = ['dry-run-preview', 'dry-run-failure', 'dry-run-parent-env',
        'dry-run-codex-forward', 'dry-run-no-pin', 'dry-run-codex-forward-rc',
        'dry-run-codex-forward-missing', 'dry-run-probe-stdin']
KEYS = ['ANTHROPIC_API_KEY', 'CLAUDE_CODE_OAUTH_TOKEN', 'GOOGLE_API_KEY',
        'GEMINI_AGENTS_API_KEY', 'OPENAI_API_KEY', 'OPENROUTER_API_KEY',
        'OLLAMA_API_KEY', 'FUTURE_SCOPE_KEY']
ALLOWED = {'CLAUDE_CODE_OAUTH_TOKEN', 'OPENAI_API_KEY', 'OPENROUTER_API_KEY', 'OLLAMA_API_KEY'}
START = '# --- SECRETS SCOPE DRY RUN START ---'
END = '# --- SECRETS SCOPE DRY RUN END ---'


def require(condition, message):
    if not condition:
        raise AssertionError(message)


def extract(text, start, end):
    require(text.count(start) == 1 and text.count(end) == 1, 'unique production block extraction')
    body = text.split(start, 1)[1].split(end, 1)[0]
    require(body.strip(), 'nonempty production block extraction')
    return body


def fixture(home, root):
    for part in ['.config/ailang', '.codex', 'zdot', 'tmp', '.ailang/state']:
        (home/part).mkdir(parents=True, exist_ok=True)
    (home/'.config/ailang/secrets.env').write_text(''.join(
        'export '+k+'=SYNTHETIC_DRY_RUN_'+k+'_END\n' for k in KEYS))
    env = dict(HOME=str(home), ZDOTDIR=str(home/'zdot'), CODEX_HOME=str(home/'.codex'),
               PATH='/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin', BASH_ENV='', ENV='',
               TMPDIR=str(home/'tmp'), AILANG_STATE_DIR=str(home/'.ailang/state'),
               MC_DRIVER_ROOT=str(root), LOG=str(home/'log'), PYTHON=sys.executable,
               MISSION_DRY_RUN='1', LC_ALL='C')
    env.update({k: 'SYNTHETIC_DRY_RUN_'+k+'_END' for k in KEYS})
    for name in ['MISSION_NAME', 'MISSION_REPO', 'MISSION_DOC', 'REPO', 'PIDFILE',
                 'PREFS', 'HARD_TIMEOUT', 'MISSION_DESIGNER_MODEL', 'MISSION_PLANNER_MODEL',
                 'MISSION_EXECUTOR_MODEL', 'MISSION_EVALUATOR_MODEL', 'PIN_STATUS',
                 'PIN_DRIFT', 'SKILL_SYNC_STATUS']:
        env[name] = 'synthetic'
    env['PIDFILE'] = str(home/'pidfile')
    # Only names/presence, argument match booleans, and hashes/counts leave these helpers.
    (home/'capture.py').write_text('''import json, os, pathlib, sys
keys = '''+repr(KEYS)+'''
pathlib.Path(os.environ['HOME'], 'probe').write_text(json.dumps(dict(
    present=[k for k in keys if k in os.environ],
    marker_ok=os.environ.get('AILANG_SECRETS_LANE') == sys.argv[1]+'-controller',
    root_ok=os.environ.get('MISSION_DRIVER_ROOT') == os.environ['MC_DRIVER_ROOT'],
    model_ok=sys.argv[2] == 'synthetic-model',
    stdin_null=os.fstat(0).st_rdev == os.stat('/dev/null').st_rdev and os.path.exists('/dev/null'),
    bash_env_present='BASH_ENV' in os.environ)))
''')
    (home/'env-hash.py').write_text('''import hashlib, json, os
# The shell changes its last-command variable between observations.
e = {k:v for k,v in os.environ.items() if k != '_'}
print(hashlib.sha256(json.dumps(e, sort_keys=True).encode()).hexdigest(), len(e))
''')
    return env


def run_case(root, home, pin, probe_rc=0, parent=False, missing=False, degraded=False, forward=None):
    env = fixture(home, root)
    if pin is not None:
        env['MISSION_MODEL'] = pin
    if missing:
        (home/'.config/ailang/secrets.env').unlink()
    driver = (root/DRIVER).read_text()
    preview = extract(driver, START, END)
    branch = extract(driver, 'if [ "${MISSION_DRY_RUN:-0}" = "1" ]; then',
                     '\nif type mc_skill_sync >/dev/null 2>&1; then mc_skill_sync apply; fi')
    # Include the actual if and closing fi, plus a sentinel for forbidden fallthrough.
    branch = 'if [ "${MISSION_DRY_RUN:-0}" = "1" ]; then'+branch
    selection = extract(driver, '_mc_set_controller() {', '\nselect_model() {')
    prefix = '''set -uo pipefail
log() { printf '%s\\n' "$*" | tee -a "$LOG"; }
mc_skill_sync() { log "synthetic skill-sync $1"; }
_lane_degraded='''+('"- synthetic invalid executor -> handoff"' if degraded else "''")+'''
. "$MC_DRIVER_ROOT/tools/launchd/lib/secrets-scope.sh"
. "$MC_DRIVER_ROOT/tools/launchd/lib/codex-env-args.sh"
_mc_set_controller() {'''+selection+'''
fake_probe() {
  "$PYTHON" "$HOME/capture.py" "$1" "$2" || return 99
  return '''+str(probe_rc)+'''
}
_mc_probe() { fake_probe claude "$1"; }
_mc_probe_codex() { fake_probe codex "$1"; }
_mc_probe_pi() { fake_probe pi "$1"; }
'''
    if forward == 'fail':
        prefix += "mc_codex_env_args() { MC_CODEX_ENV_ARGS=(); return 41; }\n"
    elif forward == 'missing-marker':
        prefix += "mc_codex_env_args() { MC_CODEX_ENV_ARGS=(-c 'shell_environment_policy.set.MISSION_DRIVER_ROOT=synthetic'); return 0; }\n"
    # Prove state paths are unchanged by observation, without seeding a state file.
    def state():
        paths = [home/'pidfile', home/'.ailang/state', home/'controller-body']
        return [(p.exists(), sorted(str(x.relative_to(p)) for x in p.rglob('*'))
                 if p.is_dir() else (hashlib.sha256(p.read_bytes()).hexdigest(),
                                    len(p.read_bytes().splitlines())) if p.is_file() else None)
                for p in paths]
    before = state()
    if parent:
        # Set controls after bash startup; the pi preview must clear BASH_ENV only in its child.
        body = '''export AILANG_SECRETS_LANE=claude-role MISSION_DRIVER_ROOT=synthetic-parent
export BASH_ENV=synthetic-nonexistent
"$PYTHON" "$HOME/env-hash.py" > "$HOME/before"
'''+preview+'''
"$PYTHON" "$HOME/env-hash.py" > "$HOME/after"
'''
    else:
        body = branch+'\nprintf body > "$HOME/controller-body"\nexit 89\n'
    (home/'run.sh').write_text(prefix+body)
    # A pipe on stdin, never /dev/null, so the probe-stdin arm can see the block's own redirect.
    result = subprocess.run(['/bin/bash', str(home/'run.sh')], env=env, stdin=subprocess.PIPE,
                            capture_output=True, timeout=15)
    require(state() == before, 'no controller body, pidfile or state writes')
    logs = (home/'log').read_bytes() if (home/'log').exists() else b''
    for key in KEYS:
        require(env[key].encode() not in result.stdout+result.stderr+logs, 'no value disclosure')
    # The controller's six live logs redirect stdout+stderr, including the inventory.
    return result.returncode, (result.stdout+result.stderr).decode(), home


def check_probe(home, provider):
    require((home/'probe').exists(), 'scoped probe executed')
    observation = json.loads((home/'probe').read_text())
    require(observation['marker_ok'], 'probe child marker')
    require(observation['root_ok'], 'probe child driver root')
    require(observation['model_ok'], 'pinned model parsed and probed')
    require(set(observation['present']) == ALLOWED, 'probe child stripped synthetic keys')
    if provider == 'pi':
        require(not observation['bash_env_present'], 'pi probe clears BASH_ENV')


def run(arm, root, home):
    if arm == 'dry-run-preview':
        for pin, provider in [('claude:synthetic-model', 'claude'), ('synthetic-model', 'claude'),
                              ('pi:synthetic-model', 'pi'), ('codex:synthetic-model', 'codex')]:
            rc, log, h = run_case(root, home/provider/('bare' if ':' not in pin else 'pin'), pin)
            require(rc == 0, 'healthy dry-run exit zero')
            check_probe(h, provider)
            require('secrets-scope: lane='+provider+'-controller kept=4[' in log, 'inventory line in captured dry-run log')
            require('DRY RUN secrets-scope: lane='+provider+'-controller marker=ok controller-probe=ok\n' in log,
                    'scoped preview success line')
            require('DRY RUN ok:' in log and '| lanes=ok |' in log, 'existing healthy summary')
        rc, log, h = run_case(root, home/'degraded', 'claude:synthetic-model', degraded=True)
        require(rc == 0 and '| lanes=DEGRADED(1)- synthetic invalid executor -> handoff |' in log,
                'existing degraded ledger preserved')
        rc, log, h = run_case(root, home/'missing', 'claude:synthetic-model', missing=True)
        require(rc == 78 and 'marker=failed rc=78' in log and 'DRY RUN failed: secrets-scope' in log,
                'scope failure refuses dry-run')
        require(not (h/'probe').exists() and 'DRY RUN ok:' not in log, 'scope failure never probes or reports ok')
    elif arm == 'dry-run-failure':
        for provider, status in [('claude', 2), ('codex', 41), ('pi', 75)]:
            rc, log, h = run_case(root, home/provider, provider+':synthetic-model', status)
            require(rc == status, 'controller probe failure returns nonzero status')
            require('marker=ok controller-probe=failed rc='+str(status)+'\n' in log, 'auth failure status line')
            require('DRY RUN failed: secrets-scope' in log and 'DRY RUN ok:' not in log,
                    'failure summary replaces success')
            check_probe(h, provider)
    elif arm == 'dry-run-parent-env':
        for provider in ['claude', 'codex', 'pi']:
            rc, log, h = run_case(root, home/provider, provider+':synthetic-model', parent=True)
            require(rc == 0, 'parent preview completes')
            require((h/'before').read_bytes() == (h/'after').read_bytes(), 'parent environment sha and count unchanged')
            check_probe(h, provider)
    elif arm == 'dry-run-codex-forward':
        rc, log, h = run_case(root, home, 'codex:synthetic-model')
        require(rc == 0, 'codex preview exit zero')
        require('DRY RUN secrets-scope: codex-marker-forward=ok driver-root-forward=ok\n' in log,
                'codex marker and root forwarded')
        check_probe(h, 'codex')
    elif arm == 'dry-run-codex-forward-rc':
        rc, log, h = run_case(root, home, 'codex:synthetic-model', forward='fail')
        require(rc == 41, 'codex forwarding failure returns its own status')
        require('DRY RUN failed: secrets-scope rc=41' in log and 'DRY RUN ok:' not in log,
                'forwarding failure summary replaces success')
        require(not (h/'probe').exists(), 'forwarding failure never probes')
    elif arm == 'dry-run-codex-forward-missing':
        rc, log, h = run_case(root, home, 'codex:synthetic-model', forward='missing-marker')
        require('codex-marker-forward=failed driver-root-forward=ok\n' in log, 'missing marker reported')
        require(rc == 78 and 'DRY RUN failed: secrets-scope rc=78' in log, 'missing marker refuses dry-run')
        require(not (h/'probe').exists() and 'DRY RUN ok:' not in log, 'missing marker never probes or reports ok')
    elif arm == 'dry-run-probe-stdin':
        for provider in ['claude', 'codex', 'pi']:
            rc, log, h = run_case(root, home/provider, provider+':synthetic-model')
            require(rc == 0, 'probe-stdin preview exit zero')
            require(json.loads((h/'probe').read_text())['stdin_null'], 'probe stdin is /dev/null')
    elif arm == 'dry-run-no-pin':
        for pin, name in [(None, 'unset'), ('', 'empty')]:
            rc, log, h = run_case(root, home/name, pin)
            require(rc == 0, 'no-pin exit unchanged')
            require('DRY RUN secrets-scope: preview=skipped reason=no-env-pin\n' in log, 'no-pin skip line')
            require(not (h/'probe').exists(), 'no-pin skips probes')
            require('DRY RUN ok:' in log and '| lanes=ok |' in log, 'no-pin summary unchanged')


def mutate(root, arm):
    target = root/DRIVER
    original = target.read_text()
    block = extract(original, START, END)
    if arm == 'dry-run-preview':
        changed = block.replace('mc_secrets_scope_apply "$AILANG_SECRETS_LANE"', ':', 1)
    elif arm == 'dry-run-failure':
        changed = block.replace('_dry_probe_rc=$?', '_dry_probe_rc=0', 1)
    elif arm == 'dry-run-parent-env':
        changed = block.replace('if ( # scoped preview child', 'if { # scoped preview child', 1).replace(
            ') ; then # scoped preview child', '} ; then # scoped preview child', 1)
    elif arm == 'dry-run-codex-forward':
        # Exercise the actual forwarding helper with the narrow exception removed.
        target = root/'tools/launchd/lib/codex-env-args.sh'
        original = target.read_text()
        changed = original.replace('AILANG_SECRETS_LANE|CODEX_HOME|', 'CODEX_HOME|', 1)
    elif arm == 'dry-run-codex-forward-rc':
        changed = block.replace('[ "$_dry_forward_rc" -eq 0 ] || exit "$_dry_forward_rc"', ':', 1)
    elif arm == 'dry-run-codex-forward-missing':
        changed = block.replace('[ "$_dry_marker_forward" = ok ] && [ "$_dry_root_forward" = ok ] || exit 78', ':', 1)
    elif arm == 'dry-run-probe-stdin':
        changed = block.replace('esac < /dev/null', 'esac', 1)
    else:
        changed = block.replace('log "DRY RUN secrets-scope: preview=skipped reason=no-env-pin"', ':', 1)
    if arm != 'dry-run-codex-forward':
        changed = original.replace(block, changed, 1)
    require(changed != original, 'effective mutation construction')
    target.write_text(changed)
    syntax = subprocess.run(['/bin/bash', '-n', str(target)], capture_output=True)
    require(syntax.returncode == 0, 'mutant remains valid bash')
    print('MUTATION-DIFF arm='+arm+' '+json.dumps(''.join(difflib.unified_diff(
        original.splitlines(True), changed.splitlines(True), fromfile=target.name, tofile=target.name, n=0))))
    return str(target.relative_to(root))


fail = 0
with tempfile.TemporaryDirectory(prefix='cases-', dir=os.environ['TMPDIR']) as temporary:
    tmp = pathlib.Path(temporary)
    for arm in ARMS:
        if SELECT and SELECT != arm:
            continue
        try:
            run(arm, ROOT, tmp/(arm+'-control'))
            if MODE != '--mutations':
                print('PASS: '+arm)
                continue
            scratch = tmp/(arm+'-mutant-root')
            for name in FILES:
                destination = scratch/name
                destination.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(ROOT/name, destination)
            target = mutate(scratch, arm)
            try:
                run(arm, scratch, tmp/(arm+'-mutant-home'))
            except AssertionError as error:
                print('KILLED: mutant='+arm+' arm='+arm+' target='+target+' assertion='+str(error))
            else:
                raise AssertionError('mutant SURVIVED')
        except (AssertionError, subprocess.TimeoutExpired) as error:
            fail += 1
            print('FAIL: '+arm+' assertion='+str(error))
sys.exit(bool(fail))
PY
