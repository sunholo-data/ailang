### Fixed — agy no longer leaves "Keychain Not Found" dialogs open on macOS

- agy saves a refreshed sign-in token to the macOS login keychain. The executor's route
  HOMEs (`~/.ailang/state/agy/<route>`) had none, so every token refresh under them raised a
  SecurityAgent "Keychain Not Found" dialog that stayed open after agy exited. The rig's
  Google quota read (`ailang mission quota`, once per mission fire) piled them up on the
  studio's screen, and a dialog could hold a quota read past the driver's 60s ration-gate bound.
- Every agy launch (executor runs, `agy.RunQuota`, the rig's quota read) now first gives its
  HOME a locked keychain of its own (`agy.EnsureLockedKeychain`). It has a random, discarded
  password and is that HOME's default, so agy's save fails at once (exit 36) without a dialog, and agy keeps
  its token file. The operator's own keychain search list is untouched. A no-op off macOS.
