#!/bin/bash
# Synthetic acceptance fixtures for the pre-publication scanner.
set +x
umask 077

helper="$(cd "$(dirname "$0")" && pwd)/mission_evidence_secret_scan.sh"
selected=''
while [ "$#" -gt 0 ]; do
  case "$1" in
    --helper) [ "$#" -ge 2 ] || exit 2; helper=$2; shift 2 ;;
    --arm) [ "$#" -ge 2 ] || exit 2; selected=$2; shift 2 ;;
    *) exit 2 ;;
  esac
done
case "$helper" in /*) ;; *) helper="$PWD/$helper" ;; esac

suite=$(mktemp -d "${TMPDIR:-/tmp}/evidence-scan-test.XXXXXX") || exit 2
worker_pid=''
helper_pid=''
cleanup() {
  if [ -n "$helper_pid" ]; then
    kill -KILL "$helper_pid" 2>/dev/null || true
    wait "$helper_pid" 2>/dev/null || true
  fi
  if [ -n "$worker_pid" ]; then kill -KILL "$worker_pid" 2>/dev/null || true; fi
  rm -rf "$suite"
}
trap cleanup EXIT
trap 'exit 2' HUP INT TERM

tool_path=/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin
export HOME="$suite/home" TMPDIR="$suite" LC_ALL=C PATH="$tool_path"
mkdir -p "$HOME" || exit 2
cd "$suite" || exit 2

control='SyntheticControl020!'
short='SyntheticShort019!!'
# shellcheck disable=SC2016 # Shell syntax is literal fixture data.
literal='literal.[x]*$(touch PWNED)'
oauth='synthetic-oauth-credential-136'
lower='synthetic-lower-credential-136'
from_file='synthetic-duplicate-file-136'
from_env='synthetic-duplicate-env-136'
hidden='synthetic-path-credential-136'
values=("$control" "$short" "$literal" "$oauth" "$lower" "$from_file" "$from_env" "$hidden")

fail() {
  printf 'FAIL %s/%s rc=%s\n' "$arm" "$1" "${2:-$rc}" >&2
  failures=$((failures + 1))
}

begin_case() {
  fixture=$(mktemp -d "$suite/case.XXXXXX") || exit 2
  fixture_home="$fixture/home"
  fixture_tmp="$fixture/tmp"
  mkdir -p "$fixture_home/.config/ailang" "$fixture_tmp" || exit 2
  config="$fixture_home/.config/ailang/secrets.env"
  exports="$fixture/exports"
  out="$fixture/stdout"
  err="$fixture/stderr"
  helper_path=$tool_path
  : > "$exports"
}

launch_helper() {
  cd "$fixture" || exit 2
  # shellcheck disable=SC2016 # The scrubbed Bash reads the fixture assignments.
  exec env -i HOME="$fixture_home" TMPDIR="$fixture_tmp" PATH="$helper_path" LC_ALL=C \
    /bin/bash -c '
      while IFS= read -r assignment; do export "$assignment"; done < "$1"
      shift
      exec /bin/bash "$@"
    ' fixture "$exports" "$helper" "$@" >"$out" 2>"$err"
}

run_helper() {
  rc=0
  (launch_helper "$@") || rc=$?
  assertions_done=0
}

expected_rc() { [ "$rc" -eq "$2" ] || fail "$1"; }

verdict() {
  local label=$1 status=$2 values_count=$3 scanned=$4 hits=$5 control_state=$6
  local expected="mission_evidence_secret_scan: verdict=$status values=$values_count scanned=$scanned hits=$hits control=$control_state"
  [ "$(tail -n 1 "$out")" = "$expected" ] || fail "$label"
  [ "$(awk '/^mission_evidence_secret_scan: /{n++} END{print n+0}' "$out")" -eq 1 ] ||
    fail single-verdict
  [ "$(awk '/^LEAK key=/{n++} END{print n+0}' "$out")" -eq "$hits" ] || fail finding-count
}

finding() { grep -Fqx -- "$2" "$out" || fail "$1"; }

finish_case() {
  local value result entries
  # Never display capture files, even for a leaking mutant.
  for value in "${values[@]}"; do
    result=0
    grep -aFq -f /dev/stdin "$out" "$err" <<< "$value" || result=$?
    case "$result" in 0) fail no-raw-values ;; 1) ;; *) fail output-inspection ;; esac
  done
  [ ! -s "$err" ] || fail empty-stderr
  entries=$(find "$fixture_tmp" -mindepth 1 -print) || fail scratch-inspection
  [ -z "$entries" ] || fail scratch-cleanup
  assertions_done=1
}

control_export() { printf 'CONTROL_TOKEN=%s\n' "$control" >> "$exports"; }

arm_clean_dir() {
  begin_case
  control_export
  mkdir -p "$fixture/tree/sub"
  printf 'credential name only: CONTROL_TOKEN\n' > "$fixture/tree/one"
  printf '\000clean\377\n' > "$fixture/tree/sub/two"
  run_helper tree
  expected_rc clean-rc 0
  verdict recursive-count clean 1 2 0 ok
  [ "$(wc -l < "$out")" -eq 1 ] || fail clean-record-count
}

arm_file_value() {
  begin_case
  control_export
  printf '  # comment\n\n  export FROM_FILE="%s"\n' "$literal" > "$config"
  printf '\000prefix %s suffix\377\n' "$literal" > "$fixture/evidence"
  run_helper evidence
  expected_rc file-literal-rc 1
  finding file-literal-finding 'LEAK key=FROM_FILE path=evidence'
  verdict file-literal-verdict LEAK 2 1 1 ok
  [ ! -e "$fixture/PWNED" ] || fail no-shell-execution
}

arm_env_only() {
  begin_case
  control_export
  printf 'CLAUDE_CODE_OAUTH_TOKEN=%s\nfixture_password=%s\n' "$oauth" "$lower" >> "$exports"
  printf '%s and %s\n' "$oauth" "$lower" > "$fixture/evidence"
  run_helper evidence
  expected_rc env-only-rc 1
  finding oauth-finding 'LEAK key=CLAUDE_CODE_OAUTH_TOKEN path=evidence'
  finding lowercase-finding 'LEAK key=fixture_password path=evidence'
  verdict env-only-verdict LEAK 3 1 2 ok
}

arm_make_dump() {
  begin_case
  printf "SHARED_KEY='%s'\n" "$from_file" > "$config"
  printf 'SHARED_KEY=%s\nBOTH_SECRET=%s\n' "$from_env" "$from_env" >> "$exports"
  printf 'NAME = %s\n' "$from_file" > "$fixture/file-dump"
  printf 'NAME = %s %s\n' "$from_env" "$from_env" > "$fixture/env-dump"
  run_helper file-dump env-dump env-dump
  expected_rc make-dump-rc 1
  finding file-side-finding 'LEAK key=SHARED_KEY path=file-dump'
  finding env-side-finding 'LEAK key=SHARED_KEY path=env-dump'
  finding second-key-finding 'LEAK key=BOTH_SECRET path=env-dump'
  verdict make-dump-verdict LEAK 2 2 3 ok
}

arm_short_value() {
  begin_case
  control_export
  printf 'SHORT_SECRET=%s\n' "$short" > "$config"
  printf '%s\n' "$short" > "$fixture/evidence"
  [ "${#short}" -eq 19 ] && [ "${#control}" -eq 20 ] || fail threshold-fixture 2
  run_helper evidence
  expected_rc short-value-rc 0
  verdict threshold-verdict clean 1 1 0 ok
}

arm_zero_values() {
  begin_case
  printf 'NOPE=%s\n' "$short" > "$config"
  : > "$fixture/evidence"
  run_helper evidence
  expected_rc zero-values-rc 2
  verdict empty-inventory instrument-failure 0 0 0 failed
}

arm_control() {
  begin_case
  control_export
  : > "$fixture/evidence"
  run_helper evidence
  expected_rc control-rc 0
  verdict firing-control clean 1 1 0 ok
}

# Positive assertions for the disposable broken-matcher probe.
arm_broken_matcher() {
  begin_case
  control_export
  : > "$fixture/evidence"
  run_helper evidence
  expected_rc broken-matcher-rc 2
  verdict broken-matcher-fail-closed instrument-failure 1 0 0 failed
}

arm_unreadable_path() {
  begin_case
  control_export
  : > "$fixture/locked"
  chmod 000 "$fixture/locked" || exit 2
  if [ -r "$fixture/locked" ]; then
    printf 'SKIP %s/permission-readable\n' "$arm"
  else
    run_helper locked
    expected_rc unreadable-rc 2
    verdict unreadable-verdict instrument-failure 1 0 0 ok
    finish_case
  fi
  begin_case
  control_export
  printf '%s\n' "$control" > "$fixture/leaking"
  run_helper absent leaking
  expected_rc missing-path-rc 2
  finding later-input-finding 'LEAK key=CONTROL_TOKEN path=leaking'
  verdict missing-path-precedence instrument-failure 1 1 1 ok
}

arm_symlinked_prefix() {
  begin_case
  control_export
  mkdir "$fixture/real" || exit 2
  printf 'clean\n' > "$fixture/real/clean.txt"
  printf '%s\n' "$control" > "$fixture/real/leak.txt"
  ln -s real "$fixture/alias" || exit 2

  # Kills restoring O_NOFOLLOW on prefix components (prefix would return 2).
  run_helper alias/clean.txt
  expected_rc symlinked-prefix-clean-rc 0
  verdict symlinked-prefix-clean clean 1 1 0 ok
  finish_case

  run_helper alias/leak.txt
  expected_rc symlinked-prefix-leak-rc 1
  finding symlinked-prefix-leak-finding 'LEAK key=CONTROL_TOKEN path=alias/leak.txt'
  verdict symlinked-prefix-leak LEAK 1 1 1 ok
  finish_case

  # Kills following the final component (link would return 0).
  ln -s clean.txt "$fixture/alias/link.txt" || exit 2
  run_helper alias/link.txt
  expected_rc final-symlink-rc 2
  verdict final-symlink instrument-failure 1 0 0 ok
  finish_case

  run_helper real/
  expected_rc in-tree-symlink-rc 2
  verdict in-tree-symlink instrument-failure 1 2 1 ok
}

arm_invalid_inputs() {
  begin_case
  control_export
  printf 'BROKEN="unterminated\n' > "$config"
  : > "$fixture/evidence"
  run_helper evidence
  expected_rc malformed-source-rc 2
  verdict malformed-source-verdict instrument-failure 1 0 0 failed
  finish_case

  begin_case
  control_export
  mkdir "$fixture/tree"
  mkfifo "$fixture/tree/pipe"
  ln -s tree "$fixture/link"
  ln -s nowhere "$fixture/tree/dangling"
  : > "$fixture/tree/regular"
  run_helper link/ tree
  expected_rc unsafe-input-rc 2
  verdict unsafe-input-verdict instrument-failure 1 1 0 ok
  finish_case

  begin_case
  control_export
  mkdir "$fixture/bin" || exit 2
  cat > "$fixture/bin/python3" <<'WORKER'
#!/bin/bash
printf '%s\n' "$CONTROL_TOKEN"
printf '%s\n' "$CONTROL_TOKEN" >&2
exit 7
WORKER
  chmod +x "$fixture/bin/python3" || exit 2
  helper_path="$fixture/bin:$tool_path"
  run_helper evidence
  expected_rc worker-failure-rc 2
  verdict worker-failure-verdict instrument-failure 0 0 0 failed
}

arm_output_guard() {
  begin_case
  control_export
  printf 'PATH_TOKEN=%s\n' "$hidden" >> "$exports"
  printf '%s\n' "$control" > "$fixture/name-$hidden"
  printf '%s\n' "$control" > "$fixture/"$'line\nbreak'
  run_helper "name-$hidden" $'line\nbreak'
  expected_rc output-guard-rc 1
  finding redacted-path 'LEAK key=CONTROL_TOKEN path=[redacted]'
  finding redacted-name 'LEAK key=PATH_TOKEN path=[redacted]'
  finding escaped-path 'LEAK key=CONTROL_TOKEN path=line\u000abreak'
  verdict output-guard-verdict LEAK 2 2 3 ok
}

# Kills skipping filesystem entry name matching while still scanning file contents.
arm_name_value() {
  begin_case
  control_export
  printf 'PATH_TOKEN=%s\n' "$hidden" >> "$exports"
  mkdir -p "$fixture/dn" "$fixture/tree/sub-$hidden" || exit 2
  : > "$fixture/f-$hidden"
  : > "$fixture/dn/f-$hidden"
  : > "$fixture/tree/sub-$hidden/clean"

  run_helper "f-$hidden"
  expected_rc input-name-rc 1
  finding input-name-finding 'LEAK key=PATH_TOKEN path=[redacted]'
  verdict input-name-verdict LEAK 2 1 1 ok
  finish_case

  run_helper dn
  expected_rc child-file-name-rc 1
  finding child-file-name-finding 'LEAK key=PATH_TOKEN path=[redacted]'
  verdict child-file-name-verdict LEAK 2 1 1 ok
  finish_case

  run_helper tree
  expected_rc child-directory-name-rc 1
  finding child-directory-name-finding 'LEAK key=PATH_TOKEN path=[redacted]'
  verdict child-directory-name-verdict LEAK 2 1 1 ok
}

arm_interrupted() {
  begin_case
  control_export
  : > "$fixture/evidence"
  mkdir "$fixture/bin"
  cat > "$fixture/bin/python3" <<'WORKER'
#!/bin/bash
PATH=${PATH#*:} exec python3 -I -B "$HOME/worker.py"
WORKER
  cat > "$fixture_home/worker.py" <<'PYTHON'
import os
import signal
from pathlib import Path

home = Path.home()
(home / "pid.pending").write_text(str(os.getpid()))
(home / "pid.pending").rename(home / "worker.pid")
signal.pause()
PYTHON
  chmod +x "$fixture/bin/python3" || exit 2
  helper_path="$fixture/bin:$tool_path"
  (launch_helper evidence) &
  helper_pid=$!
  assertions_done=0
  local deadline=$(( $(date +%s) + 8 ))
  while [ ! -s "$fixture_home/worker.pid" ] && kill -0 "$helper_pid" 2>/dev/null &&
    [ "$(date +%s)" -lt "$deadline" ]; do sleep 0.05; done
  rc=2
  if [ ! -s "$fixture_home/worker.pid" ]; then
    fail interruption-worker-ready
    return
  fi
  worker_pid=$(cat "$fixture_home/worker.pid")
  [ -n "$(find "$fixture_tmp" -mindepth 1 -print)" ] || fail interruption-scratch-ready
  kill -TERM "$helper_pid" 2>/dev/null || fail send-term
  deadline=$(( $(date +%s) + 8 ))
  while kill -0 "$helper_pid" 2>/dev/null && [ "$(date +%s)" -lt "$deadline" ]; do sleep 0.05; done
  if kill -0 "$helper_pid" 2>/dev/null; then
    fail bounded-reap
    kill -KILL "$helper_pid" 2>/dev/null || true
  fi
  rc=0
  wait "$helper_pid" 2>/dev/null || rc=$?
  helper_pid=''
  expected_rc interrupted-rc 2
  verdict interrupted-verdict instrument-failure 0 0 0 failed
  if kill -0 "$worker_pid" 2>/dev/null; then fail worker-reaped; else worker_pid=''; fi
}

write_archive_fixtures() {
  python3 -I -B - "$fixture" "$1" <<'PYTHON'
import gzip
import io
import shutil
import subprocess
import sys
import tarfile
import warnings
import zipfile
from pathlib import Path

root = Path(sys.argv[1])
scenario = sys.argv[2]
exports = dict(line.split(b"=", 1) for line in (root / "exports").read_bytes().splitlines())
secret = exports[b"CONTROL_TOKEN"]


def make_tar(entries):
    buffer = io.BytesIO()
    with tarfile.open(fileobj=buffer, mode="w") as archive:
        for name, data in entries:
            member = tarfile.TarInfo(name)
            member.size = len(data)
            archive.addfile(member, io.BytesIO(data))
    return buffer.getvalue()


def make_zip(entries):
    buffer = io.BytesIO()
    with warnings.catch_warnings():
        warnings.simplefilter("ignore", UserWarning)  # Duplicate records are intentional.
        with zipfile.ZipFile(buffer, "w", zipfile.ZIP_DEFLATED) as archive:
            for name, data in entries:
                archive.writestr(name, data)
    return buffer.getvalue()


leaking = ("earlier.txt", secret)
if scenario == "tar":
    (root / "bundle.tar").write_bytes(
        make_tar([("name with spaces.txt", secret), ("name with spaces.txt", b"clean")])
    )
    (root / "renamed.bin").write_bytes(make_tar([("header-only", b"clean")]))
elif scenario == "tgz":
    crossing = (b"\0" * (65536 - 5)) + secret
    (root / "cross.TGZ").write_bytes(gzip.compress(make_tar([("leading bytes.bin", crossing)])))
    (root / "bundle.tar.gz").write_bytes(
        gzip.compress(make_tar([("secret.txt", secret), ("safe.txt", b"clean")]))
    )
elif scenario == "zip":
    (root / "bundle.zip").write_bytes(
        make_zip([("name with spaces.txt", secret * 25), ("name with spaces.txt", b"clean")])
    )
elif scenario == "nested":
    (root / "by-name.tar").write_bytes(make_tar([leaking, ("nested.TgZ", b"opaque")]))
    (root / "by-header.tar").write_bytes(
        make_tar([
            leaking,
            ("renamed-zip.bin", make_zip([("safe.txt", b"clean")])),
            ("renamed-tar.bin", make_tar([("safe.txt", b"clean")])),
            ("renamed-gzip.bin", gzip.compress(make_tar([("safe.txt", b"clean")]))),
        ])
    )
elif scenario == "compressed-unsupported":
    plain_tar = make_tar([leaking])
    for tool, suffix, magic in [("bzip2", "bz2", b"BZh"), ("xz", "xz", b"\xfd7zXZ\x00")]:
        compressor = shutil.which(tool)
        for name, payload in [("plain", secret * 64), ("bundle.tar", plain_tar)]:
            if compressor:
                data = subprocess.run(
                    [compressor, "-c"], input=payload, check=True, capture_output=True
                ).stdout
            else:
                data = magic + payload
            (root / f"{name}.{suffix}").write_bytes(data)
    (root / "frame.zst").write_bytes(b"\x28\xb5\x2f\xfdopaque")
    (root / "archive.7z").write_bytes(b"7z\xbc\xaf\x27\x1copaque")
    renamed = (root / "plain.xz").read_bytes()
    (root / "renamed.bin").write_bytes(renamed)
    (root / "member.tar").write_bytes(make_tar([("renamed.bin", renamed)]))
    (root / "member.zip").write_bytes(make_zip([("renamed.bin", renamed)]))
elif scenario == "bad":
    (root / "bad.tar").write_bytes(b"not an archive")
    (root / "short-end.tar").write_bytes(make_tar([leaking])[:1536])
    # Put the trailer beyond the gzip reader's prefetch from the last member.
    complete = gzip.compress(make_tar([leaking]) + b"\0" * 65536)
    (root / "bad-trailer.tgz").write_bytes(complete[:-8] + bytes([complete[-8] ^ 1]) + complete[-7:])
    (root / "missing-trailer.tgz").write_bytes(complete[:-8])
    zip_data = make_zip([leaking])
    (root / "missing-end.zip").write_bytes(zip_data[:-4])
    central = zip_data.index(b"PK\x01\x02")
    corrupted = zip_data[:central + 16] + bytes([zip_data[central + 16] ^ 1]) + zip_data[central + 17:]
    (root / "crc.zip").write_bytes(corrupted)
elif scenario == "member-name":
    name = "label-" + exports[b"PATH_TOKEN"].decode("ascii")
    (root / "labels.tar").write_bytes(make_tar([(name, b"clean")]))
    (root / "labels.zip").write_bytes(make_zip([(name, b"clean")]))
elif scenario == "links":
    with tarfile.open(root / "link.tar", "w") as archive:
        member = tarfile.TarInfo("link")
        member.type = tarfile.SYMTYPE
        member.linkname = "elsewhere"
        archive.addfile(member)
    with zipfile.ZipFile(root / "link.zip", "w") as archive:
        member = zipfile.ZipInfo("link")
        member.create_system = 3
        member.external_attr = 0o120777 << 16
        archive.writestr(member, b"elsewhere")
else:
    raise AssertionError("unknown archive fixture")
PYTHON
}

arm_tar() {
  begin_case
  control_export
  write_archive_fixtures tar || exit 2
  run_helper bundle.tar renamed.bin
  expected_rc tar-rc 1
  finding tar-member-finding 'LEAK key=CONTROL_TOKEN path=bundle.tar member=name with spaces.txt'
  verdict tar-record-count LEAK 1 3 1 ok
}

arm_tgz() {
  begin_case
  control_export
  write_archive_fixtures tgz || exit 2
  run_helper cross.TGZ bundle.tar.gz
  expected_rc tgz-rc 1
  finding tgz-boundary-finding 'LEAK key=CONTROL_TOKEN path=cross.TGZ member=leading bytes.bin'
  finding tar-gz-member-finding 'LEAK key=CONTROL_TOKEN path=bundle.tar.gz member=secret.txt'
  verdict tgz-member-count LEAK 1 3 2 ok
}

arm_zip() {
  if ! command -v unzip >/dev/null; then
    printf 'SKIP %s/unzip-unavailable\n' "$arm"
    arm_skipped=1
    return
  fi
  begin_case
  control_export
  write_archive_fixtures zip || exit 2
  unzip -tqq "$fixture/bundle.zip" >/dev/null 2>&1 || { fail deflated-zip-fixture; return; }
  run_helper bundle.zip
  expected_rc zip-rc 1
  finding zip-member-finding 'LEAK key=CONTROL_TOKEN path=bundle.zip member=name with spaces.txt'
  verdict zip-member-count LEAK 1 2 1 ok
}

arm_nested_archive() {
  local archive
  for archive in by-name.tar by-header.tar; do
    begin_case
    control_export
    write_archive_fixtures nested || exit 2
    run_helper "$archive"
    expected_rc "nested-$archive-rc" 2
    finding "nested-$archive-prior-leak" "LEAK key=CONTROL_TOKEN path=$archive member=earlier.txt"
    verdict "nested-$archive-precedence" instrument-failure 1 1 1 ok
    finish_case
  done
}

# Kills dropping the unsupported-compression magic check from archive detection.
arm_compressed_unsupported() {
  local artifact
  begin_case
  control_export
  write_archive_fixtures compressed-unsupported || exit 2
  for artifact in plain.bz2 plain.xz bundle.tar.bz2 bundle.tar.xz frame.zst archive.7z \
    renamed.bin member.tar member.zip; do
    run_helper "$artifact"
    expected_rc "unsupported-$artifact-rc" 2
    verdict "unsupported-$artifact-verdict" instrument-failure 1 0 0 ok
    finish_case
  done
}

arm_bad_archive() {
  local archive scanned hits
  for archive in bad.tar short-end.tar bad-trailer.tgz missing-trailer.tgz missing-end.zip crc.zip; do
    begin_case
    control_export
    write_archive_fixtures bad || exit 2
    case "$archive" in
      short-end.tar|*-trailer.tgz) scanned=1; hits=1 ;;
      *) scanned=0; hits=0 ;;
    esac
    run_helper "$archive"
    expected_rc "invalid-$archive-rc" 2
    verdict "invalid-$archive-verdict" instrument-failure 1 "$scanned" "$hits" ok
    finish_case
  done
}

arm_member_name() {
  begin_case
  control_export
  printf 'PATH_TOKEN=%s\n' "$hidden" >> "$exports"
  write_archive_fixtures member-name || exit 2
  run_helper labels.tar labels.zip
  expected_rc member-name-rc 1
  finding redacted-tar-member 'LEAK key=PATH_TOKEN path=labels.tar member=[redacted]'
  finding redacted-zip-member 'LEAK key=PATH_TOKEN path=labels.zip member=[redacted]'
  verdict member-name-verdict LEAK 2 2 2 ok
}

arm_archive_links() {
  begin_case
  control_export
  write_archive_fixtures links || exit 2
  run_helper link.tar link.zip
  expected_rc archive-link-rc 2
  verdict archive-link-verdict instrument-failure 1 0 0 ok
}

arms=(clean-dir file-value env-only make-dump short-value zero-values control
  unreadable-path symlinked-prefix invalid-inputs output-guard name-value interrupted tar tgz zip
  nested-archive compressed-unsupported bad-archive member-name archive-links)
failures=0
ran=0
for arm in "${arms[@]}" broken-matcher; do
  if [ -z "$selected" ]; then [ "$arm" != broken-matcher ] || continue
  else [ "$arm" = "$selected" ] || continue
  fi
  before=$failures
  rc=2
  arm_skipped=0
  assertions_done=1
  "arm_${arm//-/_}"
  if [ "$assertions_done" -eq 0 ]; then finish_case; fi
  if [ "$failures" -eq "$before" ] && [ "$arm_skipped" -eq 0 ]; then printf 'PASS %s\n' "$arm"; fi
  ran=$((ran + 1))
done
[ "$ran" -gt 0 ] || { printf 'FAIL unknown-arm rc=2\n' >&2; exit 2; }
[ "$failures" -eq 0 ]
