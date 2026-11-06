#!/bin/bash
# shellcheck shell=bash
# Actual production spawn blocks, synthetic HOME, fake providers, no inference.
# Invoked by test_secrets_scope.sh in both normal and --mutations modes.
set -eu
ROOT=$(cd "$(dirname "$0")/../.." && pwd -P)
SCOPE_TMP=$(mktemp -d "${TMPDIR:-/tmp}/scope-spawn.XXXXXX")
trap 'rm -rf "$SCOPE_TMP"' EXIT
export TMPDIR="$SCOPE_TMP"
python3 - "$ROOT" "${1-}" "${2-}" <<'PY'
import difflib
import json
import os
import pathlib
import shutil
import subprocess
import sys
import tempfile

ROOT = pathlib.Path(sys.argv[1])
MODE, SELECT = sys.argv[2:]
KEYS = ['ANTHROPIC_API_KEY', 'CLAUDE_CODE_OAUTH_TOKEN', 'GEMINI_AGENTS_API_KEY',
        'GOOGLE_API_KEY', 'OLLAMA_API_KEY', 'OPENAI_API_KEY', 'OPENROUTER_API_KEY', 'FUTURE_SCOPE_KEY']
CONTROLLER = {'CLAUDE_CODE_OAUTH_TOKEN', 'OLLAMA_API_KEY', 'OPENAI_API_KEY', 'OPENROUTER_API_KEY'}
FILES = ['tools/launchd/mission-control.sh', 'tools/launchd/lib/secrets-scope.sh',
         'tools/launchd/lib/codex-env-args.sh', 'tools/launchd/lib/codex-auth-profile.sh',
         'tools/launchd/lib/pi-ext-args.sh', 'tools/launchd/lib/pi-controller-note.sh',
         'tools/launchd/transcript-secret-scan.sh']
SCANNER = FILES[-1]
FILES += ['scripts/mission_scoped_run.sh', 'scripts/mission_pi_run.sh',
          'tools/pi-extensions/sandbox/index.ts', 'tools/pi-extensions/sandbox/package.json',
          'tools/pi-extensions/sandbox/mission.ts', 'tools/pi-extensions/sandbox/sandbox.mission.json',
          'tools/pi-extensions/worktree-fence.ts']
ARMS = ['controller-codex', 'controller-pi', 'controller-claude', 'codex-ordering',
        'retry-switch', 'codex-marker-root', 'deny-isolation', 'scoped-include-only',
        'root-quoting', 'unknown-marker', 'detector', 'role-codex', 'role-claude', 'pi-runner', 'pi-bash-env']

def require(ok, message):
    if not ok:
        raise AssertionError(message)

def extract(text, start, end):
    require(text.count(start) == 1 and text.count(end) == 1, 'nonempty unique extraction')
    block = text.split(start, 1)[1].split(end, 1)[0]
    require(block.strip(), 'nonempty extraction')
    return block

def fixture(home, root):
    for part in ['.config/ailang', '.codex', 'zdot', 'bin', 'repo', 'tmp', '.ailang/state']:
        (home/part).mkdir(parents=True, exist_ok=True)
    (home/'.config/ailang/secrets.env').write_text(''.join('export '+k+'=SYNTHETIC_SCOPE_'+k+'_END\n' for k in KEYS))
    env = dict(HOME=str(home), ZDOTDIR=str(home/'zdot'), CODEX_HOME=str(home/'.codex'),
               PATH=str(home/'bin')+':/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin', BASH_ENV='', ENV='', TMPDIR=str(home/'tmp'),
               AILANG_STATE_DIR=str(home/'.ailang/state'), MISSION_DRIVER_ROOT=str(root),
               MC_DRIVER_ROOT=str(root), MODEL='fake-model', REPO=str(home/'repo'), PROMPT='fixture-prompt',
               LOG=str(home/'log'), MC_PI_SESSION_ID='fixture-session', MC_PI_RESUME_PROMPT='resume',
               PYTHON=sys.executable, LC_ALL='C')
    env.update({k: 'SYNTHETIC_SCOPE_'+k+'_END' for k in KEYS})
    (home/'capture.py').write_text('''import json, os, pathlib, sys
keys = json.loads(pathlib.Path(os.environ['HOME'], 'keys.json').read_text())
pathlib.Path(os.environ['HOME'], 'capture').write_text(json.dumps(dict(
    present=[k for k in keys if k in os.environ], marker=os.environ.get('AILANG_SECRETS_LANE'),
    root=os.environ.get('MISSION_DRIVER_ROOT'), bash_env='BASH_ENV' in os.environ, argv=sys.argv[1:])))
''')
    (home/'keys.json').write_text(json.dumps(KEYS))
    for name in ['codex', 'pi', 'claude']:
        p = home/'bin'/name
        p.write_text('#!/bin/bash\nexec "$PYTHON" "$HOME/capture.py" "$@"\n')
        p.chmod(0o755)
    return env

def bash(home, env, body):
    # The file lives under this arm's synthetic HOME. Never run a caller's startup files.
    p = home/'run.sh'
    p.write_text('set -u\n'+body)
    r = subprocess.run(['/bin/bash', str(p)], env=env, capture_output=True, timeout=30)
    for key in KEYS:
        require(env.get(key, '').encode() not in r.stdout+r.stderr if env.get(key) else True,
                'no disclosure in stdout/stderr')
    return r.returncode

def sources(root):
    lib = root/'tools/launchd/lib'
    return '\n'.join('. '+json.dumps(str(lib/name)) for name in
                     ['secrets-scope.sh', 'codex-env-args.sh', 'codex-auth-profile.sh'])+'\n'

def spawn_block(root):
    driver = (root/FILES[0]).read_text()
    load = extract(driver, '# --- SECRETS SCOPE LIBRARY START ---', '# --- SECRETS SCOPE LIBRARY END ---')
    require('. "$MC_DRIVER_ROOT/tools/launchd/lib/secrets-scope.sh"' in load, 'driver loads pure helper')
    block = extract(driver, '  # --- CODEX CONTROLLER EXEC START ---', '  CONTROLLER_PID=$!')
    return load+'''\n_mc_pi_session_exists() { return 1; }
'''+block+'\np=$!\nwait "$p"\n'

def capture(home):
    require((home/'capture').exists(), 'provider launched')
    return json.loads((home/'capture').read_text())

def check_child(home, lane):
    c = capture(home)
    expected = CONTROLLER if lane.endswith('-controller') else (
        {'CLAUDE_CODE_OAUTH_TOKEN'} if lane == 'claude-role' else
        {'OLLAMA_API_KEY', 'OPENROUTER_API_KEY'} if lane == 'pi-role' else set())
    require(set(c['present']) == expected, 'child allowlist')
    require(c['marker'] == lane, 'child marker')
    return c

def forwarded(c):
    return {a.split('=', 1)[0].rsplit('.', 1)[-1]: json.loads(a.split('=', 1)[1])
            for a in c['argv'] if a.startswith('shell_environment_policy.set.')}

def run(arm, root, home):
    env = fixture(home, root)
    prefix = sources(root)
    if arm.startswith('role-'):
        provider = arm[5:]
        lane = provider+'-role'
        env['AILANG_SECRETS_LANE'] = 'claude-controller' # Must replace inherited controller scope.
        env['MISSION_DRIVER_ROOT'] = str(home/'wrong-root')
        env['MISSION_NAME'] = 'synthetic'
        command = '/bin/bash '+json.dumps(str(root/'scripts/mission_scoped_run.sh'))+' '+lane+' -- '+provider+' exec --model fake-model -o output -\n'
        require(bash(home, env, command) == 0, 'role wrapper runs')
        c = check_child(home, lane)
        require(c['root'] == str(root), 'wrapper derives its own root')
        require(c['argv'][:6] == ['exec', '--model', 'fake-model', '-o', 'output', '-'], 'existing argv preserved')
        if provider == 'codex':
            fwd = forwarded(c)
            require(fwd.get('AILANG_SECRETS_LANE') == lane and fwd.get('MISSION_DRIVER_ROOT') == str(root), 'role controls forwarded')
            require(fwd.get('MISSION_NAME') == 'synthetic', 'permitted mission controls forwarded')
        (home/'capture').unlink()
        (home/'.config/ailang/secrets.env').unlink()
        require(bash(home, env, command) != 0 and not (home/'capture').exists(), 'role refuses scope failure')
        return
    if arm in ['pi-runner', 'pi-bash-env']:
        # A read-only fake Git supplies metadata; this arm performs NO Git writes.
        # Existing runner suites separately cover real repository and commit behavior.
        (home/'repo-meta').mkdir()
        git = home/'bin/git'
        git.write_text('''#!/bin/bash
case "$*" in
  *rev-parse*--git-common-dir*|*rev-parse*--git-dir*) printf '%s\\n' "$HOME/repo-meta" ;;
  *rev-parse*HEAD*) echo synthetic-head ;;
  *rev-list*) echo 0 ;;
  *status*|*diff*|*ls-files*) : ;;
  *hash-object*--stdin*) /usr/bin/shasum | /usr/bin/awk '{print $1}' ;;
  *) exit 70 ;;
esac
''')
        git.chmod(0o755)
        stage = home.parent/(home.name+'-stage')
        stage.mkdir()
        node = stage/'node_modules/@anthropic-ai/sandbox-runtime'
        node.mkdir(parents=True)
        (node/'package.json').write_text('{}')
        env.update(MISSION_PI_SANDBOX_STAGE_PARENT=str(stage), MISSION_PI_CLAUDE_TMP_DIR=str(stage/'claude'),
                   MISSION_PI_SANDBOX_NODE_MODULES=str(stage/'node_modules'), MISSION_PI_POLL_SECONDS='1',
                   MISSION_DRIVER_ROOT=str(home/'wrong-root'), AILANG_SECRETS_LANE='claude-controller')
        # The fake provider starts another noninteractive bash, just as pi's bash tool does.
        (home/'bin/pi').write_text('''#!/bin/bash
: > "$PI_SANDBOX_READY_FILE"
cat >/dev/null
/bin/bash -c '\"$PYTHON\" \"$HOME/capture.py\" \"$@\"' bash "$@"
printf '%s\\n' '{"type":"agent_end"}'
''')
        (home/'directive').write_text('synthetic directive\n')
        (home/'restore.sh').write_text('export FUTURE_SCOPE_KEY=SYNTHETIC_RESTORED\n')
        cmd = '/bin/bash '+json.dumps(str(root/'scripts/mission_pi_run.sh'))+' --model fake --directive "$HOME/directive" --workdir "$HOME/repo" --out "$HOME/out" --max-seconds 10 --stall-seconds 5\n'
        # Set startup control after the outer shell has started; pi must not inherit it.
        body = 'BASH_ENV="$HOME/restore.sh"; export BASH_ENV\n'+cmd
        require(bash(home, env, body) == 10, 'runner completes with honest empty_worktree')
        c = check_child(home, 'pi-role')
        require(c['root'] == str(root), 'pi runner derives its own root')
        require(not c['bash_env'], 'pi role clears BASH_ENV before provider/tool bash')
        (home/'capture').unlink()
        (home/'.config/ailang/secrets.env').unlink()
        # Failure is a typed runner verdict, never a provider launch.
        require(bash(home, env, body) != 0 and not (home/'capture').exists(), 'pi runner refuses failed apply')
        return
    if arm.startswith('controller-') or arm in ['codex-ordering', 'retry-switch', 'detector']:
        block = spawn_block(root)
        providers = ['claude', 'codex', 'pi', 'codex'] if arm == 'retry-switch' else [arm[11:] if arm.startswith('controller-') else 'codex']
        body = prefix
        for provider in providers:
            body += 'CONTROLLER_PROVIDER='+provider+'\n'+block+'\n[ "$?" -eq 0 ] || exit 80\n'
            # Parent credentials and controls must survive every child, including retries/switches.
            body += '\n'.join('[ -n "${'+key+'-}" ] || exit 81' for key in KEYS)+'\n'
            body += '[ "${AILANG_SECRETS_LANE+x}" != x ] || exit 82\n'
            body += '\n"$PYTHON" -c '+json.dumps('import os; keys='+repr(KEYS)+'; assert all(os.environ[k] == "SYNTHETIC_SCOPE_"+k+"_END" for k in keys)')+' || exit 83\n'
            body += '[ "$MISSION_DRIVER_ROOT" = "$MC_DRIVER_ROOT" ] || exit 84\n'
        # Exercise bash tool startup from fake pi, not only pi's entry environment.
        if providers[-1] == 'pi':
            (home/'restore.sh').write_text('export FUTURE_SCOPE_KEY=SYNTHETIC_RESTORED\n')
            env['BASH_ENV'] = ''  # Set after the test shell has started.
            body = prefix+'BASH_ENV="$HOME/restore.sh"; export BASH_ENV\n'+body[len(prefix):]
        require(bash(home, env, body) == 0, 'spawn and parent isolation')
        c = check_child(home, providers[-1]+'-controller')
        require(c['root'] == str(root), 'controller root')
        if providers[-1] == 'pi':
            require(not c['bash_env'], 'pi controller clears BASH_ENV')
        if arm in ['codex-ordering', 'retry-switch']:
            fwd = forwarded(c)
            require(fwd.get('AILANG_SECRETS_LANE') == 'codex-controller', 'forwarding after marker/scope')
        # Each provider must refuse its launch if scope application fails.
        if arm.startswith('controller-'):
            (home/'capture').unlink()
            (home/'.config/ailang/secrets.env').unlink()
            body = prefix+'CONTROLLER_PROVIDER='+providers[-1]+'\n'+block
            require(bash(home, env, body) != 0 and not (home/'capture').exists(), 'failed apply refuses provider')
        if arm == 'detector':
            # Run the actual driver's post-fire invocation outside the child scope.
            scan = (root/FILES[0]).read_text().split('# --- TRANSCRIPT SCAN START ---', 1)[1]
            call = extract(scan, 'if [ "$_mc_scan_ready" -eq 1 ]; then', '  _mc_scan_rc=$?')
            sessions = home/'.codex/sessions'
            sessions.mkdir()
            (sessions/'synthetic.jsonl').write_text('SYNTHETIC_SCOPE_FUTURE_SCOPE_KEY_END\n')
            (home/'since').touch()
            os.utime(home/'since', (1, 1))
            body = prefix+'''_mc_bounded() { shift; "$@"; }
_mc_scan_marker="$HOME/since"
_mc_scan_baseline="$HOME/missing-baseline"
'''+call
            require(bash(home, env, body) == 1, 'detector sees stripped synthetic key from file inventory')
        return
    env.update(AILANG_SECRETS_LANE='codex-role', MISSION_DRIVER_ROOT=str(root/'root with spaces'),
               MISSION_X_SECRET='SYNTHETIC_DENIED_SECRET', MISSION_TEST_TOKEN='SYNTHETIC_DENIED_TOKEN',
               MISSION_SECRETS_OTHER='SYNTHETIC_DENIED_OTHER', MISSION_NAME='synthetic')
    if arm == 'scoped-include-only':
        (home/'.codex/config.toml').write_text('[shell_environment_policy]\ninclude_only = ["HOME"]\n')
    if arm == 'unknown-marker':
        for marker in ['unknown', '', 'codex-role\n']:
            env['AILANG_SECRETS_LANE'] = marker
            require(bash(home, env, prefix+'mc_codex_env_args\n') != 0, 'invalid marker refuses forwarding')
        return
    rc = bash(home, env, prefix+'mc_codex_env_args || exit $?\ncodex "${MC_CODEX_ENV_ARGS[@]}"\n')
    if arm == 'scoped-include-only':
        require(rc != 0 and not (home/'capture').exists(), 'marked include_only refuses')
        env.pop('AILANG_SECRETS_LANE')
        require(bash(home, env, prefix+'mc_codex_env_args\n') == 0, 'unmarked include_only still warns')
        return
    require(rc == 0, 'forward helper succeeds')
    fwd = forwarded(capture(home))
    if arm in ['codex-marker-root', 'root-quoting']:
        require(fwd.get('AILANG_SECRETS_LANE') == 'codex-role', 'exact marker forwarded')
        require(fwd.get('MISSION_DRIVER_ROOT') == env['MISSION_DRIVER_ROOT'], 'TOML root roundtrip with spaces')
        env['MISSION_DRIVER_ROOT'] = 'synthetic-root-\x01'
        require(bash(home, env, prefix+'mc_codex_env_args\n') == 78, 'marked unencodable root refuses')
        env.pop('AILANG_SECRETS_LANE')
        require(bash(home, env, prefix+'mc_codex_env_args\n') == 0, 'unmarked unencodable root still skips')
    if arm == 'deny-isolation':
        require(not ({'MISSION_X_SECRET', 'MISSION_TEST_TOKEN', 'MISSION_SECRETS_OTHER'} & fwd.keys()), 'other secret/token names denied')

# Each mutant changes exactly one physical line in a disposable copy.
def mutate(root, arm):
    target = FILES[0]
    if arm.startswith('controller-'):
        provider = arm[11:]
        s = (root/target).read_text()
        start = s.index('# --- SECRETS SCOPE '+provider.upper()+' CHILD START ---')
        line = next(x for x in s[start:].splitlines() if 'mc_secrets_scope_apply' in x)
        # Replace only the matching provider's call.
        i = s.index(line, start)
        new = s[:i]+s[i:].replace(line, '    : # mutant: omit child scope', 1)
    elif arm == 'codex-ordering':
        s = (root/target).read_text()
        old = '    export AILANG_SECRETS_LANE=codex-controller MISSION_DRIVER_ROOT="$MC_DRIVER_ROOT"'
        new = s.replace(old, '    mc_codex_env_args >/dev/null 2>&1; '+old.strip(), 1)
        # Reject early construction even if a later invocation repairs it.
        # The selected arm below instruments each build's marker prerequisite.
    elif arm == 'retry-switch':
        s = (root/target).read_text()
        new = s.replace('  if [ "$CONTROLLER_PROVIDER" = "codex" ]; then\n    # codex', '  mc_secrets_scope_apply codex-role # mutant: parent strip\n  if [ "$CONTROLLER_PROVIDER" = "codex" ]; then\n    # codex', 1)
    elif arm == 'detector':
        target = SCANNER
        s = (root/target).read_text()
        new = s.replace('secrets_file="$HOME/.config/ailang/secrets.env"', 'secrets_file=/dev/null', 1)
    elif arm.startswith('role-'):
        target = 'scripts/mission_scoped_run.sh'
        s = (root/target).read_text()
        new = s.replace('export AILANG_SECRETS_LANE="$lane" MISSION_DRIVER_ROOT="$RUNNER_ROOT"', 'export MISSION_DRIVER_ROOT="$RUNNER_ROOT" # mutant: inherited controller marker', 1)
    elif arm in ['pi-runner', 'pi-bash-env']:
        target = 'scripts/mission_pi_run.sh'
        s = (root/target).read_text()
        if arm == 'pi-runner':
            new = s.replace('  mc_secrets_scope_apply "$AILANG_SECRETS_LANE" || exit $?', '  : # mutant: omit role apply', 1)
        else:
            new = s.replace('  unset BASH_ENV', '  : # mutant: keep startup exporter', 1)
    else:
        target = 'tools/launchd/lib/codex-env-args.sh'
        s = (root/target).read_text()
        if arm == 'codex-marker-root':
            new = s.replace('AILANG_SECRETS_LANE|CODEX_HOME|', 'CODEX_HOME|', 1)
        elif arm == 'deny-isolation':
            new = s.replace('      AILANG_SECRETS_LANE) : ;;', '      AILANG_SECRETS_LANE|*SECRETS*|*SECRET*|*TOKEN*) : ;;', 1)
        elif arm == 'scoped-include-only':
            new = s.replace('\n    if [ "${AILANG_SECRETS_LANE+x}" = x ]; then', '\n    if false; then', 1)
        elif arm == 'root-quoting':
            new = s.replace('  enc=$(mc_toml_basic_string "$2") || return 1', '  enc=$2 # mutant: no TOML encoding', 1)
        elif arm == 'unknown-marker':
            new = s.replace("      *) echo 'codex-env: refused invalid scope marker' >&2; return 64 ;;", '      *) : ;;', 1)
        else:
            raise AssertionError('unknown mutant')
    require(new != s, 'mutation constructed')
    a, b = s.splitlines(), new.splitlines()
    require(sum(x != y for x, y in zip(a, b)) == 1 if len(a) == len(b) else
            len(b) == len(a)+1 and any(b[:i]+b[i+1:] == a for i in range(len(b))), 'single line mutation')
    (root/target).write_text(new)
    print('MUTATION-DIFF arm='+arm+' '+json.dumps(''.join(difflib.unified_diff(s.splitlines(True), new.splitlines(True), fromfile=target, tofile=target, n=0))))
    return target

# For ordering, every helper invocation must see the marker, not just the final invocation.
original_sources = sources
def sources(root):
    return original_sources(root)+'''eval "$(declare -f mc_codex_env_args | sed '1s/mc_codex_env_args/mc_real_codex_env_args/')"
mc_codex_env_args() {
  if [ "${CONTROLLER_PROVIDER-}" = codex ] && [ "${AILANG_SECRETS_LANE-}" != codex-controller ]; then
    : > "$HOME/early-forward"
  fi
  mc_real_codex_env_args
}
'''
original_run = run
def run(arm, root, home):
    original_run(arm, root, home)
    if arm == 'codex-ordering':
        require(not (home/'early-forward').exists(), 'no forwarding before child marker')

fail = 0
with tempfile.TemporaryDirectory(prefix='scope-spawn-', dir=os.environ['TMPDIR']) as tmp:
    tmp = pathlib.Path(tmp)
    for arm in ARMS:
        if SELECT and SELECT != arm:
            continue
        try:
            home = tmp/(arm+'-control')
            run(arm, ROOT, home)
            if MODE != '--mutations':
                print('PASS: '+arm)
                continue
            scratch = tmp/(arm+'-mutant-root')
            for name in FILES:
                dst = scratch/name
                dst.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(ROOT/name, dst)
            target = mutate(scratch, arm)
            try:
                run(arm, scratch, tmp/(arm+'-mutant-home'))
            except (AssertionError, ValueError) as error:
                print('KILLED: mutant='+arm+' arm='+arm+' target='+target+' assertion='+str(error))
            else:
                raise AssertionError('mutant SURVIVED')
        except (AssertionError, ValueError, subprocess.TimeoutExpired) as error:
            fail += 1
            print('FAIL: '+arm+' assertion='+str(error))
sys.exit(bool(fail))
PY
