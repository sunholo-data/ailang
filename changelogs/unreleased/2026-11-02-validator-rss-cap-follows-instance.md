### Fixed — registry validator memory cap follows the instance limit

The registry validator bounded `ailang check --package .` at a hardcoded 384 MiB,
sized for its original 512 MiB Cloud Run instance. When the instance moved to
2 vCPU / 1 GiB (ailang-multivac `2cf025e`, after `sunholo/ailang_parse@0.55.0` hit
the 200 s wall cap 7/7 times on 1 vCPU) the cap did not move, and
`sunholo/celestial@0.6.0` was refused at 384 MiB on a 1 GiB box. The cap is now
the instance limit minus 128 MiB, read at startup from cgroup v2, cgroup v1 or
`/proc/meminfo` (Cloud Run gen1/gVisor), and logged with its source; 384 MiB
remains only when no limit is readable.
