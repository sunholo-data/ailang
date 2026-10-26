"""Bounded hostile-child fixture, never a production process owner.

The trusted test records this exact fixture's PID solely for emergency cleanup.
Candidate-supplied PIDs are not an ownership mechanism.
"""
import json
import os
from pathlib import Path
import sys
import time

ready, escape, leader = sys.argv[1:]
pid = os.fork()
if pid == 0:
    os.setsid()
    if escape == "escape":
        os.chdir("/")
    os.closerange(0, 256)
    # A create-only readiness file confirms setsid/cwd/descriptor changes.
    with open(ready, "x") as stream:
        json.dump(dict(pid=os.getpid(), sid=os.getsid(0), cwd=os.getcwd(),
                       closedDescriptorRange=[0, 255], maxLifetimeSeconds=30), stream)
    time.sleep(30)  # Last-resort bound even if the test controller is killed.
    os._exit(0)

deadline = time.monotonic() + 5
while not Path(ready).exists():
    if time.monotonic() > deadline:
        raise RuntimeError("child readiness deadline")
    time.sleep(0.01)
if leader != "normal":
    time.sleep(30)
