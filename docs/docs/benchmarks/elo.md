---
sidebar_position: 2
title: ELO Ratings & Difficulty
description: Model capability and benchmark difficulty derived from ELO ratings — per mode (standard vs agent), with saturation and grader-artifact flags.
---

import EloLeaderboard from '@site/src/components/EloLeaderboard';

# ELO Ratings & Difficulty

The rig rates every **(model, benchmark)** trial as a game — a PASS is the model
beating the benchmark — and fits ELO ratings from the outcomes. This gives two
things the raw pass-rate can't: a **model capability** ranking that holds even when
models are tested on different subsets, and a **benchmark difficulty** that emerges
from the data instead of being hand-assigned.

Ratings are **mode-separated**: standard (0-shot + repair) and agent (multi-turn)
are different difficulty regimes — agent mode saturates, so a combined rating would
be meaningless. Use the toggle to switch.

Numbers are **regraded** (M-EVAL-OUTPUT-NORMALIZE: boolean-case + numeric formatting
parity), so a correct answer isn't marked wrong over output formatting.

<EloLeaderboard />

**Reading it:** dimmed (Trivial) rows are saturated — every model passes, so they're
demotion candidates that no longer discriminate. A **⚠ artifact** badge means the
benchmark *looks* hard only because of a grader/benchmark quirk (e.g. a Python set
`{}` vs an expected list `[]`, or a free-text answer graded by exact match), not
genuine difficulty — those are slated to be fixed, not celebrated as hard.
