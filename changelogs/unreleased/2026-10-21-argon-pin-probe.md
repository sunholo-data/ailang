### Added — live probe W series for pinning gemini-4-argon via agent_config on Vertex and the Gemini Developer API

`TestLiveArgonPin` (opt-in, `AILANG_LIVE_MA_FEATURES=1`) sends `agent_config {type: antigravity, model: gemini-4-argon}` to both surfaces with a no-config baseline and two controls: a bogus model and a wrong `type`. It reports every model field on the wire, so acceptance is never read as proof of routing.
