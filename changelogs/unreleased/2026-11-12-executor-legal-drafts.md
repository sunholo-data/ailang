### Added
- **Draft privacy policy and terms for the hosted executor and docs server**
  (`docs/docs/legal/privacy.md`, `docs/docs/legal/terms.md`). Both directories require a privacy
  policy URL, and OpenAI also requires a terms URL. The AILANG site had neither. Both pages carry
  `draft: true`, so production builds leave them out until Mark approves the wording and fills in
  the contact address. Once published they appear at `/docs/legal/privacy` and `/docs/legal/terms`,
  which the executor's consent page already links. The privacy draft lists every record the
  executor keeps, with its lifetime as measured in the code and the cloud configuration: token
  digests, 30-day logs, code never stored, feedback, and sub-processors, including Discord and the
  AI tools used for feedback triage.
- **Hosted executor guide: "Disconnecting and your data" and "Troubleshooting".** The new sections
  cover how to remove the connector, what expires when, deleting an account, and fixes for an
  expired sign-in link, an unverified email, a blocked pop-up, a missing token, the 30/min rate
  limit, policy refusals and the 60 s timeout. The guide also notes the email sign-in, which is
  only for accounts we create.
