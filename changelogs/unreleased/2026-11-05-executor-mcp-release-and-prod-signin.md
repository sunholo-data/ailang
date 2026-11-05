### Added — release pipeline builds the `executor-mcp` image (2026-11-05)

`cloudbuild-release.yaml` now has a `build-executor-mcp` step, so every `v*` tag pushes
`executor-mcp:vX.Y.Z` and `:latest` to the test registry, with no `allowFailure`. Until now only
`cloudbuild-dev.yaml` built it, so test and prod had no executor-mcp image and
`promote-to-prod` had nothing to copy. `deploy-test` waits on the build. Neither executor service
exists in test, so the shared deploy library skips both and the step only publishes the image.
Promote copies it once ailang-multivac's `AILANG_IMAGES` lists `executor-mcp`. The prod executor
OAuth launch (ailang-multivac PR #10) pins this image by release tag.

### Added — executor sign-in page knows the prod Firebase project (2026-11-05)

`executor_oauth/static/oauth-login.js` has an `ailang-multivac` entry: the prod
`firebase-browser-docparse` browser key, which AILANG Parse's prod sign-in already uses, and
`ailang-multivac.firebaseapp.com`. Before this, a prod service would have shown "Sign-in is not
configured for this server", because the page refuses project ids it does not know.

### Changed — executor OAuth: an unset or empty allowlist admits nobody; `*` opens sign-in (2026-11-05)

Until now `EXECUTOR_ALLOWED_EMAIL_DOMAINS` unset or `""` admitted every verified Google account.
Open sign-in could therefore happen by omission, for example from an image run without the
variable or from the terraform default of `""`. Now unset and empty refuse every sign-in with
"sign-in is not configured on this environment", and open sign-in needs the explicit value `*`.
`executor_oauth/test_oauth.sh` covers unset, empty and `*` (27 checks). The three new checks fail
against the old gate. **Prod must set `executor_mcp_oauth_allowed_email_domains = "*"`, not `""`.**
