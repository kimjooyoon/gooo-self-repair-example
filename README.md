# gooo-self-repair-example

An executable, append-only example of a Gooo self-repair cycle that closes one real semantic defect:

> an unknown top-level decision must not be hidden as `FIXED_POINT`; it resolves to `FAIL_CLOSED` while preserving the full `UNKNOWN` reason tuple.

The GitHub Actions conformance job runs the complete chain:

`observation → Gooo candidate → bounded semantic mutation → impact test frontier → verification reuse → independent oracle → evidence-first selection → semantic drift gate → experience memory → second cycle`

The contract keeps one fixed denominator of twelve Gooo activities and a nine-claim receipt matrix with exactly three `CLOSED`, three `UNKNOWN`, and three historical `REFUTED` claims. Every `UNKNOWN` claim carries `stage`, `step`, `reason`, `unknown_class`, `next_operation`, and `blocked_by`. `REFUTED` has precedence over `UNKNOWN`, which has precedence over `CLOSED`.

External inputs are consumed only from the pinned GitHub release API records in [`contracts/external-release-lock-v1.json`](contracts/external-release-lock-v1.json). Each release is checked by API identity, annotated-tag resolution, immutability, and selected artifact digest before its evidence is trusted. The external utility pair is exact but remains `UNKNOWN` when its resource axes cross; the core semantic repair is `CLOSED` only because its independent oracle and repaired evaluator agree directly.

CI uses Go 1.27 and records exact integer build/test/conformance wall time, peak RSS, test totals and observation categories, Go/Gooo file and line counts, and repository writes. No local Go test execution is part of the authority boundary. The conformance run writes twelve receipts/report files to a caller-owned temporary artifact directory and verifies that the repository working tree is unchanged.

Development process authority is reported separately from semantic validity. The historical direct-main write at `5dca56d` is retained as one `REFUTED` violation with all six `UNKNOWN` coordinates; after the follow-up PR merge, the API guard closes the current PR-associated path while requiring zero repository direct writes after the guard. The preserved v0.2.0 semantic artifacts and immutable release identity do not erase that process history.

The historical `evaluate` command and its original fixture remain available for comparison. The current release path is the `integrate` command invoked by [`scripts/conformance.sh`](scripts/conformance.sh).
