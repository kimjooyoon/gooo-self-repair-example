# Execution contract

The conformance run is intentionally small enough to inspect as one causal record. It consumes immutable releases through the GitHub release API and produces one fixed-denominator integration receipt.

| Boundary | Evidence | Required result |
| --- | --- | --- |
| `.gooo` to semantic IR | `before.gooo`, `after.gooo`, lifecycle contract | 12 activities mapped 1:1 to 12 cells |
| Meaning observation | BEFORE evaluator and counterexample | unknown decision is observed as the historical `CLOSED` bug and expected `UNKNOWN` |
| Candidate generation | improvement-proposer v0.1.1 evidence | one bounded candidate; missing utility remains `UNKNOWN` |
| Semantic mutation | semantic-mutation-lab v0.1.1 source archive | 12 generated/attempted, 10 killed, 1 unknown, 1 refuted; selected fixed-point mutant is `KILLED` |
| Impact frontier | test-frontier v0.1.1 evidence | unobserved execution remains `UNKNOWN` with exact execution counts |
| Verification reuse | verification-reuse v0.1.2 source archive | authorized exact plan closes; reuse counts are explicit |
| Independent oracle | proof-kernel-boundary v0.1.0 evidence | positive direct case is `CLOSED`; verdict override remains `REFUTED` |
| Evidence-first selection | improvement-selector v0.1.1 evidence | crossing resource axes remain `UNKNOWN` |
| Semantic drift | semantic-drift-guard v0.1.1 evidence | canonical positive replay closes; authority escalation remains `REFUTED` |
| Experience memory | experience-memory v0.1.0 evidence | known refuted recurrence goes from 1 to 0 on the second cycle |
| Output | eleven named integration artifacts | no extra generated files; repository writes remain zero |

The activity denominator is fixed at 12; the receipt matrix has a fixed claim denominator of 9: `CLOSED=3`, `UNKNOWN=3`, and `REFUTED=3`, with precedence `REFUTED > UNKNOWN > CLOSED`. Each `UNKNOWN` claim contains all six coordinates: `stage`, `step`, `reason`, `unknown_class`, `next_operation`, and `blocked_by`.

The external release lock records release API identity, annotated tag object and target commit, immutability, and selected artifact digest. An exact utility pair is evidence, not authority: the pair is retained while its decision remains `UNKNOWN` if the resource axes cross and no weighting rule is present.

The parent evaluator and proof-kernel fixtures are separate immutable inputs. Their digests are pinned in the generated evaluator and checked before any legacy case result is trusted. The repaired evaluator closes only an explicit `FIXED_POINT`; hashes and replay evidence cannot close it alone.
