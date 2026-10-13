- Fixed the same day: `ailang run` binds its AI handler through `prepareRunAIBinding`, not
  `setupAIHandler`, so the exclusions were never installed for `ailang run` programs, including
  motoko's runtime. On 2026-10-13, 118 of motoko's GLM calls still reached `inference-net`. They are
  now installed on both paths, with a test on the run path.
