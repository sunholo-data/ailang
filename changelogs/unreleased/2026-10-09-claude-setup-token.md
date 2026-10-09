### Fixed — cloud Claude executors accept a `claude setup-token` token

Every cloud Claude run failed in its first turn with "OAuth session expired and could not be
refreshed". The dev and prod secrets held a JSON credential blob (access and refresh token, from
March and April). A container refreshes that blob in-process and throws the refreshed pair away,
so once the stored refresh token was spent no run could authenticate. When
`CLAUDE_CODE_OAUTH_TOKEN` holds a setup-token (`sk-ant-oat…`), the executor now passes it to the
claude child in its environment, which is Claude Code's headless path, and writes no credentials
file. A setup-token does not refresh and lasts about a year. The JSON blob keeps its existing
credentials-file path and still never reaches the child.
