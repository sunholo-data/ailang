### Changed — motoko image pins the ailang_only lane's teaching prompt (b3347a96)

- `docker/Dockerfile.agent-motoko` pins motoko_agent `b3347a96`: the `ailang_policy` lane
  extension now sends the canonical `ailang prompt` and pi's lane guidance (`.ailang-scratch/`,
  package ceilings, network/process lines) in the system role. Before, motoko's lane told the
  model what tools it had but never taught it AILANG. On 2026-10-12, Haiku 5.5 passed 6/6 real
  lane tasks on both pi and motoko.
