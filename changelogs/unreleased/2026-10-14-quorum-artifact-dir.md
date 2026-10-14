### Fixed

- Opt-in `MISSION_QUORUM_ARTIFACT_DIR` keeps future quorum evidence in a durable absolute directory shared by missions of the same repo. Policy-marked default writers remain confined to their local sandbox and report suppression.
- Mission startup validates and exports the repo namespace before either controller launch path. Both Gate 2 readers check the exported directory and current repo fallback in that order; read/configuration errors stop the pick even when another directory contains a match. Legacy evidence importing remains a separate task.
