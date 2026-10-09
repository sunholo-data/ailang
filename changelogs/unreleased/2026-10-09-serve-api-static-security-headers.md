### Fixed

- `serve-api --static` now sends nosniff, X-Frame-Options DENY, CSP frame-ancestors 'none', and Referrer-Policy no-referrer on all static statuses. Repeatable `--static-header 'Name: value'` overrides defaults; `--no-static-security-headers` opts out. Same-origin framing requires overriding both X-Frame-Options and CSP. Invalid flags fail startup. Refs #1597.
