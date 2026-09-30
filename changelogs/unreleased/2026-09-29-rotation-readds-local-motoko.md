### Changed — local motoko back in the GPU rotation

`tools/launchd/os-rotation-filler.sh` runs `motoko-local-qwen3-8-27b-microrag`
again alongside opencode and pi. It is motoko main (extension ABI 8.0, the
`~/dev/mk-main` checkout) on the full local profile, so it gets its own
leaderboard column rather than continuing the old lean `motoko-local-qwen3-8-27b`
one. Its canary passed the eval-suite pre-flight and 3 of 4 benchmarks through
the rig gateway. It needs ailang v0.48.0 or later, whose clients send the rig
lease; with older clients every motoko model call is refused.
