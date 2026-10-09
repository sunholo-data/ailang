### Fixed — independent Codex OAuth profiles and refresh ownership

Local mission controllers, probes and executor roles select a dedicated Codex home
and connect to one managed app-server daemon. Cloud subscription jobs acquire a
nonexpiring Firestore credential lease before reading auth, persist rotated tokens
before releasing ownership, and quarantine uncertain ownership or write-back failures.
Attended provisioning creates separate logins to the same account and checks that
interactive, mission and cloud credentials belong to the same user/workspace with
distinct refresh tokens. See `tools/attended/CODEX_OAUTH_PROFILES.md` for activation
and recovery; provisioned credentials alone do not activate either automation lane.
