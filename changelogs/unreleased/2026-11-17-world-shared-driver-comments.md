### Fixed — stale "World is a fork" claims in the World registry entry, the mission-loop-change skill and comments

World was de-forked on 2026-09-06 and runs the shared driver through the origin/dev pin; the registry, skill and comments now reflect that arrangement (rechecked 2026-11-03 and again 2026-11-17: the installed World plist runs `sunholo-data/ailang/tools/launchd/mission-control.sh`, and `ailang-world` carries no driver copy).
The skill's reach-check grep matched no mission because the pin is now sourced via `$MC_DRIVER_ROOT`; the check now reads installed plists and recognizes that source line (2026-11-17: 7 of 7 installed mission plists report `re-execs-from-pin=YES`).
There are no behaviour, key or assertion changes.
This is the mechanical sub-item of D-FLEET-25 (option A, ruled by Mark 2026-11-16).
