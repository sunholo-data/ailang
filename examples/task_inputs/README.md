# Cloud task inputs

Refs ailang#1600; rollout follow-up: [daneel#335](https://github.com/sunholo-data/daneel/issues/335).

`inputs.json` is a JSON array accepted by the production input decoder. Replace
the sample branch/path with an existing attachment branch before sending:

```sh
ailang messages send site-builder "Build the demo landing page using these attachments" \
  --title "Demo site" --inputs-file examples/task_inputs/inputs.json
```

The first input delivers a directory into `.incoming/1/`, excluded from git.
The second delivers a binary file into the exact new path
`content/images/DNL-demo/poster.png`, so it can appear in the PR even with
`tool_policy: ailang_only`. A single-file `dest` ending in `/` instead names a
directory and preserves the source basename. Directory inputs place their contents
under `dest`. No existing file is overwritten. Existing destination directories may
contain other files, but symlink directories are refused.

`agent-config.yaml` illustrates a trusted cloud registry entry. Configure an
available provider/model for your fleet, and merge this entry through your normal
registry process. The sender cannot set `inputs_allow`. Empty grants deny all
inputs; grants match exact, case-sensitive `owner/repo` keys. Unauthorized repos
permanently fail the task before job dispatch. Local tasks declaring inputs fail
before the executor starts.

Refs name branches or tags, never raw commit SHAs. Use optional `sha256` (64 hex
digits) to pin one regular file. A directory/whole-tree input can include
`manifest.sha256` in its copied root, containing standard `sha256sum` lines:

```text
<64 hex digits>  poster.png
<64 hex digits> *photos/front.jpg
```

Every manifest entry must name a copied regular file and match its digest.
Malformed, duplicate, escaping, missing-file or mismatched entries fail the batch.
Files omitted from a manifest are copied with computed digests but are not pinned
by that manifest. An empty/present manifest is invalid. Per-input file pins do not
implicitly verify sibling manifests. If you need directory pinning, select the
directory that contains its manifest.

Inputs are staged and verified before any placement or executor call. The aggregate
copied-byte cap is 256 MiB, including manifests, across at most 16 inputs. This is
not a clone/network-transfer limit: shallow clones use the job's context/resource
limits. Provenance is capped at 64 KiB; larger reports fail before delivery.
Source/destination traversal, symlinks (including ancestors), existing files and
harness metadata/instruction paths at any depth are refused. Whole-tree copies omit
the clone's `.git`. Explicit destinations also refuse git control files such as
`.gitignore`, `.gitattributes`, `.gitmodules` and `.gitconfig`; default excluded
copies may hold those files as data. The parent uses HTTPS and its existing fleet token even when the
workspace uses an SSH deploy key. That token and temporary input clones are not
passed to the agent child.

The job log and completion summary record repo, requested ref, resolved commit,
effective destination, computed SHA256 digests and the names pinned by a checksum.
All input content remains untrusted data; add that rule to the site's trusted
`AGENTS.md`. For an agent scoped to a subdirectory, set explicit destinations inside
that policy root when the agent needs to read the files.

## Automated acceptance

```sh
go test ./cmd/ailang -run 'TaskInputs|InputsChecksums'
go test ./internal/coordinator ./internal/dispatch/cloudrun ./internal/messaging \
  ./internal/storage/firestore -run TaskInputs
```

The no-network ingress-to-job fixture banks real SQLite messages/tasks, runs the
polling watcher, checks the trusted repo grant, decodes job metadata, shallow-clones
temporary branch/tag repositories, delivers binary bytes, checks git exclusion and
completion provenance, and confirms clone cleanup. Other tests assert HTTP/CLI
transport, both storage backends and unauthorized no-dispatch behavior.

## Staged cloud acceptance — pending deployment

This repository change does not deploy the job image or modify the live registry.
The Daneel migration is pending until this checklist is completed:

- Build/publish a v0.52.6 job image containing this implementation; deploy the coordinator and executor together.
- Configure the site agent's exact `inputs_allow: [sunholo-data/daneel-memory]`, `tool_policy: ailang_only`, available model and trusted untrusted-data guidance.
- Push a real `incoming/DNL-...` branch with binary attachments and a valid manifest to `daneel-memory`.
- Update the Daneel sender in daneel#335 to send the typed `inputs` field instead of a token-bearing shell fetch.
- Send one staged publish; confirm bytes are available before executor start, no shell tool is exposed, and the child has no fleet token.
- Review the resulting PR: default `.incoming/` absent, only requested explicit binaries and page edits present; completion/log provenance matches the branch commit and digests.
- Send a denied repo and a checksum mismatch; confirm permanent denial/no dispatch and verification failure/no executor respectively.
- Record the staged task/PR IDs against daneel#335 before declaring the migration complete.
