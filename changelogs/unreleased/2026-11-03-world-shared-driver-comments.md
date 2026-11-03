### Fixed — stale "World is a fork" claims in the World registry entry, the mission-loop-change skill and comments

World was de-forked on 2026-09-06 and runs the shared driver through the origin/dev pin; the registry, skill and comments now reflect that arrangement (rechecked 2026-11-03).
The skill's reach-check grep matched no mission because the pin is now sourced via `$MC_DRIVER_ROOT`; the check now reads installed plists and recognizes that source line.
There are no behaviour, key or assertion changes.
This is the mechanical sub-item of D-FLEET-25, proposed pending a ruling.
