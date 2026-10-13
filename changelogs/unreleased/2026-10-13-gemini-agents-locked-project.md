### Added — locked-down project setup for keyless Gemini Developer API access

`scripts/security/setup_gemini_agents_project.sh` creates `ailang-gemini-agents`: no Firebase,
no API keys (org policy) and no SA key files, with only `generativelanguage` and `iamcredentials`
allowed through `gcp.restrictServiceUsage`. Access is by impersonating a single SA whose only role
is `serviceUsageConsumer`, plus a budget alert. It is a dry run unless `APPLY=1`. Account
identifiers are read from `ailang-dev` at run time, so none are committed.
