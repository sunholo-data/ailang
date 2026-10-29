# Round1 supplementary finding

The original FAIL73 report and checksum failure are preserved unchanged. A later actual Sonar analysis at the samecb5 head reports new-code coverage77.7% below the existing80% threshold; all other gate conditions pass.

A focused meaningful regression-test repair is assigned within approved M0 scope. Production compiler semantics, coverage exclusions/thresholds and workflows stay unchanged. Existing compiler/corpus/mutation evidence can be reused only after production-byte identity is independently verified. Newly added tests require focused validation and exact-new-head CI/Sonar. Independent acceptance remains on hold.
