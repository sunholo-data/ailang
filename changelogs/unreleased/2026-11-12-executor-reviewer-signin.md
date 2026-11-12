### Added
- **Email + password sign-in on the hosted executor's consent page, for accounts we create.**
  The Claude and ChatGPT directories both need a reviewer login that works without MFA, emailed
  codes or magic links, and the executor offered only Google. `executor_oauth/static/oauth-login.*`
  now has a collapsed "Have an account we created for you? Sign in with email" form that calls
  Firebase `signInWithEmailAndPassword` only: the page never creates an account or sends a reset,
  so there is no self sign-up. It works once the Email/Password provider is enabled in the Firebase
  project, which is a console step (on 2026-11-12 it is off in both `ailang-multivac` and
  `ailang-multivac-dev`). The server's gate is unchanged and still requires a verified email, so a
  hand-made account must be marked verified. The page now also says when a signed-in account's
  email is unverified, instead of letting **Allow** end on the server's error.
- **Privacy, Terms and Help links on the consent page** (OpenAI: the sign-in flow must be
  transparent). They point at `https://ailang.sunholo.com/docs/legal/{privacy,terms}` and the
  hosted-executor guide.
- **`GET /.well-known/openai-apps-challenge` on the executor** (OpenAI plugin domain verification).
  It serves `OPENAI_APPS_CHALLENGE`, trimmed, as `text/plain`, and returns 404 when the variable is
  unset or blank. The contract is the same as AILANG Parse's.
  `executor_oauth/test_oauth.sh` grows from 43 to 48 checks: the email form is there with no
  sign-up call, the legal links are there, and the challenge route returns 404 when unset and the
  exact token when set.
