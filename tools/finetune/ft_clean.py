"""Trajectory cleaning for pilot-2 (m-eval-finetuning-data-pipeline §Pilot-2 direction).

Pilot-1 trained on trajectories as banked, and measured on traj_all.jsonl (2026-11-06):
- 12,991 of 23,740 samples carried another run's absolute eval workspace in context
  (`cd /var/folders/.../ailang_eval/<run>/<sub> && ailang run ...`), 1,359 in the target turn.
  A model trained on that `cd`s into directories that do not exist at eval time.
- 423 samples carried Go OpenTelemetry deprecation warnings in tool results.
- 14 target turns used `timeout`, which the rig's macOS does not have.

clean_messages() rewrites workspace paths to workspace-relative form (the eval runs the
agent with cwd = workspace, and its task prompt says `benchmark/solution.ail`), strips the
noise, and reports whether the target turn is still usable.
"""
import re

# <ws> = the per-run eval workspace: .../ailang_eval/<run>/<sub>
#   (the per-language <sub> level is optional; it always contains _ailang_ / _python_, so a
#    following `benchmark/` directory is never mistaken for it)
_RUN = r"ailang_eval/[^/\s\"']+(?:/[^/\s\"']*_(?:ailang|python)_[^/\s\"']*)?"
_WS = (r"(?:(?:/private)?/|(?<![\w/]))var/folders/[^/\s\"']+/[^/\s\"']+/T/" + _RUN +
       r"|/tmp/(?:[^/\s\"']+/)?" + _RUN)
CD_WS = re.compile(r"cd\s+[\"']?(?:" + _WS + r")/?[\"']?\s*(?:&&|;)\s*")
WS_PREFIX = re.compile(r"(?:" + _WS + r")/")
WS_BARE = re.compile(r"(?:" + _WS + r")(?=[\s\"'`]|$)")
OTEL = re.compile(r"^\d{4}/\d\d/\d\d \d\d:\d\d:\d\d Google Cloud OpenTelemetry Trace exporter[^\n]*\n?", re.M)
HOME = re.compile(r"/Users/[^/\s]+/")
TIMEOUT_CMD = re.compile(r"(?:^|[;&|]\s*)timeout\s+\d")


def clean_text(s: str) -> str:
    s = CD_WS.sub("", s)
    s = WS_PREFIX.sub("", s)
    s = WS_BARE.sub(".", s)
    return OTEL.sub("", s)


def _clean(obj):
    if isinstance(obj, str):
        return clean_text(obj)
    if isinstance(obj, list):
        return [_clean(x) for x in obj]
    if isinstance(obj, dict):
        return {k: _clean(v) for k, v in obj.items()}
    return obj


def clean_messages(msgs):
    """Return (cleaned messages, reject reason or None) for a per-target-turn sample."""
    out = _clean(msgs)
    target = out[-1]
    blob = (target.get("content") or "") + repr(target.get("tool_calls") or "")
    if HOME.search(blob):
        return out, "target_has_home_path"
    for tc in target.get("tool_calls") or []:
        args = tc.get("function", {}).get("arguments") or {}
        cmd = args.get("command", "") if isinstance(args, dict) else ""
        if TIMEOUT_CMD.search(cmd):
            return out, "target_uses_timeout"
    return out, None
