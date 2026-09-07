# Controlled rollout — 2026-09-07

## Candidate and upstream integration

Merged origin/dev at 1fd1c7a70 into the isolated recovery branch without conflicts. This includes
V1's #1074 event-handler extraction and the current mission records. Re-ran full make test,
lint, architecture and file-size checks, targeted race tests and vet: all passed, including
the formerly failing file-size check. Five receipt tests now explicitly skip Windows directory
sync, matching the existing fail-closed platform boundary; independent reviewer PASS.

Candidate code: dad6e8efb0d939a48f639b5f39907994a5f11792.

## Actual World driver preflight

Executed the committed candidate through the real pin helper and real mission-control driver,
using a new isolated pin worktree, World profile, existing World work repository, dry-run mode,
5-second model-probe setting and 180-second outer process deadline. No saved configuration edit.
The driver performed its ordinary small model probes; no full mission iteration ran.

Result: exit 0, PIN_STATUS=pinned, all role lanes usable, World workdir retained. The new pin
checkout's full SHA matches the candidate; World's origin is sunholo-data/ailang-world and its
charter exists. See [machine evidence](world-preflight.json). A preflight is not a successful
live mission iteration and does not justify calling the whole fleet migrated.

## Publication boundary

Automatic approval review rejected `git push origin HEAD:refs/heads/sprint/mission-runtime-recovery`
because explicit authorization is required to send the private code/history to that GitHub
destination. The user has been asked for this specific approval. No push was performed.
The local preflight uploads no code and is not a workaround for that boundary.

World's saved AILANG_DRIVER_PIN=0 remains intact. The installed ailang binary was not replaced.
V1 was running at inspection; no mission job was stopped or restarted. No weekly thread pointer,
directive watermark, approval ledger or message acknowledgement was changed.

## Once publication is authorized

1. Push candidate branch to sunholo-data/ailang and verify the remote SHA/checks.
2. Record a backup of World env and its exact hash, plus a rollback restoring that file.
3. At an idle boundary, select the reviewed candidate ref for World alone and restore its pin.
   Do not change the other three loops or assume local HEAD is the default origin/dev ref.
4. Observe the next controlled World fire: pinned driver SHA, actual work repository, charter
   reachability and stage receipt. On wrong-repository or startup failure, restore the backup.
5. Only after the live proof, propose adoption on the default driver ref. Durable role-state
   adoption, artifact acceptance and quota reservations remain separate from this pin repair.
