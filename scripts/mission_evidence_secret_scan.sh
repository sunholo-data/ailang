#!/bin/bash
# Pre-publication guard. Keep inventory values out of argv and diagnostics.
set +x
umask 077
exec 2>/dev/null

scratch=''
worker=''
worker_done=0
interrupted=0

# shellcheck disable=SC2329 # Invoked by the EXIT trap.
terminal() {
  trap - EXIT
  trap '' HUP INT TERM
  local values=0 scanned=0 hits=0 control=failed failed=1 status=2
  local saved_values saved_scanned saved_hits saved_control saved_failed
  if [ -n "$worker" ]; then
    kill -TERM "$worker" || true
    kill -KILL "$worker" || true
    wait "$worker" || true
  fi
  if [ -n "$scratch" ]; then
    if IFS=' ' read -r saved_values saved_scanned saved_hits saved_control saved_failed < "$scratch/status"; then
      case "$saved_values:$saved_scanned:$saved_hits" in
        *[!0-9:]*|:*|*::*|*:) ;;
        *) values=$saved_values; scanned=$saved_scanned; hits=$saved_hits
           [ "$saved_control" != ok ] || control=ok
           if [ "$worker_done" -eq 1 ] && [ "$interrupted" -eq 0 ] &&
             [ "$saved_failed" = 0 ] && [ "$control" = ok ]; then failed=0; fi ;;
      esac
    fi
    if [ "$worker_done" -eq 1 ]; then
      cat < "$scratch/findings" || failed=1
    fi
    rm -rf "$scratch" || failed=1
  fi
  local verdict=instrument-failure
  if [ "$failed" -eq 0 ]; then
    if [ "$hits" -gt 0 ]; then verdict=LEAK; status=1
    else verdict=clean; status=0; fi
  fi
  printf 'mission_evidence_secret_scan: verdict=%s values=%s scanned=%s hits=%s control=%s\n' \
    "$verdict" "$values" "$scanned" "$hits" "$control" || exit 2
  exit "$status"
}
trap terminal EXIT
trap 'interrupted=1' HUP INT TERM

scratch=$(mktemp -d "${TMPDIR:-/tmp}/mission-evidence-secret-scan.XXXXXX") || exit 2
[ "$interrupted" -eq 0 ] || exit 2
# Input names may themselves contain values. Hand them off without using child argv.
printf '%s\0' "$@" > "$scratch/inputs" || exit 2
SCAN_SCRATCH="$scratch" python3 -I -B - >/dev/null <<'PYTHON' &
import gzip
import os
import re
import stat
import tarfile
import unicodedata
import zipfile
from contextlib import contextmanager
from pathlib import Path

MIN_BYTES = 20
CHUNK_SIZE = 65536
CREDENTIAL_NAME = re.compile(rb"TOKEN|KEY|SECRET|AUTH|PASSWORD|CREDENTIAL", re.IGNORECASE)
ASSIGNMENT = re.compile(rb"(?:export[ \t]+)?([a-zA-Z_][a-zA-Z0-9_]*)=(.*)")
NOFOLLOW = os.O_NOFOLLOW | os.O_NONBLOCK
TAR_BLOCK = tarfile.BLOCKSIZE
ZIP_SIGNATURES = (b"PK\x03\x04", b"PK\x05\x06", b"PK\x07\x08", b"PK\x06\x06")


@contextmanager
def open_regular(name, parent=None):
    fd = os.open(name, os.O_RDONLY | NOFOLLOW, dir_fd=parent)
    try:
        metadata = os.fstat(fd)
        if not stat.S_ISREG(metadata.st_mode):
            raise ValueError("expected regular file")
        stream = os.fdopen(fd, "rb")
    except BaseException:
        os.close(fd)
        raise
    with stream:
        yield stream


def file_assignments():
    name = os.path.join(os.environb[b"HOME"], b".config/ailang/secrets.env")
    try:
        os.lstat(name)
    except FileNotFoundError:
        return
    with open_regular(name) as stream:
        for line in stream:
            line = line.removesuffix(b"\n").removesuffix(b"\r").lstrip(b" \t")
            if not line or line.startswith(b"#"):
                continue
            match = ASSIGNMENT.fullmatch(line)
            if match is None or b"\0" in line:
                raise ValueError("malformed secrets file")
            key, value = match.groups()
            if value[:1] in (b"'", b'"'):
                if len(value) < 2 or value[-1:] != value[:1]:
                    raise ValueError("unmatched quote")
                value = value[1:-1]
            yield key, value


def matching_keys(data, inventory):
    return {key for value, keys in inventory.items() if value in data for key in keys}


def match_stream(stream, inventory, prefix=b"", expected_size=None):
    carry = b""
    overlap = max(map(len, inventory)) - 1
    hits = set()
    consumed = 0
    while True:
        chunk = prefix + stream.read(CHUNK_SIZE - len(prefix))
        prefix = b""
        if not chunk:
            if expected_size is not None and consumed != expected_size:
                raise ValueError("short member")
            return hits
        consumed += len(chunk)
        data = carry + chunk
        hits.update(matching_keys(data, inventory))
        carry = data[-overlap:]


def archive_kind(name, prefix):
    if prefix.startswith(b"\x1f\x8b"):
        return "gzip"
    if prefix.startswith(ZIP_SIGNATURES):
        return "zip"
    if len(prefix) >= TAR_BLOCK:
        if prefix[257:262] == b"ustar":
            return "tar"
        try:
            tarfile.TarInfo.frombuf(prefix[:TAR_BLOCK], "utf-8", "surrogateescape")
        except tarfile.HeaderError:
            pass
        else:
            return "tar"
    suffix = name.lower()
    if suffix.endswith((b".tar.gz", b".tgz")):
        return "gzip"
    if suffix.endswith(b".tar"):
        return "tar"
    if suffix.endswith(b".zip"):
        return "zip"
    return None


def validate_tar_end(stream, offset):
    stream.seek(offset)
    if stream.read(2 * TAR_BLOCK) != b"\0" * (2 * TAR_BLOCK):
        raise ValueError("missing tar end records")
    # TAR pads with zeroes. Draining also validates gzip's CRC and size trailer.
    while True:
        padding = stream.read(CHUNK_SIZE)
        if not padding:
            return
        if padding.strip(b"\0"):
            raise ValueError("nonzero tar padding")


def validate_zip_end(stream):
    # zipfile can open an EOCD whose declared comment has been truncated.
    eocd_size = 22
    size = stream.seek(0, os.SEEK_END)
    stream.seek(max(0, size - 65535 - eocd_size))
    tail = stream.read()
    offset = tail.rfind(b"PK\x05\x06")
    if offset < 0 or len(tail) - offset < eocd_size:
        raise ValueError("missing ZIP end record")
    comment_size = int.from_bytes(tail[offset + 20:offset + 22], "little")
    if len(tail) - offset != eocd_size + comment_size:
        raise ValueError("invalid ZIP end record length")
    stream.seek(0)


def safe_name(raw, inventory):
    if any(value in raw for value in inventory):
        return b"[redacted]"
    escaped = []
    for char in os.fsdecode(raw):
        if char == "\\":
            escaped.append("\\\\")
        elif unicodedata.category(char) in {"Cc", "Cf", "Cs", "Zl", "Zp"}:
            escaped.append(f"\\u{ord(char):04x}")
        else:
            escaped.append(char)
    rendered = "".join(escaped).encode("utf-8")
    return b"[redacted]" if any(value in rendered for value in inventory) else rendered


@contextmanager
def input_parent(path):
    # Walk explicit path components too, so a symlink cannot hide in a prefix.
    root = b"/" if path.startswith(b"/") else b"."
    parts = [part for part in path.split(b"/") if part]
    fd = os.open(root, os.O_RDONLY | os.O_DIRECTORY)
    try:
        for part in parts[:-1]:
            child = os.open(part, os.O_RDONLY | os.O_DIRECTORY | NOFOLLOW, dir_fd=fd)
            os.close(fd)
            fd = child
        yield fd, parts[-1] if parts else root
    finally:
        os.close(fd)


class Scan:
    def __init__(self, scratch, findings):
        self.scratch = scratch
        self.findings = findings
        metadata = scratch.stat()
        self.scratch_inode = (metadata.st_dev, metadata.st_ino)
        self.inventory = {}
        self.scanned = self.hits = 0
        self.control = "failed"
        self.failed = False
        self.seen = set()

    def enroll(self, key, value):
        if len(value) >= MIN_BYTES:
            self.inventory.setdefault(value, set()).add(key)

    def load(self):
        for key, value in os.environb.items():
            if CREDENTIAL_NAME.search(key):
                self.enroll(key, value)
        for key, value in file_assignments():
            self.enroll(key, value)

    def check_control(self):
        value = next(iter(self.inventory))
        fixture = self.scratch / "control"
        try:
            fixture.write_bytes(value)
            with fixture.open("rb") as stream:
                if not self.inventory[value].issubset(match_stream(stream, self.inventory)):
                    raise ValueError("control missed")
        finally:
            fixture.unlink()
        self.control = "ok"

    def write_status(self):
        pending = self.scratch / "status.next"
        pending.write_text(
            f"{len(self.inventory)} {self.scanned} {self.hits} {self.control} {int(self.failed)}\n",
            encoding="ascii",
        )
        pending.replace(self.scratch / "status")

    def record(self, path, keys, member=None, regular=True):
        self.scanned += int(regular)
        for key in sorted(keys):
            record = b"LEAK key=" + safe_name(key, self.inventory)
            record += b" path=" + safe_name(path, self.inventory)
            if member is not None:
                record += b" member=" + safe_name(member, self.inventory)
            self.findings.write(record + b"\n")
            self.hits += 1

    def scan_member(self, path, name, stream, size):
        prefix = stream.read(TAR_BLOCK)
        if archive_kind(name, prefix):
            raise ValueError("nested archive")
        keys = match_stream(stream, self.inventory, prefix, size)
        self.record(path, keys | matching_keys(name, self.inventory), member=name)

    def scan_tar(self, stream, path):
        with tarfile.open(fileobj=stream, mode="r:") as archive:
            for member in archive:
                try:
                    name = os.fsencode(member.name)
                    if member.isdir() and member.size == 0:
                        self.record(path, matching_keys(name, self.inventory), member=name, regular=False)
                    elif member.isfile():
                        with archive.extractfile(member) as body:
                            self.scan_member(path, name, body, member.size)
                    else:
                        raise ValueError("unsupported tar member")
                except Exception:
                    self.failed = True
            validate_tar_end(stream, archive.offset)

    def scan_zip(self, stream, path):
        validate_zip_end(stream)
        with zipfile.ZipFile(stream) as archive:
            for member in archive.infolist():
                try:
                    encoding = "utf-8" if member.flag_bits & 0x800 else "cp437"
                    name = member.orig_filename.encode(encoding)
                    kind = stat.S_IFMT(member.external_attr >> 16)
                    if member.flag_bits & 1 or kind not in (0, stat.S_IFREG, stat.S_IFDIR):
                        raise ValueError("unsupported ZIP member")
                    if member.is_dir() or kind == stat.S_IFDIR or member.external_attr & 0x10:
                        if member.file_size:
                            raise ValueError("nonempty ZIP directory")
                        self.record(path, matching_keys(name, self.inventory), member=name, regular=False)
                    else:
                        with archive.open(member) as body:
                            self.scan_member(path, name, body, member.file_size)
                except Exception:
                    self.failed = True

    def scan_file(self, parent, name, path):
        with open_regular(name, parent) as stream:
            kind = archive_kind(name, stream.read(TAR_BLOCK))
            stream.seek(0)
            if kind == "gzip":
                with gzip.GzipFile(fileobj=stream) as uncompressed:
                    self.scan_tar(uncompressed, path)
            elif kind == "tar":
                self.scan_tar(stream, path)
            elif kind == "zip":
                self.scan_zip(stream, path)
            else:
                self.record(path, match_stream(stream, self.inventory))

    def scan_entry(self, parent, name, path):
        try:
            metadata = os.stat(name, dir_fd=parent, follow_symlinks=False)
            if stat.S_ISDIR(metadata.st_mode):
                self.scan_directory(parent, name, path)
            elif stat.S_ISREG(metadata.st_mode):
                identity = os.path.abspath(path)
                if identity not in self.seen:
                    self.seen.add(identity)
                    self.scan_file(parent, name, path)
            else:
                raise ValueError("unsupported input")
        except Exception:
            self.failed = True

    def scan_directory(self, parent, name, path):
        fd = os.open(name, os.O_RDONLY | os.O_DIRECTORY | NOFOLLOW, dir_fd=parent)
        try:
            metadata = os.fstat(fd)
            # The scanner's temporary handoff is not part of a requested directory.
            if (metadata.st_dev, metadata.st_ino) == self.scratch_inode:
                return
            with os.scandir(fd) as entries:
                for entry in entries:
                    child = os.fsencode(entry.name)
                    self.scan_entry(fd, child, os.path.join(path, child))
        finally:
            os.close(fd)

    def run(self):
        try:
            self.load()
            if not self.inventory:
                raise ValueError("empty inventory")
            self.check_control()
            self.write_status()
            inputs = self.scratch / "inputs"
            paths = inputs.read_bytes().split(b"\0")[:-1]
            inputs.unlink()
            for path in paths:
                try:
                    if not path:
                        raise ValueError("empty input")
                    with input_parent(path) as (parent, name):
                        self.scan_entry(parent, name, path)
                except Exception:
                    self.failed = True
        except Exception:
            self.failed = True
        self.findings.flush()
        self.write_status()


scratch = Path(os.environ["SCAN_SCRATCH"])
with (scratch / "findings").open("wb") as findings:
    Scan(scratch, findings).run()
PYTHON
worker=$!
[ "$interrupted" -eq 0 ] || exit 2
if wait "$worker"; then worker_done=1; fi
# If wait was interrupted, leave the PID for the EXIT handler to reap.
[ "$interrupted" -ne 0 ] || worker=''
exit 2
