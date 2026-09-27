#!/usr/bin/env python3
"""Print self and reaped-child CPU centiseconds for macOS process IDs."""

import ctypes
import sys


class TimebaseInfo(ctypes.Structure):
    _fields_ = [("numer", ctypes.c_uint32), ("denom", ctypes.c_uint32)]


class RusageInfoV2(ctypes.Structure):
    _fields_ = [("uuid", ctypes.c_uint8 * 16)] + [
        (name, ctypes.c_uint64)
        for name in (
            "user_time", "system_time", "pkg_idle_wkups", "interrupt_wkups",
            "pageins", "wired_size", "resident_size", "phys_footprint",
            "proc_start_abstime", "proc_exit_abstime", "child_user_time",
            "child_system_time", "child_pkg_idle_wkups", "child_interrupt_wkups",
            "child_pageins", "child_elapsed_abstime", "diskio_bytesread",
            "diskio_byteswritten",
        )
    ]


def main():
    if len(sys.argv) < 2 or any(not p.isdecimal() or int(p) <= 0 for p in sys.argv[1:]):
        print("usage: proc_rusage.py PID [PID...]", file=sys.stderr)
        return 2
    try:
        libproc = ctypes.CDLL("/usr/lib/libproc.dylib")
        libsystem = ctypes.CDLL("/usr/lib/libSystem.B.dylib")
        rusage = libproc.proc_pid_rusage
        rusage.argtypes = [ctypes.c_int, ctypes.c_int, ctypes.POINTER(RusageInfoV2)]
        rusage.restype = ctypes.c_int
        timebase = libsystem.mach_timebase_info
        timebase.argtypes = [ctypes.POINTER(TimebaseInfo)]
        timebase.restype = ctypes.c_int
        scale = TimebaseInfo()
        if timebase(ctypes.byref(scale)) != 0 or scale.denom == 0:
            raise OSError("mach_timebase_info failed")
    except (OSError, AttributeError) as exc:
        print("proc_rusage unavailable: {}".format(exc), file=sys.stderr)
        return 2

    def centiseconds(ticks):
        return ticks * scale.numer // (scale.denom * 10_000_000)

    for pid_text in sys.argv[1:]:
        info = RusageInfoV2()
        if rusage(int(pid_text), 2, ctypes.byref(info)) != 0:
            print("{}\tNA\tNA".format(pid_text))
        else:
            own = centiseconds(info.user_time + info.system_time)
            children = centiseconds(info.child_user_time + info.child_system_time)
            print("{}\t{}\t{}".format(pid_text, own, children))
    return 0


if __name__ == "__main__":
    sys.exit(main())
