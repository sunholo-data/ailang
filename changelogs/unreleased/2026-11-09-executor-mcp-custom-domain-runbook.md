### Added — runbook for the executor MCP custom domain (2026-11-09)

`docs/internal/executor-mcp-custom-domain.md` covers v1.0 gate G4(b): serving the prod executor MCP
(the OAuth service) on `executor.ailang.sunholo.com` instead of its run.app URL. It records how the
docparse precedent is mapped (a Cloud Run domain mapping, a GoDaddy CNAME and a Google-managed
certificate), what the OAuth issuer change touches, and the cutover order for the Terraform change in
ailang-multivac PR #12. Nothing is applied yet: the CNAME, the mapping and the promotion are Mark's to do.
