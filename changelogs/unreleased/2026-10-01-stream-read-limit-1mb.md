### Fixed — Stream: inbound read limit no longer kills messages over 64 KB

`std/stream` connections, including serve-api's outbound bridge dial to Vertex Gemini Live,
had a default inbound read limit of 64 KB (`MaxFrameSize`). gorilla applies it to the whole
reassembled message, not to one frame, so any upstream message over 64 KB ended the session
with `1006 websocket: read limit exceeded`. Live avatar video sessions failed 5/5 once Vertex
began sending a first media message over 64 KB. The default now equals `MaxMessageSize`
(1 MB), which the bridge and `send` already enforce. When the limit does trip, the error
now names it (`read limit exceeded: message over N bytes`), so the bridge end reason
carries a number. Reported by Daneel (inbox_1790847071255_f0b53c49).
### Added — `--stream-max-message` on `ailang run` and `serve-api`

Hosts can now set the Stream message cap (default 1MB). It applies to one message in both
directions, and on `serve-api` to both legs of a bridge. Sizes use the `--fs-max-bytes`
spellings (`256KB`, `8MB`). Zero, a negative value, a malformed value, or setting the flag
without `--caps Stream` is an error. `--policy` runs refuse the flag. Raise the cap for trusted
upstreams that send large single messages. Lower it for public `serve-api` endpoints, where
inbound memory is bounded by about sessions × queue frames × cap.

### Changed — one Stream size field

`StreamContext.MaxFrameSize` is gone. The transport read limit is now `MaxMessageSize` itself,
so the two can no longer drift apart; the 64KB/1MB drift is what killed Gemini Live video
sessions. `StreamDialConfig` and `streamws.AcceptOptions` carry `MaxMessageSize`.

### Fixed — streaming guide showed `serve-api`-only flags on `ailang run`

`--stream-idle-timeout` and `--stream-max-duration` exist only on `serve-api`. The guide's
`ailang run` example used them.
