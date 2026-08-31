# Execution contract

The conformance run is intentionally small enough to inspect as one causal record.

| Boundary | Evidence | Required result |
| --- | --- | --- |
| `.gooo` to semantic IR | `before.gooo`, `after.gooo`, lifecycle contract | 12 activities mapped to 12 cells |
| BEFORE evaluator | three unknown-decision scenarios | expected refutation receipt; history remains `REFUTED` |
| Candidate adoption | candidate, independent evaluation, authorization | append-only lifecycle is preserved |
| AFTER evaluator | three normal and three defensive scenarios | explicit `FIXED_POINT` closes; unknown decisions fail closed |
| Replay | one immutable-history scenario | replay evidence alone cannot close |
| Output | eight named artifacts | no extra generated files |

The parent evaluator and proof-kernel fixtures are separate immutable inputs. Their digests are pinned in the generated evaluator and checked before any case result is trusted.
