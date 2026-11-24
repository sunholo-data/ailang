### Fixed — standard-mode Gemini evals bill the project their models.yml row names

`ailang eval-suite` in standard mode ignored a Google row's `gcp_project` and `gcp_location`,
so every Vertex call billed whatever project the resolver found (`AILANG_CLOUD_PROJECT`,
`GOOGLE_CLOUD_PROJECT`, then `~/.ailang/config.yaml`). On the rig that is prod
`ailang-multivac`: the November Gemini 4 placement runs (about $30) landed there although
every row says `ailang-dev`. Agent mode already honoured the row.

The eval agent now pins the row's project and location through `factory.WithGCPProject` /
the new `factory.WithGCPLocation`. A pinned project is Vertex-only, so a missing ADC
credential fails loudly rather than falling back to an API key that bills elsewhere. Rows
without the fields behave as before. Verified end to end: with `AILANG_CLOUD_PROJECT` set
to a project that does not exist, the old binary fails with 403 on that project and the
new one passes through `ailang-dev`.
