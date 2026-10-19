### Fixed — motoko's run state stays out of agent commits (2026-10-19)

motoko writes run state into the workspace's `.motoko/` (a profile copy, exit manifests, a model catalog), and
the cloud job's commit step staged it: the first motoko lane PR on the dev plane (#1894) carried four such files
beside its report. New files under `.motoko/` are no longer staged. Edits to `.motoko` files a repository already
tracks still are, since motoko_agent commits its own `.motoko/config`.
