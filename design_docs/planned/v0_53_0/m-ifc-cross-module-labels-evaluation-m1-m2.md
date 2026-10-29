# #1771 independent M1-M2 evaluation

**PASS95/100**, plus conditional regression coverage10/10. Exact40a5ca9eac5b4175a366fc441c451f4dfb0f12aa. All11 approved criteria met. This accepts M1-M2 only; M3/M4 and issues1134/1132 remain open, with the full parent/plan still planned.

Exact-head CI has26 terminal checks:19green/7skipped,0pending, SonarSUCCESS. Existing evaluator script CI_GATE=pass; tests/lint/sizes pass, no profiling goal or newTODO. Root independently reviewed source and replayed its earlier producer-alias/HOF REDs; final IFC production is unchanged from that tested checkpoint. Deduct5 under the function-length cap: AnalyzeModuleIFC56, codec test76 and cache invalidation test57lines.

F4 forbidden lists/reserved applied fields stay in v2, canonical missing-vs-clean/JSON/gob/hash validation is fail-closed, unlabelled package identity preserved, v8 invalidates valid v1-v7 keys while retaining the shared DAG codec. Known producer references match the existing conservative eta policy in export analysis only; imported-call enforcement is not claimed. All49 interfaces and paired metadata/18CLI compatibility controls pass.

Independent narrow artifact audit checked actual960 payload hashes plus source/DAG/stamp/size data. Measured143-module Motoko cohort52,171,074bytes, maxblob3,100,965/maxmodule3,400,869, actual60core/9extensions and143warmhits0misses. Separate post-alias97-module protocol proof has48,230,073bytes,maxblob3,606,290/maxmodule4,664,163,97warmhits0misses. All below unchanged64MiB/blob128MiB/module. The143cohort excludes49 later-added cache modules; IFC473459 is compact extracted payload, not measured incremental overhead; unchanged stamps refer to hashed artifact manifests. Pre-alias8f fullMotoko/native proof and focuseda997 post-alias proof remain distinct.

Existing metadata/codec/CLI fixtures exercise this internal foundation; new enforcement showcases/guide claims remain deferredM4. Four original feedback comments are addressed. A historical245-vs247 example-count typo will be corrected in the completion audit without a source/CI respin. No whole-sprint completion/archive, issue closure, controller card action, admin bypass, version/tag or deployment.

EVALUATION_SCORE: 95/100
EVALUATION_ROUND: 1
EVALUATION_VERDICT: PASS score=95 scope=M1-M2-only

[Accepted PR1771](https://github.com/sunholo-data/ailang/pull/1771), merge65253e86e68be251eecd796d92327e4d4828c234. [Artifact audit](m-ifc-cross-module-labels-artifact-audit.md), [validation](m-ifc-cross-module-labels-repair-validation.md), [machine report](../../../.ailang/state/evaluations/eval_M-IFC-CROSS-MODULE_round_1.json). Full parent remains planned; this partial report stays with its existing plan.
