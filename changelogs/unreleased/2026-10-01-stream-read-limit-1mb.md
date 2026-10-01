### Fixed — Stream: inbound read limit no longer kills messages over 64 KB

`std/stream` connections, including serve-api's outbound bridge dial to Vertex Gemini Live,
had a default inbound read limit of 64 KB (`MaxFrameSize`). gorilla applies it to the whole
reassembled message, not to one frame, so any upstream message over 64 KB ended the session
with `1006 websocket: read limit exceeded`. Live avatar video sessions failed 5/5 once Vertex
began sending a first media message over 64 KB. The default now equals `MaxMessageSize`
(1 MB), which the bridge and `send` already enforce. When the limit does trip, the error
now names it (`read limit exceeded: message over N bytes`), so the bridge end reason
carries a number. Reported by Daneel (inbox_1790847071255_f0b53c49).
