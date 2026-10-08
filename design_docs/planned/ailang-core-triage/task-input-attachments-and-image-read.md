# Task input attachments + image-returning read so a plane agent can see images

- **Date**: 2026-10-03
- **Class**: feature
- **Recommend**: design-doc
- **Searched**: `attach`, `attachment`, `--attach`, `inputs/`, `vision`, `modalit`, `type: "image"` across `design_docs/`, `cmd/ailang/messages*.go`, `internal/messaging/`, `internal/modelreg/models.yml`, `cmd/ailang/pi_assets/`; Daneel `design_docs/planned/m-daneel-avatar-voice.md` (the motivating request, since superseded by M-DANEEL-LIVE-PAGE — the feature ask is general and still stands)
- **Source**: sunholo-data/ailang#1307 (from:daneel, 2026-09-25)

## Why design-doc

Three asks, all unbuilt at origin/dev `790169359`, and they cross three surfaces with a data-boundary and storage choice in the middle, so this is not a direct fix:

1. **No attachment on a message or task.** `ailang messages send` (`cmd/ailang/messages_send.go`) has no file flag; `internal/messaging/schema.go:14` records the `attachments` table as *removed* in schema v1.6.0 (M-DB-CLEANUP, "never used, no consumers"). The cloud workspace is only a git clone (`cmd/ailang/coordinator_cloud.go:323` `executeCloudTask`), with no hook to place input files. `internal/messaging/image_extractor.go` downloads GitHub-hosted images referenced in issue markdown — the nearest existing mechanism, but it is URL-in-body, GitHub-only and local-cache-only.
2. **`ailang_read` returns text only.** `cmd/ailang/pi_assets/ailang-exec.ts:294` always emits `{type: "text", text: JSON.stringify(payload)}`; an image arrives as bytes inside JSON.
3. **No modality in the model registry.** `internal/modelreg/models.yml` mentions "vision" only in free-text notes (e.g. :1281); there is no field a lane could check.

## Decisions the doc must make

- **Where attachments live**: re-introduce a messaging `attachments` table (local SQLite + Firestore), or store blobs beside the task's existing GCS artifacts and carry only refs (name, mime, size, sha256, gs:// URI) on the message. GCS-refs keeps Firestore docs small and matches how task artifacts already travel.
- **Materialisation**: copy into `/workspace/<task>/inputs/` before the agent starts (cloud executor and local daemon parity), read-only, outside the git tree so they are never committed by accident.
- **Limits and trust**: max size/count per task, mime allow-list (images, PDF), and whether an attachment from an external sender (feedback gate / messages API) is accepted at all — this is a data-boundary ruling, same shape as #1262's.
- **Image read**: `ailang_read` returns an image content block for image mimes (pi tool result), with a size cap and downscale policy; other executors (claude/codex) read files natively, so confirm which lanes actually need this.
- **Model modality**: add `vision: true|false` (or `input_modalities: [text, image]`) to `models.yml`; the dispatcher refuses (or reroutes) a task carrying image attachments to a text-only model rather than silently dropping them — e.g. the lane's `z-ai/glm-5.3` is text-only, `glm-5.3-flash` is multimodal.

## Suggested shape

M1 `messages send --attach FILE...` + GCS storage + refs on the task; M2 materialise into `inputs/` in both executors; M3 `vision` field + dispatch refusal; M4 pi `ailang_read` image block. M1–M3 are useful without M4 (claude/codex lanes already read images from disk).
